package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	openapiclient "k8s.io/client-go/openapi"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/kube-openapi/pkg/util/proto"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/explain"
	"k8s.io/kubectl/pkg/util/openapi"

	explainv2 "k8s.io/kubectl/pkg/explain/v2"
)

// Node represents a node in the tree
type Node struct {
	Name     string           `json:"name"`
	Children map[string]*Node `json:"children,omitempty"`
}

// SortChildren recursively sorts the children of the node
func (n *Node) SortChildren() {
	if n.Children == nil {
		return
	}

	// Extract and sort the keys
	keys := make([]string, 0, len(n.Children))
	for key := range n.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create a new sorted map
	sortedChildren := make(map[string]*Node, len(n.Children))
	for _, key := range keys {
		child := n.Children[key]
		// Recursively sort the children of each child
		child.SortChildren()
		sortedChildren[key] = child
	}

	// Replace the old map with the sorted one
	n.Children = sortedChildren
}

// AddPath adds a path to the tree
func (n *Node) AddPath(path []string) {
	if len(path) == 0 {
		return
	}

	// Get the current level key
	key := path[0]

	// Ensure the child exists
	if n.Children == nil {
		n.Children = make(map[string]*Node)
	}
	if _, exists := n.Children[key]; !exists {
		n.Children[key] = &Node{Name: key}
	}

	// Recurse to add the rest of the path
	n.Children[key].AddPath(path[1:])
}

type Options struct {
	// User input
	apiVersion       string
	inputFieldPath   string
	disablePrintPath bool
	showBrackets     bool

	// After completion
	inputFieldPathRegex *regexp.Regexp
	gvrs                []schema.GroupVersionResource

	// Dependencies
	genericclioptions.IOStreams
	discovery             discovery.CachedDiscoveryInterface
	mapper                meta.RESTMapper
	schema                openapi.Resources
	cachedOpenAPIV3Client openapiclient.Client
}

type groupVersionAPIResource struct {
	schema.GroupVersionResource
	metav1.APIResource
}

func (o *Options) discover() (map[string]*groupVersionAPIResource, []schema.GroupVersionResource, error) {
	lists, err := o.discovery.ServerPreferredResources()
	if err != nil {
		return nil, nil, err
	}
	var gvrs []schema.GroupVersionResource
	m := make(map[string]*groupVersionAPIResource)
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			gvr := gv.WithResource(resource.Name)
			gvrs = append(gvrs, gvr)
			r := groupVersionAPIResource{
				GroupVersionResource: gvr,
				APIResource:          resource,
			}
			m[resource.Name] = &r
			m[resource.Kind] = &r
			m[resource.SingularName] = &r
			for _, shortName := range resource.ShortNames {
				m[shortName] = &r
			}
		}
	}
	sort.SliceStable(gvrs, func(i, j int) bool {
		return gvrs[i].String() < gvrs[j].String()
	})
	return m, gvrs, nil
}

func NewOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		IOStreams: streams,
	}
}

// buildTreeView creates a tview.TreeView from a Node
func buildTreeView(rootNode *Node) (*tview.TreeNode, *tview.TreeView) {
	// Create the root tree node
	rootTree := tview.NewTreeNode(rootNode.Name).SetColor(tview.Styles.PrimitiveBackgroundColor).SetExpanded(true)

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
			childNode := tview.NewTreeNode(children[key].Name).SetReference(key)
			parent.AddChild(childNode)
			if children[key].Children != nil {
				addChildren(childNode, children[key].Children)
			}
		}
	}

	// no-sort
	// // Recursive function to add children
	// var addChildren func(parent *tview.TreeNode, children map[string]*Node)
	// addChildren = func(parent *tview.TreeNode, children map[string]*Node) {
	// 	if len(children) != 0 {
	// 		parent.SetColor(tcell.ColorGreen)
	// 		parent.SetExpanded(!parent.IsExpanded())
	// 	}
	// 	for _, child := range children {
	// 		childNode := tview.NewTreeNode(child.Name).SetReference(child)
	// 		parent.AddChild(childNode)
	// 		if child.Children != nil {
	// 			addChildren(childNode, child.Children)
	// 		}
	// 	}
	// }
	//

	// Add children to the root
	addChildren(rootTree, rootNode.Children)

	// Create the TreeView
	tree := tview.NewTreeView().
		SetRoot(rootTree).
		SetCurrentNode(rootTree)

	tree.SetBorder(true)
	tree.SetTitle("Resources")
	// tree.SetBorderColor(tcell.ColorBlue)
	// tree.GetRoot().SetExpanded(true)

	return rootTree, tree
}

