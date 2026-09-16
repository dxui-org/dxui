package main

import (
	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon"
)

var sink dxui.View

func main() { sink = dxui.Icon(dxui.IconProps{Data: icon.Search()}) }
