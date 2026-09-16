package dxui

import (
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

func fixedOverlayContent(label string) View {
	return Button(ButtonProps{Style: Style{Width: Px(80), Height: Px(32)}}, Text(TextProps{Value: label}))
}

func overlayActionID(app *App, popover bool) uint64 {
	for id, action := range app.inputActions {
		if popover && action.popover != nil || !popover && action.tooltip != nil {
			return id
		}
	}
	return 0
}

func TestOverlayPlacementFlipsAlignsAndClamps(t *testing.T) {
	window := layout.Rect{Width: 100, Height: 80}
	tests := []struct {
		name      string
		anchor    layout.Rect
		desired   layout.Size
		placement OverlayPlacement
		want      paint.Rect
		flipped   bool
	}{
		{name: "bottom start", anchor: layout.Rect{X: 10, Y: 10, Width: 20, Height: 10}, desired: layout.Size{Width: 30, Height: 20}, placement: OverlayBottomStart, want: paint.Rect{X: 10, Y: 24, Width: 30, Height: 20}},
		{name: "bottom end flips", anchor: layout.Rect{X: 70, Y: 65, Width: 20, Height: 10}, desired: layout.Size{Width: 30, Height: 30}, placement: OverlayBottomEnd, want: paint.Rect{X: 60, Y: 31, Width: 30, Height: 30}, flipped: true},
		{name: "right flips and clamps cross axis", anchor: layout.Rect{X: 85, Y: 70, Width: 10, Height: 10}, desired: layout.Size{Width: 30, Height: 30}, placement: OverlayRight, want: paint.Rect{X: 51, Y: 50, Width: 30, Height: 30}, flipped: true},
		{name: "window bounds", anchor: layout.Rect{X: -20, Y: 10, Width: 5, Height: 5}, desired: layout.Size{Width: 140, Height: 20}, placement: OverlayBottom, want: paint.Rect{X: 0, Y: 19, Width: 100, Height: 20}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := placeOverlay(test.anchor, test.desired, window, test.placement, 4)
			if got.Popup != test.want || got.Flipped != test.flipped {
				t.Fatalf("placement = %+v, want rect %+v flipped %v", got, test.want, test.flipped)
			}
		})
	}
}

func TestPopoverEscapesScrollAndSupportsContentFocusAndDismissal(t *testing.T) {
	open := true
	contentPresses := 0
	proposals := make([]bool, 0, 2)
	app := NewApp(AppOptions{Width: 140, Height: 100})
	app.root = func() View {
		popover := Popover(PopoverProps{
			Key: "pop", Style: Style{Width: Px(70), Height: Px(30)}, Open: open,
			OnOpenChange: func(next bool) { proposals = append(proposals, next); open = next },
		}, Text(TextProps{Value: "anchor"}), Button(ButtonProps{OnPress: func() { contentPresses++ }}, Text(TextProps{Value: "inside"})))
		return Box(BoxProps{}, Scroll(ScrollProps{Style: Style{Width: Px(80), Height: Px(34)}, Axis: ScrollVertical}, popover),
			Button(ButtonProps{Key: "after"}, Text(TextProps{Value: "after"})))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := overlayActionID(app, true)
	action := app.inputActions[id]
	if action.overlayPopup.Height <= 0 || action.overlayPopup.Y+action.overlayPopup.Height <= 34 {
		t.Fatalf("popover did not escape Scroll clip: %+v", action.overlayPopup)
	}
	if !app.interaction.Focus(id, true) {
		t.Fatal("could not focus popover anchor")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() == id {
		t.Fatal("second Tab did not enter interactive popover content")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false))
	if _, _, err := app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false)); err != nil {
		t.Fatal(err)
	}
	if contentPresses != 1 {
		t.Fatalf("content presses = %d", contentPresses)
	}
	if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEscape, false)); err != nil {
		t.Fatal(err)
	}
	if open || len(proposals) != 1 || proposals[0] || app.interaction.Focused() != id {
		t.Fatalf("Escape state proposals/open/focus = %v/%v/%d", proposals, open, app.interaction.Focused())
	}

	open = true
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseDown, 130, 90))
	app.handleEvent(appPointer(platform.EventMouseUp, 130, 90))
	if open || len(proposals) != 2 || proposals[1] {
		t.Fatalf("outside dismissal = proposals %v open %v", proposals, open)
	}
}

