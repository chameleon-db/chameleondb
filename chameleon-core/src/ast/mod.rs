use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Schema {
    pub entities: Vec<Entity>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Entity {
    pub name: String,
    pub fields: HashMap<String, Field>,
    pub relations: HashMap<String, Relation>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Field {
    pub name: String,
    pub field_type: FieldType,
    pub nullable: bool,
    pub unique: bool,
    pub primary_key: bool,
    pub default: Option<DefaultValue>,
    pub backend: Option<BackendAnnotation>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum FieldType {
    UUID,
    String,
    Int,
    Decimal,
    Bool,
    Timestamp,
    Float,
    Vector(usize),
    Array(Box<FieldType>),
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum DefaultValue {
    Now,
    UUIDv4,
    StringLiteral(String),
    BoolLiteral(bool),
    IntLiteral(i64),
    FloatLiteral(f64),
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Relation {
    pub name: String,
    pub kind: RelationKind,
    pub target_entity: String,
    pub foreign_key: Option<String>,
    pub through: Option<String>,  // para many_to_many
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum RelationKind {
    HasOne,
    HasMany,
    BelongsTo,
    ManyToMany,
}

// Helper types para el parser LALRPOP
#[derive(Debug)]
pub enum EntityItem {
    Field(Field),
    Relation(Relation),
}

#[derive(Debug)]
pub enum FieldModifier {
    Primary,
    Unique,
    Nullable,
    Default(DefaultValue),
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum BackendAnnotation {
    OLTP,                           // default, implícito
    Cache,                          // @cache
    OLAP,                           // @olap
    Vector,                         // @vector
    ML,                             // @ml (futuro)
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct BackendCapabilities {
    pub name: String,
    pub annotation: BackendAnnotation,
    pub supports_transactions: bool,
    pub supports_relations: bool,
    pub fallback: Option<BackendAnnotation>,  // ej: @cache fallback a OLTP
}

impl Schema {
    pub fn new() -> Self {
        Schema {
            entities: Vec::new(),
        }
    }

    pub fn add_entity(&mut self, entity: Entity) {
        self.entities.push(entity);
    }

    // Helper para búsquedas rápidas
    pub fn get_entity(&self, name: &str) -> Option<&Entity> {
        self.entities.iter().find(|e| e.name == name)
    }

    pub fn get_entity_mut(&mut self, name: &str) -> Option<&mut Entity> {
        self.entities.iter_mut().find(|e| e.name == name)
    }

    /// Column of `entity` that holds the key of a BelongsTo relation.
    ///
    /// The DSL has no `via` for BelongsTo, so the key is found, in order:
    /// 1. an explicit `foreign_key` on the relation;
    /// 2. the `via` of the has-many on the target that points back at `entity`
    ///    (`User.posts: [Post] via author_id` gives `Post.author` the key `author_id`);
    /// 3. a field named `<relation>_id` on `entity`.
    ///
    /// With several has-many relations pointing back, `<relation>_id` breaks the tie.
    pub fn belongs_to_key(&self, entity: &Entity, relation: &Relation) -> Result<String, BelongsToKeyError> {
        if let Some(fk) = &relation.foreign_key {
            return Ok(fk.clone());
        }

        let conventional = format!("{}_id", relation.name);
        let mut candidates: Vec<String> = self
            .get_entity(&relation.target_entity)
            .map(|target| {
                target
                    .relations
                    .values()
                    .filter(|r| r.kind == RelationKind::HasMany && r.target_entity == entity.name)
                    .filter_map(|r| r.foreign_key.clone())
                    .filter(|fk| entity.fields.contains_key(fk))
                    .collect()
            })
            .unwrap_or_default();
        candidates.sort();
        candidates.dedup();

        match candidates.len() {
            1 => Ok(candidates.remove(0)),
            0 if entity.fields.contains_key(&conventional) => Ok(conventional),
            0 => Err(BelongsToKeyError::Missing),
            _ if candidates.contains(&conventional) => Ok(conventional),
            _ => Err(BelongsToKeyError::Ambiguous(candidates)),
        }
    }
}

/// Why the key of a BelongsTo relation could not be resolved.
#[derive(Debug, Clone, PartialEq)]
pub enum BelongsToKeyError {
    Missing,
    Ambiguous(Vec<String>),
}

impl Entity {
    pub fn new(name: String) -> Self {
        Entity {
            name,
            fields: HashMap::new(),
            relations: HashMap::new(),
        }
    }
    
    pub fn add_field(&mut self, field: Field) {
        self.fields.insert(field.name.clone(), field);
    }
    
    pub fn add_relation(&mut self, relation: Relation) {
        self.relations.insert(relation.name.clone(), relation);
    }
}