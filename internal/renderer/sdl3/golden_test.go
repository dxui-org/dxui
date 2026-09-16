//go:build sdl_golden

package sdl3

import (
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/paint"
	"golang.org/x/image/bmp"
)

// TestSDLGoldenSoftwareResizeTextures covers text and image replay after the
// native software-renderer window changes size.
func TestSDLGoldenSoftwareResizeTextures(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no provisioned X11/Wayland display")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	background := Color{R: 241, G: 245, B: 249, A: 255}
	driver, err := openWithAPI(systemAPI{}, Config{
		Title: "dxui software resize regression", Width: 160, Height: 96,
		Background: background, Software: true, GlyphCacheBytes: 32 << 10, ImageCacheBytes: 32 << 10,
		Diagnostics: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()
	list := paint.DisplayList{
		{Kind: paint.CommandDrawText, Rect: paint.Rect{X: 12, Y: 12, Width: 32, Height: 32}, Text: &paint.TextBitmap{
			Key: "resize-text", Width: 4, Height: 4, Pixels: []byte{
				0, 0, 0, 0, 0, 255, 255, 0, 0, 255, 255, 0, 0, 0, 0, 0,
			},
		}, Color: paint.Color{R: 25, G: 110, B: 210, A: 255}},
		{Kind: paint.CommandDrawImage, Rect: paint.Rect{X: 60, Y: 12, Width: 32, Height: 32}, Image: &paint.ImageBitmap{
			Key: "resize-image", Width: 2, Height: 2, Pixels: []byte{
				220, 55, 45, 255, 220, 55, 45, 255, 220, 55, 45, 255, 220, 55, 45, 255,
			},
		}},
	}
	if err := driver.Render(list); err != nil {
		t.Fatal(err)
	}
	if err := driver.Present(); err != nil {
		t.Fatal(err)
	}
	oldViewport := driver.Diagnostics().Viewport
	if err := driver.window.setSize(oldViewport.LogicalWidth*2, oldViewport.LogicalHeight*2); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for driver.Diagnostics().Viewport == oldViewport {
		event, waitErr := driver.Wait(context.Background(), &deadline)
		if waitErr != nil {
			t.Fatal(waitErr)
		}
		if event.Kind == 0 && !time.Now().Before(deadline) {
			t.Fatal("timed out waiting for native resize")
		}
	}
	if err := driver.Render(list); err != nil {
		t.Fatal(err)
	}
	if err := driver.Present(); err != nil {
		t.Fatal(err)
	}
	native := driver.renderer.(systemRenderer)
	surface, err := native.value.ReadPixels(nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, transparentAlpha, err := surface.ReadPixel(14, 14)
	surface.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	if transparentAlpha != 255 {
		t.Fatalf("software target alpha after resize = %d, want opaque background", transparentAlpha)
	}
	path := filepath.Join(t.TempDir(), "software-resize.bmp")
	width, height := saveRendererBMP(t, driver, path)
	viewport := driver.Diagnostics().Viewport
	if viewport.PixelWidth != viewport.LogicalWidth || viewport.PixelHeight != viewport.LogicalHeight {
		t.Fatalf("software viewport after resize = %+v, want matching logical and presentable pixel sizes", viewport)
	}
	scaleX, scaleY := driver.scaleX, driver.scaleY
	t.Logf("renderer=%s before=%dx%d after=%dx%d output=%dx%d", driver.Diagnostics().RendererName,
		oldViewport.LogicalWidth, oldViewport.LogicalHeight, viewport.LogicalWidth, viewport.LogicalHeight, width, height)
	assertRGBNear(t, readBMPPixel(t, path, int(14*scaleX), int(14*scaleY)), [3]uint8{background.R, background.G, background.B}, 2)
	assertRGBNear(t, readBMPPixel(t, path, int(28*scaleX), int(28*scaleY)), [3]uint8{25, 110, 210}, 2)
	assertRGBNear(t, readBMPPixel(t, path, int(76*scaleX), int(28*scaleY)), [3]uint8{220, 55, 45}, 2)
}

func TestSDLGoldenOversizedImageTextureLifecycle(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no provisioned X11/Wayland display")
	}
	for _, test := range []struct {
		name     string
		software bool
	}{{"default", false}, {"software", true}} {
		t.Run(test.name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			driver, err := openWithAPI(systemAPI{}, Config{
				Title: "dxui oversized image lifecycle", Width: 160, Height: 96,
				Background: Color{R: 20, G: 20, B: 20, A: 255}, Software: test.software,
				ImageCacheBytes: 1, Diagnostics: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := driver.Close(); err != nil {
					t.Errorf("close SDL driver: %v", err)
				}
			}()
			bitmap := &paint.ImageBitmap{Key: "native-oversized", Width: 640, Height: 400, Pixels: make([]byte, 640*400*4)}
			list := paint.DisplayList{{Kind: paint.CommandDrawImage, Rect: paint.Rect{Width: 160, Height: 96}, Image: bitmap}}
			if err := driver.Render(list); err != nil {
				t.Fatal(err)
			}
			if err := driver.Render(list); err != nil {
				t.Fatal(err)
			}
			if got := driver.Diagnostics(); got.TextureCreates != 1 || got.TextureDestroys != 0 {
				t.Fatalf("steady texture counters = %d/%d", got.TextureCreates, got.TextureDestroys)
			}
			if _, err := driver.translate(nativeEvent{kind: nativeEventRendererReset}); err != nil {
				t.Fatal(err)
			}
			if err := driver.Render(list); err != nil {
				t.Fatal(err)
			}
			if got := driver.Diagnostics(); got.TextureCreates != 2 || got.TextureDestroys != 1 {
				t.Fatalf("reset texture counters = %d/%d", got.TextureCreates, got.TextureDestroys)
			}
			if err := driver.Render(nil); err != nil {
				t.Fatal(err)
			}
			if got := driver.Diagnostics(); got.TextureCreates != 2 || got.TextureDestroys != 2 {
				t.Fatalf("release texture counters = %d/%d", got.TextureCreates, got.TextureDestroys)
			}
		})
	}
}

// TestSDLGoldenAlphaRoundedRect reads the actual renderer target and compares
// stable interior pixels. Edge antialiasing is deliberately excluded: OpenGL,
// Direct3D, Metal and software rasterizers may cover boundary pixels
// differently. RGB tolerance 4 covers integer rounding in source-over blend.
func TestSDLGoldenAlphaRoundedRect(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no provisioned X11/Wayland display")
	}
	for _, software := range []bool{false, true} {
		name := "default"
		if software {
			name = "software"
		}
		t.Run(name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			background := Color{R: 10, G: 20, B: 30, A: 255}
			driver, err := openWithAPI(systemAPI{}, Config{
				Title: "dxui golden", Width: 96, Height: 72,
				Background: background, Software: software, ShadowCacheBytes: 32 << 10,
				GlyphCacheBytes: 32 << 10,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			list := paint.DisplayList{
				{Kind: paint.CommandPushClip, Rect: paint.Rect{X: 12, Y: 8, Width: 70, Height: 54}},
				{Kind: paint.CommandPushOpacity, Opacity: .5},
				{Kind: paint.CommandDrawShadow, Rect: paint.Rect{X: 20, Y: 16, Width: 48, Height: 36}, Radii: paint.Radii{TopLeft: 8, TopRight: 8, BottomRight: 8, BottomLeft: 8}, Shadow: paint.Shadow{Spread: 3, Color: paint.Color{B: 255, A: 255}}},
				{Kind: paint.CommandFillStrokeRoundedRect, Rect: paint.Rect{X: 20, Y: 16, Width: 48, Height: 36}, Radii: paint.Radii{TopLeft: 8, TopRight: 8, BottomRight: 8, BottomLeft: 8}, Width: 4, Color: paint.Color{R: 210, G: 40, B: 20, A: 255}, BorderColor: paint.Color{R: 255, G: 255, B: 255, A: 255}},
				{Kind: paint.CommandPopOpacity},
				{Kind: paint.CommandPopClip},
				{Kind: paint.CommandDrawText, Rect: paint.Rect{X: 4, Y: 60, Width: 8, Height: 8}, Text: &paint.TextBitmap{
					Key: "golden-solid-mask", Width: 4, Height: 4,
					Pixels: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
				}, Color: paint.Color{G: 200, B: 100, A: 255}},
			}
			if err := driver.Render(list); err != nil {
				t.Fatal(err)
			}
			native := driver.renderer.(systemRenderer)
			surface, err := native.value.ReadPixels(nil)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "frame.bmp")
			if err := surface.SaveBMP(path); err != nil {
				surface.Destroy()
				t.Fatal(err)
			}
			width, height := surface.W, surface.H
			surface.Destroy()
			diagnostics := driver.Diagnostics()
			t.Logf("renderer=%s screenshot=%dx%d", diagnostics.RendererName, width, height)
			scaleX := float32(diagnostics.Viewport.PixelWidth) / float32(diagnostics.Viewport.LogicalWidth)
			scaleY := float32(diagnostics.Viewport.PixelHeight) / float32(diagnostics.Viewport.LogicalHeight)
			outside := readBMPPixel(t, path, int(4*scaleX), int(4*scaleY))
			shadow := readBMPPixel(t, path, int(18*scaleX), int(34*scaleY))
			inside := readBMPPixel(t, path, int(44*scaleX), int(34*scaleY))
			border := readBMPPixel(t, path, int(21*scaleX), int(34*scaleY))
			textMask := readBMPPixel(t, path, int(8*scaleX), int(64*scaleY))
			assertRGBNear(t, outside, [3]uint8{10, 20, 30}, 2)
			assertRGBNear(t, shadow, [3]uint8{5, 10, 143}, 4)
			// The combined mesh paints either fill or border once over the 50% shadow.
			assertRGBNear(t, inside, [3]uint8{108, 25, 82}, 4)
			assertRGBNear(t, border, [3]uint8{130, 133, 199}, 5)
			assertRGBNear(t, textMask, [3]uint8{0, 200, 100}, 2)
		})
	}
}

func TestSDLGoldenShadowRemovalClearsOldOverflow(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no provisioned X11/Wayland display")
	}
	for _, software := range []bool{false, true} {
		name := "default"
		if software {
			name = "software"
		}
		t.Run(name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			background := Color{R: 241, G: 245, B: 249, A: 255}
			driver, err := openWithAPI(systemAPI{}, Config{
				Title: "dxui shadow removal", Width: 64, Height: 64,
				Background: background, Software: software, ShadowCacheBytes: 32 << 10,
			})
			if err != nil {
				t.Fatal(err)
			}
			defer driver.Close()
			withShadow := paint.DisplayList{{
				Kind: paint.CommandDrawShadow, Rect: paint.Rect{X: 20, Y: 20, Width: 24, Height: 24},
				Shadow: paint.Shadow{Spread: 4, Color: paint.Color{A: 255}},
			}}
			if err := driver.Render(withShadow); err != nil {
				t.Fatal(err)
			}
			before := filepath.Join(t.TempDir(), "before.bmp")
			saveRendererBMP(t, driver, before)
			if err := driver.Render(nil); err != nil {
				t.Fatal(err)
			}
			after := filepath.Join(t.TempDir(), "after.bmp")
			width, height := saveRendererBMP(t, driver, after)
			diagnostics := driver.Diagnostics()
			scaleX := float32(width) / float32(diagnostics.Viewport.LogicalWidth)
			scaleY := float32(height) / float32(diagnostics.Viewport.LogicalHeight)
			x, y := int(18*scaleX), int(32*scaleY)
			assertRGBNear(t, readBMPPixel(t, before, x, y), [3]uint8{0, 0, 0}, 2)
			assertRGBNear(t, readBMPPixel(t, after, x, y), [3]uint8{background.R, background.G, background.B}, 2)
		})
	}
}

