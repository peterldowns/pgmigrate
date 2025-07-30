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

func TestLoadViewsWithoutAnyViews(t *testing.T) {
	t.Parallel()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		views, err := schema.LoadViews(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, []*schema.View{}, views)
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadViewResultIsStable(t *testing.T) {
	t.Parallel()
	original := query(`--sql
CREATE VIEW foobar AS 
SELECT * from dogs;
	`)
	result := query(`--sql
CREATE VIEW public.foobar AS
   SELECT id,
    name,
    enemy_id
   FROM dogs;
`)
	checkView(t, original, result)
	checkView(t, result, result)
}

func TestLoadMaterializedViewIsStable(t *testing.T) {
	t.Parallel()
	original := query(`--sql
CREATE MATERIALIZED VIEW example AS
SELECT d.name as "Dog_Name" , d.id as dog_id, c.id as cat_id, c.name as cat_name
FROM dogs d
INNER JOIN cats c
ON d.enemy_id = c.id
ORDER BY d.name DESC;
`)
	result := query(`--sql
CREATE MATERIALIZED VIEW public.example AS
   SELECT d.name AS "Dog_Name",
    d.id AS dog_id,
    c.id AS cat_id,
    c.name AS cat_name
   FROM (dogs d
     JOIN cats c ON ((d.enemy_id = c.id)))
  ORDER BY d.name DESC;
`)
	checkView(t, original, result)
	checkView(t, result, result)
}

func checkView(t *testing.T, definition, result string) {
	t.Helper()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, query(`--sql
CREATE TABLE cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name TEXT
);

CREATE TABLE dogs (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name TEXT,
	enemy_id BIGINT REFERENCES cats (id)
);
		`)); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, definition); err != nil {
			return err
		}
		views, err := schema.LoadViews(config, db)
		if err != nil {
			return err
		}
		if check.Equal(t, 1, len(views)) {
			parsed := views[0].String()
			if !check.Equal(t, result, parsed) {
				t.Logf("expected\n%s", result)
				t.Logf(" received\n%s", parsed)
			}
		}
		return nil
	})
	assert.Nil(t, err)
}
