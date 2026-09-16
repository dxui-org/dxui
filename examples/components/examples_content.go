package main

import (
	"fmt"
	"strings"

	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon"
	"github.com/dxui-org/dxui/icon/catalog"
)

const (
	iconCatalogPageSize = 16 * 16
	iconCatalogColumns  = 16
	iconCatalogCellSize = 40
	iconCatalogGap      = 4
	iconCatalogWidth    = iconCatalogColumns*iconCatalogCellSize + (iconCatalogColumns-1)*iconCatalogGap
)

func contentExamples(app *dxui.App, state *galleryState, assets galleryAssets) []componentExample {
	return []componentExample{
		{
			Name: "Colors", Category: "Foundations", Purpose: "Presents the complete typed Primitive color palette available for literal design choices and custom theme construction.",
			Coverage: []string{"Color.Primitive typed namespace", "26 color families", "shades 50 through 950", "fixed RGBA token values"},
			Build:    colorBlocks,
		},
		{
			Name: "Theme", Category: "Foundations", Purpose: "Demonstrates semantic color tokens and their live resolution through the gallery's Light and Dark themes.",
			Coverage: []string{"Color.Semantic typed namespace", "surface, text, accent, status, and border tokens", "Light/Dark replacement", "custom ComponentTheme", "local Style precedence"},
			Build:    func() []exampleBlock { return themeBlocks(app, state) },
		},
		{
			Name: "Text", Category: "Content", Purpose: "Renders UTF-8 simple LTR/CJK text with intrinsic measurement, explicit fallback, word wrapping, line limits, and typed typography.",
			Coverage: []string{"direct common fields", "Value empty/non-empty/UTF-8", "Wrap: no-wrap/words", "MaxLines: 0/unlimited and positive limit", "Families, Size, LineHeight, Weight, Slant, Color, Align"},
			Build:    textBlocks,
		},
		{
			Name: "Icon", Category: "Content", Purpose: "Renders validated dxui vector path data in a logical ViewBox; it is deliberately not an SVG parser or interactive control.",
			Coverage: []string{"all generated Lucide names and aliases", "search and pagination", "direct common fields", "immutable Lucide resources", "Size zero-default/fixed", "Color default/token/literal", "StrokeWidth default/literal"},
			Build:    func() []exampleBlock { return iconBlocks(app, state, assets) },
		},
		{
			Name: "Image", Category: "Content", Purpose: "Displays guarded PNG, JPEG, static GIF, or Go image sources with four fit modes, normalized alignment, decode limits, and load/error feedback.",
			Coverage: []string{"direct common fields", "Source: ImageBytes/ImageFile/ImageFromGo", "Fit: contain/cover/fill/none", "Alignment: 0/.5/1 boundaries", "MaxPixels default/limited", "OnLoad and OnError"},
			Build:    func() []exampleBlock { return imageBlocks(state, assets) },
		},
		{
			Name: "Avatar", Category: "Content", Purpose: "Displays a theme-sized or custom-sized square image with centered cover fitting and circular or square clipping.",
			Coverage: []string{"direct common fields", "Source: shared ImageSource", "Shape: circle/square", "Size: zero-default/fixed", "centered ImageCover", "OnLoad and OnError", "Avatar + Badge profile composition"},
			Build:    func() []exampleBlock { return avatarBlocks(state, assets) },
		},
		{
			Name: "Badge", Category: "Content", Purpose: "Wraps one arbitrary child in a compact, non-interactive, theme-styled pill for short labels, counts, or icons.",
			Coverage: []string{"direct common fields", "one arbitrary child", "text/icon tint inheritance and local override", "theme padding/minimum height/radius", "intrinsic size and Flex composition", "non-interactive"},
			Build:    func() []exampleBlock { return badgeBlocks(assets) },
		},
		{
			Name: "ProgressBar", Category: "Content", Purpose: "Displays a deterministic completion ratio from left to right without focus, input handling, animation, or an indeterminate mode.",
			Coverage: []string{"direct common fields", "Value: 0, .25, .5, 1", "clamping below/above range", "NaN and infinities", "theme width/height/track/radius/colors", "custom size, padding, border, radius, opacity, and colors", "non-interactive"},
			Build:    progressBarBlocks,
		},
	}
}

