package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func fixedRadio(selected, disabled bool, selectRadio func()) View {
	return Radio(RadioProps{Style: Style{
		Width: Px(120), Height: Px(36), Position: PositionAbsolute,
	}, Selected: selected, Disabled: disabled, OnSelect: selectRadio}, Text(TextProps{Value: "Radio"}))
}

func activateRadioPointer(t *testing.T, app *App) {
	t.Helper()
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 5, 5)); err != nil {
		t.Fatal(err)
	}
}

func activateRadioSpace(t *testing.T, app *App) {
	t.Helper()
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	if _, _, err := app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false)); err != nil {
		t.Fatal(err)
	}
}

func TestRadioIsControlledAcrossMouseAndSpace(t *testing.T) {
	app := NewApp(AppOptions{Width: 140, Height: 50})
	selected := false
	selections := 0
	app.root = func() View {
		return fixedRadio(selected, false, func() {
			selections++
			selected = true
		})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}

	activateRadioPointer(t, app)
	if !selected || selections != 1 || !app.retained.Root().State.Checked {
		t.Fatalf("mouse controlled state = %v/%d/%v", selected, selections, app.retained.Root().State.Checked)
	}
	activateRadioPointer(t, app)
	activateRadioSpace(t, app)
	if selections != 1 || !selected {
		t.Fatalf("selected radio activated again: selections=%d selected=%v", selections, selected)
	}

	selected = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	activateRadioSpace(t, app)
	if selections != 2 || !selected || !app.retained.Root().State.Checked {
		t.Fatalf("Space controlled state = %v/%d/%v", selected, selections, app.retained.Root().State.Checked)
	}
}

func TestRadioRejectedControlledSelectionDoesNotDiverge(t *testing.T) {
	app := NewApp(AppOptions{Width: 140, Height: 50})
	selections := 0
	app.root = func() View { return fixedRadio(false, false, func() { selections++ }) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	activateRadioPointer(t, app)
	if selections != 1 || app.retained.Root().State.Checked {
		t.Fatalf("rejected selection diverged: selections=%d selected=%v", selections, app.retained.Root().State.Checked)
	}
}

func TestRadioSelectedChangeIsPaintOnly(t *testing.T) {
	app := NewApp(AppOptions{Width: 140, Height: 50})
	selected := false
	app.root = func() View { return fixedRadio(selected, false, nil) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	selected = true
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	after := app.Diagnostics()
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) || after.LayoutCount != before.LayoutCount || after.PaintCount != before.PaintCount+1 {
		t.Fatalf("selected dirty/diagnostics = %v / %+v -> %+v", changes.Dirty, before, after)
	}
}

func TestRadioThemeStatePriorityAndMetricInvalidation(t *testing.T) {
	before, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	instance := &tree.Node{
		Properties: tree.Properties{Semantics: tree.SemanticsProperties{Checked: true}},
		State:      tree.State{Hovered: true, FocusVisible: true, Pressed: true, Checked: true},
	}
	visual, err := computeVisual(viewProps{}, instance, viewRadio, before)
	if err != nil {
		t.Fatal(err)
	}
	if visual.opacity != .72 || visual.border.width != 0 || len(visual.shadows) != 1 || visual.shadows[0].Spread != 2 {
		t.Fatalf("pressed selected focus visual = %+v", visual)
	}
	instance.Properties.Semantics.Disabled = true
	visual, err = computeVisual(viewProps{}, instance, viewRadio, before)
	if err != nil {
		t.Fatal(err)
	}
	if visual.opacity != .42 {
		t.Fatalf("disabled priority opacity = %g, want .42", visual.opacity)
	}

	changed := LightTheme()
	changed.Semantic.Metrics[MetricComponentRadioGap] = Metric(12)
	after, err := prepareTheme(changed)
	if err != nil {
		t.Fatal(err)
	}
	if !themeChangesLayout(Radio(RadioProps{}, Text(TextProps{})), before, after) {
		t.Fatal("Radio gap token change did not invalidate layout")
	}
}

func TestRadioDisabledAndNilCallbackDoNotSelect(t *testing.T) {
	for _, test := range []struct {
		name     string
		disabled bool
		callback bool
	}{
		{name: "disabled", disabled: true, callback: true},
		{name: "nil callback"},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 140, Height: 50})
			selections := 0
			var callback func()
			if test.callback {
				callback = func() { selections++ }
			}
			app.root = func() View { return fixedRadio(false, test.disabled, callback) }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			activateRadioPointer(t, app)
			activateRadioSpace(t, app)
			if selections != 0 || app.retained.Root().State.Checked {
				t.Fatalf("radio changed: selections=%d selected=%v", selections, app.retained.Root().State.Checked)
			}
		})
	}
}

func TestRadioHoverFocusAndPressedState(t *testing.T) {
	app := NewApp(AppOptions{Width: 140, Height: 50})
	app.root = func() View { return fixedRadio(false, false, func() {}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 40, 34))
	if !app.retained.Root().State.Hovered {
		t.Fatal("Radio did not retain hover across its label row")
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 40, 34))
	state := app.retained.Root().State
	if !state.Pressed || !state.Focused || state.FocusVisible {
		t.Fatalf("pointer state = %+v", state)
	}
	app.handleEvent(appPointer(platform.EventMouseUp, 40, 34))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	state = app.retained.Root().State
	if !state.Pressed || !state.Focused || !state.FocusVisible {
		t.Fatalf("keyboard state = %+v", state)
	}
}

func TestRadioIndicatorContributesIntrinsicFlexSize(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Align: AlignStart}, Radio(RadioProps{}, Text(TextProps{})))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	radio := app.geometry.Children[0]
	if radio.Rect.Height != 36 || len(radio.Children) != 2 || radio.Children[0].Rect.Width != 18 || radio.Children[0].Rect.Y != 9 {
		t.Fatalf("radio intrinsic geometry = %+v children=%+v", radio.Rect, radio.Children)
	}
}

func TestRadioDisplayUsesCircularIndicatorAndCenteredDot(t *testing.T) {
	theme, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	box := paint.Rect{X: 10, Y: 20, Width: 18, Height: 18}
	var list paint.DisplayList
	if err := appendRadioDisplay(&list, box, true, theme, 7); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("selected radio commands = %#v", list)
	}
	outer, dot := list[0], list[1]
	if outer.Kind != paint.CommandFillStrokeRoundedRect || outer.Rect != box || outer.Radii != (paint.Radii{TopLeft: 9, TopRight: 9, BottomRight: 9, BottomLeft: 9}) {
		t.Fatalf("radio outer indicator = %#v", outer)
	}
	wantDot := paint.Rect{X: 14.5, Y: 24.5, Width: 9, Height: 9}
	accent, _ := theme.color(TokenColor(ColorSemanticAccent))
	if dot.Kind != paint.CommandFillRoundedRect || dot.Rect != wantDot || dot.Radii != (paint.Radii{TopLeft: 4.5, TopRight: 4.5, BottomRight: 4.5, BottomLeft: 4.5}) || dot.Color != paintColor(accent) {
		t.Fatalf("radio selected dot = %#v", dot)
	}

	list = nil
	if err := appendRadioDisplay(&list, box, false, theme, 7); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Kind != paint.CommandFillStrokeRoundedRect {
		t.Fatalf("unselected radio commands = %#v", list)
	}
}
