package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func TestBadgeDefaultsMeasureContentAndAcceptArbitraryChild(t *testing.T) {
	icon := IconData{ViewBox: Rect{Width: 10, Height: 10}, Commands: []PathCommand{
		{Verb: PathMove, Points: [3]Point{{X: 1, Y: 1}}},
		{Verb: PathLine, Points: [3]Point{{X: 9, Y: 5}}},
		{Verb: PathLine, Points: [3]Point{{X: 1, Y: 9}}},
		{Verb: PathClose},
	}}
	app := NewApp(AppOptions{Width: 300, Height: 80})
	app.root = func() View {
		return Box(BoxProps{Align: AlignStart}, Badge(BadgeProps{}, Box(BoxProps{Direction: Horizontal, Gap: 4, Align: AlignCenter},
			Icon(IconProps{Data: icon, Size: 12}),
			Text(TextProps{Value: "New"}),
		)))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	badge := app.geometry.Children[0]
	if badge.Rect.Height != 24 || badge.Content.X != 8 || badge.Content.Y != 2 || badge.Content.Width != badge.Rect.Width-16 || badge.Content.Height != 20 {
		t.Fatalf("default Badge geometry = rect %+v content %+v", badge.Rect, badge.Content)
	}
	if len(badge.Children) != 1 || len(badge.Children[0].Children) != 2 {
		t.Fatalf("arbitrary child geometry was not retained: %+v", badge.Children)
	}
	if badge.Children[0].Rect.Width+16 != badge.Rect.Width {
		t.Fatalf("Badge width %g does not wrap child width %g plus padding", badge.Rect.Width, badge.Children[0].Rect.Width)
	}
}

func TestBadgeLayoutStyleOverridesAndShrink(t *testing.T) {
	app := NewApp(AppOptions{Width: 70, Height: 40})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Align: AlignStart}, Badge(BadgeProps{Style: Style{
			MinWidth: Px(50), MaxWidth: Px(80), Padding: Edges(1, 3, 1, 3),
		}}, Text(TextProps{Style: Style{Width: Px(100), Height: Px(18)}, Value: "wide"})))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	badge := app.geometry.Children[0]
	if badge.Rect.Width != 70 || badge.Rect.Height != 24 {
		t.Fatalf("shrunk Badge geometry = %+v, want 70x24", badge.Rect)
	}
	if badge.Content.X != 3 || badge.Content.Width != 64 {
		t.Fatalf("overridden padding/content = %+v", badge.Content)
	}

	fixed := intrinsicChildGeometry(t, LightTheme(), Badge(BadgeProps{Style: Style{
		Width: Px(44), Height: Px(30), MinWidth: Px(60), MaxWidth: Px(55),
	}}, Text(TextProps{Value: "x"})))
	if fixed.Width != 60 || fixed.Height != 30 {
		t.Fatalf("Badge min/max clamp = %+v, want 60x30", fixed)
	}
}

