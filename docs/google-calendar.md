# Google Calendar integration

GoReminder syncs **Google Calendar Events** (not Google Tasks) with tasks.

## Who can connect what?

| Level | What it means |
|--------|----------------|
| **Per user** | Each GoReminder user connects **their own** Google account via OAuth. Tokens are stored encrypted and scoped to that `user_id`. |
| **Per calendar** | After connect, the user picks one or more Google calendars and creates a **binding** for each (`import` / `export` / `both`). |
| **Not global** | There is no single shared app calendar for all users. User A cannot see or sync User B’s Google calendars. |

So: **any user** can connect **their** Google account and **as many calendars** as they need. Bindings are independent (different directions, groups, delete policies).

---

## 1. Google Cloud setup (once per deployment)

1. Open [Google Cloud Console](https://console.cloud.google.com/) → create or select a project.
2. **APIs & Services → Library** → enable **Google Calendar API**.
3. Configure the OAuth app (UI name varies — see note below):
   - Open **APIs & Services → OAuth consent screen**, or in newer console **Google Auth Platform** (Branding / Audience).
   - Fill **App name**, **User support email**, **Developer contact**.
   - Scopes used by GoReminder (add if the wizard asks, or under Data access / Scopes):
     - `https://www.googleapis.com/auth/calendar.events`
     - `https://www.googleapis.com/auth/calendar.readonly`
     - `https://www.googleapis.com/auth/userinfo.email`
     - `https://www.googleapis.com/auth/userinfo.profile`
   - While status is **Testing**, add yourself under **Test users** / **Audience → Test users**.
4. **Credentials → Create credentials → OAuth client ID**
   - Application type: **Web application**.
   - Authorized redirect URI (must match config exactly), e.g.  
     `http://localhost:8080/api/v1/calendar/oauth/callback`  
     or your public URL:  
     `https://your.domain/api/v1/calendar/oauth/callback`
5. Copy **Client ID** and **Client secret**.

### OAuth consent UI notes (current Google Cloud Console)

| What you might look for | What you often see now |
|-------------------------|-------------------------|
| **External** vs **Internal** | **Internal** only for Google Workspace orgs. On a personal Gmail project there is often **no External toggle** — the app is already “external”; just complete Branding and add Test users. |
| Left nav **OAuth consent screen** | May appear as **Google Auth Platform** → **Branding**, **Audience**, **Clients**, **Data access**. |
| Where to add Test users | **Audience** (or consent screen summary) → **Test users** → Add your Gmail. Required while Publishing status is **Testing**. |
| `access_denied` / app not verified | Almost always missing Test user, or you signed in with a different Google account than the one listed. |

You do **not** need to publish the app to production for local testing; Testing + your email as Test user is enough.

---

## 2. Server config

In `cmd/core/config.yaml` (or env `GOREMINDER_GOOGLECALENDAR_*`):

```yaml
googleCalendar:
  enabled: true
  clientID: "....apps.googleusercontent.com"
  clientSecret: "...."
  redirectURL: "http://localhost:8080/api/v1/calendar/oauth/callback"
  # Any secret string (SHA-256 hashed) or 64-char hex (32-byte key)
  tokenEncryptionKey: "change-me-to-a-long-random-secret"
  syncInterval: "5m"
  defaultEventDurationMinutes: 30
  initialSyncWindowDays: 90
  # Optional gate for the whole API when enabled
  apiKeyEnabled: false
  apiKey: ""
```

Restart the API. On startup you should see `google calendar integration enabled`.

Background job: every `syncInterval` the core polls import bindings (syncToken) and processes the export outbox.

---

## 3. Per-user connect flow

Replace `{user_id}` with the GoReminder user id.

### 3.1 Start OAuth

```http
GET /api/v1/users/{user_id}/calendar/oauth/start
```

Response:

```json
{ "url": "https://accounts.google.com/o/oauth2/auth?..." }
```

Open `url` in a browser, sign in with Google, grant access. Google redirects to `redirectURL` with `code` and `state` (state = user id). The callback handler stores encrypted tokens.

```http
GET /api/v1/calendar/oauth/callback?code=...&state={user_id}
```

### 3.2 List Google calendars

```http
GET /api/v1/users/{user_id}/calendar/calendars
```

### 3.3 Create a binding (one calendar)

```http
POST /api/v1/users/{user_id}/calendar/bindings
Content-Type: application/json

{
  "google_calendar_id": "primary",
  "calendar_summary": "Personal",
  "direction": "import",
  "group_id": 1,
  "messenger_related_user_id": 1,
  "delete_policy": "soft_delete_imported"
}
```

| Field | Values |
|--------|--------|
| `direction` | `import` — Google → tasks; `export` — tasks → Google; `both` |
| `group_id` | Optional task group: imported tasks go there; for export, only tasks in that group are pushed (if set). If `group_id` is omitted on an **export** binding, eligible tasks for that user may be exported. |
| `messenger_related_user_id` | Optional. When set on an **import**/**both** binding, imported tasks get this `mru` and future occurrences are published to the messenger worker (`schedule_task`). Omit to keep calendar-only tasks (DB mirror, no chat reminders). Must belong to the same user. |
| `delete_policy` | On unbind/cancel: `soft_delete_imported` (default), `mute_imported`, `keep` |

Repeat `POST .../bindings` for additional calendars.

### Import vs autoreschedule / worker

- **Imported** tasks (`origin=imported`) are **excluded from autoreschedule**. Google does not mark past events “done”; we must not +24h them like overdue reminders.
- **One event / one series → one task** (not expanded instances). Recurring Google events store `rrule` on that task; if DTSTART is already past, import advances `start_date` to the **next** occurrence.
- **Worker**: only if the binding has `messenger_related_user_id`. The queue payload still uses `cron_expression` (worker has no RRULE arg) — so each publish is a **one-shot** at the current `start_date`. After the time passes, the next calendar sync advances `start_date` again and republishes (not autoreschedule day-by-day).
- **Status after the event**: one-shot past events become `done` (with `finish_date`) on the next sync; recurring stay `scheduled` on the next occurrence. Cancel in Google still follows `delete_policy`.
- Cancel/delete in Google (or unbind with soft-delete/mute) sends `delete_task` when a messenger was set.

### 3.4 Force sync / list / disconnect

```http
GET    /api/v1/users/{user_id}/calendar/bindings
POST   /api/v1/users/{user_id}/calendar/bindings/{binding_id}/sync
DELETE /api/v1/users/{user_id}/calendar/bindings/{binding_id}
DELETE /api/v1/users/{user_id}/calendar/disconnect
```

---

## 4. Export options

**By group:** create/use a task group, put tasks in it (`group_id` on the task), bind a calendar with `direction: export|both` and the same `group_id`.

**By single task:**

```http
POST /api/v1/tasks/{task_id}/calendar/export
Content-Type: application/json

{ "calendar_binding_id": 1 }
```

Confirmation **child** tasks are not exported separately; the parent / logical series is.

---

## 5. Seeing synced tasks

- Task detail / list may include:

```json
"external": {
  "provider": "google_calendar",
  "calendar_id": "...",
  "event_id": "...",
  "origin": "imported",
  "sync_enabled": true
}
```

- Filter: `GET /api/v1/users/{user_id}/tasks?external_provider=google_calendar`
- Explicit: `GET /api/v1/tasks/{id}/calendar/external`

---

## 6. Security notes

- Refresh/access tokens are **encrypted at rest** (`tokenEncryptionKey`). Keep it secret and stable; rotating it invalidates stored tokens (users must reconnect).
- Optional `apiKeyEnabled` + `X-API-Key` header when you want a shared gate on the API.
- Calendar ACL stays on Google’s side: anyone who can edit that Google calendar can change events; GoReminder will pick up changes on the next sync.
- Loop prevention: exported events get private extended property `goreminder_task_id`.

---

## 7. Quick checklist

1. Enable Calendar API + OAuth client + redirect URI  
2. Set `googleCalendar.enabled` and secrets in config  
3. Create GoReminder user if needed  
4. OAuth start → consent → binding(s)  
5. Force sync or wait for `syncInterval`  
6. Check tasks / `external` badge  

---

## 8. Local smoke-test algorithm (full feature walkthrough)

Goal: on your machine, exercise **OAuth → import → export (group) → export (single task) → force sync → filter/badge → disconnect**.

Assume API at `http://localhost:8080`. Base: `API=http://localhost:8080/api/v1`.

### A. One-time Google Cloud (5–10 min)

1. Create/select a GCP project.
2. Enable **Google Calendar API**.
3. Open **OAuth consent screen** / **Google Auth Platform**:
   - Fill Branding (app name, support email, developer contact).
   - Do **not** expect an **External** button on personal Gmail — that choice is mainly for Workspace (**Internal** vs external). Skip if you do not see it.
   - Under **Audience → Test users**, add **yourself** (the Google account you will use in the browser).
4. Create OAuth client → **Web application**.
5. Authorized redirect URI (exact match):

   `http://localhost:8080/api/v1/calendar/oauth/callback`

6. Save Client ID + Client secret.

### B. Local deps + config

1. Start Postgres (and optionally Rabbit):

   ```bash
   docker compose -f docker-compose.dev.yml up -d postgres
   # optional: rabbitmq if you want queue publishes
   ```

2. In [`cmd/core/config.yaml`](../cmd/core/config.yaml) uncomment/set:

   ```yaml
   googleCalendar:
     enabled: true
     clientID: "<from Google>"
     clientSecret: "<from Google>"
     redirectURL: "http://localhost:8080/api/v1/calendar/oauth/callback"
     tokenEncryptionKey: "local-dev-secret-change-me"
     syncInterval: "1m"          # faster feedback while testing
     defaultEventDurationMinutes: 30
     initialSyncWindowDays: 90
     apiKeyEnabled: false
   ```

3. Run API (migrations apply on start unless `SKIP_MIGRATIONS=true`):

   ```bash
   make run
   # or: go run ./cmd/core -config cmd/core/config.yaml
   ```

4. Log line to expect: `google calendar integration enabled`.

5. Health check: `curl -s http://localhost:8080/version`

### C. Create a GoReminder user

```bash
curl -s -X POST "$API/users" -H 'Content-Type: application/json' \
  -d '{"name":"calendar-tester","email":"you@example.com"}'
# → {"id": <USER_ID>}
export USER_ID=<USER_ID>
```

Optional: create a task group for import/export scoping:

```bash
curl -s -X POST "$API/task-groups" -H 'Content-Type: application/json' \
  -d "{\"user_id\": $USER_ID, \"name\": \"Google Personal\"}"
# → {"id": <GROUP_ID>}
export GROUP_ID=<GROUP_ID>
```

### D. Connect Google (OAuth)

1. Get consent URL:

   ```bash
   curl -s "$API/users/$USER_ID/calendar/oauth/start"
   # → {"url":"https://accounts.google.com/..."}
   ```

2. Open `url` in a **browser on the same machine** (localhost callback).
3. Sign in with the Google account you added as Test user → Allow.
4. Browser lands on `/calendar/oauth/callback?code=...&state=$USER_ID` and returns JSON with account email (tokens stored encrypted).
5. If you see `redirect_uri_mismatch` — URI in Google Console ≠ `redirectURL` in config.

### E. List calendars and create bindings

```bash
curl -s "$API/users/$USER_ID/calendar/calendars"
# pick google_calendar_id, often "primary"
```

**Binding 1 — IMPORT only** (pull events into `GROUP_ID`):

```bash
curl -s -X POST "$API/users/$USER_ID/calendar/bindings" \
  -H 'Content-Type: application/json' \
  -d "{
    \"google_calendar_id\": \"primary\",
    \"calendar_summary\": \"Primary import\",
    \"direction\": \"import\",
    \"group_id\": $GROUP_ID,
    \"delete_policy\": \"soft_delete_imported\"
  }"
# → binding id → export IMPORT_BINDING_ID=...
```

**Binding 2 — EXPORT only** (optional second calendar, or same calendar with care):

For a clean demo, create a dedicated Google calendar “GoReminder Export” in Google UI, then:

```bash
curl -s -X POST "$API/users/$USER_ID/calendar/bindings" \
  -H 'Content-Type: application/json' \
  -d "{
    \"google_calendar_id\": \"<export-calendar-id>\",
    \"calendar_summary\": \"Export\",
    \"direction\": \"export\",
    \"group_id\": $GROUP_ID
  }"
# → EXPORT_BINDING_ID=...
```

Using `both` on one calendar is fine too; duplicates are prevented via `goreminder_task_id` on events.

```bash
curl -s "$API/users/$USER_ID/calendar/bindings"
```

### F. Test IMPORT (Google → GoReminder)

1. In Google Calendar UI, create 2–3 events in the **import** calendar (one timed, optionally one recurring with RRULE).
2. Force sync:

   ```bash
   curl -s -X POST "$API/users/$USER_ID/calendar/bindings/$IMPORT_BINDING_ID/sync"
   # → {"status":"synced"}
   ```

3. List tasks (with external badge / filter):

   ```bash
   curl -s "$API/users/$USER_ID/tasks?external_provider=google_calendar&page=1&page_size=50"
   curl -s "$API/tasks/<task_id>"
   # look for "external": { "provider":"google_calendar", "origin":"imported", ... }
   curl -s "$API/tasks/<task_id>/calendar/external"
   ```

4. Change event title/time in Google → force sync again → task should update.
5. Delete/cancel event in Google → sync → with `soft_delete_imported` the task is soft-deleted.

### G. Test EXPORT by group (GoReminder → Google)

1. Create a future task **in the export group**:

   ```bash
   curl -s -X POST "$API/tasks" -H 'Content-Type: application/json' \
     -d "{
       \"title\": \"Export me (group)\",
       \"user_id\": $USER_ID,
       \"group_id\": $GROUP_ID,
       \"start_date\": \"2030-01-15T10:00:00Z\"
     }"
   # → task id
   ```

2. Wait up to `syncInterval` **or** trigger outbox by updating the task (title change) — create/update calls `OnTaskChanged` → outbox → Google.

   ```bash
   curl -s -X PUT "$API/tasks/<task_id>" -H 'Content-Type: application/json' \
     -d '{"title":"Export me (group) updated"}'
   ```

3. Open the **export** calendar in Google UI — event should appear (duration ≈ 30 min).
4. Check link:

   ```bash
   curl -s "$API/tasks/<task_id>/calendar/external"
   # origin: exported
   ```

### H. Test EXPORT by single task (no group required)

```bash
# task without group_id is fine
curl -s -X POST "$API/tasks" -H 'Content-Type: application/json' \
  -d "{
    \"title\": \"Export me (solo)\",
    \"user_id\": $USER_ID,
    \"start_date\": \"2030-02-01T12:00:00Z\"
  }"

curl -s -X POST "$API/tasks/<task_id>/calendar/export" \
  -H 'Content-Type: application/json' \
  -d "{\"calendar_binding_id\": $EXPORT_BINDING_ID}"
```

Binding must be `export` or `both`. Then check Google + `/calendar/external`.

### I. What the background job does

With `syncInterval: "1m"`:

- polls all **active import/both** bindings (incremental `syncToken`);
- drains **export outbox** (retries with backoff on failure).

Errors land on binding: `last_error`, `status: error` in `GET .../bindings`.

### J. Disconnect / cleanup

```bash
# remove one binding (applies delete_policy to imported tasks)
curl -s -X DELETE "$API/users/$USER_ID/calendar/bindings/$IMPORT_BINDING_ID"

# revoke Google tokens + tear down remaining bindings
curl -s -X DELETE "$API/users/$USER_ID/calendar/disconnect"
```

### K. Suggested order to “see everything”

| Step | Feature exercised |
|------|-------------------|
| A–D | OAuth, encrypted tokens, per-user account |
| E | Multi-calendar, direction import vs export, group scope |
| F | Import, syncToken/force sync, external badge, delete_policy |
| G | Group export + outbox |
| H | Per-task `sync_enabled` export |
| I | Scheduler / retries |
| J | Unbind + disconnect |

### L. Common local pitfalls

| Symptom | Fix |
|---------|-----|
| `redirect_uri_mismatch` | URI in Google Console must equal `redirectURL` exactly |
| `access_denied` / app not verified | Add your Google account as **Test user** (Audience / consent screen). Sign in with that same account. Personal Gmail: no External toggle needed — see §1 notes. |
| Callback 404 / connection refused | API not running on `:8080`, or browser not on same host |
| Import empty | Events outside `initialSyncWindowDays` (±90); or wrong calendar id |
| Export silent | Binding not `export`/`both`; or `group_id` set and task not in that group; wait for outbox/`syncInterval` |
| `googleCalendar.clientID is required` | `enabled: true` without credentials — fill config and restart |
