package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
)

func TestCheckboxCheckedDisplayUsesFilledBackgroundAndCenteredCheck(t *testing.T) {
	theme, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	box := paint.Rect{X: 10, Y: 20, Width: 18, Height: 18}
	var list paint.DisplayList
	remaining := 1 << 20
	if err := appendCheckboxDisplay(&list, box, true, theme, 7, 1.25, 1.25, &remaining, nil); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("checked checkbox commands = %#v, want background and check", list)
	}
	accent, _ := theme.color(TokenColor(ColorSemanticAccent))
	surface, _ := theme.color(TokenColor(ColorSemanticSurfaceHi))
	if background := list[0]; background.Kind != paint.CommandFillStrokeRoundedRect || background.Rect != box || background.Color != paintColor(accent) {
		t.Fatalf("checked checkbox background = %#v", background)
	}
	mark := list[1]
	if mark.Kind != paint.CommandDrawIcon || mark.Rect != box || mark.Color != paintColor(surface) || mark.Text == nil {
		t.Fatalf("checked checkbox mark = %#v", mark)
	}
	if mark.Text.Width != 23 || mark.Text.Height != 23 {
		t.Fatalf("check mask = %dx%d, want scale-aware 23x23", mark.Text.Width, mark.Text.Height)
	}
	if mark.Text.Pixels[0] != 0 || mark.Text.Pixels[len(mark.Text.Pixels)-1] != 0 {
		t.Fatal("check mark reaches a mask corner instead of remaining centered")
	}
	covered := 0
	for _, alpha := range mark.Text.Pixels {
		if alpha != 0 {
			covered++
		}
	}
	if covered == 0 {
		t.Fatal("check mark mask has no coverage")
	}
}

func TestCheckboxUncheckedDisplayHasNoCheckMark(t *testing.T) {
	theme, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	var list paint.DisplayList
	remaining := 1 << 20
	if err := appendCheckboxDisplay(&list, paint.Rect{Width: 18, Height: 18}, false, theme, 7, 1, 1, &remaining, nil); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Kind != paint.CommandFillStrokeRoundedRect {
		t.Fatalf("unchecked checkbox commands = %#v", list)
	}
}

func TestCheckboxIndicatorFollowsScrollTranslation(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 36})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarHidden, InitialOffset: Some(Point{Y: 40})},
			Box(BoxProps{},
				Text(TextProps{Style: Style{Height: Px(40), Shrink: Some(float32(0))}}),
				Checkbox(CheckboxProps{Style: Style{Height: Px(36), Shrink: Some(float32(0))}, Checked: true}, Text(TextProps{Value: "visible"})),
			),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	checkboxID := app.retained.Root().Children[0].Children[1].ID
	var background, mark *paint.Command
	list := app.currentDisplay()
	for index := range list {
		command := &list[index]
		if command.NodeID != checkboxID {
			continue
		}
		switch command.Kind {
		case paint.CommandFillStrokeRoundedRect:
			background = command
		case paint.CommandDrawIcon:
			mark = command
		}
	}
	if background == nil || mark == nil {
		t.Fatalf("scrolled checkbox commands missing: background=%v mark=%v", background, mark)
	}
	if background.Rect.Y != 9 || mark.Rect.Y != 9 || background.Rect.Y+background.Rect.Height > 36 {
		t.Fatalf("scrolled checkbox indicator background=%+v mark=%+v, want visible Y=9", background.Rect, mark.Rect)
	}
}
