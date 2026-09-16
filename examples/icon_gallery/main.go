// Command icon_gallery displays the complete generated Lucide catalog.
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dxui-org/dxui"
	"github.com/dxui-org/dxui/icon/catalog"
)

const (
	columns    = 14
	rowHeight  = 64
	headerSize = 78
)

var (
	lucidePage       = dxui.LiteralColor(dxui.RGBA(255, 255, 255, 255))
	lucideIcon       = dxui.LiteralColor(dxui.RGBA(60, 60, 67, 255))
	lucideTile       = dxui.LiteralColor(dxui.RGBA(246, 246, 247, 255))
	lucideTileHover  = dxui.LiteralColor(dxui.RGBA(228, 228, 233, 255))
	lucideTileActive = dxui.LiteralColor(dxui.RGBA(221, 221, 227, 255))
)

func main() {
	app := dxui.NewApp(dxui.AppOptions{Title: "dxui Lucide gallery", Width: 920, Height: 720})
	query := ""
	version := uint64(0)
	offset := dxui.Point{}

	root := func(ctx dxui.LayoutContext) dxui.View {
		icons := catalog.Icons
		search := strings.ToLower(strings.TrimSpace(query))
		if search != "" {
			icons = make([]catalog.Entry, 0, len(catalog.Icons))
			for _, entry := range catalog.Icons {
				if strings.Contains(strings.ToLower(entry.GoName), search) || strings.Contains(entry.Name, search) {
					icons = append(icons, entry)
				}
			}
		}
		rowCount := (len(icons) + columns - 1) / columns
		searchWidth := max(float32(1), ctx.Width-36)
		listHeight := max(float32(1), ctx.Height-36-headerSize-10)
		return dxui.Box(dxui.BoxProps{Style: dxui.Style{
			Width: dxui.Fill(), Height: dxui.Fill(), Padding: dxui.Padding(18), Background: lucidePage,
		}, Gap: 10},
			dxui.Box(dxui.BoxProps{Style: dxui.Style{Height: dxui.Px(headerSize)}, Gap: 8},
				dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 10, Align: dxui.AlignCenter},
					dxui.Text(dxui.TextProps{Style: dxui.Style{Grow: 1, Text: dxui.TextStyle{Size: 24, LineHeight: 30}}, Value: "Lucide icon gallery"}),
					dxui.Text(dxui.TextProps{Value: fmt.Sprintf("%d / %d icons", len(icons), len(catalog.Icons))}),
				),
				dxui.Input(dxui.InputProps{
					Key: "icon-search", Style: dxui.Style{
						Width: dxui.Px(searchWidth), MinWidth: dxui.Px(searchWidth), Height: dxui.Px(40), Shrink: dxui.NoShrink(),
					},
					Value: query, Placeholder: "Search Go constructor names...",
					OnChange: func(value string) {
						query = value
						version++
						offset = dxui.Point{}
					},
				}),
			),
			dxui.VirtualList(dxui.VirtualListProps{
				Key: "icon-gallery", Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Px(listHeight), Background: lucidePage},
				Count: rowCount, Version: version, RowHeight: rowHeight, Overscan: 2, Scrollbar: dxui.ScrollbarAuto,
				Offset: dxui.Some(offset), OnScroll: dxui.Assign(&offset),
				ItemKey: func(index int) string { return icons[index*columns].Name },
				Build: func(index int) dxui.View {
					offset := index * columns
					last := min(offset+columns, len(icons))
					cards := make([]dxui.View, 0, columns)
					for _, entry := range icons[offset:last] {
						cards = append(cards, dxui.Tooltip(dxui.TooltipProps{
							Key: entry.Name, Placement: dxui.OverlayBottom, Delay: 250 * time.Millisecond,
						}, dxui.Button(dxui.ButtonProps{
							Variant: dxui.ButtonGhost,
							Style: dxui.Style{
								Width: dxui.Px(56), Height: dxui.Px(56), MinWidth: dxui.Px(56), MinHeight: dxui.Px(56),
								Padding: dxui.Padding(12), Radius: dxui.Round(6),
								Background: lucideTile, Border: dxui.NoBorder(),
							},
							States: dxui.StateStyles{
								Hover:   dxui.StylePatch{Background: dxui.Some(lucideTileHover)},
								Pressed: dxui.StylePatch{Background: dxui.Some(lucideTileActive)},
							},
						}, dxui.Box(dxui.BoxProps{
							Direction: dxui.Horizontal,
							Style:     dxui.Style{Width: dxui.Percent(100), Height: dxui.Percent(100), Grow: 1},
							Justify:   dxui.JustifyCenter,
							Align:     dxui.AlignCenter,
						}, dxui.Icon(dxui.IconProps{Data: entry.Icon(), Size: 24, Color: lucideIcon}))),
							dxui.Text(dxui.TextProps{Value: entry.GoName}),
						))
					}
					return dxui.Box(dxui.BoxProps{
						Direction: dxui.Horizontal, Gap: 7,
						Style: dxui.Style{Width: dxui.Fill(), Height: dxui.Px(rowHeight), Padding: dxui.Edges(0, 0, 8, 0)},
					}, cards...)
				},
			}),
		)
	}

	if err := app.RunResponsive(root); err != nil {
		log.Fatal(err)
	}
}
