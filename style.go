package dxui

type lengthKind uint8

const (
	lengthAuto lengthKind = iota
	lengthPixels
	lengthPercent
)

// Length is an opaque automatic, logical-pixel, or percentage length. Its zero
// value means automatic sizing; use Px or Percent for an explicit value.
type Length struct {
	kind  lengthKind
	value float32
}

// Px creates a logical-pixel length.
func Px(value float32) Length { return Length{kind: lengthPixels, value: value} }

// Percent creates a percentage length in the range 0..100.
func Percent(value float32) Length { return Length{kind: lengthPercent, value: value} }

// Fill is the common full-available-axis length. It is equivalent to
// Percent(100) and remains subject to the parent's definite-size rules.
func Fill() Length { return Percent(100) }

// MetricToken names a theme metric.
type MetricToken string

// ColorToken names a theme color.
type ColorToken string

// MetricValue is either a literal logical-unit metric or a theme token.
type MetricValue struct {
	literal float32
	token   MetricToken
	isToken bool
	set     bool
}

// Metric creates a literal metric.
func Metric(value float32) MetricValue { return MetricValue{literal: value, set: true} }

// NoShrink explicitly disables flex shrinking. It is equivalent to Some(0)
// and is distinct from an unset Shrink, whose default is one.
func NoShrink() Option[float32] { return Some(float32(0)) }

// Padding applies one literal logical-unit metric to every edge. The returned
// metrics are explicitly set, including when value is zero.
func Padding(value float32) EdgeValues { return UniformEdges(Metric(value)) }

// PaddingXY applies literal logical-unit metrics to the horizontal and vertical
// edges. The returned metrics are explicitly set, including zero values.
func PaddingXY(horizontal, vertical float32) EdgeValues {
	return EdgeValues{
		Top: Metric(vertical), Right: Metric(horizontal),
		Bottom: Metric(vertical), Left: Metric(horizontal),
	}
}

// Margin applies one literal logical-unit metric to every edge, like Padding.
// The returned metrics are explicitly set, including when value is zero.
func Margin(value float32) EdgeValues { return Padding(value) }

// MarginXY applies literal logical-unit metrics to the horizontal and vertical
// edges, like PaddingXY. The returned metrics are explicitly set, including zero values.
func MarginXY(horizontal, vertical float32) EdgeValues { return PaddingXY(horizontal, vertical) }

// Round applies one literal logical-unit radius to every corner. The returned
// metrics are explicitly set, including when value is zero.
func Round(value float32) CornerValues { return UniformCorners(Metric(value)) }

// TokenMetric creates a token-backed metric.
func TokenMetric(token MetricToken) MetricValue {
	return MetricValue{token: token, isToken: true, set: true}
}

// ColorValue is either a literal color or a theme token.
type ColorValue struct {
	literal RGBAColor
	token   ColorToken
	isToken bool
	set     bool
}

// ColorRGBA creates an explicitly set literal paint color from RGBA channels,
// including transparent zero. Use RGBA for raw RGBAColor data instead.
func ColorRGBA(r, g, b, a uint8) ColorValue { return LiteralColor(RGBA(r, g, b, a)) }

// LiteralColor wraps existing raw RGBA data as a literal paint color.
// Use ColorRGBA when supplying channels directly.
func LiteralColor(value RGBAColor) ColorValue { return ColorValue{literal: value, set: true} }

// TokenColor creates a token-backed paint color.
func TokenColor(token ColorToken) ColorValue {
	return ColorValue{token: token, isToken: true, set: true}
}

// EdgeValues contains metrics in top, right, bottom, left order. Its zero-value
// fields are unset.
type EdgeValues struct{ Top, Right, Bottom, Left MetricValue }

// Edges creates explicit logical-unit edges in top, right, bottom, left order.
func Edges(top, right, bottom, left float32) EdgeValues {
	return EdgeValues{
		Top:    Metric(top),
		Right:  Metric(right),
		Bottom: Metric(bottom),
		Left:   Metric(left),
	}
}

// UniformEdges applies one metric to every edge.
func UniformEdges(value MetricValue) EdgeValues {
	return EdgeValues{value, value, value, value}
}

