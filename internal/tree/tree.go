// Package tree owns retained identity, transactional reconciliation, dirty
// propagation, local state, and instance resource lifetimes.
package tree

import "fmt"

// Kind identifies a framework node kind without exposing public internals.
type Kind uint8

// Key is sibling-local retained identity.
type Key string

// Dirty is a set of independent, monotonic invalidation classes.
type Dirty uint16

const (
	DirtyBuild Dirty = 1 << iota
	DirtyStyle
	DirtyMeasure
	DirtyLayout
	DirtyDisplay
	DirtyPaint
	DirtySemantics
	DirtyResource
)

const dirtyNew = DirtyStyle | DirtyMeasure | DirtyLayout | DirtyDisplay | DirtyPaint | DirtySemantics | DirtyResource

// Value is a comparable literal-or-token input. It deliberately contains no
// dynamically typed value and is used for exact typed-property comparison.
type Value struct {
	Token   string
	Literal float32
	IsToken bool
	Set     bool
}

// Length is a comparable layout length.
type Length struct {
	Kind  uint8
	Value float32
}

// Edges contains top, right, bottom, and left values.
type Edges struct{ Top, Right, Bottom, Left Value }

// LayoutProperties are inputs that can affect measurement or geometry.
type LayoutProperties struct {
	Width, Height       Length
	MinWidth, MinHeight Length
	MaxWidth, MaxHeight Length
	Margin, Padding     Edges
	InsetTop            Length
	InsetRight          Length
	InsetBottom         Length
	InsetLeft           Length
	Grow                float32
	Shrink              float32
	ShrinkSet           bool
	Basis               Length
	AlignSelf           uint8
	AlignSelfSet        bool
	Position            uint8
	ZIndex              int
	Overflow            uint8
	Axis                uint8
	Gap                 Value
	Overlap             Value
	Justify             uint8
	Align               uint8
}

// PaintProperties are visual inputs that do not affect layout in this slice.
type PaintProperties struct {
	BackgroundR, BackgroundG, BackgroundB, BackgroundA uint8
	BackgroundToken                                    string
	BackgroundIsToken                                  bool
	BackgroundSet                                      bool
	BorderWidth                                        Value
	BorderR, BorderG, BorderB, BorderA                 uint8
	BorderToken                                        string
	BorderIsToken                                      bool
	BorderColorSet                                     bool
	BorderPattern                                      uint8
	BorderSides                                        uint8
	RadiusTopLeft, RadiusTopRight                      Value
	RadiusBottomRight, RadiusBottomLeft                Value
	ShadowsHash                                        uint64
	Opacity                                            float32
	OpacitySet                                         bool
	Visibility                                         uint8
	TextR, TextG, TextB, TextA                         uint8
	TextToken                                          string
	TextIsToken                                        bool
	TextSet                                            bool
	ComponentToken                                     string
	StatesHash                                         uint64
	IconColorHash                                      uint64
	ImagePresentationHash                              uint64
}

// ContentProperties are intrinsic content inputs. A content change may alter
// both measurement and display data.
type ContentProperties struct {
	Text                       string
	FontFamilies               string
	FontSize, LineHeight       Value
	FontWeight                 uint16
	FontSlant, TextAlign, Wrap uint8
	MaxLines                   int
	VectorHash                 uint64
	Password, PasswordToggle   bool
}

// SemanticsProperties are callback/input semantics that need neither layout
// nor paint when changed in isolation.
type SemanticsProperties struct {
	Disabled  bool
	ReadOnly  bool
	HasPress  bool
	HasChange bool
	HasSubmit bool
	HasToggle bool
	HasLoad   bool
	HasError  bool
	Checked   bool
	HasSelect bool
	HasTabs   bool
	HasMenu   bool
	Pointer   uint8
}

// ResourceProperties identify description-side resource inputs. The core
// slice currently has no image/font constructors, but the class is explicit.
type ResourceProperties struct{ Identity uint64 }

