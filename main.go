package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// runKubectlExplain runs the "kubectl explain" command for a given resource/field.
func runKubectlExplain(field string) (string, error) {
	cmd := exec.Command("kubectl", "explain", field)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

// parseExplainOutput parses the output of "kubectl explain" to extract subfields and descriptions.
func parseExplainOutput(output string) []struct {
	Field       string
	Description string
} {
	lines := strings.Split(output, "\n")
	var fields []struct {
		Field       string
		Description string
	}
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FIELD:") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				description := ""
				// Look for description in subsequent lines.
				if i+1 < len(lines) {
					description = strings.TrimSpace(lines[i+1])
				}
				fields = append(fields, struct {
					Field       string
					Description string
				}{
					Field:       parts[1],
					Description: description,
				})
			}
		}
	}
	return fields
}

// Recursively expands all nodes in the tree.
func expandAll(node *tview.TreeNode) {
	node.SetExpanded(true)
	for _, child := range node.GetChildren() {
		expandAll(child)
	}
}

// Recursively collapses all nodes in the tree.
func collapseAll(node *tview.TreeNode) {
	node.SetExpanded(false)
	for _, child := range node.GetChildren() {
		collapseAll(child)
	}
}

// Recursively writes the tree structure to a file.
func writeTreeToFile(node *tview.TreeNode, indent string, file *os.File) {
	file.WriteString(fmt.Sprintf("%s%s\n", indent, node.GetText()))
	for _, child := range node.GetChildren() {
		writeTreeToFile(child, indent+"  ", file)
	}
}

// Searches for a node by name and selects it.
func searchTree(query string, node *tview.TreeNode, tree *tview.TreeView) bool {
	for _, child := range node.GetChildren() {
		if strings.Contains(strings.ToLower(child.GetText()), strings.ToLower(query)) {
			tree.SetCurrentNode(child)
			return true
		}
		if searchTree(query, child, tree) {
			return true
		}
	}
	return false
}

func main() {
	app := tview.NewApplication()

	// Create the root node for the Kubernetes resource.
	rootField := "pod"
	root := tview.NewTreeNode(rootField).
		SetColor(tcell.ColorRed).
		SetReference(rootField)

	// Create a TreeView.
	tree := tview.NewTreeView()
	tree.SetRoot(root)
	tree.SetCurrentNode(root)
	tree.SetBorder(true)
	tree.SetTitle("Kubernetes Explain Tree")
	tree.SetBorderColor(tcell.ColorBlue)

	// Create a TextView to display field details.
	detailsView := tview.NewTextView()
	detailsView.SetDynamicColors(true)
	detailsView.SetBorder(true)
	detailsView.SetTitle("Field Details")

	// Create a search input field.
	searchInput := tview.NewInputField().
		SetLabel("Search: ").
		SetFieldWidth(30)

	// Handle node selection to dynamically load children and display details.
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		field := node.GetReference().(string)

		// Fetch detailed information about the field.
		output, err := runKubectlExplain(field)
		if err != nil {
			detailsView.SetText(fmt.Sprintf("[red]Error: %v", err))
			return
		}

		// Display detailed field documentation.
		detailsView.SetText(output)

		// Load children dynamically if not already loaded.
		if len(node.GetChildren()) == 0 {
			subfields := parseExplainOutput(output)
			if len(subfields) == 0 {
				node.AddChild(tview.NewTreeNode("No subfields").SetColor(tcell.ColorGray))
			} else {
				for _, subfield := range subfields {
					childNode := tview.NewTreeNode(fmt.Sprintf("%s - %s", subfield.Field, subfield.Description)).
						SetReference(fmt.Sprintf("%s.%s", field, subfield.Field)).
						SetColor(tcell.ColorGreen)
					node.AddChild(childNode)
				}
			}
		}

		// Toggle expanded state.
		node.SetExpanded(!node.IsExpanded())
	})

	// Bind search functionality to the search input field.
	searchInput.SetDoneFunc(func(key tcell.Key) {
		query := searchInput.GetText()
		if query != "" {
			if !searchTree(query, tree.GetRoot(), tree) {
				detailsView.SetText("[red]Field not found.")
			}
		}
	})

	// Create a layout to arrange the UI components.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(searchInput, 3, 1, false).
		AddItem(
			tview.NewFlex().
				AddItem(tree, 0, 1, true).
				AddItem(detailsView, 0, 1, false),
			0, 1, true,
		)

	// Handle keyboard shortcuts.
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlE: // Expand all nodes
			expandAll(tree.GetRoot())
		case tcell.KeyCtrlC: // Collapse all nodes
			collapseAll(tree.GetRoot())
		case tcell.KeyCtrlS: // Save tree structure to file
			file, err := os.Create("tree_structure.txt")
			if err != nil {
				detailsView.SetText(fmt.Sprintf("[red]Error: %v", err))
				return nil
			}
			defer file.Close()
			writeTreeToFile(tree.GetRoot(), "", file)
			detailsView.SetText("[green]Tree structure saved to tree_structure.txt")
		case tcell.KeyCtrlH: // Show help modal
			modal := tview.NewModal().
				SetText("Shortcuts:\nCtrl+E: Expand All\nCtrl+C: Collapse All\nCtrl+S: Save Tree\nCtrl+H: Show Help\nESC: Close Help").
				AddButtons([]string{"Close"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(layout, true)
				})
			app.SetRoot(modal, true)
		}
		return event
	})

	// Set up the app and start it.
	if err := app.SetRoot(layout, true).Run(); err != nil {
		panic(err)
	}
}
