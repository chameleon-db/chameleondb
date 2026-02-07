package engine

import (
	"encoding/json"
	"fmt"
)

// Schema represents the complete database schema
type Schema struct {
	Entities []*Entity `json:"entities"`
}

// Entity represents a database entity (table)
type Entity struct {
	Name      string               `json:"name"`
	Fields    map[string]*Field    `json:"fields"`
	Relations map[string]*Relation `json:"relations"`
}

// Field represents an entity field (column)
type Field struct {
	Name       string       `json:"name"`
	Type       FieldType    `json:"field_type"`
	Nullable   bool         `json:"nullable"`
	Unique     bool         `json:"unique"`
	PrimaryKey bool         `json:"primary_key"`
	Default    *interface{} `json:"default,omitempty"`
	Backend    *string      `json:"backend,omitempty"`
}

// FieldType represents the type of a field and can be simple or complex
type FieldType struct {
	Kind  string      `json:"-"` // e.g., "UUID", "String", "Vector", "Array"
	Param interface{} `json:"-"` // e.g., size for Vector, inner type for Array
}

// Simple field type constants
var (
	FieldTypeUUID      = FieldType{Kind: "UUID"}
	FieldTypeString    = FieldType{Kind: "String"}
	FieldTypeInt       = FieldType{Kind: "Int"}
	FieldTypeDecimal   = FieldType{Kind: "Decimal"}
	FieldTypeBool      = FieldType{Kind: "Bool"}
	FieldTypeTimestamp = FieldType{Kind: "Timestamp"}
	FieldTypeFloat     = FieldType{Kind: "Float"}
)

// UnmarshalJSON deserializes FieldType from JSON
// Can be: "UUID" (string) or {"Vector": 1536} or {"Array": "String"} (object)
func (ft *FieldType) UnmarshalJSON(data []byte) error {
	// Try as string first (simple types)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*ft = FieldType{Kind: s}
		return nil
	}

	// Try as object (complex types like Vector and Array)
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		// Should have exactly one key
		if len(obj) != 1 {
			return fmt.Errorf("invalid FieldType object: expected 1 key, got %d", len(obj))
		}

		for key, value := range obj {
			*ft = FieldType{Kind: key, Param: value}
			return nil
		}
	}

	return fmt.Errorf("cannot unmarshal FieldType from %s", string(data))
}

// MarshalJSON serializes FieldType to JSON
func (ft FieldType) MarshalJSON() ([]byte, error) {
	if ft.Param == nil {
		return json.Marshal(ft.Kind)
	}
	obj := map[string]interface{}{ft.Kind: ft.Param}
	return json.Marshal(obj)
}

// String returns a string representation of the FieldType
func (ft FieldType) String() string {
	if ft.Param == nil {
		return ft.Kind
	}
	return fmt.Sprintf("%s(%v)", ft.Kind, ft.Param)
}

// Relation represents a relationship between entities
type Relation struct {
	Name         string       `json:"name"`
	Kind         RelationKind `json:"kind"`
	TargetEntity string       `json:"target_entity"`
	ForeignKey   *string      `json:"foreign_key,omitempty"`
	Through      *string      `json:"through,omitempty"`
}

// RelationKind represents the type of relationship
type RelationKind string

const (
	RelationHasOne     RelationKind = "HasOne"
	RelationHasMany    RelationKind = "HasMany"
	RelationBelongsTo  RelationKind = "BelongsTo"
	RelationManyToMany RelationKind = "ManyToMany"
)

// ParseSchemaJSON parses a JSON string into a Schema
func ParseSchemaJSON(jsonStr string) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal([]byte(jsonStr), &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ToJSON converts a Schema to JSON string
func (s *Schema) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
