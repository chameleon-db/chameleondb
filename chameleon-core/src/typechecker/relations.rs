use crate::ast::{Schema, RelationKind};
use super::errors::TypeCheckError;

/// Validates all relations in the schema
pub fn check_relations(schema: &Schema) -> Vec<TypeCheckError> {
    let mut errors = Vec::new();

    for (entity_name, entity) in &schema.entities {
        for (_, relation) in &entity.relations {
            // 1. Target entity exists
            if !schema.entities.contains_key(&relation.target_entity) {
                errors.push(TypeCheckError::UnknownRelationTarget {
                    entity: entity_name.clone(),
                    relation: relation.name.clone(),
                    target: relation.target_entity.clone(),
                });
                continue; // No tiene sentido validar más si el target no existe
            }

            // 2. HasMany requiere foreign key
            if relation.kind == RelationKind::HasMany && relation.foreign_key.is_none() {
                errors.push(TypeCheckError::MissingForeignKey {
                    entity: entity_name.clone(),
                    relation: relation.name.clone(),
                });
                continue;
            }

            // 3. Foreign key existe en la entidad target
            if let Some(fk) = &relation.foreign_key {
                let target = schema.entities.get(&relation.target_entity).unwrap();
                if !target.fields.contains_key(fk) {
                    errors.push(TypeCheckError::InvalidForeignKey {
                        entity: entity_name.clone(),
                        relation: relation.name.clone(),
                        target: relation.target_entity.clone(),
                        foreign_key: fk.clone(),
                    });
                }
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

    for entity_name in schema.entities.keys() {
        if !visited.contains(entity_name) {
            if let Some(cycle) = dfs(schema, entity_name, &mut visited, &mut in_stack) {
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

    if let Some(entity) = schema.entities.get(current) {
        for (_, relation) in &entity.relations {
            // BelongsTo is just the inverse side of a relation, skip it
            if relation.kind == RelationKind::BelongsTo {
                continue;
            }

            let target = &relation.target_entity;

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