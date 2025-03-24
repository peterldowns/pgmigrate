package schema

import (
	"strings"

	"golang.org/x/exp/constraints"
)

// DBObject is an interface satisifed by [Table], [View], [Enum], etc.
// and allows for easier interaction during the sorting and printing parts of
// the code.
type DBObject interface {
	SortKey() string
	String() string
	AddDependency(string)
	DependsOn() []string
}

// query is a helper for writing sql queries that look nice in vscode when using
// the "Inline SQL for go" extension by @jhnj, which gives syntax highlighting
// for strings that begin with `--sql`.
//
// https://marketplace.visualstudio.com/items?itemName=jhnj.vscode-go-inline-sql
func query(x string) string {
	return strings.TrimSpace(strings.TrimPrefix(x, "--sql"))
}

// asMap turns a slice of objects into a map of objects keyed by their
// SortKey().
func asMap[K constraints.Ordered, T Sortable[K]](collections ...[]T) map[K]T {
	total := 0
	for _, obj := range collections {
		total += len(obj)
	}
	out := make(map[K]T, total)
	for _, collection := range collections {
		for _, object := range collection {
			out[object.SortKey()] = object
		}
	}
	return out
}
