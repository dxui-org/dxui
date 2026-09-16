package dxui

import (
	"context"
	"errors"
	"fmt"
	"math"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/dxui-org/dxui/internal/backend"
	internalimage "github.com/dxui-org/dxui/internal/image"
	internalinput "github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer/sdl3"
	uiruntime "github.com/dxui-org/dxui/internal/runtime"
	internaltext "github.com/dxui-org/dxui/internal/text"
	"github.com/dxui-org/dxui/internal/tree"
)

var (
	// ErrAppNotRunning reports an Update attempted outside App.Run.
	ErrAppNotRunning = errors.New("dxui: app is not running")
	// ErrAppClosed reports an operation submitted after shutdown was requested.
	ErrAppClosed = errors.New("dxui: app is closing")
)

type windowOptions struct {
	Title               string
	Width, Height       float32
	MinWidth, MinHeight float32
	Background          RGBAColor
}

// RuntimeDiagnostics is a backend-neutral snapshot of the running or most
// recently stopped native runtime.
type RuntimeDiagnostics struct {
	SDLVersion             string
	RendererName           string
	LogicalSize            Size
	PixelSize              Size
	PixelDensity           float32
	DisplayScale           float32
	FrameCount             uint64
	WindowCreates          uint64
	RendererCreateAttempts uint64
	RendererCreates        uint64
	ExposeEvents           uint64
	ResizeEvents           uint64
	ScaleEvents            uint64
	NoopViewportEvents     uint64
	RendererResetEvents    uint64
	BuildCount             uint64
	LayoutCount            uint64
	PaintCount             uint64
	SoftwareFallback       bool
	CountersEnabled        bool
	EventCount             uint64
	ReconcileCount         uint64
	PaintNodeCount         uint64
	TextureCreates         uint64
	TextureDestroys        uint64
	CacheBytes             uint64
	CacheBudgetBytes       uint64
	CacheEntries           uint64
	FontResources          uint64
	ImageResources         uint64
	RendererResources      uint64
	Goroutines             int
	GoHeapBytes            uint64
	GoHeapObjects          uint64
	GoTotalAllocBytes      uint64
	GoMallocs              uint64
	EventToPresent         TimingSummary
	FrameTime              TimingSummary
}

// TimingSummary reports a bounded percentile distribution in nanoseconds.
type TimingSummary struct {
	Count, Samples      uint64
	P50NS, P95NS, P99NS int64
}

// App owns one application runtime, one main window and any child windows. An
// App is single-use: Run may be called exactly once.
type App struct {
	mu         sync.Mutex
	options    AppOptions
	root       func() View
	view       View
	retained   tree.Tree
	layout     *layout.Engine
	text       *internaltext.Engine
	images     *internalimage.Cache
	textErr    error
	geometry   *layout.Result
	display    paint.DisplayList
	theme      resolvedTheme
	themeErr   error
	themeDirty tree.Dirty

	started, running                                              bool
	closed                                                        bool
	invalidated                                                   bool
	updates                                                       []func()
	events                                                        platform.EventSource
	backend                                                       backend.Runtime
	diagnostics                                                   RuntimeDiagnostics
	asyncErr                                                      error
	building                                                      bool
	buildCount                                                    uint64
	layoutCount                                                   uint64
	paintCount                                                    uint64
	reconcileCount                                                uint64
	paintNodeCount                                                uint64
	loopMetrics                                                   *uiruntime.Metrics
	resourceCacheBytes, resourceCacheBudget, resourceCacheEntries uint64
	fontResources, imageResources                                 uint64
	interaction                                                   internalinput.Controller
	inputActions                                                  map[uint64]inputAction
	editors                                                       map[uint64]*retainedEditor
	focusedEditor                                                 uint64
	nativeTextActive                                              bool
	windowFocused                                                 bool
	caretVisible                                                  bool
	caretChangedAt                                                time.Time
	clock                                                         platform.Clock
	dragEditor                                                    uint64
	modifiers                                                     platform.Modifiers
	scrollDrag                                                    scrollbarDrag
	sliderDrag                                                    sliderDrag
	tabsCapture                                                   tabsPointerCapture
	menuCapture                                                   menuPointerCapture
	selectCapture                                                 selectPointerCapture
	overlayCapture                                                overlayPointerCapture
	passwordToggleCapture                                         uint64
	imageStatus                                                   map[uint64]uint64
	constraintAware                                               bool
	buildSize                                                     Size
	windows                                                       map[platform.WindowID]*Window
	mainWindow                                                    *Window
	multiRuntime                                                  backend.MultiWindowRuntime
}

// LayoutContext is the logical space available to the root builder. It
// contains no backend values. A constraint-aware build runs once initially
// and once for the final resize/scale event in each drained event batch.
type LayoutContext struct{ Width, Height float32 }

// NewApp creates an application configuration without loading SDL.
func NewApp(options AppOptions) *App {
	options.Fonts = append([]Font(nil), options.Fonts...)
	options.Shortcuts = append([]Shortcut(nil), options.Shortcuts...)
	resolved, themeErr := prepareTheme(options.Theme)
	app := &App{
		options:       options,
		theme:         resolved,
		themeErr:      themeErr,
		layout:        layout.NewEngine(layout.EngineOptions{}),
		images:        internalimage.NewCache(options.Caches.ImageBytes),
		imageStatus:   make(map[uint64]uint64),
		editors:       make(map[uint64]*retainedEditor),
		windowFocused: true,
		caretVisible:  true,
		clock:         platform.SystemClock{},
		windows:       make(map[platform.WindowID]*Window),
	}
	if options.Diagnostics {
		app.loopMetrics = uiruntime.NewMetrics()
	}
	return app
}

// RunResponsive is Run with a root builder that may choose a different view
// structure from the current logical window size. Measuring a result never
// invokes the builder again; only a later coalesced viewport event can do so.
func (a *App) RunResponsive(root func(LayoutContext) View) error {
	if root == nil {
		return errors.New("dxui: nil responsive root builder")
	}
	a.mu.Lock()
	a.constraintAware = true
	a.mu.Unlock()
	return a.Run(func() View {
		a.mu.Lock()
		size := a.buildSize
		a.mu.Unlock()
		if size.Width <= 0 || size.Height <= 0 {
			constraints := a.currentLayoutConstraints()
			size = Size{Width: constraints.Width.Max, Height: constraints.Height.Max}
		}
		return root(LayoutContext{Width: size.Width, Height: size.Height})
	})
}

