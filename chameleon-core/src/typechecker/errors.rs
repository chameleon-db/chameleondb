use thiserror::Error;

#[derive(Error, Debug, Clone, PartialEq)]
pub enum TypeCheckError {
    // Relaciones
    #[error("Entity '{entity}' references unknown entity '{target}' in relation '{relation}'")]
    UnknownRelationTarget {
        entity: String,
        relation: String,
        target: String,
    },

    #[error("Relation '{relation}' in '{entity}' references foreign key '{foreign_key}', but '{target}' has no such field")]
    InvalidForeignKey {
        entity: String,
        relation: String,
        target: String,
        foreign_key: String,
    },

    #[error("HasMany relation '{relation}' in '{entity}' requires a 'via' foreign key")]
    MissingForeignKey {
        entity: String,
        relation: String,
    },

    // Primary keys
    #[error("Entity '{entity}' has no primary key")]
    MissingPrimaryKey {
        entity: String,
    },

    #[error("Entity '{entity}' has multiple primary keys: {fields:?}")]
    MultiplePrimaryKeys {
        entity: String,
        fields: Vec<String>,
    },

    // Annotations
    #[error("Field '{field}' in '{entity}' uses @vector but type is '{actual_type}', expected vector(N)")]
    InvalidVectorAnnotation {
        entity: String,
        field: String,
        actual_type: String,
    },

    #[error("Field '{field}' in '{entity}' is {constraint} and cannot have @{annotation} annotation")]
    AnnotationOnConstrainedField {
        entity: String,
        field: String,
        constraint: String,  // "primary" or "unique"
        annotation: String,
    },

    // Circular dependencies
    #[error("Circular dependency detected: {cycle:?}")]
    CircularDependency {
        cycle: Vec<String>,
    },

    // Duplicates
    #[error(
        "Duplicate entity name '{entity}' (first defined at #{first}, redefined at #{second})"
    )]
    DuplicateEntity {
        entity: String,
        first: usize,
        second: usize,
    },

    #[error("Duplicate field name '{field}' in entity '{entity}'")]
    DuplicateField {
        entity: String,
        field: String,
    }

}