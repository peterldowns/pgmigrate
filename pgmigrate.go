package pgmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
)

// Load walks a filesystem from its root and extracts all files ending in `.sql`
// as Migrations, with the filename (without extension) being the ID and the
// file's contents being the SQL.
//
// From disk:
//
//		// the migration files will be read at run time
//	    fs := os.DirFS("./path/to/migrations/directory/*.sql")
//
// From an embedded fs:
//
//		// the migration files will be embedded at compile time
//	    //go:embed path/to/migrations/directory/*.sql
//		var fs embed.FS
//
// Load returns the migrations in sorted order.
func Load(filesystem fs.FS) ([]Migration, error) {
	var migrations []Migration
	if err := fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".sql") {
			return nil
		}
		migration := Migration{
			ID: IDFromFilename(d.Name()),
		}
		data, err := fs.ReadFile(filesystem, path)
		if err != nil {
			return err
		}
		migration.SQL = string(data)
		migrations = append(migrations, migration)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	SortByID(migrations)
	return migrations, nil
}

// Migrate will apply any previously applied migrations. It stores metadata in the
// [DefaultTableName] table, with the following schema:
// - id: text not null
// - checksum: text not null
// - execution_time_in_millis: integer not null
// - applied_at: timestamp with time zone not null
//
// It does the following things:
//
// First, acquire an advisory lock to prevent conflicts with other instances
// that may be running in parallel. This way only one migrator will attempt to
// run the migrations at any point in time.
//
// Then, calculate a plan of migrations to apply. The plan will be a list of
// migrations that have not yet been marked as applied in the migrations table.
// The migrations in the plan will be ordered by their IDs, in ascending
// lexicographical order.
//
// For each migration in the plan,
//
//   - Begin a transaction
//   - Run the migration
//   - Create a record in the migrations table saying that the migration was applied
//   - Commit the transaction
//
// If a migration fails at any point, the transaction will roll back. A failed
// migration results in NO record for that migration in the migrations table,
// which means that future attempts to run the migrations will include it in
// their plan.
//
// Migrate() will immediately return the error related to a failed migration,
// and will NOT attempt to run any further migrations. Any migrations applied
// before the failure will remain applied. Any migrations not yet applied will
// not be attempted.
//
// If all the migrations in the plan are applied successfully, then call Verify()
// to double-check that all known migrations have been marked as applied in the
// migrations table.
//
// Finally, the advisory lock is released.
func Migrate(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]VerificationError, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.Migrate(ctx, db)
}

// Verify returns a list of [VerificationError]s with warnings for any migrations that:
//
//   - Are marked as applied in the database table but do not exist in the
//     migrations directory.
//   - Have a different checksum in the database than the current file hash.
//
// These warnings usually signify that the schema described by the migrations no longer
// matches the schema in the database. Usually the cause is removing/editing a migration
// without realizing that it was already applied to a database.
//
// The most common cause of a warning is in the case that a new
// release/deployment contains migrations, the migrations are applied
// successfully, but the release is then rolled back due to other issues.  In
// this case the warning is just that, a warning, and should not be a long-term
// problem.
//
// These warnings should not prevent your application from starting, but are
// worth showing to a human devops/db-admin/sre-type person for them to
// investigate.
func Verify(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]VerificationError, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.Verify(ctx, db)
}

// Plan shows which migrations, if any, would be applied, in the order that they
// would be applied in.
//
// The plan will be a list of [Migration]s that are present in the migrations
// directory that have not yet been marked as applied in the migrations table.
//
// The migrations in the plan will be ordered by their IDs, in ascending
// lexicographical order. This is the same order that you see if you use "ls".
// This is also the same order that they will be applied in.
//
// The ID of a migration is its filename without the ".sql" suffix.
//
// A migration will only ever be applied once. Editing the contents of the
// migration file will NOT result in it being re-applied. Instead, you will see a
// verification error warning that the contents of the migration differ from its
// contents when it was previously applied.
//
// Migrations can be applied "out of order". For instance, if there were three
// migrations that had been applied:
//
//   - 001_initial
//   - 002_create_users
//   - 003_create_viewers
//
// And a new migration "002_create_companies" is merged:
//
//   - 001_initial
//   - 002_create_companies
//   - 002_create_users
//   - 003_create_viewers
//
// Running "pgmigrate plan" will show:
//
//   - 002_create_companies
//
// Because the other migrations have already been applied. This is by design; most
// of the time, when you're working with your coworkers, you will not write
// migrations that conflict with each other. As long as you use a migration
// name/number higher than that of any dependencies, you will not have any
// problems.
func Plan(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]Migration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.Plan(ctx, db)
}

// Applied returns a list of [AppliedMigration]s in the order that they were
// applied in (applied_at ASC, id ASC).
//
// If there are no applied migrations, or the specified table does not exist,
// this will return an empty list without an error.
func Applied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.Applied(ctx, db)
}

// MarkApplied (⚠️ danger) is a manual operation that marks specific migrations
// as applied without running them.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s that have been marked as
// applied.
func MarkApplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, ids ...string) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkApplied(ctx, db, ids...)
}

// MarkAllApplied (⚠️ danger) is a manual operation that marks all known migrations as
// applied without running them.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s that have been marked as
// applied.
func MarkAllApplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkAllApplied(ctx, db)
}

// MarkUnapplied (⚠️ danger) is a manual operation that marks specific migrations as
// unapplied (not having been run) by removing their records from the migrations
// table.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s that have been marked as
// unapplied.
func MarkUnapplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, ids ...string) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkUnapplied(ctx, db, ids...)
}

// MarkAllUnapplied (⚠️ danger) is a manual operation that marks all known migrations as
// unapplied (not having been run) by removing their records from the migrations
// table.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s that have been marked as
// unapplied.
func MarkAllUnapplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkAllUnapplied(ctx, db)
}

// SetChecksums (⚠️ danger) is a manual operation that explicitly sets the recorded
// checksum of applied migrations in the migrations table.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s whose checksums have been
// updated.
func SetChecksums(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, updates ...ChecksumUpdate) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.SetChecksums(ctx, db, updates...)
}

// RecalculateChecksums (⚠️ danger) is a manual operation that explicitly
// recalculates the checksums of the specified migrations and updates their
// records in the migrations table to have the calculated checksum.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s whose checksums have been
// recalculated.
func RecalculateChecksums(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, ids ...string) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.RecalculateChecksums(ctx, db, ids...)
}

// RecalculateChecksums (⚠️ danger) is a manual operation that explicitly
// recalculates the checksums of all known migrations and updates their records
// in the migrations table to have the calculated checksum.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s whose checksums have been
// recalculated.
func RecalculateAllChecksums(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.RecalculateAllChecksums(ctx, db)
}
