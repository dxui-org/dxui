package dxui

import (
	"fmt"
	"math"

	"github.com/dxui-org/dxui/icon"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	internaltext "github.com/dxui-org/dxui/internal/text"
	"github.com/dxui-org/dxui/internal/tree"
)

type selectOverlayGeometry struct {
	Popup      paint.Rect
	ItemHeight float32
	ScrollMax  float32
	Flipped    bool
}

type selectOverlay struct {
	view     View
	instance *tree.Node
	anchor   layout.Rect
	geometry selectOverlayGeometry
}

const (
	selectIndicatorSize = float32(10)
	selectIndicatorGap  = float32(8)
)

var selectIndicatorIcon = icon.ChevronDown()

func selectPopupGeometry(anchor layout.Rect, window layout.Rect, count int, theme resolvedTheme) (selectOverlayGeometry, error) {
	itemHeight, err := theme.metric(TokenMetric(MetricComponentSelectItemHeight))
	if err != nil {
		return selectOverlayGeometry{}, err
	}
	maximum, err := theme.metric(TokenMetric(MetricComponentSelectMaxHeight))
	if err != nil {
		return selectOverlayGeometry{}, err
	}
	gap, err := theme.metric(TokenMetric(MetricComponentSelectGap))
	if err != nil {
		return selectOverlayGeometry{}, err
	}
	if itemHeight <= 0 || maximum <= 0 || count <= 0 || window.Width <= 0 || window.Height <= 0 {
		return selectOverlayGeometry{ItemHeight: itemHeight}, nil
	}
	desired := min(float32(count)*itemHeight, maximum)
	placed := placeOverlay(anchor, layout.Size{Width: anchor.Width, Height: desired}, window, OverlayBottomStart, gap)
	height := placed.Popup.Height
	return selectOverlayGeometry{
		Popup:      placed.Popup,
		ItemHeight: itemHeight, ScrollMax: max(float32(0), float32(count)*itemHeight-height), Flipped: placed.Flipped,
	}, nil
}

func appendSelectAnchor(list *paint.DisplayList, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, engine *internaltext.Engine, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse, visual computedVisual) error {
	props := view.node.selectp
	label := props.Placeholder
	color := visual.textColor
	for _, option := range props.Options {
		if option.Value == props.Value {
			label = option.Label
			break
		}
	}
	if !visual.textColorSet {
		var err error
		color, err = theme.color(TokenColor(ColorSemanticText))
		if err != nil {
			return err
		}
	}
	content := paintRect(geometry.Content)
	indicatorWidth := min(selectIndicatorSize, content.Width)
	indicatorHeight := min(selectIndicatorSize, content.Height)
	indicatorBounds := layout.Rect{
		X:      content.X + content.Width - indicatorWidth,
		Y:      content.Y + (content.Height-indicatorHeight)/2,
		Width:  indicatorWidth,
		Height: indicatorHeight,
	}
	textBounds := content
	textBounds.Width = max(0, textBounds.Width-indicatorWidth-selectIndicatorGap)
	if err := appendSelectText(list, instance.ID, label, props.common(), textBounds, color, theme, engine, scaleX, scaleY, remaining, reuse); err != nil {
		return err
	}
	if indicatorWidth <= 0 || indicatorHeight <= 0 {
		return nil
	}
	mask, maskRect, err := rasterIconReuse(IconProps{Data: selectIndicatorIcon, Size: selectIndicatorSize}, indicatorBounds, theme, scaleX, scaleY, reuse)
	if err != nil {
		return fmt.Errorf("indicator: %w", err)
	}
	if err := accountIconMask(mask, remaining, reuse); err != nil {
		return fmt.Errorf("indicator source budget: %w", err)
	}
	*list = append(*list, paint.Command{Kind: paint.CommandDrawIcon, NodeID: instance.ID, Rect: maskRect, Text: mask, Color: paintColor(color)})
	return nil
}

