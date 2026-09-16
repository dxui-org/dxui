package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/tree"
)

func TestDashedButtonBorderBuildsBoundedSegments(t *testing.T) {
	borderColor := RGBA(12, 88, 190, 255)
	app := NewApp(AppOptions{Width: 180, Height: 60})
	app.root = func() View {
		return Button(ButtonProps{Style: Style{
			Width: Px(160), Height: Px(44),
			Border: Border{Width: Metric(2), Color: LiteralColor(borderColor), Pattern: BorderDashed},
		}}, Text(TextProps{Value: "Dashed"}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	segments := 0
	corners := [4]bool{}
	for _, command := range app.display {
		if command.Kind == paint.CommandStrokeRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect {
			t.Fatalf("dashed border emitted solid stroke command: %+v", command)
		}
		if command.Kind == paint.CommandFillRoundedRect && command.Color == (paint.Color{R: 12, G: 88, B: 190, A: 255}) {
			segments++
			centerX := command.Rect.X + command.Rect.Width/2
			centerY := command.Rect.Y + command.Rect.Height/2
			corners[0] = corners[0] || centerX < 8 && centerY < 8
			corners[1] = corners[1] || centerX > 152 && centerY < 8
			corners[2] = corners[2] || centerX > 152 && centerY > 36
			corners[3] = corners[3] || centerX < 8 && centerY > 36
		}
	}
	if segments < 8 || segments > 512 {
		t.Fatalf("dashed segment count = %d, want bounded visible segments", segments)
	}
	for index, covered := range corners {
		if !covered {
			t.Errorf("rounded dashed border corner %d has no painted segment", index)
		}
	}
}

func TestDefaultButtonHasNoShadowOrBaseBorder(t *testing.T) {
	for name, theme := range map[string]Theme{"light": LightTheme(), "dark": DarkTheme()} {
		t.Run(name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 160, Height: 60, Theme: theme})
			app.root = func() View {
				return Button(ButtonProps{}, Text(TextProps{Value: "Default"}))
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			buttonID := app.retained.Root().ID
			for _, command := range app.display {
				if command.NodeID != buttonID {
					continue
				}
				if command.Kind == paint.CommandDrawShadow || command.Kind == paint.CommandStrokeRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect {
					t.Fatalf("default Button emitted shadow/base-border command: %+v", command)
				}
			}
		})
	}
}

func TestBorderPatternChangeIsPaintOnly(t *testing.T) {
	pattern := BorderSolid
	app := NewApp(AppOptions{Width: 160, Height: 60})
	app.root = func() View {
		return Button(ButtonProps{Style: Style{
			Width: Px(140), Height: Px(40), Border: Border{Width: Metric(1), Color: LiteralColor(RGBA(1, 2, 3, 255)), Pattern: pattern},
		}}, Text(TextProps{Value: "Pattern"}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	pattern = BorderDashed
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) {
		t.Fatalf("border pattern dirty = %v, want paint without layout", changes.Dirty)
	}
}

func TestInvalidBorderPatternRejectsViewAndTheme(t *testing.T) {
	invalid := BorderPattern(2)
	view := Button(ButtonProps{Style: Style{Border: Border{Pattern: invalid}}}, Text(TextProps{}))
	if _, err := describeView(view); err == nil {
		t.Fatal("invalid local border pattern was accepted")
	}
	theme := LightTheme()
	theme.Components[ComponentButton] = ComponentTheme{Base: StylePatch{Border: Some(Border{Pattern: invalid})}}
	if _, err := prepareTheme(theme); err == nil {
		t.Fatal("invalid themed border pattern was accepted")
	}
}
