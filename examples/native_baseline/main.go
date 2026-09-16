// Command native_baseline provides deterministic native measurement scenes.
// It intentionally imports only dxui's root package and the Go standard library.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"sync"
	"time"

	"github.com/dxui-org/dxui"
)

type record struct {
	Kind        string                  `json:"kind"`
	AtMS        int64                   `json:"at_ms"`
	Operation   int                     `json:"operation,omitempty"`
	Diagnostics dxui.RuntimeDiagnostics `json:"diagnostics"`
}

type state struct {
	mu           sync.Mutex
	offset       float32
	value        int
	dark         bool
	remove       bool
	small, large dxui.ImageSource
}

func main() {
	scenario := flag.String("scenario", "minimal", "minimal, login, list, or images")
	software := flag.Bool("software", false, "force SDL's named software renderer")
	mode := flag.String("mode", "steady", "steady, idle, caret, or interact")
	duration := flag.Duration("duration", 30*time.Second, "measurement duration after OnShown")
	sample := flag.Duration("diagnostic-sample", time.Second, "diagnostic JSONL interval")
	outPath := flag.String("diagnostic-file", "", "diagnostic JSONL output path")
	flag.Parse()
	if *scenario != "minimal" && *scenario != "login" && *scenario != "list" && *scenario != "images" {
		log.Fatalf("unknown scenario %q", *scenario)
	}
	if *mode != "steady" && *mode != "idle" && *mode != "caret" && *mode != "interact" {
		log.Fatalf("unknown mode %q", *mode)
	}

	out := os.Stdout
	if *outPath != "" {
		file, err := os.Create(*outPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		out = file
	}
	w := bufio.NewWriter(out)
	defer w.Flush()
	started := time.Now()
	write := func(kind string, app *dxui.App, operation int) {
		_ = json.NewEncoder(w).Encode(record{Kind: kind, AtMS: time.Since(started).Milliseconds(), Operation: operation, Diagnostics: app.Diagnostics()})
		_ = w.Flush()
	}

	renderer := dxui.RendererAuto
	if *software {
		renderer = dxui.RendererSoftware
	}
	s := &state{}
	if *scenario == "images" {
		s.small = dxui.ImageFromGo(pattern(128, 128))
		s.large = dxui.ImageFromGo(pattern(1600, 1000))
	}
	options := dxui.AppOptions{Title: "dxui native baseline: " + *scenario, Width: 900, Height: 680, MinWidth: 560, MinHeight: 400, Renderer: renderer, Diagnostics: true}
	var app *dxui.App
	options.OnShown = func(shown *dxui.App) {
		write("shown", shown, 0)
		go drive(shown, s, *scenario, *mode, *duration, *sample, write)
	}
	app = dxui.NewApp(options)
	if err := app.Run(func() dxui.View { s.mu.Lock(); defer s.mu.Unlock(); return scene(app, s, *scenario, *mode) }); err != nil {
		log.Fatal(err)
	}
	write("stopped", app, 0)
}

func drive(app *dxui.App, s *state, scenario, mode string, duration, sample time.Duration, write func(string, *dxui.App, int)) {
	deadline := time.NewTimer(duration)
	defer deadline.Stop()
	ticker := time.NewTicker(sample)
	defer ticker.Stop()
	var actions *time.Ticker
	if mode == "interact" {
		actions = time.NewTicker(500 * time.Millisecond)
		defer actions.Stop()
	}
	operation := 0
	for {
		var action <-chan time.Time
		if actions != nil {
			action = actions.C
		}
		select {
		case <-ticker.C:
			write("sample", app, operation)
		case <-action:
			operation++
			err := app.Update(func() {
				s.mu.Lock()
				defer s.mu.Unlock()
				s.value = operation
				s.offset = float32((operation * 37) % 22000)
				s.dark = operation%8 >= 4
				s.remove = operation%20 >= 16
				if scenario == "login" {
					if s.dark {
						_ = app.SetTheme(dxui.DarkTheme())
					} else {
						_ = app.SetTheme(dxui.LightTheme())
					}
				}
			})
			if err != nil {
				write("update_error", app, operation)
				return
			}
		case <-deadline.C:
			write("complete", app, operation)
			_ = app.Update(app.Close)
			return
		}
	}
}

func scene(app *dxui.App, s *state, scenario, mode string) dxui.View {
	switch scenario {
	case "login":
		return loginScene(s, mode)
	case "list":
		return listScene(s)
	case "images":
		return imageScene(s)
	default:
		return dxui.Box(dxui.BoxProps{Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Fill(), Padding: dxui.Padding(24)}}, dxui.Label("dxui minimal"))
	}
}

