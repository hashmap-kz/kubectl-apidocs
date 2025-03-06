package apidocs

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	focusColor   = tcell.ColorSteelBlue
	noFocusColor = tcell.ColorLightGray
)

// Helper function to reset all node colors
func resetNodeColors(node *tview.TreeNode) {
	if node == nil {
		return
	}
	data, err := extractTreeData(node)
	if err != nil {
		return
	}
	switch data.nodeType {
	case nodeTypeRoot:
		node.SetColor(tcell.ColorYellow)
	case nodeTypeGroup:
		node.SetColor(tcell.ColorGreen)
	case nodeTypeResource:
		node.SetColor(tcell.ColorSteelBlue)
	case nodeTypeField:
		node.SetColor(tcell.ColorLightGray)
	default:
		node.SetColor(tcell.ColorLightGray)
	}

	for _, child := range node.GetChildren() {
		resetNodeColors(child)
	}
}
