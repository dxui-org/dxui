package sdl3

import (
	"math"
	"testing"

	"github.com/dxui-org/dxui/internal/paint"
	"github.com/dxui-org/dxui/internal/renderer"
)

func TestRoundedCoverageGeometryUsesOnePhysicalPixelFringe(t *testing.T) {
	rect := paint.Rect{X: 10, Y: 20, Width: 40, Height: 24}
	radii := normalizeRadii(rect, paint.Radii{TopLeft: 100, TopRight: 8, BottomRight: 7, BottomLeft: 6})
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		vertices, indices := roundedRingMesh(rect, radii, 1, paint.Color{R: 20, G: 40, B: 60, A: 200}, scale, scale)
		if len(vertices) == 0 || len(indices) == 0 {
			t.Fatalf("scale %g generated no border geometry", scale)
		}
		minimumX, maximumX := float32(1e6), float32(-1e6)
		transparent, covered := 0, 0
		for _, vertex := range vertices {
			minimumX, maximumX = min(minimumX, vertex.X), max(maximumX, vertex.X)
			if vertex.A == 0 {
				transparent++
			} else {
				covered++
			}
		}
		wantMinimum, wantMaximum := rect.X-edgeFringePixels/scale, rect.X+rect.Width+edgeFringePixels/scale
		if abs32(minimumX-wantMinimum) > .001 || abs32(maximumX-wantMaximum) > .001 {
			t.Fatalf("scale %g fringe x=[%g,%g], want [%g,%g]", scale, minimumX, maximumX, wantMinimum, wantMaximum)
		}
		if transparent == 0 || covered == 0 {
			t.Fatalf("scale %g alpha bands transparent=%d covered=%d", scale, transparent, covered)
		}
	}
	if radii.TopLeft != 12 {
		t.Fatalf("oversized radius = %v, want half short edge", radii.TopLeft)
	}
}

func TestRoundedCurveSegmentsFollowPhysicalSagitta(t *testing.T) {
	for _, test := range []struct {
		name   string
		radius float32
		scale  float32
	}{
		{"small-1x", 4, 1},
		{"large-1x", 100, 1},
		{"large-2x", 100, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, radii, _, _ := physicalRoundedRect(paint.Rect{Width: 2 * test.radius, Height: 2 * test.radius}, testUniformRadii(test.radius), test.scale, test.scale)
			segments := roundedSegmentCount(radii, edgeFringePixels)
			physicalRadius := test.radius*test.scale + edgeFringePixels
			sagitta := physicalRadius * (1 - float32(math.Cos(math.Pi/float64(4*segments))))
			if sagitta > curveTolerancePixels+.001 {
				t.Fatalf("segments=%d sagitta=%gpx exceeds %gpx", segments, sagitta, curveTolerancePixels)
			}
		})
	}
}

func TestRoundedFillHasSingleNonOverlappingCoverageBand(t *testing.T) {
	vertices, indices := roundedFillMesh(paint.Rect{X: .25, Y: .5, Width: 32, Height: 20}, testUniformRadii(6), paint.Color{R: 10, G: 20, B: 30, A: 128}, 1.5, 1.5)
	if len(vertices) == 0 || len(indices) == 0 || len(indices)%3 != 0 {
		t.Fatalf("invalid fill mesh vertices=%d indices=%d", len(vertices), len(indices))
	}
	for _, vertex := range vertices {
		if vertex.A != 0 && abs32(vertex.A-float32(128)/255) > .0001 {
			t.Fatalf("unexpected interpolant alpha %g", vertex.A)
		}
		if vertex.R != float32(10)/255 || vertex.G != float32(20)/255 || vertex.B != float32(30)/255 {
			t.Fatalf("transparent fringe changed RGB: %+v", vertex)
		}
	}
}