func appendSelectOverlays(list *paint.DisplayList, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, engine *internaltext.Engine, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse) error {
	if !hasOpenSelect(view, instance) {
		return nil
	}
	overlays := make([]selectOverlay, 0, 1)
	window := geometry.Rect
	if err := collectSelectOverlays(&overlays, view, instance, geometry, window, theme, 0, 0); err != nil {
		return err
	}
	for _, overlay := range overlays {
		if overlay.geometry.Popup.Width <= 0 || overlay.geometry.Popup.Height <= 0 {
			continue
		}
		props := overlay.view.node.selectp
		popup := overlay.geometry.Popup
		surface, err := theme.color(TokenColor(ColorSemanticSurfaceHi))
		if err != nil {
			return err
		}
		border, err := theme.color(TokenColor(ColorSemanticBorder))
		if err != nil {
			return err
		}
		radius, err := theme.metric(TokenMetric(MetricSemanticRadius))
		if err != nil {
			return err
		}
		*list = append(*list,
			paint.Command{Kind: paint.CommandItem, NodeID: overlay.instance.ID, Bounds: popup, Visibility: true, Opacity: 1, Interactive: true, ZIndex: math.MaxInt},
			paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: overlay.instance.ID, Rect: popup, Radii: paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}, Color: paintColor(surface)},
			paint.Command{Kind: paint.CommandPushClip, NodeID: overlay.instance.ID, Rect: popup},
		)
		hover, err := theme.color(TokenColor(ColorSemanticSelectHover))
		if err != nil {
			return err
		}
		selected, err := theme.color(TokenColor(ColorSemanticSelectSelected))
		if err != nil {
			return err
		}
		disabled, err := theme.color(TokenColor(ColorSemanticSelectDisabled))
		if err != nil {
			return err
		}
		textColor, err := theme.color(TokenColor(ColorSemanticText))
		if err != nil {
			return err
		}
		first := max(0, int(overlay.instance.State.SelectScroll/overlay.geometry.ItemHeight))
		last := min(len(props.Options), int((overlay.instance.State.SelectScroll+popup.Height)/overlay.geometry.ItemHeight)+1)
		for index := first; index < last; index++ {
			option := props.Options[index]
			item := paint.Rect{X: popup.X, Y: popup.Y + float32(index)*overlay.geometry.ItemHeight - overlay.instance.State.SelectScroll, Width: popup.Width, Height: overlay.geometry.ItemHeight}
			background, radii := selectOptionBackground(item, popup, radius)
			if option.Value == props.Value {
				*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: overlay.instance.ID, Rect: background, Radii: radii, Color: paintColor(selected)})
			}
			if index == overlay.instance.State.SelectActive {
				*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: overlay.instance.ID, Rect: background, Radii: radii, Color: paintColor(hover)})
			}
			color := textColor
			if option.Disabled {
				color = disabled
			}
			textRect := item
			textRect.X += 10
			textRect.Width = max(0, textRect.Width-20)
			if err := appendSelectText(list, overlay.instance.ID, option.Label, props.common(), textRect, color, theme, engine, scaleX, scaleY, remaining, reuse); err != nil {
				return err
			}
		}
		thickness, metricErr := theme.metric(TokenMetric(MetricComponentScrollThickness))
		if metricErr != nil {
			return metricErr
		}
		minimum, metricErr := theme.metric(TokenMetric(MetricComponentScrollMinThumb))
		if metricErr != nil {
			return metricErr
		}
		inset, metricErr := theme.metric(TokenMetric(MetricComponentScrollInset))
		if metricErr != nil {
			return metricErr
		}
		bars := scrollbarsFor(
			layout.Rect{X: popup.X, Y: popup.Y, Width: popup.Width, Height: popup.Height},
			layout.Size{Width: popup.Width, Height: float32(len(props.Options)) * overlay.geometry.ItemHeight},
			Point{Y: overlay.instance.State.SelectScroll}, ScrollVertical, ScrollbarAuto,
			thickness, minimum, inset,
		)
		thumb, colorErr := theme.color(TokenColor(ColorSemanticScrollThumb))
		if colorErr != nil {
			return colorErr
		}
		thumb.A = uint8(float32(thumb.A) * scrollbarIdleOpacityRatio)
		track, colorErr := theme.color(TokenColor(ColorSemanticScrollTrack))
		if colorErr != nil {
			return colorErr
		}
		for _, bar := range bars {
			visualThumb := scrollbarVisualThumb(bar.Thumb, bar.Axis, false)
			visualThickness := visualThumb.Width
			if bar.Axis == scrollbarAxisHorizontal {
				visualThickness = visualThumb.Height
			}
			trackRadius := paint.Radii{TopLeft: thickness / 2, TopRight: thickness / 2, BottomRight: thickness / 2, BottomLeft: thickness / 2}
			thumbRadius := paint.Radii{TopLeft: visualThickness / 2, TopRight: visualThickness / 2, BottomRight: visualThickness / 2, BottomLeft: visualThickness / 2}
			*list = append(*list,
				paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: overlay.instance.ID, Rect: bar.Track, Radii: trackRadius, Color: paintColor(track)},
				paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: overlay.instance.ID, Rect: visualThumb, Radii: thumbRadius, Color: paintColor(thumb)},
			)
		}
		*list = append(*list,
			paint.Command{Kind: paint.CommandPopClip, NodeID: overlay.instance.ID},
			paint.Command{Kind: paint.CommandStrokeRoundedRect, NodeID: overlay.instance.ID, Rect: popup, Radii: paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}, Width: 1, Color: paintColor(border)},
		)
	}
	return nil
}

