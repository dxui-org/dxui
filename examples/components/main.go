// Command components runs the complete interactive dxui component showcase.
package main

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"strings"
	"time"

	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon"
)

const defaultComponent = "Box"

func main() {
	software := flag.Bool("software", false, "force SDL's named software renderer")
	initial := flag.String("component", defaultComponent, "component selected at startup")
	duration := flag.Duration("duration", 0, "close automatically after this smoke-test duration")
	width := flag.Float64("width", 1180, "initial logical window width")
	height := flag.Float64("height", 760, "initial logical window height")
	flag.Parse()

	renderer := dxui.RendererAuto
	if *software {
		renderer = dxui.RendererSoftware
	}
	state := newGalleryState(*initial)
	options := dxui.AppOptions{
		Title: "dxui component showcase", Width: float32(*width), Height: float32(*height),
		MinWidth: 480, MinHeight: 420, Renderer: renderer,
		Theme: showcaseTheme(false, state.theme), Background: dxui.RGBA(241, 245, 249, 255),
	}
	if *duration > 0 {
		options.OnShown = func(app *dxui.App) { go closeShowcaseAfter(app, *duration) }
	}
	app := dxui.NewApp(options)
	assets := makeAssets()
	examples := componentRegistry(app, state, assets)
	state.selectKnown(examples, *initial)

	if err := app.RunResponsive(func(context dxui.LayoutContext) dxui.View {
		return galleryView(app, state, examples, context)
	}); err != nil {
		log.Fatal(err)
	}
}

func closeShowcaseAfter(app *dxui.App, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	<-timer.C
	if err := app.Update(app.Close); err != nil {
		log.Printf("automatic close: %v", err)
	}
}

type galleryState struct {
	selected, query, feedback string
	iconQuery                 string
	iconPage                  int
	dark, toggle, checkbox    bool
	slider                    float32
	input, password           string
	inputSelection            dxui.TextRange
	textarea                  string
	textareaSelection         dxui.TextRange
	selectValue               string
	tabsValue                 string
	menuValue                 string
	radioValue                string
	controlledScroll          dxui.Point
	keyedInputsReversed       bool
	popoverOpen               bool
	imageStatus               string
	pressCount                int
	theme                     galleryThemeSettings
}

type galleryThemeSettings struct {
	primary, surface, danger, success string
}

const showcaseButtonTheme dxui.ComponentToken = "showcase.button.emphasis"

func newGalleryState(selected string) *galleryState {
	return &galleryState{
		selected: selected, feedback: "Interact with a preview to see event feedback here.",
		input: "Editable value", password: "secret", inputSelection: dxui.TextRange{Start: 0, End: 8},
		slider:            35,
		textarea:          "First line\nSecond editable line\nA long line demonstrates simple word wrapping inside the editor.",
		textareaSelection: dxui.TextRange{Start: 0, End: 5}, selectValue: "stable", tabsValue: "overview", menuValue: "none", radioValue: "stable",
		controlledScroll: dxui.Point{Y: 24}, imageStatus: "Waiting for image callbacks.",
		theme: galleryThemeSettings{primary: "default", surface: "default", danger: "default", success: "default"},
	}
}

func (s *galleryState) selectKnown(examples []componentExample, requested string) {
	for _, example := range examples {
		if strings.EqualFold(example.Name, requested) {
			s.selected = example.Name
			return
		}
	}
	if len(examples) != 0 {
		s.selected = examples[0].Name
	}
}

type galleryAssets struct {
	checker              dxui.ImageSource
	checkerPNG           dxui.ImageSource
	brokenImage          dxui.ImageSource
	star, curve, spinner dxui.IconData
}

func makeAssets() galleryAssets {
	checker := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for y := 0; y < checker.Bounds().Dy(); y++ {
		for x := 0; x < checker.Bounds().Dx(); x++ {
			shade := uint8(65)
			if (x/12+y/12)%2 == 0 {
				shade = 195
			}
			checker.SetNRGBA(x, y, color.NRGBA{R: shade, G: 118, B: 232, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, checker); err != nil {
		panic(err)
	}
	return galleryAssets{
		checker:     dxui.ImageFromGo(checker),
		checkerPNG:  dxui.ImageBytes(encoded.Bytes()),
		brokenImage: dxui.ImageBytes([]byte("not an image")),
		star:        icon.Star(),
		curve:       icon.Spline(),
		spinner:     icon.LoaderCircle(),
	}
}

func showcaseTheme(dark bool, settings galleryThemeSettings) dxui.Theme {
	var theme dxui.Theme
	if dark {
		theme = dxui.DarkTheme()
	} else {
		theme = dxui.LightTheme()
	}
	applyPrimaryTheme(&theme, settings.primary, dark)
	applySurfaceTheme(&theme, settings.surface, dark)
	applyStatusTheme(&theme, dxui.Color.Semantic.Danger, settings.danger, dark)
	applyStatusTheme(&theme, dxui.Color.Semantic.Success, settings.success, dark)
	theme.Components[showcaseButtonTheme] = dxui.ComponentTheme{
		Base: dxui.StylePatch{
			Background: dxui.Some(dxui.TokenColor(dxui.Color.Semantic.Danger)),
			Border:     dxui.Some(dxui.Stroke(1, dxui.TokenColor(dxui.Color.Semantic.Danger))),
			TextColor:  dxui.Some(dxui.TokenColor(dxui.Color.Primitive.White)),
		},
		States: dxui.StateStyles{Hover: dxui.StylePatch{
			Opacity: dxui.Some(float32(.82)),
		}},
	}
	return theme
}

func applyPrimaryTheme(theme *dxui.Theme, name string, dark bool) {
	scale, ok := primitiveColorScaleByName(name)
	if !ok {
		return
	}
	accent, hover := scale.tokens[6], scale.tokens[7]
	if dark {
		accent, hover = scale.tokens[4], scale.tokens[3]
	}
	theme.Semantic.Colors[dxui.Color.Semantic.Accent] = dxui.TokenColor(accent)
	theme.Semantic.Colors[dxui.Color.Semantic.AccentHover] = dxui.TokenColor(hover)
}

func applySurfaceTheme(theme *dxui.Theme, name string, dark bool) {
	scale, ok := primitiveColorScaleByName(name)
	if !ok {
		return
	}
	values := [4]dxui.ColorToken{scale.tokens[0], scale.tokens[1], scale.tokens[10], scale.tokens[4]}
	if dark {
		values = [4]dxui.ColorToken{scale.tokens[10], scale.tokens[9], scale.tokens[0], scale.tokens[6]}
	}
	theme.Semantic.Colors[dxui.Color.Semantic.Surface] = dxui.TokenColor(values[0])
	theme.Semantic.Colors[dxui.Color.Semantic.SurfaceHigh] = dxui.TokenColor(values[1])
	theme.Semantic.Colors[dxui.Color.Semantic.Text] = dxui.TokenColor(values[2])
	theme.Semantic.Colors[dxui.Color.Semantic.Border] = dxui.TokenColor(values[3])
}

func applyStatusTheme(theme *dxui.Theme, semantic dxui.ColorToken, name string, dark bool) {
	scale, ok := primitiveColorScaleByName(name)
	if !ok {
		return
	}
	token := scale.tokens[6]
	if dark {
		token = scale.tokens[4]
	}
	theme.Semantic.Colors[semantic] = dxui.TokenColor(token)
}

func primitiveColorScaleByName(name string) (primitiveColorScale, bool) {
	for _, scale := range primitiveColorScales {
		if scale.name == name {
			return scale, true
		}
	}
	return primitiveColorScale{}, false
}
