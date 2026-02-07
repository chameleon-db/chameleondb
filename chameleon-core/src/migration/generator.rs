use crate::ast::{Schema, Entity, RelationKind};
use crate::sql::naming::entity_to_table;
use super::type_map::{to_postgres_type, to_postgres_default};

/// Full migration output
#[derive(Debug, Clone, PartialEq)]
pub struct Migration {
    /// Complete DDL script ready to execute
    pub sql: String,
    /// Ordered list of (entity_name, CREATE TABLE statement)
    pub statements: Vec<(String, String)>,
}

/// Generate a full migration from a validated schema
pub fn generate_migration(schema: &Schema) -> Result<Migration, MigrationError> {
    // 1. Resolve creation order (dependencies first)
    let order = resolve_creation_order(schema)?;

    // 2. Generate CREATE TABLE for each entity in order
    let mut statements = Vec::new();
    for entity_name in &order {
        let entity = schema.get_entity(entity_name).unwrap();
        let sql = generate_create_table(entity, schema)?;
        statements.push((entity_name.clone(), sql));
    }

    // 3. Join into full script
    let sql = statements.iter()
        .map(|(_, stmt)| stmt.as_str())
        .collect::<Vec<&str>>()
        .join("\n\n");

    Ok(Migration { sql, statements })
}

/// Generate a single CREATE TABLE statement
fn generate_create_table(entity: &Entity, schema: &Schema) -> Result<String, MigrationError> {
    let table_name = entity_to_table(&entity.name);
    let mut columns = Vec::new();
    let mut constraints = Vec::new();

    // Columns (skip relation-only fields, keep actual data fields)
    for (_, field) in &entity.fields {
        let pg_type = to_postgres_type(&field.field_type);

        let mut col = format!("    {} {}", field.name, pg_type);

        // PRIMARY KEY
        if field.primary_key {
            col.push_str(" PRIMARY KEY");
        }

        // NOT NULL (skip if nullable or primary key already implies NOT NULL)
        if !field.nullable && !field.primary_key {
            col.push_str(" NOT NULL");
        }

        // UNIQUE
        if field.unique {
            col.push_str(" UNIQUE");
        }

        // DEFAULT
        if let Some(default) = &field.default {
            col.push_str(&format!(" DEFAULT {}", to_postgres_default(&default)));
        }

        columns.push(col);
    }

    // Foreign key constraints from HasMany relations in OTHER entities
    // that point TO this entity
    for other_entity in &schema.entities {
        if other_entity.name == entity.name {
            continue;
        }
        for (_, relation) in &other_entity.relations {
            if relation.target_entity == entity.name && relation.kind == RelationKind::HasMany {
                // The FK is in other_entity, not here — skip
            }
        }
    }

    // Foreign key constraints FROM this entity
    // Look at HasMany relations where THIS entity is the target
    // → that means another entity has a FK field pointing here
    // But actually, FKs are defined by fields like user_id in Order
    // We detect them by matching HasMany relations
    for other_entity in &schema.entities {
        for (_, relation) in &other_entity.relations {
            if relation.kind == RelationKind::HasMany
                && relation.target_entity == entity.name
            {
                // other_entity HasMany this entity via FK
                // The FK field is IN this entity
                if let Some(fk) = &relation.foreign_key {
                    let other_table = entity_to_table(&other_entity.name);
                    constraints.push(format!(
                        "    FOREIGN KEY ({}) REFERENCES {}(id)",
                        fk, other_table
                    ));
                }
            }
        }
    }

    // Build CREATE TABLE
    let mut all_parts = columns;
    all_parts.extend(constraints);

    Ok(format!(
        "CREATE TABLE {} (\n{}\n);",
        table_name,
        all_parts.join(",\n")
    ))
}

/// Resolve entity creation order using topological sort
/// Entities referenced by FKs must be created first
fn resolve_creation_order(schema: &Schema) -> Result<Vec<String>, MigrationError> {
    let mut order = Vec::new();
    let mut visited = Vec::new();
    let mut in_stack = Vec::new();

    for entity in &schema.entities {
        if !visited.contains(&entity.name) {
            topo_sort(schema, &entity.name, &mut order, &mut visited, &mut in_stack)?;
        }
    }

    Ok(order)
}

/// Topological sort via DFS
fn topo_sort(
    schema: &Schema,
    current: &str,
    order: &mut Vec<String>,
    visited: &mut Vec<String>,
    in_stack: &mut Vec<String>,
) -> Result<(), MigrationError> {
    if in_stack.contains(&current.to_string()) {
        return Err(MigrationError::CircularDependency(current.to_string()));
    }

    if visited.contains(&current.to_string()) {
        return Ok(());
    }

    in_stack.push(current.to_string());

    // Find all entities that have a HasMany relation pointing to this entity
    // Those are the dependencies - they must be created first
    for other_entity in &schema.entities {
        if other_entity.name == current {
            continue;
        }
        for (_, relation) in &other_entity.relations {
            if relation.kind == RelationKind::HasMany && relation.target_entity == current {
                // other_entity has HasMany to current, so other_entity must be created first
                topo_sort(schema, &other_entity.name, order, visited, in_stack)?;
            }
        }
    }

    in_stack.pop();
    visited.push(current.to_string());
    order.push(current.to_string());

    Ok(())
}

/// Migration errors
#[derive(Debug, Clone, PartialEq)]
pub enum MigrationError {
    CircularDependency(String),
    UnknownEntity(String),
}

impl std::fmt::Display for MigrationError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            MigrationError::CircularDependency(name) =>
                write!(f, "Circular dependency detected at '{}'", name),
            MigrationError::UnknownEntity(name) =>
                write!(f, "Unknown entity: '{}'", name),
        }
    }
}