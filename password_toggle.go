package dxui

import (
	"fmt"

	"github.com/dxui-org/dxui/icon"
	internalinput "github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

const (
	passwordToggleGap       float32 = 4
	passwordToggleIconScale float32 = .8
)

var passwordVisibleIcon = icon.Eye()
var passwordHiddenIcon = icon.EyeOff()

func passwordToggleRects(props *InputProps, geometry *layout.Result, theme resolvedTheme) (hit, icon layout.Rect, err error) {
	if props == nil || geometry == nil || !props.Password || !props.ShowPasswordToggle {
		return layout.Rect{}, layout.Rect{}, nil
	}
	size, err := theme.metric(TokenMetric(MetricComponentIconSize))
	if err != nil {
		return layout.Rect{}, layout.Rect{}, fmt.Errorf("password toggle size: %w", err)
	}
	size *= passwordToggleIconScale
	rightPadding, err := theme.metric(props.common().Style.Padding.Right)
	if err != nil {
		return layout.Rect{}, layout.Rect{}, fmt.Errorf("password toggle padding: %w", err)
	}
	rect := geometry.Rect
	size = min(size, rect.Width, rect.Height)
	size = max(0, size)
	iconX := rect.X + rect.Width - rightPadding - size
	iconX = min(max(iconX, rect.X), rect.X+rect.Width-size)
	icon = layout.Rect{X: iconX, Y: rect.Y + (rect.Height-size)/2, Width: size, Height: size}
	hitX := max(rect.X, geometry.Content.X+geometry.Content.Width)
	hit = layout.Rect{X: hitX, Y: rect.Y, Width: max(0, rect.X+rect.Width-hitX), Height: rect.Height}
	return hit, icon, nil
}

func appendPasswordToggleDisplay(list *paint.DisplayList, props *InputProps, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse, visual computedVisual) error {
	if props == nil || !props.Password || !props.ShowPasswordToggle {
		return nil
	}
	_, iconBounds, err := passwordToggleRects(props, geometry, theme)
	if err != nil {
		return err
	}
	if iconBounds.Width <= 0 || iconBounds.Height <= 0 {
		return nil
	}
	outer := paintRect(geometry.Rect)
	*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: outer})
	defer func() { *list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: instance.ID}) }()
	if instance.State.PasswordToggleHover && !props.Disabled {
		background, colorErr := theme.color(TokenColor(ColorSemanticAccent))
		if colorErr != nil {
			return colorErr
		}
		background.A = 32
		if instance.State.PasswordTogglePressed {
			background.A = 64
		}
		radius := min(iconBounds.Width, iconBounds.Height) / 4
		*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID,
			Rect: paintRect(iconBounds), Radii: paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}, Color: paintColor(background)})
	}
	data := passwordVisibleIcon
	if instance.State.PasswordVisible {
		data = passwordHiddenIcon
	}
	mask, maskRect, err := rasterIconReuse(IconProps{Data: data, Size: iconBounds.Width}, iconBounds, theme, scaleX, scaleY, reuse)
	if err != nil {
		return fmt.Errorf("password toggle icon: %w", err)
	}
	if err := accountIconMask(mask, remaining, reuse); err != nil {
		return fmt.Errorf("password toggle icon source budget: %w", err)
	}
	color := visual.textColor
	if !visual.textColorSet {
		color = RGBA(0, 0, 0, 255)
	}
	color.A = uint8(uint16(color.A) * 160 / 255)
	if props.Disabled {
		color.A /= 2
	}
	*list = append(*list, paint.Command{Kind: paint.CommandDrawIcon, NodeID: instance.ID, Rect: maskRect, Text: mask, Color: paintColor(color)})
	return nil
}

