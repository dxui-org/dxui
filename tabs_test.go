package dxui

import (
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func tabItems() []TabItem {
	return []TabItem{
		{Value: "a", Label: "Alpha"},
		{Value: "b", Label: "Beta", Disabled: true},
		{Value: "c", Label: "Gamma"},
		{Value: "d", Label: "Delta"},
	}
}

func tabsAction(app *App) (uint64, inputAction) {
	for id, action := range app.inputActions {
		if action.tabs != nil {
			return id, action
		}
	}
	return 0, inputAction{}
}

func clickTab(t *testing.T, app *App, index int) {
	t.Helper()
	_, action := tabsAction(app)
	rect := action.tabItems[index]
	x, y := rect.X+rect.Width/2, rect.Y+rect.Height/2
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseDown, x, y)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, x, y)); err != nil {
		t.Fatal(err)
	}
}

func TestTabsCopiesAndNormalizesItems(t *testing.T) {
	items := []TabItem{{Value: "a\xff", Label: "A\xff"}}
	view := Tabs(TabsProps{Value: "a\xff", Items: items})
	items[0] = TabItem{Value: "changed", Label: "changed", Disabled: true}
	if got := view.node.tabs.Items[0]; got.Value != "a�" || got.Label != "A�" || got.Disabled {
		t.Fatalf("copied item = %+v", got)
	}
	if view.node.tabs.Value != "a�" {
		t.Fatalf("normalized controlled value = %q", view.node.tabs.Value)
	}
}

