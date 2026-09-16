package dxui

import (
	"math"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func progressShapes(app *App, nodeID uint64) []paint.Command {
	var result []paint.Command
	for _, command := range app.display {
		if command.NodeID == nodeID && command.Kind == paint.CommandFillRoundedRect {
			result = append(result, command)
		}
	}
	return result
}

func TestProgressBarNormalizesBoundaryAndExceptionalValues(t *testing.T) {
	tests := []struct {
		name  string
		value float32
		want  float32
		fills int
	}{
		{name: "negative", value: -0.25, want: 0, fills: 1},
		{name: "zero", value: 0, want: 0, fills: 1},
		{name: "quarter", value: 0.25, want: 0.25, fills: 2},
		{name: "one", value: 1, want: 1, fills: 2},
		{name: "above one", value: 2, want: 1, fills: 2},
		{name: "NaN", value: float32(math.NaN()), want: 0, fills: 1},
		{name: "negative infinity", value: float32(math.Inf(-1)), want: 0, fills: 1},
		{name: "positive infinity", value: float32(math.Inf(1)), want: 1, fills: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 200, Height: 30})
			app.root = func() View { return ProgressBar(ProgressBarProps{Value: test.value}) }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			if got := app.retained.Root().Properties.Progress.Value; got != test.want || math.IsNaN(float64(got)) || math.IsInf(float64(got), 0) {
				t.Fatalf("normalized retained value = %g, want %g", got, test.want)
			}
			shapes := progressShapes(app, app.retained.Root().ID)
			if len(shapes) != test.fills {
				t.Fatalf("fill command count = %d, want %d: %+v", len(shapes), test.fills, shapes)
			}
			for _, shape := range shapes {
				values := []float32{shape.Rect.X, shape.Rect.Y, shape.Rect.Width, shape.Rect.Height}
				for _, value := range values {
					if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || value < 0 {
						t.Fatalf("invalid display geometry value %g in %+v", value, shape.Rect)
					}
				}
			}
			if test.fills == 2 {
				track, fill := shapes[0], shapes[1]
				if fill.Rect.X != track.Rect.X || fill.Rect.Width != track.Rect.Width*test.want {
					t.Fatalf("left-to-right fill = %+v for track %+v and value %g", fill.Rect, track.Rect, test.want)
				}
			}
		})
	}
}

func TestProgressBarLayoutContentPaintingAndLocalStyle(t *testing.T) {
	trackColor := RGBA(10, 20, 30, 255)
	fillColor := RGBA(40, 50, 60, 230)
	borderColor := RGBA(70, 80, 90, 220)
	app := NewApp(AppOptions{Width: 140, Height: 40})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Align: AlignStart}, ProgressBar(ProgressBarProps{Style: Style{
			Width: Px(100), Height: Px(20), MinWidth: Px(90), MaxWidth: Px(110),
			Padding:    Edges(2, 4, 2, 4),
			Background: LiteralColor(trackColor), Text: TextStyle{Color: LiteralColor(fillColor)},
			Border: Border{Width: Metric(2), Color: LiteralColor(borderColor)},
			Radius: UniformCorners(Metric(5)), Opacity: Some(float32(.5)), Overflow: OverflowClip,
		}, Value: .25}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := app.geometry.Children[0].Rect; got.Width != 100 || got.Height != 20 {
		t.Fatalf("explicit geometry = %+v, want 100x20", got)
	}
	shapes := progressShapes(app, app.retained.Root().Children[0].ID)
	if len(shapes) != 2 {
		t.Fatalf("progress shapes = %+v", shapes)
	}
	track, fill := shapes[0], shapes[1]
	if track.Rect != (paint.Rect{X: 4, Y: 6, Width: 92, Height: 8}) || fill.Rect != (paint.Rect{X: 4, Y: 6, Width: 23, Height: 8}) {
		t.Fatalf("content track/fill geometry = %+v / %+v", track.Rect, fill.Rect)
	}
	if track.Color != paintColor(trackColor) || fill.Color != paintColor(fillColor) || track.Radii.TopLeft != 5 || fill.Radii.TopRight != 5 {
		t.Fatalf("local track/fill style = %+v / %+v", track, fill)
	}
	var border, opacity, clip bool
	for _, command := range app.display {
		border = border || command.Kind == paint.CommandStrokeRoundedRect && command.Color == paintColor(borderColor) && command.Width == 2
		opacity = opacity || command.Kind == paint.CommandPushOpacity && command.Opacity == .5
		clip = clip || command.Kind == paint.CommandPushClip && command.Rect == (paint.Rect{Width: 100, Height: 20})
	}
	if !border || !opacity || !clip {
		t.Fatalf("border/opacity/clip = %t/%t/%t", border, opacity, clip)
	}
}

