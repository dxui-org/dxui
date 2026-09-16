package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/layout"
)

func TestPublicBoxAndStyleProduceGeometry(t *testing.T) {
	view := Box(BoxProps{
		Style: Style{Padding: UniformEdges(Metric(10))},
		Gap:   5, Align: AlignStretch,
	},
		Text(TextProps{Style: Style{
			Width: Px(20), Height: Px(10), AlignSelf: Some(AlignCenter),
		}, Value: "fixed"}),
		Box(BoxProps{Direction: Horizontal, Style: Style{
			Height: Px(20),
		}},
			Text(TextProps{Style: Style{Grow: 1}, Value: "left"}),
			Text(TextProps{Style: Style{Grow: 3}, Value: "right"}),
		),
	)
	result, _, err := layoutView(view, exactPublic(100, 100), nil, func(View) layout.IntrinsicMeasurer {
		return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
			return layout.IntrinsicSize{Preferred: layout.Size{Width: 10, Height: 10}}, nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Children[0].Rect; got.X != 40 || got.Y != 10 || got.Width != 20 || got.Height != 10 {
		t.Fatalf("fixed child geometry = %#v", got)
	}
	row := result.Children[1]
	if row.Rect.X != 10 || row.Rect.Y != 25 || row.Rect.Width != 80 || row.Rect.Height != 20 {
		t.Fatalf("row geometry = %#v", row.Rect)
	}
	if row.Children[0].Rect.Width != 25 || row.Children[1].Rect.Width != 55 {
		t.Fatalf("nested grow widths = %v/%v", row.Children[0].Rect.Width, row.Children[1].Rect.Width)
	}
}

func TestPublicGapLiteral(t *testing.T) {
	view := Box(BoxProps{Direction: Horizontal, Gap: 8},
		Text(TextProps{Style: Style{Width: Px(10), Height: Px(10)}}),
		Text(TextProps{Style: Style{Width: Px(10), Height: Px(10)}}),
	)
	result, _, err := layoutView(view, exactPublic(100, 20), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Children[1].Rect.X - (result.Children[0].Rect.X + result.Children[0].Rect.Width); got != 8 {
		t.Fatalf("gap = %v, want 8", got)
	}
}

func exactPublic(width, height float32) layout.Constraints {
	return layout.Constraints{
		Width:  layout.Limit{Min: width, Max: width, MaxSet: true, Definite: true},
		Height: layout.Limit{Min: height, Max: height, MaxSet: true, Definite: true},
	}
}
