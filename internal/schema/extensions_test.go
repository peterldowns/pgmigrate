package schema_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestLoadExtensionsSucceedsWithoutAnyExtensions(t *testing.T) {
	t.Parallel()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
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
	result := `CREATE EXTENSION IF NOT EXISTS pgcrypto SCHEMA public;`
	checkExtension(t, original, result)
	checkExtension(t, result, result)
}

func TestLoadExtensionWithSchemaClause(t *testing.T) {
	t.Parallel()
	// Test the bug fix: extensions with SCHEMA clause should roundtrip correctly
	original := "CREATE EXTENSION IF NOT EXISTS unaccent SCHEMA public;"
	result := `CREATE EXTENSION IF NOT EXISTS unaccent SCHEMA public;`
	checkExtension(t, original, result)
}

func TestSchemaOrderingWithExtensions(t *testing.T) {
	t.Parallel()
	// Test that schemas come before extensions in the dump
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS unaccent SCHEMA public;"); err != nil {
			return err
		}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}

		dump := result.String()

		// Check that "CREATE SCHEMA" appears before "CREATE EXTENSION"
		schemaPos := strings.Index(dump, "CREATE SCHEMA")
		extensionPos := strings.Index(dump, "CREATE EXTENSION")

		if schemaPos == -1 {
			t.Error("Schema definition not found in dump")
		}
		if extensionPos == -1 {
			t.Error("Extension definition not found in dump")
		}
		if schemaPos > extensionPos {
			t.Errorf("Schema should come before extension. Schema at %d, Extension at %d\nDump:\n%s",
				schemaPos, extensionPos, dump)
		}

		return nil
	})
	assert.Nil(t, err)
}

func checkExtension(t *testing.T, definition, result string) {
	t.Helper()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
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
