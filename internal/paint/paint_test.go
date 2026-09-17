package paint

import (
	"fmt"
	"math"
	"testing"
)

func TestSortedUsesZIndexThenSourceOrder(t *testing.T) {
	items := DisplayList{
		{ZIndex: 2, SourceIndex: 0},
		{ZIndex: 1, SourceIndex: 2},
		{ZIndex: 1, SourceIndex: 1},
	}.Sorted()
	if items[0].SourceIndex != 1 || items[1].SourceIndex != 2 || items[2].ZIndex != 2 {
		t.Fatalf("unexpected order: %#v", items)
	}
}

func TestUnionIncludesOldAndNewBounds(t *testing.T) {
	got := Union(Rect{X: 2, Y: 3, Width: 4, Height: 5}, Rect{X: 0, Y: 5, Width: 3, Height: 6})
	want := (Rect{X: 0, Y: 3, Width: 6, Height: 8})
	if got != want {
		t.Fatalf("Union = %#v, want %#v", got, want)
	}
}

func TestSnapRectUsesSharedRoundedEdgesAtMultipleScales(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		left, err := SnapRect(Rect{X: 0, Width: 33.3, Height: 10}, scale)
		if err != nil {
			t.Fatal(err)
		}
		right, err := SnapRect(Rect{X: 33.3, Width: 66.7, Height: 10}, scale)
		if err != nil {
			t.Fatal(err)
		}
		whole, err := SnapRect(Rect{Width: 100, Height: 10}, scale)
		if err != nil {
			t.Fatal(err)
		}
		if left.X+left.Width != right.X || left.Width+right.Width != whole.Width {
			t.Fatalf("scale %v snapped left/right/whole = %#v/%#v/%#v", scale, left, right, whole)
		}
	}
}

func TestSnapRectRejectsInvalidInputs(t *testing.T) {
	for _, test := range []struct {
		rect  Rect
		scale float32
	}{
		{Rect{Width: -1}, 1},
		{Rect{Width: 1}, 0},
		{Rect{Width: 1}, float32(math.NaN())},
	} {
		if _, err := SnapRect(test.rect, test.scale); err == nil {
			t.Fatal("invalid snap input accepted")
		}
	}
}

func TestHitTestOrderSkipsHiddenAndWorldTransparentItems(t *testing.T) {
	items := DisplayList{
		{ZIndex: 1, SourceIndex: 0, Visibility: true, Opacity: 1, Interactive: true},
		{ZIndex: 2, SourceIndex: 1, Visibility: true, Opacity: 0, Interactive: true},
		{ZIndex: 3, SourceIndex: 2, Visibility: false, Opacity: 1, Interactive: true},
		{ZIndex: 1, SourceIndex: 3, Visibility: true, Opacity: 1, Interactive: true},
	}
	ordered := items.HitTestOrder()
	if len(ordered) != 2 || ordered[0].SourceIndex != 3 || ordered[1].SourceIndex != 0 {
		t.Fatalf("hit-test order = %#v", ordered)
	}
}

func TestHitTestAtHonorsClipBeforeZOrder(t *testing.T) {
	items := DisplayList{
		{Bounds: Rect{Width: 100, Height: 100}, ZIndex: 0, Visibility: true, Opacity: 1, Interactive: true},
		{Bounds: Rect{Width: 100, Height: 100}, Clip: Rect{Width: 20, Height: 20}, ClipSet: true, ZIndex: 4, SourceIndex: 1, Visibility: true, Opacity: 1, Interactive: true},
	}
	if item, ok := items.HitTestAt(10, 10); !ok || item.SourceIndex != 1 {
		t.Fatalf("inside clip hit = %#v/%v", item, ok)
	}
	if item, ok := items.HitTestAt(50, 50); !ok || item.SourceIndex != 0 {
		t.Fatalf("outside top clip hit = %#v/%v", item, ok)
	}
}

