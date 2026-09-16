package dxui

import (
	"fmt"
	"math"
	"strings"

	"github.com/dxui-org/dxui/icon"
	internalimage "github.com/dxui-org/dxui/internal/image"
	internalinput "github.com/dxui-org/dxui/internal/input"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	internaltext "github.com/dxui-org/dxui/internal/text"
	"github.com/dxui-org/dxui/internal/tree"
)

type computedVisual struct {
	background      RGBAColor
	backgroundSet   bool
	border          resolvedBorder
	radii           paint.Radii
	joinedEdges     paint.JoinedEdges
	shadows         []paint.Shadow
	opacity         float32
	visibility      Visibility
	textColor       RGBAColor
	textColorSet    bool
	buttonFocusRing bool
}

type resolvedBorder struct {
	width    float32
	color    RGBAColor
	colorSet bool
	pattern  BorderPattern
	sides    BorderSides
}

type editorDisplayState struct {
	selection        internalinput.Range
	composition      internalinput.Composition
	scrollX, scrollY float32
	caretVisible     bool
}

type textMaskReuse struct {
	values    map[string]*paint.TextBitmap
	accounted map[string]struct{}
}

func buildDisplayList(view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, textEngine *internaltext.Engine, images *internalimage.Cache, scaleX, scaleY float32, textSourceBudget int, previous paint.DisplayList, editorMaps ...map[uint64]editorDisplayState) (paint.DisplayList, error) {
	if view.node == nil || instance == nil || geometry == nil {
		return nil, fmt.Errorf("dxui: display requires a view, instance, and geometry")
	}
	list := make(paint.DisplayList, 1, 16)
	list[0] = paint.Command{Kind: paint.CommandBeginDisplayList}
	remaining := textSourceBudget
	var editors map[uint64]editorDisplayState
	if len(editorMaps) != 0 {
		editors = editorMaps[0]
	}
	reuse := retainedTextMasks(previous)
	if err := appendDisplay(&list, view, instance, geometry, theme, textEngine, images, scaleX, scaleY, &remaining, &reuse, editors, 1, paint.Rect{}, false, RGBAColor{}, false, 0, 0, "root"); err != nil {
		return nil, err
	}
	if err := appendWindowOverlays(&list, view, instance, geometry, theme, textEngine, images, scaleX, scaleY, &remaining, &reuse, editors); err != nil {
		return nil, err
	}
	return list, nil
}

func retainedTextMasks(previous paint.DisplayList) textMaskReuse {
	var reuse textMaskReuse
	for index := range previous {
		bitmap := previous[index].Text
		if bitmap != nil {
			if reuse.values == nil {
				reuse.values = make(map[string]*paint.TextBitmap)
			}
			reuse.values[bitmap.Key] = bitmap
		}
	}
	return reuse
}

func (reuse *textMaskReuse) get(key string, width, height int) ([]byte, bool) {
	if reuse == nil {
		return nil, false
	}
	bitmap, ok := reuse.values[key]
	if !ok || bitmap.Width != width || bitmap.Height != height {
		return nil, false
	}
	return bitmap.Pixels, true
}

func (reuse *textMaskReuse) remember(bitmap *paint.TextBitmap) {
	if reuse != nil && bitmap != nil {
		if reuse.values == nil {
			reuse.values = make(map[string]*paint.TextBitmap)
		}
		reuse.values[bitmap.Key] = bitmap
	}
}

func reuseMaskFunc(reuse *textMaskReuse) func(string, int, int) ([]byte, bool) {
	if reuse == nil {
		return nil
	}
	return reuse.get
}

