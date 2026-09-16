package dxui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/platform"
	uiruntime "github.com/dxui-org/dxui/internal/runtime"
)

type viewportStormEvents struct {
	waits  []platform.Event
	queued []platform.Event
	index  int
}

func (events *viewportStormEvents) Wait(context.Context, *time.Time) (platform.Event, error) {
	if events.index >= len(events.waits) {
		return platform.Event{}, errors.New("viewport storm ran out of events")
	}
	event := events.waits[events.index]
	events.index++
	return event, nil
}

func (events *viewportStormEvents) Poll() (platform.Event, bool, error) {
	if len(events.queued) == 0 {
		return platform.Event{}, false, nil
	}
	event := events.queued[0]
	events.queued = events.queued[1:]
	return event, true, nil
}

func (*viewportStormEvents) Wake() error { return nil }

func TestLoginFlowWithFakeBackend(t *testing.T) {
	type state struct {
		username, password, role string
		remember, dark           bool
	}
	model := state{role: "developer"}
	var loginResult string
	var selectedAnchor, selectedPopup Rect

	app := NewApp(AppOptions{Width: 900, Height: 680, Theme: LightTheme()})
	field := func() Style {
		return Style{Width: Px(420), Height: Px(44)}
	}
	app.root = func() View {
		return Box(BoxProps{Style: Style{Padding: UniformEdges(Metric(24))}, Gap: 8},
			Input(InputProps{Key: "username", Style: field(), Value: model.username,
				OnChange: func(value string) { model.username = value }}),
			Input(InputProps{Key: "password", Style: field(), Value: model.password, Password: true,
				OnChange: func(value string) { model.password = value }}),
			Select(SelectProps{Key: "role", Style: field(), Value: model.role, Options: []SelectOption{
				{Value: "developer", Label: "Developer"},
				{Value: "designer", Label: "Designer"},
			}, OnChange: func(value string) {
				model.role = value
				for _, action := range app.inputActions {
					if action.selectp != nil {
						selectedAnchor = Rect(action.selectAnchor)
						selectedPopup = Rect(action.selectOverlay.Popup)
					}
				}
			}}),
			Checkbox(CheckboxProps{Key: "remember", Checked: model.remember,
				OnChange: func(value bool) { model.remember = value }}, Text(TextProps{Value: "Remember me"})),
			ToggleSwitch(ToggleSwitchProps{Key: "theme", Checked: model.dark,
				OnChange: func(value bool) {
					model.dark = value
					if err := app.SetTheme(DarkTheme()); err != nil {
						t.Errorf("SetTheme: %v", err)
					}
				}}),
			Button(ButtonProps{Key: "cancel", OnPress: app.Close}, Text(TextProps{Value: "Cancel"})),
			Button(ButtonProps{Key: "login", Disabled: model.username == "" || model.password == "", OnPress: func() {
				loginResult = fmt.Sprintf("username=%s role=%s remember=%t", model.username, model.role, model.remember)
			}}, Text(TextProps{Value: "Login"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}

	events := &frameTestEvents{events: []platform.Event{
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		{Kind: platform.EventTextInput, Text: platform.TextEvent{Text: "alice"}},
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: "candidate", EditingStart: 0, EditingLength: 9}},
		{Kind: platform.EventTextInput, Text: platform.TextEvent{Text: "s3cret"}},
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		appKey(platform.EventKeyDown, platform.KeyEnter, false),
		appKey(platform.EventKeyUp, platform.KeyEnter, false),
		appKey(platform.EventKeyDown, platform.KeyDown, false),
		appKey(platform.EventKeyDown, platform.KeyEnter, false),
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		appKey(platform.EventKeyDown, platform.KeySpace, false),
		appKey(platform.EventKeyUp, platform.KeySpace, false),
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		appKey(platform.EventKeyDown, platform.KeySpace, false),
		appKey(platform.EventKeyUp, platform.KeySpace, false),
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		appKey(platform.EventKeyDown, platform.KeyEnter, false),
		appKey(platform.EventKeyUp, platform.KeyEnter, false),
		appKey(platform.EventKeyDown, platform.KeyTab, true),
		appKey(platform.EventKeyDown, platform.KeyEnter, false),
		appKey(platform.EventKeyUp, platform.KeyEnter, false),
	}}
	renderer := &frameTestRenderer{}
	nativeText := &editorNative{}
	app.running, app.events = true, events
	app.backend = &editorRuntime{
		events: events, render: renderer, text: nativeText,
		viewport: platform.Viewport{LogicalWidth: 900, LogicalHeight: 680, PixelWidth: 1800, PixelHeight: 1360, PixelDensity: 2, DisplayScale: 2},
	}
	loop := uiruntime.Loop{
		Clock: platform.SystemClock{}, Events: events, Renderer: renderer,
		Display: app.currentDisplay, Deadline: app.nextDeadline, Handle: app.handleEvent,
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if model.username != "alice" || model.password != "s3cret" || model.role != "designer" || !model.remember || !model.dark {
		t.Fatalf("controlled model = %+v", model)
	}
	if loginResult != "username=alice role=designer remember=true" || strings.Contains(loginResult, model.password) {
		t.Fatalf("non-sensitive login result = %q", loginResult)
	}
	if !app.closed || nativeText.started == 0 || nativeText.area.Height <= 0 {
		t.Fatalf("lifecycle/IME = closed:%v starts:%d area:%+v", app.closed, nativeText.started, nativeText.area)
	}
	if nativeText.area.X < 0 || nativeText.area.Y < 0 || nativeText.area.X > 900 || nativeText.area.Y > 680 || nativeText.area.Width > 500 {
		t.Fatalf("HiDPI caret area was not kept in logical window coordinates: %+v", nativeText.area)
	}
	if selectedPopup.Width != selectedAnchor.Width || selectedPopup.X != selectedAnchor.X || selectedPopup.X+selectedPopup.Width > 900 || selectedPopup.Y+selectedPopup.Height > 680 {
		t.Fatalf("HiDPI Select anchor/popup misaligned: anchor=%+v popup=%+v", selectedAnchor, selectedPopup)
	}
	for _, command := range app.currentDisplay() {
		if command.Text != nil && strings.Contains(command.Text.Key, model.password) {
			t.Fatal("password leaked into a display texture key")
		}
	}
	d := app.Diagnostics()
	if d.BuildCount > 8 || d.LayoutCount > 4 || d.PaintCount > uint64(len(events.events)+8) || renderer.presents > len(events.events)+1 || renderer.renders != renderer.presents {
		t.Fatalf("pipeline grew excessively: diagnostics=%+v render/present=%d/%d", d, renderer.renders, renderer.presents)
	}
}

func TestResizeMaximizeAndScaleStormUsesFinalViewportOnce(t *testing.T) {
	app := NewApp(AppOptions{Width: 900, Height: 680})
	app.root = func() View {
		return Box(BoxProps{Style: Style{
			Width: Percent(100), Height: Percent(100),
		}}, Text(TextProps{Value: "login"}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}

	latest := platform.Viewport{LogicalWidth: 1280, LogicalHeight: 720, PixelWidth: 2560, PixelHeight: 1440, PixelDensity: 2, DisplayScale: 2}
	events := &viewportStormEvents{waits: []platform.Event{
		{Kind: platform.EventResize, Viewport: platform.Viewport{LogicalWidth: 910, LogicalHeight: 680}},
		{Kind: platform.EventClose},
	}}
	for i := 0; i < 50; i++ {
		events.queued = append(events.queued, platform.Event{Kind: platform.EventResize, Viewport: platform.Viewport{
			LogicalWidth: int32(920 + i), LogicalHeight: int32(680 + i),
		}})
	}
	events.queued = append(events.queued, platform.Event{Kind: platform.EventScale, Viewport: latest})
	renderer := &frameTestRenderer{}
	app.running, app.events = true, events
	loop := uiruntime.Loop{Clock: platform.SystemClock{}, Events: events, Renderer: renderer, Display: app.currentDisplay, Handle: app.handleEvent}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if app.geometry.Rect.Width != 1280 || app.geometry.Rect.Height != 720 {
		t.Fatalf("final layout = %+v, want 1280x720", app.geometry.Rect)
	}
	if d := app.Diagnostics(); d.LayoutCount != 2 || d.PaintCount != 2 {
		t.Fatalf("storm pipeline counts = %+v, want initial plus one final viewport commit", d)
	}
	if renderer.presents != 2 {
		t.Fatalf("storm presents = %d, want initial plus final viewport", renderer.presents)
	}
}

func TestConstraintAwareStructureSwitchKeepsControlledValueAndUsableFocus(t *testing.T) {
	value := "kept"
	app := NewApp(AppOptions{Width: 900, Height: 600})
	runtime := &editorRuntime{render: &frameTestRenderer{}, text: &editorNative{}, viewport: platform.Viewport{LogicalWidth: 900, LogicalHeight: 600}}
	app.backend = runtime
	app.constraintAware = true
	app.buildSize = Size{Width: 900, Height: 600}
	app.root = func() View {
		input := Input(InputProps{Key: "value", Value: value, OnChange: func(next string) { value = next }})
		if app.buildSize.Width < 700 {
			return Box(BoxProps{Style: Style{Width: Fill(), Height: Fill()}}, Label("top navigation"), input)
		}
		return Box(BoxProps{Direction: Horizontal, Style: Style{Width: Fill(), Height: Fill()}}, Label("side navigation"), Box(BoxProps{}, input))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, err := app.handleInteraction(appKey(platform.EventKeyDown, platform.KeyTab, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.handleInteraction(platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: "preedit", EditingStart: 0, EditingLength: 7}}); err != nil {
		t.Fatal(err)
	}

	runtime.viewport = platform.Viewport{LogicalWidth: 520, LogicalHeight: 600}
	dirty, _, err := app.handleEvent(platform.Event{Kind: platform.EventResize, Viewport: runtime.viewport})
	if err != nil || !dirty {
		t.Fatalf("responsive resize = dirty:%t err:%v", dirty, err)
	}
	if app.view.node.kind != viewBox || app.view.node.box.Direction != Vertical || app.geometry.Rect.Width != 520 {
		t.Fatalf("compact root/layout = kind:%v rect:%+v", app.view.node.kind, app.geometry.Rect)
	}
	focus := app.interaction.Focused()
	action, ok := app.inputActions[focus]
	if !ok || !action.editor || action.value != "kept" || action.geometry.Width <= 0 || action.geometry.Height <= 0 {
		t.Fatalf("focused compact input = id:%d action:%+v", focus, action)
	}
	if editor := app.editors[focus]; editor == nil || editor.state.Composition.Active {
		t.Fatal("structure replacement did not establish a clean editor instance")
	}
	x, y := action.geometry.X+action.geometry.Width/2, action.geometry.Y+action.geometry.Height/2
	if _, err := app.handleInteraction(appPointer(platform.EventMouseDown, x, y)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.handleInteraction(appPointer(platform.EventMouseUp, x, y)); err != nil {
		t.Fatal(err)
	}
	if app.interaction.Focused() != focus {
		t.Fatal("compact input hit region did not retain focus")
	}
	if app.buildCount != 2 || app.layoutCount != 2 {
		t.Fatalf("responsive counts = build:%d layout:%d, want 2/2", app.buildCount, app.layoutCount)
	}
}
