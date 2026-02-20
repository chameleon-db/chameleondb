package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentityMap_Deduplicate_NoDuplicates(t *testing.T) {
	im := NewIdentityMap()

	rows := []Row{
		{"id": "123", "name": "Ana"},
		{"id": "456", "name": "Bob"},
	}

	result := im.Deduplicate("User", rows)

	assert.Len(t, result, 2)
	assert.Equal(t, "Ana", result[0]["name"])
	assert.Equal(t, "Bob", result[1]["name"])
}

func TestIdentityMap_Deduplicate_WithDuplicates(t *testing.T) {
	im := NewIdentityMap()

	rows := []Row{
		{"id": "123", "name": "Ana", "age": 25},
		{"id": "123", "name": "Ana", "age": 25},
		{"id": "123", "name": "Ana", "age": 25},
	}

	result := im.Deduplicate("User", rows)

	assert.Len(t, result, 3)
	result[0]["country"] = "AR"
	assert.Equal(t, "AR", result[1]["country"])
	assert.Equal(t, "AR", result[2]["country"])
}

func TestIdentityMap_Deduplicate_NoID(t *testing.T) {
	im := NewIdentityMap()

	rows := []Row{
		{"name": "Ana"},
		{"name": "Bob"},
	}

	result := im.Deduplicate("User", rows)

	assert.Len(t, result, 2)
}

func TestIdentityMap_MultipleEntities(t *testing.T) {
	im := NewIdentityMap()

	users := []Row{
		{"id": "1", "name": "Ana"},
		{"id": "1", "name": "Ana"},
	}

	posts := []Row{
		{"id": "1", "title": "Post 1"},
		{"id": "1", "title": "Post 1"},
	}

	resultUsers := im.Deduplicate("User", users)
	resultPosts := im.Deduplicate("Post", posts)

	assert.Len(t, resultUsers, 2)
	assert.Len(t, resultPosts, 2)
	resultUsers[0]["role"] = "admin"
	_, found := resultPosts[0]["role"]
	assert.False(t, found)
}
