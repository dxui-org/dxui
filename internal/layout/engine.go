package layout

import (
	"fmt"
	"math"
)

// LengthKind identifies the explicit ADR-0005 length variants.
type LengthKind uint8

const (
	LengthAuto LengthKind = iota
	LengthPixels
	LengthPercent
)

// Length never uses NaN, infinity, or a negative sentinel to represent auto.
type Length struct {
	Kind  LengthKind
	Value float32
}

// Edges are logical top, right, bottom, and left edge sizes.
type Edges struct{ Top, Right, Bottom, Left float32 }

// Insets are the four optional absolute-positioning offsets.
type Insets struct{ Top, Right, Bottom, Left Length }

// Axis identifies a leaf or one of the two single-line flex directions.
type Axis uint8

const (
	AxisLeaf Axis = iota
	AxisRow
	AxisColumn
)

// ScrollAxis marks a one-child viewport. Enabled axes give the child an
// unbounded maximum while the viewport itself keeps ordinary constraints.
type ScrollAxis uint8

const (
	ScrollNone ScrollAxis = iota
	ScrollVertical
	ScrollHorizontal
	ScrollBoth
)

// Position is normal flow or absolute positioning.
type Position uint8

const (
	PositionFlow Position = iota
	PositionAbsolute
)

// Overflow is the ADR-0005 visible/clip subset.
type Overflow uint8

const (
	OverflowVisible Overflow = iota
	OverflowClip
)

// Align is the cross-axis alignment subset.
type Align uint8

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// OptionalAlign distinguishes an explicit AlignStart from no override.
type OptionalAlign struct {
	Value Align
	Set   bool
}

// Style is the fully resolved, backend-neutral ADR-0005 layout style.
// Theme tokens are resolved before reaching this package.
type Style struct {
	Width, Height       Length
	MinWidth, MinHeight Length
	MaxWidth, MaxHeight Length
	Margin, Padding     Edges
	Position            Position
	Insets              Insets
	Grow, Shrink        float32
	ShrinkSet           bool
	Basis               Length
	AlignSelf           OptionalAlign
	ZIndex              int
	Overflow            Overflow
}

// Limit is a finite axis constraint. MaxSet=false is explicitly unbounded.
// Definite means this exact axis establishes a percentage containing size.
type Limit struct {
	Min      float32
	Max      float32
	MaxSet   bool
	Definite bool
}

// Constraints are the external width and height constraints for measurement.
type Constraints struct{ Width, Height Limit }

// Size is a logical-unit size.
type Size struct{ Width, Height float32 }

// Rect is a logical-unit rectangle.
type Rect struct{ X, Y, Width, Height float32 }

// IntrinsicRequest tells content measurers the available finite bounds. An
// unset maximum is unbounded; no special float value is used.
type IntrinsicRequest struct{ Width, Height Limit }

// IntrinsicSize provides minimum-content and preferred/max-content sizes.
type IntrinsicSize struct {
	Minimum   Size
	Preferred Size
}

// IntrinsicMeasurer is implemented by text and image adapters. It is pure Go
// and must return finite, non-negative logical sizes.
type IntrinsicMeasurer interface {
	MeasureIntrinsic(IntrinsicRequest) (IntrinsicSize, error)
}

// IntrinsicMeasureFunc adapts a function to IntrinsicMeasurer.
type IntrinsicMeasureFunc func(IntrinsicRequest) (IntrinsicSize, error)

// MeasureIntrinsic implements IntrinsicMeasurer.
func (measure IntrinsicMeasureFunc) MeasureIntrinsic(request IntrinsicRequest) (IntrinsicSize, error) {
	return measure(request)
}

// Node is one immutable input to a layout pass. ID and Version let an Engine
// reuse measurements across retained-tree commits; increment Version when the
// node's intrinsic content or resolved layout inputs change.
type Node struct {
	ID      uint64
	Version uint64
	Axis    Axis
	Style   Style
	Gap     float32
	// Overlap subtracts a fixed amount between adjacent in-flow items. It is
	// used by compound controls whose inside borders share one logical edge.
	Overlap  float32
	Justify  Justify
	Align    Align
	Children []*Node
	Measure  IntrinsicMeasurer
	Scroll   ScrollAxis
}

// ClipRect explicitly distinguishes no clip from an empty clip.
type ClipRect struct {
	Rect Rect
	Set  bool
}

// Result is the deterministic geometry for one node. Children remain in
// source/layout order; PaintOrder and HitTestOrder provide stacking order.
type Result struct {
	ID            uint64
	Rect          Rect
	Content       Rect
	Clip          ClipRect
	ZIndex        int
	SourceIndex   int
	Children      []*Result
	Viewport      Rect
	ContentExtent Size
}

