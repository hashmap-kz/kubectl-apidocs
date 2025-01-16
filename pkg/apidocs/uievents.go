package apidocs

import (
	"bytes"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
			explainer := NewExplainer(*data.gvr, uiData.OpenAPIClient)
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

func setupListenersForResourceDetailsView(uiState *UIState) error {
	uiState.apiResourcesDetailsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			uiState.app.SetFocus(uiState.apiResourcesTreeView) // Switch focus to the TreeView
			return nil
		}
		return event
	})
	return nil
}

func setupListenersForCmdInput(uiState *UIState) error {
	// Command was set, process it, close input cmd, set focus onto the tree
	uiState.cmdInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			if uiState.cmdInputIsOn && uiState.cmdInputPurpose == cmdInputPurposeSearch {
				searchTerm := uiState.cmdInput.GetText()
				// TODO: search inside current node parent
				highlightMatchingNodes(uiState, uiState.apiResourcesRootNode, searchTerm)
			}

			uiState.cmdInput.SetText("")
			uiState.cmdInputIsOn = false
			uiState.mainLayout.RemoveItem(uiState.cmdInput)    // Hide the input field
			uiState.app.SetFocus(uiState.apiResourcesTreeView) // Focus back to main layout
		}
	})

	return nil
}

func setupListenersForApp(uiState *UIState) error {
	// Set up application key events
	uiState.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Show the input field on Shift+:
		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			if uiState.cmdInputIsOn {
				return nil
			}
			uiState.cmdInput.SetLabel("Search:")
			uiState.cmdInputIsOn = true
			uiState.cmdInputPurpose = cmdInputPurposeSearch
			uiState.mainLayout.AddItem(uiState.cmdInput, 3, 1, true) // Show the input field
			uiState.app.SetFocus(uiState.cmdInput)                   // Focus on the input field
			return nil                                               // Prevent further processing
		}
		if event.Key() == tcell.KeyRune && event.Rune() == ':' {
			if uiState.cmdInputIsOn {
				return nil
			}
			uiState.cmdInput.SetLabel("Command:")
			uiState.cmdInputIsOn = true
			uiState.cmdInputPurpose = cmdInputPurposeCmd
			uiState.mainLayout.AddItem(uiState.cmdInput, 3, 1, true) // Show the input field
			uiState.app.SetFocus(uiState.cmdInput)                   // Focus on the input field
			return nil                                               // Prevent further processing
		}
		// TODO: quit on <:q>
		// Quit the app on 'q'
		//if event.Rune() == 'q' {
		//	uiState.app.Stop()
		//}
		return event
	})
	return nil
}
