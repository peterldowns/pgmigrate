package pgmigrate_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate"

	"github.com/peterldowns/pgmigrate/internal/schema"
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

// By default, pgmigrate will use the [DefaultTableName] table to
// keep track of migrations. Because this is a fully qualified table name,
// including a schema prefix, pgmigrate will not be affected by migrations
// that change the search_path of the current database connection when they're
// executed.
func TestSettingSearchPathInMigrationsDoesntBreak(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := pgmigrate.NewTestLogger(t)
	m1 := pgmigrate.Migration{
		ID: "01_devices",
		SQL: `--sql
			CREATE SCHEMA IF NOT EXISTS another_schema;
			SET search_path TO another_schema;

			CREATE TABLE IF NOT EXISTS devices (
			id uuid NOT NULL PRIMARY KEY,
			state jsonb NOT NULL
			);
		`,
	}
	m2 := pgmigrate.Migration{
		ID: "02_goals",
		SQL: `--sql
			-- This line relies on migration 01_devices having been executed,
			-- since "another_schema" will not exist otherwise.
			SET search_path TO another_schema;

			CREATE TABLE IF NOT EXISTS goals (
				device_id uuid PRIMARY KEY NOT NULL REFERENCES devices (id) ON DELETE CASCADE,
				goal jsonb
			);
		`,
	}
	m3 := pgmigrate.Migration{
		ID: "03_ambiguous",
		SQL: `--sql
			-- This ambiguous table name will be resolved using the current value of
			-- search_path. By default, the search_path is "default" but can be overridden
			-- by the connection string that is used to connect to the database, or changed
			-- by other SQL commands run on the same connection.
			--
			-- If this migration is applied as part of the same plan as the migration m2/02_goals,
			-- this will create the table "another_schema"."users".
			--
			-- If this migration is applied in a second session, for instance if the earlier
			-- migrations had already been shipped to the users and this was written and shipped
			-- as an update, then it will create the table "public"."users".
			--
			-- If you're setting the search_path inside of your migrations, you should make
			-- sure to be aware of this possibility, and consistently set/reset the search
			-- path.
			CREATE TABLE users (
				id uuid NOT NULL PRIMARY KEY,
				name text NOT NULL default ''
			);
		`,
	}
	t.Run("all_migrations_in_one_session", func(t *testing.T) {
		t.Parallel()
		err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
			migrations := []pgmigrate.Migration{m1, m2, m3}
			migrator := pgmigrate.NewMigrator(migrations)
			assert.Equal(t, pgmigrate.DefaultTableName, migrator.TableName)
			migrator.Logger = logger

			// Check to confirm that 0 migrations have been applied.
			applied, err := migrator.Applied(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, nil, applied)

			// Check to confirm that the migrations should be applied.
			plan, err := migrator.Plan(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, len(plan), 3)
			assert.Equal(t, migrations, plan)

			// Apply the migrations.
			verrs, err := migrator.Migrate(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, nil, verrs)

			// Check to make sure that the migrations have been applied.
			applied, err = migrator.Applied(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, 3, len(applied))
			assert.Equal(t, migrations[0].ID, applied[0].ID)
			assert.Equal(t, migrations[1].ID, applied[1].ID)
			assert.Equal(t, migrations[2].ID, applied[2].ID)

			// Check to make sure that the migrations will not be applied again.
			plan, err = migrator.Plan(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, len(plan), 0)

			// The [DefaultTableName] table was created correctly in the public schema.
			publicTables, err := schema.LoadTables(schema.Config{
				Schemas: []string{"public"},
			}, db)
			assert.Nil(t, err)
			check.Equal(t, 1, len(publicTables))
			check.Equal(t, "pgmigrate_migrations", publicTables[0].Name)

			// Because all three migrations were applied over the same
			// connection, when m2 modified the search_path, the m3 migration
			// was applied in that context, so the users table ended up in
			// "another_schema".
			otherTables, err := schema.LoadTables(schema.Config{
				Schemas: []string{"another_schema"},
			}, db)
			assert.Nil(t, err)
			check.Equal(t, 3, len(otherTables))
			check.Equal(t, "devices", otherTables[0].Name)
			check.Equal(t, "goals", otherTables[1].Name)
			check.Equal(t, "users", otherTables[2].Name)

			return nil
		})
		assert.Nil(t, err)
	})
	t.Run("first_two_migrations_then_the_third_separately", func(t *testing.T) {
		t.Parallel()
		err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
			// Only m1 and m2, NOT m3, to start.
			migrations := []pgmigrate.Migration{m1, m2}
			migrator := pgmigrate.NewMigrator(migrations)
			assert.Equal(t, pgmigrate.DefaultTableName, migrator.TableName)
			migrator.Logger = logger

			// Check to confirm that 0 migrations have been applied.
			applied, err := migrator.Applied(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, nil, applied)

			// Check to confirm that the migrations should be applied.
			plan, err := migrator.Plan(ctx, db)
			assert.Nil(t, err)
			check.Equal(t, migrations, plan)

			// Apply the migrations.
			verrs, err := migrator.Migrate(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, nil, verrs)

			// Check to make sure that the migrations have been applied.
			applied, err = migrator.Applied(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(applied))
			check.Equal(t, migrations[0].ID, applied[0].ID)
			check.Equal(t, migrations[1].ID, applied[1].ID)

			// Check to make sure that the migrations will not be applied again.
			plan, err = migrator.Plan(ctx, db)
			assert.Nil(t, err)
			check.Equal(t, len(plan), 0)

			// The [DefaultTableName] table was created correctly in the public schema.
			publicTables, err := schema.LoadTables(schema.Config{
				Schemas: []string{"public"},
			}, db)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(publicTables))
			check.Equal(t, "pgmigrate_migrations", publicTables[0].Name)

			// m1 and m2 were applied correctly and created their tables in "another_schema".
			otherTables, err := schema.LoadTables(schema.Config{
				Schemas: []string{"another_schema"},
			}, db)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(otherTables))
			check.Equal(t, "devices", otherTables[0].Name)
			check.Equal(t, "goals", otherTables[1].Name)

			// Now, add the third migration.
			migrations = []pgmigrate.Migration{m1, m2, m3}
			migrator.Migrations = migrations

			// Reset the search_state to public, as if we had opened a new connection to the
			// database --- for instance, if we were applying this migration a few weeks
			// after the first two had been applied.
			//
			// This is an implementation detail of this test: the *sql.DB will
			// re-use the same connection from the earlier steps, which means
			// that the search_path will still be "another_schema".
			var searchpath string
			err = db.QueryRowContext(ctx, `SHOW search_path`).Scan(&searchpath)
			assert.Nil(t, err)
			assert.Equal(t, "another_schema", searchpath)
			_, err = db.ExecContext(ctx, "SET search_path TO DEFAULT")
			assert.Nil(t, err)
			err = db.QueryRowContext(ctx, `SHOW search_path`).Scan(&searchpath)
			assert.Nil(t, err)
			assert.Equal(t, `"$user", public`, searchpath)

			// Check to confirm that just the third migration should be applied.
			plan, err = migrator.Plan(ctx, db)
			assert.Nil(t, err)
			check.Equal(t, []pgmigrate.Migration{m3}, plan)

			// Apply the third migration.
			verrs, err = migrator.Migrate(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, nil, verrs)

			// Check to make sure that all 3 migrations have been applied.
			applied, err = migrator.Applied(ctx, db)
			assert.Nil(t, err)
			assert.Equal(t, 3, len(applied))
			check.Equal(t, migrations[0].ID, applied[0].ID)
			check.Equal(t, migrations[1].ID, applied[1].ID)
			check.Equal(t, migrations[2].ID, applied[2].ID)

			// Because m3 was applied by itself, not on the same connection that
			// had previously executed m1 and m2, it was executed while the search_path
			// was still set to the default. This means that it resulted in the table "public"."users",
			// NOT "another_schema"."users", as in the previous scenario.
			publicTables, err = schema.LoadTables(schema.Config{
				Schemas: []string{"public"},
			}, db)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(publicTables))
			check.Equal(t, "pgmigrate_migrations", publicTables[0].Name)
			check.Equal(t, "users", publicTables[1].Name)

			// Nothing has changed in "another_schema", it still has the tables
			// created by m1 and m2.
			otherTables, err = schema.LoadTables(schema.Config{
				Schemas: []string{"another_schema"},
			}, db)
			assert.Nil(t, err)
			assert.Equal(t, 2, len(otherTables))
			check.Equal(t, "devices", otherTables[0].Name)
			check.Equal(t, "goals", otherTables[1].Name)
			return nil
		})
		assert.Nil(t, err)
	})
}
