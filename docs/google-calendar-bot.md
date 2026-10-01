# Google Calendar — guide for Telegram bot (and other messengers)

How to expose GoReminder’s Google Calendar API in an **external** bot (Python Telegram, etc.) without dumping every existing task into Google.

Canonical API/behavior: [google-calendar.md](./google-calendar.md) (matrix, OAuth, bindings). This doc is a **product + integration playbook**.

---

## 0. Golden rules (so current tasks do not flood Google)

1. **Never** create an `export` / `both` binding **without** `group_id` unless you truly want all top-level tasks of that user pushed.
2. Prefer:
   - **Import** calendars → binding `direction=import` + optional `group_id` + **`messenger_related_user_id` (mru)** if reminders in TG are needed.
   - **Export** only tasks the user opted in: either put them in an “Export to Google” **task group** bound with `export`/`both` + that `group_id`, or call per-task `POST /tasks/{id}/calendar/export`.
3. Existing tasks stay local until the user assigns `group_id` or enables per-task export.
4. Always pass **`mru_id`** of the current TG chat when creating import/both bindings that should notify.

---

## 1. Concepts the bot must track

| Concept | Where | Bot should store / show |
|--------|--------|-------------------------|
| GoReminder `user_id` | already from TG↔user link | session |
| `messenger_related_user_id` (**mru**) | `user_messengers` for this chat | always available in handlers |
| Google connected? | `GET .../calendar/bindings` or try `.../calendars` | “Google: connected / not” |
| Bindings | id, calendar, direction, group_id, mru, status, last_error, sync_attempts, next_retry_at | settings screen |
| Sync status | `GET .../calendar/sync-status` → bindings + outbox `{pending,processing,failed}` | health / “последний sync” |
| Task groups | `/task-groups` | list / pick for import-export |
| Task ↔ Google | `task.external` or `GET .../calendar/external` | badge “📅”, bind/unbind UI |

---

## 2. Recommended bot UX map

### A. Settings → Google Calendar

| User action | Bot step | API |
|-------------|----------|-----|
| **Connect Google** | Send OAuth link (button URL) | `GET /api/v1/users/{user_id}/calendar/oauth/start` → open `url` in browser |
| After consent | “Проверьте статус” / auto-poll | `GET .../calendar/calendars` or `.../bindings` succeeds |
| **Disconnect** | Confirm | `DELETE .../calendar/disconnect` (all calendars) or delete one binding |
| **List calendars** | Inline keyboard of calendars | `GET .../calendar/calendars` |
| **Add calendar** | Wizard: calendar → direction → group? → mru | `POST .../calendar/bindings` |
| **Sync now** | Button on binding | `POST .../bindings/{id}/sync` |
| **Errors** | Show `status` / `last_error` / outbox failed | `GET .../sync-status` (or bindings list) |

OAuth note: callback hits **core API** (`redirectURL`), not the bot. Bot only starts the flow and then refreshes status. Deep-link back to bot (`t.me/bot?start=google_ok`) is optional UX sugar after callback page.

### B. Task groups

| Action | API |
|--------|-----|
| Create “Из Google” / “В Google” | `POST /api/v1/task-groups` `{ name, user_id }` |
| List / rename / delete | `GET/PUT/DELETE /task-groups...` | **409** if calendar binding still has this `group_id` — unbind calendar first |
| Put task in group | `PUT /tasks/{id}` `{ "group_id": N }` or `0` to clear |

Suggested default groups (create on first Google setup, names localized):

- `Google · Import` — target for import bindings  
- `Google · Export` — only tasks user wants in calendar  

### C. Bind calendar to group (import / export / both)

Wizard example:

1. Pick Google calendar id  
2. Pick direction: `import` | `export` | `both`  
3. Pick task group (required for export/both; recommended for import)  
4. For import/both that should remind in TG: set `messenger_related_user_id` = current mru  

```http
POST /api/v1/users/{user_id}/calendar/bindings
Content-Type: application/json

{
  "google_calendar_id": "primary",
  "calendar_summary": "Work",
  "direction": "import",
  "group_id": 12,
  "messenger_related_user_id": 34,
  "delete_policy": "soft_delete_imported"
}
```

| direction | Effect for the bot |
|-----------|-------------------|
| `import` | Events → tasks in `group_id`; bot edits do **not** push to Google |
| `export` | Only tasks in that group (or per-task export) → Google; no pull |
| `both` | Both ways — warn user about overwrite races |

### D. List tasks / task card — bind / unbind Google

**Badge:** if `external` present (`provider=google_calendar`) → show 📅 + origin `imported`/`exported`.

**Bind (export one task without putting whole backlog in a group):**

```http
POST /api/v1/tasks/{task_id}/calendar/export
{ "calendar_binding_id": <export_or_both_binding_id> }
```

Binding must be `export` or `both`. Prefer a dedicated export binding with `group_id` so accidental “all tasks” never happens. Per-task export sets **`export_opt_in=true`**: the Google event stays even if the task later leaves the export group. Group-only exports (`export_opt_in=false`) **remove** the Google event when the task leaves that group.

