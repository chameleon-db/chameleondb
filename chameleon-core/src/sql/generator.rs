use crate::ast::{BelongsToKeyError, Entity, Relation, RelationKind, Schema};
use crate::query::{
    Query, FilterExpr, FilterCondition, FilterValue,
    ComparisonOp, LogicalOp, SortDirection, OrderByClause, FieldPath,
};
use super::naming::{entity_to_table, quote_ident};
use serde::{Deserialize, Serialize};

/// Character used to escape LIKE wildcards in user values.
const LIKE_ESCAPE: char = '\\';

/// A value bound to a placeholder (`$1`, `$2`, ...) of a generated statement.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum SqlParam {
    String(String),
    Int(i64),
    Float(f64),
    Bool(bool),
    Null,
    /// The ids of the parent rows, supplied by the executor at run time
    /// (bound as an array, used with `= ANY($n)`).
    ParentIds,
}

/// Result of generating SQL for a query
/// A single logical query can produce multiple SQL statements
/// (main query + eager loading queries)
///
/// Statement text only contains placeholders; values travel in the
/// parameter lists. The parameter lists default to empty so that older
/// consumers of the JSON keep deserializing.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct GeneratedSQL {
    /// The main SELECT query
    pub main_query: String,
    /// Eager loading queries (one per include level)
    /// Each tuple is (relation_name, sql)
    pub eager_queries: Vec<(String, String)>,
    /// Parameters of the main query, in placeholder order
    #[serde(default)]
    pub main_params: Vec<SqlParam>,
    /// Parameters of each eager query, parallel to `eager_queries`
    #[serde(default)]
    pub eager_params: Vec<Vec<SqlParam>>,
}

/// Collects parameters while a statement is built.
#[derive(Default)]
struct Params(Vec<SqlParam>);

impl Params {
    /// Register a value and return its placeholder.
    fn push(&mut self, value: SqlParam) -> String {
        self.0.push(value);
        format!("${}", self.0.len())
    }
}

/// Generate SQL from a Query + Schema
pub fn generate_sql(query: &Query, schema: &Schema) -> Result<GeneratedSQL, SqlGenError> {
    let entity = schema.get_entity(&query.entity)
        .ok_or_else(|| SqlGenError::UnknownEntity(query.entity.clone()))?;

    // Validate every identifier before any SQL is produced.
    validate_query(query, entity, schema)?;

    // Build main query
    let mut params = Params::default();
    let main_query = build_main_query(query, entity, schema, &mut params)?;

    // Build eager loading queries
    let (eager_queries, eager_params) = build_eager_queries(
        &query.entity,
        &query.includes,
        schema,
    )?;

    Ok(GeneratedSQL {
        main_query,
        eager_queries,
        main_params: params.0,
        eager_params,
    })
}

// ─── VALIDATION ───

fn table_for(entity_name: &str) -> Result<String, SqlGenError> {
    let table = entity_to_table(entity_name);
    quote_ident(&table).ok_or(SqlGenError::InvalidIdentifier(table))
}

fn quote(name: &str) -> Result<String, SqlGenError> {
    quote_ident(name).ok_or_else(|| SqlGenError::InvalidIdentifier(name.to_string()))
}

fn require_field<'a>(entity: &'a Entity, field: &str) -> Result<&'a crate::ast::Field, SqlGenError> {
    entity.fields.get(field).ok_or_else(|| SqlGenError::UnknownField {
        entity: entity.name.clone(),
        field: field.to_string(),
    })
}

fn require_relation<'a>(
    entity: &'a Entity,
    relation: &str,
) -> Result<&'a crate::ast::Relation, SqlGenError> {
    entity.relations.get(relation).ok_or_else(|| SqlGenError::UnknownRelation {
        entity: entity.name.clone(),
        relation: relation.to_string(),
    })
}

fn require_entity<'a>(schema: &'a Schema, name: &str) -> Result<&'a Entity, SqlGenError> {
    schema.get_entity(name).ok_or_else(|| SqlGenError::UnknownEntity(name.to_string()))
}