func TestJoinedRoundedFillKeepsInternalEdgesSolidWithoutFringeTriangles(t *testing.T) {
	rect := paint.Rect{X: 10, Y: 20, Width: 32, Height: 20}
	color := paint.Color{R: 10, G: 20, B: 30, A: 200}
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		vertices, indices := joinedRoundedFillMesh(rect, paint.Radii{}, color, paint.JoinedLeft|paint.JoinedRight, scale, scale)
		if len(vertices) == 0 || len(indices) == 0 {
			t.Fatalf("scale %g joined fill generated no geometry", scale)
		}
		minimumX, maximumX := float32(1e6), float32(-1e6)
		coveredLeading := 0
		for _, vertex := range vertices {
			minimumX, maximumX = min(minimumX, vertex.X), max(maximumX, vertex.X)
			if abs32(vertex.X-rect.X) < .001 && abs32(vertex.A-float32(color.A)/255) < .001 {
				coveredLeading++
			}
		}
		if abs32(minimumX-rect.X) > .001 {
			t.Fatalf("scale %g leading joined edge = %g, want %g", scale, minimumX, rect.X)
		}
		wantMaximum := rect.X + rect.Width - joinedOwnershipInsetPixels/scale
		if abs32(maximumX-wantMaximum) > .001 {
			t.Fatalf("scale %g trailing joined edge = %g, want %g", scale, maximumX, wantMaximum)
		}
		if coveredLeading == 0 {
			t.Fatalf("scale %g joined edge has no fully covered vertices", scale)
		}
		segments := roundedSegmentCount(ellipticalRadii{}, edgeFringePixels)
		perimeter := 4 * (segments + 1)
		wantIndices := perimeter*9 - 2*6
		if len(indices) != wantIndices {
			t.Fatalf("scale %g indices = %d, want %d without two joined fringe quads", scale, len(indices), wantIndices)
		}
	}
}

func TestVerticalJoinedRoundedFillCoversLeadingRasterEdge(t *testing.T) {
	rect := paint.Rect{X: 10, Y: 20, Width: 32, Height: 20}
	color := paint.Color{R: 10, G: 20, B: 30, A: 200}
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		vertices, _ := joinedRoundedFillMesh(rect, paint.Radii{}, color, paint.JoinedTop|paint.JoinedBottom, scale, scale)
		minimumY, maximumY := float32(1e6), float32(-1e6)
		for _, vertex := range vertices {
			minimumY, maximumY = min(minimumY, vertex.Y), max(maximumY, vertex.Y)
		}
		wantMaximum := rect.Y + rect.Height - joinedOwnershipInsetPixels/scale
		if abs32(minimumY-rect.Y) > .001 || abs32(maximumY-wantMaximum) > .001 {
			t.Fatalf("scale %g vertical joined edges = [%g,%g], want [%g,%g]", scale, minimumY, maximumY, rect.Y, wantMaximum)
		}
	}
}

func TestRoundedImageMeshPreservesCoverCropAndDPIFringe(t *testing.T) {
	destination := paint.Rect{X: -50, Width: 200, Height: 100}
	clip := paint.Rect{Width: 100, Height: 100}
	for _, scale := range []float32{1, 1.25, 1.5, 2} {
		vertices, indices := roundedImageMesh(destination, clip, testUniformRadii(50), 128, scale, scale)
		if len(vertices) == 0 || len(indices) == 0 {
			t.Fatalf("scale %g generated no image geometry", scale)
		}
		center := vertices[0]
		if abs32(center.U-.5) > .0001 || abs32(center.V-.5) > .0001 || abs32(center.A-float32(128)/255) > .0001 {
			t.Fatalf("scale %g center vertex = %+v", scale, center)
		}
		transparent := 0
		for _, vertex := range vertices {
			if vertex.A == 0 {
				transparent++
			}
		}
		if transparent == 0 {
			t.Fatalf("scale %g has no antialiased outer fringe", scale)
		}
	}
}

