"""
E2E tests for User API endpoints.

Run with: pytest tests/test_users_e2e.py -v
"""
import pytest
import requests

BASE_URL = "http://localhost:8080/api/v1/users"

@pytest.fixture
def user_payload():
    return {
        "name": "Test User",
        "email": "testuser@example.com",
        "password_hash": "s3cr3t"
    }

def test_create_user(user_payload):
    """Test creating a user and verify response structure"""
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    data = resp.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    assert data["id"] > 0
    return data["id"]

def test_get_user(user_payload):
    """Test getting a user and verify response DTO structure"""
    # First, create a user
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    user_id = resp.json()["id"]
    
    # Now, get the user
    resp = requests.get(f"{BASE_URL}/{user_id}")
    assert resp.status_code == 200
    data = resp.json()
    
    # Verify UserResponse DTO structure
    assert "id" in data
    assert "name" in data
    assert "email" in data
    assert "created_at" in data
    # Verify password_hash is NOT exposed
    assert "password_hash" not in data
    assert "deleted_at" not in data
    
    # Verify values
    assert data["id"] == user_id
    assert data["name"] == user_payload["name"]
    assert data["email"] == user_payload["email"]
    assert isinstance(data["created_at"], str)  # ISO 8601 format

def test_update_user(user_payload):
    """Test updating a user and verify response DTO structure"""
    # Create user
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    user_id = resp.json()["id"]
    
    # Update user
    update_payload = {
        "name": "Updated Name",
        "email": "updated@example.com"
    }
    resp = requests.put(f"{BASE_URL}/{user_id}", json=update_payload)
    assert resp.status_code == 200
    data = resp.json()
    
    # Verify UserResponse DTO structure
    assert "id" in data
    assert "name" in data
    assert "email" in data
    assert "created_at" in data
    assert "password_hash" not in data
    assert "deleted_at" not in data
    
    # Verify updated values
    assert data["name"] == "Updated Name"
    assert data["email"] == "updated@example.com"
    assert data["id"] == user_id

def test_update_user_partial(user_payload):
    """Test partial update of a user"""
    # Create user
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    user_id = resp.json()["id"]
    
    # Partial update - only name
    update_payload = {
        "name": "Partially Updated Name"
    }
    resp = requests.put(f"{BASE_URL}/{user_id}", json=update_payload)
    assert resp.status_code == 200
    data = resp.json()
    
    # Verify only name was updated
    assert data["name"] == "Partially Updated Name"
    assert data["email"] == user_payload["email"]  # Should remain unchanged

def test_delete_user(user_payload):
    """Test deleting a user"""
    # Create user
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    user_id = resp.json()["id"]
    
    # Delete user
    resp = requests.delete(f"{BASE_URL}/{user_id}")
    assert resp.status_code == 204
    assert resp.text == ""  # No content
    
    # Try to get deleted user
    resp = requests.get(f"{BASE_URL}/{user_id}")
    assert resp.status_code == 404
    error_data = resp.json()
    assert "error" in error_data

def test_get_user_not_found():
    """Test getting a non-existent user"""
    resp = requests.get(f"{BASE_URL}/99999")
    assert resp.status_code == 404
    data = resp.json()
    assert "error" in data
    assert "not found" in data["error"].lower()

def test_create_user_invalid_email():
    """Test creating a user with invalid email"""
    invalid_payload = {
        "name": "Test User",
        "email": "invalid-email",  # Invalid email format
        "password_hash": "s3cr3t"
    }
    resp = requests.post(BASE_URL, json=invalid_payload)
    assert resp.status_code == 400
    data = resp.json()
    assert "error" in data

def test_create_user_missing_required_fields():
    """Test creating a user with missing required fields"""
    invalid_payload = {
        "name": "Test User"
        # Missing email and password_hash
    }
    resp = requests.post(BASE_URL, json=invalid_payload)
    assert resp.status_code == 400
    data = resp.json()
    assert "error" in data
