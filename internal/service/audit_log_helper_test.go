package service

import (
	"context"
	"testing"

	"github.com/boskuv/goreminder/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestChangedFieldsFromMaps(t *testing.T) {
	before := map[string]interface{}{
		"title":   "old",
		"enabled": true,
	}
	after := map[string]interface{}{
		"title":      "new",
		"enabled":    true,
		"new_field":  1,
		"empty_field": "",
	}

	changed := changedFieldsFromMaps(before, after)
	assert.Equal(t, []string{"empty_field", "new_field", "title"}, changed)
}

func TestBuildAuditLogPayload_UsesFallbackActor(t *testing.T) {
	payload := buildAuditLogPayload(context.Background(), "updated", "task", 10, []string{"title"})

	assert.Equal(t, "updated", payload.Operation)
	assert.Equal(t, "task", payload.Entity)
	assert.Equal(t, int64(10), payload.EntityID)
	assert.Equal(t, unknownAuditActorID, payload.ActorID)
	assert.Equal(t, []string{"title"}, payload.ChangedFields)
}

func TestUserToAuditMap_MasksPasswordHashValue(t *testing.T) {
	user := &models.User{
		ID:           1,
		Name:         "John",
		Email:        "john@example.com",
		PasswordHash: "super-secret-hash",
	}

	mapped := userToAuditMap(user)

	assert.Equal(t, true, mapped["password_hash_set"])
	assert.NotContains(t, mapped, "password_hash")
}
