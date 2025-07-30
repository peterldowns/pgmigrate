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

// Test that if two tables don't otherwise have a dependency, you can explicitly
// specify the dependency using the dotted schema.table name in the config.
func TestExplicitCrossSchemaDependency(t *testing.T) {
	t.Parallel()
	config := schema.Config{
		Schemas: []string{
			"schema1",
			"schema2",
		},
		// If this is commented out, the CREATE TABLE
		// statements are ordered:
		//
		// - CREATE TABLE schema1.table1 (...)
		// - CREATE TABLE schema2.table2 (...)
		//
		// because of the default alphabetical sort, and the lack of
		// relationship between the tables.
		//
		// By specifying the dependency, the dumped result will create
		// schema2.table2 first, and then schema1.table1.
		Dependencies: map[string][]string{
			"schema1.table1": {"schema2.table2"},
		},
	}
	ctx := context.Background()
	def := query(`--sql
CREATE SCHEMA IF NOT EXISTS schema1;

CREATE SCHEMA IF NOT EXISTS schema2;

CREATE TABLE schema2.table2 (
  name text PRIMARY KEY NOT NULL
);

CREATE TABLE schema1.table1 (
  name text PRIMARY KEY NOT NULL
);
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

// When there are two different tables with the same name, but in different
// schemas, we correctly distinguish between the two schemas when detecting
// dependencies.
func TestCrossSchemaDependencyDifferentiatesBetweenSchemasWhenTableHasSameName(t *testing.T) {
	t.Parallel()
	config := schema.Config{
		Schemas: []string{
			"schema1",
			"schema2",
			"schema3",
		},
	}
	ctx := context.Background()
	def := query(`--sql
CREATE SCHEMA IF NOT EXISTS schema1;

CREATE SCHEMA IF NOT EXISTS schema2;

CREATE SCHEMA IF NOT EXISTS schema3;

CREATE TABLE schema1.cats (
  name text PRIMARY KEY NOT NULL
);

CREATE TABLE schema2.cats (
  name text PRIMARY KEY NOT NULL,
  owned_by text
);

CREATE TABLE schema3.people (
  name text PRIMARY KEY NOT NULL
);

ALTER TABLE schema2.cats
ADD CONSTRAINT cats_owned_by_fkey
FOREIGN KEY (owned_by) REFERENCES schema3.people(name);

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

// Test that if an object is in a schema that isn't part of the config, it is
// excluded from the dump. This is true even if it's depended on by an object that
// _is_ in a schema that is part of the config.
//
// In this test:
//   - ccc.c1 --(depends on)--> bbb.b1
//   - bbb.b1 --(depends on)--> aaa.a1
//
// But because we're only dumping objects from "aaa" and "ccc", the final dump
// will only include the definitions for ccc.c1 and aaa.a1.
func TestOnlyIncludeObjectsFromSpecifiedSchemas(t *testing.T) {
	t.Parallel()
	config := schema.Config{
		Schemas: []string{
			"aaa",
			// "bbb", // explicitly: DON'T include objects from the "bbb" schema
			"ccc",
		},
	}
	ctx := context.Background()
	def := query(`--sql
CREATE SCHEMA IF NOT EXISTS aaa;

CREATE SCHEMA IF NOT EXISTS bbb;

CREATE SCHEMA IF NOT EXISTS ccc;

CREATE TABLE aaa.a1 (
  name text PRIMARY KEY NOT NULL
);

CREATE TABLE bbb.b1 (
  name text PRIMARY KEY NOT NULL,
  a1_name text NOT NULL REFERENCES aaa.a1 (name)
);

CREATE TABLE ccc.c1 (
  name text PRIMARY KEY NOT NULL,
  b1_name text NOT NULL REFERENCES bbb.b1 (name)
);
	`)
	// Note that the definition for ccc.c1 still (correctly) carries a foreign
	// key reference to the bbb.b1 table, but that the definition of bbb.b1 is
	// missing.
	dumped := query(`--sql
CREATE SCHEMA IF NOT EXISTS aaa;

CREATE SCHEMA IF NOT EXISTS ccc;

CREATE TABLE aaa.a1 (
  name text PRIMARY KEY NOT NULL
);

CREATE TABLE ccc.c1 (
  name text PRIMARY KEY NOT NULL,
  b1_name text NOT NULL
);

ALTER TABLE ccc.c1
ADD CONSTRAINT c1_b1_name_fkey
FOREIGN KEY (b1_name) REFERENCES bbb.b1(name);
	`)
	err := withdb.WithDB(ctx, "pgx", func(db *sql.DB) error {
		if _, err := db.Exec(def); err != nil {
			return err
		}
		result, err := schema.Parse(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, dumped, result.String())
		return nil
	})
	assert.Nil(t, err)
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
