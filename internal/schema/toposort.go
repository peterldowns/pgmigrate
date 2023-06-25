package schema

import (
	"sort"

	"golang.org/x/exp/constraints"
)

type Sortable[K constraints.Ordered] interface {
	SortKey() K     // sorted by this, ascending
	DependsOn() []K // what they depend on
}

// Sort a a slice in-place by
//
// - DESC depth: nodes are grouped by depth, deepest first
// - ASC    key: nodes with the same depth are ordered by key, ascending
//
// with O(n log n + m) complexity for total number of nodes n and total number
// of edges m. So for the graph
//
//		X
//	    A -> E
//		A -> B -> C
//		A -> B -> D -> E
//		Y -> Z
//
// The result will always be
//
//	E, C, D, B, Z, A, X, Y
//	depth 3: E
//	depth 2: C, D
//	depth 1: B, Z
//	depth 0: A, X, Y
func Sort[K constraints.Ordered, T Sortable[K]](nodes []T) []T {
	// Prepare the initial state for a depth-first traversal off the graph to
	// find the max-length path for each node.
	state := &sortState[K, T]{
		deps:      make(map[K][]K, len(nodes)),
		depths:    make(map[K]int, len(nodes)),
		permanent: make(map[K]void, len(nodes)),
		temporary: make(map[K]void, len(nodes)),
		byKey:     make(map[K]T, len(nodes)),
		result:    make([]T, 0, len(nodes)),
	}
	// O(n)
	for _, obj := range nodes {
		key := obj.SortKey()
		state.byKey[key] = obj
		state.depths[key] = 0
	}
	// O(n log n) sort by SortKey() ASC
	// so that no matter the initial ordering, we're iterating
	// through the DFS nodes in the same order.
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].SortKey() < nodes[j].SortKey()
	})
	// O(n + m), visit each node and edge at most one time
	for {
		visited := 0
		for _, obj := range nodes {
			if _, ok := state.permanent[obj.SortKey()]; !ok {
				depth := state.depths[obj.SortKey()]
				visit(state, obj, depth)
				visited++
			}
		}
		if visited == 0 {
			break
		}
	}
	return state.result
}

type void struct{}

type sortState[K constraints.Ordered, T Sortable[K]] struct {
	deps      map[K][]K
	depths    map[K]int
	permanent map[K]void
	temporary map[K]void
	byKey     map[K]T
	result    []T
}

func visit[K constraints.Ordered, T Sortable[K]](state *sortState[K, T], node T, depth int) {
	key := node.SortKey()
	if state.depths[key] < depth {
		state.depths[key] = depth
	}
	if _, ok := state.permanent[key]; ok {
		return
	}
	if _, ok := state.temporary[key]; ok {
		// this is only true if the current walk includes a cycle; just ignore
		// it. If there are any unstable sorting results, this should be the
		// first place to look. May need to mark all strongly connected
		// components as having the same depth using Tarjan's algorithm.
		return
	}
	state.temporary[key] = void{}
	thisDeps := node.DependsOn()
	for _, childKey := range thisDeps {
		// nodes can have dependencies that aren't in the graph,
		// which we should ignore safely.
		if childNode, ok := state.byKey[childKey]; ok {
			visit(state, childNode, depth+1)
		}
	}
	delete(state.temporary, key)
	state.permanent[key] = void{}
	state.result = append(state.result, node)
}
