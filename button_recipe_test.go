package dxui

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func TestButtonRecipesMatrix(t *testing.T) {
	for _, dark := range []bool{false, true} {
		source := LightTheme()
		if dark {
			source = DarkTheme()
		}
		theme, err := prepareTheme(source)
		if err != nil {
			t.Fatal(err)
		}
		surface, _ := theme.color(TokenColor(ColorSemanticSurface))
		for variant := ButtonFilled; variant <= ButtonLink; variant++ {
			for tone := ButtonPrimary; tone <= ButtonDanger; tone++ {
				t.Run(fmt.Sprintf("dark=%t/variant=%d/tone=%d", dark, variant, tone), func(t *testing.T) {
					props := ButtonProps{Variant: variant, Tone: tone}.common()
					normal, err := computeVisual(props, &tree.Node{}, viewButton, theme)
					if err != nil {
						t.Fatal(err)
					}
					if !normal.textColorSet || !normal.backgroundSet {
						t.Fatal("missing recipe colors")
					}
					if variant == ButtonOutline || variant == ButtonDashed {
						if normal.background.A != 0 || normal.border.width != 1 {
							t.Fatalf("outline: %+v", normal)
						}
						if (normal.border.pattern == BorderDashed) != (variant == ButtonDashed) {
							t.Fatal("border pattern")
						}
					}
					if variant >= ButtonGhost && normal.background.A != 0 {
						t.Fatal("ghost/link default fill")
					}
					for flags := 0; flags < 32; flags++ {
						node := &tree.Node{}
						node.State.Hovered = flags&1 != 0
						node.State.FocusVisible = flags&2 != 0
						node.State.Pressed = flags&4 != 0
						node.Properties.Semantics.Disabled = flags&8 != 0
						node.State.Checked = flags&16 != 0
						got, err := computeVisual(props, node, viewButton, theme)
						if err != nil {
							t.Fatal(err)
						}
						if !node.Properties.Semantics.Disabled && variant != ButtonFilled && !(variant == ButtonGhost && tone == ButtonPrimary) {
							bg := buttonMix(surface, got.background, float32(got.background.A)/255)
							if ratio := contrastRatio(bg, got.textColor); ratio < 4.5 {
								t.Errorf("state=%d contrast=%.2f", flags, ratio)
							}
						}
						if node.Properties.Semantics.Disabled {
							disabled := &tree.Node{}
							disabled.Properties.Semantics.Disabled = true
							want, _ := computeVisual(props, disabled, viewButton, theme)
							if !reflect.DeepEqual(got, want) {
								t.Fatal("Disabled did not suppress interaction/Checked")
							}
							if got.opacity >= 1 {
								t.Fatal("missing disabled feedback")
							}
						} else if node.State.FocusVisible && (len(got.shadows) != 1 || got.shadows[0].Spread != 2) {
							t.Fatal("missing focus ring")
						}
					}
					bg := buttonMix(surface, normal.background, float32(normal.background.A)/255)
					ratio := contrastRatio(bg, normal.textColor)
					minimum := 4.5
					// Preserve the original Filled and Ghost Primary contrast verbatim.
					if variant == ButtonFilled || variant == ButtonGhost && tone == ButtonPrimary {
						minimum = 3
					}
					if ratio < minimum {
						t.Errorf("normal contrast %.2f < %.1f", ratio, minimum)
					}
				})
			}
		}
	}
}

func TestButtonDefaultAlias(t *testing.T) {
	if ButtonPrimary != 0 || ButtonDefault != ButtonPrimary {
		t.Fatal("Default must alias zero-valued Primary")
	}
	for _, source := range []Theme{LightTheme(), DarkTheme()} {
		custom := source.Components[ComponentButton]
		custom.Base.Background = Some(ColorRGBA(71, 32, 99, 255))
		source.Components[ComponentButton] = custom
		theme, err := prepareTheme(source)
		if err != nil {
			t.Fatal(err)
		}
		for variant := ButtonFilled; variant <= ButtonLink; variant++ {
			for flags := 0; flags < 16; flags++ {
				node := &tree.Node{}
				node.State.Hovered = flags&1 != 0
				node.State.FocusVisible = flags&2 != 0
				node.State.Pressed = flags&4 != 0
				node.Properties.Semantics.Disabled = flags&8 != 0
				want, err := computeVisual(ButtonProps{Variant: variant}.common(), node, viewButton, theme)
				if err != nil {
					t.Fatal(err)
				}
				for _, tone := range []ButtonTone{ButtonDefault, ButtonPrimary} {
					got, err := computeVisual(ButtonProps{Variant: variant, Tone: tone}.common(), node, viewButton, theme)
					if err != nil || !reflect.DeepEqual(got, want) {
						t.Fatalf("variant=%d flags=%d alias mismatch: %v", variant, flags, err)
					}
				}
			}
		}
	}
}