func appendDisplay(list *paint.DisplayList, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, textEngine *internaltext.Engine, images *internalimage.Cache, scaleX, scaleY float32, textSourceRemaining *int, reuse *textMaskReuse, editors map[uint64]editorDisplayState, parentOpacity float32, inheritedClip paint.Rect, inheritedClipSet bool, inheritedText RGBAColor, inheritedTextSet bool, translateX, translateY float32, path string) error {
	rawGeometry := geometry
	worldGeometry := *geometry
	worldGeometry.Rect = translatedRect(geometry.Rect, translateX, translateY)
	worldGeometry.Content = translatedRect(geometry.Content, translateX, translateY)
	worldGeometry.Viewport = translatedRect(geometry.Viewport, translateX, translateY)
	geometry = &worldGeometry
	worldRect := paintRect(geometry.Rect)
	props, err := nodeProps(view)
	if err != nil {
		return fmt.Errorf("dxui: display %s: %w", path, err)
	}
	visual, err := computeVisual(props, instance, view.node.kind, theme, view.node.groupChild)
	if err != nil {
		return fmt.Errorf("dxui: display %s: %w", path, err)
	}
	if view.node.groupedInput {
		visual.backgroundSet = false
		visual.border = resolvedBorder{}
		visual.radii = paint.Radii{}
		visual.shadows = nil
	}
	if visual.visibility == Hidden {
		return nil
	}
	if view.node.kind == viewText && inheritedTextSet && !props.Style.Text.Color.set && props.Token == "" {
		visual.textColor, visual.textColorSet = inheritedText, true
	}
	rect := worldRect
	if view.node.kind == viewAvatar {
		rect = centeredSquare(rect)
		if view.node.avatar.Shape == AvatarCircle {
			radius := rect.Width / 2
			visual.radii = paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
		}
	}
	clip, clipSet := inheritedClip, inheritedClipSet
	if props.Style.Overflow == OverflowClip || isScrollView(view) {
		clipRect := rect
		if isScrollView(view) {
			clipRect = paintRect(geometry.Viewport)
		}
		if clipSet {
			clip = paint.Intersect(clip, clipRect)
		} else {
			clip, clipSet = clipRect, true
		}
	}
	if clipSet && len(view.node.children) == 0 && rectsDisjoint(visualBounds(rect, visual.shadows), clip) {
		return nil
	}
	worldOpacity := parentOpacity * visual.opacity
	item := paint.Command{
		Kind: paint.CommandItem, NodeID: instance.ID, Bounds: rect,
		ZIndex: props.Style.ZIndex, SourceIndex: geometry.SourceIndex,
		Visibility: true, Opacity: worldOpacity,
		Interactive: (view.node.kind == viewButton || view.node.kind == viewInput || view.node.kind == viewTextarea || view.node.kind == viewToggleSwitch || view.node.kind == viewSlider || view.node.kind == viewCheckbox || view.node.kind == viewRadio || isScrollView(view) || view.node.kind == viewSelect || view.node.kind == viewTabs || view.node.kind == viewMenu) && !instance.Properties.Semantics.Disabled,
	}
	if clipSet {
		item.ClipSet = true
		item.Clip = clip
	}
	*list = append(*list, item)
	if visual.opacity != 1 {
		*list = append(*list, paint.Command{Kind: paint.CommandPushOpacity, NodeID: instance.ID, Opacity: visual.opacity})
	}
	for _, shadow := range visual.shadows {
		if shadow.Color.A == 0 || worldOpacity == 0 {
			continue
		}
		if visual.buttonFocusRing {
			spread := shadow.Spread
			outer := paint.Rect{X: rect.X - spread, Y: rect.Y - spread, Width: rect.Width + 2*spread, Height: rect.Height + 2*spread}
			radii := paint.Radii{TopLeft: visual.radii.TopLeft + spread, TopRight: visual.radii.TopRight + spread, BottomLeft: visual.radii.BottomLeft + spread, BottomRight: visual.radii.BottomRight + spread}
			*list = append(*list, paint.Command{Kind: paint.CommandStrokeRoundedRect, NodeID: instance.ID, Rect: outer, Radii: radii, Width: spread, Color: shadow.Color})
		} else {
			*list = append(*list, paint.Command{Kind: paint.CommandDrawShadow, NodeID: instance.ID, Rect: rect, Radii: visual.radii, Shadow: shadow})
		}
	}
	clipped := props.Style.Overflow == OverflowClip || isScrollView(view)
	if clipped && !isScrollView(view) {
		*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: clip})
	}
	if view.node.kind != viewProgressBar && (visual.border.pattern == BorderDashed || visual.border.sides != 0 && visual.border.sides != BorderAll) && visual.border.width > 0 && visual.border.colorSet {
		if visual.backgroundSet {
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: rect, Radii: visual.radii, Color: paintColor(visual.background)})
		}
		appendSelectedBorderDisplay(list, rect, visual.radii, visual.border, instance.ID)
	} else if view.node.kind != viewProgressBar && visual.backgroundSet && visual.border.width > 0 && visual.border.colorSet {
		*list = append(*list, paint.Command{Kind: paint.CommandFillStrokeRoundedRect, NodeID: instance.ID, Rect: rect, Radii: visual.radii, Width: visual.border.width, Color: paintColor(visual.background), BorderColor: paintColor(visual.border.color)})
	} else if view.node.kind != viewProgressBar && visual.backgroundSet {
		*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: rect, Radii: visual.radii, Color: paintColor(visual.background), JoinedEdges: visual.joinedEdges})
	} else if view.node.kind != viewProgressBar && visual.border.width > 0 && visual.border.colorSet {
		*list = append(*list, paint.Command{Kind: paint.CommandStrokeRoundedRect, NodeID: instance.ID, Rect: rect, Radii: visual.radii, Width: visual.border.width, Color: paintColor(visual.border.color)})
	}
	if isScrollView(view) {
		*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: clip})
	}
	if view.node.kind == viewToggleSwitch {
		if err := appendToggleDisplay(list, rect, instance.State.Checked, theme, instance.ID); err != nil {
			return fmt.Errorf("dxui: display %s toggle: %w", path, err)
		}
	}
	if view.node.kind == viewSlider {
		if err := appendSliderDisplay(list, rect, normalizeSlider(*view.node.slider), theme, instance.ID); err != nil {
			return fmt.Errorf("dxui: display %s slider: %w", path, err)
		}
	}
	if view.node.kind == viewProgressBar {
		content := translatedRect(geometry.Content, translateX, translateY)
		if err := appendProgressBarDisplay(list, paintRect(content), normalizeProgress(view.node.progress.Value), visual, theme, instance.ID); err != nil {
			return fmt.Errorf("dxui: display %s progress bar: %w", path, err)
		}
		if visual.border.width > 0 && visual.border.colorSet {
			if visual.border.pattern == BorderDashed || visual.border.sides != 0 && visual.border.sides != BorderAll {
				appendSelectedBorderDisplay(list, rect, visual.radii, visual.border, instance.ID)
			} else {
				*list = append(*list, paint.Command{Kind: paint.CommandStrokeRoundedRect, NodeID: instance.ID, Rect: rect, Radii: visual.radii, Width: visual.border.width, Color: paintColor(visual.border.color)})
			}
		}
	}
	if view.node.kind == viewCheckbox {
		if len(geometry.Children) != 2 {
			return fmt.Errorf("dxui: display %s checkbox geometry mismatch", path)
		}
		mark := translatedRect(geometry.Children[0].Rect, translateX, translateY)
		markRect := paint.Rect{X: mark.X, Y: mark.Y, Width: mark.Width, Height: mark.Height}
		if err := appendCheckboxDisplay(list, markRect, instance.State.Checked, theme, instance.ID, scaleX, scaleY, textSourceRemaining, reuse); err != nil {
			return fmt.Errorf("dxui: display %s checkbox: %w", path, err)
		}
	}
	if view.node.kind == viewRadio {
		if len(geometry.Children) != 2 {
			return fmt.Errorf("dxui: display %s radio geometry mismatch", path)
		}
		mark := translatedRect(geometry.Children[0].Rect, translateX, translateY)
		markRect := paint.Rect{X: mark.X, Y: mark.Y, Width: mark.Width, Height: mark.Height}
		if err := appendRadioDisplay(list, markRect, instance.State.Checked, theme, instance.ID); err != nil {
			return fmt.Errorf("dxui: display %s radio: %w", path, err)
		}
	}
	if view.node.kind == viewInput || view.node.kind == viewTextarea {
		if err := appendEditorDisplay(list, view, instance, geometry, theme, textEngine, scaleX, scaleY, textSourceRemaining, reuse, editors[instance.ID], visual); err != nil {
			return fmt.Errorf("dxui: display %s editor: %w", path, err)
		}
		if view.node.kind == viewInput {
			if err := appendPasswordToggleDisplay(list, view.node.input, instance, geometry, theme, scaleX, scaleY, textSourceRemaining, reuse, visual); err != nil {
				return fmt.Errorf("dxui: display %s password toggle: %w", path, err)
			}
		}
	}
	if view.node.kind == viewSelect {
		if err := appendSelectAnchor(list, view, instance, geometry, theme, textEngine, scaleX, scaleY, textSourceRemaining, reuse, visual); err != nil {
			return fmt.Errorf("dxui: display %s select: %w", path, err)
		}
	}
	if view.node.kind == viewTabs {
		if err := appendTabsDisplay(list, view.node.tabs, instance, geometry, theme, textEngine, scaleX, scaleY, textSourceRemaining, reuse, visual); err != nil {
			return fmt.Errorf("dxui: display %s tabs: %w", path, err)
		}
	}
	if view.node.kind == viewMenu {
		if err := appendMenuDisplay(list, view.node.menu, instance, geometry, theme, textEngine, scaleX, scaleY, textSourceRemaining, reuse, visual); err != nil {
			return fmt.Errorf("dxui: display %s menu: %w", path, err)
		}
	}
	if view.node.kind == viewText && visual.textColorSet {
		if textEngine == nil {
			return fmt.Errorf("dxui: display %s has no text engine", path)
		}
		request, requestErr := resolvedTextRequest(retainedButtonText(*view.node.text, instance), theme)
		if requestErr != nil {
			return fmt.Errorf("dxui: display %s text style: %w", path, requestErr)
		}
		if request.Wrap == internaltext.WordWrap {
			request.MaxWidth, request.MaxWidthSet = geometry.Content.Width, true
		}
		textLayout, layoutErr := textEngine.Layout(request)
		if layoutErr != nil {
			return fmt.Errorf("dxui: display %s text layout: %w", path, layoutErr)
		}
		bitmaps, rasterErr := textEngine.RasterizeLinesWithinReuse(textLayout, scaleX, scaleY, *textSourceRemaining, reuseMaskFunc(reuse))
		if rasterErr != nil {
			return fmt.Errorf("dxui: display %s text raster: %w", path, rasterErr)
		}
		lineY := make([]float32, len(textLayout.Lines))
		for index := 1; index < len(textLayout.Lines); index++ {
			lineY[index] = lineY[index-1] + textLayout.Lines[index-1].Height
		}
		for index := range bitmaps {
			bitmap := &bitmaps[index]
			*textSourceRemaining -= bitmap.AccountedBytes
			line := textLayout.Lines[bitmap.LineIndex]
			x := geometry.Content.X
			switch request.Align {
			case internaltext.AlignCenter:
				x += (geometry.Content.Width - line.Width) / 2
			case internaltext.AlignEnd:
				x += geometry.Content.Width - line.Width
			}
			mask := &paint.TextBitmap{Key: bitmap.Key, Width: bitmap.Width, Height: bitmap.Height, Pixels: bitmap.Alpha}
			*list = append(*list, paint.Command{
				Kind: paint.CommandDrawText, NodeID: instance.ID,
				Rect: paint.Rect{X: x + bitmap.OffsetX, Y: geometry.Content.Y + lineY[bitmap.LineIndex], Width: bitmap.LogicalWidth, Height: bitmap.LogicalHeight},
				Text: mask, Color: paintColor(visual.textColor),
			})
		}
	}
	if view.node.kind == viewIcon {
		mask, maskRect, maskErr := rasterIconReuse(*view.node.icon, geometry.Content, theme, scaleX, scaleY, reuse)
		if maskErr != nil {
			return fmt.Errorf("dxui: display %s icon: %w", path, maskErr)
		}
		if maskErr = accountIconMask(mask, textSourceRemaining, reuse); maskErr != nil {
			return fmt.Errorf("dxui: display %s icon source budget: %w", path, maskErr)
		}
		iconColor := view.node.icon.Color
		if !iconColor.set {
			if inheritedTextSet && props.Token == "" {
				iconColor = LiteralColor(inheritedText)
			} else {
				iconColor = TokenColor(ColorSemanticText)
			}
		}
		resolved, colorErr := theme.color(iconColor)
		if colorErr != nil {
			return fmt.Errorf("dxui: display %s icon color: %w", path, colorErr)
		}
		*list = append(*list, paint.Command{Kind: paint.CommandDrawIcon, NodeID: instance.ID, Rect: maskRect, Text: mask, Color: paintColor(resolved)})
	}
	if view.node.kind == viewImage {
		bitmap, decodeErr := decodeImage(images, *view.node.image)
		if decodeErr != nil {
			bitmap = imagePlaceholder(view.node.image.Source.source.id)
		}
		destination := fittedImageRect(geometry.Content, bitmap.Width, bitmap.Height, view.node.image.Fit, view.node.image.Alignment)
		imageBitmap := &paint.ImageBitmap{Key: fmt.Sprintf("image:%d:%d", bitmap.Key.SourceID, bitmap.Key.Pixels), Width: bitmap.Width, Height: bitmap.Height, Pixels: bitmap.Pixels}
		crop := view.node.image.Fit == ImageCover
		if crop {
			*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: paint.Rect{X: geometry.Content.X, Y: geometry.Content.Y, Width: geometry.Content.Width, Height: geometry.Content.Height}})
		}
		*list = append(*list, paint.Command{Kind: paint.CommandDrawImage, NodeID: instance.ID, Rect: destination, Image: imageBitmap})
		if crop {
			*list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: instance.ID})
		}
	}
	if view.node.kind == viewAvatar {
		imageProps := imagePropsForAvatar(view.node.avatar)
		bitmap, decodeErr := decodeImage(images, imageProps)
		if decodeErr != nil {
			bitmap = imagePlaceholder(view.node.avatar.Source.source.id)
		}
		content := centeredSquare(paint.Rect{X: geometry.Content.X, Y: geometry.Content.Y, Width: geometry.Content.Width, Height: geometry.Content.Height})
		destination := fittedImageRect(layout.Rect{X: content.X, Y: content.Y, Width: content.Width, Height: content.Height}, bitmap.Width, bitmap.Height, ImageCover, Point{X: .5, Y: .5})
		imageBitmap := &paint.ImageBitmap{Key: fmt.Sprintf("image:%d:%d", bitmap.Key.SourceID, bitmap.Key.Pixels), Width: bitmap.Width, Height: bitmap.Height, Pixels: bitmap.Pixels}
		clip := content
		if visual.border.width > 0 && visual.border.colorSet {
			inset := min(visual.border.width, min(clip.Width, clip.Height)/2)
			clip.X, clip.Y = clip.X+inset, clip.Y+inset
			clip.Width, clip.Height = max(0, clip.Width-2*inset), max(0, clip.Height-2*inset)
		}
		if view.node.avatar.Shape == AvatarCircle {
			radius := min(clip.Width, clip.Height) / 2
			radii := paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
			*list = append(*list, paint.Command{Kind: paint.CommandDrawImage, NodeID: instance.ID, Rect: destination, ImageClip: clip, Radii: radii, Image: imageBitmap})
		} else {
			*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: clip})
			*list = append(*list, paint.Command{Kind: paint.CommandDrawImage, NodeID: instance.ID, Rect: destination, Image: imageBitmap})
			*list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: instance.ID})
		}
	}
	if len(view.node.children) != 0 {
		if len(view.node.children) != len(instance.Children) || len(view.node.children) != len(geometry.Children) {
			return fmt.Errorf("dxui: display %s tree/geometry child mismatch", path)
		}
		order := paintChildOrder(view.node.children)
		childCount := len(view.node.children)
		if view.node.kind == viewPopover || view.node.kind == viewTooltip {
			childCount = 1
			order = nil
		}
		for position := 0; position < childCount; position++ {
			index := position
			if order != nil {
				index = order[position]
			}
			childGeometry := rawGeometry.Children[index]
			if index < 0 || index >= len(view.node.children) {
				return fmt.Errorf("dxui: display %s has invalid paint source index %d", path, index)
			}
			childText, childTextSet := inheritedText, inheritedTextSet
			if visual.textColorSet {
				childText, childTextSet = visual.textColor, true
			}
			childTranslateX, childTranslateY := translateX, translateY
			if isScrollView(view) {
				childTranslateX -= instance.State.ScrollX
				childTranslateY -= instance.State.ScrollY
			}
			if err := appendDisplay(list, view.node.children[index], instance.Children[index], childGeometry, theme, textEngine, images, scaleX, scaleY, textSourceRemaining, reuse, editors, worldOpacity, clip, clipSet, childText, childTextSet, childTranslateX, childTranslateY, path); err != nil {
				return fmt.Errorf("dxui: display %s child %d: %w", path, index, err)
			}
		}
	}
	if isScrollView(view) {
		if err := appendScrollbarDisplay(list, view, instance, geometry, theme); err != nil {
			return fmt.Errorf("dxui: display %s scrollbar: %w", path, err)
		}
	}
	if clipped {
		*list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: instance.ID})
	}
	if visual.opacity != 1 {
		*list = append(*list, paint.Command{Kind: paint.CommandPopOpacity, NodeID: instance.ID})
	}
	return nil
}

