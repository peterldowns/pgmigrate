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

func TestLoadDataEmpty(t *testing.T) {
	t.Parallel()
	config := schema.DumpConfig{SchemaNames: []string{"public"}}
	ctx := context.Background()
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		data, err := schema.LoadData(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, []*schema.Data{}, data)
		return nil
	})
	assert.Nil(t, err)
}

func TestLoadDataParsesAttrs(t *testing.T) {
	t.Parallel()
	config := schema.DumpConfig{
		SchemaNames: []string{"public"},
		Data: []schema.Data{
			{
				Schema:  "public",
				Name:    "cats",
				Columns: []string{"name", "created_at"},
				OrderBy: "name asc",
			},
		},
	}
	ctx := context.Background()
	def := query(`--sql
CREATE TABLE cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	name TEXT NOT NULL
);

INSERT INTO cats (name)
VALUES ('daisy'), ('sunny'), ('kimbop'), ('charlie'), ('sesame');
;
	`)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.Exec(def); err != nil {
			return err
		}
		data, err := schema.LoadData(config, db)
		if err != nil {
			return err
		}
		if !check.Equal(t, 1, len(data)) {
			return nil
		}
		parsed := data[0]
		check.Equal(t, "public", parsed.Schema)
		check.Equal(t, "cats", parsed.Name)
		check.Equal(t, []string{"name", "created_at"}, parsed.Columns)
		return nil
	})
	assert.Nil(t, err)
}
