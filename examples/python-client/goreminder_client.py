"""
GoReminder API Python Client

A simple Python client for interacting with the GoReminder API.
"""

import requests
from typing import Optional, Dict, List, Any
from datetime import datetime, timezone
import json


class GoReminderClient:
    """Client for interacting with GoReminder API"""

    def __init__(self, base_url: str = "http://localhost:8080"):
        """
        Initialize the client

        Args:
            base_url: Base URL of the API (default: http://localhost:8080)
        """
        self.base_url = base_url.rstrip('/')
        self.api_base = f"{self.base_url}/api/v1"
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        })

    def _make_request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict] = None,
        params: Optional[Dict] = None
    ) -> Dict[str, Any]:
        """
        Make an HTTP request to the API

        Args:
            method: HTTP method (GET, POST, PUT, DELETE)
            endpoint: API endpoint (without /api/v1 prefix)
            data: Request body data
            params: Query parameters

        Returns:
            Response JSON as dictionary

        Raises:
            requests.HTTPError: If the request fails
        """
        url = f"{self.api_base}{endpoint}"
        response = self.session.request(
            method=method,
            url=url,
            json=data,
            params=params
        )
        response.raise_for_status()
        return response.json()

    # Task methods
    def create_task(
        self,
        title: str,
        user_id: int,
        start_date: datetime,
        description: Optional[str] = None,
        messenger_related_user_id: Optional[int] = None,
        finish_date: Optional[datetime] = None,
        cron_expression: Optional[str] = None,
        requires_confirmation: bool = False,
        status: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Create a new task

        Args:
            title: Task title
            user_id: User ID
            start_date: Task start date (must be in the future, UTC)
            description: Task description
            messenger_related_user_id: Messenger related user ID
            finish_date: Task finish date
            cron_expression: Cron expression for recurring tasks (e.g., "0 9 * * *")
            requires_confirmation: Whether task requires confirmation
            status: Task status (pending, scheduled, done, rescheduled, postponed, deleted)

        Returns:
            Created task data
        """
        data = {
            "title": title,
            "user_id": user_id,
            "start_date": start_date.isoformat() + "Z"
        }

        if description:
            data["description"] = description
        if messenger_related_user_id:
            data["messenger_related_user_id"] = messenger_related_user_id
        if finish_date:
            data["finish_date"] = finish_date.isoformat() + "Z"
        if cron_expression:
            data["cron_expression"] = cron_expression
        if requires_confirmation:
            data["requires_confirmation"] = requires_confirmation
        if status:
            data["status"] = status

        return self._make_request("POST", "/tasks", data=data)

    def get_task(self, task_id: int) -> Dict[str, Any]:
        """Get a task by ID"""
        return self._make_request("GET", f"/tasks/{task_id}")

    def get_all_tasks(
        self,
        page: int = 1,
        page_size: int = 50,
        order_by: Optional[str] = None,
        status: Optional[str] = None,
        start_date_from: Optional[datetime] = None,
        start_date_to: Optional[datetime] = None,
        user_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """
        Get all tasks with pagination and filtering

        Args:
            page: Page number (1-indexed)
            page_size: Number of items per page
            order_by: Ordering clause (e.g., "created_at DESC")
            status: Filter by status
            start_date_from: Filter tasks with start_date >= this date
            start_date_to: Filter tasks with start_date <= this date
            user_id: Filter by user ID

        Returns:
            Paginated tasks response
        """
        params = {
            "page": page,
            "page_size": page_size
        }

        if order_by:
            params["order_by"] = order_by
        if status:
            params["status"] = status
        if start_date_from:
            params["start_date_from"] = start_date_from.isoformat() + "Z"
        if start_date_to:
            params["start_date_to"] = start_date_to.isoformat() + "Z"
        if user_id:
            params["user_id"] = user_id

        return self._make_request("GET", "/tasks", params=params)

    def update_task(
        self,
        task_id: int,
        title: Optional[str] = None,
        description: Optional[str] = None,
        status: Optional[str] = None,
        start_date: Optional[datetime] = None,
        finish_date: Optional[datetime] = None,
        requires_confirmation: Optional[bool] = None,
        cron_expression: Optional[str] = None
    ) -> Dict[str, Any]:
        """Update a task (partial update)"""
        data = {}

        if title is not None:
            data["title"] = title
        if description is not None:
            data["description"] = description
        if status is not None:
            data["status"] = status
        if start_date is not None:
            data["start_date"] = start_date.isoformat() + "Z"
        if finish_date is not None:
            data["finish_date"] = finish_date.isoformat() + "Z"
        if requires_confirmation is not None:
            data["requires_confirmation"] = requires_confirmation
        if cron_expression is not None:
            data["cron_expression"] = cron_expression

        return self._make_request("PUT", f"/tasks/{task_id}", data=data)

    def delete_task(self, task_id: int) -> None:
        """Delete a task (soft delete)"""
        self._make_request("DELETE", f"/tasks/{task_id}")

    def mark_task_as_done(self, task_id: int) -> Dict[str, Any]:
        """Mark a task as done"""
        return self._make_request("POST", f"/tasks/{task_id}/done")

    def get_task_history(self, task_id: int) -> List[Dict[str, Any]]:
        """Get task history"""
        return self._make_request("GET", f"/tasks/{task_id}/history")

    def get_user_tasks(
        self,
        user_id: int,
        page: int = 1,
        page_size: int = 50,
        order_by: Optional[str] = None,
        start_date_from: Optional[datetime] = None,
        start_date_to: Optional[datetime] = None,
        created_at_from: Optional[datetime] = None,
        created_at_to: Optional[datetime] = None,
        requires_confirmation: Optional[bool] = None
    ) -> Dict[str, Any]:
        """Get all tasks for a specific user"""
        params = {
            "page": page,
            "page_size": page_size
        }

        if order_by:
            params["order_by"] = order_by
        if start_date_from:
            params["start_date_from"] = start_date_from.isoformat() + "Z"
        if start_date_to:
            params["start_date_to"] = start_date_to.isoformat() + "Z"
        if created_at_from:
            params["created_at_from"] = created_at_from.isoformat() + "Z"
        if created_at_to:
            params["created_at_to"] = created_at_to.isoformat() + "Z"
        if requires_confirmation is not None:
            params["requires_confirmation"] = requires_confirmation

        return self._make_request("GET", f"/users/{user_id}/tasks", params=params)

    def get_user_task_history(
        self,
        user_id: int,
        limit: int = 50,
        offset: int = 0
    ) -> List[Dict[str, Any]]:
        """Get user task history"""
        params = {
            "limit": limit,
            "offset": offset
        }
        return self._make_request("GET", f"/users/{user_id}/tasks/history", params=params)

    def queue_task(
        self,
        task_id: int,
        action: str = "schedule",
        queue_name: str = "celery"
    ) -> Dict[str, Any]:
        """
        Queue a task for processing

        Args:
            task_id: Task ID
            action: Action to perform (schedule or delete)
            queue_name: Queue name (default: celery)

        Returns:
            Queue response
        """
        data = {
            "task_id": task_id,
            "action": action,
            "queue_name": queue_name
        }
        return self._make_request("POST", "/tasks/queue", data=data)

    # User methods
    def create_user(
        self,
        name: str,
        email: Optional[str] = None,
        password_hash: Optional[str] = None,
        timezone: Optional[str] = None,
        language_code: Optional[str] = None,
        role: Optional[str] = None
    ) -> Dict[str, Any]:
        """Create a new user"""
        data = {"name": name}

        if email:
            data["email"] = email
        if password_hash:
            data["password_hash"] = password_hash
        if timezone:
            data["timezone"] = timezone
        if language_code:
            data["language_code"] = language_code
        if role:
            data["role"] = role

        return self._make_request("POST", "/users", data=data)

    def get_user(self, user_id: int) -> Dict[str, Any]:
        """Get a user by ID"""
        return self._make_request("GET", f"/users/{user_id}")

    def get_all_users(
        self,
        page: int = 1,
        page_size: int = 50,
        order_by: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get all users with pagination"""
        params = {
            "page": page,
            "page_size": page_size
        }

        if order_by:
            params["order_by"] = order_by

        return self._make_request("GET", "/users", params=params)

    def update_user(
        self,
        user_id: int,
        name: Optional[str] = None,
        email: Optional[str] = None,
        password_hash: Optional[str] = None,
        timezone: Optional[str] = None,
        language_code: Optional[str] = None,
        role: Optional[str] = None
    ) -> Dict[str, Any]:
        """Update a user (partial update)"""
        data = {}

        if name is not None:
            data["name"] = name
        if email is not None:
            data["email"] = email
        if password_hash is not None:
            data["password_hash"] = password_hash
        if timezone is not None:
            data["timezone"] = timezone
        if language_code is not None:
            data["language_code"] = language_code
        if role is not None:
            data["role"] = role

        return self._make_request("PUT", f"/users/{user_id}", data=data)

    def delete_user(self, user_id: int) -> None:
        """Delete a user (soft delete)"""
        self._make_request("DELETE", f"/users/{user_id}")

    # Messenger methods
    def create_messenger(self, name: str) -> Dict[str, Any]:
        """Create a messenger type"""
        data = {"name": name}
        return self._make_request("POST", "/messengers", data=data)

    def get_messenger(self, messenger_id: int) -> Dict[str, Any]:
        """Get a messenger by ID"""
        return self._make_request("GET", f"/messengers/{messenger_id}")

    def get_messenger_id_by_name(self, messenger_name: str) -> Dict[str, Any]:
        """Get messenger ID by name"""
        return self._make_request("GET", f"/messengers/by-name/{messenger_name}")

    def get_all_messengers(
        self,
        page: int = 1,
        page_size: int = 50,
        order_by: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get all messengers with pagination"""
        params = {
            "page": page,
            "page_size": page_size
        }

        if order_by:
            params["order_by"] = order_by

        return self._make_request("GET", "/messengers", params=params)

    def create_messenger_related_user(
        self,
        user_id: int,
        messenger_id: int,
        chat_id: str,
        messenger_user_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """Create a messenger user relation"""
        data = {
            "user_id": user_id,
            "messenger_id": messenger_id,
            "chat_id": chat_id
        }

        if messenger_user_id:
            data["messenger_user_id"] = messenger_user_id

        return self._make_request("POST", "/messengerRelatedUsers", data=data)

    def get_messenger_related_user(
        self,
        chat_id: Optional[str] = None,
        messenger_user_id: Optional[str] = None,
        user_id: Optional[int] = None,
        messenger_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """Get messenger-related user"""
        params = {}

        if chat_id:
            params["chat_id"] = chat_id
        if messenger_user_id:
            params["messenger_user_id"] = messenger_user_id
        if user_id:
            params["user_id"] = user_id
        if messenger_id:
            params["messenger_id"] = messenger_id

        return self._make_request("GET", "/messengerRelatedUsers", params=params)

    def get_all_messenger_related_users(
        self,
        page: int = 1,
        page_size: int = 50,
        order_by: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get all messenger-related users with pagination"""
        params = {
            "page": page,
            "page_size": page_size
        }

        if order_by:
            params["order_by"] = order_by

        return self._make_request("GET", "/messengerRelatedUsers/all", params=params)

    def get_user_id_by_messenger_user_id(self, messenger_user_id: int) -> Dict[str, Any]:
        """Get user ID by messenger user ID"""
        return self._make_request("GET", f"/messengerRelatedUsers/{messenger_user_id}/user")

    # Backlog methods
    def create_backlog(
        self,
        title: str,
        user_id: int,
        description: Optional[str] = None,
        messenger_related_user_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """Create a new backlog item"""
        data = {
            "title": title,
            "user_id": user_id
        }

        if description:
            data["description"] = description
        if messenger_related_user_id:
            data["messenger_related_user_id"] = messenger_related_user_id

        return self._make_request("POST", "/backlogs", data=data)

    def create_backlogs_batch(
        self,
        items: str,
        user_id: int,
        separator: str = "\n",
        messenger_related_user_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """
        Create multiple backlog items at once

        Args:
            items: Items separated by separator
            user_id: User ID
            separator: Separator between items (default: newline)
            messenger_related_user_id: Messenger related user ID

        Returns:
            Created backlogs response
        """
        data = {
            "items": items,
            "user_id": user_id,
            "separator": separator
        }

        if messenger_related_user_id:
            data["messenger_related_user_id"] = messenger_related_user_id

        return self._make_request("POST", "/backlogs/batch", data=data)

    def get_backlog(self, backlog_id: int) -> Dict[str, Any]:
        """Get a backlog by ID"""
        return self._make_request("GET", f"/backlogs/{backlog_id}")

    def get_all_backlogs(
        self,
        page: int = 1,
        page_size: int = 50,
        order_by: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get all backlogs with pagination"""
        params = {
            "page": page,
            "page_size": page_size
        }

        if order_by:
            params["order_by"] = order_by

        return self._make_request("GET", "/backlogs", params=params)

    def update_backlog(
        self,
        backlog_id: int,
        title: Optional[str] = None,
        description: Optional[str] = None,
        completed_at: Optional[datetime] = None
    ) -> Dict[str, Any]:
        """Update a backlog item"""
        data = {}

        if title is not None:
            data["title"] = title
        if description is not None:
            data["description"] = description
        if completed_at is not None:
            data["completed_at"] = completed_at.isoformat() + "Z"

        return self._make_request("PUT", f"/backlogs/{backlog_id}", data=data)

    def delete_backlog(self, backlog_id: int) -> None:
        """Delete a backlog item"""
        self._make_request("DELETE", f"/backlogs/{backlog_id}")

    # Digest methods
    def get_digest(
        self,
        user_id: int,
        date: Optional[datetime] = None
    ) -> Dict[str, Any]:
        """
        Get digest for a user

        Args:
            user_id: User ID
            date: Date for digest (default: today)

        Returns:
            Digest data
        """
        params = {"user_id": user_id}

        if date:
            params["date"] = date.isoformat() + "Z"

        return self._make_request("GET", "/digests", params=params)

    def create_digest_settings(
        self,
        user_id: int,
        weekday_time: str,
        weekend_time: str,
        enabled: bool = True,
        messenger_related_user_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """
        Create digest settings

        Args:
            user_id: User ID
            weekday_time: Time for weekdays (format: HH:MM, e.g., "07:00")
            weekend_time: Time for weekends (format: HH:MM, e.g., "10:00")
            enabled: Whether digest is enabled
            messenger_related_user_id: Messenger related user ID

        Returns:
            Created digest settings
        """
        data = {
            "user_id": user_id,
            "weekday_time": weekday_time,
            "weekend_time": weekend_time,
            "enabled": enabled
        }

        if messenger_related_user_id:
            data["messenger_related_user_id"] = messenger_related_user_id

        return self._make_request("POST", "/digests/settings", data=data)

    def get_digest_settings(self, user_id: int) -> Dict[str, Any]:
        """Get digest settings for a user"""
        params = {"user_id": user_id}
        return self._make_request("GET", "/digests/settings", params=params)

    def update_digest_settings(
        self,
        user_id: int,
        enabled: Optional[bool] = None,
        weekday_time: Optional[str] = None,
        weekend_time: Optional[str] = None,
        messenger_related_user_id: Optional[int] = None
    ) -> Dict[str, Any]:
        """Update digest settings"""
        data = {}

        if enabled is not None:
            data["enabled"] = enabled
        if weekday_time is not None:
            data["weekday_time"] = weekday_time
        if weekend_time is not None:
            data["weekend_time"] = weekend_time
        if messenger_related_user_id is not None:
            data["messenger_related_user_id"] = messenger_related_user_id

        # user_id is required for update
        params = {"user_id": user_id}

        return self._make_request("PUT", "/digests/settings", data=data, params=params)

    def delete_digest_settings(self, user_id: int) -> None:
        """Delete digest settings for a user"""
        params = {"user_id": user_id}
        self._make_request("DELETE", "/digests/settings", params=params)

    def get_all_digest_settings(
        self,
        page: int = 1,
        page_size: int = 50,
        order_by: Optional[str] = None
    ) -> Dict[str, Any]:
        """Get all digest settings with pagination"""
        params = {
            "page": page,
            "page_size": page_size
        }

        if order_by:
            params["order_by"] = order_by

        return self._make_request("GET", "/digests/settings/all", params=params)

