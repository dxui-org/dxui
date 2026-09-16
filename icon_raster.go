package dxui

import (
	"fmt"
	stdimage "image"
	"image/color"
	"math"

	"github.com/dxui-org/dxui/internal/icondata"
	"github.com/dxui-org/dxui/internal/layout"
	"github.com/dxui-org/dxui/internal/paint"
	"golang.org/x/image/vector"
)

const (
	defaultIconStrokeWidth = float32(2)
	maxIconCommands        = 8192
	maxIconFlattenedPoints = 16384
	maxIconRasterVertices  = 1 << 20
	maxIconCurveSegments   = 64
	iconCurveTolerance     = float32(.2)
	maxIconCircleSegments  = 128
	iconMaskOverhead       = 128
)

func accountIconMask(bitmap *paint.TextBitmap, remaining *int, reuse *textMaskReuse) error {
	if bitmap == nil || remaining == nil {
		return fmt.Errorf("missing icon mask budget")
	}
	bytes := iconMaskOverhead + len(bitmap.Pixels)
	if reuse != nil {
		if _, exists := reuse.accounted[bitmap.Key]; exists {
			return nil
		}
	}
	if bytes > *remaining {
		return fmt.Errorf("icon mask needs %d bytes, %d remain", bytes, max(0, *remaining))
	}
	*remaining -= bytes
	if reuse != nil {
		if reuse.accounted == nil {
			reuse.accounted = make(map[string]struct{})
		}
		reuse.accounted[bitmap.Key] = struct{}{}
	}
	return nil
}

func rasterIcon(props IconProps, content layout.Rect, theme resolvedTheme, scaleX, scaleY float32) (*paint.TextBitmap, paint.Rect, error) {
	return rasterIconReuse(props, content, theme, scaleX, scaleY, nil)
}

func rasterIconReuse(props IconProps, content layout.Rect, theme resolvedTheme, scaleX, scaleY float32, reuse *textMaskReuse) (*paint.TextBitmap, paint.Rect, error) {
	sizeValue := iconSizeMetric(props.Size)
	size, err := theme.metric(sizeValue)
	if err != nil {
		return nil, paint.Rect{}, fmt.Errorf("size: %w", err)
	}
	if size <= 0 {
		return nil, paint.Rect{}, fmt.Errorf("size must resolve positive")
	}
	logicalWidth, logicalHeight := min(size, content.Width), min(size, content.Height)
	if !finite(logicalWidth) || !finite(logicalHeight) || !finite(scaleX) || !finite(scaleY) || logicalWidth < 0 || logicalHeight < 0 || scaleX <= 0 || scaleY <= 0 {
		return nil, paint.Rect{}, fmt.Errorf("invalid logical size or renderer scale")
	}
	pixelWidth64 := math.Max(1, math.Ceil(float64(logicalWidth)*float64(scaleX)))
	pixelHeight64 := math.Max(1, math.Ceil(float64(logicalHeight)*float64(scaleY)))
	if pixelWidth64 > math.MaxInt || pixelHeight64 > math.MaxInt || pixelWidth64*pixelHeight64 > 16*1024*1024 {
		return nil, paint.Rect{}, fmt.Errorf("physical mask exceeds 16777216 pixels")
	}
	pixelWidth, pixelHeight := int(pixelWidth64), int(pixelHeight64)
	strokeWidth := float32(props.StrokeWidth)
	if strokeWidth == 0 {
		strokeWidth = defaultIconStrokeWidth
	}
	if !props.Data.IsPacked() {
		strokeWidth = 0
	}
	key := fmt.Sprintf("icon:%016x:%dx%d:%.4g:%.4g:%.4g", props.Data.Identity(), pixelWidth, pixelHeight, scaleX, scaleY, strokeWidth)
	rect := paint.Rect{X: content.X + (content.Width-logicalWidth)/2, Y: content.Y + (content.Height-logicalHeight)/2, Width: logicalWidth, Height: logicalHeight}
	if reuse != nil {
		if pixels, ok := reuse.get(key, pixelWidth, pixelHeight); ok {
			return &paint.TextBitmap{Key: key, Width: pixelWidth, Height: pixelHeight, Pixels: pixels}, rect, nil
		}
	}
	paths, err := props.Data.Paths(maxIconCommands)
	if err != nil {
		return nil, paint.Rect{}, err
	}
	raster := vector.NewRasterizer(pixelWidth, pixelHeight)
	transform := iconTransform{box: props.Data.ViewBox, width: float32(pixelWidth), height: float32(pixelHeight)}
	pointCount, rasterVertices := 0, 0
	for index, path := range paths {
		switch path.Mode {
		case icondata.PaintFill:
			if err := appendFilledIconPath(raster, path.Commands, transform); err != nil {
				return nil, paint.Rect{}, fmt.Errorf("fill path %d: %w", index, err)
			}
		case icondata.PaintStroke:
			if err := appendStrokedIconPath(raster, path.Commands, transform, strokeWidth, &pointCount, &rasterVertices); err != nil {
				return nil, paint.Rect{}, fmt.Errorf("stroke path %d: %w", index, err)
			}
		default:
			return nil, paint.Rect{}, fmt.Errorf("path %d has unknown paint mode %d", index, path.Mode)
		}
	}
	alpha := stdimage.NewAlpha(stdimage.Rect(0, 0, pixelWidth, pixelHeight))
	raster.Draw(alpha, alpha.Bounds(), stdimage.NewUniform(color.Alpha{A: 255}), stdimage.Point{})
	pixels := make([]byte, pixelWidth*pixelHeight)
	for y := 0; y < pixelHeight; y++ {
		copy(pixels[y*pixelWidth:(y+1)*pixelWidth], alpha.Pix[y*alpha.Stride:y*alpha.Stride+pixelWidth])
	}
	bitmap := &paint.TextBitmap{Key: key, Width: pixelWidth, Height: pixelHeight, Pixels: pixels}
	reuse.remember(bitmap)
	return bitmap, rect, nil
}