func progressBarBlocks() []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Determinate boundary values",
			Description: "Value is a completion ratio. Zero draws only the track, one covers the complete track content area, and intermediate values fill left to right. Values outside 0..1 are clamped.",
			Code: `dxui.ProgressBar(dxui.ProgressBarProps{Value: 0})
dxui.ProgressBar(dxui.ProgressBarProps{Value: .25})
dxui.ProgressBar(dxui.ProgressBarProps{Value: .5})
dxui.ProgressBar(dxui.ProgressBarProps{Value: 1})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 10, Align: dxui.AlignStart},
				dxui.ProgressBar(dxui.ProgressBarProps{Value: 0}),
				dxui.ProgressBar(dxui.ProgressBarProps{Value: .25}),
				dxui.ProgressBar(dxui.ProgressBarProps{Value: .5}),
				dxui.ProgressBar(dxui.ProgressBarProps{Value: 1}),
			),
		},
		{
			Title:       "Custom style",
			Description: "Common Style remains final: explicit dimensions and min/max override intrinsic theme sizing, padding defines the track content area, Background colors the track, and Text.Color colors completion.",
			Code: `dxui.ProgressBar(dxui.ProgressBarProps{
    Style: dxui.Style{
        Width: dxui.Px(280), Height: dxui.Px(24),
        Padding: dxui.Padding(3),
        Background: dxui.ColorRGBA(226, 232, 240, 255),
        Text: dxui.TextStyle{Color: dxui.ColorRGBA(124, 58, 237, 255)},
        Border: dxui.Stroke(1, dxui.ColorRGBA(124, 58, 237, 255)),
        Radius: dxui.Round(6),
    },
    Value: .65,
})`,
			Preview: dxui.ProgressBar(dxui.ProgressBarProps{Style: dxui.Style{
				Width: dxui.Px(280), Height: dxui.Px(24), Padding: dxui.Padding(3),
				Background: dxui.ColorRGBA(226, 232, 240, 255),
				Text:       dxui.TextStyle{Color: dxui.ColorRGBA(124, 58, 237, 255)},
				Border:     dxui.Stroke(1, dxui.ColorRGBA(124, 58, 237, 255)),
				Radius:     dxui.Round(6), Opacity: dxui.Some(float32(.9)), Overflow: dxui.OverflowClip,
			}, Value: .65}),
		},
	}
}

func badgeBlocks(assets galleryAssets) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Text, number, and icon badges",
			Description: "Badge sizes itself from its child plus theme padding. Text and Icon inherit the Badge tint, and the entire pill remains outside focus and pointer activation.",
			Code: `dxui.Badge(dxui.BadgeProps{},
    dxui.Text(dxui.TextProps{Value: "New"}),
)
dxui.Badge(dxui.BadgeProps{},
    dxui.Text(dxui.TextProps{Value: "7"}),
)
dxui.Badge(dxui.BadgeProps{},
    dxui.Icon(dxui.IconProps{Data: icon.Star(), Size: 14}),
)`,
			Preview: previewHorizontal(
				dxui.Badge(dxui.BadgeProps{}, dxui.Text(dxui.TextProps{Value: "New"})),
				dxui.Badge(dxui.BadgeProps{}, dxui.Text(dxui.TextProps{Value: "7"})),
				dxui.Badge(dxui.BadgeProps{}, dxui.Icon(dxui.IconProps{Data: assets.star, Size: 14})),
			),
		},
		{
			Title:       "Arbitrary child and local styling",
			Description: "Use a horizontal Box for compound content. Common Style controls size, Flex, border, opacity, clipping, and stacking; local child colors override inherited Badge tint.",
			Code: `dxui.Badge(dxui.BadgeProps{Style: dxui.Style{
    Background: dxui.ColorRGBA(124, 58, 237, 255),
    Padding: dxui.PaddingXY(10, 3),
}}, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 4},
    dxui.Icon(dxui.IconProps{Data: icon.Star(), Size: 12}),
    dxui.Text(dxui.TextProps{Value: "Featured"}),
))`,
			Preview: dxui.Badge(dxui.BadgeProps{Style: dxui.Style{
				Background: dxui.ColorRGBA(124, 58, 237, 255),
				Padding:    dxui.Edges(3, 10, 3, 10),
			}}, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 4, Align: dxui.AlignCenter},
				dxui.Icon(dxui.IconProps{Data: assets.star, Size: 12}),
				dxui.Text(dxui.TextProps{Value: "Featured"}),
			)),
		},
	}
}

func avatarBlocks(state *galleryState, assets galleryAssets) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Circle, square, and custom sizes",
			Description: "Unset Size uses the Avatar component metric. Size sets equal width and height; explicit Style dimensions still win. Both shapes always use centered Cover without stretching.",
			Code: `dxui.Avatar(dxui.AvatarProps{
    Source: photo, Shape: dxui.AvatarCircle, // default shape
})
dxui.Avatar(dxui.AvatarProps{
    Source: photo, Shape: dxui.AvatarSquare,
    Size: 64,
})`,
			Preview: previewHorizontal(
				avatarTile("default circle", assets.checker, dxui.AvatarCircle, 0),
				avatarTile("48 circle", assets.checkerPNG, dxui.AvatarCircle, 48),
				avatarTile("64 square", assets.checker, dxui.AvatarSquare, 64),
			),
		},
		{
			Title:       "Shared image lifecycle and callbacks",
			Description: "Avatar reuses ImageSource decoding, byte-bounded CPU/native caches, intrinsic load dimensions, deterministic error placeholder, and callback lifecycle. It adds no URL or fallback engine.",
			Code: `dxui.Avatar(dxui.AvatarProps{
    Source: source, Size: 56,
    OnLoad: func(size dxui.Size) { status = fmt.Sprint(size) },
    OnError: func(err error) { status = err.Error() },
})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				bodyText(state.imageStatus),
				previewHorizontal(
					dxui.Avatar(dxui.AvatarProps{Source: assets.checkerPNG, Size: 56, OnLoad: func(size dxui.Size) {
						state.imageStatus = fmt.Sprintf("Avatar OnLoad -> %.0fx%.0f", size.Width, size.Height)
					}}),
					dxui.Avatar(dxui.AvatarProps{Source: assets.brokenImage, Shape: dxui.AvatarSquare, Size: 56, OnError: func(err error) { state.imageStatus = "Avatar OnError -> " + err.Error() }}),
				),
			),
		},
		{
			Title:       "Profile row with Avatar and Badge",
			Description: "A practical identity strip combines Avatar, text, and Badge with ordinary horizontal and vertical Box layout. The status badge remains informational, while the surrounding profile can be placed inside any interactive control when needed.",
			Code: `dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
    Gap: 10, Align: dxui.AlignCenter,
},
    dxui.Avatar(dxui.AvatarProps{
        Source: photo, Size: 48, Shape: dxui.AvatarCircle,
    }),
    dxui.Box(dxui.BoxProps{Gap: 3},
        dxui.Text(dxui.TextProps{Value: "Ada Lovelace"}),
        dxui.Text(dxui.TextProps{Value: "Maintainer"}),
    ),
    dxui.Badge(dxui.BadgeProps{},
        dxui.Text(dxui.TextProps{Value: "Online"}),
    ),
)`,
			Preview: dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 10, Align: dxui.AlignCenter},
				dxui.Avatar(dxui.AvatarProps{Source: assets.checker, Size: 48, Shape: dxui.AvatarCircle}),
				dxui.Box(dxui.BoxProps{Gap: 3},
					dxui.Text(dxui.TextProps{Value: "Ada Lovelace"}),
					dxui.Text(dxui.TextProps{Value: "Maintainer"}),
				),
				dxui.Badge(dxui.BadgeProps{}, dxui.Text(dxui.TextProps{Value: "Online"})),
			),
		},
	}
}

func avatarTile(label string, source dxui.ImageSource, shape dxui.AvatarShape, size float32) dxui.View {
	return dxui.Box(dxui.BoxProps{Gap: 5, Align: dxui.AlignCenter},
		dxui.Avatar(dxui.AvatarProps{
			Style: avatarBorderStyle(), Source: source, Shape: shape, Size: size,
		}),
		smallLabel(label),
	)
}

func avatarBorderStyle() dxui.Style {
	return dxui.Style{Border: dxui.Stroke(2, dxui.TokenColor(dxui.ColorSemanticBorder))}
}

func textBlocks() []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Value, no-wrap, and empty content",
			Description: "Value is normalized UTF-8. TextNoWrap is the zero/default policy. An empty string is valid and contributes no glyphs; a fixed Style size can still make it observable.",
			Code: `dxui.Label("Hello, dxui")
dxui.Text(dxui.TextProps{
    Value: "", Wrap: dxui.TextNoWrap,
    Style: dxui.Style{Width: dxui.Px(120), Height: dxui.Px(24)},
})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 6},
				dxui.Label("Hello, dxui — UTF-8 text"),
				dxui.Text(dxui.TextProps{Style: dxui.Style{Width: dxui.Px(180), Height: dxui.Px(24), Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi)}, Value: "", Wrap: dxui.TextNoWrap}),
				smallLabel("empty value above retains its explicit box"),
			),
		},
		{
			Title:       "Word wrapping and MaxLines",
			Description: "TextWrapWords applies the MVP simple word policy. MaxLines 0 is unlimited; a positive value truncates after that many laid-out lines. Negative values are invalid.",
			Code: `dxui.Text(dxui.TextProps{
    Value: longText,
    Wrap: dxui.TextWrapWords,
    MaxLines: 2, // zero means unlimited
    Style: dxui.Style{Width: dxui.Px(280)},
})`,
			Preview: previewHorizontal(
				dxui.Text(dxui.TextProps{Style: dxui.Style{Width: dxui.Px(260), Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi)}, Value: "Unlimited wrapping keeps every word and every resulting line inside this narrow text box.", Wrap: dxui.TextWrapWords}),
				dxui.Text(dxui.TextProps{Style: dxui.Style{Width: dxui.Px(260), Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi)}, Value: "MaxLines two keeps only the first two laid-out lines from this longer value.", Wrap: dxui.TextWrapWords, MaxLines: 2}),
			),
		},
		{
			Title:       "Complete TextStyle",
			Description: "Families are tried in order, then the App default and built-in fallback. The built-in theme defaults to size 15 and line height 20. Weight offers regular/medium/bold, Slant normal/italic, Color literal/token, and Align start/center/end.",
			Code: `Text: dxui.TextStyle{
    Families: []dxui.FontFamily{dxui.FontFamilyDefault},
    Size: 20, LineHeight: 30,
    Weight: dxui.WeightBold, Slant: dxui.SlantItalic,
    Color: dxui.TokenColor(dxui.ColorSemanticAccent),
    Align: dxui.TextCenter,
}`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 6},
				styledText("Regular / normal / start", dxui.WeightRegular, dxui.SlantNormal, dxui.TextStart),
				styledText("Medium / italic / center", dxui.WeightMedium, dxui.SlantItalic, dxui.TextCenter),
				styledText("Bold / normal / end", dxui.WeightBold, dxui.SlantNormal, dxui.TextEnd),
			),
		},
	}
}

func colorBlocks() []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Complete primitive color palette",
			Description: "All 26 Tailwind-inspired families expose fixed RGBA values at 50, 100, 200, 300, 400, 500, 600, 700, 800, 900, and 950 through the typed Color.Primitive namespace.",
			Code: `dxui.TokenColor(dxui.Color.Primitive.Slate50)
