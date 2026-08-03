package schema_test

import (
	"database/sql"
	"testing"

	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
)

func TestDumpUniqueIndexWithExpressionKeys(t *testing.T) {
	t.Parallel()
	original := query(`--sql
CREATE TABLE records (
  scope text NOT NULL,
  label text NOT NULL
);

CREATE UNIQUE INDEX records_scope_label_idx
ON records (
  scope,
  lower(label)
);
	`)
	expected := query(`--sql
CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE public.records (
  scope text NOT NULL,
  label text NOT NULL
);

CREATE UNIQUE INDEX records_scope_label_idx ON public.records USING btree (scope, lower(label));
	`)

	checkSchemaDump(t, original, expected)
	checkSchemaDump(t, expected, expected)
}

func TestDumpUniqueConstraintAndExpressionIndexOnSameColumn(t *testing.T) {
	t.Parallel()
	original := query(`--sql
CREATE TABLE records (
  scope text UNIQUE NOT NULL,
  label text NOT NULL
);

CREATE UNIQUE INDEX records_scope_label_idx
ON records (
  scope,
  lower(label)
);
	`)
	expected := query(`--sql
CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE public.records (
  scope text UNIQUE NOT NULL,
  label text NOT NULL
);

CREATE UNIQUE INDEX records_scope_label_idx ON public.records USING btree (scope, lower(label));
	`)

	checkSchemaDump(t, original, expected)
	checkSchemaDump(t, expected, expected)
}

func TestDumpPartialUniqueIndex(t *testing.T) {
	t.Parallel()
	original := query(`--sql
CREATE TABLE records (
  key text NOT NULL,
  retired_at timestamptz
);

CREATE UNIQUE INDEX current_records_key_idx
ON records (key)
WHERE retired_at IS NULL;
	`)
	expected := query(`--sql
CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE public.records (
  key text NOT NULL,
  retired_at timestamp with time zone
);

CREATE UNIQUE INDEX current_records_key_idx ON public.records USING btree (key) WHERE (retired_at IS NULL);
	`)

	checkSchemaDump(t, original, expected)
	checkSchemaDump(t, expected, expected)
}

func TestDumpSingleColumnUniqueConstraint(t *testing.T) {
	t.Parallel()
	original := query(`--sql
CREATE TABLE users (
  id text PRIMARY KEY,
  email text UNIQUE NOT NULL
);
	`)
	expected := query(`--sql
CREATE SCHEMA IF NOT EXISTS public;

CREATE TABLE public.users (
  id text PRIMARY KEY NOT NULL,
  email text UNIQUE NOT NULL
);
	`)

	checkSchemaDump(t, original, expected)
	checkSchemaDump(t, expected, expected)
}

func checkSchemaDump(t *testing.T, definition, expected string) {
	t.Helper()
	dbtest(t, definition, func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, expected, result.String())
		return nil
	})
}
