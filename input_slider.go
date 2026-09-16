package dxui

import (
	"fmt"

	internalinput "github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/platform"
)

func (a *App) handleSliderEvent(event platform.Event, result internalinput.Result) (dirty, handled, displayed bool, err error) {
	a.mu.Lock()
	actions := a.inputActions
	theme := a.theme
	a.mu.Unlock()

	cancel := func() bool {
		if a.sliderDrag.id == 0 {
			return false
		}
		node := findInstance(a.retained.Root(), a.sliderDrag.id)
		changed := false
		if node != nil && node.State.SliderDragging {
			node.State.SliderDragging = false
			changed = true
		}
		a.sliderDrag = sliderDrag{}
		return changed
	}

	if event.Kind == platform.EventWindowFocusLost || event.Kind == platform.EventKeyDown && event.Key.Key == platform.KeyEscape {
		captured := a.sliderDrag.id != 0
		return cancel(), captured, false, nil
	}
	if event.Kind == platform.EventMouseUp && event.Pointer.Button == platform.MouseButtonPrimary && a.sliderDrag.id != 0 {
		return cancel(), true, false, nil
	}
	if event.Kind == platform.EventKeyDown && !event.Key.Repeat {
		action, ok := actions[a.interaction.Focused()]
		if !ok || action.slider == nil || !action.sliderModel.valid || action.slider.Disabled {
			return false, false, false, nil
		}
		var proposal float32
		switch event.Key.Key {
		case platform.KeyLeft, platform.KeyDown:
			proposal = action.sliderModel.stepBy(-1)
		case platform.KeyRight, platform.KeyUp:
			proposal = action.sliderModel.stepBy(1)
		case platform.KeyHome:
			proposal = action.sliderModel.min
		case platform.KeyEnd:
			proposal = action.sliderModel.max
		default:
			return false, false, false, nil
		}
		displayed, err = a.emitSliderProposal(action, proposal)
		return false, true, displayed, err
	}
	if event.Kind == platform.EventMouseDown {
		action, ok := actions[result.Target]
		if !ok || action.slider == nil || !action.sliderModel.valid || action.slider.Disabled {
			return false, false, false, nil
		}
		node := findInstance(a.retained.Root(), result.Target)
		if node == nil {
			return false, false, false, nil
		}
		a.sliderDrag = sliderDrag{id: result.Target}
		node.State.SliderDragging = true
		proposal, valueErr := sliderPointerProposal(action, event.Pointer.X, theme)
		if valueErr != nil {
			return true, true, false, valueErr
		}
		displayed, err = a.emitSliderProposal(action, proposal)
		return true, true, displayed, err
	}
	if event.Kind == platform.EventMouseMove && a.sliderDrag.id != 0 {
		action, ok := actions[a.sliderDrag.id]
		node := findInstance(a.retained.Root(), a.sliderDrag.id)
		if !ok || action.slider == nil || node == nil || node.Properties.Semantics.Disabled {
			return cancel(), true, false, nil
		}
		proposal, valueErr := sliderPointerProposal(action, event.Pointer.X, theme)
		if valueErr != nil {
			return false, true, false, valueErr
		}
		displayed, err = a.emitSliderProposal(action, proposal)
		return false, true, displayed, err
	}
	return false, false, false, nil
}

func sliderPointerProposal(action inputAction, pointerX float32, theme resolvedTheme) (float32, error) {
	if !finite(pointerX) {
		return action.sliderModel.value, nil
	}
	thumbSize, err := theme.metric(TokenMetric(MetricComponentSliderThumbSize))
	if err != nil {
		return 0, fmt.Errorf("slider thumb size: %w", err)
	}
	rect := paintRect(action.geometry)
	thumbSize = min(max(float32(0), thumbSize), min(max(float32(0), rect.Width), max(float32(0), rect.Height)))
	start, end := sliderTravel(rect.X, rect.Width, thumbSize)
	if end <= start || pointerX <= start {
		return action.sliderModel.min, nil
	}
	if pointerX >= end {
		return action.sliderModel.max, nil
	}
	fraction := float64((pointerX - start) / (end - start))
	candidate := float64(action.sliderModel.min) + fraction*float64(action.sliderModel.max-action.sliderModel.min)
	return action.sliderModel.proposal(candidate, false), nil
}

func (a *App) emitSliderProposal(action inputAction, proposal float32) (bool, error) {
	if action.slider == nil || action.slider.OnChange == nil || proposal == action.sliderModel.value {
		return false, nil
	}
	if err := callNoArg("slider change callback", func() { action.slider.OnChange(proposal) }); err != nil {
		return false, err
	}
	a.markInputBoundary()
	return a.runQueuedWork()
}
