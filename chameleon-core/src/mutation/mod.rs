use crate::ast::Schema;
use crate::sql::naming::quote_ident;
use serde_json::{json, Value};

/// Check that a field name exists in the entity and render it as a quoted
/// identifier.
fn quoted_field(entity: &crate::ast::Entity, field_name: &str) -> Result<String, String> {
    if !entity.fields.contains_key(field_name) {
        return Err(format!("Unknown field '{}' in entity '{}'", field_name, entity.name));
    }
    quote_ident(field_name).ok_or_else(|| "Invalid identifier".to_string())
}

/// Validate every key of a JSON object against the entity.
/// Returns (name, quoted name) pairs.
fn quoted_keys(
    entity: &crate::ast::Entity,
    object: &serde_json::Map<String, Value>,
) -> Result<Vec<(String, String)>, String> {
    object
        .keys()
        .map(|k| quoted_field(entity, k).map(|q| (k.clone(), q)))
        .collect()
}

/// Generate SQL for a mutation operation
pub fn generate_mutation_sql(mutation: &Value, schema: &Schema) -> Value {
    let mutation_type = match mutation.get("type").and_then(|v| v.as_str()) {
        Some(t) => t,
        None => return error_response("Missing 'type' field in mutation"),
    };

    let entity_name = match mutation.get("entity").and_then(|v| v.as_str()) {
        Some(e) => e,
        None => return error_response("Missing 'entity' field in mutation"),
    };

    let entity = match schema.get_entity(entity_name) {
        Some(e) => e,
        None => return error_response(&format!("Entity '{}' not found in schema", entity_name)),
    };

    match mutation_type {
        "insert" => generate_insert_sql(mutation, entity),
        "update" => generate_update_sql(mutation, entity),
        "delete" => generate_delete_sql(mutation, entity),
        _ => error_response(&format!("Unknown mutation type: '{}'", mutation_type)),
    }
}

/// Generate INSERT SQL
fn generate_insert_sql(mutation: &Value, entity: &crate::ast::Entity) -> Value {
    let fields = match mutation.get("fields").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'fields' in INSERT mutation"),
    };

    let quoted = match quoted_keys(entity, fields) {
        Ok(q) => q,
        Err(e) => return error_response(&e),
    };

    let mut columns = Vec::new();
    let mut placeholders = Vec::new();
    let mut params = Vec::new();

    let mut param_index = 1;
    for (field_name, column) in quoted {
        columns.push(column);
        placeholders.push(format!("${}", param_index));
        params.push(field_name);
        param_index += 1;
    }

    let table_name = entity_to_table(&entity.name);
    let sql = format!(
        "INSERT INTO {} ({}) VALUES ({}) RETURNING *",
        table_name,
        columns.join(", "),
        placeholders.join(", ")
    );

    json!({
        "valid": true,
        "sql": sql,
        "params": params,
        "affected": 1
    })
}

/// Generate UPDATE SQL
fn generate_update_sql(mutation: &Value, entity: &crate::ast::Entity) -> Value {
    let fields = match mutation.get("fields").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'fields' in UPDATE mutation"),
    };

    let filters = match mutation.get("filters").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'filters' in UPDATE mutation"),
    };

    if filters.is_empty() {
        return error_response("UPDATE requires at least one filter (WHERE clause)");
    }

    let quoted_fields = match quoted_keys(entity, fields) {
        Ok(q) => q,
        Err(e) => return error_response(&e),
    };
    for (field_name, _) in &quoted_fields {
        if let Some(field) = entity.fields.get(field_name) {
            if field.primary_key {
                return error_response("Cannot update primary key");
            }
        }
    }

    let quoted_filters = match quoted_keys(entity, filters) {
        Ok(q) => q,
        Err(e) => return error_response(&e),
    };

    let mut set_clauses = Vec::new();
    let mut params = Vec::new();
    let mut param_index = 1;

    for (field_name, column) in quoted_fields {
        set_clauses.push(format!("{}=${}", column, param_index));
        params.push(field_name);
        param_index += 1;
    }

    let mut where_parts = Vec::new();
    for (filter_field, column) in quoted_filters {
        where_parts.push(format!("{}=${}", column, param_index));
        params.push(filter_field);
        param_index += 1;
    }

    let table_name = entity_to_table(&entity.name);
    let sql = format!(
        "UPDATE {} SET {} WHERE {} RETURNING *",
        table_name,
        set_clauses.join(", "),
        where_parts.join(" AND ")
    );

    json!({
        "valid": true,
        "sql": sql,
        "params": params,
        "affected": 1
    })
}