// PaintOrder returns siblings in ascending (z-index, source-index) order.
func (result *Result) PaintOrder() []*Result {
	ordered := append([]*Result(nil), result.Children...)
	stableStackSort(ordered)
	return ordered
}

// HitTestOrder returns topmost siblings first.
func (result *Result) HitTestOrder() []*Result {
	ordered := result.PaintOrder()
	for left, right := 0, len(ordered)-1; left < right; left, right = left+1, right-1 {
		ordered[left], ordered[right] = ordered[right], ordered[left]
	}
	return ordered
}

func stableStackSort(values []*Result) {
	// Insertion sort is deterministic, stable, and allocation-free for the
	// typically short sibling lists used by UI descriptions.
	for index := 1; index < len(values); index++ {
		value := values[index]
		position := index
		for position > 0 && stackLess(value, values[position-1]) {
			values[position] = values[position-1]
			position--
		}
		values[position] = value
	}
}

func stackLess(left, right *Result) bool {
	if left.ZIndex != right.ZIndex {
		return left.ZIndex < right.ZIndex
	}
	return left.SourceIndex < right.SourceIndex
}

const measurementCacheEntryBytes = 192

// EngineOptions bounds the measurement cache by an estimated byte budget.
type EngineOptions struct{ MeasurementCacheBytes int }

// Stats describe the most recent pass and current bounded cache.
type Stats struct {
	MeasureCalls int
	CacheHits    int
	CacheEntries int
	CacheBytes   int
}

// Engine owns constraint/measurement cache state; it has no SDL dependency.
type Engine struct {
	maxCacheBytes int
	cache         map[measureKey]measureCacheEntry
	stats         Stats
}

type measureKey struct {
	id          uint64
	identity    *Node
	versionHash uint64
	constraints Constraints
	parent      definiteSize
}

type measureCacheEntry struct {
	measurement measurement
}

type definiteSize struct {
	size           Size
	widthDefinite  bool
	heightDefinite bool
}

type measurement struct {
	size           Size
	minimum        Size
	widthDefinite  bool
	heightDefinite bool
}

// NewEngine creates a layout engine. A zero budget uses 256 KiB; a negative
// budget disables measurement caching.
func NewEngine(options EngineOptions) *Engine {
	budget := options.MeasurementCacheBytes
	if budget == 0 {
		budget = 256 << 10
	}
	return &Engine{maxCacheBytes: budget, cache: make(map[measureKey]measureCacheEntry)}
}

// Stats returns counters for the most recently completed Layout call.
func (engine *Engine) Stats() Stats { return engine.stats }

// Layout validates, measures, and places a tree in logical coordinates.
func (engine *Engine) Layout(root *Node, constraints Constraints) (*Result, error) {
	if engine == nil {
		return nil, fmt.Errorf("layout: nil engine")
	}
	if root == nil {
		return nil, fmt.Errorf("layout: nil root")
	}
	constraints = normalizeConstraints(constraints)
	if err := validateConstraints(constraints); err != nil {
		return nil, err
	}
	if err := validateNode(root, "root"); err != nil {
		return nil, err
	}
	engine.stats.MeasureCalls = 0
	engine.stats.CacheHits = 0
	parent := definiteSize{
		size:           Size{Width: constraintReference(constraints.Width), Height: constraintReference(constraints.Height)},
		widthDefinite:  constraints.Width.Definite,
		heightDefinite: constraints.Height.Definite,
	}
	measured, err := engine.measureNode(root, constraints, parent)
	if err != nil {
		return nil, err
	}
	result, err := engine.layoutNode(root, Rect{Width: measured.size.Width, Height: measured.size.Height}, measured, ClipRect{})
	if err != nil {
		return nil, err
	}
	if err := validateResult(result); err != nil {
		return nil, err
	}
	engine.evict()
	engine.stats.CacheEntries = len(engine.cache)
	engine.stats.CacheBytes = len(engine.cache) * measurementCacheEntryBytes
	return result, nil
}

