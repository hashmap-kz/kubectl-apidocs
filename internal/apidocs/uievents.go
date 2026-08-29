package apidocs

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	detailsSearchMatchStyle        = "[black:teal]"
	detailsSearchCurrentMatchStyle = "[black:yellow]"
	detailsSearchResetStyle        = "[-:-:-]"
)

func setupListeners(
	uiData *UIData,
	uiState *UIState,
) error {
	var err error
	err = setupListenersForApp(uiState)
	if err != nil {
		return err
	}
	err = setupListenersForResourcesTreeView(uiData, uiState)
	if err != nil {
		return err
	}
	err = setupListenersForResourceDetailsView(uiState)
	if err != nil {
		return err
	}
	err = setupListenersForCmdInput(uiState)
	if err != nil {
		return err
	}
	return nil
}

func setupListenersForResourcesTreeView(uiData *UIData, uiState *UIState) error {
	// To handle errors inside closures
	var listenersErr error

	// Stack to handle navigation back
	var navigationStack []*tview.TreeNode
	navigationStack = append(navigationStack, uiState.apiResourcesRootNode)

	// Add key event handler for toggling node expansion
	// Handle <ENTER>
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

		if data.IsNodeType(nodeTypeGroup, nodeTypeResource) {
			// not in preview, add to view-stack
			if !data.inPreview {
				err := setInPreview(node, true)
				if err != nil {
					listenersErr = err
					return
				}
				navigationStack = append(navigationStack, node)
				uiState.apiResourcesTreeView.SetRoot(node).SetCurrentNode(node)
				node.SetExpanded(true)
			} else {
				node.SetExpanded(!node.IsExpanded())
			}
		} else if data.IsNodeType(nodeTypeRoot) {
			// expand/collapse all groups
			for _, nc := range node.GetChildren() {
				nc.SetExpanded(!nc.IsExpanded())
			}
		} else {
			// just expand subtree
			node.SetExpanded(!node.IsExpanded())
		}
	})
	if listenersErr != nil {
		return listenersErr
	}

	// Handle event keys: tab/h/l/ESC etc...
	uiState.apiResourcesTreeView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle TAB key to switch focus between views
		if event.Key() == tcell.KeyTab {
			setFocusOn(uiState, uiState.apiResourcesDetailsView) // Switch focus to the DetailsView
			return nil
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'r' {
			currentNode := uiState.apiResourcesTreeView.GetCurrentNode()
			if currentNode == nil {
				return nil
			}
			data, err := extractTreeData(currentNode)
			if err != nil {
				listenersErr = err
				return nil
			}
			if data.IsNodeType(nodeTypeResource) {
				explainPathRecursive(uiState, data, uiData)
				return nil
			}
		}

		// h/l -> collapse/expand
		// left-arrow/right-arrow -> collapse/expand
		// NOTE: expand fields only, ignore groups and resources (they're managed by ENTER)
		if (event.Key() == tcell.KeyRune && event.Rune() == 'h') || event.Key() == tcell.KeyLeft {
			listenersErr = expandCollapseHJKL(uiState, false)
			return nil
		}
		if (event.Key() == tcell.KeyRune && event.Rune() == 'l') || event.Key() == tcell.KeyRight {
			listenersErr = expandCollapseHJKL(uiState, true)
			return nil
		}

		// back to the root (step back) by ESC
		if event.Key() == tcell.KeyEscape && (len(navigationStack) > 1 || uiState.isInFilter) {
			// restore original layout, drop filtered tree
			if uiState.isInFilter {
				uiState.isInFilter = false
				// restore full tree
				uiState.apiResourcesTreeView.SetRoot(uiState.apiResourcesRootNode).
					SetCurrentNode(uiState.apiResourcesRootNode)
				return nil
			}

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
			if data.IsNodeType(nodeTypeResource) {
				cur.SetExpanded(false)
			}
			// always expand groups
			if data.IsNodeType(nodeTypeGroup) {
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
		setDetailsText(uiState, data.path)
		if data.IsNodeType(nodeTypeField, nodeTypeResource) {
			explainPath(uiState, data, uiData)
		}
	})
	if listenersErr != nil {
		return listenersErr
	}
	return nil
}

