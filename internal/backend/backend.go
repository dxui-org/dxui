// Package backend groups the ports required by the UI runtime. Concrete SDL
// values never cross this package boundary.
package backend

import (
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
	"github.com/dxui-org/dxui/internal/text"
)

// Diagnostics reports backend-neutral native runtime state. Sizes preserve a
// deliberate logical/pixel conversion boundary for high-density displays.
type Diagnostics struct {
	SDLVersion                                             string
	RendererName                                           string
	Viewport                                               platform.Viewport
	Frames                                                 uint64
	SoftwareFallback                                       bool
	WindowCreates, RendererCreateAttempts, RendererCreates uint64
	ExposeEvents, ResizeEvents, ScaleEvents                uint64
	NoopViewportEvents, RendererResetEvents                uint64
	TextureCreates                                         uint64
	TextureDestroys                                        uint64
	CacheBytes                                             uint64
	CacheBudgetBytes                                       uint64
	CacheEntries                                           uint64
}

// TextInputArea is the candidate-window anchor in logical window units.
// Cursor is a horizontal offset within the rectangle.
type TextInputArea struct {
	X, Y, Width, Height int32
	Cursor              int32
}

// TextInput is the UI-thread-only native text/clipboard boundary.
type TextInput interface {
	StartTextInput() error
	StopTextInput() error
	ClearComposition() error
	SetTextInputArea(TextInputArea) error
	ClipboardText() (string, error)
	SetClipboardText(string) error
}

// Runtime is the lifecycle surface consumed by the root application package.
// A concrete implementation owns all services until Close returns.
type Runtime interface {
	Events() platform.EventSource
	Renderer() renderer.Renderer
	Diagnostics() Diagnostics
	Close() error
}

// WindowConfig contains backend-neutral inputs for an additional top-level
// native window.
type WindowConfig struct {
	Title               string
	Width, Height       int32
	MinWidth, MinHeight int32
	Background          [4]uint8
	Software            bool
	ShadowCacheBytes    int
	GlyphCacheBytes     int
	ImageCacheBytes     int
	Diagnostics         bool
}

// WindowRuntime owns one native window and its renderer. Close unregisters
// its native ID before destruction so queued stale events cannot be rerouted.
type WindowRuntime interface {
	Renderer() renderer.Renderer
	Diagnostics() Diagnostics
	TextInput() TextInput
	SetTitle(string) error
	SetSize(int32, int32) error
	Maximize() error
	Minimize() error
	Restore() error
	Close() error
}

// MultiWindowRuntime extends the one-App lifecycle with child-window
// creation. Its Events stream assigns stable logical WindowIDs.
type MultiWindowRuntime interface {
	Runtime
	MainWindowID() platform.WindowID
	CreateWindow(WindowConfig) (platform.WindowID, WindowRuntime, error)
}

// TextInputRuntime is implemented by runtimes that own a native text-input
// service. It is separate so headless renderer fakes remain minimal.
type TextInputRuntime interface{ TextInput() TextInput }

// Backend is the minimal set of replaceable runtime services.
type Backend interface {
	Events() platform.EventSource
	Renderer() renderer.Renderer
	TextMeasurer() text.Measurer
}
