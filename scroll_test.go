package dxui

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

func scrollItems(count int, height float32) View {
	children := make([]View, count)
	for index := range children {
		children[index] = Text(TextProps{
			Key: (fmt.Sprintf("item-%d", index)), Style: Style{Height: Px(height), Shrink: Some(float32(0))},
			Value: fmt.Sprintf("item %d", index),
		})
	}
	return Box(BoxProps{}, children...)
}

func wheel(x, y, dx, dy float32) platform.Event {
	return platform.Event{Kind: platform.EventMouseWheel, Pointer: platform.PointerEvent{X: x, Y: y, WheelX: dx, WheelY: dy}}
}

func TestScrollInitialControlledClampResizeAndContentChange(t *testing.T) {
	uncontrolledInitial := float32(80)
	uncontrolled := NewApp(AppOptions{Width: 100, Height: 100})
	uncontrolled.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, InitialOffset: Some(Point{Y: uncontrolledInitial})}, scrollItems(20, 20))
	}
	if err := uncontrolled.buildRoot(); err != nil {
		t.Fatal(err)
	}
	beforeInitialChange := uncontrolled.Diagnostics()
	uncontrolledInitial = 10
	if _, err := uncontrolled.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if got := uncontrolled.retained.Root().State.ScrollY; got != 80 {
		t.Fatalf("new InitialOffset reset retained state to %v", got)
	}
	if after := uncontrolled.Diagnostics(); after.LayoutCount != beforeInitialChange.LayoutCount || after.PaintCount != beforeInitialChange.PaintCount {
		t.Fatalf("InitialOffset rebuild caused layout/paint: before=%+v after=%+v", beforeInitialChange, after)
	}

	controlled := Point{Y: 500}
	count := 20
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		return Scroll(ScrollProps{
			Key: "scroll", Style: Style{Width: Px(100), Height: Px(100)},
			Axis: ScrollVertical, InitialOffset: Some(Point{Y: 80}), Offset: Some(controlled),
		}, scrollItems(count, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	root := app.retained.Root()
	if root.State.ScrollY != 300 {
		t.Fatalf("controlled clamp = %v, want 300", root.State.ScrollY)
	}
	controlled.Y = 40
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if root = app.retained.Root(); root.State.ScrollY != 40 {
		t.Fatalf("controlled update = %v, want 40", root.State.ScrollY)
	}
	count = 3
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if got := app.retained.Root().State.ScrollY; got != 0 {
		t.Fatalf("content shrink clamp = %v, want 0", got)
	}
	count, controlled.Y = 20, 250
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if err := app.relayout(100, 200); err != nil {
		t.Fatal(err)
	}
	if got := app.retained.Root().State.ScrollY; got != 200 {
		t.Fatalf("resize clamp = %v, want 200", got)
	}
}

func TestScrollRejectsInvalidEnumsAndOffsets(t *testing.T) {
	for _, props := range []ScrollProps{
		{Axis: ScrollAxis(99)},
		{Scrollbar: ScrollbarPolicy(99)},
		{InitialOffset: Some(Point{Y: -1})},
		{Offset: Some(Point{X: float32(math.NaN())})},
	} {
		app := NewApp(AppOptions{Width: 10, Height: 10})
		app.root = func() View { return Scroll(props, Text(TextProps{Value: "x"})) }
		if err := app.buildRoot(); err == nil {
			t.Fatalf("invalid scroll props accepted: %+v", props)
		}
	}
}

func TestScrollWheelIsPaintOnlyAndControlledCallbackIsAuthoritative(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarHidden}, scrollItems(20, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	dirty, _, err := app.handleEvent(wheel(10, 10, 0, -1))
	if err != nil || !dirty {
		t.Fatalf("wheel dirty/error = %v/%v", dirty, err)
	}
	after := app.Diagnostics()
	if after.LayoutCount != before.LayoutCount || after.PaintCount != before.PaintCount+1 {
		t.Fatalf("wheel layout/paint = %d/%d, before %d/%d", after.LayoutCount, after.PaintCount, before.LayoutCount, before.PaintCount)
	}
	if got := app.retained.Root().State.ScrollY; got != scrollWheelUnit {
		t.Fatalf("wheel offset = %v, want %v", got, scrollWheelUnit)
	}
	if deadline := app.nextDeadline(time.Now()); deadline != nil {
		t.Fatalf("Scroll unexpectedly scheduled deadline %v", deadline)
	}

	controlled := Point{}
	var proposed Point
	accept := false
	app = NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Offset: Some(controlled), OnScroll: func(offset Point) {
			proposed = offset
			if accept {
				controlled = offset
			}
		}}, scrollItems(20, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(wheel(10, 10, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if proposed.Y != scrollWheelUnit || app.retained.Root().State.ScrollY != 0 {
		t.Fatalf("rejected controlled proposal/state = %v/%v", proposed.Y, app.retained.Root().State.ScrollY)
	}
	accept = true
	dirty, _, err = app.handleEvent(wheel(10, 10, 0, -1))
	if err != nil || !dirty {
		t.Fatalf("accepted controlled wheel dirty/error = %v/%v", dirty, err)
	}
	if app.retained.Root().State.ScrollY != scrollWheelUnit {
		t.Fatal("accepted controlled offset was not applied")
	}
}

func TestScrollbarPoliciesAndMinimumThumb(t *testing.T) {
	viewport := layout.Rect{Width: 100, Height: 100}
	if bars := scrollbarsFor(viewport, layout.Size{Width: 100, Height: 80}, Point{}, ScrollVertical, ScrollbarAuto, 8, 24, 2); len(bars) != 0 {
		t.Fatalf("auto non-overflow bars = %d", len(bars))
	}
	bars := scrollbarsFor(viewport, layout.Size{Width: 100, Height: 80}, Point{}, ScrollVertical, ScrollbarAlways, 8, 24, 2)
	if len(bars) != 1 || !bars[0].Disabled || bars[0].Thumb != bars[0].Track {
		t.Fatalf("always disabled bar = %+v", bars)
	}
	if bars := scrollbarsFor(viewport, layout.Size{Width: 100, Height: 1000}, Point{}, ScrollVertical, ScrollbarHidden, 8, 24, 2); len(bars) != 0 {
		t.Fatalf("hidden bars = %d", len(bars))
	}
	bars = scrollbarsFor(viewport, layout.Size{Width: 100, Height: 1000}, Point{}, ScrollVertical, ScrollbarAuto, 8, 24, 2)
	if len(bars) != 1 || bars[0].Thumb.Height != 24 {
		t.Fatalf("minimum thumb = %+v", bars)
	}
}

func TestScrollbarIdleAndHoverVisuals(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarAlways}, scrollItems(20, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	scrollbarCommands := func() (paint.Command, paint.Command) {
		var commands []paint.Command
		for _, command := range app.currentDisplay() {
			if command.NodeID == app.retained.Root().ID && command.Kind == paint.CommandFillRoundedRect {
				commands = append(commands, command)
			}
		}
		if len(commands) != 2 {
			t.Fatalf("scrollbar fill commands = %d, want track and thumb", len(commands))
		}
		return commands[0], commands[1]
	}

	track, thumb := scrollbarCommands()
	if track.Color.A != 0 {
		t.Fatalf("idle track alpha = %d, want invisible", track.Color.A)
	}
	if track.Rect.Width != 6 || thumb.Rect.Width != 3 || thumb.Rect.X != track.Rect.X+1.5 || thumb.Color.A != 140 {
		t.Fatalf("idle track/thumb = %+v/%+v alpha %d, want 6-wide track and centered 3-wide thumb with alpha 140", track.Rect, thumb.Rect, thumb.Color.A)
	}

	// The full 6-unit geometry remains interactive even outside the idle
	// thumb's visible 3-unit width.
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 92.5, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	track, thumb = scrollbarCommands()
	if thumb.Rect.Width != 6 || thumb.Rect.Width != track.Rect.Width || thumb.Rect.X != track.Rect.X || thumb.Color.A != 255 {
		t.Fatalf("hover thumb = rect %+v alpha %d, want 6-wide opaque thumb in track %+v", thumb.Rect, thumb.Color.A, track.Rect)
	}
	hoverColor, err := app.theme.color(TokenColor(ColorSemanticScrollThumbHover))
	if err != nil {
		t.Fatal(err)
	}
	borderColor, err := app.theme.color(TokenColor(ColorSemanticBorder))
	if err != nil {
		t.Fatal(err)
	}
	accentColor, err := app.theme.color(TokenColor(ColorSemanticAccentHover))
	if err != nil {
		t.Fatal(err)
	}
	if hoverColor != borderColor || hoverColor == accentColor || thumb.Color != paintColor(borderColor) {
		t.Fatalf("hover color = %+v, want non-accent border color %+v", hoverColor, borderColor)
	}
	activeColor, err := app.theme.color(TokenColor(ColorSemanticScrollThumbActive))
	if err != nil {
		t.Fatal(err)
	}
	primaryColor, err := app.theme.color(TokenColor(ColorSemanticAccent))
	if err != nil {
		t.Fatal(err)
	}
	if activeColor != borderColor || activeColor == primaryColor {
		t.Fatalf("active color = %+v, want non-accent border color %+v", activeColor, borderColor)
	}
}

func TestScrollbarThemeMetricIsPaintOnly(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarAlways}, scrollItems(20, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	before := app.Diagnostics()
	theme := LightTheme()
	theme.Semantic.Metrics[MetricComponentScrollThickness] = Metric(10)
	if err := app.SetTheme(theme); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("theme dirty/error = %v/%v", dirty, err)
	}
	after := app.Diagnostics()
	if after.LayoutCount != before.LayoutCount || after.PaintCount != before.PaintCount+1 {
		t.Fatalf("scrollbar theme layout/paint = %d/%d, before %d/%d", after.LayoutCount, after.PaintCount, before.LayoutCount, before.PaintCount)
	}
}

func TestNestedScrollConsumesThenPassesRemainderAtBoundary(t *testing.T) {
	app := NewApp(AppOptions{Width: 120, Height: 100})
	app.root = func() View {
		inner := Scroll(ScrollProps{Key: "inner", Style: Style{Height: Px(60), Shrink: Some(float32(0))}, Axis: ScrollVertical, Scrollbar: ScrollbarHidden}, scrollItems(8, 20))
		content := Box(BoxProps{}, inner, Text(TextProps{Style: Style{Height: Px(240), Shrink: Some(float32(0))}, Value: "outer tail"}))
		return Scroll(ScrollProps{Key: "outer", Axis: ScrollVertical, Scrollbar: ScrollbarHidden}, content)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	outer := app.retained.Root()
	inner := outer.Children[0].Children[0]
	if _, _, err := app.handleEvent(wheel(10, 10, 0, -10)); err != nil {
		t.Fatal(err)
	}
	if inner.State.ScrollY != 100 {
		t.Fatalf("inner offset = %v, want boundary 100", inner.State.ScrollY)
	}
	if outer.State.ScrollY <= 0 {
		t.Fatal("outer did not consume remainder after inner boundary")
	}
}

func TestHorizontalScrollUsesShiftWheelAndKeyboard(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 60})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollHorizontal, Scrollbar: ScrollbarHidden}, Box(BoxProps{Direction: Horizontal},
			Text(TextProps{Style: Style{Width: Px(300), Shrink: Some(float32(0))}, Value: "wide"}),
		))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.modifiers.Shift = true
	if _, _, err := app.handleEvent(wheel(10, 10, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if got := app.retained.Root().State.ScrollX; got != scrollWheelUnit {
		t.Fatalf("shift-wheel X = %v", got)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 10, 10))
	app.handleEvent(appPointer(platform.EventMouseUp, 10, 10))
	if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyRight, false)); err != nil {
		t.Fatal(err)
	}
	if got := app.retained.Root().State.ScrollX; got != 2*scrollWheelUnit {
		t.Fatalf("keyboard X = %v", got)
	}
}

