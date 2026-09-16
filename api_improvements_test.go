package dxui

import (
	"runtime"
	"testing"

	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func TestViewModifiersAreImmutableAndDoNotAddIdentityLevel(t *testing.T) {
	originalShadow := []Shadow{{Blur: Metric(2), Color: LiteralColor(RGBA(1, 2, 3, 4))}}
	original := Box(BoxProps{Key: "original", Style: Style{Width: Px(10), Shadow: originalShadow}})
	replacementShadow := []Shadow{{Blur: Metric(4), Color: LiteralColor(RGBA(5, 6, 7, 8))}}
	styled := original.WithStyle(Style{Width: Px(20), Shadow: replacementShadow}).WithKey("styled")
	replacementShadow[0].Blur = Metric(99)

	originalProps, err := nodeProps(original)
	if err != nil {
		t.Fatal(err)
	}
	styledProps, err := nodeProps(styled)
	if err != nil {
		t.Fatal(err)
	}
	if original.node == styled.node || original.node.kind != styled.node.kind || len(styled.node.children) != len(original.node.children) {
		t.Fatal("modifier mutated the node or introduced a hierarchy level")
	}
	if originalProps.Key != "original" || originalProps.Style.Width != Px(10) || originalProps.Style.Shadow[0].Blur != Metric(2) {
		t.Fatalf("original changed: %+v", originalProps)
	}
	if styledProps.Key != "styled" || styledProps.Style.Width != Px(20) || styledProps.Style.Shadow[0].Blur != Metric(4) {
		t.Fatalf("styled copy = %+v", styledProps)
	}
	cleared := styled.WithStyle(Style{Shadow: []Shadow{}})
	clearedProps, _ := nodeProps(cleared)
	if clearedProps.Style.Shadow == nil || len(clearedProps.Style.Shadow) != 0 {
		t.Fatal("explicit empty shadow was not preserved")
	}
	forcedShadows := []Shadow{{Spread: Metric(3)}}
	forced := original.WithStyle(Style{Force: StylePatch{Shadow: Some(forcedShadows)}})
	forcedShadows[0].Spread = Metric(99)
	forcedProps, _ := nodeProps(forced)
	gotForced, set := forcedProps.Style.Force.Shadow.get()
	if !set || len(gotForced) != 1 || gotForced[0].Spread != Metric(3) {
		t.Fatalf("forced shadow copy = %v, set=%t", gotForced, set)
	}
}

func TestBaseStateAndForcedStylePrecedence(t *testing.T) {
	for _, source := range []Theme{LightTheme(), DarkTheme()} {
		resolved, err := prepareTheme(source)
		if err != nil {
			t.Fatal(err)
		}
		instance := &tree.Node{}
		instance.State.Hovered = true
		local := LiteralColor(RGBA(10, 20, 30, 255))
		props := viewProps{Style: Style{Background: local, Border: Border{Width: Metric(1), Color: local}}}
		visual, err := computeVisual(props, instance, viewButton, resolved)
		if err != nil {
			t.Fatal(err)
		}
		hover, _ := resolved.color(TokenColor(ColorSemanticAccentHover))
		if visual.background != hover {
			t.Fatalf("hover did not override local base: got %v want %v", visual.background, hover)
		}
		instance.State.Hovered, instance.State.FocusVisible = false, true
		visual, err = computeVisual(props, instance, viewButton, resolved)
		if err != nil {
			t.Fatal(err)
		}
		if visual.border.width != 1 || len(visual.shadows) == 0 {
			t.Fatalf("focus did not coexist with local border: border=%+v shadows=%v", visual.border, visual.shadows)
		}
		forced := RGBA(90, 80, 70, 255)
		props.Style.Force.Background = Some(LiteralColor(forced))
		props.Style.Force.Shadow = Some([]Shadow{})
		props.Style.Force.Opacity = Some(float32(0))
		visual, err = computeVisual(props, instance, viewButton, resolved)
		if err != nil || visual.background != forced || visual.opacity != 0 || len(visual.shadows) != 0 {
			t.Fatalf("forced override = %v, %v", visual.background, err)
		}
	}
}

func TestBuiltInButtonVariantsResolveAllStatesInLightAndDark(t *testing.T) {
	variants := []ComponentToken{ComponentButton, ComponentButtonSecondary, ComponentButtonDanger, ComponentButtonGhost}
	states := []struct {
		name  string
		apply func(*tree.Node)
	}{
		{"normal", func(*tree.Node) {}},
		{"hover", func(node *tree.Node) { node.State.Hovered = true }},
		{"pressed", func(node *tree.Node) { node.State.Pressed = true }},
		{"disabled", func(node *tree.Node) { node.Properties.Semantics.Disabled = true }},
		{"keyboard-focus", func(node *tree.Node) { node.State.FocusVisible = true }},
	}
	for _, source := range []Theme{LightTheme(), DarkTheme()} {
		resolved, err := prepareTheme(source)
		if err != nil {
			t.Fatal(err)
		}
		for _, variant := range variants {
			for _, state := range states {
				t.Run(string(variant)+"/"+state.name, func(t *testing.T) {
					node := &tree.Node{}
					state.apply(node)
					visual, err := computeVisual(viewProps{Token: variant}, node, viewButton, resolved)
					if err != nil || !visual.backgroundSet {
						t.Fatalf("visual = %+v, err=%v", visual, err)
					}
					if state.name == "keyboard-focus" && len(visual.shadows) == 0 {
						t.Fatal("keyboard focus has no ring")
					}
					if (state.name == "pressed" || state.name == "disabled") && visual.opacity >= 1 {
						t.Fatalf("%s opacity = %v", state.name, visual.opacity)
					}
				})
			}
		}
	}
}

func TestWindowShortcutRepeatAndEditorPriority(t *testing.T) {
	presses := 0
	app := NewApp(AppOptions{Shortcuts: []Shortcut{{Key: Key1, OnPress: func() { presses++ }}}})
	app.root = func() View { return Label("shortcut") }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, err := app.handleInteraction(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.Key1}}); err != nil {
		t.Fatal(err)
	}
	if presses != 1 {
		t.Fatalf("shortcut presses = %d, want 1", presses)
	}
	if _, err := app.handleInteraction(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.Key1, Repeat: true}}); err != nil {
		t.Fatal(err)
	}
	if presses != 1 {
		t.Fatal("non-repeat shortcut accepted native repeat")
	}

	app = NewApp(AppOptions{Shortcuts: []Shortcut{{Key: Key1, OnPress: func() { presses++ }}}})
	app.root = func() View { return Input(InputProps{Value: "", OnChange: func(string) {}}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.interaction.Handle(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyTab}})
	applyInteractionState(app.retained.Root(), &app.interaction)
	if _, err := app.handleInteraction(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.Key1}}); err != nil {
		t.Fatal(err)
	}
	if presses != 1 {
		t.Fatal("window shortcut stole a key from the focused editor")
	}
}

