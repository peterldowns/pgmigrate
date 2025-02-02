package schema

import "database/sql"

type Object struct {
	OID    int
	Schema string
	Name   string
	Kind   string
}

type Dependency struct { // TODO: explain not sortable!
	Object    Object
	DependsOn Object
}

func LoadDependencies(config Config, db *sql.DB) ([]*Dependency, error) {
	var deps []*Dependency

	rows, err := db.Query(dependenciesQuery, config.Schemas)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var dep Dependency
		if err := rows.Scan(
			&dep.Object.OID,
			&dep.Object.Schema,
			&dep.Object.Name,
			&dep.Object.Kind,
			&dep.DependsOn.OID,
			&dep.DependsOn.Schema,
			&dep.DependsOn.Name,
			&dep.DependsOn.Kind,
		); err != nil {
			return nil, err
		}
		deps = append(deps, &dep)
	}
	return deps, nil
}

// This query is inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
var dependenciesQuery = query(`--sql
with
-- Objects (tables, functions, types, views, etc.) that belong to extensions
-- will be filtered out of the final query result.
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
),
functions_tables_and_views as (
	select
		oid as "oid",
		pronamespace::regnamespace::text as "schema",
		proname as "name",
		prokind as "kind"
	from pg_proc
	-- included:
	--   'f' == normal function
	--   'p' == procedure
	-- excluded:
	--   'a' == aggregate function
	--   'w' == window function
	-- https://www.postgresql.org/docs/15/catalog-pg-proc.html
	where pg_proc.prokind in ('f', 'p')
	union
	select
		oid as "oid",
		relnamespace::regnamespace::text as "schema",
		relname as "name",
		relkind as "kind"
	from pg_class
	-- included:
	--   'r' == normal tables
	--   'v' == normal view
	--   'm' == materialized view
	-- excluded:
	--   'i' == index
	--   'S' == sequence
	--   't' == TOAST table
	--   'c' == composite type
	--   'f' == foreign table
	--   'p' == partitioned table
	--   'I' == partitioned index
	-- https://www.postgresql.org/docs/15/catalog-pg-class.html
	where relkind in ('r', 'v', 'm')
),
filtered as (
	select
		o.oid,
		o.schema,
		o.name,
		o.kind
	from functions_tables_and_views o
	left outer join extensions e
		on o.oid = e.oid
	-- exclude any objects defined/created by extensions
	-- exclude any objects outside of the desired schema
	where o.schema = ANY($1) and e.oid is null
),
dependencies as (
	select distinct
		x.oid as "oid",
		x.schema as "schema",
		x.name as "name",
		x.kind as "kind",
		y.oid as "on_oid",
		y.schema as "on_schema",
		y.name as "on_name",
		y.kind as "on_kind"
	from pg_depend d
	inner join filtered y
		on d.refobjid = y.oid
	-- https://www.postgresql.org/docs/current/rules-views.html
	inner join pg_rewrite rw
        on d.objid = rw.oid
        and y.oid != rw.ev_class
	inner join filtered x
		on rw.ev_class = x.oid
	
	-- 'n' == DEPENDENCY_NORMAL
	-- dropping x requires dropping y first, or using cascade.
	-- https://www.postgresql.org/docs/current/catalog-pg-depend.html
	where d.deptype = 'n'
)
select
	"oid",
	"schema",
	"name",
	"kind",
	"on_oid",
	"on_schema",
	"on_name",
	"on_kind"
from dependencies
order by
	"schema",
	"name",
	"on_schema",
	"on_name"
`)