func appendDashedBorderDisplay(list *paint.DisplayList, rect paint.Rect, radii paint.Radii, width float32, color paint.Color, nodeID uint64) {
	if width <= 0 || color.A == 0 || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	limit := min(rect.Width, rect.Height) / 2
	topLeft := max(width/2, min(radii.TopLeft, limit))
	topRight := max(width/2, min(radii.TopRight, limit))
	bottomRight := max(width/2, min(radii.BottomRight, limit))
	bottomLeft := max(width/2, min(radii.BottomLeft, limit))
	appendDashedBorderLine(list, paint.Rect{X: rect.X + topLeft, Y: rect.Y, Width: max(0, rect.Width-topLeft-topRight), Height: width}, true, color, nodeID)
	appendDashedBorderLine(list, paint.Rect{X: rect.X + bottomLeft, Y: rect.Y + rect.Height - width, Width: max(0, rect.Width-bottomLeft-bottomRight), Height: width}, true, color, nodeID)
	appendDashedBorderLine(list, paint.Rect{X: rect.X, Y: rect.Y + topLeft, Width: width, Height: max(0, rect.Height-topLeft-bottomLeft)}, false, color, nodeID)
	appendDashedBorderLine(list, paint.Rect{X: rect.X + rect.Width - width, Y: rect.Y + topRight, Width: width, Height: max(0, rect.Height-topRight-bottomRight)}, false, color, nodeID)
	appendDashedBorderArc(list, rect.X+topLeft, rect.Y+topLeft, topLeft, math.Pi, 3*math.Pi/2, width, color, nodeID)
	appendDashedBorderArc(list, rect.X+rect.Width-topRight, rect.Y+topRight, topRight, 3*math.Pi/2, 2*math.Pi, width, color, nodeID)
	appendDashedBorderArc(list, rect.X+rect.Width-bottomRight, rect.Y+rect.Height-bottomRight, bottomRight, 0, math.Pi/2, width, color, nodeID)
	appendDashedBorderArc(list, rect.X+bottomLeft, rect.Y+rect.Height-bottomLeft, bottomLeft, math.Pi/2, math.Pi, width, color, nodeID)
}

