use crate::ast::Schema;
use crate::error::{ChameleonError, ParseErrorDetail};

// Incluir el módulo parser generado por lalrpop
#[allow(clippy::all)]
pub mod schema {
    use crate::ast::{EntityItem, FieldModifier};
    use crate::ast::*;
    
    include!(concat!(env!("OUT_DIR"), "/parser/schema.rs"));
}

pub fn parse_schema(input: &str) -> Result<Schema, ChameleonError> {
    match schema::SchemaParser::new().parse(input) {
        Ok(schema) => Ok(schema),
        Err(e) => {
            let err: ChameleonError = e.into();
            Err(enhance_parse_error(err, input))
        }
    }
}

/// Convert byte offset to (line, column) in source text
fn offset_to_position(source: &str, offset: usize) -> (usize, usize) {
    let mut line = 1;
    let mut column = 1;
    
    for (i, ch) in source.chars().enumerate() {
        if i >= offset {
            break;
        }
        if ch == '\n' {
            line += 1;
            column = 1;
        } else {
            column += 1;
        }
    }
    
    (line, column)
}

/// Extract a snippet of source code around a position
fn extract_snippet(source: &str, line: usize, column: usize) -> String {
    let lines: Vec<&str> = source.lines().collect();
    
    if line == 0 || line > lines.len() {
        return String::new();
    }
    
    let target_line = lines[line - 1];
    let mut snippet = String::new();
    
    // Line number with proper padding
    let line_num_width = format!("{}", line).len().max(3);
    
    // Show the line with the error
    snippet.push_str(&format!("{:>width$} │ {}\n", line, target_line, width = line_num_width));
    
    // Show the error pointer
    snippet.push_str(&format!("{:>width$} │ ", "", width = line_num_width));
    for _ in 0..(column - 1) {
        snippet.push(' ');
    }
    
    // Calculate how many characters to underline
    let token_len = target_line[column - 1..]
        .chars()
        .take_while(|c| !c.is_whitespace() && *c != ',' && *c != '{' && *c != '}')
        .count()
        .max(1);
    
    for _ in 0..token_len {
        snippet.push('^');
    }
    
    snippet
}

/// Enhance parse error with source context
fn enhance_parse_error(
    err: ChameleonError, 
    source: &str
) -> ChameleonError {
    match err {
        ChameleonError::ParseError(mut detail) => {
            // If we have a column but line is 1, we need to recalculate
            if detail.line == 1 && detail.column > 1 {
                let (line, col) = offset_to_position(source, detail.column - 1);
                detail.line = line;
                detail.column = col;
            }
            
            // Add snippet
            let snippet = extract_snippet(source, detail.line, detail.column);
            detail.snippet = Some(snippet);
            
            // Add suggestions based on common mistakes
            detail = add_suggestions(detail);
            
            ChameleonError::ParseError(detail)
        }
        other => other,
    }
}

/// Add helpful suggestions based on error patterns
fn add_suggestions(mut detail: ParseErrorDetail) -> ParseErrorDetail {
    // Check for common typos in keywords
    if let Some(token) = &detail.token {
        let token_clean = token.replace("Token(", "").replace(")", "").replace("\"", "");
        let token_lower = token_clean.to_lowercase();
        
        // Typos in 'entity'
        if token_lower.contains("entiy") 
            || token_lower.contains("enity")
            || token_lower.contains("entit")
            || token_lower == "entiy" {
            detail.suggestion = Some("Did you mean 'entity'?".to_string());
        }
        // Typos in 'primary'
        else if token_lower.contains("primry") 
            || token_lower.contains("pirmary")
            || token_lower.contains("primari") {
            detail.suggestion = Some("Did you mean 'primary'?".to_string());
        }
        // Typos in 'unique'
        else if token_lower.contains("uniqu") && !token_lower.contains("unique") {
            detail.suggestion = Some("Did you mean 'unique'?".to_string());
        }
        // Typos in 'nullable'
        else if token_lower.contains("nullabe") || token_lower.contains("nulable") {
            detail.suggestion = Some("Did you mean 'nullable'?".to_string());
        }
    }
    
    // Check for common syntax mistakes based on message
    if detail.message.contains("expected one of") && detail.message.contains("\":\"") {
        if detail.suggestion.is_none() {
            detail.suggestion = Some(
                "Fields must have a type after the colon.\nExample: name: string".to_string()
            );
        }
    } 
    else if detail.message.contains("expected one of") && detail.message.contains("\"{\"") {
        if detail.suggestion.is_none() {
            detail.suggestion = Some(
                "Entity definitions must have an opening brace.\nExample: entity User {".to_string()
            );
        }
    }
    else if detail.message.contains("expected one of") && detail.message.contains("\"}\"") {
        if detail.suggestion.is_none() {
            detail.suggestion = Some(
                "Missing closing brace. Check that all entities are properly closed.".to_string()
            );
        }
    }
    else if detail.message.contains("Unexpected end of file") {
        if detail.suggestion.is_none() {
            detail.suggestion = Some(
                "The file ended unexpectedly. You may be missing a closing brace }".to_string()
            );
        }
    }
    
    detail
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
        
        let user = schema.get_entity("User").unwrap();
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
        
        let user = schema.get_entity("User").unwrap();
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
    let user = schema.get_entity("User").unwrap();
    
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
    let product = schema.get_entity("Product").unwrap();
    
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
    let analytics = schema.get_entity("UserAnalytics").unwrap();
    
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
    let product = schema.get_entity("Product").unwrap();
    
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