func TestScrollPaintHitTransformClipAndCullAgree(t *testing.T) {
	presses := 0
	children := []View{
		Button(ButtonProps{Style: Style{Height: Px(30), Shrink: Some(float32(0))}, OnPress: func() { presses++ }}, Text(TextProps{Value: "first"})),
		Button(ButtonProps{Style: Style{Height: Px(30), Shrink: Some(float32(0))}, OnPress: func() { presses += 10 }}, Text(TextProps{Value: "second"})),
	}
	for index := 0; index < 100; index++ {
		children = append(children, Text(TextProps{Style: Style{Height: Px(20), Shrink: Some(float32(0))}, Value: fmt.Sprintf("tail %d", index)}))
	}
	app := NewApp(AppOptions{Width: 100, Height: 30})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, InitialOffset: Some(Point{Y: 30}), Scrollbar: ScrollbarHidden}, Box(BoxProps{}, children...))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 5, 5))
	app.handleEvent(appPointer(platform.EventMouseUp, 5, 5))
	if presses != 10 {
		t.Fatalf("translated hit activated %d, want second button", presses)
	}
	drawText := 0
	for _, command := range app.currentDisplay() {
		if command.Kind == paint.CommandDrawText {
			drawText++
			if command.Rect.Y < 0 || command.Rect.Y >= 30 {
				t.Fatalf("offscreen text command survived cull: %+v", command.Rect)
			}
		}
	}
	if drawText == 0 || drawText > 3 {
		t.Fatalf("visible text commands = %d, want bounded viewport work", drawText)
	}
}