func appendDashedBorderLine(list *paint.DisplayList, line paint.Rect, horizontal bool, color paint.Color, nodeID uint64) {
	length := line.Height
	thickness := line.Width
	if horizontal {
		length, thickness = line.Width, line.Height
	}
	if length <= 0 {
		return
	}
	dash, gap := max(4, thickness*3), max(3, thickness*2)
	if periods := int(math.Ceil(float64(length / (dash + gap)))); periods > 64 {
		scale := float32(periods) / 64
		dash, gap = dash*scale, gap*scale
	}
	for offset := float32(0); offset < length; offset += dash + gap {
		segment := min(dash, length-offset)
		piece := paint.Rect{X: line.X, Y: line.Y, Width: thickness, Height: segment}
		if horizontal {
			piece.X += offset
			piece.Width, piece.Height = segment, thickness
		} else {
			piece.Y += offset
		}
		radius := min(thickness/2, segment/2)
		*list = append(*list, paint.Command{
			Kind: paint.CommandFillRoundedRect, NodeID: nodeID, Rect: piece,
			Radii: paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}, Color: color,
		})
	}
}

func appendDashedBorderArc(list *paint.DisplayList, centerX, centerY, radius, startAngle, endAngle, width float32, color paint.Color, nodeID uint64) {
	pathRadius := max(0, radius-width/2)
	arcLength := float64(endAngle-startAngle) * float64(pathRadius)
	if arcLength <= 0 {
		appendDashedBorderDot(list, centerX, centerY, width, color, nodeID)
		return
	}
	dash, gap := max(4, width*3), max(3, width*2)
	samples := min(64, max(1, int(math.Ceil(arcLength/float64(max(.75, width*.6))))))
	for sample := 0; sample <= samples; sample++ {
		distance := arcLength * float64(sample) / float64(samples)
		if math.Mod(distance, float64(dash+gap)) > float64(dash) {
			continue
		}
		angle := float64(startAngle) + float64(endAngle-startAngle)*float64(sample)/float64(samples)
		x := centerX + float32(math.Cos(angle))*pathRadius
		y := centerY + float32(math.Sin(angle))*pathRadius
		appendDashedBorderDot(list, x, y, width, color, nodeID)
	}
}

func appendDashedBorderDot(list *paint.DisplayList, centerX, centerY, width float32, color paint.Color, nodeID uint64) {
	radius := width / 2
	*list = append(*list, paint.Command{
		Kind: paint.CommandFillRoundedRect, NodeID: nodeID,
		Rect:  paint.Rect{X: centerX - radius, Y: centerY - radius, Width: width, Height: width},
		Radii: paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}, Color: color,
	})
}

func appendSelectedBorderDisplay(list *paint.DisplayList, rect paint.Rect, radii paint.Radii, border resolvedBorder, nodeID uint64) {
	sides := border.sides
	if sides == 0 || sides == BorderAll {
		appendDashedBorderDisplay(list, rect, radii, border.width, paintColor(border.color), nodeID)
		return
	}
	*list = append(*list, paint.Command{Kind: paint.CommandBorder, NodeID: nodeID,
		Rect: rect, Radii: radii, Width: border.width, Color: paintColor(border.color),
		Sides: paint.BorderSides(sides), Dashed: border.pattern == BorderDashed})
}

func centeredSquare(rect paint.Rect) paint.Rect {
	side := min(rect.Width, rect.Height)
	return paint.Rect{X: rect.X + (rect.Width-side)/2, Y: rect.Y + (rect.Height-side)/2, Width: side, Height: side}
}

