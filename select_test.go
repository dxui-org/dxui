package dxui

import (
	"math"
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

func selectOptions() []SelectOption {
	return []SelectOption{
		{Value: "a", Label: "Alpha"},
		{Value: "b", Label: "Beta", Disabled: true},
		{Value: "c", Label: "Gamma"},
		{Value: "d", Label: "Delta"},
	}
}

func selectNode(key string, value string, options []SelectOption, change func(string)) View {
	return Select(SelectProps{Key: key, Style: Style{Width: Px(80), Height: Px(32), Shrink: Some(float32(0))}, Value: value, Options: options, Placeholder: "Choose", OnChange: change})
}

func selectActionID(app *App) uint64 {
	for id, action := range app.inputActions {
		if action.selectp != nil {
			return id
		}
	}
	return 0
}

func openFocusedSelect(t *testing.T, app *App) uint64 {
	t.Helper()
	if action := app.inputActions[app.interaction.Focused()]; action.selectp == nil {
		if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false)); err != nil {
			t.Fatal(err)
		}
	}
	id := app.interaction.Focused()
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false))
	if _, _, err := app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false)); err != nil {
		t.Fatal(err)
	}
	if node := findInstance(app.retained.Root(), id); node == nil || !node.State.SelectOpen {
		t.Fatal("Select did not open from Enter")
	}
	return id
}

func TestSelectRejectsDuplicateAndInvalidOptionsTransactionally(t *testing.T) {
	tests := []struct {
		name    string
		options []SelectOption
		want    string
	}{
		{name: "empty value", options: []SelectOption{{Label: "A"}}, want: "empty value"},
		{name: "duplicate value", options: []SelectOption{{Value: "a"}, {Value: "a"}}, want: "duplicate option value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 100, Height: 80})
			app.root = func() View { return selectNode("select", "", test.options, nil) }
			if err := app.buildRoot(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
			if app.retained.Root() != nil {
				t.Fatal("invalid initial Select committed a retained tree")
			}
		})
	}
}

func TestSelectRejectsOptionCountAboveMVPBoundary(t *testing.T) {
	options := make([]SelectOption, selectOptionLimit+1)
	for index := range options {
		options[index] = SelectOption{Value: string(rune(index + 1)), Label: "x"}
	}
	app := NewApp(AppOptions{Width: 100, Height: 80})
	app.root = func() View { return selectNode("select", "", options, nil) }
	if err := app.buildRoot(); err == nil || !strings.Contains(err.Error(), "MVP maximum") {
		t.Fatalf("over-limit error = %v", err)
	}
}