func explainPath(uiState *UIState, data *TreeData, uiData *UIData) {
	cacheKey := explainCacheKey(data.path, false)
	if cached, ok := uiState.explainCache.Load(cacheKey); ok {
		slog.Debug("explain", slog.String("cached", cacheKey))
		setDetailsText(uiState, fmt.Sprintf("%s\n\n%s", data.path, cached))
	} else {
		slog.Debug("explain", slog.String("perform", cacheKey))
		explainer := NewExplainer(*data.gvr, uiData.OpenAPIClient)
		buf := bytes.Buffer{}
		err := explainer.Explain(&buf, data.path)
		if err == nil {
			setDetailsText(uiState, fmt.Sprintf("%s\n\n%s", data.path, buf.String()))
			uiState.explainCache.Store(cacheKey, buf.String())
		}
	}
}

func explainPathRecursive(uiState *UIState, data *TreeData, uiData *UIData) {
	cacheKey := explainCacheKey(data.path, true)
	if cached, ok := uiState.explainCache.Load(cacheKey); ok {
		slog.Debug("explain", slog.String("cached", cacheKey))
		setDetailsText(uiState, fmt.Sprintf("%s [recursive]\n\n%s", data.path, cached))
		return
	}

	slog.Debug("explain", slog.String("perform", cacheKey))
	explainer := NewExplainer(*data.gvr, uiData.OpenAPIClient)
	buf := bytes.Buffer{}
	err := explainer.ExplainRecursive(&buf, data.path)
	if err == nil {
		setDetailsText(uiState, fmt.Sprintf("%s [recursive]\n\n%s", data.path, buf.String()))
		uiState.explainCache.Store(cacheKey, buf.String())
	}
}

func explainCacheKey(path string, recursive bool) string {
	if recursive {
		return path + "#recursive"
	}
	return path
}

func expandCollapseHJKL(uiState *UIState, expanded bool) error {
	curNode := uiState.apiResourcesTreeView.GetCurrentNode()
	data, err := extractTreeData(curNode)
	if err != nil {
		return err
	}

	// expand/collapse node itself
	if data.IsNodeType(nodeTypeField, nodeTypeGroup) {
		curNode.SetExpanded(expanded)
	}

	// expand/collapse all groups
	if data.IsNodeType(nodeTypeRoot) {
		for _, nc := range curNode.GetChildren() {
			nc.SetExpanded(expanded)
		}
	}

	return nil
}

func setupListenersForResourceDetailsView(uiState *UIState) error {
	uiState.apiResourcesDetailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			if uiState.cmdInputIsOn {
				return nil
			}
			uiState.cmdInput.SetLabel("Find:")
			uiState.cmdInput.SetText(uiState.detailsSearchTerm)
			uiState.cmdInputIsOn = true
			uiState.cmdInputPurpose = cmdInputPurposeFind
			uiState.mainLayout.AddItem(uiState.cmdInput, 3, 1, true)
			setFocusOn(uiState, uiState.cmdInput)
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'n' {
			selectNextDetailsMatch(uiState)
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'N' {
			selectPreviousDetailsMatch(uiState)
			return nil
		}
		if event.Key() == tcell.KeyEscape {
			clearDetailsSearch(uiState)
			return nil
		}
		if event.Key() == tcell.KeyTab {
			setFocusOn(uiState, uiState.apiResourcesTreeView) // Switch focus to the TreeView
			return nil
		}
		return event
	})
	return nil
}

