package sdl3

import (
	"fmt"
	"math"

	"github.com/dxui-org/dxui/internal/paint"
)

type nativeRect struct{ X, Y, Width, Height int32 }
type nativeFRect struct{ X, Y, Width, Height float32 }
type nativeVertex struct {
	X, Y       float32
	R, G, B, A float32
	U, V       float32
}

const edgeFringePixels = float32(.5)
const joinedOwnershipInsetPixels = float32(1.0 / 1024)
const curveTolerancePixels = float32(.2)
const maxRoundedSegments = 64
const shadowLayers = 8
const shadowCacheEntryOverhead = 128
const shadowLayerAccountedBytes = 64
const textTextureEntryOverhead = 128

type textTextureKey struct {
	Identity      string
	Width, Height int
}

type shadowKey struct {
	Width, Height  float32
	Radii          paint.Radii
	Blur, Spread   float32
	ScaleX, ScaleY float32
}

type shadowLayer struct {
	Rect  paint.Rect
	Radii paint.Radii
	Alpha float32
}

type shadowGeometry []shadowLayer

func (driver *Driver) SetClip(rect paint.Rect, set bool) error {
	if !set {
		return driver.renderer.setClipRect(nil)
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return driver.renderer.setClipRect(&nativeRect{})
	}
	left := int32(math.Floor(float64(rect.X)))
	top := int32(math.Floor(float64(rect.Y)))
	right := int32(math.Ceil(float64(rect.X + rect.Width)))
	bottom := int32(math.Ceil(float64(rect.Y + rect.Height)))
	return driver.renderer.setClipRect(&nativeRect{X: left, Y: top, Width: max(0, right-left), Height: max(0, bottom-top)})
}

func (driver *Driver) DrawShadow(rect paint.Rect, radii paint.Radii, shadow paint.Shadow) error {
	if rect.Width <= 0 || rect.Height <= 0 || shadow.Color.A == 0 {
		return nil
	}
	key := shadowKey{
		Width: rect.Width, Height: rect.Height, Radii: normalizeRadii(rect, radii),
		Blur: shadow.Blur, Spread: shadow.Spread, ScaleX: driver.scaleX, ScaleY: driver.scaleY,
	}
	layers, ok := driver.shadowCache.Get(key)
	if !ok {
		layers = makeShadowGeometry(key)
		driver.shadowCache.Put(key, layers, shadowGeometryBytes(len(layers)))
	}
	for _, layer := range layers {
		color := shadow.Color
		color.A = uint8(math.Round(float64(color.A) * float64(layer.Alpha)))
		drawRect := layer.Rect
		drawRect.X += rect.X + shadow.OffsetX
		drawRect.Y += rect.Y + shadow.OffsetY
		if err := driver.fillRounded(drawRect, layer.Radii, color, 0); err != nil {
			return err
		}
	}
	return nil
}

func shadowGeometryBytes(layers int) int {
	return shadowCacheEntryOverhead + layers*shadowLayerAccountedBytes
}

func makeShadowGeometry(key shadowKey) shadowGeometry {
	count := 1
	if key.Blur > 0 {
		count = shadowLayers
	}
	result := make(shadowGeometry, 0, count)
	for index := 0; index < count; index++ {
		fraction := float32(index+1) / float32(count)
		expansion := key.Spread + key.Blur*(1-fraction)
		alpha := float32(1)
		if count > 1 {
			alpha = 1 / float32(count)
		}
		result = append(result, shadowLayer{
			Rect:  paint.Rect{X: -expansion, Y: -expansion, Width: key.Width + 2*expansion, Height: key.Height + 2*expansion},
			Radii: addRadius(key.Radii, expansion), Alpha: alpha,
		})
	}
	return result
}

func (driver *Driver) FillRoundedRect(rect paint.Rect, radii paint.Radii, color paint.Color, joined paint.JoinedEdges) error {
	return driver.fillRounded(rect, radii, color, joined)
}