func printTree() error {
	o := NewOptions(genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	})

	kubeConfigFlags := defaultConfigFlags().WithWarningPrinter(o.IOStreams)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	discovery, err := f.ToDiscoveryClient()
	if err != nil {
		return err
	}
	o.discovery = discovery

	o.mapper, err = f.ToRESTMapper()
	if err != nil {
		return err
	}
	o.schema, err = f.OpenAPISchema()
	if err != nil {
		return err
	}

	// TODO: cached
	c, err := f.OpenAPIV3Client()
	if err != nil {
		return err
	}
	o.cachedOpenAPIV3Client = c

	gvarMap, _, err := o.discover()
	if err != nil {
		return err
	}
	// fmt.Println(gvarMap, gvrs)

	/////// tests ///////

	// for _, gvrsItem := range gvrs {
	// 	if gvar, ok := gvarMap[gvrsItem.Resource]; ok {
	// 		o.inputFieldPathRegex = regexp.MustCompile(".*")
	// 		o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
	// 	}
	// }

	if gvar, ok := gvarMap["statefulsets"]; ok {
		o.inputFieldPathRegex = regexp.MustCompile(".*")
		o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
	}
	if gvar, ok := gvarMap["httproutes"]; ok {
		o.inputFieldPathRegex = regexp.MustCompile(".*")
		o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
	}
	if gvar, ok := gvarMap["gateways"]; ok {
		o.inputFieldPathRegex = regexp.MustCompile(".*")
		o.gvrs = append(o.gvrs, gvar.GroupVersionResource)
	}

	pathExplainers := make(map[string]explainer)
	var paths []path
	for _, gvr := range o.gvrs {
		visitor := &schemaVisitor{
			pathSchema: make(map[path]proto.Schema),
			prevPath: path{
				original:     strings.ToLower(gvr.Resource),
				withBrackets: strings.ToLower(gvr.Resource),
			},
			err: nil,
		}
		gvk, err := o.mapper.KindFor(gvr)
		if err != nil {
			return fmt.Errorf("get the group version kind: %w", err)
		}
		s := o.schema.LookupResource(gvk)
		if s == nil {
			return fmt.Errorf("no schema found for %s", gvk)
		}
		s.Accept(visitor)
		if visitor.err != nil {
			return visitor.err
		}
		filteredPaths := visitor.listPaths(func(s path) bool {
			return o.inputFieldPathRegex.MatchString(s.original)
		})
		for _, p := range filteredPaths {
			pathExplainers[p.original] = explainer{
				gvr:                 gvr,
				openAPIV3Client:     o.cachedOpenAPIV3Client,
				enablePrintPath:     !o.disablePrintPath,
				enablePrintBrackets: o.showBrackets,
			}
			paths = append(paths, p)
		}
	}

	// Create root node
	root := &Node{Name: "root"}

	// Add each path to the tree
	for _, line := range paths {
		path := strings.Split(line.original, ".")
		root.AddPath(path)
	}
	root.SortChildren()

	/////// UI ///////

	// Create the tree view
	rootTree, tree := buildTreeView(root)
	rootTree.SetExpanded(true)

	// Create a TextView to display field details.
	detailsView := tview.NewTextView()
	detailsView.SetDynamicColors(true)
	detailsView.SetBorder(true)
	detailsView.SetTitle("Field Details")
	detailsView.SetScrollable(true)
	detailsView.SetWrap(true)

	// Add key event handler for toggling node expansion
	tree.SetSelectedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}
		node.SetExpanded(!node.IsExpanded())
	})

	// Handle selection changes
	tree.SetChangedFunc(func(node *tview.TreeNode) {
		if node == nil {
			return
		}

		path := getNodePath(rootTree, node)
		path = strings.TrimPrefix(path, "/root.")
		detailsView.SetText(path)

		if explainer, ok := pathExplainers[path]; ok {
			buf := bytes.Buffer{}
			explainer.explain(&buf, path)

			detailsView.SetText(fmt.Sprintf("%s\n\n%s", path, buf.String()))
		}
	})

	// Handle node selection to display field details.
	// tree.SetSelectedFunc(func(node *tview.TreeNode) {
	// 	fieldProps := node.GetReference()
	// 	if props, ok := fieldProps.(map[string]interface{}); ok {
	// 		details := "Details:\n"
	// 		for key, value := range props {
	// 			details += fmt.Sprintf("[green]%s[white]: %v\n", key, value)
	// 		}
	// 		detailsView.SetText(details)
	// 	} else {
	// 		detailsView.SetText("[red]No details available.")
	// 	}
	// })

	app := tview.NewApplication()

	// Handle TAB key to switch focus between views
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			app.SetFocus(detailsView) // Switch focus to the DetailsView
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

	// // Create the application
	// app := tview.NewApplication()
	// if err := app.SetRoot(treeView, true).Run(); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	return err
	// }

	return nil
}