/// Validate a filter path against the schema.
/// Supported shapes: `field` and `relation.field`.
fn validate_field_path(
    path: &FieldPath,
    entity: &Entity,
    schema: &Schema,
) -> Result<(), SqlGenError> {
    let rendered = path.segments.join(".");
    match path.segments.as_slice() {
        [field] => {
            require_field(entity, field)?;
            Ok(())
        }
        [relation, field] => {
            let rel = require_relation(entity, relation)?;
            let target = require_entity(schema, &rel.target_entity)?;
            require_field(target, field)?;
            Ok(())
        }
        [] => Err(SqlGenError::InvalidPath(rendered)),
        _ => {
            // Every segment must still exist; the depth is what is unsupported.
            let mut current = entity;
            for relation in &path.segments[..path.segments.len() - 1] {
                let rel = require_relation(current, relation)?;
                current = require_entity(schema, &rel.target_entity)?;
            }
            require_field(current, &path.segments[path.segments.len() - 1])?;
            Err(SqlGenError::UnsupportedPath(rendered))
        }
    }
}

fn validate_include(
    path: &[String],
    entity: &Entity,
    schema: &Schema,
) -> Result<(), SqlGenError> {
    if path.is_empty() {
        return Err(SqlGenError::InvalidPath(String::new()));
    }
    let mut current = entity;
    for relation in path {
        let rel = require_relation(current, relation)?;
        current = require_entity(schema, &rel.target_entity)?;
    }
    Ok(())
}

fn validate_filter(expr: &FilterExpr, entity: &Entity, schema: &Schema) -> Result<(), SqlGenError> {
    match expr {
        FilterExpr::Condition(cond) => validate_field_path(&cond.field, entity, schema),
        FilterExpr::Binary { left, right, .. } => {
            validate_filter(left, entity, schema)?;
            validate_filter(right, entity, schema)
        }
    }
}

fn validate_query(query: &Query, entity: &Entity, schema: &Schema) -> Result<(), SqlGenError> {
    for field in &query.select_fields {
        require_field(entity, field)?;
    }
    for filter in &query.filters {
        validate_filter(filter, entity, schema)?;
    }
    for clause in &query.order_by {
        require_field(entity, &clause.field)?;
    }
    for include in &query.includes {
        validate_include(&include.path, entity, schema)?;
    }
    Ok(())
}

// ─── MAIN QUERY ───

/// Extract filters that target relations (e.g., "orders.total")
fn extract_join_filters(query: &Query) -> Vec<&FilterExpr> {
    query.filters.iter()
        .filter(|f| filter_expr_is_nested(f))
        .collect()
}

/// Check if a filter expression references a nested path
fn filter_expr_is_nested(expr: &FilterExpr) -> bool {
    match expr {
        FilterExpr::Condition(cond) => cond.field.is_nested(),
        FilterExpr::Binary { left, right, .. } => {
            filter_expr_is_nested(left) || filter_expr_is_nested(right)
        }
    }
}

/// Build the main SELECT query
fn build_main_query(
    query: &Query,
    entity: &Entity,
    schema: &Schema,
    params: &mut Params,
) -> Result<String, SqlGenError> {
    let table_name = table_for(&query.entity)?;
    let join_filters = extract_join_filters(query);
    let needs_join = !join_filters.is_empty();

    let mut parts: Vec<String> = Vec::new();

    // SELECT
    let columns = build_select_columns(&table_name, entity, needs_join, &query.select_fields)?;
    let distinct = if needs_join { "DISTINCT " } else { "" };
    parts.push(format!("SELECT {}{}", distinct, columns));

    // FROM
    parts.push(format!("FROM {}", table_name));

    // JOINs (for relation filters)
    if needs_join {
        let joins = build_joins(entity, &join_filters, schema)?;
        parts.push(joins);
    }

    // WHERE
    let where_clause = build_where(&query.filters, &table_name, needs_join, entity, params)?;
    if !where_clause.is_empty() {
        parts.push(format!("WHERE {}", where_clause));
    }

    // ORDER BY
    if !query.order_by.is_empty() {
        parts.push(build_order_by(&query.order_by, &table_name, needs_join)?);
    }

    // LIMIT
    if let Some(limit) = query.limit {
        parts.push(format!("LIMIT {}", limit));
    }

    // OFFSET
    if let Some(offset) = query.offset {
        parts.push(format!("OFFSET {}", offset));
    }

    Ok(parts.join("\n"))
}

