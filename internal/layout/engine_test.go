package layout

import (
	"math"
	"math/rand"
	"testing"
)

const tolerance = float32(0.0001)

func pixels(value float32) Length { return Length{Kind: LengthPixels, Value: value} }

func percent(value float32) Length { return Length{Kind: LengthPercent, Value: value} }

func exact(width, height float32) Constraints {
	return Constraints{
		Width:  Limit{Min: width, Max: width, MaxSet: true, Definite: true},
		Height: Limit{Min: height, Max: height, MaxSet: true, Definite: true},
	}
}

func fixedNode(id uint64, width, height float32) *Node {
	return &Node{ID: id, Style: Style{Width: pixels(width), Height: pixels(height)}}
}

func assertNear(t *testing.T, got, want float32) {
	t.Helper()
	if float32(math.Abs(float64(got-want))) > tolerance {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func assertRect(t *testing.T, got, want Rect) {
	t.Helper()
	assertNear(t, got.X, want.X)
	assertNear(t, got.Y, want.Y)
	assertNear(t, got.Width, want.Width)
	assertNear(t, got.Height, want.Height)
}

func TestRowFixedGeometryPaddingMarginAndGap(t *testing.T) {
	root := &Node{
		ID: 1, Axis: AxisRow, Gap: 5,
		Style: Style{Padding: Edges{Top: 4, Right: 10, Bottom: 6, Left: 10}},
		Children: []*Node{
			{ID: 2, Style: Style{Width: pixels(20), Height: pixels(10), Margin: Edges{Left: 3, Right: 2}}},
			fixedNode(3, 30, 12),
		},
	}
	result, err := NewEngine(EngineOptions{}).Layout(root, exact(100, 40))
	if err != nil {
		t.Fatal(err)
	}
	assertRect(t, result.Content, Rect{X: 10, Y: 4, Width: 80, Height: 30})
	assertRect(t, result.Children[0].Rect, Rect{X: 13, Y: 4, Width: 20, Height: 10})
	assertRect(t, result.Children[1].Rect, Rect{X: 40, Y: 4, Width: 30, Height: 12})
}

func TestMainAxisJustificationMatrix(t *testing.T) {
	tests := []struct {
		name      string
		justify   Justify
		positions [2]float32
	}{
		{"start", JustifyStart, [2]float32{0, 15}},
		{"center", JustifyCenter, [2]float32{37.5, 52.5}},
		{"end", JustifyEnd, [2]float32{75, 90}},
		{"space-between", JustifySpaceBetween, [2]float32{0, 90}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := &Node{Axis: AxisRow, Gap: 5, Justify: test.justify, Children: []*Node{fixedNode(1, 10, 10), fixedNode(2, 10, 10)}}
			result, err := NewEngine(EngineOptions{}).Layout(root, exact(100, 20))
			if err != nil {
				t.Fatal(err)
			}
			assertNear(t, result.Children[0].Rect.X, test.positions[0])
			assertNear(t, result.Children[1].Rect.X, test.positions[1])
		})
	}

	one := &Node{Axis: AxisRow, Justify: JustifySpaceBetween, Children: []*Node{fixedNode(1, 10, 10)}}
	result, err := NewEngine(EngineOptions{}).Layout(one, exact(100, 20))
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.Children[0].Rect.X, 0)
}

func TestCrossAxisAlignmentAndAlignSelf(t *testing.T) {
	tests := []struct {
		align Align
		y     float32
		h     float32
	}{
		{AlignStart, 2, 10},
		{AlignCenter, 20, 10},
		{AlignEnd, 38, 10},
		{AlignStretch, 2, 46},
	}
	for _, test := range tests {
		root := &Node{Axis: AxisRow, Align: test.align, Children: []*Node{{Style: Style{Width: pixels(10), Margin: Edges{Top: 2, Bottom: 2}}, Measure: constantIntrinsic(8, 10)}}}
		result, err := NewEngine(EngineOptions{}).Layout(root, exact(100, 50))
		if err != nil {
			t.Fatal(err)
		}
		assertNear(t, result.Children[0].Rect.Y, test.y)
		assertNear(t, result.Children[0].Rect.Height, test.h)
	}

	root := &Node{Axis: AxisRow, Align: AlignEnd, Children: []*Node{{Style: Style{Width: pixels(10), Height: pixels(10), AlignSelf: OptionalAlign{Value: AlignCenter, Set: true}}}}}
	result, err := NewEngine(EngineOptions{}).Layout(root, exact(100, 50))
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.Children[0].Rect.Y, 20)
}

