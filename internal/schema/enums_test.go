package schema_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestLoadEnumsSucceedsWithoutAnyEnums(t *testing.T) {
	t.Parallel()
	config := schema.Config{Schema: "public"}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		enums, err := schema.LoadEnums(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, []*schema.Enum{}, enums)
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadEnumResultIsStable(t *testing.T) {
	t.Parallel()
	original := sqlStatement(`--sql
create type mood as ENUM ('sad', 'ok', 'happy');
	`)
	result := sqlStatement(`--sql
CREATE TYPE public.mood AS ENUM (
	'sad',
	'ok',
	'happy'
);
	`)
	checkEnum(t, original, result)
	checkEnum(t, result, result)
}

func TestLoadEnumWithoutValues(t *testing.T) {
	t.Parallel()
	original := sqlStatement(`--sql
CREATE TYPE public.no_values AS ENUM (
);
	`)
	checkEnum(t, original, original)
}

func checkEnum(t *testing.T, definition, result string) {
	t.Helper()
	config := schema.Config{Schema: "public"}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, definition); err != nil {
			return err
		}
		enums, err := schema.LoadEnums(config, db)
		if err != nil {
			return err
		}
		if check.Equal(t, 1, len(enums)) {
			parsed := enums[0].String()
			if !check.Equal(t, result, parsed) {
				t.Logf("expected\n%s", result)
				t.Logf(" received\n%s", parsed)
			}
		}
		return nil
	})
	assert.Nil(t, err)
}