func (engine *Engine) measureNode(node *Node, constraints Constraints, parent definiteSize) (measurement, error) {
	constraints = normalizeConstraints(constraints)
	key := measureKey{id: node.ID, identity: node, versionHash: nodeVersionHash(node), constraints: constraints, parent: parent}
	if node.ID != 0 {
		key.identity = nil
	}
	if cached, ok := engine.cache[key]; ok {
		engine.stats.CacheHits++
		return cached.measurement, nil
	}
	engine.stats.MeasureCalls++
	padding := node.Style.Padding
	innerConstraints := subtractPadding(constraints, padding)

	width, widthSet := resolveLength(node.Style.Width, parent.size.Width, parent.widthDefinite)
	height, heightSet := resolveLength(node.Style.Height, parent.size.Height, parent.heightDefinite)
	if widthSet {
		innerConstraints.Width = Limit{Max: max(0, width-padding.Left-padding.Right), MaxSet: true, Definite: true}
	}
	if heightSet {
		innerConstraints.Height = Limit{Max: max(0, height-padding.Top-padding.Bottom), MaxSet: true, Definite: true}
	}
	provisional := definiteSize{
		size: Size{
			Width:  max(0, width-padding.Left-padding.Right),
			Height: max(0, height-padding.Top-padding.Bottom),
		},
		widthDefinite: widthSet, heightDefinite: heightSet,
	}
	if !widthSet && innerConstraints.Width.MaxSet {
		provisional.size.Width = innerConstraints.Width.Max
	}
	if !heightSet && innerConstraints.Height.MaxSet {
		provisional.size.Height = innerConstraints.Height.Max
	}

	intrinsic, err := engine.intrinsic(node, innerConstraints, provisional)
	if err != nil {
		return measurement{}, err
	}
	if !validSize(intrinsic.Minimum) || !validSize(intrinsic.Preferred) {
		return measurement{}, fmt.Errorf("layout: node %d intrinsic measurer returned an invalid size", node.ID)
	}
	intrinsic.Preferred.Width = max(intrinsic.Preferred.Width, intrinsic.Minimum.Width)
	intrinsic.Preferred.Height = max(intrinsic.Preferred.Height, intrinsic.Minimum.Height)

	if !widthSet {
		width = intrinsic.Preferred.Width + padding.Left + padding.Right
	}
	if !heightSet {
		height = intrinsic.Preferred.Height + padding.Top + padding.Bottom
	}
	minimum := Size{}
	if node.Axis == AxisLeaf {
		minimum = Size{
			Width:  intrinsic.Minimum.Width + padding.Left + padding.Right,
			Height: intrinsic.Minimum.Height + padding.Top + padding.Bottom,
		}
	}
	width, minimum.Width, widthSet = constrainDimension(width, minimum.Width, node.Style.MinWidth, node.Style.MaxWidth, constraints.Width, parent.size.Width, parent.widthDefinite, widthSet)
	height, minimum.Height, heightSet = constrainDimension(height, minimum.Height, node.Style.MinHeight, node.Style.MaxHeight, constraints.Height, parent.size.Height, parent.heightDefinite, heightSet)
	measured := measurement{size: Size{Width: width, Height: height}, minimum: minimum, widthDefinite: widthSet, heightDefinite: heightSet}
	if !validSize(measured.size) || !validSize(measured.minimum) {
		return measurement{}, fmt.Errorf("layout: non-finite measurement for node %d", node.ID)
	}
	if engine.maxCacheBytes >= measurementCacheEntryBytes {
		engine.makeCacheRoom()
		engine.cache[key] = measureCacheEntry{measurement: measured}
	}
	return measured, nil
}

func (engine *Engine) intrinsic(node *Node, constraints Constraints, provisional definiteSize) (IntrinsicSize, error) {
	if node.Scroll != ScrollNone {
		if len(node.Children) != 1 {
			return IntrinsicSize{}, fmt.Errorf("layout: scroll node %d must have exactly one child", node.ID)
		}
		childConstraints := scrollChildConstraints(constraints, node.Scroll)
		childMeasurement, err := engine.measureNode(node.Children[0], childConstraints, provisional)
		if err != nil {
			return IntrinsicSize{}, err
		}
		return IntrinsicSize{Preferred: childMeasurement.size}, nil
	}
	if node.Axis == AxisLeaf {
		if node.Measure == nil {
			return IntrinsicSize{}, nil
		}
		return node.Measure.MeasureIntrinsic(IntrinsicRequest{Width: constraints.Width, Height: constraints.Height})
	}
	mainRow := node.Axis == AxisRow
	mainPreferred, crossPreferred := float32(0), float32(0)
	flowCount := 0
	for _, child := range node.Children {
		if child.Style.Position == PositionAbsolute {
			continue
		}
		childConstraints := Constraints{}
		if mainRow {
			childConstraints.Height = constraints.Height
		} else {
			childConstraints.Width = constraints.Width
		}
		childMeasurement, err := engine.measureNode(child, childConstraints, provisional)
		if err != nil {
			return IntrinsicSize{}, err
		}
		basis, basisSet := resolveLength(child.Style.Basis, axisSize(provisional.size, mainRow), axisDefinite(provisional, mainRow))
		if !basisSet {
			basis = axisSize(childMeasurement.size, mainRow)
		}
		minimum := resolvedMinimum(child, mainRow, axisSize(childMeasurement.minimum, mainRow), provisional)
		basis = clamp(basis, minimum, resolvedMaximum(child, mainRow, minimum, provisional))
		mainMargins, crossMargins := axisMargins(child.Style.Margin, mainRow)
		mainPreferred, err = checkedAdd(mainPreferred, basis+mainMargins)
		if err != nil {
			return IntrinsicSize{}, err
		}
		crossPreferred = max(crossPreferred, axisSize(childMeasurement.size, !mainRow)+crossMargins)
		flowCount++
	}
	if flowCount > 1 {
		var err error
		mainPreferred, err = checkedAdd(mainPreferred, (node.Gap-node.Overlap)*float32(flowCount-1))
		if err != nil {
			return IntrinsicSize{}, err
		}
		mainPreferred = max(0, mainPreferred)
	}
	if mainRow {
		return IntrinsicSize{Preferred: Size{Width: mainPreferred, Height: crossPreferred}}, nil
	}
	return IntrinsicSize{Preferred: Size{Width: crossPreferred, Height: mainPreferred}}, nil
}