func TestRoundedImageDrawReusesTextureAndAppliesOpacityOnce(t *testing.T) {
	api := newFakeAPI()
	native := &fakeNativeRenderer{api: api}
	driver := &Driver{
		renderer: native, scaleX: 1.5, scaleY: 1.5, activeImage: make(map[textTextureKey]struct{}),
		imageCache: renderer.NewByteCache[textTextureKey, nativeTexture](1024, func(texture nativeTexture) { texture.destroy() }),
	}
	bitmap := &paint.ImageBitmap{Key: "shared", Width: 2, Height: 1, Pixels: make([]byte, 8)}
	if err := driver.DrawImage(paint.Rect{X: -20, Width: 80, Height: 40}, bitmap, 128, paint.Rect{Width: 40, Height: 40}, testUniformRadii(20)); err != nil {
		t.Fatal(err)
	}
	if native.textures != 1 || !native.geometryTexture || len(native.geometryVertices) == 0 {
		t.Fatalf("rounded draw textures=%d textured=%t vertices=%d", native.textures, native.geometryTexture, len(native.geometryVertices))
	}
	if native.lastTexture.alphaMod != 255 || abs32(native.geometryVertices[0].A-float32(128)/255) > .0001 {
		t.Fatalf("opacity texture=%d vertex=%g", native.lastTexture.alphaMod, native.geometryVertices[0].A)
	}
	if err := driver.DrawImage(paint.Rect{Width: 40, Height: 20}, bitmap, 255, paint.Rect{}, paint.Radii{}); err != nil {
		t.Fatal(err)
	}
	if native.textures != 1 {
		t.Fatalf("square and circle created %d textures", native.textures)
	}
}

func TestCombinedFillBorderUsesOneMeshWithoutTransparentColorHalo(t *testing.T) {
	fill := paint.Color{R: 230, G: 235, B: 240, A: 180}
	border := paint.Color{R: 20, G: 30, B: 40, A: 160}
	vertices, indices := roundedFillStrokeMesh(paint.Rect{Width: 40, Height: 24}, testUniformRadii(7), 1, fill, border, 1.25, 1.25)
	if len(vertices) == 0 || len(indices) == 0 {
		t.Fatal("combined fill/border generated no geometry")
	}
	if got := vertices[0]; got.R != float32(fill.R)/255 || got.G != float32(fill.G)/255 || got.B != float32(fill.B)/255 || got.A != float32(fill.A)/255 {
		t.Fatalf("interior vertex = %+v", got)
	}
	transparent := 0
	for _, got := range vertices {
		if got.A == 0 {
			transparent++
			if got.R != float32(border.R)/255 || got.G != float32(border.G)/255 || got.B != float32(border.B)/255 {
				t.Fatalf("outer transparent vertex has halo RGB: %+v", got)
			}
		}
	}
	if transparent == 0 {
		t.Fatal("combined mesh has no outer coverage ramp")
	}
}

func abs32(value float32) float32 {
	if value < 0 {
		return -value
	}
	return value
}

func testUniformRadii(radius float32) paint.Radii {
	return paint.Radii{TopLeft: radius, TopRight: radius, BottomRight: radius, BottomLeft: radius}
}

func BenchmarkRoundedCoverageMeshes(b *testing.B) {
	rect := paint.Rect{X: .25, Y: .5, Width: 120, Height: 44}
	radii := testUniformRadii(12)
	color := paint.Color{R: 20, G: 90, B: 180, A: 180}
	b.ReportAllocs()
	b.Run("fill", func(b *testing.B) {
		for range b.N {
			roundedFillMesh(rect, radii, color, 1.5, 1.5)
		}
	})
	b.Run("border", func(b *testing.B) {
		for range b.N {
			roundedRingMesh(rect, radii, 1, color, 1.5, 1.5)
		}
	})
	b.Run("fill-border", func(b *testing.B) {
		for range b.N {
			roundedFillStrokeMesh(rect, radii, 1, color, paint.Color{R: 30, G: 40, B: 50, A: 255}, 1.5, 1.5)
		}
	})
}

