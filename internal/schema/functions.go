package schema

import (
	"database/sql"
	"fmt"
)

type Function struct {
	OID           int
	Schema        string
	Name          string
	Language      string
	Kind          string
	Volatility    string
	Parallel      string
	Security      string
	ResultType    string
	ArgumentTypes string
	Definition    string
	dependencies  []string
}

func (f Function) SortKey() string {
	return f.Name
}

func (f Function) DependsOn() []string {
	return f.dependencies
}

func (f *Function) AddDependency(dep string) {
	f.dependencies = append(f.dependencies, dep)
}

func (f Function) String() string {
	return fmt.Sprintf("%s;", f.Definition)
}

func LoadFunctions(config Config, db *sql.DB) ([]*Function, error) {
	var functions []*Function

	rows, err := db.Query(functionsQuery, config.Schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var function Function
		if err := rows.Scan(
			&function.OID,
			&function.Schema,
			&function.Name,
			&function.Language,
			&function.Kind,
			&function.Volatility,
			&function.Parallel,
			&function.Security,
			&function.ResultType,
			&function.ArgumentTypes,
			&function.Definition,
		); err != nil {
			return nil, err
		}
		functions = append(functions, &function)
	}
	return Sort[string](functions), nil
}

// Based on schemainspect and psql \df+ <function>.
var functionsQuery = query(`--sql
with
extensions as (
	select
		objid as "oid"
	from pg_depend d
	where
		d.refclassid = 'pg_extension'::regclass
		and d.classid = 'pg_proc'::regclass
),
functions as (
	select
	p.oid as "oid",
	p.pronamespace::regnamespace::text as "schema",
    p.proname as "name",
	l.lanname as "language",
	case p.prokind
		when 'a' then 'agg'
		when 'w' then 'window'
		when 'p' then 'proc'
		else 'func'
	end as "kind",
	case p.provolatile
		when 'i' then 'immutable'
		when 's' then 'stable'
		when 'v' then 'volatile'
	end as "volatility",
	case p.proparallel
		when 'r' then 'restricted'
		when 's' then 'safe'
		when 'u' then 'unsafe'
	end as "parallel",
	case p.prosecdef
		when true then 'definer'
		else 'invoker'
	end as "security",
	coalesce(pg_catalog.pg_get_function_result(p.oid), '') as "result_type",
	coalesce(pg_catalog.pg_get_function_arguments(p.oid), '') as "argument_types",
    pg_catalog.pg_get_functiondef(p.oid) as "definition"
from pg_catalog.pg_proc p
	left join extensions e
		on p.oid = e.oid
	left join pg_catalog.pg_language l
		on l.oid = p.prolang
where
	e.oid is null
	and pg_function_is_visible(p.oid)
)
select
	f.oid,
	f.schema,
	f.name,
	f.language,
	f.kind,
	f.volatility,
	f.parallel,
	f.security,
	f.result_type,
	f.argument_types,
	f.definition
from functions f
where
	schema = $1
order by
	"schema",
	"name"
;
`)
