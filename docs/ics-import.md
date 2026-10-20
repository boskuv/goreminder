# ICS import

One-shot upload of an iCalendar (`.ics`) file → create GoReminder tasks. **Not** a live sync (no bindings, syncToken, or outbox).

Field semantics match the **import** column of [google-calendar.md](./google-calendar.md) §3.3 (Google Calendar Events parity).

## Endpoint

```http
POST /api/v1/users/{user_id}/ics/import
Content-Type: multipart/form-data
```

| Form field | Required | Meaning |
|------------|----------|---------|
| `file` | yes | `.ics` body |
| `group_id` | no | Applied to every created task |
| `messenger_related_user_id` | no | Applied to every created task; when set, schedules `worker.schedule_task` for eligible rows (same as calendar import mru) |

Response: created task ids plus per-event skip/error entries. A broken single `VEVENT` does not fail the whole file.

## Field matrix

| GoReminder field | ICS | Behavior |
|------------------|-----|----------|
| `title` | `SUMMARY` | Trimmed; empty → `(untitled event)` |
| `description` | `DESCRIPTION` | As-is |
| `start_date` | `DTSTART` | Converted to UTC; all-day `VALUE=DATE` → UTC midnight |
| `rrule` | first `RRULE` | Body without `RRULE:` prefix; past DTSTART advanced to next occurrence |
| `cron_expression` | — | Not imported |
| duration / `DTEND` | ignored for task row | `finish_date` is completion time, not event end (same as Google) |
| `requires_confirmation` | — | Always **false**; no confirmation children |
| `parent_id` | — | Always top-level |
| `muted`, `shift_from_completion`, `skip_digest`, `pre_remind_before_seconds` | — | Defaults / unset (`VALARM` is **not** mapped) |
| `messenger_related_user_id` | — | From request form, not from file |
| `group_id` | — | From request form |
| `status` / past one-shot | derived | Future → `scheduled`; past one-shot or exhausted series → `done` + `finish_date` |
| `UID` | ignored for identity | v1 always **creates** new tasks (no upsert / dedup) |

Product flags (confirmation, mute, pre-remind, shift, skip_digest) stay **local**. Enable confirmation after import via the normal task API if needed.

History: `task_history` `created` with `source: "ics_import"` in `new_value`.

## Excluded in v1

- `VTODO` / `VJOURNAL` (only `VEVENT`)
- `VALARM` → `pre_remind_before_seconds`
- `EXDATE` / `RDATE` / `RECURRENCE-ID`
- Attachments, attendees, geo, categories
- Live sync / export `.ics` / `X-GOREMINDER-*`
- Confirmation children (export later would use `ShouldExportTask`: parents only)

## Testing

- Unit: `pkg/ics` parse + `internal/service` map/import tables (`testdata/*.ics`)
- Smoke: upload → `GET /api/v1/users/{user_id}/tasks`
- Lint: existing `.golangci.yml` / `make lint` (no new linter rules)
