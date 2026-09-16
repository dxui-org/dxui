package dxui

import (
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

func groupedInput(key string, value *string) View {
	return Input(InputProps{
		Key: key, Value: *value, Placeholder: "Search",
		OnChange: func(next string) { *value = next },
	})
}

func TestInputGroupRequiresOneInputAndRestrictsAdornments(t *testing.T) {
	value := ""
	tests := []struct {
		name string
		view View
		want string
	}{
		{name: "missing input", view: InputGroup(InputGroupProps{}, InputGroupContent{}), want: "valid Input"},
		{name: "wrong required view", view: InputGroup(InputGroupProps{}, InputGroupContent{Input: Label("not input")}), want: "required Input"},
		{name: "editor prefix", view: InputGroup(InputGroupProps{}, InputGroupContent{Input: groupedInput("main", &value), Prefix: Some(Input(InputProps{}))}), want: "Input is not passive"},
		{name: "focusable suffix", view: InputGroup(InputGroupProps{}, InputGroupContent{Input: groupedInput("main", &value), Suffix: Some(Checkbox(CheckboxProps{}, Label("bad")))}), want: "not passive content or a Button"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := describeView(test.view)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("describe error = %v, want %q", err, test.want)
			}
		})
	}
	valid := InputGroup(InputGroupProps{}, InputGroupContent{
		Input: groupedInput("main", &value), Prefix: Some(Label("$")),
		Suffix: Some(Button(ButtonProps{}, Label("Apply"))),
	})
	if _, err := describeView(valid); err != nil {
		t.Fatalf("valid InputGroup: %v", err)
	}
}

func TestInputGroupHorizontalLayoutAndSingleSurface(t *testing.T) {
	value := "abc"
	app := NewApp(AppOptions{Width: 320, Height: 80})
	app.root = func() View {
		return InputGroup(InputGroupProps{Style: Style{Width: Px(280)}}, InputGroupContent{
			Input: groupedInput("value", &value), Prefix: Some(Label("$")), Suffix: Some(Label("kg")),
		})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	geometry := app.geometry
	if len(geometry.Children) != 3 {
		t.Fatalf("children = %d, want 3", len(geometry.Children))
	}
	prefix, input, suffix := geometry.Children[0].Rect, geometry.Children[1].Rect, geometry.Children[2].Rect
	if prefix.X >= input.X || input.X >= suffix.X {
		t.Fatalf("horizontal order = prefix %v input %v suffix %v", prefix, input, suffix)
	}
	wantInputWidth := geometry.Content.Width - prefix.Width - suffix.Width - 16
	if input.Width != wantInputWidth {
		t.Fatalf("input width = %v, want remaining %v", input.Width, wantInputWidth)
	}

	groupID := app.retained.Root().ID
	inputID := app.retained.Root().Children[1].ID
	groupSurface, inputSurface := 0, 0
	for _, command := range app.display {
		if command.NodeID == groupID && (command.Kind == paint.CommandFillStrokeRoundedRect || command.Kind == paint.CommandStrokeRoundedRect) {
			groupSurface++
		}
		if command.NodeID == inputID && (command.Kind == paint.CommandFillRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect || command.Kind == paint.CommandStrokeRoundedRect) {
			inputSurface++
		}
	}
	if groupSurface != 1 || inputSurface != 0 {
		t.Fatalf("surface commands group/input = %d/%d, want 1/0", groupSurface, inputSurface)
	}
}

func TestInputGroupFocusEditingAndButtonRemainIndependent(t *testing.T) {
	value := ""
	presses := 0
	app, native := testEditorApp(t, func() View {
		return Box(BoxProps{},
			InputGroup(InputGroupProps{Style: Style{Width: Px(260)}}, InputGroupContent{
				Input: groupedInput("query", &value), Prefix: Some(Label("?")),
				Suffix: Some(Button(ButtonProps{Key: "action", OnPress: func() { presses++ }}, Label("Go"))),
			}),
			Button(ButtonProps{Key: "after"}, Label("After")),
		)
	})
	root := app.retained.Root()
	group, input, action := root.Children[0], root.Children[0].Children[1], root.Children[0].Children[2]

	focusEditor(t, app)
	if app.interaction.Focused() != input.ID || !group.State.FocusVisible || native.started != 1 {
		t.Fatalf("input/group focus = %d/%+v native=%d", app.interaction.Focused(), group.State, native.started)
	}
	focusRing := false
	for _, command := range app.display {
		if command.NodeID == group.ID && command.Kind == paint.CommandDrawShadow && command.Shadow.Spread == 2 {
			focusRing = true
		}
	}
	if !focusRing {
		t.Fatal("InputGroup did not paint the outer keyboard focus ring")
	}
	if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: "候选", EditingStart: 0, EditingLength: 1}}); err != nil {
		t.Fatal(err)
	}
	editor := app.editors[input.ID]
	if editor == nil || editor.state.Composition.Text != "候选" {
		t.Fatalf("grouped composition = %+v", editor)
	}
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.editors[input.ID] == nil || app.editors[input.ID].state.Composition.Text != "候选" {
		t.Fatal("keyed grouped Input lost composition across rebuild")
	}
	if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventTextInput, Text: platform.TextEvent{Text: "中"}}); err != nil {
		t.Fatal(err)
	}
	if value != "中" {
		t.Fatalf("controlled edit = %q", value)
	}
	if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false)); err != nil {
		t.Fatal(err)
	}
	if app.interaction.Focused() != action.ID || !group.State.FocusVisible {
		t.Fatalf("button/group focus = %d/%+v", app.interaction.Focused(), group.State)
	}
	if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false)); err != nil {
		t.Fatal(err)
	}
	if presses != 1 {
		t.Fatalf("button presses = %d, want 1", presses)
	}
	actionRect := app.geometry.Children[0].Children[2].Rect
	x, y := actionRect.X+actionRect.Width/2, actionRect.Y+actionRect.Height/2
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseDown, x, y)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, x, y)); err != nil {
		t.Fatal(err)
	}
	if presses != 2 {
		t.Fatalf("button pointer presses = %d, want 2", presses)
	}
}

