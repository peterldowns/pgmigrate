package schema

import (
	"database/sql"
	"fmt"
	"strings"
)

type Table struct {
	OID          int
	Schema       string
	Name         string
	Columns      []*Column
	Dependencies []string
	Indexes      []*Index
	Constraints  []*Constraint
	Sequences    []*Sequence
	Triggers     []*Trigger
}

func (t Table) SortKey() string {
	return t.Name
}

func (t Table) DependsOn() []string {
	out := t.Dependencies
	for _, constraint := range t.Constraints {
		if constraint.ForeignTableName != "" {
			out = append(out, RefTable(constraint.ForeignTableName))
		}
	}
	for _, trig := range t.Triggers {
		if trig.ProcName != "" {
			out = append(out, trig.ProcName)
		}
	}
	return out
}

func (t *Table) AddDependency(dep string) {
	t.Dependencies = append(t.Dependencies, dep)
}

func (t Table) String() string {
	var colDefs []string
	pkIndexes := map[string]bool{}
	uniqueIndexes := map[string]bool{}
	implicitSeq := map[string]bool{}
	for _, c := range t.Columns {
		isPrimaryKey := false
		isUnique := false
		for _, index := range t.Indexes {
			if len(index.IndexColumns) == 1 && index.IndexColumns[0] == c.Name && index.IsPrimaryKey {
				pkIndexes[index.SortKey()] = true
				uniqueIndexes[index.SortKey()] = true
				isPrimaryKey = true
			}
			if len(index.IndexColumns) == 1 && index.IndexColumns[0] == c.Name && index.IsUnique {
				uniqueIndexes[index.SortKey()] = true
				isUnique = true
			}
		}
		if c.Sequence != nil && (c.Sequence.IsIdentity || c.Sequence.IsIdentityAlways) && (c.IsIdentity || c.IsIdentityAlways) {
			implicitSeq[c.Sequence.Name] = true
		}
		colDefs = append(colDefs, c.ToString(t.Name, isPrimaryKey, isUnique, ""))
	}
	sequenceDef := ""
	followUps := ""
	for _, sequence := range t.Sequences {
		if _, ok := implicitSeq[sequence.Name]; ok {
			continue
		}
		sequenceDef += sequence.String() + "\n\n"
		if f := sequence.Followup(); f != nil {
			followUps += f.String() + "\n\n"
		}
	}
	tableDef := fmt.Sprintf(query(`--sql
CREATE TABLE %s (
  %s
);
	`), identifier(t.Schema, t.Name), strings.Join(colDefs, ",\n  "))
	constraintsByName := asMap[string](t.Constraints)

	for _, index := range t.Indexes {
		if pkIndexes[index.SortKey()] {
			continue
		}
		if uniqueIndexes[index.SortKey()] {
			continue
		}
		if _, ok := constraintsByName[RefConstraint(index.Name)]; ok {
			continue
		}
		tableDef += "\n\n" + index.String()
	}
	for _, con := range t.Constraints {
		if uniqueIndexes[RefIndex(con.Name)] {
			continue
		}
		tableDef += "\n\n" + con.String()
	}
	for _, trig := range t.Triggers {
		tableDef += "\n\n" + trig.String()
	}
	out := sequenceDef + tableDef
	if followUps != "" {
		out += "\n\n" + followUps
	}
	return strings.TrimSpace(out) // TODO: this is garbage
}

func LoadTables(config Config, db *sql.DB) ([]*Table, error) {
	var tables []*Table
	rows, err := db.Query(tablesQuery, config.Schema)
	if err != nil {
		return nil, err
	}
	var current *Table
	for rows.Next() {
		var table Table
		var column Column
		if err := rows.Scan(
			&table.OID,
			&table.Schema,
			&table.Name,
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
		if current == nil || current.OID != table.OID {
			current = &table
			tables = append(tables, current)
		}
		column.BelongsTo = current.OID
		current.Columns = append(current.Columns, &column)
	}
	return Sort[string](tables), nil
}

var tablesQuery = query(`--sql
with r as (
	select
		c.oid as oid,
		c.relname as name,
		n.nspname as schema,
		c.relkind as relationtype
	from
		pg_catalog.pg_class c
		inner join pg_catalog.pg_namespace n
		  ON n.oid = c.relnamespace
	where c.relkind in ('r', 't', 'p')
	and n.nspname = $1
)
select
	r.oid as "table_oid",
	r.schema as "table_schema",
	r.name as "table_name",
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
	pg_catalog.obj_description(r.oid) as "comment"
FROM
	r
	left join pg_catalog.pg_attribute a
		on r.oid = a.attrelid and a.attnum > 0
	left join pg_catalog.pg_attrdef ad
		on a.attrelid = ad.adrelid
		and a.attnum = ad.adnum
where
	a.attisdropped is not true
	  and r.schema = $1
order by
	"table_schema",
	"table_name",
	"column_number"
`)
