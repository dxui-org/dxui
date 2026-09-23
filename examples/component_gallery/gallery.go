package main

import (
	"fmt"
	"strings"

	"github.com/dxui-org/dxui"
)

const (
	sidebarWidth           = 260
	sidebarPadding         = 14
	sidebarRightPadding    = 4
	sidebarScrollbarGutter = 22
	sidebarMenuWidth       = sidebarWidth - sidebarPadding - sidebarRightPadding - sidebarScrollbarGutter
)

type componentExample struct {
	Name, Category, Purpose string
	Coverage                []string
	Build                   func() []exampleBlock
}

type exampleBlock struct {
	Title, Description, Code string
	Preview                  dxui.View
}

func galleryView(app *dxui.App, state *galleryState, examples []componentExample, context dxui.LayoutContext) dxui.View {
	selected := examples[0]
	for _, example := range examples {
		if example.Name == state.selected {
			selected = example
			break
		}
	}
	if context.Width < 760 {
		return dxui.Box(dxui.BoxProps{
			Key: "showcase", Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Fill(), Background: dxui.TokenColor(dxui.ColorSemanticSurface), Overflow: dxui.OverflowClip},
			Align: dxui.AlignStretch,
		}, compactNavigation(app, state, examples), detailPane(state, selected))
	}
	return dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
		Key: "showcase", Style: dxui.Style{
			Width: dxui.Fill(), Height: dxui.Fill(),
			Background: dxui.TokenColor(dxui.ColorSemanticSurface), Overflow: dxui.OverflowClip,
		},
		Align: dxui.AlignStretch,
	}, sidebar(app, state, examples), detailPane(state, selected))
}

func compactNavigation(app *dxui.App, state *galleryState, examples []componentExample) dxui.View {
	options := make([]dxui.SelectOption, len(examples))
	for index, example := range examples {
		options[index] = dxui.SelectOption{Value: example.Name, Label: example.Name}
	}
	return dxui.Box(dxui.BoxProps{
		Key: "compact-navigation", Token: dxui.ComponentPanel,
		Style: dxui.Style{Width: dxui.Fill(), Padding: dxui.PaddingXY(12, 10), Radius: dxui.Round(0), Shrink: dxui.NoShrink()},
		Gap:   8, Align: dxui.AlignStretch,
	},
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8, Align: dxui.AlignCenter},
			dxui.Select(dxui.SelectProps{Key: "compact-page", Style: dxui.Style{Grow: 1, MinWidth: dxui.Px(0)}, Value: state.selected, Options: options, OnChange: dxui.Assign(&state.selected)}),
			dxui.ToggleSwitch(dxui.ToggleSwitchProps{Key: "compact-theme", Checked: state.dark, OnChange: func(value bool) {
				state.dark = value
				if err := app.SetTheme(showcaseTheme(value, state.theme)); err != nil {
					state.feedback = "Theme error: " + err.Error()
				}
			}}),
		),
		dxui.Input(dxui.InputProps{Key: "compact-search", Style: dxui.Style{Width: dxui.Fill()}, Value: state.query, Placeholder: "Search by name...", OnChange: dxui.Assign(&state.query)}),
	)
}