func TestReplayComposesNestedOpacityClipAndOrder(t *testing.T) {
	list := DisplayList{
		{Kind: CommandPushOpacity, Opacity: .5},
		{Kind: CommandPushClip, Rect: Rect{Width: 80, Height: 80}},
		{Kind: CommandPushClip, Rect: Rect{X: 20, Y: 10, Width: 80, Height: 30}},
		{Kind: CommandFillRoundedRect, Rect: Rect{Width: 100, Height: 100}, Color: Color{R: 9, A: 200}},
		{Kind: CommandPopClip},
		{Kind: CommandPopClip},
		{Kind: CommandPopOpacity},
	}
	recorder := &RecordingPainter{}
	if err := Replay(list, recorder); err != nil {
		t.Fatal(err)
	}
	if len(recorder.Records) != 5 {
		t.Fatalf("records = %#v", recorder.Records)
	}
	if got := recorder.Records[1].Rect; got != (Rect{X: 20, Y: 10, Width: 60, Height: 30}) {
		t.Fatalf("nested clip = %#v", got)
	}
	if got := recorder.Records[2].Color.A; got != 100 {
		t.Fatalf("composed alpha = %d, want 100", got)
	}
	if recorder.Records[0].Kind != CommandPushClip || recorder.Records[4].Kind != CommandPopClip {
		t.Fatalf("clip restore order = %#v", recorder.Records)
	}
}

func TestReplaySuppressesDrawsInsideEmptyClipAndRestoresParent(t *testing.T) {
	mask := &TextBitmap{Key: "mask", Width: 1, Height: 1, Pixels: []byte{255}}
	image := &ImageBitmap{Key: "image", Width: 1, Height: 1, Pixels: []byte{1, 2, 3, 255}}
	list := DisplayList{
		{Kind: CommandPushClip, Rect: Rect{Width: 10, Height: 10}},
		{Kind: CommandFillRoundedRect, Rect: Rect{Width: 10, Height: 10}, Color: Color{A: 255}},
		{Kind: CommandPushClip, Rect: Rect{X: 20, Y: 20, Width: 10, Height: 10}},
		{Kind: CommandDrawShadow, Rect: Rect{Width: 1, Height: 1}, Shadow: Shadow{Color: Color{A: 255}}},
		{Kind: CommandFillRoundedRect, Rect: Rect{Width: 1, Height: 1}, Color: Color{A: 255}},
		{Kind: CommandStrokeRoundedRect, Rect: Rect{Width: 1, Height: 1}, Width: 1, Color: Color{A: 255}},
		{Kind: CommandBorder, Rect: Rect{Width: 1, Height: 1}, Width: 1, Color: Color{A: 255}},
		{Kind: CommandFillStrokeRoundedRect, Rect: Rect{Width: 1, Height: 1}, Width: 1, Color: Color{A: 255}, BorderColor: Color{A: 255}},
		{Kind: CommandDrawText, Rect: Rect{Width: 1, Height: 1}, Text: mask, Color: Color{A: 255}},
		{Kind: CommandDrawIcon, Rect: Rect{Width: 1, Height: 1}, Text: mask, Color: Color{A: 255}},
		{Kind: CommandDrawImage, Rect: Rect{Width: 1, Height: 1}, Image: image},
		{Kind: CommandPopClip},
		{Kind: CommandFillRoundedRect, Rect: Rect{Width: 1, Height: 1}, Color: Color{A: 255}},
		{Kind: CommandPopClip},
	}
	recorder := &RecordingPainter{}
	if err := Replay(list, recorder); err != nil {
		t.Fatal(err)
	}
	if got, want := len(recorder.Records), 6; got != want {
		t.Fatalf("records = %#v, want %d records", recorder.Records, want)
	}
	for _, index := range []int{1, 4} {
		if recorder.Records[index].Kind != CommandFillRoundedRect {
			t.Fatalf("record %d = %#v, want visible fill", index, recorder.Records[index])
		}
	}
	if recorder.Records[0].Kind != CommandPushClip || recorder.Records[2].Kind != CommandPushClip ||
		recorder.Records[3].Kind != CommandPushClip || recorder.Records[5].Kind != CommandPopClip {
		t.Fatalf("clip stack was not replayed: %#v", recorder.Records)
	}
}

