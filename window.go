package dxui

import (
	"errors"
	"fmt"
	"sync"

	"github.com/dxui-org/dxui/internal/backend"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
)

var (
	// ErrWindowClosed reports an operation on a child window after close.
	ErrWindowClosed = errors.New("dxui: window is closed")
	// ErrWindowNotRunning reports child-window creation outside App.Run.
	ErrWindowNotRunning = errors.New("dxui: window runtime is not running")
)

// Window is an opaque, concurrency-safe handle to a child native window.
// Component construction remains on the root dxui API; a Window only owns
// window-level lifecycle and scheduling operations.
type Window struct {
	mu        sync.Mutex
	owner     *App
	app       *App
	id        platform.WindowID
	native    backend.WindowRuntime
	options   WindowOptions
	dirty     bool
	closed    bool
	shown     bool
	maximized bool
	minimized bool
}

type childRuntime struct {
	backend.WindowRuntime
	events platform.EventSource
}

func (r childRuntime) Events() platform.EventSource { return r.events }
func (r childRuntime) Renderer() renderer.Renderer  { return r.WindowRuntime.Renderer() }

// CreateWindow creates a hidden native child window and prepares its complete
// first frame. It must be called from a UI callback or App.Update closure.
// Failure releases every partially created child resource and leaves existing
// windows unchanged.
func (a *App) CreateWindow(options WindowOptions, root func() View) (*Window, error) {
	if root == nil {
		return nil, errors.New("dxui: nil child window root builder")
	}
	window := windowOptions{Title: options.Title, Width: options.Width, Height: options.Height, MinWidth: options.MinWidth, MinHeight: options.MinHeight, Background: options.Background}
	if err := validateWindowOptions(window); err != nil {
		return nil, err
	}
	if err := validateShortcuts(options.Shortcuts); err != nil {
		return nil, err
	}
	a.mu.Lock()
	multi, running, closing := a.multiRuntime, a.running, a.closed
	events := a.events
	a.mu.Unlock()
	if !running || multi == nil {
		return nil, ErrWindowNotRunning
	}
	if closing {
		return nil, ErrAppClosed
	}
	w, err := logicalDimension("width", options.Width, 640)
	if err != nil {
		return nil, err
	}
	h, err := logicalDimension("height", options.Height, 480)
	if err != nil {
		return nil, err
	}
	mw, err := optionalDimension("minimum width", options.MinWidth)
	if err != nil {
		return nil, err
	}
	mh, err := optionalDimension("minimum height", options.MinHeight)
	if err != nil {
		return nil, err
	}
	bg := options.Background
	if bg == (RGBAColor{}) {
		bg = a.options.Background
	}
	if bg == (RGBAColor{}) {
		bg = RGBA(28, 30, 36, 255)
	}
	title := options.Title
	if title == "" {
		title = "dxui"
	}
	id, native, err := multi.CreateWindow(backend.WindowConfig{Title: title, Width: w, Height: h, MinWidth: mw, MinHeight: mh, Background: [4]uint8{bg.R, bg.G, bg.B, bg.A}, Software: a.options.Renderer == RendererSoftware, ShadowCacheBytes: a.options.Caches.ShadowBytes, GlyphCacheBytes: a.options.Caches.GlyphBytes, ImageCacheBytes: a.options.Caches.ImageBytes, Diagnostics: a.options.Diagnostics})
	if err != nil {
		return nil, fmt.Errorf("dxui: create child window: %w", err)
	}
	childOptions := a.options
	childOptions.Title, childOptions.Width, childOptions.Height = title, options.Width, options.Height
	childOptions.MinWidth, childOptions.MinHeight, childOptions.Background = options.MinWidth, options.MinHeight, bg
	childOptions.Shortcuts = append([]Shortcut(nil), options.Shortcuts...)
	childOptions.OnCloseRequest, childOptions.OnShown = nil, nil
	child := NewApp(childOptions)
	child.mu.Lock()
	child.started = true
	child.running = true
	child.root = root
	child.events = events
	child.backend = childRuntime{WindowRuntime: native, events: events}
	child.buildSize = Size{Width: float32(w), Height: float32(h)}
	child.mu.Unlock()
	options.Title = title
	options.Width = float32(w)
	options.Height = float32(h)
	handle := &Window{owner: a, app: child, id: id, native: native, options: options, dirty: true}
	if _, err := child.textEngine(); err != nil {
		_ = native.Close()
		return nil, fmt.Errorf("dxui: child text: %w", err)
	}
	if _, err := child.buildAndCommit(); err != nil {
		child.releaseTextEngine()
		_ = native.Close()
		return nil, fmt.Errorf("dxui: child first frame: %w", err)
	}
	if err := native.Renderer().Render(child.currentDisplay().Sorted()); err != nil {
		child.retained.Unmount()
		child.images.Clear()
		child.releaseTextEngine()
		_ = native.Close()
		return nil, fmt.Errorf("dxui: child first frame render: %w", err)
	}
	if err := native.Renderer().Present(); err != nil {
		child.retained.Unmount()
		child.images.Clear()
		child.releaseTextEngine()
		_ = native.Close()
		return nil, fmt.Errorf("dxui: child first frame present/show: %w", err)
	}
	handle.dirty = false
	handle.shown = true
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		child.retained.Unmount()
		child.releaseTextEngine()
		_ = native.Close()
		return nil, ErrAppClosed
	}
	a.windows[id] = handle
	a.mu.Unlock()
	if options.OnShown != nil {
		if err := callNoArg("child shown callback", func() { options.OnShown(handle) }); err != nil {
			a.mu.Lock()
			delete(a.windows, id)
			a.mu.Unlock()
			_ = handle.destroy()
			return nil, err
		}
	}
	return handle, nil
}

