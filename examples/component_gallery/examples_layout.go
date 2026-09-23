package main

import (
	"fmt"

	"github.com/dxui-org/dxui"
)

func componentRegistry(app *dxui.App, state *galleryState, assets galleryAssets) []componentExample {
	result := layoutExamples(state)
	result = append(result, contentExamples(app, state, assets)...)
	result = append(result, controlExamples(app, state, assets)...)
	return result
}

func layoutExamples(state *galleryState) []componentExample {
	return []componentExample{
		{
			Name: "Box", Category: "Layout", Purpose: "A single-line vertical or horizontal flex container that retains identity when Direction changes.",
			Coverage: []string{"BoxProps: Direction, Style, Token, States, Gap, Justify, Align", "Direction: Vertical default, Horizontal", "Style layout and paint fields", "zero children and variadic custom children", "Grow, Shrink, Basis, AlignSelf", "absolute position, insets, z-index, overflow", "Padding, PaddingXY, Round helpers"},
			Build: func() []exampleBlock {
				return append(verticalBoxBlocks(), horizontalBoxBlocks()...)
			},
		},
		{
			Name: "Scroll", Category: "Layout", Purpose: "A one-child clipped viewport with retained or controlled offsets, keyboard/wheel input, nested propagation, and overlay scrollbars.",
			Coverage: []string{"direct common fields", "Axis: vertical, horizontal, both", "InitialOffset unset/set", "Offset unset/set (controlled)", "Scrollbar: auto, always, hidden", "OnScroll nil/callback and current offset feedback"},
			Build:    func() []exampleBlock { return scrollBlocks(state) },
		},
		{
			Name: "VirtualList", Category: "Layout", Purpose: "A keyed vertical fixed-row list that mounts only its visible window and bounded overscan.",
			Coverage: []string{"100,000 rows", "stable item keys", "fixed RowHeight", "bounded Overscan", "lazy Build"},
			Build: func() []exampleBlock {
				return []exampleBlock{{Title: "Virtualized rows", Description: "Only visible rows plus two rows of overscan are built.", Code: "dxui.VirtualList(dxui.VirtualListProps{...})", Preview: dxui.VirtualList(dxui.VirtualListProps{
					Key: "gallery-virtual", Style: dxui.Style{Height: dxui.Px(220)}, Count: 100_000, Version: 1, RowHeight: 36, Overscan: 2,
					ItemKey: func(index int) string { return fmt.Sprintf("gallery-row-%d", index) },
					Build:   func(index int) dxui.View { return dxui.TextButton(dxui.ButtonProps{}, fmt.Sprintf("Row %d", index)) },
				})}}
			},
		},
	}
}

func verticalBoxBlocks() []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Top-to-bottom layout",
			Description: "Box defaults to Vertical and places children from top to bottom. Gap adds space between adjacent children; AlignStart keeps their different widths visible so the direction is easy to see.",
			Code: `dxui.Box(dxui.BoxProps{
    Style: dxui.Style{Padding: dxui.Padding(12)},
    Gap: 8,
    Align: dxui.AlignStart,
},
    item("1  First", 220),
    item("2  Second", 170),
    item("3  Third", 120),
)`,
			Preview: dxui.Box(dxui.BoxProps{
				Token: dxui.ComponentPanel, Style: dxui.Style{Width: dxui.Px(280), Padding: dxui.Padding(12)},
				Gap: 8, Align: dxui.AlignStart,
			},
				layoutDemoItem("1  First child", 220, 42, dxui.RGBA(14, 165, 233, 255)),
				layoutDemoItem("2  Second child", 170, 42, dxui.RGBA(99, 102, 241, 255)),
				layoutDemoItem("3  Third child", 120, 42, dxui.RGBA(168, 85, 247, 255)),
			),
		},
		{
			Title:       "Justify and size boundaries",
			Description: "The four main-axis values are shown in fixed-height columns. The blue and purple children make the free vertical space and each distribution immediately visible. Width 0 means auto, Fill() fills a definite parent, and min/max clamp the result (a max below min is raised to min).",
			Code: `// Zero Length is auto. Percent accepts 0..100; Percent(0) collapses.
Style{Width: dxui.Fill(), Height: dxui.Px(92),
      MinWidth: dxui.Px(90), MaxWidth: dxui.Px(140),
      MinHeight: dxui.Px(40), MaxHeight: dxui.Px(120)}
Justify: dxui.JustifyStart // Center, End, SpaceBetween`,
			Preview: previewHorizontal(
				justifyVerticalBox("start", dxui.JustifyStart),
				justifyVerticalBox("center", dxui.JustifyCenter),
				justifyVerticalBox("end", dxui.JustifyEnd),
				justifyVerticalBox("between", dxui.JustifySpaceBetween),
			),
		},
		{
			Title:       "Common paint properties and typed state patches",
			Description: "Background, inside border, per-corner radius, up to four outer shadows, opacity, visibility, token overrides, and paint-only state patches belong to every component's direct common fields and Style. Hidden still occupies layout, so the middle gap remains.",
			Code: `dxui.BoxProps{
    Token: dxui.ComponentPanel,
    States: dxui.StateStyles{Default: dxui.StylePatch{
        TextColor: dxui.Some(dxui.TokenColor(dxui.ColorSemanticText)),
    }},
    Style: dxui.Style{
        Background: dxui.ColorRGBA(99, 102, 241, 255),
        Border: dxui.Stroke(2, accent),
        Radius: dxui.Corners(16, 0, 16, 0),
        Shadow: []dxui.Shadow{{OffsetY: dxui.Metric(3), Blur: dxui.Metric(8), Spread: dxui.Metric(1), Color: shadow}},
        Opacity: dxui.Some(float32(.65)), Visibility: dxui.Visible,
    },
}`,
			Preview: previewHorizontal(
				swatch("border + radius", dxui.Style{
					Background: dxui.ColorRGBA(99, 102, 241, 255),
					Border:     dxui.Stroke(2, dxui.ColorRGBA(199, 210, 254, 255)),
					Radius:     dxui.Corners(16, 0, 16, 0),
				}),
				swatch("shadow", dxui.Style{
					Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi), Radius: dxui.Round(8),
					Shadow: []dxui.Shadow{{OffsetY: dxui.Metric(3), Blur: dxui.Metric(8), Spread: dxui.Metric(1), Color: dxui.ColorRGBA(0, 0, 0, 80)}},
				}),
				swatch("opacity .65", dxui.Style{Background: dxui.TokenColor(dxui.ColorSemanticAccent), Opacity: dxui.Some(float32(.65))}),
				dxui.Text(dxui.TextProps{Style: dxui.Style{Width: dxui.Px(20), Visibility: dxui.Hidden}, Value: "hidden"}),
			),
		},
	}
}

