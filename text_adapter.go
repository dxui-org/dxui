package dxui

import (
	"fmt"
	"os"
	"runtime"

	internalimage "github.com/dxui-org/dxui/internal/image"
	"github.com/dxui-org/dxui/internal/layout"
	internaltext "github.com/dxui-org/dxui/internal/text"
)

func (a *App) textEngine() (*internaltext.Engine, error) {
	a.mu.Lock()
	if a.text != nil || a.textErr != nil {
		engine, err := a.text, a.textErr
		a.mu.Unlock()
		return engine, err
	}
	fonts := append([]Font(nil), a.options.Fonts...)
	defaultFamily := a.options.DefaultFont
	measureBudget := a.options.Caches.TextMeasureBytes
	fontBudget := a.options.Caches.FontBytes
	disableSystemFallback := a.options.DisableSystemFontFallback
	a.mu.Unlock()

	normalizedFontBudget := normalizedBudget(fontBudget, 32<<20)
	specs, _, err := resolveFontSpecsWithUsage(fonts, fontBudget)
	if err == nil {
		var fallback internaltext.FontFallbackResolver
		if !disableSystemFallback {
			fallback = newSystemFallbackResolver(systemFontCandidates(runtime.GOOS, FontFamilySystemCJK, os.Getenv("WINDIR")), osFontFileLoader{})
		}
		var engine *internaltext.Engine
		engine, err = internaltext.NewEngine(internaltext.Options{
			Fonts: specs, DefaultFamily: string(defaultFamily), MeasureCacheBytes: measureBudget,
			FontSourceBytes: normalizedFontBudget, FallbackResolver: fallback,
		})
		if err == nil {
			a.mu.Lock()
			if a.text == nil && a.textErr == nil {
				a.text = engine
			} else {
				engine.Close()
			}
			engine, err = a.text, a.textErr
			a.mu.Unlock()
			return engine, err
		}
	}
	a.mu.Lock()
	if a.text == nil && a.textErr == nil {
		a.textErr = err
	}
	engine, storedErr := a.text, a.textErr
	a.mu.Unlock()
	return engine, storedErr
}

func (a *App) releaseTextEngine() {
	a.mu.Lock()
	engine := a.text
	a.text = nil
	a.textErr = nil
	a.mu.Unlock()
	if engine != nil {
		engine.Close()
	}
}

func makeIntrinsicResolver(engine *internaltext.Engine, images *internalimage.Cache, theme resolvedTheme) intrinsicResolver {
	return func(view View) layout.IntrinsicMeasurer {
		if view.node == nil {
			return nil
		}
		switch view.node.kind {
		case viewInput:
			props := textPropsFromCommon(view.node.input.common(), view.node.input.Value, TextNoWrap, 1)
			return textIntrinsic(engine, theme, props)
		case viewTextarea:
			props := textPropsFromCommon(view.node.textarea.common(), view.node.textarea.Value, view.node.textarea.Wrap, 0)
			return textIntrinsic(engine, theme, props)
		case viewSelect:
			value := view.node.selectp.Placeholder
			for _, option := range view.node.selectp.Options {
				if option.Value == view.node.selectp.Value {
					value = option.Label
					break
				}
			}
			props := textPropsFromCommon(view.node.selectp.common(), value, TextNoWrap, 1)
			return selectIntrinsic(engine, theme, props)
		case viewTabs:
			return tabsIntrinsic(engine, theme, *view.node.tabs)
		case viewMenu:
			return menuIntrinsic(engine, theme, *view.node.menu)
		case viewToggleSwitch:
			return fixedMetricIntrinsic(theme, MetricComponentToggleWidth, MetricComponentToggleHeight)
		case viewSlider:
			return fixedMetricIntrinsic(theme, MetricComponentSliderWidth, MetricComponentSliderHeight)
		case viewProgressBar:
			return fixedMetricIntrinsic(theme, MetricComponentProgressBarWidth, MetricComponentProgressBarHeight)
		case viewIcon:
			size := iconSizeMetric(view.node.icon.Size)
			return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
				resolved, err := theme.metric(size)
				return squareIntrinsic(resolved), err
			})
		case viewImage:
			return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
				bitmap, err := decodeImage(images, *view.node.image)
				if err != nil {
					return squareIntrinsic(16), nil
				}
				size := layout.Size{Width: float32(bitmap.Width), Height: float32(bitmap.Height)}
				return layout.IntrinsicSize{Minimum: size, Preferred: size}, nil
			})
		case viewAvatar:
			if view.node.avatar.common().Style.Width.kind != lengthAuto && view.node.avatar.common().Style.Height.kind != lengthAuto {
				return nil
			}
			size := avatarSizeMetric(view.node.avatar.Size)
			return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
				resolved, err := theme.metric(size)
				return layout.IntrinsicSize{Preferred: layout.Size{Width: resolved, Height: resolved}}, err
			})
		case viewControlMark:
			return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
				size, err := theme.metric(view.node.mark.Size)
				return squareIntrinsic(size), err
			})
		case viewText:
		default:
			return nil
		}
		if view.node.text == nil {
			return nil
		}
		return layout.IntrinsicMeasureFunc(func(available layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
			request, err := resolvedTextRequest(*view.node.text, theme)
			if err != nil {
				return layout.IntrinsicSize{}, err
			}
			if request.Wrap == internaltext.WordWrap && available.Width.MaxSet {
				request.MaxWidth, request.MaxWidthSet = available.Width.Max, true
			}
			laidOut, err := engine.Layout(request)
			if err != nil {
				return layout.IntrinsicSize{}, err
			}
			preferred := layout.Size{Width: laidOut.Metrics.Width, Height: laidOut.Metrics.Height}
			minimum := layout.Size{Width: laidOut.Metrics.MinimumWidth, Height: laidOut.Metrics.Height}
			return layout.IntrinsicSize{Minimum: minimum, Preferred: preferred}, nil
		})
	}
}

