package apidocs

import "github.com/rivo/tview"

type TreeLinks struct {
	// key=child, value=parent
	ParentMap map[*tview.TreeNode]*tview.TreeNode
}

func NewTreeLinks() *TreeLinks {
	return &TreeLinks{
		ParentMap: make(map[*tview.TreeNode]*tview.TreeNode),
	}
}

func (t *TreeLinks) FillLinks(root *tview.TreeNode) {
	for _, c := range root.GetChildren() {
		t.ParentMap[c] = root
		t.FillLinks(c)
	}
}
