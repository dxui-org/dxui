package dxui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func fixedToggle(checked, disabled bool, change func(bool)) View {
	return ToggleSwitch(ToggleSwitchProps{Style: Style{
		Width: Px(40), Height: Px(24), Position: PositionAbsolute,
	}, Checked: checked, Disabled: disabled, OnChange: change})
}

func TestToggleSwitchIsControlledAcrossMouseAndSpace(t *testing.T) {
	app := NewApp(AppOptions{Width: 80, Height: 40})
	checked := false
	changes := 0
	app.root = func() View {
		return fixedToggle(checked, false, func(change bool) {
			changes++
			checked = change
		})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 5, 5)); err != nil {
		t.Fatal(err)
	}
	if !checked || changes != 1 || !app.retained.Root().State.Checked {
		t.Fatalf("mouse controlled state = %v/%d/%v", checked, changes, app.retained.Root().State.Checked)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	if _, _, err := app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false)); err != nil {
		t.Fatal(err)
	}
	if checked || changes != 2 || app.retained.Root().State.Checked {
		t.Fatalf("space controlled state = %v/%d/%v", checked, changes, app.retained.Root().State.Checked)
	}
}

func TestToggleRejectedControlledWriteDoesNotDiverge(t *testing.T) {
	app := NewApp(AppOptions{Width: 80, Height: 40})
	changes := 0
	app.root = func() View {
		return fixedToggle(false, false, func(change bool) { changes++ })
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	app.handleEvent(appPointer(platform.EventMouseUp, 5, 5))
	if changes != 1 || app.retained.Root().State.Checked {
		t.Fatalf("rejected controlled write diverged: changes=%d checked=%v", changes, app.retained.Root().State.Checked)
	}
}

func TestControlledCheckedChangeIsPaintOnly(t *testing.T) {
	app := NewApp(AppOptions{Width: 80, Height: 40})
	checked := false
	app.root = func() View { return fixedToggle(checked, false, nil) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	checked = true
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	after := app.Diagnostics()
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) || after.LayoutCount != before.LayoutCount || after.PaintCount != before.PaintCount+1 {
		t.Fatalf("checked dirty/diagnostics = %v / %+v -> %+v", changes.Dirty, before, after)
	}
}

func TestToggleDisabledAndPointerCancelDoNotCallback(t *testing.T) {
	for _, test := range []struct {
		name     string
		disabled bool
		moveOut  bool
	}{
		{name: "disabled", disabled: true},
		{name: "release outside", moveOut: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 80, Height: 40})
			changes := 0
			app.root = func() View { return fixedToggle(false, test.disabled, func(bool) { changes++ }) }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
			if test.moveOut {
				app.handleEvent(appPointer(platform.EventMouseMove, 90, 50))
			}
			app.handleEvent(appPointer(platform.EventMouseUp, 90, 50))
			if changes != 0 {
				t.Fatalf("disabled/cancel changes = %d", changes)
			}
		})
	}
}

func TestCheckboxCallbackMayRemoveItself(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	visible := true
	changes := 0
	app.root = func() View {
		if !visible {
			return Box(BoxProps{})
		}
		return Checkbox(CheckboxProps{Style: Style{Width: Px(80), Height: Px(24)}, OnChange: func(bool) {
			changes++
			visible = false
		}}, Text(TextProps{Value: "remove"}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 5, 5)); err != nil {
		t.Fatal(err)
	}
	if changes != 1 || len(app.retained.Root().Children) != 0 {
		t.Fatalf("self-removal result = changes %d children %d", changes, len(app.retained.Root().Children))
	}
}

func TestCheckboxIndicatorContributesIntrinsicFlexSize(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Align: AlignStart}, Checkbox(CheckboxProps{}, Text(TextProps{})))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	checkbox := app.geometry.Children[0]
	if checkbox.Rect.Height != 36 || len(checkbox.Children) != 2 || checkbox.Children[0].Rect.Width != 18 || checkbox.Children[0].Rect.Y != 9 {
		t.Fatalf("checkbox intrinsic geometry = %+v children=%+v", checkbox.Rect, checkbox.Children)
	}
}

func testIcon(colorValue ColorValue) View {
	return Icon(IconProps{Style: Style{Width: Px(20), Height: Px(20)},
		Data: IconData{ViewBox: Rect{Width: 10, Height: 10}, Commands: []PathCommand{
			{Verb: PathMove, Points: [3]Point{{X: 1, Y: 1}}},
			{Verb: PathLine, Points: [3]Point{{X: 9, Y: 1}}},
			{Verb: PathLine, Points: [3]Point{{X: 5, Y: 9}}},
			{Verb: PathClose},
		}}, Size: 20, Color: colorValue})
}

