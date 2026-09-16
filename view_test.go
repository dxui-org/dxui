package dxui

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
	uiruntime "github.com/dxui-org/dxui/internal/runtime"
)

func TestViewCopiesChildren(t *testing.T) {
	children := []View{Text(TextProps{Value: "first"})}
	view := Box(BoxProps{}, children...)
	children[0] = Text(TextProps{Value: "changed"})
	if got := view.node.children[0].node.text.Value; got != "first" {
		t.Fatalf("stored child = %q, want first", got)
	}
}

func TestRootBuilderProducesView(t *testing.T) {
	app := NewApp(AppOptions{Title: "test"})
	app.root = func() View {
		return Text(TextProps{Value: "hello"})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := app.view.node.text.Value; got != "hello" {
		t.Fatalf("root text = %q, want hello", got)
	}
}

func TestRootCallbackFailuresBecomeErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		root func() View
	}{
		{name: "invalid view", root: func() View { return View{} }},
		{name: "panic", root: func() View { panic("root failed") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{})
			app.root = test.root
			if err := app.buildRoot(); err == nil {
				t.Fatal("root callback failure returned nil error")
			}
		})
	}
}

type appFakeEvents struct{ wakes int }

func (*appFakeEvents) Wait(context.Context, *time.Time) (platform.Event, error) {
	return platform.Event{}, errors.New("unused")
}
func (*appFakeEvents) Poll() (platform.Event, bool, error) { return platform.Event{}, false, nil }
func (events *appFakeEvents) Wake() error                  { events.wakes++; return nil }

func TestInvalidateWakesAndRebuildsOnUIHandling(t *testing.T) {
	events := &appFakeEvents{}
	app := NewApp(AppOptions{})
	builds := 0
	app.root = func() View {
		builds++
		return Text(TextProps{Value: "root"})
	}
	app.running = true
	app.events = events
	app.invalidate()
	if builds != 0 {
		t.Fatalf("builder ran on invalidating goroutine; builds = %d", builds)
	}
	if events.wakes != 1 {
		t.Fatalf("wake count = %d, want 1", events.wakes)
	}
	dirty, err := app.runQueuedWork()
	if err != nil {
		t.Fatal(err)
	}
	if !dirty || builds != 1 {
		t.Fatalf("dirty/builds = %v/%d, want true/1", dirty, builds)
	}
}

func TestUpdateDefersCallbackUntilUIHandling(t *testing.T) {
	events := &appFakeEvents{}
	app := NewApp(AppOptions{})
	app.root = func() View { return Text(TextProps{Value: "root"}) }
	app.running = true
	app.events = events
	called := false
	if err := app.Update(func() { called = true }); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("update callback ran on caller")
	}
	if _, err := app.runQueuedWork(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("update callback did not run during UI handling")
	}
}

func TestCloseIsIdempotentAndWakesOnce(t *testing.T) {
	events := &appFakeEvents{}
	app := NewApp(AppOptions{})
	app.running = true
	app.events = events
	app.Close()
	app.Close()
	if events.wakes != 1 {
		t.Fatalf("wake count = %d, want 1", events.wakes)
	}
	if err := app.Update(func() {}); !errors.Is(err, ErrAppClosed) {
		t.Fatalf("Update error = %v, want ErrAppClosed", err)
	}
}

func TestRunRejectsNilWithoutConsumingAppAndThenIsSingleUse(t *testing.T) {
	app := NewApp(AppOptions{})
	if err := app.Run(nil); err == nil {
		t.Fatal("Run(nil) returned nil error")
	}
	if app.started {
		t.Fatal("Run(nil) consumed the App")
	}
	app.options.Width = -1
	root := func() View { return Text(TextProps{Value: "unused"}) }
	if err := app.Run(root); err == nil {
		t.Fatal("invalid startup returned nil error")
	}
	if err := app.Run(root); err == nil {
		t.Fatal("second Run returned nil error")
	}
}

func TestCloseBeforeRunIsNoOp(t *testing.T) {
	app := NewApp(AppOptions{})
	app.Close()
	if app.closed {
		t.Fatal("Close before Run changed lifecycle state")
	}
	if err := app.Update(func() {}); !errors.Is(err, ErrAppNotRunning) {
		t.Fatalf("Update before Run = %v, want ErrAppNotRunning", err)
	}
}

func TestUnchangedRebuildStopsBeforeLayoutAndPaint(t *testing.T) {
	app := NewApp(AppOptions{})
	builds := 0
	app.root = func() View {
		builds++
		return Text(TextProps{Value: "same"})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.running = true
	app.invalidate()
	dirty, err := app.runQueuedWork()
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("unchanged rebuild requested a frame")
	}
	diagnostics := app.Diagnostics()
	if builds != 2 || diagnostics.BuildCount != 2 || diagnostics.LayoutCount != 1 || diagnostics.PaintCount != 1 {
		t.Fatalf("build/layout/paint = %d/%d/%d (builder %d)", diagnostics.BuildCount, diagnostics.LayoutCount, diagnostics.PaintCount, builds)
	}
}