func appendEditorDisplay(list *paint.DisplayList, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme, engine *internaltext.Engine, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse, state editorDisplayState, visual computedVisual) error {
	if engine == nil {
		return fmt.Errorf("missing text engine")
	}
	action := inputAction{editor: true, geometry: geometry.Rect, content: geometry.Content}
	placeholder := ""
	if view.node.kind == viewInput {
		p := view.node.input
		action.node, action.value, action.password, action.passwordVisible, action.placeholder = p.common(), p.Value, p.Password, p.Password && p.ShowPasswordToggle && instance.State.PasswordVisible, p.Placeholder
	} else {
		p := view.node.textarea
		action.node, action.value, action.multiline, action.wrap, action.placeholder = p.common(), p.Value, true, p.Wrap, p.Placeholder
	}
	displayValue := action.value
	if !action.multiline {
		displayValue = strings.NewReplacer("\r", " ", "\n", " ").Replace(displayValue)
	}
	if action.password && !action.passwordVisible {
		displayValue = strings.Repeat("•", len([]rune(action.value)))
	}
	placeholderVisible := displayValue == "" && !state.composition.Active && action.placeholder != ""
	if placeholderVisible {
		placeholder, displayValue = action.placeholder, action.placeholder
	}
	layoutAction := action
	layoutAction.value = displayValue
	visualLayout, err := layoutEditorWith(engine, theme, layoutAction)
	if err != nil {
		return err
	}
	content := paint.Rect{X: geometry.Content.X, Y: geometry.Content.Y, Width: geometry.Content.Width, Height: geometry.Content.Height}
	*list = append(*list, paint.Command{Kind: paint.CommandPushClip, NodeID: instance.ID, Rect: content})
	defer func() { *list = append(*list, paint.Command{Kind: paint.CommandPopClip, NodeID: instance.ID}) }()
	selectionColor, err := theme.color(TokenColor(ColorSemanticAccent))
	if err != nil {
		return err
	}
	selectionColor.A = 80
	if !placeholderVisible {
		start, end := internalinput.Ordered(state.selection)
		for _, line := range visualLayout.lines {
			from, to := max(start, line.start), min(end, line.end)
			if from >= to {
				continue
			}
			x1, x2 := line.x[from-line.start], line.x[to-line.start]
			*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID,
				Rect: paint.Rect{X: content.X + x1 - state.scrollX, Y: content.Y + line.y - state.scrollY, Width: x2 - x1, Height: line.height}, Color: paintColor(selectionColor)})
		}
	}
	textColor := visual.textColor
	if !visual.textColorSet {
		textColor = RGBA(0, 0, 0, 255)
	}
	if placeholderVisible {
		textColor.A /= 2
	}
	runes := []rune(displayValue)
	request, err := resolvedTextRequest(textPropsFromCommon(action.node, "", TextNoWrap, 0), theme)
	if err != nil {
		return err
	}
	request.Wrap = internaltext.NoWrap
	for _, line := range visualLayout.lines {
		if line.start < 0 || line.end > len(runes) || line.start == line.end {
			continue
		}
		request.Text = string(runes[line.start:line.end])
		laidOut, layoutErr := engine.Layout(request)
		if layoutErr != nil {
			return layoutErr
		}
		bitmaps, rasterErr := engine.RasterizeLinesWithinReuse(laidOut, scaleX, scaleY, *remaining, reuseMaskFunc(reuse))
		if rasterErr != nil {
			return rasterErr
		}
		for i := range bitmaps {
			bitmap := &bitmaps[i]
			*remaining -= bitmap.AccountedBytes
			mask := &paint.TextBitmap{Key: bitmap.Key, Width: bitmap.Width, Height: bitmap.Height, Pixels: bitmap.Alpha}
			*list = append(*list, paint.Command{Kind: paint.CommandDrawText, NodeID: instance.ID,
				Rect: paint.Rect{X: content.X + bitmap.OffsetX - state.scrollX, Y: content.Y + line.y - state.scrollY, Width: bitmap.LogicalWidth, Height: bitmap.LogicalHeight}, Text: mask, Color: paintColor(textColor)})
		}
	}
	caretX, caretY, caretHeight := editorCaret(visualLayout, state.selection.End)
	if state.composition.Active {
		request.Text = state.composition.Text
		laidOut, layoutErr := engine.Layout(request)
		if layoutErr != nil {
			return layoutErr
		}
		bitmaps, rasterErr := engine.RasterizeLinesWithinReuse(laidOut, scaleX, scaleY, *remaining, reuseMaskFunc(reuse))
		if rasterErr != nil {
			return rasterErr
		}
		var compositionWidth float32
		if len(laidOut.Lines) != 0 {
			compositionWidth = laidOut.Lines[0].Width
		}
		for i := range bitmaps {
			bitmap := &bitmaps[i]
			*remaining -= bitmap.AccountedBytes
			*list = append(*list, paint.Command{Kind: paint.CommandDrawText, NodeID: instance.ID,
				Rect: paint.Rect{X: content.X + caretX + bitmap.OffsetX - state.scrollX, Y: content.Y + caretY - state.scrollY, Width: bitmap.LogicalWidth, Height: bitmap.LogicalHeight},
				Text: &paint.TextBitmap{Key: bitmap.Key, Width: bitmap.Width, Height: bitmap.Height, Pixels: bitmap.Alpha}, Color: paintColor(textColor)})
		}
		*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID,
			Rect: paint.Rect{X: content.X + caretX - state.scrollX, Y: content.Y + caretY + caretHeight - 1 - state.scrollY, Width: max(1, compositionWidth), Height: 1}, Color: paintColor(textColor)})
	}
	if instance.State.Focused && state.caretVisible {
		*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID,
			Rect: paint.Rect{X: content.X + caretX - state.scrollX, Y: content.Y + caretY - state.scrollY, Width: 1, Height: caretHeight}, Color: paintColor(textColor)})
	}
	_ = placeholder
	return nil
}

func appendScrollbarDisplay(list *paint.DisplayList, view View, instance *tree.Node, geometry *layout.Result, theme resolvedTheme) error {
	thickness, err := theme.metric(TokenMetric(MetricComponentScrollThickness))
	if err != nil {
		return err
	}
	minimum, err := theme.metric(TokenMetric(MetricComponentScrollMinThumb))
	if err != nil {
		return err
	}
	inset, err := theme.metric(TokenMetric(MetricComponentScrollInset))
	if err != nil {
		return err
	}
	trackColor, err := theme.color(TokenColor(ColorSemanticScrollTrack))
	if err != nil {
		return err
	}
	for _, bar := range scrollbarsFor(geometry.Viewport, geometry.ContentExtent, Point{X: instance.State.ScrollX, Y: instance.State.ScrollY}, view.node.scroll.Axis, view.node.scroll.Scrollbar, thickness, minimum, inset) {
		thumbToken := ColorSemanticScrollThumb
		expanded := false
		if bar.Disabled {
			thumbToken = ColorSemanticScrollThumbDisabled
		} else if instance.State.ScrollbarDrag == bar.Axis {
			thumbToken = ColorSemanticScrollThumbActive
			expanded = true
		} else if instance.State.ScrollbarHover == bar.Axis {
			thumbToken = ColorSemanticScrollThumbHover
			expanded = true
		}
		thumbColor, colorErr := theme.color(TokenColor(thumbToken))
		if colorErr != nil {
			return colorErr
		}
		if !expanded && !bar.Disabled {
			thumbColor.A = uint8(float32(thumbColor.A) * scrollbarIdleOpacityRatio)
		}
		thumb := scrollbarVisualThumb(bar.Thumb, bar.Axis, expanded)
		visualThickness := thumb.Width
		if bar.Axis == scrollbarAxisHorizontal {
			visualThickness = thumb.Height
		}
		trackRadius := paint.Radii{TopLeft: thickness / 2, TopRight: thickness / 2, BottomRight: thickness / 2, BottomLeft: thickness / 2}
		thumbRadius := paint.Radii{TopLeft: visualThickness / 2, TopRight: visualThickness / 2, BottomRight: visualThickness / 2, BottomLeft: visualThickness / 2}
		*list = append(*list,
			paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: bar.Track, Radii: trackRadius, Color: paintColor(trackColor)},
			paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: instance.ID, Rect: thumb, Radii: thumbRadius, Color: paintColor(thumbColor)},
		)
	}
	return nil
}

func nodeProps(view View) (viewProps, error) {
	switch view.node.kind {
	case viewBox:
		return view.node.box.common(), nil
	case viewText:
		return view.node.text.common(), nil
	case viewButton:
		return view.node.button.common(), nil
	case viewButtonGroup:
		return view.node.buttonGroup.common(), nil
	case viewInputGroup:
		return view.node.inputGroup.common(), nil
	case viewBadge:
		return view.node.badge.common(), nil
	case viewProgressBar:
		return view.node.progress.common(), nil
	case viewInput:
		return view.node.input.common(), nil
	case viewTextarea:
		return view.node.textarea.common(), nil
	case viewToggleSwitch:
		return view.node.toggle.common(), nil
	case viewSlider:
		return view.node.slider.common(), nil
	case viewCheckbox:
		return view.node.checkbox.common(), nil
	case viewRadio:
		return view.node.radio.common(), nil
	case viewIcon:
		return view.node.icon.common(), nil
	case viewImage:
		return view.node.image.common(), nil
	case viewAvatar:
		return view.node.avatar.common(), nil
	case viewControlMark:
		return view.node.mark.common(), nil
	case viewScroll:
		return view.node.scroll.common(), nil
	case viewVirtualList:
		return view.node.virtualList.common(), nil
	case viewSelect:
		return view.node.selectp.common(), nil
	case viewTabs:
		return view.node.tabs.common(), nil
	case viewMenu:
		return view.node.menu.common(), nil
	case viewPopover:
		return view.node.popover.common(), nil
	case viewTooltip:
		return view.node.tooltip.common(), nil
	default:
		return viewProps{}, fmt.Errorf("unsupported view kind %d", view.node.kind)
	}
}

