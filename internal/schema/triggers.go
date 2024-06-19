package schema

import (
	"database/sql"
)

type Trigger struct {
	OID          int
	Schema       string
	Name         string
	TableName    string
	Definition   string
	ProcSchema   string
	ProcName     string
	Enabled      string
	dependencies []string
}

func (t Trigger) SortKey() string {
	// Triggers on different tables may have the same name
	return identifier(t.TableName, t.Name)
}

func (t Trigger) DependsOn() []string {
	out := append(t.dependencies, t.TableName) //nolint:gocritic // appendAssign
	if t.ProcName != "" {
		out = append(out, t.ProcName)
	}
	return out
}

func (t *Trigger) AddDependency(dep string) {
	t.dependencies = append(t.dependencies, dep)
}

func (t Trigger) String() string {
	return t.Definition + ";"
}

func LoadTriggers(config Config, db *sql.DB) ([]*Trigger, error) {
	var triggers []*Trigger
	rows, err := db.Query(triggersQuery, config.Schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var trigger Trigger
		if err := rows.Scan(
			&trigger.OID,
			&trigger.Schema,
			&trigger.Name,
			&trigger.TableName,
			&trigger.Definition,
			&trigger.ProcSchema,
			&trigger.ProcName,
			&trigger.Enabled,
		); err != nil {
			return nil, err
		}
		triggers = append(triggers, &trigger)
	}
	return Sort[string](triggers), nil
}

// This query is inspired heavily by:
// - djrobstep/schemainspect https://github.com/djrobstep/schemainspect/tree/066262d6fb4668f874925305a0b7dbb3ac866882/schemainspect/pg/sql
// - psql '\d+ <table>' with '\set ECHO_HIDDEN on'
var triggersQuery = query(`--sql
with extensions as (
  select
      objid as "oid"
  from
      pg_depend d
  WHERE
     d.refclassid = 'pg_extension'::regclass and
     d.classid = 'pg_trigger'::regclass
)
select
	tg.oid as "oid",
    cls.relnamespace::regnamespace::text as "schema",
    tg.tgname "name",
    cls.relname as "table_name",
    pg_get_triggerdef(tg.oid) as  "definition",
    proc.pronamespace::regnamespace::text as "proc_schema",
    proc.proname as "proc_name",
    tg.tgenabled as "enabled"
from pg_trigger tg
join pg_class cls on cls.oid = tg.tgrelid
join pg_proc proc on proc.oid = tg.tgfoid
left outer join extensions e on tg.oid = e.oid
where
	not tg.tgisinternal
	and cls.relnamespace::regnamespace::text = $1
	and e.oid is null
order by
	"schema",
	"name", 
	"table_name"
`)
