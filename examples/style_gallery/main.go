// Command style_gallery demonstrates the M5 paint and theme slice using only
// the root dxui API.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/dxui-org/dxui"
)

func main() {
	dark := flag.Bool("dark", false, "start with the dark theme")
	software := flag.Bool("software", false, "force SDL's named software renderer")
	switchAfter := flag.Duration("switch-after", 0, "switch Light/Dark once after this duration")
	duration := flag.Duration("duration", 0, "close after this duration; zero runs until window close")
	flag.Parse()

	theme := dxui.LightTheme()
	next := dxui.DarkTheme()
	if *dark {
		theme, next = next, theme
	}
	renderer := dxui.RendererAuto
	if *software {
		renderer = dxui.RendererSoftware
	}
	options := dxui.AppOptions{
		Title: "dxui style gallery", Width: 760, Height: 520,
		Background: dxui.RGBA(14, 17, 23, 255), Theme: theme, Renderer: renderer,
	}
	options.OnShown = func(app *dxui.App) {
		if *switchAfter > 0 {
			go func(delay time.Duration) {
				timer := time.NewTimer(delay)
				defer timer.Stop()
				<-timer.C
				if err := app.Update(func() {
					if err := app.SetTheme(next); err != nil {
						log.Printf("theme switch: %v", err)
					}
				}); err != nil {
					log.Printf("theme update: %v", err)
				}
			}(*switchAfter)
		}
		if *duration > 0 {
			go func(delay time.Duration) {
				timer := time.NewTimer(delay)
				defer timer.Stop()
				<-timer.C
				if err := app.Update(app.Close); err != nil {
					log.Printf("close update: %v", err)
				}
			}(*duration)
		}
	}
	app := dxui.NewApp(options)
	if err := app.Run(gallery); err != nil {
		log.Fatal(err)
	}
	diagnostics := app.Diagnostics()
	log.Printf("renderer=%s frames=%d build=%d layout=%d paint=%d", diagnostics.RendererName, diagnostics.FrameCount, diagnostics.BuildCount, diagnostics.LayoutCount, diagnostics.PaintCount)
}

func gallery() dxui.View {
	card := func(child dxui.View) dxui.View {
		return dxui.Box(dxui.BoxProps{

			Token: dxui.ComponentPanel,
			Style: dxui.Style{
				Width: dxui.Px(330), Height: dxui.Px(190), Padding: dxui.Padding(16),
			},
		}, child)
	}

	roundedLayers := card(dxui.Box(dxui.BoxProps{Gap: 12},
		box(220, 54, dxui.RGBA(61, 126, 232, 255), 12),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 10},
			box(92, 72, dxui.RGBA(232, 111, 72, 255), 24),
			boxWithStyle(92, 72, dxui.Style{
				Background: dxui.ColorRGBA(96, 190, 132, 255),
				Border:     dxui.Stroke(3, dxui.ColorRGBA(230, 255, 239, 255)),
				Radius:     dxui.Round(10),
			}),
		),
	))

	clipAndOpacity := card(dxui.Box(dxui.BoxProps{
		Style: dxui.Style{
			Width: dxui.Px(220), Height: dxui.Px(130), Overflow: dxui.OverflowClip,
			Background: dxui.ColorRGBA(43, 48, 67, 255), Radius: dxui.Round(18), Opacity: dxui.Some(float32(.72)),
		},
	},
		boxWithStyle(190, 88, dxui.Style{
			Position: dxui.PositionAbsolute, Insets: dxui.Insets{Top: dxui.Px(30), Left: dxui.Px(80)}, ZIndex: 2,
			Background: dxui.ColorRGBA(176, 91, 213, 235), Radius: dxui.Round(28), Shadow: []dxui.Shadow{{OffsetY: dxui.Metric(4), Blur: dxui.Metric(14), Spread: dxui.Metric(2), Color: dxui.ColorRGBA(0, 0, 0, 90)}},
		}),
		boxWithStyle(100, 68, dxui.Style{
			Position: dxui.PositionAbsolute, Insets: dxui.Insets{Top: dxui.Px(8), Left: dxui.Px(12)}, ZIndex: 1,
			Background: dxui.ColorRGBA(243, 193, 73, 255), Radius: dxui.Round(8),
		}),
	))

	return dxui.Box(dxui.BoxProps{
		Style: dxui.Style{Padding: dxui.Padding(24)},
		Gap:   18,
	},
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 18}, roundedLayers, clipAndOpacity),
		borderExamples(),
		boxWithStyle(678, 72, dxui.Style{
			Background: dxui.TokenColor(dxui.ColorSemanticAccent), Radius: dxui.UniformCorners(dxui.TokenMetric(dxui.MetricSemanticRadius)),
			Shadow: []dxui.Shadow{{OffsetY: dxui.Metric(3), Blur: dxui.TokenMetric(dxui.MetricSemanticShadow), Color: dxui.ColorRGBA(0, 0, 0, 70)}},
		}),
	)
}

func borderExamples() dxui.View {
	color := dxui.TokenColor(dxui.ColorSemanticAccent)
	items := []struct {
		label  string
		border dxui.Border
		radius float32
	}{
		{"No theme border", dxui.NoBorder(), 10},
		{"Bottom solid", dxui.Border{Width: dxui.Metric(2), Color: color, Sides: dxui.BorderBottom}, 0},
		{"Left + right", dxui.Border{Width: dxui.Metric(2), Color: color, Sides: dxui.BorderLeft | dxui.BorderRight}, 0},
		{"Rounded dash", dxui.Border{Width: dxui.Metric(2), Color: color, Pattern: dxui.BorderDashed, Sides: dxui.BorderBottom}, 18},
	}
	children := make([]dxui.View, 0, len(items))
	for _, item := range items {
		children = append(children, dxui.Box(dxui.BoxProps{
			Token: dxui.ComponentPanel,
			Style: dxui.Style{Width: dxui.Px(156), Height: dxui.Px(68), Padding: dxui.Padding(10), Border: item.border, Radius: dxui.Round(item.radius)},
		}, dxui.Label(item.label)))
	}
	return dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 18}, children...)
}

func box(width, height float32, color dxui.RGBAColor, radius float32) dxui.View {
	return boxWithStyle(width, height, dxui.Style{
		Background: dxui.LiteralColor(color), Radius: dxui.Round(radius),
	})
}

func boxWithStyle(width, height float32, style dxui.Style) dxui.View {
	style.Width, style.Height = dxui.Px(width), dxui.Px(height)
	return dxui.Text(dxui.TextProps{Style: style})
}
