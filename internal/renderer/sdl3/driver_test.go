package sdl3

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/dxui-org/dxui/internal/backend"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/go-sdl3/sdl"
)

func TestKeyboardKeyTranslatesShortcutKeys(t *testing.T) {
	tests := []struct {
		name string
		in   sdl.Keycode
		want platform.Key
	}{
		{"top-row digit", sdl.K_7, platform.Key7},
		{"keypad digit", sdl.K_KP_7, platform.Key7},
		{"enter", sdl.K_RETURN, platform.KeyEnter},
		{"keypad enter", sdl.K_KP_ENTER, platform.KeyEnter},
		{"plus", sdl.K_PLUS, platform.KeyPlus},
		{"keypad plus", sdl.K_KP_PLUS, platform.KeyPlus},
		{"minus", sdl.K_MINUS, platform.KeyMinus},
		{"multiply", sdl.K_KP_MULTIPLY, platform.KeyMultiply},
		{"divide", sdl.K_KP_DIVIDE, platform.KeyDivide},
		{"decimal", sdl.K_KP_PERIOD, platform.KeyDecimal},
		{"equals", sdl.K_EQUALS, platform.KeyEquals},
		{"backspace", sdl.K_BACKSPACE, platform.KeyBackspace},
		{"unbound", sdl.K_F1, platform.KeyOther},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := keyboardKey(test.in); got != test.want {
				t.Fatalf("keyboardKey(%v) = %v, want %v", test.in, got, test.want)
			}
		})
	}
}

type fakeAPI struct {
	actions      []string
	window       *fakeWindow
	loadErr      error
	initErr      error
	versionValue string
	mainThread   bool
	clipboard    string
}

type fakeLibrary struct{ api *fakeAPI }

func (library fakeLibrary) unload() error {
	library.api.record("unload")
	return nil
}

func newFakeAPI() *fakeAPI {
	api := &fakeAPI{versionValue: RequiredSDLVersion, mainThread: true}
	api.window = &fakeWindow{api: api, viewportValue: platform.Viewport{
		LogicalWidth: 640, LogicalHeight: 400, PixelWidth: 1280, PixelHeight: 800,
		PixelDensity: 2, DisplayScale: 2,
	}}
	return api
}

func (api *fakeAPI) record(action string) { api.actions = append(api.actions, action) }
func (api *fakeAPI) load() (nativeLibrary, error) {
	api.record("load")
	if api.loadErr != nil {
		return nil, api.loadErr
	}
	return fakeLibrary{api: api}, nil
}
func (api *fakeAPI) version() string    { api.record("version"); return api.versionValue }
func (api *fakeAPI) init() error        { api.record("init"); return api.initErr }
func (api *fakeAPI) quit()              { api.record("quit") }
func (api *fakeAPI) isMainThread() bool { api.record("main-thread"); return api.mainThread }
func (api *fakeAPI) createWindow(string, int32, int32) (nativeWindow, error) {
	api.record("create-window")
	return api.window, nil
}
func (*fakeAPI) waitEvent() (nativeEvent, error)            { return nativeEvent{}, nil }
func (*fakeAPI) waitEventTimeout(int32) (nativeEvent, bool) { return nativeEvent{}, false }
func (*fakeAPI) pollEvent() (nativeEvent, bool)             { return nativeEvent{}, false }
func (api *fakeAPI) pushWake() error                        { api.record("wake"); return nil }
func (api *fakeAPI) clipboardText() (string, error) {
	api.record("clipboard-get")
	return api.clipboard, nil
}
func (api *fakeAPI) setClipboardText(value string) error {
	api.record("clipboard-set")
	api.clipboard = value
	return nil
}

type fakeWindow struct {
	api           *fakeAPI
	rendererErrs  map[string]error
	viewportValue platform.Viewport
	textArea      backend.TextInputArea
	showErr       error
}

func (*fakeWindow) id() (uint32, error)   { return 1, nil }
func (*fakeWindow) setTitle(string) error { return nil }
func (*fakeWindow) maximize() error       { return nil }
func (*fakeWindow) minimize() error       { return nil }
func (*fakeWindow) restore() error        { return nil }

