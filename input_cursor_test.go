package dxui

import (
	"fmt"
	"github.com/dxui-org/dxui/internal/tree"
	"strings"
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
		{"checkbox", Checkbox(CheckboxProps{Style: style}, Label("Check")), backend.CursorDefault},
		{"toggle", ToggleSwitch(ToggleSwitchProps{Style: style}), backend.CursorDefault},
		{"radio", Radio(RadioProps{Style: style}, Label("Radio")), backend.CursorDefault},
		{"select", Select(SelectProps{Style: style}), backend.CursorDefault},
		{"slider", Slider(SliderProps{Style: style, Max: 100}), backend.CursorDefault},
		{"text", Label("Text"), backend.CursorDefault},
		{"tabs", Tabs(TabsProps{}), backend.CursorDefault},
		{"menu", Menu(MenuProps{}), backend.CursorDefault},
		{"scroll", Scroll(ScrollProps{}, Label("Content")), backend.CursorDefault},
		{"popover", Popover(PopoverProps{}, Label("Anchor"), Label("Content")), backend.CursorDefault},
		{"box", Box(BoxProps{Style: style}), backend.CursorDefault},
		{"disabled button", Button(ButtonProps{Style: style, Disabled: true}, Label("Disabled")), backend.CursorDefault},
		{"disabled", Input(InputProps{Style: style, Disabled: true}), backend.CursorDefault},
		{"pointer none", Input(InputProps{Style: style, Pointer: PointerNone}), backend.CursorDefault},
		{"hidden", Input(InputProps{Style: Style{Width: Px(100), Height: Px(40), Visibility: Hidden}}), backend.CursorDefault},
		{"button", Button(ButtonProps{Style: style}, Text(TextProps{Value: "Button"})), backend.CursorPointer},
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
		{5, 5, backend.CursorPointer},
	} {
		if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, step.x, step.y)); err != nil {
			t.Fatal(err)
		}
		if runtime.cursor != step.want {
			t.Fatalf("cursor at (%v,%v) = %v, want %v", step.x, step.y, runtime.cursor, step.want)
		}
	}
}

func TestStyleCursorOverrides(t *testing.T) {
	for _, component := range []struct {
		name string
		view View
		auto backend.Cursor
	}{
		{"input", Input(InputProps{}), backend.CursorText},
		{"button", Button(ButtonProps{}, Label("Button")), backend.CursorPointer},
		{"text", Label("Text"), backend.CursorDefault},
		{"box", Box(BoxProps{}, Label("Child")), backend.CursorDefault},
	} {
		for _, test := range []struct {
			cursor Cursor
			want   backend.Cursor
		}{
			{CursorAuto, component.auto}, {CursorDefault, backend.CursorDefault},
			{CursorText, backend.CursorText}, {CursorPointer, backend.CursorPointer},
		} {
			t.Run(fmt.Sprintf("%s/%d", component.name, test.cursor), func(t *testing.T) {
				app, _ := testEditorApp(t, func() View { return component.view.WithStyle(Style{Cursor: test.cursor}) })
				native := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
				app.backend = native
				if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 1, 1)); err != nil {
					t.Fatal(err)
				}
				if native.cursor != test.want {
					t.Fatalf("cursor=%v want %v", native.cursor, test.want)
				}
			})
		}
	}
}

func TestCursorOnlyTargetDoesNotCaptureFocusOrRepaint(t *testing.T) {
	app, _ := testEditorApp(t, func() View { return Label("Hover").WithStyle(Style{Cursor: CursorPointer}) })
	native := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
	app.backend = native
	for _, kind := range []platform.EventKind{platform.EventMouseMove, platform.EventMouseDown, platform.EventMouseUp} {
		dirty, _, err := app.handleEvent(appPointer(kind, 1, 1))
		if err != nil {
			t.Fatal(err)
		}
		if dirty || app.interaction.Focused() != 0 {
			t.Fatal("cursor-only target changed interaction state or requested paint")
		}
	}
	if native.cursor != backend.CursorPointer {
		t.Fatal("missing cursor-only hit")
	}
}

