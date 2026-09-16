package main

import (
	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon/catalog"
)

var sink dxui.View

func main() {
	views := make([]dxui.View, len(catalog.Icons))
	for index, entry := range catalog.Icons {
		views[index] = dxui.Icon(dxui.IconProps{Data: entry.Icon()})
	}
	sink = dxui.Box(dxui.BoxProps{}, views...)
}
