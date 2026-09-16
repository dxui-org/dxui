// Command layout_gallery demonstrates the exact ADR-0005 layout subset using
// only the root dxui API. It deliberately does not demonstrate flex wrapping,
// order, baseline alignment, or aspect ratio because those are unsupported.
package main

import (
	"log"

	"github.com/dxui-org/dxui"
)

func main() {
	app := dxui.NewApp(dxui.AppOptions{
		Title: "dxui layout gallery", Width: 760, Height: 520,
		Background: dxui.RGBA(20, 23, 30, 255),
	})
	if err := app.Run(func() dxui.View {
		return gallery()
	}); err != nil {
		log.Fatal(err)
	}
}

func gallery() dxui.View {
	card := func(color dxui.RGBAColor, child dxui.View) dxui.View {
		return dxui.Box(dxui.BoxProps{
			Style: dxui.Style{

				Padding: dxui.Padding(12),
				Grow:    1, Basis: dxui.Px(180),

				Background: dxui.LiteralColor(color),
			},
		}, child)
	}

	fixedAndFlexible := card(dxui.RGBA(39, 47, 62, 255),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8, Align: dxui.AlignStretch},
			box("fixed 72", 72, dxui.RGBA(64, 125, 201, 255)),
			dxui.Text(dxui.TextProps{Style: dxui.Style{
				Grow: 1, Basis: dxui.Px(40),
			}, Value: "grow: 1"}),
			dxui.Text(dxui.TextProps{Style: dxui.Style{
				Grow: 2, Basis: dxui.Px(40),
			}, Value: "grow: 2"}),
		),
	)

	centered := card(dxui.RGBA(43, 54, 46, 255),
		dxui.Box(dxui.BoxProps{Justify: dxui.JustifyCenter, Align: dxui.AlignCenter},
			box("center", 96, dxui.RGBA(76, 170, 120, 255)),
		),
	)

	overlap := card(dxui.RGBA(58, 43, 55, 255),
		dxui.Box(dxui.BoxProps{Style: dxui.Style{Height: dxui.Px(120)}},
			box("flow", 130, dxui.RGBA(166, 83, 125, 255)),
			dxui.Text(dxui.TextProps{Style: dxui.Style{

				Width: dxui.Px(110), Height: dxui.Px(44), Position: dxui.PositionAbsolute,
				Insets: dxui.Insets{Top: dxui.Px(28), Left: dxui.Px(42)}, ZIndex: 2,

				Background: dxui.ColorRGBA(232, 163, 70, 255),
			}, Value: "absolute z=2"}),
		),
	)

	nestedClip := card(dxui.RGBA(48, 45, 68, 255),
		dxui.Box(dxui.BoxProps{Style: dxui.Style{
			Width: dxui.Px(160), Height: dxui.Px(96), Overflow: dxui.OverflowClip,
		}},
			dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Style: dxui.Style{
				Width: dxui.Px(220), Height: dxui.Px(60), Margin: dxui.EdgeValues{Top: dxui.Metric(20), Left: dxui.Metric(50)}, Overflow: dxui.OverflowClip,
			}}, box("nested clip", 180, dxui.RGBA(123, 105, 204, 255))),
		),
	)

	return dxui.Box(dxui.BoxProps{
		Style: dxui.Style{Padding: dxui.Padding(20)},
		Gap:   16,
	},
		dxui.Text(dxui.TextProps{Value: "ADR-0005 deterministic layout gallery"}),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 16, Align: dxui.AlignStretch}, fixedAndFlexible, centered),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 16, Align: dxui.AlignStretch}, overlap, nestedClip),
	)
}

func box(label string, width float32, color dxui.RGBAColor) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{
		Width: dxui.Px(width), Height: dxui.Px(44),
		Background: dxui.LiteralColor(color),
	}, Value: label})
}