func paintChildOrder(children []View) []int {
	previous := 0
	for index, child := range children {
		z := viewZIndex(child)
		if index != 0 && z < previous {
			return sortedPaintChildOrder(children)
		}
		previous = z
	}
	return nil
}

func sortedPaintChildOrder(children []View) []int {
	order := make([]int, len(children))
	z := make([]int, len(children))
	for index := range order {
		order[index] = index
		z[index] = viewZIndex(children[index])
	}
	for index := 1; index < len(order); index++ {
		value := order[index]
		position := index
		for position > 0 && z[value] < z[order[position-1]] {
			order[position] = order[position-1]
			position--
		}
		order[position] = value
	}
	return order
}

func viewZIndex(view View) int {
	props, err := nodeProps(view)
	if err != nil {
		return 0
	}
	return props.Style.ZIndex
}

func computeVisual(props viewProps, instance *tree.Node, kind viewKind, theme resolvedTheme, groupStyles ...*buttonGroupChildStyle) (computedVisual, error) {
	result := computedVisual{opacity: 1, visibility: Visible}
	if kind == viewText {
		result.textColor, result.textColorSet = RGBA(0, 0, 0, 255), true
		if _, ok := theme.source.Semantic.Colors[ColorSemanticText]; ok {
			color, err := theme.color(TokenColor(ColorSemanticText))
			if err != nil {
				return computedVisual{}, err
			}
			result.textColor = color
		}
	}
	component := ComponentTheme{}
	token := props.Token
	if token == "" {
		token = defaultComponentToken(kind)
	}
	if token != "" {
		var ok bool
		component, ok = theme.source.Components[token]
		if !ok {
			return computedVisual{}, fmt.Errorf("theme: missing component token %q", token)
		}
	}
	if kind == viewButton && props.Token == "" {
		var err error
		component, err = buttonRecipe(theme, props.buttonVariant, props.buttonTone)
		if err != nil {
			return computedVisual{}, err
		}
	}
	state := visualState{
		Hover: instance.State.Hovered, Focus: instance.State.FocusVisible,
		Pressed: instance.State.Pressed || kind == viewSlider && instance.State.SliderDragging, Checked: instance.State.Checked,
		Disabled: instance.Properties.Semantics.Disabled,
	}
	if kind == viewTabs {
		state.Hover = instance.State.TabsHover >= 0
		state.Pressed = instance.State.TabsPressed >= 0
	}
	if kind == viewMenu {
		state.Hover = instance.State.MenuHover >= 0
		state.Pressed = instance.State.MenuPressed >= 0
	}
	base := component.Base
	if len(groupStyles) != 0 && groupStyles[0] != nil {
		groupComponent, ok := theme.source.Components[ComponentButtonGroup]
		if !ok {
			return computedVisual{}, fmt.Errorf("theme: missing component token %q", ComponentButtonGroup)
		}
		base = mergeStylePatch(base, buttonGroupChildPatch(groupComponent.Base, *groupStyles[0]))
		child := groupStyles[0]
		if child.orientation == ButtonGroupVertical {
			if child.index > 0 {
				result.joinedEdges |= paint.JoinedTop
			}
			if child.index+1 < child.count {
				result.joinedEdges |= paint.JoinedBottom
			}
		} else {
			if child.index > 0 {
				result.joinedEdges |= paint.JoinedLeft
			}
			if child.index+1 < child.count {
				result.joinedEdges |= paint.JoinedRight
			}
		}
	}
	patch := mergeStylePatch(base, component.States.Default)
	if err := applyResolvedPatch(&result, patch, theme); err != nil {
		return computedVisual{}, err
	}
	style := props.Style
	if err := applyLocalStyle(&result, style, theme); err != nil {
		return computedVisual{}, err
	}
	statePatch := mergeStylePatch(props.States.Default, resolvedActiveStatePatch(component.States, props.States, state, applicability(kind)))
	if err := applyResolvedPatch(&result, statePatch, theme); err != nil {
		return computedVisual{}, err
	}
	if err := applyResolvedPatch(&result, style.Force, theme); err != nil {
		return computedVisual{}, err
	}
	if kind == viewButton && state.Focus && !state.Disabled {
		final := patch
		if style.Shadow != nil {
			final.Shadow = Some(style.Shadow)
		}
		final = mergeStylePatch(final, statePatch)
		final = mergeStylePatch(final, style.Force)
		result.buttonFocusRing = (!result.backgroundSet || result.background.A < 255) && isButtonFocusRing(final)
	}
	return result, nil
}

func applyLocalStyle(result *computedVisual, style Style, theme resolvedTheme) error {
	if style.Background.set {
		var err error
		result.background, err = theme.color(style.Background)
		if err != nil {
			return err
		}
		result.backgroundSet = true
	}
	if style.Border.Width.set || style.Border.Color.set {
		var err error
		result.border.width, err = theme.metric(style.Border.Width)
		if err != nil {
			return err
		}
		if style.Border.Color.set {
			result.border.color, err = theme.color(style.Border.Color)
			if err != nil {
				return err
			}
			result.border.colorSet = true
		}
		result.border.pattern = style.Border.Pattern
		result.border.sides = style.Border.Sides
	} else if style.Border.Pattern != BorderSolid {
		result.border.pattern = style.Border.Pattern
	}
	if style.Border.Sides != 0 {
		result.border.sides = style.Border.Sides
	}
	if cornersSet(style.Radius) {
		var err error
		result.radii, err = resolveRadii(style.Radius, theme)
		if err != nil {
			return err
		}
	}
	if style.Shadow != nil {
		var err error
		result.shadows, err = resolveShadows(style.Shadow, theme)
		if err != nil {
			return err
		}
	}
	if opacity, set := style.Opacity.get(); set {
		result.opacity = opacity
	}
	if style.Visibility == Hidden {
		result.visibility = Hidden
	}
	if style.Text.Color.set {
		var err error
		result.textColor, err = theme.color(style.Text.Color)
		if err != nil {
			return err
		}
		result.textColorSet = true
	}
	return nil
}

func buttonGroupChildPatch(base StylePatch, child buttonGroupChildStyle) StylePatch {
	if !child.dividers {
		base.Border = Option[Border]{}
	}
	radius, set := base.Radius.get()
	if !set || child.count <= 1 {
		return base
	}
	zero := Metric(0)
	if child.orientation == ButtonGroupVertical {
		if child.index > 0 {
			radius.TopLeft, radius.TopRight = zero, zero
		}
		if child.index+1 < child.count {
			radius.BottomLeft, radius.BottomRight = zero, zero
		}
	} else {
		if child.index > 0 {
			radius.TopLeft, radius.BottomLeft = zero, zero
		}
		if child.index+1 < child.count {
			radius.TopRight, radius.BottomRight = zero, zero
		}
	}
	base.Radius = Some(radius)
	return base
}

