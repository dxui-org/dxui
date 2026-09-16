package sdl3

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/dxui-org/dxui/internal/backend"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
)

// Color is an SDL-independent clear color supplied by the root package.
type Color struct{ R, G, B, A uint8 }

// Config contains validated startup inputs for the single-window runtime.
type Config struct {
	Title               string
	Width, Height       int32
	MinWidth, MinHeight int32
	Background          Color
	Software            bool
	ShadowCacheBytes    int
	GlyphCacheBytes     int
	ImageCacheBytes     int
	Diagnostics         bool
}

type nativeEventKind uint8

const (
	nativeEventOther nativeEventKind = iota
	nativeEventWake
	nativeEventExpose
	nativeEventClose
	nativeEventResize
	nativeEventScale
	nativeEventRendererReset
	nativeEventMouseMove
	nativeEventMouseDown
	nativeEventMouseUp
	nativeEventMouseWheel
	nativeEventKeyDown
	nativeEventKeyUp
	nativeEventWindowFocusGained
	nativeEventWindowFocusLost
	nativeEventWindowMouseLeave
	nativeEventWindowMinimized
	nativeEventWindowMaximized
	nativeEventWindowRestored
	nativeEventTextEditing
	nativeEventTextInput
)

type nativeEvent struct {
	windowID                    uint32
	kind                        nativeEventKind
	x, y                        float32
	wheelX, wheelY              float32
	button                      platform.MouseButton
	clicks                      uint8
	key                         platform.Key
	modifiers                   platform.Modifiers
	repeat                      bool
	text                        string
	editingStart, editingLength int
}

type nativeAPI interface {
	load() (nativeLibrary, error)
	version() string
	init() error
	quit()
	isMainThread() bool
	createWindow(string, int32, int32) (nativeWindow, error)
	waitEvent() (nativeEvent, error)
	waitEventTimeout(int32) (nativeEvent, bool)
	pollEvent() (nativeEvent, bool)
	pushWake() error
}

type nativeLibrary interface {
	unload() error
}

type nativeWindow interface {
	id() (uint32, error)
	setTitle(string) error
	maximize() error
	minimize() error
	restore() error
	createRenderer(string) (nativeRenderer, error)
	setSize(int32, int32) error
	setMinimumSize(int32, int32) error
	viewport() (platform.Viewport, error)
	show() error
	destroy()
}

type nativeRenderer interface {
	name() (string, error)
	windowToRender(float32, float32) (float32, float32, error)
	setScale(float32, float32) error
	setDrawColor(uint8, uint8, uint8, uint8) error
	setDrawBlendMode() error
	setClipRect(*nativeRect) error
	fillRect(nativeFRect) error
	renderGeometry(nativeTexture, []nativeVertex, []int32) error
	createTextTexture(int, int) (nativeTexture, error)
	renderTexture(nativeTexture, nativeFRect) error
	clear() error
	present() error
	destroy()
}

type nativeTexture interface {
	update([]byte, int32) error
	setColorMod(uint8, uint8, uint8) error
	setAlphaMod(uint8) error
	destroy()
}

// Driver owns SDL, its one window, and its renderer in strict reverse order.
type Driver struct {
	api      nativeAPI
	library  nativeLibrary
	window   nativeWindow
	renderer nativeRenderer

	mu                  sync.RWMutex
	diagnostics         backend.Diagnostics
	closed              bool
	shadowCache         *renderer.ByteCache[shadowKey, shadowGeometry]
	textCache           *renderer.ByteCache[textTextureKey, nativeTexture]
	imageCache          *renderer.ByteCache[textTextureKey, nativeTexture]
	activeText          map[textTextureKey]struct{}
	activeImage         map[textTextureKey]struct{}
	nextText            map[textTextureKey]struct{}
	nextImage           map[textTextureKey]struct{}
	scaleX              float32
	scaleY              float32
	windowScaleX        float32
	windowScaleY        float32
	background          Color
	performanceCounters bool
	softwareRenderer    bool
	shown               bool
	logicalID           platform.WindowID
	nativeID            uint32
	owner               *Driver
	windows             map[uint32]*Driver
	retiredWindowIDs    map[uint32]struct{}
	nextWindowID        platform.WindowID
	ownsSDL             bool
}

var runtimeOwner sync.Mutex

// WithBackend owns one complete native lifecycle around callback. It ensures
// callback errors and all partial startup failures release native resources.
func WithBackend(config Config, callback func(backend.Runtime) error) (backend.Diagnostics, error) {
	if callback == nil {
		return backend.Diagnostics{}, errors.New("sdl3: nil runtime callback")
	}
	if !runtimeOwner.TryLock() {
		return backend.Diagnostics{}, errors.New("sdl3: init: another native runtime is active")
	}
	defer runtimeOwner.Unlock()

	return withAPI(systemAPI{}, config, callback)
}