func (driver *Driver) StrokeRoundedRect(rect paint.Rect, radii paint.Radii, width float32, color paint.Color) error {
	if width <= 0 || rect.Width <= 0 || rect.Height <= 0 || color.A == 0 {
		return nil
	}
	radii = normalizeRadii(rect, radii)
	width = min(width, min(rect.Width, rect.Height)/2)
	if width <= 0 {
		return nil
	}
	vertices, indices := roundedRingMesh(rect, radii, width, color, driver.scaleX, driver.scaleY)
	if len(vertices) == 0 {
		return nil
	}
	if err := driver.renderer.renderGeometry(nil, vertices, indices); err != nil {
		return fmt.Errorf("stroke rounded rect geometry: %w", err)
	}
	return nil
}

func (driver *Driver) DrawBorder(rect paint.Rect, radii paint.Radii, width float32, color paint.Color, sides paint.BorderSides, dashed bool) error {
	if color.A == 0 {
		return nil
	}
	mesh, indices := paint.BorderMesh(rect, radii, width, sides, dashed, driver.scaleX, driver.scaleY)
	if len(indices) == 0 {
		return nil
	}
	vertices := make([]nativeVertex, len(mesh))
	for i, point := range mesh {
		vertices[i] = vertex(point.X, point.Y, color)
		vertices[i].A *= point.Coverage
	}
	return driver.renderer.renderGeometry(nil, vertices, indices)
}

func (driver *Driver) FillStrokeRoundedRect(rect paint.Rect, radii paint.Radii, width float32, fill, border paint.Color) error {
	if width <= 0 || border.A == 0 {
		return driver.FillRoundedRect(rect, radii, fill, 0)
	}
	if fill.A == 0 {
		return driver.StrokeRoundedRect(rect, radii, width, border)
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return nil
	}
	radii = normalizeRadii(rect, radii)
	width = min(width, min(rect.Width, rect.Height)/2)
	vertices, indices := roundedFillStrokeMesh(rect, radii, width, fill, border, driver.scaleX, driver.scaleY)
	if err := driver.renderer.renderGeometry(nil, vertices, indices); err != nil {
		return fmt.Errorf("fill/stroke rounded rect geometry: %w", err)
	}
	return nil
}

// DrawText uploads an immutable grayscale line mask on a cache miss, then
// color/alpha modulates the white RGBA texture. A hit creates no surface,
// texture, or string copy.
func (driver *Driver) DrawText(rect paint.Rect, bitmap *paint.TextBitmap, color paint.Color) error {
	if bitmap == nil || bitmap.Width == 0 || bitmap.Height == 0 || color.A == 0 {
		return nil
	}
	pixelCount := int64(bitmap.Width) * int64(bitmap.Height)
	if bitmap.Width < 0 || bitmap.Height < 0 || pixelCount > (math.MaxInt-textTextureEntryOverhead)/4 || int64(len(bitmap.Pixels)) != pixelCount {
		return fmt.Errorf("draw text: invalid bitmap")
	}
	key := textTextureKey{Identity: bitmap.Key, Width: bitmap.Width, Height: bitmap.Height}
	texture, ok := driver.textCache.Get(key)
	if !ok {
		var err error
		texture, err = driver.createTexture(bitmap.Width, bitmap.Height)
		if err != nil {
			return fmt.Errorf("draw text: create %dx%d texture: %w", bitmap.Width, bitmap.Height, err)
		}
		pixels := make([]byte, int(pixelCount)*4)
		for index, alpha := range bitmap.Pixels {
			offset := index * 4
			pixels[offset], pixels[offset+1], pixels[offset+2], pixels[offset+3] = 255, 255, 255, alpha
		}
		if err := texture.update(pixels, int32(bitmap.Width*4)); err != nil {
			driver.destroyTexture(texture)
			return fmt.Errorf("draw text: upload: %w", err)
		}
	}
	if err := texture.setColorMod(color.R, color.G, color.B); err != nil {
		if !ok {
			driver.destroyTexture(texture)
		}
		return fmt.Errorf("draw text: color modulation: %w", err)
	}
	if err := texture.setAlphaMod(color.A); err != nil {
		if !ok {
			driver.destroyTexture(texture)
		}
		return fmt.Errorf("draw text: alpha modulation: %w", err)
	}
	if err := driver.renderer.renderTexture(texture, nativeFRect{X: rect.X, Y: rect.Y, Width: rect.Width, Height: rect.Height}); err != nil {
		if !ok {
			driver.destroyTexture(texture)
		}
		return fmt.Errorf("draw text: render texture: %w", err)
	}
	if !ok {
		if driver.textCache.Put(key, texture, textTextureEntryOverhead+bitmap.Width*bitmap.Height*4) {
			if _, active := driver.activeText[key]; active {
				driver.textCache.Retain(key)
			}
		}
	}
	return nil
}