func saveRendererBMP(t *testing.T, driver *Driver, path string) (int32, int32) {
	t.Helper()
	native := driver.renderer.(systemRenderer)
	surface, err := native.value.ReadPixels(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer surface.Destroy()
	if err := surface.SaveBMP(path); err != nil {
		t.Fatal(err)
	}
	return surface.W, surface.H
}

// TestSDLGoldenButtonMouseFocusCapture retains native pixels for the confirmed
// Button focus-style regression. Baseline mode reproduces the old post-click
// white Focus border; normal mode renders the corrected mouse-focused state.
func TestSDLGoldenButtonMouseFocusCapture(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no provisioned X11/Wayland display")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	driver, err := openWithAPI(systemAPI{}, Config{
		Title: "dxui button focus capture", Width: 240, Height: 96,
		Background: Color{R: 241, G: 245, B: 249, A: 255}, ShadowCacheBytes: 32 << 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer driver.Close()

	border, width := paint.Color{R: 113, G: 122, B: 138, A: 255}, float32(1)
	if os.Getenv("DXUI_BUTTON_FOCUS_BASELINE") != "" {
		border, width = paint.Color{R: 255, G: 255, B: 255, A: 255}, 2
	}
	list := paint.DisplayList{
		{Kind: paint.CommandPushOpacity, Opacity: .92},
		{Kind: paint.CommandFillStrokeRoundedRect, Rect: paint.Rect{X: 40, Y: 24, Width: 160, Height: 48},
			Radii: paint.Radii{TopLeft: 8, TopRight: 8, BottomRight: 8, BottomLeft: 8}, Width: width,
			Color: paint.Color{R: 55, G: 118, B: 232, A: 255}, BorderColor: border},
		{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 82, Y: 45, Width: 76, Height: 6},
			Radii: paint.Radii{TopLeft: 3, TopRight: 3, BottomRight: 3, BottomLeft: 3}, Color: paint.Color{R: 255, G: 255, B: 255, A: 255}},
		{Kind: paint.CommandPopOpacity},
	}
	if err := driver.Render(list); err != nil {
		t.Fatal(err)
	}
	native := driver.renderer.(systemRenderer)
	surface, err := native.value.ReadPixels(nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "button.bmp")
	if err := surface.SaveBMP(path); err != nil {
		surface.Destroy()
		t.Fatal(err)
	}
	outputWidth, outputHeight := surface.W, surface.H
	surface.Destroy()
	if directory := os.Getenv("DXUI_BUTTON_FOCUS_CAPTURE_DIR"); directory != "" {
		writeNearestZoom(t, path, filepath.Join(directory, "mouse-click-light.png"), 1)
	}
	scaleX, scaleY := float32(outputWidth)/240, float32(outputHeight)/96
	t.Logf("renderer=%s screenshot=%dx%d scale=%.2fx%.2f", driver.Diagnostics().RendererName, outputWidth, outputHeight, scaleX, scaleY)
	if os.Getenv("DXUI_BUTTON_FOCUS_BASELINE") == "" {
		assertRGBNear(t, readBMPPixel(t, path, int(41*scaleX), int(48*scaleY)), [3]uint8{83, 129, 212}, 3)
	}
}

