package main

import (
	"fmt"

	"github.com/dxui-org/dxui"
)

type item struct {
	id    int
	label string
}

func main() {
	items := make([]item, 100_000)
	for i := range items {
		items[i] = item{id: i, label: fmt.Sprintf("Item %06d", i)}
	}
	version := uint64(1)
	offset := dxui.Point{}
	selected := -1
	nextID := len(items)
	app := dxui.NewApp(dxui.AppOptions{Title: "dxui VirtualList", Width: 720, Height: 640})
	err := app.RunResponsive(func(ctx dxui.LayoutContext) dxui.View {
		listHeight := max(float32(0), ctx.Height-116)
		return dxui.Box(dxui.BoxProps{Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Fill(), Padding: dxui.Padding(12)}, Gap: 8},
			dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 8},
				dxui.TextButton(dxui.ButtonProps{OnPress: func() {
					items[0].label += " *"
					version++
				}}, "Modify first"),
				dxui.TextButton(dxui.ButtonProps{OnPress: func() {
					items = append([]item{{id: nextID, label: fmt.Sprintf("Inserted %06d", nextID)}}, items...)
					nextID++
					version++
				}}, "Insert first"),
				dxui.TextButton(dxui.ButtonProps{OnPress: func() {
					if len(items) > 0 {
						items = items[1:]
						version++
					}
				}}, "Delete first"),
				dxui.TextButton(dxui.ButtonProps{OnPress: func() {
					if len(items) > 1 {
						items[0], items[len(items)-1] = items[len(items)-1], items[0]
						version++
					}
				}}, "Swap ends"),
				dxui.TextButton(dxui.ButtonProps{OnPress: func() { offset.Y = float32(max(0, len(items)-12)) * 40 }}, "Jump near end"),
			),
			dxui.Label(fmt.Sprintf("items=%d  selected=%d  offset=%.0f", len(items), selected, offset.Y)),
			dxui.VirtualList(dxui.VirtualListProps{
				Key: "items", Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Px(listHeight)},
				Count: len(items), Version: version, RowHeight: 40, Overscan: 3,
				Offset: dxui.Some(offset), OnScroll: dxui.Assign(&offset),
				ItemKey: func(index int) string { return fmt.Sprintf("item-%d", items[index].id) },
				Build: func(index int) dxui.View {
					value := items[index]
					return dxui.TextButton(dxui.ButtonProps{OnPress: func() { selected = value.id }}, value.label)
				},
			}),
		)
	})
	if err != nil {
		panic(err)
	}
}
