package queue

import "time"

// TaskEventType is a semantic type for task-related events sent via the queue.
type TaskEventType string

const (
	TaskEventSchedule TaskEventType = "schedule_task"
	TaskEventDelete   TaskEventType = "delete_task"
)

// TaskEvent is a high-level, typed contract for task-related events.
// It is mapped to the low-level TaskMessage used by workers (Celery-style).
type TaskEvent struct {
	Type                TaskEventType `json:"type"`
	TaskID              int64         `json:"task_id"`
	UserID              int64         `json:"user_id,omitempty"`
	MessengerName       string        `json:"messenger_name,omitempty"`
	ChatID              string        `json:"chat_id,omitempty"`
	Title               string        `json:"title,omitempty"`
	Description         string        `json:"description,omitempty"`
	StartDate           *time.Time    `json:"start_date,omitempty"`
	CronExpression      *string       `json:"cron_expression,omitempty"`
	RequiresConfirmation bool         `json:"requires_confirmation,omitempty"`
}

// ToTaskMessage converts a high-level TaskEvent into the concrete TaskMessage
// expected by the worker. This keeps the Celery-compatible JSON shape while
// giving services a typed contract.
func (e TaskEvent) ToTaskMessage() TaskMessage {
	switch e.Type {
	case TaskEventSchedule:
		// Celery worker expects:
		// [messenger_name, chat_id, task_id, title, description, start_date, cron_expression, requires_confirmation]
		return TaskMessage{
			Task: "worker.schedule_task",
			Args: []interface{}{
				e.MessengerName,
				e.ChatID,
				e.TaskID,
				e.Title,
				e.Description,
				e.StartDate,
				e.CronExpression,
				e.RequiresConfirmation,
			},
		}
	case TaskEventDelete:
		// Celery worker expects:
		// [task_id, messenger_name]
		return TaskMessage{
			Task: "worker.delete_task",
			Args: []interface{}{
				e.TaskID,
				e.MessengerName,
			},
		}
	default:
		// Fallback: keep behavior but make type visible in payload
		return TaskMessage{
			Task: string(e.Type),
			Args: []interface{}{
				e.TaskID,
				e.MessengerName,
				e.ChatID,
				e.Title,
				e.Description,
				e.StartDate,
				e.CronExpression,
				e.RequiresConfirmation,
			},
		}
	}
}

