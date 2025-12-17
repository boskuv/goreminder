"""
Example usage of GoReminder Python Client

This script demonstrates how to use the GoReminder API client.
"""

from datetime import datetime, timedelta, timezone
from goreminder_client import GoReminderClient
import json


def print_json(data):
    """Pretty print JSON data"""
    print(json.dumps(data, indent=2, default=str))


def main():
    # Initialize client
    client = GoReminderClient(base_url="http://localhost:8080")

    print("=" * 60)
    print("GoReminder API Client Examples")
    print("=" * 60)

    # Example 1: Create a user
    print("\n1. Creating a user...")
    try:
        user = client.create_user(
            name="John Doe",
            email="john.doe@example.com",
            timezone="UTC",
            language_code="en",
            role="user"
        )
        print("User created:")
        print_json(user)
        user_id = user["id"]
    except Exception as e:
        print(f"Error creating user: {e}")
        return

    # Example 2: Create a one-time task
    print("\n2. Creating a one-time task...")
    try:
        future_date = datetime.now(timezone.utc) + timedelta(days=1)
        task = client.create_task(
            title="Complete project documentation",
            user_id=user_id,
            start_date=future_date,
            description="Write comprehensive documentation for the API",
            requires_confirmation=True
        )
        print("Task created:")
        print_json(task)
        task_id = task["id"]
    except Exception as e:
        print(f"Error creating task: {e}")
        return

    # Example 3: Create a recurring task (parent task)
    print("\n3. Creating a recurring task (parent)...")
    try:
        future_date = datetime.now(timezone.utc) + timedelta(days=1)
        recurring_task = client.create_task(
            title="Daily standup reminder",
            user_id=user_id,
            start_date=future_date,
            description="Reminder for daily standup meeting",
            cron_expression="0 9 * * *",  # Daily at 9 AM
            requires_confirmation=True
        )
        print("Recurring task created:")
        print_json(recurring_task)
        recurring_task_id = recurring_task["id"]
    except Exception as e:
        print(f"Error creating recurring task: {e}")

    # Example 4: Get all tasks for user
    print("\n4. Getting all tasks for user...")
    try:
        tasks_response = client.get_user_tasks(
            user_id=user_id,
            page=1,
            page_size=10
        )
        print("User tasks:")
        print_json(tasks_response)
    except Exception as e:
        print(f"Error getting user tasks: {e}")

    # Example 5: Update a task
    print("\n5. Updating a task...")
    try:
        new_future_date = datetime.now(timezone.utc) + timedelta(days=2)
        updated_task = client.update_task(
            task_id=task_id,
            description="Updated: Write comprehensive documentation for the API",
            start_date=new_future_date
        )
        print("Task updated:")
        print_json(updated_task)
    except Exception as e:
        print(f"Error updating task: {e}")

    # Example 6: Mark task as done
    print("\n6. Marking task as done...")
    try:
        done_task = client.mark_task_as_done(task_id)
        print("Task marked as done:")
        print_json(done_task)
    except Exception as e:
        print(f"Error marking task as done: {e}")

    # Example 7: Get task history
    print("\n7. Getting task history...")
    try:
        history = client.get_task_history(task_id)
        print("Task history:")
        print_json(history)
    except Exception as e:
        print(f"Error getting task history: {e}")

    # Example 8: Create a backlog item
    print("\n8. Creating a backlog item...")
    try:
        backlog = client.create_backlog(
            title="Implement new feature",
            user_id=user_id,
            description="Add user authentication"
        )
        print("Backlog created:")
        print_json(backlog)
        backlog_id = backlog["id"]
    except Exception as e:
        print(f"Error creating backlog: {e}")

    # Example 9: Create multiple backlog items
    print("\n9. Creating multiple backlog items...")
    try:
        items = "Item 1\nItem 2\nItem 3"
        backlogs = client.create_backlogs_batch(
            items=items,
            user_id=user_id,
            separator="\n"
        )
        print("Backlogs created:")
        print_json(backlogs)
    except Exception as e:
        print(f"Error creating backlogs: {e}")

    # Example 10: Create digest settings
    print("\n10. Creating digest settings...")
    try:
        digest_settings = client.create_digest_settings(
            user_id=user_id,
            weekday_time="07:00",
            weekend_time="10:00",
            enabled=True
        )
        print("Digest settings created:")
        print_json(digest_settings)
    except Exception as e:
        print(f"Error creating digest settings: {e}")

    # Example 11: Get digest
    print("\n11. Getting digest...")
    try:
        digest = client.get_digest(user_id=user_id)
        print("Digest:")
        print_json(digest)
    except Exception as e:
        print(f"Error getting digest: {e}")

    # Example 12: Get all users
    print("\n12. Getting all users...")
    try:
        users_response = client.get_all_users(page=1, page_size=10)
        print("Users:")
        print_json(users_response)
    except Exception as e:
        print(f"Error getting users: {e}")

    print("\n" + "=" * 60)
    print("Examples completed!")
    print("=" * 60)


if __name__ == "__main__":
    main()

