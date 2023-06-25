package schema

import (
	"fmt"
	"strings"

	"golang.org/x/exp/constraints"

	"github.com/peterldowns/pgmigrate/internal/pgtools"
)

func identifier(schema, table string) string {
	return identifiers(schema, table)
}

func identifiers(things ...string) string {
	out := make([]string, 0, len(things))
	for _, s := range things {
		lowered := strings.ToLower(s)
		if lowered == s {
			out = append(out, s)
		} else {
			out = append(out, pgtools.QuoteIdentifier(s))
		}
	}
	return strings.Join(out, ".")
}

func query(x string) string {
	return strings.TrimSpace(strings.TrimPrefix(x, "--sql"))
}

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

type DBObject interface {
	Printable
	Sortable[string]
	Dependable[string]
}

type Printable interface {
	String() string
}
type Dependable[K constraints.Ordered] interface {
	AddDependency(K)
	DependsOn() []K
}

func RefExtension(name string) string {
	return name
}

func RefDomain(name string) string {
	return name
}

func RefCompoundType(name string) string {
	return name
}

func RefEnum(name string) string {
	return name
}

func RefFunction(name string) string {
	return name
}

func RefTable(name string) string {
	return name
}

func RefView(name string) string {
	return name
}

func RefSequence(name string) string {
	return name
}

func RefIndex(name string) string {
	return name
}

func RefConstraint(name string) string {
	return name
}

func RefTrigger(name string) string {
	return name
}

func ParseRef(raw string) (string, error) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) == 2 {
		switch parts[0] {
		case "extension":
			return RefExtension(parts[1]), nil
		case "domain":
			return RefDomain(parts[1]), nil
		case "type":
			return RefCompoundType(parts[1]), nil
		case "enum":
			return RefEnum(parts[1]), nil
		case "table":
			return RefTable(parts[1]), nil
		case "view":
			return RefView(parts[1]), nil
		case "sequence":
			return RefSequence(parts[1]), nil
		case "index":
			return RefIndex(parts[1]), nil
		case "constraint":
			return RefConstraint(parts[1]), nil
		case "trigger":
			return RefTrigger(parts[1]), nil
		}
	}
	return "", fmt.Errorf("invalid reference: %s", raw)
}
