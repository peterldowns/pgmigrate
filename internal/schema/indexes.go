package schema

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Index struct {
	OID                 int
	Schema              string
	TableName           string
	Name                string
	Definition          string
	IndexColumns        []string
	KeyOptions          string
	TotalColumnCount    int
	KeyColumnCount      int
	NumAtt              int
	IncludedColumnCount int
	IsUnique            bool
	IsPrimaryKey        bool
	IsExclusion         bool
	IsImmediate         bool
	IsClustered         bool
	KeyCollations       string
	KeyExpressions      sql.NullString
	PartialPredicate    sql.NullString
	Algorithm           string
	KeyColumns          []string
	IncludedColumns     []string
	dependencies        []string
}

func (i Index) SortKey() string {
	return pgtools.Identifier(i.Schema, i.Name)
}

func (i Index) DependsOn() []string {
	return append(
		i.dependencies,
		pgtools.Identifier(i.Schema, i.TableName),
	)
}

func (i *Index) AddDependency(dep string) {
	i.dependencies = append(i.dependencies, dep)
}

func (i Index) String() string {
	return fmt.Sprintf("%s;", i.Definition)
}

func LoadIndexes(config DumpConfig, db *sql.DB) ([]*Index, error) {
	var indexes []*Index
	snames := config.SchemaNames
	rows, err := db.Query(indexesQuery, snames)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var index Index
		if err := rows.Scan(
			&index.OID,
			&index.Schema,
			&index.TableName,
			&index.Name,
			&index.Definition,
			pq.Array(&index.IndexColumns),
			&index.KeyOptions,
			&index.TotalColumnCount,
			&index.KeyColumnCount,
			&index.NumAtt,
			&index.IncludedColumnCount,
			&index.IsUnique,
			&index.IsPrimaryKey,
			&index.IsExclusion,
			&index.IsImmediate,
			&index.IsClustered,
			&index.KeyCollations,
			&index.KeyExpressions,
			&index.PartialPredicate,
			&index.Algorithm,
			pq.Array(&index.KeyColumns),
			pq.Array(&index.IncludedColumns),
		); err != nil {
			return nil, err
		}
		indexes = append(indexes, &index)
	}
	return Sort[string](indexes), nil
}

// This query is inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
// - psql '\di+ <index>' with '\set ECHO_HIDDEN on'
// - psql '\di+ <table>' with '\set ECHO_HIDDEN on'
var indexesQuery = query(`--sql
with
-- Objects (tables, functions, types, views, etc.) that belong to extensions
-- will be filtered out of the final query result.
extensions as (
	select
		objid as "oid",
		classid::regclass::text as "classid"
	from pg_depend d
	where d.refclassid = 'pg_extension'::regclass
	union
	select
		t.typrelid as "oid",
		null as "classid"
	from pg_depend d
	join pg_type t on t.oid = d.objid
	where d.refclassid = 'pg_extension'::regclass
),
extension_relations as (
	select
		objid as "oid"
	from pg_depend d
	where
      d.refclassid = 'pg_extension'::regclass
      and d.classid = 'pg_class'::regclass
),
pre as (
	select
		i.oid as "oid",
		n.nspname::text as "schema",
		c.relname as "table_name",
		i.relname as "name",
		pg_get_indexdef(i.oid) as "definition",
		coalesce((
			select
				array_agg(aa.attname order by ik.n)
			from
				unnest(x.indkey) with ordinality ik(i, n)
				join pg_attribute aa on
					aa.attrelid = x.indrelid
					and ik.i = aa.attnum
		), '{}') as "index_columns",
		x.indoption as "key_options",
		x.indnatts as "total_column_count",
		x.indnkeyatts as "key_column_count",
		x.indnatts as "num_att",
		x.indnatts - x.indnkeyatts as "included_column_count",
		x.indisunique as "is_unique",
		x.indisprimary as "is_pk",
		x.indisexclusion as "is_exclusion",
		x.indimmediate as "is_immediate",
		x.indisclustered as "is_clustered",
		x.indcollation as "key_collations",
		pg_get_expr(x.indexprs, x.indrelid) as "key_expressions",
		pg_get_expr(x.indpred, x.indrelid) as "partial_predicate",
		am.amname as "algorithm"
	from pg_index x
	join pg_class c on c.oid = x.indrelid
	join pg_class i on i.oid = x.indexrelid
	join pg_am am on i.relam = am.oid
	left join pg_namespace n on n.oid = c.relnamespace
		left join extensions e
		on i.oid = e.oid
	left join extension_relations er
		on c.oid = er.oid
where
    x.indislive
    and c.relkind in ('r', 'm', 'p') AND i.relkind in ('i', 'I')
	and n.nspname::text = ANY($1)
	and e.oid is null
	and er.oid is null
)
select
	* ,
	coalesce(index_columns[1:key_column_count], '{}') as "key_columns",
	coalesce(index_columns[key_column_count+1:array_length(index_columns, 1)], '{}') as "included_columns"
from pre
order by
	"schema",
	"table_name",
	"name"
`)
