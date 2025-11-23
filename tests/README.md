# E2E Tests

This directory contains end-to-end tests for the GoReminder API.

## Test Files Structure

### Pytest Tests (Recommended)
- `test_users_e2e.py` - User API tests (pytest)
- `test_tasks_e2e.py` - Task API tests (pytest)
- `test_messengers_e2e.py` - Messenger API tests (pytest)

These files are designed to be run with **pytest** and provide detailed output, better error messages, and test discovery.

### Standalone Test Runners
- `run_users_tests.py` - Standalone test runner for user tests (no pytest required)
- `run_tasks_tests.py` - Standalone test runner for task tests (no pytest required)
- `run_messenger_tests.py` - Standalone test runner for messenger tests (no pytest required)

These are simple scripts that can be run directly without pytest. They're useful for quick testing or CI/CD pipelines that don't have pytest installed. All runners provide detailed error output with request/response information.

## Running Tests

### Using Pytest (Recommended)

Run all tests:
```bash
pytest tests/ -v
```

Run specific test file:
```bash
pytest tests/test_users_e2e.py -v
pytest tests/test_tasks_e2e.py -v
pytest tests/test_messengers_e2e.py -v
```

Run with detailed output:
```bash
pytest tests/ -v -s  # -s shows print statements
```

Run with error details:
```bash
pytest tests/ -v --tb=short  # Short traceback
pytest tests/ -v --tb=long   # Full traceback
```

### Using Standalone Runners

Run tests without pytest:
```bash
python tests/run_users_tests.py
python tests/run_tasks_tests.py
python tests/run_messenger_tests.py
```

All standalone runners provide:
- ✅ Detailed error messages with request/response information
- ✅ Server health check before running tests
- ✅ Summary statistics
- ✅ Clear indication of which tests passed/failed

## Test Output

### Pytest Output
Pytest provides:
- Detailed assertion error messages
- Traceback for failed tests
- Test discovery and organization
- Fixtures and parametrization support

### Standalone Runner Output
The standalone runner (`run_messenger_tests.py`) provides:
- Simple pass/fail status
- Error messages with details
- Response information for failed requests
- Summary statistics

## Prerequisites

1. Install Python dependencies:
```bash
pip install -r tests/requirements.txt
```

2. Start the API server:
```bash
# Make sure the server is running on http://localhost:8080
```

3. Ensure the database is set up and accessible

## Test Structure

All tests verify:
- ✅ HTTP status codes
- ✅ Response DTO structure (all required fields present)
- ✅ Security (sensitive fields like `password_hash`, `deleted_at` are not exposed)
- ✅ Data validation (required fields, email format, etc.)
- ✅ Error handling (404, 400, 422, 500 responses)

## Helper Functions

The `conftest.py` file provides helper functions for better error messages:

```python
from conftest import assert_response_status, assert_response_structure

# Check status code with detailed error
assert_response_status(response, 200, "Failed to get user")

# Check response structure
assert_response_structure(
    data,
    required_fields=["id", "name", "email"],
    forbidden_fields=["password_hash", "deleted_at"]
)
```

## Why Two Types of Test Files?

### Pytest Tests (Recommended for Development)
- `test_users_e2e.py`, `test_tasks_e2e.py`, `test_messengers_e2e.py`
  - Better integration with pytest ecosystem
  - Detailed error messages and tracebacks
  - Test discovery and organization
  - Fixtures and parametrization support
  - Better for CI/CD with pytest

### Standalone Runners (Useful for Quick Testing)
- `run_users_tests.py`, `run_tasks_tests.py`, `run_messenger_tests.py`
  - No pytest dependency required
  - Simple execution: `python run_*_tests.py`
  - Detailed error output with request/response info
  - Useful for quick manual testing or simple automation
  - Includes server health check
  - Shows detailed error context (URL, status code, response body)

**Recommendation**: 
- Use pytest tests (`test_*_e2e.py`) for development and CI/CD
- Use standalone runners (`run_*_tests.py`) for quick manual testing or when pytest is not available

## Notes

- Tests create test data and may leave it in the database
- Consider using a test database or cleanup scripts
- Tests assume user_id=1 exists for some messenger-related tests
- Some tests depend on previous tests (fixtures handle this)

