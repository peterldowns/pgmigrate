package pgmigrate

import (
	"testing"

	"github.com/peterldowns/testy/check"
)

func TestIDFromFilename(t *testing.T) {
	t.Parallel()
	check.Equal(t, "0001_initial", IDFromFilename("0001_initial.sql"))
	check.Equal(t, "0001_initial.up", IDFromFilename("0001_initial.up.sql"))
	check.Equal(t, "0001_initial", IDFromFilename("0001_initial"))
}

func TestSortByID(t *testing.T) {
	t.Parallel()

	t.Run("simple example", testcase( //nolint:paralleltest // it is parallel
		[]string{
			"0002_followup",
			"0001_initial",
		},
		[]string{
			"0001_initial",
			"0002_followup",
		},
	))
	t.Run("lexicographical ordering", testcase( //nolint:paralleltest // it is parallel
		[]string{
			"1_one",
			"0001_one",
			"01_one",
			"001_one",
		},
		[]string{
			"0001_one",
			"001_one",
			"01_one",
			"1_one",
		},
	))
	t.Run("more complicated", testcase( //nolint:paralleltest // it is parallel
		[]string{
			"0001_initial",
			"002_garbage",
			"03_something",
			"0002_followup",
			"0003_whatever",
		},
		[]string{
			"0001_initial",
			"0002_followup",
			"0003_whatever",
			"002_garbage",
			"03_something",
		},
	))
}

// testcase builds a test case for SortByID:
//   - initial contains the ids of some migrations in their original order.
//   - expected contains the ids of the same migrations in their expected sorted order.
//
// the testcase will construct the slice of Migration, sort it, and then check
// to make sure the result is in the expected ID order.
func testcase(initial, expected []string) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		migrations := make([]Migration, 0, len(initial))
		for _, id := range initial {
			migrations = append(migrations, Migration{ID: id, SQL: "-- not implemented"})
		}
		SortByID(migrations)
		check.Equal(t, expected, getIDs(migrations))
	}
}

func getIDs(migrations []Migration) []string {
	ids := make([]string, 0, len(migrations))
	for _, m := range migrations {
		ids = append(ids, m.ID)
	}
	return ids
}
