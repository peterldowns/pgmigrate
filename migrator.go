package pgmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/peterldowns/pgmigrate/internal/multierr"
	"github.com/peterldowns/pgmigrate/internal/pgtools"
	"github.com/peterldowns/pgmigrate/internal/sessionlock"
)

const (
	// DefaultTableName is the default name of the migrations table (with
	// schema) that pgmigrate will use to store a record of applied migrations.
	DefaultTableName string = "public.pgmigrate_migrations"

	// sessionLockPrefix is prefix used by pgmigrate to help prevent conflicts
	// between its lock and other users of Postgres advisory locks. This prefix
	// is used to construct a lock name which is then hashed to an integer.
	sessionLockPrefix string = "pgmigrate-"
)

// Executor is satisfied by *sql.DB as well as *sql.Conn. Many of the Migrator's
// methods are designed to work inside of a session-scoped lock, which requires
// running queries on a *sql.Conn. These methods accept an Executor so that they
// can more easily be used by an external caller.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Migrator should be instantiated with [NewMigrator] rather than used directly.
// It contains the state necessary to perform migrations-related operations.
type Migrator struct {
	// Migrations is the full set of migrations that describe the desired state
	// of the database.
	Migrations []Migration
	// Logger is used by the Migrator to log messages as it operates. It is
	// designed to be easy to adapt to whatever logging system you use.
	//
	// [NewMigrator] defaults it to `nil`, which will prevent any messages from
	// being logged.
	Logger Logger
	// TableName is the table that this migrator should use to keep track of
	// applied migrations.
	//
	// [NewMigrator] defaults it to [DefaultTableName].
	TableName string
}

// NewMigrator creates a [Migrator] and sets appropriate default values for all
// configurable fields:
//
//   - Logger: `nil`, no messages will be logged
//   - TableName: [DefaultTableName]
//
// To configure these fields, just set the values on the struct.
func NewMigrator(
	migrations []Migration,
) *Migrator {
	return &Migrator{
		Migrations: migrations,
		Logger:     nil,
		TableName:  DefaultTableName,
	}
}

// Migrate will apply any previously applied migrations. It stores metadata in the
// database with the following schema:
//
//   - id: text not null
//   - checksum: text not null
//   - execution_time_in_millis: integer not null
//   - applied_at: timestamp with time zone not null
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
func (m *Migrator) Migrate(ctx context.Context, db *sql.DB) ([]VerificationError, error) {
	var verrs []VerificationError
	lockName := fmt.Sprintf("%s-%s", sessionLockPrefix, m.TableName)
	return verrs, sessionlock.With(ctx, db, lockName, func(conn *sql.Conn) error {
		err := m.ensureMigrationsTable(ctx, conn)
		if err != nil {
			return err
		}
		plan, err := m.Plan(ctx, conn)
		if err != nil {
			return err
		}
		m.info(ctx, fmt.Sprintf("planning to apply %d migrations", len(plan)))
		for i, migration := range plan {
			m.debug(ctx, fmt.Sprintf("%d", i), LogField{Key: "migration_id", Value: migration.ID})
		}
		for _, migration := range plan {
			err = m.applyMigration(ctx, conn, migration)
			if err != nil {
				return err
			}
		}
		m.info(ctx, "checking for verification errors")
		verrs, err = m.Verify(ctx, db)
		return err
	})
}

