package pgmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

// MarkApplied (⚠️ danger) is a manual operation that marks specific migrations
// as applied without running them.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s that have been marked as
// applied.
func (m *Migrator) MarkApplied(ctx context.Context, db Executor, ids ...string) ([]AppliedMigration, error) {
	hasMigrations, err := m.hasMigrationsTable(ctx, db)
	if err != nil {
		return nil, err
	}
	if !hasMigrations {
		return nil, fmt.Errorf("migrations table %s does not exist", m.TableName)
	}
	applied, err := m.Applied(ctx, db)
	if err != nil {
		return nil, err
	}
	appliedMap := map[string]AppliedMigration{}
	for _, m := range applied {
		appliedMap[m.ID] = m
	}
	migrationsMap := map[string]Migration{}
	for _, m := range m.Migrations {
		migrationsMap[m.ID] = m
	}
	var toMarkApplied []Migration
	for _, id := range ids {
		if existing, ok := appliedMap[id]; ok {
			m.warn(ctx, "skipping previously applied migration",
				LogField{"id", existing.ID},
				LogField{"checksum", existing.Checksum},
				LogField{"applied_at", existing.AppliedAt},
			)
			continue
		}
		if migration, ok := migrationsMap[id]; ok {
			toMarkApplied = append(toMarkApplied, migration)
		} else {
			m.warn(ctx, "skipping unknown migration",
				LogField{"reason", "does not exist"},
				LogField{"id", id},
			)
		}
	}
	var markedAsApplied []AppliedMigration
	if err := m.inTx(ctx, db, func(tx *sql.Tx) error {
		for _, migration := range toMarkApplied {
			ma := AppliedMigration{Migration: migration}
			ma.Checksum = migration.MD5()
			ma.ExecutionTimeInMillis = 0
			ma.AppliedAt = time.Now().UTC()
			fields := []LogField{
				{"id", ma.ID},
				{"checksum", ma.Checksum},
			}
			query := fmt.Sprintf(`
				INSERT INTO %s ( id, checksum, execution_time_in_millis, applied_at )
				VALUES ( $1, $2, $3, $4 )
				ON CONFLICT DO NOTHING;`,
				pgtools.QuoteIdentifier(m.TableName),
			)
			_, err := tx.ExecContext(ctx, query, ma.ID, ma.Checksum, ma.ExecutionTimeInMillis, ma.AppliedAt)
			if err != nil {
				msg := "failed to mark migration as applied"
				m.error(ctx, err, msg, fields...)
				return fmt.Errorf("%s: %w", msg, err)
			}
			markedAsApplied = append(markedAsApplied, ma)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return markedAsApplied, nil
}

// MarkAllApplied (⚠️ danger) is a manual operation that marks all known migrations as
// applied without running them.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s that have been marked as
// applied.
func (m *Migrator) MarkAllApplied(ctx context.Context, db Executor) ([]AppliedMigration, error) {
	var ids []string
	for _, migration := range m.Migrations {
		ids = append(ids, migration.ID)
	}
	return m.MarkApplied(ctx, db, ids...)
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
func (m *Migrator) MarkUnapplied(ctx context.Context, db Executor, ids ...string) ([]AppliedMigration, error) {
	hasMigrations, err := m.hasMigrationsTable(ctx, db)
	if err != nil {
		return nil, err
	}
	if !hasMigrations {
		return nil, fmt.Errorf("migrations table %s does not exist", m.TableName)
	}
	applied, err := m.Applied(ctx, db)
	if err != nil {
		return nil, err
	}
	appliedMap := map[string]AppliedMigration{}
	for _, m := range applied {
		appliedMap[m.ID] = m
	}
	var toRemove []string
	for _, id := range ids {
		if _, ok := appliedMap[id]; ok {
			toRemove = append(toRemove, id)
		} else {
			m.warn(ctx, "skipping unknown migration",
				LogField{"reason", "does not exist"},
				LogField{"id", id},
			)
		}
	}
	var removed []AppliedMigration
	if err := m.inTx(ctx, db, func(tx *sql.Tx) error {
		query := fmt.Sprintf(`
			DELETE FROM %s WHERE id = any($1) RETURNING *;
		`, pgtools.QuoteIdentifier(m.TableName))
		rows, err := tx.QueryContext(ctx, query, toRemove)
		if err != nil {
			return err
		}
		removed, err = scanAppliedMigrations(rows)
		return err
	}); err != nil {
		return nil, err
	}
	return removed, err
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
func (m *Migrator) MarkAllUnapplied(ctx context.Context, db Executor) ([]AppliedMigration, error) {
	hasMigrations, err := m.hasMigrationsTable(ctx, db)
	if err != nil {
		return nil, err
	}
	if !hasMigrations {
		return nil, fmt.Errorf("migrations table %s does not exist", m.TableName)
	}
	applied, err := m.Applied(ctx, db)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(applied))
	for _, migration := range applied {
		ids = append(ids, migration.ID)
	}
	return m.MarkUnapplied(ctx, db, ids...)
}

// ChecksumUpdate represents an update to a specific migration.
// This struct is used instead of a `map[migrationID]checksum“
// in order to apply multiple updates in a consistent order.
type ChecksumUpdate struct {
	MigrationID string // The ID of the migration to update, `0001_initial`
	NewChecksum string // The checksum to set in the migrations table, `aaaabbbbccccdddd`
}

// SetChecksums (⚠️ danger) is a manual operation that explicitly sets the recorded
// checksum of applied migrations in the migrations table.
//
// You should NOT use this as part of normal operations, it exists to help
// devops/db-admin/sres interact with migration state.
//
// It returns a list of the [AppliedMigration]s whose checksums have been
// updated.
func (m *Migrator) SetChecksums(ctx context.Context, db Executor, updates ...ChecksumUpdate) ([]AppliedMigration, error) {
	hasMigrations, err := m.hasMigrationsTable(ctx, db)
	if err != nil {
		return nil, err
	}
	if !hasMigrations {
		return nil, fmt.Errorf("migrations table %s does not exist", m.TableName)
	}
	applied, err := m.Applied(ctx, db)
	if err != nil {
		return nil, err
	}
	appliedMap := map[string]AppliedMigration{}
	for _, m := range applied {
		appliedMap[m.ID] = m
	}
	var toUpdate []AppliedMigration
	for _, update := range updates {
		migrationID := update.MigrationID
		newChecksum := update.NewChecksum
		migration, ok := appliedMap[migrationID]
		if !ok {
			m.warn(ctx, "skipping migration",
				LogField{"reason", "does not exist"},
				LogField{"id", migrationID},
			)
			continue
		}
		if migration.Checksum == newChecksum {
			m.info(ctx, "skipping migration",
				LogField{"reason", "already has the desired checksum"},
				LogField{"id", migration.ID},
				LogField{"checksum", migration.Checksum},
			)
			continue
		}
		migration.Checksum = newChecksum
		toUpdate = append(toUpdate, migration)
	}
	var updated []AppliedMigration
	if err := m.inTx(ctx, db, func(tx *sql.Tx) error {
		for _, migration := range toUpdate {
			fields := []LogField{
				{"id", migration.ID},
				{"checksum", migration.Checksum},
			}
			query := fmt.Sprintf(
				`UPDATE %s SET checksum = $1 where id = $2 and checksum != $1`,
				pgtools.QuoteIdentifier(m.TableName),
			)
			_, err := tx.ExecContext(ctx, query, migration.Checksum, migration.ID)
			if err != nil {
				msg := "failed to set checksum"
				m.error(ctx, err, msg, fields...)
				return fmt.Errorf("%s: %w", msg, err)
			}
			updated = append(updated, migration)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return updated, nil
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
func (m *Migrator) RecalculateChecksums(ctx context.Context, db Executor, ids ...string) ([]AppliedMigration, error) {
	allChecksums := make(map[string]string, len(m.Migrations))
	for _, migration := range m.Migrations {
		allChecksums[migration.ID] = migration.MD5()
	}
	updates := make([]ChecksumUpdate, 0, len(ids))
	for _, id := range ids {
		if checksum, ok := allChecksums[id]; ok {
			updates = append(updates, ChecksumUpdate{MigrationID: id, NewChecksum: checksum})
		} else {
			m.warn(ctx, "skipping migration",
				LogField{"reason", "does not exist"},
				LogField{"id", id},
			)
		}
	}
	return m.SetChecksums(ctx, db, updates...)
}

func (m *Migrator) RecalculateAllChecksums(ctx context.Context, db Executor) ([]AppliedMigration, error) {
	updates := make([]ChecksumUpdate, 0, len(m.Migrations))
	for _, migration := range m.Migrations {
		updates = append(updates, ChecksumUpdate{
			MigrationID: migration.ID,
			NewChecksum: migration.MD5(),
		})
	}
	return m.SetChecksums(ctx, db, updates...)
}