func sidebar(app *dxui.App, state *galleryState, examples []componentExample) dxui.View {
	items := make([]dxui.View, 0, len(examples)+4)
	query := strings.ToLower(strings.TrimSpace(state.query))
	category := ""
	menuItems := make([]dxui.MenuItem, 0, len(examples))
	appendMenu := func() {
		if len(menuItems) == 0 {
			return
		}
		items = append(items,
			smallLabel(category),
			dxui.Menu(dxui.MenuProps{
				Key: "nav-" + category, Style: dxui.Style{
					Width: dxui.Px(sidebarMenuWidth), Shrink: dxui.NoShrink(),
				},
				Value: state.selected,
				Items: menuItems,
				OnAction: func(value string) {
					state.selected = value
					state.feedback = "Selected " + value + "."
				},
			}),
		)
		menuItems = nil
	}
	for _, example := range examples {
		if query != "" && !strings.Contains(strings.ToLower(example.Name), query) {
			continue
		}
		if category != "" && example.Category != category {
			appendMenu()
		}
		category = example.Category
		menuItems = append(menuItems, dxui.MenuItem{Value: example.Name, Label: example.Name})
	}
	appendMenu()
	if len(items) == 0 {
		items = append(items, bodyText("No component name matches the search."))
	}

	return dxui.Box(dxui.BoxProps{
		Key: "sidebar", Token: dxui.ComponentPanel, Style: dxui.Style{
			Width: dxui.Px(sidebarWidth), Height: dxui.Fill(), Shrink: dxui.NoShrink(),
			Padding:  dxui.Edges(sidebarPadding, sidebarRightPadding, sidebarPadding, sidebarPadding),
			Overflow: dxui.OverflowClip,
			Radius:   dxui.Round(0),
		},
		Gap: 10, Align: dxui.AlignStretch,
	},
		heading("dxui Showcase", 24),
		bodyText(fmt.Sprintf("%d gallery pages", len(examples))),
		dxui.Input(dxui.InputProps{
			Key: "component-search", Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Px(40), Shrink: dxui.NoShrink()},
			Value: state.query, Placeholder: "Search by name...",
			OnChange: dxui.Assign(&state.query),
			OnSubmit: func() { state.feedback = fmt.Sprintf("Search submitted: %q", state.query) },
		}),
		dxui.Scroll(dxui.ScrollProps{
			Key: "component-navigation", Style: dxui.Style{Grow: 1, MinHeight: dxui.Px(80)},
			Axis: dxui.ScrollVertical, Scrollbar: dxui.ScrollbarAuto,
		}, dxui.Box(dxui.BoxProps{Gap: 6, Align: dxui.AlignStretch}, items...)),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8, Align: dxui.AlignCenter},
			dxui.ToggleSwitch(dxui.ToggleSwitchProps{
				Key: "theme-switch", Checked: state.dark,
				OnChange: func(value bool) {
					state.dark = value
					if err := app.SetTheme(showcaseTheme(value, state.theme)); err != nil {
						state.feedback = "Theme error: " + err.Error()
						return
					}
					state.feedback = fmt.Sprintf("Theme changed: dark=%t", value)
				},
			}),
			bodyText("Dark theme"),
		),
	)
}

func detailPane(state *galleryState, example componentExample) dxui.View {
	content := []dxui.View{
		smallLabel(example.Category + " / SELECTED"),
		heading(example.Name, 34),
		bodyText(example.Purpose),
		feedbackBox(state.feedback),
	}
	if current, ok := currentValue(example.Name, state); ok {
		content = append(content, currentValueBox(current))
	}
	content = append(content,
		sectionTitle("Coverage"),
		bodyText(strings.Join(example.Coverage, "  •  ")),
	)
	for index, block := range example.Build() {
		content = append(content, exampleCard(index, block))
	}
	return dxui.Scroll(dxui.ScrollProps{
		Key: "component-detail", Style: dxui.Style{Grow: 1, MinWidth: dxui.Px(0), Height: dxui.Fill()},
		Axis: dxui.ScrollVertical, Scrollbar: dxui.ScrollbarAuto,
	}, dxui.Box(dxui.BoxProps{
		Style: dxui.Style{Padding: dxui.Padding(24)},
		Gap:   12, Align: dxui.AlignStretch,
	}, content...))
}

func currentValue(name string, state *galleryState) (string, bool) {
	switch name {
	case "Theme":
		return fmt.Sprintf("dark=%t, primary=%s, surface=%s, danger=%s, success=%s", state.dark, state.theme.primary, state.theme.surface, state.theme.danger, state.theme.success), true
	case "Icon":
		_, matches, page, pages := iconCatalogPage(state.iconQuery, state.iconPage)
		if matches == 0 {
			return fmt.Sprintf("query=%q, matches=0", state.iconQuery), true
		}
		return fmt.Sprintf("query=%q, matches=%d, page=%d/%d", state.iconQuery, matches, page+1, pages), true
	case "Image", "Avatar":
		return state.imageStatus, true
	case "Button":
		return fmt.Sprintf("press count=%d", state.pressCount), true
	case "Scroll":
		return fmt.Sprintf("offset={X: %.1f, Y: %.1f}", state.controlledScroll.X, state.controlledScroll.Y), true
	case "Input":
		return fmt.Sprintf("value=%q, selection=[%d,%d]", state.input, state.inputSelection.Start, state.inputSelection.End), true
	case "Textarea":
		return fmt.Sprintf("runes=%d, selection=[%d,%d]", len([]rune(state.textarea)), state.textareaSelection.Start, state.textareaSelection.End), true
	case "Select":
		return fmt.Sprintf("value=%q", state.selectValue), true
	case "Tabs":
		return fmt.Sprintf("value=%q", state.tabsValue), true
	case "Menu":
		return fmt.Sprintf("value=%q", state.menuValue), true
	case "Checkbox":
		return fmt.Sprintf("checked=%t", state.checkbox), true
	case "Radio":
		return fmt.Sprintf("value=%q", state.radioValue), true
	case "ToggleSwitch":
		return fmt.Sprintf("checked=%t", state.toggle), true
	case "Slider":
		return fmt.Sprintf("value=%.0f", state.slider), true
	case "Popover":
		return fmt.Sprintf("open=%t", state.popoverOpen), true
	default:
		return "", false
	}
}

