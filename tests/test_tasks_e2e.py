"""
E2E tests for Task API endpoints.

Run with: pytest tests/test_tasks_e2e.py -v
"""
import requests
import pytest

BASE_URL = "http://localhost:8080/api/v1"

@pytest.fixture
def user_id():
    """Create a test user and return user_id"""
    user_payload = {
        "name": "Test User for Tasks",
        "email": "taskuser@example.com",
        "password_hash": "test123"
    }
    resp = requests.post(f"{BASE_URL}/users", json=user_payload)
    assert resp.status_code == 201
    return resp.json()["id"]

def test_create_task(user_id):
    """Test creating a task and verify response structure"""
    payload = {
        "title": "Test Task",
        "description": "This is a test task",
        "user_id": user_id
    }
    response = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert response.status_code == 201
    data = response.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    assert data["id"] > 0

def test_get_task(user_id):
    """Test getting a task and verify TaskResponse DTO structure"""
    # First, create a task
    payload = {
        "title": "Get Task",
        "description": "Task to get",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    # Now, get the task
    get_resp = requests.get(f"{BASE_URL}/tasks/{task_id}")
    assert get_resp.status_code == 200
    data = get_resp.json()
    
    # Verify TaskResponse DTO structure
    assert "id" in data
    assert "title" in data
    assert "description" in data
    assert "user_id" in data
    assert "status" in data
    assert "start_date" in data
    assert "created_at" in data
    # Verify deleted_at is NOT exposed
    assert "deleted_at" not in data
    
    # Verify values
    assert data["id"] == task_id
    assert data["title"] == payload["title"]
    assert data["description"] == payload["description"]
    assert data["user_id"] == user_id
    assert isinstance(data["created_at"], str)  # ISO 8601 format or time.Time

def test_update_task(user_id):
    """Test updating a task and verify TaskResponse DTO structure"""
    # Create a task
    payload = {
        "title": "Update Task",
        "description": "Task to update",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    # Update the task
    update_payload = {
        "title": "Updated Task",
        "description": "Updated description"
    }
    update_resp = requests.put(f"{BASE_URL}/tasks/{task_id}", json=update_payload)
    assert update_resp.status_code == 200
    updated = update_resp.json()
    
    # Verify TaskResponse DTO structure
    assert "id" in updated
    assert "title" in updated
    assert "description" in updated
    assert "user_id" in updated
    assert "status" in updated
    assert "created_at" in updated
    
    # Verify updated values
    assert updated["title"] == "Updated Task"
    assert updated["description"] == "Updated description"
    assert updated["id"] == task_id

def test_update_task_partial(user_id):
    """Test partial update of a task"""
    # Create a task
    payload = {
        "title": "Partial Update Task",
        "description": "Task for partial update",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    # Partial update - only status
    update_payload = {
        "status": "done"
    }
    update_resp = requests.put(f"{BASE_URL}/tasks/{task_id}", json=update_payload)
    assert update_resp.status_code == 200
    updated = update_resp.json()
    
    # Verify only status was updated
    assert updated["status"] == "done"
    assert updated["title"] == payload["title"]  # Should remain unchanged

def test_delete_task(user_id):
    """Test deleting a task"""
    # Create a task
    payload = {
        "title": "Delete Task",
        "description": "Task to delete",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    # Delete the task
    del_resp = requests.delete(f"{BASE_URL}/tasks/{task_id}")
    assert del_resp.status_code == 204
    assert del_resp.text == ""  # No content

    # Ensure it's gone
    get_resp = requests.get(f"{BASE_URL}/tasks/{task_id}")
    assert get_resp.status_code == 404
    error_data = get_resp.json()
    assert "error" in error_data

def test_get_user_tasks(user_id):
    """Test getting all tasks for a user and verify response structure"""
    # Create two tasks for the user
    for i in range(2):
        payload = {
            "title": f"User Task {i}",
            "description": f"Task {i} for user",
            "user_id": user_id
        }
        resp = requests.post(f"{BASE_URL}/tasks", json=payload)
        assert resp.status_code == 201

    # Get all tasks for the user
    resp = requests.get(f"{BASE_URL}/users/{user_id}/tasks")
    assert resp.status_code == 200
    tasks = resp.json()
    assert isinstance(tasks, list)
    assert len(tasks) >= 2  # At least the two we just created
    
    # Verify each task has TaskResponse DTO structure
    for task in tasks:
        assert "id" in task
        assert "title" in task
        assert "user_id" in task
        assert "status" in task
        assert "created_at" in task
        assert "deleted_at" not in task

def test_get_task_history(user_id):
    """Test getting task history and verify TaskHistoryResponse DTO structure"""
    # Create a task
    payload = {
        "title": "History Task",
        "description": "Task to check history",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    # Update the task
    update_payload = {
        "status": "done"
    }
    requests.put(f"{BASE_URL}/tasks/{task_id}", json=update_payload)

    # Get task history
    history_resp = requests.get(f"{BASE_URL}/tasks/{task_id}/history")
    assert history_resp.status_code == 200
    history = history_resp.json()
    assert isinstance(history, list)
    assert len(history) >= 2  # At least created and updated
    
    # Verify TaskHistoryResponse DTO structure
    for entry in history:
        assert "id" in entry
        assert "task_id" in entry
        assert "user_id" in entry
        assert "action" in entry
        assert "created_at" in entry
        assert isinstance(entry["created_at"], str)  # ISO 8601 format

def test_get_user_task_history(user_id):
    """Test getting user task history and verify response structure"""
    # Create a task for the user
    payload = {
        "title": "User History Task",
        "description": "Task for user history",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201

    # Get user task history
    history_resp = requests.get(f"{BASE_URL}/users/{user_id}/tasks/history")
    assert history_resp.status_code == 200
    history = history_resp.json()
    assert isinstance(history, list)
    assert len(history) >= 1  # At least the created task
    
    # Verify TaskHistoryResponse DTO structure
    for entry in history:
        assert "id" in entry
        assert "task_id" in entry
        assert "user_id" in entry
        assert "action" in entry
        assert "created_at" in entry

def test_get_user_task_history_with_pagination(user_id):
    """Test getting user task history with pagination"""
    # Get user task history with pagination
    history_resp = requests.get(f"{BASE_URL}/users/{user_id}/tasks/history?limit=10&offset=0")
    assert history_resp.status_code == 200
    history = history_resp.json()
    assert isinstance(history, list)
    assert len(history) <= 10  # Should respect limit

def test_queue_task(user_id):
    """Test queueing a task and verify response structure"""
    # Create a task
    payload = {
        "title": "Queue Task",
        "description": "Task to queue",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]
    
    # Queue the task
    queue_payload = {
        "action": "schedule",
        "queue_name": "celery",
        "task_id": task_id
    }
    queue_resp = requests.post(f"{BASE_URL}/tasks/queue", json=queue_payload)
    assert queue_resp.status_code == 201
    data = queue_resp.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert data["id"] == task_id

def test_mark_task_as_done(user_id):
    """Test marking a task as done and verify TaskResponse DTO structure"""
    # Create a task
    payload = {
        "title": "Task to Complete",
        "description": "Task that will be marked as done",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]
    
    # Mark task as done
    done_resp = requests.post(f"{BASE_URL}/tasks/{task_id}/done")
    assert done_resp.status_code == 200
    data = done_resp.json()
    
    # Verify TaskResponse DTO structure
    assert "id" in data
    assert "title" in data
    assert "status" in data
    assert data["status"] == "done"
    assert data["id"] == task_id

def test_get_task_not_found():
    """Test getting a non-existent task"""
    resp = requests.get(f"{BASE_URL}/tasks/99999")
    assert resp.status_code == 404
    data = resp.json()
    assert "error" in data
    assert "not found" in data["error"].lower()

def test_create_task_invalid_user():
    """Test creating a task with invalid user_id"""
    payload = {
        "title": "Test Task",
        "description": "This task has invalid user",
        "user_id": 99999  # Non-existent user
    }
    resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert resp.status_code == 422  # Unprocessable entity
    data = resp.json()
    assert "error" in data

def test_create_task_missing_required_fields():
    """Test creating a task with missing required fields"""
    invalid_payload = {
        "description": "Missing title and user_id"
    }
    resp = requests.post(f"{BASE_URL}/tasks", json=invalid_payload)
    assert resp.status_code == 400
    data = resp.json()
    assert "error" in data
