<p align="center">
<img src="https://github.com/user-attachments/assets/dd5747a5-1a14-440e-b443-a080d1b664a1" width="200" />
<h1 align="center">GoReminder</h1>
<p align="center">A task management API built with Go</p>
<p align="center">
</p>

## Features
- **Task Management**: Create, fetch, and manage tasks
- **User Management**: Create, fetch and manage users
- **PostgreSQL with PgBouncer**: Efficient connection pooling
- **RESTful API**: Built with Gin framework
- **Swagger API Docs**: Auto-generated documentation
- **Containerized Setup**: Docker Compose configuration

## Prerequisites
- Docker and Docker Compose
- Go 1.22 or later
- `make` for providing commands
- `golangci-lint` for code linting
- `goose` for database migrations
- `swag` for generating swagger docs
- `mockery` for generating mocks


## Project Structure
```
.
├── cmd                  main applications of the project
│   └── core             the API server application
├── docs                 generated Swagger documentation
├── internal             private application and library code
│   ├── api              API routes, handlers, and middleware
│   ├── models           data structures and models for domain logic
│   ├── repository       interacting with the database
│   ├── service          business logic and orchestrates data flow between the repository layer and handlers
├── migrations           database migrations
├── pkg                  public library code
│   ├── args             command-line argument parsing
│   ├── config           application configuration
│   ├── logger           structured and context-aware logger
├── scripts              public library code
└── tests                test data scripts
```

## Setup Instructions

### 1. Clone the Repository
```bash
git clone https://github.com/boskuv/goreminder.git
cd goreminder
```
### 2. Run Services with Docker
```bash
docker-compose up --build
```
### 3. Run web server
```bash
make run
```
### 4. Verify Services
- PostgreSQL: Accessible on port 5432
- PgBouncer: Accessible on port 6432
- API: Accessible on port 8080

## Generate Swagger Docs
### 1. Generate docs
```bash
make swagger
```
### 2. Run web server and verify that Swagger UI is available at
```
http://localhost:8080/docs/index.html
```

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
### Fetch a Task
```bash
curl http://localhost:8080/api/v1/tasks/1
```
### Fetch a User
```bash
curl http://localhost:8080/api/v1/users/1
```
### Fetch all tasks of a User
```bash
curl -X 'GET' \
  'http://localhost:8080/api/v1/users/1/tasks' \
  -H 'accept: application/json'
```
### Update a task's description (choose any field)
```bash
curl -X 'PUT' \
  'http://localhost:8080/api/v1/tasks/1' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "description": "test"
}'
```
### Update a user's email (choose any field)
```bash
curl -X 'PUT' \
  'http://localhost:8080/api/v1/users/1' \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
  "email": "test@test.com"
}'
```
### Delete a task
```bash
curl -X 'DELETE' \
  'http://localhost:8080/api/v1/tasks/1' \
  -H 'accept: application/json'
```
### Delete a user
```bash
curl -X 'DELETE' \
  'http://localhost:8080/api/v1/users/1' \
  -H 'accept: application/json'
```

## Contributions
Feel free to open issues or pull requests to improve this project. Contributions are welcome!

## License
This project is licensed under the MIT License.