// DrawIcon shares the scale-keyed alpha-mask texture path with text while
// retaining a distinct backend-neutral display command.
func (driver *Driver) DrawIcon(rect paint.Rect, bitmap *paint.TextBitmap, color paint.Color) error {
	return driver.DrawText(rect, bitmap, color)
}

// DrawImage uploads tightly packed RGBA pixels once per renderer generation.
func (driver *Driver) DrawImage(rect paint.Rect, bitmap *paint.ImageBitmap, alpha uint8, imageClip paint.Rect, radii paint.Radii) error {
	if bitmap == nil || bitmap.Width == 0 || bitmap.Height == 0 || alpha == 0 || rect.Width <= 0 || rect.Height <= 0 {
		return nil
	}
	if bitmap.Width < 0 || bitmap.Height < 0 || bitmap.Height != 0 && int64(bitmap.Width) > math.MaxInt64/int64(bitmap.Height) {
		return fmt.Errorf("draw image: invalid bitmap")
	}
	pixelCount := int64(bitmap.Width) * int64(bitmap.Height)
	if pixelCount > (math.MaxInt-textTextureEntryOverhead)/4 || int64(len(bitmap.Pixels)) != pixelCount*4 {
		return fmt.Errorf("draw image: invalid bitmap")
	}
	key := textTextureKey{Identity: bitmap.Key, Width: bitmap.Width, Height: bitmap.Height}
	texture, ok := driver.imageCache.Get(key)
	if !ok {
		var err error
		texture, err = driver.createTexture(bitmap.Width, bitmap.Height)
		if err != nil {
			return fmt.Errorf("draw image: create %dx%d texture: %w", bitmap.Width, bitmap.Height, err)
		}
		if err := texture.update(bitmap.Pixels, int32(bitmap.Width*4)); err != nil {
			driver.destroyTexture(texture)
			return fmt.Errorf("draw image: upload: %w", err)
		}
	}
	if err := texture.setColorMod(255, 255, 255); err != nil {
		if !ok {
			driver.destroyTexture(texture)
		}
		return fmt.Errorf("draw image: color modulation: %w", err)
	}
	textureAlpha := alpha
	if imageClip != (paint.Rect{}) {
		textureAlpha = 255
	}
	if err := texture.setAlphaMod(textureAlpha); err != nil {
		if !ok {
			driver.destroyTexture(texture)
		}
		return fmt.Errorf("draw image: alpha modulation: %w", err)
	}
	var renderErr error
	if imageClip != (paint.Rect{}) {
		vertices, indices := roundedImageMesh(rect, imageClip, radii, alpha, driver.scaleX, driver.scaleY)
		renderErr = driver.renderer.renderGeometry(texture, vertices, indices)
	} else {
		renderErr = driver.renderer.renderTexture(texture, nativeFRect{X: rect.X, Y: rect.Y, Width: rect.Width, Height: rect.Height})
	}
	if renderErr != nil {
		if !ok {
			driver.destroyTexture(texture)
		}
		return fmt.Errorf("draw image: render texture: %w", renderErr)
	}
	if !ok {
		bytes := textTextureEntryOverhead + bitmap.Width*bitmap.Height*4
		if _, active := driver.activeImage[key]; active {
			driver.imageCache.PutRetained(key, texture, bytes)
		} else {
			driver.imageCache.Put(key, texture, bytes)
		}
	}
	return nil
}

