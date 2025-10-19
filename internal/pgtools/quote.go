package pgtools

import (
	"fmt"
	"strings"
)

// Literal and Identifier contains are derived almost exactly from
// lib/pq, which is released under the MIT License.
// https://github.com/lib/pq
//
// Copyright (c) 2011-2013, 'pq' Contributors Portions Copyright (C) 2011 Blake
// Mizerany
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Literal quotes a 'literal' (e.g. a parameter, often used to pass literal to
// DDL and other statements that do not accept parameters) to be used as part of
// an SQL statement.
//
// Any single quotes in name will be escaped. Any backslashes (i.e. "\") will be
// replaced by two backslashes (i.e. "\\") and the C-style escape identifier
// that PostgreSQL provides ('E') will be prepended to the string.
func Literal(literal string) string {
	// This follows the PostgreSQL internal algorithm for handling quoted literals
	// from libpq, which can be found in the "PQEscapeStringInternal" function,
	// which is found in the libpq/fe-exec.c source file:
	// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/interfaces/libpq/fe-exec.c
	//
	// substitute any single-quotes (') with two single-quotes ('')
	literal = strings.ReplaceAll(literal, `'`, `''`)
	// determine if the string has any backslashes (\) in it.
	// if it does, replace any backslashes (\) with two backslashes (\\)
	// then, we need to wrap the entire string with a PostgreSQL
	// C-style escape. Per how "PQEscapeStringInternal" handles this case, we
	// also add a space before the "E"
	if strings.Contains(literal, `\`) {
		literal = strings.ReplaceAll(literal, `\`, `\\`)
		literal = ` E'` + literal + `'`
	} else {
		literal = `'` + literal + `'`
	}
	return literal
}

// Identifier quotes an identifier (a name of an object — a table, a column, a
// function, a type, a schema, etc.) for use in a DDL statement defining or
// referencing that object. It will return the same identifier if possible, only
// introducing quotes or modifications when:
//
//   - the identifier has an upper-case character
//   - the identifier has a hyphen
//   - the identifier is a reserved keyword in PostgreSQL, or is non-reserved
//     but requires quoting in some contexts (when used as a column name, used as
//     a type name, or used as a function name)
//
// For convenience, Identifier allows you to pass the parts of a fully-qualified
// "dotted" identifier, or a single un-split dotted identifier.
func Identifier(parts ...string) string {
	if len(parts) == 1 {
		parts = strings.Split(parts[0], ".")
	}
	out := make([]string, 0, len(parts))
	for _, identifier := range parts {
		if requiresQuoting(identifier) {
			identifier = fmt.Sprintf(`"%s"`, strings.ReplaceAll(identifier, `"`, `""`))
		}
		out = append(out, identifier)
	}
	return strings.Join(out, ".")
}

func requiresQuoting(identifier string) bool {
	lowered := strings.ToLower(identifier)
	if lowered != identifier {
		return true
	}
	if _, ok := postgresKeywords[lowered]; ok {
		return true
	}
	if strings.ContainsRune(lowered, '"') {
		return true
	}
	if strings.ContainsRune(lowered, '-') {
		return true
	}
	return false
}
