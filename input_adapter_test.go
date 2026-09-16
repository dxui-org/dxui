package dxui

import (
	"context"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	uiruntime "github.com/dxui-org/dxui/internal/runtime"
)

func appPointer(kind platform.EventKind, x, y float32) platform.Event {
	return platform.Event{Kind: kind, Pointer: platform.PointerEvent{X: x, Y: y, Button: platform.MouseButtonPrimary}}
}

func appKey(kind platform.EventKind, value platform.Key, shift bool) platform.Event {
	return platform.Event{Kind: kind, Key: platform.KeyEvent{Key: value, Modifiers: platform.Modifiers{Shift: shift}}}
}

func fixedButton(key string, z int, disabled bool, callback func()) View {
	return Button(ButtonProps{
		Key: key, Style: Style{Width: Px(40), Height: Px(30), Position: PositionAbsolute, ZIndex: z},
		Disabled: disabled, OnPress: callback,
	}, Text(TextProps{Value: string(key)}))
}

func TestAppHitTestMatchesPaintZOrderAndDisabledFallback(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 50})
	bottom, top := 0, 0
	app.root = func() View {
		return Box(BoxProps{},
			fixedButton("bottom", 1, false, func() { bottom++ }),
			fixedButton("top", 2, true, func() { top++ }),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseDown, 5, 5)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 5, 5)); err != nil {
		t.Fatal(err)
	}
	if bottom != 1 || top != 0 {
		t.Fatalf("bottom/top callbacks = %d/%d, want 1/0", bottom, top)
	}
}

func TestAppHitTestSkipsHiddenTransparentAndPointerNoneSubtrees(t *testing.T) {
	for _, test := range []struct {
		name string
		node viewProps
	}{
		{name: "hidden", node: viewProps{Style: Style{Visibility: Hidden}}},
		{name: "transparent", node: viewProps{Style: Style{Opacity: Some(float32(0))}}},
		{name: "pointer-none", node: viewProps{Pointer: PointerNone}},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 50, Height: 40})
			bottom, top := 0, 0
			app.root = func() View {
				bottomNode := viewProps{Style: Style{Width: Px(40), Height: Px(30), Position: PositionAbsolute, ZIndex: 1}}
				topNode := test.node
				topNode.Style.Width = Px(40)
				topNode.Style.Height = Px(30)
				topNode.Style.Position = PositionAbsolute
				topNode.Style.ZIndex = 2
				return Box(BoxProps{},
					Button(ButtonProps{Style: bottomNode.Style, Pointer: bottomNode.Pointer, OnPress: func() { bottom++ }}, Label("bottom")),
					Button(ButtonProps{Style: topNode.Style, Pointer: topNode.Pointer, OnPress: func() { top++ }}, Label("top")),
				)
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
			if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 5, 5)); err != nil {
				t.Fatal(err)
			}
			if bottom != 1 || top != 0 {
				t.Fatalf("bottom/top = %d/%d", bottom, top)
			}
		})
	}
}

func TestAppHitTestUsesNestedAncestorClip(t *testing.T) {
	app := NewApp(AppOptions{Width: 80, Height: 50})
	presses := 0
	app.root = func() View {
		return Box(BoxProps{}, Box(BoxProps{Style: Style{
			Width: Px(20), Height: Px(20), Position: PositionAbsolute, Overflow: OverflowClip,
		}}, fixedButton("clipped", 0, false, func() { presses++ })))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 10, 10))
	app.handleEvent(appPointer(platform.EventMouseUp, 10, 10))
	app.handleEvent(appPointer(platform.EventMouseDown, 25, 10))
	app.handleEvent(appPointer(platform.EventMouseUp, 25, 10))
	if presses != 1 {
		t.Fatalf("nested clip presses = %d, want one inside activation", presses)
	}
}

func TestAppMoveWithinSameTargetDoesNotRequestAnotherPaint(t *testing.T) {
	app := NewApp(AppOptions{Width: 50, Height: 40})
	app.root = func() View { return fixedButton("same", 0, false, nil) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if dirty, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 2, 2)); err != nil || !dirty {
		t.Fatalf("first move = dirty %v, error %v", dirty, err)
	}
	paints := app.Diagnostics().PaintCount
	if dirty, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 3, 3)); err != nil || dirty {
		t.Fatalf("same-target move = dirty %v, error %v", dirty, err)
	}
	if app.Diagnostics().PaintCount != paints {
		t.Fatal("same-target move regenerated display")
	}
}