dxui.TokenColor(dxui.Color.Primitive.Blue600)
dxui.TokenColor(dxui.Color.Primitive.Red600)
dxui.TokenColor(dxui.Color.Primitive.Olive950)`,
			Preview: primitivePalettePreview(),
		},
	}
}

func themeBlocks(app *dxui.App, state *galleryState) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Semantic color preview",
			Description: "Semantic tokens preserve the Primitive -> Semantic -> Component -> state -> local override chain and automatically follow the light/dark gallery switch.",
			Code: `dxui.TokenColor(dxui.Color.Semantic.Surface)
dxui.TokenColor(dxui.Color.Semantic.SurfaceHigh)
dxui.TokenColor(dxui.Color.Semantic.Text)
dxui.TokenColor(dxui.Color.Semantic.Accent)
dxui.TokenColor(dxui.Color.Semantic.AccentHover)
dxui.TokenColor(dxui.Color.Semantic.Danger)
dxui.TokenColor(dxui.Color.Semantic.Success)
dxui.TokenColor(dxui.Color.Semantic.Border)`,
			Preview: semanticPalettePreview(),
		},
		{
			Title:       "Modify the global theme",
			Description: "Click a swatch to choose any of the 26 Primitive color families for Primary, Surface, Danger, and Success. Primary maps to Accent and AccentHover; Surface also updates SurfaceHigh, Text, and Border with coordinated Light/Dark shades. Every change replaces the complete App theme through SetTheme.",
			Code: `theme := dxui.LightTheme()
if dark {
    theme = dxui.DarkTheme()
    theme.Semantic.Colors[dxui.Color.Semantic.Accent] = dxui.TokenColor(dxui.Color.Primitive.Violet400)
    theme.Semantic.Colors[dxui.Color.Semantic.AccentHover] = dxui.TokenColor(dxui.Color.Primitive.Violet300)
    theme.Semantic.Colors[dxui.Color.Semantic.Surface] = dxui.TokenColor(dxui.Color.Primitive.Zinc950)
    theme.Semantic.Colors[dxui.Color.Semantic.SurfaceHigh] = dxui.TokenColor(dxui.Color.Primitive.Zinc900)
    theme.Semantic.Colors[dxui.Color.Semantic.Text] = dxui.TokenColor(dxui.Color.Primitive.Zinc50)
    theme.Semantic.Colors[dxui.Color.Semantic.Border] = dxui.TokenColor(dxui.Color.Primitive.Zinc600)
    theme.Semantic.Colors[dxui.Color.Semantic.Danger] = dxui.TokenColor(dxui.Color.Primitive.Rose400)
    theme.Semantic.Colors[dxui.Color.Semantic.Success] = dxui.TokenColor(dxui.Color.Primitive.Teal400)
} else {
    theme.Semantic.Colors[dxui.Color.Semantic.Accent] = dxui.TokenColor(dxui.Color.Primitive.Violet600)
    theme.Semantic.Colors[dxui.Color.Semantic.AccentHover] = dxui.TokenColor(dxui.Color.Primitive.Violet700)
    theme.Semantic.Colors[dxui.Color.Semantic.Surface] = dxui.TokenColor(dxui.Color.Primitive.Zinc50)
    theme.Semantic.Colors[dxui.Color.Semantic.SurfaceHigh] = dxui.TokenColor(dxui.Color.Primitive.Zinc100)
    theme.Semantic.Colors[dxui.Color.Semantic.Text] = dxui.TokenColor(dxui.Color.Primitive.Zinc950)
    theme.Semantic.Colors[dxui.Color.Semantic.Border] = dxui.TokenColor(dxui.Color.Primitive.Zinc400)
    theme.Semantic.Colors[dxui.Color.Semantic.Danger] = dxui.TokenColor(dxui.Color.Primitive.Rose600)
    theme.Semantic.Colors[dxui.Color.Semantic.Success] = dxui.TokenColor(dxui.Color.Primitive.Teal600)
}
if err := app.SetTheme(theme); err != nil { /* report */ }`,
			Preview: themeConfigurator(app, state),
		},
		{
			Title:       "Component theme and local Style override",
			Description: "Use a ComponentTheme when one reusable component role needs consistent defaults and state styling. Use local Style for a one-off exception; local values win after the component theme without changing other Buttons.",
			Code: `const emphasis dxui.ComponentToken = "showcase.button.emphasis"

theme := dxui.LightTheme()
theme.Components[emphasis] = dxui.ComponentTheme{
    Base: dxui.StylePatch{
        Background: dxui.Some(dxui.TokenColor(dxui.Color.Semantic.Danger)),
        Border: dxui.Some(dxui.Stroke(1, dxui.TokenColor(dxui.Color.Semantic.Danger))),
        TextColor: dxui.Some(dxui.TokenColor(dxui.Color.Primitive.White)),
    },
    States: dxui.StateStyles{Hover: dxui.StylePatch{
        Opacity: dxui.Some(float32(.82)),
    }},
}
if err := app.SetTheme(theme); err != nil { /* report */ }

dxui.Button(dxui.ButtonProps{Token: emphasis},
    dxui.Text(dxui.TextProps{Value: "Component themed"}))
