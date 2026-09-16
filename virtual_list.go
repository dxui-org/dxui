package dxui

import (
	"fmt"
	"math"

	"github.com/dxui-org/dxui/internal/tree"
)

const (
	// MaxVirtualListItems bounds key metadata and extent arithmetic.
	MaxVirtualListItems = 10_000_000
	// MaxVirtualListOverscan bounds work retained outside the viewport.
	MaxVirtualListOverscan = 256
)

type virtualListWindow struct{ First, Last int }

func virtualWindow(count int, rowHeight, viewportHeight, offset float32, overscan int) virtualListWindow {
	if count == 0 || viewportHeight <= 0 {
		return virtualListWindow{}
	}
	first := int(math.Floor(float64(offset / rowHeight)))
	last := int(math.Ceil(float64((offset + viewportHeight) / rowHeight)))
	first = max(0, first-overscan)
	last = min(count, last+overscan)
	return virtualListWindow{First: first, Last: last}
}

func virtualScrollProps(props *VirtualListProps) ScrollProps {
	return ScrollProps{Key: props.Key, Style: props.Style, Token: props.Token, States: props.States,
		Pointer: props.Pointer, Axis: ScrollVertical, InitialOffset: props.InitialOffset,
		Offset: props.Offset, Scrollbar: props.Scrollbar, OnScroll: props.OnScroll}
}

func isScrollView(view View) bool {
	return view.node != nil && (view.node.kind == viewScroll || view.node.kind == viewVirtualList)
}

func containsVirtualList(view View) bool {
	if view.node == nil {
		return false
	}
	if view.node.kind == viewVirtualList {
		return true
	}
	for _, child := range view.node.children {
		if containsVirtualList(child) {
			return true
		}
	}
	return false
}

func materializeVirtualLists(view View, previousView View, previous *tree.Node, fallbackHeight float32) (View, error) {
	return materializeVirtualNode(view, previousView, previous, fallbackHeight, "root")
}

