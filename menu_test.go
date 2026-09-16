package dxui

import (
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/tree"
)

func menuItems() []MenuItem {
	return []MenuItem{
		{Value: "new", Label: "New"},
		{Value: "locked", Label: "Locked", Disabled: true},
		{Value: "open", Label: "Open document"},
		{Value: "quit", Label: "Quit"},
	}
}

func menuAction(app *App) (uint64, inputAction) {
	for id, action := range app.inputActions {
		if action.menu != nil {
			return id, action
		}
	}
	return 0, inputAction{}
}

func clickMenuItem(t *testing.T, app *App, index int) {
	t.Helper()
	_, action := menuAction(app)
	rect := action.menuItems[index]
	x, y := rect.X+rect.Width/2, rect.Y+rect.Height/2
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseDown, x, y)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, x, y)); err != nil {
		t.Fatal(err)
	}
}

func TestMenuCopiesNormalizesAndValidatesItemsTransactionally(t *testing.T) {
	items := []MenuItem{{Value: "a\xff", Label: "A\xff"}}
	view := Menu(MenuProps{Value: "a\xff", Items: items})
	items[0] = MenuItem{Value: "changed", Label: "changed", Disabled: true}
	if got := view.node.menu.Items[0]; got.Value != "a�" || got.Label != "A�" || got.Disabled {
		t.Fatalf("copied item = %+v", got)
	}

	if got, want := view.node.menu.Value, strings.ToValidUTF8("a\xff", "\uFFFD"); got != want {
		t.Fatalf("normalized controlled value = %q, want %q", got, want)
	}

	for _, test := range []struct {
		name  string
		props MenuProps
		want  string
	}{
		{name: "empty value", props: MenuProps{Items: []MenuItem{{Label: "A"}}}, want: "empty value"},
		{name: "duplicate value", props: MenuProps{Items: []MenuItem{{Value: "a"}, {Value: "a"}}}, want: "duplicate menu item value"},
		{name: "orientation", props: MenuProps{Orientation: MenuOrientation(9)}, want: "invalid menu orientation"},
	} {
		t.Run(test.name, func(t *testing.T) {
			valid := true
			app := NewApp(AppOptions{Width: 240, Height: 180})
			app.root = func() View {
				if valid {
					return Menu(MenuProps{Key: "menu", Items: menuItems()})
				}
				props := test.props
				props.Key = "menu"
				return Menu(props)
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			before := app.retained.Root().ID
			valid = false
			if err := app.buildRoot(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if app.retained.Root().ID != before || app.view.node.menu.Items[0].Value != "new" {
				t.Fatal("invalid Menu update changed the committed tree or view")
			}
		})
	}

	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View { return Menu(MenuProps{}) }
	if err := app.buildRoot(); err != nil {
		t.Fatalf("empty Menu: %v", err)
	}
}

func TestMenuControlledSelectionPersistsWithoutFocusAndIsPaintOnly(t *testing.T) {
	value := "open"
	var actions []string
	app := NewApp(AppOptions{Width: 300, Height: 240})
	app.root = func() View {
		return Box(BoxProps{},
			Menu(MenuProps{Key: "menu", Value: value, Items: menuItems(), OnAction: func(next string) {
				actions = append(actions, next)
			}}),
			Button(ButtonProps{}, Text(TextProps{Value: "After"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id, action := menuAction(app)
	selected, _ := app.theme.color(TokenColor(ColorSemanticMenuSelected))
	if !displayHasMenuItemFill(app.currentDisplay(), id, action.menuItems[2], selected) {
		t.Fatal("controlled selected item did not paint before focus")
	}

	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
	if findInstance(app.retained.Root(), id).State.MenuActive != 2 {
		t.Fatal("keyboard current did not move onto the selected item")
	}
	if !displayHasMenuItemFill(app.currentDisplay(), id, action.menuItems[2], selected) {
		t.Fatal("controlled selected item changed when Menu received focus")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if findInstance(app.retained.Root(), id).State.Focused {
		t.Fatal("Menu retained focus after Tab moved to the next control")
	}
	if !displayHasMenuItemFill(app.currentDisplay(), id, action.menuItems[2], selected) {
		t.Fatal("controlled selected item disappeared after focus left Menu")
	}

	clickMenuItem(t, app, 2)
	if len(actions) != 1 || actions[0] != value {
		t.Fatalf("selected-item actions = %v, want [%q]", actions, value)
	}

	value = "quit"
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyMeasure) || changes.Has(tree.DirtyLayout) {
		t.Fatalf("Value-only dirty flags = %v", changes.Dirty)
	}
}

func TestMenuAcceptsEmptyAndUnmatchedControlledValues(t *testing.T) {
	for _, value := range []string{"", "missing"} {
		app := NewApp(AppOptions{Width: 240, Height: 180})
		app.root = func() View { return Menu(MenuProps{Value: value, Items: menuItems()}) }
		if err := app.buildRoot(); err != nil {
			t.Fatalf("Value %q: %v", value, err)
		}
		id, _ := menuAction(app)
		selected, _ := app.theme.color(TokenColor(ColorSemanticMenuSelected))
		if displayHasFill(app.currentDisplay(), id, selected) {
			t.Fatalf("Value %q painted a selected item", value)
		}
	}
}

func displayHasMenuItemFill(commands []paint.Command, id uint64, rect paint.Rect, color RGBAColor) bool {
	for _, command := range commands {
		if command.NodeID == id && command.Kind == paint.CommandFillRoundedRect && command.Rect == rect && command.Color == paintColor(color) {
			return true
		}
	}
	return false
}

func TestMenuLayoutDirectionsAndVerticalFill(t *testing.T) {
	vertical := NewApp(AppOptions{Width: 220, Height: 180})
	vertical.root = func() View {
		return Menu(MenuProps{Style: Style{Width: Px(180)}, Items: menuItems()})
	}
	if err := vertical.buildRoot(); err != nil {
		t.Fatal(err)
	}
	_, verticalAction := menuAction(vertical)
	if len(verticalAction.menuItems) != 4 {
		t.Fatalf("vertical item count = %d", len(verticalAction.menuItems))
	}
	for index, item := range verticalAction.menuItems {
		if item.Width != vertical.geometry.Content.Width {
			t.Errorf("vertical item %d width = %g, want content width %g", index, item.Width, vertical.geometry.Content.Width)
		}
		if index > 0 && item.Y <= verticalAction.menuItems[index-1].Y {
			t.Errorf("vertical item %d did not advance Y", index)
		}
	}

	horizontal := NewApp(AppOptions{Width: 600, Height: 80})
	horizontal.root = func() View {
		return Box(BoxProps{Align: AlignStart}, Menu(MenuProps{Orientation: MenuHorizontal, Items: menuItems()}))
	}
	if err := horizontal.buildRoot(); err != nil {
		t.Fatal(err)
	}
	_, horizontalAction := menuAction(horizontal)
	for index, item := range horizontalAction.menuItems {
		if index > 0 && item.X <= horizontalAction.menuItems[index-1].X {
			t.Errorf("horizontal item %d did not advance X", index)
		}
		if item.Width >= horizontal.geometry.Content.Width {
			t.Errorf("horizontal item %d unexpectedly filled content width", index)
		}
	}
	if horizontal.geometry.Children[0].Rect.Width >= 600 {
		t.Fatalf("horizontal intrinsic width = %g, want content-sized", horizontal.geometry.Children[0].Rect.Width)
	}
}

func TestMenuMouseDisabledStatesAndAction(t *testing.T) {
	var actions []string
	app := NewApp(AppOptions{Width: 240, Height: 180})
	app.root = func() View {
		return Menu(MenuProps{Items: menuItems(), OnAction: func(value string) { actions = append(actions, value) }})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id, action := menuAction(app)
	open := action.menuItems[2]
	x, y := open.X+open.Width/2, open.Y+open.Height/2
	app.handleEvent(appPointer(platform.EventMouseMove, x, y))
	if findInstance(app.retained.Root(), id).State.MenuHover != 2 {
		t.Fatal("hover did not identify enabled item")
	}
	hover, _ := app.theme.color(TokenColor(ColorSemanticMenuHover))
	if !displayHasFill(app.currentDisplay(), id, hover) {
		t.Fatal("hover token did not paint")
	}
	clickMenuItem(t, app, 1)
	clickMenuItem(t, app, 2)
	if len(actions) != 1 || actions[0] != "open" {
		t.Fatalf("actions = %v", actions)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	pressed, _ := app.theme.color(TokenColor(ColorSemanticMenuPressed))
	if node := findInstance(app.retained.Root(), id); node.State.MenuPressed != 2 || !displayHasFill(app.currentDisplay(), id, pressed) {
		t.Fatal("pressed item state did not paint")
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 400, y))
	if node := findInstance(app.retained.Root(), id); node.State.MenuPressed != -1 || app.menuCapture.id != id {
		t.Fatal("moving outside did not clear pressed while retaining capture")
	}
	app.handleEvent(appPointer(platform.EventMouseUp, 400, y))
	if len(actions) != 1 || app.menuCapture.id != 0 {
		t.Fatal("outside release activated or retained capture")
	}

	disabled := NewApp(AppOptions{Width: 240, Height: 180})
	disabled.root = func() View {
		return Menu(MenuProps{Items: menuItems(), Disabled: true, OnAction: func(string) { t.Fatal("disabled callback") }})
	}
	if err := disabled.buildRoot(); err != nil {
		t.Fatal(err)
	}
	disabled.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if disabled.interaction.Focused() != 0 {
		t.Fatal("disabled Menu entered focus order")
	}
}

func TestMenuDefaultStyleRoundsMenuAndItemsWithoutHoverTextShift(t *testing.T) {
	for _, theme := range []Theme{LightTheme(), DarkTheme()} {
		app := NewApp(AppOptions{Width: 240, Height: 180, Theme: theme})
		app.root = func() View { return Menu(MenuProps{Items: menuItems()}) }
		if err := app.buildRoot(); err != nil {
			t.Fatal(err)
		}
		id, action := menuAction(app)
		item := action.menuItems[2]
		if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, item.X+item.Width/2, item.Y+item.Height/2)); err != nil {
			t.Fatal(err)
		}

		surface, _ := app.theme.color(TokenColor(ColorSemanticSurfaceHi))
		hover, _ := app.theme.color(TokenColor(ColorSemanticMenuHover))
		textColor, _ := app.theme.color(TokenColor(ColorSemanticText))
		accentHover, _ := app.theme.color(TokenColor(ColorSemanticAccentHover))
		var roundedMenu, roundedItem, baseText bool
		for _, command := range app.currentDisplay() {
			if command.NodeID != id {
				continue
			}
			switch command.Kind {
			case paint.CommandFillRoundedRect:
				if command.Color == paintColor(surface) && command.Radii.TopLeft > 0 {
					roundedMenu = true
				}
				if command.Color == paintColor(hover) && command.Radii.TopLeft > 0 {
					roundedItem = true
				}
			case paint.CommandDrawText:
				if command.Color == paintColor(accentHover) {
					t.Fatal("hover changed Menu item text color")
				}
				baseText = baseText || command.Color == paintColor(textColor)
			}
		}
		if !roundedMenu || !roundedItem || !baseText {
			t.Fatalf("rounded menu/item/base text = %v/%v/%v", roundedMenu, roundedItem, baseText)
		}
	}
}

func TestMenuKeyboardNavigationSingleTabStopAndNoWrap(t *testing.T) {
	var action string
	app := NewApp(AppOptions{Width: 300, Height: 220})
	app.root = func() View {
		return Box(BoxProps{},
			Button(ButtonProps{Key: "before"}, Text(TextProps{Value: "Before"})),
			Menu(MenuProps{Key: "menu", Items: menuItems(), OnAction: func(value string) { action = value }}),
			Button(ButtonProps{Key: "after"}, Text(TextProps{Value: "After"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	root := app.retained.Root()
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	menuID := root.Children[1].ID
	if app.interaction.Focused() != menuID {
		t.Fatal("Menu was not one Tab stop")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
	if node := findInstance(app.retained.Root(), menuID); node.State.MenuActive != 2 {
		t.Fatalf("Down active = %d, want 2", node.State.MenuActive)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false))
	if action != "open" {
		t.Fatalf("Enter action = %q", action)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnd, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
	if node := findInstance(app.retained.Root(), menuID); node.State.MenuActive != 3 {
		t.Fatalf("Down wrapped from End to %d", node.State.MenuActive)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyHome, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false))
	if action != "new" {
		t.Fatalf("Space action = %q", action)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() != root.Children[2].ID {
		t.Fatal("Tab did not leave Menu")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, true))
	if app.interaction.Focused() != menuID {
		t.Fatal("Shift+Tab did not re-enter Menu")
	}

	horizontal := NewApp(AppOptions{Width: 400, Height: 80})
	horizontal.root = func() View { return Menu(MenuProps{Orientation: MenuHorizontal, Items: menuItems()}) }
	if err := horizontal.buildRoot(); err != nil {
		t.Fatal(err)
	}
	horizontal.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	id, _ := menuAction(horizontal)
	horizontal.handleEvent(appKey(platform.EventKeyDown, platform.KeyRight, false))
	if node := findInstance(horizontal.retained.Root(), id); node.State.MenuActive != 2 {
		t.Fatalf("Right active = %d, want 2", node.State.MenuActive)
	}
	horizontal.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
	if node := findInstance(horizontal.retained.Root(), id); node.State.MenuActive != 2 {
		t.Fatalf("orthogonal Down changed horizontal active to %d", node.State.MenuActive)
	}
}

func TestMenuActiveIdentityUpdateFallbackAndCallbackUnmount(t *testing.T) {
	items := menuItems()
	showMenu := true
	count := 0
	app := NewApp(AppOptions{Width: 260, Height: 180})
	app.root = func() View {
		if !showMenu {
			return Text(TextProps{Value: "removed"})
		}
		return Menu(MenuProps{Key: "menu", Items: items, OnAction: func(string) {
			count++
			showMenu = false
		}})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
	id, _ := menuAction(app)
	items = []MenuItem{items[2], items[0], items[1], items[3]}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if node := findInstance(app.retained.Root(), id); node.State.MenuActiveKey != "open" || node.State.MenuActive != 0 {
		t.Fatalf("reordered active = %q/%d", node.State.MenuActiveKey, node.State.MenuActive)
	}
	items[0].Disabled = true
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if node := findInstance(app.retained.Root(), id); node.State.MenuActiveKey != "new" || node.State.MenuActive != 1 {
		t.Fatalf("disabled fallback = %q/%d", node.State.MenuActiveKey, node.State.MenuActive)
	}
	_, beforeReorder := menuAction(app)
	pressed := beforeReorder.menuItems[1]
	x, y := pressed.X+pressed.Width/2, pressed.Y+pressed.Height/2
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	items = []MenuItem{items[3], items[0], items[1], items[2]}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if app.menuCapture.id != 0 || findInstance(app.retained.Root(), id).State.MenuPressed != -1 {
		t.Fatal("item reorder did not cancel pointer capture")
	}
	app.handleEvent(appPointer(platform.EventMouseUp, x, y))
	if count != 0 {
		t.Fatal("release after item reorder activated an action")
	}
	clickMenuItem(t, app, 2)
	if count != 1 || app.view.node.kind != viewText || app.menuCapture.id != 0 {
		t.Fatalf("callback unmount count/kind/capture = %d/%d/%d", count, app.view.node.kind, app.menuCapture.id)
	}
}

func TestMenuScrollClipAndThemeChanges(t *testing.T) {
	count := 0
	app := NewApp(AppOptions{Width: 180, Height: 50})
	app.root = func() View {
		return Scroll(ScrollProps{Style: Style{Width: Px(180), Height: Px(50)}, Axis: ScrollVertical, Scrollbar: ScrollbarHidden},
			Menu(MenuProps{Items: menuItems(), OnAction: func(string) { count++ }}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	_, action := menuAction(app)
	hidden := action.menuItems[2]
	app.handleEvent(appPointer(platform.EventMouseDown, hidden.X+2, hidden.Y+hidden.Height/2))
	app.handleEvent(appPointer(platform.EventMouseUp, hidden.X+2, hidden.Y+hidden.Height/2))
	if count != 0 {
		t.Fatal("clipped Menu item activated")
	}

	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	before := app.Diagnostics()
	paintTheme := LightTheme()
	paintTheme.Semantic.Colors[ColorSemanticMenuActive] = LiteralColor(RGBA(1, 2, 3, 80))
	if err := app.SetTheme(paintTheme); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("paint theme dirty/error = %v/%v", dirty, err)
	}
	afterPaint := app.Diagnostics()
	if afterPaint.LayoutCount != before.LayoutCount || afterPaint.PaintCount != before.PaintCount+1 {
		t.Fatalf("paint theme layout/paint = %d/%d -> %d/%d", before.LayoutCount, before.PaintCount, afterPaint.LayoutCount, afterPaint.PaintCount)
	}
	metricTheme := paintTheme
	metricTheme.Semantic.Metrics[MetricComponentMenuItemHeight] = Metric(44)
	if err := app.SetTheme(metricTheme); err != nil {
		t.Fatal(err)
	}
	if dirty, err := app.runQueuedWork(); err != nil || !dirty {
		t.Fatalf("metric theme dirty/error = %v/%v", dirty, err)
	}
	if afterMetric := app.Diagnostics(); afterMetric.LayoutCount != afterPaint.LayoutCount+1 {
		t.Fatalf("item height did not relayout: %d -> %d", afterPaint.LayoutCount, afterMetric.LayoutCount)
	}

	id, _ := menuAction(app)
	activeColor, _ := app.theme.color(TokenColor(ColorSemanticMenuActive))
	if !displayHasFill(app.currentDisplay(), id, activeColor) {
		t.Fatal("focused keyboard current item did not use active token")
	}
}
