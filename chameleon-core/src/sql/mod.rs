pub mod generator;
pub mod naming;

pub use generator::{generate_sql, GeneratedSQL, SqlGenError, SqlParam};

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ast::*;
    use crate::query::*;

    /// Helper: build a standard test schema
    /// User (HasMany) → Order (HasMany) → OrderItem
    fn test_schema() -> Schema {
        let mut schema = Schema::new();

        // User
        let mut user = Entity::new("User".to_string());
        user.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        user.add_field(Field {
            name: "email".to_string(),
            field_type: FieldType::String,
            nullable: false, unique: true, primary_key: false,
            default: None, backend: None,
        });
        user.add_field(Field {
            name: "name".to_string(),
            field_type: FieldType::String,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        user.add_field(Field {
            name: "age".to_string(),
            field_type: FieldType::Int,
            nullable: true, unique: false, primary_key: false,
            default: None, backend: None,
        });
        user.add_relation(Relation {
            name: "orders".to_string(),
            kind: RelationKind::HasMany,
            target_entity: "Order".to_string(),
            foreign_key: Some("user_id".to_string()),
            through: None,
        });
        schema.add_entity(user);

        // Order
        let mut order = Entity::new("Order".to_string());
        order.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        order.add_field(Field {
            name: "total".to_string(),
            field_type: FieldType::Decimal,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        order.add_field(Field {
            name: "status".to_string(),
            field_type: FieldType::String,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        order.add_field(Field {
            name: "user_id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        order.add_relation(Relation {
            name: "user".to_string(),
            kind: RelationKind::BelongsTo,
            target_entity: "User".to_string(),
            foreign_key: None,
            through: None,
        });
        order.add_relation(Relation {
            name: "items".to_string(),
            kind: RelationKind::HasMany,
            target_entity: "OrderItem".to_string(),
            foreign_key: Some("order_id".to_string()),
            through: None,
        });
        schema.add_entity(order);

        // OrderItem
        let mut item = Entity::new("OrderItem".to_string());
        item.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        item.add_field(Field {
            name: "quantity".to_string(),
            field_type: FieldType::Int,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        item.add_field(Field {
            name: "price".to_string(),
            field_type: FieldType::Decimal,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        item.add_field(Field {
            name: "order_id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        item.add_relation(Relation {
            name: "order".to_string(),
            kind: RelationKind::BelongsTo,
            target_entity: "Order".to_string(),
            foreign_key: None,
            through: None,
        });
        schema.add_entity(item);

        schema
    }

    // ─── NAMING ───

    #[test]
    fn test_naming_conventions() {
        assert_eq!(naming::entity_to_table("User"), "users");
        assert_eq!(naming::entity_to_table("Order"), "orders");
        assert_eq!(naming::entity_to_table("OrderItem"), "order_items");
    }

    // ─── SIMPLE QUERIES ───

    #[test]
    fn test_fetch_all() {
        let schema = test_schema();
        let query = Query::new("User");
        let result = generate_sql(&query, &schema).unwrap();

        assert!(result.main_query.contains("SELECT"));
        assert!(result.main_query.contains("FROM \"users\""));
        assert!(result.eager_queries.is_empty());
    }

    #[test]
    fn test_filter_equality() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "email", ComparisonOp::Eq,
                FilterValue::String("ana@mail.com".to_string()),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("WHERE \"email\" = $1"));
        assert_eq!(result.main_params, vec![SqlParam::String("ana@mail.com".to_string())]);
    }

    #[test]
    fn test_filter_comparison() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "age", ComparisonOp::Gte, FilterValue::Int(18),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("WHERE \"age\" >= $1"));
        assert_eq!(result.main_params, vec![SqlParam::Int(18)]);
    }

    #[test]
    fn test_multiple_filters() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Gte, FilterValue::Int(18)))
            .filter(FilterExpr::condition("age", ComparisonOp::Lte, FilterValue::Int(65)));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("\"age\" >= $1"));
        assert!(result.main_query.contains("\"age\" <= $2"));
        assert_eq!(result.main_params, vec![SqlParam::Int(18), SqlParam::Int(65)]);
        assert!(result.main_query.contains("AND"));
    }

    #[test]
    fn test_like_filter() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "name", ComparisonOp::Like,
                FilterValue::String("ana".to_string()),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("LIKE $1 ESCAPE '\\'"));
        assert_eq!(result.main_params, vec![SqlParam::String("%ana%".to_string())]);
    }

    // ─── RELATIONS ───

    #[test]
    fn test_filter_on_relation() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "orders.total", ComparisonOp::Gt, FilterValue::Int(100),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("DISTINCT"));
        assert!(result.main_query.contains("INNER JOIN \"orders\""));
        assert!(result.main_query.contains("\"orders\".\"total\" > $1"));
        assert_eq!(result.main_params, vec![SqlParam::Int(100)]);
    }

    #[test]
    fn test_include_single() {
        let schema = test_schema();
        let query = Query::new("User")
            .include("orders");

        let result = generate_sql(&query, &schema).unwrap();
        assert_eq!(result.eager_queries.len(), 1);
        assert_eq!(result.eager_queries[0].0, "orders");
        assert!(result.eager_queries[0].1.contains("FROM \"orders\""));
        assert!(result.eager_queries[0].1.contains("WHERE \"user_id\" = ANY($1)"));
        assert_eq!(result.eager_params, vec![vec![SqlParam::ParentIds]]);
    }

    #[test]
    fn test_include_nested() {
        let schema = test_schema();
        let query = Query::new("User")
            .include("orders")
            .include("orders.items");

        let result = generate_sql(&query, &schema).unwrap();
        assert_eq!(result.eager_queries.len(), 2);

        // First: orders
        assert_eq!(result.eager_queries[0].0, "orders");
        assert!(result.eager_queries[0].1.contains("FROM \"orders\""));

        // Second: orders.items
        assert_eq!(result.eager_queries[1].0, "orders.items");
        assert!(result.eager_queries[1].1.contains("FROM \"order_items\""));
        assert!(result.eager_queries[1].1.contains("WHERE \"order_id\" = ANY($1)"));
    }

    // ─── ORDER BY / LIMIT / OFFSET ───

    #[test]
    fn test_order_by() {
        let schema = test_schema();
        let query = Query::new("User")
            .order_by("name", SortDirection::Asc)
            .order_by("age", SortDirection::Desc);

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("ORDER BY \"name\" ASC, \"age\" DESC"));
    }

    #[test]
    fn test_limit_offset() {
        let schema = test_schema();
        let query = Query::new("User")
            .limit(10)
            .offset(20);

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("LIMIT 10"));
        assert!(result.main_query.contains("OFFSET 20"));
    }

    // ─── COMBINED ───

    #[test]
    fn test_full_query() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Gte, FilterValue::Int(18)))
            .filter(FilterExpr::condition("orders.total", ComparisonOp::Gt, FilterValue::Int(50)))
            .include("orders")
            .include("orders.items")
            .order_by("name", SortDirection::Desc)
            .limit(10);

        let result = generate_sql(&query, &schema).unwrap();

        // Main query
        assert!(result.main_query.contains("DISTINCT"));
        assert!(result.main_query.contains("INNER JOIN \"orders\""));
        assert!(result.main_query.contains("\"users\".\"age\" >= $1"));
        assert!(result.main_query.contains("\"orders\".\"total\" > $2"));
        assert!(result.main_query.contains("ORDER BY"));
        assert!(result.main_query.contains("LIMIT 10"));

        // Eager queries
        assert_eq!(result.eager_queries.len(), 2);
    }

    // ─── ERRORS ───

    #[test]
    fn test_unknown_entity() {
        let schema = test_schema();
        let query = Query::new("NonExistent");

        let result = generate_sql(&query, &schema);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), SqlGenError::UnknownEntity(_)));
    }

    // ─── VALUE HANDLING ───

    /// Value classes that must always travel as data.
    fn value_classes() -> Vec<&'static str> {
        vec![
            "o'brien",
            "''",
            "a'b'c",
            "x; y",
            ";",
            "a -- b",
            "/* c */",
            "back\\slash",
            "\\'",
            "caf\u{e9} \u{4e2d}\u{6587} \u{1f600}",
            "100%",
            "a_b",
            "%_%",
            "line\nbreak\ttab",
            "$1",
            "\"quoted\"",
        ]
    }

    /// SQL text with the fixed fragments (placeholders and the LIKE escape
    /// declaration) removed, so only variable content remains.
    fn without_fixed_fragments(sql: &str) -> String {
        let sql = sql.replace("ESCAPE '\\'", "ESCAPE");
        let mut out = String::new();
        let mut chars = sql.chars().peekable();
        while let Some(ch) = chars.next() {
            if ch == '$' && chars.peek().is_some_and(|c| c.is_ascii_digit()) {
                while chars.peek().is_some_and(|c| c.is_ascii_digit()) {
                    chars.next();
                }
            } else {
                out.push(ch);
            }
        }
        out
    }

    #[test]
    fn test_values_are_parameters_not_sql_text() {
        let schema = test_schema();
        for value in value_classes() {
            for op in [ComparisonOp::Eq, ComparisonOp::Neq, ComparisonOp::Gt, ComparisonOp::Like] {
                let query = Query::new("User")
                    .filter(FilterExpr::condition("name", op.clone(), FilterValue::String(value.to_string())))
                    .filter(FilterExpr::condition("orders.status", op.clone(), FilterValue::String(value.to_string())));
                let result = generate_sql(&query, &schema).unwrap();

                assert!(!without_fixed_fragments(&result.main_query).contains(value), "value in SQL text: {:?}", value);
                assert_eq!(result.main_params.len(), 2);
                let expected = if op == ComparisonOp::Like {
                    SqlParam::String(super::generator::like_pattern(value))
                } else {
                    SqlParam::String(value.to_string())
                };
                assert_eq!(result.main_params, vec![expected.clone(), expected]);
            }
        }
    }

    #[test]
    fn test_values_in_logical_expressions_are_parameters() {
        let schema = test_schema();
        for value in value_classes() {
            let expr = FilterExpr::condition("name", ComparisonOp::Eq, FilterValue::String(value.to_string()))
                .or(FilterExpr::condition("email", ComparisonOp::Neq, FilterValue::String(value.to_string())));
            let query = Query::new("User").filter(expr);
            let result = generate_sql(&query, &schema).unwrap();

            assert!(!without_fixed_fragments(&result.main_query).contains(value));
            assert!(result.main_query.contains("(\"name\" = $1 OR \"email\" != $2)"));
            assert_eq!(result.main_params, vec![
                SqlParam::String(value.to_string()),
                SqlParam::String(value.to_string()),
            ]);
        }
    }

    #[test]
    fn test_non_string_values_are_parameters() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Eq, FilterValue::Int(123456789)))
            .filter(FilterExpr::condition("age", ComparisonOp::Lt, FilterValue::Float(2.5)))
            .filter(FilterExpr::condition("age", ComparisonOp::Eq, FilterValue::Bool(true)))
            .filter(FilterExpr::condition("age", ComparisonOp::Eq, FilterValue::Null));
        let result = generate_sql(&query, &schema).unwrap();

        for literal in ["123456789", "2.5", "true", "NULL"] {
            assert!(!result.main_query.contains(literal), "{} in SQL text", literal);
        }
        assert_eq!(result.main_params, vec![
            SqlParam::Int(123456789),
            SqlParam::Float(2.5),
            SqlParam::Bool(true),
            SqlParam::Null,
        ]);
    }

    #[test]
    fn test_float_parameters_are_cast_to_numeric() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Lt, FilterValue::Float(2.5)))
            .filter(FilterExpr::condition("age", ComparisonOp::Gt, FilterValue::Int(1)));
        let result = generate_sql(&query, &schema).unwrap();

        assert!(result.main_query.contains("\"age\" < $1::numeric"));
        assert!(result.main_query.contains("\"age\" > $2"));
        assert!(!result.main_query.contains("2.5"));
        assert_eq!(result.main_params, vec![SqlParam::Float(2.5), SqlParam::Int(1)]);
    }

    #[test]
    fn test_like_escapes_wildcards_and_escape_character() {
        let schema = test_schema();
        let cases = [
            ("plain", "%plain%"),
            ("100%", "%100\\%%"),
            ("a_b", "%a\\_b%"),
            ("%_%", "%\\%\\_\\%%"),
            ("back\\slash", "%back\\\\slash%"),
            ("\\%", "%\\\\\\%%"),
        ];
        for (value, pattern) in cases {
            let query = Query::new("User").filter(FilterExpr::condition(
                "name", ComparisonOp::Like, FilterValue::String(value.to_string()),
            ));
            let result = generate_sql(&query, &schema).unwrap();

            assert!(result.main_query.contains("LIKE $1 ESCAPE '\\'"));
            assert_eq!(result.main_params, vec![SqlParam::String(pattern.to_string())]);
        }
    }

    #[test]
    fn test_like_requires_string_value() {
        let schema = test_schema();
        let query = Query::new("User").filter(FilterExpr::condition(
            "age", ComparisonOp::Like, FilterValue::Int(1),
        ));
        assert!(matches!(
            generate_sql(&query, &schema).unwrap_err(),
            SqlGenError::UnsupportedOperator(_)
        ));
    }

    #[test]
    fn test_in_operator_is_rejected() {
        let schema = test_schema();
        let query = Query::new("User").filter(FilterExpr::condition(
            "age", ComparisonOp::In, FilterValue::Int(1),
        ));
        assert!(matches!(
            generate_sql(&query, &schema).unwrap_err(),
            SqlGenError::UnsupportedOperator(_)
        ));
    }

    // ─── IDENTIFIER VALIDATION ───

    /// Names that are not in the schema, including ones that are not
    /// plausible identifiers at all.
    fn unknown_names() -> Vec<&'static str> {
        vec![
            "missing",
            "",
            "name x",
            "name\"",
            "na'me",
            "name;",
            "name--",
            "name/**/",
            "NAME",
            "caf\u{e9}",
            "name, email",
        ]
    }

    #[test]
    fn test_unknown_select_field_is_rejected() {
        let schema = test_schema();
        for name in unknown_names() {
            let query = Query::new("User").select(vec!["id", name]);
            assert!(generate_sql(&query, &schema).is_err(), "accepted select {:?}", name);
        }
        // Relations are not selectable columns.
        let query = Query::new("User").select(vec!["orders"]);
        assert!(generate_sql(&query, &schema).is_err());
    }

    #[test]
    fn test_unknown_filter_field_is_rejected() {
        let schema = test_schema();
        for name in unknown_names() {
            let query = Query::new("User").filter(FilterExpr::condition(
                name, ComparisonOp::Eq, FilterValue::Int(1),
            ));
            assert!(generate_sql(&query, &schema).is_err(), "accepted filter {:?}", name);
        }
    }

    #[test]
    fn test_unknown_filter_in_logical_expression_is_rejected() {
        let schema = test_schema();
        let expr = FilterExpr::condition("age", ComparisonOp::Eq, FilterValue::Int(1))
            .and(FilterExpr::condition("missing", ComparisonOp::Eq, FilterValue::Int(1)));
        let query = Query::new("User").filter(expr);
        assert!(generate_sql(&query, &schema).is_err());
    }

    #[test]
    fn test_unknown_relation_path_is_rejected() {
        let schema = test_schema();
        let paths = [
            "missing.total",
            "orders.missing",
            "orders.",
            ".total",
            "orders.total.extra",
            "orders.items.price",
            "orders.items.missing",
            "orders.missing.price",
            "name.total",
            "orders.\"total",
            "orders.total;",
        ];
        for path in paths {
            let query = Query::new("User").filter(FilterExpr::condition(
                path, ComparisonOp::Eq, FilterValue::Int(1),
            ));
            assert!(generate_sql(&query, &schema).is_err(), "accepted path {:?}", path);
        }
    }

    #[test]
    fn test_empty_filter_path_is_rejected() {
        let schema = test_schema();
        let query = Query::new("User").filter(FilterExpr::Condition(FilterCondition {
            field: FieldPath { segments: vec![] },
            op: ComparisonOp::Eq,
            value: FilterValue::Int(1),
        }));
        assert!(generate_sql(&query, &schema).is_err());
    }

    #[test]
    fn test_unknown_order_by_field_is_rejected() {
        let schema = test_schema();
        for name in unknown_names() {
            let query = Query::new("User").order_by(name, SortDirection::Asc);
            assert!(generate_sql(&query, &schema).is_err(), "accepted order_by {:?}", name);
        }
        let query = Query::new("User").order_by("orders.total", SortDirection::Asc);
        assert!(generate_sql(&query, &schema).is_err());
    }

    #[test]
    fn test_unknown_include_is_rejected() {
        let schema = test_schema();
        let paths = [
            "missing",
            "orders.missing",
            "missing.items",
            "name",
            "orders.",
            "",
            "orders.items.missing",
            "orders;",
        ];
        for path in paths {
            let query = Query::new("User").include(path);
            assert!(generate_sql(&query, &schema).is_err(), "accepted include {:?}", path);
        }
    }

    #[test]
    fn test_rejected_input_produces_no_sql() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Eq, FilterValue::Int(1)))
            .include("orders")
            .order_by("missing", SortDirection::Asc);
        assert!(generate_sql(&query, &schema).is_err());
    }

    #[test]
    fn test_valid_identifiers_still_work() {
        let schema = test_schema();
        let query = Query::new("User")
            .select(vec!["id", "email"])
            .filter(FilterExpr::condition("age", ComparisonOp::Gte, FilterValue::Int(18)))
            .filter(FilterExpr::condition("orders.status", ComparisonOp::Eq, FilterValue::String("open".to_string())))
            .order_by("email", SortDirection::Desc)
            .include("orders")
            .include("orders.items")
            .limit(5)
            .offset(10);
        let result = generate_sql(&query, &schema).unwrap();

        assert!(result.main_query.contains("SELECT DISTINCT \"users\".\"id\", \"users\".\"email\""));
        assert!(result.main_query.contains("INNER JOIN \"orders\" ON \"orders\".\"user_id\" = \"users\".\"id\""));
        assert!(result.main_query.contains("\"orders\".\"status\" = $2"));
        assert!(result.main_query.contains("ORDER BY \"users\".\"email\" DESC"));
        assert!(result.main_query.contains("LIMIT 5"));
        assert!(result.main_query.contains("OFFSET 10"));
        assert_eq!(result.eager_queries.len(), 2);
        assert_eq!(result.eager_params.len(), 2);
    }

    #[test]
    fn test_every_eager_query_has_parent_ids_parameter() {
        let schema = test_schema();
        let query = Query::new("User").include("orders.items");
        let result = generate_sql(&query, &schema).unwrap();

        assert_eq!(result.eager_queries.len(), 2);
        assert_eq!(result.eager_params, vec![vec![SqlParam::ParentIds], vec![SqlParam::ParentIds]]);
    }

    // ─── JSON CONTRACT ───

    #[test]
    fn test_generated_sql_json_without_parameter_lists_still_deserializes() {
        let json = r#"{"main_query":"SELECT 1","eager_queries":[["orders","SELECT 2"]]}"#;
        let parsed: GeneratedSQL = serde_json::from_str(json).unwrap();

        assert_eq!(parsed.main_query, "SELECT 1");
        assert_eq!(parsed.eager_queries.len(), 1);
        assert!(parsed.main_params.is_empty());
        assert!(parsed.eager_params.is_empty());
    }

    #[test]
    fn test_generated_sql_json_round_trip() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Gt, FilterValue::Int(1)))
            .filter(FilterExpr::condition("age", ComparisonOp::Eq, FilterValue::Null))
            .include("orders");
        let result = generate_sql(&query, &schema).unwrap();

        let json = serde_json::to_string(&result).unwrap();
        let restored: GeneratedSQL = serde_json::from_str(&json).unwrap();
        assert_eq!(result, restored);
    }
}