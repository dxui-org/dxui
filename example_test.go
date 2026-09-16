package dxui_test

import (
	"image"
	"time"

	"github.com/dxui-org/dxui"
)

func ExampleNoBorder() {
	_ = dxui.Style{Border: dxui.NoBorder()}
	_ = dxui.StylePatch{Border: dxui.Some(dxui.NoBorder())}
	// Force applies after active interaction states.
	_ = dxui.Style{Force: dxui.StylePatch{Border: dxui.Some(dxui.NoBorder())}}
	// Output:
}

func ExampleBorder() {
	_ = dxui.Border{Width: dxui.Metric(1), Color: dxui.TokenColor(dxui.ColorSemanticBorder), Sides: dxui.BorderBottom}
	_ = dxui.Border{Width: dxui.Metric(1), Color: dxui.TokenColor(dxui.ColorSemanticBorder), Sides: dxui.BorderLeft | dxui.BorderRight}
	// Output:
}

// Example is the compile-checked form of the README quickstart. App.Run is
// intentionally not called by the documentation test because it owns a native
// window and blocks until Close.
func Example() {
	app := dxui.NewApp(dxui.AppOptions{Title: "Login", Width: 900, Height: 680})
	username, password := "", ""
	root := func() dxui.View {
		return dxui.Box(dxui.BoxProps{
			Key: "form", Style: dxui.Style{Padding: dxui.PaddingXY(20, 12), Radius: dxui.Round(8)},
			Token: dxui.ComponentPanel, States: dxui.StateStyles{}, Pointer: dxui.PointerAuto,
			Gap: 12,
		},
			dxui.Input(dxui.InputProps{
				Key: "username", Value: username, Placeholder: "Username",
				OnChange: dxui.Assign(&username),
			}),
			dxui.Input(dxui.InputProps{
				Key: "password", Value: password, Password: true,
				OnChange: func(value string) { password = value },
			}),
			dxui.TextButton(dxui.ButtonProps{OnPress: app.Close}, "Cancel"),
		)
	}
	_ = app
	_ = root // Production code calls app.Run(root) directly from main.
}

func ExampleTabs() {
	section := "overview"
	view := dxui.Tabs(dxui.TabsProps{
		Value: section,
		Items: []dxui.TabItem{
			{Value: "overview", Label: "Overview"},
			{Value: "settings", Label: "Settings"},
		},
		OnChange: func(next string) { section = next },
	})
	_ = view // Render content separately according to section.
}

func ExampleMenu() {
	view := dxui.Menu(dxui.MenuProps{
		Value: "new",
		Items: []dxui.MenuItem{
			{Value: "new", Label: "New"},
			{Value: "archive", Label: "Archive", Disabled: true},
		},
		OnAction: func(value string) { _ = value },
	})
	_ = view
}

func ExampleAvatar() {
	portrait := dxui.ImageFromGo(image.NewNRGBA(image.Rect(0, 0, 96, 64)))
	view := dxui.Avatar(dxui.AvatarProps{
		Source: portrait,
		Shape:  dxui.AvatarCircle,
		Size:   48,
	})
	_ = view
}

func ExampleBadge() {
	view := dxui.Badge(
		dxui.BadgeProps{},
		dxui.Text(dxui.TextProps{Value: "New"}),
	)
	_ = view
	// Output:
}

func ExamplePopover() {
	open := false
	view := dxui.Popover(dxui.PopoverProps{
		Open:         open,
		OnOpenChange: func(next bool) { open = next },
	}, dxui.Text(dxui.TextProps{Value: "Details"}),
		dxui.Button(dxui.ButtonProps{}, dxui.Text(dxui.TextProps{Value: "Action"})))
	_ = view
}

func ExampleTooltip() {
	view := dxui.Tooltip(dxui.TooltipProps{
		Placement: dxui.OverlayTop,
		Delay:     350 * time.Millisecond,
	}, dxui.Button(dxui.ButtonProps{}, dxui.Text(dxui.TextProps{Value: "Save"})),
		dxui.Text(dxui.TextProps{Value: "Save changes"}))
	_ = view
}
