package dxui

import (
	"math"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/tree"
)

const (
	scrollWheelUnit             = float32(40)
	scrollbarIdleThicknessRatio = float32(0.5)
	scrollbarIdleOpacityRatio   = float32(0.55)
)

const (
	scrollbarAxisNone uint8 = iota
	scrollbarAxisHorizontal
	scrollbarAxisVertical
)

type scrollbarGeometry struct {
	Axis     uint8
	Track    paint.Rect
	Thumb    paint.Rect
	Maximum  float32
	Disabled bool
}

func scrollbarVisualThumb(thumb paint.Rect, axis uint8, expanded bool) paint.Rect {
	if expanded {
		return thumb
	}
	if axis == scrollbarAxisHorizontal {
		idleHeight := thumb.Height * scrollbarIdleThicknessRatio
		thumb.Y += (thumb.Height - idleHeight) / 2
		thumb.Height = idleHeight
		return thumb
	}
	idleWidth := thumb.Width * scrollbarIdleThicknessRatio
	thumb.X += (thumb.Width - idleWidth) / 2
	thumb.Width = idleWidth
	return thumb
}

func scrollAxisEnabled(axis ScrollAxis, horizontal bool) bool {
	if horizontal {
		return axis == ScrollHorizontal || axis == ScrollBoth
	}
	return axis == ScrollVertical || axis == ScrollBoth
}

func clampScrollOffset(axis ScrollAxis, offset Point, viewport layout.Rect, extent layout.Size) Point {
	maximumX := max(float32(0), extent.Width-viewport.Width)
	maximumY := max(float32(0), extent.Height-viewport.Height)
	if !scrollAxisEnabled(axis, true) {
		offset.X = 0
	}
	if !scrollAxisEnabled(axis, false) {
		offset.Y = 0
	}
	offset.X = min(max(float32(0), offset.X), maximumX)
	offset.Y = min(max(float32(0), offset.Y), maximumY)
	if !finite(offset.X) || !finite(offset.Y) {
		return Point{}
	}
	return offset
}

func syncScrollOffsets(view View, instance *tree.Node, geometry *layout.Result) bool {
	if view.node == nil || instance == nil || geometry == nil {
		return false
	}
	changed := false
	if isScrollView(view) {
		offset := Point{X: instance.State.ScrollX, Y: instance.State.ScrollY}
		if controlled, set := view.node.scroll.Offset.get(); set {
			offset = controlled
		}
		offset = clampScrollOffset(view.node.scroll.Axis, offset, geometry.Viewport, geometry.ContentExtent)
		if instance.State.ScrollX != offset.X || instance.State.ScrollY != offset.Y {
			instance.State.ScrollX, instance.State.ScrollY = offset.X, offset.Y
			changed = true
		}
	}
	for index, child := range view.node.children {
		if index < len(instance.Children) && index < len(geometry.Children) {
			changed = syncScrollOffsets(child, instance.Children[index], geometry.Children[index]) || changed
		}
	}
	return changed
}

func scrollbarsFor(viewport layout.Rect, extent layout.Size, offset Point, axis ScrollAxis, policy ScrollbarPolicy, thickness, minimum, inset float32) []scrollbarGeometry {
	if policy == ScrollbarHidden || viewport.Width <= 0 || viewport.Height <= 0 {
		return nil
	}
	showX := scrollAxisEnabled(axis, true) && (policy == ScrollbarAlways || extent.Width > viewport.Width)
	showY := scrollAxisEnabled(axis, false) && (policy == ScrollbarAlways || extent.Height > viewport.Height)
	result := make([]scrollbarGeometry, 0, 2)
	if showX {
		length := max(float32(0), viewport.Width-2*inset)
		if showY {
			length = max(float32(0), length-thickness-inset)
		}
		track := paint.Rect{X: viewport.X + inset, Y: viewport.Y + viewport.Height - thickness - inset, Width: length, Height: thickness}
		result = append(result, makeScrollbar(scrollbarAxisHorizontal, track, extent.Width, viewport.Width, offset.X, minimum))
	}
	if showY {
		length := max(float32(0), viewport.Height-2*inset)
		if showX {
			length = max(float32(0), length-thickness-inset)
		}
		track := paint.Rect{X: viewport.X + viewport.Width - thickness - inset, Y: viewport.Y + inset, Width: thickness, Height: length}
		result = append(result, makeScrollbar(scrollbarAxisVertical, track, extent.Height, viewport.Height, offset.Y, minimum))
	}
	return result
}

func makeScrollbar(axis uint8, track paint.Rect, content, viewport, offset, minimum float32) scrollbarGeometry {
	trackLength := track.Width
	if axis == scrollbarAxisVertical {
		trackLength = track.Height
	}
	maximum := max(float32(0), content-viewport)
	disabled := maximum == 0 || trackLength <= 0
	thumbLength := trackLength
	if content > 0 && !disabled {
		thumbLength = max(minimum, trackLength*viewport/content)
		thumbLength = min(trackLength, thumbLength)
	}
	position := float32(0)
	if maximum > 0 && trackLength > thumbLength {
		position = min(max(float32(0), offset), maximum) / maximum * (trackLength - thumbLength)
	}
	thumb := track
	if axis == scrollbarAxisHorizontal {
		thumb.X += position
		thumb.Width = thumbLength
	} else {
		thumb.Y += position
		thumb.Height = thumbLength
	}
	return scrollbarGeometry{Axis: axis, Track: track, Thumb: thumb, Maximum: maximum, Disabled: disabled}
}

func translatedRect(rect layout.Rect, x, y float32) layout.Rect {
	return layout.Rect{X: rect.X + x, Y: rect.Y + y, Width: rect.Width, Height: rect.Height}
}

func paintRect(rect layout.Rect) paint.Rect {
	return paint.Rect{X: rect.X, Y: rect.Y, Width: rect.Width, Height: rect.Height}
}

func rectContains(rect paint.Rect, x, y float32) bool {
	return x >= rect.X && y >= rect.Y && x < rect.X+rect.Width && y < rect.Y+rect.Height
}

func rectsDisjoint(left, right paint.Rect) bool {
	return left.Width <= 0 || left.Height <= 0 || right.Width <= 0 || right.Height <= 0 ||
		left.X+left.Width <= right.X || right.X+right.Width <= left.X ||
		left.Y+left.Height <= right.Y || right.Y+right.Height <= left.Y
}

func finitePoint(point Point) bool {
	return !math.IsNaN(float64(point.X)) && !math.IsInf(float64(point.X), 0) &&
		!math.IsNaN(float64(point.Y)) && !math.IsInf(float64(point.Y), 0)
}
