package dxui

import (
	"image"
	"math"
	"strings"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/tree"
)

func avatarTestSource(width, height int) ImageSource {
	return ImageFromGo(image.NewNRGBA(image.Rect(0, 0, width, height)))
}

func TestAvatarDefaultCustomAndExplicitSizes(t *testing.T) {
	source := avatarTestSource(200, 100)
	tests := []struct {
		name string
		view View
		want Rect
	}{
		{"theme default", Avatar(AvatarProps{Source: source}), Rect{Width: 40, Height: 40}},
		{"custom metric", Avatar(AvatarProps{Source: source, Size: 64}), Rect{Width: 64, Height: 64}},
		{"explicit style", Avatar(AvatarProps{Style: Style{Width: Px(32), Height: Px(48)}, Source: source, Size: 64}), Rect{Width: 32, Height: 48}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := intrinsicChildGeometry(t, LightTheme(), test.view)
			if got.Width != test.want.Width || got.Height != test.want.Height {
				t.Fatalf("geometry = %+v, want %+v", got, test.want)
			}
		})
	}

	theme := LightTheme()
	theme.Semantic.Metrics[MetricComponentAvatarSize] = Metric(52)
	if got := intrinsicChildGeometry(t, theme, Avatar(AvatarProps{Source: source})); got.Width != 52 || got.Height != 52 {
		t.Fatalf("themed geometry = %+v, want 52x52", got)
	}
	before, err := prepareTheme(LightTheme())
	if err != nil {
		t.Fatal(err)
	}
	after, err := prepareTheme(theme)
	if err != nil {
		t.Fatal(err)
	}
	if !themeChangesLayout(Avatar(AvatarProps{Source: source}), before, after) {
		t.Fatal("default Avatar size token change did not invalidate layout")
	}
	explicit := Avatar(AvatarProps{Style: Style{Width: Px(32), Height: Px(32)}, Source: source})
	if themeChangesLayout(explicit, before, after) {
		t.Fatal("Avatar theme size invalidated fully explicit dimensions")
	}
	withoutDefault := LightTheme()
	delete(withoutDefault.Semantic.Metrics, MetricComponentAvatarSize)
	if got := intrinsicChildGeometry(t, withoutDefault, explicit); got.Width != 32 || got.Height != 32 {
		t.Fatalf("explicit Avatar depended on default size token: %+v", got)
	}
}

func TestAvatarBuiltInThemesExposeComponentAndDefaultMetric(t *testing.T) {
	for name, theme := range map[string]Theme{"light": LightTheme(), "dark": DarkTheme()} {
		t.Run(name, func(t *testing.T) {
			if _, ok := theme.Components[ComponentAvatar]; !ok {
				t.Fatal("missing ComponentAvatar")
			}
			resolved, err := prepareTheme(theme)
			if err != nil {
				t.Fatal(err)
			}
			if size, err := resolved.metric(TokenMetric(MetricComponentAvatarSize)); err != nil || size != 40 {
				t.Fatalf("default Avatar size = %g, %v", size, err)
			}
		})
	}
}

func TestAvatarCircleAndSquareUseCenteredCover(t *testing.T) {
	source := avatarTestSource(200, 100)
	for _, test := range []struct {
		name  string
		shape AvatarShape
	}{
		{"circle", AvatarCircle},
		{"square", AvatarSquare},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := NewApp(AppOptions{Width: 100, Height: 100})
			app.root = func() View {
				return Avatar(AvatarProps{Style: Style{Width: Px(100), Height: Px(100)}, Source: source, Shape: test.shape})
			}
			if err := app.buildRoot(); err != nil {
				t.Fatal(err)
			}
			var draw paint.Command
			for _, command := range app.display {
				if command.Kind == paint.CommandDrawImage {
					draw = command
				}
			}
			if draw.Rect != (paint.Rect{X: -50, Width: 200, Height: 100}) {
				t.Fatalf("cover destination = %+v", draw.Rect)
			}
			if test.shape == AvatarCircle {
				wantRadii := paint.Radii{TopLeft: 50, TopRight: 50, BottomRight: 50, BottomLeft: 50}
				if draw.ImageClip != (paint.Rect{Width: 100, Height: 100}) || draw.Radii != wantRadii {
					t.Fatalf("circle clip = %+v radii=%+v", draw.ImageClip, draw.Radii)
				}
			} else if draw.ImageClip != (paint.Rect{}) || draw.Radii != (paint.Radii{}) {
				t.Fatalf("square unexpectedly used rounded image clip: %+v", draw)
			}
		})
	}
}

