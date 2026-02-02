use crate::ast::{FieldType, DefaultValue};

/// Maps ChameleonDB field types to PostgreSQL column types
pub fn to_postgres_type(field_type: &FieldType) -> String {
    match field_type {
        FieldType::UUID      => "UUID".to_string(),
        FieldType::String    => "VARCHAR".to_string(),
        FieldType::Int       => "INTEGER".to_string(),
        FieldType::Decimal   => "NUMERIC".to_string(),
        FieldType::Bool      => "BOOLEAN".to_string(),
        FieldType::Timestamp => "TIMESTAMP".to_string(),
        FieldType::Float     => "DOUBLE PRECISION".to_string(),
        FieldType::Vector(dim) => format!("VECTOR({})", dim),
        FieldType::Array(inner) => format!("{}[]", to_postgres_type(inner)),
    }
}

/// Maps ChameleonDB default values to PostgreSQL expressions
pub fn to_postgres_default(default: &DefaultValue) -> String {
    match default {
        DefaultValue::Now      => "NOW()".to_string(),
        DefaultValue::UUIDv4   => "gen_random_uuid()".to_string(),
        DefaultValue::Literal(s) => format!("'{}'", s),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ast::FieldType;

    #[test]
    fn test_basic_types() {
        assert_eq!(to_postgres_type(&FieldType::UUID), "UUID");
        assert_eq!(to_postgres_type(&FieldType::String), "VARCHAR");
        assert_eq!(to_postgres_type(&FieldType::Int), "INTEGER");
        assert_eq!(to_postgres_type(&FieldType::Decimal), "NUMERIC");
        assert_eq!(to_postgres_type(&FieldType::Bool), "BOOLEAN");
        assert_eq!(to_postgres_type(&FieldType::Timestamp), "TIMESTAMP");
        assert_eq!(to_postgres_type(&FieldType::Float), "DOUBLE PRECISION");
    }

    #[test]
    fn test_vector_type() {
        assert_eq!(to_postgres_type(&FieldType::Vector(384)), "VECTOR(384)");
        assert_eq!(to_postgres_type(&FieldType::Vector(1536)), "VECTOR(1536)");
    }

    #[test]
    fn test_array_type() {
        assert_eq!(
            to_postgres_type(&FieldType::Array(Box::new(FieldType::String))),
            "VARCHAR[]"
        );
        assert_eq!(
            to_postgres_type(&FieldType::Array(Box::new(FieldType::Int))),
            "INTEGER[]"
        );
    }

    #[test]
    fn test_defaults() {
        assert_eq!(to_postgres_default(&DefaultValue::Now), "NOW()");
        assert_eq!(to_postgres_default(&DefaultValue::UUIDv4), "gen_random_uuid()");
        assert_eq!(to_postgres_default(&DefaultValue::Literal("hello".to_string())), "'hello'");
    }
}