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

func Verify(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]VerificationError, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.Verify(ctx, db)
}

func Plan(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]Migration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.Plan(ctx, db)
}

func Applied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.Applied(ctx, db)
}

func MarkApplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, ids ...string) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkApplied(ctx, db, ids...)
}

func MarkAllApplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkAllApplied(ctx, db)
}

func MarkUnapplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, ids ...string) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkUnapplied(ctx, db, ids...)
}

func MarkAllUnapplied(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.MarkAllUnapplied(ctx, db)
}

func SetChecksums(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, updates ...ChecksumUpdate) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.SetChecksums(ctx, db, updates...)
}

func RecalculateChecksums(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger, ids ...string) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.RecalculateChecksums(ctx, db, ids...)
}

func RecalculateAllChecksums(ctx context.Context, db *sql.DB, dir fs.FS, logger Logger) ([]AppliedMigration, error) {
	migrations, err := Load(dir)
	if err != nil {
		return nil, err
	}
	migrator := NewMigrator(migrations)
	migrator.Logger = logger
	return migrator.RecalculateAllChecksums(ctx, db)
}