func TestButtonInfoWarnSemanticColors(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, tc := range []struct {
			tone                          ButtonTone
			token, namespace, light, dark ColorToken
		}{
			{ButtonInfo, ColorSemanticInfo, Color.Semantic.Info, Color.Primitive.Sky600, Color.Primitive.Sky400},
			{ButtonWarn, ColorSemanticWarn, Color.Semantic.Warn, Color.Primitive.Amber600, Color.Primitive.Amber400},
		} {
			source := LightTheme()
			primitive := tc.light
			if dark {
				source = DarkTheme()
				primitive = tc.dark
			}
			if tc.namespace != tc.token || source.Semantic.Colors[tc.token] != TokenColor(primitive) {
				t.Fatal("incorrect semantic mapping")
			}
			before, err := prepareTheme(source)
			if err != nil {
				t.Fatal(err)
			}
			// Changing only this semantic token must affect its recipe, while
			// the unrelated Success tone remains unchanged.
			source.Semantic.Colors[tc.token] = ColorRGBA(110, 50, 150, 255)
			after, err := prepareTheme(source)
			if err != nil {
				t.Fatal(err)
			}
			for variant := ButtonFilled; variant <= ButtonLink; variant++ {
				for flags := 0; flags < 16; flags++ {
					node := &tree.Node{}
					node.State.Hovered = flags&1 != 0
					node.State.FocusVisible = flags&2 != 0
					node.State.Pressed = flags&4 != 0
					node.Properties.Semantics.Disabled = flags&8 != 0
					for _, tone := range []ButtonTone{tc.tone, ButtonSuccess} {
						props := ButtonProps{Variant: variant, Tone: tone}.common()
						a, err := computeVisual(props, node, viewButton, before)
						if err != nil {
							t.Fatal(err)
						}
						b, err := computeVisual(props, node, viewButton, after)
						if err != nil {
							t.Fatal(err)
						}
						// Pressed Link deliberately uses semantic.text for every tone.
						changed := tone == tc.tone && !(variant == ButtonLink && node.State.Pressed && !node.Properties.Semantics.Disabled)
						if reflect.DeepEqual(a, b) == changed {
							t.Fatalf("dark=%t tone=%d variant=%d flags=%d semantic dependency", dark, tone, variant, flags)
						}
					}
				}
				if allocs := testing.AllocsPerRun(100, func() { _, _ = buttonRecipe(after, variant, tc.tone) }); allocs != 0 {
					t.Fatalf("recipe allocations=%g", allocs)
				}
			}
		}
	}
}

func TestButtonLegacyRecipeCompatibility(t *testing.T) {
	for _, source := range []Theme{LightTheme(), DarkTheme()} {
		theme, err := prepareTheme(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			variant ButtonVariant
			tone    ButtonTone
			token   ComponentToken
		}{
			{ButtonFilled, ButtonPrimary, ComponentButton}, {ButtonFilled, ButtonSecondary, ComponentButtonSecondary},
			{ButtonFilled, ButtonDanger, ComponentButtonDanger}, {ButtonGhost, ButtonPrimary, ComponentButtonGhost},
		} {
			got, err := buttonRecipe(theme, tc.variant, tc.tone)
			if err != nil || !reflect.DeepEqual(got, source.Components[tc.token]) {
				t.Fatalf("legacy %s changed: %v", tc.token, err)
			}
		}
	}
	props := ButtonProps{}.common()
	if props.Style.Padding != (EdgeValues{Top: TokenMetric(MetricComponentButtonPaddingY), Bottom: TokenMetric(MetricComponentButtonPaddingY), Left: TokenMetric(MetricComponentButtonPaddingX), Right: TokenMetric(MetricComponentButtonPaddingX)}) || props.Style.MinHeight.kind != lengthAuto {
		t.Fatal("zero metrics changed")
	}
}

