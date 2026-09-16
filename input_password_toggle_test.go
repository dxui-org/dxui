package dxui

import (
	"math"
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/platform"
)

func passwordToggleCenter(t *testing.T, app *App, id uint64) (float32, float32) {
	t.Helper()
	action := app.inputActions[id]
	geometry := resultGeometryForTest(action)
	hit, icon, err := passwordToggleRects(action.input, &geometry, app.theme)
	if err != nil {
		t.Fatal(err)
	}
	if hit.Width <= 0 || icon.Width <= 0 {
		t.Fatalf("password toggle geometry = hit %+v icon %+v", hit, icon)
	}
	return icon.X + icon.Width/2, icon.Y + icon.Height/2
}

func resultGeometryForTest(action inputAction) layout.Result {
	return layout.Result{Rect: action.geometry, Content: action.content}
}

func TestPasswordToggleClickKeepsEditorStateAndControlledValue(t *testing.T) {
	value := "a deliberately wide password value"
	selection := Some(TextRange{Start: 3, End: 11})
	changes := 0
	app, native := testEditorApp(t, func() View {
		return Input(InputProps{
			Key: "password", Style: Style{Width: Px(150), Height: Px(36)},
			Value: value, Selection: selection, Password: true, ShowPasswordToggle: true,
			OnChange: func(next string) { changes++; value = next },
		})
	})
	focusEditor(t, app)
	id := app.interaction.Focused()
	editor := app.editors[id]
	editor.state.ScrollX = 17
	wantSelection, wantScroll := editor.state.Selection, editor.state.ScrollX
	x, y := passwordToggleCenter(t, app, id)
	hiddenIcon := passwordIcon(app.currentDisplay(), id)

	if _, _, err := app.handleEvent(appPointer(platform.EventMouseMove, x, y)); err != nil {
		t.Fatal(err)
	}
	node := findInstance(app.retained.Root(), id)
	if !node.State.PasswordToggleHover {
		t.Fatal("password toggle did not enter hover state")
	}
	if background := passwordToggleBackground(app.currentDisplay(), id); background.Color.A != 32 {
		t.Fatalf("password toggle hover background = %+v", background)
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseDown, x, y)); err != nil {
		t.Fatal(err)
	}
	if !node.State.PasswordTogglePressed {
		t.Fatal("password toggle did not enter pressed state")
	}
	if background := passwordToggleBackground(app.currentDisplay(), id); background.Color.A != 64 {
		t.Fatalf("password toggle pressed background = %+v", background)
	}
	if _, _, err := app.handleEvent(appPointer(platform.EventMouseUp, x, y)); err != nil {
		t.Fatal(err)
	}
	if !node.State.PasswordVisible || node.State.PasswordTogglePressed {
		t.Fatalf("password toggle state after release = %+v", node.State)
	}
	visibleIcon := passwordIcon(app.currentDisplay(), id)
	if hiddenIcon.Text == nil || visibleIcon.Text == nil || hiddenIcon.Text.Key == visibleIcon.Text.Key {
		t.Fatalf("password toggle icons did not change: hidden=%+v visible=%+v", hiddenIcon, visibleIcon)
	}
	if app.interaction.Focused() != id || editor.state.Selection != wantSelection || editor.state.ScrollX != wantScroll {
		t.Fatalf("editor state changed: focus=%d selection=%+v scroll=%g", app.interaction.Focused(), editor.state.Selection, editor.state.ScrollX)
	}
	if changes != 0 || value != "a deliberately wide password value" {
		t.Fatalf("controlled value/change = %q/%d", value, changes)
	}
	if native.started != 1 || native.stopped != 0 {
		t.Fatalf("native focus calls = start %d stop %d", native.started, native.stopped)
	}
}

func TestPasswordToggleRetainsByIdentityAndClearsOnUnmount(t *testing.T) {
	value := "secret"
	mounted := true
	app, _ := testEditorApp(t, func() View {
		if !mounted {
			return Text(TextProps{Key: "replacement", Value: "gone"})
		}
		return Input(InputProps{Key: "stable", Style: Style{Width: Px(140), Height: Px(36)}, Value: value, Password: true, ShowPasswordToggle: true})
	})
	id := app.retained.Root().ID
	x, y := passwordToggleCenter(t, app, id)
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	app.handleEvent(appPointer(platform.EventMouseUp, x, y))
	if !app.retained.Root().State.PasswordVisible {
		t.Fatal("password did not become visible")
	}
	value = "controlled replacement"
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.retained.Root().ID != id || !app.retained.Root().State.PasswordVisible || app.editors[id].value != value {
		t.Fatalf("keyed rebuild lost state: id=%d state=%+v editor=%+v", app.retained.Root().ID, app.retained.Root().State, app.editors[id])
	}

	mounted = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if _, live := app.editors[id]; live {
		t.Fatal("unmounted password editor state remained live")
	}
	mounted = true
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.retained.Root().ID == id || app.retained.Root().State.PasswordVisible {
		t.Fatalf("remounted input reused temporary visibility: id=%d state=%+v", app.retained.Root().ID, app.retained.Root().State)
	}
}

