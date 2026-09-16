package main

import (
	"fmt"
	"time"

	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon"
)

func controlExamples(app *dxui.App, state *galleryState, assets galleryAssets) []componentExample {
	return []componentExample{
		{
			Name: "Button", Category: "Input & actions", Purpose: "A focusable semantic action with pointer, Enter, and Space activation and one arbitrary child view.",
			Coverage: []string{"default", "sizes", "colors", "soft/outline/dash/ghost/link appearances", "disabled and PointerNone", "square/circle shapes", "icon content", "loading spinner", "hover/focus/pressed states"},
			Build:    func() []exampleBlock { return buttonBlocks(state, assets) },
		},
		{
			Name: "ButtonGroup", Category: "Input & actions", Purpose: "A non-focusable horizontal or vertical arrangement of independently interactive Button views.",
			Coverage: []string{"optional dividers", "default connected layout without dividers", "equal-width vertical", "mixed Disabled", "independent hover/focus/pressed and Tab order"},
			Build:    func() []exampleBlock { return buttonGroupBlocks(state) },
		},
		{
			Name: "InputGroup", Category: "Input & actions", Purpose: "Combines one controlled Input with optional text, icon, or Button adornments under one border and focus surface.",
			Coverage: []string{"required Input", "optional Prefix/Suffix View", "search", "prefix icon", "suffix unit", "independent action Button", "group focus-within style", "remaining-width Input"},
			Build:    func() []exampleBlock { return inputGroupBlocks(state) },
		},
		{
			Name: "Input", Category: "Input & actions", Purpose: "A controlled single-line UTF-8 editor with rune-indexed selection, placeholder/password modes, clipboard shortcuts, and semantic submit.",
			Coverage: []string{"direct common fields and keyed reorder", "Value empty/non-empty", "OnChange callback/nil", "Selection unset/set", "OnSelectionChange", "Placeholder", "Password false/true", "ShowPasswordToggle false/true", "Disabled false/true", "ReadOnly false/true", "OnSubmit", "controlled form composition"},
			Build:    func() []exampleBlock { return inputBlocks(state) },
		},
		{
			Name: "Textarea", Category: "Input & actions", Purpose: "A controlled multiline editor with explicit newlines, rune-indexed selection, simple word wrapping, and internal caret scrolling.",
			Coverage: []string{"direct common fields", "Value/OnChange", "Selection/OnSelectionChange", "Placeholder", "Disabled", "ReadOnly", "Wrap: no-wrap/words", "empty, error-style, long content"},
			Build:    func() []exampleBlock { return textareaBlocks(state) },
		},
		{
			Name: "Select", Category: "Input & actions", Purpose: "A controlled custom popup select whose overlay escapes ancestor Scroll clips and supports pointer and keyboard navigation.",
			Coverage: []string{"direct common fields", "Value matched/empty/unmatched", "Options Value/Label/Disabled", "Placeholder", "Disabled false/true", "OnChange callback/nil", "selected/open/active/empty options"},
			Build:    func() []exampleBlock { return selectBlocks(state) },
		},
		{
			Name: "Tabs", Category: "Input & actions", Purpose: "A controlled horizontal label selector; the application renders content for the authoritative Value.",
			Coverage: []string{"direct common fields", "Value matched/empty/unmatched", "Items Value/Label/Disabled", "Disabled false/true", "OnChange callback/nil", "hover/focus/selected/pressed/disabled", "Left/Right, Home/End, Enter/Space, Tab"},
			Build:    func() []exampleBlock { return tabsBlocks(state) },
		},
		{
			Name: "Menu", Category: "Input & actions", Purpose: "An inline vertical or horizontal action list with one Tab entry, optional controlled selection, and no popup.",
			Coverage: []string{"direct common fields", "Value matched/empty/unmatched", "Orientation vertical/horizontal", "Items Value/Label/Disabled", "Disabled false/true", "OnAction callback/nil", "hover/focus/current/selected/pressed/disabled", "directional keys, Home/End, Enter/Space, Tab"},
			Build:    func() []exampleBlock { return menuBlocks(state) },
		},
		{
			Name: "Checkbox", Category: "Input & actions", Purpose: "A controlled two-state checkbox with one arbitrary label child; tri-state is intentionally not supported.",
			Coverage: []string{"direct common fields", "Checked false/true", "Disabled false/true", "OnChange callback/nil", "arbitrary label child", "hover/focus/pressed/checked/disabled"},
			Build:    func() []exampleBlock { return checkboxBlocks(state) },
		},
		{
			Name: "Radio", Category: "Input & actions", Purpose: "A controlled one-way selection control; applications provide mutual exclusion through one shared business value.",
			Coverage: []string{"direct common fields", "Selected false/true", "Disabled false/true", "OnSelect callback/nil", "arbitrary label child", "hover/focus/pressed/selected/disabled"},
			Build:    func() []exampleBlock { return radioBlocks(state) },
		},
		{
			Name: "ToggleSwitch", Category: "Input & actions", Purpose: "A compact controlled boolean switch with pointer/Space activation and theme-driven checked-state visuals.",
			Coverage: []string{"direct common fields", "Checked false/true", "Disabled false/true", "OnChange callback/nil", "hover/focus/pressed/checked/disabled"},
			Build:    func() []exampleBlock { return toggleBlocks(app, state) },
		},
		{
			Name: "Slider", Category: "Input & actions", Purpose: "A controlled single-value horizontal slider with track clicking, captured dragging, and stepped keyboard input.",
			Coverage: []string{"direct common fields", "Value", "Min", "Max", "Step", "Disabled", "OnChange callback/nil", "hover/focus/pressed/disabled", "track click, outside drag, arrows, Home/End"},
			Build:    func() []exampleBlock { return sliderBlocks(state) },
		},
		{
			Name: "Popover", Category: "Overlays", Purpose: "A controlled interactive window overlay positioned from an anchor, with flip, clamping, top-layer dismissal, and focus restoration.",
			Coverage: []string{"direct common fields", "Open controlled", "Placement", "Offset", "OnOpenChange", "anchor activation", "interactive content", "Escape/outside dismissal", "focus restore"},
			Build:    func() []exampleBlock { return popoverBlocks(state) },
		},
		{
			Name: "Tooltip", Category: "Overlays", Purpose: "A delayed non-interactive hint opened by anchor hover or keyboard focus without stealing focus or blocking the anchor.",
			Coverage: []string{"direct common fields", "Placement", "Offset", "Delay", "Disabled", "hover", "keyboard focus", "leave/blur", "non-interactive content"},
			Build:    func() []exampleBlock { return tooltipBlocks(state) },
		},
	}
}

