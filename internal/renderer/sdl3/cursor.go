package sdl3

import (
	"fmt"

	"github.com/dxui-org/dxui/internal/backend"
	"github.com/dxui-org/go-sdl3/sdl"
)

type nativeCursor interface {
	set() error
	destroy()
}

type nativeCursorAPI interface {
	createCursor(backend.Cursor) (nativeCursor, error)
}

type cursorState struct {
	values  [3]nativeCursor
	current backend.Cursor
	set     bool
}

// SetCursor shares SDL's process-wide cursor state across all App windows.
func (driver *Driver) SetCursor(shape backend.Cursor) error {
	if driver.closed {
		return fmt.Errorf("sdl3: set cursor: window closed")
	}
	if shape > backend.CursorPointer {
		return fmt.Errorf("sdl3: invalid cursor %d", shape)
	}
	owner := driver
	if driver.owner != nil {
		owner = driver.owner
	}
	state := &owner.cursors
	if state.set && state.current == shape {
		return nil
	}
	api, ok := owner.api.(nativeCursorAPI)
	if !ok {
		return nil
	}
	if state.values[shape] == nil {
		cursor, err := api.createCursor(shape)
		if err != nil {
			return fmt.Errorf("sdl3: create cursor: %w", err)
		}
		state.values[shape] = cursor
	}
	if err := state.values[shape].set(); err != nil {
		return fmt.Errorf("sdl3: set cursor: %w", err)
	}
	state.current, state.set = shape, true
	return nil
}

func (driver *Driver) closeCursors() {
	for i, cursor := range driver.cursors.values {
		if cursor != nil {
			cursor.destroy()
			driver.cursors.values[i] = nil
		}
	}
	driver.cursors.set = false
}

type systemCursor struct{ value *sdl.Cursor }

func (systemAPI) createCursor(shape backend.Cursor) (nativeCursor, error) {
	id := sdl.SYSTEM_CURSOR_DEFAULT
	if shape == backend.CursorText {
		id = sdl.SYSTEM_CURSOR_TEXT
	}
	if shape == backend.CursorPointer {
		id = sdl.SYSTEM_CURSOR_POINTER
	}
	value, err := sdl.CreateSystemCursor(id)
	if err != nil {
		return nil, err
	}
	return systemCursor{value}, nil
}

func (cursor systemCursor) set() error { return sdl.SetCursor(cursor.value) }
func (cursor systemCursor) destroy()   { cursor.value.Destroy() }
