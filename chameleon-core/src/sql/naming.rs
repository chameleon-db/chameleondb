/// Converts schema names to PostgreSQL naming conventions
///
/// Entity names: PascalCase → snake_case plural
///   User        → users
///   OrderItem   → order_items
///
/// Field names: already snake_case, pass through
///   email       → email
///   created_at  → created_at

/// Convert a PascalCase entity name to a snake_case plural table name
pub fn entity_to_table(entity_name: &str) -> String {
    let snake = pascal_to_snake(entity_name);
    pluralize(&snake)
}

/// Convert PascalCase to snake_case
/// "OrderItem" → "order_item"
/// "User"      → "user"
/// "UUID"      → "u_u_i_d" (edge case, handled separately if needed)
fn pascal_to_snake(name: &str) -> String {
    let mut result = String::new();

    for (i, ch) in name.chars().enumerate() {
        if ch.is_uppercase() && i > 0 {
            // Don't add underscore if previous char was also uppercase
            // This handles acronyms like "ID" better
            let prev = name.chars().nth(i - 1).unwrap();
            if prev.is_lowercase() {
                result.push('_');
            }
        }
        result.push(ch.to_lowercase().next().unwrap());
    }

    result
}

/// Simple pluralization (English rules, covers most cases)
/// "user"       → "users"
/// "order"      → "orders"
/// "order_item" → "order_items"
fn pluralize(name: &str) -> String {
    // For now: just add 's'
    // Can be extended later for irregular plurals
    format!("{}s", name)
}

/// Maximum identifier length accepted by PostgreSQL.
const MAX_IDENT_LEN: usize = 63;

/// Validate a name and render it as a quoted PostgreSQL identifier.
///
/// This is the only place identifiers are turned into SQL text. A name must
/// start with an ASCII letter or `_`, continue with ASCII letters, digits or
/// `_`, and fit in 63 bytes. Names are folded to lower case, which is how
/// PostgreSQL treats the unquoted names emitted by the migration generator.
/// Returns `None` for anything else.
pub fn quote_ident(name: &str) -> Option<String> {
    let mut chars = name.chars();
    let first = chars.next()?;
    if !(first.is_ascii_alphabetic() || first == '_') {
        return None;
    }
    if name.len() > MAX_IDENT_LEN || !chars.all(|c| c.is_ascii_alphanumeric() || c == '_') {
        return None;
    }
    Some(format!("\"{}\"", name.to_ascii_lowercase()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_quote_ident_accepts_plain_names() {
        assert_eq!(quote_ident("users").as_deref(), Some("\"users\""));
        assert_eq!(quote_ident("_a1").as_deref(), Some("\"_a1\""));
        assert_eq!(quote_ident("createdAt").as_deref(), Some("\"createdat\""));
    }

    #[test]
    fn test_quote_ident_rejects_other_names() {
        let too_long = "a".repeat(64);
        for name in [
            "", "1a", "a b", "a-b", "a.b", "a\"b", "a'b", "a;b", "a--b", "a/*b",
            "a\\b", "a\0b", "\u{e9}a", "a\u{e9}", too_long.as_str(),
        ] {
            assert!(quote_ident(name).is_none(), "accepted {:?}", name);
        }
    }

    #[test]
    fn test_simple_entity() {
        assert_eq!(entity_to_table("User"), "users");
        assert_eq!(entity_to_table("Order"), "orders");
    }

    #[test]
    fn test_compound_entity() {
        assert_eq!(entity_to_table("OrderItem"), "order_items");
    }
}