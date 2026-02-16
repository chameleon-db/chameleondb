use crate::ast::*;

/// Helper: build a standard test schema
/// User (HasMany) → Order (HasMany) → OrderItem
pub fn test_schema() -> Schema {
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