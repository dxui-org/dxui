package backend

import (
	"context"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
)

type fakeRuntime struct {
	events fakeEvents
	render fakeRenderer
}

func (fake *fakeRuntime) Events() platform.EventSource { return &fake.events }
func (fake *fakeRuntime) Renderer() renderer.Renderer  { return &fake.render }
func (*fakeRuntime) Diagnostics() Diagnostics          { return Diagnostics{} }
func (*fakeRuntime) Close() error                      { return nil }

type fakeEvents struct{}

func (*fakeEvents) Wait(context.Context, *time.Time) (platform.Event, error) {
	return platform.Event{Kind: platform.EventClose}, nil
}
func (*fakeEvents) Poll() (platform.Event, bool, error) { return platform.Event{}, false, nil }
func (*fakeEvents) Wake() error                         { return nil }

type fakeRenderer struct{}

func (*fakeRenderer) Render(paint.DisplayList) error { return nil }
func (*fakeRenderer) Present() error                 { return nil }

func TestRuntimeFakeSatisfiesBackendBoundary(t *testing.T) {
	// This compile assertion keeps fake/native tests tied to the same complete
	// lifecycle boundary rather than testing SDL-shaped values.
	var _ Runtime = (*fakeRuntime)(nil)
}
