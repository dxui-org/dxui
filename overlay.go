package dxui

import (
	"fmt"
	"time"

	internalimage "github.com/dxui-org/dxui/internal/image"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	internaltext "github.com/dxui-org/dxui/internal/text"
	"github.com/dxui-org/dxui/internal/tree"
)

const defaultTooltipDelay = 500 * time.Millisecond

type overlayGeometry struct {
	Popup     paint.Rect
	Placement OverlayPlacement
	Flipped   bool
}

// placeOverlay is the shared Select/Popover/Tooltip positioning primitive.
// All inputs and outputs are final logical window coordinates.
func placeOverlay(anchor layout.Rect, desired layout.Size, window layout.Rect, placement OverlayPlacement, offset float32) overlayGeometry {
	if window.Width <= 0 || window.Height <= 0 || desired.Width <= 0 || desired.Height <= 0 {
		return overlayGeometry{Placement: placement}
	}
	actual := placement
	vertical := placement <= OverlayTopEnd
	preferAfter := placement <= OverlayBottomEnd || placement == OverlayRight
	before, after := float32(0), float32(0)
	if vertical {
		before = max(float32(0), anchor.Y-offset-window.Y)
		after = max(float32(0), window.Y+window.Height-(anchor.Y+anchor.Height+offset))
	} else {
		before = max(float32(0), anchor.X-offset-window.X)
		after = max(float32(0), window.X+window.Width-(anchor.X+anchor.Width+offset))
	}
	preferred, opposite := before, after
	if preferAfter {
		preferred, opposite = after, before
	}
	mainDesired := desired.Width
	if vertical {
		mainDesired = desired.Height
	}
	flipped := mainDesired > preferred && opposite > preferred
	if flipped {
		actual = oppositePlacement(placement)
		preferAfter = !preferAfter
		preferred = opposite
	}
	width := min(desired.Width, window.Width)
	height := min(desired.Height, window.Height)
	if vertical {
		height = min(height, preferred)
	} else {
		width = min(width, preferred)
	}
	x, y := anchor.X, anchor.Y
	switch actual {
	case OverlayBottomStart:
		y = anchor.Y + anchor.Height + offset
	case OverlayBottom:
		x, y = anchor.X+(anchor.Width-width)/2, anchor.Y+anchor.Height+offset
	case OverlayBottomEnd:
		x, y = anchor.X+anchor.Width-width, anchor.Y+anchor.Height+offset
	case OverlayTopStart:
		y = anchor.Y - offset - height
	case OverlayTop:
		x, y = anchor.X+(anchor.Width-width)/2, anchor.Y-offset-height
	case OverlayTopEnd:
		x, y = anchor.X+anchor.Width-width, anchor.Y-offset-height
	case OverlayLeft:
		x, y = anchor.X-offset-width, anchor.Y+(anchor.Height-height)/2
	case OverlayRight:
		x, y = anchor.X+anchor.Width+offset, anchor.Y+(anchor.Height-height)/2
	}
	x = min(max(x, window.X), window.X+window.Width-width)
	y = min(max(y, window.Y), window.Y+window.Height-height)
	return overlayGeometry{Popup: paint.Rect{X: x, Y: y, Width: width, Height: height}, Placement: actual, Flipped: flipped}
}

func oppositePlacement(value OverlayPlacement) OverlayPlacement {
	switch value {
	case OverlayBottomStart:
		return OverlayTopStart
	case OverlayBottom:
		return OverlayTop
	case OverlayBottomEnd:
		return OverlayTopEnd
	case OverlayTopStart:
		return OverlayBottomStart
	case OverlayTop:
		return OverlayBottom
	case OverlayTopEnd:
		return OverlayBottomEnd
	case OverlayLeft:
		return OverlayRight
	default:
		return OverlayLeft
	}
}

type componentOverlay struct {
	hostID        uint64
	view          View
	instance      *tree.Node
	geometry      *layout.Result
	anchor        layout.Rect
	placed        overlayGeometry
	interactive   bool
	desiredWidth  float32
	desiredHeight float32
}