func inputGroupBlocks(state *galleryState) []exampleBlock {
	style := dxui.Style{Width: dxui.Px(360)}
	input := func(key, placeholder string) dxui.View {
		return dxui.Input(dxui.InputProps{Key: key, Value: state.input, Placeholder: placeholder, OnChange: dxui.Assign(&state.input)})
	}
	return []exampleBlock{
		{
			Title:       "Search field",
			Description: "The required controlled Input retains editing, IME, selection, and submit behavior. InputGroup owns the single border and keyboard-visible focus ring.",
			Code: `dxui.InputGroup(dxui.InputGroupProps{}, dxui.InputGroupContent{
    Input: dxui.Input(dxui.InputProps{
        Value: query, Placeholder: "Search...",
        OnChange: dxui.Assign(&query),
    }),
    Prefix: dxui.Some(dxui.Icon(dxui.IconProps{Data: icon.Search()})),
})`,
			Preview: dxui.InputGroup(dxui.InputGroupProps{Style: style}, dxui.InputGroupContent{
				Input: input("input-group-search", "Search..."), Prefix: dxui.Some(dxui.Icon(dxui.IconProps{Data: icon.Search()})),
			}),
		},
		{
			Title:       "Prefix icon",
			Description: "A passive icon adds no focus stop. The Input expands into the remaining horizontal space.",
			Code: `dxui.InputGroup(dxui.InputGroupProps{}, dxui.InputGroupContent{
    Input:  emailInput,
    Prefix: dxui.Some(dxui.Icon(dxui.IconProps{Data: icon.Mail()})),
})`,
			Preview: dxui.InputGroup(dxui.InputGroupProps{Style: style}, dxui.InputGroupContent{
				Input: input("input-group-email", "Email address"), Prefix: dxui.Some(dxui.Icon(dxui.IconProps{Data: icon.Mail()})),
			}),
		},
		{
			Title:       "Suffix unit",
			Description: "Text can provide an application-owned unit without creating another editor or focus target.",
			Code: `dxui.InputGroup(dxui.InputGroupProps{}, dxui.InputGroupContent{
    Input:  amountInput,
    Suffix: dxui.Some(dxui.Label("kg")),
})`,
			Preview: dxui.InputGroup(dxui.InputGroupProps{Style: style}, dxui.InputGroupContent{
				Input: input("input-group-unit", "Weight"), Suffix: dxui.Some(dxui.Label("kg")),
			}),
		},
		{
			Title:       "Suffix action Button",
			Description: "A Button remains a distinct source-order Tab stop and keeps its normal pointer, Enter, Space, Disabled, and OnPress behavior.",
			Code: `dxui.InputGroup(dxui.InputGroupProps{}, dxui.InputGroupContent{
    Input: valueInput,
    Suffix: dxui.Some(dxui.Button(dxui.ButtonProps{OnPress: apply},
        dxui.Label("Apply"))),
})`,
			Preview: dxui.InputGroup(dxui.InputGroupProps{Style: style}, dxui.InputGroupContent{
				Input: input("input-group-action", "Command"),
				Suffix: dxui.Some(dxui.Button(dxui.ButtonProps{Key: "input-group-apply", OnPress: func() {
					state.feedback = fmt.Sprintf("InputGroup action -> %q", state.input)
				}}, dxui.Label("Apply"))),
			}),
		},
	}
}

func buttonGroupBlocks(state *galleryState) []exampleBlock {
	action := func(name string) func() {
		return func() { state.feedback = "ButtonGroup: " + name }
	}
	button := func(key, label string, disabled bool) dxui.View {
		return dxui.Button(dxui.ButtonProps{
			Key: key, Disabled: disabled, OnPress: action(label),
		}, buttonText(label, 15))
	}
	return []exampleBlock{
		{
			Title:       "Default ButtonGroup",
			Description: "The default ButtonGroup joins adjacent edges without drawing separators. Each Button remains an independent focus stop and action target.",
			Code: `dxui.ButtonGroup(dxui.ButtonGroupProps{},
    previousButton, currentButton, nextButton,
)`,
			Preview: dxui.ButtonGroup(dxui.ButtonGroupProps{}, button("default-previous", "Previous", false), button("default-current", "Current", false), button("default-next", "Next", false)),
		},
		{
			Title:       "ButtonGroup with Dividers",
			Description: "With dividers enabled, Buttons overlap one themed border width so each separator is painted once.",
			Code: `dxui.ButtonGroup(dxui.ButtonGroupProps{Dividers: true},
    leftButton, centerButton, rightButton,
)`,
			Preview: dxui.ButtonGroup(dxui.ButtonGroupProps{Dividers: true}, button("divider-left", "Left", false), button("divider-center", "Center", false), button("divider-right", "Right", false)),
		},
		{
			Title:       "Vertical ButtonGroup",
			Description: "Vertical stretches automatic child widths to the widest Button. Tab traversal remains source ordered and no arrow-key navigation is added.",
			Code: `dxui.ButtonGroup(dxui.ButtonGroupProps{
    Orientation: dxui.ButtonGroupVertical,
}, topButton, bottomButton)`,
			Preview: dxui.ButtonGroup(dxui.ButtonGroupProps{Orientation: dxui.ButtonGroupVertical}, button("vertical-top", "Top", false), button("vertical-bottom", "Bottom", false)),
		},
		{
			Title:       "Mixed Disabled Buttons",
			Description: "Disabled belongs to each child Button. Tab skips only the disabled child and the surrounding group never becomes focused.",
			Code: `dxui.ButtonGroup(dxui.ButtonGroupProps{},
    enabledButton,
    dxui.Button(dxui.ButtonProps{Disabled: true}, disabledLabel),
    anotherEnabledButton,
)`,
			Preview: dxui.ButtonGroup(dxui.ButtonGroupProps{}, button("mixed-one", "Enabled", false), button("mixed-disabled", "Disabled", true), button("mixed-three", "Enabled too", false)),
		},
	}
}

func popoverBlocks(state *galleryState) []exampleBlock {
	return []exampleBlock{{
		Title:       "Controlled interactive overlay",
		Description: "Activate the anchor with pointer, Enter, or Space. Tab enters the projected content; Escape or an outside primary click proposes false and restores anchor focus.",
		Code: `dxui.Box(dxui.BoxProps{Gap: 8},
    dxui.Popover(dxui.PopoverProps{
        Open: open, Placement: dxui.OverlayBottomStart,
        Offset: dxui.Metric(8),
        OnOpenChange: dxui.Assign(&open),
    }, anchor, content),
    dxui.Text(dxui.TextProps{Value: fmt.Sprintf("Current Open: %t", open)}),
)`,
		Preview: dxui.Box(dxui.BoxProps{Gap: 8},
			dxui.Popover(dxui.PopoverProps{
				Key: "gallery-popover", Open: state.popoverOpen,
				Placement: dxui.OverlayBottomStart, Offset: dxui.Metric(8),
				OnOpenChange: func(next bool) {
					state.popoverOpen = next
					state.feedback = fmt.Sprintf("Popover.OnOpenChange -> %t", next)
				},
			}, bodyText("Open popover"), dxui.Box(dxui.BoxProps{Gap: 8},
				bodyText("Interactive window-level content"),
				dxui.Button(dxui.ButtonProps{OnPress: func() { state.feedback = "Popover content button pressed." }}, bodyText("Content action")),
			)),
			bodyText(fmt.Sprintf("Current Open: %t", state.popoverOpen)),
		),
	}}
}

