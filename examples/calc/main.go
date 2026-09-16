// Command calc is a four-function calculator styled after the supplied
// light/dark calculator reference. It uses dxui's root API and official Lucide resources.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon"
)

const (
	colorCanvas     dxui.ColorToken = "calc.canvas"
	colorShell      dxui.ColorToken = "calc.shell"
	colorDisplay    dxui.ColorToken = "calc.display"
	colorKey        dxui.ColorToken = "calc.key"
	colorUtility    dxui.ColorToken = "calc.utility"
	colorAccent     dxui.ColorToken = "calc.accent"
	colorAccentOver dxui.ColorToken = "calc.accent-hover"
	colorText       dxui.ColorToken = "calc.text"
	colorMuted      dxui.ColorToken = "calc.muted"
	colorHandle     dxui.ColorToken = "calc.handle"
	colorMode       dxui.ColorToken = "calc.mode"

	componentKey     dxui.ComponentToken = "calc.key"
	componentUtility dxui.ComponentToken = "calc.utility-key"
	componentAccent  dxui.ComponentToken = "calc.accent-key"
	componentMode    dxui.ComponentToken = "calc.mode-button"
)

var backspaceIcon = icon.Delete()
var moonIcon = icon.Moon()
var sunIcon = icon.Sun()

type calculatorKey struct {
	ID, Label string
	Token     dxui.ComponentToken
}

func main() {
	dark := flag.Bool("dark", false, "start with the dark appearance")
	software := flag.Bool("software", false, "force SDL's named software renderer")
	duration := flag.Duration("duration", 0, "close automatically after this duration")
	flag.Parse()

	state := newCalculator()
	state.dark = *dark
	renderer := dxui.RendererAuto
	if *software {
		renderer = dxui.RendererSoftware
	}
	app := dxui.NewApp(dxui.AppOptions{
		Title: "dxui Calculator", Width: 392, Height: 760,
		MinWidth: 392, MinHeight: 760,
		Renderer: renderer, Theme: calculatorTheme(state.dark),
		Background: dxui.RGBA(184, 225, 249, 255),
		Shortcuts:  calculatorShortcuts(state),
		OnShown: func(app *dxui.App) {
			if *duration > 0 {
				go closeCalculatorAfter(app, *duration)
			}
		},
	})
	if err := app.Run(func() dxui.View { return calculatorView(app, state) }); err != nil {
		log.Fatal(err)
	}
}

func closeCalculatorAfter(app *dxui.App, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	<-timer.C
	if err := app.Update(app.Close); err != nil {
		log.Printf("automatic close: %v", err)
	}
}

func calculatorShortcuts(state *calculator) []dxui.Shortcut {
	bind := func(key dxui.ShortcutKey, label string) dxui.Shortcut {
		return dxui.Shortcut{Key: key, OnPress: func() { state.input(label) }}
	}
	return []dxui.Shortcut{
		bind(dxui.Key0, "0"), bind(dxui.Key1, "1"), bind(dxui.Key2, "2"), bind(dxui.Key3, "3"), bind(dxui.Key4, "4"),
		bind(dxui.Key5, "5"), bind(dxui.Key6, "6"), bind(dxui.Key7, "7"), bind(dxui.Key8, "8"), bind(dxui.Key9, "9"),
		bind(dxui.KeyPlus, "+"), bind(dxui.KeyMinus, "-"), bind(dxui.KeyMultiply, "x"), bind(dxui.KeyDivide, "/"),
		bind(dxui.KeyDecimal, "."), bind(dxui.KeyEnter, "="), bind(dxui.KeyEquals, "="), bind(dxui.KeyBackspace, "backspace"),
	}
}