func TestReplayRejectsUnbalancedStacks(t *testing.T) {
	for _, list := range []DisplayList{
		{{Kind: CommandPopClip}},
		{{Kind: CommandPushClip, Rect: Rect{Width: 1, Height: 1}}},
		{{Kind: CommandPopOpacity}},
		{{Kind: CommandPushOpacity, Opacity: .5}},
	} {
		if err := Replay(list, &RecordingPainter{}); err == nil {
			t.Fatalf("unbalanced list accepted: %#v", list)
		}
	}
}

func TestReplayParentChildOpacityMultiplies(t *testing.T) {
	list := DisplayList{
		{Kind: CommandPushOpacity, Opacity: .5},
		{Kind: CommandPushOpacity, Opacity: .4},
		{Kind: CommandFillStrokeRoundedRect, Rect: Rect{Width: 10, Height: 10}, Width: 1, Color: Color{A: 200}, BorderColor: Color{A: 150}},
		{Kind: CommandPopOpacity},
		{Kind: CommandPopOpacity},
	}
	recorder := &RecordingPainter{}
	if err := Replay(list, recorder); err != nil {
		t.Fatal(err)
	}
	if got := recorder.Records[0].Color.A; got != 40 {
		t.Fatalf("world alpha = %d, want 40", got)
	}
	if got := recorder.Records[0].BorderColor.A; got != 30 {
		t.Fatalf("world border alpha = %d, want 30", got)
	}
}

func TestReplayComposesRoundedImageOpacity(t *testing.T) {
	bitmap := &ImageBitmap{Key: "avatar", Width: 1, Height: 1, Pixels: []byte{1, 2, 3, 255}}
	clip := Rect{Width: 40, Height: 40}
	radii := Radii{TopLeft: 20, TopRight: 20, BottomRight: 20, BottomLeft: 20}
	recorder := &RecordingPainter{}
	if err := Replay(DisplayList{
		{Kind: CommandPushOpacity, Opacity: .5},
		{Kind: CommandDrawImage, Rect: Rect{X: -20, Width: 80, Height: 40}, ImageClip: clip, Radii: radii, Image: bitmap},
		{Kind: CommandPopOpacity},
	}, recorder); err != nil {
		t.Fatal(err)
	}
	if got := recorder.Records[0]; got.Color.A != 128 || got.ImageClip != clip || got.Radii != radii {
		t.Fatalf("rounded image replay = %+v", got)
	}
}

func BenchmarkReplayDisplayList(b *testing.B) {
	for _, count := range []int{100, 1000} {
		b.Run(fmt.Sprintf("commands-%d", count), func(b *testing.B) {
			list := make(DisplayList, 0, count+4)
			list = append(list, Command{Kind: CommandPushClip, Rect: Rect{Width: 1000, Height: 1000}}, Command{Kind: CommandPushOpacity, Opacity: .8})
			for index := 0; index < count; index++ {
				list = append(list, Command{Kind: CommandFillRoundedRect, Rect: Rect{X: float32(index % 50), Y: float32(index / 50), Width: 12, Height: 8}, Radii: Radii{TopLeft: 2, TopRight: 2, BottomRight: 2, BottomLeft: 2}, Color: Color{R: 80, G: 120, B: 200, A: 220}})
			}
			list = append(list, Command{Kind: CommandPopOpacity}, Command{Kind: CommandPopClip})
			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				recorder := RecordingPainter{Records: make(DisplayList, 0, count+4)}
				if err := Replay(list, &recorder); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
