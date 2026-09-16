package dxui

import (
	"fmt"
	"hash/fnv"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/tree"
)

// metricResolver is the style/theme boundary consumed by the pure layout
// engine. M4 tests use literals; the theme milestone supplies resolved token
// metrics without changing layout's algorithm contract.
type metricResolver func(MetricValue) (float32, error)

type intrinsicResolver func(View) layout.IntrinsicMeasurer

func layoutView(view View, constraints layout.Constraints, resolve metricResolver, intrinsic intrinsicResolver) (*layout.Result, layout.Stats, error) {
	return layoutViewWithEngine(layout.NewEngine(layout.EngineOptions{}), view, nil, constraints, resolve, intrinsic)
}

func layoutViewWithEngine(engine *layout.Engine, view View, instance *tree.Node, constraints layout.Constraints, resolve metricResolver, intrinsic intrinsicResolver) (*layout.Result, layout.Stats, error) {
	if resolve == nil {
		resolve = resolveLiteralMetric
	}
	root, err := buildLayoutNode(view, instance, resolve, intrinsic, "root")
	if err != nil {
		return nil, layout.Stats{}, err
	}
	result, err := engine.Layout(root, constraints)
	return result, engine.Stats(), err
}

func buildLayoutNode(view View, instance *tree.Node, resolve metricResolver, intrinsic intrinsicResolver, path string, inherited ...buttonTypography) (*layout.Node, error) {
	if view.node == nil {
		return nil, fmt.Errorf("dxui: layout %s is an invalid View", path)
	}
	result := &layout.Node{}
	var defaultMinHeight MetricValue
	if instance != nil {
		result.ID = instance.ID
	}
	var common viewProps
	switch view.node.kind {
	case viewBox:
		if view.node.box == nil {
			return nil, fmt.Errorf("dxui: layout %s has missing box properties", path)
		}
		common = view.node.box.common()
		result.Axis = boxAxis(view.node.box.Direction)
		result.Gap = view.node.box.Gap
		result.Justify = layout.Justify(view.node.box.Justify)
		result.Align = layout.Align(view.node.box.Align)
	case viewButtonGroup:
		if view.node.buttonGroup == nil {
			return nil, fmt.Errorf("dxui: layout %s has missing button group properties", path)
		}
		common = view.node.buttonGroup.common()
		if view.node.buttonGroup.Orientation == ButtonGroupVertical {
			result.Axis = layout.AxisColumn
			result.Align = layout.AlignStretch
		} else {
			result.Axis = layout.AxisRow
		}
		var err error
		if view.node.buttonGroup.Dividers {
			result.Overlap, err = resolve(TokenMetric(MetricComponentButtonGroupBorderWidth))
		}
		if err != nil {
			return nil, fmt.Errorf("dxui: layout %s button group spacing: %w", path, err)
		}
	case viewInputGroup:
		if view.node.inputGroup == nil {
			return nil, fmt.Errorf("dxui: layout %s has missing input group properties", path)
		}
		common = view.node.inputGroup.common()
		result.Axis = layout.AxisRow
		result.Align = layout.AlignCenter
		var err error
		result.Gap, err = resolve(TokenMetric(MetricComponentInputGroupGap))
		if err != nil {
			return nil, fmt.Errorf("dxui: layout %s input group gap: %w", path, err)
		}
		defaultMinHeight = TokenMetric(MetricSemanticControlHeight)
	case viewText:
		common = view.node.text.common()
		result.Version = contentVersion(view.node.text.Value, uint64(view.node.text.Wrap), uint64(view.node.text.MaxLines), paintStateHash(view.node.text.common().Style.Text))
	case viewButton:
		common = view.node.button.common()
		result.Axis = layout.AxisColumn
		defaultMinHeight = TokenMetric(MetricSemanticControlHeight)
	case viewBadge:
		common = view.node.badge.common()
		result.Axis = layout.AxisRow
		result.Align = layout.AlignCenter
		defaultMinHeight = TokenMetric(MetricComponentBadgeMinHeight)
	case viewProgressBar:
		common = view.node.progress.common()
		result.Version = contentVersion("progress-bar")
	case viewInput:
		common = view.node.input.common()
		result.Version = contentVersion(view.node.input.Value)
		defaultMinHeight = TokenMetric(MetricSemanticControlHeight)
	case viewTextarea:
		common = view.node.textarea.common()
		result.Version = contentVersion(view.node.textarea.Value, uint64(view.node.textarea.Wrap))
		defaultMinHeight = TokenMetric(MetricComponentTextareaMinHeight)
	case viewToggleSwitch:
		common = view.node.toggle.common()
		result.Version = contentVersion("toggle", boolVersion(view.node.toggle.Checked))
	case viewSlider:
		common = view.node.slider.common()
		result.Version = contentVersion("slider")
	case viewCheckbox:
		common = view.node.checkbox.common()
		defaultMinHeight = TokenMetric(MetricSemanticControlHeight)
		result.Axis = layout.AxisRow
		var err error
		result.Gap, err = resolve(TokenMetric(MetricComponentCheckboxGap))
		if err != nil {
			return nil, fmt.Errorf("dxui: layout %s checkbox gap: %w", path, err)
		}
		result.Align = layout.AlignCenter
		result.Version = contentVersion("checkbox", boolVersion(view.node.checkbox.Checked))
	case viewRadio:
		common = view.node.radio.common()
		defaultMinHeight = TokenMetric(MetricSemanticControlHeight)
		result.Axis = layout.AxisRow
		var err error
		result.Gap, err = resolve(TokenMetric(MetricComponentRadioGap))
		if err != nil {
			return nil, fmt.Errorf("dxui: layout %s radio gap: %w", path, err)
		}
		result.Align = layout.AlignCenter
		result.Version = contentVersion("radio", boolVersion(view.node.radio.Selected))
	case viewIcon:
		common = view.node.icon.common()
		result.Version = contentVersion("icon", view.node.icon.Data.Identity(), paintStateHash(view.node.icon.Size), paintStateHash(view.node.icon.StrokeWidth))
	case viewImage:
		common = view.node.image.common()
		sourceID := uint64(0)
		if view.node.image.Source.source != nil {
			sourceID = view.node.image.Source.source.id
		}
		result.Version = contentVersion("image", sourceID, uint64(view.node.image.MaxPixels))
	case viewAvatar:
		common = view.node.avatar.common()
		result.Version = contentVersion("avatar", paintStateHash(view.node.avatar.Size))
	case viewControlMark:
		common = view.node.mark.common()
		result.Version = contentVersion("control-mark", paintStateHash(view.node.mark.Size))
	case viewScroll:
		common = view.node.scroll.common()
		result.Scroll = layout.ScrollAxis(view.node.scroll.Axis + 1)
		result.Version = contentVersion("scroll", uint64(view.node.scroll.Axis))
	case viewVirtualList:
		common = view.node.virtualList.common()
		result.Scroll = layout.ScrollVertical
		result.Version = contentVersion("virtual-list", uint64(view.node.virtualList.Count), view.node.virtualList.Version, paintStateHash(view.node.virtualList.RowHeight))
	case viewSelect:
		common = view.node.selectp.common()
		result.Version = contentVersion("select", paintStateHash(view.node.selectp.Options), paintStateHash(view.node.selectp.Value), paintStateHash(view.node.selectp.Placeholder))
		defaultMinHeight = TokenMetric(MetricSemanticControlHeight)
	case viewTabs:
		common = view.node.tabs.common()
		result.Version = contentVersion("tabs", paintStateHash(view.node.tabs.Items), paintStateHash(view.node.tabs.Value))
		defaultMinHeight = TokenMetric(MetricComponentTabsHeight)
	case viewMenu:
		common = view.node.menu.common()
		result.Version = contentVersion("menu", paintStateHash(view.node.menu.Items), uint64(view.node.menu.Orientation))
	case viewPopover:
		common = view.node.popover.common()
		result.Axis = layout.AxisColumn
		result.Version = contentVersion("popover")
	case viewTooltip:
		common = view.node.tooltip.common()
		result.Axis = layout.AxisColumn
		result.Version = contentVersion("tooltip")
	default:
		return nil, fmt.Errorf("dxui: layout %s has unsupported node kind %d", path, view.node.kind)
	}
	typography := inheritedButtonTypography(view, common.Style.Text, inherited)
	if view.node.kind == viewText && typography.active && (common.Style.Text.Size == 0 && typography.size != 0 || common.Style.Text.LineHeight == 0 && typography.lineHeight != 0) {
		text := *view.node.text
		text.Style.Text = typography.apply(text.Style.Text)
		node := *view.node
		node.text = &text
		view = View{node: &node}
		result.Version = contentVersion(text.Value, uint64(text.Wrap), uint64(text.MaxLines), paintStateHash(text.Style.Text))
	}
	style, err := resolvedLayoutStyle(common.Style, resolve, path)
	if err != nil {
		return nil, err
	}
	result.Style = style
	if view.node.kind == viewInput && view.node.input.Password && view.node.input.ShowPasswordToggle {
		iconSize, resolveErr := resolve(TokenMetric(MetricComponentIconSize))
		if resolveErr != nil {
			return nil, fmt.Errorf("dxui: layout %s password toggle size: %w", path, resolveErr)
		}
		result.Style.Padding.Right += iconSize*passwordToggleIconScale + passwordToggleGap
	}
	if defaultMinHeight.set && common.Style.Height.kind == lengthAuto && common.Style.MinHeight.kind == lengthAuto {
		minimum, resolveErr := resolve(defaultMinHeight)
		if resolveErr != nil {
			return nil, fmt.Errorf("dxui: layout %s default minimum height: %w", path, resolveErr)
		}
		result.Style.MinHeight = layout.Length{Kind: layout.LengthPixels, Value: minimum}
	}
	if intrinsic != nil {
		result.Measure = intrinsic(view)
	}
	if result.Axis != layout.AxisLeaf || result.Scroll != layout.ScrollNone {
		result.Children = make([]*layout.Node, len(view.node.children))
		for index, child := range view.node.children {
			var childInstance *tree.Node
			if instance != nil && index < len(instance.Children) {
				childInstance = instance.Children[index]
			}
			result.Children[index], err = buildLayoutNode(child, childInstance, resolve, intrinsic, fmt.Sprintf("%s child %d", path, index), typography)
			if err != nil {
				return nil, err
			}
			if (view.node.kind == viewPopover || view.node.kind == viewTooltip) && index == 1 {
				result.Children[index].Style.Position = layout.PositionAbsolute
			}
		}
	}
	return result, nil
}

