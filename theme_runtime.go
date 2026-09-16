package dxui

import (
	"fmt"
	"reflect"

	"github.com/dxui-org/dxui/internal/paint"
	internaltheme "github.com/dxui-org/dxui/internal/theme"
)

type resolvedTheme struct {
	source Theme
	tokens internaltheme.Resolved
}

func prepareTheme(source Theme) (resolvedTheme, error) {
	if themeIsZero(source) {
		source = LightTheme()
	}
	copied := copyTheme(source)
	input := internaltheme.Source{
		PrimitiveColors:  make(map[string]internaltheme.Color, len(copied.Primitive.Colors)),
		PrimitiveMetrics: make(map[string]float32, len(copied.Primitive.Metrics)),
		SemanticColors:   make(map[string]internaltheme.ColorValue, len(copied.Semantic.Colors)),
		SemanticMetrics:  make(map[string]internaltheme.MetricValue, len(copied.Semantic.Metrics)),
	}
	for token, value := range copied.Primitive.Colors {
		input.PrimitiveColors[string(token)] = themeColor(value)
	}
	for token, value := range copied.Primitive.Metrics {
		input.PrimitiveMetrics[string(token)] = value
	}
	for token, value := range copied.Semantic.Colors {
		input.SemanticColors[string(token)] = themeColorValue(value)
	}
	for token, value := range copied.Semantic.Metrics {
		input.SemanticMetrics[string(token)] = themeMetricValue(value)
	}
	resolved, err := internaltheme.Resolve(input)
	if err != nil {
		return resolvedTheme{}, err
	}
	result := resolvedTheme{source: copied, tokens: resolved}
	for token, component := range copied.Components {
		if token == "" {
			return resolvedTheme{}, fmt.Errorf("theme: empty component token")
		}
		if err := result.validatePatch(component.Base, fmt.Sprintf("component %q base", token)); err != nil {
			return resolvedTheme{}, err
		}
		for _, state := range namedStatePatches(component.States) {
			if err := result.validatePatch(state.patch, fmt.Sprintf("component %q %s", token, state.name)); err != nil {
				return resolvedTheme{}, err
			}
		}
	}
	return result, nil
}

func themeIsZero(value Theme) bool {
	return value.Primitive.Colors == nil && value.Primitive.Metrics == nil &&
		value.Semantic.Colors == nil && value.Semantic.Metrics == nil && value.Components == nil
}

func copyTheme(source Theme) Theme {
	result := Theme{
		Primitive: PrimitiveTokens{
			Colors:  make(map[ColorToken]RGBAColor, len(source.Primitive.Colors)),
			Metrics: make(map[MetricToken]float32, len(source.Primitive.Metrics)),
		},
		Semantic: SemanticTokens{
			Colors:  make(map[ColorToken]ColorValue, len(source.Semantic.Colors)),
			Metrics: make(map[MetricToken]MetricValue, len(source.Semantic.Metrics)),
		},
		Components: make(map[ComponentToken]ComponentTheme, len(source.Components)),
	}
	for key, value := range source.Primitive.Colors {
		result.Primitive.Colors[key] = value
	}
	for key, value := range source.Primitive.Metrics {
		result.Primitive.Metrics[key] = value
	}
	for key, value := range source.Semantic.Colors {
		result.Semantic.Colors[key] = value
	}
	for key, value := range source.Semantic.Metrics {
		result.Semantic.Metrics[key] = value
	}
	for key, value := range source.Components {
		value.Base = copyStylePatch(value.Base)
		value.States = copyStateStyles(value.States)
		result.Components[key] = value
	}
	return result
}

func copyStylePatch(source StylePatch) StylePatch {
	if shadows, set := source.Shadow.get(); set {
		source.Shadow = Some(cloneShadows(shadows))
	}
	return source
}

func copyStateStyles(source StateStyles) StateStyles {
	source.Default = copyStylePatch(source.Default)
	source.Hover = copyStylePatch(source.Hover)
	source.Focus = copyStylePatch(source.Focus)
	source.Disabled = copyStylePatch(source.Disabled)
	source.Pressed = copyStylePatch(source.Pressed)
	source.Checked = copyStylePatch(source.Checked)
	return source
}

func themeColor(value RGBAColor) internaltheme.Color {
	return internaltheme.Color{R: value.R, G: value.G, B: value.B, A: value.A}
}

func themeColorValue(value ColorValue) internaltheme.ColorValue {
	return internaltheme.ColorValue{Literal: themeColor(value.literal), Token: string(value.token), IsToken: value.isToken, Set: value.set}
}