func loginScene(s *state, mode string) dxui.View {
	value := fmt.Sprintf("measure-%04d", s.value)
	if mode == "caret" {
		value = "focused caret baseline"
	}
	card := dxui.Box(dxui.BoxProps{Key: "card", Token: dxui.ComponentPanel, Gap: 12, Style: dxui.Style{Width: dxui.Px(480), Padding: dxui.Padding(28)}},
		dxui.Text(dxui.TextProps{Value: "Welcome back", Style: dxui.Style{Text: dxui.TextStyle{Size: 30, LineHeight: 38, Weight: dxui.WeightBold}}}),
		dxui.Label("Sign in to continue to dxui"),
		dxui.Input(dxui.InputProps{Key: "username", Value: value, Placeholder: "name@example.com", OnChange: func(string) {}}).WithStyle(dxui.Style{Width: dxui.Fill()}),
		dxui.Input(dxui.InputProps{Key: "password", Value: "baseline", Password: true, OnChange: func(string) {}}).WithStyle(dxui.Style{Width: dxui.Fill()}),
		dxui.Select(dxui.SelectProps{Key: "role", Value: "developer", Options: []dxui.SelectOption{{Value: "developer", Label: "Developer"}, {Value: "designer", Label: "Designer"}, {Value: "operator", Label: "Operator"}}}),
		dxui.Checkbox(dxui.CheckboxProps{Key: "remember", Checked: s.value%2 == 1}, dxui.Label("Remember me")),
		dxui.TextButton(dxui.ButtonProps{Key: "login"}, "Login"))
	return dxui.Box(dxui.BoxProps{Align: dxui.AlignCenter, Justify: dxui.JustifyCenter, Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Fill(), Padding: dxui.Padding(32)}}, card)
}

func listScene(s *state) dxui.View {
	rows := make([]dxui.View, 1000)
	for i := range rows {
		value := fmt.Sprintf("Row %04d   fixed data", i)
		if i == s.value%len(rows) {
			value += fmt.Sprintf("   update %04d", s.value)
		}
		rows[i] = dxui.Text(dxui.TextProps{Key: fmt.Sprintf("row-%04d", i), Value: value, Style: dxui.Style{Height: dxui.Px(28), Shrink: dxui.NoShrink()}})
	}
	return dxui.Scroll(dxui.ScrollProps{Key: "list", Axis: dxui.ScrollVertical, Offset: dxui.Some(dxui.Point{Y: s.offset}), Scrollbar: dxui.ScrollbarAlways}, dxui.Box(dxui.BoxProps{Style: dxui.Style{Padding: dxui.Padding(16)}}, rows...))
}

func imageScene(s *state) dxui.View {
	children := []dxui.View{dxui.Label("Deterministic image resources"), dxui.Image(dxui.ImageProps{Key: "small", Source: s.small, Fit: dxui.ImageContain, Style: dxui.Style{Width: dxui.Px(128), Height: dxui.Px(128)}})}
	if !s.remove {
		children = append(children, dxui.Image(dxui.ImageProps{Key: "large", Source: s.large, Fit: dxui.ImageContain, Style: dxui.Style{Width: dxui.Px(720), Height: dxui.Px(450)}}))
	}
	return dxui.Box(dxui.BoxProps{Gap: 16, Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Fill(), Padding: dxui.Padding(20)}}, children...)
}

func pattern(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x*17 + y*3), G: uint8(x*5 + y*11), B: uint8(x ^ y), A: 255})
		}
	}
	return img
}
