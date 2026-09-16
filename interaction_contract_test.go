package dxui

import (
	"testing"

	"github.com/dxui-org/dxui/internal/platform"
)

func contractEvent(t *testing.T, app *App, event platform.Event) {
	t.Helper()
	if _, _, err := app.handleEvent(event); err != nil {
		t.Fatal(err)
	}
}

func TestEditorEligibilityTransitions(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		for _, mode := range []string{"readonly", "nil-change", "disabled", "blur", "window-loss", "hidden", "removed", "replaced"} {
			name := "Input/" + mode
			if multiline {
				name = "Textarea/" + mode
			}
			t.Run(name, func(t *testing.T) {
				value, restricted, callbacks := "A?B", false, 0
				app, native := testEditorApp(t, func() View {
					if restricted && mode == "removed" {
						return Label("removed")
					}
					style := Style{Width: Px(180), Height: Px(50)}
					if restricted && mode == "hidden" {
						style.Visibility = Hidden
					}
					key := "editor"
					if restricted && mode == "replaced" {
						key = "replacement"
					}
					change := func(string) { callbacks++ }
					if restricted && mode == "nil-change" {
						change = nil
					}
					var editor View
					if multiline {
						editor = Textarea(TextareaProps{Key: key, Style: style, Value: value, OnChange: change, ReadOnly: restricted && mode == "readonly", Disabled: restricted && mode == "disabled"})
					} else {
						editor = Input(InputProps{Key: key, Style: style, Value: value, OnChange: change, ReadOnly: restricted && mode == "readonly", Disabled: restricted && mode == "disabled"})
					}
					return Box(BoxProps{}, editor, TextButton(ButtonProps{}, "next"))
				})
				contractEvent(t, app, appPointer(platform.EventMouseDown, 20, 15))
				id := app.interaction.Focused()
				editor := app.editors[id]
				if editor == nil || app.dragEditor != id {
					t.Fatal("missing editor drag")
				}
				contractEvent(t, app, platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: "??"}})
				if !editor.state.Composition.Active {
					t.Fatal("editable pre-edit missing")
				}
				restricted = true
				switch mode {
				case "blur":
					contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyTab, false))
				case "window-loss":
					contractEvent(t, app, platform.Event{Kind: platform.EventWindowFocusLost})
				default:
					if _, err := app.buildAndCommit(); err != nil {
						t.Fatal(err)
					}
				}
				if mode != "removed" && mode != "replaced" && editor.state.Composition.Active {
					t.Fatal("composition survived restriction")
				}
				selectable := mode == "readonly" || mode == "nil-change"
				if selectable {
					if app.interaction.Focused() != id || app.dragEditor != id {
						t.Fatal("read-only transition lost selection/focus")
					}
					contractEvent(t, app, platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: "late"}})
					contractEvent(t, app, platform.Event{Kind: platform.EventTextInput, Text: platform.TextEvent{Text: "late"}})
					if editor.state.Composition.Active || callbacks != 0 {
						t.Fatal("restricted editor accepted mutation/pre-edit")
					}
					contractEvent(t, app, appPointer(platform.EventMouseUp, 20, 15))
					contractEvent(t, app, platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyA, Modifiers: platform.Modifiers{Primary: true}}})
					contractEvent(t, app, platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyC, Modifiers: platform.Modifiers{Primary: true}}})
					if native.clipboard != value {
						t.Fatalf("copy = %q", native.clipboard)
					}
				} else {
					if app.dragEditor != 0 {
						t.Fatal("ineligible editor retained drag")
					}
					selection := editor.state.Selection
					contractEvent(t, app, appPointer(platform.EventMouseMove, 150, 20))
					if editor.state.Selection != selection {
						t.Fatal("canceled drag changed selection")
					}
				}
			})
		}
	}
}