func TestButtonRecipeCascade(t *testing.T) {
	source := LightTheme()
	color := func(v uint8) ColorValue { return ColorRGBA(v, 0, 0, 255) }
	source.Components["custom-button"] = ComponentTheme{Base: StylePatch{Background: Some(color(10))}, States: StateStyles{Default: StylePatch{Background: Some(color(20))}, Hover: StylePatch{Background: Some(color(50))}}}
	theme, err := prepareTheme(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []ComponentToken{"", "custom-button"} {
		props := ButtonProps{Variant: ButtonSoft, Tone: ButtonDanger, Token: token, Style: Style{Background: color(30)}, States: StateStyles{Default: StylePatch{Background: Some(color(40))}, Hover: StylePatch{Background: Some(color(60))}, Pressed: StylePatch{Background: Some(color(70))}, Disabled: StylePatch{Background: Some(color(80))}}}.common()
		node := &tree.Node{}
		for _, want := range []uint8{40, 60, 70, 80, 90} {
			switch want {
			case 60:
				node.State.Hovered = true
			case 70:
				node.State.Pressed = true
			case 80:
				node.Properties.Semantics.Disabled = true
			case 90:
				props.Style.Force.Background = Some(color(90))
			}
			got, err := computeVisual(props, node, viewButton, theme)
			if err != nil || got.background.R != want {
				t.Fatalf("token=%s want=%d got=%+v err=%v", token, want, got, err)
			}
		}
	}
	node := &tree.Node{}
	node.State.Hovered = true
	got, err := computeVisual(ButtonProps{Variant: ButtonDashed, Tone: ButtonSuccess, Token: "custom-button"}.common(), node, viewButton, theme)
	if err != nil || got.background.R != 50 || got.border.width != 0 {
		t.Fatal("Token did not replace full recipe")
	}
}

func TestButtonSizeInheritanceAndOverrides(t *testing.T) {
	for _, tc := range []struct {
		size                           ButtonSize
		font, line, height, padx, pady float32
	}{
		{ButtonNormal, 0, 0, 36, 14, 8}, {ButtonSmall, 12, 18, 30, 10, 5}, {ButtonLarge, 18, 26, 48, 20, 12},
	} {
		t.Run(fmt.Sprint(tc.size), func(t *testing.T) {
			child := Box(BoxProps{}, Label("Inherited"), Text(TextProps{Value: "Explicit", Style: Style{Text: TextStyle{Size: 21, LineHeight: 29}}}))
			view := Button(ButtonProps{Size: tc.size}, child)
			app := NewApp(AppOptions{Width: 200, Height: 160})
			defer app.releaseTextEngine()
			app.root = func() View { return view }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			inherited := app.retained.Root().Children[0].Children[0].Properties.Content
			explicit := app.retained.Root().Children[0].Children[1].Properties.Content
			if tc.font == 0 {
				if !inherited.FontSize.IsToken {
					t.Fatal("Normal lost theme text metrics")
				}
			} else if inherited.FontSize.Literal != tc.font || inherited.LineHeight.Literal != tc.line {
				t.Fatalf("inherited %+v", inherited)
			}
			if explicit.FontSize.Literal != 21 || explicit.LineHeight.Literal != 29 {
				t.Fatal("child metrics overwritten")
			}
			if child.node.children[0].node.text.Style.Text.Size != 0 {
				t.Fatal("user child mutated")
			}
			props := view.node.button.common()
			x, _ := app.theme.metric(props.Style.Padding.Left)
			y, _ := app.theme.metric(props.Style.Padding.Top)
			if x != tc.padx || y != tc.pady {
				t.Fatalf("padding=%v/%v", x, y)
			}
			explicitView := Button(ButtonProps{Size: tc.size, Style: Style{Height: Px(60), MinHeight: Px(55), Padding: Padding(0), Text: TextStyle{Size: 19, LineHeight: 25}}}, Label("Inherited"))
			desc, err := describeView(explicitView)
			if err != nil {
				t.Fatal(err)
			}
			if desc.Properties.Layout.Height.Value != 60 || desc.Properties.Layout.MinHeight.Value != 55 || desc.Properties.Layout.Padding.Left.Literal != 0 || desc.Children[0].Properties.Content.FontSize.Literal != 19 {
				t.Fatalf("explicit override: %+v", desc)
			}
		})
	}
}

func TestButtonRecipeChangesIdentityAndRollback(t *testing.T) {
	props := ButtonProps{Key: "action"}
	count := 0
	props.OnPress = func() { count++ }
	app := NewApp(AppOptions{Width: 180, Height: 80})
	defer app.releaseTextEngine()
	app.root = func() View { return TextButton(props, "Save") }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := app.retained.Root().ID
	for _, event := range []platform.Event{appKey(platform.EventKeyDown, platform.KeyTab, false), appPointer(platform.EventMouseDown, 8, 8)} {
		if _, _, err := app.handleEvent(event); err != nil {
			t.Fatal(err)
		}
	}
	for _, variant := range []ButtonVariant{ButtonSoft, ButtonOutline, ButtonDashed, ButtonGhost} {
		props.Variant = variant
		props.Tone = ButtonDanger
		changes, err := app.buildAndCommit()
		if err != nil {
			t.Fatal(err)
		}
		node := app.retained.Root()
		if node.ID != id || !node.State.Focused || !node.State.Pressed || changes.Has(tree.DirtyLayout) {
			t.Fatalf("identity/state/invalidation %d %+v %v", node.ID, node.State, changes)
		}
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 8, 8)); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("capture lost")
	}
	props.Size = ButtonLarge
	changes, err := app.buildAndCommit()
	if err != nil || !changes.Has(tree.DirtyLayout) || app.retained.Root().ID != id || !app.retained.Root().State.Focused {
		t.Fatal("size identity/layout")
	}
	for _, bad := range []ButtonProps{{Variant: ButtonVariant(255)}, {Tone: ButtonTone(255)}, {Size: ButtonSize(255), Token: ComponentButton}} {
		root, display := app.retained.Root(), app.display
		props = bad
		if _, err := app.buildAndCommit(); err == nil {
			t.Fatal("invalid enum committed")
		}
		if app.retained.Root() != root || !displayListsEqual(app.display, display) {
			t.Fatal("failed transaction changed frame")
		}
	}
}

