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

// Test that you can manually specify a dependency across schema types, and that
// it will change the output order of the related objects to respect the
// dependency.
func TestCrossSchemaDependency(t *testing.T) {
	t.Parallel()
	t.Fatalf("not implemented")
}

// Everywhere that we do dependency tracking, ordering, sorting, etc. — need to
// make sure that the sort keys and identifiers respect and include the Schema
// now.
func TestSameObjectNameInDifferentSchemas(t *testing.T) {
	t.Parallel()
	t.Fatalf("not implemented")
}

// Test that when a table has a dependency on an object in a different
// schema, but that schema is not explicitly mentioned in "things to load",
// that the dependency is not included in the result.
//
// Also check the recursive case, where A dep on B dep on C, A and C are
// included in the config.Schemas, and object from B is not included.
//
// Or, figure out a way to implement it so that the full chain of objects are
// included? That's probably the most useful thing to do.
func TestCrossSchemaConstraints(t *testing.T) {
	t.Parallel()
	t.Fatalf("not implemented")
}

// Definitely need to update the schema.Config and the associated
// YAML format. Test loading those configs, appropriately reading
// the potentially-multiple schemas.
//
// Consider: do we want to use fully-qualified table and object names
// everywhere, instead of the {Schema, Name} split that we've been using?
// Might make it easier to specify some things?
//
// Probably want to keep the split, but change the name of this package
// to "Parse" or something like that.
func TestSchemaConfiguration(t *testing.T) {
	t.Parallel()
	t.Fatalf("not implemented")
}

// Make sure that the schema names are all being escaped appropriately, show
// examples of what is valid and what isn't.
func TestSchemaQuotingInQueries(t *testing.T) {
	t.Parallel()
	t.Fatalf("not implemented")
}

func TestMultiSchemaRoundtrip(t *testing.T) {
	t.Parallel()
	config := schema.Config{
		Schemas: []string{
			"public",
			"aaa",
			"bbb",
		},
		Data: []schema.Data{
			{
				Schema:  "public",
				Name:    "cats",
				Columns: []string{"name"},
			},
			{
				Schema: "aaa",
				Name:   "dogs",
			},
		},
	}
	ctx := context.Background()
	def := query(`--sql
CREATE SCHEMA IF NOT EXISTS public;

CREATE SCHEMA IF NOT EXISTS aaa;

CREATE SCHEMA IF NOT EXISTS bbb;

CREATE TABLE aaa.dogs (
  name text PRIMARY KEY NOT NULL,
  enemy_cat_name text
);

CREATE TABLE public.cats (
  name text PRIMARY KEY NOT NULL,
  created_at timestamp with time zone NOT NULL DEFAULT now()
);

ALTER TABLE aaa.dogs
ADD CONSTRAINT dogs_enemy_cat_name_fkey
FOREIGN KEY (enemy_cat_name) REFERENCES cats(name);

CREATE TABLE bbb.fish (
  name text PRIMARY KEY NOT NULL,
  associated_cat_name text,
  associated_dog_name text
);

ALTER TABLE bbb.fish
ADD CONSTRAINT fish_associated_cat_name_fkey
FOREIGN KEY (associated_cat_name) REFERENCES cats(name);

ALTER TABLE bbb.fish
ADD CONSTRAINT fish_associated_dog_name_fkey
FOREIGN KEY (associated_dog_name) REFERENCES aaa.dogs(name);

INSERT INTO public.cats (name) VALUES
('daisy'),
('sunny'),
('kimbop'),
('charlie'),
('sesame')
;

INSERT INTO aaa.dogs (name, enemy_cat_name) VALUES
('rufus', 'daisy'),
('bob', null),
('john', null),
('gizmo', 'charlie')
;
	`)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.Exec(def); err != nil {
			return err
		}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, def, result.String())
		return nil
	})
	assert.Nil(t, err)
}
