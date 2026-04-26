<p align="center">
<img src="https://github.com/user-attachments/assets/dd5747a5-1a14-440e-b443-a080d1b664a1" width="200" />
<h1 align="center">GoReminder</h1>
<p align="center">A comprehensive task management API built with Go</p>
<p align="center">
</p>

## Business Features
- [x] **Auto-rescheduling**: Tasks are automatically rescheduled to the next day if confirmation is not received
- [x] **Daily Task Digest**: Send daily task summaries at specified times
- [x] **Backlog Zone**: Tasks without fixed time and confirmation requirements
- [x] **Targets/Goals Management**: Create and track targets (aims/goals) with completion tracking
- [ ] **Task Muting**: Temporarily mute notifications for specific tasks
- [ ] **Task Postponement**: Ability to postpone tasks to a later time
- [ ] **Reminder Groups**: Group related tasks together for batch management
- [ ] **ICS Import**: Import tasks from iCalendar (.ics) files
- [ ] **Advanced Reminders**: Pre-reminders before task deadlines

## Tech Features
- **Task Management**: Create, fetch, update, and delete tasks with soft delete support
- **User Management**: Create, fetch, update, and delete users with soft delete support
- **Target Management**: Create, fetch, update, and delete targets (aims/goals) with soft delete support
- **Backlog Management**: Create, fetch, update, and delete backlog items with batch creation support
- **Messenger Integration**: Support for multiple messaging platforms (Telegram, etc.)
- **RESTful API**: Built with Gin framework for fast HTTP routing
- **Swagger API Documentation**: Auto-generated interactive API documentation
- **Pagination**: Built-in pagination support for list endpoints with page, page_size, and total_pages
- **Filtering & Ordering**: Advanced filtering and ordering capabilities for tasks
- **Request Validation**: Custom validators for cron expressions, task status, and future dates; recurring tasks additionally validate RRULE strings in the service layer when `rrule` is set
- **Observability**: 
  - **Metrics**: Prometheus metrics with HTTP request duration and count
  - **Tracing**: OpenTelemetry integration with Jaeger for distributed tracing
  - **Structured Logging**: Zerolog for structured, context-aware logging
- **Message Queue**: RabbitMQ integration for asynchronous task processing with retry support (can be disabled for DB-only mode)
- **Containerized Setup**: Complete Docker Compose configuration
- **Database Migrations**: Goose-based migration system
- **Comprehensive Testing**: Unit tests, integration tests, and E2E tests

## Prerequisites
- Docker and Docker Compose
- Go 1.24 or later
- `make` for build automation
- `golangci-lint` for code linting
- `goose` for database migrations
- `swag` for generating swagger docs
- `mockery` for generating mocks
- Python 3.x (for E2E tests)

## Project Structure
```
.
├── cmd/                    # Main applications of the project
│   └── core/              # The API server application
│       ├── main.go        # Application entry point
│       └── config.yaml    # Configuration file
├── docs/                  # Generated Swagger documentation
├── internal/              # Private application and library code
│   ├── api/               # API routes, handlers, and middleware
│   │   ├── dto/           # Data Transfer Objects
│   │   │   ├── mapper/    # DTO to Model mappers
│   │   │   └── *.go       # Request/Response DTOs
│   │   ├── handlers/      # HTTP request handlers
│   │   ├── middleware/    # HTTP middleware
│   │   │   ├── cors.go           # CORS middleware
│   │   │   ├── logger.go         # Request logging middleware
│   │   │   ├── metrics.go        # Prometheus metrics middleware
│   │   │   ├── ratelimit.go      # Rate limiting middleware
│   │   │   ├── request_id.go     # Request ID generation
│   │   │   └── tracing.go       # OpenTelemetry tracing middleware
│   │   ├── routes/        # Route definitions
│   │   └── validation/    # Custom validators
│   │       ├── custom_validators.go  # Custom validation rules
│   │       └── validator.go          # Validation utilities
│   ├── errors/            # Custom error definitions
│   ├── models/            # Data structures and domain models
│   ├── repository/        # Database interaction layer
│   ├── service/           # Business logic layer
│   └── mocks/             # Generated mock interfaces
├── migrations/            # Database migrations
├── pkg/                   # Public library code
│   ├── args/              # Command-line argument parsing
│   ├── config/            # Application configuration
│   ├── database/          # Database connection and retry logic
│   ├── logger/            # Structured logging
│   ├── observability/     # Metrics and tracing setup
│   └── queue/             # Message queue integration with retries
├── scripts/               # Database initialization scripts
└── tests/                 # Test suites (E2E tests)
```

## Queue Contracts

The API publishes messages to RabbitMQ using a simple, Celery-style JSON contract:

```json
{
  "task": "worker.schedule_task",
  "args": [
    "<messenger_name>",
    "<chat_id>",
    "<task_id>",
    "<title>",
    "<description>",
    "<start_date>",
    "<cron_expression|null>",
    "<requires_confirmation>"
  ]
}
```

This payload is represented in code by the low-level `queue.TaskMessage` struct and is sent via the `queue.Publisher` interface. At the domain level, task messages are modeled as `queue.TaskEvent` with a `TaskEventType` (`schedule_task`, `delete_task`, etc.), which are mapped to the Celery-style JSON. The seventh argument still reflects the task row’s `cron_expression` only; executable child tasks usually have both `cron_expression` and `rrule` unset, with the parent holding `rrule` in the database. Workers that need the RRULE string should resolve it via `task_id` (for example from the API or DB). For deployments that should not use RabbitMQ, set `producer.enabled: false` in the config – the application will then use a no-op publisher and work purely at the database level.

`internal/models.ScheduledTask` uses explicit action values (`"schedule"`, `"delete"`) via `ScheduledTaskActionSchedule` and `ScheduledTaskActionDelete` constants, which are dispatched in `TaskService.QueueTask` and converted into `queue.TaskEvent` instances before publishing.

## Database Schema

```mermaid
erDiagram
    users ||--o{ tasks : "has"
    users ||--o{ user_messengers : "has"
    users ||--o{ backlogs : "has"
    users ||--o{ targets : "has"
    users ||--o{ digest_settings : "has"
    users ||--o{ task_history : "has"
    
    messengers ||--o{ user_messengers : "has"
    
    user_messengers ||--o{ tasks : "references"
    user_messengers ||--o{ backlogs : "references"
    user_messengers ||--o{ targets : "references"
    user_messengers ||--o{ digest_settings : "references"
    
    tasks ||--o{ tasks : "parent-child"
    tasks ||--o{ task_history : "has"
    
    users {
        bigserial id PK
        varchar name
        varchar email UK
        text password_hash
        varchar timezone
        varchar language_code
        varchar role
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    tasks {
        bigserial id PK
        varchar title
        text description
        bigint user_id FK
        integer messenger_related_user_id FK
        bigint parent_id FK
        timestamp start_date
        timestamp finish_date
        varchar cron_expression
        text rrule
        boolean requires_confirmation
        varchar status
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    messengers {
        serial id PK
        varchar name UK
        timestamp created_at
    }
    
    user_messengers {
        serial id PK
        integer user_id FK
        integer messenger_id FK
        varchar chat_id
        varchar messenger_user_id
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }
    
    backlogs {
        bigserial id PK
        varchar title
        text description
        bigint user_id FK
        integer messenger_related_user_id FK
        timestamp created_at
        timestamp updated_at
        timestamp completed_at
        timestamp deleted_at
    }
    
    targets {
        bigserial id PK
        varchar title
        text description
        bigint user_id FK
        integer messenger_related_user_id FK
        timestamp created_at
        timestamp updated_at
        timestamp completed_at
        timestamp deleted_at
    }
    
    digest_settings {
        bigserial id PK
        bigint user_id FK
        integer messenger_related_user_id FK
        boolean enabled
        varchar weekday_time
        varchar weekend_time
        timestamp created_at
        timestamp updated_at
    }
    
    task_history {
        bigserial id PK
        bigint task_id FK
        bigint user_id FK
        varchar action
        jsonb old_value
        jsonb new_value
        timestamp created_at
    }
```

### Table Relationships

- **users** ↔ **tasks**: One-to-many (CASCADE delete)
- **users** ↔ **user_messengers**: One-to-many (CASCADE delete)
- **users** ↔ **backlogs**: One-to-many (CASCADE delete)
- **users** ↔ **targets**: One-to-many (CASCADE delete)
- **users** ↔ **digest_settings**: One-to-many (CASCADE delete)
- **users** ↔ **task_history**: One-to-many (CASCADE delete)
- **messengers** ↔ **user_messengers**: One-to-many (CASCADE delete)
- **user_messengers** ↔ **tasks**: One-to-many (SET NULL on delete)
- **user_messengers** ↔ **backlogs**: One-to-many (SET NULL on delete)
- **user_messengers** ↔ **targets**: One-to-many (SET NULL on delete)
- **user_messengers** ↔ **digest_settings**: One-to-many (SET NULL on delete)
- **tasks** ↔ **tasks**: Self-referential (parent-child relationship for recurring tasks)
- **tasks** ↔ **task_history**: One-to-many (CASCADE delete)

### Key Features