func (window *fakeWindow) createRenderer(name string) (nativeRenderer, error) {
	window.api.record("create-renderer:" + name)
	if err := window.rendererErrs[name]; err != nil {
		return nil, err
	}
	return &fakeNativeRenderer{api: window.api, nameValue: name}, nil
}
func (window *fakeWindow) setMinimumSize(int32, int32) error {
	window.api.record("minimum-size")
	return nil
}
func (window *fakeWindow) setSize(width, height int32) error {
	window.api.record("window-size")
	window.viewportValue.LogicalWidth, window.viewportValue.LogicalHeight = width, height
	window.viewportValue.PixelWidth = int32(float32(width) * window.viewportValue.PixelDensity)
	window.viewportValue.PixelHeight = int32(float32(height) * window.viewportValue.PixelDensity)
	return nil
}
func (window *fakeWindow) viewport() (platform.Viewport, error) {
	window.api.record("viewport")
	return window.viewportValue, nil
}
func (window *fakeWindow) show() error           { window.api.record("show-window"); return window.showErr }
func (window *fakeWindow) destroy()              { window.api.record("destroy-window") }
func (window *fakeWindow) startTextInput() error { window.api.record("text-start"); return nil }
func (window *fakeWindow) stopTextInput() error  { window.api.record("text-stop"); return nil }
func (window *fakeWindow) clearComposition() error {
	window.api.record("composition-clear")
	return nil
}
func (window *fakeWindow) setTextInputArea(x, y, width, height, cursor int32) error {
	window.api.record("text-area")
	window.textArea = backend.TextInputArea{X: x, Y: y, Width: width, Height: height, Cursor: cursor}
	return nil
}

type fakeNativeRenderer struct {
	api              *fakeAPI
	nameValue        string
	textures         int
	coordDiv         float32
	scaleX           float32
	scaleY           float32
	geometryTexture  bool
	geometryVertices []nativeVertex
	lastTexture      *fakeNativeTexture
}

func (renderer *fakeNativeRenderer) name() (string, error) {
	if renderer.nameValue == "" {
		return "gpu", nil
	}
	return renderer.nameValue, nil
}

func (renderer *fakeNativeRenderer) windowToRender(x, y float32) (float32, float32, error) {
	if renderer.coordDiv != 0 {
		return x / renderer.coordDiv, y / renderer.coordDiv, nil
	}
	return x, y, nil
}
func (renderer *fakeNativeRenderer) setScale(x, y float32) error {
	renderer.api.record("set-scale")
	renderer.scaleX, renderer.scaleY = x, y
	return nil
}
func (renderer *fakeNativeRenderer) setDrawColor(uint8, uint8, uint8, uint8) error {
	renderer.api.record("set-color")
	return nil
}
func (renderer *fakeNativeRenderer) setDrawBlendMode() error {
	renderer.api.record("set-blend")
	return nil
}
func (*fakeNativeRenderer) setClipRect(*nativeRect) error { return nil }
func (*fakeNativeRenderer) fillRect(nativeFRect) error    { return nil }
func (renderer *fakeNativeRenderer) renderGeometry(texture nativeTexture, vertices []nativeVertex, _ []int32) error {
	renderer.geometryTexture = texture != nil
	renderer.geometryVertices = append(renderer.geometryVertices[:0], vertices...)
	return nil
}
func (renderer *fakeNativeRenderer) createTextTexture(int, int) (nativeTexture, error) {
	renderer.textures++
	renderer.lastTexture = &fakeNativeTexture{api: renderer.api}
	return renderer.lastTexture, nil
}
func (*fakeNativeRenderer) renderTexture(nativeTexture, nativeFRect) error { return nil }
func (*fakeNativeRenderer) clear() error                                   { return nil }
func (renderer *fakeNativeRenderer) present() error {
	renderer.api.record("present")
	return nil
}
func (renderer *fakeNativeRenderer) destroy() {
	renderer.api.record("destroy-renderer")
}

