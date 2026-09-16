package sdl3

import (
	"fmt"
	"runtime"

	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/go-sdl3/bin/binsdl"
	"github.com/dxui-org/go-sdl3/sdl"
)

type systemAPI struct{}

type embeddedLibrary struct {
	unloadFunc func()
}

func (library embeddedLibrary) unload() error {
	library.unloadFunc()
	return nil
}

func (systemAPI) load() (nativeLibrary, error) {
	library := binsdl.Load()
	return embeddedLibrary{unloadFunc: library.Unload}, nil
}

func (systemAPI) version() string                     { return sdl.GetVersion().String() }
func (systemAPI) init() error                         { return sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS) }
func (systemAPI) quit()                               { sdl.Quit() }
func (systemAPI) isMainThread() bool                  { return sdl.IsMainThread() }
func (systemAPI) pushWake() error                     { return sdl.PushEvent(&sdl.Event{Type: sdl.EVENT_USER}) }
func (systemAPI) clipboardText() (string, error)      { return sdl.GetClipboardText() }
func (systemAPI) setClipboardText(value string) error { return sdl.SetClipboardText(value) }

func (systemAPI) createWindow(title string, width, height int32) (nativeWindow, error) {
	window, err := sdl.CreateWindow(title, int(width), int(height), sdl.WINDOW_RESIZABLE|sdl.WINDOW_HIGH_PIXEL_DENSITY|sdl.WINDOW_HIDDEN)
	if err != nil {
		return nil, err
	}
	return systemWindow{window}, nil
}

func (systemAPI) waitEvent() (nativeEvent, error) {
	var event sdl.Event
	if err := sdl.WaitEvent(&event); err != nil {
		return nativeEvent{}, err
	}
	return translateNativeEvent(&event), nil
}

func (systemAPI) waitEventTimeout(milliseconds int32) (nativeEvent, bool) {
	var event sdl.Event
	if !sdl.WaitEventTimeout(&event, milliseconds) {
		return nativeEvent{}, false
	}
	return translateNativeEvent(&event), true
}

func (systemAPI) pollEvent() (nativeEvent, bool) {
	var event sdl.Event
	if !sdl.PollEvent(&event) {
		return nativeEvent{}, false
	}
	return translateNativeEvent(&event), true
}