func TestProgressBarThemeDefaultsAndOverrides(t *testing.T) {
	for name, source := range map[string]Theme{"light": LightTheme(), "dark": DarkTheme()} {
		t.Run(name, func(t *testing.T) {
			component, ok := source.Components[ComponentProgressBar]
			if !ok || !component.Base.Background.set || !component.Base.TextColor.set || !component.Base.Radius.set {
				t.Fatalf("missing ProgressBar component theme: %+v", component)
			}
			theme, err := prepareTheme(source)
			if err != nil {
				t.Fatal(err)
			}
			for token, want := range map[MetricToken]float32{
				MetricComponentProgressBarWidth: 160, MetricComponentProgressBarHeight: 12,
				MetricComponentProgressBarTrackHeight: 8, MetricComponentProgressBarRadius: 999,
			} {
				if got, metricErr := theme.metric(TokenMetric(token)); metricErr != nil || got != want {
					t.Fatalf("metric %q = %g, %v; want %g", token, got, metricErr, want)
				}
			}
			for _, token := range []ColorToken{ColorSemanticProgressTrack, ColorSemanticProgressFill} {
				if color, colorErr := theme.color(TokenColor(token)); colorErr != nil || color.A == 0 {
					t.Fatalf("color %q = %+v, %v", token, color, colorErr)
				}
			}
		})
	}

	app := NewApp(AppOptions{Width: 200, Height: 30})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Align: AlignStart}, ProgressBar(ProgressBarProps{Value: .5}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := app.geometry.Children[0].Rect; got.Width != 160 || got.Height != 12 {
		t.Fatalf("theme intrinsic size = %+v", got)
	}

	explicit := ProgressBar(ProgressBarProps{Style: Style{Width: Px(90), Height: Px(18), MinWidth: Px(100), MaxWidth: Px(95)}})
	if got := intrinsicChildGeometry(t, LightTheme(), explicit); got.Width != 100 || got.Height != 18 {
		t.Fatalf("explicit/min/max precedence = %+v, want 100x18", got)
	}
}

