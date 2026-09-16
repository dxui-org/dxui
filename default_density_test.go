package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/platform"
)

func intrinsicChildGeometry(t *testing.T, theme Theme, child View) Rect {
	t.Helper()
	app := NewApp(AppOptions{Width: 800, Height: 240, Theme: theme})
	app.root = func() View { return Box(BoxProps{Align: AlignStart}, child) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	geometry := app.geometry.Children[0].Rect
	return Rect{X: geometry.X, Y: geometry.Y, Width: geometry.Width, Height: geometry.Height}
}

func TestComfortableBuiltInDefaultMetrics(t *testing.T) {
	theme, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	for token, want := range map[MetricToken]float32{
		MetricSemanticTextSize: 15, MetricSemanticLineHeight: 20,
		MetricSemanticControlHeight: 36, MetricComponentButtonPaddingX: 14,
		MetricComponentButtonGroupBorderWidth: 1, MetricComponentButtonGroupRadius: 8,
		MetricComponentBadgePaddingX: 8, MetricComponentBadgePaddingY: 2,
		MetricComponentBadgeMinHeight: 24, MetricComponentBadgeRadius: 999,
		MetricComponentInputPaddingX: 12, MetricComponentIconSize: 20,
		MetricComponentInputGroupPaddingX: 12, MetricComponentInputGroupGap: 8,
		MetricComponentAvatarSize:   40,
		MetricComponentCheckboxSize: 18, MetricComponentCheckboxGap: 8,
		MetricComponentRadioSize: 18, MetricComponentRadioGap: 8,
		MetricComponentSelectItemHeight: 36, MetricComponentSelectGap: 4,
		MetricComponentTabsHeight: 40, MetricComponentTabsGap: 4,
		MetricComponentTabsPaddingX: 12, MetricComponentTabsPaddingY: 8,
		MetricComponentTabsIndicator: 2, MetricComponentTabsIndicatorInset: 2,
		MetricComponentMenuItemHeight: 36, MetricComponentMenuGap: 4,
		MetricComponentMenuPaddingX: 12, MetricComponentMenuPaddingY: 8,
	} {
		got, metricErr := theme.metric(TokenMetric(token))
		if metricErr != nil || got != want {
			t.Fatalf("metric %q = %g, %v; want %g", token, got, metricErr, want)
		}
	}
}

func TestBuiltInToggleSwitchUsesPillCorners(t *testing.T) {
	want := UniformCorners(Metric(12))
	for name, theme := range map[string]Theme{"light": LightTheme(), "dark": DarkTheme()} {
		t.Run(name, func(t *testing.T) {
			got, ok := theme.Components[ComponentToggleSwitch].Base.Radius.get()
			if !ok || got != want {
				t.Fatalf("toggle switch corners = %+v, set=%t; want %+v", got, ok, want)
			}
		})
	}
}

func TestSemanticControlsUseCoordinatedDefaultGeometry(t *testing.T) {
	icon := IconData{ViewBox: Rect{Width: 10, Height: 10}, Commands: []PathCommand{{Verb: PathMove}}}
	for _, test := range []struct {
		name       string
		view       View
		wantHeight float32
		wantWidth  float32
	}{
		{name: "button", view: Button(ButtonProps{}, Text(TextProps{Value: "OK"})), wantHeight: 36},
		{name: "badge", view: Badge(BadgeProps{}, Text(TextProps{Value: "New"})), wantHeight: 24},
		{name: "input", view: Input(InputProps{Placeholder: "placeholder"}), wantHeight: 36},
		{name: "input group", view: InputGroup(InputGroupProps{}, InputGroupContent{Input: Input(InputProps{Placeholder: "placeholder"})}), wantHeight: 36},
		{name: "textarea", view: Textarea(TextareaProps{Placeholder: "placeholder"}), wantHeight: 80},
		{name: "select", view: Select(SelectProps{Placeholder: "Choose"}), wantHeight: 36},
		{name: "checkbox", view: Checkbox(CheckboxProps{}, Text(TextProps{Value: "Label"})), wantHeight: 36},
		{name: "radio", view: Radio(RadioProps{}, Text(TextProps{Value: "Label"})), wantHeight: 36},
		{name: "toggle", view: ToggleSwitch(ToggleSwitchProps{}), wantHeight: 24, wantWidth: 40},
		{name: "slider", view: Slider(SliderProps{}), wantHeight: 24, wantWidth: 160},
		{name: "tabs", view: Tabs(TabsProps{}), wantHeight: 40},
		{name: "menu", view: Menu(MenuProps{Items: []MenuItem{{Value: "one", Label: "One"}}}), wantHeight: 36},
		{name: "icon", view: Icon(IconProps{Data: icon}), wantHeight: 20, wantWidth: 20},
		{name: "avatar", view: Avatar(AvatarProps{Source: avatarTestSource(10, 10)}), wantHeight: 40, wantWidth: 40},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := intrinsicChildGeometry(t, LightTheme(), test.view)
			if got.Height != test.wantHeight || test.wantWidth != 0 && got.Width != test.wantWidth {
				t.Fatalf("default geometry = %+v, want height %g width %g", got, test.wantHeight, test.wantWidth)
			}
		})
	}
}

func TestThemeMetricsAndExplicitSizesKeepTheirPrecedence(t *testing.T) {
	theme := LightTheme()
	theme.Semantic.Metrics[MetricSemanticTextSize] = Metric(17)
	theme.Semantic.Metrics[MetricSemanticLineHeight] = Metric(24)
	theme.Semantic.Metrics[MetricSemanticControlHeight] = Metric(44)
	theme.Semantic.Metrics[MetricComponentInputPaddingX] = Metric(16)
	theme.Semantic.Metrics[MetricComponentInputPaddingY] = Metric(10)

	themed := intrinsicChildGeometry(t, theme, Input(InputProps{}))
	if themed.Height != 44 || themed.Width != 72 {
		t.Fatalf("theme-overridden input geometry = %+v, want 72x44", themed)
	}
	explicit := intrinsicChildGeometry(t, theme, Input(InputProps{Style: Style{
		Height: Px(30), Padding: UniformEdges(Metric(0)), Text: TextStyle{Size: 14, LineHeight: 18},
	}}))
	if explicit.Height != 30 || explicit.Width != 40 {
		t.Fatalf("explicit input geometry = %+v, want 40x30", explicit)
	}
	before, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	after, err := prepareTheme(theme)
	if err != nil {
		t.Fatal(err)
	}
	if !themeChangesLayout(Input(InputProps{}), before, after) {
		t.Fatal("changed default control metrics did not invalidate layout")
	}
	if themeChangesLayout(Input(InputProps{Style: Style{
		Height: Px(30), Padding: UniformEdges(Metric(0)), Text: TextStyle{Size: 14, LineHeight: 18},
	}}), before, after) {
		t.Fatal("theme defaults invalidated a fully explicit input")
	}
}

func TestDefaultCheckboxWholeRowIsClickable(t *testing.T) {
	changed := false
	app := NewApp(AppOptions{Width: 200, Height: 80})
	app.root = func() View {
		return Box(BoxProps{Align: AlignStart}, Checkbox(CheckboxProps{OnChange: func(bool) { changed = true }}, Text(TextProps{Value: "Label"})))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 30, 34))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, 30, 34)); err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("default checkbox row did not hit outside the 18-unit visual mark")
	}
}