func setupListenersForCmdInput(uiState *UIState) error {
	// Command was set, process it, close input cmd, set focus onto the tree
	uiState.cmdInput.SetDoneFunc(func(key tcell.Key) {
		// handle ENTER: search or CMD
		if key == tcell.KeyEnter {
			// search
			if uiState.cmdInputIsOn && uiState.cmdInputPurpose == cmdInputPurposeSearch {
				searchTerm := uiState.cmdInput.GetText()
				showFilteredTree(uiState, uiState.apiResourcesTreeView, searchTerm)
			}

			if uiState.cmdInputIsOn && uiState.cmdInputPurpose == cmdInputPurposeFind {
				applyDetailsSearch(uiState, uiState.cmdInput.GetText())
			}

			// quit
			if uiState.cmdInputIsOn && uiState.cmdInputPurpose == cmdInputPurposeCmd {
				cmd := uiState.cmdInput.GetText()
				if cmd == "q" {
					uiState.app.Stop()
				}
			}

			uiState.cmdInput.SetText("")
			uiState.cmdInputIsOn = false
			uiState.mainLayout.RemoveItem(uiState.cmdInput)   // Hide the input field
			if uiState.cmdInputPurpose == cmdInputPurposeFind {
				setFocusOn(uiState, uiState.apiResourcesDetailsView)
			} else {
				setFocusOn(uiState, uiState.apiResourcesTreeView) // Focus back to main layout
			}
		}

		// handle ESC: hide cmd-input on ESC
		if key == tcell.KeyEsc {
			if uiState.cmdInputIsOn {
				uiState.cmdInputIsOn = false
				uiState.mainLayout.RemoveItem(uiState.cmdInput)   // Hide the input field
				if uiState.cmdInputPurpose == cmdInputPurposeFind {
					setFocusOn(uiState, uiState.apiResourcesDetailsView)
				} else {
					setFocusOn(uiState, uiState.apiResourcesTreeView) // Focus back to main layout
				}
			}
		}
	})

	return nil
}

func setDetailsText(uiState *UIState, text string) {
	uiState.detailsRawText = text
	uiState.detailsSearchTerm = ""
	uiState.detailsSearchMatches = nil
	uiState.detailsSearchIndex = 0
	uiState.apiResourcesDetailsView.Highlight()
	uiState.apiResourcesDetailsView.SetText(text)
}

func applyDetailsSearch(uiState *UIState, term string) {
	uiState.detailsSearchTerm = term
	uiState.detailsSearchMatches = findDetailsMatches(uiState.detailsRawText, term)
	uiState.detailsSearchIndex = 0
	renderDetailsText(uiState)
}

func clearDetailsSearch(uiState *UIState) {
	if uiState.detailsSearchTerm == "" && len(uiState.detailsSearchMatches) == 0 {
		return
	}
	uiState.detailsSearchTerm = ""
	uiState.detailsSearchMatches = nil
	uiState.detailsSearchIndex = 0
	renderDetailsText(uiState)
}

func selectNextDetailsMatch(uiState *UIState) {
	if len(uiState.detailsSearchMatches) == 0 {
		return
	}
	uiState.detailsSearchIndex = (uiState.detailsSearchIndex + 1) % len(uiState.detailsSearchMatches)
	renderDetailsText(uiState)
}

func selectPreviousDetailsMatch(uiState *UIState) {
	if len(uiState.detailsSearchMatches) == 0 {
		return
	}
	uiState.detailsSearchIndex--
	if uiState.detailsSearchIndex < 0 {
		uiState.detailsSearchIndex = len(uiState.detailsSearchMatches) - 1
	}
	renderDetailsText(uiState)
}

func renderDetailsText(uiState *UIState) {
	if len(uiState.detailsSearchMatches) == 0 {
		uiState.apiResourcesDetailsView.Highlight()
		uiState.apiResourcesDetailsView.ScrollToBeginning()
		uiState.apiResourcesDetailsView.SetText(uiState.detailsRawText)
		return
	}

	currentMatch := uiState.detailsSearchMatches[uiState.detailsSearchIndex]
	uiState.apiResourcesDetailsView.SetText(buildDetailsSearchText(
		uiState.detailsRawText,
		uiState.detailsSearchMatches,
		uiState.detailsSearchIndex,
	))
	uiState.apiResourcesDetailsView.Highlight(currentMatch.regionID)
	uiState.apiResourcesDetailsView.ScrollTo(detailsSearchScrollRow(uiState.detailsRawText, currentMatch), 0)
}

