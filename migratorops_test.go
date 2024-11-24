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
		assert.Equal(t, migrations[0], mig.Migration)
		assert.Equal(t, migrations[0].MD5(), mig.Checksum)

		plan, err := migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, plan)
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

func TestMarkUnapplied(t *testing.T) {
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

		// Start with all migrations applied, empty plan.
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)
		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(applied))

		plan, err := migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, plan)

		// Unapply the migration and check that it becomes present in the plan
		unapplied, err := migrator.MarkUnapplied(ctx, db, migrations[0].ID)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(unapplied))
		assert.Equal(t, migrations[0].ID, unapplied[0].ID)

		plan, err = migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(plan))
		assert.Equal(t, migrations[0], plan[0])
		return nil
	})
	assert.Nil(t, err)
}

func TestMarkAllUnapplied(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations, err := pgmigrate.Load(migrations.FS)
		assert.Nil(t, err)

		// Start with all migrations applied, empty plan.
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)
		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(applied))

		plan, err := migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, plan)

		// Unapply the migrations and check that they become present in the plan
		unapplied, err := migrator.MarkAllUnapplied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(unapplied))

		plan, err = migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, migrations, plan)
		return nil
	})
	assert.Nil(t, err)
}
