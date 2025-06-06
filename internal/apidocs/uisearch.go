package apidocs

import (
	"strings"

	"github.com/rivo/tview"
)

func buildFilteredTree(node *tview.TreeNode, searchTerm string) *tview.TreeNode {
	if node == nil {
		return nil
	}

	data, err := extractTreeData(node)
	if err != nil {
		return nil
	}

	var matched bool
	shouldHighlight := data.IsNodeType(nodeTypeResource, nodeTypeField)
	if strings.Contains(strings.ToLower(node.GetText()), searchTerm) && shouldHighlight {
		matched = true
	}

	// Recursively process children
	var matchingChildren []*tview.TreeNode
	for _, child := range node.GetChildren() {
		filteredChild := buildFilteredTree(child, searchTerm)
		if filteredChild != nil {
			matchingChildren = append(matchingChildren, filteredChild)
			matched = true // parent will be included if child matched
		}
	}

	if matched {
		// Clone the current node
		newNode := tview.NewTreeNode(node.GetText()).
			SetReference(node.GetReference()).
			SetExpanded(true) // Expand filtered nodes so user can see them

		// Add matching children
		for _, child := range matchingChildren {
			newNode.AddChild(child)
		}

		return newNode
	}

	// If neither this node nor any child matched -> omit
	return nil
}

func showFilteredTree(uiState *UIState, treeView *tview.TreeView, searchTerm string) {
	if searchTerm == "" {
		// Show full tree again
		treeView.SetRoot(uiState.apiResourcesRootNode).
			SetCurrentNode(uiState.apiResourcesRootNode)
		return
	}

	filteredRoot := buildFilteredTree(uiState.apiResourcesRootNode, strings.ToLower(searchTerm))
	if filteredRoot == nil {
		// Nothing matched -> empty root
		filteredRoot = tview.NewTreeNode("(no matches)")
	}

	resetNodeColors(filteredRoot)
	uiState.isInFilter = true
	treeView.SetRoot(filteredRoot).SetCurrentNode(filteredRoot)
}