func TestSelectAnchorPaintsRightAlignedDropdownIndicator(t *testing.T) {
	app := NewApp(AppOptions{Width: 140, Height: 80})
	app.root = func() View { return selectNode("select", "", nil, nil) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := selectActionID(app)
	content := paintRect(app.geometry.Content)
	var indicator paint.Command
	var textClip paint.Rect
	for _, command := range app.currentDisplay() {
		if command.NodeID != id {
			continue
		}
		switch command.Kind {
		case paint.CommandPushClip:
			textClip = command.Rect
		case paint.CommandDrawIcon:
			indicator = command
		}
	}
	if indicator.Text == nil || indicator.Rect.Width != selectIndicatorSize || indicator.Rect.Height != selectIndicatorSize {
		t.Fatalf("Select indicator = %+v, want %gx%g icon", indicator, selectIndicatorSize, selectIndicatorSize)
	}
	if indicator.Rect.X+indicator.Rect.Width != content.X+content.Width || indicator.Rect.Y+(indicator.Rect.Height/2) != content.Y+(content.Height/2) {
		t.Fatalf("Select indicator rect = %+v, content = %+v", indicator.Rect, content)
	}
	if textClip.X+textClip.Width+selectIndicatorGap != indicator.Rect.X {
		t.Fatalf("Select text clip/indicator gap = %+v/%+v, want %g", textClip, indicator.Rect, selectIndicatorGap)
	}
}

func TestSelectIntrinsicWidthReservesDropdownIndicator(t *testing.T) {
	selectRect := intrinsicChildGeometry(t, LightTheme(), Select(SelectProps{}))
	inputRect := intrinsicChildGeometry(t, LightTheme(), Input(InputProps{}))
	if selectRect.Width != inputRect.Width+selectIndicatorSize+selectIndicatorGap {
		t.Fatalf("Select/Input intrinsic widths = %g/%g, want Select to reserve %g", selectRect.Width, inputRect.Width, selectIndicatorSize+selectIndicatorGap)
	}
}

func TestSelectPopupUsesCompactDefaultGap(t *testing.T) {
	theme, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	anchor := layout.Rect{X: 10, Y: 20, Width: 80, Height: 36}
	popup, err := selectPopupGeometry(anchor, layout.Rect{Width: 200, Height: 200}, 2, theme)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := popup.Popup.Y-(anchor.Y+anchor.Height), float32(4); got != want {
		t.Fatalf("Select popup gap = %g, want %g", got, want)
	}
}

func TestSelectOverlayEscapesScrollClipAndWinsStacking(t *testing.T) {
	presses := 0
	app := NewApp(AppOptions{Width: 140, Height: 120})
	app.root = func() View {
		scroll := Scroll(ScrollProps{Style: Style{Width: Px(80), Height: Px(42), Shrink: Some(float32(0))}, Axis: ScrollVertical},
			selectNode("select", "a", selectOptions(), nil))
		under := Button(ButtonProps{Style: Style{Position: PositionAbsolute, Insets: Insets{Left: Px(90), Top: Px(55)}, Width: Px(40), Height: Px(30), ZIndex: math.MaxInt}, OnPress: func() { presses++ }}, Text(TextProps{Value: "under"}))
		return Box(BoxProps{}, scroll, under)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	actionID := selectActionID(app)
	action := app.inputActions[actionID]
	x, y := action.selectAnchor.X+4, action.selectAnchor.Y+4
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, x, y)); err != nil {
		t.Fatal(err)
	}
	action = app.inputActions[actionID]
	if action.selectOverlay.Popup.Height <= 0 || action.selectOverlay.Popup.Y+action.selectOverlay.Popup.Height <= 42 {
		t.Fatalf("overlay did not escape Scroll viewport: %+v", action.selectOverlay.Popup)
	}
	foundOverlay := false
	for _, command := range app.currentDisplay() {
		if command.Kind == paint.CommandItem && command.NodeID == actionID && command.ZIndex == math.MaxInt && command.Bounds == action.selectOverlay.Popup {
			foundOverlay = true
			if command.ClipSet {
				t.Fatal("window overlay inherited ancestor Scroll clip")
			}
		}
	}
	if !foundOverlay {
		t.Fatal("window-level overlay item was not appended above normal z-index content")
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 100, 60))
	app.handleEvent(appPointer(platform.EventMouseUp, 100, 60))
	if presses != 0 {
		t.Fatalf("outside close activated underlying button %d times", presses)
	}
	if findInstance(app.retained.Root(), actionID).State.SelectOpen {
		t.Fatal("outside click did not close Select")
	}
}