type fakeNativeTexture struct {
	api      *fakeAPI
	alphaMod uint8
}

func (*fakeNativeTexture) update([]byte, int32) error            { return nil }
func (*fakeNativeTexture) setColorMod(uint8, uint8, uint8) error { return nil }
func (texture *fakeNativeTexture) setAlphaMod(alpha uint8) error {
	texture.alphaMod = alpha
	return nil
}
func (texture *fakeNativeTexture) destroy() { texture.api.record("destroy-texture") }

func validConfig() Config {
	return Config{Title: "test", Width: 640, Height: 400}
}

func TestEmbeddedLibraryLoadsAndUnloadsExactlyOnce(t *testing.T) {
	api := newFakeAPI()
	driver, err := openWithAPI(api, validConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	if got := countAction(api.actions, "load"); got != 1 {
		t.Fatalf("load count = %d, want 1; actions=%v", got, api.actions)
	}
	if got := countAction(api.actions, "unload"); got != 1 {
		t.Fatalf("unload count = %d, want 1; actions=%v", got, api.actions)
	}
}

func TestLoadFailureStopsBeforeSDLInitialization(t *testing.T) {
	api := newFakeAPI()
	api.loadErr = errors.New("embedded load failed")
	if _, err := openWithAPI(api, validConfig()); !errors.Is(err, api.loadErr) {
		t.Fatalf("error = %v, want embedded load failure", err)
	}
	if !reflect.DeepEqual(api.actions, []string{"load"}) {
		t.Fatalf("actions = %v, want load only", api.actions)
	}
}

func TestNativeTextInputAndClipboardBoundary(t *testing.T) {
	api := newFakeAPI()
	driver, err := openWithAPI(api, validConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if err := driver.StartTextInput(); err != nil {
		t.Fatal(err)
	}
	if err := driver.SetTextInputArea(backend.TextInputArea{X: 10, Y: 20, Width: 30, Height: 40, Cursor: 5}); err != nil {
		t.Fatal(err)
	}
	if api.window.textArea != (backend.TextInputArea{X: 10, Y: 20, Width: 30, Height: 40, Cursor: 5}) {
		t.Fatalf("native text area = %+v", api.window.textArea)
	}
	if err := driver.ClearComposition(); err != nil {
		t.Fatal(err)
	}
	if err := driver.SetClipboardText("中文"); err != nil {
		t.Fatal(err)
	}
	if value, err := driver.ClipboardText(); err != nil || value != "中文" {
		t.Fatalf("clipboard = %q/%v", value, err)
	}
	if err := driver.StopTextInput(); err != nil {
		t.Fatal(err)
	}
	want := []string{"text-start", "text-area", "composition-clear", "clipboard-set", "clipboard-get", "text-stop"}
	if got := api.actions[len(api.actions)-len(want):]; !reflect.DeepEqual(got, want) {
		t.Fatalf("text actions = %v", got)
	}
}

func TestLifecycleIsStrictlyReversedOnCallbackError(t *testing.T) {
	api := newFakeAPI()
	wantErr := errors.New("root failed")
	_, err := withAPI(api, validConfig(), func(backend.Runtime) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want callback error", err)
	}
	wantTail := []string{"destroy-renderer", "destroy-window", "quit", "unload"}
	if got := api.actions[len(api.actions)-len(wantTail):]; !reflect.DeepEqual(got, wantTail) {
		t.Fatalf("lifecycle tail = %v, want %v", got, wantTail)
	}
}

func TestLifecycleIsReleasedOnCallbackPanic(t *testing.T) {
	api := newFakeAPI()
	_, err := withAPI(api, validConfig(), func(backend.Runtime) error { panic("root panic") })
	if err == nil {
		t.Fatal("panic was not converted to an error")
	}
	wantTail := []string{"destroy-renderer", "destroy-window", "quit", "unload"}
	if got := api.actions[len(api.actions)-len(wantTail):]; !reflect.DeepEqual(got, wantTail) {
		t.Fatalf("lifecycle tail = %v, want %v", got, wantTail)
	}
}

func TestDefaultRendererFallsBackToNamedSoftware(t *testing.T) {
	api := newFakeAPI()
	api.window.rendererErrs = map[string]error{"": errors.New("no gpu")}
	config := validConfig()
	config.Diagnostics = true
	driver, err := openWithAPI(api, config)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	wantAttempts := []string{"create-renderer:", "create-renderer:software"}
	var attempts []string
	for _, action := range api.actions {
		if len(action) >= len("create-renderer:") && action[:len("create-renderer:")] == "create-renderer:" {
			attempts = append(attempts, action)
		}
	}
	if !reflect.DeepEqual(attempts, wantAttempts) {
		t.Fatalf("renderer attempts = %v, want %v", attempts, wantAttempts)
	}
	diagnostics := driver.Diagnostics()
	if diagnostics.RendererName != "software" || !diagnostics.SoftwareFallback {
		t.Fatalf("diagnostics = %+v, want software fallback", diagnostics)
	}
	if diagnostics.WindowCreates != 1 || diagnostics.RendererCreateAttempts != 2 || diagnostics.RendererCreates != 1 {
		t.Fatalf("startup creation diagnostics = %+v, want one window and one renderer from two attempts", diagnostics)
	}
}

func TestForcedSoftwareSkipsDefaultRenderer(t *testing.T) {
	api := newFakeAPI()
	config := validConfig()
	config.Software = true
	driver, err := openWithAPI(api, config)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	for _, action := range api.actions {
		if action == "create-renderer:" {
			t.Fatal("forced software attempted default renderer")
		}
	}
}

func TestRendererStartupFailureReleasesWindowSDLAndLibrary(t *testing.T) {
	api := newFakeAPI()
	api.window.rendererErrs = map[string]error{
		"": errors.New("no gpu"), "software": errors.New("no software"),
	}
	_, err := openWithAPI(api, validConfig())
	if err == nil || !errors.Is(err, api.window.rendererErrs["software"]) {
		t.Fatalf("error = %v, want software renderer failure", err)
	}
	wantTail := []string{"destroy-window", "quit", "unload"}
	if got := api.actions[len(api.actions)-len(wantTail):]; !reflect.DeepEqual(got, wantTail) {
		t.Fatalf("lifecycle tail = %v, want %v", got, wantTail)
	}
}

func TestInitFailureStillQuitsAndUnloads(t *testing.T) {
	api := newFakeAPI()
	api.initErr = errors.New("partial init")
	_, err := openWithAPI(api, validConfig())
	if !errors.Is(err, api.initErr) {
		t.Fatalf("error = %v, want init error", err)
	}
	wantTail := []string{"init", "quit", "unload"}
	if got := api.actions[len(api.actions)-len(wantTail):]; !reflect.DeepEqual(got, wantTail) {
		t.Fatalf("lifecycle tail = %v, want %v", got, wantTail)
	}
	if countAction(api.actions, "create-window") != 0 {
		t.Fatalf("initialization failure continued to window creation: %v", api.actions)
	}
}

func TestEmbeddedSDLVersionMismatchStopsBeforeInitialization(t *testing.T) {
	api := newFakeAPI()
	api.versionValue = "3.3.0"
	_, err := openWithAPI(api, validConfig())
	if err == nil || countAction(api.actions, "init") != 0 || countAction(api.actions, "create-window") != 0 {
		t.Fatalf("error/actions = %v/%v, want version failure before init", err, api.actions)
	}
	if got := api.actions[len(api.actions)-1]; got != "unload" {
		t.Fatalf("version failure tail = %v, want unload", api.actions)
	}
}

func TestNonMainThreadIsRejectedAndCleanedUp(t *testing.T) {
	api := newFakeAPI()
	api.mainThread = false
	_, err := openWithAPI(api, validConfig())
	if err == nil || !reflect.DeepEqual(api.actions[len(api.actions)-2:], []string{"quit", "unload"}) {
		t.Fatalf("error/actions = %v/%v, want main-thread error and cleanup", err, api.actions)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	api := newFakeAPI()
	driver, err := openWithAPI(api, validConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"destroy-renderer", "destroy-window", "quit", "unload"} {
		count := 0
		for _, got := range api.actions {
			if got == action {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("%s count = %d, want 1; actions=%v", action, count, api.actions)
		}
	}
}

func TestNativeEventsTranslateWithoutBackendValues(t *testing.T) {
	renderer := &fakeNativeRenderer{api: newFakeAPI(), coordDiv: 2}
	driver := &Driver{renderer: renderer}
	tests := []struct {
		name   string
		native nativeEvent
		kind   platform.EventKind
		check  func(platform.Event) bool
	}{
		{"move", nativeEvent{kind: nativeEventMouseMove, x: 20, y: 10}, platform.EventMouseMove, func(event platform.Event) bool { return event.Pointer.X == 10 && event.Pointer.Y == 5 }},
		{"down", nativeEvent{kind: nativeEventMouseDown, x: 8, y: 6, button: platform.MouseButtonPrimary, clicks: 2}, platform.EventMouseDown, func(event platform.Event) bool {
			return event.Pointer.Button == platform.MouseButtonPrimary && event.Pointer.Clicks == 2
		}},
		{"up", nativeEvent{kind: nativeEventMouseUp, button: platform.MouseButtonPrimary}, platform.EventMouseUp, nil},
		{"wheel", nativeEvent{kind: nativeEventMouseWheel, wheelX: 1, wheelY: -2}, platform.EventMouseWheel, func(event platform.Event) bool { return event.Pointer.WheelX == 1 && event.Pointer.WheelY == -2 }},
		{"key-down", nativeEvent{kind: nativeEventKeyDown, key: platform.KeyTab, modifiers: platform.Modifiers{Shift: true}, repeat: true}, platform.EventKeyDown, func(event platform.Event) bool {
			return event.Key.Key == platform.KeyTab && event.Key.Modifiers.Shift && event.Key.Repeat
		}},
		{"key-up", nativeEvent{kind: nativeEventKeyUp, key: platform.KeySpace}, platform.EventKeyUp, nil},
		{"focus-gained", nativeEvent{kind: nativeEventWindowFocusGained}, platform.EventWindowFocusGained, nil},
		{"focus-lost", nativeEvent{kind: nativeEventWindowFocusLost}, platform.EventWindowFocusLost, nil},
		{"mouse-leave", nativeEvent{kind: nativeEventWindowMouseLeave}, platform.EventWindowMouseLeave, nil},
		{"editing", nativeEvent{kind: nativeEventTextEditing, text: "拼", editingStart: 0, editingLength: 1}, platform.EventTextEditing, func(event platform.Event) bool { return event.Text.Text == "拼" && event.Text.EditingLength == 1 }},
		{"text", nativeEvent{kind: nativeEventTextInput, text: "字"}, platform.EventTextInput, func(event platform.Event) bool { return event.Text.Text == "字" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event, err := driver.translate(test.native)
			if err != nil {
				t.Fatal(err)
			}
			if event.Kind != test.kind || test.check != nil && !test.check(event) {
				t.Fatalf("translated event = %#v", event)
			}
		})
	}
}

func TestResizeAndScaleEventsRefreshTypedViewport(t *testing.T) {
	api := newFakeAPI()
	config := validConfig()
	config.Diagnostics = true
	driver, err := openWithAPI(api, config)
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	api.window.viewportValue = platform.Viewport{
		LogicalWidth: 800, LogicalHeight: 500, PixelWidth: 1200, PixelHeight: 750,
		PixelDensity: 1.5, DisplayScale: 1.5,
	}
	event, err := driver.translate(nativeEvent{kind: nativeEventResize})
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != platform.EventResize || event.Viewport != api.window.viewportValue {
		t.Fatalf("resize event = %#v", event)
	}
	setScales := countAction(api.actions, "set-scale")
	event, err = driver.translate(nativeEvent{kind: nativeEventScale})
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != platform.EventWake {
		t.Fatalf("unchanged scale event = %#v, want no-op wake", event)
	}
	if got := countAction(api.actions, "set-scale"); got != setScales {
		t.Fatalf("unchanged viewport set renderer scale: before=%d after=%d", setScales, got)
	}
	diagnostics := driver.Diagnostics()
	if diagnostics.ResizeEvents != 1 || diagnostics.ScaleEvents != 1 || diagnostics.NoopViewportEvents != 1 {
		t.Fatalf("viewport event diagnostics = %+v", diagnostics)
	}
}

func TestWindowIsShownOnceAfterFirstCompletePresent(t *testing.T) {
	api := newFakeAPI()
	driver, err := openWithAPI(api, validConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	if countAction(api.actions, "show-window") != 0 {
		t.Fatalf("window shown during initialization: %v", api.actions)
	}
	if err := driver.Render(nil); err != nil {
		t.Fatal(err)
	}
	if countAction(api.actions, "show-window") != 0 {
		t.Fatalf("window shown before a complete present: %v", api.actions)
	}
	if err := driver.Present(); err != nil {
		t.Fatal(err)
	}
	wantTail := []string{"present", "show-window"}
	if got := api.actions[len(api.actions)-len(wantTail):]; !reflect.DeepEqual(got, wantTail) {
		t.Fatalf("first-frame tail = %v, want %v", got, wantTail)
	}
	if err := driver.Present(); err != nil {
		t.Fatal(err)
	}
	if countAction(api.actions, "show-window") != 1 || countAction(api.actions, "present") != 2 {
		t.Fatalf("present/show counts are not one-shot: %v", api.actions)
	}
}

func TestWindowShowFailureStopsAfterPresentedFrame(t *testing.T) {
	api := newFakeAPI()
	api.window.showErr = errors.New("show failed")
	driver, err := openWithAPI(api, validConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	err = driver.Present()
	if !errors.Is(err, api.window.showErr) {
		t.Fatalf("present error = %v, want show failure", err)
	}
	wantTail := []string{"present", "show-window"}
	if got := api.actions[len(api.actions)-len(wantTail):]; !reflect.DeepEqual(got, wantTail) {
		t.Fatalf("failed first-frame tail = %v, want %v", got, wantTail)
	}
}

func TestScaleChangeUsesOutputRatioAndInvalidatesTextures(t *testing.T) {
	api := newFakeAPI()
	driver, err := openWithAPI(api, validConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	bitmap := &paint.TextBitmap{Key: "scale-2", Width: 2, Height: 2, Pixels: []byte{255, 255, 255, 255}}
	if err := driver.DrawText(paint.Rect{Width: 1, Height: 1}, bitmap, paint.Color{A: 255}); err != nil {
		t.Fatal(err)
	}
	destroyed := countAction(api.actions, "destroy-texture")
	api.window.viewportValue = platform.Viewport{
		LogicalWidth: 800, LogicalHeight: 500, PixelWidth: 1200, PixelHeight: 750,
		PixelDensity: 1.5, DisplayScale: 1.5,
	}
	if _, err := driver.translate(nativeEvent{kind: nativeEventScale}); err != nil {
		t.Fatal(err)
	}
	native := driver.renderer.(*fakeNativeRenderer)
	if native.scaleX != 1.5 || native.scaleY != 1.5 {
		t.Fatalf("renderer scale = %gx%g, want output/logical 1.5x1.5", native.scaleX, native.scaleY)
	}
	if driver.textCache.Stats().Entries != 0 || countAction(api.actions, "destroy-texture") != destroyed+1 {
		t.Fatalf("scale change retained old texture: stats=%+v actions=%v", driver.textCache.Stats(), api.actions)
	}
}

func TestWindowEventRoutingDropsUnregisteredNativeID(t *testing.T) {
	root := &Driver{logicalID: 1, windows: make(map[uint32]*Driver)}
	child := &Driver{logicalID: 2}
	root.windows[41] = child
	event, err := root.translate(nativeEvent{kind: nativeEventKeyDown, windowID: 41, key: platform.KeyEnter})
	if err != nil || event.Window != 2 || event.Kind != platform.EventKeyDown {
		t.Fatalf("routed event = %#v, %v", event, err)
	}
	delete(root.windows, 41)
	event, err = root.translate(nativeEvent{kind: nativeEventKeyDown, windowID: 41})
	if err != nil || event.Kind != platform.EventWake || event.Window != 0 {
		t.Fatalf("stale event = %#v, %v", event, err)
	}
}

func TestChildWindowRendererFailureDestroysPartialWindow(t *testing.T) {
	api := newFakeAPI()
	api.window.rendererErrs = map[string]error{"software": errors.New("renderer failed")}
	_, err := openWindowWithAPI(api, Config{Title: "child", Width: 320, Height: 200, Software: true})
	if !errors.Is(err, api.window.rendererErrs["software"]) {
		t.Fatalf("error = %v", err)
	}
	if !slices.Contains(api.actions, "destroy-window") {
		t.Fatalf("actions = %v", api.actions)
	}
}

func TestDisplayContentScaleDefinesLogicalViewportAndTextInputArea(t *testing.T) {
	api := newFakeAPI()
	api.window.viewportValue = platform.Viewport{
		LogicalWidth: 640, LogicalHeight: 400, PixelWidth: 640, PixelHeight: 400,
		PixelDensity: 1, DisplayScale: 1.25,
	}
	driver, err := openWithAPI(api, Config{Title: "test", Width: 720, Height: 544})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	viewport := driver.Diagnostics().Viewport
	if viewport.LogicalWidth != 720 || viewport.LogicalHeight != 544 {
		t.Fatalf("DPI-normalized viewport = %+v, want 720x544 logical", viewport)
	}
	if countAction(api.actions, "window-size") != 1 || api.window.viewportValue.LogicalWidth != 900 || api.window.viewportValue.LogicalHeight != 680 {
		t.Fatalf("initial logical size was not mapped once: viewport=%+v actions=%v", api.window.viewportValue, api.actions)
	}
	native := driver.renderer.(*fakeNativeRenderer)
	if native.scaleX != 1.25 || native.scaleY != 1.25 {
		t.Fatalf("renderer scale = %gx%g, want 1.25x1.25", native.scaleX, native.scaleY)
	}
	if got, scaleErr := scaleTextInputArea(backend.TextInputArea{X: 8, Y: 16, Width: 24, Height: 32, Cursor: 4}, 1.25, 1.25); scaleErr != nil || got != (backend.TextInputArea{X: 10, Y: 20, Width: 30, Height: 40, Cursor: 5}) {
		t.Fatalf("scaled text input area = %+v, %v", got, scaleErr)
	}
	if err := driver.SetTextInputArea(backend.TextInputArea{X: 8, Y: 16, Width: 24, Height: 32, Cursor: 4}); err != nil || api.window.textArea != (backend.TextInputArea{X: 10, Y: 20, Width: 30, Height: 40, Cursor: 5}) {
		t.Fatalf("native scaled text input area = %+v, %v", api.window.textArea, err)
	}
}

func TestSoftwareRendererUsesPresentableSurfaceSize(t *testing.T) {
	api := newFakeAPI()
	api.window.viewportValue = platform.Viewport{
		LogicalWidth: 800, LogicalHeight: 600, PixelWidth: 1000, PixelHeight: 750,
		PixelDensity: 1, DisplayScale: 1.25,
	}
	driver, err := openWithAPI(api, Config{Title: "test", Width: 640, Height: 480, Software: true})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	viewport := driver.Diagnostics().Viewport
	if viewport.LogicalWidth != 640 || viewport.LogicalHeight != 480 ||
		viewport.PixelWidth != 640 || viewport.PixelHeight != 480 {
		t.Fatalf("software viewport = %+v, want matching 640x480 logical and pixel surface", viewport)
	}
	native := driver.renderer.(*fakeNativeRenderer)
	if native.scaleX != 1 || native.scaleY != 1 {
		t.Fatalf("software renderer scale = %gx%g, want presentable-surface scale 1x1", native.scaleX, native.scaleY)
	}
}