func withAPI(api nativeAPI, config Config, callback func(backend.Runtime) error) (diagnostics backend.Diagnostics, runErr error) {
	driver, err := openWithAPI(api, config)
	if err != nil {
		return backend.Diagnostics{}, err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			runErr = errors.Join(runErr, fmt.Errorf("sdl3: runtime callback panic: %v", recovered))
		}
		runErr = errors.Join(runErr, driver.Close())
		diagnostics = driver.Diagnostics()
	}()
	runErr = callback(driver)
	return diagnostics, runErr
}

func openWithAPI(api nativeAPI, config Config) (_ *Driver, err error) {
	if config.Width <= 0 || config.Height <= 0 {
		return nil, fmt.Errorf("sdl3: window: invalid size %dx%d", config.Width, config.Height)
	}

	shadowBudget := config.ShadowCacheBytes
	if shadowBudget == 0 {
		shadowBudget = 2 << 20
	}
	glyphBudget := config.GlyphCacheBytes
	if glyphBudget == 0 {
		glyphBudget = 2 << 20
	}
	imageBudget := config.ImageCacheBytes
	if imageBudget == 0 {
		imageBudget = 2 << 20
	}
	driver := &Driver{api: api, shadowCache: renderer.NewByteCache[shadowKey, shadowGeometry](shadowBudget, nil), activeText: make(map[textTextureKey]struct{}), activeImage: make(map[textTextureKey]struct{}), nextText: make(map[textTextureKey]struct{}), nextImage: make(map[textTextureKey]struct{}), background: config.Background, performanceCounters: config.Diagnostics, ownsSDL: true, logicalID: 1, nextWindowID: 2, windows: make(map[uint32]*Driver), retiredWindowIDs: make(map[uint32]struct{})}
	driver.textCache = renderer.NewByteCache[textTextureKey, nativeTexture](glyphBudget, driver.destroyTexture)
	driver.imageCache = renderer.NewByteCache[textTextureKey, nativeTexture](imageBudget, driver.destroyTexture)
	driver.library, err = api.load()
	if err != nil {
		return nil, fmt.Errorf("sdl3: load embedded library: %w", err)
	}
	initAttempted := false
	defer func() {
		if err == nil {
			return
		}
		if driver.renderer != nil {
			driver.renderer.destroy()
		}
		if driver.window != nil {
			driver.window.destroy()
		}
		if initAttempted {
			api.quit()
		}
		if driver.library != nil {
			if unloadErr := driver.library.unload(); unloadErr != nil {
				err = errors.Join(err, fmt.Errorf("sdl3: unload after startup failure: %w", unloadErr))
			}
		}
	}()

	version := api.version()
	driver.diagnostics.SDLVersion = version
	if version != RequiredSDLVersion {
		return nil, fmt.Errorf("sdl3: load: native version %s, require %s", version, RequiredSDLVersion)
	}
	initAttempted = true
	if err := api.init(); err != nil {
		return nil, fmt.Errorf("sdl3: init video/events: %w", err)
	}
	if !api.isMainThread() {
		return nil, errors.New("sdl3: init: App.Run must execute on the process main thread")
	}

	driver.window, err = api.createWindow(config.Title, config.Width, config.Height)
	if err != nil {
		return nil, fmt.Errorf("sdl3: window create: %w", err)
	}
	if driver.performanceCounters {
		driver.diagnostics.WindowCreates++
	}
	driver.nativeID, err = driver.window.id()
	if err != nil {
		return nil, fmt.Errorf("sdl3: window id: %w", err)
	}
	driver.windows[driver.nativeID] = driver
	initialViewport, viewportErr := driver.window.viewport()
	if viewportErr != nil {
		return nil, fmt.Errorf("sdl3: initial window viewport: %w", viewportErr)
	}
	contentScale, scaleErr := viewportContentScale(initialViewport)
	if scaleErr != nil {
		return nil, scaleErr
	}
	windowWidth, windowHeight, sizeErr := scaledWindowSize(config.Width, config.Height, contentScale)
	if sizeErr != nil {
		return nil, sizeErr
	}
	if windowWidth != initialViewport.LogicalWidth || windowHeight != initialViewport.LogicalHeight {
		if err := driver.window.setSize(windowWidth, windowHeight); err != nil {
			return nil, fmt.Errorf("sdl3: window logical size: %w", err)
		}
	}
	if config.MinWidth > 0 || config.MinHeight > 0 {
		minimumWidth, minimumHeight, minimumErr := scaledWindowSize(config.MinWidth, config.MinHeight, contentScale)
		if minimumErr != nil {
			return nil, minimumErr
		}
		if err := driver.window.setMinimumSize(minimumWidth, minimumHeight); err != nil {
			return nil, fmt.Errorf("sdl3: window minimum size: %w", err)
		}
	}

	createRenderer := func(name string) (nativeRenderer, error) {
		if driver.performanceCounters {
			driver.diagnostics.RendererCreateAttempts++
		}
		value, createErr := driver.window.createRenderer(name)
		if createErr == nil && driver.performanceCounters {
			driver.diagnostics.RendererCreates++
		}
		return value, createErr
	}
	if config.Software {
		driver.renderer, err = createRenderer("software")
		if err != nil {
			return nil, fmt.Errorf("sdl3: renderer create software: %w", err)
		}
	} else {
		var defaultErr error
		driver.renderer, defaultErr = createRenderer("")
		if defaultErr != nil {
			driver.renderer, err = createRenderer("software")
			if err != nil {
				return nil, fmt.Errorf("sdl3: renderer create: default: %v; software: %w", defaultErr, err)
			}
			driver.diagnostics.SoftwareFallback = true
		}
	}
	driver.diagnostics.RendererName, err = driver.renderer.name()
	if err != nil {
		return nil, fmt.Errorf("sdl3: renderer name: %w", err)
	}
	driver.softwareRenderer = driver.diagnostics.RendererName == "software"
	if _, err := driver.refreshViewport(); err != nil {
		return nil, err
	}
	if err := driver.renderer.setDrawColor(config.Background.R, config.Background.G, config.Background.B, config.Background.A); err != nil {
		return nil, fmt.Errorf("sdl3: render set background: %w", err)
	}
	if err := driver.renderer.setDrawBlendMode(); err != nil {
		return nil, fmt.Errorf("sdl3: render alpha blend mode: %w", err)
	}
	return driver, nil
}

