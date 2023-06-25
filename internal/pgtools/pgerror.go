package pgtools

import "errors"

// Derived from jackc/pgconn, which is released under the MIT License.
// https://github.com/jackc/pgconn
//
// Copyright (c) 2019-2021 Jack Christensen
//
// MIT License
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// pgError represents an error reported by the PostgreSQL server. See
// http://www.postgresql.org/docs/11/static/protocol-error-fields.html for
// detailed field description.
type Error struct {
	Severity         string
	Code             string
	Message          string
	Detail           string
	Hint             string
	Position         int32
	InternalPosition int32
	InternalQuery    string
	Where            string
	SchemaName       string
	TableName        string
	ColumnName       string
	DataTypeName     string
	ConstraintName   string
	File             string
	Line             int32
	Routine          string
}

func (pe *Error) Error() string {
	return pe.Severity + ": " + pe.Message + " (SQLSTATE " + pe.Code + ")"
}

// If an error comes from postgres, return as much information as possible for
// logging purposes.
func ErrorData(err error) map[string]any {
	data := make(map[string]any)
	var perr *Error
	if errors.As(err, &perr) {
		data["pg_code"] = perr.Code
		if perr.Detail != "" {
			data["pg_detail"] = perr.Detail
		}
		if perr.Hint != "" {
			data["pg_hint"] = perr.Hint
		}
		if perr.SchemaName != "" {
			data["pg_schema"] = perr.SchemaName
		}
		if perr.TableName != "" {
			data["pg_table"] = perr.TableName
		}
		if perr.ColumnName != "" {
			data["pg_column"] = perr.ColumnName
		}
		if perr.ConstraintName != "" {
			data["pg_column"] = perr.ConstraintName
		}
		if perr.Where != "" {
			data["pg_where"] = perr.Where
		}
		if perr.Severity != "" {
			data["pg_severity"] = perr.Severity
		}
	}
	return data
}
