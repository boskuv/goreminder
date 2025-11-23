"""
E2E tests for Messenger API endpoints.

Run with: pytest tests/test_messengers_e2e.py -v

Note: For standalone execution without pytest, use run_messenger_tests.py
"""
import pytest
import requests
import re

BASE_URL = "http://localhost:8080/api/v1"

@pytest.fixture
def messenger_payload():
    return {
        "name": "Test Messenger"
    }

@pytest.fixture
def messenger_related_user_payload():
    return {
        "user_id": 1,
        "messenger_id": 1,
        "messenger_user_id": "test_user_123",
        "chat_id": "test_chat_456"
    }

def test_create_messenger(messenger_payload):
    """Test creating a new messenger type and verify response structure"""
    response = requests.post(f"{BASE_URL}/messengers", json=messenger_payload)
    assert response.status_code == 201
    data = response.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    assert data["id"] > 0
    return data["id"]

def test_get_messenger(messenger_payload):
    """Test getting a messenger by ID and verify MessengerResponse DTO structure"""
    # First, create a messenger
    create_resp = requests.post(f"{BASE_URL}/messengers", json=messenger_payload)
    assert create_resp.status_code == 201
    messenger_id = create_resp.json()["id"]
    
    # Now, get the messenger
    get_resp = requests.get(f"{BASE_URL}/messengers/{messenger_id}")
    assert get_resp.status_code == 200
    data = get_resp.json()
    
    # Verify MessengerResponse DTO structure
    assert "name" in data
    assert "created_at" in data
    # ID should NOT be in response (it's json:"-" in model)
    assert "id" not in data
    
    # Verify values
    assert data["name"] == messenger_payload["name"]
    assert isinstance(data["created_at"], str)  # ISO 8601 format

def test_get_messenger_by_name(messenger_payload):
    """Test getting a messenger ID by name and verify response structure"""
    # First, create a messenger
    create_resp = requests.post(f"{BASE_URL}/messengers", json=messenger_payload)
    assert create_resp.status_code == 201
    
    # Get messenger ID by name
    get_resp = requests.get(f"{BASE_URL}/messengers/by-name/{messenger_payload['name']}")
    assert get_resp.status_code == 200
    data = get_resp.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    assert data["id"] > 0

def test_get_messenger_not_found():
    """Test getting a non-existent messenger"""
    response = requests.get(f"{BASE_URL}/messengers/99999")
    assert response.status_code == 404
    data = response.json()
    assert "error" in data
    assert "not found" in data["error"].lower()

def test_get_messenger_by_name_not_found():
    """Test getting a non-existent messenger by name"""
    response = requests.get(f"{BASE_URL}/messengers/by-name/nonexistent_messenger")
    assert response.status_code == 404
    data = response.json()
    assert "error" in data
    assert "not found" in data["error"].lower()

def test_create_messenger_related_user(messenger_related_user_payload):
    """Test creating a new messenger-related user and verify response structure"""
    response = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=messenger_related_user_payload)
    assert response.status_code == 201
    data = response.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    assert data["id"] > 0
    return data["id"]

def test_get_messenger_related_user(messenger_related_user_payload):
    """Test getting a messenger-related user and verify MessengerRelatedUserResponse DTO structure"""
    # First, create a messenger-related user
    create_resp = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=messenger_related_user_payload)
    assert create_resp.status_code == 201
    
    # Get the messenger-related user
    params = {
        "chat_id": messenger_related_user_payload["chat_id"],
        "messenger_user_id": messenger_related_user_payload["messenger_user_id"],
        "user_id": messenger_related_user_payload["user_id"],
        "messenger_id": messenger_related_user_payload["messenger_id"]
    }
    get_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers", params=params)
    assert get_resp.status_code == 200
    data = get_resp.json()
    
    # Verify MessengerRelatedUserResponse DTO structure
    assert "id" in data
    assert "user_id" in data
    assert "messenger_id" in data
    assert "messenger_user_id" in data
    assert "chat_id" in data
    assert "created_at" in data
    
    # Verify values
    assert data["chat_id"] == messenger_related_user_payload["chat_id"]
    assert data["messenger_user_id"] == messenger_related_user_payload["messenger_user_id"]
    assert data["user_id"] == messenger_related_user_payload["user_id"]
    assert data["messenger_id"] == messenger_related_user_payload["messenger_id"]
    assert isinstance(data["created_at"], str)  # ISO 8601 format

def test_get_messenger_related_user_not_found():
    """Test getting a non-existent messenger-related user"""
    params = {
        "chat_id": "nonexistent_chat",
        "messenger_user_id": "nonexistent_user",
        "user_id": 999,
        "messenger_id": 999
    }
    response = requests.get(f"{BASE_URL}/messengerRelatedUsers", params=params)
    assert response.status_code == 404
    data = response.json()
    assert "error" in data

def test_get_user_id_by_messenger_user_id(messenger_related_user_payload):
    """Test getting a user ID by messenger user ID and verify response structure"""
    # First, create a messenger-related user
    create_resp = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=messenger_related_user_payload)
    assert create_resp.status_code == 201
    
    # Get user ID by messenger user ID
    messenger_user_id = messenger_related_user_payload["messenger_user_id"]
    get_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers/{messenger_user_id}/user")
    assert get_resp.status_code == 200
    data = get_resp.json()
    # New format: {"user_id": ...}
    assert isinstance(data, dict)
    assert "user_id" in data
    assert isinstance(data["user_id"], int)
    assert data["user_id"] == messenger_related_user_payload["user_id"]

def test_get_user_id_by_messenger_user_id_not_found():
    """Test getting a user ID for non-existent messenger user ID"""
    response = requests.get(f"{BASE_URL}/messengerRelatedUsers/nonexistent_messenger_user/user")
    assert response.status_code == 404
    data = response.json()
    assert "error" in data