func openWindowWithAPI(api nativeAPI, config Config) (_ *Driver, err error) {
	if config.Width <= 0 || config.Height <= 0 {
		return nil, fmt.Errorf("sdl3: window: invalid size %dx%d", config.Width, config.Height)
	}
	shadowBudget, glyphBudget, imageBudget := config.ShadowCacheBytes, config.GlyphCacheBytes, config.ImageCacheBytes
	if shadowBudget == 0 {
		shadowBudget = 2 << 20
	}
	if glyphBudget == 0 {
		glyphBudget = 2 << 20
	}
	if imageBudget == 0 {
		imageBudget = 2 << 20
	}
	d := &Driver{api: api, shadowCache: renderer.NewByteCache[shadowKey, shadowGeometry](shadowBudget, nil), activeText: make(map[textTextureKey]struct{}), activeImage: make(map[textTextureKey]struct{}), nextText: make(map[textTextureKey]struct{}), nextImage: make(map[textTextureKey]struct{}), background: config.Background, performanceCounters: config.Diagnostics}
	d.textCache = renderer.NewByteCache[textTextureKey, nativeTexture](glyphBudget, d.destroyTexture)
	d.imageCache = renderer.NewByteCache[textTextureKey, nativeTexture](imageBudget, d.destroyTexture)
	defer func() {
		if err != nil {
			if d.renderer != nil {
				d.renderer.destroy()
			}
			if d.window != nil {
				d.window.destroy()
			}
		}
	}()
	d.window, err = api.createWindow(config.Title, config.Width, config.Height)
	if err != nil {
		return nil, fmt.Errorf("sdl3: window create: %w", err)
	}
	d.nativeID, err = d.window.id()
	if err != nil {
		return nil, fmt.Errorf("sdl3: window id: %w", err)
	}
	initial, err := d.window.viewport()
	if err != nil {
		return nil, fmt.Errorf("sdl3: initial window viewport: %w", err)
	}
	contentScale, err := viewportContentScale(initial)
	if err != nil {
		return nil, err
	}
	w, h, err := scaledWindowSize(config.Width, config.Height, contentScale)
	if err != nil {
		return nil, err
	}
	if w != initial.LogicalWidth || h != initial.LogicalHeight {
		if err = d.window.setSize(w, h); err != nil {
			return nil, fmt.Errorf("sdl3: window logical size: %w", err)
		}
	}
	if config.MinWidth > 0 || config.MinHeight > 0 {
		mw, mh, e := scaledWindowSize(config.MinWidth, config.MinHeight, contentScale)
		if e != nil {
			return nil, e
		}
		if err = d.window.setMinimumSize(mw, mh); err != nil {
			return nil, fmt.Errorf("sdl3: window minimum size: %w", err)
		}
	}
	name := ""
	if config.Software {
		name = "software"
	}
	d.renderer, err = d.window.createRenderer(name)
	if err != nil && name == "" {
		d.renderer, err = d.window.createRenderer("software")
		d.diagnostics.SoftwareFallback = err == nil
	}
	if err != nil {
		return nil, fmt.Errorf("sdl3: renderer create: %w", err)
	}
	d.diagnostics.RendererName, err = d.renderer.name()
	if err != nil {
		return nil, fmt.Errorf("sdl3: renderer name: %w", err)
	}
	d.softwareRenderer = d.diagnostics.RendererName == "software"
	if _, err = d.refreshViewport(); err != nil {
		return nil, err
	}
	if err = d.renderer.setDrawColor(config.Background.R, config.Background.G, config.Background.B, config.Background.A); err != nil {
		return nil, err
	}
	if err = d.renderer.setDrawBlendMode(); err != nil {
		return nil, err
	}
	return d, nil
}

