package dxui

import (
	"math"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func fixedSlider(value, minimum, maximum, step float32, disabled bool, change func(float32)) View {
	return Slider(SliderProps{Style: Style{
		Width: Px(100), Height: Px(24), Position: PositionAbsolute,
	}, Value: value, Min: minimum, Max: maximum, Step: step, Disabled: disabled, OnChange: change})
}

func TestSliderNormalization(t *testing.T) {
	tests := []struct {
		name                  string
		props                 SliderProps
		min, max, step, value float32
		valid                 bool
	}{
		{name: "zero defaults", props: SliderProps{}, min: 0, max: 100, step: 1, value: 0, valid: true},
		{name: "custom", props: SliderProps{Min: -10, Max: 0, Step: .25, Value: -3.2}, min: -10, max: 0, step: .25, value: -3.2, valid: true},
		{name: "step zero defaults", props: SliderProps{Max: 10}, min: 0, max: 10, step: 1, value: 0, valid: true},
		{name: "step negative defaults", props: SliderProps{Max: 10, Step: -2}, min: 0, max: 10, step: 1, value: 0, valid: true},
		{name: "step nan defaults", props: SliderProps{Max: 10, Step: float32(math.NaN())}, min: 0, max: 10, step: 1, value: 0, valid: true},
		{name: "value below", props: SliderProps{Max: 10, Value: -4}, min: 0, max: 10, step: 1, value: 0, valid: true},
		{name: "value above", props: SliderProps{Max: 10, Value: 14}, min: 0, max: 10, step: 1, value: 10, valid: true},
		{name: "value nan", props: SliderProps{Min: 2, Max: 10, Value: float32(math.NaN())}, min: 2, max: 10, step: 1, value: 2, valid: true},
		{name: "value positive infinity", props: SliderProps{Min: 2, Max: 10, Value: float32(math.Inf(1))}, min: 2, max: 10, step: 1, value: 10, valid: true},
		{name: "value negative infinity", props: SliderProps{Min: 2, Max: 10, Value: float32(math.Inf(-1))}, min: 2, max: 10, step: 1, value: 2, valid: true},
		{name: "equal range inert", props: SliderProps{Min: 2, Max: 2}, min: 2, max: 2, step: 1, value: 2},
		{name: "reversed range inert", props: SliderProps{Min: 3, Max: 2}, min: 3, max: 3, step: 1, value: 3},
		{name: "nan bound inert", props: SliderProps{Min: float32(math.NaN()), Max: 2}, min: 0, max: 0, step: 1, value: 0},
		{name: "infinite bound inert", props: SliderProps{Min: 0, Max: float32(math.Inf(1))}, min: 0, max: 0, step: 1, value: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeSlider(test.props)
			if got.min != test.min || got.max != test.max || got.step != test.step || got.value != test.value || got.valid != test.valid {
				t.Fatalf("normalize = %+v", got)
			}
		})
	}
}

func TestSliderStepAlignmentAvoidsAccumulation(t *testing.T) {
	model := normalizeSlider(SliderProps{Min: .1, Max: 1, Step: .1, Value: .1})
	for range 7 {
		next := model.stepBy(1)
		model.value = next
	}
	if model.value != float32(.8) {
		t.Fatalf("seven steps = %.9f, want %.9f", model.value, float32(.8))
	}
	model.value = .26
	if got := model.stepBy(1); got != float32(.3) {
		t.Fatalf("right from off-grid = %.9f", got)
	}
	if got := model.stepBy(-1); got != float32(.2) {
		t.Fatalf("left from off-grid = %.9f", got)
	}
	model = normalizeSlider(SliderProps{Min: 0, Max: 10, Step: 3, Value: 9})
	if got := model.stepBy(1); got != 10 {
		t.Fatalf("upper endpoint = %v", got)
	}
	model.value = 10
	if got := model.stepBy(-1); got != 9 {
		t.Fatalf("step down from endpoint = %v", got)
	}
}

func TestSliderIsControlledAcrossTrackClickAndOutsideDrag(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 24})
	value := float32(20)
	var proposals []float32
	app.root = func() View {
		return fixedSlider(value, 0, 100, 10, false, func(next float32) {
			proposals = append(proposals, next)
			value = next
		})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 80, 12))
	if value != 50 || !app.retained.Root().State.SliderDragging {
		t.Fatalf("track click = value %v dragging %v geometry=%+v", value, app.retained.Root().State.SliderDragging, app.geometry.Rect)
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 250, 12))
	if value != 100 || len(proposals) != 2 {
		t.Fatalf("outside drag = value %v proposals %v", value, proposals)
	}
	app.handleEvent(appPointer(platform.EventMouseUp, 250, 12))
	if app.sliderDrag.id != 0 || app.retained.Root().State.SliderDragging {
		t.Fatal("mouse release retained slider capture")
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 10, 12))
	if len(proposals) != 2 {
		t.Fatalf("move after release proposed %v", proposals)
	}
}