func TestFakeEventSourceSameTargetMoveProducesNoExtraFrame(t *testing.T) {
	app := NewApp(AppOptions{Width: 50, Height: 40})
	app.root = func() View { return fixedButton("same", 0, false, nil) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	events := &frameTestEvents{events: []platform.Event{
		appPointer(platform.EventMouseMove, 2, 2),
		appPointer(platform.EventMouseMove, 3, 3),
		{Kind: platform.EventClose},
	}}
	renderer := &frameTestRenderer{}
	app.running, app.events = true, events
	loop := uiruntime.Loop{
		Clock: platform.SystemClock{}, Events: events, Renderer: renderer,
		Display: func() paint.DisplayList { return app.currentDisplay() }, Handle: app.handleEvent,
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if renderer.presents != 2 {
		t.Fatalf("presents = %d, want initial plus first hover only", renderer.presents)
	}
}

func TestAppCallbackMayDeleteCurrentCapturedNode(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 50})
	visible := true
	presses := 0
	app.root = func() View {
		if !visible {
			return Box(BoxProps{})
		}
		return Box(BoxProps{}, fixedButton("remove", 0, false, func() {
			presses++
			visible = false
		}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 5, 5)); err != nil {
		t.Fatal(err)
	}
	if presses != 1 || len(app.retained.Root().Children) != 0 {
		t.Fatalf("presses/children = %d/%d", presses, len(app.retained.Root().Children))
	}
	if app.interaction.Focused() != 0 {
		t.Fatal("removed callback target retained focus")
	}
	// A late release cannot use the removed instance or invoke its callback.
	app.handleEvent(appPointer(platform.EventMouseUp, 5, 5))
	if presses != 1 {
		t.Fatalf("late release invoked stale callback: %d", presses)
	}
}

func TestAppTabAndKeyboardActivationArePaintOnlyUntilCallback(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 80})
	presses := 0
	app.root = func() View {
		return Box(BoxProps{},
			Button(ButtonProps{Disabled: true}, Text(TextProps{Value: "skip"})),
			Button(ButtonProps{OnPress: func() { presses++ }}, Text(TextProps{Value: "go"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	beforeLayout := app.Diagnostics().LayoutCount
	if dirty, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false)); err != nil || !dirty {
		t.Fatalf("Tab result dirty/error = %v/%v", dirty, err)
	}
	if app.Diagnostics().LayoutCount != beforeLayout {
		t.Fatal("focus state caused layout")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	if _, _, err := app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false)); err != nil {
		t.Fatal(err)
	}
	if presses != 1 {
		t.Fatalf("keyboard presses = %d", presses)
	}
}

func TestButtonFocusIndicatorFollowsInputModalityAcrossThemes(t *testing.T) {
	for _, test := range []struct {
		name  string
		theme Theme
	}{
		{name: "light", theme: LightTheme()},
		{name: "dark", theme: DarkTheme()},
	} {
		t.Run(test.name, func(t *testing.T) {
			presses := 0
			button := Button(ButtonProps{Key: "button", Style: Style{
				Width: Px(40), Height: Px(30), Shrink: Some(float32(0)),
			}, OnPress: func() { presses++ }}, Text(TextProps{Value: "button"}))
			app := NewApp(AppOptions{Width: 100, Height: 50, Theme: test.theme})
			app.root = func() View { return Box(BoxProps{}, button) }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			instance := app.retained.Root().Children[0]
			props, err := nodeProps(button)
			if err != nil {
				t.Fatal(err)
			}
			assertBorder := func(width float32, color RGBAColor) {
				t.Helper()
				visual, visualErr := computeVisual(props, instance, viewButton, app.theme)
				if visualErr != nil {
					t.Fatal(visualErr)
				}
				if width == 0 {
					if visual.border.width != 0 || visual.border.colorSet {
						t.Fatalf("border = %+v, want none", visual.border)
					}
					return
				}
				if visual.border.width != width || !visual.border.colorSet || visual.border.color != color {
					t.Fatalf("border = %+v, want width %v color %+v", visual.border, width, color)
				}
			}
			assertOpacity := func(want float32) {
				t.Helper()
				visual, visualErr := computeVisual(props, instance, viewButton, app.theme)
				if visualErr != nil || visual.opacity != want {
					t.Fatalf("opacity = %v/%v, want %v", visual.opacity, visualErr, want)
				}
			}
			assertFocusRing := func() {
				t.Helper()
				visual, visualErr := computeVisual(props, instance, viewButton, app.theme)
				if visualErr != nil || len(visual.shadows) != 1 || visual.shadows[0].Spread != 2 {
					t.Fatalf("focus ring = %+v/%v", visual.shadows, visualErr)
				}
			}
			app.handleEvent(appPointer(platform.EventMouseMove, 5, 5))
			app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
			if !instance.State.Focused || instance.State.FocusVisible || !instance.State.Pressed {
				t.Fatalf("mouse down state = %+v", instance.State)
			}
			assertOpacity(.78)
			app.handleEvent(appPointer(platform.EventMouseUp, 5, 5))
			instance = app.retained.Root().Children[0]
			if !instance.State.Focused || instance.State.FocusVisible || instance.State.Pressed || presses != 1 {
				t.Fatalf("mouse release state/presses = %+v/%d", instance.State, presses)
			}
			assertBorder(0, RGBAColor{})
			assertOpacity(.92)

			if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventWindowMouseLeave}); err != nil {
				t.Fatal(err)
			}
			if instance.State.Hovered || !instance.State.Focused || instance.State.FocusVisible {
				t.Fatalf("mouse exit state = %+v", instance.State)
			}
			assertOpacity(1)
			app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
			app.handleEvent(appPointer(platform.EventMouseUp, 5, 5))
			instance = app.retained.Root().Children[0]
			if presses != 2 {
				t.Fatalf("repeat mouse presses = %d, want 2", presses)
			}
			assertBorder(0, RGBAColor{})

			app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
			if !instance.State.Pressed || !instance.State.FocusVisible {
				t.Fatalf("pointer-to-Space state = %+v", instance.State)
			}
			assertOpacity(.78)
			app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false))
			instance = app.retained.Root().Children[0]
			if presses != 3 || instance.State.Pressed || !instance.State.FocusVisible {
				t.Fatalf("Space release state/presses = %+v/%d", instance.State, presses)
			}

			app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
			app.handleEvent(appPointer(platform.EventMouseUp, 5, 5))
			instance = app.retained.Root().Children[0]
			if presses != 4 || instance.State.FocusVisible {
				t.Fatalf("keyboard-to-pointer state/presses = %+v/%d", instance.State, presses)
			}
			app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
			if !instance.State.Focused || !instance.State.FocusVisible {
				t.Fatalf("Tab state = %+v", instance.State)
			}
			assertBorder(0, RGBAColor{})
			assertFocusRing()

			app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false))
			app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false))
			instance = app.retained.Root().Children[0]
			if presses != 5 || instance.State.Pressed || !instance.State.FocusVisible {
				t.Fatalf("keyboard state/presses = %+v/%d", instance.State, presses)
			}

			app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
			if !instance.State.Focused || instance.State.FocusVisible {
				t.Fatalf("keyboard-to-mouse state = %+v", instance.State)
			}
			assertBorder(0, RGBAColor{})
		})
	}
}