func roundedImageMesh(destination, clip paint.Rect, radii paint.Radii, alpha uint8, scaleX, scaleY float32) ([]nativeVertex, []int32) {
	vertices, indices := roundedFillMesh(clip, radii, paint.Color{R: 255, G: 255, B: 255, A: alpha}, scaleX, scaleY)
	for index := range vertices {
		vertices[index].U = (vertices[index].X - destination.X) / destination.Width
		vertices[index].V = (vertices[index].Y - destination.Y) / destination.Height
	}
	return vertices, indices
}

func (driver *Driver) fillRounded(rect paint.Rect, radii paint.Radii, color paint.Color, joined paint.JoinedEdges) error {
	if rect.Width <= 0 || rect.Height <= 0 || color.A == 0 {
		return nil
	}
	radii = normalizeRadii(rect, radii)
	if radii == (paint.Radii{}) && joined == 0 {
		if err := driver.renderer.setDrawColor(color.R, color.G, color.B, color.A); err != nil {
			return err
		}
		return driver.renderer.fillRect(nativeFRect{X: rect.X, Y: rect.Y, Width: rect.Width, Height: rect.Height})
	}
	var vertices []nativeVertex
	var indices []int32
	if joined == 0 {
		vertices, indices = roundedFillMesh(rect, radii, color, driver.scaleX, driver.scaleY)
	} else {
		vertices, indices = joinedRoundedFillMesh(rect, radii, color, joined, driver.scaleX, driver.scaleY)
	}
	if err := driver.renderer.renderGeometry(nil, vertices, indices); err != nil {
		return fmt.Errorf("fill rounded rect geometry: %w", err)
	}
	return nil
}

// roundedFillMesh adds a one-physical-pixel alpha ramp centered on the
// mathematical boundary. SDL_RenderGeometry interpolates the per-vertex
// alpha, so this supplies coverage without a full-window or per-shape texture.
func roundedFillMesh(rect paint.Rect, radii paint.Radii, color paint.Color, scaleX, scaleY float32) ([]nativeVertex, []int32) {
	physical, physicalRadii, scaleX, scaleY := physicalRoundedRect(rect, radii, scaleX, scaleY)
	fringeX := min(edgeFringePixels, physical.Width/2)
	fringeY := min(edgeFringePixels, physical.Height/2)
	segments := roundedSegmentCount(physicalRadii, edgeFringePixels)
	perimeter := 4 * (segments + 1)
	vertices := make([]nativeVertex, 1, 1+2*perimeter)
	vertices[0] = vertex(rect.X+rect.Width/2, rect.Y+rect.Height/2, color)
	vertices = appendRoundedContour(vertices, physical, physicalRadii, fringeX, fringeY, segments, scaleX, scaleY, color)
	transparent := color
	transparent.A = 0
	vertices = appendRoundedContour(vertices, physical, physicalRadii, -fringeX, -fringeY, segments, scaleX, scaleY, transparent)
	indices := make([]int32, 0, perimeter*9)
	innerStart, outerStart := int32(1), int32(1+perimeter)
	for index := 0; index < perimeter; index++ {
		next := (index + 1) % perimeter
		indices = append(indices, 0, innerStart+int32(index), innerStart+int32(next))
		indices = appendQuad(indices, innerStart+int32(index), innerStart+int32(next), outerStart+int32(index), outerStart+int32(next))
	}
	return vertices, indices
}

