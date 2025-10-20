package schema_test

import (
	"database/sql"
	"testing"

	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
)

func TestLoadConstraintsSucceedsWithEmptyDB(t *testing.T) {
	t.Parallel()
	dbtest(t, "", func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		constraints, err := schema.LoadConstraints(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, 0, len(constraints))
		return nil
	})
}

func TestLoadConstraintsReadsImplicitConstraints(t *testing.T) {
	t.Parallel()
	dbtest(t, query(`--sql
CREATE TABLE cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name TEXT NOT NULL CHECK (name != 'garbage')
);

INSERT INTO cats (name)
VALUES ('daisy'), ('sunny');
	`), func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		constraints, err := schema.LoadConstraints(config, db)
		if err != nil {
			return err
		}
		byName := asMap(constraints)
		if namecheck, ok := byName["public.cats_name_check"]; check.True(t, ok) {
			check.Equal(t, "cats", namecheck.TableName)
			check.Equal(t, query(`--sql
ALTER TABLE public.cats
ADD CONSTRAINT cats_name_check
CHECK ((name <> 'garbage'::text));
				`),
				namecheck.String(),
			)
		}
		if pkey, ok := byName["public.cats_pkey"]; check.True(t, ok) {
			check.Equal(t, "primary_key", pkey.Type)
			check.Equal(t, "cats", pkey.TableName)
			check.Equal(t, query(`--sql
ALTER TABLE public.cats
ADD CONSTRAINT cats_pkey
PRIMARY KEY (id);
				`),
				pkey.String(),
			)
		}
		return nil
	})
}

func TestLoadConstraintsIgnoresNotNull(t *testing.T) {
	t.Parallel()
	dbtest(t, query(`--sql
CREATE TABLE foo (
	name TEXT NOT NULL
);
	`), func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		constraints, err := schema.LoadConstraints(config, db)
		if err != nil {
			return err
		}
		check.Equal(t, 0, len(constraints))
		return nil
	})
}

func TestLoadConstraintsReadsExplicitConstraints(t *testing.T) {
	t.Parallel()
	dbtest(t, query(`--sql
CREATE TABLE foo (
	name TEXT NOT NULL
);

ALTER TABLE foo ADD CONSTRAINT no_bobs CHECK (name != 'bob');
	`), func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		constraints, err := schema.LoadConstraints(config, db)
		if err != nil {
			return err
		}
		byName := asMap(constraints)
		if nobobs, ok := byName["public.no_bobs"]; check.True(t, ok) {
			check.Equal(t, "check", nobobs.Type)
			check.Equal(t, query(`--sql
ALTER TABLE public.foo
ADD CONSTRAINT no_bobs
CHECK ((name <> 'bob'::text));
			`), nobobs.String())
		}
		return nil
	})
}

func TestLoadConstraintsReadsForeignKeys(t *testing.T) {
	t.Parallel()
	dbtest(t, query(`--sql
CREATE TABLE foo (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);
CREATE TABLE bar (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	foo_id BIGINT REFERENCES foo (id),
	another_foo_id BIGINT UNIQUE
);

ALTER TABLE bar
ADD CONSTRAINT bar_fkey_another_foo_id
FOREIGN KEY (another_foo_id) REFERENCES foo (id) NOT VALID;
	`), func(db *sql.DB) error {
		config := schema.DumpConfig{SchemaNames: []string{"public"}}
		constraints, err := schema.LoadConstraints(config, db)
		if err != nil {
			return err
		}
		byName := asMap(constraints)

		con, ok := byName["public.bar_foo_id_fkey"]
		if check.True(t, ok) {
			check.Equal(t, "foreign_key", con.Type)
			check.Equal(t, "public", con.Schema)
			check.Equal(t, "bar", con.TableName)
			check.Equal(t, []string{"foo_id"}, con.LocalColumns)
			check.Equal(t, "public", con.ForeignTableSchema)
			check.Equal(t, "foo", con.ForeignTableName)
			check.Equal(t, []string{"id"}, con.ForeignColumns)
			check.Equal(t, "", con.Index)
		}

		con, ok = byName["public.bar_fkey_another_foo_id"]
		if check.True(t, ok) {
			check.Equal(t, "foreign_key", con.Type)
			check.Equal(t, "public", con.Schema)
			check.Equal(t, "bar", con.TableName)
			check.Equal(t, []string{"another_foo_id"}, con.LocalColumns)
			check.Equal(t, "public", con.ForeignTableSchema)
			check.Equal(t, "foo", con.ForeignTableName)
			check.Equal(t, []string{"id"}, con.ForeignColumns)
			check.Equal(t, "", con.Index)
		}

		con, ok = byName["public.bar_another_foo_id_key"]
		if check.True(t, ok) {
			check.Equal(t, "unique", con.Type)
			check.Equal(t, "public", con.Schema)
			check.Equal(t, "bar", con.TableName)
			check.Equal(t, []string{"another_foo_id"}, con.LocalColumns)
			check.Equal(t, "bar_another_foo_id_key", con.Index)
			check.Equal(t, "", con.ForeignTableSchema)
			check.Equal(t, "", con.ForeignTableName)
			check.Equal(t, nil, con.ForeignColumns)
		}
		return nil
	})
}
