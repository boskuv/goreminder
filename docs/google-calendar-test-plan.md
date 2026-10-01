# Google Calendar — step-by-step API test plan (test & prod)

Walkthrough of **every** calendar-related endpoint: what to call, expected HTTP status/body, and how to verify **import / export / both**.

Companion docs: [google-calendar.md](./google-calendar.md) (behavior), [google-calendar-bot.md](./google-calendar-bot.md) (bot UX).

---

## 0. Test vs prod differences

| | **Test / local** | **Prod** |
|--|--|--|
| Google OAuth app | Publishing **Testing** + your Gmail in **Test users** (~100 max) | **In production** (+ verification for sensitive Calendar scopes) |
| `redirectURL` | `http://localhost:8080/api/v1/calendar/oauth/callback` | Public HTTPS URL registered in GCP |
| API base | `http://localhost:8080/api/v1` | `https://<api-host>/api/v1` |
| `apiKeyEnabled` | usually `false` | often `true` → header `X-API-Key: ...` on every call |
| `syncInterval` | `1m` for faster feedback | `5m` (or your SLA) |
| Calendars | Use **separate** Google calendars: e.g. `GR Import`, `GR Export` (avoid mixing on `primary` while learning) | Same pattern per user |
| Mass export risk | Always set **`group_id`** on `export`/`both` | Same — never ship bot that creates export without group |

Optional header for all curls when API key is on:

```bash
export HAUTH=(-H "X-API-Key: $API_KEY")
# use: curl "${HAUTH[@]}" ...
```

```bash
export API=http://localhost:8080/api/v1   # or prod URL
# jq optional but recommended
```

---

## 1. Prerequisites (once)

### 1.1 Health

```bash
curl -sS http://localhost:8080/version
# Expect: 200, JSON/text with version — proves API is up
```

Log on core start: `google calendar integration enabled`.

### 1.2 User

```bash
curl -sS -X POST "$API/users" -H 'Content-Type: application/json' \
  -d '{"name":"cal-tester","email":"you@example.com"}'
# Expect: 201 {"id": <USER_ID>}
export USER_ID=<id>
```

### 1.3 Messenger + mru (needed for import reminders / worker)

```bash
# messenger "telegram" may already exist — get or create
curl -sS "$API/messengers/by-name/telegram"
# or POST /messengers {"name":"telegram"}

curl -sS -X POST "$API/messengerRelatedUsers" -H 'Content-Type: application/json' \
  -d "{\"user_id\": $USER_ID, \"messenger_id\": 1, \"messenger_user_id\": \"tg123\", \"chat_id\": \"12345\"}"
# Expect: 201 {"id": <MRU_ID>}
export MRU_ID=<id>
```

### 1.4 Task groups (Import / Export) — **required for safe export**

```bash
curl -sS -X POST "$API/task-groups" -H 'Content-Type: application/json' \
  -d "{\"user_id\": $USER_ID, \"name\": \"Google Import\"}"
# Expect: 201 {"id": <IMPORT_GROUP_ID>}
export IMPORT_GROUP_ID=<id>

curl -sS -X POST "$API/task-groups" -H 'Content-Type: application/json' \
  -d "{\"user_id\": $USER_ID, \"name\": \"Google Export\"}"
# Expect: 201 {"id": <EXPORT_GROUP_ID>}
export EXPORT_GROUP_ID=<id>

curl -sS "$API/task-groups?user_id=$USER_ID"
# Expect: 200 array including both groups
```

---

## 2. OAuth (connect)

### 2.1 Start

```bash
curl -sS "$API/users/$USER_ID/calendar/oauth/start"
```

| | Expect |
|--|--|
| Status | **200** |
| Body | `{"url":"https://accounts.google.com/o/oauth2/v2/auth?..."}` |
| Failures | `404`/`422` if user missing; `500` if calendar disabled / misconfigured |

Open `url` in browser (same machine for localhost redirect).

### 2.2 Callback (browser, not curl)

Google redirects to:

`GET /api/v1/calendar/oauth/callback?code=...&state=<USER_ID>`

| | Expect |
|--|--|
| Status | **200** |
| Body | Google account (no tokens), e.g. `{"id":1,"user_id":...,"google_sub":"...","email":"...@gmail.com","scopes":"...","created_at":"..."}` |
| Failures | `access_denied` → Test user / wrong Google account; `redirect_uri_mismatch` → URI ≠ config |

