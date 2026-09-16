package dxui

import (
	"image"
	"time"

	"github.com/dxui-org/dxui/internal/icondata"
)

// ScrollAxis selects the axes whose content is measured without a maximum
// and whose retained offset may change.
type ScrollAxis uint8

const (
	ScrollVertical ScrollAxis = iota
	ScrollHorizontal
	ScrollBoth
)

// ScrollbarPolicy controls the overlay scrollbar. Hidden suppresses scrollbar
// paint and pointer interaction without disabling wheel/trackpad scrolling.
type ScrollbarPolicy uint8

const (
	ScrollbarAuto ScrollbarPolicy = iota
	ScrollbarAlways
	ScrollbarHidden
)

// ScrollProps configures a one-child clipped viewport. Offset is authoritative
// when set. Otherwise InitialOffset is used only when the view is first
// mounted; later positions are preserved while the view keeps its identity.
// OnScroll reports the complete offset. Without Offset, movement updates
// retained state even with a nil callback; with Offset it only proposes a
// change. Scroll has no Disabled or ReadOnly property.
type ScrollProps struct {
	Key           string
	Style         Style
	Token         ComponentToken
	States        StateStyles
	Pointer       PointerBehavior
	Axis          ScrollAxis
	InitialOffset Option[Point]
	Offset        Option[Point]
	Scrollbar     ScrollbarPolicy
	// OnScroll reports movement. Nil still scrolls internally when Offset is unset; a set Offset stays authoritative.
	OnScroll func(Point)
}

// VirtualListProps configures a vertical, fixed-row-height virtual list.
// Count and Version identify one immutable data snapshot; increment Version
// whenever row keys or content may have changed. ItemKey and Build run on the
// UI thread and must be deterministic and side-effect free for that snapshot.
type VirtualListProps struct {
	Key           string
	Style         Style
	Token         ComponentToken
	States        StateStyles
	Pointer       PointerBehavior
	Count         int
	Version       uint64
	RowHeight     float32
	Overscan      int
	InitialOffset Option[Point]
	Offset        Option[Point]
	Scrollbar     ScrollbarPolicy
	ItemKey       func(index int) string
	Build         func(index int) View
	OnScroll      func(Point)
}

// TextWrap selects the MVP simple wrapping policy.
type TextWrap uint8

const (
	TextNoWrap TextWrap = iota
	TextWrapWords
)

// TextProps configures simple left-to-right text. Invalid UTF-8 is normalized
// to U+FFFD when Text is constructed. TextWrapWords collapses whitespace and
// wraps only at word boundaries; it is not Unicode line-break conformance.
type TextProps struct {
	Key      string
	Style    Style
	Token    ComponentToken
	States   StateStyles
	Pointer  PointerBehavior
	Value    string
	Wrap     TextWrap
	MaxLines int
}

// ButtonProps configures a semantic button.
type ButtonProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Variant ButtonVariant
	Tone    ButtonTone
	Size    ButtonSize
	// Disabled removes focus and activation and suppresses Hover, Pressed, and Focus styles.
	Disabled bool
	// OnPress runs on activation. Nil preserves focus and interaction visuals but emits no action.
	OnPress func()
}

// ButtonGroupOrientation selects the main axis used to arrange buttons.
type ButtonGroupOrientation uint8

const (
	ButtonGroupHorizontal ButtonGroupOrientation = iota
	ButtonGroupVertical
)

// ButtonGroupProps configures a non-focusable connected container of Button
// views.
type ButtonGroupProps struct {
	Key         string
	Style       Style
	Token       ComponentToken
	States      StateStyles
	Pointer     PointerBehavior
	Orientation ButtonGroupOrientation
	// Dividers enables one themed border between adjacent Buttons.
	Dividers bool
}

// InputGroupProps configures the non-focusable visual container around one
// Input and its optional leading and trailing content.
type InputGroupProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
}

// InputGroupContent identifies the required Input and optional adornments.
// Prefix and Suffix may contain passive views or Buttons; other focusable
// controls and nested text editors are rejected.
type InputGroupContent struct {
	Input  View
	Prefix Option[View]
	Suffix Option[View]
}

