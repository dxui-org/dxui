package dxui

import (
	"math"
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/tree"
)

func TestNamedStatePatchesPreserveNamesAndValues(t *testing.T) {
	states := StateStyles{
		Default:  StylePatch{Opacity: Some(float32(.1))},
		Hover:    StylePatch{Opacity: Some(float32(.2))},
		Focus:    StylePatch{Opacity: Some(float32(.3))},
		Checked:  StylePatch{Opacity: Some(float32(.4))},
		Pressed:  StylePatch{Opacity: Some(float32(.5))},
		Disabled: StylePatch{Opacity: Some(float32(.6))},
	}
	wantNames := [...]string{"default", "hover", "focus", "checked", "pressed", "disabled"}
	wantOpacity := [...]float32{.1, .2, .3, .4, .5, .6}
	for index, state := range namedStatePatches(states) {
		if state.name != wantNames[index] {
			t.Errorf("state %d name = %q, want %q", index, state.name, wantNames[index])
		}
		if opacity, set := state.patch.Opacity.get(); !set || opacity != wantOpacity[index] {
			t.Errorf("state %s opacity = %v, %t, want %v, true", state.name, opacity, set, wantOpacity[index])
		}
	}
}

func TestBuiltinPrimitiveColorScalesAreComplete(t *testing.T) {
	theme := LightTheme()
	families := []string{
		"red", "orange", "amber", "yellow", "lime", "green", "emerald", "teal",
		"cyan", "sky", "blue", "indigo", "violet", "purple", "fuchsia", "pink",
		"rose", "slate", "gray", "zinc", "neutral", "stone", "taupe", "mauve",
		"mist", "olive",
	}
	if got, want := len(theme.Primitive.Colors), len(families)*11+6; got != want {
		t.Fatalf("primitive color count = %d, want %d scales plus palette white/black and four compatibility tokens", got, want)
	}
	for _, family := range families {
		prefix := "primitive." + family + "."
		count := 0
		for token, color := range theme.Primitive.Colors {
			if strings.HasPrefix(string(token), prefix) {
				count++
				if color.A != 255 {
					t.Errorf("%s alpha = %d, want 255", token, color.A)
				}
			}
		}
		if count != 11 {
			t.Errorf("%s scale has %d entries, want 11", family, count)
		}
	}
	checks := map[ColorToken]RGBAColor{
		Color.Primitive.Slate50:  RGBA(248, 250, 252, 255),
		Color.Primitive.Slate950: RGBA(2, 6, 23, 255),
		Color.Primitive.Blue600:  RGBA(37, 99, 235, 255),
		Color.Primitive.Red600:   RGBA(220, 38, 38, 255),
		Color.Primitive.Green600: RGBA(22, 163, 74, 255),
		Color.Primitive.Olive950: RGBA(12, 12, 9, 255),
	}
	for token, want := range checks {
		if got := theme.Primitive.Colors[token]; got != want {
			t.Errorf("%s = %+v, want %+v", token, got, want)
		}
	}
}

func TestBuiltinSemanticLightAndDarkMappings(t *testing.T) {
	tests := []struct {
		name  string
		theme Theme
		want  map[ColorToken]ColorToken
	}{
		{name: "light", theme: LightTheme(), want: map[ColorToken]ColorToken{
			Color.Semantic.Surface: Color.Primitive.Slate50, Color.Semantic.SurfaceHigh: Color.Primitive.White,
			Color.Semantic.Text: Color.Primitive.Slate950, Color.Semantic.Accent: Color.Primitive.Emerald600,
			Color.Semantic.AccentHover: Color.Primitive.Emerald700, Color.Semantic.Danger: Color.Primitive.Red600,
			Color.Semantic.Success: Color.Primitive.Green600, Color.Semantic.Border: Color.Primitive.Slate400,
		}},
		{name: "dark", theme: DarkTheme(), want: map[ColorToken]ColorToken{
			Color.Semantic.Surface: Color.Primitive.Slate950, Color.Semantic.SurfaceHigh: Color.Primitive.Slate900,
			Color.Semantic.Text: Color.Primitive.Slate50, Color.Semantic.Accent: Color.Primitive.Emerald400,
			Color.Semantic.AccentHover: Color.Primitive.Emerald300, Color.Semantic.Danger: Color.Primitive.Red400,
			Color.Semantic.Success: Color.Primitive.Green400, Color.Semantic.Border: Color.Primitive.Slate600,
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for semantic, primitive := range test.want {
				value, ok := test.theme.Semantic.Colors[semantic]
				if !ok || !value.set || !value.isToken || value.token != primitive {
					t.Errorf("%s mapping = %+v, want token %s", semantic, value, primitive)
				}
			}
			resolved, err := prepareTheme(test.theme)
			if err != nil {
				t.Fatal(err)
			}
			surface, _ := resolved.color(TokenColor(Color.Semantic.Surface))
			text, _ := resolved.color(TokenColor(Color.Semantic.Text))
			if contrastRatio(surface, text) < 7 {
				t.Errorf("surface/text contrast = %.2f, want at least 7", contrastRatio(surface, text))
			}
		})
	}
}

