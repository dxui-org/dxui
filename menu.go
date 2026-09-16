package dxui

import (
	"fmt"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	internaltext "github.com/dxui-org/dxui/internal/text"
	"github.com/dxui-org/dxui/internal/tree"
)

type menuMetrics struct {
	itemHeight, gap, paddingX, paddingY float32
}

type menuItemGeometry struct {
	item paint.Rect
	text paint.Rect
}

func resolveMenuMetrics(theme resolvedTheme) (menuMetrics, error) {
	values := []MetricValue{
		TokenMetric(MetricComponentMenuItemHeight), TokenMetric(MetricComponentMenuGap),
		TokenMetric(MetricComponentMenuPaddingX), TokenMetric(MetricComponentMenuPaddingY),
	}
	resolved := [4]float32{}
	for index, value := range values {
		metric, err := theme.metric(value)
		if err != nil {
			return menuMetrics{}, err
		}
		resolved[index] = metric
	}
	return menuMetrics{itemHeight: resolved[0], gap: resolved[1], paddingX: resolved[2], paddingY: resolved[3]}, nil
}

func measureMenuLabels(engine *internaltext.Engine, theme resolvedTheme, props MenuProps) ([]layout.Size, error) {
	if engine == nil {
		return nil, fmt.Errorf("missing text engine")
	}
	result := make([]layout.Size, len(props.Items))
	for index, item := range props.Items {
		request, err := resolvedTextRequest(textPropsFromCommon(props.common(), item.Label, TextNoWrap, 1), theme)
		if err != nil {
			return nil, err
		}
		laidOut, err := engine.Layout(request)
		if err != nil {
			return nil, err
		}
		result[index] = layout.Size{Width: laidOut.Metrics.Width, Height: laidOut.Metrics.Height}
	}
	return result, nil
}

func menuIntrinsic(engine *internaltext.Engine, theme resolvedTheme, props MenuProps) layout.IntrinsicMeasurer {
	return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
		metrics, err := resolveMenuMetrics(theme)
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		labels, err := measureMenuLabels(engine, theme, props)
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		var width, height float32
		for index, size := range labels {
			itemWidth := size.Width + 2*metrics.paddingX
			itemHeight := max(metrics.itemHeight, size.Height+2*metrics.paddingY)
			if props.Orientation == MenuHorizontal {
				if index != 0 {
					width += metrics.gap
				}
				width += itemWidth
				height = max(height, itemHeight)
			} else {
				if index != 0 {
					height += metrics.gap
				}
				height += itemHeight
				width = max(width, itemWidth)
			}
		}
		size := layout.Size{Width: width, Height: height}
		return layout.IntrinsicSize{Minimum: size, Preferred: size}, nil
	})
}

func menuGeometry(bounds paint.Rect, labels []layout.Size, metrics menuMetrics, orientation MenuOrientation) []menuItemGeometry {
	result := make([]menuItemGeometry, len(labels))
	x, y := bounds.X, bounds.Y
	for index, size := range labels {
		if index != 0 {
			if orientation == MenuHorizontal {
				x += metrics.gap
			} else {
				y += metrics.gap
			}
		}
		width := size.Width + 2*metrics.paddingX
		height := max(metrics.itemHeight, size.Height+2*metrics.paddingY)
		if orientation == MenuVertical {
			width = bounds.Width
		} else {
			height = bounds.Height
		}
		item := paint.Rect{X: x, Y: y, Width: max(0, width), Height: max(0, height)}
		result[index] = menuItemGeometry{
			item: item,
			text: paint.Rect{
				X: item.X + metrics.paddingX, Y: item.Y + metrics.paddingY,
				Width: max(0, item.Width-2*metrics.paddingX), Height: max(0, item.Height-2*metrics.paddingY),
			},
		}
		if orientation == MenuHorizontal {
			x += width
		} else {
			y += height
		}
	}
	return result
}

