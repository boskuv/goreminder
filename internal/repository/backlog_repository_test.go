package repository

import (
	"context"
	"database/sql"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/boskuv/goreminder/internal/errors"
	"github.com/boskuv/goreminder/internal/models"
	"github.com/boskuv/goreminder/pkg/logger"
)

func newBacklogRepoWithMock(t *testing.T) (BacklogRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewBacklogRepository(sqlxDB, logger.New(io.Discard, zerolog.Disabled, false))
	cleanup := func() { _ = db.Close() }
	return repo, mock, cleanup
}

func TestBacklogRepository_CreateBacklog_ReturnsID(t *testing.T) {
	repo, mock, cleanup := newBacklogRepoWithMock(t)
	defer cleanup()

	backlog := &models.Backlog{Title: "Buy milk", Description: "2%", UserID: 7}

	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO backlogs (title,description,user_id,messenger_related_user_id) VALUES ($1,$2,$3,$4) RETURNING id`,
	)).WithArgs(backlog.Title, backlog.Description, backlog.UserID, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))

	id, err := repo.CreateBacklog(context.Background(), backlog)
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBacklogRepository_GetBacklogByID_Success(t *testing.T) {
	repo, mock, cleanup := newBacklogRepoWithMock(t)
	defer cleanup()

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, title, description, user_id, messenger_related_user_id, created_at, updated_at, completed_at FROM backlogs WHERE deleted_at IS NULL AND id = $1`,
	)).WithArgs(int64(5)).WillReturnRows(sqlmock.NewRows([]string{
		"id", "title", "description", "user_id", "messenger_related_user_id", "created_at", "updated_at", "completed_at",
	}).AddRow(int64(5), "Item", "d", int64(1), nil, now, now, nil))

	backlog, err := repo.GetBacklogByID(context.Background(), 5)
	require.NoError(t, err)
	require.NotNil(t, backlog)
	assert.Equal(t, "Item", backlog.Title)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBacklogRepository_GetBacklogByID_NotFound(t *testing.T) {
	repo, mock, cleanup := newBacklogRepoWithMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT .+ FROM backlogs WHERE deleted_at IS NULL AND id = \$1`).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	backlog, err := repo.GetBacklogByID(context.Background(), 99)
	assert.Nil(t, backlog)
	assert.ErrorIs(t, err, errs.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBacklogRepository_GetAllBacklogs_UserAndCompletedFilters(t *testing.T) {
	repo, mock, cleanup := newBacklogRepoWithMock(t)
	defer cleanup()

	userID := int64(3)
	completed := false
	now := time.Now().UTC().Truncate(time.Second)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COUNT(*) FROM backlogs WHERE deleted_at IS NULL AND user_id = $1 AND completed_at IS NULL`,
	)).WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, title, description, user_id, messenger_related_user_id, created_at, updated_at, completed_at FROM backlogs WHERE deleted_at IS NULL AND user_id = $1 AND completed_at IS NULL ORDER BY created_at DESC LIMIT 50 OFFSET 0`,
	)).WithArgs(userID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "title", "description", "user_id", "messenger_related_user_id", "created_at", "updated_at", "completed_at",
	}).AddRow(int64(1), "A", "", userID, nil, now, now, nil))

	backlogs, total, err := repo.GetAllBacklogs(context.Background(), 1, 50, "created_at DESC", &userID, &completed, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, backlogs, 1)
	assert.Equal(t, int64(1), backlogs[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBacklogRepository_DeleteBacklog_SoftDeletes(t *testing.T) {
	repo, mock, cleanup := newBacklogRepoWithMock(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE backlogs SET deleted_at = $1 WHERE deleted_at IS NULL AND id = $2`,
	)).WithArgs(sqlmock.AnyArg(), int64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteBacklog(context.Background(), 4)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBacklogRepository_DeleteBacklog_NotFound(t *testing.T) {
	repo, mock, cleanup := newBacklogRepoWithMock(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE backlogs SET deleted_at = $1 WHERE deleted_at IS NULL AND id = $2`,
	)).WithArgs(sqlmock.AnyArg(), int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteBacklog(context.Background(), 99)
	assert.ErrorIs(t, err, errs.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBacklogRepository_GetCompletedBacklogsCount(t *testing.T) {
	repo, mock, cleanup := newBacklogRepoWithMock(t)
	defer cleanup()

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COUNT(*) FROM backlogs WHERE deleted_at IS NULL AND user_id = $1 AND completed_at IS NOT NULL AND completed_at >= $2 AND completed_at <= $3`,
	)).WithArgs(int64(7), from, to).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.GetCompletedBacklogsCount(context.Background(), 7, from, to)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}