func TestGrowShrinkWeightsBasisAndFreezing(t *testing.T) {
	tests := []struct {
		name  string
		width float32
		items []*Node
		want  []float32
	}{
		{
			name: "grow multiple weights", width: 100,
			items: []*Node{
				{Style: Style{Width: pixels(10), Grow: 1}},
				{Style: Style{Width: pixels(10), Grow: 3}},
			},
			want: []float32{30, 70},
		},
		{
			name: "shrink scaled by basis", width: 60,
			items: []*Node{
				{Style: Style{Width: pixels(60)}},
				{Style: Style{Width: pixels(40)}},
			},
			want: []float32{36, 24},
		},
		{
			name: "basis overrides width", width: 80,
			items: []*Node{
				{Style: Style{Width: pixels(60), Basis: pixels(20), Grow: 1}},
				{Style: Style{Width: pixels(10), Basis: pixels(20), Grow: 1}},
			},
			want: []float32{40, 40},
		},
		{
			name: "basis auto uses specified main size", width: 80,
			items: []*Node{
				{Style: Style{Width: pixels(30), Grow: 1}},
				{Style: Style{Width: pixels(10), Grow: 1}},
			},
			want: []float32{50, 30},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := &Node{Axis: AxisRow, Children: test.items}
			result, err := NewEngine(EngineOptions{}).Layout(root, exact(test.width, 20))
			if err != nil {
				t.Fatal(err)
			}
			for index, want := range test.want {
				assertNear(t, result.Children[index].Rect.Width, want)
			}
		})
	}
}

func TestPercentageDefiniteAndIndefiniteParent(t *testing.T) {
	child := &Node{Style: Style{Width: percent(50), Height: pixels(10)}, Measure: constantIntrinsic(20, 10)}
	definite := &Node{Axis: AxisRow, Style: Style{Width: pixels(200), Height: pixels(20)}, Children: []*Node{child}}
	result, err := NewEngine(EngineOptions{}).Layout(definite, Constraints{})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.Children[0].Rect.Width, 100)

	indefinite := &Node{Axis: AxisRow, Children: []*Node{child}}
	result, err = NewEngine(EngineOptions{}).Layout(indefinite, Constraints{})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.Children[0].Rect.Width, 20)
}

func TestIntrinsicMeasurerReceivesExplicitContentWidth(t *testing.T) {
	var request IntrinsicRequest
	leaf := &Node{Style: Style{Width: pixels(80), Padding: Edges{Left: 5, Right: 5}}, Measure: IntrinsicMeasureFunc(func(value IntrinsicRequest) (IntrinsicSize, error) {
		request = value
		return IntrinsicSize{Preferred: Size{Width: 100, Height: 20}}, nil
	})}
	if _, err := NewEngine(EngineOptions{}).Layout(leaf, Constraints{}); err != nil {
		t.Fatal(err)
	}
	if !request.Width.MaxSet || !request.Width.Definite || request.Width.Max != 70 {
		t.Fatalf("intrinsic width request = %#v, want definite max 70", request.Width)
	}
}

func TestPercentageInsetsAreAutoForIndefiniteContainer(t *testing.T) {
	flow := fixedNode(1, 100, 20)
	absolute := &Node{Style: Style{
		Position: PositionAbsolute, Width: pixels(10), Height: pixels(10),
		Insets: Insets{Left: percent(50)},
	}}
	root := &Node{Axis: AxisRow, Children: []*Node{flow, absolute}}
	result, err := NewEngine(EngineOptions{}).Layout(root, Constraints{})
	if err != nil {
		t.Fatal(err)
	}
	// The root's 100-unit content-derived width is intentionally indefinite,
	// so 50% behaves as auto and uses the static content-box start.
	assertNear(t, result.Children[1].Rect.X, 0)
}

func TestNonShrinkableItemsOverflowWithoutChangingGap(t *testing.T) {
	root := &Node{Axis: AxisRow, Gap: 5, Children: []*Node{
		{Style: Style{Width: pixels(40), ShrinkSet: true}},
		{Style: Style{Width: pixels(40), ShrinkSet: true}},
	}}
	result, err := NewEngine(EngineOptions{}).Layout(root, exact(50, 20))
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.Children[0].Rect.Width, 40)
	assertNear(t, result.Children[1].Rect.X, 45)
	assertNear(t, result.Children[1].Rect.Width, 40)
}