func justifyVerticalBox(label string, justify dxui.Justify) dxui.View {
	return dxui.Box(dxui.BoxProps{
		Style: dxui.Style{Width: dxui.Px(112)}, Gap: 4, Align: dxui.AlignStretch,
	},
		smallLabel(label),
		dxui.Box(dxui.BoxProps{
			Style: dxui.Style{
				Height: dxui.Px(140), Padding: dxui.Padding(6),
				Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi),
				Border:     dxui.Stroke(1, dxui.TokenColor(dxui.ColorSemanticBorder)),
				Radius:     dxui.Round(6),
			},
			Gap: 4, Justify: justify, Align: dxui.AlignCenter,
		},
			layoutDemoItem("A", 82, 38, dxui.RGBA(14, 165, 233, 255)),
			layoutDemoItem("B", 82, 38, dxui.RGBA(168, 85, 247, 255)),
		),
	)
}

func horizontalBoxBlocks() []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Left-to-right layout",
			Description: "Box with Direction Horizontal places children from left to right. Gap separates adjacent children; AlignCenter makes the shared main axis clear even when children have different heights.",
			Code: `dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
    Style: dxui.Style{Padding: dxui.Padding(12)},
    Gap: 8,
    Align: dxui.AlignCenter,
},
    item("1  First", 42),
    item("2  Second", 62),
    item("3  Third", 34),
)`,
			Preview: dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
				Token: dxui.ComponentPanel, Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Px(100), Padding: dxui.Padding(12)},
				Gap: 8, Align: dxui.AlignCenter,
			},
				layoutDemoItem("1  First", 110, 42, dxui.RGBA(14, 165, 233, 255)),
				layoutDemoItem("2  Second", 110, 62, dxui.RGBA(99, 102, 241, 255)),
				layoutDemoItem("3  Third", 110, 34, dxui.RGBA(168, 85, 247, 255)),
			),
		},
		{
			Title:       "Flex grow, shrink, basis, margin, and AlignSelf",
			Description: "Grow distributes positive space. Shrink defaults to 1; NoShrink() explicitly prevents shrinking. Basis is auto by default. AlignSelf overrides the container cross alignment.",
			Code: `dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8, Align: dxui.AlignStretch},
    item(dxui.Style{Basis: dxui.Px(80), Shrink: dxui.NoShrink()}),
    item(dxui.Style{Grow: 1, Basis: dxui.Px(40)}),
    item(dxui.Style{Grow: 2, Basis: dxui.Percent(20),
        Margin: dxui.EdgeValues{Left: dxui.Metric(4)},
        AlignSelf: dxui.Some(dxui.AlignCenter)}),
)`,
			Preview: dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
				Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Px(86)}, Gap: 8, Align: dxui.AlignStretch,
			},
				swatch("fixed 80", dxui.Style{Width: dxui.Px(80), Shrink: dxui.NoShrink(), Background: dxui.ColorRGBA(14, 165, 233, 255)}),
				swatch("grow 1", dxui.Style{Grow: 1, Basis: dxui.Px(40), Background: dxui.ColorRGBA(99, 102, 241, 255)}),
				swatch("grow 2", dxui.Style{Grow: 2, Basis: dxui.Percent(20), Margin: dxui.EdgeValues{Left: dxui.Metric(4)}, AlignSelf: dxui.Some(dxui.AlignCenter), Background: dxui.ColorRGBA(168, 85, 247, 255)}),
			),
		},
		{
			Title:       "All cross-axis and main-axis enums",
			Description: "Each outlined row contains blue and purple children, making their position against the available space visible. AlignStart, Center, End, and Stretch control the cross axis; JustifyStart, Center, End, and SpaceBetween control the main axis.",
			Code: `BoxProps{
    Gap: 8,
    Justify: dxui.JustifySpaceBetween,
    Align: dxui.AlignCenter,
}`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 6},
				alignmentHorizontalBox("start", dxui.AlignStart, dxui.JustifyStart),
				alignmentHorizontalBox("center", dxui.AlignCenter, dxui.JustifyCenter),
				alignmentHorizontalBox("end", dxui.AlignEnd, dxui.JustifyEnd),
				alignmentHorizontalBox("stretch / between", dxui.AlignStretch, dxui.JustifySpaceBetween),
			),
		},
		{
			Title:       "Absolute positioning, insets, stacking, and clipping",
			Description: "Absolute children leave flex flow. Insets accept auto (zero Length), Px, or Percent. ZIndex changes paint/hit order only. OverflowVisible is default; OverflowClip clips descendants to the rectangular bound.",
			Code: `Style{
    Position: dxui.PositionAbsolute,
    Insets: dxui.Insets{Top: dxui.Px(18), Left: dxui.Percent(30)},
    Width: dxui.Px(150), Height: dxui.Px(50), ZIndex: 2,
}
// Parent: Overflow: dxui.OverflowClip`,
			Preview: dxui.Box(dxui.BoxProps{Style: dxui.Style{
				Width: dxui.Px(360), Height: dxui.Px(110), Overflow: dxui.OverflowClip,
				Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi),
			}},
				swatch("flow z=0", dxui.Style{Background: dxui.ColorRGBA(14, 165, 233, 255)}),
				dxui.Text(dxui.TextProps{Style: dxui.Style{
					Position: dxui.PositionAbsolute, Insets: dxui.Insets{Top: dxui.Px(28), Left: dxui.Percent(30)},
					Width: dxui.Px(210), Height: dxui.Px(62), ZIndex: 2, Padding: dxui.Padding(8),
					Background: dxui.ColorRGBA(168, 85, 247, 230),
				}, Value: "absolute z=2, clipped at parent"}),
			),
		},
	}
}

