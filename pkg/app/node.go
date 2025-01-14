package app

import "sort"

// Node represents a node in the tree
type Node struct {
	Name     string           `json:"name"`
	Children map[string]*Node `json:"children,omitempty"`
}

// SortChildren recursively sorts the children of the node
func (n *Node) SortChildren() {
	if n.Children == nil {
		return
	}

	// Extract and sort the keys
	keys := make([]string, 0, len(n.Children))
	for key := range n.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create a new sorted map
	sortedChildren := make(map[string]*Node, len(n.Children))
	for _, key := range keys {
		child := n.Children[key]
		// Recursively sort the children of each child
		child.SortChildren()
		sortedChildren[key] = child
	}

	// Replace the old map with the sorted one
	n.Children = sortedChildren
}

// AddPath adds a path to the tree
func (n *Node) AddPath(path []string) {
	if len(path) == 0 {
		return
	}

	// Get the current level key
	key := path[0]

	// Ensure the child exists
	if n.Children == nil {
		n.Children = make(map[string]*Node)
	}
	if _, exists := n.Children[key]; !exists {
		n.Children[key] = &Node{Name: key}
	}

	// Recurse to add the rest of the path
	n.Children[key].AddPath(path[1:])
}
