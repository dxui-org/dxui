package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/backend"
	"github.com/dxui-org/dxui/internal/platform"
)

type cursorRuntime struct {
	*editorRuntime
	cursor backend.Cursor
}

func (r *cursorRuntime) SetCursor(cursor backend.Cursor) error {
	r.cursor = cursor
	return nil
}

func TestEditorHoverCursor(t *testing.T) {
	style := Style{Width: Px(100), Height: Px(40)}
	for _, test := range []struct {
		name string
		view View
		want backend.Cursor
	}{
		{"input", Input(InputProps{Style: style}), backend.CursorText},
		{"readonly", Input(InputProps{Style: style, ReadOnly: true}), backend.CursorText},
		{"password", Input(InputProps{Style: style, Password: true}), backend.CursorText},
		{"textarea", Textarea(TextareaProps{Style: style}), backend.CursorText},
		{"disabled", Input(InputProps{Style: style, Disabled: true}), backend.CursorDefault},
		{"pointer none", Input(InputProps{Style: style, Pointer: PointerNone}), backend.CursorDefault},
		{"hidden", Input(InputProps{Style: Style{Width: Px(100), Height: Px(40), Visibility: Hidden}}), backend.CursorDefault},
		{"button", Button(ButtonProps{Style: style}, Text(TextProps{Value: "Button"})), backend.CursorDefault},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, native := testEditorApp(t, func() View { return test.view })
			runtime := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
			app.backend = runtime
			if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 10, 10)); err != nil {
				t.Fatal(err)
			}
			if runtime.cursor != test.want {
				t.Fatalf("cursor = %v, want %v", runtime.cursor, test.want)
			}
			if app.interaction.Focused() != 0 || native.started != 0 {
				t.Fatal("hover focused editor or started native text input")
			}
			if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 400, 200)); err != nil {
				t.Fatal(err)
			}
			if runtime.cursor != backend.CursorDefault {
				t.Fatal("leaving editor did not restore default cursor")
			}
			app.handleEvent(appPointer(platform.EventMouseMove, 10, 10))
			if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventWindowMouseLeave}); err != nil {
				t.Fatal(err)
			}
			if runtime.cursor != backend.CursorDefault {
				t.Fatal("leaving window did not restore default cursor")
			}
		})
	}
}

func TestHoverCursorReconcilesStationaryPointer(t *testing.T) {
	disabled := false
	app, _ := testEditorApp(t, func() View {
		return Input(InputProps{Key: "editor", Disabled: disabled, Style: Style{Width: Px(100), Height: Px(40)}})
	})
	runtime := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
	app.backend = runtime
	app.handleEvent(appPointer(platform.EventMouseMove, 10, 10))
	disabled = true
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if runtime.cursor != backend.CursorDefault {
		t.Fatal("disabled editor retained text cursor")
	}
	disabled = false
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if runtime.cursor != backend.CursorText {
		t.Fatal("enabled editor did not restore text cursor")
	}
}

func TestHoverCursorUsesClippedTopmostTarget(t *testing.T) {
	app, _ := testEditorApp(t, func() View {
		return Box(BoxProps{},
			Box(BoxProps{Style: Style{Width: Px(20), Height: Px(20), Position: PositionAbsolute, Overflow: OverflowClip}},
				Input(InputProps{Style: Style{Width: Px(100), Height: Px(40)}})),
			Button(ButtonProps{Style: Style{Width: Px(10), Height: Px(10), Position: PositionAbsolute, ZIndex: 1}}, Text(TextProps{Value: "x"})),
		)
	})
	runtime := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
	app.backend = runtime
	for _, step := range []struct {
		x, y float32
		want backend.Cursor
	}{
		{15, 15, backend.CursorText},
		{25, 15, backend.CursorDefault},
		{5, 5, backend.CursorDefault},
	} {
		if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, step.x, step.y)); err != nil {
			t.Fatal(err)
		}
		if runtime.cursor != step.want {
			t.Fatalf("cursor at (%v,%v) = %v, want %v", step.x, step.y, runtime.cursor, step.want)
		}
	}
}