func TestInputGroupPreservesInputPasswordAndDisabledSemantics(t *testing.T) {
	view := InputGroup(InputGroupProps{}, InputGroupContent{Input: Input(InputProps{
		Key: "secret", Value: "秘密", Password: true, ShowPasswordToggle: true,
		Disabled: true, OnChange: func(string) {},
	})})
	description, err := describeView(view)
	if err != nil {
		t.Fatal(err)
	}
	input := description.Children[0]
	if !input.Properties.Content.Password || !input.Properties.Content.PasswordToggle || !input.Properties.Semantics.Disabled || !input.Properties.Semantics.HasChange {
		t.Fatalf("grouped Input semantics = %+v/%+v", input.Properties.Content, input.Properties.Semantics)
	}
}

func TestInputGroupThemeMetricsAndHiDPILogicalGeometry(t *testing.T) {
	theme := LightTheme()
	if _, ok := theme.Components[ComponentInputGroup]; !ok {
		t.Fatal("built-in InputGroup component theme is missing")
	}
	value := "x"
	app := NewApp(AppOptions{Width: 300, Height: 80, Theme: theme})
	app.root = func() View {
		return InputGroup(InputGroupProps{Style: Style{Width: Px(240)}}, InputGroupContent{Input: groupedInput("x", &value), Prefix: Some(Label("@"))})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.geometry.Children[1].Rect.X
	app.running, app.events = true, &appFakeEvents{}
	theme.Semantic.Metrics[MetricComponentInputGroupGap] = Metric(12)
	if err := app.SetTheme(theme); err != nil {
		t.Fatal(err)
	}
	if _, err := app.runQueuedWork(); err != nil {
		t.Fatal(err)
	}
	after := app.geometry.Children[1].Rect.X
	if after-before != 4 {
		t.Fatalf("token gap delta = %v, want 4 logical units", after-before)
	}

	scaled := NewApp(AppOptions{Width: 300, Height: 80, Theme: theme})
	scaled.backend = &editorRuntime{viewport: platform.Viewport{
		LogicalWidth: 300, LogicalHeight: 80, PixelWidth: 375, PixelHeight: 100,
	}}
	scaled.root = app.root
	if err := scaled.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if scaled.geometry.Rect != app.geometry.Rect || scaled.geometry.Children[1].Rect != app.geometry.Children[1].Rect {
		t.Fatalf("1.25x scale changed logical geometry: base=%+v scaled=%+v", app.geometry, scaled.geometry)
	}
}
