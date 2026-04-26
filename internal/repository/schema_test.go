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

// schemaTestDBReady returns true when Postgres looks fully migrated for this test suite.
func schemaTestDBReady(db *sqlx.DB) bool {
	var n int
	if err := db.Get(&n, `
		SELECT count(*)::int FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'tasks' AND column_name = 'rrule'`); err != nil || n == 0 {
		return false
	}
	if err := db.Get(&n, `
		SELECT count(*)::int FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'backlogs'`); err != nil || n == 0 {
		return false
	}
	return true
}

func TestTableSchemasMatchModels(t *testing.T) {
	testCases := []tableSchemaTestCase{
		{
			tableName: "tasks",
			expectedColumns: []string{
				"id", "title", "description", "user_id", "messenger_related_user_id",
				"parent_id", "start_date", "finish_date", "cron_expression", "rrule",
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
		{
			tableName: "targets",
			expectedColumns: []string{
				"id", "title", "description", "user_id", "messenger_related_user_id",
				"created_at", "updated_at", "completed_at", "deleted_at",
			},
		},
	}

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:5432/task_manager?sslmode=disable"
	}

	db, err := sqlx.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("postgres not available for schema integration test (TEST_DATABASE_DSN): %v", err)
	}

	if !schemaTestDBReady(db) {
		t.Skip("Postgres is reachable but schema is not fully migrated (need public.tasks.rrule and public.backlogs); run goose migrations or set TEST_DATABASE_DSN to a migrated database")
	}

	for _, tc := range testCases {
		t.Run(tc.tableName, func(t *testing.T) {
			rows, err := db.Query(`
                SELECT column_name
                FROM information_schema.columns
                WHERE table_schema = 'public' AND table_name = $1
            `, tc.tableName)
			require.NoError(t, err)
			defer rows.Close()

			var dbColumns []string
			for rows.Next() {
				var col string
				require.NoError(t, rows.Scan(&col))
				dbColumns = append(dbColumns, col)
			}

			if len(dbColumns) == 0 {
				t.Fatalf("table %q not found or has no columns in schema public", tc.tableName)
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
