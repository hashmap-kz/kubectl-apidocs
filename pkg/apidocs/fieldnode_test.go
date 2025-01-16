package apidocs

import (
	"testing"
)

func TestNewResourceFieldsNode(t *testing.T) {
	node := NewResourceFieldsNode()

	if node == nil {
		t.Fatal("NewResourceFieldsNode() returned nil")
	}

	if node.Children == nil || len(node.Children) != 0 {
		t.Fatalf("Expected empty Children map, got %+v", node.Children)
	}

	if node.Name != "" {
		t.Fatalf("Expected empty Name, got %s", node.Name)
	}

	if node.Path != "" {
		t.Fatalf("Expected empty Path, got %s", node.Path)
	}
}

func TestAddPath_SingleLevel(t *testing.T) {
	node := NewResourceFieldsNode()
	node.AddPath("name")

	if len(node.Children) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(node.Children))
	}

	child, exists := node.Children["name"]
	if !exists {
		t.Fatal("Expected child 'name' to exist")
	}

	if child.Name != "name" {
		t.Fatalf("Expected child.Name to be 'name', got %s", child.Name)
	}

	if child.Path != "name" {
		t.Fatalf("Expected child.Path to be 'name', got %s", child.Path)
	}
}

func TestAddPath_MultiLevel(t *testing.T) {
	node := NewResourceFieldsNode()
	node.AddPath("metadata.name")

	if len(node.Children) != 1 {
		t.Fatalf("Expected 1 child, got %d", len(node.Children))
	}

	metadata, exists := node.Children["metadata"]
	if !exists {
		t.Fatal("Expected child 'metadata' to exist")
	}

	if metadata.Name != "metadata" {
		t.Fatalf("Expected metadata.Name to be 'metadata', got %s", metadata.Name)
	}

	if metadata.Path != "metadata" {
		t.Fatalf("Expected metadata.Path to be 'metadata', got %s", metadata.Path)
	}

	if len(metadata.Children) != 1 {
		t.Fatalf("Expected 1 child under 'metadata', got %d", len(metadata.Children))
	}

	name, exists := metadata.Children["name"]
	if !exists {
		t.Fatal("Expected child 'name' under 'metadata' to exist")
	}

	if name.Name != "name" {
		t.Fatalf("Expected name.Name to be 'name', got %s", name.Name)
	}

	if name.Path != "metadata.name" {
		t.Fatalf("Expected name.Path to be 'metadata.name', got %s", name.Path)
	}
}

func TestAddPath_OverlappingPaths(t *testing.T) {
	node := NewResourceFieldsNode()
	node.AddPath("metadata.name")
	node.AddPath("metadata.labels")

	metadata, exists := node.Children["metadata"]
	if !exists {
		t.Fatal("Expected child 'metadata' to exist")
	}

	if len(metadata.Children) != 2 {
		t.Fatalf("Expected 2 children under 'metadata', got %d", len(metadata.Children))
	}

	name, exists := metadata.Children["name"]
	if !exists {
		t.Fatal("Expected child 'name' under 'metadata' to exist")
	}

	if name.Path != "metadata.name" {
		t.Fatalf("Expected name.Path to be 'metadata.name', got %s", name.Path)
	}

	labels, exists := metadata.Children["labels"]
	if !exists {
		t.Fatal("Expected child 'labels' under 'metadata' to exist")
	}

	if labels.Path != "metadata.labels" {
		t.Fatalf("Expected labels.Path to be 'metadata.labels', got %s", labels.Path)
	}
}

func TestAddPath_NestedPathWithExistingRoot(t *testing.T) {
	node := NewResourceFieldsNode()
	node.AddPath("metadata")
	node.AddPath("metadata.name")

	metadata, exists := node.Children["metadata"]
	if !exists {
		t.Fatal("Expected child 'metadata' to exist")
	}

	if len(metadata.Children) != 1 {
		t.Fatalf("Expected 1 child under 'metadata', got %d", len(metadata.Children))
	}

	name, exists := metadata.Children["name"]
	if !exists {
		t.Fatal("Expected child 'name' under 'metadata' to exist")
	}

	if name.Path != "metadata.name" {
		t.Fatalf("Expected name.Path to be 'metadata.name', got %s", name.Path)
	}
}

func TestAddPath_EmptyPath(t *testing.T) {
	node := NewResourceFieldsNode()
	node.AddPath("")

	if len(node.Children) != 0 {
		t.Fatalf("Expected no children for empty path, got %d", len(node.Children))
	}
}