- **Soft Deletes**: `users`, `tasks`, `user_messengers`, `backlogs`, and `targets` support soft deletes via `deleted_at` column
- **Recurring Tasks**: Tasks can have a `parent_id` pointing to another task, enabling parent-child relationships for recurring tasks. Recurrence on the parent is defined by either `cron_expression` or `rrule` (iCalendar RRULE), never both.
- **Task History**: All task changes are tracked in `task_history` table with JSONB fields for old/new values
- **Messenger Integration**: Users can link multiple messenger accounts via `user_messengers` table
- **Digest Settings**: Users can configure daily digest times per messenger account

### Alternative Visualization

For a more detailed interactive diagram, you can use the [dbdiagram.io](https://dbdiagram.io) file located at `docs/database_schema.dbml`. Simply copy the contents and paste them into dbdiagram.io for an interactive ER diagram.

## Configuration

The application supports configuration via YAML files and environment variables. Environment variables take precedence over YAML file values. See `cmd/core/config.yaml.example` for a complete example.

### Configuration Sources

1. **YAML File** (default): Primary configuration source
2. **Environment Variables** (optional): Override YAML values

You can use either YAML file only, environment variables only, or a combination of both.

### Environment Variables

Environment variables use the prefix `GOREMINDER_` and nested keys are separated by underscores. For example:

- `GOREMINDER_SERVER_PORT=8080`
- `GOREMINDER_DATABASE_HOST=localhost`
- `GOREMINDER_DATABASE_PASSWORD=secret`
- `GOREMINDER_PRODUCER_HOST=rabbitmq`

**Examples:**

```bash
# Override specific values from YAML
export GOREMINDER_SERVER_PORT=9090
export GOREMINDER_DATABASE_HOST=production-db
export GOREMINDER_DATABASE_PASSWORD=secure-password

# Use only environment variables (no YAML file needed)
export GOREMINDER_SERVER_PORT=8080
export GOREMINDER_SERVER_SECRET=my-secret
export GOREMINDER_SERVER_MODE=production
export GOREMINDER_DATABASE_DRIVER=postgres
export GOREMINDER_DATABASE_DBNAME=mydb
export GOREMINDER_DATABASE_USERNAME=user
export GOREMINDER_DATABASE_PASSWORD=pass
export GOREMINDER_DATABASE_HOST=localhost
export GOREMINDER_DATABASE_PORT=5432
# ... set other required variables
```

**Note:** When using environment variables only, you can pass an empty string as config path: `./goreminder -config=""`

### Configuration Structure

```yaml
server:
  port: 8080                    # Server port
  mode: development            # development | production | test
  secret: dev-secret            # Application secret

database:
  driver: postgres              # Database driver
  dbname: task_manager          # Database name
  username: postgres            # Database username
  password: password            # Database password
  host: postgres                # Database host
  port: 5432                    # Database port
  maxOpenConns: 100             # Maximum open connections
  maxIdleConns: 10              # Maximum idle connections
  connMaxLifetime: 30m          # Connection max lifetime (duration format)
  maxRetries: 3                 # Maximum retry attempts for database operations

producer:
  enabled: true                 # Enable RabbitMQ producer (false = DB-only mode)
  host: rabbitmq                # RabbitMQ host
  port: 5672                    # RabbitMQ port
  user: guest                   # RabbitMQ username
  password: guest                # RabbitMQ password
  queueName: celery             # Queue name
  exchange: celery               # Exchange name
  connectionRetries: 5          # Connection retry attempts
  connectionRetryDelay: 2       # Delay between retries (seconds)

tracing:
  enabled: true                 # Enable OpenTelemetry tracing
  endpoint: localhost:4318      # OTLP endpoint
  serviceName: goreminder-api   # Service name for tracing
  insecure: true                 # Use insecure connection

metrics:
  enabled: true                 # Enable Prometheus metrics
  addr: :9191                   # Metrics server address

ratelimit:
  enabled: true                 # Enable rate limiting
  requests: 100                  # Requests allowed per window
  window: 1m                    # Time window (duration format)

cors:
  enabled: true                 # Enable CORS
  allowOrigins:                 # Allowed origins
    - "*"
  allowMethods:                 # Allowed HTTP methods
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
    - PATCH
  allowHeaders:                 # Allowed headers
    - Content-Type
    - Authorization
    - X-Request-ID
  exposeHeaders:                # Exposed headers
    - X-Request-ID
  allowCredentials: false       # Allow credentials
  maxAge: 3600                  # Preflight cache max age (seconds)

autoreschedule:
  enabled: false                # Enable automatic rescheduling of tasks whose startDate is already in the past
  time: "00:00"                 # Daily run time in UTC (HH:MM, 24-hour). Default 00:00 when enabled.
```

#### Autoreschedule Configuration

The `autoreschedule` option controls whether the task scheduler should run automatically to reschedule tasks.

**Behavior:**
- **When `enabled: false`** (default): The task scheduler is **not** started. Tasks are not automatically rescheduled.
- **When `enabled: true`**: The task scheduler runs daily at the specified UTC time to automatically reschedule tasks that need rescheduling.

**Time Format:**
- The `time` field specifies the UTC time when the scheduler should run daily (format: `HH:MM`, 24-hour format)
- Default value: `"00:00"` (midnight UTC) if `enabled: true` but `time` is not specified
- Valid format: `00:00` to `23:59`
- Example: `"14:30"` means the scheduler runs daily at 2:30 PM UTC

**What the Scheduler Does:**
- Finds tasks that need rescheduling (tasks with `startDate` in the past that require confirmation)
- Finds parent tasks with a recurrence rule (`cron_expression` or `rrule`) that need their `startDate` updated
- Automatically reschedules these tasks

**Use Cases:**
- Enable autoreschedule to automatically handle tasks that have passed their scheduled time
- Useful for maintaining task schedules without manual intervention
- When disabled, tasks must be manually rescheduled or updated

**Environment Variable:**
```bash
GOREMINDER_AUTORESCHEDULE_ENABLED=true
GOREMINDER_AUTORESCHEDULE_TIME=00:00
```

### Docker/Kubernetes Example

When deploying in containers, you can use environment variables instead of mounting config files:

```yaml
# docker-compose.yml example
services:
  api:
    image: goreminder:latest
    environment:
      - GOREMINDER_SERVER_PORT=8080
      - GOREMINDER_SERVER_MODE=production
      - GOREMINDER_DATABASE_HOST=postgres
      - GOREMINDER_DATABASE_DBNAME=task_manager
      - GOREMINDER_DATABASE_USERNAME=postgres
      - GOREMINDER_DATABASE_PASSWORD=${DB_PASSWORD}
      - GOREMINDER_PRODUCER_HOST=rabbitmq
      - GOREMINDER_PRODUCER_USER=guest
      - GOREMINDER_PRODUCER_PASSWORD=guest
      - GOREMINDER_AUTORESCHEDULE_ENABLED=true
      - GOREMINDER_AUTORESCHEDULE_TIME=00:00
```

Or in Kubernetes:

```yaml
# kubernetes deployment example
apiVersion: apps/v1
kind: Deployment
metadata:
  name: goreminder
spec:
  template:
    spec:
      containers:
      - name: api
        image: goreminder:latest
        env:
        - name: GOREMINDER_SERVER_PORT
          value: "8080"
        - name: GOREMINDER_DATABASE_HOST
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: host
        - name: GOREMINDER_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
```

## Setup Instructions

### 1. Clone the Repository
```bash
git clone https://github.com/boskuv/goreminder.git
cd goreminder
```

### 2. Configure the Application
```bash
# Copy example configuration
cp cmd/core/config.yaml.example cmd/core/config.yaml

# Edit configuration as needed
vim cmd/core/config.yaml
```

### 3. Run Services with Docker
```bash
make docker-up
# or
docker-compose up --build
```

### 4. Run Database Migrations

The application automatically runs database migrations on startup. However, you can run them manually if needed:

```bash
# Ensure goose is installed
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations manually
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=password dbname=task_manager sslmode=disable" up
```

**Note**: For local development, you can skip automatic migrations by setting the `SKIP_MIGRATIONS` environment variable:

```bash
# Skip migrations on startup (useful if you manage migrations manually)
SKIP_MIGRATIONS=true make run

# Or export it before running
export SKIP_MIGRATIONS=true
make run
```

You can also override the migrations directory using the `MIGRATIONS_DIR` environment variable:

```bash
# Use custom migrations directory
MIGRATIONS_DIR=custom_migrations make run
```

### 5. Run the Application

#### Debug Mode
Enable debug mode for enhanced logging and development features:
```bash
DEBUG=true make run
```

#### Release Mode
```bash
make run
```

### 6. Verify Services
- **PostgreSQL**: `localhost:5432`
- **API Server**: `localhost:8080`
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Metrics**: `http://localhost:9191/metrics`
- **Jaeger UI**: `http://localhost:16686`

## Middleware

The application uses multiple middleware layers (applied in order):

1. **Request ID Middleware**: Generates unique request IDs for tracing
   - Adds `X-Request-ID` header to requests/responses
   - Uses UUID v4 for request identification

2. **Logger Middleware**: Structured request logging
   - Logs method, path, status, latency, IP, user agent
   - Includes request ID in logs
   - Different log levels based on status codes

3. **CORS Middleware**: Cross-Origin Resource Sharing
   - Configurable allowed origins, methods, headers
   - Supports preflight requests
   - Can be enabled/disabled via configuration

4. **Rate Limit Middleware**: Request rate limiting
   - Token bucket algorithm
   - Limits by IP address
   - Configurable requests per time window
   - Can be enabled/disabled via configuration

5. **Metrics Middleware**: Prometheus metrics collection
   - HTTP request duration histogram
   - HTTP request count counter
   - Tagged by method, route, and status
   - Can be enabled/disabled via configuration

6. **Tracing Middleware**: OpenTelemetry distributed tracing
   - Automatic span creation for all requests
   - Integrates with Jaeger for visualization
   - Can be enabled/disabled via configuration

## Request Validation

The application includes custom validators for request validation:

### Custom Validators

1. **`cron`**: Validates cron expressions
   - Uses robfig/cron parser
   - Supports standard cron format
   - Example: `"0 9 * * *"` (daily at 9 AM)

2. **`rrule` (recurrence)**: When `rrule` is present on create/update, the service validates the string with [rrule-go](https://github.com/teambition/rrule-go) and rejects it if `cron_expression` is also set.

3. **`task_status`**: Validates task status values
   - Valid statuses: `pending`, `scheduled`, `done`, `rescheduled`, `postponed`, `deleted`
   - Returns HTTP 400 with descriptive error message

4. **`future_date`**: Validates that dates are in the future (UTC)
   - Ensures `start_date` is not in the past
   - Works with `time.Time` and `*time.Time` types
   - For task updates, validation is applied only when `start_date` is explicitly provided in payload (partial update semantics)
   - Returns HTTP 400 with error message: `"field 'start_date' must be a date in the future (UTC)"`

### Validation Error Handling

All validation errors return HTTP 400 (Bad Request) with descriptive error messages:
```json
{
  "error": "field 'start_date' must be a date in the future (UTC)"
}
```

## Observability

### Metrics
The application exposes Prometheus metrics at `/metrics` endpoint:
- **HTTP Request Duration**: Histogram of response times (`http_request_duration_seconds`)
- **HTTP Request Count**: Total number of requests by method, route, and status (`http_requests_total`)

### Tracing
OpenTelemetry tracing is integrated with Jaeger:
- Distributed tracing across all HTTP requests
- View traces in Jaeger UI at `http://localhost:16686`
- Automatic span creation for all API endpoints
- Request IDs are included in trace spans

### Logging
Structured logging using Zerolog:
- JSON-formatted logs
- Context-aware logging with request IDs
- Configurable log levels
- Request/response logging with latency tracking
- Audit-focused CRUD logs in service layer with unified `audit.*` attributes:
  - `audit.operation`, `audit.entity`, `audit.entity_id`
  - `audit.actor_id` (`unknown` when actor is not available in context)
  - `audit.changed_fields`, `audit.changed_count` for create/update/delete summaries
  - Sensitive values are not logged directly (for example, user password hash is represented as a boolean flag only)

## API Documentation

### Generate Swagger Docs
```bash
make swagger
```

Access Swagger UI at: `http://localhost:8080/swagger/index.html`

## API Endpoints

### Tasks

| Endpoint | Method | Description | Query Parameters |
|----------|--------|-------------|------------------|
| `/api/v1/tasks` | GET | Get all tasks with pagination | `page`, `page_size`, `order_by`, `status`, `start_date_from`, `start_date_to`, `user_id` |
| `/api/v1/tasks` | POST | Create a new task | - |
| `/api/v1/tasks/:id` | GET | Get task by ID | - |
| `/api/v1/tasks/:id` | PUT | Update task by ID | - |
| `/api/v1/tasks/:id` | DELETE | Soft delete task | - |
| `/api/v1/tasks/:id/history` | GET | Get task history | - |
| `/api/v1/tasks/:id/done` | POST | Mark task as done | - |
| `/api/v1/tasks/queue` | POST | Queue task for processing | - |
| `/api/v1/users/:user_id/tasks` | GET | Get all tasks for user | - |
| `/api/v1/users/:user_id/tasks/history` | GET | Get user task history | `limit`, `offset` |

### Users

| Endpoint | Method | Description | Query Parameters |
|----------|--------|-------------|------------------|
| `/api/v1/users` | GET | Get all users with pagination | `page`, `page_size`, `order_by` |
| `/api/v1/users` | POST | Create a new user | - |
| `/api/v1/users/:user_id` | GET | Get user by ID | - |
| `/api/v1/users/:user_id` | PUT | Update user by ID | - |
| `/api/v1/users/:user_id` | DELETE | Soft delete user | - |

### Messengers

| Endpoint | Method | Description | Query Parameters |
|----------|--------|-------------|------------------|
| `/api/v1/messengers` | GET | Get all messengers with pagination | `page`, `page_size`, `order_by` |
| `/api/v1/messengers` | POST | Create messenger type | - |
| `/api/v1/messengers/:messenger_id` | GET | Get messenger by ID | - |
| `/api/v1/messengers/by-name/:messenger_name` | GET | Get messenger ID by name | - |
| `/api/v1/messengerRelatedUsers` | GET | Get messenger-related user | `chat_id`, `messenger_user_id`, `user_id`, `messenger_id` |
| `/api/v1/messengerRelatedUsers` | POST | Create messenger user relation | - |
| `/api/v1/messengerRelatedUsers/all` | GET | Get all messenger-related users with pagination | `page`, `page_size`, `order_by` |
| `/api/v1/messengerRelatedUsers/:messenger_user_id/user` | GET | Get user ID by messenger user ID | - |

### Backlogs

| Endpoint | Method | Description | Query Parameters |
|----------|--------|-------------|------------------|
| `/api/v1/backlogs` | GET | Get all backlogs with pagination | `page`, `page_size`, `order_by`, `user_id` |
| `/api/v1/backlogs` | POST | Create a new backlog | - |
| `/api/v1/backlogs/batch` | POST | Create multiple backlogs at once | - |
| `/api/v1/backlogs/:id` | GET | Get backlog by ID | - |
| `/api/v1/backlogs/:id` | PUT | Update backlog by ID | - |
| `/api/v1/backlogs/:id` | DELETE | Delete backlog | - |

### Targets

| Endpoint | Method | Description | Query Parameters |
|----------|--------|-------------|------------------|
| `/api/v1/targets` | GET | Get all targets with pagination | `page`, `page_size`, `order_by`, `user_id` |
| `/api/v1/targets` | POST | Create a new target | - |
| `/api/v1/targets/:id` | GET | Get target by ID | - |
| `/api/v1/targets/:id` | PUT | Update target by ID | - |
| `/api/v1/targets/:id` | DELETE | Delete target (soft delete) | - |

### Digests

| Endpoint | Method | Description | Query Parameters |
|----------|--------|-------------|------------------|
| `/api/v1/digests` | GET | Get digest for user | `user_id`, `date` |
| `/api/v1/digests/settings` | POST | Create digest settings | - |
| `/api/v1/digests/settings` | GET | Get digest settings | `user_id` |
| `/api/v1/digests/settings` | PUT | Update digest settings | - |
| `/api/v1/digests/settings` | DELETE | Delete digest settings | `user_id` |
| `/api/v1/digests/settings/all` | GET | Get all digest settings | `page`, `page_size`, `order_by` |

## Task Types

GoReminder supports two types of tasks: **one-time tasks** and **recurring tasks**.

### Task Types Comparison

| Field | One-time Task | Recurring Task (Parent) | Recurring Task (Child) |
|-------|---------------|------------------------|------------------------|
| `cron_expression` / `rrule` | both `null` | Exactly one recurrence field: non-empty **cron** or non-empty **RRULE** (iCalendar rule string, e.g. `FREQ=DAILY;INTERVAL=1`). The two fields cannot both be set. | both `null` (next `start_date` is derived from the parent’s rule) |
| `requires_confirmation` | `true` | `true` or `false` | `true` only |
| `parent_id` | `null` | `null` | Points to parent task ID |
| Execution | Executes once at `start_date` | Does not execute directly | Executes at calculated `start_date` |
| Auto-creates child | No | Yes (on creation and when child is done) | No |

### How Recurring Tasks Work

1. **Parent Task**: Stores recurrence as either `cron_expression` **or** `rrule` (not both). With `requires_confirmation=true`, the parent does not execute directly; children carry the next occurrence (same model as before for cron-only parents).
2. **Child Task**: Created automatically from parent. Has `parent_id` pointing to parent. Executes at calculated `start_date`.
3. When a child task is marked as **done**, a new child task is created with `start_date` set to the parent’s next occurrence from **cron** or **RRULE**, whichever is configured.

### Reschedule Mechanism

Rescheduling occurs when a task with `requires_confirmation=true` is not confirmed by the user.

#### One-time Task (no `cron_expression` and no `rrule`, `requires_confirmation=true`)

1. Task is sent at `start_date`
2. User does not confirm
3. `RescheduleTask` is called:
   - `start_date` is moved forward by **24 hours**
   - `status` changes to `rescheduled`
   - Task is re-published to queue with new `start_date`

#### Recurring Task Child (has `parent_id`, `requires_confirmation=true`)

1. Child task is sent at `start_date`
2. User does not confirm
3. `RescheduleTask` is called:
   - Checks if new `start_date` (+24h) conflicts with the parent’s next occurrence (cron or RRULE)
   - **If conflict**: Rescheduling is skipped (parent will create a new child)
   - **If no conflict**: `start_date` is moved forward by 24 hours, task is re-published

#### Recurring Task Parent (has `cron_expression` or `rrule`, `requires_confirmation=false`)

1. `RescheduleCronTasks` is called for parent tasks without confirmation
2. `start_date` is updated to the next execution time from the parent’s cron or RRULE
3. No queue publishing (parent tasks don't execute directly)

### Mark as Done Behavior

#### Marking a Child Task as Done

1. Child task status → `done`, `finish_date` → now
2. `worker.delete_task` is queued
3. If parent exists and has a recurrence rule (`cron_expression` or `rrule`):
   - New child task is created with `start_date` = next occurrence from that rule

#### Marking a Parent Task as Done

1. Parent task status → `done`, `finish_date` → now
2. `worker.delete_task` is queued for parent
3. All non-done child tasks are:
   - Synced with parent's title/description
   - Marked as `done`
   - `worker.delete_task` is queued for each child

## Pagination

All list endpoints support pagination with the following query parameters:

- **`page`** (int, default: 1): Page number (1-indexed)
- **`page_size`** (int, default: 50): Number of items per page
- **`order_by`** (string, default: `created_at DESC`): Ordering clause (e.g., `name ASC`, `created_at DESC`)

### Pagination Response Format

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total_pages": 10,
    "total_count": 500
  }
}
```

## Filtering and Ordering

### Tasks Filtering

The `/api/v1/tasks` endpoint supports advanced filtering:

- **`status`** (string, optional): Filter by task status (`pending`, `scheduled`, `done`, `rescheduled`, `postponed`, `deleted`)
- **`start_date_from`** (string, optional): Filter tasks with `start_date >= start_date_from` (RFC3339 format)
- **`start_date_to`** (string, optional): Filter tasks with `start_date <= start_date_to` (RFC3339 format)
- **`user_id`** (int, optional): Filter tasks by user ID

**Example:**
```bash
GET /api/v1/tasks?page=1&page_size=20&status=pending&start_date_from=2024-01-01T00:00:00Z&start_date_to=2024-12-31T23:59:59Z&user_id=1&order_by=created_at DESC
```

### Ordering

All list endpoints support custom ordering via `order_by` parameter:
- Format: `field_name ASC` or `field_name DESC`
- Default: `created_at DESC`
- Examples: `name ASC`, `created_at DESC`, `updated_at ASC`

## HTTP Error Codes

- **400 Bad Request**: Invalid input data, malformed JSON, invalid parameters, validation errors
- **404 Not Found**: Resource not found (task, user, messenger)
- **422 Unprocessable Entity**: Business logic validation errors
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Unexpected server errors

## Example Requests

### Create a Task
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "New Task",
    "description": "Complete API development",
    "start_date": "2024-12-01T00:00:00Z",
    "user_id": 1,
    "cron_expression": "0 9 * * *"
  }'
```

**Note**: `start_date` must be in the future (UTC). Past dates will return HTTP 400.

For a recurring parent defined by RRULE instead of cron, send `rrule` (iCalendar string) and omit `cron_expression`, for example `"rrule": "FREQ=DAILY;INTERVAL=1"`.

### Get All Tasks with Filtering
```bash
curl -X GET "http://localhost:8080/api/v1/tasks?page=1&page_size=20&status=pending&start_date_from=2024-01-01T00:00:00Z&order_by=created_at DESC"
```

### Get All Users with Pagination
```bash
curl -X GET "http://localhost:8080/api/v1/users?page=1&page_size=50&order_by=name ASC"
```

### Create a User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john.doe@example.com",
    "name": "John Doe",
    "password_hash": "password123"
  }'