func (engine *Engine) layoutNode(node *Node, bounds Rect, measured measurement, inherited ClipRect) (*Result, error) {
	padding := node.Style.Padding
	content := Rect{
		X: bounds.X + padding.Left, Y: bounds.Y + padding.Top,
		Width:  max(0, bounds.Width-padding.Left-padding.Right),
		Height: max(0, bounds.Height-padding.Top-padding.Bottom),
	}
	clip := inherited
	if node.Style.Overflow == OverflowClip {
		clip = intersectClip(clip, bounds)
	}
	result := &Result{ID: node.ID, Rect: bounds, Content: content, Clip: clip, ZIndex: node.Style.ZIndex}
	if node.Scroll != ScrollNone {
		if len(node.Children) != 1 {
			return nil, fmt.Errorf("layout: scroll node %d must have exactly one child", node.ID)
		}
		childConstraints := scrollChildConstraints(Constraints{
			Width:  Limit{Max: content.Width, MaxSet: true, Definite: measured.widthDefinite},
			Height: Limit{Max: content.Height, MaxSet: true, Definite: measured.heightDefinite},
		}, node.Scroll)
		childMeasured, err := engine.measureNode(node.Children[0], childConstraints, definiteSize{
			size:          Size{Width: content.Width, Height: content.Height},
			widthDefinite: measured.widthDefinite, heightDefinite: measured.heightDefinite,
		})
		if err != nil {
			return nil, err
		}
		childResult, err := engine.layoutNode(node.Children[0], Rect{X: content.X, Y: content.Y, Width: childMeasured.size.Width, Height: childMeasured.size.Height}, childMeasured, intersectClip(inherited, content))
		if err != nil {
			return nil, err
		}
		childResult.SourceIndex = 0
		result.Children = []*Result{childResult}
		result.Viewport = content
		result.ContentExtent = childMeasured.size
		return result, nil
	}
	if node.Axis == AxisLeaf {
		return result, nil
	}

	mainRow := node.Axis == AxisRow
	container := definiteSize{
		size:          Size{Width: content.Width, Height: content.Height},
		widthDefinite: measured.widthDefinite, heightDefinite: measured.heightDefinite,
	}
	items := make([]FlexItem, len(node.Children))
	childMeasures := make([]measurement, len(node.Children))
	for index, child := range node.Children {
		if child.Style.Position == PositionAbsolute {
			items[index].Absolute = true
			continue
		}
		childConstraints := Constraints{}
		if mainRow {
			childConstraints.Height = Limit{Max: content.Height, MaxSet: true}
		} else {
			childConstraints.Width = Limit{Max: content.Width, MaxSet: true}
		}
		childMeasurement, err := engine.measureNode(child, childConstraints, container)
		if err != nil {
			return nil, err
		}
		childMeasures[index] = childMeasurement
		basis, basisSet := resolveLength(child.Style.Basis, axisSize(container.size, mainRow), axisDefinite(container, mainRow))
		if !basisSet {
			basis = axisSize(childMeasurement.size, mainRow)
		}
		minimum := axisSize(childMeasurement.minimum, mainRow)
		minimum = resolvedMinimum(child, mainRow, minimum, container)
		maximum := resolvedMaximum(child, mainRow, minimum, container)
		marginStart, marginEnd := mainAxisMargins(child.Style.Margin, mainRow)
		shrink := child.Style.Shrink
		if !child.Style.ShrinkSet {
			shrink = 1
		}
		items[index] = FlexItem{
			Basis: basis, Min: minimum, Max: maximum, Grow: child.Style.Grow, Shrink: shrink,
			MarginStart: marginStart, MarginEnd: marginEnd,
		}
	}
	line, err := flexLine(axisRectSize(content, mainRow), node.Gap, node.Overlap, node.Justify, items)
	if err != nil {
		return nil, err
	}
	result.Children = make([]*Result, len(node.Children))
	for index, child := range node.Children {
		var childResult *Result
		if child.Style.Position == PositionAbsolute {
			childResult, err = engine.layoutAbsolute(child, bounds, content, container, clip)
		} else {
			childResult, err = engine.layoutFlowChild(child, line[index], childMeasures[index], content, container, node.Align, mainRow, clip)
		}
		if err != nil {
			return nil, err
		}
		childResult.SourceIndex = index
		result.Children[index] = childResult
	}
	return result, nil
}

