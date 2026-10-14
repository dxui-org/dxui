package sdl3

import (
	"testing"

	"github.com/dxui-org/go-sdl3/sdl"
)

func TestMouseWheelTranslationPreservesPlatformDirection(t *testing.T) {
	tests := []struct {
		name      string
		direction sdl.MouseWheelDirection
	}{
		{name: "traditional", direction: sdl.MOUSEWHEEL_NORMAL},
		{name: "natural", direction: sdl.MOUSEWHEEL_FLIPPED},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := translateMouseWheelEvent(sdl.MouseWheelEvent{
				WindowID:  17,
				X:         1.25,
				Y:         -2.5,
				Direction: test.direction,
				MouseX:    30,
				MouseY:    40,
			})
			if got.kind != nativeEventMouseWheel || got.windowID != 17 ||
				got.x != 30 || got.y != 40 || got.wheelX != 1.25 || got.wheelY != -2.5 {
				t.Fatalf("translated wheel = %+v", got)
			}
		})
	}
}
