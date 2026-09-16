//go:build sdl_integration

package dxui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv("DXUI_SDL_SMOKE_HELPER"); mode != "" {
		if err := runSDLRuntimeSmoke(mode); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestSDLRuntimeSmoke(t *testing.T) {
	t.Setenv("DXUI_SDL3_PATH", "")
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no X11 or Wayland display; native SDL smoke requires a provisioned display")
	}

	for _, mode := range []string{"default", "software"} {
		t.Run(mode, func(t *testing.T) {
			extractionRoot := t.TempDir()
			command := exec.Command(os.Args[0], "-test.run=^TestSDLRuntimeSmoke$", "-test.v")
			command.Env = smokeEnvironment(os.Environ(), mode, extractionRoot)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("isolated native smoke: %v\n%s", err, output)
			}
			t.Log(strings.TrimSpace(string(output)))
			entries, err := os.ReadDir(extractionRoot)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("normal binsdl unload left temporary entries: %v", entries)
			}
		})
	}
}

func runSDLRuntimeSmoke(mode string) error {
	preference := RendererAuto
	wantName := ""
	if mode == "software" {
		preference = RendererSoftware
		wantName = "software"
	}
	app := NewApp(AppOptions{
		Title: "dxui SDL smoke", Width: 320, Height: 200,
		Background: RGBA(12, 34, 56, 255), Renderer: preference,
	})
	ready := make(chan struct{})
	root := func() View {
		select {
		case <-ready:
		default:
			close(ready)
		}
		return Scroll(ScrollProps{Axis: ScrollVertical, InitialOffset: Some(Point{Y: 8})},
			Box(BoxProps{},
				Select(SelectProps{Key: "smoke-select", Value: "a", Options: []SelectOption{
					{Value: "a", Label: "Alpha"},
					{Value: "b", Label: "Beta", Disabled: true},
				}}),
				Text(TextProps{Style: Style{Height: Px(180), Shrink: Some(float32(0))}, Value: "smoke"}),
				Text(TextProps{Style: Style{Height: Px(180), Shrink: Some(float32(0))}, Value: "scroll content"}),
			),
		)
	}
	updateErr := make(chan error, 1)
	go func() {
		<-ready
		updateErr <- app.Update(app.Close)
	}()
	if err := app.Run(root); err != nil {
		return err
	}
	if err := <-updateErr; err != nil {
		return err
	}
	diagnostics := app.Diagnostics()
	fmt.Printf("renderer=%s SDL=%s logical=%+v pixel=%+v density=%.3f display-scale=%.3f frames=%d fallback=%v\n",
		diagnostics.RendererName, diagnostics.SDLVersion,
		diagnostics.LogicalSize, diagnostics.PixelSize,
		diagnostics.PixelDensity, diagnostics.DisplayScale,
		diagnostics.FrameCount, diagnostics.SoftwareFallback)
	if diagnostics.SDLVersion != "3.4.0" {
		return fmt.Errorf("SDL version = %q, want 3.4.0", diagnostics.SDLVersion)
	}
	if diagnostics.RendererName == "" {
		return fmt.Errorf("empty renderer name")
	}
	if wantName != "" && diagnostics.RendererName != wantName {
		return fmt.Errorf("renderer = %q, want %q", diagnostics.RendererName, wantName)
	}
	if diagnostics.FrameCount != 1 {
		return fmt.Errorf("frames = %d, want exactly one initial frame", diagnostics.FrameCount)
	}
	if app.retained.Root() != nil || app.geometry != nil || app.display != nil || app.inputActions != nil {
		return fmt.Errorf("App.Close retained tree, overlay snapshot, geometry, or display state")
	}
	return nil
}

func smokeEnvironment(base []string, mode, extractionRoot string) []string {
	replacements := map[string]string{
		"DXUI_SDL3_PATH": "", "DXUI_SDL_SMOKE_HELPER": mode,
		"TMP": extractionRoot, "TEMP": extractionRoot, "TMPDIR": extractionRoot,
	}
	result := make([]string, 0, len(base)+len(replacements))
	for _, item := range base {
		key := item
		if index := strings.IndexByte(item, '='); index >= 0 {
			key = item[:index]
		}
		if _, replaced := replacements[strings.ToUpper(key)]; !replaced {
			result = append(result, item)
		}
	}
	for key, value := range replacements {
		result = append(result, key+"="+value)
	}
	return result
}