func TestTabsValidatesItemsTransactionally(t *testing.T) {
	for _, test := range []struct {
		name  string
		items []TabItem
		want  string
	}{
		{name: "empty value", items: []TabItem{{Label: "A"}}, want: "empty value"},
		{name: "duplicate value", items: []TabItem{{Value: "a"}, {Value: "a"}}, want: "duplicate tab item value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid := true
			app := NewApp(AppOptions{Width: 240, Height: 60})
			app.root = func() View {
				if valid {
					return Tabs(TabsProps{Key: "tabs", Value: "a", Items: tabItems()})
				}
				return Tabs(TabsProps{Key: "tabs", Items: test.items})
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			before := app.retained.Root().ID
			valid = false
			if err := app.buildRoot(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if app.retained.Root().ID != before || app.view.node.tabs.Items[0].Value != "a" {
				t.Fatal("invalid Tabs update changed the committed tree or view")
			}
		})
	}
}

func TestTabsAcceptsEmptyAndUnmatchedControlledValues(t *testing.T) {
	for _, props := range []TabsProps{
		{Value: "", Items: tabItems()},
		{Value: "missing", Items: tabItems()},
		{Value: "", Items: nil},
	} {
		app := NewApp(AppOptions{Width: 240, Height: 60})
		current := props
		app.root = func() View { return Tabs(current) }
		if err := app.buildRoot(); err != nil {
			t.Fatalf("props %+v: %v", props, err)
		}
		id, action := tabsAction(app)
		if id == 0 || len(action.tabItems) != len(props.Items) {
			t.Fatalf("interaction items = %d, want %d", len(action.tabItems), len(props.Items))
		}
		app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
		if app.interaction.Focused() != id {
			t.Fatal("enabled Tabs did not remain one focus stop")
		}
		selectedIndicators := 0
		for _, command := range app.currentDisplay() {
			if command.NodeID == id && command.Kind == paint.CommandFillRoundedRect && command.Rect.Height == 2 {
				selectedIndicators++
			}
		}
		if selectedIndicators != 0 {
			t.Fatalf("unmatched value painted %d selection indicators", selectedIndicators)
		}
	}
}

func TestTabsMouseSelectionIsControlledAndFiltered(t *testing.T) {
	value := "a"
	var proposals []string
	app := NewApp(AppOptions{Width: 320, Height: 60})
	app.root = func() View {
		return Tabs(TabsProps{Value: value, Items: tabItems(), OnChange: func(next string) {
			proposals = append(proposals, next)
		}})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	clickTab(t, app, 0)
	clickTab(t, app, 1)
	clickTab(t, app, 2)
	if len(proposals) != 1 || proposals[0] != "c" || app.view.node.tabs.Value != "a" {
		t.Fatalf("proposals/value = %v/%q", proposals, app.view.node.tabs.Value)
	}
	value = "c"
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	clickTab(t, app, 2)
	if len(proposals) != 1 {
		t.Fatalf("selected tab callback count = %d", len(proposals))
	}

	app = NewApp(AppOptions{Width: 200, Height: 60})
	app.root = func() View { return Tabs(TabsProps{Items: tabItems(), OnChange: nil}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	clickTab(t, app, 3)
}

func TestTabsKeyboardNavigationAndTabOrder(t *testing.T) {
	value := "a"
	app := NewApp(AppOptions{Width: 360, Height: 120})
	app.root = func() View {
		return Box(BoxProps{},
			Button(ButtonProps{Key: "before"}, Text(TextProps{Value: "Before"})),
			Tabs(TabsProps{Key: "tabs", Value: value, Items: tabItems(), OnChange: func(next string) { value = next }}),
			Button(ButtonProps{Key: "after"}, Text(TextProps{Value: "After"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	root := app.retained.Root()
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() != root.Children[0].ID {
		t.Fatal("first Tab did not focus preceding control")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	tabsID := root.Children[1].ID
	if app.interaction.Focused() != tabsID {
		t.Fatal("second Tab did not enter Tabs")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyRight, false))
	if node := findInstance(app.retained.Root(), tabsID); node.State.TabsActive != 2 || value != "a" {
		t.Fatalf("Right active/value = %d/%q", node.State.TabsActive, value)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false))
	if value != "c" {
		t.Fatalf("Enter selected %q, want c", value)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnd, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyLeft, false))
	if node := findInstance(app.retained.Root(), tabsID); node.State.TabsActive != 2 {
		t.Fatalf("Left active = %d, want 2", node.State.TabsActive)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnd, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false))
	if value != "d" {
		t.Fatalf("End+Space selected %q, want d", value)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyHome, false))
	if node := findInstance(app.retained.Root(), tabsID); node.State.TabsActive != 0 {
		t.Fatalf("Home active = %d", node.State.TabsActive)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() != app.retained.Root().Children[2].ID {
		t.Fatal("Tab did not leave Tabs in normal source order")
	}
}

func TestTabsActiveIdentitySurvivesItemReorder(t *testing.T) {
	items := tabItems()
	app := NewApp(AppOptions{Width: 320, Height: 60})
	app.root = func() View { return Tabs(TabsProps{Key: "tabs", Value: "a", Items: items}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyRight, false))
	id, _ := tabsAction(app)
	if node := findInstance(app.retained.Root(), id); node.State.TabsActiveKey != "c" {
		t.Fatalf("active key = %q", node.State.TabsActiveKey)
	}
	items = []TabItem{items[2], items[0], items[1], items[3]}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	node := findInstance(app.retained.Root(), id)
	if node.State.TabsActiveKey != "c" || node.State.TabsActive != 0 {
		t.Fatalf("reordered active key/index = %q/%d", node.State.TabsActiveKey, node.State.TabsActive)
	}
}

func TestTabsInvalidationSeparatesValueFromItems(t *testing.T) {
	value := "a"
	items := tabItems()
	app := NewApp(AppOptions{Width: 320, Height: 60})
	app.root = func() View { return Tabs(TabsProps{Value: value, Items: items}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	value = "c"
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) || changes.Has(tree.DirtyMeasure) {
		t.Fatalf("Value-only dirty flags = %v", changes.Dirty)
	}
	items = append([]TabItem(nil), items...)
	items[0].Label = "A much wider Alpha"
	changes, err = app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(tree.DirtyMeasure) || !changes.Has(tree.DirtyLayout) || !changes.Has(tree.DirtyPaint) {
		t.Fatalf("Items dirty flags = %v", changes.Dirty)
	}
}

func TestTabsHoverPressedDisabledAndCapture(t *testing.T) {
	app := NewApp(AppOptions{Width: 320, Height: 60})
	app.root = func() View { return Tabs(TabsProps{Value: "a", Items: tabItems(), OnChange: func(string) {}}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id, action := tabsAction(app)
	item := action.tabItems[2]
	x, y := item.X+item.Width/2, item.Y+item.Height/2
	app.handleEvent(appPointer(platform.EventMouseMove, x, y))
	if findInstance(app.retained.Root(), id).State.TabsHover != 2 {
		t.Fatal("hover did not identify the label")
	}
	hover, _ := app.theme.color(TokenColor(ColorSemanticTabsHover))
	if !displayHasFill(app.currentDisplay(), id, hover) {
		t.Fatal("hover state did not paint its semantic color")
	}
	disabledItem := action.tabItems[1]
	app.handleEvent(appPointer(platform.EventMouseMove, disabledItem.X+disabledItem.Width/2, disabledItem.Y+disabledItem.Height/2))
	if node := findInstance(app.retained.Root(), id); node.State.TabsHover != -1 {
		t.Fatalf("disabled label retained hover index %d", node.State.TabsHover)
	}
	app.handleEvent(appPointer(platform.EventMouseMove, x, y))
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	if findInstance(app.retained.Root(), id).State.TabsPressed != 2 {
		t.Fatal("mouse down did not paint the pressed label")
	}
	pressed, _ := app.theme.color(TokenColor(ColorSemanticTabsPressed))
	if !displayHasFill(app.currentDisplay(), id, pressed) {
		t.Fatal("pressed state did not paint its semantic color")
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 400, y))
	capture := app.tabsCapture
	if findInstance(app.retained.Root(), id).State.TabsPressed != -1 || capture.id != id || capture.index != 2 {
		t.Fatalf("outside capture state = %+v/%+v", findInstance(app.retained.Root(), id).State, capture)
	}
	app.handleEvent(appPointer(platform.EventMouseUp, 400, y))
	if app.tabsCapture.id != 0 {
		t.Fatal("mouse release retained Tabs capture")
	}

	app = NewApp(AppOptions{Width: 320, Height: 60})
	app.root = func() View {
		return Tabs(TabsProps{Items: tabItems(), Disabled: true, OnChange: func(string) { t.Fatal("disabled callback") }})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() != 0 {
		t.Fatal("disabled Tabs entered focus order")
	}
}

func displayHasFill(list paint.DisplayList, id uint64, color RGBAColor) bool {
	for _, command := range list {
		if command.NodeID == id && command.Kind == paint.CommandFillRoundedRect && command.Color == paintColor(color) {
			return true
		}
	}
	return false
}

func TestTabsLayoutThemeClippingLocalStyleAndDPI(t *testing.T) {
	base := Tabs(TabsProps{Value: "a", Items: tabItems()})
	geometry := intrinsicChildGeometry(t, LightTheme(), base)
	if geometry.Height != 40 || geometry.Width <= 0 {
		t.Fatalf("default intrinsic geometry = %+v", geometry)
	}
	theme := LightTheme()
	theme.Semantic.Metrics[MetricComponentTabsHeight] = Metric(52)
	theme.Semantic.Metrics[MetricComponentTabsGap] = Metric(10)
	themed := intrinsicChildGeometry(t, theme, base)
	if themed.Height != 52 || themed.Width <= geometry.Width {
		t.Fatalf("themed intrinsic geometry = %+v, base %+v", themed, geometry)
	}
	before, _ := prepareTheme(LightTheme())
	after, _ := prepareTheme(theme)
	if !themeChangesLayout(base, before, after) {
		t.Fatal("Tabs metric change did not invalidate layout")
	}

	background := RGBA(7, 8, 9, 255)
	app := NewApp(AppOptions{Width: 90, Height: 30})
	app.root = func() View {
		return Tabs(TabsProps{Style: Style{
			Width: Px(90), Height: Px(30), Background: LiteralColor(background),
			Padding: EdgeValues{Left: Metric(3), Right: Metric(4)},
		}, Value: "a", Items: tabItems()})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id, _ := tabsAction(app)
	textEngine, err := app.textEngine()
	if err != nil {
		t.Fatal(err)
	}
	display, err := buildDisplayList(app.view, app.retained.Root(), app.geometry, app.theme, textEngine, app.images, 1.5, 1.5, app.textSourceBudget(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var foundBackground, foundClip, foundIndicator, foundScaledText bool
	for _, command := range display {
		if command.NodeID != id {
			continue
		}
		if command.Kind == paint.CommandFillRoundedRect && command.Color == paintColor(background) && command.Rect == paintRect(app.geometry.Rect) {
			foundBackground = true
		}
		if command.Kind == paint.CommandPushClip && command.Rect == paintRect(app.geometry.Content) {
			foundClip = true
		}
		if command.Kind == paint.CommandFillRoundedRect && command.Rect.Height == 2 {
			foundIndicator = true
		}
		if command.Kind == paint.CommandDrawText && command.Text != nil && float32(command.Text.Height) > command.Rect.Height {
			foundScaledText = true
		}
	}
	if !foundBackground || !foundClip || !foundIndicator || !foundScaledText {
		t.Fatalf("display flags background=%v clip=%v indicator=%v scaledText=%v", foundBackground, foundClip, foundIndicator, foundScaledText)
	}
}
