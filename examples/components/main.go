// Command components is an interactive, screenshot-ready component studio.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"time"

	ui "github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon"
)

type gallery struct {
	app                                                      *ui.App
	dark, checked, notifications, popover                    bool
	name, search, note, platform, radio, tab, menu, feedback string
	slider                                                   float32
	art                                                      ui.ImageSource
}

func main() {
	dark := flag.Bool("dark", false, "start with the dark theme")
	software := flag.Bool("software", false, "use the software renderer")
	duration := flag.Duration("duration", 0, "close automatically after this duration (smoke testing)")
	flag.Parse()
	g := &gallery{dark: *dark, checked: true, notifications: true, name: "Alex Morgan", note: "A little idea. A beautiful desktop app.\nBuilt entirely in Go.", platform: "desktop", radio: "stable", tab: "preview", menu: "components", slider: 68, feedback: "Ready to create. Try any component.", art: artwork()}
	renderer := ui.RendererAuto
	if *software {
		renderer = ui.RendererSoftware
	}
	opts := ui.AppOptions{Title: "dxui — Component Studio", Width: 1380, Height: 920, MinWidth: 1000, MinHeight: 700, Theme: theme(g.dark), Renderer: renderer}
	opts.OnShown = func(app *ui.App) {
		log.Print("gallery ready")
		if *duration > 0 {
			go func() {
				time.Sleep(*duration)
				if err := app.Update(app.Close); err != nil {
					log.Print(err)
				}
			}()
		}
	}
	g.app = ui.NewApp(opts)
	if err := g.app.Run(g.view); err != nil {
		log.Fatal(err)
	}
}

func theme(dark bool) ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
	}
	t.Semantic.Colors[ui.Color.Semantic.Accent] = ui.ColorRGBA(8, 127, 140, 255)
	t.Semantic.Colors[ui.Color.Semantic.AccentHover] = ui.ColorRGBA(6, 105, 116, 255)
	t.Semantic.Colors[ui.Color.Semantic.FocusRing] = ui.ColorRGBA(8, 127, 140, 255)
	t.Semantic.Colors[ui.ColorSemanticMenuHover] = ui.ColorRGBA(8, 127, 140, 42)
	t.Semantic.Colors[ui.ColorSemanticMenuActive] = ui.ColorRGBA(8, 127, 140, 26)
	t.Semantic.Colors[ui.ColorSemanticMenuSelected] = ui.ColorRGBA(8, 127, 140, 82)
	t.Semantic.Colors[ui.ColorSemanticMenuPressed] = ui.ColorRGBA(8, 127, 140, 72)
	if dark {
		t.Semantic.Colors[ui.Color.Semantic.Accent] = ui.ColorRGBA(69, 190, 200, 255)
		t.Semantic.Colors[ui.Color.Semantic.AccentHover] = ui.ColorRGBA(105, 211, 218, 255)
	}
	return t
}

