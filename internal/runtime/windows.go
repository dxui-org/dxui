package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
)

// Window is one independently scheduled surface in a Windows loop.
type Window struct {
	Renderer renderer.Renderer
	Display  func() paint.DisplayList
	Deadline func(time.Time) *time.Time
	Shown    func() error
}

// Windows coordinates independent surfaces through one blocking event source.
// The callbacks own window registration and return only live window IDs.
type Windows struct {
	Clock   platform.Clock
	Events  platform.EventSource
	IDs     func() []platform.WindowID
	Window  func(platform.WindowID) (Window, bool)
	Dirty   func(platform.WindowID) bool
	Clean   func(platform.WindowID)
	Handle  func(platform.Event) (close bool, err error)
	Metrics *Metrics
}

// Run presents dirty windows, waits until the earliest per-window deadline,
// and routes each event by its logical window identity.
func (loop Windows) Run(ctx context.Context) error {
	if loop.Clock == nil || loop.Events == nil || loop.IDs == nil || loop.Window == nil || loop.Dirty == nil || loop.Clean == nil {
		return errors.New("runtime: incomplete multi-window loop ports")
	}
	shown := make(map[platform.WindowID]bool)
	for {
		ids := loop.IDs()
		if len(ids) == 0 {
			return nil
		}
		for _, id := range ids {
			window, ok := loop.Window(id)
			if !ok || !loop.Dirty(id) {
				continue
			}
			if err := window.Renderer.Render(window.Display().Sorted()); err != nil {
				return fmt.Errorf("runtime: window %d render: %w", id, err)
			}
			if err := window.Renderer.Present(); err != nil {
				return fmt.Errorf("runtime: window %d present: %w", id, err)
			}
			loop.Clean(id)
			if !shown[id] {
				shown[id] = true
				if window.Shown != nil {
					if err := window.Shown(); err != nil {
						return fmt.Errorf("runtime: window %d shown: %w", id, err)
					}
				}
			}
		}
		now := loop.Clock.Now()
		var earliest *time.Time
		for _, id := range loop.IDs() {
			window, ok := loop.Window(id)
			if !ok || window.Deadline == nil {
				continue
			}
			if deadline := window.Deadline(now); deadline != nil && (earliest == nil || deadline.Before(*earliest)) {
				value := *deadline
				earliest = &value
			}
		}
		event, err := loop.Events.Wait(ctx, earliest)
		if err != nil {
			return fmt.Errorf("runtime: event wait: %w", err)
		}
		batch := []platform.Event{event}
		for {
			event, ok, err := loop.Events.Poll()
			if err != nil {
				return fmt.Errorf("runtime: event poll: %w", err)
			}
			if !ok {
				break
			}
			batch = append(batch, event)
		}
		// Keep only the final consecutive resize/scale notification for each
		// window in this drained batch.
		coalesced := batch[:0]
		for _, current := range batch {
			if current.Kind == platform.EventResize || current.Kind == platform.EventScale {
				if n := len(coalesced); n > 0 && coalesced[n-1].Window == current.Window && (coalesced[n-1].Kind == platform.EventResize || coalesced[n-1].Kind == platform.EventScale) {
					coalesced[n-1] = current
					continue
				}
			}
			coalesced = append(coalesced, current)
		}
		for _, current := range coalesced {
			if loop.Handle == nil {
				continue
			}
			closeLoop, err := loop.Handle(current)
			if err != nil || closeLoop {
				return err
			}
		}
	}
}
