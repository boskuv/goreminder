# GoReminder Python Client

A simple Python client library for interacting with the GoReminder API.

## Installation

1. Install Python 3.7 or higher
2. Install dependencies:

```bash
pip install -r requirements.txt
```

## Quick Start

```python
from datetime import datetime, timedelta, timezone
from goreminder_client import GoReminderClient

# Initialize client
client = GoReminderClient(base_url="http://localhost:8080")

# Create a user
user = client.create_user(
    name="John Doe",
    email="john.doe@example.com",
    timezone="UTC"
)

# Create a task
future_date = datetime.now(timezone.utc) + timedelta(days=1)
task = client.create_task(
    title="Complete project documentation",
    user_id=user["id"],
    start_date=future_date,
    description="Write comprehensive documentation",
    requires_confirmation=True
)

# Mark task as done
client.mark_task_as_done(task["id"])
```

## Running Examples

Run the example script to see various API operations:

```bash
python example_usage.py
```

Make sure the GoReminder API server is running at `http://localhost:8080` before running the examples.

## API Methods

### Tasks

- `create_task()` - Create a new task
- `get_task(task_id)` - Get a task by ID
- `get_all_tasks()` - Get all tasks with pagination and filtering
- `update_task(task_id, ...)` - Update a task (partial update)
- `delete_task(task_id)` - Delete a task (soft delete)
- `mark_task_as_done(task_id)` - Mark a task as done
- `get_task_history(task_id)` - Get task history
- `get_user_tasks(user_id, ...)` - Get all tasks for a user
- `get_user_task_history(user_id, ...)` - Get user task history
- `queue_task(task_id, action, queue_name)` - Queue a task for processing

### Users

- `create_user()` - Create a new user
- `get_user(user_id)` - Get a user by ID
- `get_all_users()` - Get all users with pagination
- `update_user(user_id, ...)` - Update a user (partial update)
- `delete_user(user_id)` - Delete a user (soft delete)

### Messengers

- `create_messenger(name)` - Create a messenger type
- `get_messenger(messenger_id)` - Get a messenger by ID
- `get_messenger_id_by_name(name)` - Get messenger ID by name
- `get_all_messengers()` - Get all messengers with pagination
- `create_messenger_related_user()` - Create a messenger user relation
- `get_messenger_related_user()` - Get messenger-related user
- `get_all_messenger_related_users()` - Get all messenger-related users
- `get_user_id_by_messenger_user_id()` - Get user ID by messenger user ID

### Backlogs

- `create_backlog()` - Create a new backlog item
- `create_backlogs_batch()` - Create multiple backlog items at once
- `get_backlog(backlog_id)` - Get a backlog by ID
- `get_all_backlogs()` - Get all backlogs with pagination
- `update_backlog(backlog_id, ...)` - Update a backlog item
- `delete_backlog(backlog_id)` - Delete a backlog item

### Digests

- `get_digest(user_id, date)` - Get digest for a user
- `create_digest_settings()` - Create digest settings
- `get_digest_settings(user_id)` - Get digest settings for a user
- `update_digest_settings(user_id, ...)` - Update digest settings
- `delete_digest_settings(user_id)` - Delete digest settings
- `get_all_digest_settings()` - Get all digest settings with pagination

## Examples

### Creating a One-time Task

```python
from datetime import datetime, timedelta, timezone

future_date = datetime.now(timezone.utc) + timedelta(days=1)
task = client.create_task(
    title="Complete project documentation",
    user_id=1,
    start_date=future_date,
    description="Write comprehensive documentation",
    requires_confirmation=True
)
```

### Creating a Recurring Task

```python
future_date = datetime.now(timezone.utc) + timedelta(days=1)
recurring_task = client.create_task(
    title="Daily standup reminder",
    user_id=1,
    start_date=future_date,
    cron_expression="0 9 * * *",  # Daily at 9 AM
    requires_confirmation=True
)
```

### Filtering Tasks

```python
from datetime import datetime, timedelta, timezone

start_from = datetime.now(timezone.utc)
start_to = datetime.now(timezone.utc) + timedelta(days=7)

tasks = client.get_all_tasks(
    page=1,
    page_size=20,
    status="pending",
    start_date_from=start_from,
    start_date_to=start_to,
    user_id=1,
    order_by="start_date ASC"
)
```

### Creating Backlogs in Batch

```python
items = "Implement feature A\nFix bug B\nWrite tests for C"
backlogs = client.create_backlogs_batch(
    items=items,
    user_id=1,
    separator="\n"
)
```

### Setting Up Digest

```python
digest_settings = client.create_digest_settings(
    user_id=1,
    weekday_time="07:00",  # 7 AM on weekdays
    weekend_time="10:00",  # 10 AM on weekends
    enabled=True
)
```

## Error Handling

The client raises `requests.HTTPError` for HTTP errors. Handle them appropriately:

```python
from requests import HTTPError

try:
    task = client.create_task(
        title="Test",
        user_id=1,
        start_date=datetime.now(timezone.utc) + timedelta(days=1)
    )
except HTTPError as e:
    print(f"HTTP Error: {e}")
    print(f"Response: {e.response.text}")
except Exception as e:
    print(f"Error: {e}")
```

## Date Handling

All dates must be in UTC and in the future for `start_date`. Use `datetime` with `timezone.utc`:

```python
from datetime import datetime, timedelta, timezone

# Correct: Future date in UTC
future_date = datetime.now(timezone.utc) + timedelta(days=1)

# Wrong: Past date
past_date = datetime.now(timezone.utc) - timedelta(days=1)  # Will fail validation
```

## Task Types

### One-time Task
- No `cron_expression`
- Executes once at `start_date`
- Can have `requires_confirmation=true` for rescheduling

### Recurring Task (Parent)
- Has `cron_expression` (e.g., `"0 9 * * *"`)
- Has `requires_confirmation=true`
- Creates child tasks automatically
- Does not execute directly

### Recurring Task (Child)
- Has `parent_id` pointing to parent
- No `cron_expression`
- Executes at calculated `start_date`
- When marked as done, creates next child task

See the main README for more details on task types and rescheduling behavior.

## Configuration

You can customize the base URL when initializing the client:

```python
# Default: http://localhost:8080
client = GoReminderClient()

# Custom URL
client = GoReminderClient(base_url="https://api.example.com")
```

## License

This client example is provided as-is for demonstration purposes.

