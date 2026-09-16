package dxui

import (
	"image"
	"strings"
	"sync/atomic"
)

type viewKind uint8

const (
	viewInvalid viewKind = iota
	viewBox
	viewText
	viewButton
	viewButtonGroup
	viewInputGroup
	viewInput
	viewTextarea
	viewToggleSwitch
	viewSlider
	viewCheckbox
	viewRadio
	viewIcon
	viewImage
	viewAvatar
	viewBadge
	viewProgressBar
	viewControlMark
	viewScroll
	viewVirtualList
	viewSelect
	viewTabs
	viewMenu
	viewPopover
	viewTooltip
)

// View is an immutable description. Its representation is deliberately
// private and never contains backend or SDL handles. The zero value is invalid
// as a root or child; a failed tree update leaves the last valid view visible.
type View struct{ node *viewNode }

// WithKey returns an independent description whose sibling-local key is key.
// The receiver and any descriptions that share its node are unchanged. This
// does not add a container or retained identity level.
func (view View) WithKey(key string) View {
	return view.withCommon(func(props *viewProps) { props.Key = key })
}

// WithStyle returns an independent description whose complete local Style is
// style. Replacement is intentional: zero values, nil slices, and explicit
// empty slices keep their normal Style meanings. This does not add a container
// or retained identity level.
func (view View) WithStyle(style Style) View {
	style, _ = copyCommonValues(style, StateStyles{})
	return view.withCommon(func(props *viewProps) { props.Style = style })
}

func (view View) withCommon(edit func(*viewProps)) View {
	if view.node == nil {
		return view
	}
	node := *view.node
	props, err := nodeProps(view)
	if err != nil {
		return view
	}
	edit(&props)
	setNodeCommon(&node, props)
	node.key = props.Key
	return View{node: &node}
}

func setNodeCommon(node *viewNode, common viewProps) {
	switch node.kind {
	case viewBox:
		value := *node.box
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.box = &value
	case viewText:
		value := *node.text
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.text = &value
	case viewButton:
		value := *node.button
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.button = &value
	case viewButtonGroup:
		value := *node.buttonGroup
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.buttonGroup = &value
	case viewInputGroup:
		value := *node.inputGroup
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.inputGroup = &value
	case viewInput:
		value := *node.input
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.input = &value
	case viewTextarea:
		value := *node.textarea
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.textarea = &value
	case viewToggleSwitch:
		value := *node.toggle
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.toggle = &value
	case viewSlider:
		value := *node.slider
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.slider = &value
	case viewCheckbox:
		value := *node.checkbox
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.checkbox = &value
	case viewRadio:
		value := *node.radio
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.radio = &value
	case viewIcon:
		value := *node.icon
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.icon = &value
	case viewImage:
		value := *node.image
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.image = &value
	case viewAvatar:
		value := *node.avatar
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.avatar = &value
	case viewBadge:
		value := *node.badge
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.badge = &value
	case viewProgressBar:
		value := *node.progress
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.progress = &value
	case viewScroll:
		value := *node.scroll
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.scroll = &value
	case viewVirtualList:
		value := *node.virtualList
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.virtualList = &value
	case viewSelect:
		value := *node.selectp
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.selectp = &value
	case viewTabs:
		value := *node.tabs
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.tabs = &value
	case viewMenu:
		value := *node.menu
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.menu = &value
	case viewPopover:
		value := *node.popover
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.popover = &value
	case viewTooltip:
		value := *node.tooltip
		applyViewProps(&value.Key, &value.Style, &value.Token, &value.States, &value.Pointer, common)
		node.tooltip = &value
	case viewControlMark:
		value := *node.mark
		value.viewProps = common
		node.mark = &value
	}
}

func applyViewProps(key *string, style *Style, token *ComponentToken, states *StateStyles, pointer *PointerBehavior, value viewProps) {
	*key, *style, *token, *states, *pointer = value.Key, value.Style, value.Token, value.States, value.Pointer
}

type viewNode struct {
	kind         viewKind
	key          string
	children     []View
	box          *BoxProps
	text         *TextProps
	button       *ButtonProps
	buttonGroup  *ButtonGroupProps
	inputGroup   *InputGroupProps
	groupInput   int
	groupedInput bool
	groupChild   *buttonGroupChildStyle
	input        *InputProps
	textarea     *TextareaProps
	toggle       *ToggleSwitchProps
	slider       *SliderProps
	checkbox     *CheckboxProps
	radio        *RadioProps
	icon         *IconProps
	image        *ImageProps
	avatar       *AvatarProps
	badge        *BadgeProps
	progress     *ProgressBarProps
	mark         *controlMarkProps
	scroll       *ScrollProps
	virtualList  *VirtualListProps
	virtualKeys  []string
	selectp      *SelectProps
	tabs         *TabsProps
	menu         *MenuProps
	popover      *PopoverProps
	tooltip      *TooltipProps
}