func joinedRoundedFillMesh(rect paint.Rect, radii paint.Radii, color paint.Color, joined paint.JoinedEdges, scaleX, scaleY float32) ([]nativeVertex, []int32) {
	physical, physicalRadii, scaleX, scaleY := physicalRoundedRect(rect, radii, scaleX, scaleY)
	segments := roundedSegmentCount(physicalRadii, edgeFringePixels)
	perimeter := 4 * (segments + 1)
	vertices := make([]nativeVertex, 1, 1+2*perimeter)
	vertices[0] = vertex(rect.X+rect.Width/2, rect.Y+rect.Height/2, color)
	inner := joinedInsets(joined, edgeFringePixels)
	vertices = appendJoinedRoundedContour(vertices, physical, physicalRadii, inner, segments, scaleX, scaleY, color, joined, false)
	outer := joinedInsets(joined, -edgeFringePixels)
	vertices = appendJoinedRoundedContour(vertices, physical, physicalRadii, outer, segments, scaleX, scaleY, color, joined, true)
	indices := make([]int32, 0, perimeter*9)
	innerStart, outerStart := int32(1), int32(1+perimeter)
	for index := 0; index < perimeter; index++ {
		next := (index + 1) % perimeter
		indices = append(indices, 0, innerStart+int32(index), innerStart+int32(next))
		if joined&joinedEdgeAtContourSegment(index, segments) == 0 {
			indices = appendQuad(indices, innerStart+int32(index), innerStart+int32(next), outerStart+int32(index), outerStart+int32(next))
		}
	}
	return vertices, indices
}

// Contours contain each corner arc followed by the straight edge to the next
// corner. A joined straight edge needs the interior fan but no coincident
// fringe quad: some SDL backends blend that zero-width quad a second time.
func joinedEdgeAtContourSegment(index, segments int) paint.JoinedEdges {
	span := segments + 1
	switch index {
	case span - 1:
		return paint.JoinedTop
	case 2*span - 1:
		return paint.JoinedRight
	case 3*span - 1:
		return paint.JoinedBottom
	case 4*span - 1:
		return paint.JoinedLeft
	default:
		return 0
	}
}

// Insets are left, top, right, and bottom. Exposed edges receive the ordinary
// coverage band. Joined edges use source-order ownership: the later member's
// left/top edge stays on the mathematical boundary, while the earlier
// member's right/bottom edge retreats by a subpixel epsilon. This changes
// ownership only when both shapes land on the same raster sample, without
// opening a gap when the shared boundary is fractional.
func joinedInsets(joined paint.JoinedEdges, exposed float32) [4]float32 {
	result := [4]float32{exposed, exposed, exposed, exposed}
	for index, edge := range [...]paint.JoinedEdges{paint.JoinedLeft, paint.JoinedTop, paint.JoinedRight, paint.JoinedBottom} {
		if joined&edge != 0 {
			result[index] = 0
		}
	}
	if joined&paint.JoinedRight != 0 {
		result[2] = joinedOwnershipInsetPixels
	}
	if joined&paint.JoinedBottom != 0 {
		result[3] = joinedOwnershipInsetPixels
	}
	return result
}

