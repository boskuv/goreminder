<p align="center">
<img src="https://github.com/user-attachments/assets/dd5747a5-1a14-440e-b443-a080d1b664a1" width="200" />
<h1 align="center">GoReminder</h1>
<p align="center">A comprehensive task management API built with Go</p>
<p align="center">
</p>

## Business Features
- [x] **Auto-rescheduling**: Tasks are automatically rescheduled to the next day if confirmation is not received
- [x] **Daily Task Digest**: Send daily task summaries at specified times
- [ ] **Task Muting**: Temporarily mute notifications for specific tasks
- [ ] **Task Postponement**: Ability to postpone tasks to a later time
- [ ] **Reminder Groups**: Group related tasks together for batch management
- [ ] **ICS Import**: Import tasks from iCalendar (.ics) files
- [ ] **Advanced Reminders**: Pre-reminders before task deadlines
- [ ] **Chaos Zone**: Tasks without fixed time and confirmation requirements

## Tech Features
- **Task Management**: Create, fetch, update, and delete tasks with soft delete support
- **User Management**: Create, fetch, update, and delete users with soft delete support
- **Messenger Integration**: Support for multiple messaging platforms (Telegram, etc.)
- **RESTful API**: Built with Gin framework for fast HTTP routing
- **Swagger API Documentation**: Auto-generated interactive API documentation
- **Pagination**: Built-in pagination support for list endpoints with page, page_size, and total_pages
- **Filtering & Ordering**: Advanced filtering and ordering capabilities for tasks
- **Request Validation**: Custom validators for cron expressions, task status, and future dates
- **Observability**: 
  - **Metrics**: Prometheus metrics with HTTP request duration and count
  - **Tracing**: OpenTelemetry integration with Jaeger for distributed tracing
  - **Structured Logging**: Zerolog for structured, context-aware logging
- **Message Queue**: RabbitMQ integration for asynchronous task processing with retry support
- **Containerized Setup**: Complete Docker Compose configuration
- **Database Migrations**: Goose-based migration system
- **Comprehensive Testing**: Unit tests, integration tests, and E2E tests

## Prerequisites
- Docker and Docker Compose
- Go 1.22 or later
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

## Configuration

The application uses YAML configuration files. See `cmd/core/config.yaml.example` for a complete example.

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
```bash
# Ensure goose is installed
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=password dbname=task_manager sslmode=disable" up
```

### 5. Run the Application
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

2. **`task_status`**: Validates task status values
   - Valid statuses: `pending`, `scheduled`, `done`, `rescheduled`, `postponed`, `deleted`
   - Returns HTTP 400 with descriptive error message

3. **`future_date`**: Validates that dates are in the future (UTC)
   - Ensures `start_date` is not in the past
   - Works with `time.Time` and `*time.Time` types
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
| `/api/v1/backlogs` | GET | Get all backlogs with pagination | `page`, `page_size`, `order_by` |
| `/api/v1/backlogs` | POST | Create a new backlog | - |
| `/api/v1/backlogs/batch` | POST | Create multiple backlogs at once | - |
| `/api/v1/backlogs/:id` | GET | Get backlog by ID | - |
| `/api/v1/backlogs/:id` | PUT | Update backlog by ID | - |
| `/api/v1/backlogs/:id` | DELETE | Delete backlog | - |

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
| `cron_expr` | `null` | Required (e.g., `"0 9 * * *"`) | `null` |
| `requires_confirmation` | `true` or `false` | `true` | `true` (inherited) |
| `parent_id` | `null` | `null` | Points to parent task ID |
| Execution | Executes once at `start_date` | Does not execute directly | Executes at calculated `start_date` |
| Auto-creates child | No | Yes (on creation and when child is done) | No |

### How Recurring Tasks Work

1. **Parent Task**: Contains `cron_expression` and `requires_confirmation=true`. Does not execute directly.
2. **Child Task**: Created automatically from parent. Has `parent_id` pointing to parent. Executes at calculated `start_date`.
3. When a child task is marked as **done**, a new child task is created with `start_date` calculated from parent's `cron_expression`.

### Reschedule Mechanism

Rescheduling occurs when a task with `requires_confirmation=true` is not confirmed by the user.

#### One-time Task (no `cron_expr`, `requires_confirmation=true`)

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
   - Checks if new `start_date` (+24h) conflicts with parent's next cron execution
   - **If conflict**: Rescheduling is skipped (parent will create a new child)
   - **If no conflict**: `start_date` is moved forward by 24 hours, task is re-published

#### Recurring Task Parent (has `cron_expr`, `requires_confirmation=false`)

1. `RescheduleCronTasks` is called for parent tasks without confirmation
2. `start_date` is updated to next cron execution time
3. No queue publishing (parent tasks don't execute directly)

### Mark as Done Behavior

#### Marking a Child Task as Done

1. Child task status → `done`, `finish_date` → now
2. `worker.delete_task` is queued
3. If parent exists and has `cron_expression`:
   - New child task is created with `start_date` = next cron execution time

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

# Run migrations
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=password dbname=task_manager sslmode=disable" up

# Rollback migrations
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=password dbname=task_manager sslmode=disable" down
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
