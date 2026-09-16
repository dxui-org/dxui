package main

import (
	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon"
)

var sink dxui.View

func main() {
	data := []dxui.IconData{
		icon.Search(), icon.House(), icon.User(), icon.Settings(), icon.Menu(),
		icon.X(), icon.Check(), icon.ChevronLeft(), icon.ChevronRight(), icon.Plus(),
		icon.Minus(), icon.Trash2(), icon.Pencil(), icon.Heart(), icon.Star(),
		icon.Calendar(), icon.Clock(), icon.Mail(), icon.Download(), icon.Upload(),
	}
	views := make([]dxui.View, len(data))
	for index := range data {
		views[index] = dxui.Icon(dxui.IconProps{Data: data[index]})
	}
	sink = dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal}, views...)
}
