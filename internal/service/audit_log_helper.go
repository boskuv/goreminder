package service

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/rs/zerolog"
)

const unknownAuditActorID = "unknown"

type auditLogPayload struct {
	Operation     string
	Entity        string
	EntityID      int64
	ActorID       string
	RequestID     string
	ChangedFields []string
}

func buildAuditLogPayload(ctx context.Context, operation, entity string, entityID int64, changedFields []string) auditLogPayload {
	return auditLogPayload{
		Operation:     operation,
		Entity:        entity,
		EntityID:      entityID,
		ActorID:       auditActorIDFromContext(ctx),
		RequestID:     requestIDFromContext(ctx),
		ChangedFields: sanitizeAuditFields(changedFields),
	}
}

func withAuditLog(event *zerolog.Event, payload auditLogPayload) *zerolog.Event {
	event = event.
		Str("audit.operation", payload.Operation).
		Str("audit.entity", payload.Entity).
		Int64("audit.entity_id", payload.EntityID).
		Str("audit.actor_id", payload.ActorID).
		Int("audit.changed_count", len(payload.ChangedFields)).
		Strs("audit.changed_fields", payload.ChangedFields)

	if payload.RequestID != "" {
		event = event.Str("request_id", payload.RequestID)
	}

	return event
}

func changedFieldsFromMaps(before, after map[string]interface{}) []string {
	if len(before) == 0 && len(after) == 0 {
		return nil
	}

	keySet := make(map[string]struct{}, len(before)+len(after))
	for key := range before {
		keySet[key] = struct{}{}
	}
	for key := range after {
		keySet[key] = struct{}{}
	}

	changed := make([]string, 0, len(keySet))
	for key := range keySet {
		if !reflect.DeepEqual(before[key], after[key]) {
			changed = append(changed, key)
		}
	}

	sort.Strings(changed)
	return changed
}

func mapKeysForAudit(values map[string]interface{}) []string {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sanitizeAuditFields(fields []string) []string {
	if len(fields) == 0 {
		return nil
	}

	sanitized := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		sanitized = append(sanitized, field)
	}
	sort.Strings(sanitized)
	return sanitized
}

func auditActorIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return unknownAuditActorID
	}

	if actor, ok := ctx.Value("actor_id").(string); ok && actor != "" {
		return actor
	}
	if actor, ok := ctx.Value("user_id").(int64); ok {
		return fmt.Sprintf("%d", actor)
	}
	if actor, ok := ctx.Value("user_id").(string); ok && actor != "" {
		return actor
	}

	return unknownAuditActorID
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}

	return ""
}
