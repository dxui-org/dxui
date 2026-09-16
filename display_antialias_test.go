package dxui

import (
	"math"
	"testing"

	"github.com/dxui-org/dxui/internal/layout"
)

func TestIconRasterizationUsesPhysicalScaleAndCoverage(t *testing.T) {
	icon := IconProps{
		Size: 17,
		Data: IconData{
			ViewBox: Rect{Width: 17, Height: 17},
			Commands: []PathCommand{
				{Verb: PathMove, Points: [3]Point{{X: 1, Y: 1}}},
				{Verb: PathLine, Points: [3]Point{{X: 16, Y: 8}}},
				{Verb: PathLine, Points: [3]Point{{X: 1, Y: 16}}},
				{Verb: PathClose},
			},
		},
	}
	var previousKey string
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		bitmap, rect, err := rasterIcon(icon, layout.Rect{Width: 17, Height: 17}, resolvedTheme{}, scale, scale)
		if err != nil {
			t.Fatalf("scale %g: %v", scale, err)
		}
		wantPixels := int(math.Ceil(float64(17 * scale)))
		if bitmap.Width != wantPixels || bitmap.Height != wantPixels || rect.Width != 17 || rect.Height != 17 {
			t.Fatalf("scale %g bitmap=%dx%d rect=%+v want %dx%d logical 17x17", scale, bitmap.Width, bitmap.Height, rect, wantPixels, wantPixels)
		}
		partial := 0
		for _, alpha := range bitmap.Pixels {
			if alpha > 0 && alpha < 255 {
				partial++
			}
		}
		if partial == 0 {
			t.Fatalf("scale %g icon has no coverage pixels", scale)
		}
		if previousKey == bitmap.Key {
			t.Fatalf("scale %g reused preceding scale key %q", scale, bitmap.Key)
		}
		previousKey = bitmap.Key
	}
}
