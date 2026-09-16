package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

func TestWindowsRoutesAndPaintsOnlyDirtyWindow(t *testing.T) {
	one, two := platform.WindowID(1), platform.WindowID(2)
	events := &scriptedEvents{events: []platform.Event{{Window: two, Kind: platform.EventExpose}, {Window: one, Kind: platform.EventClose}}}
	r1, r2 := &recordingRenderer{}, &recordingRenderer{}
	dirty := map[platform.WindowID]bool{one: true, two: true}
	live := []platform.WindowID{one, two}
	loop := Windows{
		Clock: fixedClock{now: time.Unix(1, 0)}, Events: events,
		IDs: func() []platform.WindowID { return append([]platform.WindowID(nil), live...) },
		Window: func(id platform.WindowID) (Window, bool) {
			r := r1
			if id == two {
				r = r2
			}
			return Window{Renderer: r, Display: func() paint.DisplayList { return nil }}, true
		},
		Dirty: func(id platform.WindowID) bool { return dirty[id] },
		Clean: func(id platform.WindowID) { dirty[id] = false },
		Handle: func(event platform.Event) (bool, error) {
			if event.Kind == platform.EventExpose {
				dirty[event.Window] = true
			}
			return event.Kind == platform.EventClose, nil
		},
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r1.presents != 1 || r2.presents != 2 {
		t.Fatalf("presents = %d/%d, want 1/2", r1.presents, r2.presents)
	}
}

func TestWindowsChoosesEarliestDeadlineAndBlocksWhenIdle(t *testing.T) {
	one, two := platform.WindowID(1), platform.WindowID(2)
	now := time.Unix(10, 0)
	source := &deadlineRecordingEvents{event: platform.Event{Kind: platform.EventClose}}
	r := &recordingRenderer{}
	loop := Windows{
		Clock: fixedClock{now: now}, Events: source,
		IDs: func() []platform.WindowID { return []platform.WindowID{one, two} },
		Window: func(id platform.WindowID) (Window, bool) {
			d := now.Add(5 * time.Second)
			if id == two {
				d = now.Add(time.Second)
			}
			return Window{Renderer: r, Display: func() paint.DisplayList { return nil }, Deadline: func(time.Time) *time.Time { return &d }}, true
		},
		Dirty: func(platform.WindowID) bool { return false }, Clean: func(platform.WindowID) {},
		Handle: func(event platform.Event) (bool, error) { return true, nil },
	}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.deadline == nil || !source.deadline.Equal(now.Add(time.Second)) {
		t.Fatalf("deadline = %v", source.deadline)
	}
}

type deadlineRecordingEvents struct {
	event    platform.Event
	deadline *time.Time
}

func (s *deadlineRecordingEvents) Wait(_ context.Context, d *time.Time) (platform.Event, error) {
	s.deadline = d
	return s.event, nil
}
func (*deadlineRecordingEvents) Poll() (platform.Event, bool, error) {
	return platform.Event{}, false, nil
}
func (*deadlineRecordingEvents) Wake() error { return nil }
