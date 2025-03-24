package schema

import (
	"database/sql"
	"fmt"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Followup struct {
	Name         string
	SQL          string
	dependencies []string
}

func (f Followup) SortKey() string {
	return f.Name
}

func (f Followup) DependsOn() []string {
	return f.dependencies
}

func (f *Followup) AddDependency(dep string) {
	f.dependencies = append(f.dependencies, dep)
}

func (f Followup) String() string {
	return f.SQL
}

type Sequence struct {
	OID              int
	Schema           string
	Name             string
	DataType         string
	StartValue       int
	MinValue         int
	MaxValue         int
	IncrementBy      int
	Cache            int
	Cycle            bool
	TableName        sql.NullString
	ColumnName       sql.NullString
	IsIdentity       bool
	IsIdentityAlways bool
	dependencies     []string
}

func (s Sequence) SortKey() string {
	return pgtools.Identifier(s.Schema, s.Name)
}

func (s Sequence) DependsOn() []string {
	if s.TableName.Valid {
		return append(s.dependencies, s.TableName.String)
	}
	return s.dependencies
}

func (s *Sequence) AddDependency(dep string) {
	s.dependencies = append(s.dependencies, dep)
}

func (s Sequence) Followup() *Followup {
	if s.TableName.Valid && s.ColumnName.Valid {
		return &Followup{
			Name: s.Name,
			dependencies: []string{
				s.Name,
				s.TableName.String,
			},
			SQL: fmt.Sprintf(
				"ALTER SEQUENCE %s OWNED BY %s;",
				pgtools.Identifier(s.Schema, s.Name),
				pgtools.Identifier(s.Schema, s.TableName.String, s.ColumnName.String),
			),
		}
	}
	return nil
}

func (s Sequence) String() string {
	// TODO: StartValue, MinValue, MaxValue, etc.
	sName := pgtools.Identifier(s.Schema, s.Name)
	return fmt.Sprintf("CREATE SEQUENCE %s;", sName)
}

func LoadSequences(config Config, db *sql.DB) ([]*Sequence, error) {
	var sequences []*Sequence

	rows, err := db.Query(sequencesQuery, config.Schemas)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sequence Sequence
		if err := rows.Scan(
			&sequence.OID,
			&sequence.Schema,
			&sequence.Name,
			&sequence.DataType,
			&sequence.StartValue,
			&sequence.MinValue,
			&sequence.MaxValue,
			&sequence.IncrementBy,
			&sequence.Cache,
			&sequence.Cycle,
			&sequence.TableName,
			&sequence.ColumnName,
			&sequence.IsIdentity,
			&sequence.IsIdentityAlways,
		); err != nil {
			return nil, err
		}
		sequences = append(sequences, &sequence)
	}
	return Sort[string](sequences), nil
}

var sequencesQuery = query(`--sql
with sequences as (
    select
		c.oid as "oid",
        n.nspname as "schema",
        c.relname as "name",
		s.seqtypid::regtype::text as "data_type",
		s.seqstart as "start_value",
		s.seqmin as "min_value",
		s.seqmax as "max_value",
		s.seqincrement as "increment_by",
		s.seqcache as "cache",
		s.seqcycle as "cycle",
        c_ref.relname as "table_name",
        a.attname as "column_name",
        d.deptype is not distinct from 'i' as "is_identity",
        a.attidentity is not distinct from 'a' as "is_identity_always"
    from
        pg_class c
		inner join pg_sequence S
			ON c.oid = s.seqrelid
        inner join pg_catalog.pg_namespace n
            ON n.oid = c.relnamespace
        left join pg_depend d
            on c.oid = d.objid and d.deptype in ('i', 'a')
        left join pg_class c_ref
            on d.refobjid = c_ref.oid
        left join pg_attribute a
            ON ( a.attnum = d.refobjsubid
                AND a.attrelid = d.refobjid )
	where
		c.relkind = 'S'
		and n.nspname = ANY($1)
)
select
	"oid",
	"schema",
	"name",
	"data_type",
	"start_value",
	"min_value",
	"max_value",
	"increment_by",
	"cache",
	"cycle",
	"table_name",
	"column_name",
	"is_identity",
	"is_identity_always"
from sequences
order by "schema", "name"
`)

func RefFollowup(name string) string {
	return "12_followup." + name
}
