package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"

	openapiclient "k8s.io/client-go/openapi"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	explainv2 "k8s.io/kubectl/pkg/explain/v2"
)

type TreeDataNodeType string

var (
	nodeTypeRoot     TreeDataNodeType = "root"
	nodeTypeGroup    TreeDataNodeType = "group"
	nodeTypeResource TreeDataNodeType = "resource"
	nodeTypeField    TreeDataNodeType = "field"
)

type TreeData struct {
	nodeType TreeDataNodeType

	// it means, that node is already opened in sub-view
	// do not add it to a stack view again and again
	inPreview bool

	originalPath string
}

// TODO: cleanup
func getReference(node *tview.TreeNode) *TreeData {
	if data, ok := node.GetReference().(*TreeData); ok {
		return data
	}
	log.Fatalf("unexpected. get-ref failed: %v", node)
	return nil
}

func setInPreview(node *tview.TreeNode, inPreview bool) {
	data := getReference(node)
	data.inPreview = inPreview
	node.SetReference(data)
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

func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
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

	// flags+
	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	kubeConfigFlags := defaultConfigFlags().WithWarningPrinter(ioStreams)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)
	restMapper, err := f.ToRESTMapper()
	if err != nil {
		log.Fatal(err)
	}
	openApiSchema, err := f.OpenAPISchema()
	if err != nil {
		log.Fatal(err)
	}
	openAPIV3Client, err := f.OpenAPIV3Client()
	if err != nil {
		log.Fatal(err)
	}
	// flags-

	// Get API resources
	resources, err := discoveryClient.ServerPreferredResources()
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

	pathExplainers := make(map[string]Explainer)

	// Build the tree with API groups and resources
	for _, group := range resources {
		// Create a tree node for the API group
		groupNode := tview.NewTreeNode(group.GroupVersion).
			SetColor(tcell.ColorGreen).
			SetReference(&TreeData{nodeType: nodeTypeGroup})

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

			// fields+
			paths := getPaths(restMapper, openApiSchema, openAPIV3Client, gvr, pathExplainers)
			rootFieldsNode := &Node{Name: "root"}
			for _, line := range paths {
				rootFieldsNode.AddPath(line.original)
			}
			tmpNode := tview.NewTreeNode("tmp")
			addChildrenFields(tmpNode, rootFieldsNode.Children)
			firstChild := tmpNode.GetChildren()[0]
			firstChild.SetText(fmt.Sprintf("%s (%s)", resource.Kind, resource.Name))
			if data, ok := firstChild.GetReference().(*TreeData); ok {
				data.nodeType = nodeTypeResource
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
		detailsView.SetText(data.originalPath)
		if data.nodeType == nodeTypeField || data.nodeType == nodeTypeResource {
			if explainer, ok := pathExplainers[data.originalPath]; ok {
				buf := bytes.Buffer{}
				explainer.Explain(&buf, data.originalPath)
				detailsView.SetText(fmt.Sprintf("%s\n\n%s", data.originalPath, buf.String()))
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

func addChildrenFields(parent *tview.TreeNode, children map[string]*Node) {
	if len(children) != 0 {
		parent.SetText(parent.GetText() + " ⯈")
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
			nodeType:     nodeTypeField,
			originalPath: children[key].OriginalPath,
		})
		parent.AddChild(childNode)
		if children[key].Children != nil {
			addChildrenFields(childNode, children[key].Children)
		}
	}
}

func getPaths(restMapper meta.RESTMapper,
	openApiSchema openapi.Resources,
	openapiclient openapiclient.Client,
	gvr schema.GroupVersionResource,
	pathExplainers map[string]Explainer,
) []path {
	var paths []path
	visitor := &schemaVisitor{
		pathSchema: make(map[path]proto.Schema),
		prevPath: path{
			original: strings.ToLower(gvr.Resource),
		},
		err: nil,
	}
	gvk, err := restMapper.KindFor(gvr)
	if err != nil {
		log.Fatal(err)
	}
	s := openApiSchema.LookupResource(gvk)
	if s == nil {
		log.Fatal(err)
	}
	s.Accept(visitor)
	if visitor.err != nil {
		log.Fatal(visitor.err)
	}
	visitorPathsResult := visitor.listPaths()
	for _, p := range visitorPathsResult {
		pathExplainers[p.original] = Explainer{
			gvr:             gvr,
			openAPIV3Client: openapiclient,
		}
		paths = append(paths, p)
	}
	// resource itself
	pathExplainers[gvr.Resource] = Explainer{
		gvr:             gvr,
		openAPIV3Client: openapiclient,
	}
	return paths
}

//////////////////////////////////////////////////////////////////////
// schema visitor

type path struct {
	original string
}

type schemaVisitor struct {
	prevPath   path
	pathSchema map[path]proto.Schema
	err        error
}

var _ proto.SchemaVisitor = (*schemaVisitor)(nil)

func (v *schemaVisitor) VisitKind(k *proto.Kind) {
	keys := k.Keys()
	paths := make([]path, len(keys))
	for i, key := range keys {
		paths[i] = path{
			original: strings.Join([]string{v.prevPath.original, key}, "."),
		}
	}
	for i, key := range keys {
		schema, err := explain.LookupSchemaForField(k, []string{key})
		if err != nil {
			v.err = err
			return
		}
		// if _, ok := schema.(*proto.Array); ok {
		// 	// TODO: types for print on UI?
		// }
		v.pathSchema[paths[i]] = schema
		v.prevPath = paths[i]
		schema.Accept(v)
	}
}

var visitedReferences = map[string]struct{}{}

func (v *schemaVisitor) VisitReference(r proto.Reference) {
	if _, ok := visitedReferences[r.Reference()]; ok {
		return
	}
	visitedReferences[r.Reference()] = struct{}{}
	r.SubSchema().Accept(v)
	delete(visitedReferences, r.Reference())
}

func (*schemaVisitor) VisitPrimitive(*proto.Primitive) {
	// Nothing to do.
}

func (v *schemaVisitor) VisitArray(a *proto.Array) {
	a.SubType.Accept(v)
}

func (v *schemaVisitor) VisitMap(m *proto.Map) {
	m.SubType.Accept(v)
}

func (v *schemaVisitor) listPaths() []path {
	paths := make([]path, 0, len(v.pathSchema))
	for path := range v.pathSchema {
		paths = append(paths, path)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		return paths[i].original < paths[j].original
	})
	return paths
}

//////////////////////////////////////////////////////////////////////
// node

type Node struct {
	Name         string
	Children     map[string]*Node
	OriginalPath string
}

func NewTreeNode() *Node {
	return &Node{
		Children: make(map[string]*Node),
	}
}

func (node *Node) AddPath(path string) {
	parts := strings.Split(path, ".")
	current := node
	for i, part := range parts {

		// Ensure the child exists
		if current.Children == nil {
			current.Children = make(map[string]*Node)
		}

		if _, exists := current.Children[part]; !exists {
			current.Children[part] = NewTreeNode()
			current.Children[part].Name = part
		}
		current = current.Children[part]
		if current.OriginalPath == "" {
			current.OriginalPath = strings.Join(parts[:i+1], ".")
		}
	}
}

//////////////////////////////////////////////////////////////////////
// explainer

type Explainer struct {
	gvr             schema.GroupVersionResource
	openAPIV3Client openapiclient.Client
}

func (e Explainer) Explain(w io.Writer, path string) error {
	if len(path) == 0 {
		return fmt.Errorf("path must not be empty: %#v", path)
	}
	fields := strings.Split(path, ".")
	if len(fields) > 0 {
		// Remove resource name
		fields = fields[1:]
	}

	return explainv2.PrintModelDescription(
		fields,
		w,
		e.openAPIV3Client,
		e.gvr,
		false,
		"plaintext",
	)
}
