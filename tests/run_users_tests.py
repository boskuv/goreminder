#!/usr/bin/env python3
"""
Simple test runner for user E2E tests
Run with: python run_users_tests.py

Note: This is a standalone test runner. For pytest, use test_users_e2e.py directly.
"""

import requests
import sys
import time
import traceback
import inspect

BASE_URL = "http://localhost:8080/api/v1/users"

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

def test_create_user():
    """Test creating a user and verify response structure"""
    user_payload = {
        "name": "Test User",
        "email": "testuser@example.com",
        "password_hash": "s3cr3t"
    }
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    data = resp.json()
    assert isinstance(data, dict)
    assert "id" in data
    assert isinstance(data["id"], int)
    assert data["id"] > 0

def test_get_user():
    """Test getting a user and verify UserResponse DTO structure"""
    user_payload = {
        "name": "Get User Test",
        "email": "getuser@example.com",
        "password_hash": "s3cr3t"
    }
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    user_id = resp.json()["id"]
    
    resp = requests.get(f"{BASE_URL}/{user_id}")
    assert resp.status_code == 200
    data = resp.json()
    
    assert "id" in data
    assert "name" in data
    assert "email" in data
    assert "created_at" in data
    assert "password_hash" not in data
    assert "deleted_at" not in data
    
    assert data["name"] == user_payload["name"]
    assert data["email"] == user_payload["email"]

def test_update_user():
    """Test updating a user and verify UserResponse DTO structure"""
    user_payload = {
        "name": "Update User Test",
        "email": "updateuser@example.com",
        "password_hash": "s3cr3t"
    }
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    user_id = resp.json()["id"]
    
    update_payload = {
        "name": "Updated Name",
        "email": "updated@example.com"
    }
    resp = requests.put(f"{BASE_URL}/{user_id}", json=update_payload)
    assert resp.status_code == 200
    data = resp.json()
    
    assert data["name"] == "Updated Name"
    assert data["email"] == "updated@example.com"
    assert data["id"] == user_id

def test_delete_user():
    """Test deleting a user"""
    user_payload = {
        "name": "Delete User Test",
        "email": "deleteuser@example.com",
        "password_hash": "s3cr3t"
    }
    resp = requests.post(BASE_URL, json=user_payload)
    assert resp.status_code == 201
    user_id = resp.json()["id"]
    
    resp = requests.delete(f"{BASE_URL}/{user_id}")
    assert resp.status_code == 204
    assert resp.text == ""
    
    resp = requests.get(f"{BASE_URL}/{user_id}")
    assert resp.status_code == 404

def test_get_user_not_found():
    """Test getting a non-existent user"""
    resp = requests.get(f"{BASE_URL}/99999")
    assert resp.status_code == 404
    data = resp.json()
    assert "error" in data

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

def check_server_health():
    """Check if the server is running"""
    try:
        response = requests.get(f"{BASE_URL.replace('/api/v1/users', '')}/healthcheck", timeout=5)
        return response.status_code == 200
    except requests.exceptions.RequestException:
        return False

def main():
    """Main test runner"""
    print("=" * 60)
    print("User E2E Tests")
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
        ("Create User", test_create_user),
        ("Get User", test_get_user),
        ("Update User", test_update_user),
        ("Delete User", test_delete_user),
        ("Get User Not Found", test_get_user_not_found),
        ("Create User Invalid Email", test_create_user_invalid_email),
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
        print("  pytest tests/test_users_e2e.py -v")
        sys.exit(1)

if __name__ == "__main__":
    main()