// CornerValues contains radii in top-left, top-right, bottom-right, bottom-left order.
// Its zero-value fields are unset.
type CornerValues struct{ TopLeft, TopRight, BottomRight, BottomLeft MetricValue }

// Corners creates explicit logical-unit radii in top-left, top-right,
// bottom-right, bottom-left order.
func Corners(topLeft, topRight, bottomRight, bottomLeft float32) CornerValues {
	return CornerValues{
		TopLeft:     Metric(topLeft),
		TopRight:    Metric(topRight),
		BottomRight: Metric(bottomRight),
		BottomLeft:  Metric(bottomLeft),
	}
}

// UniformCorners applies one radius to every corner.
func UniformCorners(value MetricValue) CornerValues {
	return CornerValues{value, value, value, value}
}

// Position controls normal-flow versus absolute layout.
type Position uint8

const (
	PositionFlow Position = iota
	PositionAbsolute
)

// Insets contains absolute-position insets.
type Insets struct{ Top, Right, Bottom, Left Length }

// Overflow controls container clipping.
type Overflow uint8

const (
	OverflowVisible Overflow = iota
	OverflowClip
)

// Align controls cross-axis alignment.
type Align uint8

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// Justify controls main-axis alignment.
type Justify uint8

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
)

// Style contains the supported layout, paint, and typography controls. Box
// is always single-line; there is intentionally no wrap or order
// property. The zero value is safe and selects intrinsic sizing and theme
// defaults.
type Style struct {
	Width, Height       Length
	MinWidth, MinHeight Length
	MaxWidth, MaxHeight Length
	Margin, Padding     EdgeValues
	Position            Position
	Insets              Insets
	Grow                float32
	Shrink              Option[float32]
	Basis               Length
	AlignSelf           Option[Align]
	ZIndex              int
	Overflow            Overflow
	Background          ColorValue
	Border              Border
	Radius              CornerValues
	// Shadow is an explicitly requested list of outer shadows. A nil value
	// leaves a theme/state value unchanged; a non-nil empty slice removes it.
	Shadow     []Shadow
	Opacity    Option[float32]
	Visibility Visibility
	Text       TextStyle
	// Force is applied after component and interaction-state styles. It is for
	// deliberate paint overrides such as suppressing a focus ring; ordinary
	// base appearance belongs in the fields above so Hover/Pressed/Focus remain
	// visible. Force cannot affect layout.
	Force StylePatch
}

// Visibility controls whether a node contributes display items.
type Visibility uint8

const (
	Visible Visibility = iota
	Hidden
)

// PointerBehavior controls pointer participation for a view subtree.
type PointerBehavior uint8

const (
	PointerAuto PointerBehavior = iota
	// PointerNone excludes the view and all descendants. It is intended for
	// decorative overlays that must not intercept content below them.
	PointerNone
)

// BorderPattern selects how a border is painted. The zero value is solid.
type BorderPattern uint8

const (
	BorderSolid BorderPattern = iota
	BorderDashed
)

// BorderSides selects edges. Zero means all edges, not an absent border.
type BorderSides uint8

const (
	BorderTop BorderSides = 1 << iota
	BorderRight
	BorderBottom
	BorderLeft
	BorderAll = BorderTop | BorderRight | BorderBottom | BorderLeft
)

// Border is a paint-only border drawn inside a view's layout bounds.
// Each side owns the nearest half of its two adjacent corner arcs.
type Border struct {
	Width   MetricValue
	Color   ColorValue
	Pattern BorderPattern
	// Sides defaults to all edges. In Style, a Width/Color assignment resets
	// Sides too; otherwise nonzero Sides changes only edge selection.
	// StylePatch.Border replaces the complete Border.
	Sides BorderSides
}

// NoBorder explicitly clears the border width. In ordinary Style, later state
// patches may restore a border; Style.Force is applied after those patches.
func NoBorder() Border { return Border{Width: Metric(0)} }

// Stroke creates a solid border with a literal logical-unit width.
func Stroke(width float32, color ColorValue) Border {
	return Border{Width: Metric(width), Color: color}
}

