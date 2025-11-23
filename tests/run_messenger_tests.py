#!/usr/bin/env python3
"""
Simple test runner for messenger E2E tests
Run with: python run_messenger_tests.py

Note: This is a standalone test runner. For pytest, use test_messengers_e2e.py directly.
"""

import requests
import sys
import time
import traceback
import inspect

BASE_URL = "http://localhost:8080/api/v1"

def run_test(test_name, test_func):
    """Run a single test and report results with detailed error information"""
    print(f"Running {test_name}...", end=" ")
    try:
        test_func()
        print("✓ PASSED")
        return True
    except AssertionError as e:
        print("✗ FAILED")
        print(f"  Assertion Error: {e}")
        # Print the assertion details
        if hasattr(e, 'args') and e.args:
            print(f"  Details: {e.args[0]}")
        # Try to get more context from the assertion
        try:
            frame = inspect.trace()[-1]
            if 'response' in frame[0].f_locals:
                resp = frame[0].f_locals['response']
                print(f"  Request URL: {resp.url if hasattr(resp, 'url') else 'N/A'}")
                print(f"  Status Code: {resp.status_code if hasattr(resp, 'status_code') else 'N/A'}")
                try:
                    print(f"  Response Body: {resp.json()}")
                except:
                    print(f"  Response Text: {resp.text[:200] if hasattr(resp, 'text') else 'N/A'}")
        except:
            pass
        return False
    except requests.exceptions.RequestException as e:
        print("✗ FAILED")
        print(f"  Request Error: {type(e).__name__}: {e}")
        if hasattr(e, 'response') and e.response is not None:
            print(f"  Request URL: {e.response.url}")
            print(f"  Status Code: {e.response.status_code}")
            try:
                error_data = e.response.json()
                if "error" in error_data:
                    print(f"  API Error: {error_data['error']}")
                else:
                    print(f"  Response: {error_data}")
            except:
                print(f"  Response Text: {e.response.text[:200]}")
        elif hasattr(e, 'request'):
            print(f"  Request URL: {e.request.url if hasattr(e.request, 'url') else 'N/A'}")
        return False
    except Exception as e:
        print("✗ FAILED")
        print(f"  Error: {type(e).__name__}: {e}")
        # Print traceback for unexpected errors
        print("  Traceback:")
        tb_lines = traceback.format_exc().split('\n')
        # Show last 8 lines of traceback (more context)
        for line in tb_lines[-8:-1]:
            if line.strip():
                print(f"    {line}")
        return False

def test_create_messenger():
    """Test creating a new messenger type"""
    payload = {"name": "Test Messenger"}
    response = requests.post(f"{BASE_URL}/messengers", json=payload)
    assert response.status_code == 201
    data = response.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    return data["id"]

def test_get_messenger():
    """Test getting a messenger by ID"""
    # First, create a messenger
    payload = {"name": "Get Messenger Test"}
    create_resp = requests.post(f"{BASE_URL}/messengers", json=payload)
    assert create_resp.status_code == 201
    messenger_id = create_resp.json()["id"]
    
    # Now, get the messenger
    get_resp = requests.get(f"{BASE_URL}/messengers/{messenger_id}")
    assert get_resp.status_code == 200
    data = get_resp.json()
    # Verify MessengerResponse DTO structure
    assert data["name"] == payload["name"]
    assert "created_at" in data
    assert "id" not in data  # ID should not be in response

def test_get_messenger_by_name():
    """Test getting a messenger ID by name"""
    # First, create a messenger
    payload = {"name": "Name Test Messenger"}
    create_resp = requests.post(f"{BASE_URL}/messengers", json=payload)
    assert create_resp.status_code == 201
    
    # Get messenger ID by name
    get_resp = requests.get(f"{BASE_URL}/messengers/by-name/{payload['name']}")
    assert get_resp.status_code == 200
    data = get_resp.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)

def test_get_messenger_not_found():
    """Test getting a non-existent messenger"""
    response = requests.get(f"{BASE_URL}/messengers/99999")
    assert response.status_code == 404
    data = response.json()
    assert "error" in data
    assert "not found" in data["error"]

def test_get_messenger_by_name_not_found():
    """Test getting a non-existent messenger by name"""
    response = requests.get(f"{BASE_URL}/messengers/by-name/nonexistent_messenger")
    assert response.status_code == 404
    data = response.json()
    assert "error" in data
    assert "not found" in data["error"]

def test_create_messenger_related_user():
    """Test creating a new messenger-related user"""
    payload = {
        "user_id": 1,
        "messenger_id": 1,
        "messenger_user_id": "test_user_123",
        "chat_id": "test_chat_456"
    }
    response = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=payload)
    assert response.status_code == 201
    data = response.json()
    # New format: {"id": ...}
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    return data["id"]

def test_get_messenger_related_user():
    """Test getting a messenger-related user by parameters"""
    # First, create a messenger-related user
    payload = {
        "user_id": 1,
        "messenger_id": 1,
        "messenger_user_id": "get_user_test_123",
        "chat_id": "get_user_test_456"
    }
    create_resp = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=payload)
    assert create_resp.status_code == 201
    
    # Get the messenger-related user
    params = {
        "chat_id": payload["chat_id"],
        "messenger_user_id": payload["messenger_user_id"],
        "user_id": payload["user_id"],
        "messenger_id": payload["messenger_id"]
    }
    get_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers", params=params)
    assert get_resp.status_code == 200
    data = get_resp.json()
    # Verify MessengerRelatedUserResponse DTO structure
    assert data["chat_id"] == payload["chat_id"]
    assert data["messenger_user_id"] == payload["messenger_user_id"]
    assert data["user_id"] == payload["user_id"]
    assert data["messenger_id"] == payload["messenger_id"]
    assert "id" in data
    assert "created_at" in data

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