dxui.Button(dxui.ButtonProps{
    Token: emphasis,
    Style: dxui.Style{Background: dxui.TokenColor(dxui.Color.Semantic.Success)},
}, dxui.Text(dxui.TextProps{Value: "Local override"}))`,
			Preview: previewHorizontal(
				dxui.Button(dxui.ButtonProps{Token: showcaseButtonTheme},
					dxui.Text(dxui.TextProps{Value: "Component themed"})),
				dxui.Button(dxui.ButtonProps{
					Token: showcaseButtonTheme,
					Style: dxui.Style{Background: dxui.TokenColor(dxui.Color.Semantic.Success)},
				}, dxui.Text(dxui.TextProps{Value: "Local override"})),
			),
		},
	}
}

func themeConfigurator(app *dxui.App, state *galleryState) dxui.View {
	reset := func() {
		state.theme = galleryThemeSettings{primary: "default", surface: "default", danger: "default", success: "default"}
		applyGalleryTheme(app, state, "defaults restored")
	}
	return dxui.Box(dxui.BoxProps{Gap: 10, Align: dxui.AlignStart},
		themePaletteChoice(app, state, "primary", "Primary", state.theme.primary),
		themePaletteChoice(app, state, "surface", "Surface", state.theme.surface),
		themePaletteChoice(app, state, "danger", "Danger", state.theme.danger),
		themePaletteChoice(app, state, "success", "Success", state.theme.success),
		dxui.Button(dxui.ButtonProps{OnPress: reset}, dxui.Text(dxui.TextProps{Value: "Reset theme"})),
	)
}

const themeSwatchesPerRow = 9

func themePaletteChoice(app *dxui.App, state *galleryState, role, label, selected string) dxui.View {
	choices := make([]dxui.View, 0, len(primitiveColorScales)+1)
	defaultForeground := dxui.TokenColor(themeRoleTextToken(role))
	choices = append(choices, themePaletteSwatch(app, state, role, "default", selected == "default", dxui.TokenColor(themeRoleToken(role)), defaultForeground, dxui.TokenColor(dxui.Color.Semantic.Text)))
	for _, scale := range primitiveColorScales {
		background, foreground := themeScaleSwatchColors(scale, role, state.dark)
		selectionBorder := themeScaleSelectionBorder(scale, state.dark)
		choices = append(choices, themePaletteSwatch(app, state, role, scale.name, selected == scale.name, dxui.TokenColor(background), dxui.TokenColor(foreground), dxui.TokenColor(selectionBorder)))
	}
	rows := make([]dxui.View, 0, (len(choices)+themeSwatchesPerRow-1)/themeSwatchesPerRow)
	for start := 0; start < len(choices); start += themeSwatchesPerRow {
		end := start + themeSwatchesPerRow
		if end > len(choices) {
			end = len(choices)
		}
		rows = append(rows, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 5}, choices[start:end]...))
	}
	return dxui.Box(dxui.BoxProps{Gap: 5},
		smallLabel(fmt.Sprintf("%s — %s", label, titleCase(selected))),
		dxui.Box(dxui.BoxProps{Gap: 5}, rows...),
	)
}

func themePaletteSwatch(app *dxui.App, state *galleryState, role, value string, selected bool, background, foreground, selectionBorder dxui.ColorValue) dxui.View {
	borderWidth := float32(1)
	borderColor := foreground
	weight := dxui.WeightRegular
	if selected {
		borderWidth = 3
		borderColor = selectionBorder
		weight = dxui.WeightBold
	}
	style := dxui.Style{
		Width: dxui.Px(68), Height: dxui.Px(34), Shrink: dxui.NoShrink(),
		Padding: dxui.Padding(3), Background: background,
		Border: dxui.Stroke(borderWidth, borderColor),
		Text:   dxui.TextStyle{Color: foreground, Size: 10, LineHeight: 14, Weight: weight, Align: dxui.TextCenter},
	}
	return dxui.Button(dxui.ButtonProps{
		Key: "theme-" + role + "-" + value, Style: style,
		States:  dxui.StateStyles{Hover: dxui.StylePatch{Background: dxui.Some(background), Opacity: dxui.Some(float32(.8))}},
		OnPress: func() { setGalleryThemeChoice(app, state, role, value) },
	}, dxui.Text(dxui.TextProps{Value: titleCase(value)}))
}

func themeRoleToken(role string) dxui.ColorToken {
	switch role {
	case "surface":
		return dxui.Color.Semantic.Surface
	case "danger":
		return dxui.Color.Semantic.Danger
	case "success":
		return dxui.Color.Semantic.Success
	default:
		return dxui.Color.Semantic.Accent
	}
}

func themeRoleTextToken(role string) dxui.ColorToken {
	if role == "surface" {
		return dxui.Color.Semantic.Text
	}
	return dxui.Color.Semantic.Surface
}

func themeScaleSwatchColors(scale primitiveColorScale, role string, dark bool) (dxui.ColorToken, dxui.ColorToken) {
	if role == "surface" {
		if dark {
			return scale.tokens[10], scale.tokens[0]
		}
		return scale.tokens[0], scale.tokens[10]
	}
	if dark {
		return scale.tokens[4], scale.tokens[10]
	}
	return scale.tokens[6], scale.tokens[0]
}

func themeScaleSelectionBorder(scale primitiveColorScale, dark bool) dxui.ColorToken {
	if dark {
		return scale.tokens[0]
	}
	return scale.tokens[10]
}

func setGalleryThemeChoice(app *dxui.App, state *galleryState, role, value string) {
	switch role {
	case "primary":
		state.theme.primary = value
	case "surface":
		state.theme.surface = value
	case "danger":
		state.theme.danger = value
	case "success":
		state.theme.success = value
	default:
		return
	}
	applyGalleryTheme(app, state, titleCase(role)+" = "+titleCase(value))
}

func applyGalleryTheme(app *dxui.App, state *galleryState, label string) {
	if err := app.SetTheme(showcaseTheme(state.dark, state.theme)); err != nil {
		state.feedback = "Theme error: " + err.Error()
		return
	}
	state.feedback = "Global theme changed: " + label
}

func titleCase(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func styledText(value string, weight dxui.FontWeight, slant dxui.FontSlant, align dxui.TextAlign) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{Width: dxui.Px(430), Text: dxui.TextStyle{
		Families: []dxui.FontFamily{dxui.FontFamilyDefault}, Size: 17, LineHeight: 26,
		Weight: weight, Slant: slant, Color: dxui.TokenColor(dxui.ColorSemanticAccent), Align: align,
	}}, Value: value})
}

type primitiveColorScale struct {
	name   string
	tokens [11]dxui.ColorToken
}

const (
	paletteScaleWidth  = 74
	paletteSwatchWidth = 62
	paletteCellHeight  = 44
	paletteTextSize    = 12
	paletteLineHeight  = 18
)

var primitiveColorScales = [...]primitiveColorScale{
	{"red", [11]dxui.ColorToken{dxui.Color.Primitive.Red50, dxui.Color.Primitive.Red100, dxui.Color.Primitive.Red200, dxui.Color.Primitive.Red300, dxui.Color.Primitive.Red400, dxui.Color.Primitive.Red500, dxui.Color.Primitive.Red600, dxui.Color.Primitive.Red700, dxui.Color.Primitive.Red800, dxui.Color.Primitive.Red900, dxui.Color.Primitive.Red950}},
	{"orange", [11]dxui.ColorToken{dxui.Color.Primitive.Orange50, dxui.Color.Primitive.Orange100, dxui.Color.Primitive.Orange200, dxui.Color.Primitive.Orange300, dxui.Color.Primitive.Orange400, dxui.Color.Primitive.Orange500, dxui.Color.Primitive.Orange600, dxui.Color.Primitive.Orange700, dxui.Color.Primitive.Orange800, dxui.Color.Primitive.Orange900, dxui.Color.Primitive.Orange950}},
	{"amber", [11]dxui.ColorToken{dxui.Color.Primitive.Amber50, dxui.Color.Primitive.Amber100, dxui.Color.Primitive.Amber200, dxui.Color.Primitive.Amber300, dxui.Color.Primitive.Amber400, dxui.Color.Primitive.Amber500, dxui.Color.Primitive.Amber600, dxui.Color.Primitive.Amber700, dxui.Color.Primitive.Amber800, dxui.Color.Primitive.Amber900, dxui.Color.Primitive.Amber950}},
	{"yellow", [11]dxui.ColorToken{dxui.Color.Primitive.Yellow50, dxui.Color.Primitive.Yellow100, dxui.Color.Primitive.Yellow200, dxui.Color.Primitive.Yellow300, dxui.Color.Primitive.Yellow400, dxui.Color.Primitive.Yellow500, dxui.Color.Primitive.Yellow600, dxui.Color.Primitive.Yellow700, dxui.Color.Primitive.Yellow800, dxui.Color.Primitive.Yellow900, dxui.Color.Primitive.Yellow950}},
	{"lime", [11]dxui.ColorToken{dxui.Color.Primitive.Lime50, dxui.Color.Primitive.Lime100, dxui.Color.Primitive.Lime200, dxui.Color.Primitive.Lime300, dxui.Color.Primitive.Lime400, dxui.Color.Primitive.Lime500, dxui.Color.Primitive.Lime600, dxui.Color.Primitive.Lime700, dxui.Color.Primitive.Lime800, dxui.Color.Primitive.Lime900, dxui.Color.Primitive.Lime950}},
	{"green", [11]dxui.ColorToken{dxui.Color.Primitive.Green50, dxui.Color.Primitive.Green100, dxui.Color.Primitive.Green200, dxui.Color.Primitive.Green300, dxui.Color.Primitive.Green400, dxui.Color.Primitive.Green500, dxui.Color.Primitive.Green600, dxui.Color.Primitive.Green700, dxui.Color.Primitive.Green800, dxui.Color.Primitive.Green900, dxui.Color.Primitive.Green950}},
	{"emerald", [11]dxui.ColorToken{dxui.Color.Primitive.Emerald50, dxui.Color.Primitive.Emerald100, dxui.Color.Primitive.Emerald200, dxui.Color.Primitive.Emerald300, dxui.Color.Primitive.Emerald400, dxui.Color.Primitive.Emerald500, dxui.Color.Primitive.Emerald600, dxui.Color.Primitive.Emerald700, dxui.Color.Primitive.Emerald800, dxui.Color.Primitive.Emerald900, dxui.Color.Primitive.Emerald950}},
	{"teal", [11]dxui.ColorToken{dxui.Color.Primitive.Teal50, dxui.Color.Primitive.Teal100, dxui.Color.Primitive.Teal200, dxui.Color.Primitive.Teal300, dxui.Color.Primitive.Teal400, dxui.Color.Primitive.Teal500, dxui.Color.Primitive.Teal600, dxui.Color.Primitive.Teal700, dxui.Color.Primitive.Teal800, dxui.Color.Primitive.Teal900, dxui.Color.Primitive.Teal950}},
	{"cyan", [11]dxui.ColorToken{dxui.Color.Primitive.Cyan50, dxui.Color.Primitive.Cyan100, dxui.Color.Primitive.Cyan200, dxui.Color.Primitive.Cyan300, dxui.Color.Primitive.Cyan400, dxui.Color.Primitive.Cyan500, dxui.Color.Primitive.Cyan600, dxui.Color.Primitive.Cyan700, dxui.Color.Primitive.Cyan800, dxui.Color.Primitive.Cyan900, dxui.Color.Primitive.Cyan950}},
	{"sky", [11]dxui.ColorToken{dxui.Color.Primitive.Sky50, dxui.Color.Primitive.Sky100, dxui.Color.Primitive.Sky200, dxui.Color.Primitive.Sky300, dxui.Color.Primitive.Sky400, dxui.Color.Primitive.Sky500, dxui.Color.Primitive.Sky600, dxui.Color.Primitive.Sky700, dxui.Color.Primitive.Sky800, dxui.Color.Primitive.Sky900, dxui.Color.Primitive.Sky950}},
	{"blue", [11]dxui.ColorToken{dxui.Color.Primitive.Blue50, dxui.Color.Primitive.Blue100, dxui.Color.Primitive.Blue200, dxui.Color.Primitive.Blue300, dxui.Color.Primitive.Blue400, dxui.Color.Primitive.Blue500, dxui.Color.Primitive.Blue600, dxui.Color.Primitive.Blue700, dxui.Color.Primitive.Blue800, dxui.Color.Primitive.Blue900, dxui.Color.Primitive.Blue950}},
	{"indigo", [11]dxui.ColorToken{dxui.Color.Primitive.Indigo50, dxui.Color.Primitive.Indigo100, dxui.Color.Primitive.Indigo200, dxui.Color.Primitive.Indigo300, dxui.Color.Primitive.Indigo400, dxui.Color.Primitive.Indigo500, dxui.Color.Primitive.Indigo600, dxui.Color.Primitive.Indigo700, dxui.Color.Primitive.Indigo800, dxui.Color.Primitive.Indigo900, dxui.Color.Primitive.Indigo950}},
	{"violet", [11]dxui.ColorToken{dxui.Color.Primitive.Violet50, dxui.Color.Primitive.Violet100, dxui.Color.Primitive.Violet200, dxui.Color.Primitive.Violet300, dxui.Color.Primitive.Violet400, dxui.Color.Primitive.Violet500, dxui.Color.Primitive.Violet600, dxui.Color.Primitive.Violet700, dxui.Color.Primitive.Violet800, dxui.Color.Primitive.Violet900, dxui.Color.Primitive.Violet950}},
	{"purple", [11]dxui.ColorToken{dxui.Color.Primitive.Purple50, dxui.Color.Primitive.Purple100, dxui.Color.Primitive.Purple200, dxui.Color.Primitive.Purple300, dxui.Color.Primitive.Purple400, dxui.Color.Primitive.Purple500, dxui.Color.Primitive.Purple600, dxui.Color.Primitive.Purple700, dxui.Color.Primitive.Purple800, dxui.Color.Primitive.Purple900, dxui.Color.Primitive.Purple950}},
	{"fuchsia", [11]dxui.ColorToken{dxui.Color.Primitive.Fuchsia50, dxui.Color.Primitive.Fuchsia100, dxui.Color.Primitive.Fuchsia200, dxui.Color.Primitive.Fuchsia300, dxui.Color.Primitive.Fuchsia400, dxui.Color.Primitive.Fuchsia500, dxui.Color.Primitive.Fuchsia600, dxui.Color.Primitive.Fuchsia700, dxui.Color.Primitive.Fuchsia800, dxui.Color.Primitive.Fuchsia900, dxui.Color.Primitive.Fuchsia950}},
	{"pink", [11]dxui.ColorToken{dxui.Color.Primitive.Pink50, dxui.Color.Primitive.Pink100, dxui.Color.Primitive.Pink200, dxui.Color.Primitive.Pink300, dxui.Color.Primitive.Pink400, dxui.Color.Primitive.Pink500, dxui.Color.Primitive.Pink600, dxui.Color.Primitive.Pink700, dxui.Color.Primitive.Pink800, dxui.Color.Primitive.Pink900, dxui.Color.Primitive.Pink950}},
	{"rose", [11]dxui.ColorToken{dxui.Color.Primitive.Rose50, dxui.Color.Primitive.Rose100, dxui.Color.Primitive.Rose200, dxui.Color.Primitive.Rose300, dxui.Color.Primitive.Rose400, dxui.Color.Primitive.Rose500, dxui.Color.Primitive.Rose600, dxui.Color.Primitive.Rose700, dxui.Color.Primitive.Rose800, dxui.Color.Primitive.Rose900, dxui.Color.Primitive.Rose950}},
	{"slate", [11]dxui.ColorToken{dxui.Color.Primitive.Slate50, dxui.Color.Primitive.Slate100, dxui.Color.Primitive.Slate200, dxui.Color.Primitive.Slate300, dxui.Color.Primitive.Slate400, dxui.Color.Primitive.Slate500, dxui.Color.Primitive.Slate600, dxui.Color.Primitive.Slate700, dxui.Color.Primitive.Slate800, dxui.Color.Primitive.Slate900, dxui.Color.Primitive.Slate950}},
	{"gray", [11]dxui.ColorToken{dxui.Color.Primitive.Gray50, dxui.Color.Primitive.Gray100, dxui.Color.Primitive.Gray200, dxui.Color.Primitive.Gray300, dxui.Color.Primitive.Gray400, dxui.Color.Primitive.Gray500, dxui.Color.Primitive.Gray600, dxui.Color.Primitive.Gray700, dxui.Color.Primitive.Gray800, dxui.Color.Primitive.Gray900, dxui.Color.Primitive.Gray950}},
	{"zinc", [11]dxui.ColorToken{dxui.Color.Primitive.Zinc50, dxui.Color.Primitive.Zinc100, dxui.Color.Primitive.Zinc200, dxui.Color.Primitive.Zinc300, dxui.Color.Primitive.Zinc400, dxui.Color.Primitive.Zinc500, dxui.Color.Primitive.Zinc600, dxui.Color.Primitive.Zinc700, dxui.Color.Primitive.Zinc800, dxui.Color.Primitive.Zinc900, dxui.Color.Primitive.Zinc950}},
	{"neutral", [11]dxui.ColorToken{dxui.Color.Primitive.Neutral50, dxui.Color.Primitive.Neutral100, dxui.Color.Primitive.Neutral200, dxui.Color.Primitive.Neutral300, dxui.Color.Primitive.Neutral400, dxui.Color.Primitive.Neutral500, dxui.Color.Primitive.Neutral600, dxui.Color.Primitive.Neutral700, dxui.Color.Primitive.Neutral800, dxui.Color.Primitive.Neutral900, dxui.Color.Primitive.Neutral950}},
	{"stone", [11]dxui.ColorToken{dxui.Color.Primitive.Stone50, dxui.Color.Primitive.Stone100, dxui.Color.Primitive.Stone200, dxui.Color.Primitive.Stone300, dxui.Color.Primitive.Stone400, dxui.Color.Primitive.Stone500, dxui.Color.Primitive.Stone600, dxui.Color.Primitive.Stone700, dxui.Color.Primitive.Stone800, dxui.Color.Primitive.Stone900, dxui.Color.Primitive.Stone950}},
	{"taupe", [11]dxui.ColorToken{dxui.Color.Primitive.Taupe50, dxui.Color.Primitive.Taupe100, dxui.Color.Primitive.Taupe200, dxui.Color.Primitive.Taupe300, dxui.Color.Primitive.Taupe400, dxui.Color.Primitive.Taupe500, dxui.Color.Primitive.Taupe600, dxui.Color.Primitive.Taupe700, dxui.Color.Primitive.Taupe800, dxui.Color.Primitive.Taupe900, dxui.Color.Primitive.Taupe950}},
	{"mauve", [11]dxui.ColorToken{dxui.Color.Primitive.Mauve50, dxui.Color.Primitive.Mauve100, dxui.Color.Primitive.Mauve200, dxui.Color.Primitive.Mauve300, dxui.Color.Primitive.Mauve400, dxui.Color.Primitive.Mauve500, dxui.Color.Primitive.Mauve600, dxui.Color.Primitive.Mauve700, dxui.Color.Primitive.Mauve800, dxui.Color.Primitive.Mauve900, dxui.Color.Primitive.Mauve950}},
	{"mist", [11]dxui.ColorToken{dxui.Color.Primitive.Mist50, dxui.Color.Primitive.Mist100, dxui.Color.Primitive.Mist200, dxui.Color.Primitive.Mist300, dxui.Color.Primitive.Mist400, dxui.Color.Primitive.Mist500, dxui.Color.Primitive.Mist600, dxui.Color.Primitive.Mist700, dxui.Color.Primitive.Mist800, dxui.Color.Primitive.Mist900, dxui.Color.Primitive.Mist950}},
	{"olive", [11]dxui.ColorToken{dxui.Color.Primitive.Olive50, dxui.Color.Primitive.Olive100, dxui.Color.Primitive.Olive200, dxui.Color.Primitive.Olive300, dxui.Color.Primitive.Olive400, dxui.Color.Primitive.Olive500, dxui.Color.Primitive.Olive600, dxui.Color.Primitive.Olive700, dxui.Color.Primitive.Olive800, dxui.Color.Primitive.Olive900, dxui.Color.Primitive.Olive950}},
}

func primitivePalettePreview() dxui.View {
	shades := []string{"50", "100", "200", "300", "400", "500", "600", "700", "800", "900", "950"}
	rows := make([]dxui.View, 0, len(primitiveColorScales)+1)
	header := make([]dxui.View, 0, len(shades)+1)
	header = append(header, paletteLabel("scale", paletteScaleWidth))
	for _, shade := range shades {
		header = append(header, paletteLabel(shade, paletteSwatchWidth))
	}
	rows = append(rows, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 3}, header...))
	for _, scale := range primitiveColorScales {
		cells := make([]dxui.View, 0, len(scale.tokens)+1)
		cells = append(cells, paletteLabel(scale.name, paletteScaleWidth))
		for index, token := range scale.tokens {
			text := dxui.Color.Primitive.White
			if index < 5 {
				text = dxui.Color.Primitive.Slate950
			}
			cells = append(cells, dxui.Text(dxui.TextProps{Style: dxui.Style{
				Width: dxui.Px(paletteSwatchWidth), Height: dxui.Px(paletteCellHeight), Shrink: dxui.NoShrink(),
				Padding: dxui.Padding(6), Background: dxui.TokenColor(token),
				Text: dxui.TextStyle{Color: dxui.TokenColor(text), Size: paletteTextSize, LineHeight: paletteLineHeight},
			}, Value: shades[index]}))
		}
		rows = append(rows, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 3}, cells...))
	}
	return dxui.Box(dxui.BoxProps{Gap: 3}, rows...)
}

func paletteLabel(value string, width float32) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{
		Width: dxui.Px(width), Height: dxui.Px(paletteCellHeight), Shrink: dxui.NoShrink(),
		Text: dxui.TextStyle{Size: paletteTextSize, LineHeight: paletteLineHeight},
	}, Value: value})
}

func semanticPalettePreview() dxui.View {
	items := []struct {
		name       string
		token      dxui.ColorToken
		foreground dxui.ColorToken
	}{
		{"surface", dxui.Color.Semantic.Surface, dxui.Color.Semantic.Text},
		{"surface.high", dxui.Color.Semantic.SurfaceHigh, dxui.Color.Semantic.Text},
		{"text", dxui.Color.Semantic.Text, dxui.Color.Semantic.Surface},
		{"accent", dxui.Color.Semantic.Accent, dxui.Color.Semantic.Surface},
		{"accent.hover", dxui.Color.Semantic.AccentHover, dxui.Color.Semantic.Surface},
		{"danger", dxui.Color.Semantic.Danger, dxui.Color.Semantic.Surface},
		{"success", dxui.Color.Semantic.Success, dxui.Color.Semantic.Surface},
		{"border", dxui.Color.Semantic.Border, dxui.Color.Semantic.Text},
	}
	views := make([]dxui.View, len(items))
	for index, item := range items {
		views[index] = swatch(item.name, dxui.Style{
			Background: dxui.TokenColor(item.token),
			Border:     dxui.Stroke(1, dxui.TokenColor(dxui.Color.Semantic.Border)),
			Text:       dxui.TextStyle{Color: dxui.TokenColor(item.foreground)},
		})
	}
	return dxui.Box(dxui.BoxProps{Gap: 8},
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8}, views[:4]...),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8}, views[4:]...),
	)
}

func iconBlocks(app *dxui.App, state *galleryState, assets galleryAssets) []exampleBlock {
	return []exampleBlock{
		iconCatalogBlock(app, state),
		generatedIconScaleBlock(),
		{
			Title:       "Size and color",
			Description: "Size 0 uses the component icon-size token; a positive Size is a fixed logical-unit value. Unset Color inherits theme/state text tint; explicit token or literal colors override it.",
			Code: `dxui.Icon(dxui.IconProps{Data: icon.Star()}) // theme size/color
