use crate::ast::{Schema, FieldType, BackendAnnotation};
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