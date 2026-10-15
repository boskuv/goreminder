package models

import "time"

// SchedulableInventoryRow is a DB row for tasks expected to have a worker job.
type SchedulableInventoryRow struct {
	TaskID                 int64     `db:"task_id" json:"-"`
	Messenger              string    `db:"messenger" json:"-"`
	StartDate              time.Time `db:"start_date" json:"-"`
	Title                  string    `db:"title" json:"-"`
	CronExpression         *string   `db:"cron_expression" json:"-"`
	RRule                  *string   `db:"rrule" json:"-"`
	ParentID               *int64    `db:"parent_id" json:"-"`
	PreRemindBeforeSeconds *int64    `db:"pre_remind_before_seconds" json:"-"`
	RequiresConfirmation   bool      `db:"requires_confirmation" json:"-"`
}

// SchedulableInventoryItem is the API payload for schedule reconcile.
type SchedulableInventoryItem struct {
	TaskID    int64     `json:"task_id"`
	Messenger string    `json:"messenger"`
	StartDate time.Time `json:"start_date"`
	Kind      string    `json:"kind"` // one_shot | recurring | recurring_instance
	HasPre    bool      `json:"has_pre"`
	Title     string    `json:"title,omitempty"`
}