func (driver *Driver) Events() platform.EventSource      { return driver }
func (driver *Driver) Renderer() renderer.Renderer       { return driver }
func (driver *Driver) TextInput() backend.TextInput      { return driver }
func (driver *Driver) MainWindowID() platform.WindowID   { return driver.logicalID }
func (driver *Driver) SetTitle(title string) error       { return driver.window.setTitle(title) }
func (driver *Driver) SetSize(width, height int32) error { return driver.window.setSize(width, height) }
func (driver *Driver) Maximize() error                   { return driver.window.maximize() }
func (driver *Driver) Minimize() error                   { return driver.window.minimize() }
func (driver *Driver) Restore() error                    { return driver.window.restore() }

func (driver *Driver) CreateWindow(config backend.WindowConfig) (platform.WindowID, backend.WindowRuntime, error) {
	root := driver
	if root.owner != nil {
		root = root.owner
	}
	childConfig := Config{Title: config.Title, Width: config.Width, Height: config.Height, MinWidth: config.MinWidth, MinHeight: config.MinHeight,
		Background: Color{R: config.Background[0], G: config.Background[1], B: config.Background[2], A: config.Background[3]}, Software: config.Software,
		ShadowCacheBytes: config.ShadowCacheBytes, GlyphCacheBytes: config.GlyphCacheBytes, ImageCacheBytes: config.ImageCacheBytes, Diagnostics: config.Diagnostics}
	child, err := openWindowWithAPI(root.api, childConfig)
	if err != nil {
		return 0, nil, err
	}
	child.owner = root
	if _, reused := root.retiredWindowIDs[child.nativeID]; reused {
		_ = child.Close()
		return 0, nil, fmt.Errorf("sdl3: native window id %d was reused during one runtime", child.nativeID)
	}
	child.logicalID = root.nextWindowID
	root.nextWindowID++
	root.windows[child.nativeID] = child
	return child.logicalID, child, nil
}

type nativeTextWindow interface {
	startTextInput() error
	stopTextInput() error
	clearComposition() error
	setTextInputArea(int32, int32, int32, int32, int32) error
}

type nativeClipboardAPI interface {
	clipboardText() (string, error)
	setClipboardText(string) error
}

func (driver *Driver) StartTextInput() error {
	window, ok := driver.window.(nativeTextWindow)
	if !ok {
		return errors.New("sdl3: native text input is unavailable")
	}
	return window.startTextInput()
}

func (driver *Driver) StopTextInput() error {
	window, ok := driver.window.(nativeTextWindow)
	if !ok {
		return errors.New("sdl3: native text input is unavailable")
	}
	return window.stopTextInput()
}

func (driver *Driver) ClearComposition() error {
	window, ok := driver.window.(nativeTextWindow)
	if !ok {
		return errors.New("sdl3: native composition is unavailable")
	}
	return window.clearComposition()
}

func (driver *Driver) SetTextInputArea(area backend.TextInputArea) error {
	window, ok := driver.window.(nativeTextWindow)
	if !ok {
		return errors.New("sdl3: native text input area is unavailable")
	}
	scaled, err := scaleTextInputArea(area, driver.windowScaleX, driver.windowScaleY)
	if err != nil {
		return err
	}
	return window.setTextInputArea(scaled.X, scaled.Y, scaled.Width, scaled.Height, scaled.Cursor)
}

