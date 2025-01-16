package apidocs

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Helper function to highlight matching nodes
func highlightMatchingNodes(root *tview.TreeNode, searchTerm string) {
	if searchTerm == "" {
		// Reset all nodes to their default color
		resetNodeColors(root)
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
	oldColor := node.GetColor()

	// Check if the node's text contains the search term
	if strings.Contains(strings.ToLower(node.GetText()), searchTerm) {
		node.SetColor(tcell.ColorRed) // Highlight matching node
	} else {
		node.SetColor(oldColor) // Reset non-matching nodes
	}

	// Recursively check all children
	for _, child := range node.GetChildren() {
		searchAndHighlight(child, searchTerm)
	}
}

// Helper function to reset all node colors
func resetNodeColors(node *tview.TreeNode) {
	if node == nil {
		return
	}

	node.SetColor(tcell.ColorWhite)

	for _, child := range node.GetChildren() {
		resetNodeColors(child)
	}
}
