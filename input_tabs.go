package dxui

import (
	"github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/platform"
)

func (a *App) reconcileTabsCapture(actions map[uint64]inputAction) {
	if a.tabsCapture.id == 0 {
		return
	}
	action, live := actions[a.tabsCapture.id]
	node := findInstance(a.retained.Root(), a.tabsCapture.id)
	if !live || action.tabs == nil || node == nil || node.Properties.Semantics.Disabled || a.tabsCapture.index < 0 || a.tabsCapture.index >= len(action.tabs.Items) || action.tabs.Items[a.tabsCapture.index].Disabled {
		if node != nil {
			node.State.TabsPressed = -1
		}
		a.tabsCapture = tabsPointerCapture{}
	}
}

func (a *App) handleTabsEvent(event platform.Event, result input.Result) (dirty, handled, displayed bool, err error) {
	root := a.retained.Root()
	a.mu.Lock()
	actions := a.inputActions
	a.mu.Unlock()
	if root == nil || len(actions) == 0 {
		return false, false, false, nil
	}
	clearHover := func(keep uint64, index int) bool {
		changed := false
		for id, action := range actions {
			if action.tabs == nil {
				continue
			}
			node := findInstance(root, id)
			if node == nil {
				continue
			}
			next := -1
			if id == keep {
				next = index
			}
			if node.State.TabsHover != next {
				node.State.TabsHover = next
				changed = true
			}
		}
		return changed
	}
	clearPressed := func() bool {
		changed := false
		for id, action := range actions {
			if action.tabs == nil {
				continue
			}
			if node := findInstance(root, id); node != nil && node.State.TabsPressed != -1 {
				node.State.TabsPressed = -1
				changed = true
			}
		}
		return changed
	}

	switch event.Kind {
	case platform.EventWindowFocusLost:
		a.tabsCapture = tabsPointerCapture{}
		hoverChanged := clearHover(0, -1)
		pressedChanged := clearPressed()
		return hoverChanged || pressedChanged, false, false, nil
	case platform.EventWindowMouseLeave:
		hoverChanged := clearHover(0, -1)
		pressedChanged := clearPressed()
		return hoverChanged || pressedChanged, false, false, nil
	case platform.EventMouseMove:
		action := actions[result.Target]
		index := tabItemAt(action, event.Pointer.X, event.Pointer.Y)
		if index >= 0 && action.tabs.Items[index].Disabled {
			index = -1
		}
		dirty = clearHover(result.Target, index)
		if a.tabsCapture.id != 0 {
			node := findInstance(root, a.tabsCapture.id)
			pressed := -1
			if result.Target == a.tabsCapture.id && index == a.tabsCapture.index {
				pressed = index
			}
			if node != nil && node.State.TabsPressed != pressed {
				node.State.TabsPressed = pressed
				dirty = true
			}
			return dirty, true, false, nil
		}
		return dirty, false, false, nil
	case platform.EventMouseDown:
		action, ok := actions[result.Target]
		if !ok || action.tabs == nil || action.tabs.Disabled {
			return false, false, false, nil
		}
		index := tabItemAt(action, event.Pointer.X, event.Pointer.Y)
		node := findInstance(root, result.Target)
		if node == nil || index < 0 || index >= len(action.tabs.Items) || action.tabs.Items[index].Disabled {
			a.tabsCapture = tabsPointerCapture{}
			return false, true, false, nil
		}
		a.tabsCapture = tabsPointerCapture{id: result.Target, index: index}
		if node.State.TabsActive != index || node.State.TabsPressed != index {
			node.State.TabsActive = index
			node.State.TabsActiveKey = action.tabs.Items[index].Value
			node.State.TabsPressed = index
			dirty = true
		}
		return dirty, true, false, nil
	case platform.EventMouseUp:
		capture := a.tabsCapture
		if capture.id == 0 {
			return false, false, false, nil
		}
		a.tabsCapture = tabsPointerCapture{}
		dirty = clearPressed()
		action, live := actions[capture.id]
		if !live || action.tabs == nil || result.Activate != capture.id || tabItemAt(action, event.Pointer.X, event.Pointer.Y) != capture.index {
			return dirty, true, false, nil
		}
		displayed, err = a.activateTab(action, capture.index)
		return true, true, displayed, err
	case platform.EventKeyDown:
		if event.Key.Repeat {
			return false, false, false, nil
		}
		id := a.interaction.Focused()
		action, ok := actions[id]
		node := findInstance(root, id)
		if !ok || action.tabs == nil || node == nil || action.tabs.Disabled {
			return false, false, false, nil
		}
		next := node.State.TabsActive
		switch event.Key.Key {
		case platform.KeyLeft:
			if next < 0 {
				next = len(action.tabs.Items)
			}
			next = nextEnabledTab(action.tabs.Items, next, -1)
		case platform.KeyRight:
			next = nextEnabledTab(action.tabs.Items, next, 1)
		case platform.KeyHome:
			next = nextEnabledTab(action.tabs.Items, -1, 1)
		case platform.KeyEnd:
			next = nextEnabledTab(action.tabs.Items, len(action.tabs.Items), -1)
		case platform.KeyEnter, platform.KeySpace:
			if next >= 0 && next < len(action.tabs.Items) && !action.tabs.Items[next].Disabled && node.State.TabsPressed != next {
				node.State.TabsPressed = next
				return true, true, false, nil
			}
			return false, true, false, nil
		default:
			return false, false, false, nil
		}
		if next >= 0 && next < len(action.tabs.Items) && next != node.State.TabsActive {
			node.State.TabsActive = next
			node.State.TabsActiveKey = action.tabs.Items[next].Value
			return true, true, false, nil
		}
		return false, true, false, nil
	case platform.EventKeyUp:
		id := a.interaction.Focused()
		action, ok := actions[id]
		node := findInstance(root, id)
		if !ok || action.tabs == nil || node == nil || (event.Key.Key != platform.KeyEnter && event.Key.Key != platform.KeySpace) {
			return false, false, false, nil
		}
		if node.State.TabsPressed != -1 {
			node.State.TabsPressed = -1
			dirty = true
		}
		if result.Activate != id {
			return dirty, true, false, nil
		}
		displayed, err = a.activateTab(action, node.State.TabsActive)
		return true, true, displayed, err
	}
	return false, false, false, nil
}

func tabItemAt(action inputAction, x, y float32) int {
	if action.tabs == nil || action.tabs.Disabled {
		return -1
	}
	for index, rect := range action.tabItems {
		if rectContains(rect, x, y) {
			return index
		}
	}
	return -1
}

func (a *App) activateTab(action inputAction, index int) (bool, error) {
	if action.tabs == nil || action.tabs.Disabled || index < 0 || index >= len(action.tabs.Items) {
		return false, nil
	}
	item := action.tabs.Items[index]
	if item.Disabled || item.Value == action.tabs.Value || action.tabs.OnChange == nil {
		return false, nil
	}
	if err := callNoArg("tabs change callback", func() { action.tabs.OnChange(item.Value) }); err != nil {
		return false, err
	}
	a.markInputBoundary()
	return a.runQueuedWork()
}