func collectComponentOverlays(view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme) ([]componentOverlay, error) {
	result := make([]componentOverlay, 0, 2)
	if geometry == nil {
		return result, nil
	}
	window := geometry.Rect
	var visit func(View, *tree.Node, *layout.Result, float32, float32) error
	visit = func(current View, retained *tree.Node, box *layout.Result, tx, ty float32) error {
		if current.node == nil || retained == nil || box == nil {
			return nil
		}
		props, err := nodeProps(current)
		if err != nil {
			return err
		}
		visual, err := computeVisual(props, retained, current.node.kind, theme)
		if err != nil {
			return err
		}
		if visual.visibility == Hidden {
			return nil
		}
		isHost := current.node.kind == viewPopover || current.node.kind == viewTooltip
		var projected componentOverlay
		hasProjection := false
		if isHost && len(current.node.children) == 2 && len(retained.Children) == 2 && len(box.Children) == 2 {
			open, interactive := false, false
			placement, offsetValue := OverlayBottomStart, MetricValue{}
			gapToken := MetricComponentPopoverGap
			if current.node.kind == viewPopover {
				open, interactive = current.node.popover.Open, true
				placement, offsetValue = current.node.popover.Placement, current.node.popover.Offset
			} else {
				open = retained.State.TooltipOpen && !current.node.tooltip.Disabled
				placement, offsetValue = current.node.tooltip.Placement, current.node.tooltip.Offset
				gapToken = MetricComponentTooltipGap
			}
			if open {
				if !offsetValue.set {
					offsetValue = TokenMetric(gapToken)
				}
				offset, resolveErr := theme.metric(offsetValue)
				if resolveErr != nil {
					return resolveErr
				}
				anchor := translatedRect(box.Children[0].Rect, tx, ty)
				content := box.Children[1]
				placed := placeOverlay(anchor, layout.Size{Width: content.Rect.Width, Height: content.Rect.Height}, window, placement, offset)
				entry := componentOverlay{hostID: retained.ID, view: current.node.children[1], instance: retained.Children[1], geometry: content,
					anchor: anchor, placed: placed, interactive: interactive, desiredWidth: content.Rect.Width, desiredHeight: content.Rect.Height}
				result = append(result, entry)
				projected = entry
				hasProjection = true
			}
		}
		indices := make([]int, 0, len(current.node.children))
		if isHost {
			indices = append(indices, 0)
			if hasProjection {
				indices = append(indices, 1)
			}
		} else {
			for index := range current.node.children {
				indices = append(indices, index)
			}
		}
		for _, index := range indices {
			if index >= len(retained.Children) || index >= len(box.Children) {
				continue
			}
			nextX, nextY := tx, ty
			if isScrollView(current) {
				nextX -= retained.State.ScrollX
				nextY -= retained.State.ScrollY
			}
			if isHost && index == 1 && hasProjection {
				nextX = projected.placed.Popup.X - box.Children[1].Rect.X
				nextY = projected.placed.Popup.Y - box.Children[1].Rect.Y
			}
			if err := visit(current.node.children[index], retained.Children[index], box.Children[index], nextX, nextY); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(view, instance, geometry, 0, 0); err != nil {
		return nil, fmt.Errorf("dxui: overlay collection: %w", err)
	}
	return result, nil
}

func appendComponentOverlays(list *paint.DisplayList, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, engine *internaltext.Engine, images *internalimage.Cache, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse, editors map[uint64]editorDisplayState) error {
	overlays, err := collectComponentOverlays(view, instance, geometry, theme)
	if err != nil {
		return err
	}
	for _, overlay := range overlays {
		popup := overlay.placed.Popup
		if popup.Width <= 0 || popup.Height <= 0 {
			continue
		}
		clipped := popup.Width < overlay.desiredWidth || popup.Height < overlay.desiredHeight
		if clipped {
			*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: overlay.hostID, Rect: popup})
		}
		tx := popup.X - overlay.geometry.Rect.X
		ty := popup.Y - overlay.geometry.Rect.Y
		if err := appendDisplay(list, overlay.view, overlay.instance, overlay.geometry, theme, engine, images, scaleX, scaleY, remaining, reuse, editors, 1, paint.Rect{}, false, RGBAColor{}, false, tx, ty, "overlay"); err != nil {
			return err
		}
		if clipped {
			*list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: overlay.hostID})
		}
	}
	return nil
}

// appendWindowOverlays is the sole display entry for the App-owned overlay
// layer. Its fixed ordering is also used by input arbitration: Select first,
// followed by retained component overlays in source traversal order.
func appendWindowOverlays(list *paint.DisplayList, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, engine *internaltext.Engine, images *internalimage.Cache, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse, editors map[uint64]editorDisplayState) error {
	if err := appendSelectOverlays(list, view, instance, geometry, theme, engine, scaleX, scaleY, remaining, reuse); err != nil {
		return err
	}
	return appendComponentOverlays(list, view, instance, geometry, theme, engine, images, scaleX, scaleY, remaining, reuse, editors)
}