// ScrollProperties are split from layout so controlled offset changes do not
// force measurement or placement.
type ScrollProperties struct {
	Enabled                       bool
	Axis, Scrollbar               uint8
	Controlled, HasScrollCallback bool
	OffsetX, OffsetY              float32
	InitialSet                    bool
	InitialX, InitialY            float32
}

// SliderProperties are normalized controlled values and interaction inputs.
type SliderProperties struct {
	Enabled, HasChange bool
	Value, Min, Max    float32
	Step               float32
}

// ProgressProperties contains the normalized determinate value. It is split
// from layout inputs so Value-only changes regenerate display data only.
type ProgressProperties struct {
	Value float32
}

// TabsProperties contains controlled selection inputs that affect display but
// not the intrinsic size of the always-visible label set.
type TabsProperties struct {
	Value string
}

// MenuProperties contains optional controlled selection input that affects
// display but not the intrinsic size of the always-visible item set.
type MenuProperties struct {
	Value string
}

// OverlayProperties contains typed portal inputs. Geometry is derived from
// final layout, so these values affect display and semantics but not flow.
type OverlayProperties struct {
	Kind          uint8
	Open          bool
	Placement     uint8
	Offset        Value
	DelayNS       int64
	Disabled      bool
	HasOpenChange bool
}

// Properties is the exact, comparable typed input set used by reconciliation.
type Properties struct {
	Layout    LayoutProperties
	Paint     PaintProperties
	Content   ContentProperties
	Semantics SemanticsProperties
	Resource  ResourceProperties
	Scroll    ScrollProperties
	Slider    SliderProperties
	Progress  ProgressProperties
	Tabs      TabsProperties
	Menu      MenuProperties
	Overlay   OverlayProperties
}

// Description is the immutable reconciliation input.
type Description struct {
	Kind       Kind
	Key        Key
	Properties Properties
	Children   []Description
}

// StateSlotKind identifies a typed retained local-state slot.
type StateSlotKind uint8

const (
	StateBool StateSlotKind = iota + 1
	StateInt
	StateFloat
	StateText
)

// StateSlot is a small typed state cell; only the field selected by Kind is
// meaningful. It avoids reflection and interface-valued state.
type StateSlot struct {
	Kind  StateSlotKind
	Bool  bool
	Int   int64
	Float float64
	Text  string
}

// State contains framework-retained interaction state and typed slots.
type State struct {
	Focused, FocusVisible         bool
	Hovered, Pressed              bool
	Checked                       bool
	ScrollX, ScrollY              float32
	ScrollbarHover, ScrollbarDrag uint8
	SelectionStart, SelectionEnd  int
	PasswordVisible               bool
	PasswordToggleHover           bool
	PasswordTogglePressed         bool
	SelectOpen                    bool
	SelectActive                  int
	SelectActiveKey               string
	SelectValue                   string
	SelectValueValid              bool
	SelectScroll                  float32
	TabsActive                    int
	TabsActiveKey                 string
	TabsHover                     int
	TabsPressed                   int
	TabsValue                     string
	TabsInitialized               bool
	MenuActive                    int
	MenuActiveKey                 string
	MenuHover                     int
	MenuPressed                   int
	MenuItemsHash                 uint64
	MenuInitialized               bool
	SliderDragging                bool
	TooltipOpen                   bool
	TooltipHover                  bool
	TooltipDueNS                  int64
	Slots                         []StateSlot
}

// Resource is owned by an instance and released on the UI thread when that
// instance is replaced, removed, or the tree is unmounted.
type Resource interface{ Release() }

// Node is a retained instance. ID and State survive an identity match.
type Node struct {
	ID         uint64
	Kind       Kind
	Key        Key
	Parent     *Node
	Children   []*Node
	Properties Properties
	State      State
	Dirty      Dirty
	resources  []Resource
}