func TestDisabledButtonAndExplicitBorderIgnoreFocusModality(t *testing.T) {
	explicit := Border{Width: Metric(3), Color: LiteralColor(RGBA(12, 180, 90, 255))}
	button := Button(ButtonProps{Style: Style{
		Width: Px(40), Height: Px(30), Position: PositionAbsolute, Border: explicit,
	}}, Text(TextProps{Value: "button"}))
	app := NewApp(AppOptions{Width: 100, Height: 50, Theme: DarkTheme()})
	app.root = func() View { return Box(BoxProps{}, button) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	instance := app.retained.Root().Children[0]
	props, _ := nodeProps(button)
	for _, event := range []platform.Event{
		appPointer(platform.EventMouseDown, 5, 5), appPointer(platform.EventMouseUp, 5, 5),
		appKey(platform.EventKeyDown, platform.KeyTab, false),
	} {
		app.handleEvent(event)
		visual, err := computeVisual(props, instance, viewButton, app.theme)
		if err != nil || visual.border.width != 3 || visual.border.color != RGBA(12, 180, 90, 255) {
			t.Fatalf("explicit border after %v = %+v/%v", event.Kind, visual.border, err)
		}
	}

	disabled := fixedButton("disabled", 0, true, func() { t.Fatal("disabled button activated") })
	app = NewApp(AppOptions{Width: 100, Height: 50, Theme: LightTheme()})
	app.root = func() View { return Box(BoxProps{}, disabled) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	instance = app.retained.Root().Children[0]
	app.handleEvent(appPointer(platform.EventMouseMove, 5, 5))
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	app.handleEvent(appPointer(platform.EventMouseUp, 5, 5))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if instance.State.Focused || instance.State.FocusVisible || instance.State.Hovered || instance.State.Pressed {
		t.Fatalf("disabled interaction state = %+v", instance.State)
	}
	props, _ = nodeProps(disabled)
	visual, err := computeVisual(props, instance, viewButton, app.theme)
	if err != nil || visual.opacity != .42 {
		t.Fatalf("disabled visual opacity = %v/%v, want .42", visual.opacity, err)
	}
}

func TestAppInteractionCallbackPanicUsesOnErrorPolicy(t *testing.T) {
	var reported error
	app := NewApp(AppOptions{Width: 50, Height: 40, OnError: func(err error) { reported = err }})
	app.root = func() View {
		return fixedButton("panic", 0, false, func() { panic("boom") })
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 5, 5)); err != nil {
		t.Fatalf("handled callback panic escaped: %v", err)
	}
	if reported == nil {
		t.Fatal("callback panic was not reported")
	}
}
