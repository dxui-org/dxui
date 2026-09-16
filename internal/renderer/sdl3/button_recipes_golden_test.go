//go:build sdl_golden

package sdl3

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
)

func TestSDLGoldenButtonRecipes(t *testing.T) {
	directory := os.Getenv("DXUI_BUTTON_VISUAL_DIR")
	if directory == "" {
		t.Skip("export root TestExportButtonVisualScenes first")
	}
	paths, err := filepath.Glob(filepath.Join(directory, "*.gob"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 10 {
		t.Fatalf("want ten exported theme/state scenes, got %d", len(paths))
	}
	for _, path := range paths {
		for _, software := range []bool{false, true} {
			name := "default"
			if software {
				name = "software"
			}
			t.Run(filepath.Base(path)+"/"+name, func(t *testing.T) {
				file, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				var scene struct {
					Width, Height int32
					Background    paint.Color
					List          paint.DisplayList
				}
				err = gob.NewDecoder(file).Decode(&scene)
				file.Close()
				if err != nil {
					t.Fatal(err)
				}
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				driver, err := openWithAPI(systemAPI{}, Config{Title: "dxui Button recipes", Width: scene.Width, Height: scene.Height, Software: software, Background: Color(scene.Background)})
				if err != nil {
					t.Fatal(err)
				}
				defer driver.Close()
				if err := driver.Render(scene.List); err != nil {
					t.Fatal(err)
				}
				base := filepath.Join(directory, filepath.Base(path[:len(path)-4])+"-"+name)
				width, height := saveRendererBMP(t, driver, base+".bmp")
				writeNearestZoom(t, base+".bmp", base+".png", 1)
				// Root surface and a default filled interior validate actual native replay.
				assertRGBNear(t, readBMPPixel(t, base+".bmp", 2, 2), [3]uint8{scene.Background.R, scene.Background.G, scene.Background.B}, 2)
				if width <= 0 || height <= 0 {
					t.Fatal("empty capture")
				}
				t.Logf("renderer=%s screenshot=%dx%d %s.png", driver.Diagnostics().RendererName, width, height, base)
			})
		}
	}
}
