package schema

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Enum struct {
	OID          int
	Schema       string
	Name         string
	InternalName string
	Description  sql.NullString
	Size         string
	Elements     []string
	dependencies []string
}

func (e Enum) SortKey() string {
	return pgtools.Identifier(e.Schema, e.Name)
}

func (e Enum) DependsOn() []string {
	return e.dependencies
}

func (e *Enum) AddDependency(dep string) {
	e.dependencies = append(e.dependencies, dep)
}

func (e Enum) String() string {
	def := fmt.Sprintf("CREATE TYPE %s AS ENUM (", pgtools.Identifier(e.Schema, e.Name))
	lastIndex := len(e.Elements) - 1
	for i, element := range e.Elements {
		def = fmt.Sprintf("%s\n\t%s", def, pgtools.Literal(element))
		if i != lastIndex {
			def += ","
		}
	}
	def = fmt.Sprintf("%s\n);", def)
	return def
}

func LoadEnums(config DumpConfig, db *sql.DB) ([]*Enum, error) {
	var enums []*Enum
	rows, err := db.Query(enumsQuery, config.SchemaNames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var enum Enum
		if err := rows.Scan(
			&enum.OID,
			&enum.Schema,
			&enum.Name,
			&enum.InternalName,
			&enum.Size,
			pq.Array(&enum.Elements),
			&enum.Description,
		); err != nil {
			return nil, err
		}
		enums = append(enums, &enum)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return Sort(enums), nil
}

// This query is inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
// - psql '\dT+ <enum>' with '\set ECHO_HIDDEN on'
// - pg_dump dumpEnumType https://github.com/postgres/postgres/blob/9a2dbc614e6e47da3c49daacec106da32eba9467/src/bin/pg_dump/pg_dump.c#L10488
var enumsQuery = query(`--sql
SELECT t.oid as "oid",
  n.nspname as "schema",
  pg_catalog.format_type(t.oid, NULL) AS "name",
  t.typname AS "internal_name",
  CASE WHEN t.typrelid != 0
      THEN CAST('tuple' AS pg_catalog.text)
    WHEN t.typlen < 0
      THEN CAST('var' AS pg_catalog.text)
    ELSE CAST(t.typlen AS pg_catalog.text)
  END AS "size",
      ARRAY(
          SELECT e.enumlabel
          FROM pg_catalog.pg_enum e
          WHERE e.enumtypid = t.oid
          ORDER BY e.enumsortorder
      )::text[]
  AS "elements",
    pg_catalog.obj_description(t.oid, 'pg_type') as "description"
FROM pg_catalog.pg_type t
     LEFT JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
WHERE (t.typrelid = 0 OR (SELECT c.relkind = 'c' FROM pg_catalog.pg_class c WHERE c.oid = t.typrelid))
  AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_type el WHERE el.oid = t.typelem AND el.typarray = t.oid)
  AND pg_catalog.pg_type_is_visible(t.oid)
  AND t.typcategory = 'E'
  AND n.nspname = ANY($1)
ORDER BY 1, 2;
`)
