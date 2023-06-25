package schema_test

import (
	"testing"

	"github.com/peterldowns/testy/check"

	"github.com/peterldowns/pgmigrate/internal/schema"
)

func TestToposortWithCycles(t *testing.T) {
	t.Parallel()

	// a -> b -> c
	//           c -> d -> e
	//           c <- d <- e
	//           c <-----> e
	//                     e <- f
	//                     e -> z
	a := newSnode("a", "b")
	b := newSnode("b", "c")
	c := newSnode("c", "d", "e")
	d := newSnode("d", "e", "c")
	e := newSnode("e", "d", "c", "z")
	f := newSnode("f", "e")
	z := newSnode("z")
	initial := []snode{a, b, c, d, e, f, z}
	nodes := schema.Sort[string](initial)

	expected := []snode{z, e, d, c, b, a, f}
	if !check.Equal(t, asKeys(expected), asKeys(nodes)) {
		t.Log("expected:", expected)
		t.Log("  result:", nodes)
	}
}

func TestInitialOrderIndependent(t *testing.T) {
	t.Parallel()
	z := newSnode("z")
	x := newSnode("x")
	a := newSnode("a", "z")
	for _, initial := range [][]snode{
		{a, x, z},
		{a, z, x},
		{x, a, z},
		{x, z, a},
		{z, a, x},
		{z, x, a},
	} {
		nodes := schema.Sort[string](initial)
		expected := []snode{z, a, x}
		if !check.Equal(t, asKeys(expected), asKeys(nodes)) {
			t.Log(" initial:", initial)
			t.Log("expected:", expected)
			t.Log("  result:", nodes)
		}
	}
}

func TestComplicatedToposort(t *testing.T) {
	t.Parallel()

	// x
	// a -> e
	// a -> b -> c
	// a -> b -> d -> e
	// y -> z
	a := newSnode("a", "b", "e")
	b := newSnode("b", "c", "d")
	c := newSnode("c")
	d := newSnode("d", "e")
	e := newSnode("e")
	x := newSnode("x")
	y := newSnode("y", "z")
	z := newSnode("z")
	nodes := []snode{a, b, c, d, e, x, y, z}
	nodes = schema.Sort[string](nodes)

	// depth 3: E
	// depth 2: C, D
	// depth 1: B, Z
	// depth 0: A, X, Y
	expected := []snode{c, e, d, b, a, x, z, y}
	if !check.Equal(t, asKeys(expected), asKeys(nodes)) {
		t.Log("expected:", expected)
		t.Log("  result:", nodes)
	}
}

type snode struct {
	key  string
	deps []string
}

func (s snode) SortKey() string {
	return s.key
}

func (s snode) DependsOn() []string {
	return s.deps
}

func newSnode(key string, deps ...string) snode {
	return snode{key: key, deps: deps}
}

func asKeys(snodes []snode) []string {
	result := make([]string, 0, len(snodes))
	for _, snode := range snodes {
		result = append(result, snode.key)
	}
	return result
}