// AddResource transfers ownership of resource to the instance.
func (n *Node) AddResource(resource Resource) {
	if n != nil && resource != nil {
		n.resources = append(n.resources, resource)
	}
}

// ResourceCount returns the number of live instance resources.
func (n *Node) ResourceCount() int {
	if n == nil {
		return 0
	}
	return len(n.resources)
}

// Changes summarizes work required after a successful update.
type Changes struct {
	Dirty                  Dirty
	Added, Reused, Removed int
}

// Has reports whether a dirty class is present.
func (c Changes) Has(flag Dirty) bool { return c.Dirty&flag != 0 }

// Tree holds the last successfully committed retained tree.
type Tree struct {
	root   *Node
	nextID uint64
}

// Preview validates and reconciles into a disposable candidate without
// changing retained state, IDs, resources, or dirty flags. Pipeline stages
// use it to reject layout failures before Update commits the candidate.
func (t *Tree) Preview(description Description) (Changes, *Node, error) {
	if descriptionMatches(t.root, description) {
		return Changes{Dirty: DirtyBuild, Reused: countNodes(t.root)}, t.root, nil
	}
	if err := validate(description, "root"); err != nil {
		return Changes{}, nil, err
	}
	nextID := t.nextID
	prepared := prepare(t.root, description, nil, &nextID)
	prepared.node.Dirty |= DirtyBuild
	propagateLayout(prepared.node)
	changes := collectChanges(prepared.node)
	changes.Added = prepared.added
	changes.Reused = len(prepared.reusedNodes)
	changes.Removed = prepared.removed
	return changes, prepared.node, nil
}

// Root returns the last committed tree.
func (t *Tree) Root() *Node { return t.root }

// Commit validates and reconciles a description transactionally.
func (t *Tree) Commit(description Description) error {
	_, err := t.Update(description)
	return err
}

// Update validates and reconciles a description transactionally. Failure
// leaves the old tree, state, resources, IDs, and dirty flags intact.
func (t *Tree) Update(description Description) (Changes, error) {
	if descriptionMatches(t.root, description) {
		t.root.Dirty |= DirtyBuild
		return Changes{Dirty: DirtyBuild, Reused: countNodes(t.root)}, nil
	}
	if err := validate(description, "root"); err != nil {
		return Changes{}, err
	}
	nextID := t.nextID
	prepared := prepare(t.root, description, nil, &nextID)
	prepared.node.Dirty |= DirtyBuild
	propagateLayout(prepared.node)

	// Transfer resource ownership only after the whole new tree exists.
	for _, reused := range prepared.reusedNodes {
		reused.resources = nil
	}
	releaseSubtree(t.root)

	t.root = prepared.node
	t.nextID = nextID
	changes := collectChanges(t.root)
	changes.Added = prepared.added
	changes.Reused = len(prepared.reusedNodes)
	changes.Removed = prepared.removed
	return changes, nil
}

func descriptionMatches(node *Node, description Description) bool {
	if node == nil || node.Kind != description.Kind || node.Key != description.Key || node.Properties != description.Properties || len(node.Children) != len(description.Children) {
		return false
	}
	for index := range node.Children {
		if !descriptionMatches(node.Children[index], description.Children[index]) {
			return false
		}
	}
	return true
}

// ClearDirty acknowledges all completed pipeline stages.
func (t *Tree) ClearDirty() { clearDirty(t.root) }

// Unmount releases descendants before parents and clears the tree.
func (t *Tree) Unmount() {
	releaseSubtree(t.root)
	t.root = nil
}

func validate(description Description, path string) error {
	if description.Kind == 0 {
		return fmt.Errorf("tree: %s has invalid node kind", path)
	}
	keys := make(map[Key]int, len(description.Children))
	for index, child := range description.Children {
		if child.Key != "" {
			if first, exists := keys[child.Key]; exists {
				return fmt.Errorf("tree: %s has duplicate sibling key %q at children %d and %d", path, child.Key, first, index)
			}
			keys[child.Key] = index
		}
		if err := validate(child, fmt.Sprintf("%s child %d", path, index)); err != nil {
			return err
		}
	}
	return nil
}