func themeMetricValue(value MetricValue) internaltheme.MetricValue {
	return internaltheme.MetricValue{Literal: value.literal, Token: string(value.token), IsToken: value.isToken, Set: value.set}
}

func publicThemeColor(value internaltheme.Color) RGBAColor {
	return RGBA(value.R, value.G, value.B, value.A)
}

func (theme resolvedTheme) color(value ColorValue) (RGBAColor, error) {
	resolved, err := theme.tokens.Color(themeColorValue(value))
	return publicThemeColor(resolved), err
}

func (theme resolvedTheme) metric(value MetricValue) (float32, error) {
	return theme.tokens.Metric(themeMetricValue(value))
}

func (theme resolvedTheme) validatePatch(patch StylePatch, path string) error {
	if value, set := patch.Background.get(); set {
		if _, err := theme.color(value); err != nil {
			return fmt.Errorf("theme: %s background: %w", path, err)
		}
	}
	if value, set := patch.Border.get(); set {
		if value.Sides & ^BorderAll != 0 {
			return fmt.Errorf("theme: %s has invalid border sides", path)
		}
		if value.Pattern > BorderDashed {
			return fmt.Errorf("theme: %s has invalid border pattern", path)
		}
		if _, err := theme.metric(value.Width); err != nil {
			return fmt.Errorf("theme: %s border width: %w", path, err)
		}
		if value.Color.set {
			if _, err := theme.color(value.Color); err != nil {
				return fmt.Errorf("theme: %s border color: %w", path, err)
			}
		}
	}
	if value, set := patch.Radius.get(); set {
		for _, metric := range []MetricValue{value.TopLeft, value.TopRight, value.BottomRight, value.BottomLeft} {
			if _, err := theme.metric(metric); err != nil {
				return fmt.Errorf("theme: %s radius: %w", path, err)
			}
		}
	}
	if values, set := patch.Shadow.get(); set {
		for index, value := range values {
			for _, metric := range []MetricValue{value.OffsetX, value.OffsetY, value.Blur, value.Spread} {
				if _, err := theme.metric(metric); err != nil {
					return fmt.Errorf("theme: %s shadow %d: %w", path, index, err)
				}
			}
			if _, err := theme.color(value.Color); err != nil {
				return fmt.Errorf("theme: %s shadow %d color: %w", path, index, err)
			}
		}
	}
	if value, set := patch.Opacity.get(); set && (!finite(value) || value < 0 || value > 1) {
		return fmt.Errorf("theme: %s opacity must be between 0 and 1", path)
	}
	if value, set := patch.Visibility.get(); set && value > Hidden {
		return fmt.Errorf("theme: %s has invalid visibility", path)
	}
	if value, set := patch.TextColor.get(); set {
		if _, err := theme.color(value); err != nil {
			return fmt.Errorf("theme: %s text color: %w", path, err)
		}
	}
	return nil
}

