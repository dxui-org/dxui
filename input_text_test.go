package dxui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dxui-org/dxui/internal/backend"
	internalinput "github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
	"github.com/dxui-org/dxui/internal/renderer"
	uiruntime "github.com/dxui-org/dxui/internal/runtime"
)

type editorNative struct {
	started, stopped, cleared int
	area                      backend.TextInputArea
	clipboard                 string
	clipboardErr              error
}

func (n *editorNative) StartTextInput() error                             { n.started++; return nil }
func (n *editorNative) StopTextInput() error                              { n.stopped++; return nil }
func (n *editorNative) ClearComposition() error                           { n.cleared++; return nil }
func (n *editorNative) SetTextInputArea(area backend.TextInputArea) error { n.area = area; return nil }
func (n *editorNative) ClipboardText() (string, error)                    { return n.clipboard, nil }
func (n *editorNative) SetClipboardText(value string) error {
	if n.clipboardErr != nil {
		return n.clipboardErr
	}
	n.clipboard = value
	return nil
}

type editorRuntime struct {
	events   platform.EventSource
	render   renderer.Renderer
	text     *editorNative
	viewport platform.Viewport
}

func (r *editorRuntime) Events() platform.EventSource { return r.events }
func (r *editorRuntime) Renderer() renderer.Renderer  { return r.render }
func (r *editorRuntime) TextInput() backend.TextInput { return r.text }
func (r *editorRuntime) Diagnostics() backend.Diagnostics {
	viewport := r.viewport
	if viewport.LogicalWidth == 0 {
		viewport = platform.Viewport{LogicalWidth: 300, LogicalHeight: 120, PixelWidth: 300, PixelHeight: 120}
	}
	return backend.Diagnostics{Viewport: viewport}
}
func (r *editorRuntime) Close() error { return nil }

func testEditorApp(t *testing.T, builder func() View) (*App, *editorNative) {
	t.Helper()
	native := &editorNative{}
	app := NewApp(AppOptions{Width: 300, Height: 120})
	app.root = builder
	app.backend = &editorRuntime{text: native, events: &frameTestEvents{}, render: &frameTestRenderer{}}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	return app, native
}

func TestAppSetClipboardTextUsesNativeUIThreadPort(t *testing.T) {
	app := NewApp(AppOptions{})
	if err := app.SetClipboardText("Search"); !errors.Is(err, ErrAppNotRunning) {
		t.Fatalf("SetClipboardText before Run = %v, want ErrAppNotRunning", err)
	}

	native := &editorNative{}
	app.running = true
	app.backend = &editorRuntime{text: native}
	if err := app.SetClipboardText("Search"); err != nil {
		t.Fatal(err)
	}
	if native.clipboard != "Search" {
		t.Fatalf("clipboard = %q, want Search", native.clipboard)
	}
	if err := app.SetClipboardText("bad\xffname"); err != nil {
		t.Fatal(err)
	}
	if native.clipboard != "bad\uFFFDname" {
		t.Fatalf("normalized clipboard = %q, want replacement rune", native.clipboard)
	}

	native.clipboardErr = errors.New("native failure")
	if err := app.SetClipboardText("Search"); err == nil || !strings.Contains(err.Error(), "set clipboard text") {
		t.Fatalf("native failure = %v, want contextual error", err)
	}
	app.closed = true
	if err := app.SetClipboardText("Search"); !errors.Is(err, ErrAppClosed) {
		t.Fatalf("SetClipboardText after Close = %v, want ErrAppClosed", err)
	}
}

func focusEditor(t *testing.T, app *App) {
	t.Helper()
	if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyTab}}); err != nil {
		t.Fatal(err)
	}
}

func TestInputConsumesTextInputNotKeyDownAndKeepsRuneSelection(t *testing.T) {
	value := "A中"
	var changes []string
	app, native := testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(200), Height: Px(36)}, Value: value,
			OnChange: func(change string) { changes = append(changes, change); value = change }})
	})
	focusEditor(t, app)
	if native.started != 1 {
		t.Fatalf("StartTextInput calls = %d", native.started)
	}
	app.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyOther}})
	if value != "A中" {
		t.Fatal("KeyDown inserted text")
	}
	if _, _, err := app.handleEvent(platform.Event{Kind: platform.EventTextInput, Text: platform.TextEvent{Text: "文"}}); err != nil {
		t.Fatal(err)
	}
	if value != "A中文" || len(changes) != 1 || changes[0] != value {
		t.Fatalf("committed value/change = %q %#v", value, changes)
	}
	if native.area.Height <= 0 {
		t.Fatalf("candidate area = %+v", native.area)
	}
}