```

### Create a Messenger
```bash
curl -X POST http://localhost:8080/api/v1/messengers \
  -H "Content-Type: application/json" \
  -d '{
    "name": "telegram"
  }'
```

### Create a Backlog
```bash
curl -X POST http://localhost:8080/api/v1/backlogs \
  -H "Content-Type: application/json" \
  -d '{
    "title": "New backlog item",
    "description": "Task to be planned",
    "user_id": 1,
    "messenger_related_user_id": 123
  }'
```

### Create a Target
```bash
curl -X POST http://localhost:8080/api/v1/targets \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Learn Go programming",
    "description": "Master Go programming language",
    "user_id": 1,
    "messenger_related_user_id": 123
  }'
```

### Get Task by ID
```bash
curl http://localhost:8080/api/v1/tasks/1
```

### Get User by ID
```bash
curl http://localhost:8080/api/v1/users/1
```

### Update Task
```bash
curl -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "start_date": "2024-12-15T10:00:00Z"
  }'
```

**Note**: Updated `start_date` must be in the future (UTC).

**Recurring update behavior**: updating task fields like `title`/`description` without passing `start_date` does not auto-shift the task date. Child `start_date` recalculation is triggered only when schedule-related fields (`start_date`, `cron_expression`, `rrule`) are explicitly changed.

### Update User
```bash
curl -X PUT http://localhost:8080/api/v1/users/1 \
  -H "Content-Type: application/json" \
  -d '{
    "email": "updated@example.com"
  }'