func TestPopoverAnchorActivationIsControlled(t *testing.T) {
	open := false
	changes := 0
	app := NewApp(AppOptions{Width: 120, Height: 100})
	app.root = func() View {
		return Popover(PopoverProps{Open: open, OnOpenChange: func(next bool) { changes++; open = next }},
			Text(TextProps{Style: Style{Width: Px(50), Height: Px(24)}, Value: "anchor"}), fixedOverlayContent("content"))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := overlayActionID(app, true)
	anchor := app.inputActions[id].overlayAnchor
	app.handleEvent(appPointer(platform.EventMouseDown, anchor.X+2, anchor.Y+2))
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, anchor.X+2, anchor.Y+2)); err != nil {
		t.Fatal(err)
	}
	if !open || changes != 1 || app.inputActions[id].overlayPopup.Height <= 0 {
		t.Fatalf("activation open/changes/popup = %v/%d/%+v", open, changes, app.inputActions[id].overlayPopup)
	}
}

func TestPopoverDismissalOnlyAffectsDeterministicTopLayer(t *testing.T) {
	first, second := true, true
	firstChanges, secondChanges := 0, 0
	app := NewApp(AppOptions{Width: 180, Height: 120})
	app.root = func() View {
		return Box(BoxProps{Direction: Horizontal, Gap: 20},
			Popover(PopoverProps{Key: "first", Open: first, OnOpenChange: func(next bool) { firstChanges++; first = next }},
				Text(TextProps{Style: Style{Width: Px(40), Height: Px(24)}, Value: "one"}), fixedOverlayContent("one")),
			Popover(PopoverProps{Key: "second", Open: second, OnOpenChange: func(next bool) { secondChanges++; second = next }},
				Text(TextProps{Style: Style{Width: Px(40), Height: Px(24)}, Value: "two"}), fixedOverlayContent("two")),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEscape, false)); err != nil {
		t.Fatal(err)
	}
	if !first || second || firstChanges != 0 || secondChanges != 1 {
		t.Fatalf("top Escape result = first %v/%d second %v/%d", first, firstChanges, second, secondChanges)
	}
}