func tooltipBlocks(state *galleryState) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Hover and keyboard-focus delay",
			Description: "Hover or Tab-focus the anchor for 350 ms. The tooltip closes when both hover and focus leave, never becomes a tab stop, and cannot intercept the button action.",
			Code: `dxui.Tooltip(dxui.TooltipProps{
    Placement: dxui.OverlayTop, Delay: 350 * time.Millisecond,
}, anchor, dxui.Text(dxui.TextProps{Value: "Helpful context"}))`,
			Preview: dxui.Tooltip(dxui.TooltipProps{Key: "gallery-tooltip", Placement: dxui.OverlayTop, Delay: 350 * time.Millisecond},
				dxui.Button(dxui.ButtonProps{OnPress: func() { state.feedback = "Tooltip anchor button pressed." }}, bodyText("Hover or focus me")),
				bodyText("Helpful context without input capture")),
		},
		{
			Title:       "Disabled tooltip",
			Description: "Disabled suppresses hover/focus scheduling and removes any pending deadline or visible hint while leaving the anchor's normal input unchanged.",
			Code:        `dxui.Tooltip(dxui.TooltipProps{Disabled: true}, anchor, content)`,
			Preview: dxui.Tooltip(dxui.TooltipProps{Disabled: true},
				dxui.Button(dxui.ButtonProps{OnPress: func() { state.feedback = "Disabled-tooltip anchor still works." }}, bodyText("No tooltip")),
				bodyText("This hint stays hidden")),
		},
	}
}

func buttonBlocks(state *galleryState, assets galleryAssets) []exampleBlock {
	action := func(label string) func() {
		return func() {
			state.pressCount++
			state.feedback = fmt.Sprintf("%s button pressed (count=%d).", label, state.pressCount)
		}
	}
	return []exampleBlock{
		{Title: "Default", Description: "Zero values preserve the current default Button. Use the gallery theme switch to compare light and dark.",
			Code:    `dxui.TextButton(dxui.ButtonProps{OnPress: save}, "Default button")`,
			Preview: dxui.TextButton(dxui.ButtonProps{OnPress: action("Default button")}, "Default button"),
		},
		{Title: "Button sizes", Description: "Size supplies padding, minimum height, and inherited text metrics; explicit Style and child text metrics win.",
			Code: `dxui.TextButton(dxui.ButtonProps{Size: dxui.ButtonSmall}, "Small")
dxui.TextButton(dxui.ButtonProps{Size: dxui.ButtonNormal}, "Normal")
dxui.TextButton(dxui.ButtonProps{Size: dxui.ButtonLarge}, "Large")`,
			Preview: previewHorizontal(dxui.TextButton(dxui.ButtonProps{Size: dxui.ButtonSmall, OnPress: action("Small")}, "Small"), dxui.TextButton(dxui.ButtonProps{Size: dxui.ButtonNormal, OnPress: action("Normal")}, "Normal"), dxui.TextButton(dxui.ButtonProps{Size: dxui.ButtonLarge, OnPress: action("Large")}, "Large")),
		},
		{Title: "Built-in button variants", Description: "Tone selects semantic colors. Every appearance follows the live light/dark theme.",
			Code: `dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonPrimary}, "Primary")
dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonSecondary}, "Secondary")
dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonSuccess}, "Success")
dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonInfo}, "Info")
dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonWarn}, "Warn")
dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonDanger}, "Danger")`,
			Preview: previewHorizontal(dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonPrimary, OnPress: action("Primary")}, "Primary"), dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonSecondary, OnPress: action("Secondary")}, "Secondary"), dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonSuccess, OnPress: action("Success")}, "Success"), dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonInfo, OnPress: action("Info")}, "Info"), dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonWarn, OnPress: action("Warn")}, "Warn"), dxui.TextButton(dxui.ButtonProps{Tone: dxui.ButtonDanger, OnPress: action("Danger")}, "Danger")),
		},
		{Title: "Soft buttons", Description: "Soft semantic backgrounds and readable inherited foregrounds.",
			Code: `dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonPrimary}, "Primary")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonSuccess}, "Success")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonDanger}, "Danger")`,
			Preview: previewHorizontal(dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonPrimary, OnPress: action("Primary")}, "Primary"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonSuccess, OnPress: action("Success")}, "Success"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonDanger, OnPress: action("Danger")}, "Danger")),
		},
		{Title: "Outline buttons", Description: "Transparent background and solid semantic border; the keyboard focus ring remains independent.",
			Code: `dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonPrimary}, "Primary")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonSuccess}, "Success")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonDanger}, "Danger")`,
			Preview: previewHorizontal(dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonPrimary, OnPress: action("Primary")}, "Primary"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonSuccess, OnPress: action("Success")}, "Success"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonDanger, OnPress: action("Danger")}, "Danger")),
		},
		{Title: "Dash buttons", Description: "Transparent background and the existing bounded rounded dashed border.",
			Code: `dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonDashed, Tone: dxui.ButtonPrimary}, "Primary")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonDashed, Tone: dxui.ButtonSuccess}, "Success")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonDashed, Tone: dxui.ButtonDanger}, "Danger")`,
			Preview: previewHorizontal(dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonDashed, Tone: dxui.ButtonPrimary, OnPress: action("Primary")}, "Primary"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonDashed, Tone: dxui.ButtonSuccess, OnPress: action("Success")}, "Success"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonDashed, Tone: dxui.ButtonDanger, OnPress: action("Danger")}, "Danger")),
		},
		{Title: "Ghost buttons and Link button", Description: "Ghost reveals a hover surface. Link has compact spacing and remains an action, with Enter/Space activation.",
			Code: `dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonGhost}, "Ghost")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonLink}, "Link button")`,
			Preview: previewHorizontal(dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonGhost, OnPress: action("Ghost")}, "Ghost"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonLink, OnPress: action("Link button")}, "Link button")),
		},
		{Title: "Disabled buttons", Description: "Disabled prevents pointer and keyboard activation. PointerNone still allows Tab and Enter/Space.",
			Code: `dxui.TextButton(dxui.ButtonProps{Disabled: true}, "Filled")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonSuccess, Disabled: true}, "Soft")
dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonDanger, Disabled: true}, "Outline")
dxui.TextButton(dxui.ButtonProps{Pointer: dxui.PointerNone}, "Keyboard only")`,
			Preview: previewHorizontal(dxui.TextButton(dxui.ButtonProps{Disabled: true}, "Filled"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonSoft, Tone: dxui.ButtonSuccess, Disabled: true}, "Soft"), dxui.TextButton(dxui.ButtonProps{Variant: dxui.ButtonOutline, Tone: dxui.ButtonDanger, Disabled: true}, "Outline"), dxui.TextButton(dxui.ButtonProps{Pointer: dxui.PointerNone, OnPress: action("Keyboard only")}, "Keyboard only")),
		},
		{
			Title:       "Square button and circle button",
			Description: "Equal explicit dimensions create icon-button geometry; zero radius is square and a large radius is clamped to a circle.",
			Code: `dxui.Button(dxui.ButtonProps{Style: dxui.Style{
    Width: dxui.Px(40), Height: dxui.Px(40), MinWidth: dxui.Px(40), MinHeight: dxui.Px(40),
    Padding: dxui.Padding(10), Radius: dxui.Round(0),
}}, dxui.Icon(dxui.IconProps{Data: star, Size: 18}))
dxui.Button(dxui.ButtonProps{Style: dxui.Style{
    Width: dxui.Px(40), Height: dxui.Px(40), MinWidth: dxui.Px(40), MinHeight: dxui.Px(40),
    Padding: dxui.Padding(10), Radius: dxui.Round(999),
}}, dxui.Icon(dxui.IconProps{Data: star, Size: 18}))`,
			Preview: previewHorizontal(
				dxui.Button(dxui.ButtonProps{Style: iconButtonStyle(false), OnPress: action("Square")}, dxui.Icon(dxui.IconProps{Data: assets.star, Size: 18})),
				dxui.Button(dxui.ButtonProps{Style: iconButtonStyle(true), OnPress: action("Circle")}, dxui.Icon(dxui.IconProps{Data: assets.star, Size: 18})),
			),
		},
		{
			Title:       "Button with Icon",
			Description: "Button accepts one arbitrary child, so a horizontal Box can combine a vector Icon and Text while both inherit interaction tint.",
			Code: `dxui.Button(dxui.ButtonProps{OnPress: save},
    dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 7, Align: dxui.AlignCenter},
        dxui.Icon(dxui.IconProps{Data: star, Size: 16}),
        dxui.Text(dxui.TextProps{Value: "Favorite"}),
    ),
)`,
			Preview: dxui.Button(dxui.ButtonProps{OnPress: action("Favorite")}, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 7, Align: dxui.AlignCenter},
				dxui.Icon(dxui.IconProps{Data: assets.star, Size: 16}), dxui.Label("Favorite"),
			)),
		},
		{
			Title:       "Button with loading spinner",
			Description: "A static vector spinner communicates loading without inventing progress or scheduling work. Disabled prevents a second submission while loading.",
			Code: `dxui.Button(dxui.ButtonProps{Disabled: loading},
    dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 7, Align: dxui.AlignCenter},
        dxui.Icon(dxui.IconProps{Data: spinner, Size: 16}),
        dxui.Text(dxui.TextProps{Value: "Loading..."}),
    ),
)`,
			Preview: dxui.Button(dxui.ButtonProps{Disabled: true}, dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 7, Align: dxui.AlignCenter},
				dxui.Icon(dxui.IconProps{Data: assets.spinner, Size: 16}), dxui.Label("Loading..."),
			)),
		},
		{Title: "Advanced customization", Description: "Style customizes the base; local States follow each recipe state; Style.Force overrides all states.",
			Code: `dxui.TextButton(dxui.ButtonProps{
    Variant: dxui.ButtonOutline,
    Style: dxui.Style{Radius: dxui.Round(2)},
    States: dxui.StateStyles{Hover: dxui.StylePatch{TextColor: dxui.Some(dxui.TokenColor(dxui.Color.Semantic.Text))}},
}, "Custom")`,
			Preview: dxui.TextButton(dxui.ButtonProps{
				Variant: dxui.ButtonOutline,
				Style:   dxui.Style{Radius: dxui.Round(2)},
				States:  dxui.StateStyles{Hover: dxui.StylePatch{TextColor: dxui.Some(dxui.TokenColor(dxui.Color.Semantic.Text))}},
			}, "Custom"),
		},
	}
}

