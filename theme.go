package dxui

const (
	ColorPrimitiveWhite              ColorToken = "primitive.white"
	ColorPrimitiveBlack              ColorToken = "primitive.black"
	ColorPrimitiveBlue               ColorToken = "primitive.blue"
	ColorPrimitiveGray               ColorToken = "primitive.gray"
	ColorSemanticSurface             ColorToken = "semantic.surface"
	ColorSemanticSurfaceHi           ColorToken = "semantic.surface.high"
	ColorSemanticText                ColorToken = "semantic.text"
	ColorSemanticAccent              ColorToken = "semantic.accent"
	ColorSemanticAccentHover         ColorToken = "semantic.accent.hover"
	ColorSemanticOnAccent            ColorToken = "semantic.on-accent"
	ColorSemanticFocusRing           ColorToken = "semantic.focus-ring"
	ColorSemanticDanger              ColorToken = "semantic.danger"
	ColorSemanticSuccess             ColorToken = "semantic.success"
	ColorSemanticInfo                ColorToken = "semantic.info"
	ColorSemanticWarn                ColorToken = "semantic.warn"
	ColorSemanticBorder              ColorToken = "semantic.border"
	ColorSemanticShadow              ColorToken = "semantic.shadow"
	ColorSemanticScrollTrack         ColorToken = "semantic.scroll-track"
	ColorSemanticScrollThumb         ColorToken = "semantic.scroll-thumb"
	ColorSemanticScrollThumbHover    ColorToken = "semantic.scroll-thumb-hover"
	ColorSemanticScrollThumbActive   ColorToken = "semantic.scroll-thumb-active"
	ColorSemanticScrollThumbDisabled ColorToken = "semantic.scroll-thumb-disabled"
	ColorSemanticSelectHover         ColorToken = "semantic.select-hover"
	ColorSemanticSelectSelected      ColorToken = "semantic.select-selected"
	ColorSemanticSelectDisabled      ColorToken = "semantic.select-disabled"
	ColorSemanticSliderTrack         ColorToken = "semantic.slider-track"
	ColorSemanticSliderFill          ColorToken = "semantic.slider-fill"
	ColorSemanticSliderThumb         ColorToken = "semantic.slider-thumb"
	ColorSemanticSliderThumbBorder   ColorToken = "semantic.slider-thumb-border"
	ColorSemanticProgressTrack       ColorToken = "semantic.progress-track"
	ColorSemanticProgressFill        ColorToken = "semantic.progress-fill"
	ColorSemanticTabsHover           ColorToken = "semantic.tabs-hover"
	ColorSemanticTabsPressed         ColorToken = "semantic.tabs-pressed"
	ColorSemanticTabsSelected        ColorToken = "semantic.tabs-selected"
	ColorSemanticTabsDisabled        ColorToken = "semantic.tabs-disabled"
	ColorSemanticTabsIndicator       ColorToken = "semantic.tabs-indicator"
	ColorSemanticMenuHover           ColorToken = "semantic.menu-hover"
	ColorSemanticMenuActive          ColorToken = "semantic.menu-active"
	ColorSemanticMenuSelected        ColorToken = "semantic.menu-selected"
	ColorSemanticMenuPressed         ColorToken = "semantic.menu-pressed"
	ColorSemanticMenuDisabled        ColorToken = "semantic.menu-disabled"

	MetricPrimitive0                      MetricToken = "primitive.0"
	MetricPrimitive1                      MetricToken = "primitive.1"
	MetricPrimitive2                      MetricToken = "primitive.2"
	MetricSemanticRadius                  MetricToken = "semantic.radius"
	MetricSemanticBorder                  MetricToken = "semantic.border-width"
	MetricSemanticShadow                  MetricToken = "semantic.shadow-blur"
	MetricSemanticTextSize                MetricToken = "semantic.text-size"
	MetricSemanticLineHeight              MetricToken = "semantic.line-height"
	MetricSemanticControlHeight           MetricToken = "semantic.control-height"
	MetricComponentButtonPaddingX         MetricToken = "component.button.padding-x"
	MetricComponentButtonPaddingY         MetricToken = "component.button.padding-y"
	MetricComponentButtonGroupBorderWidth MetricToken = "component.button-group.border-width"
	MetricComponentButtonGroupRadius      MetricToken = "component.button-group.radius"
	MetricComponentBadgePaddingX          MetricToken = "component.badge.padding-x"
	MetricComponentBadgePaddingY          MetricToken = "component.badge.padding-y"
	MetricComponentBadgeMinHeight         MetricToken = "component.badge.min-height"
	MetricComponentBadgeRadius            MetricToken = "component.badge.radius"
	MetricComponentInputPaddingX          MetricToken = "component.input.padding-x"
	MetricComponentInputPaddingY          MetricToken = "component.input.padding-y"
	MetricComponentInputGroupPaddingX     MetricToken = "component.input-group.padding-x"
	MetricComponentInputGroupGap          MetricToken = "component.input-group.gap"
	MetricComponentTextareaMinHeight      MetricToken = "component.textarea.min-height"
	MetricComponentToggleWidth            MetricToken = "component.toggle.width"
	MetricComponentToggleHeight           MetricToken = "component.toggle.height"
	MetricComponentToggleKnobInset        MetricToken = "component.toggle.knob-inset"
	MetricComponentSliderWidth            MetricToken = "component.slider.width"
	MetricComponentSliderHeight           MetricToken = "component.slider.height"
	MetricComponentSliderTrackHeight      MetricToken = "component.slider.track-height"
	MetricComponentSliderThumbSize        MetricToken = "component.slider.thumb-size"
	MetricComponentProgressBarWidth       MetricToken = "component.progress-bar.width"
	MetricComponentProgressBarHeight      MetricToken = "component.progress-bar.height"
	MetricComponentProgressBarTrackHeight MetricToken = "component.progress-bar.track-height"
	MetricComponentProgressBarRadius      MetricToken = "component.progress-bar.radius"
	MetricComponentCheckboxSize           MetricToken = "component.checkbox.size"
	MetricComponentCheckboxGap            MetricToken = "component.checkbox.gap"
	MetricComponentRadioSize              MetricToken = "component.radio.size"
	MetricComponentRadioGap               MetricToken = "component.radio.gap"
	MetricComponentIconSize               MetricToken = "component.icon.size"
	MetricComponentAvatarSize             MetricToken = "component.avatar.size"
	MetricComponentScrollThickness        MetricToken = "component.scroll.thickness"
	MetricComponentScrollMinThumb         MetricToken = "component.scroll.min-thumb"
	MetricComponentScrollInset            MetricToken = "component.scroll.inset"
	MetricComponentSelectPaddingX         MetricToken = "component.select.padding-x"
	MetricComponentSelectPaddingY         MetricToken = "component.select.padding-y"
	MetricComponentSelectItemHeight       MetricToken = "component.select.item-height"
	MetricComponentSelectMaxHeight        MetricToken = "component.select.max-height"
	MetricComponentSelectGap              MetricToken = "component.select.gap"
	MetricComponentPopoverPaddingX        MetricToken = "component.popover.padding-x"
	MetricComponentPopoverPaddingY        MetricToken = "component.popover.padding-y"
	MetricComponentPopoverGap             MetricToken = "component.popover.gap"
	MetricComponentTooltipPaddingX        MetricToken = "component.tooltip.padding-x"
	MetricComponentTooltipPaddingY        MetricToken = "component.tooltip.padding-y"
	MetricComponentTooltipGap             MetricToken = "component.tooltip.gap"
	MetricComponentTabsHeight             MetricToken = "component.tabs.height"
	MetricComponentTabsGap                MetricToken = "component.tabs.gap"
	MetricComponentTabsPaddingX           MetricToken = "component.tabs.padding-x"
	MetricComponentTabsPaddingY           MetricToken = "component.tabs.padding-y"
	MetricComponentTabsIndicator          MetricToken = "component.tabs.indicator-height"
	MetricComponentTabsIndicatorInset     MetricToken = "component.tabs.indicator-inset"
	MetricComponentMenuItemHeight         MetricToken = "component.menu.item-height"
	MetricComponentMenuGap                MetricToken = "component.menu.gap"
	MetricComponentMenuPaddingX           MetricToken = "component.menu.padding-x"
	MetricComponentMenuPaddingY           MetricToken = "component.menu.padding-y"

	ComponentPanel           ComponentToken = "panel"
	ComponentText            ComponentToken = "text"
	ComponentButton          ComponentToken = "button"
	ComponentButtonSecondary ComponentToken = "button.secondary"
	ComponentButtonDanger    ComponentToken = "button.danger"
	ComponentButtonGhost     ComponentToken = "button.ghost"
	ComponentButtonGroup     ComponentToken = "button-group"
	ComponentBadge           ComponentToken = "badge"
	ComponentInput           ComponentToken = "input"
	ComponentInputGroup      ComponentToken = "input-group"
	ComponentToggleSwitch    ComponentToken = "toggle-switch"
	ComponentSlider          ComponentToken = "slider"
	ComponentProgressBar     ComponentToken = "progress-bar"
	ComponentCheckbox        ComponentToken = "checkbox"
	ComponentRadio           ComponentToken = "radio"
	ComponentIcon            ComponentToken = "icon"
	ComponentImage           ComponentToken = "image"
	ComponentAvatar          ComponentToken = "avatar"
	ComponentScroll          ComponentToken = "scroll"
	ComponentSelect          ComponentToken = "select"
	ComponentTabs            ComponentToken = "tabs"
	ComponentMenu            ComponentToken = "menu"
	ComponentPopover         ComponentToken = "popover"
	ComponentTooltip         ComponentToken = "tooltip"
)

