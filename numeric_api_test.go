package dxui

import (
	"math"
	"testing"
)

func TestEdgesMapsTRBLAndDistinguishesExplicitZero(t *testing.T) {
	want := [4]float32{1, 2, 3, 4}
	got := Edges(want[0], want[1], want[2], want[3])
	values := [4]MetricValue{got.Top, got.Right, got.Bottom, got.Left}
	for index, value := range values {
		if !value.set || value.isToken || value.literal != want[index] {
			t.Fatalf("edge %d = %+v, want explicit literal %g", index, value, want[index])
		}
	}

	explicitZero := Edges(0, 0, 0, 0)
	unset := EdgeValues{}
	zeros := [4]MetricValue{explicitZero.Top, explicitZero.Right, explicitZero.Bottom, explicitZero.Left}
	unsets := [4]MetricValue{unset.Top, unset.Right, unset.Bottom, unset.Left}
	for index := range zeros {
		if !zeros[index].set || zeros[index].literal != 0 {
			t.Fatalf("explicit zero edge %d = %+v", index, zeros[index])
		}
		if unsets[index].set {
			t.Fatalf("zero-value edge %d is set: %+v", index, unsets[index])
		}
	}
}

func TestEdgesLeavesInvalidValuesToExistingValidation(t *testing.T) {
	invalid := float32(math.NaN())
	edges := Edges(invalid, 2, 3, 4)
	if !math.IsNaN(float64(edges.Top.literal)) {
		t.Fatalf("Edges silently changed NaN to %g", edges.Top.literal)
	}
	if _, err := describeView(Box(BoxProps{Style: Style{Padding: edges}})); err == nil {
		t.Fatal("existing metric validation accepted NaN padding")
	}
}

func TestCornersMapsClockwiseAndDistinguishesExplicitZero(t *testing.T) {
	want := [4]float32{1, 2, 3, 4}
	got := Corners(want[0], want[1], want[2], want[3])
	values := [4]MetricValue{got.TopLeft, got.TopRight, got.BottomRight, got.BottomLeft}
	for index, value := range values {
		if !value.set || value.isToken || value.literal != want[index] {
			t.Fatalf("corner %d = %+v, want explicit literal %g", index, value, want[index])
		}
	}

	explicitZero := Corners(0, 0, 0, 0)
	unset := CornerValues{}
	zeros := [4]MetricValue{explicitZero.TopLeft, explicitZero.TopRight, explicitZero.BottomRight, explicitZero.BottomLeft}
	unsets := [4]MetricValue{unset.TopLeft, unset.TopRight, unset.BottomRight, unset.BottomLeft}
	for index := range zeros {
		if !zeros[index].set || zeros[index].literal != 0 {
			t.Fatalf("explicit zero corner %d = %+v", index, zeros[index])
		}
		if unsets[index].set {
			t.Fatalf("zero-value corner %d is set: %+v", index, unsets[index])
		}
	}
}

func TestCornersLeavesInvalidValuesToExistingValidation(t *testing.T) {
	invalid := float32(math.NaN())
	corners := Corners(invalid, 2, 3, 4)
	if !math.IsNaN(float64(corners.TopLeft.literal)) {
		t.Fatalf("Corners silently changed NaN to %g", corners.TopLeft.literal)
	}
	if _, err := describeView(Box(BoxProps{Style: Style{Radius: corners}})); err == nil {
		t.Fatal("existing metric validation accepted NaN radius")
	}
}

func TestStrokeCreatesSolidBorderWithLiteralWidth(t *testing.T) {
	color := TokenColor(ColorSemanticAccent)
	got := Stroke(2, color)
	if got.Width != Metric(2) || got.Color != color || got.Pattern != BorderSolid {
		t.Fatalf("Stroke(2, color) = %+v, want a 2-unit solid border", got)
	}

	explicitZero := Stroke(0, color)
	if !explicitZero.Width.set || explicitZero.Width.literal != 0 {
		t.Fatalf("Stroke(0, color) width = %+v, want explicit literal zero", explicitZero.Width)
	}
}

func TestStrokeLeavesInvalidWidthToExistingValidation(t *testing.T) {
	border := Stroke(float32(math.NaN()), LiteralColor(RGBA(1, 2, 3, 255)))
	if !math.IsNaN(float64(border.Width.literal)) {
		t.Fatalf("Stroke silently changed NaN to %g", border.Width.literal)
	}
	if _, err := describeView(Box(BoxProps{Style: Style{Border: border}})); err == nil {
		t.Fatal("existing metric validation accepted NaN border width")
	}
}