func buttonText(value string, size float32) dxui.View {
	return dxui.Text(dxui.TextProps{Style: dxui.Style{Text: dxui.TextStyle{
		Size: size, LineHeight: size * 1.35, Weight: dxui.WeightMedium,
	}}, Value: value})
}

func iconButtonStyle(circle bool) dxui.Style {
	style := dxui.Style{Width: dxui.Px(40), Height: dxui.Px(40), MinWidth: dxui.Px(40), MinHeight: dxui.Px(40), Padding: dxui.Padding(10)}
	if circle {
		style.Radius = dxui.Round(999)
	} else {
		style.Radius = dxui.Round(0)
	}
	return style
}

func inputBlocks(state *galleryState) []exampleBlock {
	fieldStyle := func() dxui.Style {
		return dxui.Style{Width: dxui.Px(280)}
	}
	return []exampleBlock{
		{
			Title:       "Controlled value and OnChange",
			Description: "Value is authoritative. OnChange receives the complete proposed value after an edit; this demo accepts it. Empty Value displays Placeholder. Leaving Value unchanged rejects a proposal.",
			Code: `dxui.Input(dxui.InputProps{
    Value: value, Placeholder: "Type here",
    OnChange: dxui.Assign(&value),
})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Input(dxui.InputProps{Key: "editable-input", Style: fieldStyle(), Value: state.input, Placeholder: "Type here", OnChange: func(next string) {
					state.input = next
					state.feedback = fmt.Sprintf("Input.OnChange -> %q", next)
				}}),
				bodyText("Current Value: "+state.input),
				dxui.Input(dxui.InputProps{Key: "empty-input", Style: fieldStyle(), Value: "", Placeholder: "Empty value shows this placeholder", OnChange: func(string) {}}),
			),
		},
		{
			Title:       "Rune-indexed controlled selection",
			Description: "Selection uses Unicode rune indices and may be left unset for retained selection. OnSelectionChange reports caret/selection moves. The demo accepts both text and selection proposals.",
			Code: `Selection: dxui.Some(dxui.TextRange{Start: 0, End: 4}),
OnSelectionChange: dxui.Assign(&selection),`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Input(dxui.InputProps{
					Key: "selection-input", Style: fieldStyle(), Value: state.input,
					Selection: dxui.Some(state.inputSelection),
					OnChange:  dxui.Assign(&state.input),
					OnSelectionChange: func(next dxui.TextRange) {
						state.inputSelection = next
						state.feedback = fmt.Sprintf("OnSelectionChange -> [%d,%d] runes", next.Start, next.End)
					},
				}),
				bodyText(fmt.Sprintf("Current selection: Start=%d, End=%d", state.inputSelection.Start, state.inputSelection.End)),
			),
		},
		{
			Title:       "Keyed identity across reorder",
			Description: "Key is sibling-local retained identity. Focus an Input, move its caret, then reorder the pair: focus, selection, and editing state follow the stable key instead of the child's position.",
			Code: `dxui.Input(dxui.InputProps{
    Key: "stable-a", Value: "key A", ReadOnly: true,
})
dxui.Input(dxui.InputProps{
    Key: "stable-b", Value: "key B", ReadOnly: true,
})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Button(dxui.ButtonProps{OnPress: func() {
					state.keyedInputsReversed = !state.keyedInputsReversed
					state.feedback = fmt.Sprintf("Keyed Inputs reordered: reversed=%t", state.keyedInputsReversed)
				}}, bodyText("Reorder keyed Inputs")),
				keyedInputs(state.keyedInputsReversed),
			),
		},
		{
			Title:       "Password, read-only, disabled, and nil OnChange",
			Description: "Password masks one bullet per rune and disables copy/cut. ShowPasswordToggle adds a trailing eye that reveals text without changing Value or focus; the disabled example shows its inert disabled state. ReadOnly still permits focus, movement, selection, and copy. Nil OnChange is read-only by behavior without forcing a visual state.",
			Code: `InputProps{Value: password, Password: true, ShowPasswordToggle: true, OnChange: setPassword}
InputProps{Value: "copy me", ReadOnly: true}
InputProps{Value: "disabled", Password: true, ShowPasswordToggle: true, Disabled: true}
InputProps{Value: "nil callback", OnChange: nil}`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Input(dxui.InputProps{Key: "password-input", Style: fieldStyle(), Value: state.password, Password: true, ShowPasswordToggle: true, Placeholder: "Password", OnChange: dxui.Assign(&state.password)}),
				dxui.Input(dxui.InputProps{Key: "readonly-input", Style: fieldStyle(), Value: "ReadOnly: selection and copy remain", ReadOnly: true}),
				dxui.Input(dxui.InputProps{Key: "disabled-input", Style: fieldStyle(), Value: "Disabled password", Password: true, ShowPasswordToggle: true, Disabled: true, OnChange: func(string) {}}),
				dxui.Input(dxui.InputProps{Key: "nil-change-input", Style: fieldStyle(), Value: "Nil OnChange: behavior-only read-only"}),
			),
		},
		{
			Title:       "Semantic submit and error styling",
			Description: "Enter invokes OnSubmit without deriving text from keycodes. Validation remains application-owned; express an error using ordinary typed Style/States and show the result in feedback.",
			Code: `InputProps{Value: value, OnChange: setValue,
    OnSubmit: func() { result = validate(value) },
    Style: dxui.Style{
        Border: dxui.Stroke(2, errorColor),
    },
}`,
			Preview: dxui.Input(dxui.InputProps{
				Key: "submit-input", Style: dxui.Style{Width: dxui.Px(360), Height: dxui.Px(42), Border: dxui.Stroke(2, dxui.TokenColor(dxui.Color.Semantic.Danger))},
				Value: state.input, OnChange: dxui.Assign(&state.input),
				OnSubmit: func() { state.feedback = fmt.Sprintf("Input.OnSubmit -> current value %q", state.input) },
			}),
		},
		{
			Title:       "Practical controlled form",
			Description: "A small form keeps every field in application state, composes Input, Select, Checkbox, and ButtonGroup directly, reports the submitted values, and offers a local reset. This is the same controlled pattern as the focused examples above.",
			Code: `dxui.Box(dxui.BoxProps{Gap: 10},
    dxui.Input(dxui.InputProps{
        Value: name, Placeholder: "Display name",
        OnChange: dxui.Assign(&name),
    }),
    dxui.Select(dxui.SelectProps{
        Value: channel,
        Options: []dxui.SelectOption{
            {Value: "fast", Label: "Fast"},
            {Value: "stable", Label: "Stable"},
        },
        OnChange: dxui.Assign(&channel),
    }),
    dxui.Checkbox(dxui.CheckboxProps{
        Checked: updates,
        OnChange: dxui.Assign(&updates),
    }, dxui.Text(dxui.TextProps{Value: "Receive updates"})),
    dxui.ButtonGroup(dxui.ButtonGroupProps{},
        dxui.Button(dxui.ButtonProps{OnPress: submit},
            dxui.Text(dxui.TextProps{Value: "Save"})),
        dxui.Button(dxui.ButtonProps{OnPress: reset},
            dxui.Text(dxui.TextProps{Value: "Reset"})),
    ),
)`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 10},
				dxui.Input(dxui.InputProps{
					Key: "form-name", Style: dxui.Style{Width: dxui.Px(320)},
					Value: state.input, Placeholder: "Display name",
					OnChange: dxui.Assign(&state.input),
				}),
				dxui.Select(dxui.SelectProps{
					Key: "form-channel", Style: dxui.Style{Width: dxui.Px(320)},
					Value: state.selectValue,
					Options: []dxui.SelectOption{
						{Value: "fast", Label: "Fast"},
						{Value: "stable", Label: "Stable"},
					},
					OnChange: dxui.Assign(&state.selectValue),
				}),
				dxui.Checkbox(dxui.CheckboxProps{
					Key: "form-updates", Checked: state.checkbox,
					OnChange: dxui.Assign(&state.checkbox),
				}, dxui.Text(dxui.TextProps{Value: "Receive updates"})),
				dxui.ButtonGroup(dxui.ButtonGroupProps{},
					dxui.Button(dxui.ButtonProps{OnPress: func() {
						state.feedback = fmt.Sprintf("Form saved -> name=%q channel=%q updates=%t", state.input, state.selectValue, state.checkbox)
					}}, dxui.Text(dxui.TextProps{Value: "Save"})),
					dxui.Button(dxui.ButtonProps{OnPress: func() {
						state.input, state.selectValue, state.checkbox = "Editable value", "stable", false
						state.inputSelection = dxui.TextRange{Start: 0, End: 8}
						state.feedback = "Form fields reset."
					}}, dxui.Text(dxui.TextProps{Value: "Reset"})),
				),
				bodyText(fmt.Sprintf("Current form: name=%q, channel=%q, updates=%t", state.input, state.selectValue, state.checkbox)),
			),
		},
	}
}