// Update queues a child-window state mutation and rebuilds only that window.
func (w *Window) Update(update func()) error {
	if w == nil {
		return ErrWindowClosed
	}
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	if closed {
		return ErrWindowClosed
	}
	w.owner.mu.Lock()
	ownerClosed := w.owner.closed
	w.owner.mu.Unlock()
	if ownerClosed {
		return ErrAppClosed
	}
	return w.app.Update(update)
}

// Invalidate requests a rebuild of only this child window.
func (w *Window) Invalidate() error { return w.Update(func() {}) }

// SetTitle changes the native title on the UI thread. It may be called from a
// UI callback or App.Update closure and does not rebuild the root.
func (w *Window) SetTitle(title string) error {
	native, err := w.runtime()
	if err != nil {
		return err
	}
	if err := native.SetTitle(title); err != nil {
		return fmt.Errorf("dxui: set child title: %w", err)
	}
	w.mu.Lock()
	w.options.Title = title
	w.mu.Unlock()
	return nil
}

// SetSize requests a new positive logical client size. The resulting native
// viewport event drives layout; it is not presented speculatively.
func (w *Window) SetSize(width, height float32) error {
	native, runtimeErr := w.runtime()
	if runtimeErr != nil {
		return runtimeErr
	}
	if width <= 0 || height <= 0 {
		return errors.New("dxui: child window size must be positive")
	}
	wv, err := logicalDimension("width", width, 0)
	if err != nil {
		return err
	}
	hv, err := logicalDimension("height", height, 0)
	if err != nil {
		return err
	}
	if err := native.SetSize(wv, hv); err != nil {
		return fmt.Errorf("dxui: set child size: %w", err)
	}
	w.mu.Lock()
	w.options.Width = width
	w.options.Height = height
	w.mu.Unlock()
	return nil
}

// Title returns the last successfully configured title.
func (w *Window) Title() string {
	if w == nil {
		return ""
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.options.Title
}

// Size returns the latest known logical client size. Native resize events and
// successful SetSize calls update this snapshot.
func (w *Window) Size() Size {
	if w == nil {
		return Size{}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return Size{Width: w.options.Width, Height: w.options.Height}
}

// Maximize requests the platform's maximized window state.
func (w *Window) Maximize() error {
	native, err := w.runtime()
	if err != nil {
		return err
	}
	if err = native.Maximize(); err != nil {
		return fmt.Errorf("dxui: maximize window: %w", err)
	}
	return nil
}

// Unmaximize restores a maximized window to its normal state.
func (w *Window) Unmaximize() error { return w.restore("unmaximize") }

// Minimize requests the platform's minimized window state.
func (w *Window) Minimize() error {
	native, err := w.runtime()
	if err != nil {
		return err
	}
	if err = native.Minimize(); err != nil {
		return fmt.Errorf("dxui: minimize window: %w", err)
	}
	return nil
}

// Unminimize restores a minimized window to its normal state.
func (w *Window) Unminimize() error { return w.restore("unminimize") }
func (w *Window) restore(action string) error {
	native, err := w.runtime()
	if err != nil {
		return err
	}
	if err = native.Restore(); err != nil {
		return fmt.Errorf("dxui: %s window: %w", action, err)
	}
	return nil
}

// IsMaximized reports the latest state confirmed by native window events.
func (w *Window) IsMaximized() bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.maximized
}

// IsMinimized reports the latest state confirmed by native window events.
func (w *Window) IsMinimized() bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.minimized
}

