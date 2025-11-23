"""
Pytest configuration and fixtures for E2E tests
"""
import pytest
import requests

BASE_URL = "http://localhost:8080/api/v1"

@pytest.fixture(scope="session")
def api_base_url():
    """Base URL for API requests"""
    return BASE_URL

@pytest.fixture(scope="session", autouse=True)
def check_server_health():
    """Check if server is running before running tests"""
    try:
        response = requests.get(f"{BASE_URL.replace('/api/v1', '')}/healthcheck", timeout=5)
        if response.status_code != 200:
            pytest.skip("Server is not running or unhealthy")
    except requests.exceptions.RequestException:
        pytest.skip("Server is not running. Please start the server first.")

def assert_response_status(response, expected_status, error_msg=None):
    """
    Assert response status code with detailed error message
    
    Args:
        response: requests.Response object
        expected_status: Expected HTTP status code
        error_msg: Optional custom error message
    """
    if response.status_code != expected_status:
        msg = error_msg or f"Expected status {expected_status}, got {response.status_code}"
        try:
            error_data = response.json()
            if "error" in error_data:
                msg += f"\n  Error from API: {error_data['error']}"
            else:
                msg += f"\n  Response: {error_data}"
        except:
            msg += f"\n  Response text: {response.text[:200]}"
        pytest.fail(msg)

def assert_response_structure(data, required_fields, forbidden_fields=None):
    """
    Assert response has required fields and doesn't have forbidden fields
    
    Args:
        data: Response JSON data (dict)
        required_fields: List of required field names
        forbidden_fields: Optional list of fields that should NOT be present
    """
    missing = [field for field in required_fields if field not in data]
    if missing:
        pytest.fail(f"Missing required fields in response: {missing}\n  Response: {data}")
    
    if forbidden_fields:
        present = [field for field in forbidden_fields if field in data]
        if present:
            pytest.fail(f"Forbidden fields found in response: {present}\n  Response: {data}")

def pytest_assertrepr_compare(op, left, right):
    """Custom assertion error messages for better debugging"""
    if op == "==":
        return [
            f"Assertion failed:",
            f"  Expected: {right}",
            f"  Got: {left}",
        ]