func buildDetailsSearchText(text string, matches []detailsMatch, currentIndex int) string {
	var b strings.Builder
	last := 0
	for i, match := range matches {
		b.WriteString(text[last:match.start])
		b.WriteString(`["`)
		b.WriteString(match.regionID)
		b.WriteString(`"]`)
		if i == currentIndex {
			b.WriteString(detailsSearchCurrentMatchStyle)
		} else {
			b.WriteString(detailsSearchMatchStyle)
		}
		b.WriteString(text[match.start:match.end])
		b.WriteString(detailsSearchResetStyle)
		b.WriteString(`[""]`)
		last = match.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func findDetailsMatches(text, term string) []detailsMatch {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil
	}

	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(term)
	var matches []detailsMatch
	offset := 0

	for {
		idx := strings.Index(lowerText[offset:], lowerTerm)
		if idx == -1 {
			break
		}
		start := offset + idx
		end := start + len(term)
		matches = append(matches, detailsMatch{
			start:    start,
			end:      end,
			regionID: fmt.Sprintf("details-%d", len(matches)),
		})
		offset = end
	}

	return matches
}

func detailsSearchScrollRow(text string, match detailsMatch) int {
	row := strings.Count(text[:match.start], "\n") - 1
	if row < 0 {
		return 0
	}
	return row
}

func getClosestParentThatHasChildren(uiState *UIState, node *tview.TreeNode) *tview.TreeNode {
	parentMap := uiState.treeLinks.ParentMap
	for node != nil {
		parent := parentMap[node]
		if parent != nil && len(parent.GetChildren()) > 0 {
			return parent
		}
		node = parent
	}
	return uiState.apiResourcesRootNode
}

func setupListenersForApp(uiState *UIState) error {
	// Set up application key events
	uiState.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// search input
		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			if uiState.app.GetFocus() == uiState.apiResourcesDetailsView {
				return event
			}
			if uiState.cmdInputIsOn {
				return nil
			}
			uiState.cmdInput.SetLabel("Search:")
			uiState.cmdInputIsOn = true
			uiState.cmdInputPurpose = cmdInputPurposeSearch
			uiState.mainLayout.AddItem(uiState.cmdInput, 3, 1, true) // Show the input field
			setFocusOn(uiState, uiState.cmdInput)                    // Focus on the input field
			return nil                                               // Prevent further processing
		}

		// command input
		if event.Key() == tcell.KeyRune && event.Rune() == ':' {
			if uiState.cmdInputIsOn {
				return nil
			}
			uiState.cmdInput.SetLabel("Command:")
			uiState.cmdInputIsOn = true
			uiState.cmdInputPurpose = cmdInputPurposeCmd
			uiState.mainLayout.AddItem(uiState.cmdInput, 3, 1, true) // Show the input field
			setFocusOn(uiState, uiState.cmdInput)                    // Focus on the input field
			return nil                                               // Prevent further processing
		}

		// back to closest-parent
		if event.Key() == tcell.KeyRune && event.Rune() == 'b' {
			currentNode := uiState.apiResourcesTreeView.GetCurrentNode()
			closestParentThatHasChildren := getClosestParentThatHasChildren(uiState, currentNode)
			if closestParentThatHasChildren != nil {
				uiState.apiResourcesTreeView.SetCurrentNode(closestParentThatHasChildren)
			}
			return nil
		}

		return event
	})
	return nil
}

func setFocusOn(uiState *UIState, curFocus tview.Primitive) {
	uiState.app.SetFocus(curFocus)

	// TODO: simplify this (loops, arrays, bitmasks ?)
	// the idea is simple: there are a bunch of views that may be focused,
	// change the border-color for the view that is under focus right now, and reset border-color
	// for all other views.
	switch curFocus {
	case uiState.apiResourcesTreeView:
		uiState.apiResourcesTreeView.SetBorderColor(focusColor)
		uiState.apiResourcesDetailsView.SetBorderColor(noFocusColor)
		uiState.cmdInput.SetBorderColor(noFocusColor)
	case uiState.apiResourcesDetailsView:
		uiState.apiResourcesTreeView.SetBorderColor(noFocusColor)
		uiState.apiResourcesDetailsView.SetBorderColor(focusColor)
		uiState.cmdInput.SetBorderColor(noFocusColor)
	case uiState.cmdInput:
		uiState.apiResourcesTreeView.SetBorderColor(noFocusColor)
		uiState.apiResourcesDetailsView.SetBorderColor(noFocusColor)
		uiState.cmdInput.SetBorderColor(focusColor)
	}
}