func TestPasswordToggleDisabledAndPasswordCopyPolicy(t *testing.T) {
	value := "secret"
	app, native := testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(140), Height: Px(36)}, Value: value,
			Selection: Some(TextRange{Start: 0, End: 6}), Password: true, ShowPasswordToggle: true, Disabled: true,
			OnChange: func(string) { t.Fatal("disabled password changed") }})
	})
	id := app.retained.Root().ID
	x, y := passwordToggleCenter(t, app, id)
	for _, kind := range []platform.EventKind{platform.EventMouseMove, platform.EventMouseDown, platform.EventMouseUp} {
		if _, _, err := app.handleEvent(appPointer(kind, x, y)); err != nil {
			t.Fatal(err)
		}
	}
	state := app.retained.Root().State
	if state.PasswordVisible || state.PasswordToggleHover || state.PasswordTogglePressed || app.interaction.Focused() != 0 {
		t.Fatalf("disabled password toggle interacted: %+v focus=%d", state, app.interaction.Focused())
	}
	if native.started != 0 {
		t.Fatal("disabled password toggle started native text input")
	}
	if icon := passwordIcon(app.currentDisplay(), id); icon.Text == nil || icon.Color.A >= 255 {
		t.Fatalf("disabled password icon = %+v", icon)
	}

	app, native = testEditorApp(t, func() View {
		return Input(InputProps{Style: Style{Width: Px(140), Height: Px(36)}, Value: value,
			Selection: Some(TextRange{Start: 0, End: 6}), Password: true, ShowPasswordToggle: true})
	})
	focusEditor(t, app)
	id = app.interaction.Focused()
	x, y = passwordToggleCenter(t, app, id)
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	app.handleEvent(appPointer(platform.EventMouseUp, x, y))
	app.handleEvent(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyEvent{Key: platform.KeyC, Modifiers: platform.Modifiers{Primary: true}}})
	if native.clipboard != "" {
		t.Fatal("visible password was copied to the clipboard")
	}
}

func TestPasswordToggleLayoutAndPaintRespectCombinationThemeAndScale(t *testing.T) {
	base := Input(InputProps{Style: Style{Width: Px(200), Height: Px(40), Padding: UniformEdges(Metric(7))}, Value: "secret", Password: true})
	toggled := Input(InputProps{Style: Style{Width: Px(200), Height: Px(40), Padding: UniformEdges(Metric(7))}, Value: "secret", Password: true, ShowPasswordToggle: true})
	baseApp, _ := testEditorApp(t, func() View { return base })
	toggleApp, _ := testEditorApp(t, func() View { return toggled })
	if baseApp.geometry.Rect != toggleApp.geometry.Rect {
		t.Fatalf("explicit outer style changed: base=%+v toggle=%+v", baseApp.geometry.Rect, toggleApp.geometry.Rect)
	}
	if got, want := baseApp.geometry.Content.Width-toggleApp.geometry.Content.Width, float32(20*passwordToggleIconScale+passwordToggleGap); math.Abs(float64(got-want)) > 1e-4 {
		t.Fatalf("reserved content width = %g, want %g", got, want)
	}
	hit, iconBounds, err := passwordToggleRects(toggleApp.view.node.input, toggleApp.geometry, toggleApp.theme)
	if err != nil {
		t.Fatal(err)
	}
	if contentRight := toggleApp.geometry.Content.X + toggleApp.geometry.Content.Width; contentRight+passwordToggleGap > iconBounds.X || hit.X != contentRight {
		t.Fatalf("text/icon separation = content %+v hit %+v icon %+v", toggleApp.geometry.Content, hit, iconBounds)
	}

	for _, props := range []InputProps{
		{Value: "secret", ShowPasswordToggle: true},
		{Value: "secret", Password: true},
	} {
		app, _ := testEditorApp(t, func() View { return Input(props) })
		if countPasswordIcons(app.currentDisplay(), app.retained.Root().ID) != 0 {
			t.Fatalf("password toggle painted for ineffective props %+v", props)
		}
	}

	theme := LightTheme()
	theme.Semantic.Metrics[MetricComponentIconSize] = Metric(26)
	themed := NewApp(AppOptions{Width: 300, Height: 120, Theme: theme})
	themed.root = func() View {
		return Input(InputProps{Style: Style{Width: Px(200), Height: Px(40), Padding: UniformEdges(Metric(7))}, Value: "secret", Password: true, ShowPasswordToggle: true})
	}
	if err := themed.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if got, want := baseApp.geometry.Content.Width-themed.geometry.Content.Width, float32(26*passwordToggleIconScale+passwordToggleGap); math.Abs(float64(got-want)) > 1e-4 {
		t.Fatalf("theme-sized reserve = %g, want %g", got, want)
	}
	one, err := buildDisplayList(themed.view, themed.retained.Root(), themed.geometry, themed.theme, themed.text, themed.images, 1, 1, themed.textSourceBudget(), nil, themed.editorDisplaySnapshot())
	if err != nil {
		t.Fatal(err)
	}
	two, err := buildDisplayList(themed.view, themed.retained.Root(), themed.geometry, themed.theme, themed.text, themed.images, 2, 2, themed.textSourceBudget(), one, themed.editorDisplaySnapshot())
	if err != nil {
		t.Fatal(err)
	}
	first, second := passwordIcon(one, themed.retained.Root().ID), passwordIcon(two, themed.retained.Root().ID)
	if first.Text == nil || second.Text == nil || first.Rect.Width != 26*passwordToggleIconScale || second.Rect.Width != 26*passwordToggleIconScale || first.Text.Width != 21 || second.Text.Width != 42 {
		t.Fatalf("password icon logical/physical scale = %+v/%+v", first, second)
	}
	if first.Color.A != 160 || second.Color.A != 160 {
		t.Fatalf("password icon alpha = %d/%d, want 160", first.Color.A, second.Color.A)
	}
	if !hasClip(one, paintRect(themed.geometry.Rect), themed.retained.Root().ID) {
		t.Fatal("password toggle was not clipped to the input bounds")
	}
}

