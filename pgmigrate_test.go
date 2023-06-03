package pgmigrate_test

import (
	"embed"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/migrations"

	"github.com/peterldowns/pgmigrate"
)

//go:embed internal/migrations/*.sql
var repoRoot embed.FS

// This test confirms that Load() will find migrations in any
// subdirectory of the given filesystem.
//
// repoRoot is an embedFS with the following contents:
// ./internal/migrations
// ├── 0001_cats.sql
// ├── 0003_dogs.sql
// ├── 0003_empty.bkp.sql
// ├── 0004_rm_me.sql
// └── migrations.go
//
// migrations.FS is an embedFS with the following contents:
// .
// ├── 0001_cats.sql
// ├── 0003_dogs.sql
// ├── 0003_empty.bkp.sql
// ├── 0004_rm_me.sql
// └── migrations.go
//
// Both should return the same set of migrations, in the same order.
func TestLoadFromFSWalksSubdirs(t *testing.T) {
	t.Parallel()
	fromRoot, err := pgmigrate.Load(repoRoot)
	check.Nil(t, err)
	fromDir, err := pgmigrate.Load(migrations.FS)
	check.Nil(t, err)
	assert.NoFailures(t)

	expected := []string{
		"0001_cats",
		"0003_dogs",
		"0003_empty.bkp",
		"0004_rm_me",
	}
	check.Equal(t, expected, getIDs(fromDir))
	check.Equal(t, expected, getIDs(fromRoot))
}

func getIDs(migs []pgmigrate.Migration) []string {
	ids := make([]string, 0, len(migs))
	for _, m := range migs {
		ids = append(ids, m.ID)
	}
	return ids
}
