package dxui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/backend"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
)

type fakeChildNative struct {
	closes                         int
	renders                        int
	renderErr                      error
	title                          string
	width, height                  int32
	maximizes, minimizes, restores int
}

func (f *fakeChildNative) Renderer() renderer.Renderer    { return f }
func (*fakeChildNative) Diagnostics() backend.Diagnostics { return backend.Diagnostics{} }
func (*fakeChildNative) TextInput() backend.TextInput     { return nil }
func (f *fakeChildNative) SetTitle(value string) error    { f.title = value; return nil }
func (f *fakeChildNative) SetSize(w, h int32) error       { f.width = w; f.height = h; return nil }
func (f *fakeChildNative) Maximize() error                { f.maximizes++; return nil }
func (f *fakeChildNative) Minimize() error                { f.minimizes++; return nil }
func (f *fakeChildNative) Restore() error                 { f.restores++; return nil }
func (f *fakeChildNative) Close() error                   { f.closes++; return nil }
func (f *fakeChildNative) Render(paint.DisplayList) error { f.renders++; return f.renderErr }
func (*fakeChildNative) Present() error                   { return nil }

func testChildWindow(t *testing.T, onClose func(*Window)) (*App, *Window, *fakeChildNative) {
	t.Helper()
	owner := NewApp(AppOptions{})
	owner.running = true
	child := NewApp(AppOptions{})
	child.running = true
	child.started = true
	child.root = func() View { return Label("child") }
	child.buildSize = Size{Width: 320, Height: 200}
	if _, err := child.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	native := &fakeChildNative{}
	w := &Window{owner: owner, app: child, id: 2, native: native, options: WindowOptions{OnCloseRequest: onClose}}
	owner.mainWindow = &Window{owner: owner, app: owner, id: 1}
	owner.windows[1] = owner.mainWindow
	owner.windows[2] = w
	return owner, w, native
}

func TestChildCloseRequestMayRejectThenCloseIndependently(t *testing.T) {
	requests := 0
	owner, w, native := testChildWindow(t, func(*Window) { requests++ })
	if closeLoop, err := owner.handleWindowEvent(platform.Event{Window: 2, Kind: platform.EventClose}); err != nil || closeLoop {
		t.Fatalf("close=%v err=%v", closeLoop, err)
	}
	if requests != 1 || w.Closed() || native.closes != 0 {
		t.Fatalf("request=%d closed=%v native closes=%d", requests, w.Closed(), native.closes)
	}
	w.options.OnCloseRequest = func(value *Window) { requests++; value.Close() }
	_, err := owner.handleWindowEvent(platform.Event{Window: 2, Kind: platform.EventClose})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !w.Closed() || native.closes != 1 {
		t.Fatalf("request=%d closed=%v native closes=%d", requests, w.Closed(), native.closes)
	}
	if err := w.Update(func() {}); !errors.Is(err, ErrWindowClosed) {
		t.Fatalf("update error=%v", err)
	}
	w.Close()
	if native.closes != 1 {
		t.Fatalf("repeated close count=%d", native.closes)
	}
}

func TestStaleLogicalWindowEventIsIgnored(t *testing.T) {
	owner, w, _ := testChildWindow(t, nil)
	w.Close()
	owner.removeClosedWindows()
	if closeLoop, err := owner.handleWindowEvent(platform.Event{Window: 2, Kind: platform.EventKeyDown}); err != nil || closeLoop {
		t.Fatalf("close=%v err=%v", closeLoop, err)
	}
}

type fakeWindowEvents struct{}

func (*fakeWindowEvents) Wait(context.Context, *time.Time) (platform.Event, error) {
	return platform.Event{}, errors.New("unused")
}
func (*fakeWindowEvents) Poll() (platform.Event, bool, error) { return platform.Event{}, false, nil }
func (*fakeWindowEvents) Wake() error                         { return nil }

type fakeMultiRuntime struct {
	events      *fakeWindowEvents
	main, child *fakeChildNative
	next        platform.WindowID
}