/// Build SELECT column list
/// If select_fields is empty, select all fields (SELECT *)
/// Otherwise, select only specified fields
fn build_select_columns(
    table_name: &str,
    entity: &Entity,
    qualify: bool,
    select_fields: &[String],
) -> Result<String, SqlGenError> {
    let names: Vec<&String> = if select_fields.is_empty() {
        entity.fields.keys().collect()
    } else {
        select_fields.iter().collect()
    };

    let columns = names.into_iter()
        .map(|name| {
            let column = quote(name)?;
            Ok(if qualify {
                format!("{}.{}", table_name, column)
            } else {
                column
            })
        })
        .collect::<Result<Vec<String>, SqlGenError>>()?;

    Ok(columns.join(", "))
}

/// Build JOIN clauses for relation filters
fn build_joins(
    entity: &Entity,
    join_filters: &[&FilterExpr],
    schema: &Schema,
) -> Result<String, SqlGenError> {
    let mut joins = Vec::new();
    let mut joined_relations: Vec<String> = Vec::new();

    for filter in join_filters {
        collect_join_relations(filter, &mut joined_relations);
    }

    // Deduplicate
    joined_relations.sort();
    joined_relations.dedup();

    for rel_name in &joined_relations {
        let relation = require_relation(entity, rel_name)?;
        require_entity(schema, &relation.target_entity)?;

        let target_table = table_for(&relation.target_entity)?;
        let source_table = table_for(&entity.name)?;

        let join = if relation.kind == RelationKind::BelongsTo {
            // The key lives on the source row: source.fk = target.id
            let fk = belongs_to_key(schema, entity, relation)?;
            format!(
                "INNER JOIN {} ON {}.\"id\" = {}.{}",
                target_table, target_table, source_table, quote(&fk)?
            )
        } else {
            let fk = relation.foreign_key.as_ref()
                .ok_or_else(|| SqlGenError::MissingForeignKey {
                    entity: entity.name.clone(),
                    relation: rel_name.clone(),
                })?;
            format!(
                "INNER JOIN {} ON {}.{} = {}.\"id\"",
                target_table, target_table, quote(fk)?, source_table
            )
        };
        joins.push(join);
    }

    Ok(joins.join("\n"))
}

/// Collect relation names from nested filter expressions
fn collect_join_relations(expr: &FilterExpr, relations: &mut Vec<String>) {
    match expr {
        FilterExpr::Condition(cond) => {
            if cond.field.is_nested() {
                relations.push(cond.field.root().to_string());
            }
        }
        FilterExpr::Binary { left, right, .. } => {
            collect_join_relations(left, relations);
            collect_join_relations(right, relations);
        }
    }
}

/// Build WHERE clause
fn build_where(
    filters: &[FilterExpr],
    table_name: &str,
    qualify: bool,
    entity: &Entity,
    params: &mut Params,
) -> Result<String, SqlGenError> {
    let conditions: Vec<String> = filters.iter()
        .map(|f| filter_expr_to_sql(f, table_name, qualify, entity, params))
        .collect::<Result<Vec<_>, _>>()?;

    Ok(conditions.join(" AND "))
}

/// Convert a FilterExpr to SQL
fn filter_expr_to_sql(
    expr: &FilterExpr,
    table_name: &str,
    qualify: bool,
    entity: &Entity,
    params: &mut Params,
) -> Result<String, SqlGenError> {
    match expr {
        FilterExpr::Condition(cond) => {
            condition_to_sql(cond, table_name, qualify, entity, params)
        }
        FilterExpr::Binary { left, op, right } => {
            let left_sql = filter_expr_to_sql(left, table_name, qualify, entity, params)?;
            let right_sql = filter_expr_to_sql(right, table_name, qualify, entity, params)?;
            let op_sql = match op {
                LogicalOp::And => "AND",
                LogicalOp::Or => "OR",
            };
            Ok(format!("({} {} {})", left_sql, op_sql, right_sql))
        }
    }
}