func TestNewSemanticTokensResolveThroughComponentStates(t *testing.T) {
	source := LightTheme()
	source.Components["status-action"] = ComponentTheme{States: StateStyles{
		Default: StylePatch{Background: Some(TokenColor(Color.Semantic.Danger))},
		Hover:   StylePatch{Background: Some(TokenColor(Color.Semantic.Success))},
	}}
	resolved, err := prepareTheme(source)
	if err != nil {
		t.Fatal(err)
	}
	instance := &tree.Node{}
	props := viewProps{Token: "status-action"}
	visual, err := computeVisual(props, instance, viewButton, resolved)
	if err != nil || visual.background != source.Primitive.Colors[Color.Primitive.Red600] {
		t.Fatalf("danger state = %+v/%v", visual, err)
	}
	instance.State.Hovered = true
	visual, err = computeVisual(props, instance, viewButton, resolved)
	if err != nil || visual.background != source.Primitive.Colors[Color.Primitive.Green600] {
		t.Fatalf("success hover state = %+v/%v", visual, err)
	}

	builtin := source.Components[ComponentButton].States.Hover
	background, ok := builtin.Background.get()
	if !ok || !background.isToken || background.token != Color.Semantic.AccentHover {
		t.Fatalf("button hover background = %+v, want semantic accent hover", background)
	}
}

func TestLegacyColorTokensRemainCompatible(t *testing.T) {
	theme := LightTheme()
	legacy := map[ColorToken]RGBAColor{
		ColorPrimitiveWhite: RGBA(248, 250, 252, 255),
		ColorPrimitiveBlack: RGBA(20, 24, 31, 255),
		ColorPrimitiveBlue:  RGBA(55, 118, 232, 255),
		ColorPrimitiveGray:  RGBA(113, 122, 138, 255),
	}
	for token, want := range legacy {
		if got := theme.Primitive.Colors[token]; got != want {
			t.Errorf("legacy %s = %+v, want %+v", token, got, want)
		}
	}
	if ColorPrimitiveSlate50 != Color.Primitive.Slate50 || ColorSemanticAccent != Color.Semantic.Accent {
		t.Fatal("flat constants and discoverable namespace diverged")
	}
	resolved, err := prepareTheme(theme)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []ColorToken{
		ColorSemanticScrollTrack, ColorSemanticScrollThumb, ColorSemanticSelectHover,
		ColorSemanticSliderFill, ColorSemanticProgressFill, ColorSemanticTabsIndicator,
		ColorSemanticMenuActive,
		ColorSemanticMenuSelected,
	} {
		if _, err := resolved.color(TokenColor(token)); err != nil {
			t.Errorf("legacy semantic token %s: %v", token, err)
		}
	}
}

func contrastRatio(left, right RGBAColor) float64 {
	luminance := func(color RGBAColor) float64 {
		channel := func(value uint8) float64 {
			v := float64(value) / 255
			if v <= .04045 {
				return v / 12.92
			}
			return math.Pow((v+.055)/1.055, 2.4)
		}
		return .2126*channel(color.R) + .7152*channel(color.G) + .0722*channel(color.B)
	}
	a, b := luminance(left), luminance(right)
	if a < b {
		a, b = b, a
	}
	return (a + .05) / (b + .05)
}

func testTheme(surface RGBAColor, spacing float32) Theme {
	return Theme{
		Primitive: PrimitiveTokens{
			Colors:  map[ColorToken]RGBAColor{"p.surface": surface},
			Metrics: map[MetricToken]float32{"p.space": spacing},
		},
		Semantic: SemanticTokens{
			Colors: map[ColorToken]ColorValue{"s.surface": TokenColor("p.surface")},
			Metrics: map[MetricToken]MetricValue{
				"s.space":              TokenMetric("p.space"),
				MetricSemanticTextSize: Metric(15), MetricSemanticLineHeight: Metric(20),
			},
		},
		Components: map[ComponentToken]ComponentTheme{
			"card": {Base: StylePatch{Background: Some(TokenColor("s.surface"))}},
		},
	}
}

