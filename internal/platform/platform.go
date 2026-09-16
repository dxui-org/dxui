// Package platform defines backend-neutral time and event ports.
package platform

import (
	"context"
	"time"
)

// Clock makes scheduling deterministic in tests.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production wall/monotonic clock adapter.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// EventKind is a normalized event category, not an SDL enum.
type EventKind uint8

const (
	EventWake EventKind = iota
	EventExpose
	EventClose
	EventResize
	EventScale
	EventDeadline
	EventMouseMove
	EventMouseDown
	EventMouseUp
	EventMouseWheel
	EventKeyDown
	EventKeyUp
	EventWindowFocusGained
	EventWindowFocusLost
	EventWindowMouseLeave
	EventWindowMinimized
	EventWindowMaximized
	EventWindowRestored
	// EventTextEditing and EventTextInput reserve the native IME/commit path.
	EventTextEditing
	EventTextInput
)

// MouseButton is a backend-neutral physical mouse button.
type MouseButton uint8

const (
	MouseButtonNone MouseButton = iota
	MouseButtonPrimary
	MouseButtonMiddle
	MouseButtonSecondary
	MouseButtonX1
	MouseButtonX2
)

// Key is the semantic key subset consumed by the interaction controller.
// KeyOther deliberately preserves no SDL/native code.
type Key uint8

const (
	KeyOther Key = iota
	KeyTab
	KeyEnter
	KeySpace
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyBackspace
	KeyDelete
	KeyA
	KeyC
	KeyX
	KeyV
	KeyZ
	KeyY
	KeyEscape
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

// Modifiers are normalized keyboard modifiers.
type Modifiers struct {
	Shift, Control, Alt, Super bool
	// Primary is Command on macOS and Control elsewhere. Keeping this decision
	// at the native boundary avoids platform guesses in the editor.
	Primary bool
}

// PointerEvent uses renderer logical coordinates.
type PointerEvent struct {
	X, Y   float32
	Button MouseButton
	Clicks uint8
	WheelX float32
	WheelY float32
}

// KeyEvent contains only backend-neutral semantics needed by dxui.
type KeyEvent struct {
	Key       Key
	Modifiers Modifiers
	Repeat    bool
}

// TextEvent reserves committed and editing text plumbing. EditingStart and
// EditingLength count Unicode code points and are -1 when unavailable.
type TextEvent struct {
	Text                        string
	EditingStart, EditingLength int
}

// Viewport is the backend-neutral boundary between logical window coordinates
// and physical renderer pixels.
type Viewport struct {
	LogicalWidth, LogicalHeight int32
	PixelWidth, PixelHeight     int32
	PixelDensity, DisplayScale  float32
}

// WindowID is a runtime-assigned, backend-neutral window identity. Values are
// never reused during one runtime, even when a native backend reuses its ID.
type WindowID uint64

// Event is the backend-neutral event envelope used by the runtime.
type Event struct {
	Window   WindowID
	Kind     EventKind
	Viewport Viewport
	Pointer  PointerEvent
	Key      KeyEvent
	Text     TextEvent
}

// EventSource blocks until an event, context cancellation, or deadline. A nil
// deadline means wait indefinitely. Implementations must not poll.
type EventSource interface {
	Wait(context.Context, *time.Time) (Event, error)
	Poll() (Event, bool, error)
	Wake() error
}