**Prod:** same flow; user must use HTTPS redirect; after Production publish, non–test-users can connect (pending verification warnings).

### 2.3 Not connected yet

```bash
curl -sS "$API/users/$USER_ID/calendar/calendars"
# Expect: 4xx/5xx with error (no Google account) until OAuth succeeds
```

---

## 3. Calendars & bindings

### 3.1 List Google calendars

```bash
curl -sS "$API/users/$USER_ID/calendar/calendars"
```

| | Expect |
|--|--|
| Status | **200** |
| Body | `[{"id":"primary","summary":"...","primary":true}, {"id":"...@group.calendar.google.com","summary":"GR Export",...}, ...]` |

Pick:

- `IMPORT_CAL` — calendar for pull (or `primary`)  
- `EXPORT_CAL` — **different** calendar for push (cleaner tests)

```bash
export IMPORT_CAL=primary
export EXPORT_CAL=<id from list>
```

### 3.2 Create binding — **import**

```bash
curl -sS -X POST "$API/users/$USER_ID/calendar/bindings" \
  -H 'Content-Type: application/json' \
  -d "{
    \"google_calendar_id\": \"$IMPORT_CAL\",
    \"calendar_summary\": \"Import\",
    \"direction\": \"import\",
    \"group_id\": $IMPORT_GROUP_ID,
    \"messenger_related_user_id\": $MRU_ID,
    \"delete_policy\": \"soft_delete_imported\"
  }"
```

| | Expect |
|--|--|
| Status | **201** |
| Body | `{"id":N,"user_id":...,"google_calendar_id":"...","direction":"import","group_id":...,"messenger_related_user_id":...,"status":"active","delete_policy":"soft_delete_imported","created_at":"...","updated_at":"..."}` |
| Check | `last_error` omitted/null; `last_synced_at` null until first sync |

```bash
export IMPORT_BINDING_ID=<id>
```

### 3.3 Create binding — **export** (always with group)

```bash
curl -sS -X POST "$API/users/$USER_ID/calendar/bindings" \
  -H 'Content-Type: application/json' \
  -d "{
    \"google_calendar_id\": \"$EXPORT_CAL\",
    \"calendar_summary\": \"Export\",
    \"direction\": \"export\",
    \"group_id\": $EXPORT_GROUP_ID
  }"
```

| | Expect |
|--|--|
| Status | **201** |
| Body | `direction":"export"`, `group_id` set, **no** mru required |

```bash
export EXPORT_BINDING_ID=<id>
```

### 3.4 Create binding — **both** (optional third calendar or careful reuse)

```bash
# Prefer a third calendar "GR Both"
curl -sS -X POST "$API/users/$USER_ID/calendar/bindings" \
  -H 'Content-Type: application/json' \
  -d "{
    \"google_calendar_id\": \"$BOTH_CAL\",
    \"direction\": \"both\",
    \"group_id\": $IMPORT_GROUP_ID,
    \"messenger_related_user_id\": $MRU_ID
  }"
```

| | Expect |
|--|--|
| Status | **201**, `direction":"both"` |
| Note | Bot edits **and** Google edits can overwrite each other |

### 3.5 List bindings

```bash
curl -sS "$API/users/$USER_ID/calendar/bindings"
```

| | Expect |
|--|--|
| Status | **200** |
| Body | Array of bindings; find import/export/both by `direction` |

### 3.6 Negative checks

| Call | Expect |
|--|--|
| Export binding **without** `group_id` | **201** (API allows) — **do not** do this in prod bot; would export all top-level tasks |
| `messenger_related_user_id` of another user | **400/422** |
| Binding for unknown calendar id | May **201** then fail on sync (`status=error`, `last_error` set) |

---

## 4. Test **import** direction

### 4.1 Arrange in Google UI

In `IMPORT_CAL` create:

1. Timed event in the future (title `Import Future`)  
2. Past timed event (title `Import Past`)  
3. Optional recurring daily (`Import Daily`)

### 4.2 Force sync

```bash
curl -sS -X POST "$API/users/$USER_ID/calendar/bindings/$IMPORT_BINDING_ID/sync"
```

