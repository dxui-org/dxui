// Command text_gallery exercises dxui Text using only the root public API.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/dxui-org/dxui"
)

func main() {
	software := flag.Bool("software", false, "force SDL's named software renderer")
	dark := flag.Bool("dark", false, "start with the dark theme")
	duration := flag.Duration("duration", 0, "close automatically after this duration")
	flag.Parse()

	theme := dxui.LightTheme()
	if *dark {
		theme = dxui.DarkTheme()
	}
	preference := dxui.RendererAuto
	if *software {
		preference = dxui.RendererSoftware
	}
	app := dxui.NewApp(dxui.AppOptions{
		Title: "dxui text gallery", Width: 760, Height: 720,
		Renderer: preference, Theme: theme,
		Caches: dxui.CacheBudgets{GlyphBytes: 2 << 20, TextMeasureBytes: 512 << 10},
	})

	inputValue := "中文输入"
	textareaValue := "第一行中文\n第二行等待输入"
	root := func() dxui.View {
		return gallery(inputValue, textareaValue, func(value string) { inputValue = value }, func(value string) { textareaValue = value })
	}
	if *duration > 0 {
		go func() {
			switchAfter := *duration / 2
			if switchAfter > 0 {
				time.Sleep(switchAfter)
				_ = app.Update(func() {
					if *dark {
						_ = app.SetTheme(dxui.LightTheme())
					} else {
						_ = app.SetTheme(dxui.DarkTheme())
					}
				})
			}
			time.Sleep(*duration - switchAfter)
			_ = app.Update(app.Close)
		}()
	}
	if err := app.Run(root); err != nil {
		log.Fatal(err)
	}
}

func gallery(inputValue, textareaValue string, onInput, onTextarea func(string)) dxui.View {
	return dxui.Box(dxui.BoxProps{
		Style: dxui.Style{
			Padding: dxui.Padding(20), Overflow: dxui.OverflowClip,
			Background: dxui.TokenColor(dxui.ColorSemanticSurface),
		},
		Gap: 10, Align: dxui.AlignStart,
	},
		label("Text gallery / English", 26, dxui.TextStart),
		label("ABCDEFGHIJKLMNOPQRSTUVWXYZ  abcdefghijklmnopqrstuvwxyz", 16, dxui.TextStart),
		label("0123456789  1,234.56  + - × ÷ =", 18, dxui.TextStart),
		label("中文文字 · 换行 · 裁剪 · 缩放 · 主题颜色 · 默认软件渲染 · 高密度", 20, dxui.TextStart),
		dxui.Text(dxui.TextProps{
			Value: "Word wrapping uses the same advances, fallback choices, kerning, and line metrics for intrinsic measure and paint. 中文文字换行保持同一套度量。",
			Wrap:  dxui.TextWrapWords,
			Style: textStyle(16, 23, dxui.TextStart, dxui.Style{Width: dxui.Px(520)}),
		}),
		alignmentSample(dxui.TextStart, "start aligned / 起始"),
		alignmentSample(dxui.TextCenter, "center aligned / 居中"),
		alignmentSample(dxui.TextEnd, "end aligned / 末端"),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 12, Align: dxui.AlignStart},
			label("Clipped:", 16, dxui.TextStart),
			dxui.Text(dxui.TextProps{
				Value: "This long line is clipped / 中文文字裁剪",
				Style: textStyle(16, 22, dxui.TextStart, dxui.Style{Width: dxui.Px(170), Height: dxui.Px(22), Overflow: dxui.OverflowClip}),
			}),
		),
		label("DPI coverage: logical metrics stay fixed; masks regenerate at native 1× / 1.5× / 2× scale.", 15, dxui.TextStart),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 18, Align: dxui.AlignEnd},
			label("1× sample", 14, dxui.TextStart),
			label("1.5× sample", 21, dxui.TextStart),
			label("2× sample", 28, dxui.TextStart),
		),
		label("Theme text color switches halfway through -duration smoke runs.", 16, dxui.TextStart),
		label("Zero-config system CJK fallback: use a native IME below to inspect composition and commit.", 15, dxui.TextStart),
		dxui.Input(dxui.InputProps{Value: inputValue, Placeholder: "请输入中文", OnChange: onInput}),
		dxui.Textarea(dxui.TextareaProps{Value: textareaValue, Placeholder: "请输入多行中文", OnChange: onTextarea}),
		dxui.Box(dxui.BoxProps{Direction: dxui.Horizontal, Gap: 10},
			dxui.TextButton(dxui.ButtonProps{}, "中文按钮"),
			dxui.Select(dxui.SelectProps{Value: "cn", Placeholder: "请选择", Options: []dxui.SelectOption{{Value: "cn", Label: "中文选项"}}}),
			dxui.Menu(dxui.MenuProps{Items: []dxui.MenuItem{{Value: "menu", Label: "中文菜单"}}}),
		),
	)
}

func alignmentSample(align dxui.TextAlign, value string) dxui.View {
	return dxui.Text(dxui.TextProps{
		Value: value,
		Style: textStyle(16, 24, align, dxui.Style{Width: dxui.Px(360)}),
	})
}

func label(value string, size float32, align dxui.TextAlign) dxui.View {
	return dxui.Text(dxui.TextProps{Value: value, Style: textStyle(size, size*1.35, align, dxui.Style{})})
}

func textStyle(size, lineHeight float32, align dxui.TextAlign, layout dxui.Style) dxui.Style {
	layout.Text = dxui.TextStyle{
		Size: size, LineHeight: lineHeight,
		Color: dxui.TokenColor(dxui.ColorSemanticText), Align: align,
	}
	return layout
}