// Shadow is an outer paint-only shadow. MVP shadows support finite,
// non-negative blur/spread and are rendered by a bounded approximation.
type Shadow struct {
	OffsetX, OffsetY MetricValue
	Blur, Spread     MetricValue
	Color            ColorValue
}

// FontWeight selects the nearest registered face weight in a family.
type FontWeight uint16

const (
	WeightRegular FontWeight = 400
	WeightMedium  FontWeight = 500
	WeightBold    FontWeight = 700
)

// FontSlant selects a registered normal or italic face.
type FontSlant uint8

const (
	SlantNormal FontSlant = iota
	SlantItalic
)

// TextAlign is horizontal alignment within a Text node's content box.
type TextAlign uint8

const (
	TextStart TextAlign = iota
	TextCenter
	TextEnd
)

// TextStyle contains the simple-LTR/CJK MVP text inputs. Families are tried in
// order before the App default and built-in Latin fallback. Size is a font
// size in logical units; zero uses the current theme's default. LineHeight is
// a logical-unit line height; zero uses the current theme's default.
// Arabic/Indic shaping, bidi/RTL, color emoji, and vertical text are not
// supported by this API.
type TextStyle struct {
	Families   []FontFamily
	Size       float32
	LineHeight float32
	Weight     FontWeight
	Slant      FontSlant
	Color      ColorValue
	Align      TextAlign
}

// ComponentToken names a component-level theme entry.
type ComponentToken string

// PrimitiveTokens are the literal foundation of a theme.
type PrimitiveTokens struct {
	Colors  map[ColorToken]RGBAColor
	Metrics map[MetricToken]float32
}

// SemanticTokens map product meaning onto primitive or earlier semantic
// tokens. A literal ColorValue/MetricValue is also accepted.
type SemanticTokens struct {
	Colors  map[ColorToken]ColorValue
	Metrics map[MetricToken]MetricValue
}

// StylePatch is an explicitly optional paint-only override. State styles are
// intentionally paint-only in MVP, so interaction never moves layout.
type StylePatch struct {
	Background Option[ColorValue]
	Border     Option[Border]
	Radius     Option[CornerValues]
	Shadow     Option[[]Shadow]
	Opacity    Option[float32]
	Visibility Option[Visibility]
	TextColor  Option[ColorValue]
}

// StateStyles contains the deterministic MVP visual-state cascade.
type StateStyles struct {
	Default  StylePatch
	Hover    StylePatch
	Focus    StylePatch
	Disabled StylePatch
	Pressed  StylePatch
	Checked  StylePatch
}

// ComponentTheme supplies semantic-token-backed defaults for one component.
type ComponentTheme struct {
	Base   StylePatch
	States StateStyles
}

// Theme is the public, type-safe Primitive -> Semantic -> Component token
// structure. SetTheme validates and copies every map and slice atomically.
type Theme struct {
	Primitive  PrimitiveTokens
	Semantic   SemanticTokens
	Components map[ComponentToken]ComponentTheme
}

// viewProps is the normalized internal representation of the five fields that
// every public component props structure declares directly.
type viewProps struct {
	Key           string
	Style         Style
	Token         ComponentToken
	States        StateStyles
	Pointer       PointerBehavior
	buttonVariant ButtonVariant
	buttonTone    ButtonTone
}

// Direction selects Box's main axis. Vertical is the zero value.
type Direction uint8

const (
	// Vertical lays out Box children from top to bottom and is the zero value.
	Vertical Direction = iota
	// Horizontal lays out Box children from left to right.
	Horizontal
)

// BoxProps configures a Box. Gap is a fixed logical-unit spacing; zero means
// no spacing between children.
type BoxProps struct {
	Key     string
	Style   Style
	Token   ComponentToken
	States  StateStyles
	Pointer PointerBehavior

	Direction Direction
	Gap       float32
	Justify   Justify
	Align     Align
}

func (props BoxProps) common() viewProps {
	return newViewProps(props.Key, props.Style, props.Token, props.States, props.Pointer)
}
