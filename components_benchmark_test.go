package dxui

import (
	"fmt"
	"image"
	"testing"

	"github.com/dxui-org/dxui/internal/tree"
)

func benchmarkLargeList(changed string) View {
	rows := make([]View, 1000)
	for index := range rows {
		value := fmt.Sprintf("row %04d", index)
		if index == 500 && changed != "" {
			value = changed
		}
		rows[index] = Text(TextProps{Key: fmt.Sprintf("row-%04d", index), Style: Style{Height: Px(24), Shrink: NoShrink()}, Value: value})
	}
	return Scroll(ScrollProps{Style: Style{Width: Px(480), Height: Px(600)}}, Box(BoxProps{}, rows...))
}

func BenchmarkLargeListPipeline(b *testing.B) {
	b.Run("description-build/no-business-change", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_ = benchmarkLargeList("")
		}
	})
	b.Run("description-build/single-field", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_ = benchmarkLargeList("row 0500 changed")
		}
	})

	base, changed := benchmarkLargeList(""), benchmarkLargeList("row 0500 changed")
	baseDescription, err := describeView(base)
	if err != nil {
		b.Fatal(err)
	}
	changedDescription, err := describeView(changed)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("reconcile/no-business-change", func(b *testing.B) {
		var retained tree.Tree
		if _, err := retained.Update(baseDescription); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if _, err := retained.Update(baseDescription); err != nil {
				b.Fatal(err)
			}
			retained.ClearDirty()
		}
	})
	b.Run("reconcile/single-field", func(b *testing.B) {
		var retained tree.Tree
		if _, err := retained.Update(baseDescription); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for index := range b.N {
			next := changedDescription
			if index%2 != 0 {
				next = baseDescription
			}
			if _, err := retained.Update(next); err != nil {
				b.Fatal(err)
			}
			retained.ClearDirty()
		}
	})
	b.Run("whole-root-update/no-business-change", func(b *testing.B) {
		app := NewApp(AppOptions{Width: 480, Height: 600})
		app.root = func() View { return benchmarkLargeList("") }
		if err := app.buildRoot(); err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() {
			app.images.Clear()
			app.releaseTextEngine()
		})
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if err := app.buildRoot(); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("whole-root-update/single-field", func(b *testing.B) {
		app := NewApp(AppOptions{Width: 480, Height: 600})
		changed := ""
		app.root = func() View { return benchmarkLargeList(changed) }
		if err := app.buildRoot(); err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() {
			app.images.Clear()
			app.releaseTextEngine()
		})
		b.ReportAllocs()
		b.ResetTimer()
		for index := range b.N {
			if index%2 == 0 {
				changed = "row 0500 changed"
			} else {
				changed = ""
			}
			if err := app.buildRoot(); err != nil {
				b.Fatal(err)
			}
		}
	})

	app := NewApp(AppOptions{Width: 480, Height: 600})
	app.root = func() View { return base }
	if err := app.buildRoot(); err != nil {
		b.Fatal(err)
	}
	b.Run("layout", func(b *testing.B) {
		textEngine, err := app.textEngine()
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if _, _, err := layoutViewWithEngine(app.layoutEngine(), app.view, app.retained.Root(), app.currentLayoutConstraints(), app.theme.metric, makeIntrinsicResolver(textEngine, app.images, app.theme)); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("paint-data", func(b *testing.B) {
		textEngine, err := app.textEngine()
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if _, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, textEngine, app.images, 1, 1, app.textSourceBudget(), nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkGalleryView(source ImageSource) View {
	icon := IconData{ViewBox: Rect{Width: 10, Height: 10}, Commands: []PathCommand{
		{Verb: PathMove, Points: [3]Point{{X: 1, Y: 1}}},
		{Verb: PathLine, Points: [3]Point{{X: 9, Y: 1}}},
		{Verb: PathLine, Points: [3]Point{{X: 5, Y: 9}}},
		{Verb: PathClose},
	}}
	return Box(BoxProps{Gap: 8},
		Text(TextProps{Value: "Components"}),
		Button(ButtonProps{}, Text(TextProps{Value: "Button"})),
		ToggleSwitch(ToggleSwitchProps{Checked: true}),
		Checkbox(CheckboxProps{Checked: true}, Text(TextProps{Value: "Checkbox"})),
		Radio(RadioProps{Selected: true}, Text(TextProps{Value: "Radio"})),
		Icon(IconProps{Data: icon, Size: 20}),
		Image(ImageProps{Style: Style{Width: Px(96), Height: Px(64)}, Source: source, Fit: ImageContain}),
		Avatar(AvatarProps{Source: source, Size: 40}),
	)
}

func BenchmarkComponentsGallery(b *testing.B) {
	source := ImageFromGo(image.NewNRGBA(image.Rect(0, 0, 96, 64)))
	b.Run("build", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_ = benchmarkGalleryView(source)
		}
	})
	b.Run("reconcile", func(b *testing.B) {
		view := benchmarkGalleryView(source)
		description, err := describeView(view)
		if err != nil {
			b.Fatal(err)
		}
		var retained tree.Tree
		if _, err := retained.Update(description); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if _, err := retained.Update(description); err != nil {
				b.Fatal(err)
			}
			retained.ClearDirty()
		}
	})
	app := NewApp(AppOptions{Width: 640, Height: 480})
	app.root = func() View { return benchmarkGalleryView(source) }
	if err := app.buildRoot(); err != nil {
		b.Fatal(err)
	}
	b.Run("layout", func(b *testing.B) {
		textEngine, err := app.textEngine()
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if _, _, err := layoutViewWithEngine(app.layoutEngine(), app.view, app.retained.Root(), app.currentLayoutConstraints(), app.theme.metric, makeIntrinsicResolver(textEngine, app.images, app.theme)); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("paint", func(b *testing.B) {
		textEngine, err := app.textEngine()
		if err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if _, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, textEngine, app.images, 1, 1, app.textSourceBudget(), nil); err != nil {
				b.Fatal(err)
			}
		}
		b.StopTimer()
		stats := app.images.Stats()
		b.ReportMetric(float64(stats.Bytes), "image-cache-bytes")
	})
}
