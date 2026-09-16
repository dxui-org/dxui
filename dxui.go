// Package dxui contains the public, backend-independent API for dxui.
//
// The current package includes the SDL3 application/window runtime, immutable
// view descriptions, deterministic ADR-0005 layout, backend-neutral paint
// commands, typed runtime themes, pure-Go text, the first semantic controls,
// lightweight vector icons, guarded pure-Go raster images, and controlled
// Input/Textarea editors with native text-input composition plumbing.
//
// The API is pre-v1. During v0.x, incompatible corrections may be made without
// deprecated aliases; release notes and the public API audit record each one.
package dxui

import "github.com/dxui-org/dxui/internal/icondata"

// Option distinguishes an explicitly supplied zero value from an unset value.
type Option[T any] struct {
	value T
	set   bool
}

// Some creates a set option.
func Some[T any](value T) Option[T] { return Option[T]{value: value, set: true} }

func (o Option[T]) get() (T, bool) { return o.value, o.set }

// Point is a position or offset in logical units.
type Point = icondata.Point

// Size is a width and height in logical units.
type Size struct{ Width, Height float32 }

// RGBAColor is an 8-bit non-premultiplied RGBA color.
//
// The former Color type name is now the discoverable color-token namespace.
type RGBAColor struct{ R, G, B, A uint8 }

// RGBA creates an 8-bit non-premultiplied color.
func RGBA(r, g, b, a uint8) RGBAColor { return RGBAColor{R: r, G: g, B: b, A: a} }

// RendererPreference selects the preferred renderer creation policy.
type RendererPreference uint8

const (
	RendererAuto RendererPreference = iota
	RendererSoftware
)

// CacheBudgets bounds CPU and renderer-owned text, icon, and image resources. Zero
// selects defaults: FontBytes 32 MiB, TextSourceBytes 2 MiB, GlyphBytes 2 MiB,
// TextMeasureBytes 1 MiB, ImageBytes 2 MiB, and ShadowBytes 2 MiB. ImageBytes
// bounds inactive reusable CPU pixels and renderer textures. Unique images in
// the committed display are working-set resources charged at four bytes per
// source pixel until that display releases them. A negative cache budget
// disables that cache; negative FontBytes or TextSourceBytes permits no
// application fonts or retained text/icon masks respectively.
type CacheBudgets struct {
	FontBytes        int
	TextSourceBytes  int
	GlyphBytes       int
	TextMeasureBytes int
	ImageBytes       int
	ShadowBytes      int
}

// AppOptions configures an App's main window and shared runtime. Its
// zero value selects documented window, renderer, cache, font, and theme
// defaults; invalid dimensions, renderer values, fonts, or themes are reported
// by App.Run before native event processing begins.
type AppOptions struct {
	Title               string
	Width, Height       float32
	MinWidth, MinHeight float32
	Renderer            RendererPreference
	Background          RGBAColor
	Caches              CacheBudgets
	Fonts               []Font
	DefaultFont         FontFamily
	// DisableSystemFontFallback prevents lazy deterministic system-CJK font
	// loading. The zero value enables fallback after all application fonts and
	// dxui's built-in Latin font.
	DisableSystemFontFallback bool
	Theme                     Theme
	Shortcuts                 []Shortcut
	// OnCloseRequest handles a native window-close request on the UI thread.
	// A nil callback closes the App. A non-nil callback must call Close when it
	// accepts the request.
	OnCloseRequest func(*App)
	// OnError observes recoverable build and callback failures on the UI
	// thread. When nil, the failure terminates Run and is returned.
	OnError func(error)
	// OnShown runs once on the UI thread after the complete first frame was
	// presented and the native window was shown successfully. It is not called
	// after startup failure or on later builds/presents.
	OnShown func(*App)
	// Diagnostics enables bounded event/timing/resource counters.
	// It is false by default so release event and render paths avoid the work.
	Diagnostics bool
}

// WindowOptions configures an independent native child window. Theme, fonts,
// renderer preference, cache budgets and App shortcuts are inherited from the
// owning App. Callbacks execute on the App UI thread.
type WindowOptions struct {
	Title               string
	Width, Height       float32
	MinWidth, MinHeight float32
	Background          RGBAColor
	Shortcuts           []Shortcut
	// OnCloseRequest may reject a native close request by returning without
	// calling Window.Close. A nil callback accepts the request.
	OnCloseRequest func(*Window)
	// OnShown runs once after the complete first frame is presented and the
	// hidden native window has been shown successfully.
	OnShown func(*Window)
}

// ShortcutKey is a backend-neutral semantic key used by an App shortcut.
type ShortcutKey uint8

const (
	KeyEnter ShortcutKey = iota + 1
	KeyBackspace
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
	KeyPlus
	KeyMinus
	KeyMultiply
	KeyDivide
	KeyDecimal
	KeyEquals
)

// ShortcutModifiers are matched exactly. Primary substitutes for Command on
// macOS and Control elsewhere; callers do not also set that physical field.
type ShortcutModifiers struct{ Shift, Control, Alt, Super, Primary bool }

// Shortcut binds one application-window key chord to a semantic action.
// Focused editors and built-in control keys have priority. Repeat enables
// repeated key-down activation; otherwise native repeat is consumed silently.
type Shortcut struct {
	Key       ShortcutKey
	Modifiers ShortcutModifiers
	Repeat    bool
	OnPress   func()
}