func (f *fakeMultiRuntime) Events() platform.EventSource   { return f.events }
func (f *fakeMultiRuntime) Renderer() renderer.Renderer    { return f.main }
func (*fakeMultiRuntime) Diagnostics() backend.Diagnostics { return backend.Diagnostics{} }
func (*fakeMultiRuntime) Close() error                     { return nil }
func (*fakeMultiRuntime) MainWindowID() platform.WindowID  { return 1 }
func (f *fakeMultiRuntime) CreateWindow(backend.WindowConfig) (platform.WindowID, backend.WindowRuntime, error) {
	id := f.next
	f.next++
	return id, f.child, nil
}

func TestCreateWindowFirstRenderFailureCleansUpAndRuntimeRecovers(t *testing.T) {
	failure := errors.New("render failed")
	events := &fakeWindowEvents{}
	bad := &fakeChildNative{renderErr: failure}
	multi := &fakeMultiRuntime{events: events, main: &fakeChildNative{}, child: bad, next: 2}
	app := NewApp(AppOptions{})
	app.running = true
	app.events = events
	app.multiRuntime = multi
	if _, err := app.CreateWindow(WindowOptions{Width: 320, Height: 200}, func() View { return Label("bad") }); !errors.Is(err, failure) {
		t.Fatalf("error=%v", err)
	}
	if bad.closes != 1 || len(app.windows) != 0 {
		t.Fatalf("closes=%d windows=%d", bad.closes, len(app.windows))
	}
	good := &fakeChildNative{}
	multi.child = good
	w, err := app.CreateWindow(WindowOptions{Width: 320, Height: 200}, func() View { return Label("good") })
	if err != nil {
		t.Fatal(err)
	}
	if w == nil || good.renders != 1 || good.closes != 0 || len(app.windows) != 1 {
		t.Fatalf("window=%v renders=%d closes=%d registered=%d", w, good.renders, good.closes, len(app.windows))
	}
}

func TestWindowPropertiesCommandsAndNativeStateEvents(t *testing.T) {
	owner, w, native := testChildWindow(t, nil)
	w.options.Title = "old"
	w.options.Width = 320
	w.options.Height = 200
	if err := w.SetTitle("new"); err != nil {
		t.Fatal(err)
	}
	if w.Title() != "new" || native.title != "new" {
		t.Fatalf("titles=%q/%q", w.Title(), native.title)
	}
	if err := w.SetSize(401, 299); err != nil {
		t.Fatal(err)
	}
	if got := w.Size(); got != (Size{401, 299}) || native.width != 401 || native.height != 299 {
		t.Fatalf("size=%+v native=%dx%d", got, native.width, native.height)
	}
	if err := w.Maximize(); err != nil {
		t.Fatal(err)
	}
	_, _ = owner.handleWindowEvent(platform.Event{Window: 2, Kind: platform.EventWindowMaximized})
	if !w.IsMaximized() || w.IsMinimized() {
		t.Fatal("maximize state not routed")
	}
	if err := w.Unmaximize(); err != nil {
		t.Fatal(err)
	}
	_, _ = owner.handleWindowEvent(platform.Event{Window: 2, Kind: platform.EventWindowRestored})
	if w.IsMaximized() || w.IsMinimized() {
		t.Fatal("restore did not clear state")
	}
	if err := w.Minimize(); err != nil {
		t.Fatal(err)
	}
	_, _ = owner.handleWindowEvent(platform.Event{Window: 2, Kind: platform.EventWindowMinimized})
	if !w.IsMinimized() || w.IsMaximized() {
		t.Fatal("minimize state not routed")
	}
	if err := w.Unminimize(); err != nil {
		t.Fatal(err)
	}
	if native.maximizes != 1 || native.minimizes != 1 || native.restores != 2 {
		t.Fatalf("commands=%d/%d/%d", native.maximizes, native.minimizes, native.restores)
	}
	_, _ = owner.handleWindowEvent(platform.Event{Window: 2, Kind: platform.EventResize, Viewport: platform.Viewport{LogicalWidth: 500, LogicalHeight: 350}})
	if got := w.Size(); got != (Size{500, 350}) {
		t.Fatalf("event size=%+v", got)
	}
}
