package dxui

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/dxui-org/dxui/internal/icondata"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
)

func packedStroke(commands ...PathCommand) IconData {
	bytes := make([]byte, 16, 16+3+len(commands)*25)
	values := []float32{0, 0, 24, 24}
	for index, value := range values {
		binary.LittleEndian.PutUint32(bytes[index*4:], math.Float32bits(value))
	}
	bytes = append(bytes, byte(icondata.PaintStroke), byte(len(commands)), byte(len(commands)>>8))
	for _, command := range commands {
		bytes = append(bytes, byte(command.Verb))
		count := map[PathVerb]int{PathMove: 1, PathLine: 1, PathQuad: 2, PathCubic: 3, PathClose: 0}[command.Verb]
		for _, point := range command.Points[:count] {
			var value [8]byte
			binary.LittleEndian.PutUint32(value[0:4], math.Float32bits(point.X))
			binary.LittleEndian.PutUint32(value[4:8], math.Float32bits(point.Y))
			bytes = append(bytes, value[:]...)
		}
	}
	return icondata.Packed(string(bytes))
}

func testIconTheme(t *testing.T) resolvedTheme {
	t.Helper()
	theme, err := prepareTheme(Theme{})
	if err != nil {
		t.Fatal(err)
	}
	return theme
}

func TestStrokeIconRoundCapsJoinsCurvesAndDegenerates(t *testing.T) {
	data := packedStroke(
		PathCommand{Verb: PathMove, Points: [3]Point{{X: 4, Y: 12}}},
		PathCommand{Verb: PathLine, Points: [3]Point{{X: 12, Y: 4}}},
		PathCommand{Verb: PathQuad, Points: [3]Point{{X: 20, Y: 4}, {X: 20, Y: 12}}},
		PathCommand{Verb: PathCubic, Points: [3]Point{{X: 20, Y: 20}, {X: 12, Y: 20}, {X: 12, Y: 12}}},
		PathCommand{Verb: PathClose},
		PathCommand{Verb: PathMove, Points: [3]Point{{X: 6, Y: 6}}},
		PathCommand{Verb: PathLine, Points: [3]Point{{X: 6, Y: 6}}},
	)
	bitmap, _, err := rasterIcon(IconProps{Data: data, Size: 24}, layout.Rect{Width: 24, Height: 24}, testIconTheme(t), 1.5, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	if bitmap.Width != 36 || bitmap.Height != 36 || len(bitmap.Pixels) != 36*36 {
		t.Fatalf("bitmap = %dx%d/%d", bitmap.Width, bitmap.Height, len(bitmap.Pixels))
	}
	for _, point := range [][2]int{{6, 18}, {18, 6}, {30, 18}, {9, 9}} {
		if alpha := bitmap.Pixels[point[1]*bitmap.Width+point[0]]; alpha == 0 {
			t.Errorf("expected coverage at %v", point)
		}
	}
}

func TestStrokeWidthAndScaleAreMaskIdentityButColorIsNot(t *testing.T) {
	data := packedStroke(
		PathCommand{Verb: PathMove, Points: [3]Point{{X: 2, Y: 12}}},
		PathCommand{Verb: PathLine, Points: [3]Point{{X: 22, Y: 12}}},
	)
	theme := testIconTheme(t)
	one, _, err := rasterIcon(IconProps{Data: data, Size: 20, Color: LiteralColor(RGBA(1, 2, 3, 255))}, layout.Rect{Width: 20, Height: 20}, theme, 1.25, 1.25)
	if err != nil {
		t.Fatal(err)
	}
	colorChange, _, _ := rasterIcon(IconProps{Data: data, Size: 20, Color: LiteralColor(RGBA(9, 8, 7, 255))}, layout.Rect{Width: 20, Height: 20}, theme, 1.25, 1.25)
	wide, _, _ := rasterIcon(IconProps{Data: data, Size: 20, StrokeWidth: IconStrokeWidth(3)}, layout.Rect{Width: 20, Height: 20}, theme, 1.25, 1.25)
	otherScale, _, _ := rasterIcon(IconProps{Data: data, Size: 20}, layout.Rect{Width: 20, Height: 20}, theme, 1.5, 1.5)
	if one.Key != colorChange.Key {
		t.Fatal("color changed mask identity")
	}
	if one.Key == wide.Key || one.Key == otherScale.Key {
		t.Fatal("stroke width or DPI did not change mask identity")
	}
}

func TestIdenticalIconsReuseOneMaskWithinAndAcrossDisplays(t *testing.T) {
	data := packedStroke(
		PathCommand{Verb: PathMove, Points: [3]Point{{X: 2, Y: 12}}},
		PathCommand{Verb: PathLine, Points: [3]Point{{X: 22, Y: 12}}},
	)
	props := IconProps{Data: data, Size: 20}
	reuse := retainedTextMasks(nil)
	one, _, err := rasterIconReuse(props, layout.Rect{Width: 20, Height: 20}, testIconTheme(t), 1.25, 1.25, &reuse)
	if err != nil {
		t.Fatal(err)
	}
	two, _, err := rasterIconReuse(props, layout.Rect{Width: 20, Height: 20}, testIconTheme(t), 1.25, 1.25, &reuse)
	if err != nil {
		t.Fatal(err)
	}
	if &one.Pixels[0] != &two.Pixels[0] {
		t.Fatal("identical icons did not share the grayscale mask")
	}
	remaining := iconMaskOverhead + len(one.Pixels)
	if err := accountIconMask(one, &remaining, &reuse); err != nil {
		t.Fatal(err)
	}
	if err := accountIconMask(two, &remaining, &reuse); err != nil || remaining != 0 {
		t.Fatalf("shared mask was charged twice: remaining=%d err=%v", remaining, err)
	}
	previous := paint.DisplayList{{Kind: paint.CommandDrawIcon, Text: one}}
	retained := retainedTextMasks(previous)
	three, _, err := rasterIconReuse(props, layout.Rect{Width: 20, Height: 20}, testIconTheme(t), 1.25, 1.25, &retained)
	if err != nil || &one.Pixels[0] != &three.Pixels[0] {
		t.Fatalf("retained display reuse failed: %v", err)
	}
}

func TestLegacyFilledIconBehavior(t *testing.T) {
	data := IconData{ViewBox: Rect{Width: 10, Height: 10}, Commands: []PathCommand{
		{Verb: PathMove, Points: [3]Point{{X: 1, Y: 1}}},
		{Verb: PathLine, Points: [3]Point{{X: 9, Y: 1}}},
		{Verb: PathLine, Points: [3]Point{{X: 9, Y: 9}}},
		{Verb: PathClose},
	}}
	one, _, err := rasterIcon(IconProps{Data: data, Size: 10}, layout.Rect{Width: 10, Height: 10}, testIconTheme(t), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	two, _, err := rasterIcon(IconProps{Data: data, Size: 10, StrokeWidth: 8}, layout.Rect{Width: 10, Height: 10}, testIconTheme(t), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if one.Key != two.Key {
		t.Fatal("legacy fill was affected by stroke width")
	}
}

func TestIconMaskSourceBudgetIsExactAndBounded(t *testing.T) {
	bitmap := &paint.TextBitmap{Width: 4, Height: 3, Pixels: make([]byte, 12)}
	remaining := iconMaskOverhead + len(bitmap.Pixels)
	if err := accountIconMask(bitmap, &remaining, nil); err != nil || remaining != 0 {
		t.Fatalf("exact budget = %d/%v", remaining, err)
	}
	if err := accountIconMask(bitmap, &remaining, nil); err == nil {
		t.Fatal("icon mask exceeded retained source budget")
	}
}

func BenchmarkImmutableIconViewConstruction(b *testing.B) {
	data := lucideReferenceIcons["search"]
	props := IconProps{Data: data, Size: 20}
	b.ReportAllocs()
	for b.Loop() {
		_ = Icon(props)
	}
}

func BenchmarkLucideRaster24AtFractionalDPI(b *testing.B) {
	theme, err := prepareTheme(Theme{})
	if err != nil {
		b.Fatal(err)
	}
	props := IconProps{Data: lucideReferenceIcons["brain-circuit"], Size: 24}
	content := layout.Rect{Width: 24, Height: 24}
	b.ReportAllocs()
	b.SetBytes(30 * 30)
	for b.Loop() {
		if _, _, err := rasterIcon(props, content, theme, 1.25, 1.25); err != nil {
			b.Fatal(err)
		}
	}
}