// LightTheme returns an independent, mutable-by-the-caller light theme value.
// App.SetTheme copies it before use.
func LightTheme() Theme {
	return builtinTheme(false)
}

// DarkTheme returns an independent, mutable-by-the-caller dark theme value.
func DarkTheme() Theme {
	return builtinTheme(true)
}

func builtinTheme(dark bool) Theme {
	primitive := builtinPrimitiveColors()
	primitive[ColorPrimitiveWhite] = RGBA(248, 250, 252, 255)
	primitive[ColorPrimitiveBlack] = RGBA(20, 24, 31, 255)
	primitive[ColorPrimitiveBlue] = RGBA(55, 118, 232, 255)
	primitive[ColorPrimitiveGray] = RGBA(113, 122, 138, 255)
	surface := Color.Primitive.Slate50
	surfaceHigh := Color.Primitive.White
	text := Color.Primitive.Slate950
	accent := Color.Primitive.Emerald600
	accentHover := Color.Primitive.Emerald700
	onAccent := Color.Primitive.White
	danger := Color.Primitive.Red600
	success := Color.Primitive.Green600
	info := Color.Primitive.Sky600
	warn := Color.Primitive.Amber600
	border := Color.Primitive.Slate400
	if dark {
		surface = Color.Primitive.Slate950
		surfaceHigh = Color.Primitive.Slate900
		text = Color.Primitive.Slate50
		accent = Color.Primitive.Emerald400
		accentHover = Color.Primitive.Emerald300
		onAccent = Color.Primitive.Slate950
		danger = Color.Primitive.Red400
		success = Color.Primitive.Green400
		info = Color.Primitive.Sky400
		warn = Color.Primitive.Amber400
		border = Color.Primitive.Slate600
	}
	panelBase := StylePatch{
		Background: Some(TokenColor(ColorSemanticSurfaceHi)),
		Border:     Some(Border{Width: TokenMetric(MetricSemanticBorder), Color: TokenColor(ColorSemanticBorder)}),
		Radius:     Some(UniformCorners(TokenMetric(MetricSemanticRadius))),
		TextColor:  Some(TokenColor(ColorSemanticText)),
	}
	buttonBase := StylePatch{
		Background: Some(TokenColor(ColorSemanticAccent)),
		Radius:     Some(UniformCorners(TokenMetric(MetricSemanticRadius))),
		TextColor:  Some(TokenColor(ColorSemanticOnAccent)),
	}
	focusPatch := StylePatch{Shadow: Some([]Shadow{{
		Blur: Metric(0), Spread: Metric(2), Color: TokenColor(ColorSemanticFocusRing),
	}})}
	focusedTextPatch := focusPatch
	focusedTextPatch.TextColor = Some(TokenColor(ColorSemanticAccent))
	secondaryButtonBase := StylePatch{
		Background: Some(TokenColor(ColorSemanticSurfaceHi)),
		Border:     Some(Border{Width: TokenMetric(MetricSemanticBorder), Color: TokenColor(ColorSemanticBorder)}),
		Radius:     Some(UniformCorners(TokenMetric(MetricSemanticRadius))),
		TextColor:  Some(TokenColor(ColorSemanticText)),
	}
	dangerButtonBase := StylePatch{
		Background: Some(TokenColor(ColorSemanticDanger)),
		Radius:     Some(UniformCorners(TokenMetric(MetricSemanticRadius))),
		TextColor:  Some(TokenColor(ColorSemanticOnAccent)),
	}
	ghostButtonBase := StylePatch{
		Background: Some(ColorRGBA(0, 0, 0, 0)),
		Radius:     Some(UniformCorners(TokenMetric(MetricSemanticRadius))),
		TextColor:  Some(TokenColor(ColorSemanticAccent)),
	}
	buttonGroupBase := StylePatch{
		Border: Some(Border{Width: TokenMetric(MetricComponentButtonGroupBorderWidth), Color: TokenColor(ColorSemanticBorder)}),
		Radius: Some(UniformCorners(TokenMetric(MetricComponentButtonGroupRadius))),
	}
	badgeBase := StylePatch{
		Background: Some(TokenColor(ColorSemanticAccent)),
		Radius:     Some(UniformCorners(TokenMetric(MetricComponentBadgeRadius))),
		TextColor:  Some(TokenColor(ColorSemanticOnAccent)),
	}
	progressBase := StylePatch{
		Background: Some(TokenColor(ColorSemanticProgressTrack)),
		Radius:     Some(UniformCorners(TokenMetric(MetricComponentProgressBarRadius))),
		TextColor:  Some(TokenColor(ColorSemanticProgressFill)),
	}
	overlayBase := panelBase
	overlayBase.Shadow = Some([]Shadow{{
		OffsetY: Metric(2), Blur: TokenMetric(MetricSemanticShadow),
		Color: TokenColor(ColorSemanticShadow),
	}})
	return Theme{
		Primitive: PrimitiveTokens{
			Colors: primitive,
			Metrics: map[MetricToken]float32{
				MetricPrimitive0: 0, MetricPrimitive1: 1, MetricPrimitive2: 8,
			},
		},
		Semantic: SemanticTokens{
			Colors: map[ColorToken]ColorValue{
				ColorSemanticSurface: TokenColor(surface), ColorSemanticSurfaceHi: TokenColor(surfaceHigh),
				ColorSemanticText: TokenColor(text), ColorSemanticAccent: TokenColor(accent),
				ColorSemanticAccentHover: TokenColor(accentHover), ColorSemanticOnAccent: TokenColor(onAccent),
				ColorSemanticFocusRing: TokenColor(Color.Primitive.White),
				ColorSemanticDanger:    TokenColor(danger), ColorSemanticSuccess: TokenColor(success),
				ColorSemanticInfo: TokenColor(info), ColorSemanticWarn: TokenColor(warn),
				ColorSemanticBorder:              TokenColor(border),
				ColorSemanticShadow:              ColorRGBA(0, 0, 0, 72),
				ColorSemanticScrollTrack:         ColorRGBA(0, 0, 0, 0),
				ColorSemanticScrollThumb:         TokenColor(ColorSemanticBorder),
				ColorSemanticScrollThumbHover:    TokenColor(ColorSemanticBorder),
				ColorSemanticScrollThumbActive:   TokenColor(ColorSemanticBorder),
				ColorSemanticScrollThumbDisabled: ColorRGBA(113, 122, 138, 80),
				ColorSemanticSelectHover:         ColorRGBA(55, 118, 232, 42),
				ColorSemanticSelectSelected:      ColorRGBA(55, 118, 232, 82),
				ColorSemanticSelectDisabled:      ColorRGBA(113, 122, 138, 150),
				ColorSemanticSliderTrack:         ColorRGBA(113, 122, 138, 150),
				ColorSemanticSliderFill:          TokenColor(ColorSemanticAccent),
				ColorSemanticSliderThumb:         TokenColor(ColorSemanticSurfaceHi),
				ColorSemanticSliderThumbBorder:   TokenColor(ColorSemanticAccent),
				ColorSemanticProgressTrack:       ColorRGBA(113, 122, 138, 90),
				ColorSemanticProgressFill:        TokenColor(ColorSemanticAccent),
				ColorSemanticTabsHover:           ColorRGBA(55, 118, 232, 42),
				ColorSemanticTabsPressed:         ColorRGBA(55, 118, 232, 72),
				ColorSemanticTabsSelected:        TokenColor(ColorSemanticAccent),
				ColorSemanticTabsDisabled:        ColorRGBA(113, 122, 138, 150),
				ColorSemanticTabsIndicator:       TokenColor(ColorSemanticAccent),
				ColorSemanticMenuHover:           ColorRGBA(55, 118, 232, 42),
				ColorSemanticMenuActive:          ColorRGBA(55, 118, 232, 26),
				ColorSemanticMenuSelected:        ColorRGBA(55, 118, 232, 82),
				ColorSemanticMenuPressed:         ColorRGBA(55, 118, 232, 72),
				ColorSemanticMenuDisabled:        ColorRGBA(113, 122, 138, 150),
			},
			Metrics: map[MetricToken]MetricValue{
				MetricSemanticRadius:                  TokenMetric(MetricPrimitive2),
				MetricSemanticBorder:                  TokenMetric(MetricPrimitive1),
				MetricSemanticShadow:                  TokenMetric(MetricPrimitive2),
				MetricSemanticTextSize:                Metric(15),
				MetricSemanticLineHeight:              Metric(20),
				MetricSemanticControlHeight:           Metric(36),
				MetricComponentButtonPaddingX:         Metric(14),
				MetricComponentButtonPaddingY:         Metric(8),
				MetricComponentButtonGroupBorderWidth: TokenMetric(MetricSemanticBorder),
				MetricComponentButtonGroupRadius:      TokenMetric(MetricSemanticRadius),
				MetricComponentBadgePaddingX:          Metric(8),
				MetricComponentBadgePaddingY:          Metric(2),
				MetricComponentBadgeMinHeight:         Metric(24),
				MetricComponentBadgeRadius:            Metric(999),
				MetricComponentInputPaddingX:          Metric(12),
				MetricComponentInputPaddingY:          Metric(8),
				MetricComponentInputGroupPaddingX:     Metric(12),
				MetricComponentInputGroupGap:          Metric(8),
				MetricComponentTextareaMinHeight:      Metric(80),
				MetricComponentToggleWidth:            Metric(40),
				MetricComponentToggleHeight:           Metric(24),
				MetricComponentToggleKnobInset:        Metric(2),
				MetricComponentSliderWidth:            Metric(160),
				MetricComponentSliderHeight:           Metric(24),
				MetricComponentSliderTrackHeight:      Metric(4),
				MetricComponentSliderThumbSize:        Metric(18),
				MetricComponentProgressBarWidth:       Metric(160),
				MetricComponentProgressBarHeight:      Metric(12),
				MetricComponentProgressBarTrackHeight: Metric(8),
				MetricComponentProgressBarRadius:      Metric(999),
				MetricComponentCheckboxSize:           Metric(18),
				MetricComponentCheckboxGap:            Metric(8),
				MetricComponentRadioSize:              Metric(18),
				MetricComponentRadioGap:               Metric(8),
				MetricComponentIconSize:               Metric(20),
				MetricComponentAvatarSize:             Metric(40),
				MetricComponentScrollThickness:        Metric(6),
				MetricComponentScrollMinThumb:         Metric(24),
				MetricComponentScrollInset:            Metric(2),
				MetricComponentSelectPaddingX:         Metric(12),
				MetricComponentSelectPaddingY:         Metric(8),
				MetricComponentSelectItemHeight:       Metric(36),
				MetricComponentSelectMaxHeight:        Metric(240),
				MetricComponentSelectGap:              Metric(4),
				MetricComponentPopoverPaddingX:        Metric(12),
				MetricComponentPopoverPaddingY:        Metric(10),
				MetricComponentPopoverGap:             Metric(6),
				MetricComponentTooltipPaddingX:        Metric(8),
				MetricComponentTooltipPaddingY:        Metric(6),
				MetricComponentTooltipGap:             Metric(6),
				MetricComponentTabsHeight:             Metric(40),
				MetricComponentTabsGap:                Metric(4),
				MetricComponentTabsPaddingX:           Metric(12),
				MetricComponentTabsPaddingY:           Metric(8),
				MetricComponentTabsIndicator:          Metric(2),
				MetricComponentTabsIndicatorInset:     Metric(2),
				MetricComponentMenuItemHeight:         Metric(36),
				MetricComponentMenuGap:                Metric(4),
				MetricComponentMenuPaddingX:           Metric(12),
				MetricComponentMenuPaddingY:           Metric(8),
			},
		},
		Components: map[ComponentToken]ComponentTheme{
			ComponentPanel: {Base: panelBase},
			ComponentText:  {Base: StylePatch{TextColor: Some(TokenColor(ColorSemanticText))}},
			ComponentButton: {
				Base: buttonBase,
				States: StateStyles{
					Hover:    StylePatch{Background: Some(TokenColor(ColorSemanticAccentHover)), Opacity: Some(float32(.92))},
					Focus:    focusPatch,
					Pressed:  StylePatch{Opacity: Some(float32(.78))},
					Disabled: StylePatch{Opacity: Some(float32(.42))},
				},
			},
			ComponentButtonSecondary: {Base: secondaryButtonBase, States: StateStyles{
				Hover: StylePatch{Background: Some(TokenColor(ColorSemanticSelectHover))}, Focus: focusPatch,
				Pressed: StylePatch{Opacity: Some(float32(.78))}, Disabled: StylePatch{Opacity: Some(float32(.42))},
			}},
			ComponentButtonDanger: {Base: dangerButtonBase, States: StateStyles{
				Hover: StylePatch{Opacity: Some(float32(.88))}, Focus: focusPatch,
				Pressed: StylePatch{Opacity: Some(float32(.72))}, Disabled: StylePatch{Opacity: Some(float32(.42))},
			}},
			ComponentButtonGhost: {Base: ghostButtonBase, States: StateStyles{
				Hover: StylePatch{Background: Some(TokenColor(ColorSemanticSelectHover))}, Focus: copyStylePatch(buttonRecipeFocus),
				Pressed: StylePatch{Opacity: Some(float32(.72))}, Disabled: StylePatch{Opacity: Some(float32(.42))},
			}},
			ComponentButtonGroup: {Base: buttonGroupBase},
			ComponentBadge:       {Base: badgeBase},
			ComponentInput: {
				Base: panelBase,
				States: StateStyles{
					Hover:    StylePatch{Border: Some(Border{Width: TokenMetric(MetricSemanticBorder), Color: TokenColor(ColorSemanticAccentHover)})},
					Focus:    focusPatch,
					Disabled: StylePatch{Opacity: Some(float32(.48))},
				},
			},
			ComponentInputGroup: {
				Base: panelBase,
				States: StateStyles{
					Hover: StylePatch{Border: Some(Border{Width: TokenMetric(MetricSemanticBorder), Color: TokenColor(ColorSemanticAccentHover)})},
					Focus: focusPatch,
				},
			},
			ComponentToggleSwitch: {
				Base: StylePatch{Background: Some(TokenColor(ColorSemanticBorder)), Radius: Some(UniformCorners(Metric(12)))},
				States: StateStyles{
					Hover:    StylePatch{Border: Some(Border{Width: TokenMetric(MetricSemanticBorder), Color: TokenColor(ColorSemanticAccentHover)})},
					Focus:    focusPatch,
					Checked:  StylePatch{Background: Some(TokenColor(ColorSemanticAccent))},
					Pressed:  StylePatch{Opacity: Some(float32(.72))},
					Disabled: StylePatch{Opacity: Some(float32(.42))},
				},
			},
			ComponentSlider: {
				States: StateStyles{
					Hover:    StylePatch{Opacity: Some(float32(.9))},
					Focus:    focusPatch,
					Pressed:  StylePatch{Opacity: Some(float32(.72))},
					Disabled: StylePatch{Opacity: Some(float32(.42))},
				},
			},
			ComponentProgressBar: {Base: progressBase},
			ComponentCheckbox: {
				Base: StylePatch{TextColor: Some(TokenColor(ColorSemanticText)), Radius: Some(UniformCorners(TokenMetric(MetricSemanticRadius)))},
				States: StateStyles{
					Hover:    StylePatch{TextColor: Some(TokenColor(ColorSemanticAccentHover))},
					Focus:    focusedTextPatch,
					Checked:  StylePatch{TextColor: Some(TokenColor(ColorSemanticAccent))},
					Pressed:  StylePatch{Opacity: Some(float32(.72))},
					Disabled: StylePatch{Opacity: Some(float32(.42))},
				},
			},
			ComponentRadio: {
				Base: StylePatch{TextColor: Some(TokenColor(ColorSemanticText)), Radius: Some(UniformCorners(TokenMetric(MetricSemanticRadius)))},
				States: StateStyles{
					Hover:    StylePatch{TextColor: Some(TokenColor(ColorSemanticAccentHover))},
					Focus:    focusedTextPatch,
					Checked:  StylePatch{TextColor: Some(TokenColor(ColorSemanticAccent))},
					Pressed:  StylePatch{Opacity: Some(float32(.72))},
					Disabled: StylePatch{Opacity: Some(float32(.42))},
				},
			},
			ComponentIcon:   {Base: StylePatch{TextColor: Some(TokenColor(ColorSemanticText))}},
			ComponentImage:  {},
			ComponentAvatar: {},
			ComponentScroll: {},
			ComponentSelect: {
				Base: panelBase,
				States: StateStyles{
					Hover:    StylePatch{Border: Some(Border{Width: TokenMetric(MetricSemanticBorder), Color: TokenColor(ColorSemanticAccentHover)})},
					Focus:    focusPatch,
					Disabled: StylePatch{Opacity: Some(float32(.48))},
				},
			},
			ComponentPopover: {Base: overlayBase},
			ComponentTooltip: {Base: overlayBase},
			ComponentTabs: {
				Base: StylePatch{TextColor: Some(TokenColor(ColorSemanticText))},
				States: StateStyles{
					Hover:    StylePatch{TextColor: Some(TokenColor(ColorSemanticAccentHover))},
					Focus:    focusPatch,
					Pressed:  StylePatch{Opacity: Some(float32(.82))},
					Disabled: StylePatch{Opacity: Some(float32(.48))},
				},
			},
			ComponentMenu: {
				Base: StylePatch{
					Background: Some(TokenColor(ColorSemanticSurfaceHi)),
					Radius:     Some(UniformCorners(TokenMetric(MetricSemanticRadius))),
					TextColor:  Some(TokenColor(ColorSemanticText)),
				},
				States: StateStyles{
					Focus:    focusPatch,
					Pressed:  StylePatch{Opacity: Some(float32(.82))},
					Disabled: StylePatch{Opacity: Some(float32(.48))},
				},
			},
		},
	}
}