def test_get_user_id_by_messenger_user_id():
    """Test getting a user ID by messenger user ID"""
    # First, create a messenger-related user
    payload = {
        "user_id": 1,
        "messenger_id": 1,
        "messenger_user_id": "user_id_test_123",
        "chat_id": "user_id_test_456"
    }
    create_resp = requests.post(f"{BASE_URL}/messengerRelatedUsers", json=payload)
    assert create_resp.status_code == 201
    
    # Get user ID by messenger user ID
    messenger_user_id = payload["messenger_user_id"]
    get_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers/{messenger_user_id}/user")
    assert get_resp.status_code == 200
    data = get_resp.json()
    # New format: {"user_id": ...}
    assert isinstance(data, dict)
    assert "user_id" in data
    assert data["user_id"] == payload["user_id"]

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

def test_create_messenger_related_user_invalid_data():
    """Test creating a messenger-related user with invalid data"""
    invalid_payload = {
        "invalid_field": "invalid_value"
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
    
    # Step 3: Get messenger by ID
    get_messenger_resp = requests.get(f"{BASE_URL}/messengers/{messenger_id}")
    assert get_messenger_resp.status_code == 200
    messenger_data = get_messenger_resp.json()
    assert messenger_data["name"] == messenger_payload["name"]
    assert "created_at" in messenger_data
    assert "id" not in messenger_data  # ID should not be in response
    
    # Step 4: Get messenger by name
    get_by_name_resp = requests.get(f"{BASE_URL}/messengers/by-name/{messenger_payload['name']}")
    assert get_by_name_resp.status_code == 200
    name_data = get_by_name_resp.json()
    assert "id" in name_data
    assert name_data["id"] == messenger_id
    
    # Step 5: Get messenger-related user
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
    
    # Step 6: Get user ID by messenger user ID
    get_user_id_resp = requests.get(f"{BASE_URL}/messengerRelatedUsers/{related_user_payload['messenger_user_id']}/user")
    assert get_user_id_resp.status_code == 200
    user_id_data = get_user_id_resp.json()
    assert "user_id" in user_id_data
    assert user_id_data["user_id"] == related_user_payload["user_id"]

def check_server_health():
    """Check if the server is running"""
    try:
        response = requests.get(f"{BASE_URL.replace('/api/v1', '')}/healthcheck", timeout=5)
        return response.status_code == 200
    except requests.exceptions.RequestException:
        return False

def main():
    """Main test runner"""
    print("=" * 60)
    print("Messenger E2E Tests")
    print("=" * 60)
    
    # Check if server is running
    print("Checking server health...", end=" ")
    if not check_server_health():
        print("✗ FAILED")
        print("Error: Server is not running. Please start the server first.")
        print("Expected server at: http://localhost:8080")
        sys.exit(1)
    print("✓ OK")
    print()
    
    # Define all tests
    tests = [
        ("Create Messenger", test_create_messenger),
        ("Get Messenger by ID", test_get_messenger),
        ("Get Messenger by Name", test_get_messenger_by_name),
        ("Get Messenger Not Found", test_get_messenger_not_found),
        ("Get Messenger by Name Not Found", test_get_messenger_by_name_not_found),
        ("Create Messenger Related User", test_create_messenger_related_user),
        ("Get Messenger Related User", test_get_messenger_related_user),
        ("Get Messenger Related User Not Found", test_get_messenger_related_user_not_found),
        ("Get User ID by Messenger User ID", test_get_user_id_by_messenger_user_id),
        ("Get User ID by Messenger User ID Not Found", test_get_user_id_by_messenger_user_id_not_found),
        ("Create Messenger Invalid Data", test_create_messenger_invalid_data),
        ("Create Messenger Related User Invalid Data", test_create_messenger_related_user_invalid_data),
        ("Get Messenger Invalid ID", test_get_messenger_invalid_id),
        ("Get Messenger Related User Invalid Params", test_get_messenger_related_user_invalid_params),
        ("Complete Messenger Workflow", test_messenger_workflow),
    ]
    
    # Run tests
    passed = 0
    failed = 0
    
    for test_name, test_func in tests:
        if run_test(test_name, test_func):
            passed += 1
        else:
            failed += 1
        time.sleep(0.1)  # Small delay between tests
    
    # Print summary
    print()
    print("=" * 60)
    print("Test Summary")
    print("=" * 60)
    print(f"Passed: {passed}")
    print(f"Failed: {failed}")
    print(f"Total: {passed + failed}")
    
    if failed == 0:
        print("🎉 All tests passed!")
        sys.exit(0)
    else:
        print("❌ Some tests failed!")
        print()
        print("Note: For more detailed output, run tests with pytest:")
        print("  pytest tests/test_messengers_e2e.py -v")
        sys.exit(1)

if __name__ == "__main__":
    main()