func defaultComponentToken(kind viewKind) ComponentToken {
	switch kind {
	case viewButton:
		return ComponentButton
	case viewBadge:
		return ComponentBadge
	case viewInputGroup:
		return ComponentInputGroup
	case viewInput, viewTextarea:
		return ComponentInput
	case viewToggleSwitch:
		return ComponentToggleSwitch
	case viewSlider:
		return ComponentSlider
	case viewProgressBar:
		return ComponentProgressBar
	case viewCheckbox:
		return ComponentCheckbox
	case viewRadio:
		return ComponentRadio
	case viewIcon:
		return ComponentIcon
	case viewImage:
		return ComponentImage
	case viewAvatar:
		return ComponentAvatar
	case viewScroll, viewVirtualList:
		return ComponentScroll
	case viewSelect:
		return ComponentSelect
	case viewTabs:
		return ComponentTabs
	case viewMenu:
		return ComponentMenu
	case viewPopover, viewTooltip:
		return ""
	default:
		return ""
	}
}

func applicability(kind viewKind) stateApplicability {
	switch kind {
	case viewButton:
		return stateHover | stateFocus | stateDisabled | statePressed
	case viewInput, viewTextarea, viewSelect, viewInputGroup:
		return stateHover | stateFocus | stateDisabled
	case viewTabs:
		return stateHover | stateFocus | stateDisabled | statePressed
	case viewMenu:
		return stateHover | stateFocus | stateDisabled | statePressed
	case viewToggleSwitch, viewCheckbox, viewRadio:
		return stateHover | stateFocus | stateDisabled | statePressed | stateChecked
	case viewSlider:
		return stateHover | stateFocus | stateDisabled | statePressed
	default:
		return 0
	}
}

func applyResolvedPatch(result *computedVisual, patch StylePatch, theme resolvedTheme) error {
	if value, set := patch.Background.get(); set {
		resolved, err := theme.color(value)
		if err != nil {
			return err
		}
		result.background, result.backgroundSet = resolved, true
	}
	if value, set := patch.Border.get(); set {
		width, err := theme.metric(value.Width)
		if err != nil {
			return err
		}
		result.border.width = width
		result.border.pattern = value.Pattern
		result.border.sides = value.Sides
		if value.Color.set {
			color, colorErr := theme.color(value.Color)
			if colorErr != nil {
				return colorErr
			}
			result.border.color, result.border.colorSet = color, true
		} else {
			result.border.colorSet = false
		}
	}
	if value, set := patch.Radius.get(); set {
		resolved, err := resolveRadii(value, theme)
		if err != nil {
			return err
		}
		result.radii = resolved
	}
	if value, set := patch.Shadow.get(); set {
		resolved, err := resolveShadows(value, theme)
		if err != nil {
			return err
		}
		result.shadows = resolved
	}
	if value, set := patch.Opacity.get(); set {
		result.opacity = value
	}
	if value, set := patch.Visibility.get(); set {
		result.visibility = value
	}
	if value, set := patch.TextColor.get(); set {
		resolved, err := theme.color(value)
		if err != nil {
			return err
		}
		result.textColor, result.textColorSet = resolved, true
	}
	return nil
}

func cornersSet(value CornerValues) bool {
	return value.TopLeft.set || value.TopRight.set || value.BottomRight.set || value.BottomLeft.set
}

func resolveRadii(value CornerValues, theme resolvedTheme) (paint.Radii, error) {
	values := []MetricValue{value.TopLeft, value.TopRight, value.BottomRight, value.BottomLeft}
	resolved := [4]float32{}
	for index, metric := range values {
		var err error
		resolved[index], err = theme.metric(metric)
		if err != nil {
			return paint.Radii{}, err
		}
	}
	return paint.Radii{TopLeft: resolved[0], TopRight: resolved[1], BottomRight: resolved[2], BottomLeft: resolved[3]}, nil
}

func resolveShadows(values []Shadow, theme resolvedTheme) ([]paint.Shadow, error) {
	result := make([]paint.Shadow, len(values))
	for index, value := range values {
		metrics := []*float32{&result[index].OffsetX, &result[index].OffsetY, &result[index].Blur, &result[index].Spread}
		inputs := []MetricValue{value.OffsetX, value.OffsetY, value.Blur, value.Spread}
		for metricIndex := range metrics {
			resolved, err := theme.metric(inputs[metricIndex])
			if err != nil {
				return nil, err
			}
			*metrics[metricIndex] = resolved
		}
		color, err := theme.color(value.Color)
		if err != nil {
			return nil, err
		}
		result[index].Color = paintColor(color)
	}
	return result, nil
}

func visualBounds(rect paint.Rect, shadows []paint.Shadow) paint.Rect {
	result := rect
	for _, shadow := range shadows {
		if shadow.Color.A == 0 {
			continue
		}
		expansion := shadow.Blur + shadow.Spread
		result = paint.Union(result, paint.Rect{
			X: rect.X + shadow.OffsetX - expansion, Y: rect.Y + shadow.OffsetY - expansion,
			Width: rect.Width + 2*expansion, Height: rect.Height + 2*expansion,
		})
	}
	return result
}

func paintColor(value RGBAColor) paint.Color {
	return paint.Color{R: value.R, G: value.G, B: value.B, A: value.A}
}

func appendToggleDisplay(list *paint.DisplayList, rect paint.Rect, checked bool, theme resolvedTheme, nodeID uint64) error {
	inset, err := theme.metric(TokenMetric(MetricComponentToggleKnobInset))
	if err != nil {
		return err
	}
	knob := max(0, rect.Height-2*inset)
	x := rect.X + inset
	if checked {
		x = rect.X + rect.Width - inset - knob
	}
	colorValue, err := theme.color(TokenColor(ColorSemanticSurfaceHi))
	if err != nil {
		return err
	}
	*list = append(*list, paint.Command{
		Kind: paint.CommandFillRoundedRect, NodeID: nodeID,
		Rect:  paint.Rect{X: x, Y: rect.Y + inset, Width: knob, Height: knob},
		Radii: paint.Radii{TopLeft: knob / 2, TopRight: knob / 2, BottomRight: knob / 2, BottomLeft: knob / 2},
		Color: paintColor(colorValue),
	})
	return nil
}

func appendSliderDisplay(list *paint.DisplayList, rect paint.Rect, model sliderModel, theme resolvedTheme, nodeID uint64) error {
	trackHeight, err := theme.metric(TokenMetric(MetricComponentSliderTrackHeight))
	if err != nil {
		return err
	}
	thumbSize, err := theme.metric(TokenMetric(MetricComponentSliderThumbSize))
	if err != nil {
		return err
	}
	trackHeight = min(max(float32(0), trackHeight), max(float32(0), rect.Height))
	thumbSize = min(max(float32(0), thumbSize), min(max(float32(0), rect.Width), max(float32(0), rect.Height)))
	trackColor, err := theme.color(TokenColor(ColorSemanticSliderTrack))
	if err != nil {
		return err
	}
	fillColor, err := theme.color(TokenColor(ColorSemanticSliderFill))
	if err != nil {
		return err
	}
	thumbColor, err := theme.color(TokenColor(ColorSemanticSliderThumb))
	if err != nil {
		return err
	}
	thumbBorder, err := theme.color(TokenColor(ColorSemanticSliderThumbBorder))
	if err != nil {
		return err
	}
	thumbBorderWidth, err := theme.metric(TokenMetric(MetricSemanticBorder))
	if err != nil {
		return err
	}
	start, end := sliderTravel(rect.X, rect.Width, thumbSize)
	center := start + (end-start)*model.fraction()
	track := paint.Rect{X: start, Y: rect.Y + (rect.Height-trackHeight)/2, Width: end - start, Height: trackHeight}
	radius := trackHeight / 2
	trackRadii := paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
	*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: nodeID, Rect: track, Radii: trackRadii, Color: paintColor(trackColor)})
	if center > start {
		filled := track
		filled.Width = center - start
		*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: nodeID, Rect: filled, Radii: trackRadii, Color: paintColor(fillColor)})
	}
	thumbRadius := thumbSize / 2
	*list = append(*list, paint.Command{
		Kind: paint.CommandFillStrokeRoundedRect, NodeID: nodeID,
		Rect:  paint.Rect{X: center - thumbRadius, Y: rect.Y + (rect.Height-thumbSize)/2, Width: thumbSize, Height: thumbSize},
		Radii: paint.Radii{TopLeft: thumbRadius, TopRight: thumbRadius, BottomRight: thumbRadius, BottomLeft: thumbRadius},
		Width: thumbBorderWidth, Color: paintColor(thumbColor), BorderColor: paintColor(thumbBorder),
	})
	return nil
}

