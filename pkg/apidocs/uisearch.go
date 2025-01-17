package apidocs

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Helper function to highlight matching nodes
func highlightMatchingNodes(uiState *UIState, root *tview.TreeNode, searchTerm string) {
	if searchTerm == "" {
		// Reset all nodes to their default color
		resetNodeColors(uiState.apiResourcesRootNode)
		return
	}

	// Recursively search and highlight nodes
	searchAndHighlight(root, strings.ToLower(searchTerm))
}

// Recursive function to search and highlight nodes
func searchAndHighlight(node *tview.TreeNode, searchTerm string) {
	if node == nil {
		return
	}
	data, err := extractTreeData(node)
	if err != nil {
		return
	}
	// Check if the node's text contains the search term
	shouldHighlight := data.IsNodeType(nodeTypeResource, nodeTypeField)
	if strings.Contains(strings.ToLower(node.GetText()), searchTerm) && shouldHighlight {
		node.SetColor(tcell.ColorRed) // Highlight matching node
		// TODO: set current node properly here
		// ...
	} else {
		resetNodeColors(node)
	}

	// Recursively check all children
	for _, child := range node.GetChildren() {
		searchAndHighlight(child, searchTerm)
	}
}
