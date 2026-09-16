// Command minimal opens one resizable dxui window and paints a solid background.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/dxui-org/dxui"
)

func main() {
	software := flag.Bool("software", false, "force SDL's named software renderer")
	duration := flag.Duration("duration", 0, "close automatically after this idle duration")
	idleSample := flag.Duration("idle-sample", 0, "after a one-second settle, sample idle frame count for this duration and close")
	flag.Parse()

	preference := dxui.RendererAuto
	if *software {
		preference = dxui.RendererSoftware
	}
	options := dxui.AppOptions{
		Title: "dxui minimal", Width: 640, Height: 400,
		MinWidth: 320, MinHeight: 200,
		Background: dxui.RGBA(28, 30, 36, 255), Renderer: preference,
	}
	options.OnShown = func(app *dxui.App) {
		if *duration > 0 {
			go func() {
				timer := time.NewTimer(*duration)
				defer timer.Stop()
				<-timer.C
				if err := app.Update(app.Close); err != nil {
					log.Printf("automatic close: %v", err)
				}
			}()
		}
		if *idleSample > 0 {
			go func() {
				settle := time.NewTimer(time.Second)
				defer settle.Stop()
				<-settle.C
				before := app.Diagnostics().FrameCount
				timer := time.NewTimer(*idleSample)
				<-timer.C
				after := app.Diagnostics().FrameCount
				_ = json.NewEncoder(log.Writer()).Encode(struct {
					IdleDuration string `json:"idleDuration"`
					FramesBefore uint64 `json:"framesBefore"`
					FramesAfter  uint64 `json:"framesAfter"`
				}{IdleDuration: idleSample.String(), FramesBefore: before, FramesAfter: after})
				if err := app.Update(app.Close); err != nil {
					log.Printf("idle sample close: %v", err)
				}
			}()
		}
	}
	app := dxui.NewApp(options)

	if err := app.Run(func() dxui.View { return dxui.Label("dxui minimal") }); err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(log.Writer()).Encode(app.Diagnostics()); err != nil {
		log.Fatal(err)
	}
}