func TestButtonOverriddenSizeIsPaintOnly(t *testing.T) {
	size := ButtonSmall
	app := NewApp(AppOptions{Width: 200, Height: 100})
	defer app.releaseTextEngine()
	app.root = func() View {
		return Button(ButtonProps{Size: size, Style: Style{Height: Px(50), Padding: Padding(4)}}, Text(TextProps{Value: "Fixed", Style: Style{Text: TextStyle{Size: 16, LineHeight: 22}}}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	size = ButtonLarge
	changes, err := app.buildAndCommit()
	if err != nil || changes.Has(tree.DirtyLayout) {
		t.Fatalf("overridden size relayout: %v %v", changes, err)
	}
}

func TestButtonRecipeThemeSwitch(t *testing.T) {
	for variant := ButtonFilled; variant <= ButtonLink; variant++ {
		for tone := ButtonPrimary; tone <= ButtonDanger; tone++ {
			app := NewApp(AppOptions{Width: 180, Height: 80})
			app.root = func() View { return TextButton(ButtonProps{Variant: variant, Tone: tone, Size: ButtonSmall}, "Theme") }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			before := app.display
			id := app.retained.Root().ID
			if err := app.SetTheme(DarkTheme()); err != nil {
				t.Fatal(err)
			}
			if app.themeDirty&tree.DirtyLayout != 0 {
				t.Fatal("color theme change relayout")
			}
			if _, err := app.runQueuedWork(); err != nil {
				t.Fatal(err)
			}
			if displayListsEqual(before, app.display) || app.retained.Root().ID != id {
				t.Fatalf("theme did not update %d/%d", variant, tone)
			}
			next := DarkTheme()
			next.Semantic.Metrics[MetricSemanticTextSize] = Metric(30)
			resolved, err := prepareTheme(next)
			if err != nil {
				t.Fatal(err)
			}
			if themeChangesLayout(app.view, app.theme, resolved) {
				t.Fatal("inherited Small font has false default-font dependency")
			}
			app.releaseTextEngine()
		}
	}
}

func TestButtonRecipeMemoryEvidence(t *testing.T) {
	type previousButtonProps struct {
		Key      string
		Style    Style
		Token    ComponentToken
		States   StateStyles
		Pointer  PointerBehavior
		Disabled bool
		OnPress  func()
	}
	type previousViewProps struct {
		Key     string
		Style   Style
		Token   ComponentToken
		States  StateStyles
		Pointer PointerBehavior
	}
	t.Logf("previous ButtonProps=%d viewProps=%d bytes (same host layouts without added fields)", unsafe.Sizeof(previousButtonProps{}), unsafe.Sizeof(previousViewProps{}))

	t.Logf("ButtonProps=%d viewProps=%d buttonTypography=%d bytes", unsafe.Sizeof(ButtonProps{}), unsafe.Sizeof(viewProps{}), unsafe.Sizeof(buttonTypography{}))
	theme, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		if _, err := buttonRecipe(theme, ButtonSoft, ButtonSuccess); err != nil {
			panic(err)
		}
	})
	t.Logf("Soft/Success recipe allocations=%g/run", allocs)
	if allocs != 0 {
		t.Fatal("recipe allocates on event path")
	}
}

func BenchmarkButtonRecipe(b *testing.B) {
	theme, err := prepareTheme(LightTheme())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := buttonRecipe(theme, ButtonSoft, ButtonSuccess); err != nil {
			b.Fatal(err)
		}
	}
}