func TestSelectBoundaryStateBackgroundsFollowPopupCorners(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  paint.Radii
	}{
		{name: "first option", value: "a", want: paint.Radii{TopLeft: 8, TopRight: 8}},
		{name: "last option", value: "d", want: paint.Radii{BottomRight: 8, BottomLeft: 8}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 140, Height: 220})
			app.root = func() View {
				return Box(BoxProps{}, selectNode("select", test.value, selectOptions(), nil))
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			id := openFocusedSelect(t, app)
			hover, err := app.theme.color(TokenColor(ColorSemanticSelectHover))
			if err != nil {
				t.Fatal(err)
			}
			selected, err := app.theme.color(TokenColor(ColorSemanticSelectSelected))
			if err != nil {
				t.Fatal(err)
			}
			stateColors := map[paint.Color]bool{paintColor(hover): true, paintColor(selected): true}
			stateCount := 0
			lastStateIndex := -1
			borderIndex := -1
			for index, command := range app.currentDisplay() {
				if command.NodeID != id {
					continue
				}
				if command.Kind == paint.CommandFillRoundedRect && stateColors[command.Color] {
					stateCount++
					lastStateIndex = index
					if command.Radii != test.want {
						t.Fatalf("state background radii = %+v, want %+v", command.Radii, test.want)
					}
				}
				if command.Kind == paint.CommandStrokeRoundedRect {
					borderIndex = index
				}
			}
			if stateCount != 2 {
				t.Fatalf("selected and hover background count = %d, want 2", stateCount)
			}
			if borderIndex <= lastStateIndex {
				t.Fatalf("popup border index = %d, want after state backgrounds ending at %d", borderIndex, lastStateIndex)
			}
		})
	}
}

func TestSelectScrolledStateBackgroundRoundsVisiblePopupEdges(t *testing.T) {
	popup := paint.Rect{X: 10, Y: 20, Width: 80, Height: 100}
	tests := []struct {
		name string
		item paint.Rect
		want paint.Radii
	}{
		{name: "clipped top row", item: paint.Rect{X: 10, Y: 4, Width: 80, Height: 36}, want: paint.Radii{TopLeft: 8, TopRight: 8}},
		{name: "clipped bottom row", item: paint.Rect{X: 10, Y: 110, Width: 80, Height: 36}, want: paint.Radii{BottomRight: 8, BottomLeft: 8}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			background, radii := selectOptionBackground(test.item, popup, 8)
			if background.Y < popup.Y || background.Y+background.Height > popup.Y+popup.Height {
				t.Fatalf("background %+v exceeds popup %+v", background, popup)
			}
			if radii != test.want {
				t.Fatalf("radii = %+v, want %+v", radii, test.want)
			}
		})
	}
}