func selectOptionBackground(item, popup paint.Rect, radius float32) (paint.Rect, paint.Radii) {
	top := max(item.Y, popup.Y)
	bottom := min(item.Y+item.Height, popup.Y+popup.Height)
	background := paint.Rect{X: popup.X, Y: top, Width: popup.Width, Height: max(0, bottom-top)}
	var radii paint.Radii
	if top == popup.Y {
		radii.TopLeft = radius
		radii.TopRight = radius
	}
	if bottom == popup.Y+popup.Height {
		radii.BottomRight = radius
		radii.BottomLeft = radius
	}
	return background, radii
}

func hasOpenSelect(view View, instance *tree.Node) bool {
	if view.node == nil || instance == nil {
		return false
	}
	if view.node.kind == viewSelect && instance.State.SelectOpen {
		return true
	}
	for index, child := range view.node.children {
		if index < len(instance.Children) && hasOpenSelect(child, instance.Children[index]) {
			return true
		}
	}
	return false
}

func collectSelectOverlays(result *[]selectOverlay, view View, instance *tree.Node, geometry *layout.Result, window layout.Rect, theme resolvedTheme, translateX, translateY float32) error {
	if view.node == nil || instance == nil || geometry == nil {
		return nil
	}
	props, propsErr := nodeProps(view)
	if propsErr != nil {
		return propsErr
	}
	visual, visualErr := computeVisual(props, instance, view.node.kind, theme)
	if visualErr != nil {
		return visualErr
	}
	if visual.visibility == Hidden {
		return nil
	}
	world := translatedRect(geometry.Rect, translateX, translateY)
	if view.node.kind == viewSelect && instance.State.SelectOpen && len(view.node.selectp.Options) != 0 {
		popup, err := selectPopupGeometry(world, window, len(view.node.selectp.Options), theme)
		if err != nil {
			return err
		}
		*result = append(*result, selectOverlay{view: view, instance: instance, anchor: world, geometry: popup})
	}
	for index, child := range view.node.children {
		if index >= len(instance.Children) || index >= len(geometry.Children) {
			continue
		}
		x, y := translateX, translateY
		if isScrollView(view) {
			x -= instance.State.ScrollX
			y -= instance.State.ScrollY
		}
		if err := collectSelectOverlays(result, child, instance.Children[index], geometry.Children[index], window, theme, x, y); err != nil {
			return err
		}
	}
	return nil
}

func appendSelectText(list *paint.DisplayList, id uint64, value string, node viewProps, bounds paint.Rect, color RGBAColor, theme resolvedTheme, engine *internaltext.Engine, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse) error {
	if value == "" || bounds.Width <= 0 || bounds.Height <= 0 || engine == nil {
		return nil
	}
	request, err := resolvedTextRequest(textPropsFromCommon(node, value, TextNoWrap, 1), theme)
	if err != nil {
		return err
	}
	laidOut, err := engine.Layout(request)
	if err != nil {
		return err
	}
	bitmaps, err := engine.RasterizeLinesWithinReuse(laidOut, scaleX, scaleY, *remaining, reuseMaskFunc(reuse))
	if err != nil {
		return err
	}
	*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: id, Rect: bounds})
	for index := range bitmaps {
		bitmap := &bitmaps[index]
		*remaining -= bitmap.AccountedBytes
		mask := &paint.TextBitmap{Key: bitmap.Key, Width: bitmap.Width, Height: bitmap.Height, Pixels: bitmap.Alpha}
		y := bounds.Y + max(float32(0), (bounds.Height-bitmap.LogicalHeight)/2)
		*list = append(*list, paint.Command{Kind: paint.CommandDrawText, NodeID: id,
			Rect: paint.Rect{X: bounds.X + bitmap.OffsetX, Y: y, Width: bitmap.LogicalWidth, Height: bitmap.LogicalHeight}, Text: mask, Color: paintColor(color)})
	}
	*list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: id})
	return nil
}