func layoutDemoItem(label string, width, height float32, background dxui.RGBAColor) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{
		Width: dxui.Px(width), Height: dxui.Px(height), Shrink: dxui.NoShrink(),
		Padding: dxui.PaddingXY(10, 8), Background: dxui.LiteralColor(background),
		Radius: dxui.Round(6), Text: dxui.TextStyle{Weight: dxui.WeightBold, Color: dxui.ColorRGBA(255, 255, 255, 255)},
	}, Value: label})
}

func alignmentHorizontalBox(label string, align dxui.Align, justify dxui.Justify) dxui.View {
	itemHeight := float32(26)
	if align == dxui.AlignStretch {
		itemHeight = 0
	}
	return dxui.Box(dxui.BoxProps{Gap: 3, Align: dxui.AlignStretch},
		smallLabel(label),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
			Style: dxui.Style{
				Width: dxui.Fill(), Height: dxui.Px(58), Padding: dxui.Padding(5),
				Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi),
				Border:     dxui.Stroke(1, dxui.TokenColor(dxui.ColorSemanticBorder)),
				Radius:     dxui.Round(6),
			},
			Gap: 8, Align: align, Justify: justify,
		},
			alignmentDemoItem("A", itemHeight, dxui.RGBA(14, 165, 233, 255)),
			alignmentDemoItem("B", itemHeight, dxui.RGBA(168, 85, 247, 255)),
		),
	)
}

func alignmentDemoItem(label string, height float32, background dxui.RGBAColor) dxui.View {
	style := dxui.Style{
		Width: dxui.Px(72), Shrink: dxui.NoShrink(), Padding: dxui.PaddingXY(8, 4),
		Background: dxui.LiteralColor(background), Radius: dxui.Round(4),
		Text: dxui.TextStyle{
			Size: 12, LineHeight: 16, Weight: dxui.WeightBold,
			Color: dxui.ColorRGBA(255, 255, 255, 255),
		},
	}
	if height > 0 {
		style.Height = dxui.Px(height)
	}
	return dxui.Text(dxui.TextProps{Style: style, Value: label})
}