func TestCommitProducesPublicBoxGeometry(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal,
			Style: Style{Padding: UniformEdges(Metric(5))},
			Gap:   10,
		},
			Text(TextProps{Style: Style{Width: Px(20), Height: Px(10)}}),
			Text(TextProps{Style: Style{Width: Px(30), Height: Px(10)}}),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if app.geometry == nil {
		t.Fatal("commit produced no geometry")
	}
	first, second := app.geometry.Children[0].Rect, app.geometry.Children[1].Rect
	if first.X != 5 || first.Y != 5 || first.Width != 20 || second.X != 35 || second.Width != 30 {
		t.Fatalf("child geometry = %#v / %#v", first, second)
	}
}

func TestBoxDirectionChangePreservesRetainedIdentity(t *testing.T) {
	direction := Vertical
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		return Box(BoxProps{Direction: direction, Gap: 5},
			Text(TextProps{Style: Style{Width: Px(10), Height: Px(10)}}),
			Text(TextProps{Style: Style{Width: Px(10), Height: Px(10)}}),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	rootID := app.retained.Root().ID
	childIDs := [2]uint64{app.retained.Root().Children[0].ID, app.retained.Root().Children[1].ID}
	if second := app.geometry.Children[1].Rect; second.X != 0 || second.Y != 15 {
		t.Fatalf("vertical second child = %+v", second)
	}

	direction = Horizontal
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	root := app.retained.Root()
	if root.ID != rootID || root.Children[0].ID != childIDs[0] || root.Children[1].ID != childIDs[1] {
		t.Fatalf("direction change replaced identities: root %d->%d children %v->[%d %d]", rootID, root.ID, childIDs, root.Children[0].ID, root.Children[1].ID)
	}
	if second := app.geometry.Children[1].Rect; second.X != 15 || second.Y != 0 {
		t.Fatalf("horizontal second child = %+v", second)
	}
}

func TestBoxRejectsInvalidDirection(t *testing.T) {
	app := NewApp(AppOptions{})
	app.root = func() View { return Box(BoxProps{Direction: Direction(2)}) }
	if err := app.buildRoot(); err == nil {
		t.Fatal("invalid Box direction was accepted")
	}
}

func TestResizeSequenceRelayoutsAgainstLatestLogicalViewport(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal}, Text(TextProps{Style: Style{Width: Percent(50), Height: Px(10)}}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := app.geometry.Children[0].Rect.Width; got != 50 {
		t.Fatalf("initial percentage width = %v", got)
	}
	for _, width := range []int32{0, 320, 75, 200} {
		dirty, closeApp, err := app.handleEvent(platform.Event{Kind: platform.EventResize, Viewport: platform.Viewport{LogicalWidth: width, LogicalHeight: 40}})
		if err != nil || !dirty || closeApp {
			t.Fatalf("resize %d result = %v/%v/%v", width, dirty, closeApp, err)
		}
		if got := app.geometry.Children[0].Rect.Width; got != float32(width)/2 {
			t.Fatalf("resize %d percentage width = %v", width, got)
		}
	}
	if app.Diagnostics().LayoutCount != 5 {
		t.Fatalf("layout count = %d, want initial plus four resizes", app.Diagnostics().LayoutCount)
	}
}

func TestLayoutFailurePreservesRetainedTreeAndGeometry(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	invalid := false
	app.root = func() View {
		gap := float32(4)
		if invalid {
			gap = -1
		}
		return Box(BoxProps{Direction: Horizontal, Gap: gap}, Text(TextProps{Style: Style{Width: Px(20), Height: Px(10)}}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	oldRoot, oldGeometry := app.retained.Root(), app.geometry
	invalid = true
	if err := app.buildRoot(); err == nil {
		t.Fatal("negative gap was accepted")
	}
	if app.retained.Root() != oldRoot || app.geometry != oldGeometry {
		t.Fatal("failed layout changed the last committed tree or geometry")
	}
}

func TestPaintOnlyChangeDoesNotLayout(t *testing.T) {
	app := NewApp(AppOptions{})
	color := RGBA(1, 2, 3, 255)
	app.root = func() View {
		return Box(BoxProps{Style: Style{Background: LiteralColor(color)}})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.running = true
	color = RGBA(4, 5, 6, 255)
	app.invalidate()
	dirty, err := app.runQueuedWork()
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Fatal("paint change did not request a frame")
	}
	diagnostics := app.Diagnostics()
	if diagnostics.LayoutCount != 1 || diagnostics.PaintCount != 2 {
		t.Fatalf("layout/paint = %d/%d, want 1/2", diagnostics.LayoutCount, diagnostics.PaintCount)
	}
}

func TestRootInvalidationsCoalesce(t *testing.T) {
	app := NewApp(AppOptions{})
	rootBuilds := 0
	value := "first"
	app.root = func() View {
		rootBuilds++
		return Box(BoxProps{}, Text(TextProps{Value: value}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.running = true
	value = "second"
	app.invalidate()
	app.invalidate()
	dirty, err := app.runQueuedWork()
	if err != nil {
		t.Fatal(err)
	}
	if !dirty || rootBuilds != 2 {
		t.Fatalf("dirty/root builds = %v/%d", dirty, rootBuilds)
	}
}

func TestBuildFailurePreservesLastUsableViewAndTree(t *testing.T) {
	app := NewApp(AppOptions{})
	mode := 0
	app.root = func() View {
		switch mode {
		case 1:
			panic("boom")
		case 2:
			return Box(BoxProps{},
				Text(TextProps{Key: "duplicate"}),
				Text(TextProps{Key: "duplicate"}),
			)
		default:
			return Text(TextProps{Value: "usable"})
		}
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	oldView := app.view.node
	oldTree := app.retained.Root()
	for _, failureMode := range []int{1, 2} {
		mode = failureMode
		if err := app.buildRoot(); err == nil {
			t.Fatalf("failure mode %d returned nil", failureMode)
		}
		if app.view.node != oldView || app.retained.Root() != oldTree {
			t.Fatalf("failure mode %d replaced last usable commit", failureMode)
		}
	}
}

func TestRunUpdateFailureReportsAndCanRecover(t *testing.T) {
	mode := 0
	var reported error
	app := NewApp(AppOptions{OnError: func(err error) { reported = err }})
	app.root = func() View {
		if mode == 1 {
			return View{}
		}
		return Text(TextProps{Value: []string{"first", "recovered"}[mode/2]})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	oldRoot := app.retained.Root()
	app.running = true
	mode = 1
	app.invalidate()
	if dirty, closeApp, err := app.handleEvent(platform.Event{Kind: platform.EventWake}); err != nil || dirty || closeApp {
		t.Fatalf("failed update result = %v/%v/%v", dirty, closeApp, err)
	}
	if reported == nil || app.retained.Root() != oldRoot {
		t.Fatal("failed update was not reported transactionally")
	}
	mode = 2
	app.invalidate()
	if dirty, _, err := app.handleEvent(platform.Event{Kind: platform.EventWake}); err != nil || !dirty {
		t.Fatalf("recovery update result = %v/%v", dirty, err)
	}
	if app.retained.Root() == oldRoot || app.view.node.text.Value != "recovered" {
		t.Fatal("valid update did not recover after reported failure")
	}
}

type frameTestEvents struct {
	events []platform.Event
	index  int
}

func (events *frameTestEvents) Wait(context.Context, *time.Time) (platform.Event, error) {
	if events.index >= len(events.events) {
		return platform.Event{}, errors.New("frame test ran out of events")
	}
	event := events.events[events.index]
	events.index++
	return event, nil
}

func (*frameTestEvents) Poll() (platform.Event, bool, error) { return platform.Event{}, false, nil }
func (*frameTestEvents) Wake() error                         { return nil }

type frameTestRenderer struct{ renders, presents int }

func (value *frameTestRenderer) Render(paint.DisplayList) error { value.renders++; return nil }
func (value *frameTestRenderer) Present() error                 { value.presents++; return nil }

var _ renderer.Renderer = (*frameTestRenderer)(nil)

func TestFakeBackendCountsCoalescedBuildLayoutPaintPresent(t *testing.T) {
	app := NewApp(AppOptions{})
	value := "before"
	app.root = func() View {
		return Text(TextProps{Value: value})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	events := &frameTestEvents{events: []platform.Event{{Kind: platform.EventWake}, {Kind: platform.EventClose}}}
	renderer := &frameTestRenderer{}
	app.running = true
	app.events = events
	if err := app.Update(func() {
		value = "after"
	}); err != nil {
		t.Fatal(err)
	}
	loop := uiruntime.Loop{
		Clock: platform.SystemClock{}, Events: events, Renderer: renderer,
		Display: func() paint.DisplayList { return nil }, Handle: app.handleEvent,
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	diagnostics := app.Diagnostics()
	if diagnostics.BuildCount != 2 || diagnostics.LayoutCount != 2 || diagnostics.PaintCount != 2 {
		t.Fatalf("build/layout/paint = %d/%d/%d", diagnostics.BuildCount, diagnostics.LayoutCount, diagnostics.PaintCount)
	}
	if renderer.renders != 2 || renderer.presents != 2 {
		t.Fatalf("render/present = %d/%d, want 2/2", renderer.renders, renderer.presents)
	}
}

func TestDetailedDiagnosticsAreOptIn(t *testing.T) {
	off := NewApp(AppOptions{})
	if off.loopMetrics != nil || off.Diagnostics().CountersEnabled {
		t.Fatal("detailed performance counters were enabled by default")
	}

	on := NewApp(AppOptions{Diagnostics: true})
	on.root = func() View { return Text(TextProps{Value: "counter probe"}) }
	if err := on.buildRoot(); err != nil {
		t.Fatal(err)
	}
	diagnostics := on.Diagnostics()
	if on.loopMetrics == nil || !diagnostics.CountersEnabled {
		t.Fatal("explicit performance counters were not enabled")
	}
	if diagnostics.ReconcileCount != 1 || diagnostics.PaintNodeCount == 0 || diagnostics.CacheBudgetBytes == 0 {
		t.Fatalf("detailed diagnostics were not populated: %+v", diagnostics)
	}
}

func TestFlattenedCommonPropsAndConvenienceConstructors(t *testing.T) {
	families := []FontFamily{"primary"}
	shadows := []Shadow{{Blur: Metric(2)}}
	button := Button(ButtonProps{
		Key: "save",
		Style: Style{
			Width: Px(126), Padding: PaddingXY(20, 0), Radius: Round(0),
			Shadow: shadows, Text: TextStyle{Families: families},
		},
		Token:   ComponentButton,
		States:  StateStyles{Hover: StylePatch{Opacity: Some(float32(0))}},
		Pointer: PointerNone,
	}, Label("Save"))

	props := button.node.button.common()
	if props.Key != "save" || props.Token != ComponentButton || props.Pointer != PointerNone || props.Style.Width != Px(126) {
		t.Fatalf("flattened common props = %+v", props)
	}
	if !props.Style.Padding.Top.set || props.Style.Padding.Top.literal != 0 || !props.Style.Radius.TopLeft.set || props.Style.Radius.TopLeft.literal != 0 {
		t.Fatalf("explicit zero padding/radius lost: padding=%+v radius=%+v", props.Style.Padding, props.Style.Radius)
	}
	if opacity, ok := props.States.Hover.Opacity.get(); !ok || opacity != 0 {
		t.Fatalf("explicit zero state opacity = %v/%t", opacity, ok)
	}
	families[0] = "mutated"
	shadows[0].Blur = Metric(9)
	if props.Style.Text.Families[0] != "primary" || props.Style.Shadow[0].Blur.literal != 2 {
		t.Fatal("flattened props did not preserve immutable constructor copies")
	}

	label := Label(string([]byte{'x', 0xff}))
	if label.node.kind != viewText || label.node.text.Value != "x\uFFFD" || label.node.text.common().Style.Text.Size != 0 || label.node.text.common().Style.Text.LineHeight != 0 {
		t.Fatalf("Label did not reuse Text defaults: %+v", label.node)
	}
	textButton := TextButton(ButtonProps{Style: Style{Width: Px(120), Height: Px(40)}}, "Go")
	if textButton.node.kind != viewButton || len(textButton.node.children) != 1 || textButton.node.children[0].node.kind != viewBox {
		t.Fatalf("TextButton composition = %+v", textButton.node)
	}
	content := textButton.node.children[0].node.box
	if content.Direction != Horizontal || content.Justify != JustifyCenter || content.Align != AlignCenter || content.common().Style.Width != Percent(100) || content.common().Style.Grow != 1 {
		t.Fatalf("TextButton alignment = %+v", content)
	}
	app := NewApp(AppOptions{Width: 120, Height: 40})
	app.root = func() View {
		return TextButton(ButtonProps{Style: Style{Width: Px(120), Height: Px(40)}}, "Go")
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	buttonRect := app.geometry.Rect
	labelRect := app.geometry.Children[0].Children[0].Rect
	if labelRect.X+labelRect.Width/2 != buttonRect.X+buttonRect.Width/2 || labelRect.Y+labelRect.Height/2 != buttonRect.Y+buttonRect.Height/2 {
		t.Fatalf("TextButton label is not centered: button=%+v label=%+v", buttonRect, labelRect)
	}
}

func TestAssign(t *testing.T) {
	value := "before"
	Assign(&value)("after")
	if value != "after" {
		t.Fatalf("Assign value = %q", value)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("Assign(nil) did not panic")
		}
	}()
	Assign[string](nil)
}

func TestNumericStyleHelpersUseExistingValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		style Style
	}{
		{name: "negative padding", style: Style{Padding: Padding(-1)}},
		{name: "negative radius", style: Style{Radius: Round(-1)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{})
			app.root = func() View { return Box(BoxProps{Style: test.style}) }
			if err := app.buildRoot(); err == nil {
				t.Fatal("invalid helper-created metric was accepted")
			}
		})
	}
}
