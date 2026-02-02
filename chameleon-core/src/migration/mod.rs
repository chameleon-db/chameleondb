pub mod generator;
pub mod type_map;

pub use generator::{generate_migration, Migration, MigrationError};

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ast::*;

    /// Helper: build the standard User → Order → OrderItem schema
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
        user.add_field(Field {
            name: "created_at".to_string(),
            field_type: FieldType::Timestamp,
            nullable: false, unique: false, primary_key: false,
            default: Some(DefaultValue::Now), backend: None,
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

    // ─── TYPE MAP ───

    #[test]
    fn test_type_map_basic() {
        assert_eq!(type_map::to_postgres_type(&FieldType::UUID), "UUID");
        assert_eq!(type_map::to_postgres_type(&FieldType::String), "VARCHAR");
        assert_eq!(type_map::to_postgres_type(&FieldType::Int), "INTEGER");
        assert_eq!(type_map::to_postgres_type(&FieldType::Decimal), "NUMERIC");
        assert_eq!(type_map::to_postgres_type(&FieldType::Bool), "BOOLEAN");
        assert_eq!(type_map::to_postgres_type(&FieldType::Timestamp), "TIMESTAMP");
        assert_eq!(type_map::to_postgres_type(&FieldType::Float), "DOUBLE PRECISION");
    }

    #[test]
    fn test_type_map_vector() {
        assert_eq!(type_map::to_postgres_type(&FieldType::Vector(384)), "VECTOR(384)");
    }

    #[test]
    fn test_type_map_array() {
        assert_eq!(
            type_map::to_postgres_type(&FieldType::Array(Box::new(FieldType::String))),
            "VARCHAR[]"
        );
    }

    // ─── CREATE TABLE ───

    #[test]
    fn test_generate_single_table() {
        let mut schema = Schema::new();
        let mut entity = Entity::new("User".to_string());
        entity.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        entity.add_field(Field {
            name: "email".to_string(),
            field_type: FieldType::String,
            nullable: false, unique: true, primary_key: false,
            default: None, backend: None,
        });
        schema.add_entity(entity);

        let migration = generate_migration(&schema).unwrap();

        assert_eq!(migration.statements.len(), 1);
        assert!(migration.sql.contains("CREATE TABLE users"));
        assert!(migration.sql.contains("id UUID PRIMARY KEY"));
        assert!(migration.sql.contains("email VARCHAR NOT NULL UNIQUE"));
    }

    #[test]
    fn test_nullable_field() {
        let mut schema = Schema::new();
        let mut entity = Entity::new("User".to_string());
        entity.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        entity.add_field(Field {
            name: "age".to_string(),
            field_type: FieldType::Int,
            nullable: true, unique: false, primary_key: false,
            default: None, backend: None,
        });
        schema.add_entity(entity);

        let migration = generate_migration(&schema).unwrap();

        // age should NOT have NOT NULL
        assert!(migration.sql.contains("age INTEGER"));
        assert!(!migration.sql.contains("age INTEGER NOT NULL"));
    }

    #[test]
    fn test_default_values() {
        let mut schema = Schema::new();
        let mut entity = Entity::new("Event".to_string());
        entity.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: Some(DefaultValue::UUIDv4), backend: None,
        });
        entity.add_field(Field {
            name: "created_at".to_string(),
            field_type: FieldType::Timestamp,
            nullable: false, unique: false, primary_key: false,
            default: Some(DefaultValue::Now), backend: None,
        });
        schema.add_entity(entity);

        let migration = generate_migration(&schema).unwrap();

        assert!(migration.sql.contains("DEFAULT gen_random_uuid()"));
        assert!(migration.sql.contains("DEFAULT NOW()"));
    }

    // ─── FOREIGN KEYS ───

    #[test]
    fn test_foreign_key_generation() {
        let schema = test_schema();
        let migration = generate_migration(&schema).unwrap();

        // Order should have FK to users
        let order_stmt = migration.statements.iter()
            .find(|(name, _)| name == "Order")
            .unwrap();
        assert!(order_stmt.1.contains("FOREIGN KEY (user_id) REFERENCES users(id)"));

        // OrderItem should have FK to orders
        let item_stmt = migration.statements.iter()
            .find(|(name, _)| name == "OrderItem")
            .unwrap();
        assert!(item_stmt.1.contains("FOREIGN KEY (order_id) REFERENCES orders(id)"));
    }

    // ─── CREATION ORDER ───

    #[test]
    fn test_creation_order() {
        let schema = test_schema();
        let migration = generate_migration(&schema).unwrap();

        let order: Vec<&str> = migration.statements.iter()
            .map(|(name, _)| name.as_str())
            .collect();

        // User must come before Order (Order has FK to users)
        let user_pos = order.iter().position(|&n| n == "User").unwrap();
        let order_pos = order.iter().position(|&n| n == "Order").unwrap();
        let item_pos = order.iter().position(|&n| n == "OrderItem").unwrap();

        assert!(user_pos < order_pos, "User must be created before Order");
        assert!(order_pos < item_pos, "Order must be created before OrderItem");
    }

    // ─── FULL SCHEMA ───

    #[test]
    fn test_full_migration() {
        let schema = test_schema();
        let migration = generate_migration(&schema).unwrap();

        // All three tables generated
        assert_eq!(migration.statements.len(), 3);

        // Full SQL contains all tables
        assert!(migration.sql.contains("CREATE TABLE users"));
        assert!(migration.sql.contains("CREATE TABLE orders"));
        assert!(migration.sql.contains("CREATE TABLE order_items"));
    }

    // ─── ANNOTATIONS DON'T AFFECT DDL ───

    #[test]
    fn test_annotations_ignored_in_ddl() {
        let mut schema = Schema::new();
        let mut entity = Entity::new("Product".to_string());
        entity.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        entity.add_field(Field {
            name: "views".to_string(),
            field_type: FieldType::Int,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: Some(BackendAnnotation::Cache),
        });
        entity.add_field(Field {
            name: "sales".to_string(),
            field_type: FieldType::Decimal,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: Some(BackendAnnotation::OLAP),
        });
        schema.add_entity(entity);

        let migration = generate_migration(&schema).unwrap();

        // Annotations don't appear in DDL
        assert!(!migration.sql.contains("@cache"));
        assert!(!migration.sql.contains("@olap"));
        // But fields are still generated
        assert!(migration.sql.contains("views INTEGER NOT NULL"));
        assert!(migration.sql.contains("sales NUMERIC NOT NULL"));
    }
}