type iconTransform struct {
	box           Rect
	width, height float32
}

func (transform iconTransform) point(point Point) Point {
	return Point{
		X: (point.X - transform.box.X) / transform.box.Width * transform.width,
		Y: (point.Y - transform.box.Y) / transform.box.Height * transform.height,
	}
}

func appendFilledIconPath(raster *vector.Rasterizer, commands []PathCommand, transform iconTransform) error {
	for _, command := range commands {
		switch command.Verb {
		case PathMove:
			point := transform.point(command.Points[0])
			raster.MoveTo(point.X, point.Y)
		case PathLine:
			point := transform.point(command.Points[0])
			raster.LineTo(point.X, point.Y)
		case PathQuad:
			one, two := transform.point(command.Points[0]), transform.point(command.Points[1])
			raster.QuadTo(one.X, one.Y, two.X, two.Y)
		case PathCubic:
			one, two, three := transform.point(command.Points[0]), transform.point(command.Points[1]), transform.point(command.Points[2])
			raster.CubeTo(one.X, one.Y, two.X, two.Y, three.X, three.Y)
		case PathClose:
			raster.ClosePath()
		default:
			return fmt.Errorf("unknown path verb %d", command.Verb)
		}
	}
	return nil
}

type iconSubpath struct {
	points []Point
	closed bool
	drawn  bool
}

func appendStrokedIconPath(raster *vector.Rasterizer, commands []PathCommand, transform iconTransform, width float32, total, rasterVertices *int) error {
	if !finite(width) || width <= 0 {
		return fmt.Errorf("stroke width must be positive")
	}
	subpaths, err := flattenIconPath(commands, transform)
	if err != nil {
		return err
	}
	radius := width / 2
	for _, subpath := range subpaths {
		*total += len(subpath.points)
		if *total > maxIconFlattenedPoints {
			return fmt.Errorf("flattened path exceeds %d points", maxIconFlattenedPoints)
		}
		if !subpath.drawn || len(subpath.points) == 0 {
			continue
		}
		circleSegments, err := strokeCircleSegments(radius, transform)
		if err != nil {
			return err
		}
		limit := len(subpath.points) - 1
		if subpath.closed {
			limit = len(subpath.points)
		}
		*rasterVertices += limit*4 + len(subpath.points)*circleSegments
		if *rasterVertices > maxIconRasterVertices {
			return fmt.Errorf("stroke geometry exceeds %d vertices", maxIconRasterVertices)
		}
		for index := 0; index < limit; index++ {
			one := subpath.points[index]
			two := subpath.points[(index+1)%len(subpath.points)]
			appendStrokeSegment(raster, one, two, radius, transform)
		}
		// A disk at every vertex implements round joins and round caps. All
		// outlines enter one rasterizer, so overlaps saturate as one mask.
		for _, point := range subpath.points {
			appendStrokeDisk(raster, point, radius, transform, circleSegments)
		}
	}
	return nil
}

func flattenIconPath(commands []PathCommand, transform iconTransform) ([]iconSubpath, error) {
	result := make([]iconSubpath, 0, 4)
	var current iconSubpath
	var point, start Point
	flush := func() {
		if len(current.points) != 0 {
			result = append(result, current)
		}
		current = iconSubpath{}
	}
	for _, command := range commands {
		switch command.Verb {
		case PathMove:
			flush()
			point, start = command.Points[0], command.Points[0]
			current.points = append(current.points, point)
		case PathLine:
			if len(current.points) == 0 {
				return nil, fmt.Errorf("line before move")
			}
			point = command.Points[0]
			current.points = append(current.points, point)
			current.drawn = true
		case PathQuad:
			if len(current.points) == 0 {
				return nil, fmt.Errorf("quadratic curve before move")
			}
			if err := flattenQuad(&current.points, point, command.Points[0], command.Points[1], transform, 0); err != nil {
				return nil, err
			}
			point = command.Points[1]
			current.drawn = true
		case PathCubic:
			if len(current.points) == 0 {
				return nil, fmt.Errorf("cubic curve before move")
			}
			if err := flattenCubic(&current.points, point, command.Points[0], command.Points[1], command.Points[2], transform, 0); err != nil {
				return nil, err
			}
			point = command.Points[2]
			current.drawn = true
		case PathClose:
			if len(current.points) == 0 {
				return nil, fmt.Errorf("close before move")
			}
			current.closed, current.drawn, point = true, true, start
		default:
			return nil, fmt.Errorf("unknown path verb %d", command.Verb)
		}
	}
	flush()
	return result, nil
}