// Helper function to find the path of a node from root
func getNodePath(root, target *tview.TreeNode) string {
	var path []string
	if findPathRecursive(root, target, &path) {
		return "/" + joinPath(path)
	}
	return "Node not found"
}

// Recursive function to find the path
func findPathRecursive(current, target *tview.TreeNode, path *[]string) bool {
	*path = append(*path, current.GetText())
	if current == target {
		return true
	}
	for _, child := range current.GetChildren() {
		if findPathRecursive(child, target, path) {
			return true
		}
	}
	// Backtrack if target is not found in this branch
	*path = (*path)[:len(*path)-1]
	return false
}

// Helper function to join path components
func joinPath(path []string) string {
	return strings.Join(path, ".")
}

func main() {
	printTree()
}

// Copy from https://github.com/kubernetes/kubectl/blob/4f380d07c5e5bb41a037a72c4b35c7f828ba2d59/pkg/cmd/cmd.go#L95-L97
func defaultConfigFlags() *genericclioptions.ConfigFlags {
	return genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag().WithDiscoveryBurst(300).WithDiscoveryQPS(50.0)
}

//////////////////////////////////////////////////////////////////////
// schema visitor

type path struct {
	original     string
	withBrackets string
}

func (p path) isEmpty() bool {
	return p.original == "" && p.withBrackets == ""
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
			original:     strings.Join([]string{v.prevPath.original, key}, "."),
			withBrackets: strings.Join([]string{v.prevPath.withBrackets, key}, "."),
		}
	}
	for i, key := range keys {
		schema, err := explain.LookupSchemaForField(k, []string{key})
		if err != nil {
			v.err = err
			return
		}
		if _, ok := schema.(*proto.Array); ok {
			paths[i].withBrackets += "[]"
		}
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

func (v *schemaVisitor) listPaths(filter func(path) bool) []path {
	paths := make([]path, 0, len(v.pathSchema))
	for path := range v.pathSchema {
		if filter(path) {
			paths = append(paths, path)
		}
	}
	sort.SliceStable(paths, func(i, j int) bool {
		return paths[i].original < paths[j].original
	})
	return paths
}

//////////////////////////////////////////////////////////////////////
// explainer

type explainer struct {
	gvr                 schema.GroupVersionResource
	openAPIV3Client     openapiclient.Client
	enablePrintPath     bool
	enablePrintBrackets bool
}

func (e explainer) explain(w io.Writer, path string) error {
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