func scrollChildConstraints(viewport Constraints, axis ScrollAxis) Constraints {
	result := viewport
	if axis == ScrollHorizontal || axis == ScrollBoth {
		result.Width = Limit{}
	}
	if axis == ScrollVertical || axis == ScrollBoth {
		result.Height = Limit{}
	}
	return result
}

func (engine *Engine) layoutFlowChild(child *Node, line FlexResult, initial measurement, content Rect, container definiteSize, align Align, mainRow bool, clip ClipRect) (*Result, error) {
	selectedAlign := align
	if child.Style.AlignSelf.Set {
		selectedAlign = child.Style.AlignSelf.Value
	}
	mainStart, _ := mainAxisMargins(child.Style.Margin, mainRow)
	crossStart, crossEnd := crossAxisMargins(child.Style.Margin, mainRow)
	crossAvailable := axisRectSize(content, !mainRow)
	crossSize := axisSize(initial.size, !mainRow)
	crossAuto := axisLength(child.Style, !mainRow).Kind == LengthAuto
	if selectedAlign == AlignStretch && crossAuto {
		crossSize = max(0, crossAvailable-crossStart-crossEnd)
		crossSize = clampCross(child, crossSize, !mainRow, container)
	}

	childConstraints := Constraints{}
	setExactAxis(&childConstraints, mainRow, line.Size)
	if selectedAlign == AlignStretch && crossAuto {
		setExactAxis(&childConstraints, !mainRow, crossSize)
	} else {
		setMaxAxis(&childConstraints, !mainRow, crossSize)
	}
	measured, err := engine.measureNode(child, childConstraints, container)
	if err != nil {
		return nil, err
	}
	if !(selectedAlign == AlignStretch && crossAuto) {
		crossSize = axisSize(measured.size, !mainRow)
	}
	crossFree := max(0, crossAvailable-crossSize-crossStart-crossEnd)
	crossPosition := crossStart
	switch selectedAlign {
	case AlignStart, AlignStretch:
	case AlignCenter:
		crossPosition += crossFree / 2
	case AlignEnd:
		crossPosition += crossFree
	default:
		return nil, fmt.Errorf("layout: unsupported alignment %d", selectedAlign)
	}
	mainPosition := axisRectStart(content, mainRow) + line.Position
	crossPosition += axisRectStart(content, !mainRow)
	_ = mainStart // line.Position already includes the main-start margin.
	bounds := axisRect(mainPosition, crossPosition, line.Size, crossSize, mainRow)
	childMeasured := measured
	if mainRow {
		childMeasured.widthDefinite = true
		childMeasured.heightDefinite = childMeasured.heightDefinite || (selectedAlign == AlignStretch && crossAuto)
	} else {
		childMeasured.heightDefinite = true
		childMeasured.widthDefinite = childMeasured.widthDefinite || (selectedAlign == AlignStretch && crossAuto)
	}
	return engine.layoutNode(child, bounds, childMeasured, clip)
}

func (engine *Engine) layoutAbsolute(child *Node, paddingBox, content Rect, container definiteSize, clip ClipRect) (*Result, error) {
	initial, err := engine.measureNode(child, Constraints{}, container)
	if err != nil {
		return nil, err
	}
	x, width, widthDefinite := absoluteAxis(
		paddingBox.X, paddingBox.Width, content.X,
		child.Style.Insets.Left, child.Style.Insets.Right, child.Style.Width,
		child.Style.MinWidth, child.Style.MaxWidth,
		child.Style.Margin.Left, child.Style.Margin.Right,
		initial.size.Width, container.widthDefinite,
	)
	y, height, heightDefinite := absoluteAxis(
		paddingBox.Y, paddingBox.Height, content.Y,
		child.Style.Insets.Top, child.Style.Insets.Bottom, child.Style.Height,
		child.Style.MinHeight, child.Style.MaxHeight,
		child.Style.Margin.Top, child.Style.Margin.Bottom,
		initial.size.Height, container.heightDefinite,
	)
	constraints := Constraints{}
	setExactAxis(&constraints, true, width)
	setExactAxis(&constraints, false, height)
	measured, err := engine.measureNode(child, constraints, container)
	if err != nil {
		return nil, err
	}
	measured.widthDefinite = measured.widthDefinite || widthDefinite
	measured.heightDefinite = measured.heightDefinite || heightDefinite
	return engine.layoutNode(child, Rect{X: x, Y: y, Width: width, Height: height}, measured, clip)
}