func TestNilCallbackDoesNotDisableControls(t *testing.T) {
	cases := []struct {
		name  string
		build func(bool) View
	}{
		{"Button", func(d bool) View { return TextButton(ButtonProps{Disabled: d}, "button") }},
		{"Checkbox", func(d bool) View { return Checkbox(CheckboxProps{Disabled: d}, Label("check")) }},
		{"ToggleSwitch", func(d bool) View { return ToggleSwitch(ToggleSwitchProps{Disabled: d}) }},
		{"Radio", func(d bool) View { return Radio(RadioProps{Disabled: d}, Label("radio")) }},
		{"Slider", func(d bool) View { return Slider(SliderProps{Disabled: d}) }},
		{"Select", func(d bool) View { return Select(SelectProps{Disabled: d, Options: selectOptions()}) }},
		{"Tabs", func(d bool) View { return Tabs(TabsProps{Disabled: d, Items: tabItems()}) }},
		{"Menu", func(d bool) View { return Menu(MenuProps{Disabled: d, Items: menuItems()}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			disabled := false
			app, _ := testEditorApp(t, func() View { return tc.build(disabled) })
			focusEditor(t, app)
			id := app.interaction.Focused()
			if id == 0 {
				t.Fatal("nil callback removed focus stop")
			}
			contractEvent(t, app, appPointer(platform.EventMouseDown, 5, 5))
			disabled = true
			if _, err := app.buildAndCommit(); err != nil {
				t.Fatal(err)
			}
			if app.interaction.Focused() != 0 {
				t.Fatal("disabled kept focus")
			}
			node := findInstance(app.retained.Root(), id)
			if node.State.Pressed || node.State.Hovered || node.State.Focused {
				t.Fatal("disabled kept active state")
			}
			if app.sliderDrag.id != 0 || app.tabsCapture.id != 0 || app.menuCapture.id != 0 {
				t.Fatal("disabled kept specialized capture")
			}
			disabled = false
			if _, err := app.buildAndCommit(); err != nil {
				t.Fatal(err)
			}
			contractEvent(t, app, appPointer(platform.EventMouseUp, 5, 5))
			if node.State.Pressed {
				t.Fatal("reenable revived press")
			}
		})
	}
}

func TestNilCallbackNavigationAndScroll(t *testing.T) {
	t.Run("Tabs", func(t *testing.T) {
		app, _ := testEditorApp(t, func() View { return Tabs(TabsProps{Value: "a", Items: tabItems()}) })
		focusEditor(t, app)
		contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEnd, false))
		node := app.retained.Root()
		if node.State.TabsActive != len(tabItems())-1 {
			t.Fatal("nil callback blocked tab navigation")
		}
		contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEnter, false))
		contractEvent(t, app, appKey(platform.EventKeyUp, platform.KeyEnter, false))
		if app.view.node.tabs.Value != "a" {
			t.Fatal("nil callback changed Tabs value")
		}
	})
	t.Run("Menu", func(t *testing.T) {
		app, _ := testEditorApp(t, func() View { return Menu(MenuProps{Items: menuItems()}) })
		focusEditor(t, app)
		contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEnd, false))
		if app.retained.Root().State.MenuActive != len(menuItems())-1 {
			t.Fatal("nil callback blocked menu navigation")
		}
	})
	for _, controlled := range []bool{false, true} {
		name := "uncontrolled"
		if controlled {
			name = "controlled"
		}
		t.Run("Scroll/"+name, func(t *testing.T) {
			props := ScrollProps{Style: Style{Width: Px(100), Height: Px(60)}}
			if controlled {
				props.Offset = Some(Point{})
			}
			app, _ := testEditorApp(t, func() View { return Scroll(props, Box(BoxProps{Style: Style{Height: Px(300)}})) })
			focusEditor(t, app)
			contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyDown, false))
			moved := app.retained.Root().State.ScrollY > 0
			if moved == controlled {
				t.Fatalf("controlled=%v moved=%v", controlled, moved)
			}
		})
	}
}

