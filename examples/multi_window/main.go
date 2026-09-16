// Command multi_window demonstrates one App event loop owning a main window
// and multiple independent native child windows.
package main

import (
	"fmt"
	"log"

	"github.com/dxui-org/dxui"
)

type childState struct {
	number, count int
	window        *dxui.Window
}

func main() {
	app := dxui.NewApp(dxui.AppOptions{Title: "dxui multi-window", Width: 560, Height: 360, MinWidth: 420, MinHeight: 260})
	children := make([]*childState, 0, 4)
	next := 1
	create := func() {
		state := &childState{number: next}
		next++
		window, err := app.CreateWindow(dxui.WindowOptions{Title: fmt.Sprintf("Child %d", state.number), Width: 360, Height: 220, MinWidth: 280, MinHeight: 180}, func() dxui.View { return childView(state) })
		if err != nil {
			log.Printf("create child: %v", err)
			return
		}
		state.window = window
		children = append(children, state)
	}
	updateLast := func() {
		for i := len(children) - 1; i >= 0; i-- {
			state := children[i]
			if state.window != nil && !state.window.Closed() {
				state.count++
				if err := state.window.Invalidate(); err != nil {
					log.Printf("update child: %v", err)
				}
				return
			}
		}
	}
	if err := app.Run(func() dxui.View {
		open := 0
		for _, state := range children {
			if state.window != nil && !state.window.Closed() {
				open++
			}
		}
		return dxui.Box(dxui.BoxProps{Gap: 12, Style: dxui.Style{Padding: dxui.Padding(20)}},
			dxui.Label(fmt.Sprintf("Independent child windows open: %d", open)),
			dxui.TextButton(dxui.ButtonProps{OnPress: create}, "Create child window"),
			dxui.TextButton(dxui.ButtonProps{OnPress: updateLast}, "Update newest child"),
			dxui.Label("Close children independently. Closing this main window exits all windows."),
		)
	}); err != nil {
		log.Fatal(err)
	}
}

func childView(state *childState) dxui.View {
	return dxui.Box(dxui.BoxProps{Gap: 12, Style: dxui.Style{Padding: dxui.Padding(20)}},
		dxui.Label(fmt.Sprintf("Child %d has isolated retained/input state.", state.number)),
		dxui.Label(fmt.Sprintf("Updates from main: %d", state.count)),
		dxui.TextButton(dxui.ButtonProps{OnPress: func() { state.count++; _ = state.window.Invalidate() }}, "Update this child"),
		dxui.TextButton(dxui.ButtonProps{OnPress: func() { state.window.Close() }}, "Close this child"),
	)
}