```

### Delete Task (Soft Delete)
```bash
curl -X DELETE http://localhost:8080/api/v1/tasks/1
```

### Delete User (Soft Delete)
```bash
curl -X DELETE http://localhost:8080/api/v1/users/1
```

### Get All Targets with Pagination
```bash
curl -X GET "http://localhost:8080/api/v1/targets?page=1&page_size=50&order_by=created_at DESC&user_id=1"
```

### Update Target
```bash
curl -X PUT http://localhost:8080/api/v1/targets/1 \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated target title",
    "description": "Updated target description",
    "completed_at": "2024-12-20T10:00:00Z"
  }'
```

### Delete Target (Soft Delete)
```bash
curl -X DELETE http://localhost:8080/api/v1/targets/1
```

## Testing

### Unit Tests
Run unit tests with coverage:
```bash
make test
```

View coverage report:
```bash
make coverage
```

### Mock Generation
Generate mocks for testing:
```bash
# Install mockery if not already installed
go install github.com/vektra/mockery/v2@latest

# Generate mocks
mockery --dir internal/repository --output internal/mocks/repository
```

### E2E Tests
Run end-to-end tests (requires running application):

```bash
# Install Python dependencies
cd tests
pip install -r requirements.txt

# Run task E2E tests
python test_tasks_e2e.py

