"""
Minimal Telegram bot that talks to GoReminder via examples/python-client
and accepts worker webhooks on POST /send_message.

Env:
  TELEGRAM_BOT_TOKEN   required
  GOREMINDER_URL       default http://localhost:8080
  GOREMINDER_MESSENGER default telegram  (messenger.type name in API)
  WEBHOOK_HOST         default 0.0.0.0
  WEBHOOK_PORT         default 8001  (worker WEBHOOK_URL)

Flow on /start:
  1) resolve or create messenger type
  2) create GoReminder user (or reuse via messenger_user_id lookup)
  3) create messenger-related user (MRU) linking Telegram chat ↔ user

Worker path:
  examples/worker → POST /send_message → bot.send_message (+ optional inline buttons)
  callback done:<id> / later:<id> → API mark-done / postpone+1h
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import sys
import threading
from datetime import datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any

from requests import HTTPError
from telegram import InlineKeyboardButton, InlineKeyboardMarkup, Update
from telegram.ext import (
    Application,
    CallbackQueryHandler,
    CommandHandler,
    ContextTypes,
    MessageHandler,
    filters,
)

CLIENT_DIR = Path(__file__).resolve().parents[1] / "python-client"
sys.path.insert(0, str(CLIENT_DIR))

from goreminder_client import GoReminderClient  # noqa: E402

logging.basicConfig(
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
    level=logging.INFO,
)
log = logging.getLogger("goreminder-tg-bot")

# Filled in main() so the HTTP thread can call the running bot.
_APP: Application | None = None
_LOOP: asyncio.AbstractEventLoop | None = None


def _api() -> GoReminderClient:
    return GoReminderClient(os.environ.get("GOREMINDER_URL", "http://localhost:8080"))


def _messenger_name() -> str:
    return os.environ.get("GOREMINDER_MESSENGER", "telegram")


def _webhook_bind() -> tuple[str, int]:
    host = os.environ.get("WEBHOOK_HOST", "0.0.0.0")
    port = int(os.environ.get("WEBHOOK_PORT", "8001"))
    return host, port


async def _ensure_link(update: Update, context: ContextTypes.DEFAULT_TYPE) -> dict:
    """Return session dict: user_id, messenger_id, mru_id, chat_id, messenger_user_id."""
    if context.user_data.get("linked"):
        return context.user_data["linked"]

    tg_user = update.effective_user
    chat = update.effective_chat
    if tg_user is None or chat is None:
        raise RuntimeError("missing telegram user/chat")

    messenger_user_id = str(tg_user.id)
    chat_id = str(chat.id)
    client = _api()
    messenger_name = _messenger_name()

    try:
        messenger_id = client.get_messenger_id_by_name(messenger_name)["id"]
    except HTTPError as e:
        if e.response is None or e.response.status_code != 404:
            raise
        messenger_id = client.create_messenger(messenger_name)["id"]

    user_id = None
    try:
        user_id = client.get_user_id_by_messenger_user_id(messenger_user_id)["user_id"]
    except HTTPError as e:
        if e.response is None or e.response.status_code != 404:
            raise

    if user_id is None:
        name = (tg_user.full_name or tg_user.username or f"tg-{messenger_user_id}")[:120]
        user_id = client.create_user(name=name, timezone="UTC", language_code="en")["id"]

    try:
        mru = client.get_messenger_related_user(
            chat_id=chat_id,
            messenger_user_id=messenger_user_id,
            user_id=user_id,
            messenger_id=messenger_id,
        )
        mru_id = mru["id"]
    except HTTPError as e:
        if e.response is None or e.response.status_code not in (404, 422):
            raise
        mru_id = client.create_messenger_related_user(
            user_id=user_id,
            messenger_id=messenger_id,
            messenger_user_id=messenger_user_id,
            chat_id=chat_id,
        )["id"]

    linked = {
        "user_id": user_id,
        "messenger_id": messenger_id,
        "mru_id": mru_id,
        "chat_id": chat_id,
        "messenger_user_id": messenger_user_id,
    }
    context.user_data["linked"] = linked
    return linked


async def cmd_start(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    linked = await _ensure_link(update, context)
    host, port = _webhook_bind()
    await update.message.reply_text(
        "Linked to GoReminder.\n"
        f"user_id={linked['user_id']} mru_id={linked['mru_id']}\n"
        f"webhook listening on {host}:{port}/send_message\n\n"
        "Commands:\n"
        "/tasks — list open-ish tasks\n"
        "/add <title> — create task in ~1 day\n"
        "/done <task_id> — mark done\n"
        "/mute <task_id> / /unmute <task_id>\n"
        "/backlog <title>\n"
        "/digest — fetch digest\n"
        "/help"
    )


async def cmd_help(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    await cmd_start(update, context)


async def cmd_tasks(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    linked = await _ensure_link(update, context)
    client = _api()
    resp = client.get_user_tasks(
        user_id=linked["user_id"],
        page=1,
        page_size=20,
        messenger_related_user_id=linked["mru_id"],
        status_not="done",
    )
    rows = resp.get("data") or []
    if not rows:
        await update.message.reply_text("No tasks.")
        return
    lines = []
    for t in rows:
        lines.append(
            f"#{t['id']} [{t.get('status')}] {t.get('title')} "
            f"{'(muted)' if t.get('muted') else ''}".strip()
        )
    await update.message.reply_text("\n".join(lines)[:4000])


async def cmd_add(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    linked = await _ensure_link(update, context)
    title = " ".join(context.args).strip() if context.args else ""
    if not title:
        await update.message.reply_text("Usage: /add <title>")
        return
    client = _api()
    created = client.create_task(
        title=title,
        user_id=linked["user_id"],
        start_date=datetime.now(timezone.utc) + timedelta(days=1),
        messenger_related_user_id=linked["mru_id"],
        requires_confirmation=True,
    )
    await update.message.reply_text(
        f"Created task id={created['id']} child_id={created.get('child_id', 0)}"
    )


async def cmd_done(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    await _ensure_link(update, context)
    if not context.args:
        await update.message.reply_text("Usage: /done <task_id>")
        return
    task_id = int(context.args[0])
    result = _api().mark_task_as_done(task_id)
    await update.message.reply_text(f"Done: #{result.get('id')} {result.get('title')}")


async def cmd_mute(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    if not context.args:
        await update.message.reply_text("Usage: /mute <task_id>")
        return
    task = _api().mute_task(int(context.args[0]))
    await update.message.reply_text(f"Muted #{task['id']} muted={task.get('muted')}")


async def cmd_unmute(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    if not context.args:
        await update.message.reply_text("Usage: /unmute <task_id>")
        return
    task = _api().unmute_task(int(context.args[0]))
    await update.message.reply_text(f"Unmuted #{task['id']} muted={task.get('muted')}")


async def cmd_backlog(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    linked = await _ensure_link(update, context)
    title = " ".join(context.args).strip() if context.args else ""
    if not title:
        await update.message.reply_text("Usage: /backlog <title>")
        return
    created = _api().create_backlog(
        title=title,
        user_id=linked["user_id"],
        messenger_related_user_id=linked["mru_id"],
    )
    await update.message.reply_text(f"Backlog id={created['id']}")


async def cmd_digest(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    linked = await _ensure_link(update, context)
    digest = _api().get_digest(
        user_id=linked["user_id"],
        messenger_related_user_id=linked["mru_id"],
    )
    tasks = digest.get("tasks") or []
    targets = digest.get("targets") or []
    await update.message.reply_text(
        f"Digest tasks={len(tasks)} targets={len(targets)} "
        f"completed_backlogs={digest.get('completed_backlogs_count', 0)}\n"
        f"range {digest.get('start_date_from')} … {digest.get('start_date_to')}"
    )


def _markup_from_payload(raw: Any) -> InlineKeyboardMarkup | None:
    if not raw or not isinstance(raw, dict):
        return None
    rows = raw.get("inline_keyboard") or []
    keyboard = []
    for row in rows:
        buttons = []
        for btn in row:
            buttons.append(
                InlineKeyboardButton(
                    text=str(btn.get("text") or "?"),
                    callback_data=str(btn.get("callback_data") or ""),
                )
            )
        if buttons:
            keyboard.append(buttons)
    return InlineKeyboardMarkup(keyboard) if keyboard else None


async def _deliver_webhook(data: dict[str, Any]) -> dict[str, Any]:
    if _APP is None:
        raise RuntimeError("bot application not ready")
    chat_id = data.get("chat_id")
    text = data.get("text")
    if chat_id is None or text is None:
        raise ValueError("chat_id and text are required")
    markup = _markup_from_payload(data.get("reply_markup"))
    await _APP.bot.send_message(
        chat_id=int(chat_id) if str(chat_id).lstrip("-").isdigit() else chat_id,
        text=str(text),
        reply_markup=markup,
    )
    return {"status": "ok", "task_id": data.get("task_id")}


class _WebhookHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args) -> None:  # quieter access log
        log.info("webhook %s", fmt % args)

    def do_POST(self) -> None:  # noqa: N802
        if self.path.rstrip("/") != "/send_message":
            self.send_error(404, "not found")
            return
        length = int(self.headers.get("Content-Length") or 0)
        try:
            body = json.loads(self.rfile.read(length).decode("utf-8") or "{}")
        except json.JSONDecodeError:
            self._json(400, {"error": "invalid json"})
            return
        if _LOOP is None:
            self._json(503, {"error": "bot loop not ready"})
            return
        try:
            fut = asyncio.run_coroutine_threadsafe(_deliver_webhook(body), _LOOP)
            result = fut.result(timeout=30)
            self._json(200, result)
        except Exception as exc:
            log.exception("send_message failed")
            self._json(500, {"error": str(exc)})

    def _json(self, code: int, payload: dict) -> None:
        raw = json.dumps(payload).encode("utf-8")
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)


def _start_webhook_server() -> ThreadingHTTPServer:
    host, port = _webhook_bind()
    server = ThreadingHTTPServer((host, port), _WebhookHandler)
    thread = threading.Thread(target=server.serve_forever, name="webhook-http", daemon=True)
    thread.start()
    log.info("webhook server on http://%s:%s/send_message", host, port)
    return server


async def on_callback(update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
    query = update.callback_query
    if query is None or not query.data:
        return
    await query.answer()
    data = query.data
    client = _api()
    try:
        if data.startswith("done:"):
            task_id = int(data.split(":", 1)[1])
            result = client.mark_task_as_done(task_id)
            await query.edit_message_text(
                f"✅ Done #{result.get('id')} {result.get('title')}"
            )
        elif data.startswith("later:"):
            task_id = int(data.split(":", 1)[1])
            later = datetime.now(timezone.utc) + timedelta(hours=1)
            client.update_task(task_id, start_date=later)
            try:
                client.queue_task(task_id, action="schedule", queue_name="celery")
            except HTTPError:
                log.exception("queue after postpone failed task_id=%s", task_id)
            await query.edit_message_text(f"⏰ Postponed #{task_id} → {later.isoformat()}")
        else:
            await query.edit_message_text(f"Unknown action: {data}")
    except Exception as exc:
        log.exception("callback failed")
        await query.message.reply_text(f"Error: {exc}")


async def on_error(update: object, context: ContextTypes.DEFAULT_TYPE) -> None:
    log.exception("handler error", exc_info=context.error)
    if isinstance(update, Update) and update.effective_message:
        err = context.error
        detail = ""
        if isinstance(err, HTTPError) and err.response is not None:
            detail = f"\n{err.response.status_code}: {err.response.text[:500]}"
        await update.effective_message.reply_text(f"Error: {err}{detail}")


async def _post_init(application: Application) -> None:
    global _APP, _LOOP
    _APP = application
    _LOOP = asyncio.get_running_loop()
    application.bot_data["webhook_server"] = _start_webhook_server()


async def _post_shutdown(application: Application) -> None:
    server = application.bot_data.get("webhook_server")
    if server is not None:
        server.shutdown()


def main() -> int:
    token = os.environ.get("TELEGRAM_BOT_TOKEN")
    if not token:
        print("Set TELEGRAM_BOT_TOKEN", file=sys.stderr)
        return 1

    app = (
        Application.builder()
        .token(token)
        .post_init(_post_init)
        .post_shutdown(_post_shutdown)
        .build()
    )
    app.add_handler(CommandHandler("start", cmd_start))
    app.add_handler(CommandHandler("help", cmd_help))
    app.add_handler(CommandHandler("tasks", cmd_tasks))
    app.add_handler(CommandHandler("add", cmd_add))
    app.add_handler(CommandHandler("done", cmd_done))
    app.add_handler(CommandHandler("mute", cmd_mute))
    app.add_handler(CommandHandler("unmute", cmd_unmute))
    app.add_handler(CommandHandler("backlog", cmd_backlog))
    app.add_handler(CommandHandler("digest", cmd_digest))
    app.add_handler(CallbackQueryHandler(on_callback))
    app.add_handler(MessageHandler(filters.COMMAND, cmd_help))
    app.add_error_handler(on_error)

    log.info(
        "starting bot against %s webhook :%s",
        os.environ.get("GOREMINDER_URL", "http://localhost:8080"),
        os.environ.get("WEBHOOK_PORT", "8001"),
    )
    app.run_polling(allowed_updates=Update.ALL_TYPES)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
