package input

import (
	"fmt"
	"testing"

	"github.com/dxui-org/dxui/internal/platform"
)

func pointer(kind platform.EventKind, x, y float32) platform.Event {
	return platform.Event{Kind: kind, Pointer: platform.PointerEvent{X: x, Y: y, Button: platform.MouseButtonPrimary}}
}

func key(kind platform.EventKind, value platform.Key, shift, repeat bool) platform.Event {
	return platform.Event{Kind: kind, Key: platform.KeyEvent{Key: value, Modifiers: platform.Modifiers{Shift: shift}, Repeat: repeat}}
}

func TestHitTestUsesReversePaintOrderClipAndHalfOpenEdges(t *testing.T) {
	controller := Controller{}
	controller.Reconcile([]Node{
		{ID: 1, Bounds: Rect{Width: 100, Height: 100}, Interactive: true, Focusable: true, Order: 0},
		{ID: 2, Bounds: Rect{Width: 100, Height: 100}, Clip: Rect{X: 10, Y: 10, Width: 20, Height: 20}, ClipSet: true, Interactive: true, Focusable: true, Order: 1},
	})
	for _, test := range []struct {
		x, y float32
		want uint64
	}{{10, 10, 2}, {29.999, 29.999, 2}, {30, 20, 1}, {99.999, 99.999, 1}, {100, 50, 0}} {
		if got := controller.Handle(pointer(platform.EventMouseMove, test.x, test.y)).Target; got != test.want {
			t.Fatalf("hit (%v,%v) = %d, want %d", test.x, test.y, got, test.want)
		}
	}
}

func TestDisabledAndExcludedNodesDoNotWinHitTest(t *testing.T) {
	controller := Controller{}
	controller.Reconcile([]Node{
		{ID: 1, Bounds: Rect{Width: 10, Height: 10}, Interactive: true},
		{ID: 2, Bounds: Rect{Width: 10, Height: 10}, Interactive: true, Disabled: true},
		// Hidden, transparent, and PointerNone entries are intentionally absent
		// from snapshots; this non-interactive entry verifies the same outcome.
		{ID: 3, Bounds: Rect{Width: 10, Height: 10}},
	})
	if got := controller.Handle(pointer(platform.EventMouseMove, 5, 5)).Target; got != 1 {
		t.Fatalf("hit = %d, want enabled node 1", got)
	}
}

func TestPressDragReleaseAndNoMeaninglessSameNodeMove(t *testing.T) {
	controller := Controller{}
	controller.Reconcile([]Node{{ID: 1, Bounds: Rect{Width: 20, Height: 20}, Interactive: true, Focusable: true, Enter: true, Space: true}})
	if result := controller.Handle(pointer(platform.EventMouseMove, 2, 2)); !result.Changed {
		t.Fatal("first hover did not change state")
	}
	if result := controller.Handle(pointer(platform.EventMouseMove, 3, 3)); result.Changed {
		t.Fatal("move within the same target caused repaint")
	}
	controller.Handle(pointer(platform.EventMouseDown, 3, 3))
	if _, pressed, focused, focusVisible := controller.State(1); !pressed || !focused || focusVisible {
		t.Fatalf("pressed/focused/focus-visible = %v/%v/%v", pressed, focused, focusVisible)
	}
	controller.Handle(pointer(platform.EventMouseMove, 30, 3))
	if _, pressed, _, _ := controller.State(1); pressed {
		t.Fatal("drag outside retained pressed visual")
	}
	if result := controller.Handle(pointer(platform.EventMouseUp, 30, 3)); result.Activate != 0 {
		t.Fatalf("outside release activated %d", result.Activate)
	}
	controller.Handle(pointer(platform.EventMouseDown, 3, 3))
	if result := controller.Handle(pointer(platform.EventMouseUp, 3, 3)); result.Activate != 1 {
		t.Fatalf("inside release activation = %d", result.Activate)
	}
}

func TestWindowFocusLossAndUnmountCancelTransientState(t *testing.T) {
	controller := Controller{}
	controller.Reconcile([]Node{{ID: 1, Bounds: Rect{Width: 20, Height: 20}, Interactive: true, Focusable: true, Space: true}})
	controller.Handle(pointer(platform.EventMouseDown, 2, 2))
	controller.Handle(platform.Event{Kind: platform.EventWindowFocusLost})
	if _, pressed, focused, _ := controller.State(1); pressed || !focused {
		t.Fatalf("after focus loss pressed/focused = %v/%v", pressed, focused)
	}
	if result := controller.Handle(pointer(platform.EventMouseUp, 2, 2)); result.Activate != 0 {
		t.Fatal("focus-loss-cancelled capture activated")
	}
	controller.Handle(pointer(platform.EventMouseDown, 2, 2))
	controller.Reconcile(nil)
	if controller.Focused() != 0 {
		t.Fatal("removed only focus target was retained")
	}
}