dxui.Icon(dxui.IconProps{
    Data: icon.Star(), Size: 32,
    Color: dxui.TokenColor(dxui.ColorSemanticAccent),
})`,
			Preview: previewHorizontal(
				dxui.Icon(dxui.IconProps{Data: assets.star}),
				dxui.Icon(dxui.IconProps{Data: assets.star, Color: dxui.TokenColor(dxui.ColorSemanticAccent)}),
				dxui.Icon(dxui.IconProps{Data: assets.star, Size: 32, Color: dxui.ColorRGBA(168, 85, 247, 255)}),
			),
		},
		{
			Title:       "Lucide resources",
			Description: "Use the official icon package for immutable Lucide resources with consistent round strokes. Icon controls size and theme tint.",
			Code: `dxui.Icon(dxui.IconProps{Data: icon.Spline(), Size: 56})
dxui.Icon(dxui.IconProps{Data: icon.Star(), Size: 56})`,
			Preview: previewHorizontal(
				dxui.Icon(dxui.IconProps{Data: assets.curve, Size: 56, Color: dxui.TokenColor(dxui.ColorSemanticAccent)}),
				dxui.Icon(dxui.IconProps{Data: assets.star, Size: 56}),
				bodyText("Lucide Spline and Star share the default stroke width 2."),
			),
		},
		{
			Title:       "Custom content in a Button",
			Description: "Icon has no event API of its own. Compose it into a Button (or another semantic control) for interaction; descendant tint inherits the button state unless Color is locally set.",
			Code: `dxui.Button(dxui.ButtonProps{OnPress: save},
    dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 6},
        dxui.Icon(dxui.IconProps{Data: icon.Star()}),
        dxui.Text(dxui.TextProps{Value: "Favorite"}),
    ),
)`,
			Preview: dxui.Button(dxui.ButtonProps{}, previewHorizontal(
				dxui.Icon(dxui.IconProps{Data: assets.star, Size: 16}),
				bodyText("Arbitrary child content"),
			)),
		},
	}
}

func generatedIconScaleBlock() exampleBlock {
	search := icon.Search()
	return exampleBlock{
		Title:       "Generated icon size and stroke width",
		Description: "One generated icon demonstrates size and view-box-unit stroke width without repeating those variants for every catalog entry. StrokeWidth zero uses Lucide's default width 2.",
		Code: `search := icon.Search()