# Run user E2E tests
python test_users_e2e.py

# Run messenger E2E tests
python run_messenger_tests.py
```

### Test Structure
- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test service layer with mocked repositories
- **E2E Tests**: Test complete API workflows using Python requests

## Versioning

The project uses [Semantic Versioning](https://semver.org/) with version information managed through a `VERSION` file and build-time injection.

### Check Version

```bash
# Show version information
make version

# Check version via API
curl http://localhost:8080/version
```

### Update Version

1. Update `VERSION` file:
```bash
echo "0.7.0-rc.1" > VERSION
```

2. Update `CHANGELOG.md` (move changes from `[Unreleased]` to version section)

3. Build with version:
```bash
make build
```

4. Tag release:
```bash
git tag -a v0.7.0-rc.1 -m "Release v0.7.0-rc.1"
git push origin v0.7.0-rc.1
```

For detailed versioning guidelines, see [docs/versioning.md](docs/versioning.md).

## Development

### Code Quality
```bash
# Run linter
make lint

# Format code
go fmt ./...

# Run tests with coverage
make test
```

### Database Operations
```bash
# Check database connectivity
make db-check

# Run migrations manually
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=password dbname=task_manager sslmode=disable" up

# Rollback migrations
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=password dbname=task_manager sslmode=disable" down
```

**Note**: By default, migrations run automatically on application startup. To skip automatic migrations (useful for local development when managing migrations manually), set `SKIP_MIGRATIONS=true`:

```bash
# Run application without automatic migrations
SKIP_MIGRATIONS=true make run
```

### Docker Operations
```bash
# Start services
make docker-up