func (g *gallery) color(light, dark uint32) ui.ColorValue {
	if g.dark {
		light = dark
	}
	return ui.ColorRGBA(uint8(light>>16), uint8(light>>8), uint8(light), 255)
}
func (g *gallery) ink() ui.ColorValue   { return g.color(0x203438, 0xe9f4f4) }
func (g *gallery) muted() ui.ColorValue { return g.color(0x6d8185, 0xa2b9bd) }
func text(s string, size float32, color ui.ColorValue, bold bool) ui.View {
	w := ui.WeightRegular
	if bold {
		w = ui.WeightBold
	}
	return ui.Text(ui.TextProps{Value: s, Style: ui.Style{Text: ui.TextStyle{Size: size, LineHeight: size + 6, Weight: w, Color: color}, Shrink: ui.NoShrink()}})
}
func row(gap float32, children ...ui.View) ui.View {
	return ui.Box(ui.BoxProps{Direction: ui.Horizontal, Gap: gap, Align: ui.AlignCenter, Style: ui.Style{Width: ui.Fill(), Shrink: ui.NoShrink()}}, children...)
}
func col(gap float32, children ...ui.View) ui.View {
	return ui.Box(ui.BoxProps{Gap: gap, Align: ui.AlignStretch, Style: ui.Style{Shrink: ui.NoShrink()}}, children...)
}
func space() ui.View                      { return ui.Box(ui.BoxProps{Style: ui.Style{Grow: 1}}) }
func (g *gallery) label(s string) ui.View { return text(s, 16, g.muted(), false) }
func (g *gallery) badge(s string, fg, bg ui.ColorValue) ui.View {
	return ui.Badge(ui.BadgeProps{Style: ui.Style{Background: bg, Radius: ui.Round(6), Padding: ui.PaddingXY(9, 4)}}, text(s, 11, fg, true))
}
func (g *gallery) action(s string) func() { return func() { g.feedback = s } }
func (g *gallery) card(number, title, caption string, height float32, children ...ui.View) ui.View {
	items := []ui.View{row(10, text(number, 11, g.color(0x087f8c, 0x69d3da), true), text(title, 17, g.ink(), true)), g.label(caption)}
	items = append(items, children...)
	return ui.Box(ui.BoxProps{Gap: 10, Align: ui.AlignStretch, Style: ui.Style{Width: ui.Fill(), Height: ui.Px(height), Padding: ui.Padding(20), Background: g.color(0xffffff, 0x1c2b30), Border: ui.Stroke(1, g.color(0xdfeaec, 0x30464b)), Radius: ui.Round(16), Shrink: ui.NoShrink()}}, items...)
}

func (g *gallery) view() ui.View {
	accent := g.color(0x087f8c, 0x69d3da)
	left := ui.Box(ui.BoxProps{Gap: 16, Align: ui.AlignStretch, Style: ui.Style{Width: ui.Px(428), Shrink: ui.NoShrink()}}, g.actions(), g.selection())
	middle := ui.Box(ui.BoxProps{Gap: 16, Align: ui.AlignStretch, Style: ui.Style{Width: ui.Px(428), Shrink: ui.NoShrink()}}, g.forms(), g.navigation())
	right := ui.Box(ui.BoxProps{Gap: 16, Align: ui.AlignStretch, Style: ui.Style{Width: ui.Px(428), Shrink: ui.NoShrink()}}, g.media(), g.collections())
	page := ui.Box(ui.BoxProps{Gap: 20, Align: ui.AlignStretch, Style: ui.Style{Width: ui.Px(1380), Padding: ui.PaddingXY(32, 26)}},
		row(12,
			ui.Box(ui.BoxProps{Style: ui.Style{Width: ui.Px(42), Height: ui.Px(42), Background: ui.ColorRGBA(8, 127, 140, 255), Radius: ui.Round(12)}, Align: ui.AlignCenter, Justify: ui.JustifyCenter}, ui.Icon(ui.IconProps{Data: icon.Layers(), Size: 24, Color: ui.ColorRGBA(255, 255, 255, 255)})),
			text("dxui", 28, g.ink(), true), text("/  COMPONENT STUDIO", 18, g.muted(), false), space(),
			g.badge("GO NATIVE", accent, g.color(0xe3f3f4, 0x193e43)),
			ui.Icon(ui.IconProps{Data: icon.Sun(), Size: 17, Color: g.muted()}),
			ui.ToggleSwitch(ui.ToggleSwitchProps{Key: "theme", Checked: g.dark, OnChange: func(v bool) {
				g.dark = v
				if err := g.app.SetTheme(theme(v)); err != nil {
					g.feedback = err.Error()
				}
			}}),
			ui.Icon(ui.IconProps{Data: icon.Moon(), Size: 17, Color: g.muted()})),
		row(20, col(4, text("Small footprint. Big possibilities.", 24, g.ink(), true), text("A declarative desktop GUI for Go. Every component below is live.", 14, g.muted(), false)), space(),
			col(5, text("Pure Go UI", 16, accent, true), g.label("No cgo required to build"))),
		ui.Box(ui.BoxProps{Direction: ui.Horizontal, Gap: 16, Align: ui.AlignStart}, left, middle, right),
		row(8, ui.Icon(ui.IconProps{Data: icon.CircleCheck(), Size: 16, Color: accent}), text(g.feedback, 16, g.muted(), false), space(), text("github.com/dxui-org/dxui", 16, g.ink(), true)),
	)
	return ui.Scroll(ui.ScrollProps{Key: "page", Axis: ui.ScrollBoth, Style: ui.Style{Width: ui.Fill(), Height: ui.Fill(), Background: g.color(0xf2f7f7, 0x122025)}}, page)
}