func TestShadowGeometryCacheIsScaleKeyedAndByteBounded(t *testing.T) {
	native := &fakeNativeRenderer{api: newFakeAPI()}
	driver := &Driver{
		renderer: native, scaleX: 1, scaleY: 1,
		shadowCache: renderer.NewByteCache[shadowKey, shadowGeometry](4*shadowGeometryBytes(shadowLayers), nil),
	}
	shadow := paint.Shadow{Blur: 8, Spread: 2, Color: paint.Color{A: 80}}
	for index, scale := range []float32{1, 1.25, 1.5, 2} {
		driver.scaleX, driver.scaleY = scale, scale
		if err := driver.DrawShadow(paint.Rect{Width: 20, Height: 20}, paint.Radii{}, shadow); err != nil {
			t.Fatal(err)
		}
		if stats := driver.shadowCache.Stats(); stats.Entries != index+1 || stats.Bytes > stats.BudgetBytes {
			t.Fatalf("scale %g cache stats = %+v", scale, stats)
		}
	}
	if err := driver.DrawShadow(paint.Rect{Width: 20, Height: 20}, paint.Radii{}, shadow); err != nil {
		t.Fatal(err)
	}
	if stats := driver.shadowCache.Stats(); stats.Entries != 4 {
		t.Fatalf("same scale missed cache: %+v", stats)
	}
	if err := driver.DrawShadow(paint.Rect{Width: 20, Height: 20}, paint.Radii{}, paint.Shadow{}); err != nil {
		t.Fatal(err)
	}
	if stats := driver.shadowCache.Stats(); stats.Entries != 4 {
		t.Fatalf("transparent shadow allocated cache entry: %+v", stats)
	}
}

func TestTextTextureCacheHitsEvictsAndReleases(t *testing.T) {
	api := newFakeAPI()
	native := &fakeNativeRenderer{api: api}
	entryBytes := textTextureEntryOverhead + 2*2*4
	driver := &Driver{
		renderer: native, scaleX: 1, scaleY: 1,
		textCache: renderer.NewByteCache[textTextureKey, nativeTexture](entryBytes, func(texture nativeTexture) { texture.destroy() }),
	}
	bitmap := &paint.TextBitmap{Key: "font:size:scale:gray:first", Width: 2, Height: 2, Pixels: []byte{0, 80, 160, 255}}
	if err := driver.DrawText(paint.Rect{Width: 2, Height: 2}, bitmap, paint.Color{R: 20, G: 30, B: 40, A: 255}); err != nil {
		t.Fatal(err)
	}
	if err := driver.DrawText(paint.Rect{Width: 2, Height: 2}, bitmap, paint.Color{R: 200, G: 30, B: 40, A: 128}); err != nil {
		t.Fatal(err)
	}
	if native.textures != 1 {
		t.Fatalf("same alpha mask created %d textures, want 1", native.textures)
	}
	second := &paint.TextBitmap{Key: "font:size:scale:gray:second", Width: 2, Height: 2, Pixels: bitmap.Pixels}
	if err := driver.DrawText(paint.Rect{Width: 2, Height: 2}, second, paint.Color{A: 255}); err != nil {
		t.Fatal(err)
	}
	stats := driver.textCache.Stats()
	if stats.Entries != 1 || stats.Bytes > stats.BudgetBytes || stats.Evictions == 0 || native.textures != 2 {
		t.Fatalf("bounded texture cache stats=%+v textures=%d", stats, native.textures)
	}
	destroyedBeforeReset := countAction(api.actions, "destroy-texture")
	if _, err := driver.translate(nativeEvent{kind: nativeEventRendererReset}); err != nil {
		t.Fatal(err)
	}
	if got := countAction(api.actions, "destroy-texture"); got != destroyedBeforeReset+1 || driver.textCache.Stats().Entries != 0 {
		t.Fatalf("reset releases = %d -> %d, stats=%+v", destroyedBeforeReset, got, driver.textCache.Stats())
	}
}

