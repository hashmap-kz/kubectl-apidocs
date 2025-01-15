package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type TreeData struct {
	gvr schema.GroupVersionResource
}

// Custom sort function to prioritize apps/v1 and v1 at the top
func customSortGroups(groups []*metav1.APIResourceList) {

	// Prioritize these groups at the top
	topLevels := []string{
		"apps/v1",
		"v1",
		"batch/v1",
		"rbac.authorization.k8s.io/v1",
		"networking.k8s.io/v1",
		"gateway.networking.k8s.io/v1",
		"gateway.networking.k8s.io/v1beta1",
	}

	sort.SliceStable(groups, func(i, j int) bool {

		for _, t := range topLevels {
			if groups[i].GroupVersion == t {
				return true
			}
			if groups[j].GroupVersion == t {
				return false
			}
		}

		// Default alphabetical sorting
		return groups[i].GroupVersion < groups[j].GroupVersion
	})
}

func main() {
	// Load the kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client config: %v", err)
	}

	// Create the Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	// Get the discovery client
	discoveryClient := clientset.Discovery()

	// Get API resources
	resources, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		log.Fatalf("Error getting API resources: %v", err)
	}

	// Create a new tview application
	app := tview.NewApplication()

	// Create the root tree node
	root := tview.NewTreeNode("API Resources").
		SetColor(tcell.ColorYellow) // Set root node color using tcell

	// Sort the API groups with custom logic to prioritize apps/v1 and v1 at the top
	customSortGroups(resources)

	// Build the tree with API groups and resources
	for _, group := range resources {
		// Create a tree node for the API group
		groupNode := tview.NewTreeNode(group.GroupVersion).
			SetColor(tcell.ColorGreen) // Set API group node color using tcell

		// groupNode.SetExpanded(false)

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
			data := &TreeData{
				gvr: gvr,
			}

			resourceNode := tview.NewTreeNode(fmt.Sprintf("%s (%s)", resource.Kind, resource.Name)).
				SetColor(tcell.ColorWhite).
				SetReference(data)
			groupNode.AddChild(resourceNode)
		}

		// Add the group node as a child of the root node
		root.AddChild(groupNode)
	}

	// Create a tree view to display the tree
	treeView := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).
		SetGraphicsColor(tcell.ColorWhite) // Set tree view graphics color

	// Create a TextView to display field details.
	detailsView := tview.NewTextView()
	detailsView.SetDynamicColors(true)
	detailsView.SetBorder(true)
	detailsView.SetTitle("Field Details")
	detailsView.SetScrollable(true)
	detailsView.SetWrap(true)

	// Stack to handle navigation back
	var stack []*tview.TreeNode
	stack = append(stack, root)

	// Add key event handler for toggling node expansion
	treeView.SetSelectedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}

		// open subview with a subtree
		if len(node.GetChildren()) > 0 {
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
			cur := stack[len(stack)-1]
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

		if data, ok := node.GetReference().(*TreeData); ok {
			detailsView.SetText(fmt.Sprintf("%v", data))
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
