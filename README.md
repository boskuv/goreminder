<p align="center">
<img src="https://github.com/user-attachments/assets/dd5747a5-1a14-440e-b443-a080d1b664a1" width="200" />
<h1 align="center">GoReminder</h1>
<p align="center">A comprehensive task management API built with Go</p>
<p align="center">
</p>

## Features
- **Task Management**: Create, fetch, update, and delete tasks with soft delete support
- **User Management**: Create, fetch, update, and delete users with soft delete support
- **Messenger Integration**: Support for multiple messaging platforms (Telegram, etc.)
- **PostgreSQL with PgBouncer**: Efficient connection pooling for high performance
- **RESTful API**: Built with Gin framework for fast HTTP routing
- **Swagger API Documentation**: Auto-generated interactive API documentation
- **Observability**: 
  - **Metrics**: Prometheus metrics with HTTP request duration and count
  - **Tracing**: OpenTelemetry integration with Jaeger for distributed tracing
  - **Structured Logging**: Zerolog for structured, context-aware logging
- **Message Queue**: RabbitMQ integration for asynchronous task processing
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
├── docs/                  # Generated Swagger documentation
├── internal/              # Private application and library code
│   ├── api/               # API routes, handlers, and middleware
│   │   ├── handlers/      # HTTP request handlers
│   │   ├── middleware/    # HTTP middleware (metrics, tracing)
│   │   └── routes/        # Route definitions
│   ├── errors/            # Custom error definitions
│   ├── models/            # Data structures and domain models
│   ├── repository/        # Database interaction layer
│   ├── service/           # Business logic layer
│   └── mocks/             # Generated mock interfaces
├── migrations/            # Database migrations
├── pkg/                   # Public library code
│   ├── args/              # Command-line argument parsing
│   ├── config/            # Application configuration
│   ├── logger/            # Structured logging
│   ├── observability/     # Metrics and tracing setup
│   └── queue/             # Message queue integration
├── scripts/               # Database initialization scripts
└── tests/                 # Test suites (E2E tests)
```

## Setup Instructions

### 1. Clone the Repository
```bash
git clone https://github.com/boskuv/goreminder.git
cd goreminder
```

### 2. Run Services with Docker
```bash
make docker-up
# or
docker-compose up --build
```

### 3. Run Database Migrations
```bash
# Ensure goose is installed
go install github.com/pressly/goose/v3/cmd/goose@latest

# Run migrations
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=password dbname=task_manager sslmode=disable" up
```

### 4. Run the Application
```bash
make run
```

### 5. Verify Services
- **PostgreSQL**: `localhost:5432`
- **PgBouncer**: `localhost:6432`
- **API Server**: `localhost:8080`
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Metrics**: `http://localhost:9191/metrics`
- **Jaeger UI**: `http://localhost:16686`

## Observability

### Metrics
The application exposes Prometheus metrics at `/metrics` endpoint:
- **HTTP Request Duration**: Histogram of response times
- **HTTP Request Count**: Total number of requests by method, route, and status

### Tracing
OpenTelemetry tracing is integrated with Jaeger:
- Distributed tracing across all HTTP requests
- View traces in Jaeger UI at `http://localhost:16686`
- Automatic span creation for all API endpoints

### Logging
Structured logging using Zerolog:
- JSON-formatted logs
- Context-aware logging with request IDs
- Configurable log levels

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

## API Documentation

### Generate Swagger Docs
```bash
make swagger
```

Access Swagger UI at: `http://localhost:8080/swagger/index.html`

## API Handlers Reference