func validateViewTheme(view View, theme resolvedTheme) error {
	if view.node == nil {
		return fmt.Errorf("invalid view")
	}
	props, err := nodeProps(view)
	if err != nil {
		return err
	}
	if view.node.kind == viewButton && props.Token == "" {
		recipe, err := buttonRecipe(theme, props.buttonVariant, props.buttonTone)
		if err != nil {
			return err
		}
		for _, patch := range [...]StylePatch{recipe.Base, recipe.States.Default, recipe.States.Hover, recipe.States.Focus, recipe.States.Pressed, recipe.States.Disabled} {
			if err := theme.validatePatch(patch, "button recipe"); err != nil {
				return err
			}
		}
	}
	for _, state := range namedStatePatches(props.States) {
		if err := theme.validatePatch(state.patch, "local "+state.name); err != nil {
			return err
		}
	}
	if err := theme.validatePatch(props.Style.Force, "local forced override"); err != nil {
		return err
	}
	metrics := []MetricValue{
		props.Style.Margin.Top, props.Style.Margin.Right,
		props.Style.Margin.Bottom, props.Style.Margin.Left,
		props.Style.Padding.Top, props.Style.Padding.Right,
		props.Style.Padding.Bottom, props.Style.Padding.Left,
	}
	if nodeUsesTextMetrics(view.node.kind) {
		for _, metric := range []struct {
			name  string
			value MetricValue
		}{
			{"text size", textSizeMetric(props.Style.Text.Size)},
			{"line height", lineHeightMetric(props.Style.Text.LineHeight)},
		} {
			resolved, metricErr := theme.metric(metric.value)
			if metricErr != nil {
				return metricErr
			}
			if resolved <= 0 {
				return fmt.Errorf("%s must resolve positive", metric.name)
			}
		}
	}
	if view.node.kind == viewButtonGroup {
		metrics = append(metrics, TokenMetric(MetricComponentButtonGroupRadius))
		if view.node.buttonGroup.Dividers {
			metrics = append(metrics, TokenMetric(MetricComponentButtonGroupBorderWidth))
		}
	}
	if view.node.kind == viewInputGroup {
		metrics = append(metrics, TokenMetric(MetricComponentInputGroupGap))
	}
	if props.Style.Height.kind == lengthAuto && props.Style.MinHeight.kind == lengthAuto {
		switch view.node.kind {
		case viewButton, viewInput, viewInputGroup, viewCheckbox, viewRadio, viewSelect:
			metrics = append(metrics, TokenMetric(MetricSemanticControlHeight))
		case viewBadge:
			metrics = append(metrics, TokenMetric(MetricComponentBadgeMinHeight))
		case viewTabs:
			metrics = append(metrics, TokenMetric(MetricComponentTabsHeight))
		case viewTextarea:
			metrics = append(metrics, TokenMetric(MetricComponentTextareaMinHeight))
		}
	}
	if view.node.kind == viewIcon {
		size := iconSizeMetric(view.node.icon.Size)
		resolved, sizeErr := theme.metric(size)
		if sizeErr != nil {
			return sizeErr
		}
		if resolved <= 0 {
			return fmt.Errorf("icon size must resolve positive")
		}
		if view.node.icon.Color.set {
			if _, err := theme.color(view.node.icon.Color); err != nil {
				return err
			}
		}
	}
	if view.node.kind == viewAvatar && (props.Style.Width.kind == lengthAuto || props.Style.Height.kind == lengthAuto) {
		size := avatarSizeMetric(view.node.avatar.Size)
		resolved, err := theme.metric(size)
		if err != nil {
			return err
		}
		if resolved <= 0 {
			return fmt.Errorf("avatar size must resolve positive")
		}
	}
	if view.node.kind == viewInput && view.node.input.Password && view.node.input.ShowPasswordToggle {
		iconSize := TokenMetric(MetricComponentIconSize)
		metrics = append(metrics, iconSize)
		resolved, err := theme.metric(iconSize)
		if err != nil {
			return err
		}
		if resolved <= 0 {
			return fmt.Errorf("password toggle icon size must resolve positive")
		}
		if _, err := theme.color(TokenColor(ColorSemanticAccent)); err != nil {
			return err
		}
	}
	if view.node.kind == viewToggleSwitch {
		metrics = append(metrics, TokenMetric(MetricComponentToggleWidth), TokenMetric(MetricComponentToggleHeight), TokenMetric(MetricComponentToggleKnobInset))
	}
	if view.node.kind == viewSlider {
		sliderMetrics := []MetricValue{
			TokenMetric(MetricComponentSliderWidth), TokenMetric(MetricComponentSliderHeight),
			TokenMetric(MetricComponentSliderTrackHeight), TokenMetric(MetricComponentSliderThumbSize),
		}
		metrics = append(metrics, sliderMetrics...)
		for _, value := range sliderMetrics {
			resolved, err := theme.metric(value)
			if err != nil {
				return err
			}
			if resolved <= 0 {
				return fmt.Errorf("slider metrics must resolve positive")
			}
		}
		for _, token := range []ColorToken{ColorSemanticSliderTrack, ColorSemanticSliderFill, ColorSemanticSliderThumb, ColorSemanticSliderThumbBorder} {
			if _, err := theme.color(TokenColor(token)); err != nil {
				return err
			}
		}
	}
	if view.node.kind == viewProgressBar {
		progressMetrics := []MetricValue{
			TokenMetric(MetricComponentProgressBarWidth), TokenMetric(MetricComponentProgressBarHeight),
			TokenMetric(MetricComponentProgressBarTrackHeight), TokenMetric(MetricComponentProgressBarRadius),
		}
		metrics = append(metrics, progressMetrics...)
		for index, value := range progressMetrics {
			resolved, err := theme.metric(value)
			if err != nil {
				return err
			}
			if resolved < 0 || index < 3 && resolved == 0 {
				return fmt.Errorf("progress bar metrics must resolve non-negative with positive sizes")
			}
		}
		for _, token := range []ColorToken{ColorSemanticProgressTrack, ColorSemanticProgressFill} {
			if _, err := theme.color(TokenColor(token)); err != nil {
				return err
			}
		}
	}
	if view.node.kind == viewCheckbox {
		metrics = append(metrics, TokenMetric(MetricComponentCheckboxSize), TokenMetric(MetricComponentCheckboxGap))
	}
	if view.node.kind == viewRadio {
		metrics = append(metrics, TokenMetric(MetricComponentRadioSize), TokenMetric(MetricComponentRadioGap))
	}
	if isScrollView(view) {
		metrics = append(metrics, TokenMetric(MetricComponentScrollThickness), TokenMetric(MetricComponentScrollMinThumb), TokenMetric(MetricComponentScrollInset))
		for _, token := range []ColorToken{ColorSemanticScrollTrack, ColorSemanticScrollThumb, ColorSemanticScrollThumbHover, ColorSemanticScrollThumbActive, ColorSemanticScrollThumbDisabled} {
			if _, err := theme.color(TokenColor(token)); err != nil {
				return err
			}
		}
	}
	if view.node.kind == viewSelect {
		metrics = append(metrics,
			TokenMetric(MetricComponentSelectItemHeight), TokenMetric(MetricComponentSelectMaxHeight), TokenMetric(MetricComponentSelectGap),
			TokenMetric(MetricComponentScrollThickness), TokenMetric(MetricComponentScrollMinThumb), TokenMetric(MetricComponentScrollInset),
		)
		for _, token := range []ColorToken{ColorSemanticSelectHover, ColorSemanticSelectSelected, ColorSemanticSelectDisabled, ColorSemanticScrollTrack, ColorSemanticScrollThumb} {
			if _, err := theme.color(TokenColor(token)); err != nil {
				return err
			}
		}
	}
	if view.node.kind == viewPopover {
		metrics = append(metrics, view.node.popover.Offset, TokenMetric(MetricComponentPopoverGap))
	}
	if view.node.kind == viewTooltip {
		metrics = append(metrics, view.node.tooltip.Offset, TokenMetric(MetricComponentTooltipGap))
	}
	if view.node.kind == viewTabs {
		tabsMetrics := []MetricValue{
			TokenMetric(MetricComponentTabsHeight), TokenMetric(MetricComponentTabsGap),
			TokenMetric(MetricComponentTabsPaddingX), TokenMetric(MetricComponentTabsPaddingY),
			TokenMetric(MetricComponentTabsIndicator), TokenMetric(MetricComponentTabsIndicatorInset),
		}
		metrics = append(metrics, tabsMetrics...)
		for _, value := range tabsMetrics {
			resolved, err := theme.metric(value)
			if err != nil {
				return err
			}
			if (value == TokenMetric(MetricComponentTabsHeight) || value == TokenMetric(MetricComponentTabsIndicator)) && resolved <= 0 {
				return fmt.Errorf("tabs height and indicator metrics must resolve positive")
			}
		}
		for _, token := range []ColorToken{ColorSemanticTabsHover, ColorSemanticTabsPressed, ColorSemanticTabsSelected, ColorSemanticTabsDisabled, ColorSemanticTabsIndicator} {
			if _, err := theme.color(TokenColor(token)); err != nil {
				return err
			}
		}
	}
	if view.node.kind == viewMenu {
		menuMetrics := []MetricValue{
			TokenMetric(MetricComponentMenuItemHeight), TokenMetric(MetricComponentMenuGap),
			TokenMetric(MetricComponentMenuPaddingX), TokenMetric(MetricComponentMenuPaddingY),
		}
		metrics = append(metrics, menuMetrics...)
		for _, value := range menuMetrics {
			resolved, err := theme.metric(value)
			if err != nil {
				return err
			}
			if value == TokenMetric(MetricComponentMenuItemHeight) && resolved <= 0 {
				return fmt.Errorf("menu item height must resolve positive")
			}
		}
		for _, token := range []ColorToken{ColorSemanticMenuHover, ColorSemanticMenuActive, ColorSemanticMenuSelected, ColorSemanticMenuPressed, ColorSemanticMenuDisabled} {
			if _, err := theme.color(TokenColor(token)); err != nil {
				return err
			}
		}
	}
	for _, value := range metrics {
		_, err := theme.metric(value)
		if err != nil {
			return err
		}
	}
	for _, child := range view.node.children {
		if err := validateViewTheme(child, theme); err != nil {
			return err
		}
	}
	return nil
}

