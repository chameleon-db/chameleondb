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
    Literal(String),
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