// TestSDLGoldenEdgeCoverage exercises the pixels that the older interior-only
// golden deliberately ignored. DXUI_EDGE_CAPTURE_DIR retains the native BMP
// and an 8x nearest-neighbor inspection image; DXUI_EDGE_CAPTURE_BASELINE is
// used only to collect a known-failing pre-fix baseline.
func TestSDLGoldenEdgeCoverage(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no provisioned X11/Wayland display")
	}
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		for _, software := range []bool{false, true} {
			name := "default"
			if software {
				name = "software"
			}
			t.Run(fmt.Sprintf("%s/scale-%g", name, scale), func(t *testing.T) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				background := Color{R: 244, G: 247, B: 250, A: 255}
				driver, err := openWithAPI(systemAPI{}, Config{
					Title: "dxui edge golden", Width: int32(160 * scale), Height: int32(96 * scale),
					Background: background, Software: software, ShadowCacheBytes: 32 << 10,
					GlyphCacheBytes: 32 << 10,
				})
				if err != nil {
					t.Fatal(err)
				}
				defer driver.Close()
				// This test controls physical target size independently from the
				// host content scale; production has already exercised its normal
				// logical-window sizing during open.
				if err := driver.window.setSize(int32(160*scale), int32(96*scale)); err != nil {
					t.Fatal(err)
				}
				if _, err := driver.refreshViewport(); err != nil {
					t.Fatal(err)
				}
				if err := driver.renderer.setScale(scale, scale); err != nil {
					t.Fatal(err)
				}
				driver.scaleX, driver.scaleY = scale, scale
				fill := paint.Color{R: 28, G: 92, B: 180, A: 255}
				border := paint.Color{R: 220, G: 58, B: 45, A: 255}
				list := paint.DisplayList{
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 8, Y: 8, Width: 36, Height: 28}, Radii: uniformRadii(8), Color: fill},
					{Kind: paint.CommandStrokeRoundedRect, Rect: paint.Rect{X: 52, Y: 8, Width: 28, Height: 28}, Radii: uniformRadii(14), Width: 1, Color: border},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 88.25, Y: 8.5, Width: 32, Height: 28}, Radii: uniformRadii(7), Color: paint.Color{R: 20, G: 190, B: 130, A: 112}},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 0, Y: 48, Width: 160, Height: 48}, Color: paint.Color{R: 18, G: 24, B: 38, A: 255}},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 8.25, Y: 57.25, Width: 36, Height: 27}, Radii: uniformRadii(5), Color: paint.Color{R: 226, G: 232, B: 240, A: 255}},
					{Kind: paint.CommandStrokeRoundedRect, Rect: paint.Rect{X: 52.25, Y: 57.25, Width: 28, Height: 28}, Radii: uniformRadii(14), Width: .75, Color: paint.Color{R: 248, G: 180, B: 45, A: 180}},
					{Kind: paint.CommandDrawIcon, Rect: paint.Rect{X: 96, Y: 58, Width: 24, Height: 24}, Text: diagonalMask(), Color: paint.Color{R: 245, G: 248, B: 252, A: 255}},
					{Kind: paint.CommandDrawText, Rect: paint.Rect{X: 128, Y: 58, Width: 20, Height: 24}, Text: letterMask(), Color: paint.Color{R: 245, G: 248, B: 252, A: 255}},
				}
				if err := driver.Render(list); err != nil {
					t.Fatal(err)
				}
				native := driver.renderer.(systemRenderer)
				surface, err := native.value.ReadPixels(nil)
				if err != nil {
					t.Fatal(err)
				}
				defer surface.Destroy()
				path := filepath.Join(t.TempDir(), "edge.bmp")
				if err := surface.SaveBMP(path); err != nil {
					t.Fatal(err)
				}
				diagnostics := driver.Diagnostics()
				bitmap := readBMP(t, path)
				coverage := countTransitionPixels(bitmap, scale, image.Rect(6, 6, 46, 38), background, Color(fill))
				borderCoverage := countTransitionPixels(bitmap, scale, image.Rect(50, 6, 82, 38), background, Color(border))
				t.Logf("renderer=%s scene=160x96 native-logical=%dx%d output=%dx%d pixel-density=%.3f display-scale=%.3f simulated-render-scale=%.2f fill-transition-pixels=%d border-transition-pixels=%d",
					diagnostics.RendererName, diagnostics.Viewport.LogicalWidth, diagnostics.Viewport.LogicalHeight,
					surface.W, surface.H, diagnostics.Viewport.PixelDensity, diagnostics.Viewport.DisplayScale, scale, coverage, borderCoverage)
				if directory := os.Getenv("DXUI_EDGE_CAPTURE_DIR"); directory != "" {
					base := fmt.Sprintf("%s-scale-%s", name, strconv.FormatFloat(float64(scale), 'f', 2, 32))
					copyCapture(t, path, filepath.Join(directory, base+".bmp"))
					writeNearestZoom(t, path, filepath.Join(directory, base+"-zoom8x.png"), 8)
				}
				if os.Getenv("DXUI_EDGE_CAPTURE_BASELINE") == "" && coverage < max(4, int(8*scale)) {
					t.Fatalf("rounded fill has only %d partially covered edge pixels", coverage)
				}
				if os.Getenv("DXUI_EDGE_CAPTURE_BASELINE") == "" && borderCoverage < max(4, int(8*scale)) {
					t.Fatalf("thin rounded border has only %d partially covered edge pixels", borderCoverage)
				}
				assertRGBANear(t, bitmap.RGBAAt(int(26*scale), int(22*scale)), color.RGBA{R: fill.R, G: fill.G, B: fill.B, A: 255}, 3)
				assertRGBANear(t, bitmap.RGBAAt(int(66*scale), int(22*scale)), color.RGBA{R: background.R, G: background.G, B: background.B, A: 255}, 3)
				translucent := blendOver(color.RGBA{R: 20, G: 190, B: 130, A: 112}, color.RGBA{R: background.R, G: background.G, B: background.B, A: 255})
				assertRGBANear(t, bitmap.RGBAAt(int(104*scale), int(22*scale)), translucent, 5)
			})
		}
	}
}