func TestThemeFallbackComponentAndLocalOverride(t *testing.T) {
	resolved, err := prepareTheme(testTheme(RGBA(10, 20, 30, 255), 8))
	if err != nil {
		t.Fatal(err)
	}
	instance := &tree.Node{}
	visual, err := computeVisual(viewProps{Token: "card"}, instance, viewBox, resolved)
	if err != nil || visual.background != RGBA(10, 20, 30, 255) {
		t.Fatalf("component fallback = %+v/%v", visual, err)
	}
	visual, err = computeVisual(viewProps{
		Token: "card", Style: Style{Background: LiteralColor(RGBA(1, 2, 3, 255))},
	}, instance, viewBox, resolved)
	if err != nil || visual.background != RGBA(1, 2, 3, 255) {
		t.Fatalf("local override = %+v/%v", visual, err)
	}
}

func TestSemanticComponentsUseBuiltInDefaultsWithoutLocalToken(t *testing.T) {
	resolved, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	visual, err := computeVisual(viewProps{}, &tree.Node{}, viewButton, resolved)
	if err != nil {
		t.Fatal(err)
	}
	want, err := resolved.color(TokenColor(ColorSemanticAccent))
	if err != nil || !visual.backgroundSet || visual.background != want {
		t.Fatalf("button default = %+v, want semantic accent %+v", visual, want)
	}
}

func TestThemeRejectsMissingTokenAtomically(t *testing.T) {
	app := NewApp(AppOptions{Theme: testTheme(RGBA(10, 20, 30, 255), 8)})
	before := app.theme
	broken := testTheme(RGBA(4, 5, 6, 255), 8)
	broken.Components["card"] = ComponentTheme{Base: StylePatch{Background: Some(TokenColor("missing"))}}
	if err := app.SetTheme(broken); err == nil {
		t.Fatal("missing component token accepted")
	}
	if !resolvedThemesEqual(app.theme, before) {
		t.Fatal("failed SetTheme changed the active theme")
	}
}

