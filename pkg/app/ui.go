package app

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/gdamore/tcell/v2"

	"github.com/rivo/tview"
)

func App(root *Node, pathExplainers map[string]Explainer) {
	/////// UI ///////

	// Create the tree view
	// Create the root tree node
	rootTree := tview.NewTreeNode(root.Name).SetColor(tview.Styles.PrimitiveBackgroundColor).SetExpanded(true)

	// Recursive function to add children
	var addChildren func(parent *tview.TreeNode, children map[string]*Node)
	addChildren = func(parent *tview.TreeNode, children map[string]*Node) {
		if len(children) != 0 {
			parent.SetColor(tcell.ColorGreen)
			parent.SetExpanded(!parent.IsExpanded())
		}

		keys := make([]string, 0, len(children))
		for key := range children {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			childNode := tview.NewTreeNode(children[key].Name).SetReference(children[key])
			parent.AddChild(childNode)
			if children[key].Children != nil {
				addChildren(childNode, children[key].Children)
			}
		}
	}

	// Add children to the root
	addChildren(rootTree, root.Children)

	// Create the TreeView
	tree := tview.NewTreeView().
		SetRoot(rootTree).
		SetCurrentNode(rootTree)

	tree.SetBorder(true)
	tree.SetTitle("Resources")

	rootTree.SetExpanded(true)

	// Create a TextView to display field details.
	detailsView := tview.NewTextView()
	detailsView.SetDynamicColors(true)
	detailsView.SetBorder(true)
	detailsView.SetTitle("Field Details")
	detailsView.SetScrollable(true)
	detailsView.SetWrap(true)

	// Stack to handle navigation back
	var stack []*tview.TreeNode
	stack = append(stack, rootTree)

	// Add key event handler for toggling node expansion
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}

		// open subview with a subtree
		if len(node.GetChildren()) > 0 &&
			(node.GetText() == "gateways" || node.GetText() == "statefulsets") { // Subtree with children
			stack = append(stack, node)
			tree.SetRoot(node).
				SetCurrentNode(node)

			node.SetExpanded(true)
		} else {
			// just expand subtree
			node.SetExpanded(!node.IsExpanded())
		}
	})

	// Handle selection changes
	tree.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}

		if data, ok := node.GetReference().(*Node); ok {
			path := data.OriginalPath
			detailsView.SetText(path)

			if explainer, ok := pathExplainers[path]; ok {
				buf := bytes.Buffer{}
				explainer.Explain(&buf, path)
				detailsView.SetText(fmt.Sprintf("%s\n\n%s", path, buf.String()))
			}
		}
	})

	app := tview.NewApplication()

	// Handle TAB key to switch focus between views
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(detailsView) // Switch focus to the DetailsView
			return nil
		}

		// back to the root (step back) by ESC
		if event.Key() == tcell.KeyEscape && len(stack) > 1 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			prevNode := stack[len(stack)-1]
			tree.SetRoot(prevNode).
				SetCurrentNode(cur)
			return nil
		}

		return event
	})

	detailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(tree) // Switch focus to the TreeView
			return nil
		}
		return event
	})

	// Create a layout to arrange the UI components.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().
				AddItem(tree, 0, 1, true).
				AddItem(detailsView, 0, 1, false),
			0, 1, true,
		)

	// Set up the app and start it.

	if err := app.SetRoot(layout, true).Run(); err != nil {
		panic(err)
	}
}