func (w *Window) runtime() (backend.WindowRuntime, error) {
	if w == nil {
		return nil, ErrWindowClosed
	}
	w.mu.Lock()
	closed := w.closed
	native := w.native
	w.mu.Unlock()
	if closed {
		return nil, ErrWindowClosed
	}
	if native != nil {
		return native, nil
	}
	w.owner.mu.Lock()
	running, closing, value := w.owner.running, w.owner.closed, w.owner.backend
	w.owner.mu.Unlock()
	if closing {
		return nil, ErrAppClosed
	}
	if !running {
		return nil, ErrWindowNotRunning
	}
	result, ok := value.(backend.WindowRuntime)
	if !ok {
		return nil, ErrWindowNotRunning
	}
	return result, nil
}

// Main-window counterparts preserve App as the main lifecycle handle.
func (a *App) Title() string {
	a.mu.Lock()
	w := a.mainWindow
	fallback := a.options.Title
	a.mu.Unlock()
	if w == nil {
		return fallback
	}
	return w.Title()
}
func (a *App) Size() Size {
	a.mu.Lock()
	w := a.mainWindow
	fallback := Size{Width: a.options.Width, Height: a.options.Height}
	a.mu.Unlock()
	if w == nil {
		return fallback
	}
	return w.Size()
}
func (a *App) SetTitle(title string) error {
	w, err := a.mainWindowHandle()
	if err != nil {
		return err
	}
	return w.SetTitle(title)
}
func (a *App) SetSize(width, height float32) error {
	w, err := a.mainWindowHandle()
	if err != nil {
		return err
	}
	return w.SetSize(width, height)
}
func (a *App) Maximize() error {
	w, err := a.mainWindowHandle()
	if err != nil {
		return err
	}
	return w.Maximize()
}
func (a *App) Unmaximize() error {
	w, err := a.mainWindowHandle()
	if err != nil {
		return err
	}
	return w.Unmaximize()
}
func (a *App) Minimize() error {
	w, err := a.mainWindowHandle()
	if err != nil {
		return err
	}
	return w.Minimize()
}
func (a *App) Unminimize() error {
	w, err := a.mainWindowHandle()
	if err != nil {
		return err
	}
	return w.Unminimize()
}
func (a *App) IsMaximized() bool {
	a.mu.Lock()
	w := a.mainWindow
	a.mu.Unlock()
	return w != nil && w.IsMaximized()
}
func (a *App) IsMinimized() bool {
	a.mu.Lock()
	w := a.mainWindow
	a.mu.Unlock()
	return w != nil && w.IsMinimized()
}
func (a *App) mainWindowHandle() (*Window, error) {
	a.mu.Lock()
	w, running, closed := a.mainWindow, a.running, a.closed
	a.mu.Unlock()
	if closed {
		return nil, ErrAppClosed
	}
	if !running || w == nil {
		return nil, ErrWindowNotRunning
	}
	return w, nil
}

// Close closes this child window without affecting its owner or siblings.
// Repeated calls are no-ops.
func (w *Window) Close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.app.mu.Lock()
	w.app.closed = true
	events := w.app.events
	w.app.mu.Unlock()
	w.mu.Unlock()
	if events != nil {
		_ = events.Wake()
	}
}

// Diagnostics returns this window's backend-neutral counters.
func (w *Window) Diagnostics() RuntimeDiagnostics {
	if w == nil {
		return RuntimeDiagnostics{}
	}
	return w.app.Diagnostics()
}

// Closed reports whether close has been requested or completed.
func (w *Window) Closed() bool {
	if w == nil {
		return true
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

func (w *Window) destroy() error {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
	}
	native := w.native
	w.native = nil
	w.mu.Unlock()
	w.app.retained.Unmount()
	w.app.images.Clear()
	w.app.releaseTextEngine()
	w.app.mu.Lock()
	w.app.running = false
	w.app.events = nil
	w.app.backend = nil
	w.app.display = nil
	w.app.geometry = nil
	w.app.mu.Unlock()
	if native != nil {
		return native.Close()
	}
	return nil
}