// Run creates the native window, builds root, and owns the process main thread
// until the app closes. Call it directly from main, before moving UI work to
// other goroutines. Run is blocking and may be called only once, including
// after a startup or runtime error. A nil root is rejected before the App is
// consumed. Startup, build, renderer, callback, and event-loop failures are
// returned; OnError also observes runtime failures when configured.
func (a *App) Run(root func() View) (runErr error) {
	if root == nil {
		return errors.New("dxui: nil root builder")
	}

	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return errors.New("dxui: App.Run may be called only once")
	}
	a.started = true
	a.root = root
	window := windowOptions{
		Title: a.options.Title, Width: a.options.Width, Height: a.options.Height,
		MinWidth: a.options.MinWidth, MinHeight: a.options.MinHeight,
		Background: a.options.Background,
	}
	preference := a.options.Renderer
	a.running = true
	a.closed = false
	a.invalidated = false
	a.updates = nil
	a.asyncErr = nil
	clear(a.imageStatus)
	a.themeDirty = 0
	a.buildCount = 0
	a.layoutCount = 0
	a.paintCount = 0
	a.reconcileCount = 0
	a.paintNodeCount = 0
	a.resourceCacheBytes = 0
	a.resourceCacheBudget = 0
	a.resourceCacheEntries = 0
	a.fontResources = 0
	a.imageResources = 0
	a.diagnostics = RuntimeDiagnostics{}
	if a.options.Diagnostics {
		a.loopMetrics = uiruntime.NewMetrics()
	} else {
		a.loopMetrics = nil
	}
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.running = false
		a.events = nil
		a.backend = nil
		a.updates = nil
		a.mu.Unlock()
	}()
	if err := validateWindowOptions(window); err != nil {
		return err
	}
	if err := validateShortcuts(a.options.Shortcuts); err != nil {
		return err
	}
	if preference != RendererAuto && preference != RendererSoftware {
		return fmt.Errorf("dxui: renderer: invalid preference %d", preference)
	}
	if a.themeErr != nil {
		return fmt.Errorf("dxui: initial theme: %w", a.themeErr)
	}
	if _, err := a.textEngine(); err != nil {
		return fmt.Errorf("dxui: initialize text: %w", err)
	}
	defer func() {
		a.images.Clear()
		a.releaseTextEngine()
		a.refreshResourceDiagnostics()
	}()

	width, err := logicalDimension("width", window.Width, 800)
	if err != nil {
		return err
	}
	height, err := logicalDimension("height", window.Height, 600)
	if err != nil {
		return err
	}
	minWidth, err := optionalDimension("minimum width", window.MinWidth)
	if err != nil {
		return err
	}
	minHeight, err := optionalDimension("minimum height", window.MinHeight)
	if err != nil {
		return err
	}
	background := window.Background
	if background == (RGBAColor{}) {
		background = RGBA(28, 30, 36, 255)
	}
	if window.Title == "" {
		window.Title = "dxui"
	}

	// go-sdl3 locks the importing main goroutine during package initialization.
	// This explicit lock prevents migration during Run; SDL's own main-thread
	// query below still rejects calls made from a non-main goroutine.
	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()

	config := sdl3.Config{
		Title: window.Title,
		Width: width, Height: height, MinWidth: minWidth, MinHeight: minHeight,
		Background:       sdl3.Color{R: background.R, G: background.G, B: background.B, A: background.A},
		Software:         preference == RendererSoftware,
		ShadowCacheBytes: a.options.Caches.ShadowBytes,
		GlyphCacheBytes:  a.options.Caches.GlyphBytes,
		ImageCacheBytes:  a.options.Caches.ImageBytes,
		Diagnostics:      a.options.Diagnostics,
	}
	nativeDiagnostics, runErr := sdl3.WithBackend(config, func(native backend.Runtime) error {
		multi, ok := native.(backend.MultiWindowRuntime)
		if !ok {
			return errors.New("dxui: backend does not support window management")
		}
		a.mu.Lock()
		a.events = native.Events()
		a.backend = native
		a.multiRuntime = multi
		a.mainWindow = &Window{owner: a, app: a, id: multi.MainWindowID(), dirty: true, options: WindowOptions{Title: window.Title, Width: float32(width), Height: float32(height), MinWidth: window.MinWidth, MinHeight: window.MinHeight, Background: background}}
		a.windows[multi.MainWindowID()] = a.mainWindow
		shouldWake := len(a.updates) != 0 || a.closed || a.invalidated
		a.mu.Unlock()
		defer func() {
			a.destroyAllWindows()
			a.retained.Unmount()
			a.mu.Lock()
			a.display = nil
			a.geometry = nil
			a.inputActions = nil
			a.selectCapture = selectPointerCapture{}
			a.overlayCapture = overlayPointerCapture{}
			a.tabsCapture = tabsPointerCapture{}
			a.menuCapture = menuPointerCapture{}
			a.closed = true
			a.events = nil
			a.backend = nil
			a.multiRuntime = nil
			a.mainWindow = nil
			clear(a.windows)
			a.mu.Unlock()
		}()
		if shouldWake {
			if err := native.Events().Wake(); err != nil {
				return fmt.Errorf("dxui: event startup wake: %w", err)
			}
		}
		if _, err := a.buildAndCommit(); err != nil {
			return err
		}
		loop := uiruntime.Windows{Clock: a.clock, Events: native.Events(), IDs: a.windowIDs, Window: a.runtimeWindow, Dirty: a.windowDirty, Clean: a.cleanWindow, Handle: a.handleWindowEvent, Metrics: a.loopMetrics}
		loopErr := loop.Run(context.Background())
		if a.nativeTextActive {
			if textRuntime, ok := native.(backend.TextInputRuntime); ok {
				loopErr = errors.Join(loopErr, textRuntime.TextInput().StopTextInput())
			}
			a.nativeTextActive = false
		}
		return loopErr
	})
	a.mu.Lock()
	a.diagnostics = publicDiagnostics(nativeDiagnostics)
	a.mu.Unlock()
	if runErr != nil && a.options.OnError != nil {
		if callbackErr := callNoArg("error callback", func() { a.options.OnError(runErr) }); callbackErr != nil {
			runErr = errors.Join(runErr, callbackErr)
		}
	}
	return runErr
}