**Unbind today:**

- There is **no** `DELETE .../calendar/export` yet. Options:
  - soft-delete the Google event by deleting the task (heavy), or  
  - remove binding (affects many tasks), or  
  - product gap: add “disable sync” API later (`sync_enabled=false` / delete sync link).  
- For **imported** tasks: don’t “unbind export”; user manages the **import binding** or deletes the task / waits for Google cancel + `delete_policy`.

**Mute / pre-remind:** messenger-only; do not present as calendar controls (see matrix in main doc).

### E. Import one or many calendars

For each chosen calendar → one binding (`import` or `both`) + group + mru. Then:

- wait `syncInterval`, or  
- `POST .../bindings/{id}/sync`

Filter imported: `GET /users/{id}/tasks?external_provider=google_calendar` (and/or show `group_id`).

### F. Digests / today / tomorrow / list — grouping

API today:

- Digests: `GET /digests?user_id=&messenger_related_user_id=&start_date_from=&start_date_to=` — **no** group/external filter.  
- Task list: `external_provider`, mru filters; **`group_id` query filter not implemented** — tasks still return `group_id` on each item.

**Bot-side (recommended now):**

1. Load digests/tasks as today.  
2. Optionally partition in UI:
   - **No group** / **By group name** (join with `/task-groups`)  
   - **📅 From Google** (`external` present) vs **Local only**  
3. Settings toggle: `digest_group_by: off | group | source`.

Future API (nice-to-have): `group_id=` filter, `external_provider=` on digests — not required for v1 bot.

---

## 3. mru_id checklist

| Flow | Use mru |
|------|---------|
| Import/both binding | Set `messenger_related_user_id` so imported tasks get reminders in **this** TG chat |
| Export-only binding | mru optional (export doesn’t schedule); tasks keep their own mru |
| List / digest | Pass current chat’s mru so user sees their channel’s tasks |
| Multi-chat same user | Separate mru per chat; don’t reuse another chat’s mru on binding |

Without mru on import binding: tasks appear in DB/list but **won’t** get `schedule_task` from calendar sync.

---

## 4. Errors the bot should handle

| Signal | User-facing | Bot action |
|--------|-------------|------------|
| Binding `status=error` | Import pull broken (auth / calendar gone / network). Export may **still** push | Show `last_error`; force sync / reconnect. Do **not** assume calendar export is paused |
| `GET .../sync-status` | Bindings + outbox `{pending,processing,failed}` | Settings / periodic health; notify on transition to error or outbox failed |
| `access_denied` on OAuth | “Добавьте аккаунт в Test users” / app not verified | Link to admin docs; Testing = max ~100 users |
| Empty calendars list | Not connected or revoked | Connect flow |
| Export “silent” | Binding not export/both; or task not in binding `group_id` (and no opt-in); wait outbox | Explain group / per-task export |
| DELETE task group **409** | Binding still references `group_id` | Delete/reassign binding first |
| History | Import writes `source: google_calendar_import` | Optional “история: из Google” |

Do **not** spam OAuth start on every transient error — only on sticky auth failures (see main doc).

---

## 5. Suggested screen order (MVP)

1. Settings → Google → Connect / Disconnect / status  
2. Task groups CRUD (at least Import + Export groups)  
3. “Add calendar” wizard → binding with **group_id always set for export/both**  
4. Task list: 📅 badge; menu “Добавить в Google” → per-task export or move to Export group  
5. Force sync button on import bindings  
6. Digest/list: optional group-by group / Google vs local (client-side)  

Skip for MVP: editing Google event fields from bot when binding is `import` (will be overwritten); calendar reminders UI for `pre_remind` (not synced to Google).

---

## 6. Sequence (happy path)

```
TG user
  → bot: Connect Google
  → API oauth/start → browser consent → API callback
  → bot: Create groups (Import/Export)
  → bot: List calendars → Create binding import+group+mru
  → bot: Create binding export+Export group (no mru required)
  → user moves few tasks into Export group OR per-task export
  → sync / wait interval
  → list tasks shows external badges; digests optionally grouped
```

---

## 7. API gap list (for later core work)

Useful if you implement the bot and hit walls:

| Gap | Why |
|-----|-----|
| No per-task **disable** export / delete sync link | “Отвязать от Google” incomplete |
| No `group_id` filter on `GET .../tasks` | Bot filters in memory |
| Digest has no group/external split | Bot-side grouping |
| OAuth callback is JSON page, not bot deep-link | Bot polls or custom HTML callback |

---

## 8. Pointers

- Behavior matrix (import/export/both × task types, mute, autoreschedule, history): [google-calendar.md](./google-calendar.md#behavior-matrix-direction--task-type)  
- Server smoke test (curl): [google-calendar.md § local smoke](./google-calendar.md)  
- Task groups REST: `/api/v1/task-groups`  