func translateNativeEvent(event *sdl.Event) nativeEvent {
	if event == nil {
		return nativeEvent{kind: nativeEventOther}
	}
	switch event.Type {
	case sdl.EVENT_USER:
		return nativeEvent{kind: nativeEventWake}
	case sdl.EVENT_QUIT, sdl.EVENT_WINDOW_CLOSE_REQUESTED:
		return nativeEvent{kind: nativeEventClose, windowID: nativeWindowID(event)}
	case sdl.EVENT_WINDOW_EXPOSED:
		return nativeEvent{kind: nativeEventExpose, windowID: nativeWindowID(event)}
	case sdl.EVENT_RENDER_TARGETS_RESET, sdl.EVENT_RENDER_DEVICE_RESET:
		return nativeEvent{kind: nativeEventRendererReset, windowID: uint32(event.RenderEvent().WindowID)}
	case sdl.EVENT_WINDOW_RESIZED, sdl.EVENT_WINDOW_PIXEL_SIZE_CHANGED:
		return nativeEvent{kind: nativeEventResize, windowID: nativeWindowID(event)}
	case sdl.EVENT_WINDOW_DISPLAY_CHANGED, sdl.EVENT_WINDOW_DISPLAY_SCALE_CHANGED,
		sdl.EVENT_DISPLAY_CONTENT_SCALE_CHANGED:
		return nativeEvent{kind: nativeEventScale, windowID: nativeWindowID(event)}
	case sdl.EVENT_MOUSE_MOTION:
		value := event.MouseMotionEvent()
		return nativeEvent{kind: nativeEventMouseMove, windowID: uint32(value.WindowID), x: value.X, y: value.Y}
	case sdl.EVENT_MOUSE_BUTTON_DOWN, sdl.EVENT_MOUSE_BUTTON_UP:
		value := event.MouseButtonEvent()
		kind := nativeEventMouseDown
		if event.Type == sdl.EVENT_MOUSE_BUTTON_UP {
			kind = nativeEventMouseUp
		}
		return nativeEvent{kind: kind, windowID: uint32(value.WindowID), x: value.X, y: value.Y, button: mouseButton(value.Button), clicks: value.Clicks}
	case sdl.EVENT_MOUSE_WHEEL:
		value := event.MouseWheelEvent()
		wheelX, wheelY := value.X, value.Y
		if value.Direction == sdl.MOUSEWHEEL_FLIPPED {
			wheelX, wheelY = -wheelX, -wheelY
		}
		return nativeEvent{kind: nativeEventMouseWheel, windowID: uint32(value.WindowID), x: value.MouseX, y: value.MouseY, wheelX: wheelX, wheelY: wheelY}
	case sdl.EVENT_KEY_DOWN, sdl.EVENT_KEY_UP:
		value := event.KeyboardEvent()
		kind := nativeEventKeyDown
		if event.Type == sdl.EVENT_KEY_UP {
			kind = nativeEventKeyUp
		}
		modifiers := platform.Modifiers{
			Shift: value.Mod&sdl.KMOD_SHIFT != 0, Control: value.Mod&sdl.KMOD_CTRL != 0,
			Alt: value.Mod&sdl.KMOD_ALT != 0, Super: value.Mod&sdl.KMOD_GUI != 0,
		}
		if runtime.GOOS == "darwin" {
			modifiers.Primary = modifiers.Super
		} else {
			modifiers.Primary = modifiers.Control
		}
		return nativeEvent{kind: kind, windowID: uint32(value.WindowID), key: keyboardKey(value.Key), modifiers: modifiers, repeat: value.Repeat}
	case sdl.EVENT_WINDOW_FOCUS_GAINED:
		return nativeEvent{kind: nativeEventWindowFocusGained, windowID: nativeWindowID(event)}
	case sdl.EVENT_WINDOW_FOCUS_LOST:
		return nativeEvent{kind: nativeEventWindowFocusLost, windowID: nativeWindowID(event)}
	case sdl.EVENT_WINDOW_MOUSE_LEAVE:
		return nativeEvent{kind: nativeEventWindowMouseLeave, windowID: nativeWindowID(event)}
	case sdl.EVENT_WINDOW_MINIMIZED:
		return nativeEvent{kind: nativeEventWindowMinimized, windowID: nativeWindowID(event)}
	case sdl.EVENT_WINDOW_MAXIMIZED:
		return nativeEvent{kind: nativeEventWindowMaximized, windowID: nativeWindowID(event)}
	case sdl.EVENT_WINDOW_RESTORED:
		return nativeEvent{kind: nativeEventWindowRestored, windowID: nativeWindowID(event)}
	case sdl.EVENT_TEXT_EDITING:
		value := event.TextEditingEvent()
		return nativeEvent{kind: nativeEventTextEditing, windowID: uint32(value.WindowID), text: value.Text, editingStart: int(value.Start), editingLength: int(value.Length)}
	case sdl.EVENT_TEXT_INPUT:
		value := event.TextInputEvent()
		return nativeEvent{kind: nativeEventTextInput, windowID: uint32(value.WindowID), text: value.Text}
	default:
		return nativeEvent{kind: nativeEventOther}
	}
}

func nativeWindowID(event *sdl.Event) uint32 { return uint32(event.WindowEvent().WindowID) }

func mouseButton(value uint8) platform.MouseButton {
	switch sdl.MouseButtonFlags(value) {
	case sdl.BUTTON_LEFT:
		return platform.MouseButtonPrimary
	case sdl.BUTTON_MIDDLE:
		return platform.MouseButtonMiddle
	case sdl.BUTTON_RIGHT:
		return platform.MouseButtonSecondary
	case sdl.BUTTON_X1:
		return platform.MouseButtonX1
	case sdl.BUTTON_X2:
		return platform.MouseButtonX2
	default:
		return platform.MouseButtonNone
	}
}

