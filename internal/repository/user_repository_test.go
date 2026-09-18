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

func newUserRepoWithMock(t *testing.T) (UserRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewUserRepository(sqlxDB, logger.New(io.Discard, zerolog.Disabled, false))
	cleanup := func() { _ = db.Close() }
	return repo, mock, cleanup
}

func TestUserRepository_CreateUser_ReturnsID(t *testing.T) {
	repo, mock, cleanup := newUserRepoWithMock(t)
	defer cleanup()

	tz := "UTC"
	lang := "en"
	role := "user"
	user := &models.User{
		Name: "Alice", Email: "a@x", PasswordHash: "hash",
		Timezone: &tz, LanguageCode: &lang, Role: &role,
	}

	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO users (name,email,password_hash,timezone,language_code,role) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
	)).WithArgs(user.Name, user.Email, user.PasswordHash, user.Timezone, user.LanguageCode, user.Role).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(31)))

	id, err := repo.CreateUser(context.Background(), user)
	require.NoError(t, err)
	assert.Equal(t, int64(31), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetUserByID_Success(t *testing.T) {
	repo, mock, cleanup := newUserRepoWithMock(t)
	defer cleanup()

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, email, password_hash, created_at, timezone, language_code, role, last_activity_at FROM users WHERE deleted_at IS NULL AND id = $1`,
	)).WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{
		"id", "name", "email", "password_hash", "created_at", "timezone", "language_code", "role", "last_activity_at",
	}).AddRow(int64(1), "Alice", "a@x", "hash", now, nil, nil, nil, nil))

	user, err := repo.GetUserByID(context.Background(), 1)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "Alice", user.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetUserByID_NotFound(t *testing.T) {
	repo, mock, cleanup := newUserRepoWithMock(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT .+ FROM users WHERE deleted_at IS NULL AND id = \$1`).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetUserByID(context.Background(), 99)
	assert.Nil(t, user)
	assert.ErrorIs(t, err, errs.ErrNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_DeleteUser_SoftDeletes(t *testing.T) {
	repo, mock, cleanup := newUserRepoWithMock(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE users SET deleted_at = $1 WHERE deleted_at IS NULL AND id = $2`,
	)).WithArgs(sqlmock.AnyArg(), int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteUser(context.Background(), 5)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetAllUsers_Pagination(t *testing.T) {
	repo, mock, cleanup := newUserRepoWithMock(t)
	defer cleanup()

	now := time.Now().UTC().Truncate(time.Second)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`,
	)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, email, password_hash, created_at, timezone, language_code, role, last_activity_at FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 10 OFFSET 10`,
	)).WillReturnRows(sqlmock.NewRows([]string{
		"id", "name", "email", "password_hash", "created_at", "timezone", "language_code", "role", "last_activity_at",
	}).AddRow(int64(11), "Bob", "b@x", "h", now, nil, nil, nil, nil))

	users, total, err := repo.GetAllUsers(context.Background(), 2, 10, "created_at DESC")
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, users, 1)
	assert.Equal(t, int64(11), users[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}


func TestUserRepository_TouchLastActivity(t *testing.T) {
	repo, mock, cleanup := newUserRepoWithMock(t)
	defer cleanup()

	at := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE users SET last_activity_at = GREATEST(COALESCE(last_activity_at, '-infinity'::timestamptz), $1::timestamptz) WHERE id = $2 AND deleted_at IS NULL`,
	)).WithArgs(at, int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.TouchLastActivity(context.Background(), 7, at)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ListRecentActivity(t *testing.T) {
	repo, mock, cleanup := newUserRepoWithMock(t)
	defer cleanup()

	at := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, last_activity_at FROM users WHERE deleted_at IS NULL AND last_activity_at IS NOT NULL ORDER BY last_activity_at DESC LIMIT 2`,
	)).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "last_activity_at"}).
		AddRow(int64(1), "Alice", at))

	activities, err := repo.ListRecentActivity(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Equal(t, int64(1), activities[0].UserID)
	assert.Equal(t, "Alice", activities[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}