func (driver *Driver) ClipboardText() (string, error) {
	api, ok := driver.api.(nativeClipboardAPI)
	if !ok {
		return "", errors.New("sdl3: clipboard is unavailable")
	}
	return api.clipboardText()
}

func (driver *Driver) SetClipboardText(value string) error {
	api, ok := driver.api.(nativeClipboardAPI)
	if !ok {
		return errors.New("sdl3: clipboard is unavailable")
	}
	return api.setClipboardText(value)
}

func (driver *Driver) Diagnostics() backend.Diagnostics {
	driver.mu.RLock()
	defer driver.mu.RUnlock()
	return driver.diagnostics
}

func (driver *Driver) createTexture(width, height int) (nativeTexture, error) {
	texture, err := driver.renderer.createTextTexture(width, height)
	if err == nil && driver.performanceCounters {
		driver.mu.Lock()
		driver.diagnostics.TextureCreates++
		driver.mu.Unlock()
	}
	return texture, err
}

func (driver *Driver) destroyTexture(texture nativeTexture) {
	if texture == nil {
		return
	}
	texture.destroy()
	if driver.performanceCounters {
		driver.mu.Lock()
		driver.diagnostics.TextureDestroys++
		driver.mu.Unlock()
	}
}

func (driver *Driver) refreshCacheDiagnostics() {
	if !driver.performanceCounters {
		return
	}
	shadow, text, image := driver.shadowCache.Stats(), driver.textCache.Stats(), driver.imageCache.Stats()
	driver.mu.Lock()
	driver.diagnostics.CacheBytes = uint64(shadow.Bytes + text.Bytes + image.Bytes)
	driver.diagnostics.CacheBudgetBytes = uint64(shadow.BudgetBytes + text.BudgetBytes + image.BudgetBytes)
	driver.diagnostics.CacheEntries = uint64(shadow.Entries + text.Entries + image.Entries)
	driver.mu.Unlock()
}

// Close is idempotent. Resource teardown is renderer, window, SDL, library.
func (driver *Driver) Close() error {
	driver.mu.Lock()
	if driver.closed {
		driver.mu.Unlock()
		return nil
	}
	driver.closed = true
	driver.mu.Unlock()
	if driver.owner != nil {
		delete(driver.owner.windows, driver.nativeID)
		driver.owner.retiredWindowIDs[driver.nativeID] = struct{}{}
	}

	driver.shadowCache.Clear()
	driver.textCache.Clear()
	if driver.imageCache != nil {
		driver.imageCache.Clear()
	}
	clear(driver.activeText)
	clear(driver.activeImage)
	clear(driver.nextText)
	clear(driver.nextImage)
	driver.refreshCacheDiagnostics()
	driver.renderer.destroy()
	driver.window.destroy()
	if driver.owner != nil {
		return nil
	}
	for nativeID, child := range driver.windows {
		if child == driver {
			continue
		}
		delete(driver.windows, nativeID)
		_ = child.Close()
	}
	driver.api.quit()
	if err := driver.library.unload(); err != nil {
		return fmt.Errorf("sdl3: unload: %w", err)
	}
	return nil
}

func (driver *Driver) Render(list paint.DisplayList) error {
	driver.syncTextureReferences(list)
	if err := driver.renderer.setDrawColor(driver.background.R, driver.background.G, driver.background.B, driver.background.A); err != nil {
		return fmt.Errorf("sdl3: render set clear color: %w", err)
	}
	if err := driver.renderer.clear(); err != nil {
		return fmt.Errorf("sdl3: render clear: %w", err)
	}
	if err := paint.Replay(list, driver); err != nil {
		return fmt.Errorf("sdl3: render display list: %w", err)
	}
	driver.refreshCacheDiagnostics()
	return nil
}

