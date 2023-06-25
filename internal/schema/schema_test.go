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

func sqlStatement(x string) string {
	return strings.TrimSpace(strings.TrimPrefix(x, "--sql"))
}

func getDB(t *testing.T) *sql.DB {
	db, err := sql.Open("pgx", "postgres://postgres:password@localhost:5433/postgres?sslmode=disable")
	assert.Nil(t, err)
	return db
}

func TestParseEmptyDatabase(t *testing.T) {
	t.Parallel()
	config := schema.Config{Schema: "public"}
	db := getDB(t)
	result, err := schema.Parse(config, db)
	assert.Nil(t, err)
	assert.NotEqual(t, nil, result)
	assert.NotEqual(t, nil, result.Extensions)
	assert.NotEqual(t, nil, result.Domains)
	assert.NotEqual(t, nil, result.Enums)
	assert.NotEqual(t, nil, result.Functions)
	assert.NotEqual(t, nil, result.Tables)
	assert.NotEqual(t, nil, result.Views)
	assert.NotEqual(t, nil, result.Sequences)
	assert.NotEqual(t, nil, result.Indexes)
	_ = schema.Dump(result)
}

func TestParseSimpleExample(t *testing.T) {
	t.Parallel()
	config := schema.Config{Schema: "public"}
	ctx := context.Background()
	original := sqlStatement(`--sql
CREATE DOMAIN public.score AS double precision
CHECK (VALUE >= 0::double precision AND VALUE <= 100::double precision);

CREATE EXTENSION "pgcrypto";
CREATE EXTENSION "pg_trgm";
	`)

	expected := sqlStatement(`--sql
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

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
		check.Equal(t, expected, schema.Dump(result))
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
		check.Equal(t, expected, schema.Dump(result))
		return nil
	}))
}