func TestInputSelectionPasteUndoRedoAndPasswordCopyPolicy(t *testing.T) {
	value := "A中B"
	selection := Some(TextRange{1, 2})
	app, native := testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(180), Height: Px(36)}, Value: value, Selection: selection,
			OnSelectionChange: func(r TextRange) { selection = Some(r) }, OnChange: func(change string) { value = change }})
	})
	focusEditor(t, app)
	native.clipboard = "文"
	key := func(k platform.Key, shift bool) {
		t.Helper()
		_, _, err := app.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: k, Modifiers: platform.Modifiers{Primary: true, Shift: shift}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	key(platform.KeyV, false)
	if value != "A文B" {
		t.Fatalf("paste = %q", value)
	}
	key(platform.KeyZ, false)
	if value != "A中B" {
		t.Fatalf("undo = %q", value)
	}
	key(platform.KeyY, false)
	if value != "A文B" {
		t.Fatalf("redo = %q", value)
	}

	password := "secret"
	passwordApp, passwordNative := testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(160), Height: Px(36)}, Value: password, Password: true, Selection: Some(TextRange{0, 6}), OnChange: func(string) {}})
	})
	focusEditor(t, passwordApp)
	passwordApp.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyC, Modifiers: platform.Modifiers{Primary: true}}})
	if passwordNative.clipboard != "" {
		t.Fatal("password copied plaintext")
	}
	for _, command := range passwordApp.currentDisplay() {
		if command.Text != nil && strings.Contains(command.Text.Key, password) {
			t.Fatal("password leaked into texture key")
		}
	}
}

func TestInputDoubleClickSelectsAllRunes(t *testing.T) {
	value := "A中B"
	selection := Some(TextRange{Start: 1, End: 1})
	var changes []TextRange
	app, _ := testEditorApp(t, func() View {
		return Input(InputProps{
			Style:     Style{Width: Px(180), Height: Px(36)},
			Value:     value,
			Selection: selection,
			OnSelectionChange: func(next TextRange) {
				changes = append(changes, next)
				selection = Some(next)
			},
		})
	})
	event := appPointer(platform.EventMouseDown, 20, 18)
	event.Pointer.Clicks = 2
	if _, _, err := app.handleEvent(event); err != nil {
		t.Fatal(err)
	}

	want := TextRange{Start: 0, End: 3}
	editor := app.editors[app.interaction.Focused()]
	if editor == nil || publicRange(editor.state.Selection) != want {
		t.Fatalf("double-click selection = %+v, want %+v", editor, want)
	}
	if len(changes) != 1 || changes[0] != want {
		t.Fatalf("selection callbacks = %+v, want [%+v]", changes, want)
	}
}

func TestCompositionUpdateCommitCancelAndExternalValueRule(t *testing.T) {
	value := "中"
	app, native := testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(180), Height: Px(36)}, Value: value,
			OnChange: func(change string) { value = change }})
	})
	focusEditor(t, app)
	app.handleEvent(platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: "候选", EditingStart: 1, EditingLength: 1}})
	editor := app.editors[app.interaction.Focused()]
	if editor == nil || editor.state.Composition.Text != "候选" || editor.state.Composition.Selection != (internalinput.Range{Start: 1, End: 2}) {
		t.Fatalf("composition = %+v", editor)
	}
	app.handleEvent(platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: ""}})
	if editor.state.Composition.Active {
		t.Fatal("empty editing event did not cancel")
	}
	app.handleEvent(platform.Event{Kind: platform.EventTextEditing, Text: platform.TextEvent{Text: "文", EditingStart: 0, EditingLength: 1}})
	value = "external"
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if editor.state.Composition.Active {
		t.Fatal("external Value change retained composition")
	}
	if native.area.Height <= 0 {
		t.Fatalf("composition area = %+v", native.area)
	}
}

