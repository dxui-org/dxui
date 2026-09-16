package dxui

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/dxui-org/dxui/internal/backend"
	internalinput "github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	internaltext "github.com/dxui-org/dxui/internal/text"
	"github.com/dxui-org/dxui/internal/tree"
)

type inputAction struct {
	press                                     func()
	toggle                                    func(bool)
	slider                                    *SliderProps
	sliderModel                               sliderModel
	selectRadio                               func()
	checked                                   bool
	submit                                    func()
	change                                    func(string)
	selectionChange                           func(TextRange)
	value                                     string
	selection                                 Option[TextRange]
	placeholder                               string
	password, passwordVisible, passwordToggle bool
	multiline, readOnly, disabled             bool
	wrap                                      TextWrap
	editor                                    bool
	node                                      viewProps
	geometry, content                         layout.Rect
	scroll                                    *ScrollProps
	viewport                                  layout.Rect
	extent                                    layout.Size
	parentScroll                              uint64
	depth                                     int
	paintOrder                                int
	selectp                                   *SelectProps
	tabs                                      *TabsProps
	tabItems                                  []paint.Rect
	menu                                      *MenuProps
	menuItems                                 []paint.Rect
	input                                     *InputProps
	selectAnchor                              layout.Rect
	selectOverlay                             selectOverlayGeometry
	popover                                   *PopoverProps
	tooltip                                   *TooltipProps
	overlayAnchor, overlayPopup               layout.Rect
	overlayLayer                              int
	tooltipFocus                              uint64
	overlayOwner                              uint64
}

type selectPointerCapture struct {
	id      uint64
	index   int
	consume bool
}

type scrollbarDrag struct {
	id                                uint64
	axis                              uint8
	startPointer, startOffset         float32
	trackLength, thumbLength, maximum float32
}

type sliderDrag struct{ id uint64 }

type tabsPointerCapture struct {
	id    uint64
	index int
}

type menuPointerCapture struct {
	id    uint64
	index int
	value string
}

type retainedEditor struct {
	state            internalinput.State
	value            string
	pendingValue     string
	pendingSelection internalinput.Range
	pending          bool
}

func buildInteractionSnapshot(view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, textEngine *internaltext.Engine) ([]internalinput.Node, map[uint64]inputAction, error) {
	orders := make(map[uint64]int)
	nextOrder := 0
	assignDocumentOrder(view, instance, orders, &nextOrder)
	nodes := make([]internalinput.Node, 0, nextOrder)
	actions := make(map[uint64]inputAction)
	if err := appendInteractionSnapshot(&nodes, actions, view, instance, geometry, theme, textEngine, orders, 1, paint.Rect{}, false, true, 0, 0, 0, 0, "root"); err != nil {
		return nil, nil, err
	}
	overlays := make([]selectOverlay, 0, 1)
	for id, action := range actions {
		if action.selectp == nil {
			continue
		}
		candidate := findInstance(instance, id)
		if candidate != nil && candidate.State.SelectOpen {
			if err := collectSelectOverlays(&overlays, view, instance, geometry, geometry.Rect, theme, 0, 0); err != nil {
				return nil, nil, err
			}
			break
		}
	}
	for _, overlay := range overlays {
		action := actions[overlay.instance.ID]
		action.selectAnchor, action.selectOverlay = overlay.anchor, overlay.geometry
		actions[overlay.instance.ID] = action
		nodes = append(nodes, internalinput.Node{ID: overlay.instance.ID, Bounds: inputRect(overlay.geometry.Popup), Order: orders[overlay.instance.ID], Interactive: true})
	}
	componentOverlays, err := collectComponentOverlays(view, instance, geometry, theme)
	if err != nil {
		return nil, nil, err
	}
	for overlayIndex, overlay := range componentOverlays {
		action := actions[overlay.hostID]
		action.overlayAnchor = overlay.anchor
		action.overlayPopup = layout.Rect(overlay.placed.Popup)
		action.overlayLayer = overlayIndex + 1
		actions[overlay.hostID] = action
		if !overlay.interactive || overlay.placed.Popup.Width <= 0 || overlay.placed.Popup.Height <= 0 {
			continue
		}
		before := make(map[uint64]struct{}, len(actions))
		for id := range actions {
			before[id] = struct{}{}
		}
		tx := overlay.placed.Popup.X - overlay.geometry.Rect.X
		ty := overlay.placed.Popup.Y - overlay.geometry.Rect.Y
		if err := appendInteractionSnapshot(&nodes, actions, overlay.view, overlay.instance, overlay.geometry, theme, textEngine, orders, 1,
			overlay.placed.Popup, true, true, tx, ty, 0, 0, "overlay"); err != nil {
			return nil, nil, err
		}
		for id, contentAction := range actions {
			if _, existed := before[id]; !existed {
				contentAction.overlayOwner = overlay.hostID
				actions[id] = contentAction
			}
		}
	}
	return nodes, actions, nil
}

func assignDocumentOrder(view View, instance *tree.Node, orders map[uint64]int, next *int) {
	if view.node == nil || instance == nil {
		return
	}
	if focusableViewKind(view.node.kind) {
		orders[instance.ID] = *next
		*next++
	}
	indices := make([]int, 0, len(view.node.children))
	switch view.node.kind {
	case viewPopover:
		indices = append(indices, 1)
	case viewTooltip:
		indices = append(indices, 0)
	default:
		for index := range view.node.children {
			indices = append(indices, index)
		}
	}
	for _, index := range indices {
		if index < len(instance.Children) {
			assignDocumentOrder(view.node.children[index], instance.Children[index], orders, next)
		}
	}
}

func focusableViewKind(kind viewKind) bool {
	return kind == viewButton || kind == viewInput || kind == viewTextarea || kind == viewToggleSwitch || kind == viewSlider || kind == viewCheckbox || kind == viewRadio || kind == viewScroll || kind == viewVirtualList || kind == viewSelect || kind == viewTabs || kind == viewMenu || kind == viewPopover
}