func TestScrollbarThumbAndDrag(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarAlways}, scrollItems(20, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	beforeLayout := app.Diagnostics().LayoutCount
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, 95, 5)); err != nil {
		t.Fatal(err)
	}
	if app.retained.Root().State.ScrollbarHover != scrollbarAxisVertical {
		t.Fatal("thumb hover state was not set")
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 95, 5))
	app.handleEvent(appPointer(platform.EventMouseMove, 95, 35))
	app.handleEvent(appPointer(platform.EventMouseUp, 95, 35))
	if app.retained.Root().State.ScrollY <= 0 {
		t.Fatal("thumb drag did not change offset")
	}
	if app.Diagnostics().LayoutCount != beforeLayout {
		t.Fatal("thumb drag caused layout")
	}
}

func TestScrollbarDragCancelsWhenNodeIsRemoved(t *testing.T) {
	visible := true
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		if !visible {
			return Text(TextProps{Value: "removed"})
		}
		return Scroll(ScrollProps{Key: "drag", Axis: ScrollVertical, Scrollbar: ScrollbarAlways}, scrollItems(20, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 95, 5))
	if app.scrollDrag.id == 0 {
		t.Fatal("drag did not start")
	}
	visible = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.scrollDrag.id != 0 {
		t.Fatal("removed Scroll retained drag capture")
	}
}

