"""
GoReminder API Python client aligned with /api/v1 (see Swagger at /swagger/index.html).

Create endpoints typically return {"id": ...} (tasks also return "child_id").
DELETE endpoints return 204 No Content (this client returns None).
"""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Union

import requests


JsonDict = Dict[str, Any]
JsonResponse = Optional[Union[JsonDict, List[Any]]]


def _rfc3339(dt: datetime) -> str:
    """Format datetime as UTC RFC3339 with Z suffix (no fractional seconds)."""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    else:
        dt = dt.astimezone(timezone.utc)
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


class GoReminderClient:
    """HTTP client for the GoReminder REST API."""

    def __init__(self, base_url: str = "http://localhost:8080", timeout: float = 30.0):
        self.base_url = base_url.rstrip("/")
        self.api_base = f"{self.base_url}/api/v1"
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update(
            {
                "Content-Type": "application/json",
                "Accept": "application/json",
            }
        )

    def _make_request(
        self,
        method: str,
        endpoint: str,
        data: Optional[JsonDict] = None,
        params: Optional[JsonDict] = None,
        *,
        files: Optional[Dict[str, Any]] = None,
        raw: bool = False,
    ) -> Union[JsonResponse, bytes]:
        url = f"{self.api_base}{endpoint}"
        headers = None
        json_body = data
        if files is not None:
            # Let requests set multipart Content-Type with boundary.
            headers = {k: v for k, v in self.session.headers.items() if k.lower() != "content-type"}
            json_body = None

        response = self.session.request(
            method=method,
            url=url,
            json=json_body,
            params=params,
            files=files,
            headers=headers,
            timeout=self.timeout,
        )
        response.raise_for_status()

        if raw:
            return response.content
        if response.status_code == 204 or not response.content:
            return None
        return response.json()

    # --- System -----------------------------------------------------------------

    def healthcheck(self) -> JsonDict:
        url = f"{self.base_url}/healthcheck"
        response = self.session.get(url, timeout=self.timeout)
        response.raise_for_status()
        return response.json()

    def version(self) -> JsonDict:
        url = f"{self.base_url}/version"
        response = self.session.get(url, timeout=self.timeout)
        response.raise_for_status()
        return response.json()

    # --- Tasks ------------------------------------------------------------------

    def create_task(
        self,
        title: str,
        user_id: int,
        *,
        start_date: Optional[datetime] = None,
        description: Optional[str] = None,
        messenger_related_user_id: Optional[int] = None,
        finish_date: Optional[datetime] = None,
        cron_expression: Optional[str] = None,
        rrule: Optional[str] = None,
        requires_confirmation: bool = False,
        muted: bool = False,
        pre_remind_before_seconds: Optional[int] = None,
        status: Optional[str] = None,
    ) -> JsonDict:
        """POST /tasks → {"id": int, "child_id": int}."""
        data: JsonDict = {"title": title, "user_id": user_id}
        if description is not None:
            data["description"] = description
        if messenger_related_user_id is not None:
            data["messenger_related_user_id"] = messenger_related_user_id
        if start_date is not None:
            data["start_date"] = _rfc3339(start_date)
        if finish_date is not None:
            data["finish_date"] = _rfc3339(finish_date)
        if cron_expression is not None:
            data["cron_expression"] = cron_expression
        if rrule is not None:
            data["rrule"] = rrule
        if requires_confirmation:
            data["requires_confirmation"] = True
        if muted:
            data["muted"] = True
        if pre_remind_before_seconds is not None:
            data["pre_remind_before_seconds"] = pre_remind_before_seconds
        if status is not None:
            data["status"] = status
        return self._make_request("POST", "/tasks", data=data)  # type: ignore[return-value]

    def get_task(self, task_id: int) -> JsonDict:
        return self._make_request("GET", f"/tasks/{task_id}")  # type: ignore[return-value]

    def get_all_tasks(
        self,
        page: int = 1,
        page_size: int = 50,
        *,
        order_by: Optional[str] = None,
        status: Optional[str] = None,
        status_not: Optional[str] = None,
        start_date_from: Optional[datetime] = None,
        start_date_to: Optional[datetime] = None,
        user_id: Optional[int] = None,
        cron_expression: Optional[str] = None,
        cron_expression_is_null: Optional[bool] = None,
        requires_confirmation: Optional[bool] = None,
        exclude_cron_with_confirmation: Optional[bool] = None,
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        if status:
            params["status"] = status
        if status_not:
            params["status_not"] = status_not
        if start_date_from:
            params["start_date_from"] = _rfc3339(start_date_from)
        if start_date_to:
            params["start_date_to"] = _rfc3339(start_date_to)
        if user_id is not None:
            params["user_id"] = user_id
        if cron_expression is not None:
            params["cron_expression"] = cron_expression
        if cron_expression_is_null is not None:
            params["cron_expression_is_null"] = cron_expression_is_null
        if requires_confirmation is not None:
            params["requires_confirmation"] = requires_confirmation
        if exclude_cron_with_confirmation is not None:
            params["exclude_cron_with_confirmation"] = exclude_cron_with_confirmation
        return self._make_request("GET", "/tasks", params=params)  # type: ignore[return-value]

    def update_task(
        self,
        task_id: int,
        *,
        title: Optional[str] = None,
        description: Optional[str] = None,
        status: Optional[str] = None,
        start_date: Optional[datetime] = None,
        finish_date: Optional[datetime] = None,
        requires_confirmation: Optional[bool] = None,
        muted: Optional[bool] = None,
        pre_remind_before_seconds: Optional[int] = None,
        cron_expression: Optional[str] = None,
        rrule: Optional[str] = None,
    ) -> JsonDict:
        data: JsonDict = {}
        if title is not None:
            data["title"] = title
        if description is not None:
            data["description"] = description
        if status is not None:
            data["status"] = status
        if start_date is not None:
            data["start_date"] = _rfc3339(start_date)
        if finish_date is not None:
            data["finish_date"] = _rfc3339(finish_date)
        if requires_confirmation is not None:
            data["requires_confirmation"] = requires_confirmation
        if muted is not None:
            data["muted"] = muted
        if pre_remind_before_seconds is not None:
            data["pre_remind_before_seconds"] = pre_remind_before_seconds
        if cron_expression is not None:
            data["cron_expression"] = cron_expression
        if rrule is not None:
            data["rrule"] = rrule
        return self._make_request("PUT", f"/tasks/{task_id}", data=data)  # type: ignore[return-value]

    def delete_task(self, task_id: int) -> None:
        self._make_request("DELETE", f"/tasks/{task_id}")

    def mark_task_as_done(self, task_id: int) -> JsonDict:
        return self._make_request("POST", f"/tasks/{task_id}/done")  # type: ignore[return-value]

    def mute_task(self, task_id: int) -> JsonDict:
        return self._make_request("POST", f"/tasks/{task_id}/mute")  # type: ignore[return-value]

    def unmute_task(self, task_id: int) -> JsonDict:
        return self._make_request("POST", f"/tasks/{task_id}/unmute")  # type: ignore[return-value]

    def get_task_history(self, task_id: int) -> List[JsonDict]:
        return self._make_request("GET", f"/tasks/{task_id}/history")  # type: ignore[return-value]

    def get_user_tasks(
        self,
        user_id: int,
        page: int = 1,
        page_size: int = 50,
        *,
        order_by: Optional[str] = None,
        start_date_from: Optional[datetime] = None,
        start_date_to: Optional[datetime] = None,
        created_at_from: Optional[datetime] = None,
        created_at_to: Optional[datetime] = None,
        requires_confirmation: Optional[bool] = None,
        status: Optional[str] = None,
        status_not: Optional[str] = None,
        cron_expression: Optional[str] = None,
        cron_expression_is_null: Optional[bool] = None,
        exclude_cron_with_confirmation: Optional[bool] = None,
        messenger_related_user_id: Optional[int] = None,
        messenger_user_id: Optional[str] = None,
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        if start_date_from:
            params["start_date_from"] = _rfc3339(start_date_from)
        if start_date_to:
            params["start_date_to"] = _rfc3339(start_date_to)
        if created_at_from:
            params["created_at_from"] = _rfc3339(created_at_from)
        if created_at_to:
            params["created_at_to"] = _rfc3339(created_at_to)
        if requires_confirmation is not None:
            params["requires_confirmation"] = requires_confirmation
        if status:
            params["status"] = status
        if status_not:
            params["status_not"] = status_not
        if cron_expression is not None:
            params["cron_expression"] = cron_expression
        if cron_expression_is_null is not None:
            params["cron_expression_is_null"] = cron_expression_is_null
        if exclude_cron_with_confirmation is not None:
            params["exclude_cron_with_confirmation"] = exclude_cron_with_confirmation
        if messenger_related_user_id is not None:
            params["messenger_related_user_id"] = messenger_related_user_id
        if messenger_user_id:
            params["messenger_user_id"] = messenger_user_id
        return self._make_request("GET", f"/users/{user_id}/tasks", params=params)  # type: ignore[return-value]

    def get_user_task_history(self, user_id: int, limit: int = 50, offset: int = 0) -> List[JsonDict]:
        return self._make_request(  # type: ignore[return-value]
            "GET",
            f"/users/{user_id}/tasks/history",
            params={"limit": limit, "offset": offset},
        )

    def queue_task(
        self,
        task_id: int,
        action: str = "schedule",
        queue_name: str = "celery",
    ) -> JsonDict:
        """POST /tasks/queue — action/queue_name/task_id are all required by the API."""
        data = {"task_id": task_id, "action": action, "queue_name": queue_name}
        return self._make_request("POST", "/tasks/queue", data=data)  # type: ignore[return-value]

    # --- Task attachments (require attachments.enabled on the server) -----------

    def init_attachment(
        self,
        task_id: int,
        original_name: str,
        content_type: str,
        size_bytes: int,
        idempotency_key: Optional[str] = None,
    ) -> JsonDict:
        """JSON init → pending + upload_url (presigned)."""
        data: JsonDict = {
            "original_name": original_name,
            "content_type": content_type,
            "size_bytes": size_bytes,
        }
        if idempotency_key:
            data["idempotency_key"] = idempotency_key
        return self._make_request("POST", f"/tasks/{task_id}/attachments", data=data)  # type: ignore[return-value]

    def upload_attachment_direct(
        self,
        task_id: int,
        file_path: str,
        *,
        content_type: Optional[str] = None,
        idempotency_key: Optional[str] = None,
    ) -> JsonDict:
        """multipart direct upload (small files; status ready)."""
        params = {}
        if idempotency_key:
            params["idempotency_key"] = idempotency_key
        with open(file_path, "rb") as fh:
            files = {
                "file": (
                    file_path.rsplit("/", 1)[-1],
                    fh,
                    content_type or "application/octet-stream",
                )
            }
            return self._make_request(  # type: ignore[return-value]
                "POST",
                f"/tasks/{task_id}/attachments",
                params=params or None,
                files=files,
            )

    def complete_attachment(self, task_id: int, attachment_id: str) -> JsonDict:
        return self._make_request(  # type: ignore[return-value]
            "POST", f"/tasks/{task_id}/attachments/{attachment_id}/complete"
        )

    def list_attachments(self, task_id: int) -> JsonDict:
        return self._make_request("GET", f"/tasks/{task_id}/attachments")  # type: ignore[return-value]

    def get_attachment_download_url(self, task_id: int, attachment_id: str) -> JsonDict:
        return self._make_request(  # type: ignore[return-value]
            "GET", f"/tasks/{task_id}/attachments/{attachment_id}/download"
        )

    def get_attachment_content(self, task_id: int, attachment_id: str) -> bytes:
        return self._make_request(  # type: ignore[return-value]
            "GET", f"/tasks/{task_id}/attachments/{attachment_id}/content", raw=True
        )

    def delete_attachment(self, task_id: int, attachment_id: str) -> None:
        self._make_request("DELETE", f"/tasks/{task_id}/attachments/{attachment_id}")

    # --- Users ------------------------------------------------------------------

    def create_user(
        self,
        name: str,
        *,
        email: Optional[str] = None,
        password_hash: Optional[str] = None,
        timezone: Optional[str] = None,
        language_code: Optional[str] = None,
        role: Optional[str] = None,
    ) -> JsonDict:
        """POST /users → {"id": int}."""
        data: JsonDict = {"name": name}
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
        return self._make_request("POST", "/users", data=data)  # type: ignore[return-value]

    def get_user(self, user_id: int) -> JsonDict:
        return self._make_request("GET", f"/users/{user_id}")  # type: ignore[return-value]

    def get_all_users(
        self, page: int = 1, page_size: int = 50, order_by: Optional[str] = None
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        return self._make_request("GET", "/users", params=params)  # type: ignore[return-value]

    def update_user(
        self,
        user_id: int,
        *,
        name: Optional[str] = None,
        email: Optional[str] = None,
        password_hash: Optional[str] = None,
        timezone: Optional[str] = None,
        language_code: Optional[str] = None,
        role: Optional[str] = None,
    ) -> JsonDict:
        data: JsonDict = {}
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
        return self._make_request("PUT", f"/users/{user_id}", data=data)  # type: ignore[return-value]

    def delete_user(self, user_id: int) -> None:
        self._make_request("DELETE", f"/users/{user_id}")

    # --- Messengers -------------------------------------------------------------

    def create_messenger(self, name: str) -> JsonDict:
        return self._make_request("POST", "/messengers", data={"name": name})  # type: ignore[return-value]

    def get_messenger(self, messenger_id: int) -> JsonDict:
        return self._make_request("GET", f"/messengers/{messenger_id}")  # type: ignore[return-value]

    def get_messenger_id_by_name(self, messenger_name: str) -> JsonDict:
        return self._make_request("GET", f"/messengers/by-name/{messenger_name}")  # type: ignore[return-value]

    def get_all_messengers(
        self, page: int = 1, page_size: int = 50, order_by: Optional[str] = None
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        return self._make_request("GET", "/messengers", params=params)  # type: ignore[return-value]

    def create_messenger_related_user(
        self,
        user_id: int,
        messenger_id: int,
        messenger_user_id: str,
        chat_id: str,
    ) -> JsonDict:
        """All four fields are required by the API."""
        data = {
            "user_id": user_id,
            "messenger_id": messenger_id,
            "messenger_user_id": messenger_user_id,
            "chat_id": chat_id,
        }
        return self._make_request("POST", "/messengerRelatedUsers", data=data)  # type: ignore[return-value]

    def get_messenger_related_user(
        self,
        chat_id: str,
        messenger_user_id: str,
        *,
        user_id: Optional[int] = None,
        messenger_id: Optional[int] = None,
    ) -> JsonDict:
        """chat_id and messenger_user_id are required query params."""
        params: JsonDict = {"chat_id": chat_id, "messenger_user_id": messenger_user_id}
        if user_id is not None:
            params["user_id"] = user_id
        if messenger_id is not None:
            params["messenger_id"] = messenger_id
        return self._make_request("GET", "/messengerRelatedUsers", params=params)  # type: ignore[return-value]

    def get_all_messenger_related_users(
        self,
        page: int = 1,
        page_size: int = 50,
        *,
        order_by: Optional[str] = None,
        user_id: Optional[int] = None,
        chat_id: Optional[str] = None,
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        if user_id is not None:
            params["user_id"] = user_id
        if chat_id:
            params["chat_id"] = chat_id
        return self._make_request("GET", "/messengerRelatedUsers/all", params=params)  # type: ignore[return-value]

    def get_user_id_by_messenger_user_id(self, messenger_user_id: str) -> JsonDict:
        return self._make_request(  # type: ignore[return-value]
            "GET", f"/messengerRelatedUsers/{messenger_user_id}/user"
        )

    # --- Backlogs ---------------------------------------------------------------

    def create_backlog(
        self,
        title: str,
        user_id: int,
        *,
        description: Optional[str] = None,
        messenger_related_user_id: Optional[int] = None,
    ) -> JsonDict:
        data: JsonDict = {"title": title, "user_id": user_id}
        if description is not None:
            data["description"] = description
        if messenger_related_user_id is not None:
            data["messenger_related_user_id"] = messenger_related_user_id
        return self._make_request("POST", "/backlogs", data=data)  # type: ignore[return-value]

    def create_backlogs_batch(
        self,
        items: str,
        user_id: int,
        *,
        separator: str = "\n",
        messenger_related_user_id: Optional[int] = None,
    ) -> JsonDict:
        data: JsonDict = {"items": items, "user_id": user_id, "separator": separator}
        if messenger_related_user_id is not None:
            data["messenger_related_user_id"] = messenger_related_user_id
        return self._make_request("POST", "/backlogs/batch", data=data)  # type: ignore[return-value]

    def get_backlog(self, backlog_id: int) -> JsonDict:
        return self._make_request("GET", f"/backlogs/{backlog_id}")  # type: ignore[return-value]

    def get_all_backlogs(
        self,
        page: int = 1,
        page_size: int = 50,
        *,
        order_by: Optional[str] = None,
        user_id: Optional[int] = None,
        completed: Optional[bool] = None,
        messenger_user_id: Optional[str] = None,
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        if user_id is not None:
            params["user_id"] = user_id
        if completed is not None:
            params["completed"] = completed
        if messenger_user_id:
            params["messenger_user_id"] = messenger_user_id
        return self._make_request("GET", "/backlogs", params=params)  # type: ignore[return-value]

    def update_backlog(
        self,
        backlog_id: int,
        *,
        title: Optional[str] = None,
        description: Optional[str] = None,
        completed_at: Optional[datetime] = None,
    ) -> JsonDict:
        data: JsonDict = {}
        if title is not None:
            data["title"] = title
        if description is not None:
            data["description"] = description
        if completed_at is not None:
            data["completed_at"] = _rfc3339(completed_at)
        return self._make_request("PUT", f"/backlogs/{backlog_id}", data=data)  # type: ignore[return-value]

    def delete_backlog(self, backlog_id: int) -> None:
        self._make_request("DELETE", f"/backlogs/{backlog_id}")

    # --- Targets ----------------------------------------------------------------

    def create_target(
        self,
        title: str,
        user_id: int,
        *,
        description: Optional[str] = None,
        messenger_related_user_id: Optional[int] = None,
    ) -> JsonDict:
        data: JsonDict = {"title": title, "user_id": user_id}
        if description is not None:
            data["description"] = description
        if messenger_related_user_id is not None:
            data["messenger_related_user_id"] = messenger_related_user_id
        return self._make_request("POST", "/targets", data=data)  # type: ignore[return-value]

    def get_target(self, target_id: int) -> JsonDict:
        return self._make_request("GET", f"/targets/{target_id}")  # type: ignore[return-value]

    def get_all_targets(
        self,
        page: int = 1,
        page_size: int = 50,
        *,
        order_by: Optional[str] = None,
        user_id: Optional[int] = None,
        messenger_user_id: Optional[str] = None,
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        if user_id is not None:
            params["user_id"] = user_id
        if messenger_user_id:
            params["messenger_user_id"] = messenger_user_id
        return self._make_request("GET", "/targets", params=params)  # type: ignore[return-value]

    def update_target(
        self,
        target_id: int,
        *,
        title: Optional[str] = None,
        description: Optional[str] = None,
        completed_at: Optional[datetime] = None,
    ) -> JsonDict:
        data: JsonDict = {}
        if title is not None:
            data["title"] = title
        if description is not None:
            data["description"] = description
        if completed_at is not None:
            data["completed_at"] = _rfc3339(completed_at)
        return self._make_request("PUT", f"/targets/{target_id}", data=data)  # type: ignore[return-value]

    def delete_target(self, target_id: int) -> None:
        self._make_request("DELETE", f"/targets/{target_id}")

    # --- Digests ----------------------------------------------------------------

    def get_digest(
        self,
        user_id: int,
        *,
        messenger_related_user_id: Optional[int] = None,
        messenger_user_id: Optional[str] = None,
        start_date_from: Optional[datetime] = None,
        start_date_to: Optional[datetime] = None,
    ) -> JsonDict:
        params: JsonDict = {"user_id": user_id}
        if messenger_related_user_id is not None:
            params["messenger_related_user_id"] = messenger_related_user_id
        if messenger_user_id:
            params["messenger_user_id"] = messenger_user_id
        if start_date_from:
            params["start_date_from"] = _rfc3339(start_date_from)
        if start_date_to:
            params["start_date_to"] = _rfc3339(start_date_to)
        return self._make_request("GET", "/digests", params=params)  # type: ignore[return-value]

    def create_digest_settings(
        self,
        user_id: int,
        weekday_time: str,
        weekend_time: str,
        *,
        enabled: bool = True,
        messenger_related_user_id: Optional[int] = None,
    ) -> JsonDict:
        data: JsonDict = {
            "user_id": user_id,
            "weekday_time": weekday_time,
            "weekend_time": weekend_time,
            "enabled": enabled,
        }
        if messenger_related_user_id is not None:
            data["messenger_related_user_id"] = messenger_related_user_id
        return self._make_request("POST", "/digests/settings", data=data)  # type: ignore[return-value]

    def get_digest_settings(
        self,
        user_id: int,
        *,
        messenger_related_user_id: Optional[int] = None,
    ) -> JsonDict:
        params: JsonDict = {"user_id": user_id}
        if messenger_related_user_id is not None:
            params["messenger_related_user_id"] = messenger_related_user_id
        return self._make_request("GET", "/digests/settings", params=params)  # type: ignore[return-value]

    def update_digest_settings(
        self,
        user_id: int,
        *,
        enabled: Optional[bool] = None,
        weekday_time: Optional[str] = None,
        weekend_time: Optional[str] = None,
        messenger_related_user_id: Optional[int] = None,
    ) -> JsonDict:
        data: JsonDict = {}
        if enabled is not None:
            data["enabled"] = enabled
        if weekday_time is not None:
            data["weekday_time"] = weekday_time
        if weekend_time is not None:
            data["weekend_time"] = weekend_time
        if messenger_related_user_id is not None:
            data["messenger_related_user_id"] = messenger_related_user_id
        params: JsonDict = {"user_id": user_id}
        if messenger_related_user_id is not None:
            params["messenger_related_user_id"] = messenger_related_user_id
        return self._make_request(  # type: ignore[return-value]
            "PUT", "/digests/settings", data=data, params=params
        )

    def delete_digest_settings(
        self,
        user_id: int,
        *,
        messenger_related_user_id: Optional[int] = None,
    ) -> None:
        params: JsonDict = {"user_id": user_id}
        if messenger_related_user_id is not None:
            params["messenger_related_user_id"] = messenger_related_user_id
        self._make_request("DELETE", "/digests/settings", params=params)

    def get_all_digest_settings(
        self, page: int = 1, page_size: int = 50, order_by: Optional[str] = None
    ) -> JsonDict:
        params: JsonDict = {"page": page, "page_size": page_size}
        if order_by:
            params["order_by"] = order_by
        return self._make_request("GET", "/digests/settings/all", params=params)  # type: ignore[return-value]