dxui.Icon(dxui.IconProps{Data: search, Size: 16, StrokeWidth: 1})
dxui.Icon(dxui.IconProps{Data: search, Size: 24}) // default width 2
dxui.Icon(dxui.IconProps{Data: search, Size: 32, StrokeWidth: 3})`,
		Preview: previewHorizontal(
			iconVariant("16 / 1", search, 16, 1),
			iconVariant("24 / 2", search, 24, 0),
			iconVariant("32 / 3", search, 32, 3),
		),
	}
}

func iconVariant(label string, data dxui.IconData, size float32, strokeWidth dxui.IconStrokeWidth) dxui.View {
	return dxui.Box(dxui.BoxProps{Gap: 6, Align: dxui.AlignCenter},
		dxui.Icon(dxui.IconProps{Data: data, Size: size, StrokeWidth: strokeWidth}),
		bodyText(label),
	)
}

func iconCatalogBlock(app *dxui.App, state *galleryState) exampleBlock {
	entries, matches, page, pages := iconCatalogPage(state.iconQuery, state.iconPage)
	rows := make([]dxui.View, 0, (len(entries)+iconCatalogColumns-1)/iconCatalogColumns)
	for offset := 0; offset < len(entries); offset += iconCatalogColumns {
		end := min(offset+iconCatalogColumns, len(entries))
		buttons := make([]dxui.View, 0, end-offset)
		for _, entry := range entries[offset:end] {
			entry := entry
			buttons = append(buttons, dxui.Button(dxui.ButtonProps{
				Key: entry.Name, Style: dxui.Style{
					Width: dxui.Px(iconCatalogCellSize), Height: dxui.Px(iconCatalogCellSize), Padding: dxui.Padding(0),
					Background: dxui.TokenColor(dxui.Color.Semantic.SurfaceHigh),
					Radius:     dxui.Round(6),
				},
				OnPress: func() {
					if err := app.SetClipboardText(entry.GoName); err != nil {
						state.feedback = "Copy failed: " + err.Error()
						return
					}
					state.feedback = "Copied " + entry.GoName + "."
				},
			}, dxui.Icon(dxui.IconProps{
				Data: entry.Icon(), Size: 24,
				Color: dxui.TokenColor(dxui.Color.Semantic.Border),
			})))
		}
		rows = append(rows, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: iconCatalogGap}, buttons...))
	}
	if matches == 0 {
		rows = append(rows, bodyText("No icon name matches the search."))
	}

	pageLabel := fmt.Sprintf("%d matches", matches)
	if matches != 0 {
		pageLabel = fmt.Sprintf("%d matches · page %d/%d", matches, page+1, pages)
	}
	previous := func() {
		if page > 0 {
			state.iconPage = page - 1
		}
	}
	next := func() {
		if page+1 < pages {
			state.iconPage = page + 1
		}
	}

	return exampleBlock{
		Title:       "Complete searchable Lucide catalog",
		Description: fmt.Sprintf("Search all %d generated Lucide names, including aliases. Results use stable catalog order and show up to %d names per page in a %d×%d icon-only grid with a light semantic background and neutral gray icons. Click an icon to copy its exact Go constructor name, such as Search.", len(catalog.Icons), iconCatalogPageSize, iconCatalogColumns, iconCatalogPageSize/iconCatalogColumns),
		Code: `import "github.com/dxui-org/dxui/icon/catalog"

