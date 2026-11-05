pub mod errors;
mod relations;
mod constraints;

use crate::ast::Schema;
use errors::TypeCheckError;
use std::collections::HashMap;

/// Result of type checking a schema
#[derive(Debug, Clone)]
pub struct TypeCheckResult {
    pub errors: Vec<TypeCheckError>,
}

impl TypeCheckResult {
    /// Returns true if the schema passed all checks
    pub fn is_valid(&self) -> bool {
        self.errors.is_empty()
    }

    /// Returns a formatted error report
    pub fn error_report(&self) -> String {
        if self.is_valid() {
            return "✅ Schema is valid".to_string();
        }

        let mut report = format!("❌ Found {} error(s):\n\n", self.errors.len());
        for (i, error) in self.errors.iter().enumerate() {
            report.push_str(&format!("  {}. {}\n", i + 1, error));
        }
        report
    }
}

/// Run all type checks on a schema
pub fn type_check(schema: &Schema) -> TypeCheckResult {
    let mut errors: Vec<TypeCheckError> = Vec::new();
    let mut entity_names: HashMap<String, Vec<usize>> = HashMap::new();

    // ===== ENTITIES =====
    // Rastrear índices de aparición
    for (i, entity) in schema.entities.iter().enumerate() {
        entity_names
            .entry(entity.name.clone())
            .or_insert_with(Vec::new)
            .push(i);
    }

    // Detectar duplicados
    for (entity_name, indices) in entity_names.iter() {
        if indices.len() > 1 {
            // Hay duplicados
            for (pos, &idx) in indices.iter().enumerate() {
                if pos > 0 {  // Solo reportar a partir del segundo
                    errors.push(TypeCheckError::DuplicateEntity {
                        entity: entity_name.clone(),
                        first: indices[0] + 1,
                        second: idx + 1,
                    });
                }
            }
        }
    }

    // ===== FIELDS =====
    for entity in &schema.entities {
        let mut field_names: HashMap<String, Vec<usize>> = HashMap::new();
        
        for (i, field) in entity.fields.values().enumerate() {
            field_names
                .entry(field.name.clone())
                .or_insert_with(Vec::new)
                .push(i);
        }

        for (field_name, indices) in field_names.iter() {
            if indices.len() > 1 {
                errors.push(TypeCheckError::DuplicateField {
                    entity: entity.name.clone(),
                    field: field_name.clone(),
                });
            }
        }
    }

    // Relations
    errors.extend(relations::check_relations(schema));
    errors.extend(relations::check_circular_dependencies(schema));

    // Constraints
    errors.extend(constraints::check_primary_keys(schema));
    errors.extend(constraints::check_field_modifiers(schema));
    errors.extend(constraints::check_annotations(schema));
    errors.extend(constraints::check_defaults(schema));

    TypeCheckResult { errors }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ast::*;
    use std::collections::HashMap;

    // Helper para construir schemas en tests
    fn build_schema(entities: Vec<(&str, Vec<(&str, FieldType, bool, bool, Option<BackendAnnotation>)>, Vec<(&str, RelationKind, &str, Option<&str>)>)>) -> Schema {
        let mut schema = Schema::new();

        for (name, fields, relations) in entities {
            let mut entity = Entity::new(name.to_string());

            for (fname, ftype, primary, unique, annotation) in fields {
                entity.add_field(Field {
                    name: fname.to_string(),
                    field_type: ftype,
                    nullable: false,
                    unique,
                    primary_key: primary,
                    default: None,
                    backend: annotation,
                });
            }

            for (rname, kind, target, fk) in relations {
                entity.add_relation(Relation {
                    name: rname.to_string(),
                    kind,
                    target_entity: target.to_string(),
                    foreign_key: fk.map(|s| s.to_string()),
                    through: None,
                });
            }

            schema.add_entity(entity);
        }

        schema
    }

    // ─── VALID SCHEMAS ───

    #[test]
    fn test_valid_simple_schema() {
        let schema = build_schema(vec![
            ("User", 
                vec![("id", FieldType::UUID, true, false, None),
                     ("email", FieldType::String, false, true, None)],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(result.is_valid(), "{}", result.error_report());
    }

    #[test]
    fn test_valid_schema_with_relations() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, true, false, None),
                     ("email", FieldType::String, false, true, None)],
                vec![("orders", RelationKind::HasMany, "Order", Some("user_id"))]),
            ("Order",
                vec![("id", FieldType::UUID, true, false, None),
                     ("user_id", FieldType::UUID, false, false, None),
                     ("total", FieldType::Decimal, false, false, None)],
                vec![("user", RelationKind::BelongsTo, "User", None)]),
        ]);

        let result = type_check(&schema);
        assert!(result.is_valid(), "{}", result.error_report());
    }

    #[test]
    fn test_valid_annotations() {
        let schema = build_schema(vec![
            ("Product",
                vec![("id", FieldType::UUID, true, false, None),
                     ("views", FieldType::Int, false, false, Some(BackendAnnotation::Cache)),
                     ("sales", FieldType::Decimal, false, false, Some(BackendAnnotation::OLAP)),
                     ("embedding", FieldType::Vector(384), false, false, Some(BackendAnnotation::Vector))],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(result.is_valid(), "{}", result.error_report());
    }

    // ─── RELATION ERRORS ───

    #[test]
    fn test_unknown_relation_target() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, true, false, None)],
                vec![("orders", RelationKind::HasMany, "NonExistent", Some("user_id"))]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::UnknownRelationTarget { .. })));
    }

    #[test]
    fn test_invalid_foreign_key() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, true, false, None)],
                vec![("orders", RelationKind::HasMany, "Order", Some("wrong_fk"))]),
            ("Order",
                vec![("id", FieldType::UUID, true, false, None),
                     ("user_id", FieldType::UUID, false, false, None)],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::InvalidForeignKey { .. })));
    }

    #[test]
    fn test_has_many_missing_foreign_key() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, true, false, None)],
                vec![("orders", RelationKind::HasMany, "Order", None)]),
            ("Order",
                vec![("id", FieldType::UUID, true, false, None)],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::MissingForeignKey { .. })));
    }

    // ─── PRIMARY KEY ERRORS ───

    #[test]
    fn test_missing_primary_key() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, false, false, None),
                     ("email", FieldType::String, false, true, None)],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::MissingPrimaryKey { .. })));
    }

    #[test]
    fn test_multiple_primary_keys() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, true, false, None),
                     ("email", FieldType::String, true, false, None)],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::MultiplePrimaryKeys { .. })));
    }

    #[test]
    fn test_valid_default_values() {
        let mut schema = Schema::new();
        let mut entity = Entity::new("User".to_string());

        entity.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false,
            unique: false,
            primary_key: true,
            default: Some(DefaultValue::UUIDv4),
            backend: None,
        });

        entity.add_field(Field {
            name: "verified".to_string(),
            field_type: FieldType::Bool,
            nullable: false,
            unique: false,
            primary_key: false,
            default: Some(DefaultValue::BoolLiteral(false)),
            backend: None,
        });

        entity.add_field(Field {
            name: "created_at".to_string(),
            field_type: FieldType::Timestamp,
            nullable: false,
            unique: false,
            primary_key: false,
            default: Some(DefaultValue::Now),
            backend: None,
        });

        schema.add_entity(entity);

        let result = type_check(&schema);
        assert!(result.is_valid(), "{}", result.error_report());
    }

    #[test]
    fn test_invalid_default_value_type_mismatch() {
        let mut schema = Schema::new();
        let mut entity = Entity::new("User".to_string());

        entity.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false,
            unique: false,
            primary_key: true,
            default: None,
            backend: None,
        });

        entity.add_field(Field {
            name: "verified".to_string(),
            field_type: FieldType::Bool,
            nullable: false,
            unique: false,
            primary_key: false,
            default: Some(DefaultValue::StringLiteral("yes".to_string())),
            backend: None,
        });

        schema.add_entity(entity);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::InvalidDefaultValue { .. })));
    }

    #[test]
    fn test_invalid_default_function_for_field_type() {
        let mut schema = Schema::new();
        let mut entity = Entity::new("User".to_string());

        entity.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false,
            unique: false,
            primary_key: true,
            default: None,
            backend: None,
        });

        entity.add_field(Field {
            name: "is_active".to_string(),
            field_type: FieldType::Bool,
            nullable: false,
            unique: false,
            primary_key: false,
            default: Some(DefaultValue::Now),
            backend: None,
        });

        schema.add_entity(entity);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::UnsupportedDefaultFunction { .. })));
    }

    // ─── ANNOTATION ERRORS ───

    #[test]
    fn test_vector_annotation_on_wrong_type() {
        let schema = build_schema(vec![
            ("Product",
                vec![("id", FieldType::UUID, true, false, None),
                     ("embedding", FieldType::String, false, false, Some(BackendAnnotation::Vector))],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::InvalidVectorAnnotation { .. })));
    }

    #[test]
    fn test_annotation_on_primary_key() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, true, false, Some(BackendAnnotation::Cache))],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::AnnotationOnConstrainedField { .. })));
    }

    #[test]
    fn test_annotation_on_unique_field() {
        let schema = build_schema(vec![
            ("User",
                vec![("id", FieldType::UUID, true, false, None),
                     ("email", FieldType::String, false, true, Some(BackendAnnotation::Cache))],
                vec![]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::AnnotationOnConstrainedField { .. })));
    }

    // ─── CIRCULAR DEPENDENCY ───

    #[test]
    fn test_circular_dependency() {
        let schema = build_schema(vec![
            ("A",
                vec![("id", FieldType::UUID, true, false, None)],
                vec![("b", RelationKind::HasOne, "B", None)]),
            ("B",
                vec![("id", FieldType::UUID, true, false, None)],
                vec![("c", RelationKind::HasOne, "C", None)]),
            ("C",
                vec![("id", FieldType::UUID, true, false, None)],
                vec![("a", RelationKind::HasOne, "A", None)]),
        ]);

        let result = type_check(&schema);
        assert!(!result.is_valid());
        assert!(result.errors.iter().any(|e| matches!(e, TypeCheckError::CircularDependency { .. })));
    }

    // ─── ERROR REPORT ───

    #[test]
    fn test_error_report_format() {
        let schema = build_schema(vec![
            ("User",
                vec![("email", FieldType::String, false, false, None)], // No primary key
                vec![("orders", RelationKind::HasMany, "NonExistent", Some("user_id"))]), // Unknown target
        ]);

        let result = type_check(&schema);
        let report = result.error_report();

        assert!(report.contains("❌"));
        assert!(report.contains("2"));  // 2 errors
        println!("{}", report);
    }

    // ─── SCHEMA CHECKS FROM PARSED SOURCE ───

    fn check_source(src: &str) -> Vec<TypeCheckError> {
        let schema = crate::parse_schema(src).expect("schema should parse");
        type_check(&schema).errors
    }

    #[test]
    fn test_primary_key_cannot_be_nullable() {
        let errors = check_source("entity U {\n    id: uuid primary nullable,\n}\n");
        assert!(errors.iter().any(|e| matches!(e, TypeCheckError::NullablePrimaryKey { .. })));
    }

    #[test]
    fn test_vector_dimension_zero_is_rejected() {
        let errors = check_source("entity U {\n    id: uuid primary,\n    e: vector(0),\n    f: [vector(0)],\n}\n");
        let count = errors.iter().filter(|e| matches!(e, TypeCheckError::InvalidVectorDimension { .. })).count();
        assert_eq!(count, 2);
    }

    #[test]
    fn test_vector_dimension_positive_is_accepted() {
        let errors = check_source("entity U {\n    id: uuid primary,\n    e: vector(384) @vector,\n}\n");
        assert!(errors.is_empty(), "{:?}", errors);
    }

    #[test]
    fn test_foreign_key_type_mismatch() {
        let errors = check_source(
            "entity U {\n    id: uuid primary,\n    os: [O] via u_id,\n}\nentity O {\n    id: uuid primary,\n    u_id: string,\n}\n",
        );
        assert!(errors.iter().any(|e| matches!(e, TypeCheckError::ForeignKeyTypeMismatch { .. })), "{:?}", errors);
    }

    #[test]
    fn test_foreign_key_matching_type_is_accepted() {
        let errors = check_source(
            "entity U {\n    id: uuid primary,\n    os: [O] via u_id,\n}\nentity O {\n    id: uuid primary,\n    u_id: uuid,\n}\n",
        );
        assert!(errors.is_empty(), "{:?}", errors);
    }

    #[test]
    fn test_has_many_parent_key_must_be_id() {
        let errors = check_source(
            "entity C {\n    code: string primary,\n    os: [O] via c_code,\n}\nentity O {\n    id: uuid primary,\n    c_code: string,\n}\n",
        );
        assert!(errors.iter().any(|e| matches!(e, TypeCheckError::UnsupportedParentKey { .. })), "{:?}", errors);
    }

    #[test]
    fn test_non_id_primary_key_without_has_many_is_accepted() {
        let errors = check_source("entity C {\n    code: string primary,\n}\n");
        assert!(errors.is_empty(), "{:?}", errors);
    }

    #[test]
    fn test_self_referential_has_many_is_accepted() {
        let src = "entity Category {\n    id: uuid primary,\n    parent_id: uuid nullable,\n    parent: Category,\n    children: [Category] via parent_id,\n}\n";
        let errors = check_source(src);
        assert!(errors.is_empty(), "{:?}", errors);

        let schema = crate::parse_schema(src).unwrap();
        let migration = crate::generate_migration(&schema).unwrap();
        assert!(
            migration.sql.contains("FOREIGN KEY (parent_id) REFERENCES categorys(id)"),
            "{}", migration.sql
        );
    }

    #[test]
    fn test_mutual_has_many_is_still_a_cycle() {
        let errors = check_source(
            "entity A {\n    id: uuid primary,\n    bs: [B] via a_id,\n    b_id: uuid,\n}\nentity B {\n    id: uuid primary,\n    a_id: uuid,\n    as_: [A] via b_id,\n}\n",
        );
        assert!(errors.iter().any(|e| matches!(e, TypeCheckError::CircularDependency { .. })), "{:?}", errors);
    }
}