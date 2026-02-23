package ops

import (
	"fmt"
	"io/fs"

	"github.com/peterldowns/pgmigrate"
)

// newMigrator loads migrations from dir and returns a configured migrator for
// ops commands, including the caller-provided table name and logger.
func newMigrator(dir fs.FS, tableName string, logger pgmigrate.Logger) (*pgmigrate.Migrator, error) {
	migrations, err := pgmigrate.Load(dir)
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	m := pgmigrate.NewMigrator(migrations)
	m.Logger = logger
	m.TableName = tableName
	return m, nil
}
