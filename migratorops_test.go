package pgmigrate_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

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

func TestSetChecksums(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations, err := pgmigrate.Load(migrations.FS)
		assert.Nil(t, err)

		// Apply the migration
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)
		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(applied))

		updates := []pgmigrate.ChecksumUpdate{
			{
				MigrationID: applied[0].ID,
				NewChecksum: "veryfakechecksum",
			},
			{
				MigrationID: applied[1].ID,
				NewChecksum: "anotherfakechecksum",
			},
		}
		assert.NotEqual(t, applied[0].Checksum, updates[0].NewChecksum)
		assert.NotEqual(t, applied[1].Checksum, updates[1].NewChecksum)
		updated, err := migrator.SetChecksums(ctx, db, updates...)
		assert.Nil(t, err)
		assert.Equal(t, 2, len(updated))
		assert.Equal(t, updates[0].NewChecksum, updated[0].Checksum)
		assert.Equal(t, updates[0].MigrationID, updated[0].ID)
		assert.Equal(t, updates[1].NewChecksum, updated[1].Checksum)
		assert.Equal(t, updates[1].MigrationID, updated[1].ID)
		return nil
	})
	assert.Nil(t, err)
}

func TestRecalculateChecksums(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations, err := pgmigrate.Load(migrations.FS)
		assert.Nil(t, err)

		// Apply the migration
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)
		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(applied))

		recalculated, err := migrator.RecalculateChecksums(ctx, db, migrations[0].ID)
		assert.Nil(t, err)
		assert.Equal(t, nil, recalculated)

		updated, err := migrator.SetChecksums(ctx, db, pgmigrate.ChecksumUpdate{
			MigrationID: migrations[0].ID,
			NewChecksum: "somethingfake",
		})
		assert.Nil(t, err)
		assert.Equal(t, 1, len(updated))
		assert.Equal(t, migrations[0].ID, updated[0].ID)

		recalculated, err = migrator.RecalculateChecksums(ctx, db, migrations[0].ID)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(recalculated))
		assert.Equal(t, migrations[0].MD5(), recalculated[0].Checksum)
		return nil
	})
	assert.Nil(t, err)
}

func TestRecalculateAllChecksums(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations, err := pgmigrate.Load(migrations.FS)
		assert.Nil(t, err)

		// Apply the migration
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)
		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(applied))

		recalculated, err := migrator.RecalculateChecksums(ctx, db, migrations[0].ID)
		assert.Nil(t, err)
		assert.Equal(t, nil, recalculated)

		updates := []pgmigrate.ChecksumUpdate{}
		for _, migration := range migrations {
			updates = append(updates, pgmigrate.ChecksumUpdate{
				MigrationID: migration.ID,
				NewChecksum: "fakefakefake!",
			})
		}
		updated, err := migrator.SetChecksums(ctx, db, updates...)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(updated))

		verrs, err = migrator.Verify(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(verrs))
		for i, verr := range verrs {
			check.Equal(t, migrations[i].ID, verr.Fields["migration_id"].(string))
			check.Equal(t, migrations[i].MD5(), verr.Fields["calculated_checksum"].(string))
			check.Equal(t, updates[i].NewChecksum, verr.Fields["migration_checksum_from_db"].(string))
		}

		recalculated, err = migrator.RecalculateAllChecksums(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, 4, len(recalculated))
		for i, recalc := range recalculated {
			check.Equal(t, migrations[i].ID, recalc.ID)
			check.Equal(t, migrations[i].MD5(), recalc.Checksum)
		}
		verrs, err = migrator.Verify(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)
		return nil
	})
	assert.Nil(t, err)
}