func appendJoinedRoundedContour(vertices []nativeVertex, rect paint.Rect, radii ellipticalRadii, inset [4]float32, segments int, scaleX, scaleY float32, color paint.Color, joined paint.JoinedEdges, outer bool) []nativeVertex {
	left, top := rect.X+inset[0], rect.Y+inset[1]
	right, bottom := rect.X+rect.Width-inset[2], rect.Y+rect.Height-inset[3]
	type corner struct {
		x, y, radiusX, radiusY, start float32
		from, to                      paint.JoinedEdges
	}
	tlx, tly := max(0, radii.topLeftX-inset[0]), max(0, radii.topLeftY-inset[1])
	trx, try := max(0, radii.topRightX-inset[2]), max(0, radii.topRightY-inset[1])
	brx, bry := max(0, radii.bottomRightX-inset[2]), max(0, radii.bottomRightY-inset[3])
	blx, bly := max(0, radii.bottomLeftX-inset[0]), max(0, radii.bottomLeftY-inset[3])
	corners := [...]corner{
		{left + tlx, top + tly, tlx, tly, math.Pi, paint.JoinedLeft, paint.JoinedTop},
		{right - trx, top + try, trx, try, -math.Pi / 2, paint.JoinedTop, paint.JoinedRight},
		{right - brx, bottom - bry, brx, bry, 0, paint.JoinedRight, paint.JoinedBottom},
		{left + blx, bottom - bly, blx, bly, math.Pi / 2, paint.JoinedBottom, paint.JoinedLeft},
	}
	for _, corner := range corners {
		for segment := 0; segment <= segments; segment++ {
			t := float32(segment) / float32(segments)
			angle := float64(corner.start) + float64(t)*math.Pi/2
			vertexColor := color
			if outer {
				fromAlpha, toAlpha := float32(0), float32(0)
				if joined&corner.from != 0 {
					fromAlpha = float32(color.A)
				}
				if joined&corner.to != 0 {
					toAlpha = float32(color.A)
				}
				vertexColor.A = uint8(math.Round(float64(fromAlpha + (toAlpha-fromAlpha)*t)))
			}
			vertices = append(vertices, vertex(
				(corner.x+corner.radiusX*float32(math.Cos(angle)))/scaleX,
				(corner.y+corner.radiusY*float32(math.Sin(angle)))/scaleY,
				vertexColor,
			))
		}
	}
	return vertices
}

// roundedRingMesh emits the border and both coverage ramps as one mesh. The
// bands do not overlap, which prevents translucent borders from darkening at
// the inner/outer seams.
func roundedRingMesh(rect paint.Rect, radii paint.Radii, width float32, color paint.Color, scaleX, scaleY float32) ([]nativeVertex, []int32) {
	physical, physicalRadii, scaleX, scaleY := physicalRoundedRect(rect, radii, scaleX, scaleY)
	widthX, widthY := width*scaleX, width*scaleY
	fringeX := min(edgeFringePixels, widthX/2)
	fringeY := min(edgeFringePixels, widthY/2)
	segments := roundedSegmentCount(physicalRadii, edgeFringePixels)
	peak := color
	physicalWidth := min(widthX, widthY)
	if physicalWidth < 1 {
		peak.A = uint8(math.Round(float64(peak.A) * float64(max(0, physicalWidth))))
	}
	transparent := color
	transparent.A = 0
	colors := []paint.Color{transparent, peak, peak, transparent}
	insets := [][2]float32{
		{-edgeFringePixels, -edgeFringePixels},
		{fringeX, fringeY},
		{widthX - fringeX, widthY - fringeY},
		{widthX + edgeFringePixels, widthY + edgeFringePixels},
	}
	perimeter := 4 * (segments + 1)
	vertices := make([]nativeVertex, 0, len(insets)*perimeter)
	for index, inset := range insets {
		vertices = appendRoundedContour(vertices, physical, physicalRadii, inset[0], inset[1], segments, scaleX, scaleY, colors[index])
	}
	indices := make([]int32, 0, (len(insets)-1)*perimeter*6)
	for band := 0; band < len(insets)-1; band++ {
		first, second := int32(band*perimeter), int32((band+1)*perimeter)
		for index := 0; index < perimeter; index++ {
			next := (index + 1) % perimeter
			indices = appendQuad(indices, first+int32(index), first+int32(next), second+int32(index), second+int32(next))
		}
	}
	return vertices, indices
}