func TestPublicTextMetricsAndSizeDefaultsTrackThemeAndLiteralsDoNot(t *testing.T) {
	beforeTheme := LightTheme()
	afterTheme := LightTheme()
	afterTheme.Semantic.Metrics[MetricSemanticTextSize] = Metric(19)
	afterTheme.Semantic.Metrics[MetricSemanticLineHeight] = Metric(25)
	afterTheme.Semantic.Metrics[MetricComponentIconSize] = Metric(27)
	afterTheme.Semantic.Metrics[MetricComponentAvatarSize] = Metric(53)
	before, err := prepareTheme(beforeTheme)
	if err != nil {
		t.Fatal(err)
	}
	after, err := prepareTheme(afterTheme)
	if err != nil {
		t.Fatal(err)
	}

	iconData := IconData{ViewBox: Rect{Width: 1, Height: 1}, Commands: []PathCommand{{Verb: PathMove}}}
	source := avatarTestSource(2, 2)
	tests := []struct {
		name     string
		defaultV View
		explicit View
	}{
		{"text", Text(TextProps{Value: "x"}), Text(TextProps{Style: Style{Text: TextStyle{Size: 14, LineHeight: 18}}, Value: "x"})},
		{"icon", Icon(IconProps{Data: iconData}), Icon(IconProps{Data: iconData, Size: 14})},
		{"avatar", Avatar(AvatarProps{Source: source}), Avatar(AvatarProps{Source: source, Size: 14})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !themeChangesLayout(test.defaultV, before, after) {
				t.Fatal("zero text metric or Size did not retain its theme dependency")
			}
			if themeChangesLayout(test.explicit, before, after) {
				t.Fatal("positive text metric or Size retained a theme dependency")
			}
		})
	}
	defaultText, err := resolvedTextRequest(*Text(TextProps{Value: "x"}).node.text, after)
	if err != nil {
		t.Fatal(err)
	}
	explicitText, err := resolvedTextRequest(*Text(TextProps{Style: Style{Text: TextStyle{Size: 14, LineHeight: 18}}, Value: "x"}).node.text, after)
	if err != nil {
		t.Fatal(err)
	}
	if defaultText.Size != 19 || defaultText.LineHeight != 25 || explicitText.Size != 14 || explicitText.LineHeight != 18 {
		t.Fatalf("text metrics = default %v/%v explicit %v/%v, want 19/25 and 14/18", defaultText.Size, defaultText.LineHeight, explicitText.Size, explicitText.LineHeight)
	}

	if got := intrinsicChildGeometry(t, afterTheme, Icon(IconProps{Data: iconData})); got.Width != 27 || got.Height != 27 {
		t.Fatalf("default icon geometry = %+v, want 27x27", got)
	}
	if got := intrinsicChildGeometry(t, afterTheme, Avatar(AvatarProps{Source: source})); got.Width != 53 || got.Height != 53 {
		t.Fatalf("default avatar geometry = %+v, want 53x53", got)
	}
}

func TestPublicGapZeroAndFixedValues(t *testing.T) {
	for _, test := range []struct {
		gap  float32
		want float32
	}{{0, 0}, {8, 8}} {
		view := Box(BoxProps{Direction: Horizontal, Gap: test.gap},
			Text(TextProps{Style: Style{Width: Px(10), Height: Px(10)}}),
			Text(TextProps{Style: Style{Width: Px(10), Height: Px(10)}}),
		)
		result, _, err := layoutView(view, exactPublic(100, 20), nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		got := result.Children[1].Rect.X - result.Children[0].Rect.Width
		if got != test.want {
			t.Fatalf("Gap %v produced %v, want %v", test.gap, got, test.want)
		}
	}
}

func TestPublicTextMetricsSizeAndGapRejectInvalidScalars(t *testing.T) {
	iconData := IconData{ViewBox: Rect{Width: 1, Height: 1}, Commands: []PathCommand{{Verb: PathMove}}}
	source := avatarTestSource(1, 1)
	for _, invalid := range []float32{-1, float32(math.NaN()), float32(math.Inf(-1)), float32(math.Inf(1))} {
		views := []View{
			Box(BoxProps{Gap: invalid}),
			Text(TextProps{Style: Style{Text: TextStyle{Size: invalid}}, Value: "x"}),
			Text(TextProps{Style: Style{Text: TextStyle{LineHeight: invalid}}, Value: "x"}),
			Icon(IconProps{Data: iconData, Size: invalid}),
			Avatar(AvatarProps{Source: source, Size: invalid}),
		}
		for _, view := range views {
			if _, err := describeView(view); err == nil {
				t.Fatalf("invalid scalar %v accepted for kind %v", invalid, view.node.kind)
			}
		}
	}
}

func TestInvalidLineHeightPreservesCommittedTransaction(t *testing.T) {
	lineHeight := float32(18)
	app := NewApp(AppOptions{Width: 100, Height: 40})
	app.root = func() View {
		return Text(TextProps{Style: Style{Text: TextStyle{Size: 14, LineHeight: lineHeight}}, Value: "stable"})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	oldRoot, oldGeometry := app.retained.Root(), app.geometry
	lineHeight = float32(math.NaN())
	if err := app.buildRoot(); err == nil {
		t.Fatal("NaN line height was accepted")
	}
	if app.retained.Root() != oldRoot || app.geometry != oldGeometry {
		t.Fatal("invalid line height changed the committed tree or geometry")
	}
}
