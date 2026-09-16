package dxui

import (
	"fmt"
	"testing"

	"github.com/dxui-org/dxui/internal/tree"
)

type longListUpdateState struct {
	activeRow int
	revision  int
	offset    float32
}

type longListUpdateScenario struct {
	name          string
	before, after longListUpdateState
}

var longListUpdateScenarios = []longListUpdateScenario{
	{name: "no-change", before: longListUpdateState{activeRow: -1}, after: longListUpdateState{activeRow: -1}},
	{name: "scroll-only", before: longListUpdateState{activeRow: -1}, after: longListUpdateState{activeRow: -1, offset: 370}},
	{name: "row-only", before: longListUpdateState{activeRow: 500}, after: longListUpdateState{activeRow: 500, revision: 1}},
	{name: "scroll-and-row", before: longListUpdateState{activeRow: 1, revision: 1, offset: 37}, after: longListUpdateState{activeRow: 2, revision: 2, offset: 74}},
}

func TestLongListUpdateScenarioDirtyClasses(t *testing.T) {
	for _, scenario := range longListUpdateScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			before, err := describeView(benchmarkLongListUpdate(scenario.before))
			if err != nil {
				t.Fatal(err)
			}
			after, err := describeView(benchmarkLongListUpdate(scenario.after))
			if err != nil {
				t.Fatal(err)
			}
			var retained tree.Tree
			if _, err := retained.Update(before); err != nil {
				t.Fatal(err)
			}
			retained.ClearDirty()
			changes, err := retained.Update(after)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario.name {
			case "no-change":
				if changes.Dirty != tree.DirtyBuild {
					t.Fatalf("dirty = %08b, want build only", changes.Dirty)
				}
			case "scroll-only":
				want := tree.DirtyBuild | tree.DirtyDisplay | tree.DirtyPaint | tree.DirtySemantics
				if changes.Dirty&want != want || changes.Dirty&(tree.DirtyMeasure|tree.DirtyLayout) != 0 {
					t.Fatalf("dirty = %08b, want %08b without measure/layout", changes.Dirty, want)
				}
			case "row-only":
				want := tree.DirtyBuild | tree.DirtyMeasure | tree.DirtyLayout | tree.DirtyDisplay | tree.DirtyPaint
				if changes.Dirty&want != want || changes.Dirty&tree.DirtySemantics != 0 {
					t.Fatalf("dirty = %08b, want %08b without semantics", changes.Dirty, want)
				}
			case "scroll-and-row":
				want := tree.DirtyBuild | tree.DirtyMeasure | tree.DirtyLayout | tree.DirtyDisplay | tree.DirtyPaint | tree.DirtySemantics
				if changes.Dirty&want != want {
					t.Fatalf("dirty = %08b, want at least %08b", changes.Dirty, want)
				}
			}
		})
	}
}

func benchmarkLongListUpdate(state longListUpdateState) View {
	rows := make([]View, 1000)
	for index := range rows {
		value := fmt.Sprintf("Row %04d   fixed data", index)
		if index == state.activeRow {
			value += fmt.Sprintf("   update %04d", state.revision)
		}
		rows[index] = Text(TextProps{
			Key:   fmt.Sprintf("row-%04d", index),
			Value: value,
			Style: Style{Height: Px(28), Shrink: NoShrink()},
		})
	}
	return Scroll(ScrollProps{
		Key:       "list",
		Axis:      ScrollVertical,
		Offset:    Some(Point{Y: state.offset}),
		Scrollbar: ScrollbarAlways,
	}, Box(BoxProps{Style: Style{Padding: Padding(16)}}, rows...))
}

// BenchmarkLongListUpdateStages keeps the native-baseline list shape while
// separating work that the root update pipeline otherwise reports as one
// event-to-present interval. Layout is benchmarked for every scenario as a
// phase cost; the evidence report records whether reconciliation schedules it.
func BenchmarkLongListUpdateStages(b *testing.B) {
	for _, scenario := range longListUpdateScenarios {
		scenario := scenario
		b.Run(scenario.name, func(b *testing.B) {
			beforeView := benchmarkLongListUpdate(scenario.before)
			afterView := benchmarkLongListUpdate(scenario.after)
			beforeDescription, err := describeView(beforeView)
			if err != nil {
				b.Fatal(err)
			}
			afterDescription, err := describeView(afterView)
			if err != nil {
				b.Fatal(err)
			}

			b.Run("view-build", func(b *testing.B) {
				b.ReportAllocs()
				for index := range b.N {
					state := scenario.after
					if index%2 != 0 {
						state = scenario.before
					}
					_ = benchmarkLongListUpdate(state)
				}
			})

			b.Run("whole-root-update", func(b *testing.B) {
				state := scenario.before
				updateApp := NewApp(AppOptions{Width: 900, Height: 680})
				updateApp.root = func() View { return benchmarkLongListUpdate(state) }
				if err := updateApp.buildRoot(); err != nil {
					b.Fatal(err)
				}
				b.Cleanup(func() {
					updateApp.images.Clear()
					updateApp.releaseTextEngine()
				})
				b.ReportAllocs()
				b.ResetTimer()
				for index := range b.N {
					state = scenario.after
					if index%2 != 0 {
						state = scenario.before
					}
					if err := updateApp.buildRoot(); err != nil {
						b.Fatal(err)
					}
				}
			})

			app := NewApp(AppOptions{Width: 900, Height: 680})
			app.root = func() View { return beforeView }
			if err := app.buildRoot(); err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() {
				app.images.Clear()
				app.releaseTextEngine()
			})

			b.Run("theme-validation", func(b *testing.B) {
				b.ReportAllocs()
				for index := range b.N {
					view := afterView
					if index%2 != 0 {
						view = beforeView
					}
					if err := validateViewTheme(view, app.theme); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("description-adapter", func(b *testing.B) {
				b.ReportAllocs()
				for index := range b.N {
					view := afterView
					if index%2 != 0 {
						view = beforeView
					}
					if _, err := describeView(view); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("retained-transaction", func(b *testing.B) {
				var retained tree.Tree
				if _, err := retained.Update(beforeDescription); err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for index := range b.N {
					next := afterDescription
					if index%2 != 0 {
						next = beforeDescription
					}
					if _, _, err := retained.Preview(next); err != nil {
						b.Fatal(err)
					}
					if _, err := retained.Update(next); err != nil {
						b.Fatal(err)
					}
					retained.ClearDirty()
				}
			})

			textEngine, err := app.textEngine()
			if err != nil {
				b.Fatal(err)
			}
			b.Run("layout", func(b *testing.B) {
				b.ReportAllocs()
				for index := range b.N {
					view, retained := afterView, app.retained.Root()
					if index%2 != 0 {
						view = beforeView
					}
					if _, _, err := layoutViewWithEngine(app.layoutEngine(), view, retained, app.currentLayoutConstraints(), app.theme.metric, makeIntrinsicResolver(textEngine, app.images, app.theme)); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("display", func(b *testing.B) {
				b.ReportAllocs()
				for index := range b.N {
					view := afterView
					if index%2 != 0 {
						view = beforeView
					}
					if _, err := buildDisplayList(view, app.retained.Root(), app.geometry, app.theme, textEngine, app.images, 1, 1, app.textSourceBudget(), app.display); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run("input-snapshot", func(b *testing.B) {
				b.ReportAllocs()
				for index := range b.N {
					view := afterView
					if index%2 != 0 {
						view = beforeView
					}
					if _, _, err := buildInteractionSnapshot(view, app.retained.Root(), app.geometry, app.theme, textEngine); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