// BadgeProps configures a compact, non-interactive label around one arbitrary
// child. The child inherits the Badge text/icon tint unless locally overridden.
type BadgeProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
}

// ProgressBarProps configures a deterministic, non-interactive progress
// indicator. Value is clamped to [0,1] for painting. NaN and negative
// infinity paint empty; positive infinity paints complete.
type ProgressBarProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Value   float32
}

// PathVerb identifies one command in dxui's deliberately small vector icon
// format. It is not an SVG parser or a general scene graph.
type PathVerb = icondata.PathVerb

const (
	PathMove  = icondata.PathMove
	PathLine  = icondata.PathLine
	PathQuad  = icondata.PathQuad
	PathCubic = icondata.PathCubic
	PathClose = icondata.PathClose
)

// PathCommand stores up to three points. Move/Line use Points[0], Quad uses
// Points[0:2], and Cubic uses all three points.
type PathCommand = icondata.PathCommand

// Rect is a rectangle in logical units.
type Rect = icondata.Rect

// IconData is immutable dxui path data in ViewBox coordinates.
type IconData = icondata.Data

// IconStrokeWidth is a stroke width in icon view-box units. Zero selects the
// Lucide default of 2. It affects immutable stroked resources and is ignored
// by legacy filled IconData.
type IconStrokeWidth float32

// IconProps configures a vector icon. Size is measured in logical units; zero
// uses the current component theme default.
type IconProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Data    IconData
	Size    float32
	Color   ColorValue
	// StrokeWidth uses icon view-box units and defaults to 2. Values must be
	// finite and non-negative; zero means the default rather than no stroke.
	StrokeWidth IconStrokeWidth
}

// ToggleSwitchProps configures a controlled switch.
type ToggleSwitchProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	// Checked is authoritative. Disabled cancels interaction and removes the focus stop.
	Checked, Disabled bool
	// OnChange proposes Checked. Nil preserves focus/press visuals without changing Checked.
	OnChange func(bool)
}

// SliderProps configures a controlled single-value horizontal slider. Value
// is authoritative; OnChange receives a clamped, step-aligned proposal. Zero
// Min and Max select the default 0..100 range. Step defaults to 1 when it is
// non-positive or non-finite. Other invalid ranges are inert.
type SliderProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Value   float32
	Min     float32
	Max     float32
	Step    float32
	// Disabled cancels dragging and removes focus and pointer interaction.
	Disabled bool
	// OnChange proposes Value. Nil preserves focus and drag visuals without changing Value.
	OnChange func(float32)
}

// CheckboxProps configures a controlled two-state checkbox.
type CheckboxProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	// Checked is authoritative. Disabled cancels interaction and removes the focus stop.
	Checked, Disabled bool
	// OnChange proposes Checked. Nil preserves focus/press visuals without changing Checked.
	OnChange func(bool)
}

// RadioProps configures a controlled radio button. Selected is authoritative;
// OnSelect is called only when an enabled, unselected Radio is activated.
type RadioProps struct {
	Key      string
	Style    Style
	Token    ComponentToken
	States   StateStyles
	Pointer  PointerBehavior
	Selected bool
	// Disabled cancels interaction and removes the focus stop.
	Disabled bool
	// OnSelect proposes selection of an unselected Radio. Nil preserves focus and press visuals.
	OnSelect func()
}

type imageSource struct {
	id    uint64
	bytes []byte
	path  string
	goImg image.Image
}

// ImageSource is an immutable, comparable handle to application image data.
type ImageSource struct{ source *imageSource }

// ImageBytes copies encoded PNG, JPEG, or GIF bytes immediately.
func ImageBytes(data []byte) ImageSource {
	return ImageSource{source: newImageByteSource(data)}
}

// ImageFile records a path that is read by the image engine on demand.
func ImageFile(path string) ImageSource { return ImageSource{source: newImageFileSource(path)} }

// ImageFromGo records an immutable-by-contract Go image source.
func ImageFromGo(value image.Image) ImageSource { return ImageSource{source: newGoImageSource(value)} }