func validateShortcuts(shortcuts []Shortcut) error {
	type chord struct {
		key       ShortcutKey
		modifiers ShortcutModifiers
	}
	seen := make(map[chord]struct{}, len(shortcuts))
	for index, shortcut := range shortcuts {
		if shortcutPlatformKey(shortcut.Key) == platform.KeyOther {
			return fmt.Errorf("dxui: shortcut %d has an invalid key", index)
		}
		value := chord{shortcut.Key, shortcut.Modifiers}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("dxui: shortcut %d duplicates an earlier chord", index)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func (a *App) afterFirstPresent() (bool, error) {
	if a.options.OnShown != nil {
		if err := callNoArg("shown callback", func() { a.options.OnShown(a) }); err != nil {
			return false, err
		}
	}
	return a.isClosed(), nil
}

// SetTheme validates and copies a complete Primitive -> Semantic -> Component
// theme atomically. Equal themes are a no-op. A relevant metric-token change
// schedules layout; visual-only resolved changes schedule display/paint only.
// While Run is active, call SetTheme from a UI callback or inside Update.
func (a *App) SetTheme(source Theme) error {
	prepared, err := prepareTheme(source)
	if err != nil {
		return fmt.Errorf("dxui: set theme: %w", err)
	}
	a.mu.Lock()
	children := make([]*App, 0, len(a.windows))
	for _, window := range a.windows {
		if window != nil && window.app != a {
			children = append(children, window.app)
		}
	}
	a.mu.Unlock()
	for _, child := range children {
		child.mu.Lock()
		view := child.view
		child.mu.Unlock()
		if view.node != nil {
			if err := validateViewTheme(view, prepared); err != nil {
				return fmt.Errorf("dxui: set theme child active tree: %w", err)
			}
		}
	}
	a.mu.Lock()
	if resolvedThemesEqual(a.theme, prepared) {
		a.mu.Unlock()
		return nil
	}
	old := a.theme
	view := a.view
	instance := a.retained.Root()
	geometry := a.geometry
	previousDisplay := a.display
	running := a.running
	building := a.building
	beforePending := a.invalidated || a.themeDirty != 0
	a.mu.Unlock()

	dirty := tree.Dirty(0)
	if view.node != nil && instance != nil && geometry != nil {
		if err := validateViewTheme(view, prepared); err != nil {
			return fmt.Errorf("dxui: set theme active tree: %w", err)
		}
		textEngine, textErr := a.textEngine()
		if textErr != nil {
			return fmt.Errorf("dxui: set theme text: %w", textErr)
		}
		scaleX, scaleY := a.textScale()
		textSourceBudget := a.textSourceBudget()
		oldList, oldErr := buildDisplayList(view, instance, geometry, old, textEngine, a.images, scaleX, scaleY, textSourceBudget, previousDisplay, a.editorDisplaySnapshot())
		newList, newErr := buildDisplayList(view, instance, geometry, prepared, textEngine, a.images, scaleX, scaleY, textSourceBudget, oldList, a.editorDisplaySnapshot())
		if oldErr != nil || newErr != nil {
			return fmt.Errorf("dxui: set theme display: %w", errors.Join(oldErr, newErr))
		}
		if themeChangesLayout(view, old, prepared) {
			dirty = tree.DirtyLayout | tree.DirtyDisplay | tree.DirtyPaint
		} else if !displayListsEqual(oldList, newList) {
			dirty = tree.DirtyDisplay | tree.DirtyPaint
		}
	}
	a.mu.Lock()
	a.theme = prepared
	a.options.Theme = prepared.source
	a.themeErr = nil
	a.mu.Unlock()
	for _, child := range children {
		if err := child.SetTheme(source); err != nil {
			return err
		}
	}
	if dirty == 0 {
		return nil
	}
	a.mu.Lock()
	a.themeDirty |= dirty
	events := a.events
	a.mu.Unlock()
	if running && events != nil && !building && !beforePending {
		if err := events.Wake(); err != nil {
			return fmt.Errorf("dxui: set theme wake: %w", err)
		}
	}
	return nil
}

// SetClipboardText writes UTF-8 text to the system clipboard. While Run is
// active, call it from a UI callback or inside Update so the native operation
// remains on the UI thread. Invalid UTF-8 is normalized to replacement runes.
func (a *App) SetClipboardText(value string) error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrAppClosed
	}
	if !a.running {
		a.mu.Unlock()
		return ErrAppNotRunning
	}
	runtime := a.backend
	a.mu.Unlock()
	textRuntime, ok := runtime.(backend.TextInputRuntime)
	if !ok {
		return errors.New("dxui: clipboard is unavailable")
	}
	textInput := textRuntime.TextInput()
	if textInput == nil {
		return errors.New("dxui: clipboard is unavailable")
	}
	value = strings.ToValidUTF8(value, "\uFFFD")
	if err := textInput.SetClipboardText(value); err != nil {
		return fmt.Errorf("dxui: set clipboard text: %w", err)
	}
	return nil
}

// Close requests application shutdown and safely wakes a blocked event wait.
// It may be called from callbacks or any goroutine. Repeated calls and calls
// made before Run are no-ops.
func (a *App) Close() {
	a.mu.Lock()
	if a.closed || !a.running {
		a.mu.Unlock()
		return
	}
	a.closed = true
	events := a.events
	a.mu.Unlock()
	if events != nil {
		if err := events.Wake(); err != nil {
			a.recordAsyncError(fmt.Errorf("dxui: close wake: %w", err))
		}
	}
}

// Update queues a state update for FIFO execution on the UI thread, then
// rebuilds the root once after the batch. It never executes update on the
// caller's goroutine and is safe to call from any goroutine.
func (a *App) Update(update func()) error {
	if update == nil {
		return errors.New("dxui: nil update callback")
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrAppClosed
	}
	if !a.running {
		a.mu.Unlock()
		return ErrAppNotRunning
	}
	a.updates = append(a.updates, update)
	events := a.events
	a.mu.Unlock()
	if events != nil {
		if err := events.Wake(); err != nil {
			return fmt.Errorf("dxui: update wake: %w", err)
		}
	}
	return nil
}

// Invalidate requests a rebuild of the main window only. It is safe from any
// goroutine; child windows use Window.Invalidate.
func (a *App) Invalidate() error {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return ErrAppClosed
	}
	if !a.running {
		a.mu.Unlock()
		return ErrAppNotRunning
	}
	already := a.invalidated
	a.invalidated = true
	events := a.events
	a.mu.Unlock()
	if !already && events != nil {
		if err := events.Wake(); err != nil {
			return fmt.Errorf("dxui: invalidate wake: %w", err)
		}
	}
	return nil
}

