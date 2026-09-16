package dxui

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"testing"

	"github.com/dxui-org/dxui/internal/layout"
)

func TestLucideMasksAgainstIndependentSVGReferences(t *testing.T) {
	theme := testIconTheme(t)
	for name, data := range lucideReferenceIcons {
		t.Run(name, func(t *testing.T) {
			bitmap, _, err := rasterIcon(IconProps{Data: data, Size: 24}, layout.Rect{Width: 24, Height: 24}, theme, 1.25, 1.25)
			if err != nil {
				t.Fatal(err)
			}
			referencePNG, err := base64.StdEncoding.DecodeString(lucideReferencePNGs[name])
			if err != nil {
				t.Fatal(err)
			}
			reference, err := png.Decode(bytes.NewReader(referencePNG))
			if err != nil {
				t.Fatal(err)
			}
			bounds := reference.Bounds()
			if bounds.Dx() != bitmap.Width || bounds.Dy() != bitmap.Height {
				t.Fatalf("reference = %dx%d, mask = %dx%d", bounds.Dx(), bounds.Dy(), bitmap.Width, bitmap.Height)
			}
			intersection, union := 0, 0
			difference := 0
			for y := 0; y < bitmap.Height; y++ {
				for x := 0; x < bitmap.Width; x++ {
					actual := bitmap.Pixels[y*bitmap.Width+x]
					_, _, _, alpha := reference.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
					want := uint8(alpha >> 8)
					if actual > 32 || want > 32 {
						union++
						if actual > 32 && want > 32 {
							intersection++
						}
					}
					delta := int(actual) - int(want)
					if delta < 0 {
						delta = -delta
					}
					difference += delta
				}
			}
			iou := float64(intersection) / float64(union)
			mae := float64(difference) / float64(bitmap.Width*bitmap.Height*255)
			if iou < .72 || mae > .16 {
				t.Fatalf("SVG agreement IoU=%.3f MAE=%.3f", iou, mae)
			}
		})
	}
}

