package queue

// TaskMessage represents a Celery-like task invocation payload.
// It preserves the existing JSON structure:
//
//	{
//	  "task": "worker.schedule_task",
//	  "args": [...]
//	}
//
// so consumers are not affected while producers use a typed contract.
type TaskMessage struct {
	Task string        `json:"task"`
	Args []interface{} `json:"args"`
}