| | Expect |
|--|--|
| Status | **202** |
| Body | `{"status":"synced"}` |
| On auth fail | **5xx** or sync marks binding `status=error` + `last_error` containing `401` / `Invalid Credentials` |

Re-check binding:

```bash
curl -sS "$API/users/$USER_ID/calendar/bindings" | jq '.[] | select(.id=='$IMPORT_BINDING_ID')'
# Expect: status=active, last_synced_at set, last_error null
```

### 4.3 Verify tasks

```bash
curl -sS "$API/users/$USER_ID/tasks?external_provider=google_calendar&page=1&page_size=50"
```

| Check | Expect |
|--|--|
| Future event | Task `status=scheduled`, `group_id=IMPORT_GROUP_ID`, `messenger_related_user_id=MRU_ID`, `external.origin=imported` |
| Past one-shot | Task `status=done`, `finish_date` set |
| Recurring | One task with `rrule`, `status=scheduled`, `start_date` = next occurrence |
| History | `GET .../tasks/{id}/history` (or user history) has `source: google_calendar_import` |

```bash
curl -sS "$API/tasks/<task_id>/calendar/external"
# Expect: 200 {"provider":"google_calendar","origin":"imported","sync_enabled":true,...}
```

### 4.4 Import isolation (no push)

```bash
# Change title of an imported task via API
curl -sS -X PUT "$API/tasks/<imported_task_id>" -H 'Content-Type: application/json' \
  -d '{"title":"Changed locally"}'
```

In Google UI: title should **stay** original (import-only). Next sync may **overwrite** local title from Google.

### 4.5 Cancel in Google

Delete/cancel the event in Google → force sync → with `soft_delete_imported` task is soft-deleted; history `deleted` + `source: google_calendar_import`.

---

## 5. Test **export** direction

### 5.1 Sanity: existing tasks **not** in Export group must not appear in Google

Create a task **without** `group_id` (or wrong group), wait `syncInterval` / touch update — **no** new event on `EXPORT_CAL`.

### 5.2 Export via group

```bash
# Future one-shot in Export group
START=$(date -u -d '+2 hours' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v+2H +%Y-%m-%dT%H:%M:%SZ)
curl -sS -X POST "$API/tasks" -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $USER_ID,
    \"messenger_related_user_id\": $MRU_ID,
    \"title\": \"Export me\",
    \"start_date\": \"$START\",
    \"group_id\": $EXPORT_GROUP_ID,
    \"status\": \"scheduled\"
  }"
# Expect: 201 {"id":...,"child_id":0}
export TASK_EXPORT_ID=<id>
```

Wait outbox (`syncInterval`) **or** bump title:

```bash
curl -sS -X PUT "$API/tasks/$TASK_EXPORT_ID" -H 'Content-Type: application/json' \
  -d '{"title":"Export me v2"}'
```

| Check | Expect |
|--|--|
| Google `EXPORT_CAL` | Event “Export me v2”, ~30 min duration |
| `GET .../calendar/external` | `origin=exported`, `sync_enabled=true` |
| Autoreschedule later | Does **not** move Google start (by design) |

### 5.3 Export recurring (cron → RRULE)

```bash
curl -sS -X POST "$API/tasks" -H 'Content-Type: application/json' \
  -d "{
    \"user_id\": $USER_ID,
    \"messenger_related_user_id\": $MRU_ID,
    \"title\": \"Daily export\",
    \"start_date\": \"$START\",
    \"group_id\": $EXPORT_GROUP_ID,
    \"cron_expression\": \"0 9 * * *\",
    \"status\": \"scheduled\"
  }"
```

| Check | Expect |
|--|--|
| Google | One series, Recurrence ≈ `FREQ=DAILY` (time from DTSTART) |

### 5.4 Per-task export (opt-in without group membership)

Task **not** in Export group:

```bash
curl -sS -X POST "$API/tasks/$OTHER_TASK_ID/calendar/export" \
  -H 'Content-Type: application/json' \
  -d "{\"calendar_binding_id\": $EXPORT_BINDING_ID}"
```

| | Expect |
|--|--|
| Status | **200** external payload (`origin` exported / pending then real event id after outbox) |
| Fail if binding is import-only | **400** `binding direction must be export or both` |
| External | `export_opt_in=true` |

