// Package runtime coordinates the event-driven UI-thread commit loop.
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

// Loop contains injected ports so its behavior is testable without SDL.
type Loop struct {
	Clock    platform.Clock
	Events   platform.EventSource
	Renderer renderer.Renderer
	Display  func() paint.DisplayList
	Deadline func(time.Time) *time.Time
	Handle   func(platform.Event) (paintDirty bool, close bool, err error)
	// AfterFirstPresent runs once after the renderer's first successful
	// Present. Concrete renderers may use that Present to show a prepared
	// native window before returning.
	AfterFirstPresent func() (close bool, err error)
	Metrics           *Metrics
}

// Run paints the initial frame, then blocks on events or real deadlines. It
// never creates a polling ticker or a resident goroutine.
func (loop Loop) Run(ctx context.Context) error {
	if loop.Clock == nil || loop.Events == nil || loop.Renderer == nil || loop.Display == nil {
		return errors.New("runtime: incomplete loop ports")
	}
	dirty := true
	firstPresented := false
	var batchStarted time.Time
	for {
		if dirty {
			started := loop.Clock.Now()
			if err := loop.Renderer.Render(loop.Display().Sorted()); err != nil {
				return fmt.Errorf("runtime: render frame: %w", err)
			}
			if err := loop.Renderer.Present(); err != nil {
				return fmt.Errorf("runtime: render present: %w", err)
			}
			if loop.Metrics != nil {
				var latency *time.Duration
				if !batchStarted.IsZero() {
					value := loop.Clock.Now().Sub(batchStarted)
					latency = &value
				}
				loop.Metrics.recordFrame(loop.Clock.Now().Sub(started), latency)
			}
			if !firstPresented {
				firstPresented = true
				if loop.AfterFirstPresent != nil {
					closeLoop, err := loop.AfterFirstPresent()
					if err != nil {
						return fmt.Errorf("runtime: after first present: %w", err)
					}
					if closeLoop {
						return nil
					}
				}
			}
			batchStarted = time.Time{}
			dirty = false
		}
		var deadline *time.Time
		if loop.Deadline != nil {
			deadline = loop.Deadline(loop.Clock.Now())
		}
		event, err := loop.Events.Wait(ctx, deadline)
		if err != nil {
			return fmt.Errorf("runtime: event wait: %w", err)
		}
		batchStarted = loop.Clock.Now()

		var pendingViewport platform.Event
		hasPendingViewport := false
		process := func(current platform.Event) (bool, error) {
			if current.Kind == platform.EventExpose || current.Kind == platform.EventResize || current.Kind == platform.EventScale {
				dirty = true
			}
			if loop.Handle == nil {
				return current.Kind == platform.EventClose, nil
			}
			eventDirty, closeLoop, handleErr := loop.Handle(current)
			if handleErr != nil {
				return false, fmt.Errorf("runtime: event handle: %w", handleErr)
			}
			dirty = dirty || eventDirty
			return closeLoop, nil
		}
		flushViewport := func() (bool, error) {
			if !hasPendingViewport {
				return false, nil
			}
			hasPendingViewport = false
			return process(pendingViewport)
		}
		accept := func(current platform.Event) (bool, error) {
			if loop.Metrics != nil {
				loop.Metrics.recordEvent()
			}
			if current.Kind == platform.EventResize || current.Kind == platform.EventScale {
				pendingViewport = current
				hasPendingViewport = true
				return false, nil
			}
			if closeLoop, flushErr := flushViewport(); closeLoop || flushErr != nil {
				return closeLoop, flushErr
			}
			return process(current)
		}

		if closeLoop, acceptErr := accept(event); closeLoop || acceptErr != nil {
			return acceptErr
		}
		for {
			queued, ok, pollErr := loop.Events.Poll()
			if pollErr != nil {
				return fmt.Errorf("runtime: event poll: %w", pollErr)
			}
			if !ok {
				break
			}
			if closeLoop, acceptErr := accept(queued); closeLoop || acceptErr != nil {
				return acceptErr
			}
		}
		if closeLoop, flushErr := flushViewport(); closeLoop || flushErr != nil {
			return flushErr
		}
	}
}