func keyboardKey(value sdl.Keycode) platform.Key {
	switch value {
	case sdl.K_TAB:
		return platform.KeyTab
	case sdl.K_RETURN, sdl.K_RETURN2, sdl.K_KP_ENTER:
		return platform.KeyEnter
	case sdl.K_SPACE:
		return platform.KeySpace
	case sdl.K_LEFT:
		return platform.KeyLeft
	case sdl.K_RIGHT:
		return platform.KeyRight
	case sdl.K_UP:
		return platform.KeyUp
	case sdl.K_DOWN:
		return platform.KeyDown
	case sdl.K_HOME:
		return platform.KeyHome
	case sdl.K_END:
		return platform.KeyEnd
	case sdl.K_BACKSPACE:
		return platform.KeyBackspace
	case sdl.K_DELETE:
		return platform.KeyDelete
	case sdl.K_A:
		return platform.KeyA
	case sdl.K_C:
		return platform.KeyC
	case sdl.K_X:
		return platform.KeyX
	case sdl.K_V:
		return platform.KeyV
	case sdl.K_Z:
		return platform.KeyZ
	case sdl.K_Y:
		return platform.KeyY
	case sdl.K_ESCAPE:
		return platform.KeyEscape
	case sdl.K_0, sdl.K_KP_0:
		return platform.Key0
	case sdl.K_1, sdl.K_KP_1:
		return platform.Key1
	case sdl.K_2, sdl.K_KP_2:
		return platform.Key2
	case sdl.K_3, sdl.K_KP_3:
		return platform.Key3
	case sdl.K_4, sdl.K_KP_4:
		return platform.Key4
	case sdl.K_5, sdl.K_KP_5:
		return platform.Key5
	case sdl.K_6, sdl.K_KP_6:
		return platform.Key6
	case sdl.K_7, sdl.K_KP_7:
		return platform.Key7
	case sdl.K_8, sdl.K_KP_8:
		return platform.Key8
	case sdl.K_9, sdl.K_KP_9:
		return platform.Key9
	case sdl.K_PLUS, sdl.K_KP_PLUS:
		return platform.KeyPlus
	case sdl.K_MINUS, sdl.K_KP_MINUS:
		return platform.KeyMinus
	case sdl.K_ASTERISK, sdl.K_KP_MULTIPLY:
		return platform.KeyMultiply
	case sdl.K_SLASH, sdl.K_KP_DIVIDE:
		return platform.KeyDivide
	case sdl.K_PERIOD, sdl.K_KP_PERIOD:
		return platform.KeyDecimal
	case sdl.K_EQUALS, sdl.K_KP_EQUALS:
		return platform.KeyEquals
	default:
		return platform.KeyOther
	}
}

type systemWindow struct{ value *sdl.Window }

func (window systemWindow) id() (uint32, error) {
	value, err := window.value.ID()
	return uint32(value), err
}
func (window systemWindow) setTitle(title string) error { return window.value.SetTitle(title) }
func (window systemWindow) maximize() error             { return window.value.Maximize() }
func (window systemWindow) minimize() error             { return window.value.Minimize() }
func (window systemWindow) restore() error              { return window.value.Restore() }

func (window systemWindow) createRenderer(name string) (nativeRenderer, error) {
	renderer, err := window.value.CreateRenderer(name)
	if err != nil {
		return nil, err
	}
	return systemRenderer{renderer}, nil
}

func (window systemWindow) setMinimumSize(width, height int32) error {
	return window.value.SetMinimumSize(width, height)
}

func (window systemWindow) setSize(width, height int32) error {
	return window.value.SetSize(width, height)
}

func (window systemWindow) viewport() (platform.Viewport, error) {
	logicalWidth, logicalHeight, err := window.value.Size()
	if err != nil {
		return platform.Viewport{}, err
	}
	pixelWidth, pixelHeight, err := window.value.SizeInPixels()
	if err != nil {
		return platform.Viewport{}, err
	}
	pixelDensity, err := window.value.PixelDensity()
	if err != nil {
		return platform.Viewport{}, err
	}
	displayScale, err := window.value.DisplayScale()
	if err != nil {
		return platform.Viewport{}, err
	}
	return platform.Viewport{
		LogicalWidth: logicalWidth, LogicalHeight: logicalHeight,
		PixelWidth: pixelWidth, PixelHeight: pixelHeight,
		PixelDensity: pixelDensity, DisplayScale: displayScale,
	}, nil
}

