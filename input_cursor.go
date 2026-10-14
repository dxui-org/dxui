package dxui

import (
	"fmt"

	"github.com/dxui-org/dxui/internal/backend"
	"github.com/dxui-org/dxui/internal/platform"
)

func (a *App) updatePointerCursor(event platform.Event) error {
	switch event.Kind {
	case platform.EventMouseMove, platform.EventMouseDown, platform.EventMouseUp, platform.EventMouseWheel:
		if !finitePoint(Point{X: event.Pointer.X, Y: event.Pointer.Y}) {
			return nil
		}
		a.cursorX, a.cursorY, a.cursorKnown = event.Pointer.X, event.Pointer.Y, true
		return a.syncPointerCursor()
	case platform.EventWindowMouseLeave:
		a.cursorKnown = false
		return a.setPointerCursor(backend.CursorDefault)
	}
	return nil
}

func (a *App) syncPointerCursor() error {
	if !a.cursorKnown {
		return nil
	}
	cursor := backend.CursorDefault
	id := a.interaction.TargetAt(a.cursorX, a.cursorY)
	if action := a.inputActions[id]; action.editor && !action.disabled {
		cursor = backend.CursorText
	}
	// Select options are painted in an overlay outside the ordinary hit snapshot.
	if id := openSelectID(a.retained.Root(), a.inputActions); id != 0 {
		if rectContains(a.inputActions[id].selectOverlay.Popup, a.cursorX, a.cursorY) {
			cursor = backend.CursorDefault
		}
	}
	return a.setPointerCursor(cursor)
}

func (a *App) setPointerCursor(cursor backend.Cursor) error {
	if native, ok := a.backend.(backend.CursorRuntime); ok {
		if err := native.SetCursor(cursor); err != nil {
			return fmt.Errorf("dxui: set pointer cursor: %w", err)
		}
	}
	return nil
}

func (r childRuntime) SetCursor(cursor backend.Cursor) error {
	if native, ok := r.WindowRuntime.(backend.CursorRuntime); ok {
		return native.SetCursor(cursor)
	}
	return nil
}
