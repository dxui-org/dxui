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
	return a.setPointerCursor(a.interaction.CursorAt(a.cursorX, a.cursorY))
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

// resolveCursor translates public choices without exposing backend values.
func resolveCursor(kind viewKind, cursor Cursor) backend.Cursor {
	switch cursor {
	case CursorDefault:
		return backend.CursorDefault
	case CursorText:
		return backend.CursorText
	case CursorPointer:
		return backend.CursorPointer
	}
	switch kind {
	case viewInput, viewTextarea:
		return backend.CursorText
	case viewButton:
		return backend.CursorPointer
	default:
		return backend.CursorDefault
	}
}