func absoluteAxis(boxStart, boxSize, staticStart float32, startLength, endLength, sizeLength, minimumLength, maximumLength Length, marginStart, marginEnd, intrinsic float32, parentDefinite bool) (position, size float32, definite bool) {
	start, startSet := resolveLength(startLength, boxSize, parentDefinite)
	end, endSet := resolveLength(endLength, boxSize, parentDefinite)
	size, sizeSet := resolveLength(sizeLength, boxSize, parentDefinite)
	if startSet && endSet && !sizeSet {
		size = max(0, boxSize-start-end-marginStart-marginEnd)
		sizeSet = true
	} else if !sizeSet {
		size = intrinsic
	}
	minSize := float32(0)
	if value, ok := resolveLength(minimumLength, boxSize, parentDefinite); ok {
		minSize = value
	}
	maxSize := unbounded
	if value, ok := resolveLength(maximumLength, boxSize, parentDefinite); ok {
		maxSize = max(value, minSize)
	}
	size = clamp(size, minSize, maxSize)
	switch {
	case startSet:
		position = boxStart + start + marginStart
	case endSet:
		position = boxStart + boxSize - end - marginEnd - size
	default:
		position = staticStart + marginStart
	}
	return position, size, sizeSet
}

func constrainDimension(value, automaticMinimum float32, minimumLength, maximumLength Length, constraint Limit, parentSize float32, parentDefinite, definite bool) (float32, float32, bool) {
	minimum := automaticMinimum
	if resolved, ok := resolveLength(minimumLength, parentSize, parentDefinite); ok {
		minimum = resolved
	}
	maximum := unbounded
	if resolved, ok := resolveLength(maximumLength, parentSize, parentDefinite); ok {
		maximum = max(resolved, minimum)
	}
	minimum = max(minimum, constraint.Min)
	if constraint.MaxSet {
		maximum = max(min(constraint.Max, maximum), minimum)
	}
	clamped := clamp(value, minimum, maximum)
	if clamped != value || constraint.Definite {
		definite = true
	}
	return clamped, minimum, definite
}

func resolvedMinimum(node *Node, width bool, automatic float32, parent definiteSize) float32 {
	length := node.Style.MinHeight
	reference, definite := parent.size.Height, parent.heightDefinite
	if width {
		length, reference, definite = node.Style.MinWidth, parent.size.Width, parent.widthDefinite
	}
	if value, ok := resolveLength(length, reference, definite); ok {
		return value
	}
	if node.Axis != AxisLeaf {
		return 0
	}
	return automatic
}

func resolvedMaximum(node *Node, width bool, minimum float32, parent definiteSize) float32 {
	length := node.Style.MaxHeight
	reference, definite := parent.size.Height, parent.heightDefinite
	if width {
		length, reference, definite = node.Style.MaxWidth, parent.size.Width, parent.widthDefinite
	}
	if value, ok := resolveLength(length, reference, definite); ok {
		return max(value, minimum)
	}
	return unbounded
}

func clampCross(node *Node, value float32, width bool, parent definiteSize) float32 {
	minimum := resolvedMinimum(node, width, 0, parent)
	maximum := resolvedMaximum(node, width, minimum, parent)
	return clamp(value, minimum, maximum)
}

func resolveLength(length Length, reference float32, definite bool) (float32, bool) {
	switch length.Kind {
	case LengthPixels:
		return length.Value, true
	case LengthPercent:
		if definite {
			return (reference / 100) * length.Value, true
		}
	}
	return 0, false
}

const unbounded = float32(math.MaxFloat32 / 1024)