func materializeVirtualNode(view View, previousView View, previous *tree.Node, fallbackHeight float32, path string) (View, error) {
	if view.node == nil {
		return view, nil
	}
	if view.node.kind == viewVirtualList {
		p := view.node.virtualList
		if p == nil {
			return View{}, fmt.Errorf("dxui: %s virtual list has missing properties", path)
		}
		if p.Count < 0 || p.Count > MaxVirtualListItems || !finite(p.RowHeight) || p.RowHeight <= 0 ||
			p.Overscan < 0 || p.Overscan > MaxVirtualListOverscan || p.ItemKey == nil || p.Build == nil {
			return View{}, fmt.Errorf("dxui: %s has invalid VirtualList count, row height, overscan, ItemKey, or Build", path)
		}
		extent64 := float64(p.Count) * float64(p.RowHeight)
		if math.IsInf(extent64, 0) || extent64 > math.MaxFloat32 {
			return View{}, fmt.Errorf("dxui: %s virtual list extent overflows", path)
		}
		if p.Style.Height.kind != lengthPixels {
			return View{}, fmt.Errorf("dxui: %s virtual list requires an explicit finite pixel Height", path)
		}
		viewportHeight := p.Style.Height.value
		var oldProps *VirtualListProps
		if previousView.node != nil && previousView.node.kind == viewVirtualList {
			oldProps = previousView.node.virtualList
		}
		keys := previousView.nodeKeys(p.Count, p.Version)
		keysChanged := keys == nil
		seen := make(map[string]int)
		if keysChanged {
			keys = make([]string, p.Count)
			seen = make(map[string]int, p.Count)
			for i := 0; i < p.Count; i++ {
				key, err := callVirtualKey(p.ItemKey, i)
				if err != nil {
					return View{}, fmt.Errorf("dxui: %s: %w", path, err)
				}
				if key == "" {
					return View{}, fmt.Errorf("dxui: %s item %d has an empty key", path, i)
				}
				if first, ok := seen[key]; ok {
					return View{}, fmt.Errorf("dxui: %s has duplicate item key %q at %d and %d", path, key, first, i)
				}
				seen[key], keys[i] = i, key
			}
		}
		offset := float32(0)
		if initial, set := p.InitialOffset.get(); set {
			offset = initial.Y
		}
		if previous != nil && previous.Kind == tree.Kind(viewVirtualList) {
			offset = previous.State.ScrollY
		}
		if keysChanged && oldProps != nil && previousView.node != nil && len(previousView.node.virtualKeys) != 0 && p.Offset.set == false {
			oldIndex := min(len(previousView.node.virtualKeys)-1, max(0, int(offset/oldProps.RowHeight)))
			anchor := previousView.node.virtualKeys[oldIndex]
			if nextIndex, ok := seen[anchor]; ok {
				offset = float32(nextIndex)*p.RowHeight + float32(math.Mod(float64(offset), float64(oldProps.RowHeight)))
			}
		}
		if controlled, set := p.Offset.get(); set {
			offset = controlled.Y
		}
		maxOffset := max(float32(0), float32(extent64)-viewportHeight)
		offset = min(max(float32(0), offset), maxOffset)
		window := virtualWindow(p.Count, p.RowHeight, viewportHeight, offset, p.Overscan)
		children := make([]View, 0, window.Last-window.First+2)
		var reusable map[string]View
		if !keysChanged && previousView.node != nil && len(previousView.node.children) == 1 && previousView.node.children[0].node != nil {
			reusable = make(map[string]View, len(previousView.node.children[0].node.children))
			for _, oldRow := range previousView.node.children[0].node.children {
				if oldRow.node != nil && oldRow.node.key != "" {
					reusable[oldRow.node.key] = oldRow
				}
			}
		}
		if top := float32(window.First) * p.RowHeight; top > 0 {
			children = append(children, Box(BoxProps{Style: Style{Height: Px(top), Shrink: Some(float32(0))}}))
		}
		for i := window.First; i < window.Last; i++ {
			if oldRow, ok := reusable[keys[i]]; ok {
				children = append(children, oldRow)
				continue
			}
			row, err := callVirtualBuild(p.Build, i)
			if err != nil {
				return View{}, fmt.Errorf("dxui: %s item %d: %w", path, i, err)
			}
			children = append(children, Box(BoxProps{Key: keys[i], Align: AlignStretch,
				Style: Style{Height: Px(p.RowHeight), Shrink: Some(float32(0)), Overflow: OverflowClip}}, row))
		}
		if bottom := float32(p.Count-window.Last) * p.RowHeight; bottom > 0 {
			children = append(children, Box(BoxProps{Style: Style{Height: Px(bottom), Shrink: Some(float32(0))}}))
		}
		content := Box(BoxProps{Align: AlignStretch, Style: Style{Width: Percent(100), Shrink: Some(float32(0))}}, children...)
		node := *view.node
		node.scroll = ptrScrollProps(virtualScrollProps(p))
		node.virtualKeys = keys
		node.children = []View{content}
		return View{node: &node}, nil
	}
	node := *view.node
	node.children = append([]View(nil), view.node.children...)
	for i, child := range node.children {
		var pv View
		var pn *tree.Node
		if previousView.node != nil && i < len(previousView.node.children) {
			pv = previousView.node.children[i]
		}
		if previous != nil && i < len(previous.Children) {
			pn = previous.Children[i]
		}
		materialized, err := materializeVirtualNode(child, pv, pn, fallbackHeight, fmt.Sprintf("%s child %d", path, i))
		if err != nil {
			return View{}, err
		}
		node.children[i] = materialized
	}
	return View{node: &node}, nil
}

func (view View) nodeKeys(count int, version uint64) []string {
	if view.node == nil || view.node.kind != viewVirtualList || view.node.virtualList == nil ||
		view.node.virtualList.Count != count || view.node.virtualList.Version != version || len(view.node.virtualKeys) != count {
		return nil
	}
	return view.node.virtualKeys
}

func ptrScrollProps(value ScrollProps) *ScrollProps { return &value }

func callVirtualKey(fn func(int) string, index int) (key string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("VirtualList ItemKey panic: %v", recovered)
		}
	}()
	return fn(index), nil
}

func callVirtualBuild(fn func(int) View, index int) (view View, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("VirtualList Build panic: %v", recovered)
		}
	}()
	view = fn(index)
	if view.node == nil {
		return View{}, fmt.Errorf("VirtualList Build returned an invalid View")
	}
	return view, nil
}
