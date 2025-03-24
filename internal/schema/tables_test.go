package schema_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/peterldowns/testy/assert"

	"github.com/peterldowns/pgmigrate/internal/schema"
	"github.com/peterldowns/pgmigrate/internal/withdb"
)

func TestDumpingTablesWithPartitions(t *testing.T) {
	t.Parallel()

	config := schema.Config{Schema: "public"}
	ctx := context.Background()
	original := query(`--sql
create table events (
    created timestamp with time zone not null default now(),
    event text
) partition by range (created);

create table events_p20250101 partition of events for values from ('2025-01-01 00:00:00Z') to ('2025-02-01 00:00:00Z');

	`)
	expected := query(`--sql
CREATE TABLE public.events (
  created timestamp with time zone NOT NULL DEFAULT now(),
  event text
) PARTITION BY RANGE (created);

CREATE TABLE public.events_p20250101 PARTITION OF public.events FOR VALUES FROM ('2025-01-01 00:00:00+00') TO ('2025-02-01 00:00:00+00');
	`)

	var result *schema.Schema
	// Check that the "original" parses correctly and results in the "expected" SQL.
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		var err error
		if _, err = db.ExecContext(ctx, original); err != nil {
			return err
		}
		result, err = schema.Parse(config, db)
		return err
	})
	assert.Nil(t, err)
	assert.NotEqual(t, nil, result)
	assert.Equal(t, expected, result.String())
	// Check that the "expected" result perfectly roundtrips and results in itself.
	err = withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		var err error
		if _, err = db.ExecContext(ctx, expected); err != nil {
			return err
		}
		result, err = schema.Parse(config, db)
		return err
	})
	assert.Nil(t, err)
	assert.NotEqual(t, nil, result)
	assert.Equal(t, expected, result.String())
}

func TestPartitionedTablesDependOnEachOther(t *testing.T) {
	t.Parallel()

	config := schema.Config{Schema: "public"}
	ctx := context.Background()
	original := query(`--sql
create table events (
    created timestamp with time zone not null default now(),
    event text
) partition by range (created);

create table events_p20250101 partition of events for values from ('2025-01-01 00:00:00Z') to ('2025-02-01 00:00:00Z');

	`)
	var result *schema.Schema
	// Check that the "original" parses correctly and results in the "expected" SQL.
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		var err error
		if _, err = db.ExecContext(ctx, original); err != nil {
			return err
		}
		result, err = schema.Parse(config, db)
		return err
	})
	assert.Nil(t, err)
	assert.NotEqual(t, nil, result)

	tables := asMap(result.Tables)
	parent, ok := tables["public.events"]
	assert.True(t, ok)
	child, ok := tables["public.events_p20250101"]
	assert.True(t, ok)
	assert.Equal(t, []string{"public.events"}, child.DependsOn())
	assert.Equal(t, nil, parent.DependsOn())
}