type namedStatePatch struct {
	name  string
	patch StylePatch
}

func namedStatePatches(value StateStyles) [6]namedStatePatch {
	return [6]namedStatePatch{
		{name: "default", patch: value.Default},
		{name: "hover", patch: value.Hover},
		{name: "focus", patch: value.Focus},
		{name: "checked", patch: value.Checked},
		{name: "pressed", patch: value.Pressed},
		{name: "disabled", patch: value.Disabled},
	}
}

type visualState struct{ Hover, Focus, Disabled, Pressed, Checked bool }

type stateApplicability uint8

const (
	stateHover stateApplicability = 1 << iota
	stateFocus
	stateDisabled
	statePressed
	stateChecked
)

func resolvedStatePatch(component, local StateStyles, state visualState, applies stateApplicability) StylePatch {
	result := mergeStylePatch(StylePatch{}, component.Default)
	result = mergeStylePatch(result, local.Default)
	if state.Disabled && applies&stateDisabled != 0 {
		// Disabled suppresses all interaction states. Checked is controlled
		// value state rather than interaction and remains visible underneath.
		if state.Checked && applies&stateChecked != 0 {
			result = mergeStylePatch(result, component.Checked)
			result = mergeStylePatch(result, local.Checked)
		}
		result = mergeStylePatch(result, component.Disabled)
		return mergeStylePatch(result, local.Disabled)
	}
	if state.Hover && applies&stateHover != 0 {
		result = mergeStylePatch(result, component.Hover)
		result = mergeStylePatch(result, local.Hover)
	}
	if state.Focus && applies&stateFocus != 0 {
		result = mergeStylePatch(result, component.Focus)
		result = mergeStylePatch(result, local.Focus)
	}
	if state.Checked && applies&stateChecked != 0 {
		result = mergeStylePatch(result, component.Checked)
		result = mergeStylePatch(result, local.Checked)
	}
	if state.Pressed && applies&statePressed != 0 {
		result = mergeStylePatch(result, component.Pressed)
		result = mergeStylePatch(result, local.Pressed)
	}
	return result
}