/// Convert a single condition to SQL
fn condition_to_sql(
    cond: &FilterCondition,
    table_name: &str,
    qualify: bool,
    entity: &Entity,
    params: &mut Params,
) -> Result<String, SqlGenError> {
    let field_sql = match cond.field.segments.as_slice() {
        [relation, field] => {
            // Nested: "orders.total" → "orders"."total"
            let relation = require_relation(entity, relation)?;
            let target_table = table_for(&relation.target_entity)?;
            format!("{}.{}", target_table, quote(field)?)
        }
        [field] if qualify => format!("{}.{}", table_name, quote(field)?),
        [field] => quote(field)?,
        _ => return Err(SqlGenError::UnsupportedPath(cond.field.segments.join("."))),
    };

    let value = filter_value_to_param(&cond.value);

    match (&cond.op, &cond.value) {
        (ComparisonOp::Like, FilterValue::String(s)) => {
            let placeholder = params.push(SqlParam::String(like_pattern(s)));
            Ok(format!("{} LIKE {} ESCAPE '{}'", field_sql, placeholder, LIKE_ESCAPE))
        }
        (ComparisonOp::Like, _) => Err(SqlGenError::UnsupportedOperator("Like".to_string())),
        (ComparisonOp::In, _) => Err(SqlGenError::UnsupportedOperator("In".to_string())),
        (op, _) => {
            let op_str = match op {
                ComparisonOp::Eq  => "=",
                ComparisonOp::Neq => "!=",
                ComparisonOp::Gt  => ">",
                ComparisonOp::Gte => ">=",
                ComparisonOp::Lt  => "<",
                ComparisonOp::Lte => "<=",
                ComparisonOp::Like | ComparisonOp::In => {
                    return Err(SqlGenError::UnsupportedOperator(format!("{:?}", op)))
                }
            };
            // Floats are bound as numeric so that comparing them with an
            // integer column keeps the exact value instead of being narrowed
            // to the column type.
            let cast = if matches!(value, SqlParam::Float(_)) { "::numeric" } else { "" };
            let placeholder = params.push(value);
            Ok(format!("{} {} {}{}", field_sql, op_str, placeholder, cast))
        }
    }
}

/// Convert a filter value into a bound parameter.
fn filter_value_to_param(value: &FilterValue) -> SqlParam {
    match value {
        FilterValue::String(s) => SqlParam::String(s.clone()),
        FilterValue::Int(n)    => SqlParam::Int(*n),
        FilterValue::Float(f)  => SqlParam::Float(*f),
        FilterValue::Bool(b)   => SqlParam::Bool(*b),
        FilterValue::Null      => SqlParam::Null,
    }
}

/// Build a "contains" pattern: wildcards and the escape character in the
/// value are escaped so they match literally.
pub(super) fn like_pattern(value: &str) -> String {
    let mut pattern = String::with_capacity(value.len() + 2);
    pattern.push('%');
    for ch in value.chars() {
        if ch == '%' || ch == '_' || ch == LIKE_ESCAPE {
            pattern.push(LIKE_ESCAPE);
        }
        pattern.push(ch);
    }
    pattern.push('%');
    pattern
}

/// Build ORDER BY clause
fn build_order_by(
    order_by: &[OrderByClause],
    table_name: &str,
    qualify: bool,
) -> Result<String, SqlGenError> {
    let clauses = order_by.iter()
        .map(|o| {
            let column = quote(&o.field)?;
            let field = if qualify {
                format!("{}.{}", table_name, column)
            } else {
                column
            };
            let dir = match o.direction {
                SortDirection::Asc  => "ASC",
                SortDirection::Desc => "DESC",
            };
            Ok(format!("{} {}", field, dir))
        })
        .collect::<Result<Vec<String>, SqlGenError>>()?;

    Ok(format!("ORDER BY {}", clauses.join(", ")))
}

// ─── EAGER QUERIES ───

type EagerQueries = (Vec<(String, String)>, Vec<Vec<SqlParam>>);

/// Build eager loading queries
fn build_eager_queries(
    root_entity: &str,
    includes: &[crate::query::IncludePath],
    schema: &Schema,
) -> Result<EagerQueries, SqlGenError> {
    let mut queries = Vec::new();
    let mut params = Vec::new();
    let mut processed: Vec<String> = Vec::new();

    for include in includes {
        build_eager_query_for_path(
            root_entity,
            &include.path,
            "",
            schema,
            &mut queries,
            &mut params,
            &mut processed,
        )?;
    }

    Ok((queries, params))
}

