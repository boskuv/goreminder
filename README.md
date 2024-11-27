useful commands:
- `swag init --dir ./cmd/core,./internal/api/handlers,./internal/models --output ./docs`

Test data for handlers:
```user
{
  "createdAt": "2024-11-27T10:00:00Z",
  "email": "john.doe@example.com",
  "id": 1,
  "name": "John Doe",
  "passwordHash": "password123"
}
```

```task
{
  "created_at": "2024-11-27T10:00:00Z",
  "description": "string",
  "due_date": "2024-11-27T10:00:00Z",
  "id": 0,
  "status": "pending",
  "title": "test",
  "user_id": 2
}
```
