package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

type scriptedEvents struct {
	events []platform.Event
	waits  int
	queued []platform.Event
}

func (source *scriptedEvents) Wait(context.Context, *time.Time) (platform.Event, error) {
	event := source.events[source.waits]
	source.waits++
	return event, nil
}

func (*scriptedEvents) Wake() error { return nil }

func (source *scriptedEvents) Poll() (platform.Event, bool, error) {
	if len(source.queued) == 0 {
		return platform.Event{}, false, nil
	}
	event := source.queued[0]
	source.queued = source.queued[1:]
	return event, true, nil
}

type recordingRenderer struct {
	renders, presents int
	presentErr        error
}

func (renderer *recordingRenderer) Render(paint.DisplayList) error {
	renderer.renders++
	return nil
}

func TestLoopCoalescesConsecutiveResizeAndScaleToLatest(t *testing.T) {
	latest := platform.Viewport{LogicalWidth: 900, LogicalHeight: 700}
	events := &scriptedEvents{
		events: []platform.Event{{Kind: platform.EventResize}},
		queued: []platform.Event{
			{Kind: platform.EventResize, Viewport: platform.Viewport{LogicalWidth: 800, LogicalHeight: 600}},
			{Kind: platform.EventScale, Viewport: latest},
			{Kind: platform.EventClose},
		},
	}
	renderer := &recordingRenderer{}
	var handled []platform.Event
	loop := Loop{
		Clock: fixedClock{now: time.Unix(1, 0)}, Events: events, Renderer: renderer,
		Display: func() paint.DisplayList { return nil },
		Handle: func(event platform.Event) (bool, bool, error) {
			handled = append(handled, event)
			return false, event.Kind == platform.EventClose, nil
		},
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(handled) != 2 || handled[0].Kind != platform.EventScale || handled[0].Viewport != latest {
		t.Fatalf("handled events = %#v, want latest resize then close", handled)
	}
	if renderer.presents != 1 {
		t.Fatalf("presents = %d, want one initial frame before close", renderer.presents)
	}
}

func TestLoopIdleWakeDoesNotAccumulateFrames(t *testing.T) {
	events := &scriptedEvents{events: []platform.Event{
		{Kind: platform.EventWake},
		{Kind: platform.EventWake},
		{Kind: platform.EventClose},
	}}
	renderer := &recordingRenderer{}
	loop := Loop{
		Clock: fixedClock{now: time.Unix(1, 0)}, Events: events, Renderer: renderer,
		Display: func() paint.DisplayList { return nil },
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if renderer.presents != 1 || renderer.renders != 1 {
		t.Fatalf("renders/presents = %d/%d, want 1/1", renderer.renders, renderer.presents)
	}
}

func TestAfterFirstPresentRunsOnceAndMayCloseWithoutWaiting(t *testing.T) {
	events := &scriptedEvents{}
	renderer := &recordingRenderer{}
	calls := 0
	loop := Loop{
		Clock: fixedClock{now: time.Unix(1, 0)}, Events: events, Renderer: renderer,
		Display: func() paint.DisplayList { return nil },
		AfterFirstPresent: func() (bool, error) {
			calls++
			return true, nil
		},
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || renderer.presents != 1 || events.waits != 0 {
		t.Fatalf("shown callback=%d presents=%d waits=%d", calls, renderer.presents, events.waits)
	}
}

func TestAfterFirstPresentDoesNotRepeatAndSkipsPresentFailure(t *testing.T) {
	events := &scriptedEvents{events: []platform.Event{{Kind: platform.EventExpose}, {Kind: platform.EventClose}}}
	renderer := &recordingRenderer{}
	calls := 0
	loop := Loop{
		Clock: fixedClock{now: time.Unix(1, 0)}, Events: events, Renderer: renderer,
		Display:           func() paint.DisplayList { return nil },
		AfterFirstPresent: func() (bool, error) { calls++; return false, nil },
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || renderer.presents != 2 {
		t.Fatalf("shown callback=%d presents=%d, want 1/2", calls, renderer.presents)
	}

	wantErr := errors.New("present failed")
	renderer = &recordingRenderer{presentErr: wantErr}
	calls = 0
	loop.Events = &scriptedEvents{}
	loop.Renderer = renderer
	if err := loop.Run(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("present error = %v, want %v", err, wantErr)
	}
	if calls != 0 {
		t.Fatalf("shown callback ran %d times after failed present", calls)
	}
}

func (renderer *recordingRenderer) Present() error {
	renderer.presents++
	return renderer.presentErr
}

func TestLoopPresentsOnlyInitialAndDirtyFrames(t *testing.T) {
	events := &scriptedEvents{events: []platform.Event{
		{Kind: platform.EventWake},
		{Kind: platform.EventExpose},
		{Kind: platform.EventClose},
	}}
	renderer := &recordingRenderer{}
	loop := Loop{
		Clock: fixedClock{now: time.Unix(1, 0)}, Events: events, Renderer: renderer,
		Display: func() paint.DisplayList { return nil },
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if renderer.presents != 2 || renderer.renders != 2 {
		t.Fatalf("renders/presents = %d/%d, want 2/2", renderer.renders, renderer.presents)
	}
}

func TestOptionalMetricsCountEventsAndFrames(t *testing.T) {
	events := &scriptedEvents{events: []platform.Event{{Kind: platform.EventExpose}, {Kind: platform.EventClose}}}
	renderer := &recordingRenderer{}
	metrics := NewMetrics()
	loop := Loop{
		Clock: fixedClock{now: time.Unix(1, 0)}, Events: events, Renderer: renderer, Metrics: metrics,
		Display: func() paint.DisplayList { return nil },
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	eventCount, latency, frames := metrics.Snapshot()
	if eventCount != 2 || latency.Count != 1 || frames.Count != 2 {
		t.Fatalf("events=%d latency=%+v frames=%+v", eventCount, latency, frames)
	}
}