// Diagnostics returns a race-safe, SDL-free runtime snapshot.
func (a *App) Diagnostics() RuntimeDiagnostics {
	a.mu.Lock()
	native := a.backend
	last := a.diagnostics
	last.BuildCount = a.buildCount
	last.LayoutCount = a.layoutCount
	last.PaintCount = a.paintCount
	last.ReconcileCount = a.reconcileCount
	last.PaintNodeCount = a.paintNodeCount
	resourceBytes, resourceBudget, resourceEntries := a.resourceCacheBytes, a.resourceCacheBudget, a.resourceCacheEntries
	fontResources, imageResources := a.fontResources, a.imageResources
	metrics := a.loopMetrics
	enabled := a.options.Diagnostics
	a.mu.Unlock()
	if native != nil {
		current := publicDiagnostics(native.Diagnostics())
		current.BuildCount = last.BuildCount
		current.LayoutCount = last.LayoutCount
		current.PaintCount = last.PaintCount
		current.ReconcileCount = last.ReconcileCount
		current.PaintNodeCount = last.PaintNodeCount
		current.CacheBytes += resourceBytes
		current.CacheBudgetBytes += resourceBudget
		current.CacheEntries += resourceEntries
		current.FontResources = fontResources
		current.ImageResources = imageResources
		current.RendererResources = current.TextureCreates - current.TextureDestroys
		last = current
	} else {
		last.CacheBytes += resourceBytes
		last.CacheBudgetBytes += resourceBudget
		last.CacheEntries += resourceEntries
		last.FontResources = fontResources
		last.ImageResources = imageResources
		last.RendererResources = last.TextureCreates - last.TextureDestroys
	}
	last.CountersEnabled = enabled
	if enabled {
		last.EventCount, last.EventToPresent, last.FrameTime = publicTiming(metrics)
		var memory goruntime.MemStats
		goruntime.ReadMemStats(&memory)
		last.Goroutines = goruntime.NumGoroutine()
		last.GoHeapBytes = memory.HeapAlloc
		last.GoHeapObjects = memory.HeapObjects
		last.GoTotalAllocBytes = memory.TotalAlloc
		last.GoMallocs = memory.Mallocs
	}
	return last
}

func publicTiming(metrics *uiruntime.Metrics) (uint64, TimingSummary, TimingSummary) {
	events, eventLatency, frameTime := metrics.Snapshot()
	convert := func(value uiruntime.TimingSummary) TimingSummary {
		return TimingSummary{Count: value.Count, Samples: value.Samples, P50NS: int64(value.P50), P95NS: int64(value.P95), P99NS: int64(value.P99)}
	}
	return events, convert(eventLatency), convert(frameTime)
}

func (a *App) windowIDs() []platform.WindowID {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]platform.WindowID, 0, len(a.windows))
	for id := range a.windows {
		ids = append(ids, id)
	}
	return ids
}

func (a *App) runtimeWindow(id platform.WindowID) (uiruntime.Window, bool) {
	a.mu.Lock()
	handle, ok := a.windows[id]
	native := a.backend
	a.mu.Unlock()
	if !ok {
		return uiruntime.Window{}, false
	}
	r := native.Renderer()
	if handle.native != nil {
		r = handle.native.Renderer()
	}
	return uiruntime.Window{Renderer: r, Display: handle.app.currentDisplay, Deadline: handle.app.nextDeadline, Shown: func() error {
		handle.mu.Lock()
		if handle.shown {
			handle.mu.Unlock()
			return nil
		}
		handle.shown = true
		callback := handle.options.OnShown
		handle.mu.Unlock()
		if handle == a.mainWindow {
			_, err := a.afterFirstPresent()
			return err
		}
		if callback != nil {
			return callNoArg("child shown callback", func() { callback(handle) })
		}
		return nil
	}}, true
}

func (a *App) windowDirty(id platform.WindowID) bool {
	a.mu.Lock()
	w := a.windows[id]
	a.mu.Unlock()
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.dirty
}
func (a *App) cleanWindow(id platform.WindowID) {
	a.mu.Lock()
	w := a.windows[id]
	a.mu.Unlock()
	if w != nil {
		w.mu.Lock()
		w.dirty = false
		w.mu.Unlock()
	}
}
func (a *App) markWindowDirty(w *Window) {
	if w != nil {
		w.mu.Lock()
		w.dirty = true
		w.mu.Unlock()
	}
}

func (a *App) handleWindowEvent(event platform.Event) (bool, error) {
	if event.Kind == platform.EventDeadline && event.Window == 0 {
		for _, id := range a.windowIDs() {
			a.mu.Lock()
			w := a.windows[id]
			a.mu.Unlock()
			if w == nil {
				continue
			}
			dirty, _, err := w.app.handleEvent(event)
			if err != nil {
				return false, err
			}
			if dirty {
				a.markWindowDirty(w)
			}
		}
		return a.isClosed(), nil
	}
	if event.Kind == platform.EventWake {
		for _, id := range a.windowIDs() {
			a.mu.Lock()
			w := a.windows[id]
			a.mu.Unlock()
			if w == nil {
				continue
			}
			dirty, err := w.app.runQueuedWork()
			if err != nil {
				if reported := a.reportUpdateError(err); reported != nil {
					return false, reported
				}
			}
			if dirty {
				a.markWindowDirty(w)
			}
		}
		a.removeClosedWindows()
		return a.isClosed(), nil
	}
	a.mu.Lock()
	w := a.windows[event.Window]
	a.mu.Unlock()
	if w == nil {
		return a.isClosed(), nil
	}
	w.mu.Lock()
	switch event.Kind {
	case platform.EventResize, platform.EventScale:
		if event.Viewport.LogicalWidth > 0 && event.Viewport.LogicalHeight > 0 {
			w.options.Width = float32(event.Viewport.LogicalWidth)
			w.options.Height = float32(event.Viewport.LogicalHeight)
		}
	case platform.EventWindowMaximized:
		w.maximized = true
		w.minimized = false
	case platform.EventWindowMinimized:
		w.minimized = true
		w.maximized = false
	case platform.EventWindowRestored:
		w.minimized = false
		w.maximized = false
	}
	w.mu.Unlock()
	if w != a.mainWindow && event.Kind == platform.EventClose {
		w.mu.Lock()
		callback := w.options.OnCloseRequest
		w.mu.Unlock()
		if callback == nil {
			w.Close()
		} else if err := callNoArg("child close callback", func() { callback(w) }); err != nil {
			return false, err
		}
		if !w.Closed() {
			changes, err := w.app.buildAndCommit()
			if err != nil {
				return false, err
			}
			if changes.Has(tree.DirtyPaint) {
				a.markWindowDirty(w)
			}
		}
		a.removeClosedWindows()
		return a.isClosed(), nil
	}
	dirty, closeLoop, err := w.app.handleEvent(event)
	if dirty {
		a.markWindowDirty(w)
	}
	if w != a.mainWindow && closeLoop {
		w.Close()
		closeLoop = false
	}
	a.removeClosedWindows()
	return closeLoop || a.isClosed(), err
}

