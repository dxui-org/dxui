package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/dxui-org/dxui/internal/icondata"
)

type pathParser struct {
	source string
	index  int
}

func parsePathData(source string) ([]icondata.PathCommand, error) {
	parser := pathParser{source: source}
	var result []icondata.PathCommand
	var current, start, cubicControl, quadControl icondata.Point
	var command, previous byte
	for {
		parser.separators()
		if parser.index == len(source) {
			break
		}
		if letter := source[parser.index]; unicode.IsLetter(rune(letter)) {
			command = letter
			parser.index++
		} else if command == 0 || command == 'Z' || command == 'z' {
			return nil, parser.errorf("expected command")
		}
		relative := command >= 'a' && command <= 'z'
		upper := command
		if relative {
			upper -= 'a' - 'A'
		}
		rel := func(point icondata.Point) icondata.Point {
			if relative {
				point.X += current.X
				point.Y += current.Y
			}
			return point
		}
		switch upper {
		case 'M':
			point, err := parser.point()
			if err != nil {
				return nil, err
			}
			point = rel(point)
			result = append(result, icondata.PathCommand{Verb: icondata.PathMove, Points: [3]icondata.Point{point}})
			current, start = point, point
			if relative {
				command = 'l'
			} else {
				command = 'L'
			}
		case 'L':
			point, err := parser.point()
			if err != nil {
				return nil, err
			}
			point = rel(point)
			result = append(result, icondata.PathCommand{Verb: icondata.PathLine, Points: [3]icondata.Point{point}})
			current = point
		case 'H':
			value, err := parser.number()
			if err != nil {
				return nil, err
			}
			if relative {
				value += current.X
			}
			current.X = value
			result = append(result, icondata.PathCommand{Verb: icondata.PathLine, Points: [3]icondata.Point{current}})
		case 'V':
			value, err := parser.number()
			if err != nil {
				return nil, err
			}
			if relative {
				value += current.Y
			}
			current.Y = value
			result = append(result, icondata.PathCommand{Verb: icondata.PathLine, Points: [3]icondata.Point{current}})
		case 'Q':
			control, err := parser.point()
			if err != nil {
				return nil, err
			}
			end, err := parser.point()
			if err != nil {
				return nil, err
			}
			control, end = rel(control), rel(end)
			result = append(result, icondata.PathCommand{Verb: icondata.PathQuad, Points: [3]icondata.Point{control, end}})
			quadControl, current = control, end
		case 'T':
			end, err := parser.point()
			if err != nil {
				return nil, err
			}
			end = rel(end)
			control := current
			if previous == 'Q' || previous == 'q' || previous == 'T' || previous == 't' {
				control = icondata.Point{X: 2*current.X - quadControl.X, Y: 2*current.Y - quadControl.Y}
			}
			result = append(result, icondata.PathCommand{Verb: icondata.PathQuad, Points: [3]icondata.Point{control, end}})
			quadControl, current = control, end
		case 'C':
			one, err := parser.point()
			if err != nil {
				return nil, err
			}
			two, err := parser.point()
			if err != nil {
				return nil, err
			}
			end, err := parser.point()
			if err != nil {
				return nil, err
			}
			one, two, end = rel(one), rel(two), rel(end)
			result = append(result, icondata.PathCommand{Verb: icondata.PathCubic, Points: [3]icondata.Point{one, two, end}})
			cubicControl, current = two, end
		case 'S':
			two, err := parser.point()
			if err != nil {
				return nil, err
			}
			end, err := parser.point()
			if err != nil {
				return nil, err
			}
			two, end = rel(two), rel(end)
			one := current
			if previous == 'C' || previous == 'c' || previous == 'S' || previous == 's' {
				one = icondata.Point{X: 2*current.X - cubicControl.X, Y: 2*current.Y - cubicControl.Y}
			}
			result = append(result, icondata.PathCommand{Verb: icondata.PathCubic, Points: [3]icondata.Point{one, two, end}})
			cubicControl, current = two, end
		case 'A':
			rx, err := parser.number()
			if err != nil {
				return nil, err
			}
			ry, err := parser.number()
			if err != nil {
				return nil, err
			}
			rotation, err := parser.number()
			if err != nil {
				return nil, err
			}
			large, err := parser.flag()
			if err != nil {
				return nil, err
			}
			sweep, err := parser.flag()
			if err != nil {
				return nil, err
			}
			end, err := parser.point()
			if err != nil {
				return nil, err
			}
			end = rel(end)
			curves, err := arcToCubics(current, end, rx, ry, rotation, large, sweep)
			if err != nil {
				return nil, parser.errorf("arc: %v", err)
			}
			result = append(result, curves...)
			if len(curves) != 0 {
				cubicControl = curves[len(curves)-1].Points[1]
			}
			current = end
		case 'Z':
			result = append(result, icondata.PathCommand{Verb: icondata.PathClose})
			current = start
			command = 0
		default:
			return nil, parser.errorf("unsupported command %q", command)
		}
		previous = command
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	return result, nil
}

func (parser *pathParser) separators() {
	for parser.index < len(parser.source) {
		switch parser.source[parser.index] {
		case ' ', '\t', '\n', '\r', ',':
			parser.index++
		default:
			return
		}
	}
}

func (parser *pathParser) number() (float32, error) {
	parser.separators()
	start := parser.index
	if parser.index < len(parser.source) && (parser.source[parser.index] == '+' || parser.source[parser.index] == '-') {
		parser.index++
	}
	digits := false
	for parser.index < len(parser.source) && parser.source[parser.index] >= '0' && parser.source[parser.index] <= '9' {
		parser.index++
		digits = true
	}
	if parser.index < len(parser.source) && parser.source[parser.index] == '.' {
		parser.index++
		for parser.index < len(parser.source) && parser.source[parser.index] >= '0' && parser.source[parser.index] <= '9' {
			parser.index++
			digits = true
		}
	}
	if !digits {
		return 0, parser.errorf("expected number")
	}
	if parser.index < len(parser.source) && (parser.source[parser.index] == 'e' || parser.source[parser.index] == 'E') {
		parser.index++
		if parser.index < len(parser.source) && (parser.source[parser.index] == '+' || parser.source[parser.index] == '-') {
			parser.index++
		}
		exponent := parser.index
		for parser.index < len(parser.source) && parser.source[parser.index] >= '0' && parser.source[parser.index] <= '9' {
			parser.index++
		}
		if exponent == parser.index {
			return 0, parser.errorf("invalid exponent")
		}
	}
	value, err := strconv.ParseFloat(parser.source[start:parser.index], 32)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, parser.errorf("invalid number %q", parser.source[start:parser.index])
	}
	return float32(value), nil
}