func keyedInputs(reversed bool) dxui.View {
	first := dxui.Input(dxui.InputProps{Key: "stable-a", Style: dxui.Style{Width: dxui.Px(150)}, Value: "key A", ReadOnly: true})
	second := dxui.Input(dxui.InputProps{Key: "stable-b", Style: dxui.Style{Width: dxui.Px(150)}, Value: "key B", ReadOnly: true})
	if reversed {
		first, second = second, first
	}
	return previewHorizontal(first, second)
}

func textareaBlocks(state *galleryState) []exampleBlock {
	areaStyle := func(height float32) dxui.Style {
		return dxui.Style{Width: dxui.Px(420), Height: dxui.Px(height), Shrink: dxui.NoShrink()}
	}
	return []exampleBlock{
		{
			Title:       "Controlled multiline editing",
			Description: "Textarea preserves explicit newlines. OnChange proposes the complete value and the next build accepts or rejects it. Enter inserts a newline rather than submitting.",
			Code: `dxui.Textarea(dxui.TextareaProps{
    Value: value,
    OnChange: dxui.Assign(&value),
    Wrap: dxui.TextWrapWords,
})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Textarea(dxui.TextareaProps{Key: "editable-textarea", Style: areaStyle(130), Value: state.textarea, Wrap: dxui.TextWrapWords, OnChange: func(next string) {
					state.textarea = next
					state.feedback = fmt.Sprintf("Textarea.OnChange -> %d runes", len([]rune(next)))
				}}),
				bodyText(fmt.Sprintf("Current value has %d runes", len([]rune(state.textarea)))),
			),
		},
		{
			Title:       "Wrap modes and empty placeholder",
			Description: "TextNoWrap is the zero/default and enables horizontal caret scrolling. TextWrapWords soft-wraps to the viewport. Empty Value displays Placeholder.",
			Code: `TextareaProps{Value: longLine, Wrap: dxui.TextNoWrap}
TextareaProps{Value: longLine, Wrap: dxui.TextWrapWords}
TextareaProps{Value: "", Placeholder: "Empty notes"}`,
			Preview: previewHorizontal(
				dxui.Textarea(dxui.TextareaProps{Key: "nowrap-textarea", Style: areaStyle(105), Value: "No wrap: a deliberately long editable line continues horizontally beyond this narrow viewport.", Wrap: dxui.TextNoWrap, OnChange: func(string) {}}),
				dxui.Textarea(dxui.TextareaProps{Key: "placeholder-textarea", Style: areaStyle(105), Value: "", Placeholder: "Empty notes placeholder", Wrap: dxui.TextWrapWords, OnChange: func(string) {}}),
			),
		},
		{
			Title:       "Selection, ReadOnly, and Disabled",
			Description: "Selection and OnSelectionChange use rune indices. ReadOnly permits selection/copy but no mutation; nil OnChange is behavior-only read-only. Disabled removes focus and editing.",
			Code: `Selection: dxui.Some(dxui.TextRange{Start: 0, End: 5}),
OnSelectionChange: dxui.Assign(&selection),
ReadOnly: true // or Disabled: true`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Textarea(dxui.TextareaProps{
					Key: "selection-textarea", Style: areaStyle(90), Value: state.textarea, Wrap: dxui.TextWrapWords,
					Selection: dxui.Some(state.textareaSelection), OnChange: dxui.Assign(&state.textarea),
					OnSelectionChange: func(next dxui.TextRange) {
						state.textareaSelection = next
						state.feedback = fmt.Sprintf("Textarea selection -> [%d,%d]", next.Start, next.End)
					},
				}),
				previewHorizontal(
					dxui.Textarea(dxui.TextareaProps{Key: "readonly-textarea", Style: areaStyle(80), Value: "ReadOnly\nCopy remains available.", ReadOnly: true}),
					dxui.Textarea(dxui.TextareaProps{Key: "disabled-textarea", Style: areaStyle(80), Value: "Disabled", Disabled: true, OnChange: func(string) {}}),
				),
			),
		},
	}
}