func appendMenuDisplay(list *paint.DisplayList, props *MenuProps, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, engine *internaltext.Engine, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse, visual computedVisual) error {
	metrics, err := resolveMenuMetrics(theme)
	if err != nil {
		return err
	}
	itemRadius, err := theme.metric(TokenMetric(MetricSemanticRadius))
	if err != nil {
		return err
	}
	itemRadii := paint.Radii{TopLeft: itemRadius, TopRight: itemRadius, BottomRight: itemRadius, BottomLeft: itemRadius}
	labels, err := measureMenuLabels(engine, theme, *props)
	if err != nil {
		return err
	}
	bounds := paintRect(geometry.Content)
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return nil
	}
	items := menuGeometry(bounds, labels, metrics, props.Orientation)
	baseText := visual.textColor
	if !visual.textColorSet {
		baseText, err = theme.color(TokenColor(ColorSemanticText))
		if err != nil {
			return err
		}
	}
	hover, err := theme.color(TokenColor(ColorSemanticMenuHover))
	if err != nil {
		return err
	}
	active, err := theme.color(TokenColor(ColorSemanticMenuActive))
	if err != nil {
		return err
	}
	selected, err := theme.color(TokenColor(ColorSemanticMenuSelected))
	if err != nil {
		return err
	}
	pressed, err := theme.color(TokenColor(ColorSemanticMenuPressed))
	if err != nil {
		return err
	}
	disabled, err := theme.color(TokenColor(ColorSemanticMenuDisabled))
	if err != nil {
		return err
	}
	*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: bounds})
	for index, item := range props.Items {
		box := items[index]
		switch {
		case index == instance.State.MenuPressed:
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: box.item, Radii: itemRadii, Color: paintColor(pressed)})
		case index == instance.State.MenuHover && !item.Disabled && !props.Disabled:
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: box.item, Radii: itemRadii, Color: paintColor(hover)})
		case item.Value == props.Value:
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: box.item, Radii: itemRadii, Color: paintColor(selected)})
		case index == instance.State.MenuActive && instance.State.Focused && !item.Disabled && !props.Disabled:
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: box.item, Radii: itemRadii, Color: paintColor(active)})
		}
		color := baseText
		if item.Disabled || props.Disabled {
			color = disabled
		}
		if props.common().Style.Text.Color.set {
			color = baseText
		}
		if err := appendSelectText(list, instance.ID, item.Label, props.common(), box.text, color, theme, engine, scaleX, scaleY, remaining, reuse); err != nil {
			return err
		}
	}
	*list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: instance.ID})
	return nil
}

func menuIndexForValue(items []MenuItem, value string) int {
	for index := range items {
		if items[index].Value == value {
			return index
		}
	}
	return -1
}

func nextEnabledMenuItem(items []MenuItem, current, direction int) int {
	for index := current + direction; index >= 0 && index < len(items); index += direction {
		if !items[index].Disabled {
			return index
		}
	}
	return current
}

func fallbackMenuItem(items []MenuItem, preferred int) int {
	if preferred < 0 {
		preferred = 0
	}
	for index := preferred; index < len(items); index++ {
		if !items[index].Disabled {
			return index
		}
	}
	for index := min(preferred-1, len(items)-1); index >= 0; index-- {
		if !items[index].Disabled {
			return index
		}
	}
	return -1
}

func syncMenuStates(view View, instance *tree.Node) bool {
	if view.node == nil || instance == nil {
		return false
	}
	changed := false
	if view.node.kind == viewMenu {
		props := view.node.menu
		itemsHash := paintStateHash(props.Items)
		itemsChanged := !instance.State.MenuInitialized || instance.State.MenuItemsHash != itemsHash
		active := menuIndexForValue(props.Items, instance.State.MenuActiveKey)
		if !instance.State.MenuInitialized || active < 0 || props.Items[active].Disabled {
			active = fallbackMenuItem(props.Items, instance.State.MenuActive)
		}
		activeKey := ""
		if active >= 0 && active < len(props.Items) {
			activeKey = props.Items[active].Value
		}
		hover, pressed := instance.State.MenuHover, instance.State.MenuPressed
		if itemsChanged || props.Disabled {
			hover, pressed = -1, -1
		}
		changed = instance.State.MenuActive != active || instance.State.MenuActiveKey != activeKey ||
			instance.State.MenuHover != hover || instance.State.MenuPressed != pressed ||
			instance.State.MenuItemsHash != itemsHash || !instance.State.MenuInitialized
		instance.State.MenuActive, instance.State.MenuActiveKey = active, activeKey
		instance.State.MenuHover, instance.State.MenuPressed = hover, pressed
		instance.State.MenuItemsHash = itemsHash
		instance.State.MenuInitialized = true
	}
	for index, child := range view.node.children {
		if index < len(instance.Children) {
			changed = syncMenuStates(child, instance.Children[index]) || changed
		}
	}
	return changed
}
