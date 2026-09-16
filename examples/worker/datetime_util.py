"""Flexible RFC3339 / ISO8601 parsing (incl. fractional seconds from Go JSON)."""

from __future__ import annotations

import re
from datetime import datetime, timezone


_FRAC = re.compile(
    r"^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\.(\d+)([+-]\d{2}:\d{2})$"
)


def parse_datetime(value: str | datetime | None) -> datetime | None:
    if value is None or value == "":
        return None
    if isinstance(value, datetime):
        dt = value
        if dt.tzinfo is None:
            return dt.replace(tzinfo=timezone.utc)
        return dt.astimezone(timezone.utc)

    normalized = str(value).replace("Z", "+00:00")
    match = _FRAC.match(normalized)
    if match:
        base, frac, tz = match.groups()
        frac = frac[:6].ljust(6, "0")
        normalized = f"{base}.{frac}{tz}"

    try:
        return datetime.fromisoformat(normalized).astimezone(timezone.utc)
    except ValueError as exc:
        raise ValueError(f"Invalid datetime format: {value}") from exc