// TestSDLGoldenJoinedButtonGroupSeams exercises the native raster column at
// each divider-free ButtonGroup join. In particular, a translucent Hover fill
// must be composed over the panel just like its interior, not over the prior
// Button, which would create a dark one-pixel line.
func TestSDLGoldenJoinedButtonGroupSeams(t *testing.T) {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("no provisioned X11/Wayland display")
	}
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		for _, software := range []bool{false, true} {
			name := "default"
			if software {
				name = "software"
			}
			t.Run(fmt.Sprintf("%s/scale-%g", name, scale), func(t *testing.T) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				background := Color{R: 241, G: 245, B: 249, A: 255}
				driver, err := openWithAPI(systemAPI{}, Config{
					Title: "dxui joined button group", Width: int32(160 * scale), Height: int32(120 * scale),
					Background: background, Software: software, ShadowCacheBytes: 32 << 10,
				})
				if err != nil {
					t.Fatal(err)
				}
				defer driver.Close()
				if err := driver.window.setSize(int32(160*scale), int32(120*scale)); err != nil {
					t.Fatal(err)
				}
				if _, err := driver.refreshViewport(); err != nil {
					t.Fatal(err)
				}
				if err := driver.renderer.setScale(scale, scale); err != nil {
					t.Fatal(err)
				}
				driver.scaleX, driver.scaleY = scale, scale
				base := paint.Color{R: 37, G: 99, B: 235, A: 255}
				hover := paint.Color{R: 29, G: 78, B: 216, A: 255}
				list := paint.DisplayList{
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 20, Y: 20, Width: 40, Height: 36}, Radii: paint.Radii{TopLeft: 8, BottomLeft: 8}, Color: base, JoinedEdges: paint.JoinedRight},
					{Kind: paint.CommandPushOpacity, Opacity: .92},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 60, Y: 20, Width: 40, Height: 36}, Color: hover, JoinedEdges: paint.JoinedLeft | paint.JoinedRight},
					{Kind: paint.CommandPopOpacity},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 100, Y: 20, Width: 40, Height: 36}, Radii: paint.Radii{TopRight: 8, BottomRight: 8}, Color: base, JoinedEdges: paint.JoinedLeft},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 20.3, Y: 68, Width: 40, Height: 36}, Radii: paint.Radii{TopLeft: 8, BottomLeft: 8}, Color: base, JoinedEdges: paint.JoinedRight},
					{Kind: paint.CommandPushOpacity, Opacity: .92},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 60.3, Y: 68, Width: 40, Height: 36}, Color: hover, JoinedEdges: paint.JoinedLeft | paint.JoinedRight},
					{Kind: paint.CommandPopOpacity},
					{Kind: paint.CommandFillRoundedRect, Rect: paint.Rect{X: 100.3, Y: 68, Width: 40, Height: 36}, Radii: paint.Radii{TopRight: 8, BottomRight: 8}, Color: base, JoinedEdges: paint.JoinedLeft},
				}
				if err := driver.Render(list); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "joined-buttons.bmp")
				width, height := saveRendererBMP(t, driver, path)
				bitmap := readBMP(t, path)
				y := int(38 * scale)
				middleInterior := bitmap.RGBAAt(int(72*scale), y)
				leftJoin := bitmap.RGBAAt(int(60*scale), y)
				rightInterior := bitmap.RGBAAt(int(120*scale), y)
				rightJoin := bitmap.RGBAAt(int(100*scale), y)
				t.Logf("renderer=%s screenshot=%dx%d scale=%.2f left=%v/middle=%v right=%v/interior=%v", driver.Diagnostics().RendererName, width, height, scale, leftJoin, middleInterior, rightJoin, rightInterior)
				assertRGBANear(t, leftJoin, middleInterior, 3)
				assertRGBANear(t, rightJoin, rightInterior, 3)

				fractionalY := int(86 * scale)
				leftPixel := int(math.Ceil(float64(60.3 * scale)))
				rightPixel := int(math.Ceil(float64(100.3 * scale)))
				fractionalFirst := bitmap.RGBAAt(int(40.3*scale), fractionalY)
				fractionalMiddle := bitmap.RGBAAt(int(72.3*scale), fractionalY)
				fractionalRight := bitmap.RGBAAt(int(120.3*scale), fractionalY)
				assertRGBAOneOfNear(t, bitmap.RGBAAt(leftPixel-1, fractionalY), fractionalFirst, fractionalMiddle, 3)
				assertRGBAOneOfNear(t, bitmap.RGBAAt(leftPixel, fractionalY), fractionalFirst, fractionalMiddle, 3)
				assertRGBAOneOfNear(t, bitmap.RGBAAt(rightPixel-1, fractionalY), fractionalMiddle, fractionalRight, 3)
				assertRGBAOneOfNear(t, bitmap.RGBAAt(rightPixel, fractionalY), fractionalMiddle, fractionalRight, 3)
			})
		}
	}
}

