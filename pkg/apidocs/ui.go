package apidocs

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

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

type UIState struct {
	app                     *tview.Application
	apiResourcesRootNode    *tview.TreeNode
	apiResourcesTreeView    *tview.TreeView
	apiResourcesDetailsView *tview.TextView
	apiResourcesViewsLayout *tview.Flex
	mainLayout              *tview.Flex
	cmdInput                *tview.InputField
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
	apiResourcesRootNode := tview.NewTreeNode("API Resources").
		SetColor(tcell.ColorYellow).
		SetReference(&TreeData{nodeType: nodeTypeRoot})

	// Sort the API groups with custom logic to prioritize apps/v1 and v1 at the top
	customSortGroups(serverPreferredResources)

	// Populate root node with groups/resources/fields
	err = populateRootNodeWithResources(apiResourcesRootNode, uiData, serverPreferredResources)
	if err != nil {
		return err
	}

	// Create the help menu (top)
	helpMenu := tview.NewTextView()
	helpMenu.SetDynamicColors(true)
	helpMenu.SetTextAlign(tview.AlignLeft)
	helpMenu.SetText(getHelpMenuContent())
	helpMenu.SetBorder(true)

	// Create a main tree view (lhs)
	apiResourcesTreeView := tview.NewTreeView()
	apiResourcesTreeView.SetRoot(apiResourcesRootNode)
	apiResourcesTreeView.SetCurrentNode(apiResourcesRootNode)
	apiResourcesTreeView.SetGraphicsColor(tcell.ColorWhite)
	apiResourcesTreeView.SetTitle("Resources")
	apiResourcesTreeView.SetBorder(true)

	// Create a main details view (rhs)
	apiResourcesDetailsView := tview.NewTextView()
	apiResourcesDetailsView.SetDynamicColors(true)
	apiResourcesDetailsView.SetBorder(true)
	apiResourcesDetailsView.SetTitle("Details")
	apiResourcesDetailsView.SetScrollable(true)
	apiResourcesDetailsView.SetWrap(true)

	// Create a horizontal flex layout for resources-tree-view and resources-details-view
	apiResourcesViewsLayout := tview.NewFlex()
	apiResourcesViewsLayout.AddItem(apiResourcesTreeView, 0, 1, true)
	apiResourcesViewsLayout.AddItem(apiResourcesDetailsView, 0, 1, false)

	// Create a main layout for app
	mainLayout := tview.NewFlex()
	mainLayout.SetDirection(tview.FlexRow)
	mainLayout.AddItem(helpMenu, 4, 1, false)
	mainLayout.AddItem(apiResourcesViewsLayout, 0, 2, true)

	// Create the input field (bottom, hidden by default)
	cmdInput := tview.NewInputField()
	cmdInput.SetLabel("Command: ")
	cmdInput.SetFieldWidth(20)
	cmdInput.SetBorder(true)

	// Set up listeners for app state.
	err = setupListeners(uiData, &UIState{
		app:                     app,
		apiResourcesRootNode:    apiResourcesRootNode,
		apiResourcesTreeView:    apiResourcesTreeView,
		apiResourcesDetailsView: apiResourcesDetailsView,
		apiResourcesViewsLayout: apiResourcesViewsLayout,
		mainLayout:              mainLayout,
		cmdInput:                cmdInput,
	})
	if err != nil {
		return err
	}

	// Set up the app and start it.
	if err := app.SetRoot(mainLayout, true).Run(); err != nil {
		return err
	}

	return nil
}