func TestMinGreaterThanMaxRaisesMaxAndPaddingCanExceedSize(t *testing.T) {
	root := &Node{Axis: AxisRow, Style: Style{
		Width: pixels(10), Height: pixels(5), MinWidth: pixels(20), MaxWidth: pixels(8),
		Padding: Edges{Top: 4, Right: 15, Bottom: 4, Left: 15},
	}}
	result, err := NewEngine(EngineOptions{}).Layout(root, Constraints{})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.Rect.Width, 20)
	assertNear(t, result.Content.Width, 0)
	assertNear(t, result.Content.Height, 0)
}

func TestAbsoluteInsetCombinations(t *testing.T) {
	tests := []struct {
		name  string
		style Style
		want  Rect
	}{
		{
			name:  "start and size",
			style: Style{Position: PositionAbsolute, Width: pixels(20), Height: pixels(10), Insets: Insets{Left: pixels(5), Top: pixels(6)}},
			want:  Rect{X: 5, Y: 6, Width: 20, Height: 10},
		},
		{
			name:  "end and size",
			style: Style{Position: PositionAbsolute, Width: pixels(20), Height: pixels(10), Insets: Insets{Right: pixels(5), Bottom: pixels(6)}},
			want:  Rect{X: 75, Y: 34, Width: 20, Height: 10},
		},
		{
			name:  "both and auto stretch",
			style: Style{Position: PositionAbsolute, Height: pixels(10), Insets: Insets{Left: pixels(5), Right: pixels(15), Top: pixels(2)}},
			want:  Rect{X: 5, Y: 2, Width: 80, Height: 10},
		},
		{
			name:  "both and definite ignores end",
			style: Style{Position: PositionAbsolute, Width: pixels(20), Height: pixels(10), Insets: Insets{Left: pixels(5), Right: pixels(70), Top: pixels(2)}},
			want:  Rect{X: 5, Y: 2, Width: 20, Height: 10},
		},
		{
			name:  "neither uses content static start and margins",
			style: Style{Position: PositionAbsolute, Width: pixels(20), Height: pixels(10), Margin: Edges{Left: 3, Top: 4}},
			want:  Rect{X: 13, Y: 12, Width: 20, Height: 10},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := &Node{Axis: AxisRow, Style: Style{Padding: Edges{Top: 8, Right: 10, Bottom: 8, Left: 10}}, Children: []*Node{{Style: test.style}}}
			result, err := NewEngine(EngineOptions{}).Layout(root, exact(100, 50))
			if err != nil {
				t.Fatal(err)
			}
			assertRect(t, result.Children[0].Rect, test.want)
		})
	}
}

func TestAbsoluteRemovedFromFlowAndNestedOffset(t *testing.T) {
	abs := &Node{Style: Style{Position: PositionAbsolute, Width: pixels(10), Height: pixels(10), Insets: Insets{Left: pixels(7), Top: pixels(8)}}}
	nested := &Node{Axis: AxisRow, Style: Style{Width: pixels(40), Height: pixels(30), Padding: Edges{Left: 5, Top: 6}}, Children: []*Node{abs}}
	root := &Node{Axis: AxisRow, Gap: 10, Children: []*Node{fixedNode(1, 20, 20), nested}}
	result, err := NewEngine(EngineOptions{}).Layout(root, exact(70, 40))
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.Children[1].Rect.X, 30)
	assertRect(t, result.Children[1].Children[0].Rect, Rect{X: 37, Y: 8, Width: 10, Height: 10})
}

func TestNestedClipIntersectionAndStableStacking(t *testing.T) {
	grandchild := fixedNode(3, 80, 80)
	grandchild.Style.ZIndex = 2
	child := &Node{ID: 2, Axis: AxisRow, Style: Style{Width: pixels(70), Height: pixels(70), Overflow: OverflowClip, Margin: Edges{Left: 50, Top: 50}, ShrinkSet: true}, Children: []*Node{grandchild}}
	root := &Node{ID: 1, Axis: AxisRow, Style: Style{Overflow: OverflowClip}, Children: []*Node{child, fixedNode(4, 10, 10), fixedNode(5, 10, 10)}}
	root.Children[1].Style.ZIndex = 1
	root.Children[2].Style.ZIndex = 1
	result, err := NewEngine(EngineOptions{}).Layout(root, exact(100, 100))
	if err != nil {
		t.Fatal(err)
	}
	clip := result.Children[0].Children[0].Clip
	if !clip.Set {
		t.Fatal("descendant clip is unset")
	}
	assertRect(t, clip.Rect, Rect{X: 50, Y: 50, Width: 50, Height: 50})
	ordered := result.PaintOrder()
	if ordered[0].SourceIndex != 0 || ordered[1].SourceIndex != 1 || ordered[2].SourceIndex != 2 {
		t.Fatalf("paint order = %#v", ordered)
	}
	hit := result.HitTestOrder()
	if hit[0].SourceIndex != 2 || hit[1].SourceIndex != 1 {
		t.Fatalf("hit-test order = %#v", hit)
	}
}