func TestSelectKeyboardFocusSelectionEscapeAndTab(t *testing.T) {
	value := "a"
	app := NewApp(AppOptions{Width: 140, Height: 160})
	app.root = func() View {
		return Box(BoxProps{},
			selectNode("select", value, selectOptions(), func(change string) { value = change }),
			Button(ButtonProps{Key: "after"}, Text(TextProps{Value: "after"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := openFocusedSelect(t, app)
	if app.interaction.Focused() != id {
		t.Fatal("opening Select moved focus away from its anchor")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
	if active := findInstance(app.retained.Root(), id).State.SelectActive; active != 2 {
		t.Fatalf("Down active = %d, want 2 after skipping disabled option", active)
	}
	if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false)); err != nil {
		t.Fatal(err)
	}
	if value != "c" || findInstance(app.retained.Root(), id).State.SelectOpen {
		t.Fatalf("keyboard selection value/open = %q/%v", value, findInstance(app.retained.Root(), id).State.SelectOpen)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEscape, false))
	if app.interaction.Focused() != id || findInstance(app.retained.Root(), id).State.SelectOpen {
		t.Fatal("Escape did not close and restore anchor focus")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeySpace, false))
	app.handleEvent(appKey(platform.EventKeyUp, platform.KeySpace, false))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() == id || findInstance(app.retained.Root(), id).State.SelectOpen {
		t.Fatal("Tab did not close Select and advance focus")
	}
}

func TestSelectMouseChoosesEnabledOption(t *testing.T) {
	value := "a"
	app := NewApp(AppOptions{Width: 140, Height: 180})
	app.root = func() View {
		return Box(BoxProps{}, selectNode("select", value, selectOptions(), func(change string) { value = change }))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := selectActionID(app)
	anchor := app.inputActions[id].selectAnchor
	app.handleEvent(appPointer(platform.EventMouseDown, anchor.X+4, anchor.Y+4))
	app.handleEvent(appPointer(platform.EventMouseUp, anchor.X+4, anchor.Y+4))
	action := app.inputActions[id]
	x := action.selectOverlay.Popup.X + 8
	y := action.selectOverlay.Popup.Y + action.selectOverlay.ItemHeight*2.5 - findInstance(app.retained.Root(), id).State.SelectScroll
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, x, y)); err != nil {
		t.Fatal(err)
	}
	if value != "c" || findInstance(app.retained.Root(), id).State.SelectOpen {
		t.Fatalf("mouse selection value/open = %q/%v", value, findInstance(app.retained.Root(), id).State.SelectOpen)
	}
}

func TestSelectOpenThemeSwitchesLightAndDark(t *testing.T) {
	app := NewApp(AppOptions{Width: 140, Height: 120})
	app.root = func() View {
		return Box(BoxProps{}, selectNode("select", "a", selectOptions(), nil))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	openFocusedSelect(t, app)
	for _, theme := range []Theme{DarkTheme(), LightTheme()} {
		if err := app.SetTheme(theme); err != nil {
			t.Fatal(err)
		}
		if dirty, err := app.runQueuedWork(); err != nil || !dirty {
			t.Fatalf("theme switch dirty/error = %v/%v", dirty, err)
		}
		if id := selectActionID(app); id == 0 || app.inputActions[id].selectOverlay.Popup.Height <= 0 {
			t.Fatal("theme switch dropped open Select overlay")
		}
	}
}

func TestSelectDynamicDeletionRemovalAndResizeFlip(t *testing.T) {
	value := "c"
	options := selectOptions()
	show := true
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View {
		children := make([]View, 0, 2)
		if show {
			children = append(children, Select(SelectProps{Key: "select", Style: Style{Position: PositionAbsolute, Insets: Insets{Top: Px(72)}, Width: Px(90), Height: Px(24)}, Value: value, Options: options, Placeholder: "gone"}))
		}
		children = append(children, Button(ButtonProps{Key: "after"}, Text(TextProps{Value: "after"})))
		return Box(BoxProps{}, children...)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := openFocusedSelect(t, app)
	if !app.inputActions[id].selectOverlay.Flipped {
		t.Fatal("bottom-edge Select did not flip above")
	}
	if err := app.relayout(100, 240); err != nil {
		t.Fatal(err)
	}
	if app.inputActions[id].selectOverlay.Flipped {
		t.Fatal("resized window did not recompute Select below placement")
	}
	options = options[:2]
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if node := findInstance(app.retained.Root(), id); node == nil || node.State.SelectOpen || node.State.SelectValueValid {
		t.Fatalf("deleted controlled option state = %+v", node)
	}
	openFocusedSelect(t, app)
	show = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if _, live := app.inputActions[id]; live || app.interaction.Focused() == id {
		t.Fatal("removed Select retained overlay action or focus")
	}
}

func TestSelectActiveOptionUsesValueIdentityAcrossReorder(t *testing.T) {
	options := selectOptions()
	app := NewApp(AppOptions{Width: 140, Height: 160})
	app.root = func() View {
		return Box(BoxProps{}, selectNode("select", "a", options, nil))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := openFocusedSelect(t, app)
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
	if key := findInstance(app.retained.Root(), id).State.SelectActiveKey; key != "c" {
		t.Fatalf("active key before reorder = %q", key)
	}
	options = []SelectOption{options[2], options[0], options[1], options[3]}
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	node := findInstance(app.retained.Root(), id)
	if node.State.SelectActiveKey != "c" || node.State.SelectActive != 0 {
		t.Fatalf("active key/index after reorder = %q/%d", node.State.SelectActiveKey, node.State.SelectActive)
	}
}
