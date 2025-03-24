package withdb_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for postgres
	"github.com/peterldowns/testy/assert"

	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestWithDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		_, err := db.Exec("select 1")
		return err
	})
	assert.Nil(t, err)
}