func exampleCard(index int, block exampleBlock) dxui.View {
	children := []dxui.View{
		heading(block.Title, 20),
		bodyText(block.Description),
		dxui.Scroll(dxui.ScrollProps{Style: dxui.Style{Width: dxui.Fill()}, Axis: dxui.ScrollHorizontal}, dxui.Box(dxui.BoxProps{
			Token: dxui.ComponentPanel, Style: dxui.Style{
				Padding: dxui.Padding(14), MinHeight: dxui.Px(56), Overflow: dxui.OverflowClip,
			},
		}, block.Preview)),
		codeView(block.Code),
	}
	return dxui.Box(dxui.BoxProps{
		Key: fmt.Sprintf("example-%d", index), Token: dxui.ComponentPanel, Style: dxui.Style{
			Padding: dxui.Padding(16),
		},
		Gap: 10, Align: dxui.AlignStretch,
	}, children...)
}

func codeView(code string) dxui.View {
	lines := strings.Count(code, "\n") + 1
	height := float32(32 + lines*22)
	if height > 220 {
		height = 220
	}
	return dxui.Textarea(dxui.TextareaProps{
		Style: dxui.Style{
			Width: dxui.Fill(), Height: dxui.Px(height), Shrink: dxui.NoShrink(),
			Background: dxui.ColorRGBA(15, 23, 42, 255), Padding: dxui.Padding(10),
			Text: dxui.TextStyle{
				Families: []dxui.FontFamily{dxui.FontFamilyDefault}, Size: 16, LineHeight: 22,
				Color: dxui.ColorRGBA(226, 232, 240, 255),
			}},
		Value: code, ReadOnly: true, Wrap: dxui.TextNoWrap,
	})
}

func heading(value string, size float32) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{Text: dxui.TextStyle{
		Size: size, LineHeight: size * 1.25, Weight: dxui.WeightBold,
	}}, Value: value})
}

func sectionTitle(value string) dxui.View { return heading(value, 18) }

func bodyText(value string) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{Text: dxui.TextStyle{
		Size: 16, LineHeight: 22,
	}}, Value: value, Wrap: dxui.TextWrapWords})
}

func smallLabel(value string) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{Shrink: dxui.NoShrink(), Text: dxui.TextStyle{
		Size: 12, LineHeight: 18, Weight: dxui.WeightBold,
		Color: dxui.TokenColor(dxui.ColorSemanticAccent),
	}}, Value: strings.ToUpper(value)})
}

func feedbackBox(value string) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{
		Padding: dxui.Padding(10), Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi),
		Border: dxui.Stroke(1, dxui.TokenColor(dxui.ColorSemanticAccent)),
		Radius: dxui.Round(6), Text: dxui.TextStyle{Size: 13, LineHeight: 18},
	}, Value: "EVENT  " + value, Wrap: dxui.TextWrapWords})
}

func currentValueBox(value string) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{
		Padding: dxui.Padding(10), Background: dxui.TokenColor(dxui.ColorSemanticSurfaceHi),
		Radius: dxui.Round(6), Text: dxui.TextStyle{Size: 13, LineHeight: 18},
	}, Value: "CURRENT  " + value, Wrap: dxui.TextWrapWords})
}

func previewHorizontal(children ...dxui.View) dxui.View {
	return dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 12, Align: dxui.AlignCenter}, children...)
}

func swatch(label string, style dxui.Style) dxui.View {
	style.Width, style.Height, style.Shrink = dxui.Px(116), dxui.Px(52), dxui.NoShrink()
	style.Padding = dxui.Padding(8)
	return dxui.Text(dxui.TextProps{Style: style, Value: label, Wrap: dxui.TextWrapWords})
}
