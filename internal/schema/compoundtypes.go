package schema

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

type CompoundTypeColumn struct {
	Name string
	Type string
}

func (tc *CompoundTypeColumn) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed type assertion to []byte")
	}
	return json.Unmarshal(b, &tc)
}

type CompoundType struct {
	OID          int
	Schema       string
	Name         string
	Columns      []CompoundTypeColumn
	dependencies []string
}

func (t CompoundType) SortKey() string {
	return t.Name
}

func (t CompoundType) DependsOn() []string {
	deps := t.dependencies
	for _, col := range t.Columns {
		deps = append(deps, col.Type)
	}
	return deps
}

func (t *CompoundType) AddDependency(dep string) {
	t.dependencies = append(t.dependencies, dep)
}

func (t CompoundType) String() string {
	out := fmt.Sprintf("CREATE TYPE %s AS (\n", identifier(t.Schema, t.Name))
	colDefs := make([]string, 0, len(t.Columns))
	for _, col := range t.Columns {
		colDefs = append(colDefs, fmt.Sprintf("  %s %s", identifier(col.Name), col.Type))
	}
	out += strings.Join(colDefs, ",\n")
	out += "\n);"
	return out
}

func LoadCompoundTypes(config Config, db *sql.DB) ([]*CompoundType, error) {
	var types []*CompoundType
	rows, err := db.Query(compoundTypesQuery, config.Schemas)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var ct CompoundType
		if err := rows.Scan(
			&ct.OID,
			&ct.Schema,
			&ct.Name,
			pq.Array(&ct.Columns),
		); err != nil {
			return nil, err
		}
		types = append(types, &ct)
	}
	return Sort[string](types), nil
}

// This query is inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
var compoundTypesQuery = query(`--sql
with
extensions as (
	select
		objid as "oid"
	from pg_depend d
	where d.refclassid = 'pg_extension'::regclass
	union
	select
		t.typrelid as "oid"
	from pg_depend d
	join pg_type t on t.oid = d.objid
	where d.refclassid = 'pg_extension'::regclass
)
SELECT
  t.oid as "oid",
  t.typnamespace::regnamespace::text as "schema",
  pg_catalog.format_type (t.oid, NULL) AS "name",
  array(
    select
      jsonb_build_object('name', attname, 'type', a.typname)
    from pg_class
    join pg_attribute on (attrelid = pg_class.oid)
    join pg_type a on (atttypid = a.oid)
    where (pg_class.reltype = t.oid)
  ) as columns
FROM
  pg_catalog.pg_type t
  left outer join extensions e on t.oid = e.oid
WHERE
	e.oid is null
	and t.typnamespace::regnamespace::text = ANY($1)
	AND pg_catalog.pg_type_is_visible ( t.oid )
	and t.typcategory = 'C'
	and (
		t.typrelid = 0
		OR (
			SELECT c.relkind = 'c'
			FROM pg_catalog.pg_class c
			WHERE c.oid = t.typrelid
		)
	)
ORDER BY 1, 2;
`)
