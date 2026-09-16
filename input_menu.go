package dxui

import (
	"github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/platform"
)

func (a *App) reconcileMenuCapture(actions map[uint64]inputAction) {
	if a.menuCapture.id == 0 {
		return
	}
	action, live := actions[a.menuCapture.id]
	node := findInstance(a.retained.Root(), a.menuCapture.id)
	if !live || action.menu == nil || node == nil || node.Properties.Semantics.Disabled || a.menuCapture.index < 0 || a.menuCapture.index >= len(action.menu.Items) || action.menu.Items[a.menuCapture.index].Disabled || action.menu.Items[a.menuCapture.index].Value != a.menuCapture.value {
		if node != nil {
			node.State.MenuPressed = -1
		}
		a.menuCapture = menuPointerCapture{}
	}
}

func (a *App) handleMenuEvent(event platform.Event, result input.Result) (dirty, handled, displayed bool, err error) {
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
			if action.menu == nil {
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
			if node.State.MenuHover != next {
				node.State.MenuHover = next
				changed = true
			}
		}
		return changed
	}
	clearPressed := func() bool {
		changed := false
		for id, action := range actions {
			if action.menu == nil {
				continue
			}
			if node := findInstance(root, id); node != nil && node.State.MenuPressed != -1 {
				node.State.MenuPressed = -1
				changed = true
			}
		}
		return changed
	}

	switch event.Kind {
	case platform.EventWindowFocusLost:
		a.menuCapture = menuPointerCapture{}
		hoverChanged := clearHover(0, -1)
		pressedChanged := clearPressed()
		return hoverChanged || pressedChanged, false, false, nil
	case platform.EventWindowMouseLeave:
		hoverChanged := clearHover(0, -1)
		pressedChanged := clearPressed()
		return hoverChanged || pressedChanged, false, false, nil
	case platform.EventMouseMove:
		action := actions[result.Target]
		index := menuItemAt(action, event.Pointer.X, event.Pointer.Y)
		if index >= 0 && action.menu.Items[index].Disabled {
			index = -1
		}
		dirty = clearHover(result.Target, index)
		if a.menuCapture.id != 0 {
			node := findInstance(root, a.menuCapture.id)
			pressed := -1
			if result.Target == a.menuCapture.id && index == a.menuCapture.index {
				pressed = index
			}
			if node != nil && node.State.MenuPressed != pressed {
				node.State.MenuPressed = pressed
				dirty = true
			}
			return dirty, true, false, nil
		}
		return dirty, false, false, nil
	case platform.EventMouseDown:
		action, ok := actions[result.Target]
		if !ok || action.menu == nil || action.menu.Disabled {
			return false, false, false, nil
		}
		index := menuItemAt(action, event.Pointer.X, event.Pointer.Y)
		node := findInstance(root, result.Target)
		if node == nil || index < 0 || index >= len(action.menu.Items) || action.menu.Items[index].Disabled {
			a.menuCapture = menuPointerCapture{}
			return false, true, false, nil
		}
		a.menuCapture = menuPointerCapture{id: result.Target, index: index, value: action.menu.Items[index].Value}
		if node.State.MenuActive != index || node.State.MenuPressed != index {
			node.State.MenuActive = index
			node.State.MenuActiveKey = action.menu.Items[index].Value
			node.State.MenuPressed = index
			dirty = true
		}
		return dirty, true, false, nil
	case platform.EventMouseUp:
		capture := a.menuCapture
		if capture.id == 0 {
			return false, false, false, nil
		}
		a.menuCapture = menuPointerCapture{}
		dirty = clearPressed()
		action, live := actions[capture.id]
		if !live || action.menu == nil || capture.index < 0 || capture.index >= len(action.menu.Items) || action.menu.Items[capture.index].Value != capture.value || result.Activate != capture.id || menuItemAt(action, event.Pointer.X, event.Pointer.Y) != capture.index {
			return dirty, true, false, nil
		}
		displayed, err = a.activateMenu(action, capture.index)
		return true, true, displayed, err
	case platform.EventKeyDown:
		if event.Key.Repeat {
			return false, false, false, nil
		}
		id := a.interaction.Focused()
		action, ok := actions[id]
		node := findInstance(root, id)
		if !ok || action.menu == nil || node == nil || action.menu.Disabled {
			return false, false, false, nil
		}
		next := node.State.MenuActive
		direction := 0
		switch event.Key.Key {
		case platform.KeyUp:
			if action.menu.Orientation != MenuVertical {
				return false, false, false, nil
			}
			direction = -1
		case platform.KeyDown:
			if action.menu.Orientation != MenuVertical {
				return false, false, false, nil
			}
			direction = 1
		case platform.KeyLeft:
			if action.menu.Orientation != MenuHorizontal {
				return false, false, false, nil
			}
			direction = -1
		case platform.KeyRight:
			if action.menu.Orientation != MenuHorizontal {
				return false, false, false, nil
			}
			direction = 1
		case platform.KeyHome:
			next = nextEnabledMenuItem(action.menu.Items, -1, 1)
		case platform.KeyEnd:
			next = nextEnabledMenuItem(action.menu.Items, len(action.menu.Items), -1)
		case platform.KeyEnter, platform.KeySpace:
			if next >= 0 && next < len(action.menu.Items) && !action.menu.Items[next].Disabled && node.State.MenuPressed != next {
				node.State.MenuPressed = next
				return true, true, false, nil
			}
			return false, true, false, nil
		default:
			return false, false, false, nil
		}
		if direction != 0 {
			if next < 0 {
				if direction < 0 {
					next = len(action.menu.Items)
				}
			}
			next = nextEnabledMenuItem(action.menu.Items, next, direction)
		}
		if next >= 0 && next < len(action.menu.Items) && next != node.State.MenuActive {
			node.State.MenuActive = next
			node.State.MenuActiveKey = action.menu.Items[next].Value
			return true, true, false, nil
		}
		return false, true, false, nil
	case platform.EventKeyUp:
		id := a.interaction.Focused()
		action, ok := actions[id]
		node := findInstance(root, id)
		if !ok || action.menu == nil || node == nil || (event.Key.Key != platform.KeyEnter && event.Key.Key != platform.KeySpace) {
			return false, false, false, nil
		}
		if node.State.MenuPressed != -1 {
			node.State.MenuPressed = -1
			dirty = true
		}
		if result.Activate != id {
			return dirty, true, false, nil
		}
		displayed, err = a.activateMenu(action, node.State.MenuActive)
		return true, true, displayed, err
	}
	return false, false, false, nil
}

func menuItemAt(action inputAction, x, y float32) int {
	if action.menu == nil || action.menu.Disabled {
		return -1
	}
	for index, rect := range action.menuItems {
		if rectContains(rect, x, y) {
			return index
		}
	}
	return -1
}

func (a *App) activateMenu(action inputAction, index int) (bool, error) {
	if action.menu == nil || action.menu.Disabled || index < 0 || index >= len(action.menu.Items) {
		return false, nil
	}
	item := action.menu.Items[index]
	if item.Disabled || action.menu.OnAction == nil {
		return false, nil
	}
	if err := callNoArg("menu action callback", func() { action.menu.OnAction(item.Value) }); err != nil {
		return false, err
	}
	a.markInputBoundary()
	return a.runQueuedWork()
}