func validateNode(node *Node, path string) error {
	if node.Axis > AxisColumn || node.Scroll > ScrollBoth || node.Style.Position > PositionAbsolute || node.Style.Overflow > OverflowClip || node.Align > AlignStretch || node.Justify > JustifySpaceBetween {
		return fmt.Errorf("layout: %s has an unsupported enum value", path)
	}
	if !finiteNonNegative(node.Gap) || !finiteNonNegative(node.Overlap) || !finiteNonNegative(node.Style.Grow) || (node.Style.ShrinkSet && !finiteNonNegative(node.Style.Shrink)) {
		return fmt.Errorf("layout: %s has a negative or non-finite flex metric", path)
	}
	if node.Style.AlignSelf.Set && node.Style.AlignSelf.Value > AlignStretch {
		return fmt.Errorf("layout: %s has unsupported align-self", path)
	}
	lengths := []struct {
		name  string
		value Length
	}{
		{"width", node.Style.Width}, {"height", node.Style.Height},
		{"min-width", node.Style.MinWidth}, {"min-height", node.Style.MinHeight},
		{"max-width", node.Style.MaxWidth}, {"max-height", node.Style.MaxHeight},
		{"basis", node.Style.Basis}, {"top", node.Style.Insets.Top},
		{"right", node.Style.Insets.Right}, {"bottom", node.Style.Insets.Bottom}, {"left", node.Style.Insets.Left},
	}
	for _, item := range lengths {
		if err := validateLength(item.value); err != nil {
			return fmt.Errorf("layout: %s %s: %w", path, item.name, err)
		}
	}
	for _, value := range []float32{
		node.Style.Margin.Top, node.Style.Margin.Right, node.Style.Margin.Bottom, node.Style.Margin.Left,
		node.Style.Padding.Top, node.Style.Padding.Right, node.Style.Padding.Bottom, node.Style.Padding.Left,
	} {
		if !finiteNonNegative(value) {
			return fmt.Errorf("layout: %s has a negative or non-finite edge", path)
		}
	}
	for index, child := range node.Children {
		if child == nil {
			return fmt.Errorf("layout: %s child %d is nil", path, index)
		}
		if err := validateNode(child, fmt.Sprintf("%s child %d", path, index)); err != nil {
			return err
		}
	}
	return nil
}

func validateLength(length Length) error {
	if length.Kind > LengthPercent || !finite(length.Value) {
		return fmt.Errorf("invalid length")
	}
	if length.Kind == LengthPixels && length.Value < 0 {
		return fmt.Errorf("negative pixel length")
	}
	if length.Kind == LengthPercent && (length.Value < 0 || length.Value > 100) {
		return fmt.Errorf("percentage outside 0..100")
	}
	return nil
}

func validateConstraints(constraints Constraints) error {
	for _, value := range []Limit{constraints.Width, constraints.Height} {
		if !finiteNonNegative(value.Min) || (value.MaxSet && !finiteNonNegative(value.Max)) {
			return fmt.Errorf("layout: constraints must be finite and non-negative")
		}
	}
	return nil
}

func normalizeConstraints(value Constraints) Constraints {
	if value.Width.MaxSet && value.Width.Max < value.Width.Min {
		value.Width.Max = value.Width.Min
	}
	if value.Height.MaxSet && value.Height.Max < value.Height.Min {
		value.Height.Max = value.Height.Min
	}
	return value
}

func constraintReference(limit Limit) float32 {
	if limit.MaxSet {
		return limit.Max
	}
	return limit.Min
}

func subtractPadding(value Constraints, padding Edges) Constraints {
	value.Width.Min = max(0, value.Width.Min-padding.Left-padding.Right)
	value.Height.Min = max(0, value.Height.Min-padding.Top-padding.Bottom)
	if value.Width.MaxSet {
		value.Width.Max = max(0, value.Width.Max-padding.Left-padding.Right)
	}
	if value.Height.MaxSet {
		value.Height.Max = max(0, value.Height.Max-padding.Top-padding.Bottom)
	}
	return value
}

func intersectClip(parent ClipRect, current Rect) ClipRect {
	if !parent.Set {
		return ClipRect{Rect: current, Set: true}
	}
	left := max(parent.Rect.X, current.X)
	top := max(parent.Rect.Y, current.Y)
	right := min(parent.Rect.X+parent.Rect.Width, current.X+current.Width)
	bottom := min(parent.Rect.Y+parent.Rect.Height, current.Y+current.Height)
	return ClipRect{Rect: Rect{X: left, Y: top, Width: max(0, right-left), Height: max(0, bottom-top)}, Set: true}
}

func axisSize(size Size, width bool) float32 {
	if width {
		return size.Width
	}
	return size.Height
}

func axisDefinite(value definiteSize, width bool) bool {
	if width {
		return value.widthDefinite
	}
	return value.heightDefinite
}

func axisRectSize(rect Rect, width bool) float32 {
	if width {
		return rect.Width
	}
	return rect.Height
}

func axisRectStart(rect Rect, width bool) float32 {
	if width {
		return rect.X
	}
	return rect.Y
}

func axisRect(mainPosition, crossPosition, mainSize, crossSize float32, mainRow bool) Rect {
	if mainRow {
		return Rect{X: mainPosition, Y: crossPosition, Width: mainSize, Height: crossSize}
	}
	return Rect{X: crossPosition, Y: mainPosition, Width: crossSize, Height: mainSize}
}

func axisLength(style Style, width bool) Length {
	if width {
		return style.Width
	}
	return style.Height
}