func appendInteractionSnapshot(nodes *[]internalinput.Node, actions map[uint64]inputAction, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, textEngine *internaltext.Engine, orders map[uint64]int, parentOpacity float32, inheritedClip paint.Rect, inheritedClipSet, pointerEnabled bool, translateX, translateY float32, parentScroll uint64, depth int, path string) error {
	if view.node == nil || instance == nil || geometry == nil {
		return fmt.Errorf("%s has incomplete tree geometry", path)
	}
	rawGeometry := geometry
	worldGeometry := *geometry
	worldGeometry.Rect = translatedRect(geometry.Rect, translateX, translateY)
	worldGeometry.Content = translatedRect(geometry.Content, translateX, translateY)
	worldGeometry.Viewport = translatedRect(geometry.Viewport, translateX, translateY)
	geometry = &worldGeometry
	worldRect := paintRect(geometry.Rect)
	if inheritedClipSet && len(view.node.children) == 0 && rectsDisjoint(worldRect, inheritedClip) {
		return nil
	}
	props, err := nodeProps(view)
	if err != nil {
		return err
	}
	visual, err := computeVisual(props, instance, view.node.kind, theme, view.node.groupChild)
	if err != nil {
		return err
	}
	if visual.visibility == Hidden {
		return nil
	}
	rect := worldRect
	if (view.node.kind == viewPopover || view.node.kind == viewTooltip) && len(rawGeometry.Children) >= 1 {
		rect = paintRect(translatedRect(rawGeometry.Children[0].Rect, translateX, translateY))
	}
	clip, clipSet := inheritedClip, inheritedClipSet
	if props.Style.Overflow == OverflowClip || isScrollView(view) {
		clipRect := rect
		if isScrollView(view) {
			clipRect = paintRect(geometry.Viewport)
		}
		if clipSet {
			clip = paint.Intersect(clip, clipRect)
		} else {
			clip, clipSet = clipRect, true
		}
	}
	if clipSet && len(view.node.children) == 0 && rectsDisjoint(rect, clip) {
		return nil
	}
	worldOpacity := parentOpacity * visual.opacity
	childPointerEnabled := pointerEnabled && props.Pointer != PointerNone
	disabled := instance.Properties.Semantics.Disabled
	focusable := focusableViewKind(view.node.kind)
	interactive := childPointerEnabled && worldOpacity > 0 && (focusable || isScrollView(view))
	node := internalinput.Node{
		ID: instance.ID, Bounds: inputRect(rect), Order: orders[instance.ID],
		Interactive: interactive, Focusable: focusable, Disabled: disabled,
		Enter: view.node.kind == viewButton || view.node.kind == viewSelect || view.node.kind == viewTabs || view.node.kind == viewMenu || view.node.kind == viewPopover || view.node.kind == viewInput && view.node.input.OnSubmit != nil,
		Space: view.node.kind == viewButton || view.node.kind == viewToggleSwitch || view.node.kind == viewCheckbox || view.node.kind == viewRadio || view.node.kind == viewSelect || view.node.kind == viewTabs || view.node.kind == viewMenu || view.node.kind == viewPopover,
	}
	if clipSet {
		node.Clip, node.ClipSet = inputRect(clip), true
	}
	paintOrder := len(*nodes)
	if focusable {
		*nodes = append(*nodes, node)
	}
	switch view.node.kind {
	case viewButton:
		actions[instance.ID] = inputAction{press: view.node.button.OnPress}
	case viewInput:
		p := view.node.input
		actions[instance.ID] = inputAction{editor: true, input: p, submit: p.OnSubmit, change: p.OnChange,
			selectionChange: p.OnSelectionChange,
			value:           p.Value, selection: p.Selection, placeholder: p.Placeholder,
			password: p.Password, passwordVisible: p.Password && p.ShowPasswordToggle && instance.State.PasswordVisible,
			passwordToggle: p.Password && p.ShowPasswordToggle,
			readOnly:       p.ReadOnly, disabled: p.Disabled,
			node: p.common(), geometry: geometry.Rect, content: geometry.Content}
	case viewTextarea:
		p := view.node.textarea
		actions[instance.ID] = inputAction{editor: true, change: p.OnChange,
			selectionChange: p.OnSelectionChange,
			value:           p.Value, selection: p.Selection, placeholder: p.Placeholder,
			multiline: true, readOnly: p.ReadOnly, disabled: p.Disabled, wrap: p.Wrap,
			node: p.common(), geometry: geometry.Rect, content: geometry.Content}
	case viewToggleSwitch:
		actions[instance.ID] = inputAction{toggle: view.node.toggle.OnChange, checked: view.node.toggle.Checked}
	case viewSlider:
		actions[instance.ID] = inputAction{slider: view.node.slider, sliderModel: normalizeSlider(*view.node.slider), geometry: geometry.Rect}
	case viewCheckbox:
		actions[instance.ID] = inputAction{toggle: view.node.checkbox.OnChange, checked: view.node.checkbox.Checked}
	case viewRadio:
		if !view.node.radio.Selected {
			actions[instance.ID] = inputAction{selectRadio: view.node.radio.OnSelect}
		}
	case viewScroll:
		actions[instance.ID] = inputAction{scroll: view.node.scroll, viewport: geometry.Viewport, extent: geometry.ContentExtent, parentScroll: parentScroll, depth: depth, paintOrder: paintOrder}
		parentScroll = instance.ID
		depth++
	case viewVirtualList:
		actions[instance.ID] = inputAction{scroll: view.node.scroll, viewport: geometry.Viewport, extent: geometry.ContentExtent, parentScroll: parentScroll, depth: depth, paintOrder: paintOrder}
		parentScroll = instance.ID
		depth++
	case viewSelect:
		actions[instance.ID] = inputAction{selectp: view.node.selectp, selectAnchor: geometry.Rect}
	case viewTabs:
		metrics, metricsErr := resolveTabsMetrics(theme)
		if metricsErr != nil {
			return metricsErr
		}
		labels, labelsErr := measureTabLabels(textEngine, theme, *view.node.tabs)
		if labelsErr != nil {
			return labelsErr
		}
		boxes := tabsGeometry(paintRect(geometry.Content), labels, metrics)
		items := make([]paint.Rect, len(boxes))
		for index := range boxes {
			items[index] = boxes[index].item
		}
		actions[instance.ID] = inputAction{tabs: view.node.tabs, tabItems: items, geometry: geometry.Rect}
	case viewMenu:
		metrics, metricsErr := resolveMenuMetrics(theme)
		if metricsErr != nil {
			return metricsErr
		}
		labels, labelsErr := measureMenuLabels(textEngine, theme, *view.node.menu)
		if labelsErr != nil {
			return labelsErr
		}
		boxes := menuGeometry(paintRect(geometry.Content), labels, metrics, view.node.menu.Orientation)
		items := make([]paint.Rect, len(boxes))
		for index := range boxes {
			items[index] = boxes[index].item
		}
		actions[instance.ID] = inputAction{menu: view.node.menu, menuItems: items, geometry: geometry.Rect}
	case viewPopover:
		actions[instance.ID] = inputAction{popover: view.node.popover, overlayAnchor: layout.Rect(rect)}
	case viewTooltip:
		actions[instance.ID] = inputAction{tooltip: view.node.tooltip, overlayAnchor: layout.Rect(rect), tooltipFocus: firstFocusableDescendant(view.node.children[0], instance.Children[0], orders)}
	}
	order := paintChildOrder(view.node.children)
	childCount := len(view.node.children)
	if view.node.kind == viewPopover {
		childCount = 0
		order = nil
	} else if view.node.kind == viewTooltip {
		childCount = min(childCount, 1)
		order = nil
	}
	for position := 0; position < childCount; position++ {
		index := position
		if order != nil {
			index = order[position]
		}
		if index >= len(instance.Children) || index >= len(geometry.Children) {
			return fmt.Errorf("%s child geometry mismatch", path)
		}
		childTranslateX, childTranslateY := translateX, translateY
		if isScrollView(view) {
			childTranslateX -= instance.State.ScrollX
			childTranslateY -= instance.State.ScrollY
		}
		if err := appendInteractionSnapshot(nodes, actions, view.node.children[index], instance.Children[index], rawGeometry.Children[index], theme, textEngine, orders, worldOpacity, clip, clipSet, childPointerEnabled, childTranslateX, childTranslateY, parentScroll, depth, path); err != nil {
			return fmt.Errorf("%s child %d: %w", path, index, err)
		}
	}
	return nil
}