func TestPopoverExternallyControlledCloseRestoresFocusedContentToAnchor(t *testing.T) {
	open := true
	app := NewApp(AppOptions{Width: 140, Height: 100})
	app.root = func() View {
		return Popover(PopoverProps{Key: "pop", Open: open},
			Text(TextProps{Style: Style{Width: Px(50), Height: Px(24)}, Value: "anchor"}),
			fixedOverlayContent("content"))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	hostID := overlayActionID(app, true)
	app.interaction.Focus(hostID, true)
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if app.interaction.Focused() == hostID {
		t.Fatal("Tab did not focus popover content")
	}
	open = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.interaction.Focused() != hostID {
		t.Fatalf("controlled close focus = %d, want anchor %d", app.interaction.Focused(), hostID)
	}
}

func TestPopoverTracksControlledScrollWithoutRelayout(t *testing.T) {
	offset := Point{}
	app := NewApp(AppOptions{Width: 160, Height: 180})
	app.root = func() View {
		return Box(BoxProps{}, Scroll(ScrollProps{Style: Style{Width: Px(120), Height: Px(80), Shrink: Some(float32(0))}, Axis: ScrollVertical, Offset: Some(offset)},
			Box(BoxProps{Style: Style{Height: Px(174), Shrink: Some(float32(0))}},
				Text(TextProps{Style: Style{Height: Px(50)}, Value: "spacer"}),
				Popover(PopoverProps{Key: "pop", Open: true},
					Text(TextProps{Style: Style{Width: Px(60), Height: Px(24)}, Value: "anchor"}), fixedOverlayContent("content")),
				Text(TextProps{Style: Style{Height: Px(100)}, Value: "tail"}),
			)))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := overlayActionID(app, true)
	beforeAction := app.inputActions[id]
	before := beforeAction.overlayPopup.Y
	layouts := app.Diagnostics().LayoutCount
	offset.Y = 20
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	afterAction := app.inputActions[id]
	after := afterAction.overlayPopup.Y
	if before-after != 20 || app.Diagnostics().LayoutCount != layouts {
		t.Fatalf("scroll tracking before/after/layout = %g/%g anchors=%+v/%+v %d->%d", before, after, beforeAction.overlayAnchor, afterAction.overlayAnchor, layouts, app.Diagnostics().LayoutCount)
	}
}

func TestPopoverTracksLogicalResizeAndDPIEvents(t *testing.T) {
	app := NewApp(AppOptions{Width: 120, Height: 80})
	app.root = func() View {
		return Box(BoxProps{},
			Text(TextProps{Style: Style{Height: Px(50), Shrink: Some(float32(0))}, Value: "spacer"}),
			Popover(PopoverProps{Key: "pop", Open: true},
				Text(TextProps{Style: Style{Width: Px(50), Height: Px(24)}, Value: "anchor"}), fixedOverlayContent("content")),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := overlayActionID(app, true)
	before := app.inputActions[id]
	if before.overlayPopup.Y >= before.overlayAnchor.Y {
		t.Fatalf("small-window popup did not flip above anchor: anchor=%+v popup=%+v", before.overlayAnchor, before.overlayPopup)
	}
	if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventScale, Viewport: platform.Viewport{
		LogicalWidth: 120, LogicalHeight: 80, PixelWidth: 240, PixelHeight: 160, PixelDensity: 2, DisplayScale: 2,
	}}); err != nil {
		t.Fatal(err)
	}
	afterScale := app.inputActions[id]
	if afterScale.overlayAnchor != before.overlayAnchor || afterScale.overlayPopup != before.overlayPopup {
		t.Fatalf("DPI event changed logical overlay geometry: before=%+v/%+v after=%+v/%+v", before.overlayAnchor, before.overlayPopup, afterScale.overlayAnchor, afterScale.overlayPopup)
	}
	if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventResize, Viewport: platform.Viewport{
		LogicalWidth: 120, LogicalHeight: 140, PixelWidth: 240, PixelHeight: 280, PixelDensity: 2, DisplayScale: 2,
	}}); err != nil {
		t.Fatal(err)
	}
	afterResize := app.inputActions[id]
	if afterResize.overlayPopup.Y <= afterResize.overlayAnchor.Y+afterResize.overlayAnchor.Height || afterResize.overlayPopup.Y+afterResize.overlayPopup.Height > 140 {
		t.Fatalf("resized popup did not return below anchor inside window: anchor=%+v popup=%+v", afterResize.overlayAnchor, afterResize.overlayPopup)
	}
}

