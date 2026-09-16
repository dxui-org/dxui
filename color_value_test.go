package dxui

import "testing"

var colorValueSink ColorValue

func TestColorRGBAExplicitZeroAndThemeResolution(t *testing.T) {
	for _, raw := range []RGBAColor{RGBA(0, 0, 0, 0), RGBA(1, 2, 3, 4), RGBA(255, 255, 255, 255)} {
		color := ColorRGBA(raw.R, raw.G, raw.B, raw.A)
		if color != LiteralColor(raw) || color == (ColorValue{}) {
			t.Fatal("literal constructor lost value or explicit set")
		}
		theme := LightTheme()
		theme.Semantic.Colors[ColorSemanticAccent] = color
		app := NewApp(AppOptions{Theme: theme})
		app.root = func() View {
			return Box(BoxProps{Style: Style{Width: Px(10), Height: Px(10), Background: TokenColor(ColorSemanticAccent)}})
		}
		if err := app.buildRoot(); err != nil {
			t.Fatal(err)
		}
		resolved, err := app.theme.color(TokenColor(ColorSemanticAccent))
		if err != nil || resolved != raw {
			t.Fatalf("resolved=%v err=%v", resolved, err)
		}
	}
	if allocs := testing.AllocsPerRun(100, func() { colorValueSink = ColorRGBA(1, 2, 3, 4) }); allocs != 0 {
		t.Fatalf("constructor allocations=%v", allocs)
	}
}

func BenchmarkColorRGBA(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		colorValueSink = ColorRGBA(1, 2, 3, 4)
	}
}