func uniformRadii(radius float32) paint.Radii {
	return paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
}

func diagonalMask() *paint.TextBitmap {
	const size = 24
	pixels := make([]byte, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			distance := x - y
			if distance < 0 {
				distance = -distance
			}
			switch distance {
			case 0:
				pixels[y*size+x] = 255
			case 1:
				pixels[y*size+x] = 96
			}
		}
	}
	return &paint.TextBitmap{Key: "edge-diagonal-24", Width: size, Height: size, Pixels: pixels}
}

func letterMask() *paint.TextBitmap {
	const width, height = 10, 12
	pixels := make([]byte, width*height)
	for y := 1; y < height; y++ {
		left, right := 4-y/3, 5+y/3
		if left >= 0 {
			pixels[y*width+left] = 255
		}
		if right < width {
			pixels[y*width+right] = 255
		}
		if y == 7 {
			for x := left; x <= right; x++ {
				pixels[y*width+x] = 255
			}
		}
	}
	return &paint.TextBitmap{Key: "edge-letter-a", Width: width, Height: height, Pixels: pixels}
}

func countTransitionPixels(bitmap *image.RGBA, scale float32, logical image.Rectangle, background, foreground Color) int {
	left, top := int(float32(logical.Min.X)*scale), int(float32(logical.Min.Y)*scale)
	right, bottom := int(float32(logical.Max.X)*scale), int(float32(logical.Max.Y)*scale)
	count := 0
	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			pixel := bitmap.RGBAAt(x, y)
			if !rgbNear(pixel, background, 2) && !rgbNear(pixel, foreground, 2) {
				count++
			}
		}
	}
	return count
}

