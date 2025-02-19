package schema_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for postgres
	"github.com/peterldowns/testy/assert"
	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
	"github.com/peterldowns/pgmigrate/internal/withdb"
)

// query is a helper for writing sql queries that look nice in vscode when using
// the "Inline SQL for go" extension by @jhnj, which gives syntax highlighting
// for strings that begin with `--sql`.
//
// https://marketplace.visualstudio.com/items?itemName=jhnj.vscode-go-inline-sql
func query(x string) string {
	return strings.TrimSpace(strings.TrimPrefix(x, "--sql"))
}

// dbtest is a helper for creating a new database,
// running some sql statements, and then running tests against
// that database.
func dbtest(t *testing.T, statements string, cb func(*sql.DB) error) {
	t.Helper()
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.Exec(statements); err != nil {
			return err
		}
		return cb(db)
	})
	check.Nil(t, err)
}

// asMap turns a slice of objects into a map of objects keyed by their
// SortKey().
func asMap[T schema.Sortable[string]](collections ...[]T) map[string]T {
	total := 0
	for _, obj := range collections {
		total += len(obj)
	}
	out := make(map[string]T, total)
	for _, collection := range collections {
		for _, object := range collection {
			out[object.SortKey()] = object
		}
	}
	return out
}

func TestParseEmptyDatabase(t *testing.T) {
	t.Parallel()
	dbtest(t, "", func(db *sql.DB) error {
		config := schema.Config{Schemas: []string{"public"}}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		check.NotEqual(t, nil, result)
		check.NotEqual(t, nil, result.Extensions)
		check.NotEqual(t, nil, result.Domains)
		check.NotEqual(t, nil, result.Enums)
		check.NotEqual(t, nil, result.Functions)
		check.NotEqual(t, nil, result.Tables)
		check.NotEqual(t, nil, result.Views)
		check.NotEqual(t, nil, result.Sequences)
		check.NotEqual(t, nil, result.Indexes)
		check.Equal(t, "CREATE SCHEMA IF NOT EXISTS public;", result.String())
		return nil
	})
}

func TestParseSimpleExample(t *testing.T) {
	t.Parallel()
	config := schema.Config{Schemas: []string{"public"}}
	ctx := context.Background()
	original := query(`--sql
CREATE DOMAIN public.score AS double precision
CHECK (VALUE >= 0::double precision AND VALUE <= 100::double precision);

CREATE EXTENSION "pgcrypto";
CREATE EXTENSION "pg_trgm";
	`)

	expected := query(`--sql
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE SCHEMA IF NOT EXISTS public;

CREATE DOMAIN public.score AS double precision
CHECK (VALUE >= 0::double precision AND VALUE <= 100::double precision);

	`)

	assert.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, original); err != nil {
			return err
		}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, expected, result.String())
		return nil
	}))
	assert.Nil(t, withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, expected); err != nil {
			return err
		}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, expected, result.String())
		return nil
	}))
}