for _, entry := range catalog.Icons {
    if strings.Contains(entry.Name, query) {
        view := dxui.Button(dxui.ButtonProps{
            OnPress: func() { _ = app.SetClipboardText(entry.GoName) },
        }, dxui.Icon(dxui.IconProps{Data: entry.Icon()}))
        // Add the icon-only button to a bounded multi-column page.
    }
}`,
		Preview: dxui.Box(dxui.BoxProps{Gap: 10, Align: dxui.AlignStretch},
			dxui.Input(dxui.InputProps{
				Key: "icon-search", Style: dxui.Style{Width: dxui.Px(iconCatalogWidth), Height: dxui.Px(40)},
				Value: state.iconQuery, Placeholder: "Search icon names, e.g. arrow, circle, user...",
				OnChange: func(value string) {
					state.iconQuery = value
					state.iconPage = 0
				},
				OnSubmit: func() { state.feedback = fmt.Sprintf("Icon search submitted: %q", state.iconQuery) },
			}),
			dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8, Align: dxui.AlignCenter},
				dxui.Button(dxui.ButtonProps{Disabled: page == 0, OnPress: previous}, dxui.Text(dxui.TextProps{Value: "Previous"})),
				dxui.Button(dxui.ButtonProps{Disabled: pages == 0 || page+1 >= pages, OnPress: next}, dxui.Text(dxui.TextProps{Value: "Next"})),
				bodyText(pageLabel),
			),
			dxui.Box(dxui.BoxProps{Gap: iconCatalogGap, Align: dxui.AlignStart}, rows...),
		),
	}
}

func iconCatalogPage(query string, requestedPage int) ([]catalog.Entry, int, int, int) {
	query = strings.ToLower(strings.TrimSpace(query))
	matches := 0
	for _, entry := range catalog.Icons {
		if query == "" || strings.Contains(entry.Name, query) {
			matches++
		}
	}
	if matches == 0 {
		return nil, 0, 0, 0
	}
	pages := (matches + iconCatalogPageSize - 1) / iconCatalogPageSize
	page := requestedPage
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * iconCatalogPageSize
	end := min(start+iconCatalogPageSize, matches)
	entries := make([]catalog.Entry, 0, end-start)
	matched := 0
	for _, entry := range catalog.Icons {
		if query != "" && !strings.Contains(entry.Name, query) {
			continue
		}
		if matched >= start && matched < end {
			entries = append(entries, entry)
		}
		matched++
		if matched == end {
			break
		}
	}
	return entries, matches, page, pages
}

func imageBlocks(state *galleryState, assets galleryAssets) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "All fit modes",
			Description: "Contain preserves the whole image, Cover fills and clips, Fill stretches both axes, and None keeps intrinsic logical size. The same generated image is used for direct Go and encoded PNG sources.",
			Code: `dxui.Image(dxui.ImageProps{
    Source: dxui.ImageFromGo(img), // or ImageBytes(encoded)
    Fit: dxui.ImageCover,
    Alignment: dxui.Point{X: .5, Y: .5},
    Style: dxui.Style{
        Width: dxui.Px(120), Height: dxui.Px(86), Overflow: dxui.OverflowClip,
    },
})`,
			Preview: previewHorizontal(
				imageTile("contain", assets.checker, dxui.ImageContain, dxui.Point{X: .5, Y: .5}, 0, nil, nil),
				imageTile("cover", assets.checkerPNG, dxui.ImageCover, dxui.Point{X: .5, Y: .5}, 0, nil, nil),
				imageTile("fill", assets.checker, dxui.ImageFill, dxui.Point{}, 0, nil, nil),
				imageTile("none", assets.checkerPNG, dxui.ImageNone, dxui.Point{X: .5, Y: .5}, 0, nil, nil),
			),
		},
		{
			Title:       "Alignment boundaries",
			Description: "Alignment X/Y must be within 0..1: 0 anchors start/top, .5 centers, and 1 anchors end/bottom. It is observable when Cover or None leaves content to crop or position.",
			Code: `Alignment: dxui.Point{X: 0, Y: 0}   // start/top