func appendProgressBarDisplay(list *paint.DisplayList, content paint.Rect, value float32, visual computedVisual, theme resolvedTheme, nodeID uint64) error {
	trackHeight, err := theme.metric(TokenMetric(MetricComponentProgressBarTrackHeight))
	if err != nil {
		return err
	}
	trackHeight = min(max(float32(0), trackHeight), max(float32(0), content.Height))
	track := content
	track.Y += (track.Height - trackHeight) / 2
	track.Height = trackHeight
	if track.Width <= 0 || track.Height <= 0 {
		return nil
	}
	trackColor, err := theme.color(TokenColor(ColorSemanticProgressTrack))
	if err != nil {
		return err
	}
	if visual.backgroundSet {
		trackColor = visual.background
	}
	fillColor, err := theme.color(TokenColor(ColorSemanticProgressFill))
	if err != nil {
		return err
	}
	if visual.textColorSet {
		fillColor = visual.textColor
	}
	*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: nodeID, Rect: track, Radii: visual.radii, Color: paintColor(trackColor)})
	if value > 0 {
		filled := track
		filled.Width *= value
		*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: nodeID, Rect: filled, Radii: visual.radii, Color: paintColor(fillColor)})
	}
	return nil
}

var checkboxCheckIcon = icon.Check()

func appendCheckboxDisplay(list *paint.DisplayList, box paint.Rect, checked bool, theme resolvedTheme, nodeID uint64, scaleX, scaleY float32, remaining *int, reuse *textMaskReuse) error {
	size := min(box.Width, box.Height)
	backgroundToken := ColorSemanticSurfaceHi
	if checked {
		backgroundToken = ColorSemanticAccent
	}
	background, err := theme.color(TokenColor(backgroundToken))
	if err != nil {
		return err
	}
	border, err := theme.color(TokenColor(ColorSemanticBorder))
	if err != nil {
		return err
	}
	radius, err := theme.metric(TokenMetric(MetricSemanticRadius))
	if err != nil {
		return err
	}
	borderWidth, err := theme.metric(TokenMetric(MetricSemanticBorder))
	if err != nil {
		return err
	}
	radii := paint.Radii{TopLeft: radius / 2, TopRight: radius / 2, BottomRight: radius / 2, BottomLeft: radius / 2}
	*list = append(*list, paint.Command{Kind: paint.CommandFillStrokeRoundedRect, NodeID: nodeID, Rect: box, Radii: radii, Width: borderWidth, Color: paintColor(background), BorderColor: paintColor(border)})
	if checked {
		mark, colorErr := theme.color(TokenColor(ColorSemanticSurfaceHi))
		if colorErr != nil {
			return colorErr
		}
		mask, maskRect, maskErr := rasterIconReuse(IconProps{Data: checkboxCheckIcon, Size: size}, layout.Rect{
			X: box.X, Y: box.Y, Width: box.Width, Height: box.Height,
		}, theme, scaleX, scaleY, reuse)
		if maskErr != nil {
			return fmt.Errorf("check mark: %w", maskErr)
		}
		if maskErr = accountIconMask(mask, remaining, reuse); maskErr != nil {
			return fmt.Errorf("check mark source budget: %w", maskErr)
		}
		*list = append(*list, paint.Command{Kind: paint.CommandDrawIcon, NodeID: nodeID, Rect: maskRect, Text: mask, Color: paintColor(mark)})
	}
	return nil
}

func appendRadioDisplay(list *paint.DisplayList, box paint.Rect, selected bool, theme resolvedTheme, nodeID uint64) error {
	size := min(box.Width, box.Height)
	background, err := theme.color(TokenColor(ColorSemanticSurfaceHi))
	if err != nil {
		return err
	}
	borderToken := ColorSemanticBorder
	if selected {
		borderToken = ColorSemanticAccent
	}
	border, err := theme.color(TokenColor(borderToken))
	if err != nil {
		return err
	}
	borderWidth, err := theme.metric(TokenMetric(MetricSemanticBorder))
	if err != nil {
		return err
	}
	radius := size / 2
	radii := paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
	*list = append(*list, paint.Command{Kind: paint.CommandFillStrokeRoundedRect, NodeID: nodeID, Rect: box, Radii: radii, Width: borderWidth, Color: paintColor(background), BorderColor: paintColor(border)})
	if selected {
		dotSize := size / 2
		dot := paint.Rect{X: box.X + (box.Width-dotSize)/2, Y: box.Y + (box.Height-dotSize)/2, Width: dotSize, Height: dotSize}
		dotRadius := dotSize / 2
		*list = append(*list, paint.Command{Kind: paint.CommandFillRoundedRect, NodeID: nodeID, Rect: dot, Radii: paint.Radii{TopLeft: dotRadius, TopRight: dotRadius, BottomRight: dotRadius, BottomLeft: dotRadius}, Color: paintColor(border)})
	}
	return nil
}

func fittedImageRect(content layout.Rect, width, height int, fit ImageFit, alignment Point) paint.Rect {
	sourceWidth, sourceHeight := float32(width), float32(height)
	destinationWidth, destinationHeight := sourceWidth, sourceHeight
	switch fit {
	case ImageFill:
		destinationWidth, destinationHeight = content.Width, content.Height
	case ImageContain, ImageCover:
		if sourceWidth > 0 && sourceHeight > 0 {
			xScale, yScale := content.Width/sourceWidth, content.Height/sourceHeight
			scale := min(xScale, yScale)
			if fit == ImageCover {
				scale = max(xScale, yScale)
			}
			destinationWidth, destinationHeight = sourceWidth*scale, sourceHeight*scale
		}
	}
	return paint.Rect{
		X:     content.X + (content.Width-destinationWidth)*alignment.X,
		Y:     content.Y + (content.Height-destinationHeight)*alignment.Y,
		Width: destinationWidth, Height: destinationHeight,
	}
}

func imagePlaceholder(sourceID uint64) *internalimage.Bitmap {
	return &internalimage.Bitmap{Key: internalimage.Key{SourceID: sourceID}, Width: 2, Height: 2,
		Pixels: []byte{220, 38, 38, 255, 40, 40, 40, 255, 40, 40, 40, 255, 220, 38, 38, 255}}
}
