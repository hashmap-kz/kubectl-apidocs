package apidocs

import (
	"fmt"

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

// TreeData is used for store custom properties in *tview.TreeNode references
type TreeData struct {
	nodeType TreeDataNodeType

	// it means, that node is already opened in sub-view
	// do not add it to a stack view again and again
	inPreview bool

	path string
	gvr  *schema.GroupVersionResource
}

func extractTreeData(node *tview.TreeNode) (*TreeData, error) {
	if data, ok := node.GetReference().(*TreeData); ok {
		return data, nil
	}
	return nil, fmt.Errorf("unexpected. get-ref failed: %v", node)
}

func setInPreview(node *tview.TreeNode, inPreview bool) error {
	data, err := extractTreeData(node)
	if err != nil {
		return err
	}
	data.inPreview = inPreview
	node.SetReference(data)
	return nil
}

func (d *TreeData) IsNodeType(nodeTypes ...TreeDataNodeType) bool {
	for _, t := range nodeTypes {
		if d.nodeType == t {
			return true
		}
	}
	return false
}