func (a *App) removeClosedWindows() {
	for _, id := range a.windowIDs() {
		a.mu.Lock()
		w := a.windows[id]
		main := w == a.mainWindow
		a.mu.Unlock()
		if w == nil || main || !w.Closed() {
			continue
		}
		a.mu.Lock()
		delete(a.windows, id)
		a.mu.Unlock()
		if err := w.destroy(); err != nil {
			a.recordAsyncError(err)
		}
	}
}

func (a *App) destroyAllWindows() {
	for _, id := range a.windowIDs() {
		a.mu.Lock()
		w := a.windows[id]
		if w != a.mainWindow {
			delete(a.windows, id)
		}
		a.mu.Unlock()
		if w != nil && w != a.mainWindow {
			_ = w.destroy()
		}
	}
}

func (a *App) refreshResourceDiagnostics() {
	if !a.options.Diagnostics {
		return
	}
	var textStats internaltext.CacheStats
	a.mu.Lock()
	textEngine := a.text
	a.mu.Unlock()
	if textEngine != nil {
		textStats = textEngine.Stats()
	}
	imageStats := a.images.Stats()
	layoutStats := a.layoutEngine().Stats()
	a.mu.Lock()
	a.resourceCacheBytes = uint64(textStats.Bytes + imageStats.Bytes + layoutStats.CacheBytes)
	// Layout exposes current bytes but not its configured budget; the App uses
	// its strict 256 KiB default and includes it in the aggregate budget.
	a.resourceCacheBudget = uint64(textStats.BudgetBytes+imageStats.BudgetBytes) + 256<<10
	a.resourceCacheEntries = uint64(textStats.Entries + imageStats.Entries + layoutStats.CacheEntries)
	a.fontResources = uint64(textStats.Faces)
	a.imageResources = uint64(imageStats.ActiveEntries)
	a.mu.Unlock()
}

func (a *App) invalidate() {
	a.mu.Lock()
	alreadyPending := a.invalidated
	a.invalidated = true
	events := a.events
	running := a.running
	building := a.building
	a.mu.Unlock()
	if running && events != nil && !building && !alreadyPending {
		if err := events.Wake(); err != nil {
			a.recordAsyncError(fmt.Errorf("dxui: invalidate wake: %w", err))
		}
	}
}

func (a *App) handleEvent(event platform.Event) (bool, bool, error) {
	if err := a.takeAsyncError(); err != nil {
		return false, false, err
	}
	switch event.Kind {
	case platform.EventDeadline:
		dirty, err := a.handleDeadlines()
		return dirty, a.isClosed(), err
	case platform.EventResize, platform.EventScale:
		a.mu.Lock()
		constraintAware := a.constraintAware
		if event.Viewport.LogicalWidth > 0 && event.Viewport.LogicalHeight > 0 {
			a.buildSize = Size{Width: float32(event.Viewport.LogicalWidth), Height: float32(event.Viewport.LogicalHeight)}
		}
		a.mu.Unlock()
		if constraintAware {
			changes, err := a.buildAndCommit()
			if err != nil {
				return false, false, err
			}
			if changes.Has(tree.DirtyLayout) {
				return true, a.isClosed(), nil
			}
		}
		if err := a.relayout(event.Viewport.LogicalWidth, event.Viewport.LogicalHeight); err != nil {
			return false, false, err
		}
		return true, a.isClosed(), nil
	case platform.EventClose:
		if a.options.OnCloseRequest == nil {
			a.Close()
		} else if err := callNoArg("close callback", func() { a.options.OnCloseRequest(a) }); err != nil {
			return false, false, err
		} else if !a.isClosed() {
			changes, err := a.buildAndCommit()
			if err != nil {
				if reportErr := a.reportUpdateError(err); reportErr != nil {
					return false, false, reportErr
				}
				return false, a.isClosed(), nil
			}
			return changes.Has(tree.DirtyPaint), false, nil
		}
	case platform.EventWake:
		dirty, err := a.runQueuedWork()
		if err != nil {
			if reportErr := a.reportUpdateError(err); reportErr != nil {
				return false, false, reportErr
			}
			return false, a.isClosed(), nil
		}
		return dirty, a.isClosed(), nil
	case platform.EventMouseMove, platform.EventMouseDown, platform.EventMouseUp,
		platform.EventMouseWheel, platform.EventKeyDown, platform.EventKeyUp,
		platform.EventWindowFocusGained, platform.EventWindowFocusLost,
		platform.EventWindowMouseLeave, platform.EventTextEditing, platform.EventTextInput:
		dirty, err := a.handleInteraction(event)
		if err != nil {
			if reportErr := a.reportUpdateError(err); reportErr != nil {
				return dirty, false, reportErr
			}
		}
		return dirty, a.isClosed(), nil
	}
	return false, a.isClosed(), nil
}

func (a *App) reportUpdateError(err error) error {
	if a.options.OnError == nil {
		return err
	}
	if callbackErr := callNoArg("error callback", func() { a.options.OnError(err) }); callbackErr != nil {
		return errors.Join(err, callbackErr)
	}
	return nil
}