type buttonGroupChildStyle struct {
	orientation ButtonGroupOrientation
	index       int
	count       int
	dividers    bool
}

type controlMarkProps struct {
	viewProps viewProps
	Size      MetricValue
}

func (props controlMarkProps) common() viewProps { return props.viewProps }

// Box creates a single-line flex description. Direction defaults to Vertical.
// Zero children are valid; a zero View among the supplied children is rejected
// transactionally.
func Box(props BoxProps, children ...View) View {
	copyChildren := append([]View(nil), children...)
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{
		kind: viewBox, key: props.Key, box: &copyProps, children: copyChildren,
	}}
}

// Scroll creates a clipped, one-child scrolling viewport.
func Scroll(props ScrollProps, child View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewScroll, key: props.Key, scroll: &copyProps, children: []View{child}}}
}

// VirtualList creates a vertical fixed-row virtualized viewport. The runtime
// builds only the visible rows plus the configured bounded overscan.
func VirtualList(props VirtualListProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewVirtualList, key: props.Key, virtualList: &copyProps}}
}

// Select creates a controlled custom popup Select. Options are copied.
func Select(props SelectProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	copyProps.Value = strings.ToValidUTF8(props.Value, "\uFFFD")
	copyProps.Placeholder = strings.ToValidUTF8(props.Placeholder, "\uFFFD")
	copyProps.Options = append([]SelectOption(nil), props.Options...)
	for index := range copyProps.Options {
		copyProps.Options[index].Value = strings.ToValidUTF8(copyProps.Options[index].Value, "\uFFFD")
		copyProps.Options[index].Label = strings.ToValidUTF8(copyProps.Options[index].Label, "\uFFFD")
	}
	applyDefaultPadding(&copyProps.Style, TokenMetric(MetricComponentSelectPaddingY), TokenMetric(MetricComponentSelectPaddingX))
	return View{node: &viewNode{kind: viewSelect, key: props.Key, selectp: &copyProps}}
}

// Popover creates a controlled interactive overlay around one anchor and one
// content view. The content remains retained but is painted only in the
// window-level overlay layer while Open is true.
func Popover(props PopoverProps, anchor, content View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	surface := overlaySurface(ComponentPopover, MetricComponentPopoverPaddingY, MetricComponentPopoverPaddingX, content)
	return View{node: &viewNode{kind: viewPopover, key: props.Key, popover: &copyProps, children: []View{anchor, surface}}}
}

// Tooltip creates a delayed, non-interactive overlay around one anchor and
// one content view.
func Tooltip(props TooltipProps, anchor, content View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	surface := overlaySurface(ComponentTooltip, MetricComponentTooltipPaddingY, MetricComponentTooltipPaddingX, content)
	return View{node: &viewNode{kind: viewTooltip, key: props.Key, tooltip: &copyProps, children: []View{anchor, surface}}}
}

func overlaySurface(token ComponentToken, vertical, horizontal MetricToken, content View) View {
	return Box(BoxProps{
		Token: token,
		Style: Style{Padding: EdgeValues{
			Top: TokenMetric(vertical), Right: TokenMetric(horizontal),
			Bottom: TokenMetric(vertical), Left: TokenMetric(horizontal),
		}},
	}, content)
}

// Tabs creates a controlled horizontal tab selector. Items are copied.
func Tabs(props TabsProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	copyProps.Value = strings.ToValidUTF8(props.Value, "\uFFFD")
	copyProps.Items = append([]TabItem(nil), props.Items...)
	for index := range copyProps.Items {
		copyProps.Items[index].Value = strings.ToValidUTF8(copyProps.Items[index].Value, "\uFFFD")
		copyProps.Items[index].Label = strings.ToValidUTF8(copyProps.Items[index].Label, "\uFFFD")
	}
	return View{node: &viewNode{kind: viewTabs, key: props.Key, tabs: &copyProps}}
}

// Menu creates an inline vertical or horizontal action list with an optional
// controlled selected Value. Items are copied. Vertical is the zero value.
func Menu(props MenuProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	copyProps.Value = strings.ToValidUTF8(props.Value, "\uFFFD")
	copyProps.Items = append([]MenuItem(nil), props.Items...)
	for index := range copyProps.Items {
		copyProps.Items[index].Value = strings.ToValidUTF8(copyProps.Items[index].Value, "\uFFFD")
		copyProps.Items[index].Label = strings.ToValidUTF8(copyProps.Items[index].Label, "\uFFFD")
	}
	return View{node: &viewNode{kind: viewMenu, key: props.Key, menu: &copyProps}}
}

// Text creates a text description.
func Text(props TextProps) View {
	copyProps := props
	copyProps.Value = strings.ToValidUTF8(props.Value, "\uFFFD")
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewText, key: props.Key, text: &copyProps}}
}

