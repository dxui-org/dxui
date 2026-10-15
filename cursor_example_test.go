package dxui_test

import "github.com/dxui-org/dxui"

func ExampleStyle_cursor() {
	button := dxui.Button(dxui.ButtonProps{
		Style: dxui.Style{Cursor: dxui.CursorDefault},
	}, dxui.Label("Arrow on this button"))
	label := dxui.Label("Text cursor on hover").WithStyle(dxui.Style{
		Cursor: dxui.CursorText,
	})
	_, _ = button, label
}
