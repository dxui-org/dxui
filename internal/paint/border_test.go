package paint

import (
	"math"
	"reflect"
	"testing"
)

// Sample strictly inside triangles, away from shared raster edges. A sum above
// one exposes double ownership even when the source color is translucent.
func borderCoverage(vertices []BorderVertex, indices []int32, x, y float32) float32 {
	coverage := float32(0)
	for i := 0; i < len(indices); i += 3 {
		a, b, c := vertices[indices[i]], vertices[indices[i+1]], vertices[indices[i+2]]
		den := (b.Y-c.Y)*(a.X-c.X) + (c.X-b.X)*(a.Y-c.Y)
		if den == 0 {
			continue
		}
		u := ((b.Y-c.Y)*(x-c.X) + (c.X-b.X)*(y-c.Y)) / den
		v := ((c.Y-a.Y)*(x-c.X) + (a.X-c.X)*(y-c.Y)) / den
		w := 1 - u - v
		if u > 0 && v > 0 && w > 0 {
			coverage += u*a.Coverage + v*b.Coverage + w*c.Coverage
		}
	}
	return coverage
}

func TestBorderSidesGeometry(t *testing.T) {
	for _, radii := range []Radii{{}, {12, 12, 12, 12}, {3, 9, 12, 6}} {
		for _, dashed := range []bool{false, true} {
			rect := Rect{Width: 60, Height: 40}
			all, ai := BorderMesh(rect, radii, 4, BorderAll, dashed, 1.25, 1.5)
			zero, zi := BorderMesh(rect, radii, 4, 0, dashed, 1.25, 1.5)
			if !reflect.DeepEqual(all, zero) || !reflect.DeepEqual(ai, zi) {
				t.Fatal("zero sides differs from all")
			}
			var single [4][]BorderVertex
			var si [4][]int32
			for side := 0; side < 4; side++ {
				single[side], si[side] = BorderMesh(rect, radii, 4, 1<<side, dashed, 1.25, 1.5)
			}
			for mask := BorderSides(1); mask <= BorderAll; mask++ {
				vertices, indices := BorderMesh(rect, radii, 4, mask, dashed, 1.25, 1.5)
				for _, v := range vertices {
					if v.X < -.0001 || v.Y < -.0001 || v.X > 60.0001 || v.Y > 40.0001 {
						t.Fatalf("outside layout: %+v", v)
					}
				}
				for y := float32(.173); y < 40; y += 1.371 {
					for x := float32(.319); x < 60; x += 1.913 {
						got := borderCoverage(vertices, indices, x, y)
						want := float32(0)
						for side := 0; side < 4; side++ {
							if mask&(1<<side) != 0 {
								want += borderCoverage(single[side], si[side], x, y)
							}
						}
						if got > 1.0001 || math.Abs(float64(got-want)) > .0001 {
							t.Fatalf("radii=%v dashed=%t mask=%d point=%g,%g coverage=%g sum=%g", radii, dashed, mask, x, y, got, want)
						}
					}
				}
			}
		}
	}
}

func TestBorderSingleEdgesAndCornerOwnership(t *testing.T) {
	points := [4][2]float32{{30.17, 1.71}, {58.19, 20.13}, {30.17, 38.21}, {1.79, 20.13}}
	for side := 0; side < 4; side++ {
		v, i := BorderMesh(Rect{Width: 60, Height: 40}, Radii{12, 12, 12, 12}, 4, 1<<side, false, 1, 1)
		for edge, p := range points {
			got := borderCoverage(v, i, p[0], p[1])
			if (edge == side && got < .99) || (edge != side && got != 0) {
				t.Fatalf("side=%d edge=%d coverage=%g", side, edge, got)
			}
		}
	}
	// Both halves of the top-right arc meet with one alpha contribution.
	v, i := BorderMesh(Rect{Width: 60, Height: 40}, Radii{12, 12, 12, 12}, 4, BorderTop|BorderRight, false, 2, 2)
	for _, angle := range []float64{-.9, -.8, -.77, -.6} {
		x, y := 48+10*float32(math.Cos(angle)), 12+10*float32(math.Sin(angle))
		alpha := .4 * borderCoverage(v, i, x, y)
		if math.Abs(float64(alpha-.4)) > .0001 {
			t.Fatalf("translucent join alpha=%g", alpha)
		}
	}
}

func TestBorderMeshBoundsAndDegenerates(t *testing.T) {
	for _, size := range []float32{.01, 1, 100, 1e9, math.MaxFloat32 / 4} {
		for _, width := range []float32{.001, 1, 5000} {
			v, i := BorderMesh(Rect{Width: size, Height: size}, Radii{size, size, size, size}, width, BorderAll, true, 2, 2)
			if len(v) > 6176 || len(i) > 13896 {
				t.Fatalf("mesh exceeds bound: %d/%d", len(v), len(i))
			}
			for _, p := range v {
				if !finite(p.X) || !finite(p.Y) {
					t.Fatal("nonfinite vertex")
				}
			}
		}
	}
	for _, width := range []float32{0, -1, float32(math.NaN()), float32(math.Inf(1))} {
		v, i := BorderMesh(Rect{Width: 10, Height: 10}, Radii{}, width, 0, true, 1, 1)
		if len(v)+len(i) != 0 {
			t.Fatal("invalid width generated mesh")
		}
	}
}

func TestBorderReplayOpacityAndClip(t *testing.T) {
	r := &RecordingPainter{}
	err := Replay(DisplayList{
		{Kind: CommandPushClip, Rect: Rect{Width: 40, Height: 20}},
		{Kind: CommandPushOpacity, Opacity: .5},
		{Kind: CommandBorder, Rect: Rect{Width: 60, Height: 40}, Width: 2, Color: Color{R: 20, A: 128}, Sides: BorderBottom, Dashed: true},
		{Kind: CommandPopOpacity}, {Kind: CommandPopClip},
	}, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Records) != 3 || r.Records[1].Color.A != 64 || r.Records[1].Sides != BorderBottom || !r.Records[1].Dashed {
		t.Fatalf("replay=%+v", r.Records)
	}
}

func BenchmarkBorderMesh(b *testing.B) {
	for _, dashed := range []bool{false, true} {
		name := "solid"
		if dashed {
			name = "dashed"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				BorderMesh(Rect{Width: 160, Height: 44}, Radii{12, 12, 12, 12}, 2, BorderBottom, dashed, 1.25, 1.25)
			}
		})
	}
}