type preparedNode struct {
	node        *Node
	reusedNodes []*Node
	added       int
	removed     int
}

func prepare(previous *Node, description Description, parent *Node, nextID *uint64) preparedNode {
	matched := previous != nil && previous.Kind == description.Kind && previous.Key == description.Key
	node := &Node{Kind: description.Kind, Key: description.Key, Parent: parent, Properties: description.Properties}
	result := preparedNode{node: node}
	if matched {
		node.ID = previous.ID
		node.State = cloneState(previous.State)
		node.Dirty = previous.Dirty
		node.resources = append([]Resource(nil), previous.resources...)
		result.reusedNodes = append(result.reusedNodes, previous)
		node.Dirty |= propertyDirty(previous.Properties, description.Properties)
	} else {
		*nextID = *nextID + 1
		node.ID = *nextID
		node.Dirty = dirtyNew
		result.added++
		if description.Properties.Scroll.Enabled && description.Properties.Scroll.InitialSet {
			node.State.ScrollX = description.Properties.Scroll.InitialX
			node.State.ScrollY = description.Properties.Scroll.InitialY
		}
		if previous != nil {
			result.removed += countNodes(previous)
		}
	}
	// Checked is controlled description state. Retained interaction state must
	// never outlive or diverge from the application-supplied value.
	node.State.Checked = description.Properties.Semantics.Checked
	if description.Properties.Scroll.Enabled && description.Properties.Scroll.Controlled {
		node.State.ScrollX = description.Properties.Scroll.OffsetX
		node.State.ScrollY = description.Properties.Scroll.OffsetY
	}

	var previousChildren []*Node
	if matched {
		previousChildren = previous.Children
	}
	keyed := make(map[Key]*Node, len(previousChildren))
	for _, child := range previousChildren {
		if child.Key != "" {
			keyed[child.Key] = child
		}
	}

	used := make(map[*Node]struct{}, len(previousChildren))
	node.Children = make([]*Node, len(description.Children))
	structureChanged := len(previousChildren) != len(description.Children)
	for index, childDescription := range description.Children {
		var candidate *Node
		if childDescription.Key != "" {
			candidate = keyed[childDescription.Key]
			if candidate != nil && candidate.Kind != childDescription.Kind {
				candidate = nil
			}
		} else if index < len(previousChildren) {
			indexed := previousChildren[index]
			if indexed.Key == "" && indexed.Kind == childDescription.Kind {
				candidate = indexed
			}
		}
		child := prepare(candidate, childDescription, node, nextID)
		node.Children[index] = child.node
		result.reusedNodes = append(result.reusedNodes, child.reusedNodes...)
		result.added += child.added
		result.removed += child.removed
		if candidate != nil {
			used[candidate] = struct{}{}
			if index >= len(previousChildren) || previousChildren[index] != candidate {
				structureChanged = true
			}
		} else {
			structureChanged = true
		}
	}
	if matched {
		for _, oldChild := range previousChildren {
			if _, ok := used[oldChild]; !ok {
				result.removed += countNodes(oldChild)
			}
		}
	}
	if structureChanged {
		node.Dirty |= DirtyMeasure | DirtyLayout | DirtyDisplay | DirtyPaint | DirtySemantics
	}
	return result
}

