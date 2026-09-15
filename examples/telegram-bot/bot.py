"""
Minimal Telegram bot that talks to GoReminder via examples/python-client.

Env:
  TELEGRAM_BOT_TOKEN   required
  GOREMINDER_URL       default http://localhost:8080
  GOREMINDER_MESSENGER default telegram  (messenger.type name in API)

Flow on /start:
  1) resolve or create messenger type
  2) create GoReminder user (or reuse via messenger_user_id lookup)
  3) create messenger-related user (MRU) linking Telegram chat ↔ user
"""

from __future__ import annotations

import logging
import os
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

from requests import HTTPError
from telegram import Update
from telegram.ext import Application, CommandHandler, ContextTypes, MessageHandler, filters

CLIENT_DIR = Path(__file__).resolve().parents[1] / "python-client"
sys.path.insert(0, str(CLIENT_DIR))

from goreminder_client import GoReminderClient  # noqa: E402

logging.basicConfig(
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
    level=logging.INFO,
)
log = logging.getLogger("goreminder-tg-bot")


def _api() -> GoReminderClient:
    return GoReminderClient(os.environ.get("GOREMINDER_URL", "http://localhost:8080"))


def _messenger_name() -> str:
    return os.environ.get("GOREMINDER_MESSENGER", "telegram")


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
    await update.message.reply_text(
        "Linked to GoReminder.\n"
        f"user_id={linked['user_id']} mru_id={linked['mru_id']}\n\n"
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
    # Task must already have messenger_related_user_id or API returns 422.
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


async def on_error(update: object, context: ContextTypes.DEFAULT_TYPE) -> None:
    log.exception("handler error", exc_info=context.error)
    if isinstance(update, Update) and update.effective_message:
        err = context.error
        detail = ""
        if isinstance(err, HTTPError) and err.response is not None:
            detail = f"\n{err.response.status_code}: {err.response.text[:500]}"
        await update.effective_message.reply_text(f"Error: {err}{detail}")


def main() -> int:
    token = os.environ.get("TELEGRAM_BOT_TOKEN")
    if not token:
        print("Set TELEGRAM_BOT_TOKEN", file=sys.stderr)
        return 1

    app = Application.builder().token(token).build()
    app.add_handler(CommandHandler("start", cmd_start))
    app.add_handler(CommandHandler("help", cmd_help))
    app.add_handler(CommandHandler("tasks", cmd_tasks))
    app.add_handler(CommandHandler("add", cmd_add))
    app.add_handler(CommandHandler("done", cmd_done))
    app.add_handler(CommandHandler("mute", cmd_mute))
    app.add_handler(CommandHandler("unmute", cmd_unmute))
    app.add_handler(CommandHandler("backlog", cmd_backlog))
    app.add_handler(CommandHandler("digest", cmd_digest))
    app.add_handler(MessageHandler(filters.COMMAND, cmd_help))
    app.add_error_handler(on_error)

    log.info("starting bot against %s", os.environ.get("GOREMINDER_URL", "http://localhost:8080"))
    app.run_polling(allowed_updates=Update.ALL_TYPES)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
