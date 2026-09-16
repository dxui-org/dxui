package text

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

type countingReaderAt struct {
	data  []byte
	calls int
	bytes int
	err   error
	short bool
}

func (r *countingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	r.calls++
	if r.err != nil {
		return 0, r.err
	}
	if off < 0 || off >= int64(len(r.data)) {
		return 0, errors.New("outside source")
	}
	n := copy(p, r.data[off:])
	if r.short && n > 0 {
		n--
		return n, errors.New("short read")
	}
	r.bytes += n
	if n != len(p) {
		return n, errors.New("short source")
	}
	return n, nil
}

type countingCloser struct {
	closes int
	err    error
}

func (c *countingCloser) Close() error { c.closes++; return c.err }

type runeMeasurer struct{}

func (runeMeasurer) Measure(request MeasureRequest) (Metrics, error) {
	return Metrics{Width: float32(len([]rune(request.Text))), Height: 1}, nil
}

func TestWrapWordsUsesInjectedMeasurer(t *testing.T) {
	lines, err := WrapWords("one two 三四", 7, runeMeasurer{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"one two", "三四"}
	if fmt.Sprint(lines) != fmt.Sprint(want) {
		t.Fatalf("lines = %#v, want %#v", lines, want)
	}
}

func TestLayoutAndRasterUseSameMetrics(t *testing.T) {
	engine := testEngine(t, 64<<10)
	request := Request{Text: "AV 123\nwrapped words here", Size: 18, LineHeight: 24, Wrap: WordWrap, MaxWidth: 105, MaxWidthSet: true}
	layout, err := engine.Layout(request)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Metrics.Lines < 2 || layout.Metrics.Height != float32(layout.Metrics.Lines)*24 {
		t.Fatalf("metrics = %+v", layout.Metrics)
	}
	bitmaps, err := engine.RasterizeLines(layout, 1.5, 1.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(bitmaps) != layout.Metrics.Lines {
		t.Fatalf("bitmap lines = %d, metric lines = %d", len(bitmaps), layout.Metrics.Lines)
	}
	for _, bitmap := range bitmaps {
		if bitmap.LogicalHeight < 24 || bitmap.LogicalHeight > 25 || !hasInk(bitmap.Alpha) {
			t.Fatalf("bitmap does not match logical line metrics: %+v", bitmap)
		}
	}
}

func TestEmptyNewlineLongAndInvalidUTF8(t *testing.T) {
	engine := testEngine(t, 64<<10)
	tests := []struct {
		name  string
		value string
		lines int
	}{
		{name: "empty", value: "", lines: 0},
		{name: "newline", value: "\n", lines: 2},
		{name: "long", value: strings.Repeat("label ", 300), lines: 300},
		{name: "invalid", value: string([]byte{'a', 0xff, 'b'}), lines: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := Request{Text: test.value}
			if test.name == "long" {
				request.Wrap, request.MaxWidth, request.MaxWidthSet = WordWrap, 40, true
			}
			layout, err := engine.Layout(request)
			if err != nil {
				t.Fatal(err)
			}
			if layout.Metrics.Lines != test.lines {
				t.Fatalf("lines = %d, want %d", layout.Metrics.Lines, test.lines)
			}
			if !utf8.ValidString(layout.Request.Text) {
				t.Fatalf("layout retained invalid UTF-8 %q", layout.Request.Text)
			}
		})
	}
}

func TestFallbackAndMissingGlyphPolicy(t *testing.T) {
	cmap, err := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(Options{Fonts: []FontSpec{
		{Family: "latin", Weight: 400, Data: goregular.TTF, Identity: "latin"},
		{Family: "cjk-test", Weight: 400, Data: cmap, Identity: "cjk-test"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	layout, err := engine.Layout(Request{Text: "A中", Families: []string{"latin", "cjk-test"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := layout.Lines[0].Glyphs[1].FontIdentity; got != "cjk-test" {
		t.Fatalf("CJK face = %q, want explicit fallback", got)
	}
	missing, err := engine.Layout(Request{Text: "🙂", Families: []string{"latin", "cjk-test"}})
	if err != nil {
		t.Fatal(err)
	}
	if missing.Metrics.MissingGlyphs != 1 || missing.Lines[0].Glyphs[0].Rune != utf8.RuneError {
		t.Fatalf("missing glyph policy = %+v", missing)
	}
}

type fakeFallbackResolver struct {
	specs []FontSpec
	calls int
	tries int
}

func (resolver *fakeFallbackResolver) Resolve(_ rune, remaining int, try func(FontSpec) (bool, error)) {
	resolver.calls++
	for len(resolver.specs) != 0 {
		spec := resolver.specs[0]
		resolver.specs = resolver.specs[1:]
		resolver.tries++
		if len(spec.Data) > remaining {
			continue
		}
		covered, err := try(spec)
		if err == nil {
			remaining -= len(spec.Data)
			if covered {
				return
			}
		}
	}
}

func TestSystemFallbackIsLazyCachedAndNonFatal(t *testing.T) {
	cmap, err := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		specs        []FontSpec
		wantIdentity string
		wantMissing  int
		wantTries    int
	}{
		{name: "first candidate", specs: []FontSpec{{Family: "system-cjk", Data: cmap, Identity: "first"}}, wantIdentity: "first", wantTries: 1},
		{name: "corrupt then covering", specs: []FontSpec{{Family: "system-cjk", Data: []byte("broken"), Identity: "bad"}, {Family: "system-cjk", Data: cmap, Identity: "second"}}, wantIdentity: "second", wantTries: 2},
		{name: "non-covering then covering", specs: []FontSpec{{Family: "system-cjk", Data: goregular.TTF, Identity: "latin-only"}, {Family: "system-cjk", Data: cmap, Identity: "second"}}, wantIdentity: "second", wantTries: 2},
		{name: "all fail", specs: []FontSpec{{Family: "system-cjk", Data: []byte("broken"), Identity: "bad"}, {Family: "system-cjk", Data: goregular.TTF, Identity: "latin-only"}}, wantMissing: 1, wantTries: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := &fakeFallbackResolver{specs: append([]FontSpec(nil), test.specs...)}
			engine, err := NewEngine(Options{FallbackResolver: resolver, FontSourceBytes: len(cmap) + len(goregular.TTF) + 1024})
			if err != nil {
				t.Fatal(err)
			}
			defer engine.Close()
			if _, err := engine.Layout(Request{Text: "Latin only"}); err != nil {
				t.Fatal(err)
			}
			if resolver.calls != 0 {
				t.Fatalf("Latin layout called resolver %d times", resolver.calls)
			}
			layout, err := engine.Layout(Request{Text: "中"})
			if err != nil {
				t.Fatal(err)
			}
			if got := layout.Lines[0].Glyphs[0].FontIdentity; test.wantIdentity != "" && got != test.wantIdentity {
				t.Fatalf("identity = %q, want %q", got, test.wantIdentity)
			}
			if layout.Metrics.MissingGlyphs != test.wantMissing || resolver.tries != test.wantTries {
				t.Fatalf("metrics=%+v tries=%d", layout.Metrics, resolver.tries)
			}
			if _, err := engine.Layout(Request{Text: "中中文"}); err != nil {
				t.Fatal(err)
			}
			if resolver.tries != test.wantTries {
				t.Fatalf("repeated CJK retried candidates: %d", resolver.tries)
			}
		})
	}
}

func TestApplicationFontsAndExplicitFamiliesPrecedeSystemFallback(t *testing.T) {
	cmap, err := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeFallbackResolver{specs: []FontSpec{{Family: "system-cjk", Data: cmap, Identity: "system"}}}
	engine, err := NewEngine(Options{Fonts: []FontSpec{
		{Family: "first", Data: cmap, Identity: "first"},
		{Family: "second", Data: cmap, Identity: "second"},
	}, FallbackResolver: resolver, FontSourceBytes: len(cmap) * 3})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	for _, test := range []struct {
		families []string
		want     string
	}{{want: "first"}, {families: []string{"second", "first"}, want: "second"}, {families: []string{"first", "second"}, want: "first"}} {
		layout, err := engine.Layout(Request{Text: "中", Families: test.families})
		if err != nil {
			t.Fatal(err)
		}
		if got := layout.Lines[0].Glyphs[0].FontIdentity; got != test.want {
			t.Fatalf("families=%v identity=%q want=%q", test.families, got, test.want)
		}
	}
	if resolver.calls != 0 {
		t.Fatalf("application coverage called system resolver %d times", resolver.calls)
	}
}

func TestSystemFallbackHonorsFontSourceBudget(t *testing.T) {
	cmap, err := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeFallbackResolver{specs: []FontSpec{{Family: "system-cjk", Data: cmap, Identity: "too-large"}}}
	engine, err := NewEngine(Options{FallbackResolver: resolver, FontSourceBytes: len(cmap) - 1})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	layout, err := engine.Layout(Request{Text: "中"})
	if err != nil || layout.Metrics.MissingGlyphs != 1 || resolver.tries != 1 || len(engine.systemFaces) != 0 {
		t.Fatalf("layout=%+v tries=%d faces=%d err=%v", layout, resolver.tries, len(engine.systemFaces), err)
	}
}

func TestSystemFallbackRejectsOutOfRangeCollectionFaceAndContinues(t *testing.T) {
	cmap, err := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeFallbackResolver{specs: []FontSpec{
		{Family: "system-cjk", Data: cmap, FaceIndex: 1, Identity: "bad-index"},
		{Family: "system-cjk", Data: cmap, FaceIndex: 0, Identity: "valid"},
	}}
	engine, err := NewEngine(Options{FallbackResolver: resolver, FontSourceBytes: len(cmap) * 2})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	layout, err := engine.Layout(Request{Text: "中"})
	if err != nil || layout.Lines[0].Glyphs[0].FontIdentity != "valid" || resolver.tries != 2 {
		t.Fatalf("layout=%+v tries=%d err=%v", layout, resolver.tries, err)
	}
}

func TestLegacyKerningChangesAdvance(t *testing.T) {
	engine := testEngine(t, 64<<10)
	pair, err := engine.Layout(Request{Text: "AV", Size: 32})
	if err != nil {
		t.Fatal(err)
	}
	glyphs := pair.Lines[0].Glyphs
	kern, err := engine.faces[glyphs[0].face].font.Kern(nil, glyphs[0].glyph, glyphs[1].glyph, fixed.I(32), font.HintingNone)
	if err != nil {
		t.Fatal(err)
	}
	wantSecondX := glyphs[0].Advance + fixedToFloat(kern)
	if glyphs[1].X != wantSecondX || pair.Metrics.Width != glyphs[1].X+glyphs[1].Advance {
		t.Fatalf("sfnt.Kern placement mismatch: glyphs=%+v kern=%v width=%v", glyphs, kern, pair.Metrics.Width)
	}
}

func TestScaleChangesRasterButNotLogicalLayout(t *testing.T) {
	engine := testEngine(t, 64<<10)
	layout, err := engine.Layout(Request{Text: "Scale stable", Size: 16})
	if err != nil {
		t.Fatal(err)
	}
	one, err := engine.RasterizeLines(layout, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	two, err := engine.RasterizeLines(layout, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if layout.Metrics.Width <= 0 || len(one) != 1 || len(two) != 1 || float32(two[0].Width) < float32(one[0].Width)*1.8 || one[0].Key == two[0].Key {
		t.Fatalf("scale/layout mismatch: metrics=%+v widths=%d/%d keys-equal=%v", layout.Metrics, one[0].Width, two[0].Width, one[0].Key == two[0].Key)
	}
}

func TestRasterSourceBudgetIsPreflighted(t *testing.T) {
	engine, err := NewEngine(Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	layout, err := engine.Layout(Request{Text: "budgeted mask"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.RasterizeLinesWithin(layout, 1, 1, 1); err == nil {
		t.Fatal("raster source exceeding byte budget was accepted")
	}
	bitmaps, err := engine.RasterizeLinesWithin(layout, 1, 1, 1<<20)
	if err != nil || len(bitmaps) != 1 || bitmaps[0].AccountedBytes <= len(bitmaps[0].Alpha) {
		t.Fatalf("budgeted raster = %+v, err=%v", bitmaps, err)
	}
}

func TestRasterSourceReusesMatchingRetainedMask(t *testing.T) {
	engine, err := NewEngine(Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	layout, err := engine.Layout(Request{Text: "retained mask"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := engine.RasterizeLinesWithin(layout, 1, 1, 1<<20)
	if err != nil || len(first) != 1 {
		t.Fatalf("first raster = %+v, err=%v", first, err)
	}
	reused := 0
	second, err := engine.RasterizeLinesWithinReuse(layout, 1, 1, 1<<20, func(key string, width, height int) ([]byte, bool) {
		reused++
		if key != first[0].Key || width != first[0].Width || height != first[0].Height {
			return nil, false
		}
		return first[0].Alpha, true
	})
	if err != nil || reused != 1 || len(second) != 1 || &second[0].Alpha[0] != &first[0].Alpha[0] {
		t.Fatalf("reuse calls=%d second=%+v err=%v", reused, second, err)
	}
}

func TestMeasureCacheHitsEvictsAndNeverExceedsBudget(t *testing.T) {
	engine := testEngine(t, 2048)
	request := Request{Text: "reused label"}
	first, err := engine.Layout(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := engine.Layout(request)
	if err != nil || first != second {
		t.Fatalf("cache did not return shared immutable layout: %v", err)
	}
	for index := 0; index < 100; index++ {
		if _, err := engine.Layout(Request{Text: fmt.Sprintf("label-%03d", index)}); err != nil {
			t.Fatal(err)
		}
	}
	stats := engine.Stats()
	if stats.Hits == 0 || stats.Evictions == 0 || stats.Bytes > stats.BudgetBytes {
		t.Fatalf("cache stats = %+v", stats)
	}
	engine.Close()
	if stats := engine.Stats(); stats.Bytes != 0 || stats.Entries != 0 {
		t.Fatalf("closed cache stats = %+v", stats)
	}
}

func TestReaderAtFontParsesLayoutsCachesRasterizesAndClosesOnce(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	if err != nil {
		t.Fatal(err)
	}
	reader, closer := &countingReaderAt{data: data}, &countingCloser{}
	source := &FontSource{ReaderAt: reader, Size: int64(len(data)), Identity: "reader", Closer: closer}
	engine, err := NewEngine(Options{Fonts: []FontSpec{{Family: "reader", Source: source, Identity: "reader#0"}}, FontSourceBytes: len(data)})
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Text: "中", Families: []string{"reader"}}
	layout, err := engine.Layout(request)
	if err != nil {
		t.Fatal(err)
	}
	afterLayout := reader.calls
	if _, err := engine.Layout(request); err != nil {
		t.Fatal(err)
	}
	if reader.calls != afterLayout {
		t.Fatalf("layout cache hit added reads: %d -> %d", afterLayout, reader.calls)
	}
	if _, err := engine.RasterizeLines(layout, 1, 1); err != nil {
		t.Fatal(err)
	}
	if reader.calls <= afterLayout {
		t.Fatal("rasterization did not read glyph data")
	}
	stats := engine.Stats()
	if stats.OpenFontSources != 1 || stats.FileFontLogicalBytes != int64(len(data)) || stats.MemoryFontBytes != 0 {
		t.Fatalf("stats=%+v", stats)
	}
	engine.Close()
	engine.Close()
	if closer.closes != 1 {
		t.Fatalf("close count=%d", closer.closes)
	}
}

func TestReaderAtFailuresAndPartialInitializationCloseSources(t *testing.T) {
	data, _ := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	for _, tc := range []struct {
		name   string
		reader *countingReaderAt
	}{
		{name: "read error", reader: &countingReaderAt{data: data, err: errors.New("read failure")}},
		{name: "short read", reader: &countingReaderAt{data: data, short: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			closer := &countingCloser{}
			_, err := NewEngine(Options{Fonts: []FontSpec{{Family: "bad", Source: &FontSource{ReaderAt: tc.reader, Size: int64(len(data)), Closer: closer}}}, FontSourceBytes: len(data)})
			if err == nil || closer.closes != 1 {
				t.Fatalf("err=%v closes=%d", err, closer.closes)
			}
		})
	}
	firstClose, badClose := &countingCloser{}, &countingCloser{}
	_, err := NewEngine(Options{Fonts: []FontSpec{
		{Family: "ok", Source: &FontSource{ReaderAt: bytes.NewReader(data), Size: int64(len(data)), Closer: firstClose}},
		{Family: "bad", Source: &FontSource{ReaderAt: bytes.NewReader([]byte("bad")), Size: 3, Closer: badClose}},
	}, FontSourceBytes: len(data) + 3})
	if err == nil || firstClose.closes != 1 || badClose.closes != 1 {
		t.Fatalf("err=%v closes=%d/%d", err, firstClose.closes, badClose.closes)
	}
}

func TestNonCoveringSystemFaceCanCoverLaterRune(t *testing.T) {
	cmap, _ := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	resolver := &fakeFallbackResolver{specs: []FontSpec{{Family: "system-cjk", Data: cmap, Identity: "cjk"}}}
	engine, err := NewEngine(Options{FallbackResolver: resolver, FontSourceBytes: len(cmap)})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	if _, err := engine.Layout(Request{Text: "\U00031350"}); err != nil {
		t.Fatal(err)
	}
	layout, err := engine.Layout(Request{Text: "中"})
	if err != nil || layout.Lines[0].Glyphs[0].FontIdentity != "cjk" || resolver.tries != 1 {
		t.Fatalf("layout=%+v tries=%d err=%v", layout, resolver.tries, err)
	}
}

func BenchmarkLayoutShortLabels1000(b *testing.B) {
	engine, err := NewEngine(Options{MeasureCacheBytes: 8 << 20})
	if err != nil {
		b.Fatal(err)
	}
	defer engine.Close()
	labels := make([]string, 1000)
	for index := range labels {
		labels[index] = fmt.Sprintf("Label %d", index)
		_, _ = engine.Layout(Request{Text: labels[index]})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		for _, label := range labels {
			if _, err := engine.Layout(Request{Text: label}); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkLayoutLatinCachedWithoutSystemFallbackWork(b *testing.B) {
	resolver := &fakeFallbackResolver{}
	engine, err := NewEngine(Options{MeasureCacheBytes: 64 << 10, FontSourceBytes: 32 << 20, FallbackResolver: resolver})
	if err != nil {
		b.Fatal(err)
	}
	defer engine.Close()
	request := Request{Text: "Latin idle label"}
	if _, err := engine.Layout(request); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Layout(request); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	if resolver.calls != 0 {
		b.Fatalf("Latin path called fallback resolver %d times", resolver.calls)
	}
}

func BenchmarkReaderAtCJKFirstParseAndLayout(b *testing.B) {
	data, _ := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	b.ReportAllocs()
	var reads int
	for i := 0; i < b.N; i++ {
		reader := &countingReaderAt{data: data}
		engine, err := NewEngine(Options{Fonts: []FontSpec{{Family: "cjk", Source: &FontSource{ReaderAt: reader, Size: int64(len(data)), Identity: "bench"}}}, FontSourceBytes: len(data)})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := engine.Layout(Request{Text: "中文", Families: []string{"cjk"}}); err != nil {
			b.Fatal(err)
		}
		reads += reader.calls
		engine.Close()
	}
	b.ReportMetric(float64(reads)/float64(b.N), "reader_calls/op")
}

func BenchmarkReaderAtCJKRepeatedLayout(b *testing.B) {
	data, _ := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	reader := &countingReaderAt{data: data}
	engine, err := NewEngine(Options{Fonts: []FontSpec{{Family: "cjk", Source: &FontSource{ReaderAt: reader, Size: int64(len(data)), Identity: "bench"}}}, FontSourceBytes: len(data)})
	if err != nil {
		b.Fatal(err)
	}
	defer engine.Close()
	req := Request{Text: "中文", Families: []string{"cjk"}}
	_, _ = engine.Layout(req)
	start := reader.calls
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Layout(req); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(reader.calls-start)/float64(b.N), "reader_calls/op")
}

func BenchmarkReaderAtCJKRasterize(b *testing.B) {
	data, _ := base64.StdEncoding.DecodeString(cmapTestFontBase64Fixed)
	reader := &countingReaderAt{data: data}
	engine, err := NewEngine(Options{Fonts: []FontSpec{{Family: "cjk", Source: &FontSource{ReaderAt: reader, Size: int64(len(data)), Identity: "bench"}}}, FontSourceBytes: len(data)})
	if err != nil {
		b.Fatal(err)
	}
	defer engine.Close()
	layout, _ := engine.Layout(Request{Text: "中文", Families: []string{"cjk"}})
	start := reader.calls
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.RasterizeLines(layout, 1, 1); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(reader.calls-start)/float64(b.N), "reader_calls/op")
}

func testEngine(t *testing.T, budget int) *Engine {
	t.Helper()
	engine, err := NewEngine(Options{MeasureCacheBytes: budget})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.Close)
	return engine
}

func hasInk(values []byte) bool {
	for _, value := range values {
		if value != 0 {
			return true
		}
	}
	return false
}

// cmapTest.ttf is a 2,528-byte BSD-licensed Go Authors test font. It contains
// a deliberately small U+4E2D glyph used to verify deterministic CJK fallback
// without depending on a developer machine's installed fonts.
const cmapTestFontBase64Fixed = "AAEAAAAOAIAAAwBgRkZUTXWV1gEAAAnEAAAAHEdERUYADwAeAAAJpAAAAB5PUy8yZdJ0NAAAAWgAAABg" +
	"Y21hcA2Hw6kAAAHwAAAB1mN2dCAARAURAAADyAAAAARnYXNw//8AAwAACZwAAAAIZ2x5Zg2Jda8AAAPw" +
	"AAABxGhlYWQLsXEiAAAA7AAAADZoaGVhCUIDiAAAASQAAAAkaG10eAlgAEQAAAHIAAAAKGxvY2EEDgOs" +
	"AAADzAAAACJtYXhwAFQAOQAAAUgAAAAgbmFtZQ1ZFWAAAAW0AAADdXBvc3Q8nyaPAAAJLAAAAG8AAQAA" +
	"AAEAAIY/Z/5fDzz1AAsIAAAAAADUuJNoAAAAANS4k2gAAAAAAyAFVQACAAgAAgAAAAAAAAABAAAFVQAA" +
	"ALgDIAAAAAADIABkABQAAAAAAAAAAAAAAAAABAABAAAAEAAIAAIAAAAAAAIAAAABAAEAAABAAC4AAAAA" +
	"AAQDIAGQAAUACAUzBZkANwEeBTMFmf9BA9cAZgISAAACAAUJAAAAAAAAAAAABwoAAAAAAAAAAAAAAFBm" +
	"RWQAgQAw//8GZv5mALgFVQAAAAAAAQAAAAADIAMgAAAAIAABAyAARAAAAAADIAAAAyAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAUAAAADAAAALAAAAAoAAABsAAEAAAAAANAAAwABAAAALAADAAoAAABs" +
	"AAQAQAAAAAwACAACAAQAMgBCAGEBAU4t//8AAAAwAEEAYQD/Ti3////T/8X/p/8Ksd8AAQAAAAAAAAAA" +
	"AAAAAAAMAAAAAABkAAAAAAAAAAcAAAAwAAAAMgAAAAMAAABBAAAAQgAAAAYAAABhAAAAYQAAAAgAAAD/" +
	"AAABAQAAAAkAAE4tAABOLQAAAAwAAfChAAHwoQAAAA0AAfCxAAHwsgAAAA4AAAEGAAABAAAAAAAAAAEC" +
	"AAAAAgAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAADBAUAAAAAAAAAAAAAAAAAAAYHAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAARAURAAAALAAsACwAOgBIAFYAZAByAIAAjgCcAKoAuADGANQA4gAAAAIARAAAAmQFVQAD" +
	"AAcALrEBAC88sgcEAO0ysQYF3DyyAwIA7TIAsQMALzyyBQQA7TKyBwYB/DyyAQIA7TIzESERJSERIUQC" +
	"IP4kAZj+aAVV+qtEBM0AAAABAAAAAAMgAyAAAgAAMQkBAZABkAMg/OAAAAAAAQAAAAADIAMgAAIAADEJ" +
	"AQGQAZADIPzgAAAAAAEAAAAAAyADIAACAAAxCQEBkAGQAyD84AAAAAABAAAAAAMgAyAAAgAAMQkBAZAB" +
	"kAMg/OAAAAAAAQAAAAADIAMgAAIAADEJAQGQAZADIPzgAAAAAAEAAAAAAyADIAACAAAxCQEBkAGQAyD8" +
	"4AAAAAABAAAAAAMgAyAAAgAAMQkBAZABkAMg/OAAAAAAAQAAAAADIAMgAAIAADEJAQGQAZADIPzgAAAA" +
	"AAEAAAAAAyADIAACAAAxCQEBkAGQAyD84AAAAAABAAAAAAMgAyAAAgAAMQkBAZABkAMg/OAAAAAAAQAA" +
	"AAADIAMgAAIAADEJAQGQAZADIPzgAAAAAAEAAAAAAyADIAACAAAxCQEBkAGQAyD84AAAAAABAAAAAAMg" +
	"AyAAAgAAMQkBAZABkAMg/OAAAAAAAAAOAK4AAQAAAAAAAACYATIAAQAAAAAAAQAIAd0AAQAAAAAAAgAH" +
	"AfYAAQAAAAAAAwAfAj4AAQAAAAAABAAIAnAAAQAAAAAABQAQApsAAQAAAAAABgAIAr4AAwABBAkAAAEw" +
	"AAAAAwABBAkAAQAQAcsAAwABBAkAAgAOAeYAAwABBAkAAwA+Af4AAwABBAkABAAQAl4AAwABBAkABQAg" +
	"AnkAAwABBAkABgAQAqwAQwBvAHAAeQByAGkAZwBoAHQAIAAyADAAMQA2ACAAVABoAGUAIABHAG8AIABB" +
	"AHUAdABoAG8AcgBzAC4AIABBAGwAbAAgAHIAaQBnAGgAdABzACAAcgBlAHMAZQByAHYAZQBkAC4ACgBV" +
	"AHMAZQAgAG8AZgAgAHQAaABpAHMAIABmAG8AbgB0ACAAaQBzACAAZwBvAHYAZQByAG4AZQBkACAAYgB5" +
	"ACAAYQAgAEIAUwBEAC0AcwB0AHkAbABlACAAbABpAGMAZQBuAHMAZQAgAHQAaABhAHQAIABjAGEAbgAg" +
	"AGIAZQAgAGYAbwB1AG4AZAAgAGEAdAAgAGgAdAB0AHAAcwA6AC8ALwBnAG8AbABhAG4AZwAuAG8AcgBn" +
	"AC8ATABJAEMARQBOAFMARQAuAABDb3B5cmlnaHQgMjAxNiBUaGUgR28gQXV0aG9ycy4gQWxsIHJpZ2h0" +
	"cyByZXNlcnZlZC4KVXNlIG9mIHRoaXMgZm9udCBpcyBnb3Zlcm5lZCBieSBhIEJTRC1zdHlsZSBsaWNl" +
	"bnNlIHRoYXQgY2FuIGJlIGZvdW5kIGF0IGh0dHBzOi8vZ29sYW5nLm9yZy9MSUNFTlNFLgAAYwBtAGEA" +
	"cABUAGUAcwB0AABjbWFwVGVzdAAAUgBlAGcAdQBsAGEAcgAAUmVndWxhcgAARgBvAG4AdABGAG8AcgBn" +
	"AGUAIAA6ACAAYwBtAGEAcABUAGUAcwB0ACAAOgAgADIALQAyAC0AMgAwADEANwAARm9udEZvcmdlIDog" +
	"Y21hcFRlc3QgOiAyLTItMjAxNwAAYwBtAGEAcABUAGUAcwB0AABjbWFwVGVzdAAAVgBlAHIAcwBpAG8A" +
	"bgAgADAAMAAxAC4AMAAwADAAIAAAVmVyc2lvbiAwMDEuMDAwIAAAYwBtAGEAcABUAGUAcwB0AABjbWFw" +
	"VGVzdAAAAAAAAgAA//TAAP8BAGYAAAABAAAAAAAAAAAAAAAAAAAAAAAQAAAAAQACABMAFAAVACQAJQBE" +
	"ALoBAgEDAQQBBQEGAQcHQW1hY3JvbgdhbWFjcm9uB3VuaTRFMkQGdTFGMEExBnUxRjBCMQZ1MUYwQjIA" +
	"AAAAAf//AAIAAQAAAAAAAAAOABYAAAAEAAAAAgAAAAEAAAABAAAAAAAAAAEAAAAAzD2izwAAAADUn5/f" +
	"AAAAANS4k1I="