func syncSelectStates(view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme) bool {
	changed := false
	var visit func(View, *tree.Node, *layout.Result, float32, float32)
	window := geometry.Rect
	visit = func(current View, retained *tree.Node, box *layout.Result, tx, ty float32) {
		if current.node == nil || retained == nil || box == nil {
			return
		}
		if current.node.kind == viewSelect {
			props := current.node.selectp
			selected := selectIndexForValue(props.Options, props.Value)
			active := selectIndexForKey(props.Options, retained.State.SelectActiveKey)
			if active < 0 || props.Options[active].Disabled {
				active = selected
				if active < 0 || props.Options[active].Disabled {
					active = nextEnabledOption(props.Options, -1, 1)
				}
			}
			visual, _ := computeVisual(props.common(), retained, current.node.kind, theme)
			if props.Disabled || visual.visibility == Hidden || len(props.Options) == 0 || retained.State.SelectValueValid && selected < 0 {
				retained.State.SelectOpen = false
			}
			retained.State.SelectValue = props.Value
			retained.State.SelectValueValid = selected >= 0
			retained.State.SelectActive = active
			retained.State.SelectActiveKey = ""
			if active >= 0 {
				retained.State.SelectActiveKey = string(props.Options[active].Value)
			}
			popup, _ := selectPopupGeometry(translatedRect(box.Rect, tx, ty), window, len(props.Options), theme)
			oldScroll := retained.State.SelectScroll
			retained.State.SelectScroll = min(max(retained.State.SelectScroll, 0), popup.ScrollMax)
			changed = changed || oldScroll != retained.State.SelectScroll
		}
		for index, child := range current.node.children {
			if index >= len(retained.Children) || index >= len(box.Children) {
				continue
			}
			x, y := tx, ty
			if isScrollView(current) {
				x -= retained.State.ScrollX
				y -= retained.State.ScrollY
			}
			visit(child, retained.Children[index], box.Children[index], x, y)
		}
	}
	visit(view, instance, geometry, 0, 0)
	return changed
}

func selectIndexForValue(options []SelectOption, value string) int {
	for index := range options {
		if options[index].Value == value {
			return index
		}
	}
	return -1
}

func selectIndexForKey(options []SelectOption, key string) int {
	for index := range options {
		if string(options[index].Value) == key {
			return index
		}
	}
	return -1
}

func nextEnabledOption(options []SelectOption, current, direction int) int {
	if len(options) == 0 {
		return -1
	}
	for step := 0; step < len(options); step++ {
		current += direction
		if current < 0 {
			current = len(options) - 1
		} else if current >= len(options) {
			current = 0
		}
		if !options[current].Disabled {
			return current
		}
	}
	return -1
}

func ensureSelectActiveVisible(state *tree.State, popup selectOverlayGeometry) {
	if state.SelectActive < 0 || popup.ItemHeight <= 0 {
		return
	}
	top := float32(state.SelectActive) * popup.ItemHeight
	bottom := top + popup.ItemHeight
	if top < state.SelectScroll {
		state.SelectScroll = top
	} else if bottom > state.SelectScroll+popup.Popup.Height {
		state.SelectScroll = bottom - popup.Popup.Height
	}
	state.SelectScroll = min(max(state.SelectScroll, 0), popup.ScrollMax)
}

func finiteSelectPoint(x, y float32) bool {
	return !math.IsNaN(float64(x)) && !math.IsNaN(float64(y)) && !math.IsInf(float64(x), 0) && !math.IsInf(float64(y), 0)
}

func selectError(path string, err error) error { return fmt.Errorf("dxui: %s: %w", path, err) }