// ImageFit selects the supported destination fitting policy.
type ImageFit uint8

const (
	ImageContain ImageFit = iota
	ImageCover
	ImageFill
	ImageNone
)

// ImageProps configures a decoded raster image. Alignment components are in
// [0,1].
type ImageProps struct {
	Key       string
	Style     Style
	Token     ComponentToken
	States    StateStyles
	Pointer   PointerBehavior
	Source    ImageSource
	Fit       ImageFit
	Alignment Point
	MaxPixels int64
	// OnLoad and OnError are optional notifications; nil does not stop decoding.
	OnLoad  func(Size)
	OnError func(error)
}

// AvatarShape selects the fixed Avatar clipping shape.
type AvatarShape uint8

const (
	AvatarCircle AvatarShape = iota
	AvatarSquare
)

// AvatarProps configures a square, centered cover image. Size is measured in
// logical units; zero uses the current component theme default. Explicit Style
// width and height take precedence.
type AvatarProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Source  ImageSource
	Shape   AvatarShape
	Size    float32
	// OnLoad and OnError are optional notifications; nil does not stop decoding.
	OnLoad  func(Size)
	OnError func(error)
}

// TextRange uses Unicode code-point (rune) offsets, never UTF-8 byte offsets.
type TextRange struct{ Start, End int }

// InputProps configures a controlled single-line input. OnChange receives the
// complete proposed value after committed text, paste, cut, deletion, undo, or
// redo. The application accepts the edit by returning that value from the next
// build; leaving Value unchanged rejects it. IME composition does not call
// OnChange. Selection and TextRange use rune indices. A nil OnChange makes the
// input read-only by behavior, without Disabled styling. ReadOnly also blocks
// pre-edit; focus, selection, non-password copy, and OnSubmit remain available.
type InputProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Value   string
	// OnChange proposes Value. Nil blocks editing and pre-edit, but preserves selection and copy.
	OnChange  func(string)
	Selection Option[TextRange]
	// OnSelectionChange reports rune selection. Nil retains internal selection; Selection, when set, wins on rebuild.
	OnSelectionChange func(TextRange)
	Placeholder       string
	Password          bool
	// ShowPasswordToggle adds an internal trailing visibility button only when
	// Password is also true. Visibility is temporary state owned by the Input.
	ShowPasswordToggle bool
	// Disabled removes focus, stops native text input, and cancels composition, drag, and press.
	Disabled bool
	// ReadOnly blocks edits and pre-edit while allowing focus, navigation, selection, and non-password copy.
	ReadOnly bool
	// OnSubmit handles Enter independently of OnChange and ReadOnly. Nil emits no submit; Disabled blocks it.
	OnSubmit func()
}

// TextareaProps configures a controlled multiline editor and follows the same
// controlled Value and rune-indexed selection contract as Input. TextWrapWords
// is a simple LTR/CJK word-wrap policy, not full Unicode line breaking. A nil
// OnChange preserves navigation, selection, copy, and scrolling without edits
// or pre-edit; it does not imply Disabled styling.
type TextareaProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Value   string
	// OnChange proposes Value. Nil blocks editing and pre-edit, but preserves selection and copy.
	OnChange  func(string)
	Selection Option[TextRange]
	// OnSelectionChange reports rune selection. Nil retains internal selection; Selection, when set, wins on rebuild.
	OnSelectionChange func(TextRange)
	Placeholder       string
	// Disabled removes focus, stops native text input, and cancels composition, drag, and press.
	Disabled bool
	// ReadOnly blocks edits and pre-edit while allowing focus, navigation, selection, and non-password copy.
	ReadOnly bool
	Wrap     TextWrap
}

// SelectOption is one immutable entry in a Select. Value must be non-empty and
// unique within the Select; it is both the controlled application value and
// the stable identity used when options reorder.
type SelectOption struct {
	Value    string
	Label    string
	Disabled bool
}

const selectOptionLimit = 4096