func calculatorView(app *dxui.App, state *calculator) dxui.View {
	mode := "DARK MODE"
	modeIcon := moonIcon
	if state.dark {
		mode = "LIGHT MODE"
		modeIcon = sunIcon
	}
	modeButton := dxui.Button(dxui.ButtonProps{
		Key: "theme", Token: componentMode, Style: dxui.Style{
			Width: dxui.Px(126), Height: dxui.Px(38),
		},
		OnPress: func() {
			state.dark = !state.dark
			if err := app.SetTheme(calculatorTheme(state.dark)); err != nil {
				log.Printf("theme switch: %v", err)
			}
		},
	}, centeredRow(7,
		dxui.Icon(dxui.IconProps{Data: modeIcon, Size: 17, Color: dxui.TokenColor(colorAccent)}),
		dxui.Text(dxui.TextProps{Value: mode, Style: dxui.Style{Text: dxui.TextStyle{
			Size: 11, LineHeight: 16, Weight: dxui.WeightBold, Align: dxui.TextCenter,
		}}}),
	))

	menuDots := dxui.Box(dxui.BoxProps{Gap: 3, Align: dxui.AlignCenter},
		dot("menu-dot-1"), dot("menu-dot-2"), dot("menu-dot-3"),
	)
	header := dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
		Style: dxui.Style{Width: dxui.Fill()},
		Align: dxui.AlignCenter, Justify: dxui.JustifySpaceBetween,
	}, modeButton, menuDots)

	expression := state.expression
	if expression == "" {
		expression = " "
	}
	readout := dxui.Box(dxui.BoxProps{
		Style: dxui.Style{Width: dxui.Fill()},
		Gap:   4, Align: dxui.AlignEnd,
	},
		readoutText(expression, 24, 30, dxui.WeightRegular, colorMuted),
		readoutText(state.display, 50, 58, dxui.WeightMedium, colorAccent),
	)

	display := dxui.Box(dxui.BoxProps{
		Style: dxui.Style{
			Width: dxui.Fill(), Height: dxui.Px(260), Padding: dxui.Padding(26),
			Background: dxui.TokenColor(colorDisplay), Radius: dxui.CornerValues{
				BottomRight: dxui.Metric(42), BottomLeft: dxui.Metric(42),
			},
		},
		Justify: dxui.JustifySpaceBetween, Align: dxui.AlignStretch,
	}, header, readout)

	keys := [][]calculatorKey{
		{{ID: "clear", Label: "C", Token: componentUtility}, {ID: "sign", Label: "{}", Token: componentUtility}, {ID: "percent", Label: "%", Token: componentUtility}, {ID: "divide", Label: "/", Token: componentAccent}},
		{{ID: "seven", Label: "7", Token: componentKey}, {ID: "eight", Label: "8", Token: componentKey}, {ID: "nine", Label: "9", Token: componentKey}, {ID: "multiply", Label: "x", Token: componentAccent}},
		{{ID: "four", Label: "4", Token: componentKey}, {ID: "five", Label: "5", Token: componentKey}, {ID: "six", Label: "6", Token: componentKey}, {ID: "subtract", Label: "-", Token: componentAccent}},
		{{ID: "one", Label: "1", Token: componentKey}, {ID: "two", Label: "2", Token: componentKey}, {ID: "three", Label: "3", Token: componentKey}, {ID: "add", Label: "+", Token: componentAccent}},
		{{ID: "zero", Label: "0", Token: componentKey}, {ID: "decimal", Label: ".", Token: componentKey}, {ID: "backspace", Label: "backspace", Token: componentKey}, {ID: "equals", Label: "=", Token: componentAccent}},
	}
	rows := make([]dxui.View, 0, len(keys))
	for _, row := range keys {
		buttons := make([]dxui.View, 0, len(row))
		for _, key := range row {
			buttons = append(buttons, calculatorButton(key, state))
		}
		rows = append(rows, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
			Style: dxui.Style{Width: dxui.Fill()},
			Gap:   12, Justify: dxui.JustifySpaceBetween,
		}, buttons...))
	}

	handle := dxui.Text(dxui.TextProps{Style: dxui.Style{
		Width: dxui.Px(38), Height: dxui.Px(5), Background: dxui.TokenColor(colorHandle),
		Radius: dxui.Round(3),
	}})
	keypadChildren := []dxui.View{handle}
	keypadChildren = append(keypadChildren, rows...)
	keypad := dxui.Box(dxui.BoxProps{
		Style: dxui.Style{
			Width: dxui.Fill(), Grow: 1,
			Padding: dxui.Edges(28, 22, 24, 22),
		},
		Gap: 12, Align: dxui.AlignCenter, Justify: dxui.JustifySpaceBetween,
	}, keypadChildren...)

	shell := dxui.Box(dxui.BoxProps{
		Style: dxui.Style{
			Width: dxui.Fill(), Height: dxui.Fill(), Background: dxui.TokenColor(colorShell),
			Radius: dxui.Corners(42, 42, 0, 0), Overflow: dxui.OverflowClip,
		},
	}, display, keypad)

	return dxui.Box(dxui.BoxProps{
		Key: "calculator-page", Style: dxui.Style{
			Width: dxui.Fill(), Height: dxui.Fill(),
			Background: dxui.TokenColor(colorCanvas),
		},
		Align: dxui.AlignCenter, Justify: dxui.JustifyCenter,
	}, shell)
}

func calculatorButton(key calculatorKey, state *calculator) dxui.View {
	content := dxui.View{}
	if key.Label == "backspace" {
		content = dxui.Icon(dxui.IconProps{Data: backspaceIcon, Size: 27, Color: dxui.TokenColor(colorText)})
	} else {
		content = dxui.Text(dxui.TextProps{Value: key.Label, Style: dxui.Style{Text: dxui.TextStyle{
			Size: 29, LineHeight: 34, Weight: dxui.WeightMedium, Align: dxui.TextCenter,
		}}})
	}
	return dxui.Button(dxui.ButtonProps{
		Key: key.ID, Token: key.Token,
		Style:   dxui.Style{Width: dxui.Px(72), Height: dxui.Px(68)},
		OnPress: func() { state.input(key.Label) },
	}, centeredRow(0, content))
}

