#!/usr/bin/env python3
"""
Simple test runner for task E2E tests
Run with: python run_tasks_tests.py

Note: This is a standalone test runner. For pytest, use test_tasks_e2e.py directly.
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

def create_test_user():
    """Helper to create a test user"""
    user_payload = {
        "name": "Test User for Tasks",
        "email": "taskuser@example.com",
        "password_hash": "test123"
    }
    resp = requests.post(f"{BASE_URL}/users", json=user_payload)
    if resp.status_code == 201:
        return resp.json()["id"]
    return 1  # Fallback to user_id=1 if creation fails

def test_create_task():
    """Test creating a task and verify response structure"""
    user_id = create_test_user()
    payload = {
        "title": "Test Task",
        "description": "This is a test task",
        "user_id": user_id
    }
    response = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    assert data["id"] > 0

def test_get_task():
    """Test getting a task and verify TaskResponse DTO structure"""
    user_id = create_test_user()
    payload = {
        "title": "Get Task",
        "description": "Task to get",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    get_resp = requests.get(f"{BASE_URL}/tasks/{task_id}")
    assert get_resp.status_code == 200
    data = get_resp.json()
    
    assert "id" in data
    assert "title" in data
    assert "description" in data
    assert "user_id" in data
    assert "status" in data
    assert "created_at" in data
    assert "deleted_at" not in data
    
    assert data["title"] == payload["title"]
    assert data["user_id"] == user_id

def test_update_task():
    """Test updating a task and verify TaskResponse DTO structure"""
    user_id = create_test_user()
    payload = {
        "title": "Update Task",
        "description": "Task to update",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    update_payload = {
        "title": "Updated Task",
        "description": "Updated description"
    }
    update_resp = requests.put(f"{BASE_URL}/tasks/{task_id}", json=update_payload)
    assert update_resp.status_code == 200
    updated = update_resp.json()
    
    assert updated["title"] == "Updated Task"
    assert updated["description"] == "Updated description"
    assert updated["id"] == task_id

def test_delete_task():
    """Test deleting a task"""
    user_id = create_test_user()
    payload = {
        "title": "Delete Task",
        "description": "Task to delete",
        "user_id": user_id
    }
    create_resp = requests.post(f"{BASE_URL}/tasks", json=payload)
    assert create_resp.status_code == 201
    task_id = create_resp.json()["id"]

    del_resp = requests.delete(f"{BASE_URL}/tasks/{task_id}")
    assert del_resp.status_code == 204
    assert del_resp.text == ""

    get_resp = requests.get(f"{BASE_URL}/tasks/{task_id}")
    assert get_resp.status_code == 404

def test_get_user_tasks():
    """Test getting all tasks for a user"""
    user_id = create_test_user()
    for i in range(2):
        payload = {
            "title": f"User Task {i}",
            "description": f"Task {i} for user",
            "user_id": user_id
        }
        resp = requests.post(f"{BASE_URL}/tasks", json=payload)
        assert resp.status_code == 201

    resp = requests.get(f"{BASE_URL}/users/{user_id}/tasks")
    assert resp.status_code == 200
    tasks = resp.json()
    assert isinstance(tasks, list)
    assert len(tasks) >= 2

def test_get_task_not_found():
    """Test getting a non-existent task"""
    resp = requests.get(f"{BASE_URL}/tasks/99999")
    assert resp.status_code == 404
    data = resp.json()
    assert "error" in data

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
    print("Task E2E Tests")
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
        ("Create Task", test_create_task),
        ("Get Task", test_get_task),
        ("Update Task", test_update_task),
        ("Delete Task", test_delete_task),
        ("Get User Tasks", test_get_user_tasks),
        ("Get Task Not Found", test_get_task_not_found),
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
        print("  pytest tests/test_tasks_e2e.py -v")
        sys.exit(1)

if __name__ == "__main__":
    main()