// SelectProps configures a controlled custom popup Select. Value is
// authoritative; an empty or unmatched Value displays Placeholder.
type SelectProps struct {
	Key         string
	Style       Style
	Token       ComponentToken
	States      StateStyles
	Pointer     PointerBehavior
	Value       string
	Options     []SelectOption
	Placeholder string
	// Disabled closes the popup, cancels interaction, and removes the focus stop.
	Disabled bool
	// OnChange proposes Value. Nil still allows popup browsing, navigation, and dismissal.
	OnChange func(string)
}

// OverlayPlacement selects the preferred side and alignment of a window-level
// overlay. Placement automatically flips to the opposite side when it has
// more usable room and the preferred side cannot fit the content.
type OverlayPlacement uint8

const (
	OverlayBottomStart OverlayPlacement = iota
	OverlayBottom
	OverlayBottomEnd
	OverlayTopStart
	OverlayTop
	OverlayTopEnd
	OverlayLeft
	OverlayRight
)

// PopoverProps configures a controlled, interactive window-level overlay.
// Open is authoritative. Anchor activation, Escape, and an outside primary
// click submit the proposed state through OnOpenChange. Nil leaves Open
// unchanged and preserves open-content interaction. There is no Disabled or
// ReadOnly property; an anchor child does not disable the Popover host.
type PopoverProps struct {
	Key       string
	Style     Style
	Token     ComponentToken
	States    StateStyles
	Pointer   PointerBehavior
	Open      bool
	Placement OverlayPlacement
	Offset    MetricValue
	// OnOpenChange proposes Open. Nil leaves Open unchanged; open content remains interactive.
	OnOpenChange func(bool)
}

// TooltipProps configures a non-interactive window-level hint. Hovering or
// keyboard-focusing the anchor starts Delay; leaving both closes it. A zero
// Delay selects the built-in 500 ms default.
type TooltipProps struct {
	Key       string
	Style     Style
	Token     ComponentToken
	States    StateStyles
	Pointer   PointerBehavior
	Placement OverlayPlacement
	Offset    MetricValue
	Delay     time.Duration
	Disabled  bool
}

// TabItem is one immutable label in Tabs. Value must be non-empty and unique
// within the component.
type TabItem struct {
	Value    string
	Label    string
	Disabled bool
}

// TabsProps configures a controlled horizontal tab selector. Value is
// authoritative; Tabs renders only the selector and applications render the
// corresponding content separately.
type TabsProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior
	Value   string
	Items   []TabItem
	// Disabled cancels interaction and removes the focus stop.
	Disabled bool
	// OnChange proposes a different Value. Nil retains active-item navigation and press visuals.
	OnChange func(string)
}

// MenuOrientation selects the axis used to arrange Menu items.
type MenuOrientation uint8

const (
	MenuVertical MenuOrientation = iota
	MenuHorizontal
)

// MenuItem is one immutable action in a Menu. Value must be non-empty and
// unique within the component.
type MenuItem struct {
	Value    string
	Label    string
	Disabled bool
}

// MenuProps configures an inline action list. Value optionally identifies the
// application-controlled selected item. Activating an enabled item calls
// OnAction with that item's Value, including when it is already selected.
type MenuProps struct {
	Key         string
	Style       Style
	Token       ComponentToken
	States      StateStyles
	Pointer     PointerBehavior
	Orientation MenuOrientation
	Value       string
	Items       []MenuItem
	// Disabled cancels interaction and removes the focus stop.
	Disabled bool
	// OnAction invokes an enabled item, including the selected one. Nil retains navigation and visuals.
	OnAction func(string)
}

func (props ScrollProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props VirtualListProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props TextProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props ButtonProps) common() viewProps {
	result := newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
	result.buttonVariant, result.buttonTone = props.Variant, props.Tone
	variant := props.Variant
	if props.Token != "" {
		variant = ButtonFilled
	}
	applyButtonSize(&result.Style, variant, props.Size)
	return result
}
func (props ButtonGroupProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props InputGroupProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props BadgeProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props ProgressBarProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props IconProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props ToggleSwitchProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props SliderProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props CheckboxProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props RadioProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props ImageProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props AvatarProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props InputProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props TextareaProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props SelectProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props PopoverProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props TooltipProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props TabsProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
func (props MenuProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
