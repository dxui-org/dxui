package dxui

import (
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

func groupButton(key string, disabled bool, onPress func()) View {
	return Button(ButtonProps{
		Key: key, Style: Style{Width: Px(40), Height: Px(30), Shrink: Some(float32(0))},
		Disabled: disabled, OnPress: onPress,
	}, Text(TextProps{Value: key}))
}

func TestButtonGroupValidatesOrientationAndButtonChildren(t *testing.T) {
	for _, view := range []View{
		ButtonGroup(ButtonGroupProps{Orientation: ButtonGroupOrientation(2)}),
		ButtonGroup(ButtonGroupProps{}, Text(TextProps{Value: "not a button"})),
	} {
		if _, err := describeView(view); err == nil {
			t.Fatal("invalid ButtonGroup was accepted")
		}
	}
	if _, err := describeView(ButtonGroup(ButtonGroupProps{})); err != nil {
		t.Fatalf("empty ButtonGroup: %v", err)
	}
	if _, err := describeView(ButtonGroup(ButtonGroupProps{}, groupButton("only", false, nil))); err != nil {
		t.Fatalf("single-button group: %v", err)
	}
}

func TestButtonGroupBuiltInThemeProvidesConnectedStyle(t *testing.T) {
	for name, theme := range map[string]Theme{"light": LightTheme(), "dark": DarkTheme()} {
		t.Run(name, func(t *testing.T) {
			component, ok := theme.Components[ComponentButtonGroup]
			if !ok {
				t.Fatal("ComponentButtonGroup is missing")
			}
			border, borderSet := component.Base.Border.get()
			radius, radiusSet := component.Base.Radius.get()
			if !borderSet || border.Width != TokenMetric(MetricComponentButtonGroupBorderWidth) || !radiusSet || radius != UniformCorners(TokenMetric(MetricComponentButtonGroupRadius)) {
				t.Fatalf("connected style = %+v/%+v", border, radius)
			}
		})
	}
}

func TestButtonGroupHorizontalVerticalAndDividerLayout(t *testing.T) {
	tests := []struct {
		name    string
		props   ButtonGroupProps
		secondX float32
		secondY float32
	}{
		{name: "horizontal default", secondX: 40},
		{name: "vertical default", props: ButtonGroupProps{Orientation: ButtonGroupVertical}, secondY: 30},
		{name: "horizontal dividers", props: ButtonGroupProps{Dividers: true}, secondX: 39},
		{name: "vertical dividers", props: ButtonGroupProps{Orientation: ButtonGroupVertical, Dividers: true}, secondY: 29},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 160, Height: 100})
			app.root = func() View {
				return ButtonGroup(test.props, groupButton("one", false, nil), groupButton("two", false, nil))
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			second := app.geometry.Children[1].Rect
			if second.X != test.secondX || second.Y != test.secondY {
				t.Fatalf("second position = (%v,%v), want (%v,%v)", second.X, second.Y, test.secondX, test.secondY)
			}
		})
	}
}