func scrollBlocks(state *galleryState) []exampleBlock {
	rows := make([]dxui.View, 12)
	for index := range rows {
		rows[index] = dxui.Text(dxui.TextProps{Key: fmt.Sprintf("vertical-%d", index), Style: dxui.Style{Height: dxui.Px(28), Shrink: dxui.NoShrink()}, Value: fmt.Sprintf("Vertical row %02d", index+1)})
	}
	cards := make([]dxui.View, 8)
	for index := range cards {
		cards[index] = swatch(fmt.Sprintf("card %d", index+1), dxui.Style{Background: dxui.TokenColor(dxui.ColorSemanticAccent)})
	}
	both := dxui.Box(dxui.BoxProps{Style: dxui.Style{Width: dxui.Px(520)}, Gap: 5}, rows...)
	return []exampleBlock{
		{
			Title:       "Axes and retained InitialOffset",
			Description: "Vertical is the zero/default axis. InitialOffset applies once on mount; later wheel, arrow, Home/End, and thumb movement is retained by identity. Horizontal accepts Shift-wheel and vertical wheel when no horizontal delta exists.",
			Code: `dxui.Scroll(dxui.ScrollProps{
    Key: "list", Style: dxui.Style{Height: dxui.Px(130)},
    Axis: dxui.ScrollVertical,
    InitialOffset: dxui.Some(dxui.Point{Y: 24}),
    Scrollbar: dxui.ScrollbarAuto,
}, content)`,
			Preview: previewHorizontal(
				dxui.Scroll(dxui.ScrollProps{Key: "vertical-demo", Style: dxui.Style{Width: dxui.Px(210), Height: dxui.Px(130)}, Axis: dxui.ScrollVertical, InitialOffset: dxui.Some(dxui.Point{Y: 24}), Scrollbar: dxui.ScrollbarAuto}, dxui.Box(dxui.BoxProps{}, rows...)),
				dxui.Scroll(dxui.ScrollProps{Key: "horizontal-demo", Style: dxui.Style{Grow: 1, Height: dxui.Px(90)}, Axis: dxui.ScrollHorizontal, Scrollbar: dxui.ScrollbarAlways}, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8}, cards...)),
			),
		},
		{
			Title:       "Both axes and scrollbar policies",
			Description: "ScrollBoth enables X and Y. Auto appears only with overflow, Always remains visible (disabled when there is no range), and Hidden suppresses track/thumb while wheel and keyboard scrolling continue.",
			Code: `Axis: dxui.ScrollBoth
Scrollbar: dxui.ScrollbarHidden // Auto and Always are also supported`,
			Preview: previewHorizontal(
				dxui.Scroll(dxui.ScrollProps{Key: "both-auto", Style: dxui.Style{Width: dxui.Px(250), Height: dxui.Px(120)}, Axis: dxui.ScrollBoth, Scrollbar: dxui.ScrollbarAuto}, both),
				dxui.Scroll(dxui.ScrollProps{Key: "both-hidden", Style: dxui.Style{Width: dxui.Px(250), Height: dxui.Px(120)}, Axis: dxui.ScrollBoth, Scrollbar: dxui.ScrollbarHidden}, both),
			),
		},
		{
			Title:       "Controlled Offset and OnScroll",
			Description: "When Offset is set it is authoritative. OnScroll proposes a complete clamped Point; this demo accepts it and displays the current value. A nil callback is valid. Zero is the lower boundary and oversized offsets clamp to the content extent.",
			Code: `dxui.Scroll(dxui.ScrollProps{
    Offset: dxui.Some(offset),
    OnScroll: dxui.Assign(&offset),
    Axis: dxui.ScrollVertical,
}, content)`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				bodyText(fmt.Sprintf("Current controlled offset: X=%.1f, Y=%.1f", state.controlledScroll.X, state.controlledScroll.Y)),
				dxui.Scroll(dxui.ScrollProps{
					Key: "controlled-scroll", Style: dxui.Style{Height: dxui.Px(120)}, Axis: dxui.ScrollVertical,
					Offset: dxui.Some(state.controlledScroll), Scrollbar: dxui.ScrollbarAlways,
					OnScroll: func(next dxui.Point) {
						state.controlledScroll = next
						state.feedback = fmt.Sprintf("OnScroll -> {X: %.1f, Y: %.1f}", next.X, next.Y)
					},
				}, dxui.Box(dxui.BoxProps{}, rows...)),
			),
		},
	}
}
