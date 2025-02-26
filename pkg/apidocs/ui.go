package apidocs

import (
	"fmt"
	"sort"
	"strings"
	"sync"

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

type cmdInputPurpose string

var (
	cmdInputPurposeCmd    cmdInputPurpose = "cmd"
	cmdInputPurposeSearch cmdInputPurpose = "search"
)

type UIState struct {
	app                     *tview.Application
	apiResourcesRootNode    *tview.TreeNode
	apiResourcesTreeView    *tview.TreeView
	apiResourcesDetailsView *tview.TextView
	apiResourcesViewsLayout *tview.Flex
	mainLayout              *tview.Flex
	cmdInput                *tview.InputField
	cmdInputIsOn            bool
	cmdInputPurpose         cmdInputPurpose
	treeLinks               *TreeLinks
	explainCache            *sync.Map
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
	apiResourcesDetailsView.SetTextColor(tcell.ColorLightGray)

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
	cmdInput.SetFieldWidth(32)
	cmdInput.SetBorder(true)
	cmdInput.SetFieldTextColor(tcell.ColorLightGray)
	cmdInput.SetBackgroundColor(tcell.ColorBlack)
	cmdInput.SetLabelColor(tcell.ColorYellow)
	cmdInput.SetFieldBackgroundColor(tcell.ColorBlack)

	// parent/child relationships (used for searching)
	treeLinks := NewTreeLinks()
	treeLinks.FillLinks(apiResourcesRootNode)

	// Set up listeners for app state.
	err = setupListeners(uiData, &UIState{
		app:                     app,
		apiResourcesRootNode:    apiResourcesRootNode,
		apiResourcesTreeView:    apiResourcesTreeView,
		apiResourcesDetailsView: apiResourcesDetailsView,
		apiResourcesViewsLayout: apiResourcesViewsLayout,
		mainLayout:              mainLayout,
		cmdInput:                cmdInput,
		treeLinks:               treeLinks,
		explainCache:            &sync.Map{},
	})
	if err != nil {
		return err
	}

	// Set colors
	resetNodeColors(apiResourcesRootNode)

	// Set up the app and start it.
	if err := app.SetRoot(mainLayout, true).Run(); err != nil {
		return err
	}

	return nil
}

func populateRootNodeWithResources(
	apiResourcesRootNode *tview.TreeNode,
	uiData *UIData,
	serverPreferredResources []*metav1.APIResourceList,
) error {
	// Build the tree with API groups and resources
	for _, group := range serverPreferredResources {
		// Create a tree node for the API group
		groupNode := tview.NewTreeNode(group.GroupVersion).
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
		apiResourcesRootNode.AddChild(groupNode)
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
[yellow]</term>[-]  Search | [yellow]<:cmd>[-] Command            | [yellow]<ENTER>[-] Select (gr/res) | [yellow]<hjkl>[-]   Navigate |
[yellow]<ctrl-c>[-] Quit   | [yellow]<TAB>[-]  Focus tree/details | [yellow]<ESC>[-]   Step back       | [yellow]<ARROWS>[-] Navigate |
`)
}