func (a *App) handlePasswordToggleEvent(event platform.Event, result internalinput.Result) (dirty, handled bool, err error) {
	if event.Kind != platform.EventMouseMove && event.Kind != platform.EventMouseDown && event.Kind != platform.EventMouseUp && event.Kind != platform.EventWindowMouseLeave && event.Kind != platform.EventWindowFocusLost {
		return false, false, nil
	}
	root := a.retained.Root()
	setPointerState := func(node *tree.Node, hovered, pressed bool) {
		if node.State.PasswordToggleHover != hovered || node.State.PasswordTogglePressed != pressed {
			node.State.PasswordToggleHover, node.State.PasswordTogglePressed = hovered, pressed
			dirty = true
		}
	}
	clearPointerState := func() {
		var visit func(*tree.Node)
		visit = func(node *tree.Node) {
			if node == nil {
				return
			}
			if action, live := a.inputActions[node.ID]; live && action.passwordToggle {
				setPointerState(node, false, false)
			}
			for _, child := range node.Children {
				visit(child)
			}
		}
		visit(root)
	}
	if event.Kind == platform.EventWindowMouseLeave {
		clearPointerState()
		return dirty, a.passwordToggleCapture != 0, nil
	}
	if event.Kind == platform.EventWindowFocusLost {
		clearPointerState()
		handled = a.passwordToggleCapture != 0
		a.passwordToggleCapture = 0
		return dirty, handled, nil
	}

	updateHover := func() error {
		var visit func(*tree.Node) error
		visit = func(node *tree.Node) error {
			if node == nil {
				return nil
			}
			action, live := a.inputActions[node.ID]
			if live && action.passwordToggle {
				geometry := layout.Result{Rect: action.geometry, Content: action.content}
				hit, _, rectErr := passwordToggleRects(action.input, &geometry, a.theme)
				if rectErr != nil {
					return rectErr
				}
				hovered := !action.disabled && result.Target == node.ID && layoutRectContains(hit, event.Pointer.X, event.Pointer.Y)
				setPointerState(node, hovered, a.passwordToggleCapture == node.ID && hovered)
			}
			for _, child := range node.Children {
				if err := visit(child); err != nil {
					return err
				}
			}
			return nil
		}
		return visit(root)
	}
	if err := updateHover(); err != nil {
		return dirty, false, err
	}

	switch event.Kind {
	case platform.EventMouseMove:
		return dirty, a.passwordToggleCapture != 0, nil
	case platform.EventMouseDown:
		if event.Pointer.Button != platform.MouseButtonPrimary {
			return dirty, false, nil
		}
		action, ok := a.inputActions[result.Target]
		if !ok || !action.passwordToggle || action.disabled {
			return dirty, false, nil
		}
		node := findInstance(root, result.Target)
		if node == nil || !node.State.PasswordToggleHover {
			return dirty, false, nil
		}
		a.passwordToggleCapture = result.Target
		a.dragEditor = 0
		setPointerState(node, true, true)
		return dirty, true, nil
	case platform.EventMouseUp:
		captured := a.passwordToggleCapture
		if captured == 0 {
			return dirty, false, nil
		}
		a.passwordToggleCapture = 0
		node := findInstance(root, captured)
		action, live := a.inputActions[captured]
		activate := live && action.passwordToggle && !action.disabled && node != nil && node.State.PasswordToggleHover
		if node != nil {
			setPointerState(node, activate, false)
		}
		if activate {
			node.State.PasswordVisible = !node.State.PasswordVisible
			dirty = true
		}
		return dirty, true, nil
	}
	return dirty, false, nil
}

func (a *App) reconcilePasswordToggleState(actions map[uint64]inputAction) {
	if action, live := actions[a.passwordToggleCapture]; a.passwordToggleCapture != 0 && (!live || !action.passwordToggle || action.disabled) {
		a.passwordToggleCapture = 0
	}
	var visit func(*tree.Node)
	visit = func(node *tree.Node) {
		if node == nil {
			return
		}
		action, live := actions[node.ID]
		if !live || !action.passwordToggle || action.disabled {
			node.State.PasswordToggleHover = false
			node.State.PasswordTogglePressed = false
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(a.retained.Root())
}

func layoutRectContains(rect layout.Rect, x, y float32) bool {
	return rect.Width > 0 && rect.Height > 0 && x >= rect.X && y >= rect.Y && x < rect.X+rect.Width && y < rect.Y+rect.Height
}
