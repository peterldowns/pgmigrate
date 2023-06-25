package pgmigrate_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate"

	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestApplyNoMigrationsSucceeds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations := []pgmigrate.Migration{}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		check.Nil(t, err)
		check.Equal(t, nil, verrs)
		return nil
	})
	assert.Nil(t, err)
}

func TestApplyOneMigrationSucceeds(t *testing.T) {
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
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)

		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, len(applied), 1)
		check.Equal(t, migrations[0].ID, applied[0].ID)
		check.Equal(t, migrations[0].MD5(), applied[0].Checksum)
		return nil
	})
	assert.Nil(t, err)
}

func TestApplySameMigrationTwiceSucceeds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		m := pgmigrate.Migration{
			ID:  "0001_initial",
			SQL: "CREATE TABLE users (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY);",
		}
		migrations := []pgmigrate.Migration{m}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)

		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, len(applied), 1)
		check.Equal(t, applied[0].ID, m.ID)
		check.Equal(t, applied[0].Checksum, m.MD5())

		// Running apply again with the same migrations should succeed without
		// any errors and without attempting to re-apply the migration.
		verrs, err = migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)
		applied, err = migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, len(applied), 1)
		check.Equal(t, applied[0].ID, m.ID)
		check.Equal(t, applied[0].Checksum, m.MD5())
		return nil
	})
	assert.Nil(t, err)
}

func TestApplyMultipleSucceedsInCorrectOrder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		migrations := []pgmigrate.Migration{
			{ // Depends on 0003_houses
				ID:  "0004_users",
				SQL: "CREATE TABLE users (name text, house_id int references houses (id));",
			},
			{
				ID:  "0002_dogs",
				SQL: "CREATE TABLE dogs (id int primary key, furry bool);",
			},
			{
				// Dpeends on 0002_dogs
				ID:  "0003_cats",
				SQL: "CREATE TABLE cats (id int primary key, enemy_id int references dogs(id));",
			},
			{ // Depends on 0003_cats
				ID:  "0003_houses",
				SQL: "CREATE TABLE houses (id int primary key, cat_id int references cats (id));",
			},
		}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		// The computed plan should sort ascending by ID
		plan, err := migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, len(plan), 4)
		assert.Equal(t, "0002_dogs", plan[0].ID)
		assert.Equal(t, "0003_cats", plan[1].ID)
		assert.Equal(t, "0003_houses", plan[2].ID)
		assert.Equal(t, "0004_users", plan[3].ID)

		// Applying should happen in the same order as the plan.
		verrs, err := migrator.Migrate(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, verrs)

		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, len(applied), 4)
		return nil
	})
	assert.Nil(t, err)
}

func TestApplyFailsWithConflictingIDs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		m1 := pgmigrate.Migration{
			ID:  "0001_initial",
			SQL: "CREATE TABLE users (name text);",
		}
		// Because this migration re-uses the earlier migration's ID, it will fail to apply
		m2 := pgmigrate.Migration{
			ID:  "0001_initial",
			SQL: "CREATE TABLE money (amount bigint);",
		}
		// Because m2 fails to be applied, this migration (which would succeed) is not applied
		m3 := pgmigrate.Migration{
			ID:  "0002_something_else",
			SQL: "CREATE TABLE dogs (furry bool);",
		}
		migrations := []pgmigrate.Migration{m1, m2, m3}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		check.Error(t, err)
		check.Equal(t, nil, verrs)

		applied, err := migrator.Applied(ctx, db)
		check.Nil(t, err)
		check.Equal(t, len(applied), 1)
		check.Equal(t, applied[0].ID, m1.ID)
		check.Equal(t, applied[0].Checksum, m1.MD5())
		return nil
	})
	assert.Nil(t, err)
}

func TestApplyFailsWithInvalidSQL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		m1 := pgmigrate.Migration{
			ID:  "0001_initial",
			SQL: "this is definitely not valid sql!!!",
		}
		// Because the first migration failed, this will not be applied
		m2 := pgmigrate.Migration{
			ID:  "0002_money",
			SQL: "CREATE TABLE money (amount bigint);",
		}
		migrations := []pgmigrate.Migration{m1, m2}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		check.Error(t, err)
		check.Equal(t, nil, verrs)

		applied, err := migrator.Applied(ctx, db)
		check.Nil(t, err)
		check.Equal(t, 0, len(applied))
		return nil
	})
	assert.Nil(t, err)
}

func TestVerifyMD5Mismatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		m1 := pgmigrate.Migration{
			ID:  "0001_initial",
			SQL: "CREATE TABLE users (name text);",
		}
		migrations := []pgmigrate.Migration{m1}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		check.Nil(t, err)
		check.Equal(t, nil, verrs)

		applied, err := migrator.Applied(ctx, db)
		check.Nil(t, err)
		check.Equal(t, len(applied), 1)
		check.Equal(t, applied[0].ID, m1.ID)
		check.Equal(t, applied[0].Checksum, m1.MD5())

		// With the same ID, but different query content, the MD5 will differ
		// and we should get a warning.
		m1modified := m1
		m1modified.SQL = "CREATE TABLE dogs (furry bool);"
		migrator = pgmigrate.NewMigrator([]pgmigrate.Migration{m1modified})
		migrator.Logger = logger
		verrs, err = migrator.Migrate(ctx, db)
		check.Nil(t, err)
		check.Equal(t, len(verrs), 1)
		verr := verrs[0]
		check.Equal(t, verr.Message, "found applied migration with a different checksum")
		check.Equal(t, m1modified.MD5(), verr.Fields["calculated_checksum"].(string))
		check.Equal(t, m1.MD5(), verr.Fields["migration_checksum_from_db"].(string))
		return nil
	})
	assert.Nil(t, err)
}

func TestVerifyMissingMigration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		m1 := pgmigrate.Migration{
			ID:  "0001_initial",
			SQL: "CREATE TABLE users (name text);",
		}
		migrations := []pgmigrate.Migration{m1}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		verrs, err := migrator.Migrate(ctx, db)
		check.Nil(t, err)
		check.Equal(t, nil, verrs)

		applied, err := migrator.Applied(ctx, db)
		check.Nil(t, err)
		check.Equal(t, len(applied), 1)
		check.Equal(t, applied[0].ID, m1.ID)
		check.Equal(t, applied[0].Checksum, m1.MD5())

		migrator = pgmigrate.NewMigrator(nil)
		migrator.Logger = logger
		verrs, err = migrator.Migrate(ctx, db)
		check.Nil(t, err)
		check.Equal(t, len(verrs), 1)
		verr := verrs[0]
		check.Equal(t, verr.Message, "found applied migration not present on disk")
		check.Equal(t, m1.ID, verr.Fields["migration_id"].(string))
		check.Equal(t, m1.MD5(), verr.Fields["migration_checksum"].(string))
		return nil
	})
	assert.Nil(t, err)
}

func TestAppliedAndPlanWithoutMigrationsTable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	// Starting from an empty database, Applied() and Plan() should work without
	// issues and act as if no migrations had previously been applied.
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		m1 := pgmigrate.Migration{
			ID:  "0001_initial",
			SQL: "CREATE TABLE users (name text);",
		}
		migrations := []pgmigrate.Migration{m1}
		migrator := pgmigrate.NewMigrator(migrations)
		migrator.Logger = logger
		applied, err := migrator.Applied(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, nil, applied)
		plan, err := migrator.Plan(ctx, db)
		assert.Nil(t, err)
		assert.Equal(t, len(plan), 1)
		assert.Equal(t, m1, plan[0])
		return nil
	})
	assert.Nil(t, err)
}
