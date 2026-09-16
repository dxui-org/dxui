package dxui

import (
	"fmt"
	"math"
	"testing"
)

func virtualRows(count int, version uint64, offset Option[Point], builds *int) View {
	return VirtualList(VirtualListProps{
		Key: "list", Style: Style{Width: Px(120), Height: Px(100)}, Count: count,
		Version: version, RowHeight: 20, Overscan: 1, Offset: offset,
		ItemKey: func(i int) string { return fmt.Sprintf("row-%d", i) },
		Build: func(i int) View {
			*builds++
			return TextButton(ButtonProps{}, fmt.Sprintf("row %d", i))
		},
	})
}

func mountedVirtualRows(view View) int {
	if view.node == nil || len(view.node.children) != 1 || view.node.children[0].node == nil {
		return 0
	}
	count := 0
	for _, child := range view.node.children[0].node.children {
		if child.node != nil && child.node.key != "" {
			count++
		}
	}
	return count
}

func TestVirtualWindowBoundaries(t *testing.T) {
	tests := []struct {
		name             string
		count            int
		viewport, offset float32
		over             int
		want             virtualListWindow
	}{
		{"empty", 0, 100, 0, 2, virtualListWindow{}},
		{"zero viewport", 10, 0, 0, 2, virtualListWindow{}},
		{"start", 100, 45, 0, 1, virtualListWindow{0, 4}},
		{"partial", 100, 45, 21, 1, virtualListWindow{0, 5}},
		{"end", 10, 40, 160, 1, virtualListWindow{7, 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := virtualWindow(tt.count, 20, tt.viewport, tt.offset, tt.over); got != tt.want {
				t.Fatalf("window = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVirtualListBuildsOnlyWindowAndReusesStableRows(t *testing.T) {
	builds := 0
	app := NewApp(AppOptions{Width: 120, Height: 100})
	app.root = func() View { return virtualRows(100_000, 1, Option[Point]{}, &builds) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := mountedVirtualRows(app.view); got != 6 || builds != 6 {
		t.Fatalf("mounted=%d builds=%d, want 6/6", got, builds)
	}
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if builds != 6 {
		t.Fatalf("unchanged rebuild called builder: %d", builds)
	}
	app.retained.Root().State.ScrollY = 10_000
	if _, err := app.commitView(app.view); err != nil {
		t.Fatal(err)
	}
	if got := mountedVirtualRows(app.view); got > 7 || builds > 13 {
		t.Fatalf("cross-window mounted=%d builds=%d", got, builds)
	}
}

func TestVirtualListControlledOffsetAndContentClamp(t *testing.T) {
	builds := 0
	offset := Some(Point{Y: 10_000})
	count := 100
	app := NewApp(AppOptions{Width: 120, Height: 100})
	app.root = func() View { return virtualRows(count, uint64(count), offset, &builds) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got := app.retained.Root().State.ScrollY; got != 1900 {
		t.Fatalf("clamp=%v", got)
	}
	offset = Some(Point{Y: 40})
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if got := app.retained.Root().State.ScrollY; got != 40 {
		t.Fatalf("controlled=%v", got)
	}
	count = 2
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if got := app.retained.Root().State.ScrollY; got != 0 {
		t.Fatalf("short clamp=%v", got)
	}
}

func TestVirtualListRejectsInvalidAndRollsBack(t *testing.T) {
	builds := 0
	app := NewApp(AppOptions{Width: 120, Height: 100})
	app.root = func() View { return virtualRows(10, 1, Option[Point]{}, &builds) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	oldRoot, oldView := app.retained.Root(), app.view
	cases := []VirtualListProps{
		{Count: 1, RowHeight: 20, ItemKey: func(int) string { return "x" }, Build: func(int) View { return Label("x") }},
		{Style: Style{Height: Px(100)}, Count: -1, RowHeight: 20, ItemKey: func(int) string { return "x" }, Build: func(int) View { return Label("x") }},
		{Style: Style{Height: Px(100)}, Count: 1, RowHeight: float32(math.NaN()), ItemKey: func(int) string { return "x" }, Build: func(int) View { return Label("x") }},
		{Style: Style{Height: Px(100)}, Count: 2, RowHeight: 20, ItemKey: func(int) string { return "same" }, Build: func(int) View { return Label("x") }},
		{Style: Style{Height: Px(100)}, Count: 1, RowHeight: 20, ItemKey: func(int) string { panic("key") }, Build: func(int) View { return Label("x") }},
		{Style: Style{Height: Px(100)}, Count: 1, RowHeight: 20, ItemKey: func(int) string { return "x" }, Build: func(int) View { panic("build") }},
	}
	for i, props := range cases {
		if _, err := app.commitView(VirtualList(props)); err == nil {
			t.Fatalf("case %d accepted", i)
		}
		if app.retained.Root() != oldRoot || app.view.node != oldView.node {
			t.Fatalf("case %d changed committed state", i)
		}
	}
}

func TestVirtualListVersionRebuildsVisibleOnly(t *testing.T) {
	builds := 0
	version := uint64(1)
	app := NewApp(AppOptions{Width: 120, Height: 100})
	app.root = func() View { return virtualRows(100_000, version, Option[Point]{}, &builds) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	first := builds
	version++
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if delta := builds - first; delta != mountedVirtualRows(app.view) {
		t.Fatalf("visible update builds=%d", delta)
	}
}
