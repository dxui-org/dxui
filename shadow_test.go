package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/tree"
)

func TestBuiltInComponentsHaveNoImplicitElevationShadow(t *testing.T) {
	for _, themeSource := range []Theme{LightTheme(), DarkTheme()} {
		theme, err := prepareTheme(themeSource)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range []struct {
			name string
			kind viewKind
		}{
			{"panel", viewBox},
			{"button", viewButton},
			{"input", viewInput},
			{"input-group", viewInputGroup},
			{"textarea", viewTextarea},
			{"select", viewSelect},
			{"tabs", viewTabs},
			{"menu", viewMenu},
		} {
			t.Run(test.name, func(t *testing.T) {
				props := viewProps{}
				if test.name == "panel" {
					props.Token = ComponentPanel
				}
				instance := &tree.Node{State: tree.State{Hovered: true, Pressed: true}}
				visual, err := computeVisual(props, instance, test.kind, theme)
				if err != nil {
					t.Fatal(err)
				}
				if len(visual.shadows) != 0 {
					t.Fatalf("resolved %d implicit shadows", len(visual.shadows))
				}
			})
		}
	}
}

func TestExplicitShadowCanBeSetAndRemovedWithoutLayout(t *testing.T) {
	shadow := []Shadow{{
		OffsetY: Metric(3), Blur: Metric(8), Spread: Metric(1),
		Color: LiteralColor(RGBA(0, 0, 0, 80)),
	}}
	app := NewApp(AppOptions{Width: 100, Height: 60})
	app.root = func() View {
		return Box(BoxProps{Style: Style{
			Width: Px(40), Height: Px(20), Background: LiteralColor(RGBA(255, 255, 255, 255)),
			Shadow: shadow,
		}})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := countShadowCommands(app.currentDisplay()); got != 1 {
		t.Fatalf("explicit shadow commands = %d, want 1", got)
	}
	before := app.Diagnostics()
	shadow = []Shadow{}
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	after := app.Diagnostics()
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) {
		t.Fatalf("removing shadow dirty flags = %v", changes.Dirty)
	}
	if after.LayoutCount != before.LayoutCount || after.PaintCount != before.PaintCount+1 {
		t.Fatalf("layout/paint = %d/%d -> %d/%d", before.LayoutCount, before.PaintCount, after.LayoutCount, after.PaintCount)
	}
	if got := countShadowCommands(app.currentDisplay()); got != 0 {
		t.Fatalf("shadow commands after removal = %d", got)
	}
}

func TestExplicitEmptyShadowOverridesThemeAndSurvivesViewCopy(t *testing.T) {
	theme := LightTheme()
	theme.Components["raised"] = ComponentTheme{Base: StylePatch{Shadow: Some([]Shadow{{
		Blur: Metric(4), Color: LiteralColor(RGBA(0, 0, 0, 80)),
	}})}}
	app := NewApp(AppOptions{Width: 40, Height: 20, Theme: theme})
	app.root = func() View {
		return Box(BoxProps{Token: "raised", Style: Style{Shadow: []Shadow{}}})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if app.view.node.box.common().Style.Shadow == nil {
		t.Fatal("explicit empty Shadow became nil while copying the View")
	}
	if got := countShadowCommands(app.currentDisplay()); got != 0 {
		t.Fatalf("explicit empty Shadow emitted %d commands", got)
	}
}

func TestShadowOverflowIsPaintedButDoesNotExpandHitBounds(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 60})
	app.root = func() View {
		return Box(BoxProps{Style: Style{Overflow: OverflowClip}},
			Button(ButtonProps{Style: Style{
				Position: PositionAbsolute, Insets: Insets{Left: Px(105)}, Width: Px(10), Height: Px(10),
				Shadow: []Shadow{{Blur: Metric(20), Color: LiteralColor(RGBA(0, 0, 0, 100))}},
			}}, Text(TextProps{Value: "x"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := countShadowCommands(app.currentDisplay()); got != 1 {
		t.Fatalf("shadow extending into ancestor clip was culled: commands=%d", got)
	}
	if _, ok := app.currentDisplay().HitTestAt(99, 5); ok {
		t.Fatal("shadow overflow expanded the button hit area")
	}
	command := firstShadowCommand(app.currentDisplay())
	if got := visualBounds(command.Rect, []paint.Shadow{command.Shadow}); got.X > 99 || got.X+got.Width <= 99 {
		t.Fatalf("shadow bounds %#v do not cover the clipped visible point", got)
	}
}

func countShadowCommands(list paint.DisplayList) int {
	count := 0
	for _, command := range list {
		if command.Kind == paint.CommandDrawShadow {
			count++
		}
	}
	return count
}

func firstShadowCommand(list paint.DisplayList) paint.Command {
	for _, command := range list {
		if command.Kind == paint.CommandDrawShadow {
			return command
		}
	}
	return paint.Command{}
}