def test_create_messenger_invalid_data():
    """Test creating a messenger with invalid data"""
    invalid_payload = {
        "invalid_field": "invalid_value"
    }
    response = requests.post(f"{BASE_URL}/messengers", json=invalid_payload)
    assert response.status_code == 400
    data = response.json()
    assert "error" in data

def test_create_messenger_missing_required_fields():
    """Test creating a messenger with missing required fields"""
    invalid_payload = {}
    response = requests.post(f"{BASE_URL}/messengers", json=invalid_payload)
    assert response.status_code == 400
    data = response.json()
    assert "error" in data

def test_create_messenger_related_user_invalid_data():
    """Test creating a messenger-related user with invalid data"""
    invalid_payload = {
        "invalid_field": "invalid_value"
    }
    response = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=invalid_payload)
    assert response.status_code == 400
    data = response.json()
    assert "error" in data

def test_create_messenger_related_user_missing_required_fields():
    """Test creating a messenger-related user with missing required fields"""
    invalid_payload = {
        "user_id": 1
        # Missing messenger_id, messenger_user_id, chat_id
    }
    response = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=invalid_payload)
    assert response.status_code == 400
    data = response.json()
    assert "error" in data

def test_get_messenger_invalid_id():
    """Test getting a messenger with invalid ID"""
    response = requests.get(f"{BASE_URL}/messengers/invalid_id")
    assert response.status_code == 400
    data = response.json()
    assert "error" in data

def test_get_messenger_related_user_invalid_params():
    """Test getting a messenger-related user with invalid parameters"""
    params = {
        "chat_id": "test_chat",
        "messenger_user_id": "test_user",
        "user_id": "invalid_user_id",  # Should be integer
        "messenger_id": "invalid_messenger_id"  # Should be integer
    }
    response = requests.get(f"{BASE_URL}/messengerRelatedUsers", params=params)
    assert response.status_code == 400
    data = response.json()
    assert "error" in data

def test_get_messenger_related_user_optional_params():
    """Test getting a messenger-related user with optional parameters"""
    # Create a messenger-related user
    payload = {
        "user_id": 1,
        "messenger_id": 1,
        "messenger_user_id": "optional_params_test_123",
        "chat_id": "optional_params_test_456"
    }
    create_resp = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=payload)
    assert create_resp.status_code == 201
    
    # Get with only required params (user_id and messenger_id are optional)
    params = {
        "chat_id": payload["chat_id"],
        "messenger_user_id": payload["messenger_user_id"]
    }
    get_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers", params=params)
    assert get_resp.status_code == 200
    data = get_resp.json()
    assert data["chat_id"] == payload["chat_id"]

def test_messenger_workflow():
    """Test a complete workflow: create messenger, create related user, get user ID"""
    # Step 1: Create a messenger
    messenger_payload = {"name": "Workflow Messenger"}
    messenger_resp = requests.post(f"{BASE_URL}/messengers", json=messenger_payload)
    assert messenger_resp.status_code == 201
    messenger_id = messenger_resp.json()["id"]
    
    # Step 2: Create a messenger-related user
    related_user_payload = {
        "user_id": 1,
        "messenger_id": messenger_id,
        "messenger_user_id": "workflow_user_123",
        "chat_id": "workflow_chat_456"
    }
    related_user_resp = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=related_user_payload)
    assert related_user_resp.status_code == 201
    
    # Step 3: Get messenger by ID and verify structure
    get_messenger_resp = requests.get(f"{BASE_URL}/messengers/{messenger_id}")
    assert get_messenger_resp.status_code == 200
    messenger_data = get_messenger_resp.json()
    assert messenger_data["name"] == messenger_payload["name"]
    assert "created_at" in messenger_data
    assert "id" not in messenger_data  # ID should not be in response
    
    # Step 4: Get messenger by name and verify structure
    get_by_name_resp = requests.get(f"{BASE_URL}/messengers/by-name/{messenger_payload['name']}")
    assert get_by_name_resp.status_code == 200
    name_data = get_by_name_resp.json()
    assert "id" in name_data
    assert name_data["id"] == messenger_id
    
    # Step 5: Get messenger-related user and verify structure
    params = {
        "chat_id": related_user_payload["chat_id"],
        "messenger_user_id": related_user_payload["messenger_user_id"],
        "user_id": related_user_payload["user_id"],
        "messenger_id": related_user_payload["messenger_id"]
    }
    get_related_user_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers", params=params)
    assert get_related_user_resp.status_code == 200
    related_user_data = get_related_user_resp.json()
    assert related_user_data["chat_id"] == related_user_payload["chat_id"]
    assert "id" in related_user_data
    assert "created_at" in related_user_data
    
    # Step 6: Get user ID by messenger user ID and verify structure
    get_user_id_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers/{related_user_payload['messenger_user_id']}/user")
    assert get_user_id_resp.status_code == 200
    user_id_data = get_user_id_resp.json()
    assert "user_id" in user_id_data
    assert user_id_data["user_id"] == related_user_payload["user_id"]

if __name__ == "__main__":
    """
    This file is designed to be run with pytest:
      pytest tests/test_messengers_e2e.py -v
    
    For standalone execution, use run_messenger_tests.py instead.
    """
    import sys
    print("=" * 60)
    print("Messenger E2E Tests (Pytest)")
    print("=" * 60)
    print()
    print("This file is designed to be run with pytest.")
    print("Run with: pytest tests/test_messengers_e2e.py -v")
    print()
    print("For standalone execution, use: python tests/run_messenger_tests.py")
    print()
    sys.exit(1)