func TestReadOnlyInputSubmitIsIndependent(t *testing.T) {
	calls := 0
	app, _ := testEditorApp(t, func() View { return Input(InputProps{Value: "fixed", ReadOnly: true, OnSubmit: func() { calls++ }}) })
	focusEditor(t, app)
	contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEnter, false))
	contractEvent(t, app, appKey(platform.EventKeyUp, platform.KeyEnter, false))
	if calls != 1 {
		t.Fatalf("submit calls=%d", calls)
	}
}

func TestSelectNilCallbackAndCanceledPopupCapture(t *testing.T) {
	disabled := false
	app, _ := testEditorApp(t, func() View {
		return Box(BoxProps{}, Select(SelectProps{Style: Style{Width: Px(100), Height: Px(32)}, Value: "a", Options: selectOptions(), Disabled: disabled}))
	})
	id := openFocusedSelect(t, app)
	contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEnd, false))
	if findInstance(app.retained.Root(), id).State.SelectActive != len(selectOptions())-1 {
		t.Fatal("nil callback blocked select navigation")
	}
	contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEnter, false))
	if findInstance(app.retained.Root(), id).State.SelectOpen || app.view.node.children[0].node.selectp.Value != "a" {
		t.Fatal("nil confirm must close without changing Value")
	}
	openFocusedSelect(t, app)
	popup := app.inputActions[id].selectOverlay
	x, y := popup.Popup.X+10, popup.Popup.Y+10
	contractEvent(t, app, appPointer(platform.EventMouseDown, x, y))
	if app.selectCapture.id != id {
		t.Fatalf("missing popup capture: %+v popup=%+v open=%v", app.selectCapture, popup, findInstance(app.retained.Root(), id).State.SelectOpen)
	}
	disabled = true
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.selectCapture.id != 0 || findInstance(app.retained.Root(), id).State.SelectOpen {
		t.Fatal("disable retained popup capture/open")
	}
	disabled = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	openFocusedSelect(t, app)
	contractEvent(t, app, appPointer(platform.EventMouseUp, x, y))
	if !findInstance(app.retained.Root(), id).State.SelectOpen {
		t.Fatal("stale release activated reopened popup")
	}
}

func TestPopoverNilCallbackKeepsControlledOpenAndContent(t *testing.T) {
	open, calls := true, 0
	app, _ := testEditorApp(t, func() View {
		return Popover(PopoverProps{Open: open}, Label("trigger"), TextButton(ButtonProps{OnPress: func() { calls++ }}, "content"))
	})
	focusEditor(t, app)
	trigger := app.interaction.Focused()
	focusEditor(t, app)
	if app.interaction.Focused() == trigger {
		t.Fatal("nil callback blocked content focus")
	}
	contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEnter, false))
	contractEvent(t, app, appKey(platform.EventKeyUp, platform.KeyEnter, false))
	if calls != 1 {
		t.Fatal("nil callback blocked content activation")
	}
	contractEvent(t, app, appKey(platform.EventKeyDown, platform.KeyEscape, false))
	if !app.view.node.popover.Open || app.interaction.Focused() != trigger {
		t.Fatal("nil dismissal changed Open or failed focus restoration")
	}
	open = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	focusEditor(t, app)
	if app.interaction.Focused() != trigger {
		t.Fatal("closed content remained focusable")
	}
}

func TestDisabledEditorTransfersNativeCaretOwner(t *testing.T) {
	disabled := false
	app, _ := testEditorApp(t, func() View {
		return Box(BoxProps{},
			Input(InputProps{Key: "first", Disabled: disabled, Value: "one", OnChange: func(string) {}}),
			Input(InputProps{Key: "second", Value: "two", OnChange: func(string) {}}))
	})
	focusEditor(t, app)
	old := app.interaction.Focused()
	disabled = true
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	next := app.interaction.Focused()
	if next == 0 || next == old || app.focusedEditor != next || !app.nativeTextActive {
		t.Fatal("native caret owner did not follow transferred focus")
	}
}