// syncTextureReferences pins textures used by the current retained display
// list. Capacity eviction therefore considers only zero-reference textures.
func (driver *Driver) syncTextureReferences(list paint.DisplayList) {
	if driver.nextText == nil {
		driver.nextText = make(map[textTextureKey]struct{})
	}
	if driver.nextImage == nil {
		driver.nextImage = make(map[textTextureKey]struct{})
	}
	clear(driver.nextText)
	clear(driver.nextImage)
	for _, command := range list {
		if (command.Kind == paint.CommandDrawText || command.Kind == paint.CommandDrawIcon) && command.Text != nil {
			key := textTextureKey{Identity: command.Text.Key, Width: command.Text.Width, Height: command.Text.Height}
			driver.nextText[key] = struct{}{}
		}
		if command.Kind == paint.CommandDrawImage && command.Image != nil {
			key := textTextureKey{Identity: command.Image.Key, Width: command.Image.Width, Height: command.Image.Height}
			driver.nextImage[key] = struct{}{}
		}
	}
	for key := range driver.activeText {
		if _, stillActive := driver.nextText[key]; !stillActive {
			driver.textCache.Release(key)
		}
	}
	for key := range driver.nextText {
		if _, alreadyActive := driver.activeText[key]; !alreadyActive {
			driver.textCache.Retain(key)
		}
	}
	for key := range driver.activeImage {
		if _, stillActive := driver.nextImage[key]; !stillActive {
			if driver.imageCache != nil {
				driver.imageCache.Release(key)
			}
		}
	}
	for key := range driver.nextImage {
		if _, alreadyActive := driver.activeImage[key]; !alreadyActive {
			if driver.imageCache != nil {
				driver.imageCache.Retain(key)
			}
		}
	}
	driver.activeText, driver.nextText = driver.nextText, driver.activeText
	driver.activeImage, driver.nextImage = driver.nextImage, driver.activeImage
}

func (driver *Driver) Present() error {
	if err := driver.renderer.present(); err != nil {
		return fmt.Errorf("sdl3: render present: %w", err)
	}
	if !driver.shown {
		if err := driver.window.show(); err != nil {
			return fmt.Errorf("sdl3: window show after first frame: %w", err)
		}
		driver.shown = true
	}
	driver.mu.Lock()
	driver.diagnostics.Frames++
	driver.mu.Unlock()
	return nil
}

func (driver *Driver) Wait(ctx context.Context, deadline *time.Time) (platform.Event, error) {
	select {
	case <-ctx.Done():
		return platform.Event{}, ctx.Err()
	default:
	}

	var event nativeEvent
	if deadline == nil {
		var err error
		event, err = driver.api.waitEvent()
		if err != nil {
			return platform.Event{}, fmt.Errorf("sdl3: event wait: %w", err)
		}
	} else {
		remaining := time.Until(*deadline)
		if remaining <= 0 {
			return platform.Event{Kind: platform.EventDeadline}, nil
		}
		milliseconds := remaining / time.Millisecond
		if remaining%time.Millisecond != 0 {
			milliseconds++
		}
		if milliseconds > time.Duration(math.MaxInt32) {
			milliseconds = time.Duration(math.MaxInt32)
		}
		var ok bool
		event, ok = driver.api.waitEventTimeout(int32(milliseconds))
		if !ok {
			return platform.Event{Kind: platform.EventDeadline}, nil
		}
	}
	return driver.translate(event)
}

func (driver *Driver) Poll() (platform.Event, bool, error) {
	event, ok := driver.api.pollEvent()
	if !ok {
		return platform.Event{}, false, nil
	}
	translated, err := driver.translate(event)
	return translated, true, err
}

func (driver *Driver) Wake() error {
	if err := driver.api.pushWake(); err != nil {
		return fmt.Errorf("sdl3: event wake: %w", err)
	}
	return nil
}