func (g *gallery) actions() ui.View {
	b := func(s string, v ui.ButtonVariant, t ui.ButtonTone) ui.View {
		return ui.TextButton(ui.ButtonProps{Variant: v, Tone: t, OnPress: g.action(s + " button pressed")}, s)
	}
	return g.card("01", "Actions", "Button  /  ButtonGroup  /  Tooltip", 306,
		row(8, b("Create project", ui.ButtonFilled, ui.ButtonPrimary), b("Preview", ui.ButtonOutline, ui.ButtonPrimary),
			ui.Tooltip(ui.TooltipProps{Key: "add-tip", Placement: ui.OverlayTop, Delay: 200 * time.Millisecond},
				ui.Button(ui.ButtonProps{Key: "add-with-icon", Variant: ui.ButtonSoft, Tone: ui.ButtonPrimary, OnPress: g.action("Add button with icon pressed")},
					ui.Box(ui.BoxProps{Direction: ui.Horizontal, Gap: 6, Align: ui.AlignCenter}, ui.Icon(ui.IconProps{Data: icon.Plus(), Size: 18}), ui.Label("Add"))),
				ui.Label("Add a new item"))),
		row(8, b("Success", ui.ButtonSoft, ui.ButtonSuccess), b("Info", ui.ButtonSoft, ui.ButtonInfo), b("Warning", ui.ButtonSoft, ui.ButtonWarn), b("Delete", ui.ButtonSoft, ui.ButtonDanger)),
		row(8, b("Dash", ui.ButtonDashed, ui.ButtonPrimary), b("Ghost", ui.ButtonGhost, ui.ButtonPrimary), b("Link", ui.ButtonLink, ui.ButtonPrimary), ui.TextButton(ui.ButtonProps{Disabled: true}, "Disabled")),
		g.label("A connected group. Independent actions."),
		row(10, ui.ButtonGroup(ui.ButtonGroupProps{Dividers: true}, b("Day", ui.ButtonOutline, ui.ButtonPrimary), b("Week", ui.ButtonOutline, ui.ButtonPrimary), b("Month", ui.ButtonOutline, ui.ButtonPrimary)), space(),
			ui.Tooltip(ui.TooltipProps{Key: "tip", Placement: ui.OverlayTop, Delay: 200 * time.Millisecond}, ui.Button(ui.ButtonProps{Variant: ui.ButtonGhost, OnPress: g.action("Helpful context, right where you need it.")}, ui.Icon(ui.IconProps{Data: icon.Info(), Size: 20})), ui.Label("Made with real dxui components"))),
	)
}

func (g *gallery) forms() ui.View {
	return g.card("02", "Form essentials", "Input  /  InputGroup  /  Select  /  Textarea", 306,
		ui.InputGroup(ui.InputGroupProps{}, ui.InputGroupContent{Prefix: ui.Some(ui.Icon(ui.IconProps{Data: icon.Search(), Size: 17})), Input: ui.Input(ui.InputProps{Key: "search", Value: g.search, Placeholder: "Search your next idea...", OnChange: ui.Assign(&g.search)})}),
		ui.Input(ui.InputProps{Key: "name", Value: g.name, Placeholder: "Your name", OnChange: ui.Assign(&g.name)}),
		ui.Select(ui.SelectProps{Key: "platform", Value: g.platform, Options: []ui.SelectOption{{Value: "desktop", Label: "Desktop application"}, {Value: "tool", Label: "Developer tool"}, {Value: "dashboard", Label: "Local dashboard"}}, OnChange: ui.Assign(&g.platform)}),
		ui.Textarea(ui.TextareaProps{Key: "note", Value: g.note, Wrap: ui.TextWrapWords, OnChange: ui.Assign(&g.note), Style: ui.Style{Height: ui.Px(76)}}),
	)
}

