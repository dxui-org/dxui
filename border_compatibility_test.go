package dxui

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
)

var borderCompatibilityCases = []struct {
	name   string
	width  float32
	radius CornerValues
	radii  paint.Radii
	want   string
}{
	{"square", 2, Round(0), paint.Radii{}, "ae0c74cd0b22173fff78dcbc2b595463169c58b0c08baad9d0b627145969a794"},
	{"rounded", 2, Round(12), paint.Radii{TopLeft: 12, TopRight: 12, BottomRight: 12, BottomLeft: 12}, "680772afc612ef5b9b49cf6c0cda034a7c64920bccb8d74aae59009a0575c714"},
	{"asymmetric", 2, Corners(2, 8, 16, 4), paint.Radii{TopLeft: 2, TopRight: 8, BottomRight: 16, BottomLeft: 4}, "8351055a22da1994ab8557731209ebdfd8dc03948193ae38942e8ac675a7e79b"},
	{"clamped", 2, Round(100), paint.Radii{TopLeft: 100, TopRight: 100, BottomRight: 100, BottomLeft: 100}, "8c950fab1737da6fbfe465227ee615349bf657c3e4004ef85638ac54eb446204"},
	{"thin", .5, Round(12), paint.Radii{TopLeft: 12, TopRight: 12, BottomRight: 12, BottomLeft: 12}, "badf0067edcfbc275c2c2270ff3f1809c25be6501df1abc85882114945f11d09"},
	{"thick", 6, Round(12), paint.Radii{TopLeft: 12, TopRight: 12, BottomRight: 12, BottomLeft: 12}, "603abad10c972d205f552da30fe74271689565a4d8e00bf798dc91c9cddb9659"},
}

func borderFingerprint(list paint.DisplayList) string {
	h := sha256.New()
	for _, c := range list {
		fmt.Fprintf(h, "%d/%v/%v/%g/%v;", c.Kind, c.Rect, c.Radii, c.Width, c.Color)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Hashes were generated from the original HEAD dash expansion before adding
// Sides. They lock its geometry, caps, corner dots, color, and command order.
func TestCompleteDashedBorderPreservesOriginalCommands(t *testing.T) {
	for _, tc := range borderCompatibilityCases {
		for _, sides := range []BorderSides{0, BorderAll} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, sides), func(t *testing.T) {
				color := ColorRGBA(17, 83, 149, 128)
				app := NewApp(AppOptions{Width: 160, Height: 44})
				app.root = func() View {
					return Box(BoxProps{Style: Style{Background: ColorRGBA(1, 2, 3, 255), Radius: tc.radius, Border: Border{Width: Metric(tc.width), Color: color, Pattern: BorderDashed, Sides: sides}}})
				}
				if err := app.buildRoot(); err != nil {
					t.Fatal(err)
				}
				var border paint.DisplayList
				for _, c := range app.display {
					if c.Kind == paint.CommandBorder {
						t.Fatal("complete border entered partial mesh")
					}
					if c.Color == (paint.Color{R: 17, G: 83, B: 149, A: 128}) {
						border = append(border, c)
					}
				}
				if got := borderFingerprint(border); got != tc.want {
					t.Fatalf("legacy commands changed: got %s want %s", got, tc.want)
				}
			})
		}
	}
}

func TestCompleteSolidBorderPreservesOriginalCommands(t *testing.T) {
	for _, tc := range borderCompatibilityCases {
		for _, sides := range []BorderSides{0, BorderAll} {
			for _, background := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/fill=%t", tc.name, sides, background), func(t *testing.T) {
					style := Style{Radius: tc.radius, Border: Border{Width: Metric(tc.width), Color: ColorRGBA(17, 83, 149, 128), Sides: sides}}
					if background {
						style.Background = ColorRGBA(1, 2, 3, 192)
					}
					app := NewApp(AppOptions{Width: 160, Height: 44})
					app.root = func() View { return Box(BoxProps{Style: style}) }
					if err := app.buildRoot(); err != nil {
						t.Fatal(err)
					}
					want := paint.Command{Kind: paint.CommandStrokeRoundedRect, NodeID: app.retained.Root().ID,
						Rect: paint.Rect{Width: 160, Height: 44}, Radii: tc.radii, Width: tc.width, Color: paint.Color{R: 17, G: 83, B: 149, A: 128}}
					if background {
						want.Kind = paint.CommandFillStrokeRoundedRect
						want.BorderColor = want.Color
						want.Color = paint.Color{R: 1, G: 2, B: 3, A: 192}
					}
					var draws paint.DisplayList
					for _, c := range app.display {
						if c.Kind != paint.CommandBeginDisplayList && c.Kind != paint.CommandItem {
							draws = append(draws, c)
						}
					}
					if len(draws) != 1 || draws[0] != want {
						t.Fatalf("original solid command changed: %v, want %v", draws, want)
					}
				})
			}
		}
	}
}