func TestScrollKeyedReorderRetainsOffsetAndReplacementDropsIt(t *testing.T) {
	order := []string{"a", "b"}
	app := NewApp(AppOptions{Width: 200, Height: 100})
	app.root = func() View {
		children := make([]View, len(order))
		for index, key := range order {
			children[index] = Scroll(ScrollProps{Key: key, Style: Style{Width: Px(100), Height: Px(100)}, Axis: ScrollVertical}, scrollItems(20, 20))
		}
		return Box(BoxProps{Direction: Horizontal}, children...)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	a := app.retained.Root().Children[0]
	a.State.ScrollY = 75
	aID := a.ID
	order = []string{"b", "a"}
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	a = app.retained.Root().Children[1]
	if a.ID != aID || a.State.ScrollY != 75 {
		t.Fatalf("reordered identity/offset = %d/%v, want %d/75", a.ID, a.State.ScrollY, aID)
	}
	order[1] = "c"
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if replacement := app.retained.Root().Children[1]; replacement.ID == aID || replacement.State.ScrollY != 0 {
		t.Fatalf("replacement retained old state: id=%d offset=%v", replacement.ID, replacement.State.ScrollY)
	}
}

func BenchmarkScroll1000Children(b *testing.B) {
	app := NewApp(AppOptions{Width: 320, Height: 480})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarAuto}, scrollItems(1000, 20))
	}
	if err := app.buildRoot(); err != nil {
		b.Fatal(err)
	}
	event := wheel(10, 10, 0, -1)
	before := app.Diagnostics()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, _, err := app.handleEvent(event); err != nil {
			b.Fatal(err)
		}
		if app.retained.Root().State.ScrollY >= 19000 {
			app.retained.Root().State.ScrollY = 0
		}
	}
	b.StopTimer()
	diagnostics := app.Diagnostics()
	b.ReportMetric(float64(diagnostics.LayoutCount-before.LayoutCount)/float64(b.N), "layouts/op")
	b.ReportMetric(float64(diagnostics.PaintCount-before.PaintCount)/float64(b.N), "paints/op")
}

func BenchmarkResize1000Children(b *testing.B) {
	app := NewApp(AppOptions{Width: 320, Height: 480})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarAuto}, scrollItems(1000, 20))
	}
	if err := app.buildRoot(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := range b.N {
		if err := app.relayout(int32(320+index%20), int32(480+index%16)); err != nil {
			b.Fatal(err)
		}
	}
}

func TestScroll1000AllocationRegressionGate(t *testing.T) {
	app := NewApp(AppOptions{Width: 320, Height: 480})
	app.root = func() View {
		return Scroll(ScrollProps{Axis: ScrollVertical, Scrollbar: ScrollbarAuto}, scrollItems(1000, 20))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	event := wheel(10, 10, 0, -1)
	beforeLayout := app.Diagnostics().LayoutCount
	var eventErr error
	allocations := testing.AllocsPerRun(20, func() {
		_, _, eventErr = app.handleEvent(event)
		if app.retained.Root().State.ScrollY >= 19000 {
			app.retained.Root().State.ScrollY = 0
		}
	})
	if eventErr != nil {
		t.Fatal(eventErr)
	}
	if allocations > 300 {
		t.Fatalf("scroll allocations/op = %.0f, gate 300", allocations)
	}
	if app.Diagnostics().LayoutCount != beforeLayout {
		t.Fatal("scroll event unexpectedly ran layout")
	}
}