func (g *gallery) selection() ui.View {
	return g.card("04", "Make it yours", "Checkbox  /  Radio  /  ToggleSwitch  /  Slider", 300,
		row(10, ui.Checkbox(ui.CheckboxProps{Key: "checked", Checked: g.checked, OnChange: ui.Assign(&g.checked)}, text("Keep things simple", 13, g.ink(), false)), space(), ui.ToggleSwitch(ui.ToggleSwitchProps{Key: "notifications", Checked: g.notifications, OnChange: ui.Assign(&g.notifications)})),
		row(18, ui.Radio(ui.RadioProps{Key: "stable", Selected: g.radio == "stable", OnSelect: func() { g.radio = "stable" }}, ui.Label("Stable")), ui.Radio(ui.RadioProps{Key: "preview", Selected: g.radio == "preview", OnSelect: func() { g.radio = "preview" }}, ui.Label("Preview"))),
		row(8, g.label("Creative energy"), space(), text(fmt.Sprintf("%.0f%%", g.slider), 13, g.ink(), true)),
		ui.Slider(ui.SliderProps{Key: "slider", Min: 0, Max: 100, Value: g.slider, OnChange: ui.Assign(&g.slider)}),
		row(8, g.label("ProgressBar"), space(), g.badge("IN PROGRESS", g.color(0x087f8c, 0x69d3da), g.color(0xe3f3f4, 0x193e43))),
		ui.ProgressBar(ui.ProgressBarProps{Value: g.slider / 100, Style: ui.Style{Height: ui.Px(8), Radius: ui.Round(4)}}),
	)
}

func (g *gallery) navigation() ui.View {
	tabCopy := map[string]string{"preview": "Your next desktop app starts here.", "code": "Compose views with typed Go APIs.", "details": "State stays in your application."}
	return g.card("05", "Find your flow", "Tabs  /  Menu  /  Popover", 300,
		ui.Tabs(ui.TabsProps{Key: "tabs", Value: g.tab, Items: []ui.TabItem{{Value: "preview", Label: "Preview"}, {Value: "code", Label: "Code"}, {Value: "details", Label: "Details"}}, OnChange: ui.Assign(&g.tab)}),
		text(tabCopy[g.tab], 12, g.muted(), false),
		ui.Menu(ui.MenuProps{Key: "menu", Value: g.menu, Orientation: ui.MenuHorizontal, Items: []ui.MenuItem{{Value: "components", Label: "Components"}, {Value: "themes", Label: "Themes"}}, OnAction: func(s string) { g.menu = s; g.feedback = "Menu selected: " + s }}),
		ui.Popover(ui.PopoverProps{Key: "popover", Open: g.popover, Placement: ui.OverlayTop, Offset: ui.Metric(8), OnOpenChange: ui.Assign(&g.popover)},
			row(8, ui.Icon(ui.IconProps{Data: icon.SlidersHorizontal(), Size: 17}), ui.Label("Open quick settings")),
			col(12, text("A little more control", 16, g.ink(), true), g.label("An interactive, window-level popover."), ui.TextButton(ui.ButtonProps{Tone: ui.ButtonPrimary, OnPress: func() { g.popover = false; g.feedback = "Settings applied" }}, "Done"))),
		g.label("Click settings. Hover the info icon for a tooltip."),
	)
}