func roundedFillStrokeMesh(rect paint.Rect, radii paint.Radii, width float32, fill, border paint.Color, scaleX, scaleY float32) ([]nativeVertex, []int32) {
	physical, physicalRadii, scaleX, scaleY := physicalRoundedRect(rect, radii, scaleX, scaleY)
	widthX, widthY := width*scaleX, width*scaleY
	fringeX := min(edgeFringePixels, widthX/2)
	fringeY := min(edgeFringePixels, widthY/2)
	segments := roundedSegmentCount(physicalRadii, edgeFringePixels)
	peakBorder := border
	if physicalWidth := min(widthX, widthY); physicalWidth < 1 {
		peakBorder.A = uint8(math.Round(float64(peakBorder.A) * float64(max(0, physicalWidth))))
	}
	transparentBorder := border
	transparentBorder.A = 0
	insets := [][2]float32{
		{-edgeFringePixels, -edgeFringePixels},
		{fringeX, fringeY},
		{widthX - fringeX, widthY - fringeY},
		{widthX + edgeFringePixels, widthY + edgeFringePixels},
	}
	colors := []paint.Color{transparentBorder, peakBorder, peakBorder, fill}
	perimeter := 4 * (segments + 1)
	vertices := make([]nativeVertex, 1, 1+len(insets)*perimeter)
	vertices[0] = vertex(rect.X+rect.Width/2, rect.Y+rect.Height/2, fill)
	for index, inset := range insets {
		vertices = appendRoundedContour(vertices, physical, physicalRadii, inset[0], inset[1], segments, scaleX, scaleY, colors[index])
	}
	indices := make([]int32, 0, perimeter*(len(insets)-1)*6+perimeter*3)
	for band := 0; band < len(insets)-1; band++ {
		first, second := int32(1+band*perimeter), int32(1+(band+1)*perimeter)
		for index := 0; index < perimeter; index++ {
			next := (index + 1) % perimeter
			indices = appendQuad(indices, first+int32(index), first+int32(next), second+int32(index), second+int32(next))
		}
	}
	fillStart := int32(1 + (len(insets)-1)*perimeter)
	for index := 0; index < perimeter; index++ {
		next := (index + 1) % perimeter
		indices = append(indices, 0, fillStart+int32(index), fillStart+int32(next))
	}
	return vertices, indices
}

type ellipticalRadii struct {
	topLeftX, topLeftY         float32
	topRightX, topRightY       float32
	bottomRightX, bottomRightY float32
	bottomLeftX, bottomLeftY   float32
}

func physicalRoundedRect(rect paint.Rect, radii paint.Radii, scaleX, scaleY float32) (paint.Rect, ellipticalRadii, float32, float32) {
	if scaleX <= 0 || !finiteFloat32(scaleX) {
		scaleX = 1
	}
	if scaleY <= 0 || !finiteFloat32(scaleY) {
		scaleY = 1
	}
	radii = normalizeRadii(rect, radii)
	return paint.Rect{X: rect.X * scaleX, Y: rect.Y * scaleY, Width: rect.Width * scaleX, Height: rect.Height * scaleY}, ellipticalRadii{
		topLeftX: radii.TopLeft * scaleX, topLeftY: radii.TopLeft * scaleY,
		topRightX: radii.TopRight * scaleX, topRightY: radii.TopRight * scaleY,
		bottomRightX: radii.BottomRight * scaleX, bottomRightY: radii.BottomRight * scaleY,
		bottomLeftX: radii.BottomLeft * scaleX, bottomLeftY: radii.BottomLeft * scaleY,
	}, scaleX, scaleY
}

func finiteFloat32(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}

func roundedSegmentCount(radii ellipticalRadii, expansion float32) int {
	radius := max(radii.topLeftX, radii.topLeftY, radii.topRightX, radii.topRightY,
		radii.bottomRightX, radii.bottomRightY, radii.bottomLeftX, radii.bottomLeftY) + expansion
	if radius <= curveTolerancePixels {
		return 2
	}
	step := math.Acos(max(-1, min(1, 1-float64(curveTolerancePixels/radius))))
	if step <= 0 || math.IsNaN(step) {
		return maxRoundedSegments
	}
	return max(2, min(maxRoundedSegments, int(math.Ceil((math.Pi/2)/step))))
}