func (parser *pathParser) point() (icondata.Point, error) {
	x, err := parser.number()
	if err != nil {
		return icondata.Point{}, err
	}
	y, err := parser.number()
	return icondata.Point{X: x, Y: y}, err
}

func (parser *pathParser) flag() (bool, error) {
	parser.separators()
	if parser.index >= len(parser.source) || (parser.source[parser.index] != '0' && parser.source[parser.index] != '1') {
		return false, parser.errorf("arc flag must be 0 or 1")
	}
	value := parser.source[parser.index] == '1'
	parser.index++
	return value, nil
}

func (parser *pathParser) errorf(format string, arguments ...any) error {
	return fmt.Errorf("path byte %d: %s", parser.index, fmt.Sprintf(format, arguments...))
}

func arcToCubics(start, end icondata.Point, rx, ry, rotation float32, large, sweep bool) ([]icondata.PathCommand, error) {
	if rx == 0 || ry == 0 || start == end {
		if start == end {
			return nil, nil
		}
		return []icondata.PathCommand{{Verb: icondata.PathLine, Points: [3]icondata.Point{end}}}, nil
	}
	rx, ry = float32(math.Abs(float64(rx))), float32(math.Abs(float64(ry)))
	phi := float64(rotation) * math.Pi / 180
	cosPhi, sinPhi := math.Cos(phi), math.Sin(phi)
	dx, dy := float64(start.X-end.X)/2, float64(start.Y-end.Y)/2
	x1 := cosPhi*dx + sinPhi*dy
	y1 := -sinPhi*dx + cosPhi*dy
	rx64, ry64 := float64(rx), float64(ry)
	lambda := x1*x1/(rx64*rx64) + y1*y1/(ry64*ry64)
	if lambda > 1 {
		scale := math.Sqrt(lambda)
		rx64 *= scale
		ry64 *= scale
	}
	numerator := rx64*rx64*ry64*ry64 - rx64*rx64*y1*y1 - ry64*ry64*x1*x1
	denominator := rx64*rx64*y1*y1 + ry64*ry64*x1*x1
	factor := 0.0
	if denominator != 0 {
		factor = math.Sqrt(math.Max(0, numerator/denominator))
	}
	if large == sweep {
		factor = -factor
	}
	cx1 := factor * rx64 * y1 / ry64
	cy1 := factor * -ry64 * x1 / rx64
	cx := cosPhi*cx1 - sinPhi*cy1 + float64(start.X+end.X)/2
	cy := sinPhi*cx1 + cosPhi*cy1 + float64(start.Y+end.Y)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
	}
	ux, uy := (x1-cx1)/rx64, (y1-cy1)/ry64
	vx, vy := (-x1-cx1)/rx64, (-y1-cy1)/ry64
	theta := math.Atan2(uy, ux)
	delta := angle(ux, uy, vx, vy)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}
	segments := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	if segments < 1 || segments > 4 {
		return nil, fmt.Errorf("invalid segment count %d", segments)
	}
	step := delta / float64(segments)
	mapPoint := func(x, y float64) icondata.Point {
		return icondata.Point{X: float32(cx + rx64*cosPhi*x - ry64*sinPhi*y), Y: float32(cy + rx64*sinPhi*x + ry64*cosPhi*y)}
	}
	result := make([]icondata.PathCommand, 0, segments)
	for index := 0; index < segments; index++ {
		one, two := theta+float64(index)*step, theta+float64(index+1)*step
		factor := 4.0 / 3.0 * math.Tan((two-one)/4)
		c1 := mapPoint(math.Cos(one)-factor*math.Sin(one), math.Sin(one)+factor*math.Cos(one))
		c2 := mapPoint(math.Cos(two)+factor*math.Sin(two), math.Sin(two)-factor*math.Cos(two))
		finish := mapPoint(math.Cos(two), math.Sin(two))
		if index == segments-1 {
			finish = end
		}
		result = append(result, icondata.PathCommand{Verb: icondata.PathCubic, Points: [3]icondata.Point{c1, c2, finish}})
	}
	return result, nil
}

func parsePointList(source string) ([]icondata.Point, error) {
	parser := pathParser{source: strings.TrimSpace(source)}
	var result []icondata.Point
	for {
		parser.separators()
		if parser.index == len(parser.source) {
			break
		}
		point, err := parser.point()
		if err != nil {
			return nil, err
		}
		result = append(result, point)
	}
	if len(result) < 2 {
		return nil, fmt.Errorf("point list has %d points", len(result))
	}
	return result, nil
}
