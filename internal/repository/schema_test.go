package repository

import (
	"os"
	"testing"

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
				"id", "title", "description", "user_id", "messenger_related_user_id", "due_date", "status", "created_at", "updated_at", "deleted_at",
			},
		},
	}

	// TODO: remove
	t.Setenv("TEST_DATABASE_DSN", "postgres://postgres:password@localhost:5432/beshop?sslmode=disable")

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