func (a *App) runQueuedWork() (bool, error) {
	a.mu.Lock()
	updates := append([]func(){}, a.updates...)
	a.updates = nil
	rootBuild := a.invalidated || len(updates) != 0
	a.invalidated = false
	a.building = true
	themeDirty := a.themeDirty
	a.themeDirty = 0
	a.mu.Unlock()
	defer a.endUpdateBatch()
	for _, update := range updates {
		if err := callNoArg("update callback", update); err != nil {
			return false, err
		}
	}
	a.mu.Lock()
	rootBuild = rootBuild || a.invalidated
	a.invalidated = false
	themeDirty |= a.themeDirty
	a.themeDirty = 0
	a.mu.Unlock()
	if rootBuild && !a.isClosed() {
		changes, err := a.buildAndCommit()
		if err != nil {
			return false, err
		}
		if themeDirty&tree.DirtyLayout != 0 && !changes.Has(tree.DirtyLayout) {
			if err := a.relayoutCurrent(); err != nil {
				return false, err
			}
			return true, nil
		}
		if themeDirty&tree.DirtyPaint != 0 && !changes.Has(tree.DirtyPaint) {
			if err := a.redisplayCurrent(); err != nil {
				return false, err
			}
			return true, nil
		}
		return changes.Has(tree.DirtyPaint), nil
	}
	if themeDirty != 0 && !a.isClosed() {
		if themeDirty&tree.DirtyLayout != 0 {
			if err := a.relayoutCurrent(); err != nil {
				return false, err
			}
			return true, nil
		}
		if themeDirty&tree.DirtyPaint != 0 {
			if err := a.redisplayCurrent(); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (a *App) endUpdateBatch() {
	a.mu.Lock()
	a.building = false
	pending := (a.invalidated || a.themeDirty != 0) && a.running && !a.closed
	events := a.events
	a.mu.Unlock()
	if pending && events != nil {
		if err := events.Wake(); err != nil {
			a.recordAsyncError(fmt.Errorf("dxui: deferred invalidate wake: %w", err))
		}
	}
}

func (a *App) buildRoot() (err error) {
	_, err = a.buildAndCommit()
	return err
}

func (a *App) buildAndCommit() (tree.Changes, error) {
	a.mu.Lock()
	root := a.root
	themeErr := a.themeErr
	a.buildCount++
	a.mu.Unlock()
	if themeErr != nil {
		return tree.Changes{}, fmt.Errorf("dxui: initial theme: %w", themeErr)
	}
	view, err := callBuilder("root", root)
	if err != nil {
		return tree.Changes{}, err
	}
	return a.commitView(view)
}

func (a *App) commitView(view View) (tree.Changes, error) {
	if a.options.Diagnostics {
		a.mu.Lock()
		a.reconcileCount++
		a.mu.Unlock()
	}
	a.mu.Lock()
	theme := a.theme
	a.mu.Unlock()
	textEngine, err := a.textEngine()
	if err != nil {
		return tree.Changes{}, fmt.Errorf("dxui: text: %w", err)
	}
	a.mu.Lock()
	previousView := a.view
	previousRoot := a.retained.Root()
	a.mu.Unlock()
	view, err = materializeVirtualLists(view, previousView, previousRoot, a.currentLayoutConstraints().Height.Max)
	if err != nil {
		return tree.Changes{}, err
	}
	if err := validateViewTheme(view, theme); err != nil {
		return tree.Changes{}, fmt.Errorf("dxui: theme root: %w", err)
	}
	description, err := describeView(view)
	if err != nil {
		return tree.Changes{}, err
	}
	preview, candidate, err := a.retained.Preview(description)
	if err != nil {
		return tree.Changes{}, fmt.Errorf("dxui: reconcile root: %w", err)
	}
	previousImageKeys := a.images.ActiveKeys()
	preparedImageKeys := preview.Has(tree.DirtyPaint)
	var nextImageKeys []internalimage.Key
	imageKeysCommitted := false
	if preparedImageKeys {
		nextImageKeys = activeImageKeys(view)
		preparingImageKeys := append(append(make([]internalimage.Key, 0, len(previousImageKeys)+len(nextImageKeys)), previousImageKeys...), nextImageKeys...)
		a.images.SetActive(preparingImageKeys)
		defer func() {
			if !imageKeysCommitted {
				a.images.SetActive(previousImageKeys)
			}
		}()
	}
	var geometry *layout.Result
	a.mu.Lock()
	currentGeometry := a.geometry
	previousDisplay := a.display
	a.mu.Unlock()
	if preview.Has(tree.DirtyLayout) {
		geometry, _, err = layoutViewWithEngine(a.layoutEngine(), view, candidate, a.currentLayoutConstraints(), theme.metric, makeIntrinsicResolver(textEngine, a.images, theme))
		if err != nil {
			return tree.Changes{}, fmt.Errorf("dxui: layout root: %w", err)
		}
	}
	if geometry == nil {
		geometry = currentGeometry
	}
	scrollStateChanged := syncScrollOffsets(view, candidate, geometry)
	selectStateChanged := syncSelectStates(view, candidate, geometry, theme)
	tabsStateChanged := syncTabsStates(view, candidate)
	menuStateChanged := syncMenuStates(view, candidate)
	if scrollStateChanged || selectStateChanged || tabsStateChanged || menuStateChanged {
		preview.Dirty |= tree.DirtyDisplay | tree.DirtyPaint | tree.DirtySemantics
	}
	var display paint.DisplayList
	if preview.Has(tree.DirtyPaint) || scrollStateChanged || selectStateChanged || tabsStateChanged || menuStateChanged {
		scaleX, scaleY := a.textScale()
		display, err = buildDisplayList(view, candidate, geometry, theme, textEngine, a.images, scaleX, scaleY, a.textSourceBudget(), previousDisplay, a.editorDisplaySnapshot())
		if err != nil {
			return tree.Changes{}, fmt.Errorf("dxui: display root: %w", err)
		}
	}
	// Prepare hit/focus data against the disposable candidate. A failure here
	// must not commit a partially prepared virtual-list window or any other tree.
	snapshot, actions, snapshotErr := buildInteractionSnapshot(view, candidate, geometry, theme, textEngine)
	if snapshotErr != nil {
		return tree.Changes{}, fmt.Errorf("dxui: input root: %w", snapshotErr)
	}
	changes, err := a.retained.Update(description)
	if err != nil {
		return tree.Changes{}, fmt.Errorf("dxui: reconcile root: %w", err)
	}
	if scrollStateChanged {
		changes.Dirty |= tree.DirtyDisplay | tree.DirtyPaint | tree.DirtySemantics
	}
	if preparedImageKeys {
		a.images.SetActive(nextImageKeys)
	}
	imageKeysCommitted = true
	syncScrollOffsets(view, a.retained.Root(), geometry)
	syncSelectStates(view, a.retained.Root(), geometry, theme)
	syncTabsStates(view, a.retained.Root())
	syncMenuStates(view, a.retained.Root())
	a.restoreClosedPopoverFocus(actions)
	interactionResult := a.interaction.Reconcile(snapshot)
	a.reconcileScrollDrag(actions)
	a.reconcileSliderDrag(actions)
	a.reconcileSelectCapture(actions)
	a.reconcileTabsCapture(actions)
	a.reconcileMenuCapture(actions)
	a.reconcilePasswordToggleState(actions)
	tooltipLifecycleChanged := a.syncTooltipLifecycle(actions)
	editorsChanged := a.syncEditors(actions)
	interactionChanged := applyInteractionState(a.retained.Root(), &a.interaction) || interactionResult.Changed || tooltipLifecycleChanged || editorsChanged
	if interactionChanged {
		scaleX, scaleY := a.textScale()
		display, err = buildDisplayList(view, a.retained.Root(), geometry, theme, textEngine, a.images, scaleX, scaleY, a.textSourceBudget(), previousDisplay, a.editorDisplaySnapshot())
		if err != nil {
			return tree.Changes{}, fmt.Errorf("dxui: interaction display root: %w", err)
		}
		changes.Dirty |= tree.DirtyDisplay | tree.DirtyPaint
	}
	a.mu.Lock()
	a.view = view
	if preview.Has(tree.DirtyLayout) {
		a.geometry = geometry
	}
	if preview.Has(tree.DirtyPaint) {
		a.display = display
	}
	if interactionChanged {
		a.display = display
	}
	a.inputActions = actions
	if changes.Has(tree.DirtyLayout) {
		a.layoutCount++
	}
	if changes.Has(tree.DirtyPaint) {
		a.paintCount++
		if a.options.Diagnostics {
			a.paintNodeCount += countPaintNodes(display)
		}
	}
	a.mu.Unlock()
	a.refreshResourceDiagnostics()
	if a.nativeTextActive {
		focusedAction, textFocused := actions[a.interaction.Focused()]
		if !textFocused || !focusedAction.editor || focusedAction.disabled {
			if native := a.textInputBackend(); native != nil {
				if err := native.StopTextInput(); err != nil {
					return changes, fmt.Errorf("dxui: stop removed text input: %w", err)
				}
			}
			a.nativeTextActive, a.focusedEditor = false, 0
		} else {
			a.focusedEditor = a.interaction.Focused()
		}
	}
	a.retained.ClearDirty()
	if err := a.notifyImageCallbacks(view, a.retained.Root()); err != nil {
		return changes, err
	}
	return changes, nil
}

func activeImageKeys(view View) []internalimage.Key {
	keys := make([]internalimage.Key, 0)
	var visit func(View)
	visit = func(current View) {
		if current.node == nil {
			return
		}
		if imageProps, ok := imagePropsForView(current); ok && imageProps.Source.source != nil {
			keys = append(keys, internalimage.SourceKey(imageProps.Source.source.id, imageProps.MaxPixels))
		}
		for _, child := range current.node.children {
			visit(child)
		}
	}
	visit(view)
	return keys
}

func (a *App) notifyImageCallbacks(view View, instance *tree.Node) error {
	next := make(map[uint64]uint64)
	type notification struct {
		load func(Size)
		fail func(error)
		size Size
		err  error
	}
	notifications := make([]notification, 0)
	var visit func(View, *tree.Node)
	visit = func(current View, retained *tree.Node) {
		if current.node == nil || retained == nil {
			return
		}
		if imageProps, ok := imagePropsForView(current); ok && imageProps.Source.source != nil {
			status := paintStateHash(struct {
				Resource uint64
				HasLoad  bool
				HasError bool
			}{retained.Properties.Resource.Identity, imageProps.OnLoad != nil, imageProps.OnError != nil})
			next[retained.ID] = status
			a.mu.Lock()
			previous, seen := a.imageStatus[retained.ID]
			a.mu.Unlock()
			if !seen || previous != status {
				bitmap, err := decodeImage(a.images, imageProps)
				n := notification{load: imageProps.OnLoad, fail: imageProps.OnError, err: err}
				if bitmap != nil {
					n.size = Size{Width: float32(bitmap.Width), Height: float32(bitmap.Height)}
				}
				notifications = append(notifications, n)
			}
		}
		for index, child := range current.node.children {
			if index < len(retained.Children) {
				visit(child, retained.Children[index])
			}
		}
	}
	visit(view, instance)
	a.mu.Lock()
	a.imageStatus = next
	a.mu.Unlock()
	for _, notification := range notifications {
		invoked := false
		if notification.err != nil && notification.fail != nil {
			invoked = true
			if err := callNoArg("image error callback", func() { notification.fail(notification.err) }); err != nil {
				return err
			}
		} else if notification.err == nil && notification.load != nil {
			invoked = true
			if err := callNoArg("image load callback", func() { notification.load(notification.size) }); err != nil {
				return err
			}
		}
		if invoked {
			a.invalidate()
		}
	}
	return nil
}

func (a *App) relayout(width, height int32) error {
	a.mu.Lock()
	view := a.view
	instance := a.retained.Root()
	previousDisplay := a.display
	a.mu.Unlock()
	if view.node == nil || instance == nil {
		return nil
	}
	constraints := a.currentLayoutConstraints()
	if width >= 0 && height >= 0 {
		constraints = exactLayoutConstraints(float32(width), float32(height))
	}
	a.mu.Lock()
	theme := a.theme
	a.mu.Unlock()
	textEngine, err := a.textEngine()
	if err != nil {
		return fmt.Errorf("dxui: resize text: %w", err)
	}
	geometry, _, err := layoutViewWithEngine(a.layoutEngine(), view, instance, constraints, theme.metric, makeIntrinsicResolver(textEngine, a.images, theme))
	if err != nil {
		return fmt.Errorf("dxui: resize layout: %w", err)
	}
	syncScrollOffsets(view, instance, geometry)
	if a.hasSelectAction() {
		syncSelectStates(view, instance, geometry, theme)
	}
	if _, err := a.refreshInteraction(view, instance, geometry, theme); err != nil {
		return fmt.Errorf("dxui: resize input: %w", err)
	}
	scaleX, scaleY := a.textScale()
	display, err := buildDisplayList(view, instance, geometry, theme, textEngine, a.images, scaleX, scaleY, a.textSourceBudget(), previousDisplay, a.editorDisplaySnapshot())
	if err != nil {
		return fmt.Errorf("dxui: resize display: %w", err)
	}
	a.mu.Lock()
	a.geometry = geometry
	a.display = display
	a.layoutCount++
	a.paintCount++
	if a.options.Diagnostics {
		a.paintNodeCount += countPaintNodes(display)
	}
	a.mu.Unlock()
	a.refreshResourceDiagnostics()
	return a.updateTextInputArea()
}

func (a *App) relayoutCurrent() error {
	return a.relayout(-1, -1)
}

func (a *App) redisplayCurrent() error {
	a.mu.Lock()
	view, instance, geometry, theme, previousDisplay := a.view, a.retained.Root(), a.geometry, a.theme, a.display
	a.mu.Unlock()
	if view.node == nil || instance == nil || geometry == nil {
		return nil
	}
	if a.hasSelectAction() {
		syncSelectStates(view, instance, geometry, theme)
	}
	textEngine, err := a.textEngine()
	if err != nil {
		return fmt.Errorf("dxui: theme text: %w", err)
	}
	if _, err := a.refreshInteraction(view, instance, geometry, theme); err != nil {
		return fmt.Errorf("dxui: display input: %w", err)
	}
	scaleX, scaleY := a.textScale()
	display, err := buildDisplayList(view, instance, geometry, theme, textEngine, a.images, scaleX, scaleY, a.textSourceBudget(), previousDisplay, a.editorDisplaySnapshot())
	if err != nil {
		return fmt.Errorf("dxui: theme display: %w", err)
	}
	a.mu.Lock()
	a.display = display
	a.paintCount++
	if a.options.Diagnostics {
		a.paintNodeCount += countPaintNodes(display)
	}
	a.mu.Unlock()
	a.refreshResourceDiagnostics()
	return nil
}

func countPaintNodes(display paint.DisplayList) uint64 {
	var count uint64
	for _, command := range display {
		switch command.Kind {
		case paint.CommandDrawShadow, paint.CommandFillRoundedRect,
			paint.CommandStrokeRoundedRect, paint.CommandFillStrokeRoundedRect, paint.CommandDrawText,
			paint.CommandDrawIcon, paint.CommandDrawImage, paint.CommandBorder:
			count++
		}
	}
	return count
}

func (a *App) hasSelectAction() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, action := range a.inputActions {
		if action.selectp != nil {
			return true
		}
	}
	return false
}

func (a *App) currentDisplay() paint.DisplayList {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.display
}

func (a *App) layoutEngine() *layout.Engine {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.layout == nil {
		a.layout = layout.NewEngine(layout.EngineOptions{})
	}
	return a.layout
}

func (a *App) currentLayoutConstraints() layout.Constraints {
	a.mu.Lock()
	native := a.backend
	width, height := a.options.Width, a.options.Height
	a.mu.Unlock()
	if native != nil {
		viewport := native.Diagnostics().Viewport
		if viewport.LogicalWidth >= 0 && viewport.LogicalHeight >= 0 {
			return exactLayoutConstraints(float32(viewport.LogicalWidth), float32(viewport.LogicalHeight))
		}
	}
	if !finite(width) || width <= 0 {
		width = 800
	}
	if !finite(height) || height <= 0 {
		height = 600
	}
	return exactLayoutConstraints(width, height)
}

func exactLayoutConstraints(width, height float32) layout.Constraints {
	return layout.Constraints{
		Width:  layout.Limit{Min: width, Max: width, MaxSet: true, Definite: true},
		Height: layout.Limit{Min: height, Max: height, MaxSet: true, Definite: true},
	}
}

func (a *App) isClosed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closed
}

func (a *App) recordAsyncError(err error) {
	a.mu.Lock()
	a.asyncErr = errors.Join(a.asyncErr, err)
	a.mu.Unlock()
}

func (a *App) takeAsyncError() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	err := a.asyncErr
	a.asyncErr = nil
	return err
}

