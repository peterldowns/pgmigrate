package pgmigrate_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/peterldowns/testy/assert"

	"github.com/peterldowns/pgmigrate"
	"github.com/peterldowns/pgmigrate/internal/migrations"
	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestMarkApplied(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations := []pgmigrate.Migration{
			{
				ID:  "0001_initial",
				SQL: "CREATE TABLE users (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY);",
			},
		}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		applied, err := migrator.MarkApplied(ctx, db, "0001_initial")
		assert.Nil(t, err)
		assert.Equal(t, 1, len(applied))
		mig := applied[0]
		assert.Equal(t, migrations[0].ID, mig.ID)
		assert.Equal(t, migrations[0].MD5(), mig.Checksum)
		return nil
	})
	assert.Nil(t, err)
}

func TestMarkAllApplied(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations, err := pgmigrate.Load(migrations.FS)
		assert.Nil(t, err)

		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		applied, err := migrator.MarkAllApplied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(applied))

		plan, err := migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, plan)
		return nil
	})
	assert.Nil(t, err)
}
