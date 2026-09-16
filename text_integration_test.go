package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
)

func TestTextParticipatesInFlexIntrinsicSizingAndPaint(t *testing.T) {
	app := NewApp(AppOptions{Width: 320, Height: 120})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Align: AlignStart},
			Text(TextProps{Value: "short"}),
			Text(TextProps{Value: "a much wider label"}),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if app.geometry == nil || len(app.geometry.Children) != 2 {
		t.Fatalf("geometry = %+v", app.geometry)
	}
	first, second := app.geometry.Children[0].Rect, app.geometry.Children[1].Rect
	if first.Width <= 0 || second.Width <= first.Width || first.Height <= 0 || second.Height != first.Height {
		t.Fatalf("intrinsic text geometry = %+v / %+v", first, second)
	}
	commands := textCommands(app.display)
	if len(commands) != 2 || commands[0].Text == nil || commands[1].Text == nil || commands[0].Text.Key == commands[1].Text.Key {
		t.Fatalf("text display commands = %+v", commands)
	}
}

func TestTextWrapLineHeightMeasureMatchesPaint(t *testing.T) {
	app := NewApp(AppOptions{Width: 240, Height: 200})
	app.root = func() View {
		return Box(BoxProps{Align: AlignStart}, Text(TextProps{
			Value: "one two three four", Wrap: TextWrapWords,
			Style: Style{
				Width: Px(70), Shrink: Some(float32(0)),
				Text: TextStyle{Size: 14, LineHeight: 20},
			},
		}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	child := app.geometry.Children[0]
	commands := textCommands(app.display)
	if len(commands) < 2 || child.Content.Height != float32(len(commands))*20 {
		t.Fatalf("measure/paint mismatch: content=%+v text commands=%d", child.Content, len(commands))
	}
}

func TestTextObeysClipOpacityZIndexAndVisibility(t *testing.T) {
	app := NewApp(AppOptions{Width: 180, Height: 80})
	app.root = func() View {
		return Box(BoxProps{Style: Style{Overflow: OverflowClip}},
			Text(TextProps{Value: "behind", Style: Style{ZIndex: -1}}),
			Text(TextProps{Value: "front", Style: Style{
				ZIndex: 2, Opacity: Some(float32(.5)),
				Text: TextStyle{Color: LiteralColor(RGBA(20, 40, 60, 200))},
			}}),
			Text(TextProps{Value: "hidden", Style: Style{Visibility: Hidden}}),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	commands := textCommands(app.display)
	if len(commands) != 2 || commands[0].NodeID == commands[1].NodeID {
		t.Fatalf("visibility/z text commands = %+v", commands)
	}
	recorder := &paint.RecordingPainter{}
	if err := paint.Replay(app.display, recorder); err != nil {
		t.Fatal(err)
	}
	var sawClip, sawHalfAlpha bool
	for _, record := range recorder.Records {
		if record.Kind == paint.CommandPushClip {
			sawClip = true
		}
		if record.Kind == paint.CommandDrawText && record.Color.A == 100 {
			sawHalfAlpha = true
		}
	}
	if !sawClip || !sawHalfAlpha {
		t.Fatalf("replayed text clip/opacity records = %+v", recorder.Records)
	}
}

func TestTextConstructorNormalizesInvalidUTF8AndCopiesFamilies(t *testing.T) {
	families := []FontFamily{"first", "second"}
	view := Text(TextProps{Value: string([]byte{'x', 0xff}), Style: Style{Text: TextStyle{Families: families}}})
	families[0] = "changed"
	if view.node.text.Value != "x\uFFFD" || view.node.text.common().Style.Text.Families[0] != "first" {
		t.Fatalf("immutable normalized Text = %+v", view.node.text)
	}
}

func TestTextDisplayRejectsRetainedMaskOverBudget(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40, Caches: CacheBudgets{TextSourceBytes: 1}})
	app.root = func() View { return Text(TextProps{Value: "bounded"}) }
	if err := app.buildRoot(); err == nil {
		t.Fatal("text display exceeding retained source budget was accepted")
	}
}

func textCommands(list paint.DisplayList) []paint.Command {
	var result []paint.Command
	for _, command := range list {
		if command.Kind == paint.CommandDrawText {
			result = append(result, command)
		}
	}
	return result
}