// Label creates text with the default text style.
func Label(value string) View { return Text(TextProps{Value: value}) }

// Button creates a semantic button description.
func Button(props ButtonProps, child View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{
		kind: viewButton, key: props.Key, button: &copyProps, children: []View{child},
	}}
}

// TextButton creates a Button whose text is centered in its content box.
// Use Button directly when the content is not a simple label.
func TextButton(props ButtonProps, label string) View {
	content := Box(BoxProps{Direction: Horizontal,
		Style:   Style{Width: Percent(100), Grow: 1},
		Justify: JustifyCenter,
		Align:   AlignCenter,
	}, Label(label))
	return Button(props, content)
}

// ButtonGroup creates a non-focusable horizontal or vertical group containing
// only Button views. Horizontal is the zero-value orientation.
func ButtonGroup(props ButtonGroupProps, buttons ...View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	copyButtons := make([]View, len(buttons))
	for index, button := range buttons {
		copyButtons[index] = button
		if button.node == nil || button.node.kind != viewButton {
			continue
		}
		childNode := *button.node
		childNode.groupChild = &buttonGroupChildStyle{
			orientation: props.Orientation,
			index:       index,
			count:       len(buttons),
			dividers:    props.Dividers,
		}
		copyButtons[index] = View{node: &childNode}
	}
	return View{node: &viewNode{
		kind: viewButtonGroup, key: props.Key, buttonGroup: &copyProps, children: copyButtons,
	}}
}

// InputGroup creates one horizontal visual control from a required Input and
// optional prefix/suffix views. The supplied Input keeps its own identity and
// editing state; the group suppresses only the Input's internal surface paint.
func InputGroup(props InputGroupProps, content InputGroupContent) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	applyDefaultPadding(&copyProps.Style, Metric(0), TokenMetric(MetricComponentInputGroupPaddingX))

	children := make([]View, 0, 3)
	inputIndex := 0
	if prefix, set := content.Prefix.get(); set {
		children = append(children, prefix)
		inputIndex = 1
	}
	input := content.Input
	if input.node != nil && input.node.kind == viewInput && input.node.input != nil {
		inputNode := *input.node
		inputProps := *input.node.input
		inputProps.Style, inputProps.States = copyCommonValues(inputProps.Style, inputProps.States)
		inputProps.Style.Grow = 1
		inputProps.Style.MinWidth = Px(0)
		inputProps.Style.Padding.Left = Metric(0)
		inputProps.Style.Padding.Right = Metric(0)
		inputProps.Style.Background = ColorRGBA(0, 0, 0, 0)
		inputProps.Style.Border = Border{Width: Metric(0)}
		inputProps.Style.Radius = Round(0)
		inputNode.input = &inputProps
		inputNode.groupedInput = true
		input = View{node: &inputNode}
	}
	children = append(children, input)
	if suffix, set := content.Suffix.get(); set {
		children = append(children, suffix)
	}
	return View{node: &viewNode{
		kind: viewInputGroup, key: props.Key, inputGroup: &copyProps, groupInput: inputIndex, children: children,
	}}
}

// Badge creates a compact, non-interactive one-child label.
func Badge(props BadgeProps, child View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	applyDefaultPadding(&copyProps.Style, TokenMetric(MetricComponentBadgePaddingY), TokenMetric(MetricComponentBadgePaddingX))
	return View{node: &viewNode{
		kind: viewBadge, key: props.Key, badge: &copyProps, children: []View{child},
	}}
}

// ProgressBar creates a deterministic, non-interactive progress indicator.
func ProgressBar(props ProgressBarProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewProgressBar, key: props.Key, progress: &copyProps}}
}

// ToggleSwitch creates a controlled semantic switch description.
func ToggleSwitch(props ToggleSwitchProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewToggleSwitch, key: props.Key, toggle: &copyProps}}
}

// Slider creates a controlled single-value horizontal slider description.
func Slider(props SliderProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewSlider, key: props.Key, slider: &copyProps}}
}

// Checkbox creates a controlled semantic checkbox with one label child.
func Checkbox(props CheckboxProps, label View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	mark := View{node: &viewNode{kind: viewControlMark, mark: &controlMarkProps{Size: TokenMetric(MetricComponentCheckboxSize)}}}
	return View{node: &viewNode{kind: viewCheckbox, key: props.Key, checkbox: &copyProps, children: []View{mark, label}}}
}

// Radio creates a controlled semantic radio button with one label child.
func Radio(props RadioProps, label View) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	mark := View{node: &viewNode{kind: viewControlMark, mark: &controlMarkProps{Size: TokenMetric(MetricComponentRadioSize)}}}
	return View{node: &viewNode{kind: viewRadio, key: props.Key, radio: &copyProps, children: []View{mark, label}}}
}

