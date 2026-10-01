package models

import (
	"encoding/json"
	"time"
)

// SyncOutboxStatus is the processing state of an outbox row.
type SyncOutboxStatus string

const (
	SyncOutboxStatusPending    SyncOutboxStatus = "pending"
	SyncOutboxStatusProcessing SyncOutboxStatus = "processing"
	SyncOutboxStatusDone       SyncOutboxStatus = "done"
	SyncOutboxStatusFailed     SyncOutboxStatus = "failed"
)

// SyncOutbox queues asynchronous calendar sync work.
type SyncOutbox struct {
	ID          int64            `db:"id" json:"id"`
	Kind        string           `db:"kind" json:"kind"`
	Payload     json.RawMessage  `db:"payload" json:"payload"`
	Attempts    int              `db:"attempts" json:"attempts"`
	NextRetryAt time.Time        `db:"next_retry_at" json:"next_retry_at"`
	LastError   *string          `db:"last_error" json:"last_error,omitempty"`
	Status      SyncOutboxStatus `db:"status" json:"status"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at" json:"updated_at"`
}
