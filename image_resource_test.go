package dxui

import (
	"image"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
)

const (
	imageEvidenceWidth  = 1024
	imageEvidenceHeight = 600
	imageEvidencePixels = imageEvidenceWidth * imageEvidenceHeight * 4
)

func oversizedImageEvidenceApp() *App {
	source := ImageFromGo(image.NewNRGBA(image.Rect(0, 0, imageEvidenceWidth, imageEvidenceHeight)))
	app := NewApp(AppOptions{Width: 320, Height: 200})
	app.root = func() View { return Image(ImageProps{Source: source}) }
	return app
}

func TestOversizedImageDecodeAndLayerAccounting(t *testing.T) {
	app := oversizedImageEvidenceApp()
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	stats := app.images.Stats()
	displayBytes := uniqueDisplayImageBytes(app.display)
	t.Logf("decode_count=%d decoded_pixel_bytes=%d cpu_cache_pixel_bytes=%d active_cpu_pixel_bytes=%d display_pixel_bytes=%d cache_budget_bytes=%d",
		stats.DecodeCount, stats.DecodedBytes, stats.PixelBytes, stats.ActiveBytes, displayBytes, stats.BudgetBytes)
	if stats.DecodeCount != 1 || stats.DecodedBytes != imageEvidencePixels {
		t.Fatalf("decode work = %d/%d bytes, want one/%d", stats.DecodeCount, stats.DecodedBytes, imageEvidencePixels)
	}
	if stats.PixelBytes != imageEvidencePixels || stats.ActiveBytes != imageEvidencePixels || displayBytes != imageEvidencePixels {
		t.Fatalf("layer bytes cache=%d active=%d display=%d, want %d", stats.PixelBytes, stats.ActiveBytes, displayBytes, imageEvidencePixels)
	}
}

func BenchmarkOversizedImageInitialCommit(b *testing.B) {
	for range b.N {
		app := oversizedImageEvidenceApp()
		if err := app.buildRoot(); err != nil {
			b.Fatal(err)
		}
	}
}

func uniqueDisplayImageBytes(list paint.DisplayList) int {
	seen := make(map[*paint.ImageBitmap]struct{})
	bytes := 0
	for _, command := range list {
		if command.Kind != paint.CommandDrawImage || command.Image == nil {
			continue
		}
		if _, ok := seen[command.Image]; ok {
			continue
		}
		seen[command.Image] = struct{}{}
		bytes += len(command.Image.Pixels)
	}
	return bytes
}