func TestTooltipUsesDeadlineFocusLeaveDisableAndUnmount(t *testing.T) {
	clock := &editorClock{now: time.Unix(10, 0)}
	show, disabled := true, false
	anchorPresses := 0
	app := NewApp(AppOptions{Width: 120, Height: 80})
	app.clock = clock
	app.root = func() View {
		if !show {
			return Box(BoxProps{})
		}
		return Box(BoxProps{},
			Tooltip(TooltipProps{Key: "tip", Delay: 200 * time.Millisecond, Disabled: disabled},
				Button(ButtonProps{Style: Style{Width: Px(60), Height: Px(24)}, OnPress: func() { anchorPresses++ }}, Text(TextProps{Value: "anchor"})),
				Text(TextProps{Value: "hint"})),
			Button(ButtonProps{Key: "after"}, Text(TextProps{Value: "after"})),
		)
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := overlayActionID(app, false)
	anchor := app.inputActions[id].overlayAnchor
	app.handleEvent(appPointer(platform.EventMouseMove, anchor.X+2, anchor.Y+2))
	deadline := app.nextDeadline(clock.now)
	if deadline == nil || !deadline.Equal(clock.now.Add(200*time.Millisecond)) {
		t.Fatalf("tooltip deadline = %v", deadline)
	}
	if findInstance(app.retained.Root(), id).State.TooltipOpen {
		t.Fatal("tooltip opened before its deadline")
	}
	clock.now = *deadline
	if dirty, _, err := app.handleEvent(platform.Event{Kind: platform.EventDeadline}); err != nil || !dirty {
		t.Fatalf("deadline dirty/error = %v/%v", dirty, err)
	}
	if !findInstance(app.retained.Root(), id).State.TooltipOpen || app.inputActions[id].overlayPopup.Height <= 0 {
		t.Fatal("tooltip deadline did not project content")
	}
	app.handleEvent(appPointer(platform.EventMouseDown, anchor.X+2, anchor.Y+2))
	app.handleEvent(appPointer(platform.EventMouseUp, anchor.X+2, anchor.Y+2))
	if anchorPresses != 1 {
		t.Fatalf("visible tooltip blocked anchor activation: %d", anchorPresses)
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 119, 79))
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	if findInstance(app.retained.Root(), id).State.TooltipOpen {
		t.Fatal("tooltip stayed open after hover and focus left")
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, true))
	if app.nextDeadline(clock.now) == nil {
		t.Fatal("keyboard focus did not start tooltip delay")
	}
	disabled = true
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if node := findInstance(app.retained.Root(), id); node.State.TooltipOpen || node.State.TooltipDueNS != 0 {
		t.Fatalf("disabled tooltip retained state: %+v", node.State)
	}
	show = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if _, live := app.inputActions[id]; live || app.nextTooltipDeadline() != nil {
		t.Fatal("unmounted tooltip retained action or deadline")
	}
}

func TestTooltipPaintDoesNotEnterInteractionSnapshot(t *testing.T) {
	app := NewApp(AppOptions{Width: 120, Height: 80})
	app.root = func() View {
		return Tooltip(TooltipProps{}, fixedOverlayContent("anchor"), fixedOverlayContent("hint"))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	id := overlayActionID(app, false)
	node := findInstance(app.retained.Root(), id)
	node.State.TooltipOpen = true
	if err := app.redisplayCurrent(); err != nil {
		t.Fatal(err)
	}
	if _, err := app.refreshInteraction(app.view, app.retained.Root(), app.geometry, app.theme); err != nil {
		t.Fatal(err)
	}
	contentID := node.Children[1].Children[0].ID
	if _, exists := app.inputActions[contentID]; exists {
		t.Fatal("tooltip content blocked input or entered focus order")
	}
}

func TestOverlaySurfacesUseThemedPaddingBorderRadiusAndShadow(t *testing.T) {
	for _, test := range []struct {
		name string
		view View
	}{
		{name: "popover", view: Popover(PopoverProps{Open: true}, Text(TextProps{Value: "anchor"}), Text(TextProps{Value: "content"}))},
		{name: "tooltip", view: Tooltip(TooltipProps{}, Text(TextProps{Value: "anchor"}), Text(TextProps{Value: "content"}))},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 160, Height: 100})
			app.root = func() View { return test.view }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			host := app.retained.Root()
			if test.name == "tooltip" {
				host.State.TooltipOpen = true
				if err := app.redisplayCurrent(); err != nil {
					t.Fatal(err)
				}
			}
			surfaceID := host.Children[1].ID
			var shadow, fill, border bool
			for _, command := range app.currentDisplay() {
				if command.NodeID != surfaceID {
					continue
				}
				shadow = shadow || command.Kind == paint.CommandDrawShadow
				fill = fill || command.Kind == paint.CommandFillRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect
				border = border || command.Kind == paint.CommandStrokeRoundedRect || command.Kind == paint.CommandFillStrokeRoundedRect
			}
			if !shadow || !fill || !border {
				t.Fatalf("surface commands shadow/fill/border = %v/%v/%v", shadow, fill, border)
			}
		})
	}
}