// ensureMigrationsTable will create the migrations table if it does not exist.
func (m *Migrator) ensureMigrationsTable(ctx context.Context, db Executor) error {
	m.info(ctx, "ensuring migrations table exists", LogField{Key: "table_name", Value: m.TableName})
	schema, _ := pgtools.ParseTableName(m.TableName)
	if schema != "" {
		query := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, pgtools.Identifier(schema))
		m.debug(ctx, query)
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("ensureMigrationsTable/create schema: %w", err)
		}
	}
	query := fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					id TEXT PRIMARY KEY,
					checksum TEXT NOT NULL,
					execution_time_in_millis BIGINT NOT NULL,
					applied_at TIMESTAMPTZ NOT NULL
				)
			`, pgtools.Identifier(m.TableName))
	m.debug(ctx, query)
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("ensureMigrationsTable: %w", err)
	}
	return nil
}

// hasMigrationsTable returns true if the migrations table exists, false
// otherwise.
func (m *Migrator) hasMigrationsTable(ctx context.Context, db Executor) (bool, error) {
	schema, tablename := pgtools.ParseTableName(m.TableName)
	query := fmt.Sprintf(`
				SELECT EXISTS (
					SELECT FROM pg_tables
					WHERE tablename = %s AND schemaname = %s
				);
			`, pgtools.Literal(tablename), pgtools.Literal(schema))
	m.debug(ctx, query)
	var exists bool
	err := db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("hasMigrationsTable: %w", err)
	}
	return exists, nil
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
func (m *Migrator) Plan(ctx context.Context, db Executor) ([]Migration, error) {
	applied, err := m.Applied(ctx, db)
	if err != nil {
		return nil, err
	}
	appliedMap := map[string]AppliedMigration{}
	for _, m := range applied {
		appliedMap[m.ID] = m
	}
	var plan []Migration
	for _, migration := range m.Migrations {
		_, exists := appliedMap[migration.ID]
		if !exists {
			plan = append(plan, migration)
		}
	}
	SortByID(plan)
	return plan, nil
}

// Applied returns a list of [AppliedMigration]s in the order that they were
// applied in (applied_at ASC, id ASC).
//
// If there are no applied migrations, or the specified table does not exist,
// this will return an empty list without an error.
func (m *Migrator) Applied(ctx context.Context, db Executor) ([]AppliedMigration, error) {
	hasMigrations, err := m.hasMigrationsTable(ctx, db)
	if err != nil {
		return nil, err
	}
	if !hasMigrations {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT id, checksum, execution_time_in_millis, applied_at
		FROM %s ORDER BY applied_at, id ASC
	`, pgtools.Identifier(m.TableName))
	m.debug(ctx, query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return scanAppliedMigrations(rows)
}

func (m *Migrator) inTx(ctx context.Context, db Executor, cb func(tx *sql.Tx) error) (final error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		msg := "tx open"
		m.error(ctx, err, msg)
		return fmt.Errorf("%s: %w", msg, err)
	}
	defer func() {
		if final != nil {
			if err := tx.Rollback(); err != nil {
				final = multierr.Join(final, fmt.Errorf("tx rollback: %w", err))
			}
		} else {
			if err := tx.Commit(); err != nil {
				final = multierr.Join(final, fmt.Errorf("tx commit: %w", err))
			}
		}
	}()
	return cb(tx)
}