func TestInputReadOnlyAndDisabledRejectMutation(t *testing.T) {
	for _, test := range []struct {
		name               string
		disabled, readOnly bool
	}{
		{name: "read-only", readOnly: true}, {name: "disabled", disabled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, changes := "fixed", 0
			app, native := testEditorApp(t, func() View {
				return Input(InputProps{Style: Style{Width: Px(120), Height: Px(30)}, Value: value,
					Disabled: test.disabled, ReadOnly: test.readOnly, OnChange: func(change string) { changes++; value = change }})
			})
			focusEditor(t, app)
			app.handleEvent(platform.Event{Kind: platform.EventTextInput, Text: platform.TextEvent{Text: "X"}})
			app.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyBackspace}})
			if value != "fixed" || changes != 0 {
				t.Fatalf("value/changes = %q/%d", value, changes)
			}
			if test.disabled && native.started != 0 {
				t.Fatal("disabled input started native text input")
			}
			if test.readOnly && native.started != 1 {
				t.Fatal("read-only input did not participate in focus/text selection")
			}
		})
	}
}

func TestInputHorizontalCaretScrollAndPasswordErrorsDoNotLeak(t *testing.T) {
	value := strings.Repeat("wide", 40)
	app, _ := testEditorApp(t, func() View {
		return Box(BoxProps{}, Input(InputProps{Style: Style{Width: Px(70), MaxWidth: Px(70)}, Value: value, OnChange: func(change string) { value = change }}))
	})
	focusEditor(t, app)
	editor := app.editors[app.interaction.Focused()]
	action := app.inputActions[app.interaction.Focused()]
	if err := app.ensureEditorCaretVisible(action, editor); err != nil {
		t.Fatal(err)
	}
	if editor.state.ScrollX <= 0 {
		t.Fatalf("horizontal scroll = %v", editor.state.ScrollX)
	}

	secret := "do-not-log-this-password"
	leakApp := NewApp(AppOptions{Width: 100, Height: 40})
	leakApp.root = func() View {
		return Input(InputProps{Value: secret, Password: true, Style: Style{Width: Px(-1)}})
	}
	err := leakApp.buildRoot()
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("password validation error = %v", err)
	}
}

func TestTextareaMouseDragNewlineAndScrollCaretVisibleAfterResize(t *testing.T) {
	value := "one\ntwo\nthree\nfour"
	app, native := testEditorApp(t, func() View {
		return Box(BoxProps{}, Textarea(TextareaProps{Style: Style{Width: Px(100), Height: Px(35), MaxHeight: Px(35), Shrink: Some(float32(0))}, Value: value, Wrap: TextNoWrap, OnChange: func(change string) { value = change }}))
	})
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseDown, 5, 5)); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(appPointer(platform.EventMouseMove, 60, 28))
	app.handleEvent(appPointer(platform.EventMouseUp, 60, 28))
	editor := app.editors[app.interaction.Focused()]
	if editor.state.Selection.Start == editor.state.Selection.End {
		t.Fatal("mouse drag did not select")
	}
	app.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyEnd}})
	app.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyEnter}})
	if !strings.Contains(value, "\n") {
		t.Fatalf("textarea enter = %q", value)
	}
	editor.state.Selection = internalinput.Range{Start: len([]rune(value)), End: len([]rune(value))}
	if err := app.ensureEditorCaretVisible(app.inputActions[app.interaction.Focused()], editor); err != nil {
		t.Fatal(err)
	}
	if editor.state.ScrollY <= 0 {
		action := app.inputActions[app.interaction.Focused()]
		layoutValue, _ := app.layoutEditor(action)
		t.Fatalf("vertical scroll = %v, content=%+v layout=%+v selection=%+v", editor.state.ScrollY, action.content, layoutValue, editor.state.Selection)
	}
	if err := app.relayout(80, 30); err != nil {
		t.Fatal(err)
	}
	if native.area.Height <= 0 {
		t.Fatalf("candidate area after resize = %+v", native.area)
	}
	if err := app.ensureEditorCaretVisible(app.inputActions[app.interaction.Focused()], editor); err != nil {
		t.Fatal(err)
	}
}

