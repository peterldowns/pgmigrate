package schema

import (
	"database/sql"
	"fmt"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type View struct {
	OID            int
	Schema         string
	Name           string
	Definition     string
	Comment        sql.NullString
	IsMaterialized bool
	Columns        []Column
	Dependencies   []string
}

func (v View) SortKey() string {
	return v.Name
}

func (v View) DependsOn() []string {
	return v.Dependencies
}

func (v *View) AddDependency(dep string) {
	v.Dependencies = append(v.Dependencies, dep)
}

func (v View) String() string {
	var def string
	if v.IsMaterialized {
		def = fmt.Sprintf("CREATE MATERIALIZED VIEW %s AS", pgtools.Identifier(v.Schema, v.Name))
	} else {
		def = fmt.Sprintf("CREATE VIEW %s AS", pgtools.Identifier(v.Schema, v.Name))
	}
	// The definition is not pretty printed, but has strange indentation rules:
	// - 1 leading space before the beginning SELECT
	// - 4 spaces as the tab indent on the columns
	// - 3 spaces before the final FROM
	// so this indents the first line by two additional spaces to make things a
	// little more sane (just barely)
	def = fmt.Sprintf("%s\n  %s", def, v.Definition)

	if v.Comment.Valid {
		def = def + "\n\n" + fmt.Sprintf(
			"COMMENT ON VIEW %s IS %s;",
			pgtools.Identifier(v.Schema, v.Name),
			pgtools.Literal(v.Comment.String),
		)
	}
	for _, column := range v.Columns {
		if column.Comment.Valid {
			def = def + "\n\n" + fmt.Sprintf(
				"COMMENT ON COLUMN %s IS %s;",
				pgtools.Identifier(v.Schema, v.Name, column.Name),
				pgtools.Literal(column.Comment.String),
			)
		}
	}
	return def
}

func LoadViews(config Config, db *sql.DB) ([]*View, error) {
	var views []*View
	rows, err := db.Query(viewsQuery, config.Schemas)
	if err != nil {
		return nil, err
	}
	var current *View
	for rows.Next() {
		var view View
		var column Column
		if err := rows.Scan(
			&view.OID,
			&view.Schema,
			&view.Name,
			&view.IsMaterialized,
			&view.Definition,
			&view.Comment,
			&column.Number,
			&column.Name,
			&column.NotNull,
			&column.DataType,
			&column.IsIdentity,
			&column.IsIdentityAlways,
			&column.IsGenerated,
			&column.Collation,
			&column.DefaultDef,
			&column.Comment,
		); err != nil {
			return nil, err
		}
		if current == nil || current.OID != view.OID {
			current = &view
			views = append(views, current)
		}
		current.Columns = append(current.Columns, column)
	}
	return Sort[string](views), nil
}

// This query was inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
// - psql '\dv+ <view>' with '\set ECHO_HIDDEN on'
var viewsQuery = query(`--sql
with r as (
	select
		c.oid as "oid",
		c.relname as "name",
		n.nspname as "schema",
		c.relkind = 'm' as "is_materialized",
		pg_get_viewdef(c.oid) as "definition"
	from
		pg_catalog.pg_class c
		inner join pg_catalog.pg_namespace n
		  ON n.oid = c.relnamespace
	where c.relkind in ('m', 'v')
	and n.nspname = ANY($1)
)
select
	r.oid as "view_oid",
	r.schema as "view_schema",
	r.name as "view_name",
	r.is_materialized as "view_is_materialized",
	r.definition as "view_definition",
	obj_description(r.oid) as "view_comment",
	a.attnum as "column_number",
	a.attname as "name",
	a.attnotnull as "not_null",
	format_type(atttypid, atttypmod) AS "data_type",
	a.attidentity != '' as "is_identity",
	a.attidentity = 'a' as "is_identity_always",
	a.attgenerated != '' as "is_generated",
	( SELECT c.collname FROM pg_catalog.pg_collation c, pg_catalog.pg_type t
	  WHERE c.oid = a.attcollation AND t.oid = a.atttypid AND a.attcollation <> t.typcollation
	) AS "collation",
	pg_get_expr(ad.adbin, ad.adrelid) as "default_def",
	col_description(r.oid, a.attnum) as "column_comment"
FROM
	r
	left join pg_catalog.pg_attribute a
		on r.oid = a.attrelid and a.attnum > 0
	left join pg_catalog.pg_attrdef ad
		on a.attrelid = ad.adrelid
		and a.attnum = ad.adnum
where
	a.attisdropped is not true
order by
	"view_schema",
	"view_name",
	"column_number"
`)