func firstFocusableDescendant(view View, instance *tree.Node, orders map[uint64]int) uint64 {
	if view.node == nil || instance == nil {
		return 0
	}
	bestID, bestOrder := uint64(0), int(^uint(0)>>1)
	var visit func(View, *tree.Node)
	visit = func(current View, retained *tree.Node) {
		if current.node == nil || retained == nil {
			return
		}
		if order, ok := orders[retained.ID]; ok && order < bestOrder {
			bestID, bestOrder = retained.ID, order
		}
		for index, child := range current.node.children {
			if index < len(retained.Children) {
				visit(child, retained.Children[index])
			}
		}
	}
	visit(view, instance)
	return bestID
}

func inputRect(value paint.Rect) internalinput.Rect {
	return internalinput.Rect{X: value.X, Y: value.Y, Width: value.Width, Height: value.Height}
}

type scrollProposal struct {
	action inputAction
	offset Point
}

func (a *App) handleScrollEvent(event platform.Event) (dirty, handled, displayed bool, err error) {
	if event.Kind != platform.EventMouseWheel && event.Kind != platform.EventMouseMove && event.Kind != platform.EventMouseDown && event.Kind != platform.EventMouseUp && event.Kind != platform.EventKeyDown && event.Kind != platform.EventWindowMouseLeave && event.Kind != platform.EventWindowFocusLost {
		return false, false, false, nil
	}
	a.mu.Lock()
	actions := make(map[uint64]inputAction, len(a.inputActions))
	focusedAction := a.inputActions[a.interaction.Focused()]
	for id, action := range a.inputActions {
		if action.scroll != nil {
			actions[id] = action
		}
	}
	theme := a.theme
	a.mu.Unlock()
	if event.Kind == platform.EventMouseWheel && focusedAction.editor && focusedAction.multiline && rectContains(paintRect(focusedAction.content), event.Pointer.X, event.Pointer.Y) {
		return false, false, false, nil
	}
	if len(actions) == 0 {
		a.scrollDrag = scrollbarDrag{}
		return false, false, false, nil
	}
	root := a.retained.Root()
	if event.Kind == platform.EventKeyDown && !event.Key.Repeat {
		id := a.interaction.Focused()
		action, live := actions[id]
		node := findInstance(root, id)
		if !live || node == nil {
			return false, false, false, nil
		}
		current := Point{X: node.State.ScrollX, Y: node.State.ScrollY}
		proposed := current
		switch event.Key.Key {
		case platform.KeyUp:
			if scrollAxisEnabled(action.scroll.Axis, false) {
				proposed.Y -= scrollWheelUnit
			} else {
				proposed.X -= scrollWheelUnit
			}
		case platform.KeyDown:
			if scrollAxisEnabled(action.scroll.Axis, false) {
				proposed.Y += scrollWheelUnit
			} else {
				proposed.X += scrollWheelUnit
			}
		case platform.KeyLeft:
			proposed.X -= scrollWheelUnit
		case platform.KeyRight:
			proposed.X += scrollWheelUnit
		case platform.KeyHome:
			if event.Key.Modifiers.Shift {
				proposed.X = 0
			} else {
				proposed.Y = 0
			}
		case platform.KeyEnd:
			if event.Key.Modifiers.Shift {
				proposed.X = action.extent.Width
			} else {
				proposed.Y = action.extent.Height
			}
		default:
			return false, false, false, nil
		}
		proposed = clampScrollOffset(action.scroll.Axis, proposed, action.viewport, action.extent)
		if proposed == current {
			return false, true, false, nil
		}
		changed, callback := a.applyScrollProposal(node, action, proposed)
		if callback {
			displayed, err = a.emitScrollProposals([]scrollProposal{{action: action, offset: proposed}})
		}
		return changed, true, displayed, err
	}
	if event.Kind == platform.EventWindowMouseLeave || event.Kind == platform.EventWindowFocusLost {
		dirty = clearScrollbarStates(root)
		a.scrollDrag = scrollbarDrag{}
		return dirty, false, false, nil
	}
	if event.Kind == platform.EventMouseMove && a.scrollDrag.id != 0 {
		drag := a.scrollDrag
		node := findInstance(root, drag.id)
		action, live := actions[drag.id]
		if node == nil || !live || drag.maximum <= 0 || drag.trackLength <= drag.thumbLength {
			a.scrollDrag = scrollbarDrag{}
			return clearScrollbarStates(root), true, false, nil
		}
		pointer := event.Pointer.X
		start := node.State.ScrollX
		if drag.axis == scrollbarAxisVertical {
			pointer, start = event.Pointer.Y, node.State.ScrollY
		}
		_ = start
		proposedAxis := drag.startOffset + (pointer-drag.startPointer)*drag.maximum/(drag.trackLength-drag.thumbLength)
		offset := Point{X: node.State.ScrollX, Y: node.State.ScrollY}
		if drag.axis == scrollbarAxisHorizontal {
			offset.X = proposedAxis
		} else {
			offset.Y = proposedAxis
		}
		offset = clampScrollOffset(action.scroll.Axis, offset, action.viewport, action.extent)
		changed, callback := a.applyScrollProposal(node, action, offset)
		dirty = changed
		if callback {
			displayed, err = a.emitScrollProposals([]scrollProposal{{action: action, offset: offset}})
		}
		return dirty, true, displayed, err
	}
	if event.Kind == platform.EventMouseUp && a.scrollDrag.id != 0 {
		node := findInstance(root, a.scrollDrag.id)
		if node != nil && node.State.ScrollbarDrag != 0 {
			node.State.ScrollbarDrag = 0
			dirty = true
		}
		a.scrollDrag = scrollbarDrag{}
		return dirty, true, false, nil
	}
	if event.Kind == platform.EventMouseMove || event.Kind == platform.EventMouseDown {
		axis, id, bar, hitErr := a.scrollbarAt(actions, root, event.Pointer.X, event.Pointer.Y, theme)
		if hitErr != nil {
			return false, false, false, hitErr
		}
		dirty = setScrollbarHover(root, id, axis)
		if event.Kind == platform.EventMouseDown && id != 0 && !bar.Disabled {
			node := findInstance(root, id)
			pointer, offset, trackLength, thumbLength := event.Pointer.X, node.State.ScrollX, bar.Track.Width, bar.Thumb.Width
			if axis == scrollbarAxisVertical {
				pointer, offset, trackLength, thumbLength = event.Pointer.Y, node.State.ScrollY, bar.Track.Height, bar.Thumb.Height
			}
			a.scrollDrag = scrollbarDrag{id: id, axis: axis, startPointer: pointer, startOffset: offset, trackLength: trackLength, thumbLength: thumbLength, maximum: bar.Maximum}
			node.State.ScrollbarDrag = axis
			dirty = true
			return dirty, true, false, nil
		}
		if event.Kind == platform.EventMouseMove {
			return dirty, false, false, nil
		}
	}
	if event.Kind != platform.EventMouseWheel || !finitePoint(Point{X: event.Pointer.X, Y: event.Pointer.Y}) {
		return dirty, false, false, nil
	}
	id := deepestScrollAt(actions, event.Pointer.X, event.Pointer.Y)
	if id == 0 {
		return false, false, false, nil
	}
	dx, dy := -event.Pointer.WheelX*scrollWheelUnit, -event.Pointer.WheelY*scrollWheelUnit
	if a.modifiers.Shift && dx == 0 {
		dx, dy = dy, 0
	}
	proposals := make([]scrollProposal, 0, 2)
	for id != 0 && (dx != 0 || dy != 0) {
		action, live := actions[id]
		if !live {
			break
		}
		node := findInstance(root, id)
		if node == nil {
			break
		}
		usedX, usedY := dx, dy
		if action.scroll.Axis == ScrollHorizontal && usedX == 0 {
			usedX, usedY = usedY, 0
		}
		if !scrollAxisEnabled(action.scroll.Axis, true) {
			usedX = 0
		}
		if !scrollAxisEnabled(action.scroll.Axis, false) {
			usedY = 0
		}
		current := Point{X: node.State.ScrollX, Y: node.State.ScrollY}
		proposed := clampScrollOffset(action.scroll.Axis, Point{X: current.X + usedX, Y: current.Y + usedY}, action.viewport, action.extent)
		movedX, movedY := proposed.X-current.X, proposed.Y-current.Y
		if movedX != 0 || movedY != 0 {
			changed, callback := a.applyScrollProposal(node, action, proposed)
			dirty = dirty || changed
			if callback {
				proposals = append(proposals, scrollProposal{action: action, offset: proposed})
			}
			handled = true
		}
		if action.scroll.Axis == ScrollHorizontal && dx == 0 {
			dy -= movedX
		} else {
			dx -= movedX
			dy -= movedY
		}
		id = action.parentScroll
	}
	if len(proposals) != 0 {
		displayed, err = a.emitScrollProposals(proposals)
	}
	return dirty, handled, displayed, err
}