func TestBadgeThemeDefaultsInheritanceAndLocalOverrides(t *testing.T) {
	for name, source := range map[string]Theme{"light": LightTheme(), "dark": DarkTheme()} {
		t.Run(name, func(t *testing.T) {
			if _, ok := source.Components[ComponentBadge]; !ok {
				t.Fatal("missing ComponentBadge")
			}
			theme, err := prepareTheme(source)
			if err != nil {
				t.Fatal(err)
			}
			for token, want := range map[MetricToken]float32{
				MetricComponentBadgePaddingX: 8, MetricComponentBadgePaddingY: 2,
				MetricComponentBadgeMinHeight: 24, MetricComponentBadgeRadius: 999,
			} {
				if got, metricErr := theme.metric(TokenMetric(token)); metricErr != nil || got != want {
					t.Fatalf("metric %q = %g, %v; want %g", token, got, metricErr, want)
				}
			}
		})
	}

	icon := IconData{ViewBox: Rect{Width: 2, Height: 2}, Commands: []PathCommand{{Verb: PathMove}, {Verb: PathLine, Points: [3]Point{{X: 2, Y: 2}}}}}
	app := NewApp(AppOptions{Width: 200, Height: 40})
	app.root = func() View {
		return Badge(BadgeProps{Style: Style{Width: Px(200), Height: Px(40)}}, Box(BoxProps{Direction: Horizontal, Gap: 4},
			Text(TextProps{Value: "New"}), Icon(IconProps{Data: icon, Size: 12}),
			Icon(IconProps{Data: icon, Size: 12, Color: LiteralColor(RGBA(6, 5, 4, 255))}),
			Text(TextProps{Style: Style{Text: TextStyle{Color: LiteralColor(RGBA(9, 8, 7, 255))}}, Value: "custom"}),
		))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	wantInherited := paint.Color{R: 255, G: 255, B: 255, A: 255}
	var inheritedText, localText, inheritedIcon, localIcon bool
	for _, command := range app.display {
		switch command.Kind {
		case paint.CommandDrawText:
			if command.Color == wantInherited {
				inheritedText = true
			}
			if command.Color == (paint.Color{R: 9, G: 8, B: 7, A: 255}) {
				localText = true
			}
		case paint.CommandDrawIcon:
			if command.Color == wantInherited {
				inheritedIcon = true
			}
			if command.Color == (paint.Color{R: 6, G: 5, B: 4, A: 255}) {
				localIcon = true
			}
		}
	}
	if !inheritedText || !localText || !inheritedIcon || !localIcon {
		t.Fatalf("Badge tint inheritance/local override = text:%t local-text:%t icon:%t local-icon:%t", inheritedText, localText, inheritedIcon, localIcon)
	}
}

func TestBadgeIsNotInteractiveOrFocusable(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	app.root = func() View {
		return Badge(BadgeProps{Style: Style{Width: Px(100), Height: Px(40)}}, Text(TextProps{Value: "passive"}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.currentDisplay().HitTestAt(10, 10); ok {
		t.Fatal("Badge unexpectedly participated in hit testing")
	}
	for _, event := range []platform.Event{
		appPointer(platform.EventMouseMove, 10, 10),
		appPointer(platform.EventMouseDown, 10, 10),
		appPointer(platform.EventMouseUp, 10, 10),
		appKey(platform.EventKeyDown, platform.KeyTab, false),
		appKey(platform.EventKeyDown, platform.KeyEnter, false),
		appKey(platform.EventKeyDown, platform.KeySpace, false),
	} {
		if dirty, callback, err := app.handleEvent(event); err != nil || dirty || callback {
			t.Fatalf("passive Badge event result = dirty:%t callback:%t err:%v", dirty, callback, err)
		}
	}
	if app.interaction.Focused() != 0 {
		t.Fatalf("Badge received focus identity %d", app.interaction.Focused())
	}
}

func TestBadgeThemeSwitchInvalidationAndPaint(t *testing.T) {
	app := NewApp(AppOptions{Width: 100, Height: 40})
	app.root = func() View { return Badge(BadgeProps{}, Text(TextProps{Value: "New"})) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.running, app.events = true, &appFakeEvents{}

	paintOnly := DarkTheme()
	component := paintOnly.Components[ComponentBadge]
	component.Base.Background = Some(LiteralColor(RGBA(120, 30, 40, 200)))
	component.Base.TextColor = Some(LiteralColor(RGBA(2, 3, 4, 230)))
	component.Base.Border = Some(Border{Width: Metric(2), Color: LiteralColor(RGBA(5, 6, 7, 210))})
	component.Base.Opacity = Some(float32(.5))
	paintOnly.Components[ComponentBadge] = component
	if err := app.SetTheme(paintOnly); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("Badge paint theme switch = dirty:%t err:%v", dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 1 || got.PaintCount != 2 {
		t.Fatalf("paint-only Badge theme switch diagnostics = %+v", got)
	}
	var combined paint.Command
	var opacity bool
	for _, command := range app.display {
		if command.Kind == paint.CommandFillStrokeRoundedRect {
			combined = command
		}
		if command.Kind == paint.CommandPushOpacity && command.Opacity == .5 {
			opacity = true
		}
	}
	if combined.Color != (paint.Color{R: 120, G: 30, B: 40, A: 200}) || combined.BorderColor != (paint.Color{R: 5, G: 6, B: 7, A: 210}) || combined.Width != 2 {
		t.Fatalf("themed Badge fill/border = %+v", combined)
	}
	if !opacity {
		t.Fatal("themed Badge opacity did not enter the display stack")
	}

	metric := paintOnly
	metric.Semantic.Metrics[MetricComponentBadgePaddingX] = Metric(12)
	metric.Semantic.Metrics[MetricComponentBadgeMinHeight] = Metric(30)
	if err := app.SetTheme(metric); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("Badge metric theme switch = dirty:%t err:%v", dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 2 || got.PaintCount != 3 {
		t.Fatalf("metric Badge theme switch diagnostics = %+v", got)
	}
}

func TestBadgeContentAndStyleInvalidation(t *testing.T) {
	value := "1"
	background := LiteralColor(RGBA(10, 20, 30, 255))
	app := NewApp(AppOptions{Width: 200, Height: 40})
	app.root = func() View {
		return Box(BoxProps{Align: AlignStart}, Badge(BadgeProps{Style: Style{Background: background}}, Text(TextProps{Value: value})))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	background = LiteralColor(RGBA(40, 50, 60, 255))
	changes, err := app.buildAndCommit()
	if err != nil || !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) {
		t.Fatalf("Badge visual Style update = dirty:%v err:%v", changes.Dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 1 || got.PaintCount != 2 {
		t.Fatalf("Badge visual Style invalidation = %+v", got)
	}
	value = "a much wider count"
	changes, err = app.buildAndCommit()
	if err != nil || !changes.Has(tree.DirtyMeasure) || !changes.Has(tree.DirtyLayout) || !changes.Has(tree.DirtyPaint) {
		t.Fatalf("Badge content update = dirty:%v err:%v", changes.Dirty, err)
	}
	if got := app.Diagnostics(); got.LayoutCount != 2 || got.PaintCount != 3 {
		t.Fatalf("Badge content invalidation = %+v", got)
	}
}

func TestBadgeScrollTranslationClipZIndexAndScale(t *testing.T) {
	badge := func(key string, z int, color RGBAColor) View {
		return Badge(BadgeProps{Key: key, Style: Style{
			Width: Px(50), Height: Px(24), Position: PositionAbsolute, Insets: Insets{Top: Px(10)},
			ZIndex: z, Background: LiteralColor(color), Overflow: OverflowClip,
		}}, Text(TextProps{Value: key}))
	}
	app := NewApp(AppOptions{Width: 60, Height: 24})
	app.root = func() View {
		return Scroll(ScrollProps{Style: Style{Width: Px(60), Height: Px(24)}, Axis: ScrollVertical, Offset: Some(Point{Y: 10}), Scrollbar: ScrollbarHidden},
			Box(BoxProps{Style: Style{Height: Px(34)}, Align: AlignStart},
				badge("front", 2, RGBA(20, 0, 0, 255)), badge("back", 1, RGBA(10, 0, 0, 255))))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	var fills []paint.Command
	var clipped bool
	for _, command := range app.display {
		if command.Kind == paint.CommandFillRoundedRect {
			fills = append(fills, command)
		}
		if command.Kind == paint.CommandPushClip && command.Rect == (paint.Rect{Width: 60, Height: 24}) {
			clipped = true
		}
	}
	if len(fills) != 2 || fills[0].Color.R != 10 || fills[1].Color.R != 20 {
		t.Fatalf("Badge z-index fill order = %+v", fills)
	}
	if !clipped {
		t.Fatal("Scroll/Badge clipping was not represented in the display stack")
	}
	for _, command := range fills {
		if command.Rect.Y != 0 {
			t.Fatalf("Badge Scroll translation Y = %g, want 0", command.Rect.Y)
		}
		if command.Radii.TopLeft != 999 || command.Radii.TopRight != 999 {
			t.Fatalf("Badge pill radius token = %+v, want renderer-clamped 999", command.Radii)
		}
	}

	one, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, app.text, app.images, 1, 1, app.textSourceBudget(), nil)
	if err != nil {
		t.Fatal(err)
	}
	two, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, app.text, app.images, 2, 2, app.textSourceBudget(), nil)
	if err != nil {
		t.Fatal(err)
	}
	firstMask, secondMask := 0, 0
	for _, command := range one {
		if command.Kind == paint.CommandDrawText {
			firstMask = command.Text.Width
			break
		}
	}
	for _, command := range two {
		if command.Kind == paint.CommandDrawText {
			secondMask = command.Text.Width
			break
		}
	}
	if firstMask == 0 || secondMask < firstMask*2-4 || secondMask > firstMask*2+4 {
		t.Fatalf("Badge child DPI masks = %d at 1x, %d at 2x", firstMask, secondMask)
	}
}