func setupListeners(
	uiData *UIData,
	uiState *UIState,
) error {
	// To handle errors inside closures
	var listenersErr error

	// Stack to handle navigation back
	var navigationStack []*tview.TreeNode
	navigationStack = append(navigationStack, uiState.apiResourcesRootNode)

	// Set up application key events
	uiState.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Show the input field on Shift+:
		if event.Key() == tcell.KeyRune && event.Rune() == ':' {
			uiState.mainLayout.AddItem(uiState.cmdInput, 3, 1, true) // Show the input field
			uiState.app.SetFocus(uiState.cmdInput)                   // Focus on the input field
			return nil                                               // Prevent further processing
		}
		// Quit the app on 'q'
		if event.Rune() == 'q' {
			uiState.app.Stop()
		}
		return event
	})

	// Command was set, process it, close input cmd, set focus onto the tree
	uiState.cmdInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			uiState.mainLayout.RemoveItem(uiState.cmdInput)    // Hide the input field
			uiState.app.SetFocus(uiState.apiResourcesTreeView) // Focus back to main layout
		}
	})

	uiState.apiResourcesDetailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			uiState.app.SetFocus(uiState.apiResourcesTreeView) // Switch focus to the TreeView
			return nil
		}
		return event
	})

	// Add key event handler for toggling node expansion
	uiState.apiResourcesTreeView.SetSelectedFunc(func(node *tview.TreeNode) {
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
			uiState.apiResourcesTreeView.SetRoot(node).SetCurrentNode(node)
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
	uiState.apiResourcesTreeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			uiState.app.SetFocus(uiState.apiResourcesDetailsView) // Switch focus to the DetailsView
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
			data, err := extractTreeData(cur)
			if err != nil {
				listenersErr = err
				return nil
			}
			// don't need to expand the resource, we need just its name
			if data.nodeType == nodeTypeResource {
				cur.SetExpanded(false)
			}
			// always expand groups
			if data.nodeType == nodeTypeGroup {
				cur.SetExpanded(true)
			}

			navigationStack = navigationStack[:len(navigationStack)-1]
			prevNode := navigationStack[len(navigationStack)-1]
			uiState.apiResourcesTreeView.SetRoot(prevNode).SetCurrentNode(cur)
			return nil
		}

		return event
	})
	if listenersErr != nil {
		return listenersErr
	}

	// Handle selection changes
	uiState.apiResourcesTreeView.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		data, err := extractTreeData(node)
		if err != nil {
			listenersErr = err
			return
		}

		uiState.apiResourcesDetailsView.SetText(data.path)
		if data.nodeType == nodeTypeField || data.nodeType == nodeTypeResource {
			explainer := Explainer{
				gvr:           *data.gvr,
				openAPIClient: uiData.OpenAPIClient,
			}

			buf := bytes.Buffer{}
			err := explainer.Explain(&buf, data.path)
			if err == nil {
				uiState.apiResourcesDetailsView.SetText(fmt.Sprintf("%s\n\n%s", data.path, buf.String()))
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
			resourceNode, err := createResourceNodeWithAllFieldsSet(group, &resource, uiData)
			if err != nil {
				return err
			}
			groupNode.AddChild(resourceNode)
		}

		// Add the group node as a child of the root node
		root.AddChild(groupNode)
	}
	return nil
}

func createResourceNodeWithAllFieldsSet(
	group *metav1.APIResourceList,
	resource *metav1.APIResource,
	uiData *UIData,
) (*tview.TreeNode, error) {
	gv, err := schema.ParseGroupVersion(group.GroupVersion)
	if err != nil {
		return nil, err
	}

	gvr := gv.WithResource(resource.Name)

	paths, err := getPaths(uiData.RestMapper, uiData.OpenAPISchema, gvr)
	if err != nil {
		return nil, err
	}

	// Create internal tree from a given paths
	rootFieldsNode := &ResourceFieldsNode{Name: "root"}
	for _, fieldPath := range paths {
		rootFieldsNode.AddPath(fieldPath)
	}

	// Convert internal tree to a tree-view
	tempNode := tview.NewTreeNode("tmp")
	populateNodeWithResourceFields(tempNode, rootFieldsNode.Children, &gvr)
	if len(tempNode.GetChildren()) != 1 {
		return nil, fmt.Errorf("error when populating fields for tree node")
	}

	// Fetch the result after conversion
	resourceNodeTreeView := tempNode.GetChildren()[0]
	resourceNodeTreeView.SetColor(tcell.ColorBlue)
	resourceNodeTreeView.SetText(fmt.Sprintf("%s (%s)", resource.Kind, resource.Name))

	// Customize node internal data
	resourceNodeData, err := extractTreeData(resourceNodeTreeView)
	if err != nil {
		return nil, err
	}
	resourceNodeData.nodeType = nodeTypeResource
	resourceNodeData.gvr = &gvr
	resourceNodeTreeView.SetReference(resourceNodeData)

	return resourceNodeTreeView, nil
}

func populateNodeWithResourceFields(
	parent *tview.TreeNode,
	children map[string]*ResourceFieldsNode,
	gvr *schema.GroupVersionResource,
) {
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

func getHelpMenuContent() string {
	return strings.TrimSpace(`
[blue]<:>[-] Search
[blue]<q>[-] Quit
`)
}
