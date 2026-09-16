package dxui

import (
	"fmt"

	"github.com/dxui-org/dxui/internal/tree"
)

// ButtonVariant selects a Button's surface and border recipe.
type ButtonVariant uint8

const (
	ButtonFilled ButtonVariant = iota // Filled is the compatible default.
	ButtonSoft
	ButtonOutline
	ButtonDashed
	ButtonGhost
	ButtonLink // Link is an action, with compact spacing; it never navigates.
)

// ButtonTone selects semantic colors independently of the surface recipe.
type ButtonTone uint8

const (
	ButtonPrimary ButtonTone = iota
	ButtonSecondary
	ButtonSuccess
	ButtonInfo
	ButtonWarn
	ButtonDanger
)

// ButtonDefault is an alias for the zero-value Primary tone.
const ButtonDefault = ButtonPrimary

// ButtonSize selects overridable padding, minimum height, and inherited text metrics.
type ButtonSize uint8

const (
	ButtonNormal ButtonSize = iota
	ButtonSmall
	ButtonLarge
)

func applyButtonSize(style *Style, variant ButtonVariant, size ButtonSize) {
	x, y := TokenMetric(MetricComponentButtonPaddingX), TokenMetric(MetricComponentButtonPaddingY)
	var minimum float32
	switch size {
	case ButtonSmall:
		x, y, minimum = Metric(10), Metric(5), 30
	case ButtonLarge:
		x, y, minimum = Metric(20), Metric(12), 48
	}
	if variant == ButtonLink {
		x, y = Metric(2), Metric(2)
		minimum = 24
		if size == ButtonSmall {
			minimum = 20
		}
		if size == ButtonLarge {
			minimum = 30
		}
	}
	applyDefaultPadding(style, y, x)
	if minimum != 0 && style.Height.kind == lengthAuto && style.MinHeight.kind == lengthAuto {
		style.MinHeight = Px(minimum)
	}
}

// A semantic-text ring remains visible on both built-in surfaces. This private
// immutable patch is shared; resolving an event never allocates its slice.
var buttonRecipeFocus = StylePatch{Shadow: Some([]Shadow{{
	Blur: Metric(0), Spread: Metric(2), Color: TokenColor(ColorSemanticText),
}})}

// Resolve only a small recipe, never a theme table. Legacy combinations share
// the actual current component entries, including application customizations.
func buttonRecipe(theme resolvedTheme, variant ButtonVariant, tone ButtonTone) (ComponentTheme, error) {
	token := ComponentToken("")
	if variant == ButtonFilled {
		switch tone {
		case ButtonPrimary:
			token = ComponentButton
		case ButtonSecondary:
			token = ComponentButtonSecondary
		case ButtonDanger:
			token = ComponentButtonDanger
		}
	} else if variant == ButtonGhost && tone == ButtonPrimary {
		token = ComponentButtonGhost
	}
	if token != "" {
		component, ok := theme.source.Components[token]
		if !ok {
			return ComponentTheme{}, fmt.Errorf("theme: missing component token %q", token)
		}
		return component, nil
	}
	primary, ok := theme.source.Components[ComponentButton]
	if !ok {
		return ComponentTheme{}, fmt.Errorf("theme: missing component token %q", ComponentButton)
	}
	colorToken := ColorSemanticAccent
	switch tone {
	case ButtonSecondary:
		colorToken = ColorSemanticText
	case ButtonSuccess:
		colorToken = ColorSemanticSuccess
	case ButtonInfo:
		colorToken = ColorSemanticInfo
	case ButtonWarn:
		colorToken = ColorSemanticWarn
	case ButtonDanger:
		colorToken = ColorSemanticDanger
	}
	toneColor, err := theme.color(TokenColor(colorToken))
	if err != nil {
		return ComponentTheme{}, err
	}
	surface, err := theme.color(TokenColor(ColorSemanticSurface))
	if err != nil {
		return ComponentTheme{}, err
	}
	text, err := theme.color(TokenColor(ColorSemanticText))
	if err != nil {
		return ComponentTheme{}, err
	}
	foreground := LiteralColor(buttonMix(toneColor, text, .38))
	soft := LiteralColor(buttonMix(surface, toneColor, .12))
	hover := LiteralColor(buttonMix(surface, toneColor, .20))
	pressed := LiteralColor(buttonMix(surface, toneColor, .28))
	radius := primary.Base.Radius
	focus := primary.States.Focus
	if isButtonFocusRing(focus) {
		focus = buttonRecipeFocus
	}
	recipe := ComponentTheme{
		Base: StylePatch{Background: Some(ColorRGBA(0, 0, 0, 0)), Border: Some(NoBorder()), Radius: radius, TextColor: Some(foreground)},
		States: StateStyles{
			Hover:    StylePatch{Background: Some(hover)},
			Focus:    focus,
			Pressed:  StylePatch{Background: Some(pressed)},
			Disabled: StylePatch{Opacity: Some(float32(.42))},
		},
	}
	switch variant {
	case ButtonFilled:
		recipe.Base.Background = Some(foreground)
		recipe.Base.TextColor = Some(TokenColor(ColorSemanticOnAccent))
		recipe.States.Hover = StylePatch{Opacity: Some(float32(.88))}
		recipe.States.Pressed = StylePatch{Opacity: Some(float32(.72))}
	case ButtonSoft:
		recipe.Base.Background = Some(soft)
	case ButtonOutline, ButtonDashed:
		border := Border{Width: TokenMetric(MetricSemanticBorder), Color: foreground}
		if variant == ButtonDashed {
			border.Pattern = BorderDashed
		}
		recipe.Base.Border = Some(border)
	case ButtonLink:
		recipe.States.Hover = StylePatch{TextColor: Some(LiteralColor(buttonMix(toneColor, text, .55)))}
		recipe.States.Pressed = StylePatch{TextColor: Some(TokenColor(ColorSemanticText))}
	}
	return recipe, nil
}

