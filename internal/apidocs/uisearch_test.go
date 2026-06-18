package apidocs

import (
	"sync"
	"testing"

	"github.com/rivo/tview"
)

func TestBuildFilteredTree_ResourceMatchKeepsFullSubtree(t *testing.T) {
	root := tviewNode("API Resources >", &TreeData{nodeType: nodeTypeRoot})
	group := tviewNode("apps/v1", &TreeData{nodeType: nodeTypeGroup})
	resource := tviewNode("Deployment (deployments)", &TreeData{
		nodeType: nodeTypeResource,
		path:     "Deployment",
	})
	fieldA := tviewNode("metadata", &TreeData{nodeType: nodeTypeField, path: "Deployment.metadata"})
	fieldB := tviewNode("spec", &TreeData{nodeType: nodeTypeField, path: "Deployment.spec"})

	resource.AddChild(fieldA)
	resource.AddChild(fieldB)
	group.AddChild(resource)
	root.AddChild(group)

	filtered := buildFilteredTree(root, "deployment")
	if filtered == nil {
		t.Fatal("expected filtered tree")
	}

	filteredGroup := filtered.GetChildren()[0]
	filteredResource := filteredGroup.GetChildren()[0]
	if len(filteredResource.GetChildren()) != 2 {
		t.Fatalf("expected full resource subtree, got %d children", len(filteredResource.GetChildren()))
	}
	if filteredResource.IsExpanded() {
		t.Fatal("expected matched resource node to stay collapsed")
	}
	if filteredResource.GetChildren()[0].IsExpanded() {
		t.Fatal("expected resource field subtree to stay collapsed")
	}
}

func TestBuildFilteredTree_FieldMatchKeepsOnlyMatchingBranch(t *testing.T) {
	root := tviewNode("API Resources >", &TreeData{nodeType: nodeTypeRoot})
	group := tviewNode("apps/v1", &TreeData{nodeType: nodeTypeGroup})
	resource := tviewNode("Deployment (deployments)", &TreeData{
		nodeType: nodeTypeResource,
		path:     "Deployment",
	})
	fieldA := tviewNode("metadata", &TreeData{nodeType: nodeTypeField, path: "Deployment.metadata"})
	fieldB := tviewNode("spec", &TreeData{nodeType: nodeTypeField, path: "Deployment.spec"})

	resource.AddChild(fieldA)
	resource.AddChild(fieldB)
	group.AddChild(resource)
	root.AddChild(group)

	filtered := buildFilteredTree(root, "metadata")
	if filtered == nil {
		t.Fatal("expected filtered tree")
	}

	filteredGroup := filtered.GetChildren()[0]
	filteredResource := filteredGroup.GetChildren()[0]
	if len(filteredResource.GetChildren()) != 1 {
		t.Fatalf("expected only matching branch, got %d children", len(filteredResource.GetChildren()))
	}
	if filteredResource.GetChildren()[0].GetText() != "metadata" {
		t.Fatalf("expected metadata branch, got %q", filteredResource.GetChildren()[0].GetText())
	}
}

func TestGetSearchRoot_UsesCurrentTreeRoot(t *testing.T) {
	globalRoot := tviewNode("API Resources >", &TreeData{nodeType: nodeTypeRoot})
	group := tviewNode("apps/v1", &TreeData{nodeType: nodeTypeGroup})
	resource := tviewNode("Deployment (deployments)", &TreeData{nodeType: nodeTypeResource, path: "Deployment"})
	globalRoot.AddChild(group)
	group.AddChild(resource)

	treeView := tview.NewTreeView().SetRoot(resource)
	uiState := &UIState{
		apiResourcesRootNode: globalRoot,
		apiResourcesTreeView: treeView,
		explainCache:         &sync.Map{},
	}

	got := getSearchRoot(uiState, treeView)
	if got != resource {
		t.Fatalf("expected current tree root, got %q", got.GetText())
	}
}

func TestGetSearchRoot_FallsBackToGlobalRoot(t *testing.T) {
	globalRoot := tviewNode("API Resources >", &TreeData{nodeType: nodeTypeRoot})
	uiState := &UIState{
		apiResourcesRootNode: globalRoot,
		explainCache:         &sync.Map{},
	}

	got := getSearchRoot(uiState, nil)
	if got != globalRoot {
		t.Fatal("expected fallback to global root")
	}
}

func tviewNode(text string, data *TreeData) *tview.TreeNode {
	return tview.NewTreeNode(text).SetReference(data)
}
