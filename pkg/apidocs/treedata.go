package apidocs

import (
	"log"

	"github.com/rivo/tview"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	path string
	gvr  *schema.GroupVersionResource
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