func TestSliderRejectedProposalDoesNotDiverge(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 24})
	var proposals []float32
	app.root = func() View {
		return fixedSlider(20, 0, 100, 10, false, func(next float32) { proposals = append(proposals, next) })
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 150, 12))
	if len(proposals) != 1 || proposals[0] != 100 || app.retained.Root().Properties.Slider.Value != 20 {
		t.Fatalf("rejected proposal = %v, retained value %v geometry=%+v", proposals, app.retained.Root().Properties.Slider.Value, app.geometry.Rect)
	}
}

func TestSliderKeyboardAndCancellation(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 24})
	value := float32(50)
	app.root = func() View { return fixedSlider(value, 0, 100, 10, false, func(next float32) { value = next }) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 80, 12))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEscape, false))
	if app.sliderDrag.id != 0 {
		t.Fatal("Escape did not cancel drag")
	}
	keys := []struct {
		key  platform.Key
		want float32
	}{{platform.KeyLeft, 40}, {platform.KeyDown, 30}, {platform.KeyRight, 40}, {platform.KeyUp, 50}, {platform.KeyHome, 0}, {platform.KeyEnd, 100}}
	for _, test := range keys {
		app.handleEvent(appKey(platform.EventKeyDown, test.key, false))
		if value != test.want {
			t.Fatalf("key %v = %v, want %v", test.key, value, test.want)
		}
	}
}

func TestSliderDisabledInvalidAndNilCallbackAreSafe(t *testing.T) {
	for _, props := range []SliderProps{
		{Style: Style{Width: Px(100), Height: Px(24)}, Max: 100, Disabled: true, OnChange: func(float32) { t.Fatal("disabled callback") }},
		{Style: Style{Width: Px(100), Height: Px(24)}, Min: 5, Max: 5, OnChange: func(float32) { t.Fatal("invalid callback") }},
		{Style: Style{Width: Px(100), Height: Px(24)}, Max: 100, OnChange: nil},
	} {
		app := NewApp(AppOptions{Width: 100, Height: 24})
		current := props
		app.root = func() View { return Slider(current) }
		if err := app.buildRoot(); err != nil {
			t.Fatal(err)
		}
		app.handleEvent(appPointer(platform.EventMouseDown, 50, 12))
		app.handleEvent(appPointer(platform.EventMouseMove, 90, 12))
		app.handleEvent(appPointer(platform.EventMouseUp, 90, 12))
	}
}