func buttonMix(from, to RGBAColor, amount float32) RGBAColor {
	channel := func(a, b uint8) uint8 { return uint8(float32(a) + (float32(b)-float32(a))*amount + .5) }
	return RGBA(channel(from.R, to.R), channel(from.G, to.G), channel(from.B, to.B), channel(from.A, to.A))
}

// Typography is carried by adapters, not installed into arbitrary child Views.
type buttonTypography struct {
	size, lineHeight float32
	active           bool
}

func inheritedButtonTypography(view View, style TextStyle, inherited []buttonTypography) buttonTypography {
	var result buttonTypography
	if len(inherited) != 0 {
		result = inherited[0]
	}
	if view.node.kind == viewButton {
		result = buttonTypography{active: true}
		switch view.node.button.Size {
		case ButtonSmall:
			result.size, result.lineHeight = 12, 18
		case ButtonLarge:
			result.size, result.lineHeight = 18, 26
		}
	}
	if result.active {
		if style.Size != 0 {
			result.size = style.Size
		}
		if style.LineHeight != 0 {
			result.lineHeight = style.LineHeight
		}
	}
	return result
}

func (value buttonTypography) apply(style TextStyle) TextStyle {
	if style.Size == 0 {
		style.Size = value.size
	}
	if style.LineHeight == 0 {
		style.LineHeight = value.lineHeight
	}
	return style
}

func retainedButtonText(props TextProps, instance *tree.Node) TextProps {
	if instance != nil {
		content := instance.Properties.Content
		if props.Style.Text.Size == 0 && !content.FontSize.IsToken {
			props.Style.Text.Size = content.FontSize.Literal
		}
		if props.Style.Text.LineHeight == 0 && !content.LineHeight.IsToken {
			props.Style.Text.LineHeight = content.LineHeight.Literal
		}
	}
	return props
}

// Only the standard semantic keyboard indicator is a hollow ring. Arbitrary
// user shadows retain the established filled-shadow renderer behavior.
func isButtonFocusRing(patch StylePatch) bool {
	shadows, set := patch.Shadow.get()
	if !set || len(shadows) != 1 {
		return false
	}
	shadow := shadows[0]
	return shadow.OffsetX == (MetricValue{}) && shadow.OffsetY == (MetricValue{}) && shadow.Blur == Metric(0) && shadow.Spread == Metric(2) && (shadow.Color == TokenColor(ColorSemanticFocusRing) || shadow.Color == TokenColor(ColorSemanticText))
}
