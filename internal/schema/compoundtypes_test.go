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

func TestLoadCompoundTypesSucceedsWithEmptyDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		types, err := schema.LoadCompoundTypes(config, db)
		if err != nil {
			return err
		}
		assert.Equal(t, 0, len(types))
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadCompoundTypes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		_, err := db.ExecContext(ctx, `CREATE TYPE public.foo AS (age int4, name text);`)
		if err != nil {
			return err
		}

		types, err := schema.LoadCompoundTypes(config, db)
		if err != nil {
			return err
		}
		assert.Equal(t, 1, len(types))
		compfoo := types[0]
		check.Equal(t, "public", compfoo.Schema)
		check.Equal(t, "foo", compfoo.Name)
		check.Equal(t, []schema.CompoundTypeColumn{
			{Name: "age", Type: "int4"},
			{Name: "name", Type: "text"},
		}, compfoo.Columns)
		return nil
	})
	assert.Nil(t, err)
}