func (a *App) applyScrollProposal(node *tree.Node, action inputAction, offset Point) (changed, callback bool) {
	_, controlled := action.scroll.Offset.get()
	if controlled {
		return false, action.scroll.OnScroll != nil
	}
	changed = node.State.ScrollX != offset.X || node.State.ScrollY != offset.Y
	if changed {
		node.State.ScrollX, node.State.ScrollY = offset.X, offset.Y
	}
	return changed, changed && action.scroll.OnScroll != nil
}

func (a *App) emitScrollProposals(proposals []scrollProposal) (bool, error) {
	for _, proposal := range proposals {
		if err := callNoArg("scroll callback", func() { proposal.action.scroll.OnScroll(proposal.offset) }); err != nil {
			return false, err
		}
		a.markInputBoundary()
	}
	return a.runQueuedWork()
}

func (a *App) scrollbarAt(actions map[uint64]inputAction, root *tree.Node, x, y float32, theme resolvedTheme) (uint8, uint64, scrollbarGeometry, error) {
	thickness, err := theme.metric(TokenMetric(MetricComponentScrollThickness))
	if err != nil {
		return 0, 0, scrollbarGeometry{}, err
	}
	minimum, err := theme.metric(TokenMetric(MetricComponentScrollMinThumb))
	if err != nil {
		return 0, 0, scrollbarGeometry{}, err
	}
	inset, err := theme.metric(TokenMetric(MetricComponentScrollInset))
	if err != nil {
		return 0, 0, scrollbarGeometry{}, err
	}
	bestDepth, bestOrder := -1, -1
	var bestID uint64
	var best scrollbarGeometry
	for id, action := range actions {
		if action.depth < bestDepth || action.depth == bestDepth && action.paintOrder < bestOrder || !scrollContains(actions, id, x, y) {
			continue
		}
		node := findInstance(root, id)
		if node == nil {
			continue
		}
		for _, bar := range scrollbarsFor(action.viewport, action.extent, Point{X: node.State.ScrollX, Y: node.State.ScrollY}, action.scroll.Axis, action.scroll.Scrollbar, thickness, minimum, inset) {
			if rectContains(bar.Thumb, x, y) && (action.depth > bestDepth || action.depth == bestDepth && action.paintOrder >= bestOrder) {
				bestDepth, bestOrder, bestID, best = action.depth, action.paintOrder, id, bar
			}
		}
	}
	return best.Axis, bestID, best, nil
}

func deepestScrollAt(actions map[uint64]inputAction, x, y float32) uint64 {
	bestDepth, bestOrder, bestID := -1, -1, uint64(0)
	for id, action := range actions {
		if (action.depth > bestDepth || action.depth == bestDepth && action.paintOrder > bestOrder) && scrollContains(actions, id, x, y) {
			bestDepth, bestOrder, bestID = action.depth, action.paintOrder, id
		}
	}
	return bestID
}

func scrollContains(actions map[uint64]inputAction, id uint64, x, y float32) bool {
	for id != 0 {
		action, ok := actions[id]
		if !ok || !rectContains(paintRect(action.viewport), x, y) {
			return false
		}
		id = action.parentScroll
	}
	return true
}

func findInstance(root *tree.Node, id uint64) *tree.Node {
	if root == nil {
		return nil
	}
	if root.ID == id {
		return root
	}
	for _, child := range root.Children {
		if found := findInstance(child, id); found != nil {
			return found
		}
	}
	return nil
}

func clearScrollbarStates(root *tree.Node) bool { return setScrollbarHover(root, 0, 0) }