func textIntrinsic(engine *internaltext.Engine, theme resolvedTheme, props TextProps) layout.IntrinsicMeasurer {
	return layout.IntrinsicMeasureFunc(func(available layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
		request, err := resolvedTextRequest(props, theme)
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		if request.Wrap == internaltext.WordWrap && available.Width.MaxSet {
			request.MaxWidth, request.MaxWidthSet = available.Width.Max, true
		}
		laidOut, err := engine.Layout(request)
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		size := layout.Size{Width: max(40, laidOut.Metrics.Width), Height: max(laidOut.Metrics.Height, 16)}
		return layout.IntrinsicSize{Minimum: layout.Size{Width: min(size.Width, 40), Height: min(size.Height, 16)}, Preferred: size}, nil
	})
}

func selectIntrinsic(engine *internaltext.Engine, theme resolvedTheme, props TextProps) layout.IntrinsicMeasurer {
	text := textIntrinsic(engine, theme, props)
	return layout.IntrinsicMeasureFunc(func(available layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
		size, err := text.MeasureIntrinsic(available)
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		reserved := selectIndicatorSize + selectIndicatorGap
		size.Minimum.Width += reserved
		size.Preferred.Width += reserved
		return size, nil
	})
}

func fixedMetricIntrinsic(theme resolvedTheme, widthToken, heightToken MetricToken) layout.IntrinsicMeasurer {
	return layout.IntrinsicMeasureFunc(func(layout.IntrinsicRequest) (layout.IntrinsicSize, error) {
		width, err := theme.metric(TokenMetric(widthToken))
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		height, err := theme.metric(TokenMetric(heightToken))
		if err != nil {
			return layout.IntrinsicSize{}, err
		}
		size := layout.Size{Width: width, Height: height}
		return layout.IntrinsicSize{Minimum: size, Preferred: size}, nil
	})
}

func squareIntrinsic(size float32) layout.IntrinsicSize {
	value := layout.Size{Width: size, Height: size}
	return layout.IntrinsicSize{Minimum: value, Preferred: value}
}

func decodeImage(images *internalimage.Cache, props ImageProps) (*internalimage.Bitmap, error) {
	source := props.Source.source
	if images == nil || source == nil {
		return nil, fmt.Errorf("image source is unavailable")
	}
	return images.Decode(internalimage.Source{ID: source.id, Encoded: source.bytes, File: source.path, Image: source.goImg}, props.MaxPixels)
}

func resolvedTextRequest(props TextProps, theme resolvedTheme) (internaltext.Request, error) {
	style := props.Style.Text
	size := float32(0)
	lineHeight := float32(0)
	var err error
	size, err = theme.metric(textSizeMetric(style.Size))
	if err != nil {
		return internaltext.Request{}, fmt.Errorf("font size: %w", err)
	}
	if size <= 0 {
		return internaltext.Request{}, fmt.Errorf("font size must resolve positive")
	}
	lineHeight, err = theme.metric(lineHeightMetric(style.LineHeight))
	if err != nil {
		return internaltext.Request{}, fmt.Errorf("line height: %w", err)
	}
	if lineHeight <= 0 {
		return internaltext.Request{}, fmt.Errorf("line height must resolve positive")
	}
	families := make([]string, len(style.Families))
	for index, family := range style.Families {
		families[index] = string(family)
	}
	return internaltext.Request{
		Text: props.Value, Families: families, Size: size, LineHeight: lineHeight,
		Weight: uint16(style.Weight), Slant: uint8(style.Slant), Wrap: internaltext.Wrap(props.Wrap),
		MaxLines: props.MaxLines, Align: internaltext.Align(style.Align),
	}, nil
}

func (a *App) textScale() (float32, float32) {
	a.mu.Lock()
	native := a.backend
	a.mu.Unlock()
	if native == nil {
		return 1, 1
	}
	viewport := native.Diagnostics().Viewport
	if viewport.LogicalWidth <= 0 || viewport.LogicalHeight <= 0 || viewport.PixelWidth <= 0 || viewport.PixelHeight <= 0 {
		return 1, 1
	}
	return float32(viewport.PixelWidth) / float32(viewport.LogicalWidth), float32(viewport.PixelHeight) / float32(viewport.LogicalHeight)
}

func (a *App) textSourceBudget() int {
	a.mu.Lock()
	value := a.options.Caches.TextSourceBytes
	a.mu.Unlock()
	return normalizedBudget(value, 2<<20)
}