func TestIconTextureCacheReusesMaskAcrossColorsAndPinsActiveBudget(t *testing.T) {
	api := newFakeAPI()
	native := &fakeNativeRenderer{api: api}
	entryBytes := textTextureEntryOverhead + 3*3*4
	driver := &Driver{
		renderer: native, scaleX: 1.25, scaleY: 1.25,
		textCache:  renderer.NewByteCache[textTextureKey, nativeTexture](entryBytes, func(texture nativeTexture) { texture.destroy() }),
		activeText: make(map[textTextureKey]struct{}), nextText: make(map[textTextureKey]struct{}),
		activeImage: make(map[textTextureKey]struct{}), nextImage: make(map[textTextureKey]struct{}),
	}
	bitmap := &paint.TextBitmap{Key: "icon:search:24:1.25:2", Width: 3, Height: 3, Pixels: make([]byte, 9)}
	list := paint.DisplayList{{Kind: paint.CommandDrawIcon, Text: bitmap}}
	driver.syncTextureReferences(list)
	if err := driver.DrawIcon(paint.Rect{Width: 3, Height: 3}, bitmap, paint.Color{R: 10, A: 255}); err != nil {
		t.Fatal(err)
	}
	if err := driver.DrawIcon(paint.Rect{Width: 3, Height: 3}, bitmap, paint.Color{B: 220, A: 120}); err != nil {
		t.Fatal(err)
	}
	if native.textures != 1 {
		t.Fatalf("color-only redraw created %d textures", native.textures)
	}
	second := &paint.TextBitmap{Key: "icon:other:24:1.25:2", Width: 3, Height: 3, Pixels: make([]byte, 9)}
	if err := driver.DrawIcon(paint.Rect{Width: 3, Height: 3}, second, paint.Color{A: 255}); err != nil {
		t.Fatal(err)
	}
	if stats := driver.textCache.Stats(); stats.Entries != 1 || stats.Bytes > stats.BudgetBytes {
		t.Fatalf("pinned budget stats = %+v", stats)
	}
	if _, ok := driver.textCache.Get(textTextureKey{Identity: bitmap.Key, Width: 3, Height: 3}); !ok {
		t.Fatal("active icon texture was evicted")
	}
	driver.syncTextureReferences(nil)
	if _, err := driver.translate(nativeEvent{kind: nativeEventRendererReset}); err != nil {
		t.Fatal(err)
	}
	if driver.textCache.Stats().Entries != 0 {
		t.Fatal("renderer reset retained icon texture")
	}
}

func TestOversizedTextTextureRendersOnceThenReleases(t *testing.T) {
	api := newFakeAPI()
	native := &fakeNativeRenderer{api: api}
	driver := &Driver{
		renderer:  native,
		textCache: renderer.NewByteCache[textTextureKey, nativeTexture](1, func(texture nativeTexture) { texture.destroy() }),
	}
	bitmap := &paint.TextBitmap{Key: "oversized", Width: 2, Height: 2, Pixels: []byte{1, 2, 3, 4}}
	if err := driver.DrawText(paint.Rect{Width: 2, Height: 2}, bitmap, paint.Color{A: 255}); err != nil {
		t.Fatal(err)
	}
	if driver.textCache.Stats().Entries != 0 || countAction(api.actions, "destroy-texture") != 1 {
		t.Fatalf("oversized resource was retained: stats=%+v actions=%v", driver.textCache.Stats(), api.actions)
	}
}

func TestImageTextureCacheReusesEvictsAndReleases(t *testing.T) {
	api := newFakeAPI()
	native := &fakeNativeRenderer{api: api}
	entryBytes := textTextureEntryOverhead + 2*2*4
	driver := &Driver{
		renderer: native, activeImage: make(map[textTextureKey]struct{}),
		imageCache: renderer.NewByteCache[textTextureKey, nativeTexture](entryBytes, func(texture nativeTexture) { texture.destroy() }),
	}
	first := &paint.ImageBitmap{Key: "first", Width: 2, Height: 2, Pixels: make([]byte, 16)}
	if err := driver.DrawImage(paint.Rect{Width: 2, Height: 2}, first, 255, paint.Rect{}, paint.Radii{}); err != nil {
		t.Fatal(err)
	}
	if err := driver.DrawImage(paint.Rect{Width: 2, Height: 2}, first, 128, paint.Rect{}, paint.Radii{}); err != nil {
		t.Fatal(err)
	}
	if native.textures != 1 {
		t.Fatalf("same image created %d textures", native.textures)
	}
	second := &paint.ImageBitmap{Key: "second", Width: 2, Height: 2, Pixels: make([]byte, 16)}
	if err := driver.DrawImage(paint.Rect{Width: 2, Height: 2}, second, 255, paint.Rect{}, paint.Radii{}); err != nil {
		t.Fatal(err)
	}
	if stats := driver.imageCache.Stats(); stats.Entries != 1 || stats.Evictions == 0 || native.textures != 2 {
		t.Fatalf("image cache stats=%+v textures=%d", stats, native.textures)
	}
	destroyed := countAction(api.actions, "destroy-texture")
	if _, err := driver.translate(nativeEvent{kind: nativeEventRendererReset}); err != nil {
		t.Fatal(err)
	}
	if countAction(api.actions, "destroy-texture") != destroyed+1 || driver.imageCache.Stats().Entries != 0 {
		t.Fatal("renderer reset did not release image textures")
	}
}

