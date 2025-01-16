package apidocs

import (
	"bytes"
	"fmt"
	"log"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (o *Options) RunApp() {
	// Get API resources
	resources, err := o.discoveryClient.ServerPreferredResources()
	if err != nil {
		log.Fatalf("Error getting API resources: %v", err)
	}

	// Create a new tview application
	app := tview.NewApplication()

	// Create the root tree node
	root := tview.NewTreeNode("API Resources").
		SetColor(tcell.ColorYellow).SetReference(&TreeData{nodeType: nodeTypeRoot})

	// Sort the API groups with custom logic to prioritize apps/v1 and v1 at the top
	customSortGroups(resources)

	// Build the tree with API groups and resources
	for _, group := range resources {
		// Create a tree node for the API group
		groupNode := tview.NewTreeNode(group.GroupVersion).
			SetColor(tcell.ColorGreen).
			SetReference(&TreeData{nodeType: nodeTypeGroup})

		// Sort the resources inside each group alphabetically
		sort.SliceStable(group.APIResources, func(i, j int) bool {
			return group.APIResources[i].Name < group.APIResources[j].Name
		})

		if len(group.APIResources) == 0 {
			continue
		}

		// Add resources as child nodes to the group node
		for _, resource := range group.APIResources {
			gv, err := schema.ParseGroupVersion(group.GroupVersion)
			if err != nil {
				continue
			}
			gvr := gv.WithResource(resource.Name)

			// fields+
			paths, err := getPaths(o.restMapper, o.openAPISchema, gvr)
			if err != nil {
				log.Fatal(err)
			}

			rootFieldsNode := &ResourceFieldsNode{Name: "root"}
			for _, fieldPath := range paths {
				rootFieldsNode.AddPath(fieldPath)
			}
			tmpNode := tview.NewTreeNode("tmp")
			addChildrenFields(tmpNode, rootFieldsNode.Children, &gvr)
			firstChild := tmpNode.GetChildren()[0]
			firstChild.SetText(fmt.Sprintf("%s (%s)", resource.Kind, resource.Name))
			if data, ok := firstChild.GetReference().(*TreeData); ok {
				data.nodeType = nodeTypeResource
				data.gvr = &gvr
				firstChild.SetReference(data)
			}
			// fields-

			groupNode.AddChild(firstChild)
		}

		// Add the group node as a child of the root node
		root.AddChild(groupNode)
	}

	// Create a tree view to display the tree
	treeView := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).
		SetGraphicsColor(tcell.ColorWhite)

	treeView.SetTitle("Resources")
	treeView.SetBorder(true)

	// Create a TextView to display field details.
	detailsView := tview.NewTextView()
	detailsView.SetDynamicColors(true)
	detailsView.SetBorder(true)
	detailsView.SetTitle("Details")
	detailsView.SetScrollable(true)
	detailsView.SetWrap(true)

	detailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(treeView) // Switch focus to the TreeView
			return nil
		}
		return event
	})

	// Stack to handle navigation back
	var stack []*tview.TreeNode
	stack = append(stack, root)

	// Add key event handler for toggling node expansion
	treeView.SetSelectedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}

		// open subview with a subtree
		data := getReference(node)

		if (data.nodeType == nodeTypeGroup || data.nodeType == nodeTypeResource) && !data.inPreview {
			setInPreview(node, true)

			stack = append(stack, node)

			treeView.SetRoot(node).
				SetCurrentNode(node)

			node.SetExpanded(true)
		} else {
			// just expand subtree
			node.SetExpanded(!node.IsExpanded())
		}
	})

	// Handle TAB key to switch focus between views
	treeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(detailsView) // Switch focus to the DetailsView
			return nil
		}

		// back to the root (step back) by ESC
		if event.Key() == tcell.KeyEscape && len(stack) > 1 {
			// a node, that was used for preview, we need to clear the flag
			cur := stack[len(stack)-1]
			setInPreview(cur, false)

			stack = stack[:len(stack)-1]
			prevNode := stack[len(stack)-1]
			treeView.SetRoot(prevNode).
				SetCurrentNode(cur)
			return nil
		}

		return event
	})

	// Handle selection changes
	treeView.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		data := getReference(node)
		detailsView.SetText(data.path)
		if data.nodeType == nodeTypeField || data.nodeType == nodeTypeResource {
			explainer := Explainer{
				gvr:             *data.gvr,
				openAPIV3Client: o.openAPIClient,
			}

			buf := bytes.Buffer{}
			err := explainer.Explain(&buf, data.path)
			if err == nil {
				detailsView.SetText(fmt.Sprintf("%s\n\n%s", data.path, buf.String()))
			}
		}
	})

	// Create a layout to arrange the UI components.
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(
			tview.NewFlex().
				AddItem(treeView, 0, 1, true).
				AddItem(detailsView, 0, 1, false),
			0, 1, true,
		)

	// Set up the app and start it.

	if err := app.SetRoot(layout, true).Run(); err != nil {
		panic(err)
	}
}

func addChildrenFields(parent *tview.TreeNode, children map[string]*ResourceFieldsNode, gvr *schema.GroupVersionResource) {
	if len(children) != 0 {
		parent.SetText(parent.GetText() + " >")
		parent.SetColor(tcell.ColorGreen)
		parent.SetExpanded(!parent.IsExpanded())
	}

	keys := make([]string, 0, len(children))
	for key := range children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		childNode := tview.NewTreeNode(children[key].Name).SetReference(&TreeData{
			nodeType: nodeTypeField,
			path:     children[key].Path,
			gvr:      gvr,
		})
		parent.AddChild(childNode)
		if children[key].Children != nil {
			addChildrenFields(childNode, children[key].Children, gvr)
		}
	}
}
