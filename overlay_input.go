package dxui

import (
	"time"

	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

type overlayPointerCapture struct {
	id      uint64
	consume bool
}

func (a *App) handlePopoverOverlayEvent(event platform.Event) (dirty, handled, displayed bool, err error) {
	if a.overlayCapture.consume && event.Kind == platform.EventMouseUp {
		a.overlayCapture = overlayPointerCapture{}
		return false, true, false, nil
	}
	var topID uint64
	var action inputAction
	for id, candidate := range a.inputActions {
		if candidate.popover != nil && candidate.popover.Open && candidate.overlayLayer > action.overlayLayer {
			topID, action = id, candidate
		}
	}
	if topID == 0 {
		return false, false, false, nil
	}
	switch event.Kind {
	case platform.EventKeyDown:
		if event.Key.Repeat || event.Key.Key != platform.KeyEscape {
			return false, false, false, nil
		}
		a.interaction.Focus(topID, true)
		return a.emitPopoverChange(topID, action.popover, false)
	case platform.EventMouseDown:
		if event.Pointer.Button != platform.MouseButtonPrimary || !finiteSelectPoint(event.Pointer.X, event.Pointer.Y) {
			return false, false, false, nil
		}
		if rectContains(paintRect(action.overlayPopup), event.Pointer.X, event.Pointer.Y) || rectContains(paintRect(action.overlayAnchor), event.Pointer.X, event.Pointer.Y) {
			return false, false, false, nil
		}
		a.overlayCapture = overlayPointerCapture{id: topID, consume: true}
		a.interaction.Focus(topID, false)
		return a.emitPopoverChange(topID, action.popover, false)
	}
	return false, false, false, nil
}

func (a *App) hasOpenPopoverOverlay() bool {
	for _, action := range a.inputActions {
		if action.popover != nil && action.popover.Open {
			return true
		}
	}
	return false
}

func (a *App) emitPopoverChange(id uint64, props *PopoverProps, open bool) (dirty, handled, displayed bool, err error) {
	if props == nil || props.Open == open {
		return false, true, false, nil
	}
	if props.OnOpenChange == nil {
		return false, true, false, nil
	}
	callback := props.OnOpenChange
	if err := callNoArg("popover open change callback", func() { callback(open) }); err != nil {
		return false, true, false, err
	}
	a.markInputBoundary()
	displayed, err = a.runQueuedWork()
	return true, true, displayed, err
}

func (a *App) syncTooltipTriggers(event platform.Event) bool {
	now := a.clock.Now()
	changed := false
	for id, action := range a.inputActions {
		if action.tooltip == nil {
			continue
		}
		node := findInstance(a.retained.Root(), id)
		if node == nil {
			continue
		}
		props := action.tooltip
		oldOpen := node.State.TooltipOpen
		switch event.Kind {
		case platform.EventMouseMove, platform.EventMouseDown, platform.EventMouseUp:
			node.State.TooltipHover = finiteSelectPoint(event.Pointer.X, event.Pointer.Y) && rectContains(paintRect(action.overlayAnchor), event.Pointer.X, event.Pointer.Y)
		case platform.EventWindowMouseLeave, platform.EventWindowFocusLost:
			node.State.TooltipHover = false
		}
		focused := action.tooltipFocus != 0 && a.interaction.Focused() == action.tooltipFocus && a.windowFocused
		triggered := !props.Disabled && (node.State.TooltipHover || focused)
		if !triggered {
			node.State.TooltipOpen = false
			node.State.TooltipDueNS = 0
		} else if !node.State.TooltipOpen && node.State.TooltipDueNS == 0 {
			delay := props.Delay
			if delay == 0 {
				delay = defaultTooltipDelay
			}
			node.State.TooltipDueNS = now.Add(delay).UnixNano()
		}
		changed = changed || oldOpen != node.State.TooltipOpen
	}
	return changed
}

func (a *App) syncTooltipLifecycle(actions map[uint64]inputAction) bool {
	changed := false
	var visit func(*tree.Node)
	visit = func(node *tree.Node) {
		if node == nil {
			return
		}
		action, live := actions[node.ID]
		if node.State.TooltipOpen || node.State.TooltipDueNS != 0 || node.State.TooltipHover {
			if !live || action.tooltip == nil || action.tooltip.Disabled {
				changed = changed || node.State.TooltipOpen
				node.State.TooltipOpen = false
				node.State.TooltipDueNS = 0
				node.State.TooltipHover = false
			}
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(a.retained.Root())
	return changed
}

func (a *App) nextTooltipDeadline() *time.Time {
	var next *time.Time
	for id, action := range a.inputActions {
		if action.tooltip == nil || action.tooltip.Disabled {
			continue
		}
		node := findInstance(a.retained.Root(), id)
		if node == nil || node.State.TooltipDueNS == 0 || node.State.TooltipOpen {
			continue
		}
		value := time.Unix(0, node.State.TooltipDueNS)
		if next == nil || value.Before(*next) {
			copyValue := value
			next = &copyValue
		}
	}
	return next
}

func (a *App) handleTooltipDeadline(now time.Time) bool {
	changed := false
	for id, action := range a.inputActions {
		if action.tooltip == nil || action.tooltip.Disabled {
			continue
		}
		node := findInstance(a.retained.Root(), id)
		if node == nil || node.State.TooltipDueNS == 0 || node.State.TooltipOpen || now.UnixNano() < node.State.TooltipDueNS {
			continue
		}
		focused := action.tooltipFocus != 0 && a.interaction.Focused() == action.tooltipFocus && a.windowFocused
		if node.State.TooltipHover || focused {
			node.State.TooltipOpen = true
			changed = true
		}
		node.State.TooltipDueNS = 0
	}
	return changed
}

func earlierDeadline(first, second *time.Time) *time.Time {
	if first == nil {
		return second
	}
	if second == nil || first.Before(*second) {
		return first
	}
	return second
}