func resolvedActiveStatePatch(component, local StateStyles, state visualState, applies stateApplicability) StylePatch {
	component.Default, local.Default = StylePatch{}, StylePatch{}
	return resolvedStatePatch(component, local, state, applies)
}

func mergeStylePatch(base, overlay StylePatch) StylePatch {
	if _, set := overlay.Background.get(); set {
		base.Background = overlay.Background
	}
	if _, set := overlay.Border.get(); set {
		base.Border = overlay.Border
	}
	if _, set := overlay.Radius.get(); set {
		base.Radius = overlay.Radius
	}
	if _, set := overlay.Shadow.get(); set {
		base.Shadow = overlay.Shadow
	}
	if _, set := overlay.Opacity.get(); set {
		base.Opacity = overlay.Opacity
	}
	if _, set := overlay.Visibility.get(); set {
		base.Visibility = overlay.Visibility
	}
	if _, set := overlay.TextColor.get(); set {
		base.TextColor = overlay.TextColor
	}
	return base
}

func resolvedThemesEqual(left, right resolvedTheme) bool {
	return reflect.DeepEqual(left.source, right.source)
}

func displayListsEqual(left, right []paint.Command) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		leftCommand, rightCommand := left[index], right[index]
		leftText, rightText := leftCommand.Text, rightCommand.Text
		leftImage, rightImage := leftCommand.Image, rightCommand.Image
		leftCommand.Text, rightCommand.Text = nil, nil
		leftCommand.Image, rightCommand.Image = nil, nil
		if leftCommand != rightCommand {
			return false
		}
		if (leftText == nil) != (rightText == nil) {
			return false
		}
		if leftText != nil && (leftText.Key != rightText.Key || leftText.Width != rightText.Width || leftText.Height != rightText.Height) {
			return false
		}
		if (leftImage == nil) != (rightImage == nil) {
			return false
		}
		if leftImage != nil && (leftImage.Key != rightImage.Key || leftImage.Width != rightImage.Width || leftImage.Height != rightImage.Height) {
			return false
		}
	}
	return true
}

