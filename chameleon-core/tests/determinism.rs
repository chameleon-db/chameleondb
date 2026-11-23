//! The same schema must give the same output every time (E1-04).
//!
//! A HashMap picks a new random iteration order for every instance, so parsing
//! the same text again is enough to expose any output that depends on it. Each
//! test parses the schema many times and compares the results byte for byte.

use chameleon::{generate_migration, generate_sql, parse_schema, type_check, IncludePath, Query};

const RUNS: usize = 100;

// Many fields and relations on purpose: with n items, a random order matches by
// chance with probability 1/n!.
const SCHEMA: &str = r#"
entity User {
    id: uuid primary,
    email: string unique,
    name: string,
    age: int nullable,
    score: float nullable,
    active: bool,
    bio: string nullable,
    city: string nullable,
    created_at: timestamp default now(),
    orders: [Order] via user_id,
    posts: [Post] via author_id,
    comments: [Comment] via user_id,
}

entity Order {
    id: uuid primary,
    total: decimal,
    status: string,
    note: string nullable,
    created_at: timestamp default now(),
    user_id: uuid,
    user: User,
    items: [OrderItem] via order_id,
}

entity OrderItem {
    id: uuid primary,
    quantity: int,
    price: decimal,
    sku: string,
    order_id: uuid,
    order: Order,
}

entity Post {
    id: uuid primary,
    title: string,
    body: string,
    published: bool,
    author_id: uuid,
    author: User,
    comments: [Comment] via post_id,
}

entity Comment {
    id: uuid primary,
    content: string,
    post_id: uuid,
    user_id: uuid,
    post: Post,
    user: User,
}
"#;

fn all_equal<T: PartialEq + std::fmt::Debug>(what: &str, mut produce: impl FnMut() -> T) {
    let first = produce();
    for run in 1..RUNS {
        let next = produce();
        assert_eq!(first, next, "{what}: run {run} differs from run 0");
    }
}

#[test]
fn parsed_schema_json_is_stable() {
    all_equal("schema JSON", || {
        serde_json::to_string_pretty(&parse_schema(SCHEMA).unwrap()).unwrap()
    });
}

#[test]
fn migration_sql_is_stable() {
    all_equal("migration", || {
        let schema = parse_schema(SCHEMA).unwrap();
        let migration = generate_migration(&schema).unwrap();
        (migration.sql, migration.statements)
    });
}

#[test]
fn select_all_and_include_columns_are_stable() {
    all_equal("query SQL", || {
        let schema = parse_schema(SCHEMA).unwrap();
        let mut query = Query::new("User");
        query.includes.push(IncludePath::parse("orders.items"));
        query.includes.push(IncludePath::parse("posts.comments"));
        let sql = generate_sql(&query, &schema).unwrap();
        (sql.main_query, sql.eager_queries)
    });
}

#[test]
fn select_all_columns_are_documented_order() {
    // The primary key first, then alphabetical by name.
    let schema = parse_schema(SCHEMA).unwrap();
    let sql = generate_sql(&Query::new("User"), &schema).unwrap().main_query;
    let columns = [
        "\"id\"", "\"active\"", "\"age\"", "\"bio\"", "\"city\"", "\"created_at\"",
        "\"email\"", "\"name\"", "\"score\"",
    ];
    let mut at = 0;
    for column in columns {
        let found = sql[at..]
            .find(column)
            .unwrap_or_else(|| panic!("{column} missing or out of order in:\n{sql}"));
        at += found + column.len();
    }
}

#[test]
fn type_check_errors_come_in_the_same_order() {
    // Several independent problems, so the order of the report matters.
    const BAD: &str = r#"
entity A {
    id: uuid primary,
    b: B,
    c: [Missing] via a_id,
    x: string unique primary nullable,
    y: string nullable primary,
    z: uuid primary,
}

entity B {
    name: string,
    other: int default "text",
    n: [A] via b_id,
}

entity A {
    id: uuid primary,
}
"#;
    all_equal("type check report", || match parse_schema(BAD) {
        Ok(schema) => type_check(&schema).error_report(),
        Err(e) => format!("{e:?}"),
    });
}
