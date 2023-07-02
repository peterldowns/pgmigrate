package schema

import (
	"sort"

	"golang.org/x/exp/constraints"
)

type Sortable[K constraints.Ordered] interface {
	SortKey() K     // sorted by this, ascending
	DependsOn() []K // what they depend on
}

// Sort a a slice in-place by name, and then return a new slice that is sorted
// in dependency order. The initial sort makes the dependency-ordered result
// stable regardless of the initial ordering of the input slice.
func Sort[K constraints.Ordered, T Sortable[K]](nodes []T) []T {
	// Prepare the initial state for a depth-first traversal off the graph to
	// find the max-length path for each node.
	state := &sortState[K, T]{
		permanent: make(map[K]void, len(nodes)),
		temporary: make(map[K]void, len(nodes)),
		byKey:     make(map[K]T, len(nodes)),
		result:    make([]T, 0, len(nodes)),
	}
	// O(n)
	for _, obj := range nodes {
		key := obj.SortKey()
		state.byKey[key] = obj
	}
	// O(n log n) sort by SortKey() ASC
	// so that no matter the initial ordering, the DFS visits the nodes in the
	// same order.
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].SortKey() < nodes[j].SortKey()
	})
	// O(n + m), visit each node and edge at most one time
	for {
		visited := 0
		for _, obj := range nodes {
			if _, ok := state.permanent[obj.SortKey()]; !ok {
				visit(state, obj)
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
	permanent map[K]void
	temporary map[K]void
	byKey     map[K]T
	result    []T
}

func visit[K constraints.Ordered, T Sortable[K]](state *sortState[K, T], node T) {
	key := node.SortKey()
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
		// Ignore dependencies that aren't in the graph.
		if childNode, ok := state.byKey[childKey]; ok {
			visit(state, childNode)
		}
	}
	delete(state.temporary, key)
	state.permanent[key] = void{}
	state.result = append(state.result, node)
}