func (driver *Driver) translate(event nativeEvent) (platform.Event, error) {
	target := driver
	if driver.owner != nil {
		target = driver.owner
	}
	if event.windowID != 0 {
		value, ok := target.windows[event.windowID]
		if !ok {
			return platform.Event{Kind: platform.EventWake}, nil
		}
		target = value
	}
	windowID := target.logicalID
	switch event.kind {
	case nativeEventWake:
		return platform.Event{Window: windowID, Kind: platform.EventWake}, nil
	case nativeEventExpose:
		driver.recordNativeEvent(nativeEventExpose)
		return platform.Event{Window: windowID, Kind: platform.EventExpose}, nil
	case nativeEventClose:
		return platform.Event{Window: windowID, Kind: platform.EventClose}, nil
	case nativeEventResize, nativeEventScale:
		driver.recordNativeEvent(event.kind)
		changed, err := target.refreshViewport()
		if err != nil {
			return platform.Event{}, err
		}
		if !changed {
			driver.recordViewportNoop()
			return platform.Event{Kind: platform.EventWake}, nil
		}
		kind := platform.EventResize
		if event.kind == nativeEventScale {
			kind = platform.EventScale
		}
		return platform.Event{Window: windowID, Kind: kind, Viewport: target.Diagnostics().Viewport}, nil
	case nativeEventRendererReset:
		driver.recordNativeEvent(nativeEventRendererReset)
		if target.textCache != nil {
			target.textCache.Clear()
		}
		if target.imageCache != nil {
			target.imageCache.Clear()
		}
		target.refreshCacheDiagnostics()
		return platform.Event{Window: windowID, Kind: platform.EventExpose}, nil
	case nativeEventMouseMove, nativeEventMouseDown, nativeEventMouseUp, nativeEventMouseWheel:
		x, y, err := target.renderer.windowToRender(event.x, event.y)
		if err != nil {
			return platform.Event{}, fmt.Errorf("sdl3: pointer coordinate conversion: %w", err)
		}
		kind := platform.EventMouseMove
		switch event.kind {
		case nativeEventMouseDown:
			kind = platform.EventMouseDown
		case nativeEventMouseUp:
			kind = platform.EventMouseUp
		case nativeEventMouseWheel:
			kind = platform.EventMouseWheel
		}
		return platform.Event{Window: windowID, Kind: kind, Pointer: platform.PointerEvent{
			X: x, Y: y, Button: event.button, Clicks: event.clicks,
			WheelX: event.wheelX, WheelY: event.wheelY,
		}}, nil
	case nativeEventKeyDown, nativeEventKeyUp:
		kind := platform.EventKeyDown
		if event.kind == nativeEventKeyUp {
			kind = platform.EventKeyUp
		}
		return platform.Event{Window: windowID, Kind: kind, Key: platform.KeyEvent{Key: event.key, Modifiers: event.modifiers, Repeat: event.repeat}}, nil
	case nativeEventWindowFocusGained:
		return platform.Event{Window: windowID, Kind: platform.EventWindowFocusGained}, nil
	case nativeEventWindowFocusLost:
		return platform.Event{Window: windowID, Kind: platform.EventWindowFocusLost}, nil
	case nativeEventWindowMouseLeave:
		return platform.Event{Window: windowID, Kind: platform.EventWindowMouseLeave}, nil
	case nativeEventWindowMinimized:
		return platform.Event{Window: windowID, Kind: platform.EventWindowMinimized}, nil
	case nativeEventWindowMaximized:
		return platform.Event{Window: windowID, Kind: platform.EventWindowMaximized}, nil
	case nativeEventWindowRestored:
		return platform.Event{Window: windowID, Kind: platform.EventWindowRestored}, nil
	case nativeEventTextEditing:
		return platform.Event{Window: windowID, Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: event.text, EditingStart: event.editingStart, EditingLength: event.editingLength}}, nil
	case nativeEventTextInput:
		return platform.Event{Window: windowID, Kind: platform.EventTextInput, Text: platform.TextEvent{Text: event.text}}, nil
	default:
		return platform.Event{Kind: platform.EventWake}, nil
	}
}

func (driver *Driver) recordNativeEvent(kind nativeEventKind) {
	if !driver.performanceCounters {
		return
	}
	driver.mu.Lock()
	switch kind {
	case nativeEventExpose:
		driver.diagnostics.ExposeEvents++
	case nativeEventResize:
		driver.diagnostics.ResizeEvents++
	case nativeEventScale:
		driver.diagnostics.ScaleEvents++
	case nativeEventRendererReset:
		driver.diagnostics.RendererResetEvents++
	}
	driver.mu.Unlock()
}

func (driver *Driver) recordViewportNoop() {
	if !driver.performanceCounters {
		return
	}
	driver.mu.Lock()
	driver.diagnostics.NoopViewportEvents++
	driver.mu.Unlock()
}