### 5.5 Delete exported task

```bash
curl -sS -X DELETE "$API/tasks/$TASK_EXPORT_ID"
```

| Check | Expect |
|--|--|
| Google | Event removed (after outbox) |
| External | 404 |

---

## 6. Test **both** direction

Use `BOTH_BINDING_ID` on a dedicated calendar.

| Step | Expect |
|--|--|
| Event created in Google | Appears as imported task in binding `group_id` after sync |
| Task created/updated in Export-eligible way for that group | Pushed to same calendar |
| Edit title in Google then sync | Overwrites local title |
| Edit title in API then outbox | Overwrites Google |
| Anti-loop | Exported events carry `goreminder_task_id`; should not create a second task |

---

## 7. Force sync / disconnect / cleanup

### 7.1 Force sync (import/both only; export binding returns synced no-op for pull)

```bash
curl -sS -X POST "$API/users/$USER_ID/calendar/bindings/$IMPORT_BINDING_ID/sync"
# Expect: 202 {"status":"synced"}
```

### 7.2 Delete one binding

```bash
curl -sS -X DELETE "$API/users/$USER_ID/calendar/bindings/$IMPORT_BINDING_ID"
# Expect: 200/204/OK (handler returns success after policy applied)
```

Imported tasks follow `delete_policy` (soft-delete / mute / keep).

### 7.3 Full disconnect

```bash
curl -sS -X DELETE "$API/users/$USER_ID/calendar/disconnect"
# Expect: success; calendars list fails until new OAuth
```

---

## 8. Endpoint cheat sheet

| Method | Path | Success | Notes |
|--|--|--|--|
| GET | `/users/{id}/calendar/oauth/start` | 200 `{url}` | |
| GET | `/calendar/oauth/callback` | 200 account | Browser |
| GET | `/users/{id}/calendar/calendars` | 200 `[{id,summary,primary}]` | Needs OAuth |
| POST | `/users/{id}/calendar/bindings` | 201 binding | Always `group_id` for export/both in real use |
| GET | `/users/{id}/calendar/bindings` | 200 `[]` | Check `status` / `last_error` / `sync_attempts` / `next_retry_at` |
| GET | `/users/{id}/calendar/sync-status` | 200 `{bindings,outbox}` | Import health + export outbox counts |
| POST | `/users/{id}/calendar/bindings/{bid}/sync` | 202 `{status:synced}` | Import/both pull; recovers `status=error` |
| DELETE | `/users/{id}/calendar/bindings/{bid}` | OK | Applies delete_policy |
| DELETE | `/users/{id}/calendar/disconnect` | OK | Revoke all |
| POST | `/tasks/{id}/calendar/export` | 200 external | export/both only |
| GET | `/tasks/{id}/calendar/external` | 200 / 404 | |
| POST/GET/PUT/DELETE | `/task-groups` | as usual | DELETE **409** if calendar binding references the group |
| GET | `/users/{id}/tasks?external_provider=google_calendar` | 200 | Filter imported/exported links |

---

## 9. Prod go-live checklist

1. GCP: OAuth **Production** (+ verification if required).  
2. `redirectURL` HTTPS matches Console.  
3. `tokenEncryptionKey` stable secret (rotation = force reconnect).  
4. `apiKeyEnabled` + bot sends `X-API-Key`.  
5. Bot never creates export binding without `group_id`.  
6. Import/both bindings set **`messenger_related_user_id`** for the TG chat.  
7. Monitor `GET .../calendar/sync-status` (`status=error`, outbox `failed`) → force sync / reconnect.  
8. Smoke the same steps 2→6 against prod API with a Test user (or verified accounts).

---

## 10. Quick pass/fail matrix

| Scenario | Pass |
|--|--|
| OAuth | Callback 200 + email |
| Import sync | Future → scheduled+external; past → done |
| Import no push | Local title change does not change Google |
| Export group | Only Export-group tasks appear in Google |
| Export no group flood | Tasks outside group absent from Google |
| Cron export | Google shows recurring series |
| Both | Round-trip without duplicate tasks |
| Auth death | `last_error` + reconnect fixes sync |
| History | Import mutations include `source=google_calendar_import` |
