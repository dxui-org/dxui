//go:build sdl_golden

package dxui

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/tree"
)

// Generates actual root-component display data for the SDL-boundary replay test.
// Explicit export keeps ordinary tests SDL-free and adds no production capture API.
func TestExportButtonVisualScenes(t *testing.T) {
	directory := os.Getenv("DXUI_BUTTON_VISUAL_DIR")
	if directory == "" {
		t.Skip("set DXUI_BUTTON_VISUAL_DIR to export native scenes")
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	variants := []string{"Filled", "Soft", "Outline", "Dashed", "Ghost", "Link"}
	tones := []string{"Primary", "Secondary", "Success", "Info", "Warn", "Danger"}
	for _, dark := range []bool{false, true} {
		source, name := LightTheme(), "light"
		if dark {
			source, name = DarkTheme(), "dark"
		}
		for state := 0; state < 5; state++ {
			app := NewApp(AppOptions{Width: 1060, Height: 480, Theme: source})
			app.root = func() View {
				rows := []View{Label(fmt.Sprintf("%s | %s | columns: Primary / Secondary / Success / Info / Warn / Danger", name, []string{"Default", "Hover", "Pressed", "Keyboard Focus", "Disabled"}[state]))}
				for v, label := range variants {
					cells := []View{Text(TextProps{Value: label, Style: Style{Width: Px(70)}})}
					for tone, toneName := range tones {
						cells = append(cells, TextButton(ButtonProps{Variant: ButtonVariant(v), Tone: ButtonTone(tone), Disabled: state == 4, Style: Style{Width: Px(142)}}, toneName))
					}
					rows = append(rows, Box(BoxProps{Direction: Horizontal, Gap: 14, Align: AlignCenter}, cells...))
				}
				rows = append(rows, Label("Sizes: Small / Normal / Large; Outline + Danger"), Box(BoxProps{Direction: Horizontal, Gap: 20, Align: AlignCenter},
					TextButton(ButtonProps{Variant: ButtonOutline, Tone: ButtonDanger, Size: ButtonSmall}, "Small"),
					TextButton(ButtonProps{Variant: ButtonOutline, Tone: ButtonDanger}, "Normal"),
					TextButton(ButtonProps{Variant: ButtonOutline, Tone: ButtonDanger, Size: ButtonLarge}, "Large")))
				return Box(BoxProps{Gap: 14, Style: Style{Padding: Padding(16), Background: TokenColor(ColorSemanticSurface)}}, rows...)
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			var setState func(*tree.Node)
			setState = func(node *tree.Node) {
				if node.Kind == tree.Kind(viewButton) {
					node.State.Hovered = state == 1
					node.State.Pressed = state == 2
					node.State.FocusVisible = state == 3
				}
				for _, child := range node.Children {
					setState(child)
				}
			}
			setState(app.retained.Root())
			list, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, app.text, app.images, 1.25, 1.25, 8<<20, nil)
			if err != nil {
				t.Fatal(err)
			}
			background, _ := app.theme.color(TokenColor(ColorSemanticSurface))
			scene := struct {
				Width, Height int32
				Background    paint.Color
				List          paint.DisplayList
			}{1060, 480, paintColor(background), list}
			path := filepath.Join(directory, fmt.Sprintf("%s-%d.gob", name, state))
			file, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := gob.NewEncoder(file).Encode(scene); err != nil {
				file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			app.releaseTextEngine()
			t.Log(path)
		}
	}
}