func TestActiveOversizedImageTextureReusesUntilDisplayRelease(t *testing.T) {
	api := newFakeAPI()
	native := &fakeNativeRenderer{api: api}
	bitmap := &paint.ImageBitmap{Key: "oversized", Width: 2, Height: 2, Pixels: make([]byte, 16)}
	key := textTextureKey{Identity: bitmap.Key, Width: bitmap.Width, Height: bitmap.Height}
	driver := &Driver{
		renderer: native, activeImage: map[textTextureKey]struct{}{key: {}},
		imageCache: renderer.NewByteCache[textTextureKey, nativeTexture](1, func(texture nativeTexture) { texture.destroy() }),
	}
	for range 2 {
		if err := driver.DrawImage(paint.Rect{Width: 2, Height: 2}, bitmap, 255, paint.Rect{}, paint.Radii{}); err != nil {
			t.Fatal(err)
		}
	}
	if native.textures != 1 || driver.imageCache.Stats().Entries != 1 {
		t.Fatalf("textures=%d stats=%+v", native.textures, driver.imageCache.Stats())
	}
	driver.imageCache.Clear()
	if err := driver.DrawImage(paint.Rect{Width: 2, Height: 2}, bitmap, 255, paint.Rect{}, paint.Radii{}); err != nil {
		t.Fatal(err)
	}
	if native.textures != 2 || driver.imageCache.Stats().Entries != 1 {
		t.Fatalf("reset recovery textures=%d stats=%+v", native.textures, driver.imageCache.Stats())
	}
	driver.imageCache.Release(key)
	if driver.imageCache.Stats().Entries != 0 || countAction(api.actions, "destroy-texture") != 2 {
		t.Fatal("released oversized texture was not destroyed")
	}
}

func TestActiveDisplayTextTextureIsPinnedUntilDisplayChanges(t *testing.T) {
	api := newFakeAPI()
	native := &fakeNativeRenderer{api: api}
	entryBytes := textTextureEntryOverhead + 2*2*4
	driver := &Driver{
		renderer: native, activeText: make(map[textTextureKey]struct{}),
		textCache: renderer.NewByteCache[textTextureKey, nativeTexture](entryBytes, func(texture nativeTexture) { texture.destroy() }),
	}
	first := &paint.TextBitmap{Key: "first", Width: 2, Height: 2, Pixels: []byte{1, 2, 3, 4}}
	second := &paint.TextBitmap{Key: "second", Width: 2, Height: 2, Pixels: []byte{4, 3, 2, 1}}
	display := func(bitmap *paint.TextBitmap) paint.DisplayList {
		return paint.DisplayList{{Kind: paint.CommandDrawText, Rect: paint.Rect{Width: 2, Height: 2}, Text: bitmap, Color: paint.Color{A: 255}}}
	}
	if err := driver.Render(display(first)); err != nil {
		t.Fatal(err)
	}
	if err := driver.DrawText(paint.Rect{Width: 2, Height: 2}, second, paint.Color{A: 255}); err != nil {
		t.Fatal(err)
	}
	if stats := driver.textCache.Stats(); stats.Entries != 1 {
		t.Fatalf("active texture was evicted: %+v", stats)
	}
	if err := driver.Render(display(second)); err != nil {
		t.Fatal(err)
	}
	if stats := driver.textCache.Stats(); stats.Entries != 1 || stats.Bytes > stats.BudgetBytes {
		t.Fatalf("changed display did not release/reuse capacity: %+v", stats)
	}
}

func countAction(actions []string, target string) int {
	count := 0
	for _, action := range actions {
		if action == target {
			count++
		}
	}
	return count
}
