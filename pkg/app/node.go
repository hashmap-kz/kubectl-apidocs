package app

import "strings"

type Node struct {
	Name         string
	Children     map[string]*Node
	OriginalPath string
}

func NewTreeNode() *Node {
	return &Node{
		Children: make(map[string]*Node),
	}
}

func (node *Node) AddPath(path string) {
	parts := strings.Split(path, ".")
	current := node
	for i, part := range parts {

		// Ensure the child exists
		if current.Children == nil {
			current.Children = make(map[string]*Node)
		}

		if _, exists := current.Children[part]; !exists {
			current.Children[part] = NewTreeNode()
			current.Children[part].Name = part
		}
		current = current.Children[part]
		if current.OriginalPath == "" {
			current.OriginalPath = strings.Join(parts[:i+1], ".")
		}
	}
}
