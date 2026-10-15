package sdl3

import (
	"errors"
	"slices"
	"testing"

	"github.com/dxui-org/dxui/internal/backend"
)

type cursorAPI struct {
	*fakeAPI
	creates, sets, destroys int
	setErr                  error
}

type fakeCursor struct{ api *cursorAPI }

func (api *cursorAPI) createCursor(backend.Cursor) (nativeCursor, error) {
	api.creates++
	return fakeCursor{api}, nil
}
func (cursor fakeCursor) set() error { cursor.api.sets++; return cursor.api.setErr }
func (cursor fakeCursor) destroy() {
	if slices.Contains(cursor.api.actions, "quit") {
		panic("cursor destroyed after SDL quit")
	}
	cursor.api.destroys++
}

func TestCursorSharedAcrossWindowsAndDestroyedBeforeQuit(t *testing.T) {
	api := &cursorAPI{fakeAPI: newFakeAPI()}
	driver, err := openWithAPI(api, Config{Width: 100, Height: 50})
	if err != nil {
		t.Fatal(err)
	}
	child := &Driver{owner: driver}
	for _, step := range []struct {
		driver *Driver
		cursor backend.Cursor
	}{
		{driver, backend.CursorText}, {driver, backend.CursorText},
		{child, backend.CursorDefault}, {driver, backend.CursorText},
		{child, backend.CursorPointer}, {driver, backend.CursorPointer},
	} {
		if err := step.driver.SetCursor(step.cursor); err != nil {
			t.Fatal(err)
		}
	}
	if api.creates != 3 || api.sets != 4 {
		t.Fatalf("creates/sets = %d/%d", api.creates, api.sets)
	}
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	if api.destroys != 3 {
		t.Fatalf("destroyed = %d", api.destroys)
	}
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	if api.destroys != 3 {
		t.Fatal("cursors destroyed twice")
	}
}

func TestCursorFailureCanRetry(t *testing.T) {
	api := &cursorAPI{fakeAPI: newFakeAPI(), setErr: errors.New("set failed")}
	driver := &Driver{api: api}
	defer driver.closeCursors()
	if err := driver.SetCursor(backend.CursorText); err == nil {
		t.Fatal("missing set error")
	}
	api.setErr = nil
	if err := driver.SetCursor(backend.CursorText); err != nil {
		t.Fatal(err)
	}
	if api.creates != 1 || api.sets != 2 {
		t.Fatal("failed set prevented retry or recreated cursor")
	}
}

func TestInvalidCursorDoesNotCreateNativeResource(t *testing.T) {
	api := &cursorAPI{fakeAPI: newFakeAPI()}
	driver := &Driver{api: api}
	if err := driver.SetCursor(backend.Cursor(255)); err == nil {
		t.Fatal("invalid shape accepted")
	}
	if api.creates != 0 || api.sets != 0 {
		t.Fatal("invalid shape touched native cursors")
	}
}