func (driver *Driver) refreshViewport() (bool, error) {
	viewport, err := driver.window.viewport()
	if err != nil {
		return false, fmt.Errorf("sdl3: window viewport diagnostics: %w", err)
	}
	if viewport.LogicalWidth <= 0 || viewport.LogicalHeight <= 0 || viewport.PixelWidth <= 0 || viewport.PixelHeight <= 0 {
		return false, fmt.Errorf("sdl3: window viewport diagnostics: invalid logical %dx%d pixel %dx%d",
			viewport.LogicalWidth, viewport.LogicalHeight, viewport.PixelWidth, viewport.PixelHeight)
	}
	rawWidth, rawHeight := viewport.LogicalWidth, viewport.LogicalHeight
	contentScale, err := viewportContentScale(viewport)
	if err != nil {
		return false, err
	}
	logicalWidth := int64(math.Round(float64(rawWidth) / float64(contentScale)))
	logicalHeight := int64(math.Round(float64(rawHeight) / float64(contentScale)))
	if logicalWidth <= 0 || logicalWidth > math.MaxInt32 || logicalHeight <= 0 || logicalHeight > math.MaxInt32 {
		return false, fmt.Errorf("sdl3: window viewport diagnostics: content scale %.3f produced invalid logical size %dx%d", contentScale, logicalWidth, logicalHeight)
	}
	viewport.LogicalWidth, viewport.LogicalHeight = int32(logicalWidth), int32(logicalHeight)
	if driver.softwareRenderer {
		// SDL's software renderer presents to the window surface, whose drawable
		// coordinates already match the DPI-normalized logical viewport.
		viewport.PixelWidth, viewport.PixelHeight = viewport.LogicalWidth, viewport.LogicalHeight
	}
	scaleX := float32(viewport.PixelWidth) / float32(viewport.LogicalWidth)
	scaleY := float32(viewport.PixelHeight) / float32(viewport.LogicalHeight)
	windowScaleX := float32(rawWidth) / float32(viewport.LogicalWidth)
	windowScaleY := float32(rawHeight) / float32(viewport.LogicalHeight)
	driver.mu.RLock()
	previous := driver.diagnostics.Viewport
	driver.mu.RUnlock()
	if driver.scaleX != 0 && previous == viewport && driver.scaleX == scaleX && driver.scaleY == scaleY &&
		driver.windowScaleX == windowScaleX && driver.windowScaleY == windowScaleY {
		return false, nil
	}
	scaleChanged := driver.scaleX == 0 || driver.scaleX != scaleX || driver.scaleY != scaleY
	if driver.textCache != nil && driver.scaleX != 0 && scaleChanged {
		driver.textCache.Clear()
		if driver.imageCache != nil {
			driver.imageCache.Clear()
		}
		driver.refreshCacheDiagnostics()
	}
	if scaleChanged {
		if err := driver.renderer.setScale(scaleX, scaleY); err != nil {
			return false, fmt.Errorf("sdl3: render coordinate scale %.3fx%.3f: %w", scaleX, scaleY, err)
		}
	}
	driver.scaleX, driver.scaleY = scaleX, scaleY
	driver.windowScaleX, driver.windowScaleY = windowScaleX, windowScaleY
	driver.mu.Lock()
	driver.diagnostics.Viewport = viewport
	driver.mu.Unlock()
	return true, nil
}

func finitePositive(value float32) bool {
	return value > 0 && !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

func viewportContentScale(viewport platform.Viewport) (float32, error) {
	if !finitePositive(viewport.PixelDensity) || !finitePositive(viewport.DisplayScale) {
		return 0, fmt.Errorf("sdl3: window viewport diagnostics: invalid pixel density %.3f or display scale %.3f", viewport.PixelDensity, viewport.DisplayScale)
	}
	return viewport.DisplayScale / viewport.PixelDensity, nil
}

func scaledWindowSize(width, height int32, scale float32) (int32, int32, error) {
	scaled := func(value int32) (int32, error) {
		if value == 0 {
			return 0, nil
		}
		result := math.Round(float64(value) * float64(scale))
		if result <= 0 || result > math.MaxInt32 {
			return 0, fmt.Errorf("sdl3: logical window dimension %d at content scale %.3f is invalid", value, scale)
		}
		return int32(result), nil
	}
	scaledWidth, err := scaled(width)
	if err != nil {
		return 0, 0, err
	}
	scaledHeight, err := scaled(height)
	if err != nil {
		return 0, 0, err
	}
	return scaledWidth, scaledHeight, nil
}

func scaleTextInputArea(area backend.TextInputArea, scaleX, scaleY float32) (backend.TextInputArea, error) {
	if !finitePositive(scaleX) || !finitePositive(scaleY) {
		return backend.TextInputArea{}, fmt.Errorf("sdl3: text input area has invalid window scale %.3fx%.3f", scaleX, scaleY)
	}
	scale := func(value int32, factor float32) (int32, error) {
		result := math.Round(float64(value) * float64(factor))
		if result < math.MinInt32 || result > math.MaxInt32 {
			return 0, errors.New("sdl3: text input area exceeds native coordinate range")
		}
		return int32(result), nil
	}
	var result backend.TextInputArea
	var err error
	if result.X, err = scale(area.X, scaleX); err != nil {
		return backend.TextInputArea{}, err
	}
	if result.Y, err = scale(area.Y, scaleY); err != nil {
		return backend.TextInputArea{}, err
	}
	if result.Width, err = scale(area.Width, scaleX); err != nil {
		return backend.TextInputArea{}, err
	}
	if result.Height, err = scale(area.Height, scaleY); err != nil {
		return backend.TextInputArea{}, err
	}
	if result.Cursor, err = scale(area.Cursor, scaleX); err != nil {
		return backend.TextInputArea{}, err
	}
	return result, nil
}

var _ backend.Runtime = (*Driver)(nil)
var _ platform.EventSource = (*Driver)(nil)
var _ renderer.Renderer = (*Driver)(nil)