func TestFocusVisibleTracksPointerAndKeyboardModalityWithoutDroppingFocus(t *testing.T) {
	controller := Controller{}
	controller.Reconcile([]Node{{ID: 1, Bounds: Rect{Width: 20, Height: 20}, Interactive: true, Focusable: true, Enter: true, Space: true}})

	controller.Handle(pointer(platform.EventMouseDown, 2, 2))
	controller.Handle(pointer(platform.EventMouseUp, 2, 2))
	if _, _, focused, visible := controller.State(1); !focused || visible {
		t.Fatalf("mouse focus/visible = %v/%v, want true/false", focused, visible)
	}

	if result := controller.Handle(key(platform.EventKeyDown, platform.KeyTab, false, false)); !result.Changed {
		t.Fatal("Tab on the sole pointer-focused node did not expose keyboard focus")
	}
	if _, _, focused, visible := controller.State(1); !focused || !visible {
		t.Fatalf("keyboard focus/visible = %v/%v, want true/true", focused, visible)
	}

	controller.Handle(pointer(platform.EventMouseDown, 2, 2))
	if _, pressed, focused, visible := controller.State(1); !pressed || !focused || visible {
		t.Fatalf("keyboard-to-pointer pressed/focused/visible = %v/%v/%v", pressed, focused, visible)
	}
	controller.Handle(pointer(platform.EventMouseUp, 2, 2))
	controller.Handle(key(platform.EventKeyDown, platform.KeySpace, false, false))
	if _, pressed, focused, visible := controller.State(1); !pressed || !focused || !visible {
		t.Fatalf("pointer-to-keyboard pressed/focused/visible = %v/%v/%v", pressed, focused, visible)
	}
}

func TestTabTraversalRetentionDeletionAndNoFocusableNodes(t *testing.T) {
	nodes := []Node{
		{ID: 1, Focusable: true, Order: 0},
		{ID: 2, Focusable: true, Disabled: true, Order: 1},
		{ID: 3, Focusable: true, Order: 2},
	}
	controller := Controller{}
	controller.Reconcile(nodes)
	controller.Handle(key(platform.EventKeyDown, platform.KeyTab, false, false))
	if controller.Focused() != 1 {
		t.Fatalf("first Tab focus = %d", controller.Focused())
	}
	controller.Handle(key(platform.EventKeyDown, platform.KeyTab, false, false))
	if controller.Focused() != 3 {
		t.Fatalf("second Tab focus = %d", controller.Focused())
	}
	controller.Handle(key(platform.EventKeyDown, platform.KeyTab, true, false))
	if controller.Focused() != 1 {
		t.Fatalf("Shift+Tab focus = %d", controller.Focused())
	}
	controller.Reconcile([]Node{{ID: 3, Focusable: true, Order: 2}, {ID: 1, Focusable: true, Order: 0}})
	if controller.Focused() != 1 {
		t.Fatal("identity focus did not survive reorder")
	}
	controller.Reconcile([]Node{{ID: 3, Focusable: true, Order: 2}})
	if controller.Focused() != 3 {
		t.Fatalf("deleted focus did not transfer deterministically: %d", controller.Focused())
	}
	controller.Reconcile(nil)
	controller.Handle(key(platform.EventKeyDown, platform.KeyTab, false, false))
	if controller.Focused() != 0 {
		t.Fatal("empty traversal produced focus")
	}
}

func TestKeyboardActivationIgnoresAutoRepeat(t *testing.T) {
	controller := Controller{}
	controller.Reconcile([]Node{{ID: 1, Focusable: true, Enter: true, Space: true, Order: 0}})
	controller.Handle(key(platform.EventKeyDown, platform.KeyTab, false, false))
	if result := controller.Handle(key(platform.EventKeyDown, platform.KeySpace, false, true)); result.Changed || result.Activate != 0 {
		t.Fatal("auto-repeat changed semantic state")
	}
	controller.Handle(key(platform.EventKeyDown, platform.KeySpace, false, false))
	if result := controller.Handle(key(platform.EventKeyUp, platform.KeyEnter, false, false)); result.Activate != 0 {
		t.Fatal("mismatched key up activated")
	}
	if result := controller.Handle(key(platform.EventKeyUp, platform.KeySpace, false, false)); result.Activate != 1 {
		t.Fatalf("Space activation = %d", result.Activate)
	}
	controller.Handle(key(platform.EventKeyDown, platform.KeyEnter, false, false))
	if result := controller.Handle(key(platform.EventKeyUp, platform.KeyEnter, false, false)); result.Activate != 1 {
		t.Fatalf("Enter activation = %d", result.Activate)
	}
}

func BenchmarkMouseMove1000Nodes(b *testing.B) {
	nodes := make([]Node, 1000)
	for index := range nodes {
		nodes[index] = Node{ID: uint64(index + 1), Bounds: Rect{X: float32(index % 40 * 12), Y: float32(index / 40 * 12), Width: 10, Height: 10}, Interactive: true, Order: index}
	}
	controller := Controller{}
	controller.Reconcile(nodes)
	events := make([]platform.Event, 64)
	for index := range events {
		events[index] = pointer(platform.EventMouseMove, float32(index%40*12+1), float32(index/40*12+1))
	}
	b.ReportAllocs()
	for iteration := 0; iteration < b.N; iteration++ {
		result := controller.Handle(events[iteration%len(events)])
		if result.Target == 0 {
			b.Fatal(fmt.Errorf("benchmark point missed"))
		}
	}
}
