package repository

import (
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // Import PGX driver for sqlx
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

type tableSchemaTestCase struct {
	tableName       string
	expectedColumns []string
}

func TestTableSchemasMatchModels(t *testing.T) {
	testCases := []tableSchemaTestCase{
		{
			tableName: "tasks",
			expectedColumns: []string{
				"id", "title", "description", "user_id", "messenger_related_user_id",
				"parent_id", "start_date", "finish_date", "cron_expression",
				"requires_confirmation", "status", "created_at", "updated_at", "deleted_at",
			},
		},
		{
			tableName: "users",
			expectedColumns: []string{
				"id", "name", "email", "password_hash", "timezone", "language_code", "role",
				"created_at", "updated_at", "deleted_at",
			},
		},
		{
			tableName: "messengers",
			expectedColumns: []string{
				"id", "name", "created_at",
			},
		},
		{
			tableName: "user_messengers",
			expectedColumns: []string{
				"id", "messenger_id", "user_id", "chat_id", "messenger_user_id",
				"created_at", "updated_at", "deleted_at",
			},
		},
		{
			tableName: "backlogs",
			expectedColumns: []string{
				"id", "title", "description", "user_id", "messenger_related_user_id",
				"created_at", "updated_at", "completed_at", "deleted_at",
			},
		},
		{
			tableName: "digest_settings",
			expectedColumns: []string{
				"id", "user_id", "messenger_related_user_id", "enabled",
				"weekday_time", "weekend_time", "created_at", "updated_at",
			},
		},
		{
			tableName: "task_history",
			expectedColumns: []string{
				"id", "task_id", "user_id", "action", "old_value", "new_value", "created_at",
			},
		},
	}

	t.Setenv("TEST_DATABASE_DSN", "postgres://postgres:password@localhost:5432/task_manager?sslmode=disable")

	dsn := os.Getenv("TEST_DATABASE_DSN")
	db, err := sqlx.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	for _, tc := range testCases {
		t.Run(tc.tableName, func(t *testing.T) {
			rows, err := db.Query(`
                SELECT column_name
                FROM information_schema.columns
                WHERE table_name = $1
            `, tc.tableName)
			require.NoError(t, err)
			defer rows.Close()

			var dbColumns []string
			for rows.Next() {
				var col string
				require.NoError(t, rows.Scan(&col))
				dbColumns = append(dbColumns, col)
			}

			for _, dbCol := range dbColumns {
				found := false
				for _, expCol := range tc.expectedColumns {
					if dbCol == expCol {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected column in DB: %s", dbCol)
				}
			}

			for _, expCol := range tc.expectedColumns {
				found := false
				for _, dbCol := range dbColumns {
					if dbCol == expCol {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected column missing in DB: %s", expCol)
				}
			}
		})
	}
}