Alignment: dxui.Point{X: .5, Y: .5} // center
Alignment: dxui.Point{X: 1, Y: 1}   // end/bottom`,
			Preview: previewHorizontal(
				imageTile("0, 0", assets.checker, dxui.ImageCover, dxui.Point{}, 0, nil, nil),
				imageTile(".5, .5", assets.checker, dxui.ImageCover, dxui.Point{X: .5, Y: .5}, 0, nil, nil),
				imageTile("1, 1", assets.checker, dxui.ImageCover, dxui.Point{X: 1, Y: 1}, 0, nil, nil),
			),
		},
		{
			Title:       "Decode limits and load/error events",
			Description: "MaxPixels 0 selects the default guarded limit. A positive value is a per-image boundary. OnLoad reports intrinsic Size once per source identity; OnError reports decode/source/limit failure. The broken source and MaxPixels=32 exercise real error paths.",
			Code: `dxui.Image(dxui.ImageProps{
    Source: dxui.ImageBytes(pngBytes), MaxPixels: 32,
    OnLoad: func(size dxui.Size) { status = fmt.Sprint(size) },
    OnError: func(err error) { status = err.Error() },
})
// ImageFile("assets/photo.png") reads that exact path on demand.`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				bodyText(state.imageStatus),
				previewHorizontal(
					imageTile("OnLoad", assets.checkerPNG, dxui.ImageContain, dxui.Point{X: .5, Y: .5}, 0,
						func(size dxui.Size) { state.imageStatus = fmt.Sprintf("OnLoad -> %.0fx%.0f", size.Width, size.Height) }, nil),
					imageTile("limit error", assets.checkerPNG, dxui.ImageContain, dxui.Point{X: .5, Y: .5}, 32, nil,
						func(err error) { state.imageStatus = "OnError (MaxPixels) -> " + err.Error() }),
					imageTile("decode error", assets.brokenImage, dxui.ImageContain, dxui.Point{X: .5, Y: .5}, 0, nil,
						func(err error) { state.imageStatus = "OnError (decode) -> " + err.Error() }),
				),
			),
		},
	}
}

func imageTile(label string, source dxui.ImageSource, fit dxui.ImageFit, alignment dxui.Point, maxPixels int64, onLoad func(dxui.Size), onError func(error)) dxui.View {
	return dxui.Box(dxui.BoxProps{Gap: 5, Align: dxui.AlignCenter},
		dxui.Image(dxui.ImageProps{
			Style:  dxui.Style{Width: dxui.Px(120), Height: dxui.Px(82), Overflow: dxui.OverflowClip, Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi)},
			Source: source, Fit: fit, Alignment: alignment, MaxPixels: maxPixels, OnLoad: onLoad, OnError: onError,
		}),
		smallLabel(label),
	)
}