func blendOver(source, destination color.RGBA) color.RGBA {
	alpha := int(source.A)
	blend := func(src, dst uint8) uint8 {
		return uint8((int(src)*alpha + int(dst)*(255-alpha) + 127) / 255)
	}
	return color.RGBA{R: blend(source.R, destination.R), G: blend(source.G, destination.G), B: blend(source.B, destination.B), A: 255}
}

func assertRGBANear(t *testing.T, got, want color.RGBA, tolerance int) {
	t.Helper()
	for _, pair := range [][2]uint8{{got.R, want.R}, {got.G, want.G}, {got.B, want.B}} {
		delta := int(pair[0]) - int(pair[1])
		if delta < 0 {
			delta = -delta
		}
		if delta > tolerance {
			t.Fatalf("pixel=%v want=%v within %d", got, want, tolerance)
		}
	}
}

func assertRGBAOneOfNear(t *testing.T, got, first, second color.RGBA, tolerance int) {
	t.Helper()
	near := func(want color.RGBA) bool {
		for _, pair := range [][2]uint8{{got.R, want.R}, {got.G, want.G}, {got.B, want.B}} {
			delta := int(pair[0]) - int(pair[1])
			if delta < 0 {
				delta = -delta
			}
			if delta > tolerance {
				return false
			}
		}
		return true
	}
	if !near(first) && !near(second) {
		t.Fatalf("pixel=%v want one of %v or %v within %d", got, first, second, tolerance)
	}
}