func TestStyleCursorChangeUpdatesStationaryPointerWithoutPaint(t *testing.T) {
	cursor := CursorAuto
	app, _ := testEditorApp(t, func() View { return Button(ButtonProps{Key: "button", Style: Style{Cursor: cursor}}, Label("Button")) })
	native := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
	app.backend = native
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 1, 1)); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	cursor = CursorText
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	after := app.Diagnostics()
	if native.cursor != backend.CursorText {
		t.Fatal("stationary pointer not updated")
	}
	if !changes.Has(tree.DirtySemantics) || changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) || before.PaintCount != after.PaintCount || before.LayoutCount != after.LayoutCount {
		t.Fatalf("cursor-only change performed layout/paint: %+v", changes)
	}
	cursor = Cursor(255)
	oldRoot, oldGeometry := app.retained.Root(), app.geometry
	if err := app.buildRoot(); err == nil || !strings.Contains(err.Error(), "invalid cursor") {
		t.Fatalf("invalid cursor error=%v", err)
	}
	if app.retained.Root() != oldRoot || app.geometry != oldGeometry || native.cursor != backend.CursorText {
		t.Fatal("invalid cursor changed accepted state")
	}
}

func TestExplicitChildCursorPreservesButtonActivation(t *testing.T) {
	presses := 0
	app, _ := testEditorApp(t, func() View {
		return Button(ButtonProps{OnPress: func() { presses++ }}, Label("Child").WithStyle(Style{Cursor: CursorDefault}))
	})
	native := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
	app.backend = native
	// Use the child's actual box, which may be centered by the button recipe.
	child := app.geometry.Children[0].Rect
	x, y := child.X+child.Width/2, child.Y+child.Height/2
	for _, kind := range []platform.EventKind{platform.EventMouseMove, platform.EventMouseDown, platform.EventMouseUp} {
		if _, _, err := app.handleEvent(appPointer(kind, x, y)); err != nil {
			t.Fatal(err)
		}
		if native.cursor != backend.CursorDefault {
			t.Fatal("explicit child cursor lost to button default")
		}
	}
	if presses != 1 {
		t.Fatalf("presses=%d, want 1", presses)
	}
}

func TestExplicitCursorRespectsSubtreeExclusions(t *testing.T) {
	for _, test := range []struct {
		name string
		view View
	}{
		{"pointer none", Box(BoxProps{Pointer: PointerNone}, Label("Child").WithStyle(Style{Cursor: CursorPointer}))},
		{"transparent", Box(BoxProps{Style: Style{Opacity: Some(float32(0))}}, Label("Child").WithStyle(Style{Cursor: CursorPointer}))},
		{"hidden", Box(BoxProps{Style: Style{Visibility: Hidden}}, Label("Child").WithStyle(Style{Cursor: CursorPointer}))},
		{"disabled", Button(ButtonProps{Disabled: true}, Label("Child").WithStyle(Style{Cursor: CursorPointer}))},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, _ := testEditorApp(t, func() View { return test.view })
			native := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
			app.backend = native
			rect := app.geometry.Children[0].Rect
			if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, rect.X+rect.Width/2, rect.Y+rect.Height/2)); err != nil {
				t.Fatal(err)
			}
			if native.cursor != backend.CursorDefault {
				t.Fatal("excluded subtree selected explicit cursor")
			}
		})
	}
}

func TestSelectOverlayCursorOverridesUnderlyingEditor(t *testing.T) {
	cursor := CursorAuto
	app, _ := testEditorApp(t, func() View {
		return Box(BoxProps{},
			Select(SelectProps{Key: "select", Options: selectOptions(), Style: Style{Width: Px(80), Height: Px(32), Position: PositionAbsolute, Cursor: cursor}}),
			Input(InputProps{Style: Style{Width: Px(300), Height: Px(120), Position: PositionAbsolute, ZIndex: -1}}),
		)
	})
	native := &cursorRuntime{editorRuntime: app.backend.(*editorRuntime)}
	app.backend = native
	id := openFocusedSelect(t, app)
	for _, test := range []struct {
		cursor Cursor
		want   backend.Cursor
	}{
		{CursorAuto, backend.CursorDefault}, {CursorText, backend.CursorText}, {CursorPointer, backend.CursorPointer},
	} {
		cursor = test.cursor
		if err := app.buildRoot(); err != nil {
			t.Fatal(err)
		}
		popup := app.inputActions[id].selectOverlay.Popup
		if popup.Width <= 0 || popup.Height <= 0 {
			t.Fatal("missing Select popup")
		}
		if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, popup.X+1, popup.Y+1)); err != nil {
			t.Fatal(err)
		}
		if native.cursor != test.want {
			t.Fatalf("popup cursor=%v want %v", native.cursor, test.want)
		}
	}
}