# Stop services
make docker-down

# Rebuild and start
docker-compose up --build
```

## Architecture

### Layers
1. **Handlers**: HTTP request/response handling, validation
2. **Services**: Business logic and orchestration
3. **Repository**: Data access layer with retry support
4. **Models**: Domain entities and DTOs

### Key Components
- **Gin Router**: Fast HTTP routing
- **PostgreSQL**: Primary database with connection pooling
- **RabbitMQ**: Message queue with retry support
- **OpenTelemetry**: Distributed tracing
- **Prometheus**: Metrics collection
- **Jaeger**: Trace visualization
- **Zerolog**: Structured logging

### Retry Mechanisms

1. **Database Retries**: Configurable retry attempts for database operations
   - Configured via `database.maxRetries` in config
   - Default: 3 retries

2. **Producer Retries**: Retry logic for RabbitMQ connection
   - Configured via `producer.connectionRetries` and `producer.connectionRetryDelay`
   - Default: 5 retries with 2 second delay

## Contributing
Feel free to open issues or pull requests to improve this project. Contributions are welcome!

### Development Guidelines
1. Follow Go coding standards
2. Write tests for new features
3. Update documentation
4. Run linter before committing
5. Ensure all tests pass
6. Add Swagger annotations for new endpoints
7. Follow the existing middleware and validation patterns

## License
This project is licensed under the MIT License.