func TestProgressBarValueChangeIsPaintOnly(t *testing.T) {
	value := float32(.25)
	app := NewApp(AppOptions{Width: 200, Height: 30})
	app.root = func() View { return ProgressBar(ProgressBarProps{Value: value}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	value = .75
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	after := app.Diagnostics()
	if !changes.Has(tree.DirtyDisplay) || !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyMeasure) || changes.Has(tree.DirtyLayout) || after.LayoutCount != before.LayoutCount || after.PaintCount != before.PaintCount+1 {
		t.Fatalf("value dirty/diagnostics = %v / %+v -> %+v", changes.Dirty, before, after)
	}
}

func TestProgressBarIsPassiveAndComposesWithScrollAndZIndex(t *testing.T) {
	bar := func(key string, z int, color RGBAColor) View {
		return ProgressBar(ProgressBarProps{Key: key, Style: Style{
			Width: Px(50), Height: Px(12), Position: PositionAbsolute, Insets: Insets{Top: Px(10)},
			ZIndex: z, Text: TextStyle{Color: LiteralColor(color)}, Overflow: OverflowClip,
		}, Value: 1})
	}
	app := NewApp(AppOptions{Width: 60, Height: 12})
	app.root = func() View {
		return Scroll(ScrollProps{Style: Style{Width: Px(60), Height: Px(12)}, Axis: ScrollVertical, Offset: Some(Point{Y: 10}), Scrollbar: ScrollbarHidden},
			Box(BoxProps{Style: Style{Height: Px(22)}, Align: AlignStart},
				bar("front", 2, RGBA(20, 0, 0, 255)), bar("back", 1, RGBA(10, 0, 0, 255))))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	var fills []paint.Command
	for _, command := range app.display {
		if command.Kind == paint.CommandFillRoundedRect && (command.Color.R == 10 || command.Color.R == 20) {
			fills = append(fills, command)
		}
	}
	if len(fills) != 2 || fills[0].Color.R != 10 || fills[1].Color.R != 20 || fills[0].Rect.Y != -8 || fills[1].Rect.Y != -8 {
		t.Fatalf("Scroll translation/z-index fills = %+v", fills)
	}
	for _, command := range app.display {
		if (command.NodeID == app.retained.Root().Children[0].Children[0].ID || command.NodeID == app.retained.Root().Children[0].Children[1].ID) && command.Kind == paint.CommandItem && command.Interactive {
			t.Fatal("ProgressBar unexpectedly marked interactive")
		}
	}

	passive := NewApp(AppOptions{Width: 100, Height: 20})
	passive.root = func() View {
		return ProgressBar(ProgressBarProps{Style: Style{Width: Px(100), Height: Px(20)}, Value: .5})
	}
	if err := passive.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, ok := passive.currentDisplay().HitTestAt(10, 5); ok {
		t.Fatal("standalone ProgressBar unexpectedly participated in hit testing")
	}
	for _, event := range []platform.Event{
		appPointer(platform.EventMouseMove, 10, 5), appPointer(platform.EventMouseDown, 10, 5), appPointer(platform.EventMouseUp, 10, 5),
		appKey(platform.EventKeyDown, platform.KeyTab, false), appKey(platform.EventKeyDown, platform.KeyEnter, false), appKey(platform.EventKeyDown, platform.KeySpace, false),
	} {
		if dirty, callback, err := passive.handleEvent(event); err != nil || dirty || callback {
			t.Fatalf("passive event result = dirty:%t callback:%t err:%v", dirty, callback, err)
		}
	}
	if passive.interaction.Focused() != 0 {
		t.Fatalf("ProgressBar received focus identity %d", passive.interaction.Focused())
	}
}

func TestProgressBarThemeInvalidation(t *testing.T) {
	app := NewApp(AppOptions{Width: 200, Height: 30})
	app.root = func() View { return ProgressBar(ProgressBarProps{Value: .5}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.running, app.events = true, &appFakeEvents{}
	before := app.Diagnostics()
	paintTheme := LightTheme()
	paintTheme.Semantic.Colors[ColorSemanticProgressFill] = LiteralColor(RGBA(1, 2, 3, 255))
	paintTheme.Semantic.Metrics[MetricComponentProgressBarTrackHeight] = Metric(6)
	if err := app.SetTheme(paintTheme); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("paint theme switch = dirty:%t err:%v", dirty, err)
	}
	afterPaint := app.Diagnostics()
	if afterPaint.LayoutCount != before.LayoutCount || afterPaint.PaintCount != before.PaintCount+1 {
		t.Fatalf("paint theme diagnostics = %+v -> %+v", before, afterPaint)
	}
	layoutTheme := paintTheme
	layoutTheme.Semantic.Metrics[MetricComponentProgressBarWidth] = Metric(180)
	if err := app.SetTheme(layoutTheme); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("layout theme switch = dirty:%t err:%v", dirty, err)
	}
	if after := app.Diagnostics(); after.LayoutCount != afterPaint.LayoutCount+1 {
		t.Fatalf("width token did not relayout: %+v -> %+v", afterPaint, after)
	}
}