func TestSetThemeRejectsMissingActiveLayoutTokenAtomically(t *testing.T) {
	initial := testTheme(RGBA(10, 20, 30, 255), 8)
	app := NewApp(AppOptions{Width: 80, Height: 40, Theme: initial})
	app.root = func() View {
		return Box(BoxProps{Style: Style{Padding: UniformEdges(TokenMetric("s.space"))}})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.theme
	missing := Theme{Primitive: PrimitiveTokens{Colors: map[ColorToken]RGBAColor{"only": RGBA(1, 2, 3, 255)}}}
	if err := app.SetTheme(missing); err == nil {
		t.Fatal("theme missing active layout token was accepted")
	}
	if !resolvedThemesEqual(before, app.theme) || app.Diagnostics().LayoutCount != 1 || app.Diagnostics().PaintCount != 1 {
		t.Fatalf("failed theme changed state: theme=%v diagnostics=%+v", resolvedThemesEqual(before, app.theme), app.Diagnostics())
	}
}

func TestThemeInputIsDeepCopied(t *testing.T) {
	source := testTheme(RGBA(10, 20, 30, 255), 8)
	app := NewApp(AppOptions{Theme: source})
	source.Primitive.Colors["p.surface"] = RGBA(200, 0, 0, 255)
	source.Components["card"] = ComponentTheme{}
	color, err := app.theme.color(TokenColor("s.surface"))
	_, backgroundSet := app.theme.source.Components["card"].Base.Background.get()
	if err != nil || color != RGBA(10, 20, 30, 255) || !backgroundSet {
		t.Fatalf("theme was not copied: color=%+v err=%v component=%+v", color, err, app.theme.source.Components["card"])
	}
}

func TestBuiltinPaletteThemeIsDeepCopied(t *testing.T) {
	source := LightTheme()
	want := source.Primitive.Colors[Color.Primitive.Emerald600]
	app := NewApp(AppOptions{Theme: source})
	source.Primitive.Colors[Color.Primitive.Emerald600] = RGBA(1, 2, 3, 255)
	got, err := app.theme.color(TokenColor(Color.Semantic.Accent))
	if err != nil || got != want {
		t.Fatalf("copied accent = %+v/%v, want %+v", got, err, want)
	}
}

func TestStatePrecedenceAllCombinations(t *testing.T) {
	styles := StateStyles{
		Default:  StylePatch{Opacity: Some(float32(.1))},
		Hover:    StylePatch{Opacity: Some(float32(.2))},
		Focus:    StylePatch{Opacity: Some(float32(.3))},
		Checked:  StylePatch{Opacity: Some(float32(.4))},
		Pressed:  StylePatch{Opacity: Some(float32(.5))},
		Disabled: StylePatch{Opacity: Some(float32(.6))},
	}
	for bits := 0; bits < 32; bits++ {
		state := visualState{
			Hover: bits&1 != 0, Focus: bits&2 != 0, Checked: bits&4 != 0,
			Pressed: bits&8 != 0, Disabled: bits&16 != 0,
		}
		patch := resolvedStatePatch(styles, StateStyles{}, state, stateHover|stateFocus|stateChecked|statePressed|stateDisabled)
		got, _ := patch.Opacity.get()
		want := float32(.1)
		switch {
		case state.Disabled:
			want = .6
		case state.Pressed:
			want = .5
		case state.Checked:
			want = .4
		case state.Focus:
			want = .3
		case state.Hover:
			want = .2
		}
		if got != want {
			t.Fatalf("bits %05b opacity = %v, want %v", bits, got, want)
		}
	}
}

func TestStateResolutionIgnoresInapplicableStates(t *testing.T) {
	styles := StateStyles{
		Default:  StylePatch{Opacity: Some(float32(.1))},
		Hover:    StylePatch{Opacity: Some(float32(.2))},
		Focus:    StylePatch{Opacity: Some(float32(.3))},
		Checked:  StylePatch{Opacity: Some(float32(.4))},
		Pressed:  StylePatch{Opacity: Some(float32(.5))},
		Disabled: StylePatch{Opacity: Some(float32(.6))},
	}
	state := visualState{Hover: true, Focus: true, Checked: true, Pressed: true, Disabled: true}
	patch := resolvedStatePatch(styles, StateStyles{}, state, stateChecked)
	got, _ := patch.Opacity.get()
	if got != .4 {
		t.Fatalf("opacity = %v, want checked style .4", got)
	}
}

func TestThemeSwitchDirtyScopeAndNoOp(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 60, Theme: testTheme(RGBA(10, 20, 30, 255), 8)})
	app.root = func() View {
		return Box(BoxProps{
			Token: "card", Style: Style{Padding: UniformEdges(TokenMetric("s.space"))},
		})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	events := &appFakeEvents{}
	app.running, app.events = true, events

	colorOnly := testTheme(RGBA(40, 50, 60, 255), 8)
	if err := app.SetTheme(colorOnly); err != nil {
		t.Fatal(err)
	}
	dirty, err := app.runQueuedWork()
	if err != nil || !dirty {
		t.Fatalf("color theme work = %v/%v", dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 1 || got.PaintCount != 2 {
		t.Fatalf("color theme diagnostics = %+v", got)
	}
	wakes := events.wakes
	if err := app.SetTheme(colorOnly); err != nil {
		t.Fatal(err)
	}
	if events.wakes != wakes {
		t.Fatal("unchanged SetTheme woke the frame loop")
	}

	metric := testTheme(RGBA(40, 50, 60, 255), 12)
	if err := app.SetTheme(metric); err != nil {
		t.Fatal(err)
	}
	if dirty, err = app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("metric theme work = %v/%v", dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 2 || got.PaintCount != 3 {
		t.Fatalf("metric theme diagnostics = %+v", got)
	}
}

func TestBuiltinPrimitiveColorChangeRepaintsWithoutLayout(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40, Theme: LightTheme()})
	app.root = func() View {
		return Button(ButtonProps{}, Text(TextProps{Value: "accent"}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.running, app.events = true, &appFakeEvents{}
	next := LightTheme()
	next.Primitive.Colors[Color.Primitive.Emerald600] = RGBA(12, 34, 56, 255)
	if err := app.SetTheme(next); err != nil {
		t.Fatal(err)
	}
	dirty, err := app.runQueuedWork()
	if err != nil || !dirty {
		t.Fatalf("palette theme work = %v/%v", dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 1 || got.PaintCount != 2 {
		t.Fatalf("palette-only theme diagnostics = %+v", got)
	}
}

func TestUnusedThemeTokenChangeDoesNotWakeOrPaint(t *testing.T) {
	base := testTheme(RGBA(10, 20, 30, 255), 8)
	base.Primitive.Colors["unused"] = RGBA(1, 1, 1, 255)
	app := NewApp(AppOptions{Width: 20, Height: 20, Theme: base})
	app.root = func() View { return Text(TextProps{}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	events := &appFakeEvents{}
	app.running, app.events = true, events
	next := copyTheme(base)
	next.Primitive.Colors["unused"] = RGBA(2, 2, 2, 255)
	if err := app.SetTheme(next); err != nil {
		t.Fatal(err)
	}
	if events.wakes != 0 || app.themeDirty != 0 || app.Diagnostics().PaintCount != 1 {
		t.Fatalf("unused token scheduled work: wakes=%d dirty=%v diagnostics=%+v", events.wakes, app.themeDirty, app.Diagnostics())
	}
}

func TestQueuedThemeSwitchStillProducesPaint(t *testing.T) {
	app := NewApp(AppOptions{Width: 40, Height: 20, Theme: testTheme(RGBA(10, 20, 30, 255), 8)})
	app.root = func() View { return Box(BoxProps{Token: "card"}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	events := &appFakeEvents{}
	app.running, app.events = true, events
	if err := app.Update(func() {
		if err := app.SetTheme(testTheme(RGBA(50, 60, 70, 255), 8)); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	dirty, err := app.runQueuedWork()
	if err != nil || !dirty {
		t.Fatalf("queued theme switch = %v/%v", dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 1 || got.PaintCount != 2 {
		t.Fatalf("queued theme diagnostics = %+v", got)
	}
	if events.wakes != 1 {
		t.Fatalf("update/theme wakes = %d, want update wake only", events.wakes)
	}
}

func TestDisplayListStableZOrderVisibilityAndBounds(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 60})
	app.root = func() View {
		box := func(key string, z int, hidden bool, color RGBAColor) View {
			visibility := Visible
			if hidden {
				visibility = Hidden
			}
			return Text(TextProps{Key: key, Style: Style{
				Width: Px(20), Height: Px(20), Position: PositionAbsolute, ZIndex: z,
				Background: LiteralColor(color), Radius: UniformCorners(Metric(4)), Visibility: visibility,
			}})
		}
		return Box(BoxProps{Style: Style{
			Overflow: OverflowClip, Opacity: Some(float32(.5)),
		}}, box("front", 2, false, RGBA(2, 0, 0, 200)), box("hidden", 1, true, RGBA(1, 0, 0, 255)), box("back", 0, false, RGBA(0, 0, 0, 255)))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	list := app.currentDisplay()
	if err := paint.Replay(list, &paint.RecordingPainter{}); err != nil {
		t.Fatal(err)
	}
	var fills []paint.Command
	for _, command := range list {
		if command.Kind == paint.CommandFillRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect {
			fills = append(fills, command)
		}
	}
	if len(fills) != 2 || fills[0].Color.R != 0 || fills[1].Color.R != 2 {
		t.Fatalf("fill z/visibility order = %#v", fills)
	}
}

func TestZIndexAndOverflowChangesStayPaintOnlyAndUpdateHitTest(t *testing.T) {
	z := 2
	clip := OverflowVisible
	app := NewApp(AppOptions{Width: 100, Height: 60})
	app.root = func() View {
		button := Button(ButtonProps{Style: Style{
			Width: Px(30), Height: Px(20), Position: PositionAbsolute, Insets: Insets{Left: Px(40)}, ZIndex: z,
			Background: LiteralColor(RGBA(200, 0, 0, 255)),
		}}, Text(TextProps{}))
		return Box(BoxProps{},
			Box(BoxProps{Style: Style{Width: Px(50), Height: Px(40), Overflow: clip}}, button),
			Text(TextProps{Style: Style{
				Width: Px(20), Height: Px(20), Position: PositionAbsolute, ZIndex: 1,
				Background: LiteralColor(RGBA(0, 0, 200, 255)),
			}}),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.currentDisplay().HitTestAt(65, 10); !ok {
		t.Fatal("unclipped button was not hit")
	}
	app.running = true
	z, clip = -1, OverflowClip
	app.invalidate()
	dirty, err := app.runQueuedWork()
	if err != nil || !dirty {
		t.Fatalf("paint-only update = %v/%v", dirty, err)
	}
	if app.Diagnostics().LayoutCount != 1 {
		t.Fatalf("z-index/overflow change relaid out: %+v", app.Diagnostics())
	}
	if _, ok := app.currentDisplay().HitTestAt(65, 10); ok {
		t.Fatal("nested overflow clip did not remove hit")
	}
	var colors []paint.Color
	for _, command := range app.currentDisplay() {
		if command.Kind == paint.CommandFillRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect {
			colors = append(colors, command.Color)
		}
	}
	if len(colors) != 2 || colors[0].R != 200 || colors[1].B != 200 {
		t.Fatalf("paint-only z-order = %#v", colors)
	}
}
