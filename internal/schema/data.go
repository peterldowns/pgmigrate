package schema

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Data struct {
	Schema       string   `yaml:"schema"`
	Name         string   `yaml:"name"`
	Columns      []string `yaml:"columns"`
	OrderBy      string   `yaml:"orderBy"`
	rows         []any
	dependencies []string
}

func (d Data) SortKey() string {
	return pgtools.Identifier(d.Schema, d.Name)
}

func (d *Data) AddDependency(dep string) {
	d.dependencies = append(d.dependencies, dep)
}

func (d Data) DependsOn() []string {
	return d.dependencies
}

// from pgx: https://github.com/jackc/pgtype/blob/6830cc09847cfe17ae59177e7f81b67312496108/timestamptz.go#L152
const pgTimestamptzSecondFormat = "2006-01-02 15:04:05.999999999Z07:00:00"

func tsToString(t time.Time) string {
	return t.Truncate(time.Microsecond).Format(pgTimestamptzSecondFormat)
}

func (d Data) String() string {
	if len(d.rows) == 0 || len(d.Columns) == 0 {
		return ""
	}
	prelude := fmt.Sprintf("INSERT INTO %s (%s) VALUES\n", pgtools.Identifier(d.Schema, d.Name), strings.Join(d.Columns, ", "))
	rowLen := len(d.Columns)
	out := prelude
	for i := 0; i < len(d.rows); i += rowLen {
		rowValues := d.rows[i : i+rowLen]
		values := make([]string, 0, len(rowValues))
		for _, val := range rowValues {
			if val == nil {
				values = append(values, "null")
				continue
			}
			var literal string
			switch tval := val.(type) {
			case time.Time:
				literal = tsToString(tval)
			case *time.Time:
				literal = tsToString(*tval)
			case string:
				literal = tval
			case *string:
				literal = *tval
			default:
				literal = fmt.Sprintf("%v", tval)
			}
			values = append(values, pgtools.Literal(literal))
		}
		out += fmt.Sprintf("(%s)", strings.Join(values, ", "))
		if i != len(d.rows)-rowLen {
			out += ",\n"
		} else {
			out += "\n;"
		}
	}
	return out
}

func LoadData(config DumpConfig, db *sql.DB) ([]*Data, error) {
	var toLoad []*Data
	for _, d := range config.Data {
		if strings.Contains(d.Name, "%") {
			rows, err := db.Query(query(`--sql
select
	c.relnamespace::text as schema_name,
	c.relname as name
from pg_catalog.pg_class c
where c.relnamespace::text = ANY($1)
and c.relkind in ('r', 't', 'p', 'm', 'v')
and c.relname like $2;
			`), config.SchemaNames, d.Name)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var schemaName string
				var name string
				if err := rows.Scan(&schemaName, &name); err != nil {
					return nil, err
				}
				toLoad = append(toLoad, &Data{
					Schema:  schemaName,
					Name:    name,
					Columns: d.Columns,
					OrderBy: d.OrderBy,
					rows:    []any{},
				})
			}
		} else {
			toLoad = append(toLoad, &Data{
				Schema:  d.Schema,
				Name:    d.Name,
				Columns: d.Columns,
				OrderBy: d.OrderBy,
				rows:    []any{},
			})
		}
	}
	for _, d := range toLoad {
		cols := strings.Join(d.Columns, ", ")
		if len(cols) == 0 {
			cols = "*"
		}
		q := fmt.Sprintf(query(`--sql
select %s
from %s
		`), cols, pgtools.Identifier(d.Schema, d.Name))
		if d.OrderBy != "" {
			q += "\norder by " + d.OrderBy
		}
		rows, err := db.Query(q)
		if err != nil {
			return nil, err
		}
		columnTypeInfo, err := rows.ColumnTypes()
		if err != nil {
			return nil, err
		}
		var columnTypes []reflect.Type
		var columns []string
		for _, cti := range columnTypeInfo {
			t := cti.ScanType()
			t = reflect.PointerTo(t)
			columnTypes = append(columnTypes, t)
			columns = append(columns, cti.Name())
		}
		d.Columns = columns

		for rows.Next() {
			scans := make([]any, len(columnTypes))
			values := make([]reflect.Value, len(columnTypes))
			for i := range columnTypes {
				valuePtr := reflect.New(columnTypes[i])
				scans[i] = valuePtr.Interface()
				values[i] = valuePtr.Elem()
			}
			if err := rows.Scan(scans...); err != nil {
				return nil, fmt.Errorf("scan failure: %w", err)
			}
			ifaces := make([]any, len(values))
			for i := range values {
				v := values[i]
				if v.IsNil() {
					ifaces[i] = nil
				} else {
					ifaces[i] = v.Elem().Interface()
				}
			}
			d.rows = append(d.rows, ifaces...)
		}
	}
	return Sort[string](toLoad), nil
}
