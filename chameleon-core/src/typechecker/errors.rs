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

    #[error("BelongsTo relation '{relation}' in '{entity}': {reason}")]
    InvalidBelongsTo {
        entity: String,
        relation: String,
        reason: String,
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

    // Constraints
    #[error("Field '{field}' in '{entity}' is primary and cannot be nullable")]
    NullablePrimaryKey {
        entity: String,
        field: String,
    },

    #[error("Field '{field}' in '{entity}' has vector dimension 0; dimension must be at least 1")]
    InvalidVectorDimension {
        entity: String,
        field: String,
    },

    // Foreign keys
    #[error("Relation '{relation}' in '{entity}' uses foreign key '{target}.{foreign_key}' of type '{fk_type}', but '{entity}' has primary key '{pk}' of type '{pk_type}'")]
    ForeignKeyTypeMismatch {
        entity: String,
        relation: String,
        target: String,
        foreign_key: String,
        fk_type: String,
        pk: String,
        pk_type: String,
    },

    #[error("Relation '{relation}' in '{entity}' is a has-many, which requires '{entity}' to have a primary key named 'id' (found '{primary_key}')")]
    UnsupportedParentKey {
        entity: String,
        relation: String,
        primary_key: String,
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
    },

    #[error("Field '{field}' in '{entity}' has invalid default '{default}' for type '{field_type}'")]
    InvalidDefaultValue {
        entity: String,
        field: String,
        field_type: String,
        default: String,
    },

    #[error("Field '{field}' in '{entity}' uses unsupported default function '{function}' for type '{field_type}'")]
    UnsupportedDefaultFunction {
        entity: String,
        field: String,
        field_type: String,
        function: String,
    }

}