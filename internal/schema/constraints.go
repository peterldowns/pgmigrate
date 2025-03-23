package schema

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Constraint struct {
	OID                int
	Schema             string
	Name               string
	TableName          string
	Definition         string
	Type               string
	Index              string
	ForeignTableSchema string
	ForeignTableName   string
	ForeignColumns     []string
	LocalColumns       []string
	IsDeferrable       bool
	InitiallyDeferred  bool
	dependencies       []string
}

func (c Constraint) SortKey() string {
	return c.Name
}

func (c *Constraint) DependsOn() []string {
	deps := append(c.dependencies, c.TableName) //nolint:gocritic // appendAssign
	if c.ForeignTableName != "" {
		deps = append(deps, c.ForeignTableName)
	}
	if c.Index != "" {
		deps = append(deps, c.Index)
	}
	return deps
}

func (c *Constraint) AddDependency(dep string) {
	c.dependencies = append(c.dependencies, dep)
}

func (c Constraint) String() string {
	return fmt.Sprintf(query(`--sql
ALTER TABLE %s
ADD CONSTRAINT %s
%s;
	`),
		pgtools.Identifier(c.Schema, c.TableName),
		pgtools.Identifier(c.Name),
		c.Definition,
	)
}

func LoadConstraints(config Config, db *sql.DB) ([]*Constraint, error) {
	var constraints []*Constraint
	rows, err := db.Query(constraintsQuery, config.Schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var constraint Constraint
		if err := rows.Scan(
			&constraint.OID,
			&constraint.Schema,
			&constraint.Name,
			&constraint.TableName,
			&constraint.Definition,
			&constraint.Type,
			&constraint.Index,
			&constraint.ForeignTableSchema,
			&constraint.ForeignTableName,
			pq.Array(&constraint.ForeignColumns),
			pq.Array(&constraint.LocalColumns),
			&constraint.IsDeferrable,
			&constraint.InitiallyDeferred,
		); err != nil {
			return nil, err
		}
		constraints = append(constraints, &constraint)
	}
	return Sort[string](constraints), nil
}

// This query is inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
var constraintsQuery = query(`--sql
with
extensions as (
  select
      objid as "oid"
  from
      pg_depend d
  WHERE
      d.refclassid = 'pg_extension'::regclass
      and d.classid = 'pg_constraint'::regclass
),
indexes as (
    select
        schemaname as schema,
        tablename as table_name,
        indexname as name,
        indexdef as definition,
        indexdef as create_statement
    FROM
        pg_indexes
    order by
        schemaname, tablename, indexname
)
select
	pg_constraint.oid as "oid",
    nspname as "schema",
    conname as "name",
    relname as "table_name",
    pg_get_constraintdef(pg_constraint.oid) as "definition",
    case contype
        when 'c' then 'check'
        when 'f' then 'foreign_key'
        when 'p' then 'primary_key'
        when 'u' then 'unique'
        when 'x' then 'exclude'
    end as "type",
    coalesce(i.name, '') as "index",
    case when contype = 'f' then
        (
            SELECT nspname
            FROM pg_catalog.pg_class AS c
            JOIN pg_catalog.pg_namespace AS ns
            ON c.relnamespace = ns.oid
            WHERE c.oid = confrelid::regclass
        )
	else ''
    end as "foreign_table_schema",
    case when contype = 'f' then
        (
            select relname
            from pg_catalog.pg_class c
            where c.oid = confrelid::regclass
        )
	else ''
    end as "foreign_table_name",
	(
		select
			array_agg(ta.attname order by c.rn)
		from
		pg_attribute ta
		join unnest(confkey) with ordinality c(cn, rn)

		on
			ta.attrelid = confrelid and ta.attnum = c.cn
	) as "foreign_columns",
	(
		select
			array_agg(ta.attname order by c.rn)
		from
		pg_attribute ta
		join unnest(conkey) with ordinality c(cn, rn)

		on
			ta.attrelid = conrelid and ta.attnum = c.cn
	) as "local_columns",
    condeferrable as is_deferrable,
    condeferred as initially_deferred
from
    pg_constraint 
    INNER JOIN pg_class
        ON conrelid=pg_class.oid
    INNER JOIN pg_namespace
        ON pg_namespace.oid=pg_class.relnamespace
    left outer join indexes i
        on nspname = i.schema
        and conname = i.name
        and relname = i.table_name
    left outer join extensions e
      on pg_class.oid = e.oid
    where contype in ('c', 'f', 'p', 'u', 'x')
		and nspname = $1
		and e.oid is null
order by 1, 3, 2;
`)
