package paint

import "math"

// BorderSides uses clockwise bits; zero selects all four sides.
type BorderSides uint8

const (
	BorderTop BorderSides = 1 << iota
	BorderRight
	BorderBottom
	BorderLeft
	BorderAll = BorderTop | BorderRight | BorderBottom | BorderLeft
)

// BorderVertex is a logical position with unit alpha coverage. Color is supplied
// at replay, so adjacent strips never blend separate copies of the border.
type BorderVertex struct{ X, Y, Coverage float32 }

type borderPoint struct{ x, y float32 }
type borderStation struct {
	band [4]borderPoint
	side BorderSides
}

// BorderMesh tessellates a selected/dashed inside ring. Work and storage are
// bounded independently of dimensions: 260 contour intervals plus at most 512
// dash cuts. Nothing is cached or allocated for an empty border.
func BorderMesh(rect Rect, radii Radii, width float32, sides BorderSides, dashed bool, scaleX, scaleY float32) ([]BorderVertex, []int32) {
	if !validRect(rect) || !finite(rect.X+rect.Width) || !finite(rect.Y+rect.Height) || !finite(width) || width <= 0 || rect.Width <= 0 || rect.Height <= 0 || sides & ^BorderAll != 0 {
		return nil, nil
	}
	if sides == 0 {
		sides = BorderAll
	}
	if !finite(scaleX) || scaleX <= 0 {
		scaleX = 1
	}
	if !finite(scaleY) || scaleY <= 0 {
		scaleY = 1
	}
	limit := min(rect.Width, rect.Height) / 2
	width = min(width, limit)
	r := [4]float32{radii.TopLeft, radii.TopRight, radii.BottomRight, radii.BottomLeft}
	for i := range r {
		if !finite(r[i]) {
			return nil, nil
		}
		r[i] = max(0, min(r[i], limit))
	}
	physicalRadius := float64(max(r[0], r[1], r[2], r[3])) * float64(max(scaleX, scaleY))
	n := 2
	if physicalRadius > .2 {
		angle := math.Acos(max(-1, min(1, 1-.2/physicalRadius)))
		if angle <= 0 {
			n = 64
		} else {
			n = int(min(64, math.Ceil(math.Pi/2/angle)))
		}
		n = min(64, max(2, (n+1)/2*2)) // exact corner bisector
	}
	fringe := min(width/2, .5/max(scaleX, scaleY))
	insets := [4]float32{0, fringe, width - fringe, width}
	var contour [260]borderStation
	count := 0
	for corner := 0; corner < 4; corner++ {
		for step := 0; step <= n; step++ {
			station := &contour[count]
			// A quadrant runs from the previous edge to this clockwise edge.
			station.side = BorderSides(1 << corner)
			if step < n/2 {
				station.side = BorderSides(1 << ((corner + 3) % 4))
			}
			angle := math.Pi + float64(corner)*math.Pi/2 + float64(step)*math.Pi/2/float64(n)
			for band, inset := range insets {
				radius := max(0, r[corner]-inset)
				x, y := rect.X+inset+radius, rect.Y+inset+radius
				if corner == 1 || corner == 2 {
					x = rect.X + rect.Width - inset - radius
				}
				if corner >= 2 {
					y = rect.Y + rect.Height - inset - radius
				}
				station.band[band] = borderPoint{x + radius*float32(math.Cos(angle)), y + radius*float32(math.Sin(angle))}
			}
			count++
		}
	}
	var lengths [260]float64
	total := float64(0)
	for i := 0; i < count; i++ {
		a, b := contour[i], contour[(i+1)%count]
		dx := (float64(b.band[0].x) - float64(a.band[0].x) + float64(b.band[3].x) - float64(a.band[3].x)) / 2
		dy := (float64(b.band[0].y) - float64(a.band[0].y) + float64(b.band[3].y) - float64(a.band[3].y)) / 2
		lengths[i] = math.Hypot(dx, dy)
		total += lengths[i]
	}
	if total == 0 {
		return nil, nil
	}
	dash, gap := max(4, float64(width)*3), max(3, float64(width)*2)
	period := dash + gap
	if total/period > 256 {
		dash *= total / (256 * period)
		period = total / 256
	}
	// Allocate once for the conservative contour + dash-cut upper bound.
	capacity := count
	if dashed {
		capacity += min(512, int(math.Ceil(total/period))*2)
	}
	vertices := make([]BorderVertex, 0, capacity*8)
	indices := make([]int32, 0, capacity*18)
	distance := float64(0)
	for i := 0; i < count; i++ {
		length := lengths[i]
		start, end := distance, distance+length
		distance = end
		if length == 0 || contour[i].side&sides == 0 {
			continue
		}
		a, b := contour[i], contour[(i+1)%count]
		if !dashed {
			vertices, indices = appendBorderStrip(vertices, indices, a, b, 0, 1)
			continue
		}
		first, last := int(math.Floor(start/period)), int(math.Floor(end/period))
		for d := first; d <= last; d++ {
			lo, hi := max(start, float64(d)*period), min(end, float64(d)*period+dash)
			if hi > lo {
				vertices, indices = appendBorderStrip(vertices, indices, a, b, float32((lo-start)/length), float32((hi-start)/length))
			}
		}
	}
	return vertices, indices
}

func appendBorderStrip(vertices []BorderVertex, indices []int32, a, b borderStation, start, end float32) ([]BorderVertex, []int32) {
	base := int32(len(vertices))
	for band := 0; band < 4; band++ {
		coverage := float32(1)
		if band == 0 || band == 3 {
			coverage = 0
		}
		for _, t := range [2]float32{start, end} {
			p, q := a.band[band], b.band[band]
			point := p
			if t >= 1 {
				point = q
			} else if t > 0 {
				point = borderPoint{p.x + (q.x-p.x)*t, p.y + (q.y-p.y)*t}
			}
			vertices = append(vertices, BorderVertex{point.x, point.y, coverage})
		}
	}
	for band := int32(0); band < 3; band++ {
		a := base + band*2
		indices = append(indices, a, a+1, a+3, a, a+3, a+2)
	}
	return vertices, indices
}