func TestScrollMeasuresEnabledAxisUnboundedAndKeepsViewport(t *testing.T) {
	content := &Node{ID: 2, Axis: AxisColumn, Children: []*Node{
		fixedNode(3, 80, 40), fixedNode(4, 80, 40), fixedNode(5, 80, 40),
	}}
	for _, child := range content.Children {
		child.Style.Shrink, child.Style.ShrinkSet = 0, true
	}
	root := &Node{ID: 1, Scroll: ScrollVertical, Style: Style{Width: pixels(100), Height: pixels(60)}, Children: []*Node{content}}
	result, err := NewEngine(EngineOptions{}).Layout(root, Constraints{
		Width:  Limit{Min: 100, Max: 100, MaxSet: true, Definite: true},
		Height: Limit{Min: 60, Max: 60, MaxSet: true, Definite: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Viewport.Width != 100 || result.Viewport.Height != 60 || result.ContentExtent.Width != 80 || result.ContentExtent.Height != 120 {
		t.Fatalf("viewport/extent = %+v/%+v", result.Viewport, result.ContentExtent)
	}
	if result.Children[0].Rect.Height != 120 {
		t.Fatalf("content was constrained on scroll axis: %+v", result.Children[0].Rect)
	}
}

func TestMeasurementCacheReusesOnlyMatchingInputs(t *testing.T) {
	calls := 0
	leaf := &Node{ID: 2, Version: 1, Measure: IntrinsicMeasureFunc(func(IntrinsicRequest) (IntrinsicSize, error) {
		calls++
		return IntrinsicSize{Preferred: Size{Width: 12, Height: 8}}, nil
	})}
	root := &Node{ID: 1, Version: 1, Axis: AxisRow, Children: []*Node{leaf}}
	engine := NewEngine(EngineOptions{MeasurementCacheBytes: 4096})
	if _, err := engine.Layout(root, exact(100, 20)); err != nil {
		t.Fatal(err)
	}
	firstCalls := calls
	if _, err := engine.Layout(root, exact(100, 20)); err != nil {
		t.Fatal(err)
	}
	if calls != firstCalls || engine.Stats().CacheHits == 0 {
		t.Fatalf("unchanged pass calls/cache hits = %d/%d", calls, engine.Stats().CacheHits)
	}
	beforeStackChange := calls
	leaf.Style.ZIndex = 9
	leaf.Style.Overflow = OverflowClip
	if result, err := engine.Layout(root, exact(100, 20)); err != nil {
		t.Fatal(err)
	} else if result.Children[0].ZIndex != 9 || !result.Children[0].Clip.Set {
		t.Fatalf("stack/clip output did not update: %#v", result.Children[0])
	}
	if calls != beforeStackChange {
		t.Fatal("stack/clip-only change invalidated intrinsic measurement")
	}
	leaf.Version++
	if _, err := engine.Layout(root, exact(100, 20)); err != nil {
		t.Fatal(err)
	}
	if calls == firstCalls {
		t.Fatal("intrinsic content version change reused stale measurement")
	}
	beforeStyleChange := calls
	leaf.Style.MinWidth = pixels(20)
	if _, err := engine.Layout(root, exact(100, 20)); err != nil {
		t.Fatal(err)
	}
	if calls == beforeStyleChange {
		t.Fatal("layout style change reused stale measurement")
	}
	if engine.Stats().CacheBytes > 4096 {
		t.Fatalf("cache bytes = %d, budget 4096", engine.Stats().CacheBytes)
	}
}

func TestInvalidAndExtremeInputs(t *testing.T) {
	tests := []struct {
		name string
		node *Node
	}{
		{"negative size", &Node{Style: Style{Width: pixels(-1)}}},
		{"nan gap", &Node{Axis: AxisRow, Gap: float32(math.NaN())}},
		{"infinite grow", &Node{Style: Style{Grow: float32(math.Inf(1))}}},
		{"negative margin", &Node{Style: Style{Margin: Edges{Left: -1}}}},
		{"percentage over 100", &Node{Style: Style{Width: percent(101)}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewEngine(EngineOptions{}).Layout(test.node, Constraints{}); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}

	huge := &Node{Axis: AxisRow, Gap: math.MaxFloat32, Children: []*Node{fixedNode(1, math.MaxFloat32, 1), fixedNode(2, math.MaxFloat32, 1)}}
	if _, err := NewEngine(EngineOptions{}).Layout(huge, Constraints{}); err == nil {
		t.Fatal("overflowing arithmetic accepted")
	}
}

func TestEmptyAndZeroSizedContainers(t *testing.T) {
	for _, root := range []*Node{{Axis: AxisRow}, {Axis: AxisColumn}} {
		result, err := NewEngine(EngineOptions{}).Layout(root, exact(0, 0))
		if err != nil {
			t.Fatal(err)
		}
		assertRect(t, result.Rect, Rect{})
		if len(result.Children) != 0 {
			t.Fatal("empty container produced children")
		}
	}
}

func constantIntrinsic(width, height float32) IntrinsicMeasurer {
	return IntrinsicMeasureFunc(func(IntrinsicRequest) (IntrinsicSize, error) {
		return IntrinsicSize{Minimum: Size{Width: min(width, 4), Height: min(height, 4)}, Preferred: Size{Width: width, Height: height}}, nil
	})
}

func FuzzLayoutRandomTrees(f *testing.F) {
	f.Add(uint64(1), uint8(3), float32(1))
	f.Add(uint64(42), uint8(6), float32(1.25))
	f.Fuzz(func(t *testing.T, seed uint64, depth uint8, scale float32) {
		if !finite(scale) || scale <= 0 || scale > 8 {
			scale = 1
		}
		random := rand.New(rand.NewSource(int64(seed)))
		var nextID uint64
		var makeNode func(int) *Node
		makeNode = func(level int) *Node {
			nextID++
			node := &Node{ID: nextID, Version: 1}
			if level < int(depth%5) && random.Intn(3) != 0 {
				node.Axis = Axis(random.Intn(2) + 1)
				count := random.Intn(5)
				node.Children = make([]*Node, count)
				for index := range node.Children {
					node.Children[index] = makeNode(level + 1)
				}
			} else {
				node.Measure = constantIntrinsic(float32(random.Intn(80)), float32(random.Intn(40)))
			}
			node.Gap = float32(random.Intn(8))
			node.Align = Align(random.Intn(4))
			node.Justify = Justify(random.Intn(4))
			node.Style.Grow = float32(random.Intn(4))
			node.Style.ShrinkSet = true
			node.Style.Shrink = float32(random.Intn(3))
			node.Style.Padding = Edges{Top: float32(random.Intn(5)), Right: float32(random.Intn(5)), Bottom: float32(random.Intn(5)), Left: float32(random.Intn(5))}
			if random.Intn(5) == 0 {
				node.Style.Position = PositionAbsolute
				node.Style.Insets.Left = pixels(float32(random.Intn(20)))
				node.Style.Insets.Top = pixels(float32(random.Intn(20)))
			}
			return node
		}
		result, err := NewEngine(EngineOptions{MeasurementCacheBytes: 16 << 10}).Layout(makeNode(0), exact(float32(random.Intn(400)), float32(random.Intn(300))))
		if err != nil {
			t.Fatal(err)
		}
		var visit func(*Result)
		visit = func(current *Result) {
			if !finite(current.Rect.X) || !finite(current.Rect.Y) || !validSize(Size{Width: current.Rect.Width, Height: current.Rect.Height}) {
				t.Fatalf("invalid result: %#v", current.Rect)
			}
			for _, child := range current.Children {
				visit(child)
			}
		}
		visit(result)
	})
}

func BenchmarkLayout(b *testing.B) {
	for _, count := range []int{100, 1000} {
		b.Run(benchmarkName(count), func(b *testing.B) {
			root := benchmarkTree(count)
			engine := NewEngine(EngineOptions{MeasurementCacheBytes: 1 << 20})
			constraints := exact(1200, 800)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := engine.Layout(root, constraints); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func benchmarkTree(count int) *Node {
	root := &Node{ID: 1, Version: 1, Axis: AxisColumn, Gap: 1, Align: AlignStretch}
	root.Children = make([]*Node, 0, count/4+1)
	nextID := uint64(2)
	for remaining := count - 1; remaining > 0; {
		row := &Node{ID: nextID, Version: 1, Axis: AxisRow, Gap: 1, Style: Style{Height: pixels(12)}}
		nextID++
		for range min(4, remaining) {
			row.Children = append(row.Children, &Node{ID: nextID, Version: 1, Style: Style{Grow: 1}, Measure: constantIntrinsic(20, 10)})
			nextID++
			remaining--
		}
		root.Children = append(root.Children, row)
	}
	return root
}

func benchmarkName(count int) string {
	if count == 100 {
		return "100_nodes"
	}
	return "1000_nodes"
}