func TestAvatarCallbacksAndImageCacheAreShared(t *testing.T) {
	source := avatarTestSource(7, 5)
	loaded := Size{}
	app := NewApp(AppOptions{Width: 40, Height: 40})
	app.root = func() View {
		return Avatar(AvatarProps{Source: source, OnLoad: func(size Size) { loaded = size }})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if loaded != (Size{Width: 7, Height: 5}) {
		t.Fatalf("load size = %+v", loaded)
	}
	fromImage, err := decodeImage(app.images, ImageProps{Source: source})
	if err != nil {
		t.Fatal(err)
	}
	fromAvatar, err := decodeImage(app.images, imagePropsForAvatar(&AvatarProps{Source: source}))
	if err != nil {
		t.Fatal(err)
	}
	if fromImage != fromAvatar {
		t.Fatal("Image and Avatar did not reuse the decoded cache entry")
	}

	var reported error
	bad := NewApp(AppOptions{Width: 40, Height: 40})
	bad.root = func() View {
		return Avatar(AvatarProps{Source: ImageBytes([]byte("bad")), OnError: func(err error) { reported = err }})
	}
	if err := bad.buildRoot(); err != nil {
		t.Fatal(err)
	}
	if reported == nil || !strings.Contains(reported.Error(), "decode header") {
		t.Fatalf("error callback = %v", reported)
	}
}

func TestAvatarDisplayComposesScrollClipTranslationOpacityAndZIndex(t *testing.T) {
	source := avatarTestSource(100, 100)
	app := NewApp(AppOptions{Width: 80, Height: 80})
	app.root = func() View {
		return Scroll(ScrollProps{
			Style: Style{Width: Px(80), Height: Px(80), Opacity: Some(float32(.5))},
			Axis:  ScrollBoth, InitialOffset: Some(Point{Y: 20}), Scrollbar: ScrollbarHidden,
		}, Avatar(AvatarProps{
			Style:  Style{ZIndex: 7, Opacity: Some(float32(.5))},
			Source: source, Size: 100, Shape: AvatarCircle,
		}))
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	var item, draw paint.Command
	for _, command := range app.display {
		if command.Kind == paint.CommandItem && command.ZIndex == 7 {
			item = command
		}
		if command.Kind == paint.CommandDrawImage {
			draw = command
		}
	}
	if item.Bounds.Y != -20 || item.Opacity != .25 || !item.ClipSet || item.Clip != (paint.Rect{Width: 80, Height: 80}) {
		t.Fatalf("avatar item = %+v", item)
	}
	if draw.ImageClip.Y != -20 || draw.ImageClip.Width != 100 || draw.Radii.TopLeft != 50 {
		t.Fatalf("translated circle draw = %+v", draw)
	}
}

func TestAvatarExplicitBorderRemainsVisibleAboveCircleImage(t *testing.T) {
	app := NewApp(AppOptions{Width: 40, Height: 40})
	app.root = func() View {
		return Avatar(AvatarProps{
			Style: Style{
				Width: Px(40), Height: Px(40), Opacity: Some(float32(.5)),
				Border: Border{Width: Metric(2), Color: LiteralColor(RGBA(10, 20, 30, 255))},
			},
			Source: avatarTestSource(200, 100), Shape: AvatarCircle,
		})
	}
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	var border, draw paint.Command
	for _, command := range app.display {
		switch command.Kind {
		case paint.CommandStrokeRoundedRect:
			border = command
		case paint.CommandDrawImage:
			draw = command
		}
	}
	if border.Width != 2 || border.Radii.TopLeft != 20 {
		t.Fatalf("circle border = %+v", border)
	}
	if draw.ImageClip != (paint.Rect{X: 2, Y: 2, Width: 36, Height: 36}) || draw.Radii.TopLeft != 18 {
		t.Fatalf("border-safe image clip = %+v", draw)
	}
	recorder := &paint.RecordingPainter{}
	if err := paint.Replay(app.display, recorder); err != nil {
		t.Fatal(err)
	}
	for _, command := range recorder.Records {
		if command.Kind == paint.CommandDrawImage && command.Color.A != 128 {
			t.Fatalf("image alpha = %d, want 128", command.Color.A)
		}
	}
}

func TestAvatarRejectsInvalidShapeAndSize(t *testing.T) {
	source := avatarTestSource(1, 1)
	for _, view := range []View{
		Avatar(AvatarProps{Source: source, Shape: AvatarShape(99)}),
		Avatar(AvatarProps{Source: source, Size: -1}),
		Avatar(AvatarProps{Source: source, Size: float32(math.NaN())}),
		Avatar(AvatarProps{Source: source, Size: float32(math.Inf(1))}),
	} {
		if _, err := describeView(view); err == nil {
			t.Fatal("invalid Avatar accepted")
		}
	}
}

func TestAvatarInvalidationSeparatesShapeSizeAndSource(t *testing.T) {
	shape, size, source := AvatarCircle, float32(40), avatarTestSource(20, 10)
	app := NewApp(AppOptions{Width: 100, Height: 100})
	app.root = func() View { return Avatar(AvatarProps{Source: source, Shape: shape, Size: size}) }
	if err := app.buildRoot(); err != nil {
		t.Fatal(err)
	}
	shape = AvatarSquare
	changes, err := app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(tree.DirtyPaint) || changes.Has(tree.DirtyMeasure) || changes.Has(tree.DirtyLayout) {
		t.Fatalf("shape dirty flags = %v", changes.Dirty)
	}
	size = 52
	changes, err = app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(tree.DirtyMeasure) || !changes.Has(tree.DirtyLayout) || !changes.Has(tree.DirtyPaint) {
		t.Fatalf("size dirty flags = %v", changes.Dirty)
	}
	source = avatarTestSource(20, 10)
	changes, err = app.buildAndCommit()
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(tree.DirtyResource) || !changes.Has(tree.DirtyPaint) {
		t.Fatalf("source dirty flags = %v", changes.Dirty)
	}
}