/// Build a single eager loading query for a given path
fn build_eager_query_for_path(
    current_entity: &str,
    path: &[String],
    parent_path: &str,
    schema: &Schema,
    queries: &mut Vec<(String, String)>,
    params: &mut Vec<Vec<SqlParam>>,
    processed: &mut Vec<String>,
) -> Result<(), SqlGenError> {
    if path.is_empty() {
        return Ok(());
    }

    let rel_name = &path[0];
    let full_path = if parent_path.is_empty() {
        rel_name.clone()
    } else {
        format!("{}.{}", parent_path, rel_name)
    };

    let entity = require_entity(schema, current_entity)?;
    let relation = require_relation(entity, rel_name)?;

    // Skip if already processed, but continue processing deeper paths
    if !processed.contains(&full_path) {
        let target_entity = require_entity(schema, &relation.target_entity)?;
        let target_table = table_for(&relation.target_entity)?;

        // Build columns
        let columns = target_entity.fields.keys()
            .map(|name| quote(name))
            .collect::<Result<Vec<String>, SqlGenError>>()?;

        // Parent ids are bound as one array parameter
        let sql = if relation.kind == RelationKind::BelongsTo {
            // The key lives on the parent rows, which the executor does not
            // expose, so select the targets through the parents' ids.
            let fk = belongs_to_key(schema, entity, relation)?;
            format!(
                "SELECT {}\nFROM {}\nWHERE \"id\" IN (SELECT {} FROM {} WHERE \"id\" = ANY($1))",
                columns.join(", "),
                target_table,
                quote(&fk)?,
                table_for(current_entity)?,
            )
        } else {
            let fk = relation.foreign_key.as_ref()
                .ok_or_else(|| SqlGenError::MissingForeignKey {
                    entity: current_entity.to_string(),
                    relation: rel_name.clone(),
                })?;
            format!(
                "SELECT {}\nFROM {}\nWHERE {} = ANY($1)",
                columns.join(", "),
                target_table,
                quote(fk)?,
            )
        };

        queries.push((full_path.clone(), sql));
        params.push(vec![SqlParam::ParentIds]);
        processed.push(full_path.clone());
    }

    // Process deeper paths
    if path.len() > 1 {
        build_eager_query_for_path(
            &relation.target_entity,
            &path[1..],
            &full_path,
            schema,
            queries,
            params,
            processed,
        )?;
    }

    Ok(())
}

/// Resolve the key column of a BelongsTo relation, as a generation error.
fn belongs_to_key(schema: &Schema, entity: &Entity, relation: &Relation) -> Result<String, SqlGenError> {
    schema.belongs_to_key(entity, relation).map_err(|e| match e {
        BelongsToKeyError::Missing => SqlGenError::MissingForeignKey {
            entity: entity.name.clone(),
            relation: relation.name.clone(),
        },
        BelongsToKeyError::Ambiguous(candidates) => SqlGenError::AmbiguousForeignKey {
            entity: entity.name.clone(),
            relation: relation.name.clone(),
            candidates,
        },
    })
}

/// Errors during SQL generation
#[derive(Debug, Clone, PartialEq)]
pub enum SqlGenError {
    UnknownEntity(String),
    UnknownRelation { entity: String, relation: String },
    UnknownField { entity: String, field: String },
    MissingForeignKey { entity: String, relation: String },
    AmbiguousForeignKey { entity: String, relation: String, candidates: Vec<String> },
    InvalidIdentifier(String),
    InvalidPath(String),
    UnsupportedPath(String),
    UnsupportedOperator(String),
}

impl std::fmt::Display for SqlGenError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SqlGenError::UnknownEntity(name) =>
                write!(f, "Unknown entity: {}", name),
            SqlGenError::UnknownRelation { entity, relation } =>
                write!(f, "Unknown relation '{}' in entity '{}'", relation, entity),
            SqlGenError::UnknownField { entity, field } =>
                write!(f, "Unknown field '{}' in entity '{}'", field, entity),
            SqlGenError::MissingForeignKey { entity, relation } =>
                write!(f, "Missing foreign key for relation '{}' in '{}'", relation, entity),
            SqlGenError::AmbiguousForeignKey { entity, relation, candidates } =>
                write!(f, "Ambiguous foreign key for relation '{}' in '{}' ({}): name the field '{}_id'",
                    relation, entity, candidates.join(", "), relation),
            SqlGenError::InvalidIdentifier(_) =>
                write!(f, "Invalid identifier"),
            SqlGenError::InvalidPath(path) =>
                write!(f, "Invalid field path '{}'", path),
            SqlGenError::UnsupportedPath(path) =>
                write!(f, "Unsupported field path '{}': only 'field' and 'relation.field' are supported", path),
            SqlGenError::UnsupportedOperator(op) =>
                write!(f, "Operator '{}' is not supported for this value", op),
        }
    }
}
