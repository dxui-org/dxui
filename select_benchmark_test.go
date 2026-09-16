package dxui

import (
	"fmt"
	"testing"

	"github.com/dxui-org/dxui/internal/platform"
)

func benchmarkSelect(b *testing.B, count int) {
	options := make([]SelectOption, count)
	for index := range options {
		options[index] = SelectOption{Value: fmt.Sprintf("value-%d", index), Label: fmt.Sprintf("Option %d", index)}
	}
	app := NewApp(AppOptions{Width: 240, Height: 300})
	app.root = func() View { return selectNode("bench", "value-0", options, nil) }
	if err := app.buildRoot(); err != nil {
		b.Fatal(err)
	}
	app.handleEvent(appKey(platform.EventKeyDown, platform.KeyTab, false))
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEnter, false))
		app.handleEvent(appKey(platform.EventKeyUp, platform.KeyEnter, false))
		app.handleEvent(appKey(platform.EventKeyDown, platform.KeyDown, false))
		app.handleEvent(appKey(platform.EventKeyDown, platform.KeyEscape, false))
	}
}

func BenchmarkSelectOpenAndKeyboard100(b *testing.B)  { benchmarkSelect(b, 100) }
func BenchmarkSelectOpenAndKeyboard1000(b *testing.B) { benchmarkSelect(b, 1000) }
