package dxui

import (
	"fmt"
	"testing"

	"github.com/dxui-org/dxui/internal/tree"
)

func benchmarkVirtualList(count int, version uint64, offset float32, calls *int) View {
	return VirtualList(VirtualListProps{Key: "bench", Style: Style{Width: Px(800), Height: Px(600)},
		Count: count, Version: version, RowHeight: 30, Overscan: 3, Offset: Some(Point{Y: offset}),
		ItemKey: func(i int) string { return fmt.Sprintf("row-%d", i) },
		Build:   func(i int) View { *calls++; return Text(TextProps{Value: fmt.Sprintf("Row %d", i)}) },
	})
}

func BenchmarkVirtualListStages(b *testing.B) {
	for _, count := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("items-%d", count), func(b *testing.B) {
			calls := 0
			base, err := materializeVirtualLists(benchmarkVirtualList(count, 1, 0, &calls), View{}, nil, 600)
			if err != nil {
				b.Fatal(err)
			}
			app := NewApp(AppOptions{Width: 800, Height: 600})
			app.root = func() View { return base }
			if err := app.buildRoot(); err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { app.images.Clear(); app.releaseTextEngine() })
			mounted := mountedVirtualRows(app.view)

			b.Run("initial-build", func(b *testing.B) {
				builderCalls := 0
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if _, err := materializeVirtualLists(benchmarkVirtualList(count, 1, 0, &builderCalls), View{}, nil, 600); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(builderCalls)/float64(b.N), "builder-calls/op")
				b.ReportMetric(float64(mounted), "mounted-nodes")
			})

			b.Run("steady-scroll-build", func(b *testing.B) {
				builderCalls := 0
				previous := app.view
				b.ReportAllocs()
				b.ResetTimer()
				for i := range b.N {
					next, err := materializeVirtualLists(benchmarkVirtualList(count, 1, float32(i%10), &builderCalls), previous, app.retained.Root(), 600)
					if err != nil {
						b.Fatal(err)
					}
					previous = next
				}
				b.ReportMetric(float64(builderCalls)/float64(b.N), "builder-calls/op")
				b.ReportMetric(float64(mounted), "mounted-nodes")
			})

			b.Run("cross-window-build", func(b *testing.B) {
				builderCalls := 0
				previous := app.view
				b.ReportAllocs()
				b.ResetTimer()
				for i := range b.N {
					offset := float32((i%2)*(count/2)) * 30
					next, err := materializeVirtualLists(benchmarkVirtualList(count, 1, offset, &builderCalls), previous, app.retained.Root(), 600)
					if err != nil {
						b.Fatal(err)
					}
					previous = next
				}
				b.ReportMetric(float64(builderCalls)/float64(b.N), "builder-calls/op")
				b.ReportMetric(float64(mounted), "mounted-nodes")
			})

			description, err := describeView(app.view)
			if err != nil {
				b.Fatal(err)
			}
			b.Run("reconcile", func(b *testing.B) {
				var retained tree.Tree
				b.ReportAllocs()
				for range b.N {
					if _, _, err := retained.Preview(description); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(mounted), "mounted-nodes")
			})

			textEngine, err := app.textEngine()
			if err != nil {
				b.Fatal(err)
			}
			b.Run("layout", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					if _, _, err := layoutViewWithEngine(app.layoutEngine(), app.view, app.retained.Root(), app.currentLayoutConstraints(), app.theme.metric, makeIntrinsicResolver(textEngine, app.images, app.theme)); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(mounted), "mounted-nodes")
			})
			b.Run("display-hit", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					if _, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, textEngine, app.images, 1, 1, app.textSourceBudget(), app.display); err != nil {
						b.Fatal(err)
					}
					if _, _, err := buildInteractionSnapshot(app.view, app.retained.Root(), app.geometry, app.theme, textEngine); err != nil {
						b.Fatal(err)
					}
				}
				b.ReportMetric(float64(mounted), "mounted-nodes")
			})
		})
	}
}