func TestButtonRecipeActivationMatrix(t *testing.T) {
	for variant := ButtonFilled; variant <= ButtonLink; variant++ {
		for _, disabled := range []bool{false, true} {
			for _, pointerNone := range []bool{false, true} {
				count := 0
				props := ButtonProps{Variant: variant, Tone: ButtonSuccess, Disabled: disabled, OnPress: func() { count++ }}
				if pointerNone {
					props.Pointer = PointerNone
				}
				app := NewApp(AppOptions{Width: 180, Height: 70})
				app.root = func() View { return TextButton(props, "Action") }
				if err := app.buildRoot(); err != nil {
					t.Fatal(err)
				}
				events := []platform.Event{appPointer(platform.EventMouseDown, 5, 5), appPointer(platform.EventMouseUp, 5, 5), appKey(platform.EventKeyDown, platform.KeyTab, false)}
				for _, key := range []platform.Key{platform.KeyEnter, platform.KeySpace} {
					events = append(events, appKey(platform.EventKeyDown, key, false), platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: key, Repeat: true}}, appKey(platform.EventKeyUp, key, false))
				}
				for _, event := range events {
					if _, _, err := app.handleEvent(event); err != nil {
						t.Fatal(err)
					}
				}
				want := 3
				if pointerNone {
					want = 2
				}
				if disabled {
					want = 0
				}
				if count != want {
					t.Fatalf("variant=%d disabled=%t pointerNone=%t actions=%d want=%d", variant, disabled, pointerNone, count, want)
				}
				app.releaseTextEngine()
			}
		}
	}
}

func TestButtonTransparentFocusRing(t *testing.T) {
	for _, source := range []Theme{LightTheme(), DarkTheme()} {
		for _, variant := range []ButtonVariant{ButtonFilled, ButtonOutline, ButtonDashed, ButtonGhost, ButtonLink} {
			app := NewApp(AppOptions{Width: 180, Height: 80, Theme: source})
			app.root = func() View { return TextButton(ButtonProps{Variant: variant}, "Focus") }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			node := app.retained.Root()
			node.State.FocusVisible = true
			list, err := buildDisplayList(app.view, node, app.geometry, app.theme, app.text, app.images, 1, 1, 2<<20, nil)
			if err != nil {
				t.Fatal(err)
			}
			ring, shadow := false, false
			for _, command := range list {
				if command.NodeID != node.ID {
					continue
				}
				if command.Kind == paint.CommandDrawShadow {
					shadow = true
				}
				if command.Kind == paint.CommandStrokeRoundedRect && command.Width == 2 {
					ring = true
				}
			}
			if variant == ButtonFilled {
				if !shadow || ring {
					t.Fatal("opaque legacy focus changed")
				}
			} else if shadow || !ring {
				t.Fatal("transparent focus paints an interior")
			}
			app.releaseTextEngine()
		}
	}
}

func TestButtonRecipeThemeValidationAndCopies(t *testing.T) {
	app := NewApp(AppOptions{Width: 180, Height: 80})
	defer app.releaseTextEngine()
	app.root = func() View { return TextButton(ButtonProps{Variant: ButtonSoft, Tone: ButtonSuccess}, "Theme") }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	old, display := app.theme, app.display
	broken := LightTheme()
	delete(broken.Semantic.Colors, ColorSemanticSuccess)
	if err := app.SetTheme(broken); err == nil {
		t.Fatal("missing recipe token accepted")
	}
	if !resolvedThemesEqual(old, app.theme) || !displayListsEqual(display, app.display) {
		t.Fatal("theme failure changed frame")
	}
	first := LightTheme()
	shadows, _ := first.Components[ComponentButtonGhost].States.Focus.Shadow.get()
	shadows[0].Spread = Metric(99)
	second := LightTheme()
	independent, _ := second.Components[ComponentButtonGhost].States.Focus.Shadow.get()
	if independent[0].Spread != Metric(2) {
		t.Fatal("theme factory shares mutable recipe shadows")
	}
	props := ButtonProps{Variant: ButtonLink, Size: ButtonSmall, Token: ComponentButton}.common()
	if props.Style.MinHeight != Px(30) || props.Style.Padding.Left != Metric(10) {
		t.Fatal("explicit token did not replace Link spacing")
	}
}