func boxAxis(direction Direction) layout.Axis {
	if direction == Horizontal {
		return layout.AxisRow
	}
	return layout.AxisColumn
}

func boolVersion(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func contentVersion(value string, extra ...uint64) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(value))
	result := hash.Sum64()
	for _, item := range extra {
		result = (result ^ item) * 1099511628211
	}
	return result
}

func resolvedLayoutStyle(source Style, resolve metricResolver, path string) (layout.Style, error) {
	margin, err := resolvedEdges(source.Margin, resolve, path+" margin")
	if err != nil {
		return layout.Style{}, err
	}
	padding, err := resolvedEdges(source.Padding, resolve, path+" padding")
	if err != nil {
		return layout.Style{}, err
	}
	shrink, shrinkSet := source.Shrink.get()
	alignSelf, alignSelfSet := source.AlignSelf.get()
	return layout.Style{
		Width: layoutLength(source.Width), Height: layoutLength(source.Height),
		MinWidth: layoutLength(source.MinWidth), MinHeight: layoutLength(source.MinHeight),
		MaxWidth: layoutLength(source.MaxWidth), MaxHeight: layoutLength(source.MaxHeight),
		Margin: margin, Padding: padding,
		Position: layout.Position(source.Position),
		Insets: layout.Insets{
			Top: layoutLength(source.Insets.Top), Right: layoutLength(source.Insets.Right),
			Bottom: layoutLength(source.Insets.Bottom), Left: layoutLength(source.Insets.Left),
		},
		Grow: source.Grow, Shrink: shrink, ShrinkSet: shrinkSet,
		Basis:     layoutLength(source.Basis),
		AlignSelf: layout.OptionalAlign{Value: layout.Align(alignSelf), Set: alignSelfSet},
		ZIndex:    source.ZIndex, Overflow: layout.Overflow(source.Overflow),
	}, nil
}

func resolvedEdges(source EdgeValues, resolve metricResolver, path string) (layout.Edges, error) {
	values := []*MetricValue{&source.Top, &source.Right, &source.Bottom, &source.Left}
	resolved := [4]float32{}
	for index, value := range values {
		var err error
		resolved[index], err = resolve(*value)
		if err != nil {
			return layout.Edges{}, fmt.Errorf("dxui: layout %s: %w", path, err)
		}
	}
	return layout.Edges{Top: resolved[0], Right: resolved[1], Bottom: resolved[2], Left: resolved[3]}, nil
}

func resolveLiteralMetric(value MetricValue) (float32, error) {
	if value.isToken {
		return 0, fmt.Errorf("metric token %q has not been resolved", value.token)
	}
	return value.literal, nil
}

func layoutLength(value Length) layout.Length {
	return layout.Length{Kind: layout.LengthKind(value.kind), Value: value.value}
}