func selectBlocks(state *galleryState) []exampleBlock {
	options := []dxui.SelectOption{
		{Value: "fast", Label: "Fast"},
		{Value: "stable", Label: "Stable"},
		{Value: "legacy", Label: "Legacy (disabled)", Disabled: true},
		{Value: "experimental", Label: "Experimental"},
	}
	selectStyle := func() dxui.Style {
		return dxui.Style{Width: dxui.Px(250)}
	}
	return []exampleBlock{
		{
			Title:       "Controlled value, options, and OnChange",
			Description: "Value is authoritative. Each non-empty unique option Value is also retained identity. Open with pointer/Enter/Space; move with arrows/Home/End; Enter chooses; Escape closes; Tab transfers focus. Disabled options are skipped.",
			Code: `dxui.Select(dxui.SelectProps{
    Value: value,
    Options: []dxui.SelectOption{
        {Value: "stable", Label: "Stable"},
        {Value: "legacy", Label: "Legacy", Disabled: true},
    },
    OnChange: dxui.Assign(&value),
})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Select(dxui.SelectProps{Key: "interactive-select", Style: selectStyle(), Value: state.selectValue, Options: options, Placeholder: "Choose a channel", OnChange: func(next string) {
					state.selectValue = next
					state.feedback = fmt.Sprintf("Select.OnChange -> %q", next)
				}}),
				bodyText("Current Value: "+state.selectValue),
			),
		},
		{
			Title:       "Placeholder, unmatched value, and empty options",
			Description: "An empty or unmatched Value displays Placeholder. An empty Options slice is valid but cannot open. A nil OnChange is valid and makes selection proposals unaccepted.",
			Code: `SelectProps{Value: "", Options: options, Placeholder: "Choose..."}
SelectProps{Value: "missing", Options: options, Placeholder: "Unmatched"}
SelectProps{Options: nil, OnChange: nil}`,
			Preview: previewHorizontal(
				dxui.Select(dxui.SelectProps{Key: "placeholder-select", Style: selectStyle(), Value: "", Options: options, Placeholder: "Empty value placeholder"}),
				dxui.Select(dxui.SelectProps{Key: "unmatched-select", Style: selectStyle(), Value: "missing", Options: options, Placeholder: "Unmatched value"}),
				dxui.Select(dxui.SelectProps{Key: "empty-select", Style: selectStyle(), Options: nil, Placeholder: "No options"}),
			),
		},
		{
			Title:       "Disabled control and overlay behavior",
			Description: "Disabled Select cannot focus or open. The enabled Select sits inside a short Scroll; its window-level popup is still clamped to the window and escapes the ancestor viewport clip.",
			Code: `dxui.Select(dxui.SelectProps{Disabled: true, Options: options})
dxui.Scroll(scrollProps, dxui.Select(selectProps)) // popup escapes clip`,
			Preview: previewHorizontal(
				dxui.Select(dxui.SelectProps{Key: "disabled-select", Style: selectStyle(), Value: "stable", Options: options, Disabled: true, OnChange: func(string) {}}),
				dxui.Scroll(dxui.ScrollProps{Key: "select-scroll", Style: dxui.Style{Width: dxui.Px(290), Height: dxui.Px(70)}, Axis: dxui.ScrollVertical},
					dxui.Box(dxui.BoxProps{Gap: 8}, bodyText("Inside Scroll"), dxui.Select(dxui.SelectProps{Key: "scroll-select", Style: selectStyle(), Value: state.selectValue, Options: options, OnChange: dxui.Assign(&state.selectValue)}), bodyText("Clipped tail"))),
			),
		},
	}
}

