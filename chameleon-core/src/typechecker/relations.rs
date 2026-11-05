use crate::ast::{Schema, RelationKind};
use super::errors::TypeCheckError;

/// Validates all relations in the schema
pub fn check_relations(schema: &Schema) -> Vec<TypeCheckError> {
    let mut errors = Vec::new();

    for entity in &schema.entities {
        for (_, relation) in &entity.relations {
            // 1. Target entity exists
            if schema.get_entity(&relation.target_entity).is_none() {
                errors.push(TypeCheckError::UnknownRelationTarget {
                    entity: entity.name.clone(),
                    relation: relation.name.clone(),
                    target: relation.target_entity.clone(),
                });
                continue;
            }

            // 2. Foreign key exists in target entity
            if let Some(fk) = &relation.foreign_key {
                let target_entity = schema.get_entity(&relation.target_entity).unwrap();
                
                if !target_entity.fields.contains_key(fk) {
                    errors.push(TypeCheckError::InvalidForeignKey {
                        entity: entity.name.clone(),
                        relation: relation.name.clone(),
                        target: relation.target_entity.clone(),
                        foreign_key: fk.clone(),
                    });
                }
            }

            // 3. HasMany: parent key must be `id`, and the foreign key must have its type
            if relation.kind == RelationKind::HasMany {
                if let Some(fk) = &relation.foreign_key {
                    let target_entity = schema.get_entity(&relation.target_entity).unwrap();
                    let pk = entity.fields.values().find(|f| f.primary_key);

                    if let Some(pk) = pk {
                        if pk.name != "id" {
                            errors.push(TypeCheckError::UnsupportedParentKey {
                                entity: entity.name.clone(),
                                relation: relation.name.clone(),
                                primary_key: pk.name.clone(),
                            });
                        }

                        if let Some(fk_field) = target_entity.fields.get(fk) {
                            if fk_field.field_type != pk.field_type {
                                errors.push(TypeCheckError::ForeignKeyTypeMismatch {
                                    entity: entity.name.clone(),
                                    relation: relation.name.clone(),
                                    target: relation.target_entity.clone(),
                                    foreign_key: fk.clone(),
                                    fk_type: format!("{:?}", fk_field.field_type),
                                    pk: pk.name.clone(),
                                    pk_type: format!("{:?}", pk.field_type),
                                });
                            }
                        }
                    }
                }
            }

            // 4. HasMany relations MUST have a foreign key
            if relation.kind == RelationKind::HasMany && relation.foreign_key.is_none() {
                errors.push(TypeCheckError::MissingForeignKey {
                    entity: entity.name.clone(),
                    relation: relation.name.clone(),
                });
            }
        }
    }

    errors
}

/// Detects circular dependencies between entities using DFS
pub fn check_circular_dependencies(schema: &Schema) -> Vec<TypeCheckError> {
    let mut errors = Vec::new();
    let mut visited: Vec<String> = Vec::new();
    let mut in_stack: Vec<String> = Vec::new();

    for entity in &schema.entities {
        if !visited.contains(&entity.name) {
            if let Some(cycle) = dfs(schema, &entity.name, &mut visited, &mut in_stack) {
                errors.push(TypeCheckError::CircularDependency { cycle });
            }
        }
    }

    errors
}

fn dfs(
    schema: &Schema,
    current: &str,
    visited: &mut Vec<String>,
    in_stack: &mut Vec<String>,
) -> Option<Vec<String>> {
    visited.push(current.to_string());
    in_stack.push(current.to_string());
    
    if let Some(entity) = schema.get_entity(current) {
        for (_, relation) in &entity.relations {
            // BelongsTo is just the inverse side of a relation, skip it
            if relation.kind == RelationKind::BelongsTo {
                continue;
            }
            
            let target = &relation.target_entity;

            // A self-reference (e.g. Category.children) is a tree, not a cycle
            if target == current {
                continue;
            }

            if !visited.contains(target) {
                if let Some(cycle) = dfs(schema, target, visited, in_stack) {
                    return Some(cycle);
                }
            } else if in_stack.contains(target) {
                let start = in_stack.iter().position(|n| n == target).unwrap();
                let mut cycle: Vec<String> = in_stack[start..].to_vec();
                cycle.push(target.clone());
                return Some(cycle);
            }
        }
    }
    
    in_stack.pop();
    None
}