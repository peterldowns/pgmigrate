package schema

import (
	"database/sql"
	"fmt"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Domain struct {
	Schema           string
	Name             string
	UnderlyingType   string
	NotNull          bool
	Collation        sql.NullString
	Default          sql.NullString
	CheckConstraints sql.NullString
	dependencies     []string
}

func (d Domain) SortKey() string {
	return d.Name
}

func (d Domain) DependsOn() []string {
	return d.dependencies
}

func (d *Domain) AddDependency(dep string) {
	d.dependencies = append(d.dependencies, dep)
}

func (d Domain) String() string {
	def := fmt.Sprintf("CREATE DOMAIN %s AS %s", pgtools.Identifier(d.Schema, d.Name), d.UnderlyingType)
	if d.Collation.Valid {
		def = fmt.Sprintf("%s\nCOLLATE %s", def, pgtools.Identifier(d.Collation.String))
	}
	if d.Default.Valid {
		def = fmt.Sprintf("%s\nDEFAULT %s", def, d.Default.String)
	}
	if d.CheckConstraints.Valid {
		def = fmt.Sprintf("%s\n%s", def, d.CheckConstraints.String)
	}
	if d.NotNull {
		def = fmt.Sprintf("%s\nNOT NULL", def)
	}
	return def + ";"
}

func LoadDomains(config Config, db *sql.DB) ([]*Domain, error) {
	var domains []*Domain
	rows, err := db.Query(domainsQuery, config.Schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var domain Domain
		if err := rows.Scan(
			&domain.Schema,
			&domain.Name,
			&domain.UnderlyingType,
			&domain.NotNull,
			&domain.Collation,
			&domain.Default,
			&domain.CheckConstraints,
		); err != nil {
			return nil, err
		}
		domains = append(domains, &domain)
	}
	return Sort[string](domains), nil
}

// This query is inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
// - psql '\dD' with '\set ECHO_HIDDEN on'
// - Erwin Brandstetter's answer https://stackoverflow.com/a/68249694/829926
var domainsQuery = query(`--sql
select
	n.nspname as "schema",
	t.typname as "name",
	pg_catalog.format_type(t.typbasetype, t.typtypmod) as "underlying_type",
	t.typnotnull as "not_null",
	(
		select
			c.collname
		from pg_catalog.pg_collation c, pg_catalog.pg_type bt
		where
			c.oid = t.typcollation
			and bt.oid = t.typbasetype
			and t.typcollation <> bt.typcollation
	) as "collation",
	t.typdefault as "default",
	pg_catalog.array_to_string(array(
		select
			pg_catalog.pg_get_constraintdef(r.oid, true)
		from pg_catalog.pg_constraint r
		where t.oid = r.contypid
	), ' ') as "check_constraints"
from pg_catalog.pg_type t
left join pg_catalog.pg_namespace n
	on n.oid = t.typnamespace
where
	t.typtype = 'd'  -- domains
	and n.nspname = $1
	and n.nspname <> 'pg_catalog'
	and n.nspname <> 'information_schema'
	and pg_catalog.pg_type_is_visible(t.oid)
order by
	"schema",
	"name"
`)
