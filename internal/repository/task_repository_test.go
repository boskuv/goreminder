package repository_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/boskuv/goreminder/internal/repository"
)

func TestGetTaskByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	taskRepo := repository.NewTaskRepository(sqlxDB)

	// Setup mock expectations
	dueDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT title, description, user_id, due_date, status, created_at FROM tasks WHERE deleted_at IS NULL AND id = \$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{
			"title", "description", "user_id", "due_date", "status", "created_at"}).
			AddRow("Test Task", "Description", 1, dueDate, "open", time.Now()))

	// Call the function
	task, err := taskRepo.GetTaskByID(1)

	// Assert the results
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, "Description", task.Description)
	assert.EqualValues(t, 1, task.UserID)
	assert.Equal(t, "open", task.Status)

	// Verify that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
