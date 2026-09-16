package input

import (
	"math"

	"github.com/dxui-org/dxui/internal/platform"
)

// Rect is a half-open logical hit rectangle.
type Rect struct{ X, Y, Width, Height float32 }

// Node is one immutable interaction snapshot entry. Entries are supplied in
// final paint order; Order is the stable source/tab order.
type Node struct {
	ID          uint64
	Bounds      Rect
	Clip        Rect
	ClipSet     bool
	Order       int
	Interactive bool
	Focusable   bool
	Disabled    bool
	Enter       bool
	Space       bool
}

// Result summarizes state changes and at most one semantic activation.
type Result struct {
	Changed  bool
	Target   uint64
	Activate uint64
}

// Controller owns the single hover, pointer-capture, focus, and keyboard
// press state for one window. It is UI-thread confined.
type Controller struct {
	nodes           []Node
	hover           uint64
	capture         uint64
	captureInside   bool
	focus           uint64
	focusVisible    bool
	keyboardPressed uint64
	keyboardKey     platform.Key
	pointerX        float32
	pointerY        float32
	pointerKnown    bool
}

// Reconcile replaces the hit/focus snapshot while preserving live identity.
// If focus disappears or becomes ineligible, it moves to the next source
// position, wrapping to the previous last eligible node, or clears.
func (c *Controller) Reconcile(nodes []Node) Result {
	oldFocusOrder := -1
	if old, ok := c.node(c.focus); ok {
		oldFocusOrder = old.Order
	}
	c.nodes = append(c.nodes[:0], nodes...)
	changed := false
	if !c.interactive(c.capture) {
		changed = changed || c.capture != 0
		c.capture = 0
		c.captureInside = false
	} else {
		inside := c.pointerKnown && c.hit(c.pointerX, c.pointerY) == c.capture
		changed = changed || inside != c.captureInside
		c.captureInside = inside
	}
	if c.focus != 0 && !c.focusable(c.focus) {
		next := c.focusAtOrAfter(oldFocusOrder)
		changed = changed || next != c.focus
		c.focus = next
	}
	if c.keyboardPressed != c.focus || !c.activatable(c.keyboardPressed) {
		changed = changed || c.keyboardPressed != 0
		c.keyboardPressed = 0
		c.keyboardKey = platform.KeyOther
	}
	nextHover := uint64(0)
	if c.pointerKnown {
		nextHover = c.hit(c.pointerX, c.pointerY)
	}
	changed = changed || nextHover != c.hover
	c.hover = nextHover
	return Result{Changed: changed}
}

// Handle applies one normalized platform event. Semantic key auto-repeat is
// ignored: Tab moves once per physical press and Enter/Space activate once on
// release, allowing one shared policy for Button/Checkbox/Toggle.
func (c *Controller) Handle(event platform.Event) Result {
	result := Result{}
	switch event.Kind {
	case platform.EventMouseMove:
		if !finitePoint(event.Pointer.X, event.Pointer.Y) {
			return result
		}
		c.pointerX, c.pointerY, c.pointerKnown = event.Pointer.X, event.Pointer.Y, true
		target := c.hit(c.pointerX, c.pointerY)
		result.Target = target
		result.Changed = c.setHover(target)
		if c.capture != 0 {
			inside := target == c.capture
			result.Changed = result.Changed || inside != c.captureInside
			c.captureInside = inside
		}
	case platform.EventMouseDown:
		if event.Pointer.Button != platform.MouseButtonPrimary || !finitePoint(event.Pointer.X, event.Pointer.Y) {
			return result
		}
		c.pointerX, c.pointerY, c.pointerKnown = event.Pointer.X, event.Pointer.Y, true
		target := c.hit(c.pointerX, c.pointerY)
		result.Target = target
		result.Changed = c.setHover(target)
		if target != c.capture {
			result.Changed = true
		}
		c.capture = target
		c.captureInside = target != 0
		if c.focusVisible {
			c.focusVisible = false
			result.Changed = true
		}
		if c.focus != target {
			c.focus = 0
			if c.focusable(target) {
				c.focus = target
			}
			c.keyboardPressed = 0
			c.keyboardKey = platform.KeyOther
			result.Changed = true
		}
	case platform.EventMouseUp:
		if event.Pointer.Button != platform.MouseButtonPrimary || !finitePoint(event.Pointer.X, event.Pointer.Y) {
			return result
		}
		c.pointerX, c.pointerY, c.pointerKnown = event.Pointer.X, event.Pointer.Y, true
		target := c.hit(c.pointerX, c.pointerY)
		result.Target = target
		result.Changed = c.setHover(target)
		captured := c.capture
		if captured != 0 {
			result.Changed = true
		}
		c.capture = 0
		c.captureInside = false
		if captured != 0 && target == captured && c.interactive(captured) {
			result.Activate = captured
		}
	case platform.EventMouseWheel:
		if finitePoint(event.Pointer.X, event.Pointer.Y) {
			result.Target = c.hit(event.Pointer.X, event.Pointer.Y)
		}
	case platform.EventWindowMouseLeave:
		c.pointerKnown = false
		result.Changed = c.setHover(0)
		if c.captureInside {
			c.captureInside = false
			result.Changed = true
		}
	case platform.EventWindowFocusLost:
		result.Changed = c.capture != 0 || c.keyboardPressed != 0
		c.capture, c.keyboardPressed, c.captureInside = 0, 0, false
		c.keyboardKey = platform.KeyOther
	case platform.EventKeyDown:
		if event.Key.Repeat {
			return result
		}
		if c.focus != 0 && !c.focusVisible {
			c.focusVisible = true
			result.Changed = true
		}
		switch event.Key.Key {
		case platform.KeyTab:
			next := c.nextFocus(event.Key.Modifiers.Shift)
			if next != c.focus {
				c.focus, c.keyboardPressed = next, 0
				c.keyboardKey = platform.KeyOther
				result.Changed = true
			}
			if next != 0 && !c.focusVisible {
				c.focusVisible = true
				result.Changed = true
			}
		case platform.KeyEnter:
			if node, ok := c.node(c.focus); ok && node.Enter {
				c.keyboardPressed = c.focus
				c.keyboardKey = platform.KeyEnter
				result.Changed = true
			}
		case platform.KeySpace:
			if node, ok := c.node(c.focus); ok && node.Space {
				c.keyboardPressed = c.focus
				c.keyboardKey = platform.KeySpace
				result.Changed = true
			}
		}
	case platform.EventKeyUp:
		if event.Key.Key != platform.KeyEnter && event.Key.Key != platform.KeySpace {
			return result
		}
		pressed := c.keyboardPressed
		if pressed != 0 && event.Key.Key != c.keyboardKey {
			return result
		}
		if pressed != 0 {
			c.keyboardPressed = 0
			c.keyboardKey = platform.KeyOther
			result.Changed = true
		}
		if pressed == c.focus && c.activatable(pressed) {
			result.Activate = pressed
		}
	}
	return result
}