/// Generate DELETE SQL
fn generate_delete_sql(mutation: &Value, entity: &crate::ast::Entity) -> Value {
    let filters = match mutation.get("filters").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'filters' in DELETE mutation"),
    };

    if filters.is_empty() {
        return error_response("DELETE requires at least one filter (WHERE clause) for safety");
    }

    let quoted_filters = match quoted_keys(entity, filters) {
        Ok(q) => q,
        Err(e) => return error_response(&e),
    };

    let mut where_parts = Vec::new();
    let mut params = Vec::new();
    let mut param_index = 1;

    for (filter_field, column) in quoted_filters {
        where_parts.push(format!("{}=${}", column, param_index));
        params.push(filter_field);
        param_index += 1;
    }

    let table_name = entity_to_table(&entity.name);
    let sql = format!(
        "DELETE FROM {} WHERE {}",
        table_name,
        where_parts.join(" AND ")
    );

    json!({
        "valid": true,
        "sql": sql,
        "params": params,
        "affected": 1
    })
}

/// Helper: Convert entity name to table name
fn entity_to_table(entity_name: &str) -> String {
    let mut table_name = String::new();
    for (i, ch) in entity_name.chars().enumerate() {
        if i > 0 && ch.is_uppercase() {
            table_name.push('_');
        }
        table_name.push(ch.to_lowercase().next().unwrap());
    }
    table_name
}

/// Helper: Build error response JSON
fn error_response(message: &str) -> Value {
    json!({
        "valid": false,
        "error": message
    })
}

// === Tests ===

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ast::{Entity, Field, FieldType};

    fn test_schema() -> Schema {
        let mut schema = Schema::new();

        let mut user = Entity::new("User".to_string());
        user.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false,
            unique: false,
            primary_key: true,
            default: None,
            backend: None,
        });
        user.add_field(Field {
            name: "email".to_string(),
            field_type: FieldType::String,
            nullable: false,
            unique: true,
            primary_key: false,
            default: None,
            backend: None,
        });
        user.add_field(Field {
            name: "name".to_string(),
            field_type: FieldType::String,
            nullable: false,
            unique: false,
            primary_key: false,
            default: None,
            backend: None,
        });
        user.add_field(Field {
            name: "age".to_string(),
            field_type: FieldType::Int,
            nullable: true,
            unique: false,
            primary_key: false,
            default: None,
            backend: None,
        });
        schema.add_entity(user);

        schema
    }

    // ============================================================
    // HELPER TESTS
    // ============================================================

    #[test]
    fn test_entity_to_table() {
        assert_eq!(entity_to_table("User"), "user");
        assert_eq!(entity_to_table("UserProfile"), "user_profile");
        assert_eq!(entity_to_table("OrderItem"), "order_item");
        assert_eq!(entity_to_table("HTTPServer"), "h_t_t_p_server");
    }

    // ============================================================
    // INSERT TESTS
    // ============================================================

