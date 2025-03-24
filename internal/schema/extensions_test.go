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

func TestLoadExtensionsSucceedsWithoutAnyExtensions(t *testing.T) {
	t.Parallel()
	config := schema.Config{Schemas: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		extensions, err := schema.LoadExtensions(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, 0, len(extensions))
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadExtensionRoundtrips(t *testing.T) {
	t.Parallel()
	original := "CREATE EXTENSION pgcrypto;"
	result := `CREATE EXTENSION IF NOT EXISTS pgcrypto;`
	checkExtension(t, original, result)
	checkExtension(t, result, result)
}

func checkExtension(t *testing.T, definition, result string) {
	t.Helper()
	config := schema.Config{Schemas: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, definition); err != nil {
			return err
		}
		extensions, err := schema.LoadExtensions(config, db)
		if err != nil {
			return err
		}
		asStrings := make([]string, 0, len(extensions))
		for _, ext := range extensions {
			asStrings = append(asStrings, ext.String())
		}
		if !check.In(t, result, asStrings) {
			t.Logf("   slice\n%+v", asStrings)
			t.Logf("expected\n%s", result)
		}
		return nil
	})
	assert.Nil(t, err)
}