func appendRoundedContour(vertices []nativeVertex, rect paint.Rect, radii ellipticalRadii, insetX, insetY float32, segments int, scaleX, scaleY float32, color paint.Color) []nativeVertex {
	adjusted := paint.Rect{X: rect.X + insetX, Y: rect.Y + insetY, Width: max(0, rect.Width-2*insetX), Height: max(0, rect.Height-2*insetY)}
	type corner struct{ x, y, radiusX, radiusY, start float32 }
	tlx, tly := max(0, radii.topLeftX-insetX), max(0, radii.topLeftY-insetY)
	trx, try := max(0, radii.topRightX-insetX), max(0, radii.topRightY-insetY)
	brx, bry := max(0, radii.bottomRightX-insetX), max(0, radii.bottomRightY-insetY)
	blx, bly := max(0, radii.bottomLeftX-insetX), max(0, radii.bottomLeftY-insetY)
	corners := []corner{
		{adjusted.X + tlx, adjusted.Y + tly, tlx, tly, math.Pi},
		{adjusted.X + adjusted.Width - trx, adjusted.Y + try, trx, try, -math.Pi / 2},
		{adjusted.X + adjusted.Width - brx, adjusted.Y + adjusted.Height - bry, brx, bry, 0},
		{adjusted.X + blx, adjusted.Y + adjusted.Height - bly, blx, bly, math.Pi / 2},
	}
	for _, corner := range corners {
		for segment := 0; segment <= segments; segment++ {
			angle := float64(corner.start) + float64(segment)*math.Pi/2/float64(segments)
			vertices = append(vertices, vertex(
				(corner.x+corner.radiusX*float32(math.Cos(angle)))/scaleX,
				(corner.y+corner.radiusY*float32(math.Sin(angle)))/scaleY,
				color,
			))
		}
	}
	return vertices
}

func appendQuad(indices []int32, first, firstNext, second, secondNext int32) []int32 {
	return append(indices, first, firstNext, secondNext, first, secondNext, second)
}

func normalizeRadii(rect paint.Rect, radii paint.Radii) paint.Radii {
	radii.TopLeft = max(0, radii.TopLeft)
	radii.TopRight = max(0, radii.TopRight)
	radii.BottomRight = max(0, radii.BottomRight)
	radii.BottomLeft = max(0, radii.BottomLeft)
	limit := min(rect.Width, rect.Height) / 2
	radii.TopLeft = min(radii.TopLeft, limit)
	radii.TopRight = min(radii.TopRight, limit)
	radii.BottomRight = min(radii.BottomRight, limit)
	radii.BottomLeft = min(radii.BottomLeft, limit)
	return radii
}

func addRadius(value paint.Radii, amount float32) paint.Radii {
	return paint.Radii{TopLeft: value.TopLeft + amount, TopRight: value.TopRight + amount, BottomRight: value.BottomRight + amount, BottomLeft: value.BottomLeft + amount}
}

func subtractRadius(value paint.Radii, amount float32) paint.Radii {
	return paint.Radii{TopLeft: max(0, value.TopLeft-amount), TopRight: max(0, value.TopRight-amount), BottomRight: max(0, value.BottomRight-amount), BottomLeft: max(0, value.BottomLeft-amount)}
}

func vertex(x, y float32, color paint.Color) nativeVertex {
	const divisor = float32(255)
	return nativeVertex{X: x, Y: y, R: float32(color.R) / divisor, G: float32(color.G) / divisor, B: float32(color.B) / divisor, A: float32(color.A) / divisor}
}

var _ paint.Painter = (*Driver)(nil)
