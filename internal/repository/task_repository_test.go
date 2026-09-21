package repository

import (
	"context"
	"database/sql"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
)

func newTaskRepoWithMock(t *testing.T) (TaskRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewTaskRepository(sqlxDB, logger.New(io.Discard, zerolog.Disabled, false))
	cleanup := func() { _ = db.Close() }
	return repo, mock, cleanup
}

func TestTaskRepository_GetTaskByID_Success(t *testing.T) {
	repo, mock, cleanup := newTaskRepoWithMock(t)
	defer cleanup()

	now := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{
		"id", "title", "description", "user_id", "messenger_related_user_id", "parent_id",
		"start_date", "finish_date", "cron_expression", "rrule", "status", "created_at",
		"requires_confirmation", "muted", "pre_remind_before_seconds",
	}).AddRow(
		int64(1), "t", "d", int64(10), nil, nil,
		now, nil, nil, nil, "scheduled", now,
		false, false, nil,
	)

	// Match soft-delete + id + status <> done filters from GetTaskByID.
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, title, description, user_id, messenger_related_user_id, parent_id, start_date, finish_date, cron_expression, rrule, status, created_at, requires_confirmation, muted, pre_remind_before_seconds FROM tasks WHERE deleted_at IS NULL AND id = $1 AND status <> $2`,
	)).WithArgs(int64(1), "done").WillReturnRows(rows)

	task, err := repo.GetTaskByID(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, int64(1), task.ID)
	assert.Equal(t, "t", task.Title)
	assert.Equal(t, "scheduled", task.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_GetTaskByID_NotFound(t *testing.T) {
	repo, mock, cleanup := newTaskRepoWithMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT .+ FROM tasks WHERE deleted_at IS NULL AND id = \$1 AND status <> \$2`).
		WithArgs(int64(99), "done").
		WillReturnError(sql.ErrNoRows)

	task, err := repo.GetTaskByID(context.Background(), 99)
	assert.Nil(t, task)
	assert.ErrorIs(t, err, errs.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_GetAllTasks_AppliesStatusFilterAndPagination(t *testing.T) {
	repo, mock, cleanup := newTaskRepoWithMock(t)
	defer cleanup()

	status := "scheduled"
	now := time.Now().UTC().Truncate(time.Second)

	// Count query with soft-delete + status filter.
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COUNT(*) FROM tasks WHERE deleted_at IS NULL AND status = $1`,
	)).WithArgs(status).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Data query with ORDER BY / LIMIT / OFFSET (page=2, pageSize=10 → offset 10).
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, title, description, user_id, messenger_related_user_id, parent_id, start_date, finish_date, cron_expression, rrule, status, created_at, requires_confirmation, muted, pre_remind_before_seconds FROM tasks WHERE deleted_at IS NULL AND status = $1 ORDER BY created_at DESC LIMIT 10 OFFSET 10`,
	)).WithArgs(status).WillReturnRows(sqlmock.NewRows([]string{
		"id", "title", "description", "user_id", "messenger_related_user_id", "parent_id",
		"start_date", "finish_date", "cron_expression", "rrule", "status", "created_at",
		"requires_confirmation", "muted", "pre_remind_before_seconds",
	}).AddRow(
		int64(11), "paged", "d", int64(1), nil, nil,
		now, nil, nil, nil, status, now,
		false, false, nil,
	))

	tasks, total, err := repo.GetAllTasks(
		context.Background(),
		2, 10, "created_at DESC",
		&status, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, tasks, 1)
	assert.Equal(t, int64(11), tasks[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Ensure squirrel dollar placeholders stay aligned with expectations if builder defaults change.
func TestTaskRepository_GetTaskByID_QueryShape(t *testing.T) {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := sb.Select("id").
		From("tasks").
		Where(squirrel.Eq{"deleted_at": nil}).
		Where(squirrel.Eq{"id": int64(1)}).
		Where(squirrel.NotEq{"status": "done"}).
		ToSql()
	require.NoError(t, err)
	assert.Contains(t, query, "deleted_at IS NULL")
	assert.Contains(t, query, "status <>")
	assert.Equal(t, []interface{}{int64(1), "done"}, args)
}

func TestTaskRepository_GetTaskByIDWithoutStatusFilter_AllowsDone(t *testing.T) {
	repo, mock, cleanup := newTaskRepoWithMock(t)
	defer cleanup()

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, title, description, user_id, messenger_related_user_id, parent_id, start_date, finish_date, cron_expression, rrule, status, created_at, requires_confirmation, muted, pre_remind_before_seconds FROM tasks WHERE deleted_at IS NULL AND id = $1`,
	)).WithArgs(int64(2)).WillReturnRows(sqlmock.NewRows([]string{
		"id", "title", "description", "user_id", "messenger_related_user_id", "parent_id",
		"start_date", "finish_date", "cron_expression", "rrule", "status", "created_at",
		"requires_confirmation", "muted", "pre_remind_before_seconds",
	}).AddRow(
		int64(2), "done-task", "d", int64(1), nil, nil,
		now, &now, nil, nil, "done", now,
		false, false, nil,
	))

	task, err := repo.GetTaskByIDWithoutStatusFilter(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, "done", task.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_CreateTask_ReturnsID(t *testing.T) {
	repo, mock, cleanup := newTaskRepoWithMock(t)
	defer cleanup()

	start := time.Now().UTC().Truncate(time.Second)
	task := &models.Task{
		Title:                "created",
		Description:          "desc",
		UserID:               1,
		Status:               "scheduled",
		StartDate:            start,
		RequiresConfirmation: false,
		Muted:                false,
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO tasks (title,description,user_id,messenger_related_user_id,status,parent_id,start_date,finish_date,cron_expression,rrule,requires_confirmation,muted,pre_remind_before_seconds) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`,
	)).WithArgs(
		task.Title, task.Description, task.UserID, nil, task.Status, nil,
		task.StartDate, nil, nil, nil, false, false, nil,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(77)))

	id, err := repo.CreateTask(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, int64(77), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_DeleteTask_SoftDeletes(t *testing.T) {
	repo, mock, cleanup := newTaskRepoWithMock(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tasks SET deleted_at = $1, status = $2 WHERE deleted_at IS NULL AND id = $3`,
	)).WithArgs(sqlmock.AnyArg(), "deleted", int64(15)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteTask(context.Background(), 15)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepository_GetAllTasks_UserAndDateFilters(t *testing.T) {
	repo, mock, cleanup := newTaskRepoWithMock(t)
	defer cleanup()

	userID := int64(5)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COUNT(*) FROM tasks WHERE deleted_at IS NULL AND start_date >= $1 AND start_date <= $2 AND user_id = $3`,
	)).WithArgs(from, to, userID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, title, description, user_id, messenger_related_user_id, parent_id, start_date, finish_date, cron_expression, rrule, status, created_at, requires_confirmation, muted, pre_remind_before_seconds FROM tasks WHERE deleted_at IS NULL AND start_date >= $1 AND start_date <= $2 AND user_id = $3 ORDER BY created_at DESC LIMIT 50 OFFSET 0`,
	)).WithArgs(from, to, userID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "title", "description", "user_id", "messenger_related_user_id", "parent_id",
		"start_date", "finish_date", "cron_expression", "rrule", "status", "created_at",
		"requires_confirmation", "muted", "pre_remind_before_seconds",
	}))

	tasks, total, err := repo.GetAllTasks(
		context.Background(),
		1, 50, "",
		nil, nil, &from, &to, &userID, nil, nil, nil, nil,
	)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, tasks)
	assert.NoError(t, mock.ExpectationsWereMet())
}