func (g *gallery) media() ui.View {
	return g.card("03", "A little personality", "Image  /  Avatar  /  Icon  /  Badge", 306,
		ui.Image(ui.ImageProps{Source: g.art, Fit: ui.ImageCover, Alignment: ui.Point{X: .5, Y: .5}, Style: ui.Style{Width: ui.Fill(), MinWidth: ui.Px(0), Height: ui.Px(108), MinHeight: ui.Px(0), MaxHeight: ui.Px(108), Radius: ui.Round(10), Overflow: ui.OverflowClip}}),
		row(10, ui.Avatar(ui.AvatarProps{Source: g.art, Size: 38}), col(0, text("Designed to feel at home", 14, g.ink(), true), g.label("Native windows. Your visual language."))),
		row(8, g.badge("OPEN SOURCE", g.color(0x187650, 0x8ce1b7), g.color(0xe4f5ec, 0x213e34)), g.badge("MADE IN GO", g.color(0x087f8c, 0x69d3da), g.color(0xe3f3f4, 0x193e43)), space(), ui.Icon(ui.IconProps{Data: icon.Heart(), Size: 18, Color: g.color(0xe17a96, 0xf5a0ba)}), ui.Icon(ui.IconProps{Data: icon.Sparkles(), Size: 18, Color: g.color(0x087f8c, 0x69d3da)})),
	)
}

func (g *gallery) collections() ui.View {
	// Cached virtual rows use literal palette colors, so a theme change also
	// changes their data version and rebuilds the visible rows.
	version := uint64(1)
	if g.dark {
		version = 2
	}
	return g.card("06", "Room to grow", "VirtualList  /  Scroll  /  Box  /  Text", 300,
		row(8, text("Project library", 13, g.ink(), true), space(), g.badge("100,000 ROWS", g.muted(), g.color(0xeef5f5, 0x293e43))),
		ui.VirtualList(ui.VirtualListProps{Key: "projects", Count: 100_000, Version: version, RowHeight: 38, Overscan: 2, Style: ui.Style{Height: ui.Px(126)}, ItemKey: func(i int) string { return fmt.Sprintf("project-%d", i) }, Build: func(i int) ui.View {
			names := []string{"Component explorer", "Notes & little ideas", "Your next big thing", "Desktop dashboard", "A tiny developer tool"}
			return ui.Box(ui.BoxProps{Direction: ui.Horizontal, Gap: 10, Align: ui.AlignCenter, Style: ui.Style{Height: ui.Px(38), Padding: ui.PaddingXY(8, 4), Background: g.color(0xf5f9f9, 0x22343a), Radius: ui.Round(6)}}, ui.Icon(ui.IconProps{Data: icon.Folder(), Size: 17, Color: g.muted()}), text(fmt.Sprintf("%s  %02d", names[i%len(names)], i+1), 13, g.ink(), false), space(), ui.Icon(ui.IconProps{Data: icon.ChevronRight(), Size: 14, Color: g.muted()}))
		}}),
		g.label("Scroll the list. Only visible rows are mounted."),
		row(6, ui.Icon(ui.IconProps{Data: icon.Code(), Size: 16, Color: g.muted()}), text("go get github.com/dxui-org/dxui", 16, g.muted(), false)),
	)
}

// A small deterministic landscape supplies an offline Image and Avatar asset.
func artwork() ui.ImageSource {
	im := image.NewNRGBA(image.Rect(0, 0, 720, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 720; x++ {
			t := float64(y) / 240
			c := color.NRGBA{uint8(204 - 52*t), uint8(236 - 35*t), uint8(237 - 31*t), 255}
			if math.Hypot(float64(x-553), float64(y-59)) < 29 {
				c = color.NRGBA{255, 236, 207, 255}
			}
			if float64(y) > 130+30*math.Sin(float64(x)/108) {
				c = color.NRGBA{101, 174, 179, 255}
			}
			if float64(y) > 166+35*math.Sin(float64(x)/132+2) {
				c = color.NRGBA{8, 127, 140, 255}
			}
			if float64(y) > 210+25*math.Sin(float64(x)/93+4) {
				c = color.NRGBA{7, 79, 89, 255}
			}
			im.SetNRGBA(x, y, c)
		}
	}
	return ui.ImageFromGo(im)
}
