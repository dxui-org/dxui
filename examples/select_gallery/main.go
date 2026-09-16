package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/dxui-org/dxui"
)

func main() {
	software := flag.Bool("software", false, "force SDL's named software renderer")
	flag.Parse()
	renderer := dxui.RendererAuto
	if *software {
		renderer = dxui.RendererSoftware
	}
	app := dxui.NewApp(dxui.AppOptions{Title: "dxui Select gallery", Width: 720, Height: 620, Renderer: renderer})

	value := "green"
	dynamic := []dxui.SelectOption{
		{Value: "red", Label: "Red"},
		{Value: "green", Label: "Green"},
		{Value: "blue", Label: "Blue", Disabled: true},
		{Value: "violet", Label: "Violet"},
	}
	makeSelect := func(key string, selected string, options []dxui.SelectOption, change func(string)) dxui.View {
		return dxui.Select(dxui.SelectProps{
			Key: key, Style: dxui.Style{Width: dxui.Px(220), Height: dxui.Px(40), Shrink: dxui.NoShrink()},
			Value: selected, Options: options, Placeholder: "Choose an option", OnChange: change,
		})
	}

	root := func() dxui.View {
		rows := make([]dxui.View, 18)
		for index := range rows {
			if index == 2 {
				rows[index] = makeSelect("inside-scroll", value, dynamic, func(change string) { value = change })
				continue
			}
			rows[index] = dxui.Text(dxui.TextProps{
				Key: (fmt.Sprintf("row-%d", index)), Style: dxui.Style{Height: dxui.Px(30), Shrink: dxui.NoShrink()},
				Value: fmt.Sprintf("Scrollable row %02d", index+1),
			})
		}
		removeLabel := "Remove current option"
		if len(dynamic) == 0 {
			removeLabel = "Restore options"
		}
		return dxui.Box(dxui.BoxProps{
			Style: dxui.Style{Padding: dxui.Padding(18)},
			Gap:   12,
		},
			dxui.Text(dxui.TextProps{Value: "Select gallery — Tab focus; Enter/Space open; arrows/Home/End move; Enter chooses; Escape closes."}),
			makeSelect("top-edge", value, dynamic, func(change string) { value = change }),
			dxui.Text(dxui.TextProps{Value: "The next Select lives inside Scroll; its popup escapes the content clip."}),
			dxui.Scroll(dxui.ScrollProps{
				Key: "scroll", Style: sizedStyle(250), Axis: dxui.ScrollVertical,
			}, dxui.Box(dxui.BoxProps{Gap: 4}, rows...)),
			dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 10},
				dxui.Button(dxui.ButtonProps{OnPress: func() {
					if len(dynamic) == 0 {
						dynamic = []dxui.SelectOption{{Value: "red", Label: "Red"}, {Value: "green", Label: "Green"}, {Value: "blue", Label: "Blue", Disabled: true}, {Value: "violet", Label: "Violet"}}
						return
					}
					filtered := dynamic[:0]
					for _, option := range dynamic {
						if option.Value != value {
							filtered = append(filtered, option)
						}
					}
					dynamic = filtered
				}}, dxui.Text(dxui.TextProps{Value: removeLabel})),
				dxui.Button(dxui.ButtonProps{OnPress: func() {
					if len(dynamic) > 0 {
						dynamic = dynamic[:len(dynamic)-1]
					}
				}}, dxui.Text(dxui.TextProps{Value: "Remove last"})),
			),
			dxui.Text(dxui.TextProps{Value: "Bottom-edge placement flips above when space below is insufficient."}),
			makeSelect("bottom-edge", value, dynamic, func(change string) { value = change }),
		)
	}
	if err := app.Run(root); err != nil {
		log.Fatal(err)
	}
}

func sizedStyle(height float32) dxui.Style {
	return dxui.Style{Height: dxui.Px(height), Shrink: dxui.NoShrink()}
}
