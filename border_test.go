package dxui

import (
	"reflect"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/tree"
)

func TestNoBorderCascadeAndSidesMerge(t *testing.T) {
	if NoBorder() == (Border{}) || NoBorder().Width != Metric(0) {
		t.Fatal("NoBorder must explicitly set zero width")
	}
	color := ColorRGBA(20, 30, 40, 128)
	theme := LightTheme()
	theme.Components[ComponentButton] = ComponentTheme{
		Base:   StylePatch{Border: Some(Border{Width: Metric(3), Color: color, Sides: BorderLeft, Pattern: BorderDashed})},
		States: StateStyles{Hover: StylePatch{Border: Some(Stroke(5, color))}},
	}
	resolved, err := prepareTheme(theme)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		style    Style
		states   StateStyles
		hover    bool
		width    float32
		sides    BorderSides
		pattern  BorderPattern
		colorSet bool
	}{
		{"unset", Style{}, StateStyles{}, false, 3, BorderLeft, BorderDashed, true},
		{"none", Style{Border: NoBorder()}, StateStyles{}, false, 0, 0, BorderSolid, true},
		{"state restores", Style{Border: NoBorder()}, StateStyles{}, true, 5, 0, BorderSolid, true},
		{"local state clears", Style{}, StateStyles{Hover: StylePatch{Border: Some(NoBorder())}}, true, 0, 0, BorderSolid, false},
		{"force clears", Style{Force: StylePatch{Border: Some(NoBorder())}}, StateStyles{}, true, 0, 0, BorderSolid, false},
		{"sides only", Style{Border: Border{Sides: BorderBottom}}, StateStyles{}, false, 3, BorderBottom, BorderDashed, true},
		{"width resets sides", Style{Border: Border{Width: Metric(2)}}, StateStyles{}, false, 2, 0, BorderSolid, true},
		{"color resets width and sides", Style{Border: Border{Color: color}}, StateStyles{}, false, 0, 0, BorderSolid, true},
		{"pattern only preserves sides", Style{Border: Border{Pattern: BorderDashed}}, StateStyles{}, false, 3, BorderLeft, BorderDashed, true},
		{"patch replaces all", Style{}, StateStyles{Default: StylePatch{Border: Some(Border{Sides: BorderBottom})}}, false, 0, BorderBottom, BorderSolid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &tree.Node{}
			node.State.Hovered = tc.hover
			v, err := computeVisual(viewProps{Style: tc.style, States: tc.states}, node, viewButton, resolved)
			if err != nil || v.border.width != tc.width || v.border.sides != tc.sides || v.border.pattern != tc.pattern || v.border.colorSet != tc.colorSet {
				t.Fatalf("border=%+v err=%v", v.border, err)
			}
		})
	}
}

func borderStateEntry(states *StateStyles, index int) *StylePatch {
	switch index {
	case 0:
		return &states.Default
	case 1:
		return &states.Hover
	case 2:
		return &states.Focus
	case 3:
		return &states.Checked
	case 4:
		return &states.Pressed
	default:
		return &states.Disabled
	}
}

func TestInvalidBorderSidesPreserveFrame(t *testing.T) {
	for _, mask := range []BorderSides{16, 128, BorderAll | 32} {
		for entry := 0; entry < 8; entry++ {
			props := ButtonProps{Style: Style{Width: Px(60), Height: Px(40), Border: Stroke(2, ColorRGBA(1, 2, 3, 128))}}
			app := NewApp(AppOptions{Width: 80, Height: 60})
			app.root = func() View { return Button(props, Text(TextProps{})) }
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			root, geometry := app.retained.Root(), app.geometry
			display := append(paint.DisplayList(nil), app.display...)
			invalid := Border{Sides: mask}
			switch entry {
			case 6:
				props.Style.Border = invalid
			case 7:
				props.Style.Force.Border = Some(invalid)
			default:
				borderStateEntry(&props.States, entry).Border = Some(invalid)
			}
			if _, err := app.buildAndCommit(); err == nil {
				t.Fatalf("accepted mask=%d entry=%d", mask, entry)
			}
			if root != app.retained.Root() || geometry != app.geometry || !reflect.DeepEqual(display, app.display) {
				t.Fatal("failed view changed frame")
			}
			theme := LightTheme()
			component := theme.Components[ComponentButton]
			if entry < 6 {
				borderStateEntry(&component.States, entry).Border = Some(invalid)
			} else {
				component.Base.Border = Some(invalid)
			}
			theme.Components[ComponentButton] = component
			if err := app.SetTheme(theme); err == nil {
				t.Fatal("accepted invalid theme")
			}
			if root != app.retained.Root() || !reflect.DeepEqual(display, app.display) {
				t.Fatal("failed theme changed frame")
			}
		}
	}
}