| Endpoint | Method | Handler | Description | Success Codes | Error Codes |
|----------|--------|---------|-------------|---------------|-------------|
| `/api/v1/tasks` | POST | `CreateTask` | Create a new task | 201 | 400, 422, 500 |
| `/api/v1/tasks/:id` | GET | `GetTask` | Get task by ID | 200 | 400, 404, 500 |
| `/api/v1/tasks/:id` | PUT | `UpdateTask` | Update task by ID | 200 | 400, 404, 422, 500 |
| `/api/v1/tasks/:id` | DELETE | `DeleteTask` | Soft delete task | 204 | 400, 404, 500 |
| `/api/v1/users/:user_id/tasks` | GET | `GetUserTasks` | Get all tasks for user | 200 | 400, 422, 500 |
| `/api/v1/users` | POST | `CreateUser` | Create a new user | 201 | 400, 500 |
| `/api/v1/users/:user_id` | GET | `GetUser` | Get user by ID | 200 | 400, 404, 500 |
| `/api/v1/users/:user_id` | PUT | `UpdateUser` | Update user by ID | 200 | 400, 404, 422, 500 |
| `/api/v1/users/:user_id` | DELETE | `DeleteUser` | Soft delete user | 204 | 400, 404, 500 |
| `/api/v1/messengers` | POST | `CreateMessenger` | Create messenger type | 201 | 400, 500 |
| `/api/v1/messengers/:messenger_id` | GET | `GetMessenger` | Get messenger by ID | 200 | 400, 404, 500 |
| `/api/v1/messengers/by-name/:messenger_name` | GET | `GetMessengerIDByName` | Get messenger ID by name | 200 | 404, 500 |
| `/api/v1/messengerRelatedUsers` | POST | `CreateMessengerRelatedUser` | Create messenger user relation | 201 | 400, 422, 500 |
| `/api/v1/messengerRelatedUsers` | GET | `GetMessengerRelatedUser` | Get messenger user relation | 200 | 400, 404, 422, 500 |
| `/api/v1/messengerRelatedUsers/:messenger_user_id/user` | GET | `GetUserID` | Get user ID by messenger user ID | 200 | 404, 500 |

### HTTP Error Codes
- **400 Bad Request**: Invalid input data, malformed JSON, invalid parameters
- **404 Not Found**: Resource not found (task, user, messenger)
- **422 Unprocessable Entity**: Business logic validation errors
- **500 Internal Server Error**: Unexpected server errors

## Example Requests

### Create a Task
```bash
curl -X POST http://localhost:8080/api/v1/tasks \
-H "Content-Type: application/json" \
-d '{
  "title": "New Task",
  "description": "Complete API development",
  "due_date": "2024-12-01T00:00:00Z",
  "user_id": 1
}'
```

### Create a User
```bash
curl -X POST http://localhost:8080/api/v1/users \
-H "Content-Type: application/json" \
-d '{
  "email": "john.doe@example.com",
  "name": "John Doe",
  "passwordHash": "password123"
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

### Get All Tasks for User
```bash
curl -X 'GET' \
  'http://localhost:8080/api/v1/users/1/tasks' \
  -H 'accept: application/json'
```

### Update Task
```bash
curl -X 'PUT' \
  'http://localhost:8080/api/v1/tasks/1' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "description": "Updated description"
}'
```

### Update User
```bash
curl -X 'PUT' \
  'http://localhost:8080/api/v1/users/1' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "email": "updated@example.com"
}'
```

### Delete Task (Soft Delete)
```bash
curl -X 'DELETE' \
  'http://localhost:8080/api/v1/tasks/1' \
  -H 'accept: application/json'
```

### Delete User (Soft Delete)
```bash
curl -X 'DELETE' \
  'http://localhost:8080/api/v1/users/1' \
  -H 'accept: application/json'
```

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
1. **Handlers**: HTTP request/response handling
2. **Services**: Business logic and orchestration
3. **Repository**: Data access layer
4. **Models**: Domain entities and DTOs

### Key Components
- **Gin Router**: Fast HTTP routing
- **PostgreSQL**: Primary database
- **PgBouncer**: Connection pooling
- **RabbitMQ**: Message queue (configured but not active)
- **OpenTelemetry**: Distributed tracing
- **Prometheus**: Metrics collection
- **Jaeger**: Trace visualization

## Contributing
Feel free to open issues or pull requests to improve this project. Contributions are welcome!

### Development Guidelines
1. Follow Go coding standards
2. Write tests for new features
3. Update documentation
4. Run linter before committing
5. Ensure all tests pass

## License
This project is licensed under the MIT License.
