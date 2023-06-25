package schema

import (
	"database/sql"
	"fmt"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Extension struct {
	OID          int
	Schema       string
	Name         string
	Version      string
	Description  string
	dependencies []string
}

func (e Extension) SortKey() string {
	return e.Name
}

func (e Extension) DependsOn() []string {
	return e.dependencies
}

func (e *Extension) AddDependency(dep string) {
	e.dependencies = append(e.dependencies, dep)
}

func (e Extension) String() string {
	def := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", pgtools.QuoteIdentifier(e.Name))
	return def
}

func LoadExtensions(config Config, db *sql.DB) ([]*Extension, error) {
	var extensions []*Extension
	// Query based on psql
	//
	//  \set ECHO_HIDDEN on
	//  \dx
	//
	// and djrobstep/schemainspect
	// https://github.com/djrobstep/schemainspect/blob/master/schemainspect/pg/sql/extensions.sql
	//
	// and pg_dump getExtensions
	// https://github.com/postgres/postgres/blob/9a2dbc614e6e47da3c49daacec106da32eba9467/src/bin/pg_dump/pg_dump.c#L5306
	rows, err := db.Query(extensionsQuery, config.Schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var extension Extension
		if err := rows.Scan(
			&extension.OID,
			&extension.Schema,
			&extension.Name,
			&extension.Version,
			&extension.Description,
		); err != nil {
			return nil, err
		}
		extensions = append(extensions, &extension)
	}
	return Sort[string](extensions), nil
}

var extensionsQuery = query(`--sql
SELECT
	e.oid as "oid"
	, n.nspname AS "schema"
	, e.extname AS "name"
	, e.extversion AS "version"
	, c.description AS "description"
FROM
	pg_catalog.pg_extension e
LEFT JOIN pg_catalog.pg_namespace n
	ON n.oid = e.extnamespace
LEFT JOIN pg_catalog.pg_description c
	ON c.objoid = e.oid
	AND c.classoid = 'pg_catalog.pg_extension'::pg_catalog.regclass
WHERE n.nspname = $1
ORDER BY 1;
`)
