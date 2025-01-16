package apidocs

import (
	"bytes"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	"k8s.io/kubectl/pkg/util/openapi"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type UIData struct {
	DiscoveryClient discovery.CachedDiscoveryInterface
	RestMapper      meta.RESTMapper
	OpenAPISchema   openapi.Resources
	OpenAPIClient   openapiclient.Client
}

func RunApp(uiData *UIData) error {
	// Get API serverPreferredResources
	serverPreferredResources, err := uiData.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return fmt.Errorf("error getting API serverPreferredResources: %v", err)
	}

	// Create a new tview application
	app := tview.NewApplication()

	// Create the root tree node
	root := tview.NewTreeNode("API Resources").
		SetColor(tcell.ColorYellow).
		SetReference(&TreeData{nodeType: nodeTypeRoot})

	// Sort the API groups with custom logic to prioritize apps/v1 and v1 at the top
	customSortGroups(serverPreferredResources)

	// Populate root node with groups/resources/fields
	err = populateRootNodeWithResources(root, uiData, serverPreferredResources)
	if err != nil {
		return err
	}

	// Create a main tree view (lhs)
	treeView := tview.NewTreeView()
	treeView.SetRoot(root)
	treeView.SetCurrentNode(root)
	treeView.SetGraphicsColor(tcell.ColorWhite)
	treeView.SetTitle("Resources")
	treeView.SetBorder(true)

	// Create a main details view (rhs)
	detailsView := tview.NewTextView()
	detailsView.SetDynamicColors(true)
	detailsView.SetBorder(true)
	detailsView.SetTitle("Details")
	detailsView.SetScrollable(true)
	detailsView.SetWrap(true)

	err = setupLIsteners(uiData, app, root, treeView, detailsView)
	if err != nil {
		return err
	}

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
		return err
	}

	return nil
}

func setupLIsteners(uiData *UIData, app *tview.Application, root *tview.TreeNode, treeView *tview.TreeView, detailsView *tview.TextView) error {
	var listenersErr error

	// Stack to handle navigation back
	var navigationStack []*tview.TreeNode
	navigationStack = append(navigationStack, root)

	detailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(treeView) // Switch focus to the TreeView
			return nil
		}
		return event
	})

	// Add key event handler for toggling node expansion
	treeView.SetSelectedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}

		// open subview with a subtree
		data, err := extractTreeData(node)
		if err != nil {
			listenersErr = err
			return
		}

		if (data.nodeType == nodeTypeGroup || data.nodeType == nodeTypeResource) && !data.inPreview {
			err := setInPreview(node, true)
			if err != nil {
				listenersErr = err
				return
			}
			navigationStack = append(navigationStack, node)
			treeView.SetRoot(node).SetCurrentNode(node)
			node.SetExpanded(true)
		} else {
			// just expand subtree
			node.SetExpanded(!node.IsExpanded())
		}
	})
	if listenersErr != nil {
		return listenersErr
	}

	// Handle TAB key to switch focus between views
	treeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(detailsView) // Switch focus to the DetailsView
			return nil
		}

		// back to the root (step back) by ESC
		if event.Key() == tcell.KeyEscape && len(navigationStack) > 1 {
			// a node, that was used for preview, we need to clear the flag
			cur := navigationStack[len(navigationStack)-1]
			err := setInPreview(cur, false)
			if err != nil {
				listenersErr = err
				return nil
			}

			navigationStack = navigationStack[:len(navigationStack)-1]
			prevNode := navigationStack[len(navigationStack)-1]
			treeView.SetRoot(prevNode).SetCurrentNode(cur)
			return nil
		}

		return event
	})
	if listenersErr != nil {
		return listenersErr
	}

	// Handle selection changes
	treeView.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		data, err := extractTreeData(node)
		if err != nil {
			listenersErr = err
			return
		}

		detailsView.SetText(data.path)
		if data.nodeType == nodeTypeField || data.nodeType == nodeTypeResource {
			explainer := Explainer{
				gvr:           *data.gvr,
				openAPIClient: uiData.OpenAPIClient,
			}

			buf := bytes.Buffer{}
			err := explainer.Explain(&buf, data.path)
			if err == nil {
				detailsView.SetText(fmt.Sprintf("%s\n\n%s", data.path, buf.String()))
			}
		}
	})
	if listenersErr != nil {
		return listenersErr
	}
	return nil
}

func populateRootNodeWithResources(root *tview.TreeNode, uiData *UIData, resources []*metav1.APIResourceList) error {
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

		// lint: rangeValCopy
		resources := group.APIResources

		// Add resources as child nodes to the group node
		for i := 0; i < len(resources); i++ {
			resource := resources[i]

			gv, err := schema.ParseGroupVersion(group.GroupVersion)
			if err != nil {
				continue
			}
			gvr := gv.WithResource(resource.Name)

			// fields+
			paths, err := getPaths(uiData.RestMapper, uiData.OpenAPISchema, gvr)
			if err != nil {
				return err
			}

			rootFieldsNode := &ResourceFieldsNode{Name: "root"}
			for _, fieldPath := range paths {
				rootFieldsNode.AddPath(fieldPath)
			}
			tempNode := tview.NewTreeNode("tmp")
			populateNodeWithResourceFields(tempNode, rootFieldsNode.Children, &gvr)
			if len(tempNode.GetChildren()) != 1 {
				return fmt.Errorf("error when populating fields for tree node")
			}
			firstChild := tempNode.GetChildren()[0]
			firstChild.SetText(fmt.Sprintf("%s (%s)", resource.Kind, resource.Name))
			// Customize first child, which is actually a root for the resource: deployment, statefulset, etc...
			firstChildData, err := extractTreeData(firstChild)
			if err != nil {
				return err
			}
			firstChildData.nodeType = nodeTypeResource
			firstChildData.gvr = &gvr
			firstChild.SetReference(firstChildData)
			// fields-

			groupNode.AddChild(firstChild)
		}

		// Add the group node as a child of the root node
		root.AddChild(groupNode)
	}
	return nil
}

func populateNodeWithResourceFields(parent *tview.TreeNode, children map[string]*ResourceFieldsNode, gvr *schema.GroupVersionResource) {
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
			populateNodeWithResourceFields(childNode, children[key].Children, gvr)
		}
	}
}