func setScrollbarHover(root *tree.Node, id uint64, axis uint8) bool {
	changed := false
	var visit func(*tree.Node)
	visit = func(node *tree.Node) {
		if node == nil {
			return
		}
		next := uint8(0)
		if node.ID == id {
			next = axis
		}
		if node.State.ScrollbarHover != next {
			node.State.ScrollbarHover, changed = next, true
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(root)
	return changed
}

func applyInteractionState(root *tree.Node, controller *internalinput.Controller) bool {
	changed := false
	type descendantState struct{ hover, pressed, focused, focusVisible bool }
	var visit func(*tree.Node) descendantState
	visit = func(node *tree.Node) descendantState {
		if node == nil {
			return descendantState{}
		}
		hovered, pressed, focused, focusVisible := controller.State(node.ID)
		children := descendantState{}
		for _, child := range node.Children {
			state := visit(child)
			children.hover = children.hover || state.hover
			children.pressed = children.pressed || state.pressed
			children.focused = children.focused || state.focused
			children.focusVisible = children.focusVisible || state.focusVisible
		}
		if node.Kind == tree.Kind(viewInputGroup) {
			hovered, focused, focusVisible = children.hover, children.focused, children.focusVisible
		}
		if node.State.Hovered != hovered || node.State.Pressed != pressed || node.State.Focused != focused || node.State.FocusVisible != focusVisible {
			node.State.Hovered, node.State.Pressed = hovered, pressed
			node.State.Focused, node.State.FocusVisible = focused, focusVisible
			changed = true
		}
		return descendantState{hovered || children.hover, pressed || children.pressed, focused || children.focused, focusVisible || children.focusVisible}
	}
	visit(root)
	return changed
}

func (a *App) refreshInteraction(view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme) (bool, error) {
	textEngine, err := a.textEngine()
	if err != nil {
		return false, err
	}
	snapshot, actions, err := buildInteractionSnapshot(view, instance, geometry, theme, textEngine)
	if err != nil {
		return false, err
	}
	a.restoreClosedPopoverFocus(actions)
	result := a.interaction.Reconcile(snapshot)
	a.reconcileScrollDrag(actions)
	a.reconcileSliderDrag(actions)
	a.reconcileSelectCapture(actions)
	a.reconcileTabsCapture(actions)
	a.reconcileMenuCapture(actions)
	a.reconcilePasswordToggleState(actions)
	tooltipLifecycleChanged := a.syncTooltipLifecycle(actions)
	changed := applyInteractionState(instance, &a.interaction) || result.Changed || tooltipLifecycleChanged
	a.mu.Lock()
	a.inputActions = actions
	a.mu.Unlock()
	editorsChanged := a.syncEditors(actions)
	return changed || editorsChanged, nil
}

func (a *App) restoreClosedPopoverFocus(actions map[uint64]inputAction) {
	focused := a.interaction.Focused()
	old := a.inputActions[focused]
	if old.overlayOwner == 0 {
		return
	}
	owner, live := actions[old.overlayOwner]
	if live && owner.popover != nil && !owner.popover.Open {
		a.interaction.Focus(old.overlayOwner, true)
	}
}

func (a *App) reconcileSliderDrag(actions map[uint64]inputAction) {
	if a.sliderDrag.id == 0 {
		return
	}
	action, live := actions[a.sliderDrag.id]
	node := findInstance(a.retained.Root(), a.sliderDrag.id)
	if !live || action.slider == nil || node == nil || node.Properties.Semantics.Disabled {
		if node != nil {
			node.State.SliderDragging = false
		}
		a.sliderDrag = sliderDrag{}
	}
}

func (a *App) reconcileScrollDrag(actions map[uint64]inputAction) {
	if a.scrollDrag.id == 0 {
		return
	}
	action, live := actions[a.scrollDrag.id]
	if !live || action.scroll == nil || findInstance(a.retained.Root(), a.scrollDrag.id) == nil {
		a.scrollDrag = scrollbarDrag{}
	}
}

func (a *App) handleInteraction(event platform.Event) (bool, error) {
	if event.Kind == platform.EventKeyDown || event.Kind == platform.EventKeyUp {
		a.modifiers = event.Key.Modifiers
	}
	overlayDirty, overlayHandled, overlayDisplayed, overlayErr := a.handlePopoverOverlayEvent(event)
	if overlayErr != nil {
		return overlayDirty, overlayErr
	}
	if overlayHandled {
		if overlayDirty && !overlayDisplayed {
			if err := a.redisplayCurrent(); err != nil {
				return overlayDirty, err
			}
		}
		return overlayDirty || overlayDisplayed, nil
	}
	selectDirty, selectHandled, selectDisplayed, selectErr := false, false, false, error(nil)
	if !a.hasOpenPopoverOverlay() {
		selectDirty, selectHandled, selectDisplayed, selectErr = a.handleSelectEvent(event)
	}
	if selectErr != nil {
		return selectDirty, selectErr
	}
	if selectHandled {
		if selectDirty && !selectDisplayed {
			if err := a.redisplayCurrent(); err != nil {
				return selectDirty, err
			}
		}
		return selectDirty || selectDisplayed, nil
	}
	oldFocus := a.interaction.Focused()
	result := a.interaction.Handle(event)
	tabsDirty, tabsHandled, tabsDisplayed, tabsErr := a.handleTabsEvent(event, result)
	if tabsErr != nil {
		return tabsDirty, tabsErr
	}
	menuDirty, menuHandled, menuDisplayed, menuErr := a.handleMenuEvent(event, result)
	if menuErr != nil {
		return menuDirty, menuErr
	}
	sliderDirty, sliderHandled, sliderDisplayed, sliderErr := a.handleSliderEvent(event, result)
	if sliderErr != nil {
		return sliderDirty, sliderErr
	}
	var scrollDirty, scrollHandled, scrollDisplayed bool
	var scrollErr error
	if (!sliderHandled && !tabsHandled && !menuHandled) || event.Kind == platform.EventWindowFocusLost || event.Kind == platform.EventWindowMouseLeave {
		scrollDirty, scrollHandled, scrollDisplayed, scrollErr = a.handleScrollEvent(event)
	}
	if scrollErr != nil {
		return scrollDirty, scrollErr
	}
	newFocus := a.interaction.Focused()
	if event.Kind == platform.EventWindowFocusLost {
		a.windowFocused = false
	}
	if event.Kind == platform.EventWindowFocusGained {
		a.windowFocused = true
	}
	stateChanged := applyInteractionState(a.retained.Root(), &a.interaction)
	tooltipDirty := a.syncTooltipTriggers(event)
	dirty := selectDirty || selectDisplayed || result.Changed || stateChanged || tooltipDirty || tabsDirty || tabsDisplayed || menuDirty || menuDisplayed || sliderDirty || sliderDisplayed || scrollDirty || scrollDisplayed
	if oldFocus != newFocus || event.Kind == platform.EventWindowFocusLost || event.Kind == platform.EventWindowFocusGained {
		focusDirty, err := a.syncTextInputFocus(oldFocus, newFocus)
		if err != nil {
			return dirty, err
		}
		dirty = dirty || focusDirty
	}
	passwordDirty, passwordHandled, err := a.handlePasswordToggleEvent(event, result)
	if err != nil {
		return dirty || passwordDirty, err
	}
	dirty = dirty || passwordDirty
	editHandled := false
	if !passwordHandled {
		editDirty, handled, editErr := a.handleEditorEvent(event, result, newFocus)
		if editErr != nil {
			return dirty || editDirty, editErr
		}
		dirty, editHandled = dirty || editDirty, handled
	}
	if dirty && !scrollDisplayed && !sliderDisplayed && !tabsDisplayed && !menuDisplayed {
		if err := a.redisplayCurrent(); err != nil {
			return false, err
		}
		if err := a.updateTextInputArea(); err != nil {
			return dirty, err
		}
	}
	if scrollHandled {
		a.mu.Lock()
		currentView := a.view
		a.mu.Unlock()
		if scrollDirty && containsVirtualList(currentView) {
			changes, commitErr := a.commitView(currentView)
			if commitErr != nil {
				return dirty, commitErr
			}
			dirty = dirty || changes.Has(tree.DirtyPaint)
		}
		return dirty, nil
	}
	if tabsHandled {
		return dirty, nil
	}
	if menuHandled {
		return dirty, nil
	}
	if sliderHandled {
		return dirty, nil
	}
	if editHandled {
		return dirty, nil
	}
	if passwordHandled {
		return dirty, nil
	}
	if shortcutHandled, shortcutDisplayed, shortcutErr := a.handleShortcut(event); shortcutHandled {
		return dirty || shortcutDisplayed, shortcutErr
	}
	if result.Activate == 0 {
		return dirty, nil
	}
	a.mu.Lock()
	action, ok := a.inputActions[result.Activate]
	a.mu.Unlock()
	if !ok {
		return dirty, nil
	}
	if action.press != nil {
		if err := callNoArg("button press callback", action.press); err != nil {
			return dirty, err
		}
	} else if action.submit != nil {
		if err := callNoArg("input submit callback", action.submit); err != nil {
			return dirty, err
		}
	} else if action.toggle != nil {
		if err := callNoArg("toggle change callback", func() { action.toggle(!action.checked) }); err != nil {
			return dirty, err
		}
	} else if action.selectRadio != nil {
		if err := callNoArg("radio select callback", action.selectRadio); err != nil {
			return dirty, err
		}
	} else if action.selectp != nil {
		node := findInstance(a.retained.Root(), result.Activate)
		if node == nil || action.selectp.Disabled || len(action.selectp.Options) == 0 {
			return dirty, nil
		}
		openSelect(a.retained.Root(), result.Activate, action.selectp, node)
		if err := a.redisplayCurrent(); err != nil {
			return true, err
		}
		return true, nil
	} else if action.popover != nil {
		changed, _, displayed, err := a.emitPopoverChange(result.Activate, action.popover, !action.popover.Open)
		return dirty || changed || displayed, err
	} else {
		return dirty, nil
	}
	a.mu.Lock()
	a.invalidated = true
	a.mu.Unlock()
	commitDirty, err := a.runQueuedWork()
	return dirty || commitDirty, err
}

func (a *App) handleShortcut(event platform.Event) (bool, bool, error) {
	if event.Kind != platform.EventKeyDown {
		return false, false, nil
	}
	a.mu.Lock()
	focused := a.inputActions[a.interaction.Focused()]
	shortcuts := a.options.Shortcuts
	a.mu.Unlock()
	// Text entry, composition, and the activation keys owned by a focused
	// semantic control always win over application-window shortcuts.
	if focused.editor || (event.Key.Key == platform.KeyEnter || event.Key.Key == platform.KeySpace) &&
		(focused.press != nil || focused.toggle != nil || focused.selectRadio != nil || focused.selectp != nil || focused.tabs != nil || focused.menu != nil || focused.popover != nil) {
		return false, false, nil
	}
	for _, shortcut := range shortcuts {
		if shortcutPlatformKey(shortcut.Key) != event.Key.Key || !shortcutModifiersMatch(shortcut.Modifiers, event.Key.Modifiers) {
			continue
		}
		if event.Key.Repeat && !shortcut.Repeat {
			return true, false, nil
		}
		if shortcut.OnPress == nil {
			return true, false, nil
		}
		if err := callNoArg("shortcut callback", shortcut.OnPress); err != nil {
			return true, false, err
		}
		a.mu.Lock()
		a.invalidated = true
		a.mu.Unlock()
		displayed, err := a.runQueuedWork()
		return true, displayed, err
	}
	return false, false, nil
}

func shortcutModifiersMatch(want ShortcutModifiers, got platform.Modifiers) bool {
	if want.Primary {
		if !got.Primary {
			return false
		}
		// Primary substitutes for the platform's physical primary bit so callers
		// do not have to request both Primary and Control/Command. The other raw
		// modifier remains available for cross-platform chords such as Cmd+Ctrl.
		if runtime.GOOS == "darwin" {
			got.Super = false
		} else {
			got.Control = false
		}
	}
	return want.Shift == got.Shift && want.Control == got.Control && want.Alt == got.Alt && want.Super == got.Super && want.Primary == got.Primary
}

func shortcutPlatformKey(key ShortcutKey) platform.Key {
	switch key {
	case KeyEnter:
		return platform.KeyEnter
	case KeyBackspace:
		return platform.KeyBackspace
	case Key0:
		return platform.Key0
	case Key1:
		return platform.Key1
	case Key2:
		return platform.Key2
	case Key3:
		return platform.Key3
	case Key4:
		return platform.Key4
	case Key5:
		return platform.Key5
	case Key6:
		return platform.Key6
	case Key7:
		return platform.Key7
	case Key8:
		return platform.Key8
	case Key9:
		return platform.Key9
	case KeyPlus:
		return platform.KeyPlus
	case KeyMinus:
		return platform.KeyMinus
	case KeyMultiply:
		return platform.KeyMultiply
	case KeyDivide:
		return platform.KeyDivide
	case KeyDecimal:
		return platform.KeyDecimal
	case KeyEquals:
		return platform.KeyEquals
	default:
		return platform.KeyOther
	}
}

func (a *App) syncEditors(actions map[uint64]inputAction) bool {
	changed := false
	if action, live := actions[a.dragEditor]; !live || !action.editor || action.disabled || a.interaction.Focused() != a.dragEditor {
		a.dragEditor = 0
	}
	for id, action := range actions {
		if !action.editor {
			continue
		}
		editor := a.editors[id]
		if editor == nil {
			selection := internalinput.Range{Start: len([]rune(action.value)), End: len([]rune(action.value))}
			if controlled, ok := action.selection.get(); ok {
				selection = toInternalRange(controlled)
			}
			editor = &retainedEditor{value: action.value}
			editor.state.SetControlledValue(action.value, selection)
			a.editors[id] = editor
			continue
		}
		selection := editor.state.Selection
		accepted := false
		if editor.pending {
			accepted = action.value == editor.pendingValue
			if accepted {
				selection = editor.pendingSelection
			}
			editor.state.AcceptControlled(action.value, selection, accepted)
			editor.pending = false
		} else if action.value != editor.value {
			editor.state.CancelComposition()
			editor.state.SetControlledValue(action.value)
		}
		if controlled, ok := action.selection.get(); ok {
			editor.state.SetControlledValue(action.value, toInternalRange(controlled))
		}
		editor.value = action.value
		if action.disabled || action.readOnly || action.change == nil || a.interaction.Focused() != id {
			changed = editor.state.CancelComposition() || changed
		}
	}
	for id := range a.editors {
		if _, live := actions[id]; !live {
			changed = a.editors[id].state.CancelComposition() || changed
			delete(a.editors, id)
			if a.focusedEditor == id {
				a.focusedEditor = 0
			}
		}
	}
	return changed
}

func (a *App) syncTextInputFocus(oldFocus, newFocus uint64) (bool, error) {
	if oldFocus != newFocus || !a.windowFocused {
		a.dragEditor = 0
	}
	oldAction, oldText := a.inputActions[oldFocus]
	newAction, newText := a.inputActions[newFocus]
	oldText = oldText && oldAction.editor
	newText = newText && newAction.editor
	if oldText && (oldFocus != newFocus || !a.windowFocused) {
		if editor := a.editors[oldFocus]; editor != nil {
			editor.state.CancelComposition()
		}
	}
	native := a.textInputBackend()
	if (!newText || !a.windowFocused || newAction.disabled) && a.nativeTextActive {
		if native != nil {
			if err := native.StopTextInput(); err != nil {
				return true, fmt.Errorf("dxui: stop text input: %w", err)
			}
		}
		a.nativeTextActive = false
		a.focusedEditor = 0
	}
	if newText && a.windowFocused && !newAction.disabled {
		if !a.nativeTextActive && native != nil {
			if err := native.StartTextInput(); err != nil {
				return true, fmt.Errorf("dxui: start text input: %w", err)
			}
		}
		a.nativeTextActive = true
		a.focusedEditor = newFocus
		a.caretVisible = true
		a.caretChangedAt = a.clock.Now()
	}
	return oldText != newText || oldFocus != newFocus, nil
}

func (a *App) handleEditorEvent(event platform.Event, result internalinput.Result, focus uint64) (bool, bool, error) {
	action, ok := a.inputActions[focus]
	editor := a.editors[focus]
	if event.Kind == platform.EventMouseDown {
		if targetAction, targetOK := a.inputActions[result.Target]; targetOK {
			if targetEditor := a.editors[result.Target]; targetEditor != nil {
				if !targetAction.multiline && event.Pointer.Clicks == 2 {
					targetEditor.state.SelectAll(targetAction.value)
					a.dragEditor = 0
					if err := a.selectionChanged(targetAction, targetEditor); err != nil {
						return true, true, err
					}
					return true, true, nil
				}
				layoutValue, err := a.layoutEditor(targetAction)
				if err != nil {
					return false, true, err
				}
				index := editorIndexAt(layoutValue, event.Pointer.X-targetAction.content.X+targetEditor.state.ScrollX, event.Pointer.Y-targetAction.content.Y+targetEditor.state.ScrollY)
				if a.modifiers.Shift {
					targetEditor.state.Selection.End = index
				} else {
					targetEditor.state.Selection = internalinput.Range{Start: index, End: index}
				}
				a.dragEditor = result.Target
				if err := a.selectionChanged(targetAction, targetEditor); err != nil {
					return true, true, err
				}
				return true, true, nil
			}
		}
	}
	if event.Kind == platform.EventMouseMove && a.dragEditor != 0 {
		dragAction, live := a.inputActions[a.dragEditor]
		dragEditor := a.editors[a.dragEditor]
		if live && dragEditor != nil {
			layoutValue, err := a.layoutEditor(dragAction)
			if err != nil {
				return false, true, err
			}
			dragEditor.state.Selection.End = editorIndexAt(layoutValue, event.Pointer.X-dragAction.content.X+dragEditor.state.ScrollX, event.Pointer.Y-dragAction.content.Y+dragEditor.state.ScrollY)
			if err := a.selectionChanged(dragAction, dragEditor); err != nil {
				return true, true, err
			}
			return true, true, nil
		}
	}
	if event.Kind == platform.EventMouseUp {
		a.dragEditor = 0
	}
	if !ok || editor == nil {
		return false, false, nil
	}
	if event.Kind == platform.EventMouseWheel && action.multiline && result.Target == focus {
		layoutValue, err := a.layoutEditor(action)
		if err != nil {
			return false, true, err
		}
		editor.state.ScrollY = max(0, editor.state.ScrollY-event.Pointer.WheelY*24)
		clampEditorScroll(action, editor, layoutValue)
		return true, true, nil
	}
	switch event.Kind {
	case platform.EventTextEditing:
		if action.disabled || action.readOnly || action.change == nil || !a.windowFocused {
			return editor.state.CancelComposition(), true, nil
		}
		selection := internalinput.Range{Start: event.Text.EditingStart, End: event.Text.EditingStart + event.Text.EditingLength}
		editor.state.SetComposition(event.Text.Text, selection)
		return true, true, nil
	case platform.EventTextInput:
		if action.disabled || action.readOnly || action.change == nil || !a.windowFocused {
			return false, true, nil
		}
		inserted := internalinput.Normalize(event.Text.Text)
		if !action.multiline {
			inserted = strings.NewReplacer("\r", "", "\n", "").Replace(inserted)
		}
		change, err := editor.state.Insert(action.value, inserted, internalinput.ReasonIMECommit)
		if err != nil {
			return false, true, err
		}
		editor.state.Commit(change)
		return a.emitEditorChange(focus, action, editor, change)
	case platform.EventKeyDown:
		return a.handleEditorKey(focus, action, editor, event.Key)
	}
	return false, false, nil
}

func (a *App) handleEditorKey(id uint64, action inputAction, editor *retainedEditor, key platform.KeyEvent) (bool, bool, error) {
	if key.Modifiers.Primary {
		switch key.Key {
		case platform.KeyA:
			editor.state.SelectAll(action.value)
			return true, true, a.selectionChanged(action, editor)
		case platform.KeyC:
			if action.password {
				return false, true, nil
			}
			if text, native := editor.state.Selected(action.value), a.textInputBackend(); text != "" && native != nil {
				if err := native.SetClipboardText(text); err != nil {
					return false, true, fmt.Errorf("dxui: copy: %w", err)
				}
			}
			return false, true, nil
		case platform.KeyX:
			if action.password || action.readOnly || action.change == nil {
				return false, true, nil
			}
			selected := editor.state.Selected(action.value)
			if selected == "" {
				return false, true, nil
			}
			if native := a.textInputBackend(); native != nil {
				if err := native.SetClipboardText(selected); err != nil {
					return false, true, fmt.Errorf("dxui: cut: %w", err)
				}
			}
			change, _ := editor.state.Insert(action.value, "", internalinput.ReasonCut)
			editor.state.Commit(change)
			return a.emitEditorChange(id, action, editor, change)
		case platform.KeyV:
			native := a.textInputBackend()
			if action.readOnly || action.change == nil || native == nil {
				return false, true, nil
			}
			value, err := native.ClipboardText()
			if err != nil {
				return false, true, fmt.Errorf("dxui: paste: %w", err)
			}
			if !action.multiline {
				value = strings.NewReplacer("\r", "", "\n", "").Replace(value)
			}
			change, err := editor.state.Insert(action.value, value, internalinput.ReasonPaste)
			if err != nil {
				return false, true, err
			}
			editor.state.Commit(change)
			return a.emitEditorChange(id, action, editor, change)
		case platform.KeyZ:
			if action.readOnly || action.change == nil {
				return false, true, nil
			}
			var change internalinput.Change
			var ok bool
			if key.Modifiers.Shift {
				change, ok = editor.state.Redo(action.value)
			} else {
				change, ok = editor.state.Undo(action.value)
			}
			if !ok {
				return false, true, nil
			}
			editor.state.Commit(change)
			return a.emitEditorChange(id, action, editor, change)
		case platform.KeyY:
			if action.readOnly || action.change == nil {
				return false, true, nil
			}
			change, ok := editor.state.Redo(action.value)
			if !ok {
				return false, true, nil
			}
			editor.state.Commit(change)
			return a.emitEditorChange(id, action, editor, change)
		}
	}
	changed := false
	switch key.Key {
	case platform.KeyLeft:
		changed = editor.state.MoveHorizontal(action.value, -1, key.Modifiers.Shift)
	case platform.KeyRight:
		changed = editor.state.MoveHorizontal(action.value, 1, key.Modifiers.Shift)
	case platform.KeyHome:
		changed = editor.state.MoveLineBoundary(action.value, false, key.Modifiers.Shift)
	case platform.KeyEnd:
		changed = editor.state.MoveLineBoundary(action.value, true, key.Modifiers.Shift)
	case platform.KeyUp, platform.KeyDown:
		layoutValue, err := a.layoutEditor(action)
		if err != nil {
			return false, true, err
		}
		x, _, _ := editorCaret(layoutValue, editor.state.Selection.End)
		if !editor.state.PreferredXSet {
			editor.state.PreferredX, editor.state.PreferredXSet = x, true
		}
		delta := -1
		if key.Key == platform.KeyDown {
			delta = 1
		}
		index := editorVertical(layoutValue, editor.state.Selection.End, delta, editor.state.PreferredX)
		if key.Modifiers.Shift {
			editor.state.Selection.End = index
		} else {
			editor.state.Selection = internalinput.Range{Start: index, End: index}
		}
		changed = true
	case platform.KeyBackspace:
		if action.readOnly || action.change == nil {
			return false, true, nil
		}
		change, ok := editor.state.DeleteBackward(action.value)
		if !ok {
			return false, true, nil
		}
		editor.state.Commit(change)
		return a.emitEditorChange(id, action, editor, change)
	case platform.KeyDelete:
		if action.readOnly || action.change == nil {
			return false, true, nil
		}
		change, ok := editor.state.DeleteForward(action.value)
		if !ok {
			return false, true, nil
		}
		editor.state.Commit(change)
		return a.emitEditorChange(id, action, editor, change)
	case platform.KeyEnter:
		if !action.multiline || key.Repeat || action.readOnly || action.change == nil {
			return false, false, nil
		}
		change, err := editor.state.Insert(action.value, "\n", internalinput.ReasonTyping)
		if err != nil {
			return false, true, err
		}
		editor.state.Commit(change)
		return a.emitEditorChange(id, action, editor, change)
	case platform.KeyEscape:
		if editor.state.CancelComposition() {
			if native := a.textInputBackend(); native != nil {
				_ = native.ClearComposition()
			}
			return true, true, nil
		}
	}
	if changed {
		if err := a.selectionChanged(action, editor); err != nil {
			return true, true, err
		}
		return true, true, nil
	}
	return false, false, nil
}

func (a *App) selectionChanged(action inputAction, editor *retainedEditor) error {
	editor.state.SetControlledValue(action.value)
	if err := a.ensureEditorCaretVisible(action, editor); err != nil {
		return err
	}
	if action.selectionChange != nil {
		selection := publicRange(editor.state.Selection)
		if err := callNoArg("input selection callback", func() { action.selectionChange(selection) }); err != nil {
			return err
		}
		a.markInputBoundary()
		if _, err := a.runQueuedWork(); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) emitEditorChange(id uint64, action inputAction, editor *retainedEditor, change internalinput.Change) (bool, bool, error) {
	if action.change == nil {
		return false, true, nil
	}
	editor.pending, editor.pendingValue, editor.pendingSelection = true, change.Value, change.Selection
	if err := callNoArg("input change callback", func() { action.change(change.Value) }); err != nil {
		return true, true, err
	}
	if action.selectionChange != nil {
		selection := publicRange(change.Selection)
		if err := callNoArg("input selection callback", func() { action.selectionChange(selection) }); err != nil {
			return true, true, err
		}
	}
	a.markInputBoundary()
	commitDirty, err := a.runQueuedWork()
	_ = commitDirty
	return true, true, err
}

func (a *App) markInputBoundary() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.invalidated = true
}

func (a *App) nextCaretDeadline(now time.Time) *time.Time {
	action, ok := a.inputActions[a.focusedEditor]
	if !ok || !a.windowFocused || action.disabled || !a.nativeTextActive {
		return nil
	}
	if a.caretChangedAt.IsZero() {
		a.caretChangedAt = now
	}
	next := a.caretChangedAt.Add(500 * time.Millisecond)
	return &next
}

func (a *App) nextDeadline(now time.Time) *time.Time {
	return earlierDeadline(a.nextCaretDeadline(now), a.nextTooltipDeadline())
}

func (a *App) handleCaretDeadline() (bool, error) {
	if a.nextCaretDeadline(a.clock.Now()) == nil {
		return false, nil
	}
	a.caretVisible = !a.caretVisible
	a.caretChangedAt = a.clock.Now()
	if err := a.redisplayCurrent(); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) handleDeadlines() (bool, error) {
	now := a.clock.Now()
	caret := a.nextCaretDeadline(now)
	tooltip := a.nextTooltipDeadline()
	due := earlierDeadline(caret, tooltip)
	if due == nil {
		return false, nil
	}
	dirty := a.handleTooltipDeadline(*due)
	if caret != nil && caret.Equal(*due) {
		a.caretVisible = !a.caretVisible
		a.caretChangedAt = now
		dirty = true
	}
	if dirty {
		if err := a.redisplayCurrent(); err != nil {
			return false, err
		}
	}
	return dirty, nil
}

func (a *App) updateTextInputArea() error {
	native := a.textInputBackend()
	if !a.nativeTextActive || native == nil {
		return nil
	}
	action, ok := a.inputActions[a.focusedEditor]
	editor := a.editors[a.focusedEditor]
	if !ok || editor == nil {
		return nil
	}
	layoutValue, err := a.layoutEditor(action)
	if err != nil {
		return err
	}
	x, y, h := editorCaret(layoutValue, editor.state.Selection.End)
	area := backend.TextInputArea{X: int32(math.Round(float64(action.content.X - editor.state.ScrollX))), Y: int32(math.Round(float64(action.content.Y + y - editor.state.ScrollY))), Width: int32(math.Ceil(float64(max(1, action.content.Width)))), Height: int32(math.Ceil(float64(max(1, h)))), Cursor: int32(math.Round(float64(x)))}
	if err := native.SetTextInputArea(area); err != nil {
		return fmt.Errorf("dxui: set text input area: %w", err)
	}
	return nil
}

func (a *App) textInputBackend() backend.TextInput {
	if runtime, ok := a.backend.(backend.TextInputRuntime); ok {
		return runtime.TextInput()
	}
	return nil
}

func (a *App) editorDisplaySnapshot() map[uint64]editorDisplayState {
	result := make(map[uint64]editorDisplayState, len(a.editors))
	for id, editor := range a.editors {
		result[id] = editorDisplayState{selection: editor.state.Selection, composition: editor.state.Composition,
			scrollX: editor.state.ScrollX, scrollY: editor.state.ScrollY,
			caretVisible: a.caretVisible && id == a.focusedEditor}
	}
	return result
}
