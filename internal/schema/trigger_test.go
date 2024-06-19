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

var schemaWithTriggers = query(`--sql
CREATE OR REPLACE FUNCTION public.updated_at()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
  BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
  END;
$function$
;

CREATE TABLE pull_requests (
	id bigint primary key,
	updated_at timestamptz not null default now()
);

CREATE TABLE background_jobs (
	id bigint primary key,
	updated_at timestamptz not null default now()
);

CREATE TABLE background_job_settings (
	id bigint primary key,
	updated_at timestamptz not null default now()
);

create trigger updated_at
before update on pull_requests
for each row execute procedure updated_at(); -- PROCEDURE

create trigger updated_at
before update on background_jobs
for each row execute function updated_at(); -- FUNCTION

create trigger updated_at
before update on background_job_settings
for each row execute function updated_at(); -- FUNCTIOn
	`)

func TestLoadTriggersWithSameName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, schemaWithTriggers); err != nil {
			return err
		}
		config := schema.Config{Schema: "public"}
		_, err := schema.LoadTables(config, db)
		if err != nil {
			return err
		}
		triggers, err := schema.LoadTriggers(config, db)
		if err != nil {
			return err
		}
		// Check that there are 3 separate triggers, not 1, because the three
		// triggers all have the same name and there used to be a bug where only
		// 1 would be returned.
		check.Equal(t, 3, len(triggers))
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadTriggersWithoutAnyTriggers(t *testing.T) {
	t.Parallel()
	config := schema.Config{Schema: "public"}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		views, err := schema.LoadTriggers(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, []*schema.Trigger{}, views)
		return nil
	})
	assert.Nil(t, err)
}