func propertyDirty(old, next Properties) Dirty {
	dirty := Dirty(0)
	oldGeometry, nextGeometry := old.Layout, next.Layout
	oldGeometry.ZIndex, nextGeometry.ZIndex = 0, 0
	oldGeometry.Overflow, nextGeometry.Overflow = 0, 0
	if oldGeometry != nextGeometry {
		dirty |= DirtyStyle | DirtyMeasure | DirtyLayout | DirtyDisplay | DirtyPaint
	}
	if old.Layout.ZIndex != next.Layout.ZIndex || old.Layout.Overflow != next.Layout.Overflow {
		dirty |= DirtyStyle | DirtyDisplay | DirtyPaint | DirtySemantics
	}
	if old.Paint != next.Paint {
		dirty |= DirtyStyle | DirtyDisplay | DirtyPaint
	}
	if old.Content != next.Content {
		dirty |= DirtyMeasure | DirtyLayout | DirtyDisplay | DirtyPaint
	}
	if old.Semantics != next.Semantics {
		dirty |= DirtySemantics
		if old.Semantics.Disabled != next.Semantics.Disabled || old.Semantics.Checked != next.Semantics.Checked {
			dirty |= DirtyStyle | DirtyDisplay | DirtyPaint
		}
	}
	if old.Resource != next.Resource {
		dirty |= DirtyResource | DirtyMeasure | DirtyLayout | DirtyDisplay | DirtyPaint
	}
	if old.Scroll.Enabled != next.Scroll.Enabled || old.Scroll.Axis != next.Scroll.Axis {
		dirty |= DirtyMeasure | DirtyLayout | DirtyDisplay | DirtyPaint | DirtySemantics
	}
	if old.Scroll.Scrollbar != next.Scroll.Scrollbar || old.Scroll.Controlled != next.Scroll.Controlled || old.Scroll.OffsetX != next.Scroll.OffsetX || old.Scroll.OffsetY != next.Scroll.OffsetY {
		dirty |= DirtyDisplay | DirtyPaint | DirtySemantics
	}
	if old.Scroll.HasScrollCallback != next.Scroll.HasScrollCallback {
		dirty |= DirtySemantics
	}
	if old.Slider.HasChange != next.Slider.HasChange || old.Slider.Step != next.Slider.Step {
		dirty |= DirtySemantics
	}
	oldSlider, nextSlider := old.Slider, next.Slider
	oldSlider.HasChange, nextSlider.HasChange = false, false
	oldSlider.Step, nextSlider.Step = 0, 0
	if oldSlider != nextSlider {
		dirty |= DirtySemantics | DirtyDisplay | DirtyPaint
	}
	if old.Progress != next.Progress {
		dirty |= DirtyDisplay | DirtyPaint
	}
	if old.Tabs != next.Tabs {
		dirty |= DirtyDisplay | DirtyPaint
	}
	if old.Menu != next.Menu {
		dirty |= DirtyDisplay | DirtyPaint
	}
	if old.Overlay != next.Overlay {
		dirty |= DirtyDisplay | DirtyPaint | DirtySemantics
	}
	return dirty
}

func cloneState(state State) State {
	state.Slots = append([]StateSlot(nil), state.Slots...)
	return state
}

func propagateLayout(node *Node) Dirty {
	if node == nil {
		return 0
	}
	for _, child := range node.Children {
		childDirty := propagateLayout(child)
		if childDirty&(DirtyMeasure|DirtyLayout) != 0 {
			node.Dirty |= DirtyMeasure | DirtyLayout | DirtyDisplay | DirtyPaint
		}
	}
	return node.Dirty
}

func collectChanges(node *Node) Changes {
	var changes Changes
	var visit func(*Node)
	visit = func(current *Node) {
		if current == nil {
			return
		}
		changes.Dirty |= current.Dirty
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	return changes
}

func clearDirty(node *Node) {
	if node == nil {
		return
	}
	node.Dirty = 0
	for _, child := range node.Children {
		clearDirty(child)
	}
}

func releaseSubtree(node *Node) {
	if node == nil {
		return
	}
	for _, child := range node.Children {
		releaseSubtree(child)
	}
	for _, resource := range node.resources {
		resource.Release()
	}
	node.resources = nil
}

func countNodes(node *Node) int {
	if node == nil {
		return 0
	}
	count := 1
	for _, child := range node.Children {
		count += countNodes(child)
	}
	return count
}
