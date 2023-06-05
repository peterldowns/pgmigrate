package pgmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/peterldowns/pgmigrate/internal/sessionlock"
	"github.com/peterldowns/pgmigrate/logging"
)

const (
	// DefaultTableName is the default name of the migrations table that
	// pgmigrate will use to store a record of applied migrations.
	DefaultTableName string = "pgmigrate_migrations"

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
type Migrator struct {
	// Migrations is the full set of migrations that describe the desired state
	// of the database.
	Migrations []Migration
	// Logger is used by the Migrator to log messages as it operates. It is
	// designed to be easy to adapt to whatever logging system you use.
	//
	// [NewMigrator] defaults it to `nil`, which will prevent any messages from
	// being logged.
	Logger logging.Logger
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
			m.debug(ctx, fmt.Sprintf("%d", i), logging.Field{Key: "migration_id", Value: migration.ID})
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

func (m *Migrator) ensureMigrationsTable(ctx context.Context, db Executor) error {
	m.info(ctx, "ensuring migrations table exists", logging.Field{Key: "table_name", Value: m.TableName})
	query := fmt.Sprintf(`
				CREATE TABLE IF NOT EXISTS %s (
					id TEXT PRIMARY KEY,
					checksum TEXT NOT NULL,
					execution_time_in_millis BIGINT NOT NULL,
					applied_at TIMESTAMPTZ NOT NULL
				)
			`, quoteIdentifier(m.TableName))
	_, err := db.ExecContext(ctx, query)
	return err
}

func (m *Migrator) hasMigrationsTable(ctx context.Context, db Executor) (bool, error) {
	query := fmt.Sprintf(`
				SELECT EXISTS (
					SELECT FROM pg_tables
					WHERE tablename = %s
				);
			`, quoteLiteral(m.TableName))
	var exists bool
	err := db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

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

// Applied retrieves all data from the migrations tracking table
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
		FROM %s ORDER BY id ASC
	`, quoteIdentifier(m.TableName))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := []AppliedMigration{}
	for rows.Next() {
		migration := AppliedMigration{}
		err = rows.Scan(&migration.ID, &migration.Checksum, &migration.ExecutionTimeInMillis, &migration.AppliedAt)
		if err != nil {
			return nil, err
		}
		migration.AppliedAt = migration.AppliedAt.UTC()
		applied = append(applied, migration)
	}

	return applied, rows.Err()
}

// applyMigration runs a single migration inside a transaction:
// - BEGIN;
// - apply the migration
// - insert a record marking the migration as applied
// - COMMIT;
// TODO: multierr here
func (m *Migrator) applyMigration(ctx context.Context, db Executor, migration Migration) error {
	startedAt := time.Now().UTC()
	fields := []logging.Field{
		{Key: "migration_id", Value: migration.ID},
		{Key: "migration_checksum", Value: migration.MD5()},
		{Key: "started_at", Value: startedAt},
	}
	// Open the transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		msg := "failed to open transaction"
		m.error(ctx, err, msg, fields...)
		return fmt.Errorf("%s: %w", msg, err)
	}
	// Rolling back is a no-op if the transaction was already committed.
	defer func() { _ = tx.Rollback() }()
	// Apply the migration
	m.info(ctx, "applying migration", fields...)
	_, err = tx.ExecContext(ctx, migration.SQL)
	finishedAt := time.Now().UTC()
	executionTimeMs := finishedAt.Sub(startedAt).Milliseconds()
	fields = append(fields,
		logging.Field{Key: "execution_time_ms", Value: executionTimeMs},
		logging.Field{Key: "finished_at", Value: finishedAt},
	)
	if err != nil {
		msg := "failed to apply migration"
		for key, val := range pgErrorData(err) {
			fields = append(fields, logging.Field{Key: key, Value: val})
		}
		m.error(ctx, err, msg, fields...)
		return fmt.Errorf("%s: %w", msg, err)
	}
	m.info(ctx, "migration succeeded", fields...)
	// Mark the migration as applied
	applied := AppliedMigration{}
	applied.ID = migration.ID
	applied.SQL = migration.SQL
	applied.ExecutionTimeInMillis = executionTimeMs
	applied.AppliedAt = startedAt
	query := fmt.Sprintf(`
		INSERT INTO %s
		( id, checksum, execution_time_in_millis, applied_at )
		VALUES
		( $1, $2, $3, $4 )`,
		quoteIdentifier(m.TableName),
	)
	_, err = tx.ExecContext(ctx, query, applied.ID, applied.MD5(), applied.ExecutionTimeInMillis, applied.AppliedAt)
	if err != nil {
		msg := "failed to mark migration as applied"
		m.error(ctx, err, msg, fields...)
		return fmt.Errorf("%s: %w", msg, err)
	}
	m.info(ctx, "marked as applied", fields...)
	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		msg := "failed to commit migration"
		m.error(ctx, err, msg, fields...)
		return fmt.Errorf("%s: %w", msg, err)
	}
	return nil
}

// Verify will detect and return any verification errors.
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
					"migration_id":                   appliedMigration.ID,
					"migration_applied_at":           appliedMigration.AppliedAt,
					"checksum_of_previously_applied": appliedMigration.Checksum,
					"checksum_of_current_calculated": md5,
				},
			})
		}
	}
	return verrs, nil
}

func (m *Migrator) log(ctx context.Context, level logging.Level, msg string, args ...logging.Field) {
	if m.Logger != nil {
		if hl, ok := m.Logger.(logging.Helper); ok {
			hl.Helper()
		}
		m.Logger.Log(ctx, level, msg, args...)
	}
}

func (m *Migrator) info(ctx context.Context, msg string, args ...logging.Field) {
	if logger, ok := m.Logger.(logging.Helper); ok {
		logger.Helper()
	}
	m.log(ctx, logging.LevelInfo, msg, args...)
}

func (m *Migrator) debug(ctx context.Context, msg string, args ...logging.Field) {
	if logger, ok := m.Logger.(logging.Helper); ok {
		logger.Helper()
	}
	m.log(ctx, logging.LevelDebug, msg, args...)
}

func (m *Migrator) error(ctx context.Context, err error, msg string, args ...logging.Field) {
	args = append(args, logging.Field{Key: "error", Value: err})
	if logger, ok := m.Logger.(logging.Helper); ok {
		logger.Helper()
	}
	m.log(ctx, logging.LevelError, msg, args...)
}