// State returns the controller-owned interaction flags for an identity.
func (c *Controller) State(id uint64) (hovered, pressed, focused, focusVisible bool) {
	hovered = id != 0 && c.hover == id
	pressed = id != 0 && (c.keyboardPressed == id || c.capture == id && c.captureInside)
	focused = id != 0 && c.focus == id
	focusVisible = focused && c.focusVisible
	return
}

func (c *Controller) Focused() uint64 { return c.focus }

// Focus moves focus to a live eligible identity. It is used by typed overlay
// dismissal to restore the trigger without exposing a public focus API.
func (c *Controller) Focus(id uint64, visible bool) bool {
	if id != 0 && !c.focusable(id) {
		return false
	}
	changed := c.focus != id || c.focusVisible != (id != 0 && visible) || c.keyboardPressed != 0
	c.focus = id
	c.focusVisible = id != 0 && visible
	c.keyboardPressed = 0
	c.keyboardKey = platform.KeyOther
	return changed
}

func (c *Controller) hit(x, y float32) uint64 {
	for index := len(c.nodes) - 1; index >= 0; index-- {
		node := c.nodes[index]
		if node.Disabled || !node.Interactive || !contains(node.Bounds, x, y) {
			continue
		}
		if !node.ClipSet || contains(node.Clip, x, y) {
			return node.ID
		}
	}
	return 0
}

func (c *Controller) setHover(id uint64) bool {
	if c.hover == id {
		return false
	}
	c.hover = id
	return true
}

func (c *Controller) node(id uint64) (Node, bool) {
	if id == 0 {
		return Node{}, false
	}
	for index := range c.nodes {
		if c.nodes[index].ID == id {
			return c.nodes[index], true
		}
	}
	return Node{}, false
}

func (c *Controller) interactive(id uint64) bool {
	node, ok := c.node(id)
	return ok && node.Interactive && !node.Disabled
}

func (c *Controller) focusable(id uint64) bool {
	node, ok := c.node(id)
	return ok && node.Focusable && !node.Disabled
}

func (c *Controller) activatable(id uint64) bool {
	node, ok := c.node(id)
	return ok && !node.Disabled && (node.Enter || node.Space)
}

func (c *Controller) focusAtOrAfter(order int) uint64 {
	bestID, bestOrder := uint64(0), int(^uint(0)>>1)
	lastID, lastOrder := uint64(0), -1
	for _, node := range c.nodes {
		if !node.Focusable || node.Disabled {
			continue
		}
		if node.Order >= order && node.Order < bestOrder {
			bestID, bestOrder = node.ID, node.Order
		}
		if node.Order > lastOrder {
			lastID, lastOrder = node.ID, node.Order
		}
	}
	if bestID != 0 {
		return bestID
	}
	return lastID
}

func (c *Controller) nextFocus(reverse bool) uint64 {
	eligible := make([]Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		if node.Focusable && !node.Disabled {
			eligible = append(eligible, node)
		}
	}
	for index := 1; index < len(eligible); index++ {
		value := eligible[index]
		position := index
		for position > 0 && value.Order < eligible[position-1].Order {
			eligible[position] = eligible[position-1]
			position--
		}
		eligible[position] = value
	}
	if len(eligible) == 0 {
		return 0
	}
	current := -1
	for index := range eligible {
		if eligible[index].ID == c.focus {
			current = index
			break
		}
	}
	if reverse {
		if current <= 0 {
			return eligible[len(eligible)-1].ID
		}
		return eligible[current-1].ID
	}
	return eligible[(current+1)%len(eligible)].ID
}

func contains(rect Rect, x, y float32) bool {
	return rect.Width > 0 && rect.Height > 0 && x >= rect.X && y >= rect.Y && x < rect.X+rect.Width && y < rect.Y+rect.Height
}

func finitePoint(x, y float32) bool {
	return !math.IsNaN(float64(x)) && !math.IsNaN(float64(y)) && !math.IsInf(float64(x), 0) && !math.IsInf(float64(y), 0)
}