func TestShortcutValidationRejectsUnknownAndDuplicateChords(t *testing.T) {
	if err := validateShortcuts([]Shortcut{{Key: ShortcutKey(255)}}); err == nil {
		t.Fatal("unknown shortcut key was accepted")
	}
	modifiers := ShortcutModifiers{Shift: true}
	if err := validateShortcuts([]Shortcut{
		{Key: KeyPlus, Modifiers: modifiers},
		{Key: KeyPlus, Modifiers: modifiers},
	}); err == nil {
		t.Fatal("duplicate shortcut chord was accepted")
	}
	if err := validateShortcuts([]Shortcut{
		{Key: KeyPlus},
		{Key: KeyPlus, Modifiers: modifiers},
	}); err != nil {
		t.Fatalf("distinct modifier chords rejected: %v", err)
	}
}

func TestShortcutPrimaryModifierSubstitutesForPlatformModifier(t *testing.T) {
	got := platform.Modifiers{Primary: true}
	if runtime.GOOS == "darwin" {
		got.Super = true
	} else {
		got.Control = true
	}
	if !shortcutModifiersMatch(ShortcutModifiers{Primary: true}, got) {
		t.Fatal("Primary required callers to repeat the platform modifier")
	}
	got.Shift = true
	if shortcutModifiersMatch(ShortcutModifiers{Primary: true}, got) {
		t.Fatal("an unrequested extra modifier was accepted")
	}
}