func callNoArg(stage string, callback func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("dxui: %s panic: %v", stage, recovered)
		}
	}()
	callback()
	return nil
}

func validateWindowOptions(options windowOptions) error {
	values := []struct {
		name  string
		value float32
	}{
		{"width", options.Width}, {"height", options.Height},
		{"minimum width", options.MinWidth}, {"minimum height", options.MinHeight},
	}
	for _, item := range values {
		if math.IsNaN(float64(item.value)) || math.IsInf(float64(item.value), 0) || item.value < 0 || item.value > math.MaxInt32 {
			return fmt.Errorf("dxui: invalid window %s %v", item.name, item.value)
		}
	}
	return nil
}

func logicalDimension(name string, value float32, defaultValue int32) (int32, error) {
	if value == 0 {
		return defaultValue, nil
	}
	result, err := optionalDimension(name, value)
	if err != nil {
		return 0, err
	}
	if result <= 0 {
		return 0, fmt.Errorf("dxui: window %s must be positive", name)
	}
	return result, nil
}

func optionalDimension(name string, value float32) (int32, error) {
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || value < 0 || value > math.MaxInt32 {
		return 0, fmt.Errorf("dxui: invalid window %s %v", name, value)
	}
	return int32(math.Round(float64(value))), nil
}

func publicDiagnostics(value backend.Diagnostics) RuntimeDiagnostics {
	return RuntimeDiagnostics{
		SDLVersion:   value.SDLVersion,
		RendererName: value.RendererName,
		LogicalSize:  Size{Width: float32(value.Viewport.LogicalWidth), Height: float32(value.Viewport.LogicalHeight)},
		PixelSize:    Size{Width: float32(value.Viewport.PixelWidth), Height: float32(value.Viewport.PixelHeight)},
		PixelDensity: value.Viewport.PixelDensity, DisplayScale: value.Viewport.DisplayScale,
		FrameCount: value.Frames, SoftwareFallback: value.SoftwareFallback,
		WindowCreates: value.WindowCreates, RendererCreateAttempts: value.RendererCreateAttempts,
		RendererCreates: value.RendererCreates, ExposeEvents: value.ExposeEvents,
		ResizeEvents: value.ResizeEvents, ScaleEvents: value.ScaleEvents,
		NoopViewportEvents: value.NoopViewportEvents, RendererResetEvents: value.RendererResetEvents,
		TextureCreates: value.TextureCreates, TextureDestroys: value.TextureDestroys,
		CacheBytes: value.CacheBytes, CacheBudgetBytes: value.CacheBudgetBytes, CacheEntries: value.CacheEntries,
	}
}
