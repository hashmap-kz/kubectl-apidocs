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

	// Check if the node's text contains the search term
	if strings.Contains(strings.ToLower(node.GetText()), searchTerm) {
		node.SetColor(tcell.ColorRed) // Highlight matching node
	} else {
		resetNodeColors(node)
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
	data, err := extractTreeData(node)
	if err != nil {
		return
	}
	nodeType := data.nodeType
	switch nodeType {
	case nodeTypeRoot:
		node.SetColor(tcell.ColorYellow)
	case nodeTypeGroup:
		node.SetColor(tcell.ColorGreen)
	case nodeTypeResource:
		node.SetColor(tcell.ColorBlue)
	case nodeTypeField:
		if len(node.GetChildren()) > 0 {
			node.SetColor(tcell.ColorGreen)
		} else {
			node.SetColor(tcell.ColorWhite)
		}
	default:
		node.SetColor(tcell.ColorLightGray)
	}

	for _, child := range node.GetChildren() {
		resetNodeColors(child)
	}
}
