package main

import (
	"fmt"
	"log"

	"github.com/dxui-org/dxui"
)

func label(value string, height float32) dxui.View {
	return dxui.Text(dxui.TextProps{
		Style: dxui.Style{
			Height: heightLength(height), Shrink: dxui.NoShrink(),
		},
		Value: value,
	})
}

func heightLength(value float32) dxui.Length { return dxui.Px(value) }

func main() {
	app := dxui.NewApp(dxui.AppOptions{Title: "dxui scroll gallery", Width: 760, Height: 680})
	wide := false
	notes := "Textarea keeps its own caret scrolling while the surrounding Scroll handles the gallery.\nResize the native window, use a wheel or trackpad, hold Shift for horizontal scrolling, and drag the overlay thumbs."

	root := func() dxui.View {
		listCount := 40
		cardWidth := float32(180)
		if wide {
			listCount, cardWidth = 65, 260
		}
		items := make([]dxui.View, listCount)
		for index := range items {
			items[index] = label(fmt.Sprintf("Long list row %02d", index+1), 28)
		}
		cards := make([]dxui.View, 10)
		for index := range cards {
			cards[index] = dxui.Box(dxui.BoxProps{Style: dxui.Style{
				Width: dxui.Px(cardWidth), Height: dxui.Px(64), Shrink: dxui.NoShrink(),
				Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi),
			}}, label(fmt.Sprintf("Horizontal card %d", index+1), 28))
		}
		nestedItems := make([]dxui.View, 18)
		for index := range nestedItems {
			nestedItems[index] = label(fmt.Sprintf("Inner row %02d", index+1), 24)
		}

		content := dxui.Box(dxui.BoxProps{
			Style: dxui.Style{Padding: dxui.Padding(18)},
			Gap:   14,
		},
			label("Scroll gallery", 34),
			dxui.Button(dxui.ButtonProps{OnPress: func() { wide = !wide }}, label("Toggle dynamic content size", 24)),
			label("Vertical list", 26),
			dxui.Scroll(dxui.ScrollProps{
				Key: "long-list", Style: dxui.Style{Height: dxui.Px(220)},
				Axis: dxui.ScrollVertical, Scrollbar: dxui.ScrollbarAuto,
			}, dxui.Box(dxui.BoxProps{}, items...)),
			label("Horizontal / Shift-wheel", 26),
			dxui.Scroll(dxui.ScrollProps{
				Key: "horizontal", Style: dxui.Style{Height: dxui.Px(90)},
				Axis: dxui.ScrollHorizontal, Scrollbar: dxui.ScrollbarAlways,
			}, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 10}, cards...)),
			label("Nested boundary propagation", 26),
			dxui.Scroll(dxui.ScrollProps{
				Key: "outer", Style: dxui.Style{Height: dxui.Px(180)},
				Axis: dxui.ScrollVertical,
			}, dxui.Box(dxui.BoxProps{Gap: 8},
				label("Outer header", 38),
				dxui.Scroll(dxui.ScrollProps{
					Key: "inner", Style: dxui.Style{Height: dxui.Px(100), Shrink: dxui.NoShrink()},
					Axis: dxui.ScrollVertical,
				}, dxui.Box(dxui.BoxProps{}, nestedItems...)),
				label("Outer tail: continue scrolling here after the inner boundary", 180),
			)),
			label("Text area", 26),
			dxui.Textarea(dxui.TextareaProps{
				Style: dxui.Style{Height: dxui.Px(130)},
				Value: notes, Wrap: dxui.TextWrapWords,
				OnChange: func(value string) { notes = value },
			}),
			label("Resize the window: viewport and content clamps update in the same layout commit.", 44),
		)
		return dxui.Scroll(dxui.ScrollProps{Key: "gallery", Axis: dxui.ScrollVertical}, content)
	}

	if err := app.Run(root); err != nil {
		log.Fatal(err)
	}
}
