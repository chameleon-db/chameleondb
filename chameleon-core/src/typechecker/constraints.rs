use crate::ast::{Schema, FieldType, BackendAnnotation, DefaultValue};
use super::errors::TypeCheckError;

/// Validates primary key constraints
pub fn check_primary_keys(schema: &Schema) -> Vec<TypeCheckError> {
    let mut errors = Vec::new();
    
    for entity in &schema.entities {
        let primary_keys: Vec<String> = entity.fields.iter()
            .filter(|(_, field)| field.primary_key)
            .map(|(name, _)| name.clone())
            .collect();
        
        match primary_keys.len() {
            0 => errors.push(TypeCheckError::MissingPrimaryKey {
                entity: entity.name.clone(),
            }),
            1 => {} // Ok
            _ => errors.push(TypeCheckError::MultiplePrimaryKeys {
                entity: entity.name.clone(),
                fields: primary_keys,
            }),
        }
    }
    
    errors
}

/// Validates modifier combinations that make no sense together
pub fn check_field_modifiers(schema: &Schema) -> Vec<TypeCheckError> {
    let mut errors = Vec::new();

    for entity in &schema.entities {
        let mut fields: Vec<_> = entity.fields.values().collect();
        fields.sort_by(|a, b| a.name.cmp(&b.name));

        for field in fields {
            // A primary key is always NOT NULL in PostgreSQL
            if field.primary_key && field.nullable {
                errors.push(TypeCheckError::NullablePrimaryKey {
                    entity: entity.name.clone(),
                    field: field.name.clone(),
                });
            }

            if has_zero_dimension_vector(&field.field_type) {
                errors.push(TypeCheckError::InvalidVectorDimension {
                    entity: entity.name.clone(),
                    field: field.name.clone(),
                });
            }
        }
    }

    errors
}

fn has_zero_dimension_vector(field_type: &FieldType) -> bool {
    match field_type {
        FieldType::Vector(0) => true,
        FieldType::Array(inner) => has_zero_dimension_vector(inner),
        _ => false,
    }
}

/// Validates backend annotation consistency
pub fn check_annotations(schema: &Schema) -> Vec<TypeCheckError> {
    let mut errors = Vec::new();

    for entity in &schema.entities {
    for (_, field) in &entity.fields {
        if let Some(annotation) = &field.backend {
            // 1. @vector solo con tipo vector(N)
            if *annotation == BackendAnnotation::Vector {
                if !matches!(field.field_type, FieldType::Vector(_)) {
                    errors.push(TypeCheckError::InvalidVectorAnnotation {
                        entity: entity.name.clone(),
                        field: field.name.clone(),
                        actual_type: format!("{:?}", field.field_type),
                    });
                }
            }

                // 2. Primary keys no pueden tener annotations
                if field.primary_key {
                    errors.push(TypeCheckError::AnnotationOnConstrainedField {
                        entity: entity.name.clone(),
                        field: field.name.clone(),
                        constraint: "primary".to_string(),
                        annotation: format!("{:?}", annotation).to_lowercase(),
                    });
                }

                // 3. Unique fields no pueden tener annotations
                if field.unique {
                    errors.push(TypeCheckError::AnnotationOnConstrainedField {
                        entity: entity.name.clone(),
                        field: field.name.clone(),
                        constraint: "unique".to_string(),
                        annotation: format!("{:?}", annotation).to_lowercase(),
                    });
                }
            }
        }
    }

    errors
}

/// Validates default value compatibility with field types.
pub fn check_defaults(schema: &Schema) -> Vec<TypeCheckError> {
    let mut errors = Vec::new();

    for entity in &schema.entities {
        for (_, field) in &entity.fields {
            let Some(default) = &field.default else {
                continue;
            };

            match (&field.field_type, default) {
                (FieldType::Timestamp, DefaultValue::Now) => {}
                (FieldType::UUID, DefaultValue::UUIDv4) => {}
                (FieldType::String, DefaultValue::StringLiteral(_)) => {}
                (FieldType::Bool, DefaultValue::BoolLiteral(_)) => {}
                (FieldType::Int, DefaultValue::IntLiteral(_)) => {}
                (FieldType::Decimal, DefaultValue::FloatLiteral(_)) => {}
                (FieldType::Decimal, DefaultValue::IntLiteral(_)) => {}
                (FieldType::Float, DefaultValue::FloatLiteral(_)) => {}
                (FieldType::Float, DefaultValue::IntLiteral(_)) => {}
                (_, DefaultValue::Now) => errors.push(TypeCheckError::UnsupportedDefaultFunction {
                    entity: entity.name.clone(),
                    field: field.name.clone(),
                    field_type: format!("{:?}", field.field_type),
                    function: "now()".to_string(),
                }),
                (_, DefaultValue::UUIDv4) => errors.push(TypeCheckError::UnsupportedDefaultFunction {
                    entity: entity.name.clone(),
                    field: field.name.clone(),
                    field_type: format!("{:?}", field.field_type),
                    function: "uuid()".to_string(),
                }),
                _ => errors.push(TypeCheckError::InvalidDefaultValue {
                    entity: entity.name.clone(),
                    field: field.name.clone(),
                    field_type: format!("{:?}", field.field_type),
                    default: format!("{:?}", default),
                }),
            }
        }
    }

    errors
}