func tabsBlocks(state *galleryState) []exampleBlock {
	items := []dxui.TabItem{
		{Value: "overview", Label: "Overview"},
		{Value: "activity", Label: "Activity"},
		{Value: "legacy", Label: "Legacy", Disabled: true},
		{Value: "settings", Label: "Settings"},
	}
	return []exampleBlock{
		{
			Title:       "Controlled selection and application-owned content",
			Description: "Click a label or use Left/Right/Home/End followed by Enter/Space. Value is authoritative, disabled labels are skipped, and Tab enters or leaves the component normally. The content below is ordinary application rendering, not an embedded TabPanel.",
			Code: `dxui.Box(dxui.BoxProps{},
    dxui.Tabs(dxui.TabsProps{
        Value: value,
        Items: []dxui.TabItem{
            {Value: "overview", Label: "Overview"},
            {Value: "legacy", Label: "Legacy", Disabled: true},
        },
        OnChange: dxui.Assign(&value),
    }),
    contentFor(value),
)`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 10},
				dxui.Tabs(dxui.TabsProps{Key: "interactive-tabs", Value: state.tabsValue, Items: items, OnChange: func(next string) {
					state.tabsValue = next
					state.feedback = fmt.Sprintf("Tabs.OnChange -> %q", next)
				}}),
				bodyText("Application content for: "+state.tabsValue),
			),
		},
		{
			Title:       "Empty, unmatched, disabled, and nil callback",
			Description: "Empty and unmatched Value are valid and draw no indicator. Empty Items are valid. Disabled removes the component from focus and pointer input; a nil OnChange keeps navigation visuals but submits nothing.",
			Code: `dxui.Tabs(dxui.TabsProps{Value: "", Items: items})
dxui.Tabs(dxui.TabsProps{Value: "missing", Items: items, OnChange: nil})
dxui.Tabs(dxui.TabsProps{Items: nil})
dxui.Tabs(dxui.TabsProps{Value: "overview", Items: items, Disabled: true})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Tabs(dxui.TabsProps{Key: "empty-value-tabs", Value: "", Items: items}),
				dxui.Tabs(dxui.TabsProps{Key: "unmatched-tabs", Value: "missing", Items: items, OnChange: nil}),
				dxui.Tabs(dxui.TabsProps{Key: "empty-tabs", Items: nil}),
				dxui.Tabs(dxui.TabsProps{Key: "disabled-tabs", Value: "overview", Items: items, Disabled: true, OnChange: func(string) {}}),
			),
		},
	}
}

func menuBlocks(state *galleryState) []exampleBlock {
	items := []dxui.MenuItem{
		{Value: "new", Label: "New document"},
		{Value: "open", Label: "Open document"},
		{Value: "archive", Label: "Archive", Disabled: true},
		{Value: "quit", Label: "Quit"},
	}
	action := func(value string) {
		state.menuValue = value
		state.feedback = fmt.Sprintf("Menu.OnAction -> %q", value)
	}
	return []exampleBlock{
		{
			Title:       "Vertical action menu",
			Description: "The zero-value orientation stacks items vertically and each row fills the Menu width. Value controls the persistent selected row independently of keyboard current. Use Up/Down or Home/End to move current, then Enter/Space to invoke OnAction once.",
			Code: `dxui.Menu(dxui.MenuProps{
    Style: dxui.Style{Width: dxui.Px(240)},
    Value: selected,
    Items: []dxui.MenuItem{
        {Value: "new", Label: "New document"},
        {Value: "archive", Label: "Archive", Disabled: true},
    },
    OnAction: dxui.Assign(&selected),
})`,
			Preview: dxui.Menu(dxui.MenuProps{Key: "vertical-menu", Style: dxui.Style{Width: dxui.Px(240)}, Value: state.menuValue, Items: items, OnAction: action}),
		},
		{
			Title:       "Horizontal action menu",
			Description: "Horizontal items keep content-sized widths and use Left/Right navigation. The component remains in ordinary layout and does not open an overlay.",
			Code: `dxui.Menu(dxui.MenuProps{
    Orientation: dxui.MenuHorizontal,
    Value: selected,
    Items: items,
    OnAction: dxui.Assign(&selected),
})`,
			Preview: dxui.Menu(dxui.MenuProps{Key: "horizontal-menu", Orientation: dxui.MenuHorizontal, Value: state.menuValue, Items: items, OnAction: action}),
		},
		{
			Title:       "Disabled control and item",
			Description: "A disabled Menu is omitted from focus and pointer targeting. A disabled item stays visible but is skipped by pointer and keyboard navigation. Empty Items and a nil callback are also valid.",
			Code: `dxui.Menu(dxui.MenuProps{Items: items, Disabled: true})
dxui.Menu(dxui.MenuProps{Items: nil, OnAction: nil})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 8},
				dxui.Menu(dxui.MenuProps{Key: "disabled-menu", Style: dxui.Style{Width: dxui.Px(240)}, Items: items, Disabled: true, OnAction: action}),
				dxui.Menu(dxui.MenuProps{Key: "empty-menu", Items: nil}),
			),
		},
	}
}

func checkboxBlocks(state *galleryState) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Controlled checked value and label child",
			Description: "OnChange proposes the opposite Checked value. The next build remains authoritative. The label is any one View and inherits state tint.",
			Code: `dxui.Checkbox(dxui.CheckboxProps{
    Checked: checked,
    OnChange: dxui.Assign(&checked),
}, dxui.Text(dxui.TextProps{Value: "Remember me"}))`,
			Preview: dxui.Checkbox(dxui.CheckboxProps{Key: "interactive-checkbox", Checked: state.checkbox, OnChange: func(next bool) {
				state.checkbox = next
				state.feedback = fmt.Sprintf("Checkbox.OnChange -> %t", next)
			}}, bodyText(fmt.Sprintf("Controlled value: %t", state.checkbox))),
		},
		{
			Title:       "Unchecked, checked, disabled, and nil callback",
			Description: "Checked and Disabled each show both boolean values. Disabled suppresses hover/focus/pressed. Nil OnChange is valid and leaves the controlled value unchanged. There is no indeterminate state.",
			Code: `CheckboxProps{Checked: false}
CheckboxProps{Checked: true}
CheckboxProps{Checked: true, Disabled: true}
CheckboxProps{Checked: false, OnChange: nil}`,
			Preview: previewHorizontal(
				dxui.Checkbox(dxui.CheckboxProps{}, bodyText("Unchecked")),
				dxui.Checkbox(dxui.CheckboxProps{Checked: true}, bodyText("Checked")),
				dxui.Checkbox(dxui.CheckboxProps{Checked: true, Disabled: true}, bodyText("Checked + disabled")),
				dxui.Checkbox(dxui.CheckboxProps{OnChange: nil}, bodyText("Nil callback")),
			),
		},
	}
}