func TestPasswordTogglePropTransitionInvalidatesLayout(t *testing.T) {
	show := false
	app, _ := testEditorApp(t, func() View {
		return Input(InputProps{Key: "password", Style: Style{Width: Px(180)}, Value: "secret", Password: true, ShowPasswordToggle: show})
	})
	beforeLayout := app.Diagnostics().LayoutCount
	beforeWidth := app.geometry.Content.Width
	show = true
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if app.Diagnostics().LayoutCount != beforeLayout+1 {
		t.Fatalf("layout count = %d, want %d", app.Diagnostics().LayoutCount, beforeLayout+1)
	}
	if got, want := beforeWidth-app.geometry.Content.Width, float32(20*passwordToggleIconScale+passwordToggleGap); math.Abs(float64(got-want)) > 1e-4 {
		t.Fatalf("transition reserve = %g, want %g", got, want)
	}
}

func TestDisablingPasswordToggleMasksRetainedVisibleState(t *testing.T) {
	show := true
	app, _ := testEditorApp(t, func() View {
		return Input(InputProps{Key: "password", Style: Style{Width: Px(180)}, Value: "secret", Password: true, ShowPasswordToggle: show})
	})
	id := app.retained.Root().ID
	x, y := passwordToggleCenter(t, app, id)
	app.handleEvent(appPointer(platform.EventMouseDown, x, y))
	app.handleEvent(appPointer(platform.EventMouseUp, x, y))
	if !app.retained.Root().State.PasswordVisible || !app.inputActions[id].passwordVisible {
		t.Fatal("password toggle did not reveal before the prop transition")
	}
	show = false
	if _, err := app.buildAndCommit(); err != nil {
		t.Fatal(err)
	}
	if !app.retained.Root().State.PasswordVisible {
		t.Fatal("stable identity did not retain temporary visibility state")
	}
	if app.inputActions[id].passwordVisible || countPasswordIcons(app.currentDisplay(), id) != 0 {
		t.Fatal("disabled password toggle left plaintext display active")
	}
	for _, command := range app.currentDisplay() {
		if command.Text != nil && strings.Contains(command.Text.Key, "secret") {
			t.Fatal("disabled password toggle left plaintext in a text mask key")
		}
	}
}

func countPasswordIcons(list paint.DisplayList, id uint64) int {
	count := 0
	for _, command := range list {
		if command.Kind == paint.CommandDrawIcon && command.NodeID == id {
			count++
		}
	}
	return count
}

func passwordIcon(list paint.DisplayList, id uint64) paint.Command {
	for _, command := range list {
		if command.Kind == paint.CommandDrawIcon && command.NodeID == id {
			return command
		}
	}
	return paint.Command{}
}

func passwordToggleBackground(list paint.DisplayList, id uint64) paint.Command {
	for _, command := range list {
		if command.Kind == paint.CommandFillRoundedRect && command.NodeID == id && (command.Color.A == 32 || command.Color.A == 64) {
			return command
		}
	}
	return paint.Command{}
}

func hasClip(list paint.DisplayList, rect paint.Rect, id uint64) bool {
	for _, command := range list {
		if command.Kind == paint.CommandPushClip && command.NodeID == id && command.Rect == rect {
			return true
		}
	}
	return false
}
