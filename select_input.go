package dxui

import (
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func (a *App) reconcileSelectCapture(actions map[uint64]inputAction) {
	if a.selectCapture.id == 0 {
		return
	}
	action, live := actions[a.selectCapture.id]
	node := findInstance(a.retained.Root(), a.selectCapture.id)
	if !live || action.selectp == nil || action.selectp.Disabled || node == nil || !node.State.SelectOpen {
		// Still consume the paired release, but never activate a reopened popup.
		a.selectCapture.id, a.selectCapture.index = 0, -1
	}
}

func (a *App) handleSelectEvent(event platform.Event) (dirty, handled, displayed bool, err error) {
	root := a.retained.Root()
	a.mu.Lock()
	actions := a.inputActions
	theme := a.theme
	geometry := a.geometry
	a.mu.Unlock()
	if root == nil || geometry == nil {
		return false, false, false, nil
	}
	openID := openSelectID(root, actions)
	if openID == 0 {
		if a.selectCapture.consume && event.Kind == platform.EventMouseUp {
			a.selectCapture = selectPointerCapture{}
			return false, true, false, nil
		}
		if event.Kind != platform.EventKeyDown || event.Key.Repeat || (event.Key.Key != platform.KeyUp && event.Key.Key != platform.KeyDown) {
			return false, false, false, nil
		}
		id := a.interaction.Focused()
		action, ok := actions[id]
		node := findInstance(root, id)
		if !ok || action.selectp == nil || node == nil || action.selectp.Disabled || len(action.selectp.Options) == 0 {
			return false, false, false, nil
		}
		openSelect(root, id, action.selectp, node)
		popup, popupErr := selectPopupGeometry(action.selectAnchor, geometry.Rect, len(action.selectp.Options), theme)
		if popupErr != nil {
			return false, true, false, popupErr
		}
		direction := 1
		if event.Key.Key == platform.KeyUp {
			direction = -1
		}
		node.State.SelectActive = nextEnabledOption(action.selectp.Options, node.State.SelectActive, direction)
		setSelectActiveKey(node, action.selectp.Options)
		ensureSelectActiveVisible(&node.State, popup)
		return true, true, false, nil
	}
	action := actions[openID]
	node := findInstance(root, openID)
	if action.selectp == nil || node == nil {
		a.selectCapture = selectPointerCapture{}
		return false, false, false, nil
	}
	props := action.selectp
	popup := action.selectOverlay
	if popup.Popup.Width <= 0 && len(props.Options) != 0 {
		popup, err = selectPopupGeometry(action.selectAnchor, geometry.Rect, len(props.Options), theme)
		if err != nil {
			return false, true, false, err
		}
	}
	switch event.Kind {
	case platform.EventWindowFocusLost:
		node.State.SelectOpen = false
		a.selectCapture = selectPointerCapture{}
		return true, false, false, nil
	case platform.EventMouseWheel:
		if !rectContains(popup.Popup, event.Pointer.X, event.Pointer.Y) {
			return false, false, false, nil
		}
		before := node.State.SelectScroll
		node.State.SelectScroll = min(max(node.State.SelectScroll-event.Pointer.WheelY*scrollWheelUnit, 0), popup.ScrollMax)
		return before != node.State.SelectScroll, true, false, nil
	case platform.EventMouseMove:
		if !finiteSelectPoint(event.Pointer.X, event.Pointer.Y) || !rectContains(popup.Popup, event.Pointer.X, event.Pointer.Y) {
			return false, false, false, nil
		}
		index := selectOptionAt(popup, node.State.SelectScroll, len(props.Options), event.Pointer.X, event.Pointer.Y)
		if index >= 0 && !props.Options[index].Disabled && node.State.SelectActive != index {
			node.State.SelectActive = index
			setSelectActiveKey(node, props.Options)
			return true, true, false, nil
		}
		return false, true, false, nil
	case platform.EventMouseDown:
		if event.Pointer.Button != platform.MouseButtonPrimary || !finiteSelectPoint(event.Pointer.X, event.Pointer.Y) {
			return false, false, false, nil
		}
		index := -1
		if rectContains(popup.Popup, event.Pointer.X, event.Pointer.Y) {
			index = selectOptionAt(popup, node.State.SelectScroll, len(props.Options), event.Pointer.X, event.Pointer.Y)
			if index >= 0 && props.Options[index].Disabled {
				index = -1
			}
		} else {
			node.State.SelectOpen = false
			dirty = true
		}
		a.selectCapture = selectPointerCapture{id: openID, index: index, consume: true}
		return dirty, true, false, nil
	case platform.EventMouseUp:
		capture := a.selectCapture
		if !capture.consume {
			return false, false, false, nil
		}
		a.selectCapture = selectPointerCapture{}
		if capture.id != openID || capture.index < 0 || capture.index >= len(props.Options) || props.Options[capture.index].Disabled {
			return false, true, false, nil
		}
		index := selectOptionAt(popup, node.State.SelectScroll, len(props.Options), event.Pointer.X, event.Pointer.Y)
		if index != capture.index {
			return false, true, false, nil
		}
		return a.selectOption(openID, node, action, index)
	case platform.EventKeyDown:
		if event.Key.Repeat {
			return false, true, false, nil
		}
		switch event.Key.Key {
		case platform.KeyEscape:
			node.State.SelectOpen = false
			a.selectCapture = selectPointerCapture{}
			return true, true, false, nil
		case platform.KeyTab:
			node.State.SelectOpen = false
			a.selectCapture = selectPointerCapture{}
			return true, false, false, nil
		case platform.KeyUp, platform.KeyDown:
			direction := 1
			if event.Key.Key == platform.KeyUp {
				direction = -1
			}
			next := nextEnabledLinear(props.Options, node.State.SelectActive, direction)
			if next >= 0 && next != node.State.SelectActive {
				node.State.SelectActive = next
				setSelectActiveKey(node, props.Options)
				ensureSelectActiveVisible(&node.State, popup)
				return true, true, false, nil
			}
			return false, true, false, nil
		case platform.KeyHome:
			next := nextEnabledLinear(props.Options, -1, 1)
			return updateSelectActive(node, props.Options, popup, next), true, false, nil
		case platform.KeyEnd:
			next := nextEnabledLinear(props.Options, len(props.Options), -1)
			return updateSelectActive(node, props.Options, popup, next), true, false, nil
		case platform.KeyEnter, platform.KeySpace:
			if node.State.SelectActive < 0 {
				return false, true, false, nil
			}
			return a.selectOption(openID, node, action, node.State.SelectActive)
		default:
			return false, true, false, nil
		}
	case platform.EventKeyUp:
		return false, true, false, nil
	}
	return false, false, false, nil
}

func (a *App) selectOption(id uint64, node *tree.Node, action inputAction, index int) (dirty, handled, displayed bool, err error) {
	props := action.selectp
	if index < 0 || index >= len(props.Options) || props.Options[index].Disabled {
		return false, true, false, nil
	}
	node.State.SelectOpen = false
	value := props.Options[index].Value
	if value == props.Value || props.OnChange == nil {
		return true, true, false, nil
	}
	if err := callNoArg("select change callback", func() { props.OnChange(value) }); err != nil {
		return true, true, false, err
	}
	a.markInputBoundary()
	displayed, err = a.runQueuedWork()
	return true, true, displayed, err
}

func openSelect(root *tree.Node, id uint64, props *SelectProps, node *tree.Node) {
	closeOtherSelects(root, id)
	node.State.SelectOpen = true
	selected := selectIndexForValue(props.Options, props.Value)
	if selected < 0 || props.Options[selected].Disabled {
		selected = nextEnabledLinear(props.Options, -1, 1)
	}
	node.State.SelectActive = selected
	setSelectActiveKey(node, props.Options)
}

func closeOtherSelects(root *tree.Node, keep uint64) {
	if root == nil {
		return
	}
	if root.ID != keep {
		root.State.SelectOpen = false
	}
	for _, child := range root.Children {
		closeOtherSelects(child, keep)
	}
}

func openSelectID(root *tree.Node, actions map[uint64]inputAction) uint64 {
	hasSelect := false
	for _, action := range actions {
		if action.selectp != nil {
			hasSelect = true
			break
		}
	}
	if !hasSelect {
		return 0
	}
	return findOpenSelectID(root, actions)
}

func findOpenSelectID(root *tree.Node, actions map[uint64]inputAction) uint64 {
	if root == nil {
		return 0
	}
	if action, ok := actions[root.ID]; ok && action.selectp != nil && root.State.SelectOpen {
		return root.ID
	}
	for _, child := range root.Children {
		if id := findOpenSelectID(child, actions); id != 0 {
			return id
		}
	}
	return 0
}

func selectOptionAt(popup selectOverlayGeometry, scroll float32, count int, x, y float32) int {
	if !rectContains(popup.Popup, x, y) || popup.ItemHeight <= 0 {
		return -1
	}
	index := int((y - popup.Popup.Y + scroll) / popup.ItemHeight)
	if index < 0 || index >= count {
		return -1
	}
	return index
}

func nextEnabledLinear(options []SelectOption, current, direction int) int {
	for index := current + direction; index >= 0 && index < len(options); index += direction {
		if !options[index].Disabled {
			return index
		}
	}
	return current
}

func updateSelectActive(node *tree.Node, options []SelectOption, popup selectOverlayGeometry, next int) bool {
	if next < 0 || next >= len(options) || next == node.State.SelectActive {
		return false
	}
	node.State.SelectActive = next
	setSelectActiveKey(node, options)
	ensureSelectActiveVisible(&node.State, popup)
	return true
}

func setSelectActiveKey(node *tree.Node, options []SelectOption) {
	node.State.SelectActiveKey = ""
	if node.State.SelectActive >= 0 && node.State.SelectActive < len(options) {
		node.State.SelectActiveKey = string(options[node.State.SelectActive].Value)
	}
}