#[test]
fn test_insert_single_field() {
    let schema = test_schema();
    let mutation = serde_json::json!({
        "type": "insert",
        "entity": "User",
        "fields": {
            "email": "ana@mail.com"
        }
    });

    let result = generate_mutation_sql(&mutation, &schema);

    println!("DEBUG: Full result = {}", serde_json::to_string_pretty(&result).unwrap());
    println!("DEBUG: SQL value = {:?}", result["sql"]);

    assert_eq!(result["valid"], true);
    assert!(result["sql"].as_str().unwrap().contains("INSERT INTO user"));
    assert!(result["sql"].as_str().unwrap().contains("email"));
    assert_eq!(result["params"][0], "email");
}

    #[test]
    fn test_insert_multiple_fields() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "insert",
            "entity": "User",
            "fields": {
                "email": "ana@mail.com",
                "name": "Ana García",
                "age": 28
            }
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true);
        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("INSERT INTO user"));
        assert!(sql.contains("RETURNING *"));
        assert_eq!(result["params"].as_array().unwrap().len(), 3);
    }

    #[test]
    fn test_insert_missing_entity() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "insert",
            "entity": "NonExistent",
            "fields": {"field": "value"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("not found"));
    }

    #[test]
    fn test_insert_unknown_field() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "insert",
            "entity": "User",
            "fields": {
                "email": "ana@mail.com",
                "unknown_field": "value"
            }
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("Unknown field"));
    }

    #[test]
    fn test_insert_missing_fields() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "insert",
            "entity": "User",
            "fields": {}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true); // SQL generated, DB will enforce NOT NULL
        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("INSERT INTO user"));
    }

    // ============================================================
    // UPDATE TESTS
    // ============================================================

    #[test]
    fn test_update_with_filter() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "update",
            "entity": "User",
            "filters": {"id": "uuid-123"},
            "fields": {"name": "Ana María"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true);
        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("UPDATE user"));
        assert!(sql.contains("SET"));
        assert!(sql.contains("WHERE"));
        assert!(sql.contains("RETURNING *"));
    }

    #[test]
    fn test_update_multiple_fields_single_filter() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "update",
            "entity": "User",
            "filters": {"id": "uuid-123"},
            "fields": {
                "name": "Ana",
                "age": 30
            }
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true);
        assert_eq!(result["params"].as_array().unwrap().len(), 3); // 2 SET + 1 WHERE
    }

    #[test]
    fn test_update_multiple_filters() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "update",
            "entity": "User",
            "filters": {
                "id": "uuid-123",
                "email": "old@mail.com"
            },
            "fields": {"name": "Ana"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true);
        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("WHERE"));
        assert!(sql.contains("AND"));
    }

    #[test]
    fn test_update_without_filter() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "update",
            "entity": "User",
            "filters": {},
            "fields": {"name": "Ana"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("at least one filter"));
    }

    #[test]
    fn test_update_primary_key() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "update",
            "entity": "User",
            "filters": {"id": "uuid-123"},
            "fields": {"id": "new-uuid"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("primary key"));
    }

    #[test]
    fn test_update_unknown_field() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "update",
            "entity": "User",
            "filters": {"id": "uuid-123"},
            "fields": {"unknown": "value"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("Unknown field"));
    }

    // ============================================================
    // DELETE TESTS
    // ============================================================

    #[test]
    fn test_delete_with_filter() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "delete",
            "entity": "User",
            "filters": {"id": "uuid-123"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true);
        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("DELETE FROM user"));
        assert!(sql.contains("WHERE"));
    }

    #[test]
    fn test_delete_multiple_filters() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "delete",
            "entity": "User",
            "filters": {
                "id": "uuid-123",
                "email": "ana@mail.com"
            }
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true);
        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("AND"));
    }

    #[test]
    fn test_delete_without_filter() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "delete",
            "entity": "User",
            "filters": {}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("at least one filter"));
    }

    #[test]
    fn test_delete_safety_guard() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "delete",
            "entity": "User"
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("filter"));
    }

    // ============================================================
    // ERROR HANDLING TESTS
    // ============================================================

    #[test]
    fn test_missing_mutation_type() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "entity": "User",
            "fields": {}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("type"));
    }

    #[test]
    fn test_unknown_mutation_type() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "upsert",
            "entity": "User",
            "fields": {}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("Unknown mutation type"));
    }

    #[test]
    fn test_missing_entity() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "insert",
            "fields": {}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], false);
        assert!(result["error"].as_str().unwrap().contains("entity"));
    }

    // ============================================================
    // SQL GENERATION TESTS
    // ============================================================

    #[test]
    fn test_insert_sql_format() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "insert",
            "entity": "User",
            "fields": {"email": "test@mail.com", "name": "Test"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        let sql = result["sql"].as_str().unwrap();
        // Should use parameterized queries
        assert!(sql.contains("$1") || sql.contains("$2"));
        // Should quote field names
        assert!(sql.contains("\"email\"") || sql.contains("\"name\""));
    }

    #[test]
    fn test_update_sql_format() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "update",
            "entity": "User",
            "filters": {"id": "uuid"},
            "fields": {"name": "Test"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("UPDATE"));
        assert!(sql.contains("SET"));
        assert!(sql.contains("WHERE"));
        assert!(sql.contains("$"));
    }

    #[test]
    fn test_delete_sql_format() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "delete",
            "entity": "User",
            "filters": {"id": "uuid"}
        });

        let result = generate_mutation_sql(&mutation, &schema);

        let sql = result["sql"].as_str().unwrap();
        assert!(sql.contains("DELETE FROM"));
        assert!(sql.contains("WHERE"));
        assert!(sql.contains("$"));
    }

    // ============================================================
    // IDENTIFIER VALIDATION TESTS
    // ============================================================

    fn unknown_keys() -> Vec<&'static str> {
        vec![
            "missing",
            "",
            "id x",
            "id\"",
            "i'd",
            "id;",
            "id--",
            "id/**/",
            "ID",
            "id=id OR id",
            "caf\u{e9}",
        ]
    }

    #[test]
    fn test_update_unknown_filter_key_is_rejected() {
        let schema = test_schema();
        for key in unknown_keys() {
            let mutation = serde_json::json!({
                "type": "update",
                "entity": "User",
                "filters": { key: "x" },
                "fields": {"name": "Ana"}
            });
            let result = generate_mutation_sql(&mutation, &schema);
            assert_eq!(result["valid"], false, "accepted filter key {:?}", key);
            assert!(result.get("sql").is_none());
        }
    }

    #[test]
    fn test_delete_unknown_filter_key_is_rejected() {
        let schema = test_schema();
        for key in unknown_keys() {
            let mutation = serde_json::json!({
                "type": "delete",
                "entity": "User",
                "filters": { key: "x" }
            });
            let result = generate_mutation_sql(&mutation, &schema);
            assert_eq!(result["valid"], false, "accepted filter key {:?}", key);
            assert!(result.get("sql").is_none());
        }
    }

    #[test]
    fn test_unknown_key_among_valid_filters_is_rejected() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "delete",
            "entity": "User",
            "filters": {"id": "x", "missing": "y"}
        });
        assert_eq!(generate_mutation_sql(&mutation, &schema)["valid"], false);
    }

    #[test]
    fn test_insert_and_update_unknown_field_keys_are_rejected() {
        let schema = test_schema();
        for key in unknown_keys() {
            let insert = serde_json::json!({
                "type": "insert", "entity": "User", "fields": { key: "x" }
            });
            assert_eq!(generate_mutation_sql(&insert, &schema)["valid"], false, "insert {:?}", key);

            let update = serde_json::json!({
                "type": "update", "entity": "User",
                "filters": {"id": "x"}, "fields": { key: "x" }
            });
            assert_eq!(generate_mutation_sql(&update, &schema)["valid"], false, "update {:?}", key);
        }
    }

    #[test]
    fn test_valid_filter_keys_are_quoted() {
        let schema = test_schema();
        let mutation = serde_json::json!({
            "type": "delete",
            "entity": "User",
            "filters": {"email": "a"}
        });
        let result = generate_mutation_sql(&mutation, &schema);

        assert_eq!(result["valid"], true);
        assert!(result["sql"].as_str().unwrap().contains("WHERE \"email\"=$1"));
    }
}