use crate::ast::Schema;
use crate::error::ChameleonError;

// Incluir el módulo parser generado por lalrpop
#[allow(clippy::all)]
pub mod schema {
    use crate::ast::{EntityItem, FieldModifier};
    use crate::ast::*;
    
    include!(concat!(env!("OUT_DIR"), "/parser/schema.rs"));
}

pub fn parse_schema(input: &str) -> Result<Schema, ChameleonError> {
    schema::SchemaParser::new()
        .parse(input)
        .map_err(|e| ChameleonError::ParseError(format!("{:?}", e)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ast::{BackendAnnotation, FieldType};
    use pretty_assertions::assert_eq;

    #[test]
    fn test_simple_entity() {
        let input = r#"
            entity User {
                id: uuid primary,
                email: string unique,
                age: int,
            }
        "#;
        
        let schema = parse_schema(input).unwrap();
        assert_eq!(schema.entities.len(), 1);
        
        let user = schema.entities.get("User").unwrap();
        assert_eq!(user.fields.len(), 3);
        assert!(user.fields.get("id").unwrap().primary_key);
        assert!(user.fields.get("email").unwrap().unique);
    }

    #[test]
    fn test_with_relations() {
        let input = r#"
            entity User {
                id: uuid primary,
                email: string,
                orders: [Order] via user_id,
            }
            
            entity Order {
                id: uuid primary,
                total: decimal,
                user: User,
            }
        "#;
        
        let schema = parse_schema(input).unwrap();
        assert_eq!(schema.entities.len(), 2);
        
        let user = schema.entities.get("User").unwrap();
        assert_eq!(user.relations.len(), 1);
        
        let orders_rel = user.relations.get("orders").unwrap();
        assert_eq!(orders_rel.target_entity, "Order");
    }

    #[test]
fn test_backend_annotations() {
    let input = r#"
        entity User {
            id: uuid primary,
            email: string unique,
            session_token: string @cache,
            monthly_spent: decimal @olap,
        }
    "#;
    
    let schema = parse_schema(input).unwrap();
    let user = schema.entities.get("User").unwrap();
    
    // Fields sin anotación son OLTP implícito
    let id = user.fields.get("id").unwrap();
    assert!(id.backend.is_none());  // None = OLTP implícito
    
    // @cache
    let session = user.fields.get("session_token").unwrap();
    assert_eq!(session.backend, Some(BackendAnnotation::Cache));
    
    // @olap
    let spent = user.fields.get("monthly_spent").unwrap();
    assert_eq!(spent.backend, Some(BackendAnnotation::OLAP));
}

#[test]
fn test_vector_type() {
    let input = r#"
        entity Product {
            id: uuid primary,
            embedding: vector(1536) @vector,
        }
    "#;
    
    let schema = parse_schema(input).unwrap();
    let product = schema.entities.get("Product").unwrap();
    
    let embedding = product.fields.get("embedding").unwrap();
    assert_eq!(embedding.field_type, FieldType::Vector(1536));
    assert_eq!(embedding.backend, Some(BackendAnnotation::Vector));
}

#[test]
fn test_array_types() {
    let input = r#"
        entity UserAnalytics {
            id: uuid primary,
            tags: [string],
            scores: [decimal],
        }
    "#;
    
    let schema = parse_schema(input).unwrap();
    let analytics = schema.entities.get("UserAnalytics").unwrap();
    
    let tags = analytics.fields.get("tags").unwrap();
    assert_eq!(tags.field_type, FieldType::Array(Box::new(FieldType::String)));
    
    let scores = analytics.fields.get("scores").unwrap();
    assert_eq!(scores.field_type, FieldType::Array(Box::new(FieldType::Decimal)));
}

#[test]
fn test_complex_multi_backend_schema() {
    let input = r#"
        entity Product {
            id: uuid primary,
            name: string,
            price: decimal,
            views_today: int @cache,
            monthly_sales: decimal @olap,
            embedding: vector(384) @vector,
            tags: [string],
        }
    "#;
    
    let schema = parse_schema(input).unwrap();
    let product = schema.entities.get("Product").unwrap();
    
    // Verify field count
    assert_eq!(product.fields.len(), 7);
    
    // Verify mixed backends
    assert!(product.fields.get("id").unwrap().backend.is_none());           // OLTP implicit
    assert_eq!(product.fields.get("views_today").unwrap().backend, Some(BackendAnnotation::Cache));
    assert_eq!(product.fields.get("monthly_sales").unwrap().backend, Some(BackendAnnotation::OLAP));
    assert_eq!(product.fields.get("embedding").unwrap().backend, Some(BackendAnnotation::Vector));
    
    // Verify types
    assert_eq!(product.fields.get("embedding").unwrap().field_type, FieldType::Vector(384));
    assert_eq!(product.fields.get("tags").unwrap().field_type, FieldType::Array(Box::new(FieldType::String)));
}
}