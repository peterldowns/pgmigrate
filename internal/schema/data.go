package schema

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

type Data struct {
	Schema  string
	Name    string
	Columns []string
	Data    []any
	OrderBy string
}

func (d Data) String() string {
	prelude := fmt.Sprintf("INSERT INTO %s (%s) VALUES\n", identifier(d.Schema, d.Name), strings.Join(d.Columns, ", "))
	rowLen := len(d.Columns)
	out := prelude
	for i := 0; i < len(d.Data); i += rowLen {
		rowValues := d.Data[i : i+rowLen]
		values := make([]string, 0, len(rowValues))
		for _, val := range rowValues {
			if val == nil {
				values = append(values, "null")
			} else {
				values = append(values, pgtools.QuoteLiteral(fmt.Sprintf("%v", val)))
			}
		}
		out += fmt.Sprintf("(%s)", strings.Join(values, ", "))
		if i != len(d.Data)-rowLen {
			out += ",\n"
		} else {
			out += "\n;"
		}
	}
	return out
}

func LoadData(config Config, db *sql.DB) ([]*Data, error) {
	var results []*Data
	for _, d := range config.Data {
		out := &Data{
			Schema:  config.Schema,
			Name:    d.Name,
			Columns: d.Columns,
			OrderBy: d.OrderBy,
			Data:    []any{},
		}
		cols := strings.Join(d.Columns, ", ")
		if len(cols) == 0 {
			cols = "*"
		}
		q := fmt.Sprintf(query(`--sql
select %s
from %s
		`), cols, identifier(config.Schema, d.Name))
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
			// if nullable, _ := cti.Nullable(); nullable {
			t = reflect.PointerTo(t)
			// }
			columnTypes = append(columnTypes, t)
			columns = append(columns, cti.Name())
		}
		out.Columns = columns

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
			out.Data = append(out.Data, ifaces...)
		}
		results = append(results, out)
	}
	return results, nil
}