// lucideReferencePNGs contains the original independently rendered SVG
// fixtures. Keeping the small reference images in the test source prevents a
// source archive from silently omitting untracked binary testdata.
var lucideReferencePNGs = map[string]string{
	"atom":          "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeCAYAAAA7MK6iAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAABmJLR0QAAAAAAAD5Q7t/AAAACXBIWXMAAAB4AAAAeACd9VpgAAABL0lEQVRIx+2Wsa3CQAyGv6BXR09QZAoaVmABmixAGwkxBuVbgjJlqkxDSpZ4NBfJcexLKiwhLEXK2f7vl30+n+ErgfIDXIB/9fVADZQOrkz23sBe0r5Z0ocB1N8NqBKmSuslzCNHriPtgBYYnM3ujn5IuM6I3BTptFO2LXBeiOqc/KTslE+WuMscx8khPWUwMvIscevYr4rsqdZXB9euJR6UvmBeQE3SN8wLr1D4YYlYXgV5Vpr0qHBHg3yUrdD3HnHNtFCs9B4c7MFJuyzI2iMumRfMGlKPXOPLHNhrBjK9e2XbZ9Jupd+UygA1ysfaeCPsjWGvWCGyIz2ZV+lo+xP/v8JeML1q9zWkYRGHnHFYVYfd47DOFdarw1+nsPf47RNI2MwVNmWO5G+fqz9fXiYPYtR94Y+uAAAAJXRFWHRkYXRlOmNyZWF0ZQAyMDI2LTA5LTA3VDA3OjQ2OjMyKzAwOjAwhHNKjQAAACV0RVh0ZGF0ZTptb2RpZnkAMjAyNi0wOS0wN1QwNzo0NjozMiswMDowMPUu8jEAAAAydEVYdHN2Zzpjb21tZW50ACBAbGljZW5zZSBsdWNpZGUtc3RhdGljIHYxLjQxLjAgLSBJU0MgRSg2+wAAAABJRU5ErkJggg==",
	"badge-cent":    "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeCAYAAAA7MK6iAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAABmJLR0QAAAAAAAD5Q7t/AAAACXBIWXMAAAB4AAAAeACd9VpgAAACFklEQVRIx8WWzYoTQRDHf0kkicY4aiTBXZBV8SDEu+BNvPgEgoeIXlUkUfcuXoJo4iMk3iQP4BP4BLtHhSAYETR60IgmEg9TAzVtd0++dP7QTE/Xv6r6o6q7ICXkVtQrAqeBOpAHJsDsX040D7SBuaW1Rb4xZIEAqABjh9OojYUXiN5KOAS0PE6awBX5ujj3xc7COAP8chh7LyvSCGTcxp+KvRgyjpV+IX5efeAzcALYBT5a9GrAE+CTbPcNJZvJ5Ca+1eqtGwClNY7qpbHtTmSNbfI57Qqnm+Bc23MGXKBIfQfnGPDAMHjTM8me4gUODscVqWORN/Cn01WLzjMlr9icFoDvitQz5HcSnEbtoqHXJ57nf10yHUUYAlUlqxnGrxv/j4z/oqGrU62tnZpBUDYm9VzJrsmY5gM8djiGeOzE5OfV4G3LMbxR8pzDcQa4BGxjh07TcwdkcEsR9i1KZ+U7BH47DM+B17ixp/ono7waqcG6RemtfHdwP6VJK76g+h+iTmpnDClFNaSYx/Afby7z0tYRa7veXsjkHhrjt4DDwCuLjrbjrMs2/TqVDHvecmiZ9/hogtOBstXSwk1WIFXCCiQq+BpKNpVJeisQSK65jhj8MmEK2vg/sdRcPiRVmfeAy8BdD6fJklWmhq6rv+JPp2+EUb9WXW1DgfAcbU6finwhZBYlGjgInCJ81UbAO+DHMgb+AD5bJae42O3xAAAAJXRFWHRkYXRlOmNyZWF0ZQAyMDI2LTA5LTA3VDA3OjQ2OjMyKzAwOjAwhHNKjQAAACV0RVh0ZGF0ZTptb2RpZnkAMjAyNi0wOS0wN1QwNzo0NjozMiswMDowMPUu8jEAAAAydEVYdHN2Zzpjb21tZW50ACBAbGljZW5zZSBsdWNpZGUtc3RhdGljIHYxLjQxLjAgLSBJU0MgRSg2+wAAAABJRU5ErkJggg==",
	"brain-circuit": "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeCAYAAAA7MK6iAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAABmJLR0QAAAAAAAD5Q7t/AAAACXBIWXMAAAB4AAAAeACd9VpgAAACGElEQVRIx8WWz0tVQRTHP88oa2MPF5FZmCHWpkh0I1bYwiAKRAoEA1uUYIsgSETEtVvBTdFaoVoEobvCRZj/gNBCooUIClEqD4p+wGsx78KXaebOzC3xwOHdd+Z7zvecOefOHdgnOZCAbQE6gKNABfi118m1AR+BqqVjQMnjcwKYA+Zrz4VIqzn6xOHTWtuNDLNZhHhDAiwA14EHFnm34I95EkyS0+L42lq7IGuLYn/qIH2cQtpnOT+y1kuyVhH7jthPAk0ppHccWU85cK6tLLy9Z3D36Pc/EJ8F7gHtecTPxfE2pj++PlULqJP8oAC+RexOEeL7mXOdBKqX5xepPYqUdy7jEcnsK3AzJ4BiQ4PUTkSPv0cGHEogDkqXFWzWg+vk795lUo85Zg+lEG9KoEnHegPwDPfQzGDObbU1xpAeF4eXHsw0aRM8nEeYTXWz2OY82Pe13x/AtYhiPsVU3COZDuTgGiRZra5c01FgJVStin7OFiN9/ttUr0mg/r0m1qtLD7As/z8A48A6sOohzuSqJ/42ptcVAjKCe0LPByoO6a2YXfgcQVyXSFwFLoaI9foy6Kn2nGDeAL0efSi4VyHiKwJew9yjVcrAlmC6cmI1WlUHT7O3lsM4cAOYsOxL+O/WAHctfPD9LuO+xKt+wRwoeXLZ8rkUIgbzzbUrzHSa+C/QMI7TrBTheBg4henPLua9/BlJ6pU/auMbuiEH244AAAAldEVYdGRhdGU6Y3JlYXRlADIwMjYtMDktMDdUMDc6NDY6MzMrMDA6MDAiBEE5AAAAJXRFWHRkYXRlOm1vZGlmeQAyMDI2LTA5LTA3VDA3OjQ2OjMzKzAwOjAwU1n5hQAAADJ0RVh0c3ZnOmNvbW1lbnQAIEBsaWNlbnNlIGx1Y2lkZS1zdGF0aWMgdjEuNDEuMCAtIElTQyBFKDb7AAAAAElFTkSuQmCC",
	"search":        "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeCAYAAAA7MK6iAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAAABmJLR0QAAAAAAAD5Q7t/AAAACXBIWXMAAAB4AAAAeACd9VpgAAABiElEQVRIx+2WMU8CMRTHfxgJLBAWBhcHB2MuIZG4GlYnmHGSuLAzMBG/gxOLu87oB7jE3ZyjYXJASZydHHAoJu9qj/baMzr4T5qU8N7/99611x7864+pAQyABFgZxhRoFwksA+MMmGnMgVYodAdYGMyfgCvgErjLKGAYAtXNzoCaIbYERIYixj6P90UYxKg1dlFPg3fzgOWaxsB2zsI7Gtyp6IZPkkEj4XHhkjAgvaa+qmgNlG0JiQiu2YItmggv6ysmX5lQHQq/gf7nVkbSfQHghZjvu4LfCwB/iHnVFbxXALgu5ktbsNyJpUCwPExObMFTERwFgmPh1bQFt0XwbQA0Ej6Ja9JcJPU8oBXgVXgcuya2SK91Jyd0KXJneaseavDR2nSTIq3TFXDu8cSMXx4T1InURF0gu6jliA2xX6PvA+9uMMwas3WnwfAG6mqzARPSG6m/CZ7nkCgDB8AR6uytojbSI/AAvBly+sC1+H0K3Ph07yO983qYnR/8mez74cdU/w3oN30C3Gauzkr7udIAAAAldEVYdGRhdGU6Y3JlYXRlADIwMjYtMDktMDdUMDc6NDY6MzMrMDA6MDAiBEE5AAAAJXRFWHRkYXRlOm1vZGlmeQAyMDI2LTA5LTA3VDA3OjQ2OjMzKzAwOjAwU1n5hQAAADJ0RVh0c3ZnOmNvbW1lbnQAIEBsaWNlbnNlIGx1Y2lkZS1zdGF0aWMgdjEuNDEuMCAtIElTQyBFKDb7AAAAAElFTkSuQmCC",
}