func radioBlocks(state *galleryState) []exampleBlock {
	choice := func(value, label string, disabled bool) dxui.View {
		return dxui.Radio(dxui.RadioProps{
			Key: "radio-" + value, Selected: state.radioValue == value, Disabled: disabled,
			OnSelect: func() {
				state.radioValue = value
				state.feedback = fmt.Sprintf("Radio.OnSelect -> %q", value)
			},
		}, bodyText(label))
	}
	return []exampleBlock{
		{
			Title:       "Shared controlled selection",
			Description: "All three Radio values derive Selected from the same application string. Activating an unselected Radio assigns that business value; the selected Radio cannot clear or reselect itself.",
			Code: `value := "stable"
dxui.Box(dxui.BoxProps{},
    dxui.Radio(dxui.RadioProps{
        Selected: value == "fast",
        OnSelect: func() { value = "fast" },
    }, dxui.Text(dxui.TextProps{Value: "Fast"})),
    dxui.Radio(dxui.RadioProps{
        Selected: value == "stable",
        OnSelect: func() { value = "stable" },
    }, dxui.Text(dxui.TextProps{Value: "Stable"})),
)`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 4},
				choice("fast", "Fast", false),
				choice("stable", "Stable", false),
				choice("experimental", "Experimental", false),
				bodyText("Current business value: "+state.radioValue),
			),
		},
		{
			Title:       "Selected, disabled, and nil callback",
			Description: "Selected is authoritative. Disabled and nil OnSelect controls never emit a callback. Pointer or Space activates only an enabled, unselected Radio.",
			Code: `RadioProps{Selected: false, OnSelect: selectValue}
RadioProps{Selected: true, OnSelect: selectValue}
RadioProps{Disabled: true, OnSelect: selectValue}
RadioProps{OnSelect: nil}`,
			Preview: previewHorizontal(
				dxui.Radio(dxui.RadioProps{OnSelect: func() { state.feedback = "Standalone unselected Radio selected." }}, bodyText("Unselected")),
				dxui.Radio(dxui.RadioProps{Selected: true, OnSelect: func() { state.feedback = "Selected Radio should not callback." }}, bodyText("Selected")),
				dxui.Radio(dxui.RadioProps{Disabled: true, OnSelect: func() { state.feedback = "Disabled Radio should not callback." }}, bodyText("Disabled")),
				dxui.Radio(dxui.RadioProps{}, bodyText("Nil OnSelect")),
			),
		},
	}
}

func toggleBlocks(app *dxui.App, state *galleryState) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Controlled checked event",
			Description: "Click or focus and press Space. OnChange proposes !Checked; this demo accepts it and reports the current value.",
			Code: `dxui.ToggleSwitch(dxui.ToggleSwitchProps{
    Checked: checked,
    OnChange: dxui.Assign(&checked),
})`,
			Preview: previewHorizontal(
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{Key: "interactive-toggle", Checked: state.toggle, OnChange: func(next bool) {
					state.toggle = next
					state.feedback = fmt.Sprintf("ToggleSwitch.OnChange -> %t", next)
				}}),
				bodyText(fmt.Sprintf("Current Checked: %t", state.toggle)),
			),
		},
		{
			Title:       "Boolean state matrix",
			Description: "Unchecked/checked and enabled/disabled are shown together. Hover, Focus, and Pressed are available only on enabled controls. The final enabled switch has nil OnChange, so it animates state feedback but remains controlled false.",
			Code: `ToggleSwitchProps{Checked: false, Disabled: false}
ToggleSwitchProps{Checked: true, Disabled: false}
ToggleSwitchProps{Checked: false, Disabled: true}
ToggleSwitchProps{Checked: true, Disabled: true}`,
			Preview: previewHorizontal(
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{}), bodyText("off"),
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{Checked: true}), bodyText("on"),
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{Disabled: true}), bodyText("off disabled"),
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{Checked: true, Disabled: true}), bodyText("on disabled"),
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{OnChange: nil}), bodyText("nil callback"),
			),
		},
		{
			Title:       "Real runtime theme change",
			Description: "ToggleSwitch consumes component theme tokens. This control changes the App's complete Light/Dark theme through SetTheme; the sidebar switch mirrors the same state.",
			Code: `OnChange: func(dark bool) {
    state.dark = dark
    if err := app.SetTheme(themeFor(dark)); err != nil { /* report */ }
}`,
			Preview: previewHorizontal(
				dxui.ToggleSwitch(dxui.ToggleSwitchProps{Checked: state.dark, OnChange: func(next bool) {
					state.dark = next
					if err := app.SetTheme(showcaseTheme(next, state.theme)); err != nil {
						state.feedback = "SetTheme error: " + err.Error()
					} else {
						state.feedback = fmt.Sprintf("SetTheme -> dark=%t", next)
					}
				}}),
				bodyText("Switch Light / Dark theme"),
			),
		},
	}
}

func sliderBlocks(state *galleryState) []exampleBlock {
	return []exampleBlock{
		{
			Title:       "Controlled value and stepped interaction",
			Description: "Click the track, drag the circular thumb beyond the component, or use arrows/Home/End after focusing it. Value remains authoritative and this demo accepts each step-aligned proposal.",
			Code: `dxui.Slider(dxui.SliderProps{
    Value: value, Min: 0, Max: 100, Step: 5,
    OnChange: dxui.Assign(&value),
})`,
			Preview: previewHorizontal(
				dxui.Slider(dxui.SliderProps{Key: "interactive-slider", Value: state.slider, Min: 0, Max: 100, Step: 5, OnChange: func(next float32) {
					state.slider = next
					state.feedback = fmt.Sprintf("Slider.OnChange -> %.0f", next)
				}}),
				bodyText(fmt.Sprintf("Current Value: %.0f", state.slider)),
			),
		},
		{
			Title:       "Defaults, boundaries, and inert states",
			Description: "The zero value uses 0..100 with Step 1. Invalid or reversed ranges are inert. A non-positive/non-finite Step falls back to 1; nil OnChange is valid, and Disabled removes focus and pointer interaction.",
			Code: `dxui.Slider(dxui.SliderProps{}) // 0..100, Step 1
dxui.Slider(dxui.SliderProps{Value: 75, Disabled: true})
dxui.Slider(dxui.SliderProps{Min: 10, Max: 10}) // inert
dxui.Slider(dxui.SliderProps{Value: 40, OnChange: nil})`,
			Preview: dxui.Box(dxui.BoxProps{Gap: 10},
				dxui.Slider(dxui.SliderProps{}),
				dxui.Slider(dxui.SliderProps{Value: 75, Disabled: true}),
				dxui.Slider(dxui.SliderProps{Min: 10, Max: 10}),
				dxui.Slider(dxui.SliderProps{Value: 40, OnChange: nil}),
			),
		},
	}
}
