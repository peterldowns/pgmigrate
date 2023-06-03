package pgmigrate

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a single SQL migration.
type Migration struct {
	ID  string // the filename of the migration, without the .sql extension
	SQL string // the contents of the migration file
}

// MD5 computes the MD5 hash of the SQL for this migration so that it can be
// uniquely identified. After a Migration is applied, the [AppliedMigration]
// will store this hash in the `Checksum` field.
func (m *Migration) MD5() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(m.SQL)))
}

// AppliedMigration represents a successfully-executed [Migration]. It embeds
// the [Migration], and adds fields for execution results.
type AppliedMigration struct {
	Migration
	Checksum              string    // The MD5 hash of the SQL of this migration
	ExecutionTimeInMillis int64     // How long it took to run this migration
	AppliedAt             time.Time // When the migration was run
}

// IDFromFilename removes directory paths and extensions from the filename to
// return just the filename (no extension).
//
// Examples:
//
//	"0001_initial" == IDFromFilename("0001_initial.sql")
//	"0002_whatever.up" == IDFromFilename("0002_whatever.up.sql")
func IDFromFilename(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}

// SortByID sorts a slice of [Migration] in ascending lexicographical order by
// their ID. This means that they should show up in the same order that they
// appear when you use `ls` or `sort`.
func SortByID(migrations []Migration) {
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID < migrations[j].ID
	})
}