func themeChangesLayout(view View, before, after resolvedTheme, inherited ...buttonTypography) bool {
	if view.node == nil {
		return false
	}
	props, err := nodeProps(view)
	if err != nil {
		return false
	}
	typography := inheritedButtonTypography(view, props.Style.Text, inherited)
	if view.node.kind == viewText {
		props.Style.Text = typography.apply(props.Style.Text)
	}
	values := []MetricValue{
		props.Style.Margin.Top, props.Style.Margin.Right,
		props.Style.Margin.Bottom, props.Style.Margin.Left,
		props.Style.Padding.Top, props.Style.Padding.Right,
		props.Style.Padding.Bottom, props.Style.Padding.Left,
	}
	if nodeUsesTextMetrics(view.node.kind) {
		values = append(values,
			textSizeMetric(props.Style.Text.Size),
			lineHeightMetric(props.Style.Text.LineHeight),
		)
	}
	if view.node.kind == viewButtonGroup {
		if view.node.buttonGroup.Dividers {
			values = append(values, TokenMetric(MetricComponentButtonGroupBorderWidth))
		}
	}
	if view.node.kind == viewInputGroup {
		values = append(values, TokenMetric(MetricComponentInputGroupGap))
	}
	if props.Style.Height.kind == lengthAuto && props.Style.MinHeight.kind == lengthAuto {
		switch view.node.kind {
		case viewButton, viewInput, viewInputGroup, viewCheckbox, viewRadio, viewSelect:
			values = append(values, TokenMetric(MetricSemanticControlHeight))
		case viewBadge:
			values = append(values, TokenMetric(MetricComponentBadgeMinHeight))
		case viewTabs:
			values = append(values, TokenMetric(MetricComponentTabsHeight))
		case viewTextarea:
			values = append(values, TokenMetric(MetricComponentTextareaMinHeight))
		}
	}
	if view.node.kind == viewIcon {
		values = append(values, iconSizeMetric(view.node.icon.Size))
	}
	if view.node.kind == viewAvatar && (props.Style.Width.kind == lengthAuto || props.Style.Height.kind == lengthAuto) {
		values = append(values, avatarSizeMetric(view.node.avatar.Size))
	}
	if view.node.kind == viewInput && view.node.input.Password && view.node.input.ShowPasswordToggle {
		values = append(values, TokenMetric(MetricComponentIconSize))
	}
	if view.node.kind == viewToggleSwitch {
		values = append(values, TokenMetric(MetricComponentToggleWidth), TokenMetric(MetricComponentToggleHeight))
	}
	if view.node.kind == viewSlider {
		if props.Style.Width.kind == lengthAuto {
			values = append(values, TokenMetric(MetricComponentSliderWidth))
		}
		if props.Style.Height.kind == lengthAuto {
			values = append(values, TokenMetric(MetricComponentSliderHeight))
		}
	}
	if view.node.kind == viewProgressBar {
		if props.Style.Width.kind == lengthAuto {
			values = append(values, TokenMetric(MetricComponentProgressBarWidth))
		}
		if props.Style.Height.kind == lengthAuto {
			values = append(values, TokenMetric(MetricComponentProgressBarHeight))
		}
	}
	if view.node.kind == viewCheckbox {
		values = append(values, TokenMetric(MetricComponentCheckboxSize), TokenMetric(MetricComponentCheckboxGap))
	}
	if view.node.kind == viewRadio {
		values = append(values, TokenMetric(MetricComponentRadioSize), TokenMetric(MetricComponentRadioGap))
	}
	if view.node.kind == viewSelect {
		values = append(values, TokenMetric(MetricComponentSelectItemHeight), TokenMetric(MetricComponentSelectMaxHeight), TokenMetric(MetricComponentSelectGap))
	}
	if view.node.kind == viewTabs {
		values = append(values,
			TokenMetric(MetricComponentTabsHeight), TokenMetric(MetricComponentTabsGap),
			TokenMetric(MetricComponentTabsPaddingX), TokenMetric(MetricComponentTabsPaddingY),
			TokenMetric(MetricComponentTabsIndicator), TokenMetric(MetricComponentTabsIndicatorInset),
		)
	}
	if view.node.kind == viewMenu {
		values = append(values,
			TokenMetric(MetricComponentMenuItemHeight), TokenMetric(MetricComponentMenuGap),
			TokenMetric(MetricComponentMenuPaddingX), TokenMetric(MetricComponentMenuPaddingY),
		)
	}
	for _, value := range values {
		if !value.isToken {
			continue
		}
		left, leftErr := before.metric(value)
		right, rightErr := after.metric(value)
		if leftErr != nil || rightErr != nil || left != right {
			return true
		}
	}
	for _, child := range view.node.children {
		if themeChangesLayout(child, before, after, typography) {
			return true
		}
	}
	return false
}
