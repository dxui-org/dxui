package dxui

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	internalinput "github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/tree"
)

func describeView(view View) (description tree.Description, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("dxui: build view panic: %v", recovered)
		}
	}()
	return describeNode(view, "root")
}

func describeNode(view View, path string, inherited ...buttonTypography) (tree.Description, error) {
	if view.node == nil || view.node.kind == viewInvalid {
		return tree.Description{}, fmt.Errorf("dxui: %s is an invalid View", path)
	}
	node := view.node
	description := tree.Description{Kind: tree.Kind(node.kind), Key: tree.Key(node.key)}

	var common viewProps
	switch node.kind {
	case viewBox:
		if node.box == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s box has missing properties", path)
		}
		common = node.box.common()
		description.Properties.Layout.Axis = uint8(boxAxis(node.box.Direction))
		description.Properties.Layout.Gap = treeValue(Metric(node.box.Gap))
		description.Properties.Layout.Justify = uint8(node.box.Justify)
		description.Properties.Layout.Align = uint8(node.box.Align)
		if err := validateBoxProperties(*node.box, path); err != nil {
			return tree.Description{}, err
		}
	case viewText:
		if node.text == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s text has missing properties", path)
		}
		common = node.text.common()
		description.Properties.Content = tree.ContentProperties{
			Text: node.text.Value, Wrap: uint8(node.text.Wrap), MaxLines: node.text.MaxLines,
		}
		if node.text.Wrap > TextWrapWords || node.text.MaxLines < 0 {
			return tree.Description{}, fmt.Errorf("dxui: %s has invalid text wrapping properties", path)
		}
	case viewButton:
		if node.button == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s button has missing properties", path)
		}
		if node.button.Variant > ButtonLink || node.button.Tone > ButtonDanger || node.button.Size > ButtonLarge {
			return tree.Description{}, fmt.Errorf("dxui: %s has invalid Button Variant/Tone/Size", path)
		}
		common = node.button.common()
		description.Properties.Semantics.Disabled = node.button.Disabled
		description.Properties.Semantics.HasPress = node.button.OnPress != nil
	case viewButtonGroup:
		if node.buttonGroup == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s button group has missing properties", path)
		}
		if node.buttonGroup.Orientation > ButtonGroupVertical {
			return tree.Description{}, fmt.Errorf("dxui: %s has invalid button group orientation %d", path, node.buttonGroup.Orientation)
		}
		common = node.buttonGroup.common()
		description.Properties.Layout.Axis = uint8(node.buttonGroup.Orientation) + 1
		if node.buttonGroup.Dividers {
			description.Properties.Layout.Overlap = treeValue(TokenMetric(MetricComponentButtonGroupBorderWidth))
		}
		for index, child := range node.children {
			if child.node == nil || child.node.kind != viewButton {
				return tree.Description{}, fmt.Errorf("dxui: %s child %d must be a Button", path, index)
			}
		}
	case viewInputGroup:
		if node.inputGroup == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s input group has missing properties", path)
		}
		common = node.inputGroup.common()
		description.Properties.Layout.Axis = uint8(boxAxis(Horizontal))
		if err := validateInputGroupChildren(node.children, node.groupInput, path); err != nil {
			return tree.Description{}, err
		}
	case viewBadge:
		if node.badge == nil || len(node.children) != 1 {
			return tree.Description{}, fmt.Errorf("dxui: %s badge must have exactly one child", path)
		}
		common = node.badge.common()
	case viewProgressBar:
		if node.progress == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s progress bar has missing properties", path)
		}
		common = node.progress.common()
		description.Properties.Progress.Value = normalizeProgress(node.progress.Value)
	case viewInput:
		if node.input == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s input has missing properties", path)
		}
		common = node.input.common()
		description.Properties.Content.Text = node.input.Value
		description.Properties.Content.Password = node.input.Password
		description.Properties.Content.PasswordToggle = node.input.Password && node.input.ShowPasswordToggle
		description.Properties.Semantics = tree.SemanticsProperties{
			Disabled: node.input.Disabled, ReadOnly: node.input.ReadOnly,
			HasChange: node.input.OnChange != nil, HasSubmit: node.input.OnSubmit != nil,
		}
		if utf8.RuneCountInString(node.input.Value) > internalinput.MaxTextRunes {
			return tree.Description{}, fmt.Errorf("dxui: %s input exceeds maximum text length", path)
		}
	case viewTextarea:
		if node.textarea == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s textarea has missing properties", path)
		}
		common = node.textarea.common()
		description.Properties.Content.Text = node.textarea.Value
		description.Properties.Content.Wrap = uint8(node.textarea.Wrap)
		description.Properties.Semantics = tree.SemanticsProperties{
			Disabled: node.textarea.Disabled, ReadOnly: node.textarea.ReadOnly,
			HasChange: node.textarea.OnChange != nil,
		}
		if node.textarea.Wrap > TextWrapWords {
			return tree.Description{}, fmt.Errorf("dxui: %s textarea has invalid wrap", path)
		}
		if utf8.RuneCountInString(node.textarea.Value) > internalinput.MaxTextRunes {
			return tree.Description{}, fmt.Errorf("dxui: %s textarea exceeds maximum text length", path)
		}
	case viewToggleSwitch:
		if node.toggle == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s toggle switch has missing properties", path)
		}
		common = node.toggle.common()
		description.Properties.Semantics.Disabled = node.toggle.Disabled
		description.Properties.Semantics.Checked = node.toggle.Checked
		description.Properties.Semantics.HasToggle = node.toggle.OnChange != nil
	case viewSlider:
		if node.slider == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s slider has missing properties", path)
		}
		common = node.slider.common()
		model := normalizeSlider(*node.slider)
		description.Properties.Semantics.Disabled = node.slider.Disabled || !model.valid
		description.Properties.Slider = tree.SliderProperties{
			Enabled: model.valid, HasChange: node.slider.OnChange != nil,
			Value: model.value, Min: model.min, Max: model.max, Step: model.step,
		}
	case viewCheckbox:
		if node.checkbox == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s checkbox has missing properties", path)
		}
		common = node.checkbox.common()
		description.Properties.Semantics.Disabled = node.checkbox.Disabled
		description.Properties.Semantics.Checked = node.checkbox.Checked
		description.Properties.Semantics.HasToggle = node.checkbox.OnChange != nil
	case viewRadio:
		if node.radio == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s radio has missing properties", path)
		}
		common = node.radio.common()
		description.Properties.Semantics.Disabled = node.radio.Disabled
		description.Properties.Semantics.Checked = node.radio.Selected
		description.Properties.Semantics.HasToggle = node.radio.OnSelect != nil
	case viewIcon:
		if node.icon == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s icon has missing properties", path)
		}
		common = node.icon.common()
		if err := validateIcon(*node.icon, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Content.VectorHash = paintStateHash(struct {
			DataID      uint64
			Size        float32
			StrokeWidth IconStrokeWidth
		}{node.icon.Data.Identity(), node.icon.Size, node.icon.StrokeWidth})
	case viewImage:
		if node.image == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s image has missing properties", path)
		}
		common = node.image.common()
		if err := validateImage(*node.image, path); err != nil {
			return tree.Description{}, err
		}
		if node.image.Source.source != nil {
			description.Properties.Resource.Identity = paintStateHash(struct {
				ID        uint64
				MaxPixels int64
			}{node.image.Source.source.id, node.image.MaxPixels})
		}
		description.Properties.Semantics.HasLoad = node.image.OnLoad != nil
		description.Properties.Semantics.HasError = node.image.OnError != nil
	case viewAvatar:
		if node.avatar == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s avatar has missing properties", path)
		}
		common = node.avatar.common()
		if err := validateAvatar(*node.avatar, path); err != nil {
			return tree.Description{}, err
		}
		if node.avatar.Source.source != nil {
			description.Properties.Resource.Identity = paintStateHash(struct {
				ID        uint64
				MaxPixels int64
			}{node.avatar.Source.source.id, int64(0)})
		}
		description.Properties.Content.VectorHash = paintStateHash(node.avatar.Size)
		description.Properties.Semantics.HasLoad = node.avatar.OnLoad != nil
		description.Properties.Semantics.HasError = node.avatar.OnError != nil
	case viewControlMark:
		if node.mark == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s control mark has missing properties", path)
		}
		common = node.mark.common()
		if err := validateMetric(node.mark.Size, path+" control mark size"); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Content.VectorHash = paintStateHash(node.mark.Size)
	case viewScroll:
		if node.scroll == nil || len(node.children) != 1 {
			return tree.Description{}, fmt.Errorf("dxui: %s scroll must have exactly one child", path)
		}
		common = node.scroll.common()
		if err := validateScrollProperties(*node.scroll, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Scroll = tree.ScrollProperties{
			Enabled: true, Axis: uint8(node.scroll.Axis), Scrollbar: uint8(node.scroll.Scrollbar),
			HasScrollCallback: node.scroll.OnScroll != nil,
		}
		if initial, set := node.scroll.InitialOffset.get(); set {
			description.Properties.Scroll.InitialSet = true
			description.Properties.Scroll.InitialX, description.Properties.Scroll.InitialY = initial.X, initial.Y
		}
		if offset, set := node.scroll.Offset.get(); set {
			description.Properties.Scroll.Controlled = true
			description.Properties.Scroll.OffsetX, description.Properties.Scroll.OffsetY = offset.X, offset.Y
		}
	case viewVirtualList:
		if node.virtualList == nil || node.scroll == nil || len(node.children) != 1 {
			return tree.Description{}, fmt.Errorf("dxui: %s virtual list is not prepared", path)
		}
		common = node.virtualList.common()
		if err := validateScrollProperties(*node.scroll, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Scroll = tree.ScrollProperties{Enabled: true, Axis: uint8(ScrollVertical), Scrollbar: uint8(node.virtualList.Scrollbar), HasScrollCallback: node.virtualList.OnScroll != nil}
		description.Properties.Content.VectorHash = paintStateHash(struct {
			Count     int
			Version   uint64
			RowHeight float32
			Overscan  int
		}{node.virtualList.Count, node.virtualList.Version, node.virtualList.RowHeight, node.virtualList.Overscan})
		if initial, set := node.virtualList.InitialOffset.get(); set {
			description.Properties.Scroll.InitialSet = true
			description.Properties.Scroll.InitialX, description.Properties.Scroll.InitialY = initial.X, initial.Y
		}
		if offset, set := node.virtualList.Offset.get(); set {
			description.Properties.Scroll.Controlled = true
			description.Properties.Scroll.OffsetX, description.Properties.Scroll.OffsetY = offset.X, offset.Y
		}
	case viewSelect:
		if node.selectp == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s select has missing properties", path)
		}
		common = node.selectp.common()
		if err := validateSelectProperties(*node.selectp, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Content.Text = node.selectp.Value
		description.Properties.Content.VectorHash = paintStateHash(node.selectp.Options)
		description.Properties.Semantics.Disabled = node.selectp.Disabled
		description.Properties.Semantics.HasSelect = node.selectp.OnChange != nil
	case viewTabs:
		if node.tabs == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s tabs has missing properties", path)
		}
		common = node.tabs.common()
		if err := validateTabsProperties(*node.tabs, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Content.VectorHash = paintStateHash(node.tabs.Items)
		description.Properties.Tabs.Value = node.tabs.Value
		description.Properties.Semantics.Disabled = node.tabs.Disabled
		description.Properties.Semantics.HasTabs = node.tabs.OnChange != nil
	case viewMenu:
		if node.menu == nil {
			return tree.Description{}, fmt.Errorf("dxui: %s menu has missing properties", path)
		}
		common = node.menu.common()
		if err := validateMenuProperties(*node.menu, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Layout.Axis = uint8(node.menu.Orientation) + 1
		description.Properties.Content.VectorHash = paintStateHash(node.menu.Items)
		description.Properties.Menu.Value = node.menu.Value
		description.Properties.Semantics.Disabled = node.menu.Disabled
		description.Properties.Semantics.HasMenu = node.menu.OnAction != nil
	case viewPopover:
		if node.popover == nil || len(node.children) != 2 {
			return tree.Description{}, fmt.Errorf("dxui: %s popover must have exactly an anchor and content", path)
		}
		common = node.popover.common()
		if err := validateOverlayProperties(node.popover.Placement, node.popover.Offset, 0, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Overlay = tree.OverlayProperties{
			Kind: 1, Open: node.popover.Open, Placement: uint8(node.popover.Placement), Offset: treeValue(node.popover.Offset),
			HasOpenChange: node.popover.OnOpenChange != nil,
		}
	case viewTooltip:
		if node.tooltip == nil || len(node.children) != 2 {
			return tree.Description{}, fmt.Errorf("dxui: %s tooltip must have exactly an anchor and content", path)
		}
		common = node.tooltip.common()
		if err := validateOverlayProperties(node.tooltip.Placement, node.tooltip.Offset, node.tooltip.Delay, path); err != nil {
			return tree.Description{}, err
		}
		description.Properties.Overlay = tree.OverlayProperties{
			Kind: 2, Placement: uint8(node.tooltip.Placement), Offset: treeValue(node.tooltip.Offset),
			DelayNS: int64(node.tooltip.Delay), Disabled: node.tooltip.Disabled,
		}
	default:
		return tree.Description{}, fmt.Errorf("dxui: %s has unsupported node kind %d", path, node.kind)
	}
	if err := validateNodeProperties(common, path); err != nil {
		return tree.Description{}, err
	}
	typography := inheritedButtonTypography(view, common.Style.Text, inherited)
	if node.kind == viewText {
		common.Style.Text = typography.apply(common.Style.Text)
	}
	applyNodeProperties(&description.Properties, common, nodeUsesTextMetrics(node.kind))
	if node.kind == viewIcon {
		description.Properties.Paint.IconColorHash = paintStateHash(node.icon.Color)
	}
	if node.kind == viewImage {
		description.Properties.Paint.ImagePresentationHash = paintStateHash(struct {
			Fit       ImageFit
			Alignment Point
		}{node.image.Fit, node.image.Alignment})
	}
	if node.kind == viewAvatar {
		description.Properties.Paint.ImagePresentationHash = paintStateHash(node.avatar.Shape)
	}

	if len(node.children) != 0 {
		description.Children = make([]tree.Description, len(node.children))
		for index, child := range node.children {
			childDescription, childErr := describeNode(child, fmt.Sprintf("%s child %d", path, index), typography)
			if childErr != nil {
				return tree.Description{}, childErr
			}
			description.Children[index] = childDescription
		}
	}
	return description, nil
}

func validateInputGroupChildren(children []View, inputIndex int, path string) error {
	if inputIndex < 0 || inputIndex >= len(children) || children[inputIndex].node == nil {
		return fmt.Errorf("dxui: %s must contain exactly one valid Input", path)
	}
	if children[inputIndex].node.kind != viewInput {
		return fmt.Errorf("dxui: %s required Input is %s", path, viewKindName(children[inputIndex].node.kind))
	}
	for index, child := range children {
		if child.node == nil {
			return fmt.Errorf("dxui: %s child %d is an invalid View", path, index)
		}
		if index == inputIndex {
			continue
		}
		if err := validateInputGroupAdornment(child); err != nil {
			return fmt.Errorf("dxui: %s adornment child %d: %w", path, index, err)
		}
	}
	return nil
}

func validateInputGroupAdornment(view View) error {
	return validateInputGroupAdornmentNode(view, true)
}

func validateInputGroupAdornmentNode(view View, buttonAllowed bool) error {
	if view.node == nil {
		return errors.New("invalid View")
	}
	switch view.node.kind {
	case viewButton:
		if !buttonAllowed {
			return errors.New("nested Button is not passive content")
		}
	case viewInput, viewTextarea, viewSelect, viewToggleSwitch, viewSlider,
		viewCheckbox, viewRadio, viewScroll, viewVirtualList, viewTabs, viewMenu, viewPopover,
		viewInputGroup:
		return fmt.Errorf("%s is not passive content or a Button", viewKindName(view.node.kind))
	}
	for _, child := range view.node.children {
		if err := validateInputGroupAdornmentNode(child, false); err != nil {
			return err
		}
	}
	return nil
}

func viewKindName(kind viewKind) string {
	switch kind {
	case viewInput:
		return "Input"
	case viewTextarea:
		return "Textarea"
	case viewSelect:
		return "Select"
	case viewToggleSwitch:
		return "ToggleSwitch"
	case viewSlider:
		return "Slider"
	case viewCheckbox:
		return "Checkbox"
	case viewRadio:
		return "Radio"
	case viewScroll:
		return "Scroll"
	case viewVirtualList:
		return "VirtualList"
	case viewTabs:
		return "Tabs"
	case viewMenu:
		return "Menu"
	case viewPopover:
		return "Popover"
	case viewInputGroup:
		return "InputGroup"
	default:
		return fmt.Sprintf("focusable view kind %d", kind)
	}
}

func validateOverlayProperties(placement OverlayPlacement, offset MetricValue, delay time.Duration, path string) error {
	if placement > OverlayRight {
		return fmt.Errorf("dxui: %s has invalid overlay placement %d", path, placement)
	}
	if err := validateMetric(offset, path+" overlay offset"); err != nil {
		return err
	}
	if delay < 0 {
		return fmt.Errorf("dxui: %s tooltip delay must be non-negative", path)
	}
	return nil
}

func validateSelectProperties(properties SelectProps, path string) error {
	if len(properties.Options) > selectOptionLimit {
		return fmt.Errorf("dxui: %s has %d options; MVP maximum is %d", path, len(properties.Options), selectOptionLimit)
	}
	values := make(map[string]int, len(properties.Options))
	for index, option := range properties.Options {
		if option.Value == "" {
			return fmt.Errorf("dxui: %s option %d has an empty value", path, index)
		}
		if first, exists := values[option.Value]; exists {
			return fmt.Errorf("dxui: %s has duplicate option value %q at %d and %d", path, option.Value, first, index)
		}
		values[option.Value] = index
	}
	return nil
}

func validateTabsProperties(properties TabsProps, path string) error {
	values := make(map[string]int, len(properties.Items))
	for index, item := range properties.Items {
		if item.Value == "" {
			return fmt.Errorf("dxui: %s tab item %d has an empty value", path, index)
		}
		if first, exists := values[item.Value]; exists {
			return fmt.Errorf("dxui: %s has duplicate tab item value %q at %d and %d", path, item.Value, first, index)
		}
		values[item.Value] = index
	}
	return nil
}

func validateMenuProperties(properties MenuProps, path string) error {
	if properties.Orientation > MenuHorizontal {
		return fmt.Errorf("dxui: %s has invalid menu orientation %d", path, properties.Orientation)
	}
	values := make(map[string]int, len(properties.Items))
	for index, item := range properties.Items {
		if item.Value == "" {
			return fmt.Errorf("dxui: %s menu item %d has an empty value", path, index)
		}
		if first, exists := values[item.Value]; exists {
			return fmt.Errorf("dxui: %s has duplicate menu item value %q at %d and %d", path, item.Value, first, index)
		}
		values[item.Value] = index
	}
	return nil
}

func validateScrollProperties(properties ScrollProps, path string) error {
	if properties.Axis > ScrollBoth || properties.Scrollbar > ScrollbarHidden {
		return fmt.Errorf("dxui: %s has invalid scroll properties", path)
	}
	for name, option := range map[string]Option[Point]{"initial offset": properties.InitialOffset, "offset": properties.Offset} {
		if point, set := option.get(); set && (!finiteNonNegative(point.X) || !finiteNonNegative(point.Y)) {
			return fmt.Errorf("dxui: %s %s must be finite and non-negative", path, name)
		}
	}
	return nil
}

func validateIcon(properties IconProps, path string) error {
	box := properties.Data.ViewBox
	if !finite(box.X) || !finite(box.Y) || !finite(box.Width) || !finite(box.Height) || box.Width <= 0 || box.Height <= 0 {
		return fmt.Errorf("dxui: %s icon has an invalid view box", path)
	}
	if !finiteNonNegative(properties.Size) {
		return fmt.Errorf("dxui: %s icon size must be finite and non-negative", path)
	}
	if properties.Color.isToken && properties.Color.token == "" {
		return fmt.Errorf("dxui: %s icon has an empty color token", path)
	}
	if !finite(float32(properties.StrokeWidth)) || properties.StrokeWidth < 0 {
		return fmt.Errorf("dxui: %s icon stroke width must be finite and non-negative", path)
	}
	if properties.Data.IsPacked() {
		return nil
	}
	if len(properties.Data.Commands) > maxIconCommands {
		return fmt.Errorf("dxui: %s icon has %d commands; maximum is %d", path, len(properties.Data.Commands), maxIconCommands)
	}
	for index, command := range properties.Data.Commands {
		if command.Verb > PathClose {
			return fmt.Errorf("dxui: %s icon command %d has an invalid verb", path, index)
		}
		for _, point := range command.Points {
			if !finite(point.X) || !finite(point.Y) {
				return fmt.Errorf("dxui: %s icon command %d has a non-finite point", path, index)
			}
		}
	}
	return nil
}

func validateImage(properties ImageProps, path string) error {
	if properties.Source.source == nil {
		return fmt.Errorf("dxui: %s image has no source", path)
	}
	if properties.Fit > ImageNone || !finite(properties.Alignment.X) || !finite(properties.Alignment.Y) || properties.Alignment.X < 0 || properties.Alignment.X > 1 || properties.Alignment.Y < 0 || properties.Alignment.Y > 1 {
		return fmt.Errorf("dxui: %s image has invalid fit or alignment", path)
	}
	if properties.MaxPixels < 0 {
		return fmt.Errorf("dxui: %s image maximum pixels must be non-negative", path)
	}
	return nil
}

func validateAvatar(properties AvatarProps, path string) error {
	if properties.Shape > AvatarSquare {
		return fmt.Errorf("dxui: %s avatar has invalid shape", path)
	}
	if !finiteNonNegative(properties.Size) {
		return fmt.Errorf("dxui: %s avatar size must be finite and non-negative", path)
	}
	return validateImage(imagePropsForAvatar(&properties), path)
}

func validateBoxProperties(properties BoxProps, path string) error {
	if !finiteNonNegative(properties.Gap) {
		return fmt.Errorf("dxui: %s gap must be finite and non-negative", path)
	}
	if properties.Direction > Horizontal {
		return fmt.Errorf("dxui: %s has invalid box direction", path)
	}
	if properties.Justify > JustifySpaceBetween || properties.Align > AlignStretch {
		return fmt.Errorf("dxui: %s has invalid flex alignment", path)
	}
	return nil
}

func validateNodeProperties(properties viewProps, path string) error {
	if properties.Pointer > PointerNone {
		return fmt.Errorf("dxui: %s has invalid pointer behavior", path)
	}
	layout := properties.Style
	lengths := []struct {
		name  string
		value Length
	}{
		{"width", layout.Width}, {"height", layout.Height},
		{"minimum width", layout.MinWidth}, {"minimum height", layout.MinHeight},
		{"maximum width", layout.MaxWidth}, {"maximum height", layout.MaxHeight},
		{"top inset", layout.Insets.Top}, {"right inset", layout.Insets.Right},
		{"bottom inset", layout.Insets.Bottom}, {"left inset", layout.Insets.Left},
		{"flex basis", layout.Basis},
	}
	for _, item := range lengths {
		if err := validateLength(item.value, path+" "+item.name); err != nil {
			return err
		}
	}
	for _, edge := range []struct {
		name  string
		value EdgeValues
	}{{"margin", layout.Margin}, {"padding", layout.Padding}} {
		for _, side := range []struct {
			name  string
			value MetricValue
		}{{"top", edge.value.Top}, {"right", edge.value.Right}, {"bottom", edge.value.Bottom}, {"left", edge.value.Left}} {
			if err := validateMetric(side.value, path+" "+edge.name+" "+side.name); err != nil {
				return err
			}
		}
	}
	if !finiteNonNegative(layout.Grow) {
		return fmt.Errorf("dxui: %s flex grow must be finite and non-negative", path)
	}
	if shrink, set := layout.Shrink.get(); set && !finiteNonNegative(shrink) {
		return fmt.Errorf("dxui: %s flex shrink must be finite and non-negative", path)
	}
	if align, set := layout.AlignSelf.get(); set && align > AlignStretch {
		return fmt.Errorf("dxui: %s has invalid align-self", path)
	}
	if layout.Position > PositionAbsolute || layout.Overflow > OverflowClip {
		return fmt.Errorf("dxui: %s has invalid layout enum", path)
	}
	paint := properties.Style
	if opacity, set := paint.Opacity.get(); set && (!finite(opacity) || opacity < 0 || opacity > 1) {
		return fmt.Errorf("dxui: %s opacity must be between 0 and 1", path)
	}
	if paint.Visibility > Hidden {
		return fmt.Errorf("dxui: %s has invalid visibility", path)
	}
	if paint.Border.Sides & ^BorderAll != 0 {
		return fmt.Errorf("dxui: %s has invalid border sides", path)
	}
	if paint.Border.Pattern > BorderDashed {
		return fmt.Errorf("dxui: %s has invalid border pattern", path)
	}
	for _, color := range []ColorValue{paint.Background, paint.Border.Color, properties.Style.Text.Color} {
		if color.isToken && color.token == "" {
			return fmt.Errorf("dxui: %s has an empty color token", path)
		}
	}
	textStyle := properties.Style.Text
	for _, family := range textStyle.Families {
		if family == "" || strings.IndexByte(string(family), 0) >= 0 {
			return fmt.Errorf("dxui: %s has an invalid font family %q", path, family)
		}
	}
	if !finiteNonNegative(textStyle.Size) {
		return fmt.Errorf("dxui: %s font size must be finite and non-negative", path)
	}
	if !finiteNonNegative(textStyle.LineHeight) {
		return fmt.Errorf("dxui: %s line height must be finite and non-negative", path)
	}
	if textStyle.Weight > 1000 || textStyle.Slant > SlantItalic || textStyle.Align > TextEnd {
		return fmt.Errorf("dxui: %s has invalid text style", path)
	}
	if err := validateMetric(paint.Border.Width, path+" border width"); err != nil {
		return err
	}
	for _, radius := range []MetricValue{paint.Radius.TopLeft, paint.Radius.TopRight, paint.Radius.BottomRight, paint.Radius.BottomLeft} {
		if err := validateMetric(radius, path+" radius"); err != nil {
			return err
		}
	}
	if len(paint.Shadow) > 4 {
		return fmt.Errorf("dxui: %s has %d shadows; MVP maximum is 4", path, len(paint.Shadow))
	}
	for index, shadow := range paint.Shadow {
		for _, metric := range []MetricValue{shadow.OffsetX, shadow.OffsetY, shadow.Blur, shadow.Spread} {
			if err := validateMetric(metric, fmt.Sprintf("%s shadow %d", path, index)); err != nil {
				return err
			}
		}
		if !shadow.Color.set || shadow.Color.isToken && shadow.Color.token == "" {
			return fmt.Errorf("dxui: %s shadow %d has no valid color", path, index)
		}
	}
	return nil
}

func validateLength(value Length, path string) error {
	if value.kind > lengthPercent || !finite(value.value) {
		return fmt.Errorf("dxui: %s is invalid", path)
	}
	if value.kind == lengthPixels && value.value < 0 {
		return fmt.Errorf("dxui: %s must be non-negative", path)
	}
	if value.kind == lengthPercent && (value.value < 0 || value.value > 100) {
		return fmt.Errorf("dxui: %s percentage must be between 0 and 100", path)
	}
	return nil
}

func validateMetric(value MetricValue, path string) error {
	if value.isToken {
		if value.token == "" {
			return fmt.Errorf("dxui: %s has an empty metric token", path)
		}
		return nil
	}
	if !finiteNonNegative(value.literal) {
		return fmt.Errorf("dxui: %s must be finite and non-negative", path)
	}
	return nil
}

func finiteNonNegative(value float32) bool { return value >= 0 && finite(value) }

func finite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

func callBuilder(stage string, builder func() View) (view View, err error) {
	if builder == nil {
		return View{}, errors.New("dxui: nil builder")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("dxui: %s builder panic: %v", stage, recovered)
		}
	}()
	view = builder()
	if view.node == nil {
		return View{}, fmt.Errorf("dxui: %s builder returned an invalid View", stage)
	}
	return view, nil
}

func applyNodeProperties(properties *tree.Properties, node viewProps, usesTextMetrics bool) {
	properties.Semantics.Pointer = uint8(node.Pointer)
	layout := node.Style
	shrink, shrinkSet := layout.Shrink.get()
	alignSelf, alignSelfSet := layout.AlignSelf.get()
	properties.Layout.Width = treeLength(layout.Width)
	properties.Layout.Height = treeLength(layout.Height)
	properties.Layout.MinWidth = treeLength(layout.MinWidth)
	properties.Layout.MinHeight = treeLength(layout.MinHeight)
	properties.Layout.MaxWidth = treeLength(layout.MaxWidth)
	properties.Layout.MaxHeight = treeLength(layout.MaxHeight)
	properties.Layout.Margin = treeEdges(layout.Margin)
	properties.Layout.Padding = treeEdges(layout.Padding)
	properties.Layout.InsetTop = treeLength(layout.Insets.Top)
	properties.Layout.InsetRight = treeLength(layout.Insets.Right)
	properties.Layout.InsetBottom = treeLength(layout.Insets.Bottom)
	properties.Layout.InsetLeft = treeLength(layout.Insets.Left)
	properties.Layout.Grow = layout.Grow
	properties.Layout.Shrink = shrink
	properties.Layout.ShrinkSet = shrinkSet
	properties.Layout.Basis = treeLength(layout.Basis)
	properties.Layout.AlignSelf = uint8(alignSelf)
	properties.Layout.AlignSelfSet = alignSelfSet
	properties.Layout.Position = uint8(layout.Position)
	properties.Layout.ZIndex = layout.ZIndex
	properties.Layout.Overflow = uint8(layout.Overflow)

	paint := node.Style
	properties.Paint.BackgroundR = paint.Background.literal.R
	properties.Paint.BackgroundG = paint.Background.literal.G
	properties.Paint.BackgroundB = paint.Background.literal.B
	properties.Paint.BackgroundA = paint.Background.literal.A
	properties.Paint.BackgroundToken = string(paint.Background.token)
	properties.Paint.BackgroundIsToken = paint.Background.isToken
	properties.Paint.BackgroundSet = paint.Background.set
	properties.Paint.BorderWidth = treeValue(paint.Border.Width)
	properties.Paint.BorderR = paint.Border.Color.literal.R
	properties.Paint.BorderG = paint.Border.Color.literal.G
	properties.Paint.BorderB = paint.Border.Color.literal.B
	properties.Paint.BorderA = paint.Border.Color.literal.A
	properties.Paint.BorderToken = string(paint.Border.Color.token)
	properties.Paint.BorderIsToken = paint.Border.Color.isToken
	properties.Paint.BorderColorSet = paint.Border.Color.set
	properties.Paint.BorderPattern = uint8(paint.Border.Pattern)
	properties.Paint.BorderSides = uint8(paint.Border.Sides)
	properties.Paint.RadiusTopLeft = treeValue(paint.Radius.TopLeft)
	properties.Paint.RadiusTopRight = treeValue(paint.Radius.TopRight)
	properties.Paint.RadiusBottomRight = treeValue(paint.Radius.BottomRight)
	properties.Paint.RadiusBottomLeft = treeValue(paint.Radius.BottomLeft)
	properties.Paint.ShadowsHash = paintStateHash(paint.Shadow)
	properties.Paint.Opacity, properties.Paint.OpacitySet = paint.Opacity.get()
	properties.Paint.Visibility = uint8(paint.Visibility)
	properties.Paint.TextR = node.Style.Text.Color.literal.R
	properties.Paint.TextG = node.Style.Text.Color.literal.G
	properties.Paint.TextB = node.Style.Text.Color.literal.B
	properties.Paint.TextA = node.Style.Text.Color.literal.A
	properties.Paint.TextToken = string(node.Style.Text.Color.token)
	properties.Paint.TextIsToken = node.Style.Text.Color.isToken
	properties.Paint.TextSet = node.Style.Text.Color.set
	properties.Paint.ComponentToken = string(node.Token)
	properties.Paint.StatesHash = paintStateHash(struct {
		States  StateStyles
		Force   StylePatch
		Variant ButtonVariant
		Tone    ButtonTone
	}{node.States, node.Style.Force, node.buttonVariant, node.buttonTone})

	textStyle := node.Style.Text
	families := make([]string, len(textStyle.Families))
	for index, family := range textStyle.Families {
		families[index] = string(family)
	}
	properties.Content.FontFamilies = strings.Join(families, "\x00")
	if usesTextMetrics {
		properties.Content.FontSize = treeValue(textSizeMetric(textStyle.Size))
		properties.Content.LineHeight = treeValue(lineHeightMetric(textStyle.LineHeight))
	}
	properties.Content.FontWeight = uint16(textStyle.Weight)
	properties.Content.FontSlant = uint8(textStyle.Slant)
	properties.Content.TextAlign = uint8(textStyle.Align)
}

func treeLength(value Length) tree.Length {
	return tree.Length{Kind: uint8(value.kind), Value: value.value}
}

func treeValue(value MetricValue) tree.Value {
	return tree.Value{Token: string(value.token), Literal: value.literal, IsToken: value.isToken, Set: value.set}
}

func defaultMetric(value float32, token MetricToken) MetricValue {
	if value == 0 {
		return TokenMetric(token)
	}
	return Metric(value)
}

func textSizeMetric(value float32) MetricValue {
	return defaultMetric(value, MetricSemanticTextSize)
}

func lineHeightMetric(value float32) MetricValue {
	return defaultMetric(value, MetricSemanticLineHeight)
}

func nodeUsesTextMetrics(kind viewKind) bool {
	switch kind {
	case viewText, viewInput, viewTextarea, viewSelect, viewTabs, viewMenu:
		return true
	default:
		return false
	}
}

func iconSizeMetric(value float32) MetricValue {
	return defaultMetric(value, MetricComponentIconSize)
}

func avatarSizeMetric(value float32) MetricValue {
	return defaultMetric(value, MetricComponentAvatarSize)
}

func treeEdges(value EdgeValues) tree.Edges {
	return tree.Edges{
		Top: treeValue(value.Top), Right: treeValue(value.Right),
		Bottom: treeValue(value.Bottom), Left: treeValue(value.Left),
	}
}

func paintStateHash[T any](value T) uint64 {
	hash := fnv.New64a()
	_, _ = fmt.Fprintf(hash, "%#v", value)
	return hash.Sum64()
}