func TestIconColorIsPaintOnlyAndScaleKeepsLogicalSize(t *testing.T) {
	app := NewApp(AppOptions{Width: 40, Height: 40})
	iconColor := LiteralColor(RGBA(10, 20, 30, 255))
	app.root = func() View { return testIcon(iconColor) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	beforeLayout := app.Diagnostics().LayoutCount
	iconColor = LiteralColor(RGBA(50, 60, 70, 255))
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.Diagnostics().LayoutCount != beforeLayout {
		t.Fatal("icon tint change caused layout")
	}
	one, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, app.text, app.images, 1, 1, app.textSourceBudget(), nil)
	if err != nil {
		t.Fatal(err)
	}
	two, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, app.text, app.images, 2, 2, app.textSourceBudget(), nil)
	if err != nil {
		t.Fatal(err)
	}
	find := func(list paint.DisplayList) paint.Command {
		for _, command := range list {
			if command.Kind == paint.CommandDrawIcon {
				return command
			}
		}
		return paint.Command{}
	}
	first, second := find(one), find(two)
	if first.Rect.Width != 20 || second.Rect.Width != 20 || first.Text.Width != 20 || second.Text.Width != 40 {
		t.Fatalf("icon logical/physical sizes = %+v/%+v", first, second)
	}
}

func TestImageCallbacksPlaceholderAndIntrinsicSize(t *testing.T) {
	good := image.NewNRGBA(image.Rect(0, 0, 7, 5))
	good.SetNRGBA(0, 0, color.NRGBA{R: 1, A: 255})
	loaded := Size{}
	source := ImageFromGo(good)
	app := NewApp(AppOptions{Width: 50, Height: 40})
	app.root = func() View {
		return Image(ImageProps{Source: source, OnLoad: func(size Size) { loaded = size }})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if loaded != (Size{Width: 7, Height: 5}) {
		t.Fatalf("image load size = %+v", loaded)
	}

	var reported error
	badSource := ImageBytes([]byte("bad"))
	bad := NewApp(AppOptions{Width: 20, Height: 20})
	bad.root = func() View {
		return Image(ImageProps{Source: badSource, OnError: func(err error) { reported = err }})
	}
	if err := bad.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if reported == nil || !strings.Contains(reported.Error(), "decode header") {
		t.Fatalf("image error callback = %v", reported)
	}
	found := false
	for _, command := range bad.display {
		if command.Kind == paint.CommandDrawImage {
			found = command.Image != nil && command.Image.Width == 2
		}
	}
	if !found {
		t.Fatal("decode failure did not paint the documented placeholder")
	}
}

func TestImageBytesAndIconCommandsAreCopied(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	source := ImageBytes(data)
	data[0] = 0
	app := NewApp(AppOptions{})
	if _, err := decodeImage(app.images, ImageProps{Source: source}); err != nil {
		t.Fatalf("ImageBytes retained mutable caller bytes: %v", err)
	}
	commands := []PathCommand{{Verb: PathMove, Points: [3]Point{{X: 1, Y: 1}}}}
	icon := Icon(IconProps{Data: IconData{ViewBox: Rect{Width: 2, Height: 2}, Commands: commands}, Size: 2})
	commands[0].Points[0] = Point{X: 99, Y: 99}
	if got := icon.node.icon.Data.Commands[0].Points[0]; got != (Point{X: 1, Y: 1}) {
		t.Fatalf("Icon retained mutable command slice: %+v", got)
	}
}

func TestImageFitGeometrySubset(t *testing.T) {
	content := layout.Rect{Width: 100, Height: 100}
	for _, test := range []struct {
		fit  ImageFit
		want paint.Rect
	}{
		{ImageContain, paint.Rect{Y: 25, Width: 100, Height: 50}},
		{ImageCover, paint.Rect{X: -50, Width: 200, Height: 100}},
		{ImageFill, paint.Rect{Width: 100, Height: 100}},
		{ImageNone, paint.Rect{X: -50, Width: 200, Height: 100}},
	} {
		if got := fittedImageRect(content, 200, 100, test.fit, Point{X: .5, Y: .5}); got != test.want {
			t.Fatalf("fit %d = %+v, want %+v", test.fit, got, test.want)
		}
	}
}