func flattenQuad(points *[]Point, p0, p1, p2 Point, transform iconTransform, depth int) error {
	one, control, two := transform.point(p0), transform.point(p1), transform.point(p2)
	if pointLineDistance(control, one, two) <= iconCurveTolerance {
		*points = append(*points, p2)
		return nil
	}
	if depth == 6 {
		return fmt.Errorf("quadratic curve exceeds %.1f-pixel error at %d segments", iconCurveTolerance, maxIconCurveSegments)
	}
	p01, p12 := midpoint(p0, p1), midpoint(p1, p2)
	middle := midpoint(p01, p12)
	if err := flattenQuad(points, p0, p01, middle, transform, depth+1); err != nil {
		return err
	}
	return flattenQuad(points, middle, p12, p2, transform, depth+1)
}

func flattenCubic(points *[]Point, p0, p1, p2, p3 Point, transform iconTransform, depth int) error {
	one, controlOne := transform.point(p0), transform.point(p1)
	controlTwo, two := transform.point(p2), transform.point(p3)
	if max(pointLineDistance(controlOne, one, two), pointLineDistance(controlTwo, one, two)) <= iconCurveTolerance {
		*points = append(*points, p3)
		return nil
	}
	if depth == 6 {
		return fmt.Errorf("cubic curve exceeds %.1f-pixel error at %d segments", iconCurveTolerance, maxIconCurveSegments)
	}
	p01, p12, p23 := midpoint(p0, p1), midpoint(p1, p2), midpoint(p2, p3)
	p012, p123 := midpoint(p01, p12), midpoint(p12, p23)
	middle := midpoint(p012, p123)
	if err := flattenCubic(points, p0, p01, p012, middle, transform, depth+1); err != nil {
		return err
	}
	return flattenCubic(points, middle, p123, p23, p3, transform, depth+1)
}

func midpoint(one, two Point) Point { return Point{X: (one.X + two.X) / 2, Y: (one.Y + two.Y) / 2} }

func pointLineDistance(point, one, two Point) float32 {
	dx, dy := two.X-one.X, two.Y-one.Y
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length == 0 {
		return float32(math.Hypot(float64(point.X-one.X), float64(point.Y-one.Y)))
	}
	return abs32(dy*point.X-dx*point.Y+two.X*one.Y-two.Y*one.X) / length
}

func appendStrokeSegment(raster *vector.Rasterizer, one, two Point, radius float32, transform iconTransform) {
	dx, dy := two.X-one.X, two.Y-one.Y
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length == 0 {
		return
	}
	nx, ny := -dy/length*radius, dx/length*radius
	appendPolygon(raster, transform, []Point{
		{X: one.X - nx, Y: one.Y - ny}, {X: two.X - nx, Y: two.Y - ny},
		{X: two.X + nx, Y: two.Y + ny}, {X: one.X + nx, Y: one.Y + ny},
	})
}

func appendStrokeDisk(raster *vector.Rasterizer, center Point, radius float32, transform iconTransform, segments int) {
	points := make([]Point, segments)
	for index := range points {
		angle := 2 * math.Pi * float64(index) / float64(segments)
		points[index] = Point{X: center.X + radius*float32(math.Cos(angle)), Y: center.Y + radius*float32(math.Sin(angle))}
	}
	appendPolygon(raster, transform, points)
}

func strokeCircleSegments(radius float32, transform iconTransform) (int, error) {
	physicalRadius := max(radius*transform.width/transform.box.Width, radius*transform.height/transform.box.Height)
	if physicalRadius <= iconCurveTolerance {
		return 8, nil
	}
	angle := math.Acos(max(-1, min(1, 1-float64(iconCurveTolerance/physicalRadius))))
	segments := int(math.Ceil(math.Pi / angle))
	segments = max(8, segments)
	if segments > maxIconCircleSegments {
		return 0, fmt.Errorf("round cap exceeds %.1f-pixel error at %d segments", iconCurveTolerance, maxIconCircleSegments)
	}
	return segments, nil
}

func appendPolygon(raster *vector.Rasterizer, transform iconTransform, points []Point) {
	if len(points) == 0 {
		return
	}
	first := transform.point(points[0])
	raster.MoveTo(first.X, first.Y)
	for _, source := range points[1:] {
		point := transform.point(source)
		raster.LineTo(point.X, point.Y)
	}
	raster.ClosePath()
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}