func TestSliderDisabledIsSkippedByFocusAndRemovalCancelsDrag(t *testing.T) {
	visible := true
	app := NewApp(AppOptions{Width: 200, Height: 60})
	app.root = func() View {
		if !visible {
			return Slider(SliderProps{Key: "enabled", Max: 100})
		}
		return Box(BoxProps{},
			Slider(SliderProps{Key: "disabled", Max: 100, Disabled: true}),
			Slider(SliderProps{Key: "drag", Max: 100}),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if got, want := app.interaction.Focused(), app.retained.Root().Children[1].ID; got != want {
		t.Fatalf("Tab focus = %d, want enabled Slider %d", got, want)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 20, 36))
	if app.sliderDrag.id == 0 {
		t.Fatal("Slider drag did not start")
	}
	visible = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.sliderDrag.id != 0 {
		t.Fatal("removed Slider retained drag capture")
	}
}

func TestSliderValueChangeIsPaintOnly(t *testing.T) {
	app := NewApp(AppOptions{Width: 200, Height: 40})
	value := float32(20)
	app.root = func() View { return Slider(SliderProps{Value: value}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	value = 80
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	after := app.Diagnostics()
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) || after.LayoutCount != before.LayoutCount || after.PaintCount != before.PaintCount+1 {
		t.Fatalf("slider value dirty/diagnostics = %v / %+v -> %+v", changes.Dirty, before, after)
	}
}

func TestSliderCallbackAndStepChangesAreSemanticOnly(t *testing.T) {
	app := NewApp(AppOptions{Width: 200, Height: 40})
	step := float32(1)
	var change func(float32)
	app.root = func() View { return Slider(SliderProps{Value: 20, Step: step, OnChange: change}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	step = 5
	change = func(float32) {}
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	after := app.Diagnostics()
	if !changes.Has(tree.DirtySemantics) || changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) || after.PaintCount != before.PaintCount || after.LayoutCount != before.LayoutCount {
		t.Fatalf("semantic-only dirty/diagnostics = %v / %+v -> %+v", changes.Dirty, before, after)
	}
}

func TestSliderIntrinsicLayoutAndDisplay(t *testing.T) {
	app := NewApp(AppOptions{Width: 200, Height: 50})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Align: AlignStart}, Slider(SliderProps{Value: 25}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	sliderGeometry := app.geometry.Children[0]
	if got := sliderGeometry.Rect; got.Width != 160 || got.Height != 24 {
		t.Fatalf("slider intrinsic rect = %+v", got)
	}
	sliderNode := app.retained.Root().Children[0]
	var shapes []paint.Command
	for _, command := range app.display {
		if command.NodeID == sliderNode.ID && (command.Kind == paint.CommandFillRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect) {
			shapes = append(shapes, command)
		}
	}
	if len(shapes) != 3 {
		t.Fatalf("slider shape count = %d", len(shapes))
	}
	track, fill, thumb := shapes[0], shapes[1], shapes[2]
	if track.Rect.X != 9 || track.Rect.Y != 10 || track.Rect.Width != 142 || track.Rect.Height != 4 || fill.Rect.Width != 35.5 {
		t.Fatalf("track/fill geometry = %+v / %+v", track.Rect, fill.Rect)
	}
	if thumb.Kind != paint.CommandFillStrokeRoundedRect || thumb.Rect.X != 35.5 || thumb.Rect.Y != 3 || thumb.Rect.Width != 18 || thumb.Radii.TopLeft != 9 || thumb.Width != 1 || thumb.BorderColor.A == 0 {
		t.Fatalf("thumb geometry = %+v radii=%+v", thumb.Rect, thumb.Radii)
	}
	list2, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, app.text, app.images, 2, 2, app.options.Caches.TextSourceBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !displayListsEqual(app.display, list2) {
		t.Fatal("DPI scale changed Slider logical display geometry")
	}
}

func TestSliderThemeInvalidationClasses(t *testing.T) {
	app := NewApp(AppOptions{Width: 200, Height: 50})
	app.root = func() View { return Slider(SliderProps{Value: 50}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	paintTheme := LightTheme()
	paintTheme.Semantic.Metrics[MetricComponentSliderTrackHeight] = Metric(8)
	if err := app.SetTheme(paintTheme); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("paint theme dirty/error = %v/%v", dirty, err)
	}
	afterPaint := app.Diagnostics()
	if afterPaint.LayoutCount != before.LayoutCount || afterPaint.PaintCount != before.PaintCount+1 {
		t.Fatalf("track metric layout/paint = %d/%d, before %d/%d", afterPaint.LayoutCount, afterPaint.PaintCount, before.LayoutCount, before.PaintCount)
	}
	layoutTheme := paintTheme
	layoutTheme.Semantic.Metrics[MetricComponentSliderWidth] = Metric(180)
	if err := app.SetTheme(layoutTheme); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("layout theme dirty/error = %v/%v", dirty, err)
	}
	if afterLayout := app.Diagnostics(); afterLayout.LayoutCount != afterPaint.LayoutCount+1 {
		t.Fatalf("width metric did not relayout: %d -> %d", afterPaint.LayoutCount, afterLayout.LayoutCount)
	}
	beforeTheme, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	afterTheme, err := prepareTheme(layoutTheme)
	if err != nil {
		t.Fatal(err)
	}
	explicit := Slider(SliderProps{Style: Style{Width: Px(90), Height: Px(18)}})
	if themeChangesLayout(explicit, beforeTheme, afterTheme) {
		t.Fatal("Slider intrinsic theme dimensions invalidated explicit dimensions")
	}
}

func TestSliderLocalStyleAndStatesAreApplied(t *testing.T) {
	app := NewApp(AppOptions{Width: 180, Height: 40})
	app.root = func() View {
		return Slider(SliderProps{Style: Style{
			Background: LiteralColor(RGBA(1, 2, 3, 255)), Overflow: OverflowClip,
		}, States: StateStyles{Hover: StylePatch{Opacity: Some(float32(.5))}}, Value: 50})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 20, 10))
	if !app.retained.Root().State.Hovered {
		t.Fatal("Slider did not enter hover state")
	}
	foundBackground, foundClip := false, false
	for _, command := range app.display {
		if command.NodeID != app.retained.Root().ID {
			continue
		}
		if command.Kind == paint.CommandFillRoundedRect && command.Color == paintColor(RGBA(1, 2, 3, 255)) {
			foundBackground = true
		}
		if command.Kind == paint.CommandPushClip {
			foundClip = true
		}
	}
	if !foundBackground || !foundClip {
		t.Fatalf("local style display background/clip = %v/%v", foundBackground, foundClip)
	}
}

func TestSliderPressedStateSurvivesOutsideDrag(t *testing.T) {
	app := NewApp(AppOptions{Width: 160, Height: 24})
	app.root = func() View { return Slider(SliderProps{Value: 50}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 80, 12))
	app.handleEvent(appPointer(platform.EventMouseMove, 300, 12))
	if app.retained.Root().State.Hovered || app.retained.Root().State.Pressed || !app.retained.Root().State.SliderDragging {
		t.Fatalf("outside drag states = hover %v pressed %v dragging %v", app.retained.Root().State.Hovered, app.retained.Root().State.Pressed, app.retained.Root().State.SliderDragging)
	}
	foundPressedOpacity := false
	for _, command := range app.display {
		if command.NodeID == app.retained.Root().ID && command.Kind == paint.CommandPushOpacity && command.Opacity == .72 {
			foundPressedOpacity = true
		}
	}
	if !foundPressedOpacity {
		t.Fatal("captured outside drag lost Pressed visual")
	}
	app.handleEvent(appPointer(platform.EventMouseUp, 300, 12))
	if app.retained.Root().State.SliderDragging {
		t.Fatal("release did not clear Pressed drag state")
	}
}