func rgbNear(got color.RGBA, want Color, tolerance int) bool {
	for _, pair := range [][2]uint8{{got.R, want.R}, {got.G, want.G}, {got.B, want.B}} {
		delta := int(pair[0]) - int(pair[1])
		if delta < 0 {
			delta = -delta
		}
		if delta > tolerance {
			return false
		}
	}
	return true
}

func copyCapture(t *testing.T, source, target string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeNearestZoom(t *testing.T, source, target string, zoom int) {
	t.Helper()
	bitmap := readBMP(t, source)
	zoomed := image.NewRGBA(image.Rect(0, 0, bitmap.Bounds().Dx()*zoom, bitmap.Bounds().Dy()*zoom))
	for y := 0; y < zoomed.Bounds().Dy(); y++ {
		for x := 0; x < zoomed.Bounds().Dx(); x++ {
			zoomed.SetRGBA(x, y, bitmap.RGBAAt(x/zoom, y/zoom))
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, zoomed); err != nil {
		t.Fatal(err)
	}
}

func readBMP(t *testing.T, path string) *image.RGBA {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	source, err := bmp.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	bitmap := image.NewRGBA(image.Rect(0, 0, source.Bounds().Dx(), source.Bounds().Dy()))
	for y := 0; y < bitmap.Bounds().Dy(); y++ {
		for x := 0; x < bitmap.Bounds().Dx(); x++ {
			bitmap.Set(x, y, source.At(source.Bounds().Min.X+x, source.Bounds().Min.Y+y))
		}
	}
	return bitmap
}

func readBMPPixel(t *testing.T, path string, x, y int) [3]uint8 {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 54 || string(data[:2]) != "BM" {
		t.Fatal("SDL screenshot is not a BMP")
	}
	offset := int(binary.LittleEndian.Uint32(data[10:14]))
	width := int(int32(binary.LittleEndian.Uint32(data[18:22])))
	heightSigned := int32(binary.LittleEndian.Uint32(data[22:26]))
	bpp := int(binary.LittleEndian.Uint16(data[28:30]))
	if width <= 0 || heightSigned == 0 || (bpp != 24 && bpp != 32) {
		t.Fatalf("unsupported BMP width=%d height=%d bpp=%d", width, heightSigned, bpp)
	}
	height := int(heightSigned)
	topDown := height < 0
	if topDown {
		height = -height
	}
	if x < 0 || x >= width || y < 0 || y >= height {
		t.Fatalf("sample (%d,%d) outside %dx%d", x, y, width, height)
	}
	bytesPerPixel := bpp / 8
	stride := (width*bytesPerPixel + 3) &^ 3
	row := y
	if !topDown {
		row = height - 1 - y
	}
	index := offset + row*stride + x*bytesPerPixel
	if index+2 >= len(data) {
		t.Fatal("truncated BMP")
	}
	return [3]uint8{data[index+2], data[index+1], data[index]}
}

func assertRGBNear(t *testing.T, got, want [3]uint8, tolerance int) {
	t.Helper()
	for index := range got {
		delta := int(got[index]) - int(want[index])
		if delta < 0 {
			delta = -delta
		}
		if delta > tolerance {
			t.Fatalf("pixel = %v, want %v within %d (%s)", got, want, tolerance, fmt.Sprint(index))
		}
	}
}