func axisMargins(edges Edges, mainRow bool) (main, cross float32) {
	if mainRow {
		return edges.Left + edges.Right, edges.Top + edges.Bottom
	}
	return edges.Top + edges.Bottom, edges.Left + edges.Right
}

func mainAxisMargins(edges Edges, mainRow bool) (start, end float32) {
	if mainRow {
		return edges.Left, edges.Right
	}
	return edges.Top, edges.Bottom
}

func crossAxisMargins(edges Edges, mainRow bool) (start, end float32) {
	if mainRow {
		return edges.Top, edges.Bottom
	}
	return edges.Left, edges.Right
}

func setExactAxis(constraints *Constraints, width bool, value float32) {
	limit := Limit{Min: value, Max: value, MaxSet: true, Definite: true}
	if width {
		constraints.Width = limit
	} else {
		constraints.Height = limit
	}
}

func setMaxAxis(constraints *Constraints, width bool, value float32) {
	limit := Limit{Max: value, MaxSet: true}
	if width {
		constraints.Width = limit
	} else {
		constraints.Height = limit
	}
}

func checkedAdd(left, right float32) (float32, error) {
	result := float64(left) + float64(right)
	if math.IsNaN(result) || math.IsInf(result, 0) || result > math.MaxFloat32 {
		return 0, fmt.Errorf("layout: dimension arithmetic overflow")
	}
	return float32(result), nil
}

func validSize(size Size) bool {
	return finiteNonNegative(size.Width) && finiteNonNegative(size.Height)
}

func finite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

func nodeVersionHash(node *Node) uint64 {
	hash := uint64(1469598103934665603)
	mix := func(value uint64) { hash = (hash ^ value) * 1099511628211 }
	mix(node.ID)
	mix(node.Version)
	mix(uint64(node.Axis))
	mix(uint64(node.Scroll))
	mix(uint64(node.Justify))
	mix(uint64(node.Align))
	mix(uint64(math.Float32bits(node.Gap)))
	mix(uint64(math.Float32bits(node.Overlap)))
	mixStyle := func(style Style) {
		for _, length := range []Length{
			style.Width, style.Height, style.MinWidth, style.MinHeight,
			style.MaxWidth, style.MaxHeight, style.Basis,
			style.Insets.Top, style.Insets.Right, style.Insets.Bottom, style.Insets.Left,
		} {
			mix(uint64(length.Kind))
			mix(uint64(math.Float32bits(length.Value)))
		}
		for _, value := range []float32{
			style.Margin.Top, style.Margin.Right, style.Margin.Bottom, style.Margin.Left,
			style.Padding.Top, style.Padding.Right, style.Padding.Bottom, style.Padding.Left,
			style.Grow, style.Shrink,
		} {
			mix(uint64(math.Float32bits(value)))
		}
		mix(uint64(style.Position))
		mix(uint64(style.AlignSelf.Value))
		if style.ShrinkSet {
			mix(1)
		}
		if style.AlignSelf.Set {
			mix(2)
		}
	}
	mixStyle(node.Style)
	mix(uint64(len(node.Children)))
	for _, child := range node.Children {
		mix(nodeVersionHash(child))
	}
	return hash
}

func (engine *Engine) evict() {
	if engine.maxCacheBytes < measurementCacheEntryBytes {
		clear(engine.cache)
		return
	}
	for len(engine.cache)*measurementCacheEntryBytes > engine.maxCacheBytes {
		engine.evictOne()
	}
}

func (engine *Engine) makeCacheRoom() {
	maximumEntries := engine.maxCacheBytes / measurementCacheEntryBytes
	if maximumEntries <= 0 || len(engine.cache) < maximumEntries {
		return
	}
	engine.evictOne()
}

func (engine *Engine) evictOne() {
	// Measurements are immutable values with no external references. Any entry
	// is therefore safe to evict; choosing the first map entry avoids an O(n)
	// LRU scan at the byte bound during resize storms.
	for key := range engine.cache {
		delete(engine.cache, key)
		return
	}
}

func validateResult(result *Result) error {
	if result == nil || !finite(result.Rect.X) || !finite(result.Rect.Y) || !validSize(Size{Width: result.Rect.Width, Height: result.Rect.Height}) {
		return fmt.Errorf("layout: produced invalid geometry")
	}
	if result.Clip.Set && (!finite(result.Clip.Rect.X) || !finite(result.Clip.Rect.Y) || !validSize(Size{Width: result.Clip.Rect.Width, Height: result.Clip.Rect.Height})) {
		return fmt.Errorf("layout: produced invalid clip geometry")
	}
	for _, child := range result.Children {
		if err := validateResult(child); err != nil {
			return err
		}
	}
	return nil
}