func applyDefaultPadding(style *Style, vertical, horizontal MetricValue) {
	padding := &style.Padding
	if !padding.Top.set {
		padding.Top = vertical
	}
	if !padding.Bottom.set {
		padding.Bottom = vertical
	}
	if !padding.Left.set {
		padding.Left = horizontal
	}
	if !padding.Right.set {
		padding.Right = horizontal
	}
}

// Icon creates a lightweight vector icon description.
func Icon(props IconProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	if !props.Data.IsPacked() {
		count := min(len(props.Data.Commands), maxIconCommands+1)
		copyProps.Data.Commands = append([]PathCommand(nil), props.Data.Commands[:count]...)
	}
	return View{node: &viewNode{kind: viewIcon, key: props.Key, icon: &copyProps}}
}

// Image creates a guarded raster image description.
func Image(props ImageProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewImage, key: props.Key, image: &copyProps}}
}

// Avatar creates a centered cover image with circular or square clipping.
func Avatar(props AvatarProps) View {
	copyProps := props
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	return View{node: &viewNode{kind: viewAvatar, key: props.Key, avatar: &copyProps}}
}

func imagePropsForAvatar(props *AvatarProps) ImageProps {
	return ImageProps{
		Key: props.Key, Style: props.Style, Token: props.Token, States: props.States, Pointer: props.Pointer, Source: props.Source, Fit: ImageCover,
		Alignment: Point{X: .5, Y: .5}, OnLoad: props.OnLoad, OnError: props.OnError,
	}
}

func imagePropsForView(view View) (ImageProps, bool) {
	if view.node == nil {
		return ImageProps{}, false
	}
	switch view.node.kind {
	case viewImage:
		if view.node.image != nil {
			return *view.node.image, true
		}
	case viewAvatar:
		if view.node.avatar != nil {
			return imagePropsForAvatar(view.node.avatar), true
		}
	}
	return ImageProps{}, false
}

// Input creates a controlled, single-line input description.
func Input(props InputProps) View {
	copyProps := props
	copyProps.Value = strings.ToValidUTF8(props.Value, "\uFFFD")
	copyProps.Placeholder = strings.ToValidUTF8(props.Placeholder, "\uFFFD")
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	applyDefaultPadding(&copyProps.Style, TokenMetric(MetricComponentInputPaddingY), TokenMetric(MetricComponentInputPaddingX))
	return View{node: &viewNode{kind: viewInput, key: props.Key, input: &copyProps}}
}

// Textarea creates a controlled multiline text editor description.
func Textarea(props TextareaProps) View {
	copyProps := props
	copyProps.Value = strings.ToValidUTF8(props.Value, "\uFFFD")
	copyProps.Placeholder = strings.ToValidUTF8(props.Placeholder, "\uFFFD")
	copyProps.Style, copyProps.States = copyCommonValues(props.Style, props.States)
	applyDefaultPadding(&copyProps.Style, TokenMetric(MetricComponentInputPaddingY), TokenMetric(MetricComponentInputPaddingX))
	return View{node: &viewNode{kind: viewTextarea, key: props.Key, textarea: &copyProps}}
}

var nextImageSourceID atomic.Uint64

func newImageByteSource(data []byte) *imageSource {
	return &imageSource{id: nextImageSourceID.Add(1), bytes: append([]byte(nil), data...)}
}

func newImageFileSource(path string) *imageSource {
	return &imageSource{id: nextImageSourceID.Add(1), path: path}
}

func newGoImageSource(value image.Image) *imageSource {
	return &imageSource{id: nextImageSourceID.Add(1), goImg: value}
}

func newViewProps(key string, style Style, token ComponentToken, states StateStyles, pointer PointerBehavior) viewProps {
	return viewProps{Key: key, Style: style, Token: token, States: states, Pointer: pointer}
}

func copyPropsValue(source viewProps) viewProps {
	source.Style.Shadow = cloneShadows(source.Style.Shadow)
	source.Style.Text.Families = append([]FontFamily(nil), source.Style.Text.Families...)
	source.Style.Force = copyStylePatch(source.Style.Force)
	source.States = copyStateStyles(source.States)
	return source
}

func copyCommonValues(style Style, states StateStyles) (Style, StateStyles) {
	props := copyPropsValue(viewProps{Style: style, States: states})
	return props.Style, props.States
}

func textPropsFromCommon(common viewProps, value string, wrap TextWrap, maxLines int) TextProps {
	return TextProps{
		Key: common.Key, Style: common.Style, Token: common.Token,
		States: common.States, Pointer: common.Pointer,
		Value: value, Wrap: wrap, MaxLines: maxLines,
	}
}

func cloneShadows(source []Shadow) []Shadow {
	if source == nil {
		return nil
	}
	return append(make([]Shadow, 0, len(source)), source...)
}