// applyMigration runs a single migration inside a transaction:
// - BEGIN;
// - apply the migration
// - insert a record marking the migration as applied
// - COMMIT;
func (m *Migrator) applyMigration(ctx context.Context, db Executor, migration Migration) error {
	startedAt := time.Now().UTC()
	fields := []LogField{
		{Key: "migration_id", Value: migration.ID},
		{Key: "migration_checksum", Value: migration.MD5()},
		{Key: "started_at", Value: startedAt},
	}
	m.info(ctx, "applying migration", fields...)
	return m.inTx(ctx, db, func(tx *sql.Tx) error {
		// Run the migration SQL
		_, err := tx.ExecContext(ctx, migration.SQL)
		finishedAt := time.Now().UTC()
		executionTimeMs := finishedAt.Sub(startedAt).Milliseconds()
		fields = append(fields,
			LogField{Key: "execution_time_ms", Value: executionTimeMs},
			LogField{Key: "finished_at", Value: finishedAt},
		)
		if err != nil {
			msg := "failed to apply migration"
			for key, val := range pgtools.ErrorData(err) {
				fields = append(fields, LogField{Key: key, Value: val})
			}
			m.error(ctx, err, msg, fields...)
			return fmt.Errorf("%s: %w", msg, err)
		}
		m.info(ctx, "migration succeeded", fields...)
		// Mark the migration as applied
		applied := AppliedMigration{Migration: migration}
		applied.Checksum = migration.MD5()
		applied.ExecutionTimeInMillis = executionTimeMs
		applied.AppliedAt = startedAt
		query := fmt.Sprintf(`
			INSERT INTO %s
			( id, checksum, execution_time_in_millis, applied_at )
			VALUES
			( $1, $2, $3, $4 )`,
			pgtools.Identifier(m.TableName),
		)
		m.debug(ctx, query)
		_, err = tx.ExecContext(ctx, query, applied.ID, applied.Checksum, applied.ExecutionTimeInMillis, applied.AppliedAt)
		if err != nil {
			msg := "failed to mark migration as applied"
			m.error(ctx, err, msg, fields...)
			return fmt.Errorf("%s: %w", msg, err)
		}
		m.info(ctx, "marked as applied", fields...)
		return nil
	})
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
func (m *Migrator) Verify(ctx context.Context, db Executor) ([]VerificationError, error) {
	migrations := m.Migrations
	applied, err := m.Applied(ctx, db)
	if err != nil {
		return nil, err
	}

	hashes := map[string]string{}
	for _, migration := range migrations {
		hashes[migration.ID] = migration.MD5()
	}

	var verrs []VerificationError
	for _, appliedMigration := range applied {
		md5, ok := hashes[appliedMigration.ID]
		if !ok {
			verrs = append(verrs, VerificationError{
				Message: "found applied migration not present on disk",
				Fields: map[string]any{
					"migration_id":         appliedMigration.ID,
					"migration_applied_at": appliedMigration.AppliedAt,
					"migration_checksum":   appliedMigration.Checksum,
				},
			})
			continue
		}
		if appliedMigration.Checksum != md5 {
			verrs = append(verrs, VerificationError{
				Message: "found applied migration with a different checksum",
				Fields: map[string]any{
					"migration_id":               appliedMigration.ID,
					"migration_applied_at":       appliedMigration.AppliedAt,
					"migration_checksum_from_db": appliedMigration.Checksum,
					"calculated_checksum":        md5,
				},
			})
		}
	}
	return verrs, nil
}

func (m *Migrator) log(ctx context.Context, level LogLevel, msg string, args ...LogField) {
	if m.Logger != nil {
		if hl, ok := m.Logger.(Helper); ok {
			hl.Helper()
		}
		m.Logger.Log(ctx, level, msg, args...)
	}
}

func (m *Migrator) info(ctx context.Context, msg string, args ...LogField) {
	if logger, ok := m.Logger.(Helper); ok {
		logger.Helper()
	}
	m.log(ctx, LogLevelInfo, msg, args...)
}

func (m *Migrator) debug(ctx context.Context, msg string, args ...LogField) {
	if logger, ok := m.Logger.(Helper); ok {
		logger.Helper()
	}
	m.log(ctx, LogLevelDebug, msg, args...)
}

func (m *Migrator) error(ctx context.Context, err error, msg string, args ...LogField) {
	args = append(args, LogField{Key: "error", Value: err})
	if logger, ok := m.Logger.(Helper); ok {
		logger.Helper()
	}
	m.log(ctx, LogLevelError, msg, args...)
}

func (m *Migrator) warn(ctx context.Context, msg string, args ...LogField) {
	if logger, ok := m.Logger.(Helper); ok {
		logger.Helper()
	}
	m.log(ctx, LogLevelWarning, msg, args...)
}

func scanAppliedMigrations(rows *sql.Rows) ([]AppliedMigration, error) {
	defer rows.Close()
	var migrations []AppliedMigration
	for rows.Next() {
		migration := AppliedMigration{}
		err := rows.Scan(
			&migration.ID,
			&migration.Checksum,
			&migration.ExecutionTimeInMillis,
			&migration.AppliedAt,
		)
		if err != nil {
			return nil, err
		}
		migration.AppliedAt = migration.AppliedAt.UTC()
		migrations = append(migrations, migration)
	}
	return migrations, nil
}
