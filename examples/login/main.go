// Command login is the dxui MVP reference application. It imports only the
// root package and exercises controlled fields, focus, overlays, themes, and
// the blocking application lifecycle.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/dxui-org/dxui"
)

const (
	primitiveBrand  dxui.ColorToken = "login.primitive.brand"
	primitiveCanvas dxui.ColorToken = "login.primitive.canvas"
	primitiveCard   dxui.ColorToken = "login.primitive.card"
)

type loginState struct {
	username string
	password string
	remember bool
	role     string
	dark     bool
}

func main() {
	software := flag.Bool("software", false, "force SDL's named software renderer")
	diagnostics := flag.Bool("diagnostics", false, "print renderer and pipeline diagnostics after the window closes")
	duration := flag.Duration("duration", 0, "close automatically after this package-smoke duration")
	flag.Parse()

	renderer := dxui.RendererAuto
	if *software {
		renderer = dxui.RendererSoftware
	}
	options := dxui.AppOptions{
		Title: "dxui Login", Width: 480, Height: 500,
		MinWidth: 480, MinHeight: 500,
		Renderer: renderer, Theme: loginTheme(false),
		Background:  dxui.RGBA(241, 245, 249, 255),
		Diagnostics: *diagnostics,
	}

	state := &loginState{role: "developer"}
	if *duration > 0 {
		options.OnShown = func(app *dxui.App) { go closeAfter(app, *duration) }
	}
	app := dxui.NewApp(options)

	if err := app.Run(func() dxui.View { return loginView(app, state) }); err != nil {
		log.Fatal(err)
	}
	if *diagnostics {
		d := app.Diagnostics()
		log.Printf("renderer=%s sdl=%s software-fallback=%t logical=%.0fx%.0f pixel=%.0fx%.0f density=%.2f scale=%.2f windows=%d renderer-attempts=%d renderer-creates=%d expose=%d resize=%d scale-events=%d viewport-noops=%d renderer-resets=%d frames=%d build=%d layout=%d paint=%d",
			d.RendererName, d.SDLVersion, d.SoftwareFallback,
			d.LogicalSize.Width, d.LogicalSize.Height,
			d.PixelSize.Width, d.PixelSize.Height,
			d.PixelDensity, d.DisplayScale,
			d.WindowCreates, d.RendererCreateAttempts, d.RendererCreates,
			d.ExposeEvents, d.ResizeEvents, d.ScaleEvents, d.NoopViewportEvents, d.RendererResetEvents,
			d.FrameCount, d.BuildCount, d.LayoutCount, d.PaintCount)
	}
}

func closeAfter(app *dxui.App, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	<-timer.C
	if err := app.Update(app.Close); err != nil {
		log.Printf("automatic close: %v", err)
	}
}

func loginView(app *dxui.App, state *loginState) dxui.View {
	login := func() {
		// Passwords never enter logs, diagnostics, or result text.
		log.Printf("login requested username=%q role=%q remember=%t", state.username, state.role, state.remember)
	}

	form := dxui.Box(dxui.BoxProps{
		Key: "login-card", Token: dxui.ComponentPanel, Style: dxui.Style{
			Width: dxui.Fill(), Height: dxui.Fill(), Padding: dxui.Padding(28),
			Border: dxui.NoBorder(), Radius: dxui.Round(0),
		},
		Gap: 12, Align: dxui.AlignStretch,
	},
		dxui.Text(dxui.TextProps{Value: "Welcome back", Style: dxui.Style{Text: dxui.TextStyle{
			Size: 30, LineHeight: 38, Weight: dxui.WeightBold,
		}}}),
		dxui.Label("Sign in to continue to dxui"),
		formField("Username", dxui.Input(dxui.InputProps{
			Value: state.username, Placeholder: "name@example.com",
			OnChange: dxui.Assign(&state.username),
		})).WithKey("username-field").WithStyle(dxui.Style{Width: dxui.Fill()}),
		formField("Password", dxui.Input(dxui.InputProps{
			Value: state.password, Password: true,
			Placeholder: "Enter your password",
			OnChange:    dxui.Assign(&state.password),
			OnSubmit:    login,
		})).WithKey("password-field").WithStyle(dxui.Style{Width: dxui.Fill()}),
		formField("Role", dxui.Select(dxui.SelectProps{
			Value: state.role, Placeholder: "Choose a role",
			Options: []dxui.SelectOption{
				{Value: "developer", Label: "Developer"},
				{Value: "designer", Label: "Designer"},
				{Value: "operator", Label: "Operator"},
			},
			OnChange: dxui.Assign(&state.role),
		})).WithKey("role-field").WithStyle(dxui.Style{Width: dxui.Fill()}),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Justify: dxui.JustifySpaceBetween, Align: dxui.AlignCenter},
			dxui.Checkbox(dxui.CheckboxProps{
				Key: "remember", Checked: state.remember,
				OnChange: dxui.Assign(&state.remember),
			}, dxui.Label("Remember me")),
			dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8, Align: dxui.AlignCenter},
				dxui.Label("Dark theme"),
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{
					Key: "theme", Checked: state.dark,
					OnChange: func(dark bool) {
						state.dark = dark
						if err := app.SetTheme(loginTheme(dark)); err != nil {
							log.Printf("theme switch: %v", err)
						}
					},
				}),
			),
		),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 10, Justify: dxui.JustifyEnd},
			dxui.TextButton(dxui.ButtonProps{Key: "cancel", Token: dxui.ComponentButtonSecondary, OnPress: app.Close}, "Cancel"),
			dxui.TextButton(dxui.ButtonProps{
				Key: "login", Disabled: state.username == "" || state.password == "", OnPress: login,
			}, "Login"),
		),
	)

	return form
}

func formField(label string, control dxui.View) dxui.View {
	return dxui.Box(dxui.BoxProps{Gap: 6, Align: dxui.AlignStretch},
		dxui.Text(dxui.TextProps{Value: label, Style: dxui.Style{Text: dxui.TextStyle{Weight: dxui.WeightMedium}}}),
		control,
	)
}

func loginTheme(dark bool) dxui.Theme {
	theme := dxui.LightTheme()
	canvas := dxui.RGBA(241, 245, 249, 255)
	card := dxui.RGBA(255, 255, 255, 255)
	brand := dxui.RGBA(37, 99, 235, 255)
	if dark {
		theme = dxui.DarkTheme()
		canvas = dxui.RGBA(15, 23, 42, 255)
		card = dxui.RGBA(30, 41, 59, 255)
		brand = dxui.RGBA(96, 165, 250, 255)
	}
	// Primitive literals feed semantic meanings; built-in ComponentButton and
	// ComponentPanel styles then consume those semantic tokens.
	theme.Primitive.Colors[primitiveBrand] = brand
	theme.Primitive.Colors[primitiveCanvas] = canvas
	theme.Primitive.Colors[primitiveCard] = card
	theme.Semantic.Colors[dxui.ColorSemanticAccent] = dxui.TokenColor(primitiveBrand)
	theme.Semantic.Colors[dxui.ColorSemanticSurface] = dxui.TokenColor(primitiveCanvas)
	theme.Semantic.Colors[dxui.ColorSemanticSurfaceHi] = dxui.TokenColor(primitiveCard)
	return theme
}