func TestTextareaWheelScrollClampsToContent(t *testing.T) {
	value := "one\ntwo\nthree\nfour"
	app, _ := testEditorApp(t, func() View {
		return Textarea(TextareaProps{
			Style: Style{Width: Px(100), Height: Px(35), MaxHeight: Px(35), Shrink: Some(float32(0))},
			Value: value,
		})
	})
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseDown, 5, 5)); err != nil {
		t.Fatal(err)
	}
	id := app.interaction.Focused()
	action := app.inputActions[id]
	layoutValue, err := app.layoutEditor(action)
	if err != nil {
		t.Fatal(err)
	}
	want := max(float32(0), layoutValue.height-action.content.Height)
	for range 10 {
		if _, _, err := app.handleEvent(wheel(5, 5, 0, -100)); err != nil {
			t.Fatal(err)
		}
	}
	if got := app.editors[id].state.ScrollY; got != want {
		t.Fatalf("scroll after repeated wheel events = %v, want content maximum %v", got, want)
	}

	value = "one"
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.handleEvent(wheel(5, 5, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if got := app.editors[id].state.ScrollY; got != 0 {
		t.Fatalf("scroll after content shrink = %v, want 0", got)
	}
}

type editorClock struct{ now time.Time }

func (c *editorClock) Now() time.Time { return c.now }

func TestCaretBlinkUsesOnlyFocusedDeadlines(t *testing.T) {
	clock := &editorClock{now: time.Unix(1, 0)}
	app, _ := testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(100), Height: Px(30)}})
	})
	app.clock = clock
	if deadline := app.nextDeadline(clock.now); deadline != nil {
		t.Fatalf("unfocused deadline = %v", deadline)
	}
	focusEditor(t, app)
	deadline := app.nextDeadline(clock.now)
	if deadline == nil || !deadline.Equal(clock.now.Add(500*time.Millisecond)) {
		t.Fatalf("focused deadline = %v", deadline)
	}
	before := app.Diagnostics().PaintCount
	clock.now = clock.now.Add(500 * time.Millisecond)
	dirty, err := app.handleCaretDeadline()
	if err != nil || !dirty {
		t.Fatalf("deadline dirty/error = %v/%v", dirty, err)
	}
	if app.Diagnostics().PaintCount != before+1 {
		t.Fatal("caret deadline did not redraw exactly once")
	}
	app.handleEvent(platform.Event{Kind: platform.EventWindowFocusLost})
	if deadline := app.nextDeadline(clock.now); deadline != nil {
		t.Fatalf("hidden deadline = %v", deadline)
	}
}

type deadlineEvents struct {
	events    []platform.Event
	deadlines []*time.Time
}

func (e *deadlineEvents) Wait(_ context.Context, deadline *time.Time) (platform.Event, error) {
	if deadline == nil {
		e.deadlines = append(e.deadlines, nil)
	} else {
		copyValue := *deadline
		e.deadlines = append(e.deadlines, &copyValue)
	}
	event := e.events[0]
	e.events = e.events[1:]
	return event, nil
}
func (*deadlineEvents) Poll() (platform.Event, bool, error) { return platform.Event{}, false, nil }
func (*deadlineEvents) Wake() error                         { return nil }

func TestCaretBlinkLoopHasNoUnfocusedDeadlineAndOneFramePerExpiry(t *testing.T) {
	clock := &editorClock{now: time.Unix(2, 0)}
	app, native := testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(100), Height: Px(30)}})
	})
	app.clock = clock
	events := &deadlineEvents{events: []platform.Event{
		{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyTab}},
		{Kind: platform.EventDeadline},
		{Kind: platform.EventClose},
	}}
	rendererValue := &frameTestRenderer{}
	app.backend = &editorRuntime{text: native, events: events, render: rendererValue}
	app.events, app.running = events, true
	loop := uiruntime.Loop{Clock: clock, Events: events, Renderer: rendererValue, Display: func() paint.DisplayList { return app.currentDisplay() }, Deadline: app.nextDeadline, Handle: app.handleEvent}
	if err := loop.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(events.deadlines) != 3 || events.deadlines[0] != nil || events.deadlines[1] == nil {
		t.Fatalf("deadlines = %#v", events.deadlines)
	}
	if rendererValue.presents != 3 {
		t.Fatalf("presents = %d, want initial, focus, expiry", rendererValue.presents)
	}
}

func BenchmarkInputEdit10K(b *testing.B) {
	value := strings.Repeat("界", 10_000)
	state := internalinput.State{Selection: internalinput.Range{Start: 5_000, End: 5_000}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := state.Insert(value, "文", internalinput.ReasonTyping); err != nil {
			b.Fatal(err)
		}
	}
}

var _ backend.Runtime = (*editorRuntime)(nil)
var _ backend.TextInputRuntime = (*editorRuntime)(nil)