func TestVerticalButtonGroupStretchesAutomaticButtonWidths(t *testing.T) {
	app := NewApp(AppOptions{Width: 300, Height: 160})
	app.root = func() View {
		return Box(BoxProps{Align: AlignStart}, ButtonGroup(ButtonGroupProps{Orientation: ButtonGroupVertical},
			Button(ButtonProps{}, Text(TextProps{Value: "Short"})),
			Button(ButtonProps{}, Text(TextProps{Value: "A much wider button"})),
		))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	group := app.geometry.Children[0]
	if len(group.Children) != 2 || group.Children[0].Rect.Width != group.Children[1].Rect.Width {
		t.Fatalf("vertical button widths = %v/%v, want equal", group.Children[0].Rect.Width, group.Children[1].Rect.Width)
	}
}

func TestButtonGroupPaintsOneSharedBorderAndOuterRadii(t *testing.T) {
	app := NewApp(AppOptions{Width: 160, Height: 40})
	app.root = func() View {
		return ButtonGroup(ButtonGroupProps{Dividers: true},
			groupButton("first", false, nil), groupButton("middle", false, nil), groupButton("last", false, nil))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	wantRadii := []paint.Radii{
		{TopLeft: 8, BottomLeft: 8},
		{},
		{TopRight: 8, BottomRight: 8},
	}
	for index, child := range app.retained.Root().Children {
		var command *paint.Command
		for commandIndex := range app.display {
			candidate := &app.display[commandIndex]
			if candidate.NodeID == child.ID && candidate.Kind == paint.CommandFillStrokeRoundedRect {
				command = candidate
				break
			}
		}
		if command == nil {
			t.Fatalf("button %d emitted no connected fill/stroke", index)
		}
		if command.Width != 1 || command.Radii != wantRadii[index] {
			t.Errorf("button %d border/radii = %v/%+v, want 1/%+v", index, command.Width, command.Radii, wantRadii[index])
		}
	}
	if first, second := app.geometry.Children[0].Rect, app.geometry.Children[1].Rect; first.X+first.Width-second.X != 1 {
		t.Fatalf("connected border overlap = %v, want 1", first.X+first.Width-second.X)
	}
}

func TestButtonGroupDefaultsToNoDividers(t *testing.T) {
	app := NewApp(AppOptions{Width: 120, Height: 40})
	app.root = func() View {
		return ButtonGroup(ButtonGroupProps{}, groupButton("first", false, nil), groupButton("last", false, nil))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	for index, child := range app.retained.Root().Children {
		fillFound := false
		for _, command := range app.display {
			if command.NodeID == child.ID && command.Kind == paint.CommandDrawShadow {
				t.Fatalf("default ButtonGroup Button emitted a shadow: %+v", command)
			}
			if command.NodeID == child.ID && (command.Kind == paint.CommandStrokeRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect) {
				t.Fatalf("default ButtonGroup Button emitted divider border: %+v", command)
			}
			if command.NodeID == child.ID && command.Kind == paint.CommandFillRoundedRect {
				fillFound = true
				want := paint.JoinedRight
				if index > 0 {
					want = paint.JoinedLeft
				}
				if command.JoinedEdges != want {
					t.Fatalf("default ButtonGroup Button %d joined edges = %d, want %d", child.ID, command.JoinedEdges, want)
				}
			}
		}
		if !fillFound {
			t.Fatalf("default ButtonGroup Button %d emitted no joined fill", child.ID)
		}
	}
	first, second := app.geometry.Children[0].Rect, app.geometry.Children[1].Rect
	if overlap := first.X + first.Width - second.X; overlap != 0 {
		t.Fatalf("default ButtonGroup overlap = %v, want 0: first=%+v second=%+v", overlap, first, second)
	}
}

func TestButtonGroupHoverCoversJoinedLeadingEdge(t *testing.T) {
	app := NewApp(AppOptions{Width: 160, Height: 40})
	app.root = func() View {
		return ButtonGroup(ButtonGroupProps{}, groupButton("first", false, nil), groupButton("second", false, nil), groupButton("third", false, nil))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{1, 2} {
		rect := app.geometry.Children[index].Rect
		if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, rect.X+rect.Width/2, rect.Y+rect.Height/2)); err != nil {
			t.Fatal(err)
		}
		id := app.retained.Root().Children[index].ID
		var background *paint.Command
		fillCount := 0
		for commandIndex := range app.display {
			command := &app.display[commandIndex]
			if command.NodeID != id || command.Kind != paint.CommandFillRoundedRect {
				continue
			}
			background = command
			fillCount++
		}
		wantEdges := paint.JoinedLeft
		if index+1 < len(app.geometry.Children) {
			wantEdges |= paint.JoinedRight
		}
		if background == nil || fillCount != 1 || background.JoinedEdges != wantEdges {
			t.Fatalf("hovered Button %d fill count/background = %d/%+v", index, fillCount, background)
		}
	}
}

func TestButtonGroupExplicitButtonStyleWins(t *testing.T) {
	custom := RGBA(180, 30, 90, 255)
	styled := Button(ButtonProps{Key: "styled", Style: Style{
		Width: Px(40), Height: Px(30), Border: Border{Width: Metric(3), Color: LiteralColor(custom)},
		Radius: UniformCorners(Metric(4)),
	}}, Text(TextProps{Value: "styled"}))
	app := NewApp(AppOptions{Width: 120, Height: 40})
	app.root = func() View {
		return ButtonGroup(ButtonGroupProps{}, groupButton("first", false, nil), styled)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := app.retained.Root().Children[1].ID
	for _, command := range app.display {
		if command.NodeID == id && command.Kind == paint.CommandFillStrokeRoundedRect {
			if command.Width != 3 || command.BorderColor != (paint.Color{R: 180, G: 30, B: 90, A: 255}) || command.Radii != (paint.Radii{TopLeft: 4, TopRight: 4, BottomRight: 4, BottomLeft: 4}) {
				t.Fatalf("explicit style lost: %+v", command)
			}
			return
		}
	}
	t.Fatal("styled button emitted no fill/stroke")
}

func TestButtonGroupFocusOrderActivationAndKeyedDynamics(t *testing.T) {
	order := []string{"one", "disabled", "three"}
	presses := []string{}
	app := NewApp(AppOptions{Width: 200, Height: 50})
	app.root = func() View {
		buttons := make([]View, 0, len(order))
		for _, key := range order {
			key := key
			buttons = append(buttons, groupButton(key, key == "disabled", func() { presses = append(presses, key) }))
		}
		return ButtonGroup(ButtonGroupProps{}, buttons...)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	root := app.retained.Root()
	firstID, thirdID := root.Children[0].ID, root.Children[2].ID
	if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false)); err != nil || app.interaction.Focused() != firstID {
		t.Fatalf("first Tab focus = %d/%v, want %d", app.interaction.Focused(), err, firstID)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() != thirdID {
		t.Fatalf("second Tab focus = %d, want %d", app.interaction.Focused(), thirdID)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false))
	if got := strings.Join(presses, ","); got != "one,three" {
		t.Fatalf("activation order = %q", got)
	}

	order = []string{"three", "new", "one"}
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	root = app.retained.Root()
	if len(root.Children) != 3 || root.Children[0].ID != thirdID || root.Children[2].ID != firstID {
		t.Fatalf("keyed reorder IDs = %d/%d/%d, want %d/new/%d", root.Children[0].ID, root.Children[1].ID, root.Children[2].ID, thirdID, firstID)
	}
	if app.interaction.Focused() != thirdID {
		t.Fatalf("focused keyed button changed after reorder: %d", app.interaction.Focused())
	}
	order = []string{"new", "one"}
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if len(app.retained.Root().Children) != 2 || app.interaction.Focused() == thirdID {
		t.Fatal("focused removed button survived dynamic deletion")
	}
}

func TestButtonGroupThemeMetricInvalidation(t *testing.T) {
	tests := []struct {
		name     string
		dividers bool
		token    MetricToken
		value    float32
	}{
		{name: "divider overlap", dividers: true, token: MetricComponentButtonGroupBorderWidth, value: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 120, Height: 40, Theme: LightTheme()})
			app.root = func() View {
				return ButtonGroup(ButtonGroupProps{Dividers: test.dividers}, groupButton("one", false, nil), groupButton("two", false, nil))
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			app.running, app.events = true, &appFakeEvents{}
			next := LightTheme()
			next.Semantic.Metrics[test.token] = Metric(test.value)
			if err := app.SetTheme(next); err != nil {
				t.Fatal(err)
			}
			if dirty, err := app.runQueuedWork(); err != nil || !dirty {
				t.Fatalf("theme work = %v/%v", dirty, err)
			}
			if got := app.Diagnostics(); got.LayoutCount != 2 || got.PaintCount != 2 {
				t.Fatalf("metric invalidation = %+v", got)
			}
		})
	}

	app := NewApp(AppOptions{Width: 120, Height: 40, Theme: LightTheme()})
	app.root = func() View {
		return ButtonGroup(ButtonGroupProps{}, groupButton("one", false, nil), groupButton("two", false, nil))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.running, app.events = true, &appFakeEvents{}
	next := LightTheme()
	next.Semantic.Metrics[MetricComponentButtonGroupRadius] = Metric(4)
	if err := app.SetTheme(next); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("radius theme work = %v/%v", dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 1 || got.PaintCount != 2 {
		t.Fatalf("radius invalidation = %+v, want paint-only", got)
	}
}