func centeredRow(gap float32, children ...dxui.View) dxui.View {
	return dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal,
		Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Fill()},
		Gap:   gap, Align: dxui.AlignCenter, Justify: dxui.JustifyCenter,
	}, children...)
}

func readoutText(value string, size, lineHeight float32, weight dxui.FontWeight, color dxui.ColorToken) dxui.View {
	return dxui.Text(dxui.TextProps{Value: value, Style: dxui.Style{Text: dxui.TextStyle{
		Size: size, LineHeight: lineHeight, Weight: weight,
		Color: dxui.TokenColor(color), Align: dxui.TextEnd,
	}}})
}

func dot(key string) dxui.View {
	return dxui.Text(dxui.TextProps{Key: key, Style: dxui.Style{
		Width: dxui.Px(5), Height: dxui.Px(5), Background: dxui.TokenColor(colorText),
		Radius: dxui.Round(3),
	}})
}

func calculatorTheme(dark bool) dxui.Theme {
	theme := dxui.LightTheme()
	colors := map[dxui.ColorToken]dxui.RGBAColor{
		colorCanvas:     dxui.RGBA(184, 225, 249, 255),
		colorShell:      dxui.RGBA(255, 255, 255, 255),
		colorDisplay:    dxui.RGBA(231, 243, 255, 255),
		colorKey:        dxui.RGBA(255, 255, 255, 255),
		colorUtility:    dxui.RGBA(224, 228, 236, 255),
		colorAccent:     dxui.RGBA(54, 169, 241, 255),
		colorAccentOver: dxui.RGBA(35, 151, 226, 255),
		colorText:       dxui.RGBA(12, 18, 26, 255),
		colorMuted:      dxui.RGBA(102, 113, 126, 255),
		colorHandle:     dxui.RGBA(194, 194, 194, 255),
		colorMode:       dxui.RGBA(255, 255, 255, 235),
	}
	if dark {
		theme = dxui.DarkTheme()
		colors[colorShell] = dxui.RGBA(52, 67, 84, 255)
		colors[colorDisplay] = dxui.RGBA(85, 101, 119, 255)
		colors[colorKey] = dxui.RGBA(49, 64, 81, 255)
		colors[colorUtility] = dxui.RGBA(72, 87, 104, 255)
		colors[colorText] = dxui.RGBA(248, 250, 252, 255)
		colors[colorMuted] = dxui.RGBA(166, 177, 188, 255)
		colors[colorHandle] = dxui.RGBA(222, 222, 216, 255)
		colors[colorMode] = dxui.RGBA(111, 126, 140, 255)
	}
	for token, color := range colors {
		theme.Primitive.Colors[token] = color
	}

	button := func(background, foreground dxui.ColorToken, shadow bool) dxui.ComponentTheme {
		base := dxui.StylePatch{
			Background: dxui.Some(dxui.TokenColor(background)),
			Radius:     dxui.Some(dxui.Round(15)),
			TextColor:  dxui.Some(dxui.TokenColor(foreground)),
		}
		if shadow {
			base.Shadow = dxui.Some([]dxui.Shadow{{
				OffsetY: dxui.Metric(7), Blur: dxui.Metric(13),
				Color: dxui.ColorRGBA(68, 131, 166, 42),
			}})
		}
		return dxui.ComponentTheme{Base: base, States: dxui.StateStyles{
			Hover:   dxui.StylePatch{Opacity: dxui.Some(float32(.84))},
			Focus:   dxui.StylePatch{Border: dxui.Some(dxui.Stroke(2, dxui.TokenColor(colorAccent)))},
			Pressed: dxui.StylePatch{Opacity: dxui.Some(float32(.68))},
		}}
	}
	theme.Components[componentKey] = button(colorKey, colorText, true)
	theme.Components[componentUtility] = button(colorUtility, colorText, true)
	theme.Components[componentAccent] = button(colorAccent, dxui.Color.Primitive.White, false)
	theme.Components[componentAccent] = dxui.ComponentTheme{
		Base: theme.Components[componentAccent].Base,
		States: dxui.StateStyles{
			Hover:   dxui.StylePatch{Background: dxui.Some(dxui.TokenColor(colorAccentOver))},
			Focus:   dxui.StylePatch{Border: dxui.Some(dxui.Stroke(2, dxui.TokenColor(dxui.Color.Primitive.White)))},
			Pressed: dxui.StylePatch{Opacity: dxui.Some(float32(.7))},
		},
	}
	theme.Components[componentMode] = dxui.ComponentTheme{
		Base: dxui.StylePatch{
			Background: dxui.Some(dxui.TokenColor(colorMode)),
			Radius:     dxui.Some(dxui.Round(19)),
			TextColor:  dxui.Some(dxui.TokenColor(colorAccent)),
		},
		States: dxui.StateStyles{
			Hover:   dxui.StylePatch{Opacity: dxui.Some(float32(.82))},
			Focus:   dxui.StylePatch{Opacity: dxui.Some(float32(.9))},
			Pressed: dxui.StylePatch{Opacity: dxui.Some(float32(.68))},
		},
	}
	return theme
}
