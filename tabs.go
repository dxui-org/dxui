package dxui

import (
	"fmt"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	internaltext "github.com/dxui-org/dxui/internal/text"
	"github.com/dxui-org/dxui/internal/tree"
)

type tabsMetrics struct {
	height, gap, paddingX, paddingY float32
	indicator, indicatorInset       float32
}

type tabGeometry struct {
	item paint.Rect
	text paint.Rect
}

func resolveTabsMetrics(theme resolvedTheme) (tabsMetrics, error) {
	values := []MetricValue{
		TokenMetric(MetricComponentTabsHeight), TokenMetric(MetricComponentTabsGap),
		TokenMetric(MetricComponentTabsPaddingX), TokenMetric(MetricComponentTabsPaddingY),
		TokenMetric(MetricComponentTabsIndicator), TokenMetric(MetricComponentTabsIndicatorInset),
	}
	resolved := [6]float32{}
	for index, value := range values {
		metric, err := theme.metric(value)
		if err != nil {
			return tabsMetrics{}, err
		}
		resolved[index] = metric
	}
	return tabsMetrics{
		height: resolved[0], gap: resolved[1], paddingX: resolved[2], paddingY: resolved[3],
		indicator: resolved[4], indicatorInset: resolved[5],
	}, nil
}

func measureTabLabels(engine *internaltext.Engine, theme resolvedTheme, props TabsProps) ([]layout.Size, error) {
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

func tabsIntrinsic(engine *internaltext.Engine, theme resolvedTheme, props TabsProps) layout.IntrinsicMeasurer {
	return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
		metrics, err := resolveTabsMetrics(theme)
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		labels, err := measureTabLabels(engine, theme, props)
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		var width, textHeight float32
		for index, size := range labels {
			if index != 0 {
				width += metrics.gap
			}
			width += size.Width + 2*metrics.paddingX
			textHeight = max(textHeight, size.Height)
		}
		height := max(metrics.height, textHeight+2*metrics.paddingY+metrics.indicator)
		size := layout.Size{Width: width, Height: height}
		return layout.IntrinsicSize{Minimum: size, Preferred: size}, nil
	})
}

func tabsGeometry(bounds paint.Rect, labels []layout.Size, metrics tabsMetrics) []tabGeometry {
	result := make([]tabGeometry, len(labels))
	x := bounds.X
	for index, size := range labels {
		if index != 0 {
			x += metrics.gap
		}
		width := size.Width + 2*metrics.paddingX
		item := paint.Rect{X: x, Y: bounds.Y, Width: width, Height: bounds.Height}
		textHeight := max(0, item.Height-2*metrics.paddingY-metrics.indicator)
		result[index] = tabGeometry{
			item: item,
			text: paint.Rect{X: item.X + metrics.paddingX, Y: item.Y + metrics.paddingY, Width: size.Width, Height: textHeight},
		}
		x += width
	}
	return result
}

func appendTabsDisplay(list *paint.DisplayList, props *TabsProps, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, engine *internaltext.Engine, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse, visual computedVisual) error {
	metrics, err := resolveTabsMetrics(theme)
	if err != nil {
		return err
	}
	labels, err := measureTabLabels(engine, theme, *props)
	if err != nil {
		return err
	}
	bounds := paintRect(geometry.Content)
	items := tabsGeometry(bounds, labels, metrics)
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return nil
	}
	baseText := visual.textColor
	if !visual.textColorSet {
		baseText, err = theme.color(TokenColor(ColorSemanticText))
		if err != nil {
			return err
		}
	}
	hover, err := theme.color(TokenColor(ColorSemanticTabsHover))
	if err != nil {
		return err
	}
	pressed, err := theme.color(TokenColor(ColorSemanticTabsPressed))
	if err != nil {
		return err
	}
	selected, err := theme.color(TokenColor(ColorSemanticTabsSelected))
	if err != nil {
		return err
	}
	disabled, err := theme.color(TokenColor(ColorSemanticTabsDisabled))
	if err != nil {
		return err
	}
	indicator, err := theme.color(TokenColor(ColorSemanticTabsIndicator))
	if err != nil {
		return err
	}
	*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: bounds})
	for index, item := range props.Items {
		box := items[index]
		if index == instance.State.TabsPressed {
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: box.item, Color: paintColor(pressed)})
		} else if index == instance.State.TabsHover && !item.Disabled && !props.Disabled {
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: box.item, Color: paintColor(hover)})
		}
		color := baseText
		if item.Value == props.Value {
			color = selected
		}
		if item.Disabled {
			color = disabled
		}
		if props.common().Style.Text.Color.set {
			color = baseText
		}
		if err := appendSelectText(list, instance.ID, item.Label, props.common(), box.text, color, theme, engine, scaleX, scaleY, remaining, reuse); err != nil {
			return err
		}
		if item.Value == props.Value {
			lineColor := indicator
			if item.Disabled {
				lineColor = disabled
			}
			line := paint.Rect{
				X:     box.item.X + metrics.indicatorInset,
				Y:     box.item.Y + max(0, box.item.Height-metrics.indicator),
				Width: max(0, box.item.Width-2*metrics.indicatorInset), Height: min(metrics.indicator, box.item.Height),
			}
			if line.Width > 0 && line.Height > 0 {
				*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: line, Color: paintColor(lineColor)})
			}
		}
	}
	*list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: instance.ID})
	return nil
}

func tabIndexForValue(items []TabItem, value string) int {
	for index := range items {
		if items[index].Value == value {
			return index
		}
	}
	return -1
}

func nextEnabledTab(items []TabItem, current, direction int) int {
	for index := current + direction; index >= 0 && index < len(items); index += direction {
		if !items[index].Disabled {
			return index
		}
	}
	return current
}

func syncTabsStates(view View, instance *tree.Node) bool {
	if view.node == nil || instance == nil {
		return false
	}
	changed := false
	if view.node.kind == viewTabs {
		props := view.node.tabs
		active := tabIndexForValue(props.Items, instance.State.TabsActiveKey)
		valueChanged := !instance.State.TabsInitialized || instance.State.TabsValue != props.Value
		if valueChanged || active < 0 || active < len(props.Items) && props.Items[active].Disabled {
			active = tabIndexForValue(props.Items, props.Value)
			if active < 0 || props.Items[active].Disabled {
				active = nextEnabledTab(props.Items, -1, 1)
			}
		}
		activeKey := ""
		if active >= 0 && active < len(props.Items) {
			activeKey = props.Items[active].Value
		}
		hover, pressed := instance.State.TabsHover, instance.State.TabsPressed
		if !instance.State.TabsInitialized || props.Disabled {
			hover, pressed = -1, -1
		}
		changed = instance.State.TabsActive != active || instance.State.TabsActiveKey != activeKey ||
			instance.State.TabsHover != hover || instance.State.TabsPressed != pressed ||
			instance.State.TabsValue != props.Value || !instance.State.TabsInitialized
		instance.State.TabsActive, instance.State.TabsActiveKey = active, activeKey
		instance.State.TabsHover, instance.State.TabsPressed = hover, pressed
		instance.State.TabsValue, instance.State.TabsInitialized = props.Value, true
	}
	for index, child := range view.node.children {
		if index < len(instance.Children) {
			changed = syncTabsStates(child, instance.Children[index]) || changed
		}
	}
	return changed
}
