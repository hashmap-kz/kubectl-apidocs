package apidocs

import "strings"

// ResourceFieldsNode used for construct a tree-structure from 'sts.metadata.name' paths
type ResourceFieldsNode struct {
	Name     string
	Children map[string]*ResourceFieldsNode
	Path     string
}

func NewResourceFieldsNode() *ResourceFieldsNode {
	return &ResourceFieldsNode{
		Children: make(map[string]*ResourceFieldsNode),
	}
}

func (node *ResourceFieldsNode) AddPath(path string) {
	parts := strings.Split(path, ".")
	current := node
	for i, part := range parts {
		// Ensure the child exists
		if current.Children == nil {
			current.Children = make(map[string]*ResourceFieldsNode)
		}

		if _, exists := current.Children[part]; !exists {
			current.Children[part] = NewResourceFieldsNode()
			current.Children[part].Name = part
		}
		current = current.Children[part]
		if current.Path == "" {
			current.Path = strings.Join(parts[:i+1], ".")
		}
	}
}