func (window systemWindow) show() error             { return window.value.Show() }
func (window systemWindow) destroy()                { window.value.Destroy() }
func (window systemWindow) startTextInput() error   { return window.value.StartTextInput() }
func (window systemWindow) stopTextInput() error    { return window.value.StopTextInput() }
func (window systemWindow) clearComposition() error { return window.value.ClearComposition() }
func (window systemWindow) setTextInputArea(x, y, width, height, cursor int32) error {
	return window.value.SetTextInputArea(&sdl.Rect{X: x, Y: y, W: width, H: height}, cursor)
}

type systemRenderer struct{ value *sdl.Renderer }

func (renderer systemRenderer) name() (string, error) { return renderer.value.Name() }
func (renderer systemRenderer) windowToRender(x, y float32) (float32, float32, error) {
	return renderer.value.RenderCoordinatesFromWindow(x, y)
}
func (renderer systemRenderer) setScale(x, y float32) error {
	return renderer.value.SetScale(x, y)
}
func (renderer systemRenderer) setDrawColor(r, g, b, a uint8) error {
	return renderer.value.SetDrawColor(r, g, b, a)
}
func (renderer systemRenderer) setDrawBlendMode() error {
	return renderer.value.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
}
func (renderer systemRenderer) setClipRect(rect *nativeRect) error {
	if rect == nil {
		return renderer.value.SetClipRect(nil)
	}
	value := sdl.Rect{X: rect.X, Y: rect.Y, W: rect.Width, H: rect.Height}
	return renderer.value.SetClipRect(&value)
}
func (renderer systemRenderer) fillRect(rect nativeFRect) error {
	value := sdl.FRect{X: rect.X, Y: rect.Y, W: rect.Width, H: rect.Height}
	return renderer.value.RenderFillRect(&value)
}
func (renderer systemRenderer) renderGeometry(texture nativeTexture, vertices []nativeVertex, indices []int32) error {
	values := make([]sdl.Vertex, len(vertices))
	for index, vertex := range vertices {
		values[index] = sdl.Vertex{
			Position: sdl.FPoint{X: vertex.X, Y: vertex.Y},
			Color:    sdl.FColor{R: vertex.R, G: vertex.G, B: vertex.B, A: vertex.A},
			TexCoord: sdl.FPoint{X: vertex.U, Y: vertex.V},
		}
	}
	var native *sdl.Texture
	if texture != nil {
		value, ok := texture.(systemTexture)
		if !ok {
			return fmt.Errorf("sdl3: render geometry has incompatible texture")
		}
		native = value.value
	}
	return renderer.value.RenderGeometry(native, values, indices)
}

func (renderer systemRenderer) createTextTexture(width, height int) (nativeTexture, error) {
	texture, err := renderer.value.CreateTexture(sdl.PIXELFORMAT_RGBA32, sdl.TEXTUREACCESS_STATIC, width, height)
	if err != nil {
		return nil, err
	}
	if err := texture.SetBlendMode(sdl.BLENDMODE_BLEND); err != nil {
		texture.Destroy()
		return nil, err
	}
	if err := texture.SetScaleMode(sdl.SCALEMODE_LINEAR); err != nil {
		texture.Destroy()
		return nil, err
	}
	return systemTexture{texture}, nil
}

func (renderer systemRenderer) renderTexture(texture nativeTexture, rect nativeFRect) error {
	value, ok := texture.(systemTexture)
	if !ok {
		return fmt.Errorf("sdl3: foreign text texture")
	}
	destination := sdl.FRect{X: rect.X, Y: rect.Y, W: rect.Width, H: rect.Height}
	return renderer.value.RenderTexture(value.value, nil, &destination)
}

type systemTexture struct{ value *sdl.Texture }

func (texture systemTexture) update(pixels []byte, pitch int32) error {
	return texture.value.Update(nil, pixels, pitch)
}
func (texture systemTexture) setColorMod(r, g, b uint8) error {
	return texture.value.SetColorMod(r, g, b)
}
func (texture systemTexture) setAlphaMod(alpha uint8) error { return texture.value.SetAlphaMod(alpha) }
func (texture systemTexture) destroy()                      { texture.value.Destroy() }

func (renderer systemRenderer) clear() error   { return renderer.value.Clear() }
func (renderer systemRenderer) present() error { return renderer.value.Present() }
func (renderer systemRenderer) destroy()       { renderer.value.Destroy() }