func TestBorderSidesDisplayInvalidationAndGeometry(t *testing.T) {
	for _, pattern := range []BorderPattern{BorderSolid, BorderDashed} {
		border := Border{Width: Metric(2), Color: ColorRGBA(1, 2, 3, 128), Pattern: pattern}
		app := NewApp(AppOptions{Width: 80, Height: 60})
		app.root = func() View {
			return Button(ButtonProps{Style: Style{Width: Px(60), Height: Px(40), Border: border, Radius: Round(10)}}, Text(TextProps{}))
		}
		if err := app.buildRoot(); err != nil {
			t.Fatal(err)
		}
		original := append(paint.DisplayList(nil), app.display...)
		geometry := app.geometry
		hit, ok := app.display.HitTestAt(1, 20)
		if !ok {
			t.Fatal("missing initial hit")
		}
		border.Sides = BorderAll
		if _, err := app.buildAndCommit(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original, app.display) {
			t.Fatal("explicit all changed four-side display")
		}
		for _, sides := range []BorderSides{BorderTop, BorderRight, BorderBottom, BorderLeft, BorderLeft | BorderRight, BorderTop | BorderRight, BorderAll} {
			border.Sides = sides
			changes, err := app.buildAndCommit()
			if err != nil {
				t.Fatal(err)
			}
			if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) || app.geometry != geometry {
				t.Fatalf("wrong invalidation: %+v", changes)
			}
			got, ok := app.display.HitTestAt(1, 20)
			if !ok || got != hit {
				t.Fatal("side selection changed hit bounds")
			}
			found := false
			for _, command := range app.display {
				if command.Kind == paint.CommandBorder {
					found = true
					if command.Sides != paint.BorderSides(sides) || command.Dashed != (pattern == BorderDashed) {
						t.Fatalf("command=%+v", command)
					}
				}
			}
			if !found && sides != BorderAll {
				t.Fatal("missing selected border")
			}
		}
		border = NoBorder()
		changes, err := app.buildAndCommit()
		if err != nil || !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyLayout) {
			t.Fatalf("clear=%+v err=%v", changes, err)
		}
		for _, c := range app.display {
			if c.Kind == paint.CommandBorder || c.Kind == paint.CommandStrokeRoundedRect || c.Kind == paint.CommandFillStrokeRoundedRect {
				t.Fatal("NoBorder emitted border")
			}
		}
		changes, err = app.buildAndCommit()
		if err != nil || changes.Has(tree.DirtyPaint) {
			t.Fatal("unchanged border requested repaint")
		}
	}
}

func TestBorderStorageEvidence(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(Border{}), reflect.TypeOf(resolvedBorder{}), reflect.TypeOf(tree.PaintProperties{}), reflect.TypeOf(paint.Command{})} {
		var oldFields []reflect.StructField
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.Name != "Sides" && f.Name != "sides" && f.Name != "BorderSides" && f.Name != "Dashed" {
				oldFields = append(oldFields, f)
			}
		}
		t.Logf("%s: before=%d after=%d bytes", typ, reflect.StructOf(oldFields).Size(), typ.Size())
	}
}

func TestThemeBorderSidesInvalidatePaintOnly(t *testing.T) {
	theme := LightTheme()
	component := theme.Components[ComponentPanel]
	border, _ := component.Base.Border.get()
	border.Sides = BorderBottom
	component.Base.Border = Some(border)
	theme.Components[ComponentPanel] = component
	app := NewApp(AppOptions{Width: 80, Height: 60, Theme: theme})
	app.root = func() View { return Box(BoxProps{Token: ComponentPanel, Style: Style{Width: Px(60), Height: Px(40)}}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	geometry := app.geometry
	before := app.display[1].Bounds
	border.Sides = BorderLeft | BorderRight
	component.Base.Border = Some(border)
	theme.Components[ComponentPanel] = component
	if err := app.SetTheme(theme); err != nil {
		t.Fatal(err)
	}
	if app.themeDirty != tree.DirtyDisplay|tree.DirtyPaint || app.geometry != geometry {
		t.Fatalf("theme dirty=%v", app.themeDirty)
	}
	if err := app.redisplayCurrent(); err != nil {
		t.Fatal(err)
	}
	// Inside borders preserve conservative visual bounds; the full dirty frame
	// clears the removed bottom edge as well as drawing the new side edges.
	after := app.display[1].Bounds
	if paint.Union(before, after) != before || before != paintRect(geometry.Rect) {
		t.Fatalf("dirty bounds changed: %v -> %v", before, after)
	}
	for _, command := range app.display {
		if command.Kind == paint.CommandBorder && command.Sides != paint.BorderLeft|paint.BorderRight {
			t.Fatal("stale themed border")
		}
	}
}

func TestProgressBorderUsesSelectedPattern(t *testing.T) {
	for _, pattern := range []BorderPattern{BorderSolid, BorderDashed} {
		app := NewApp(AppOptions{Width: 80, Height: 60})
		app.root = func() View {
			return ProgressBar(ProgressBarProps{Value: .5, Style: Style{Width: Px(60), Height: Px(40), Border: Border{Width: Metric(2), Color: ColorRGBA(1, 2, 3, 128), Sides: BorderBottom, Pattern: pattern}}})
		}
		if err := app.buildRoot(); err != nil {
			t.Fatal(err)
		}
		last := app.display[len(app.display)-1]
		if last.Kind != paint.CommandBorder || last.Sides != paint.BorderBottom || last.Dashed != (pattern == BorderDashed) {
			t.Fatalf("progress border=%+v", last)
		}
	}
}
