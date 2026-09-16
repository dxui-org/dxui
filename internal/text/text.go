// Package text contains the pure-Go font registry, deterministic fallback,
// simple LTR/CJK layout, and grayscale glyph rasterization used by dxui.
// It deliberately does not implement bidi or OpenType shaping.
package text

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	// BuiltinFamily is the small BSD-licensed Go font used after application
	// fonts and before an optional system fallback.
	BuiltinFamily            = "dxui-default"
	defaultMeasureCacheBytes = 1 << 20
	defaultFontSize          = float32(15)
	maxRasterLinePixels      = 4 << 20
	cacheEntryOverhead       = 160
	bitmapAccountedOverhead  = 128
)

// Wrap is the explicitly limited MVP wrapping policy.
type Wrap uint8

const (
	NoWrap Wrap = iota
	WordWrap
)

// Align is horizontal alignment within the laid-out content width.
type Align uint8

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
)

// FontSource is exactly one immutable memory source or bounded ReaderAt source.
// A non-nil Closer is owned by the Engine after a successful registration.
type FontSource struct {
	Data     []byte
	ReaderAt io.ReaderAt
	Size     int64
	Identity string
	Closer   io.Closer
	mu       sync.Mutex
	closed   bool
}

func (source *FontSource) close() error {
	if source == nil || source.Closer == nil {
		return nil
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.closed {
		return nil
	}
	source.closed = true
	return source.Closer.Close()
}

// FontSpec is an application-lifetime font registration.
type FontSpec struct {
	Family    string
	Weight    uint16
	Slant     uint8
	FaceIndex int
	Data      []byte // retained compatibility for in-memory callers
	Source    *FontSource
	Identity  string
}

// FontFallbackResolver supplies lazy, backend-neutral font bytes. Resolve may
// call try for ordered candidates until one covers r. Candidate failures are
// deliberately local to the resolver and must not turn missing glyphs into
// layout errors.
type FontFallbackResolver interface {
	Resolve(r rune, remainingBytes int, try func(FontSpec) (covers bool, err error))
}

// Options configure one application-owned text engine.
type Options struct {
	Fonts             []FontSpec
	DefaultFamily     string
	MeasureCacheBytes int
	FontSourceBytes   int
	FallbackResolver  FontFallbackResolver
}

// Request contains all inputs that affect logical metrics. MaxWidth is used
// only for WrapWords when MaxWidthSet is true.
type Request struct {
	Text        string
	Families    []string
	Size        float32
	LineHeight  float32
	Weight      uint16
	Slant       uint8
	Wrap        Wrap
	MaxWidth    float32
	MaxWidthSet bool
	MaxLines    int
	Align       Align
}

// Metrics are logical-unit text metrics.
type Metrics struct {
	Width, Height float32
	MinimumWidth  float32
	Lines         int
	MissingGlyphs int
}

// MeasureRequest is the minimal legacy measurer request used by backend ports.
type MeasureRequest struct{ Text string }

// Measurer lets layout tests run without SDL, a display, or a GPU.
type Measurer interface {
	Measure(MeasureRequest) (Metrics, error)
}

// Glyph is one code-point glyph in logical coordinates. Fallback has already
// selected a concrete face; no shaping claim is implied.
type Glyph struct {
	Rune         rune
	OriginalRune rune
	FontIdentity string
	X            float32
	Advance      float32
	face         int
	glyph        sfnt.GlyphIndex
}

// Line is one visual LTR line. Baseline and glyph positions use logical units.
type Line struct {
	Text            string
	Glyphs          []Glyph
	Width, Height   float32
	Ascent, Descent float32
	Baseline        float32
	MissingGlyphs   int
}

// Layout is an immutable-by-convention result shared by measurement and paint.
type Layout struct {
	Request Request
	Lines   []Line
	Metrics Metrics
	key     layoutKey
}

// Bitmap is a single line's alpha mask at a specific renderer scale. Key
// includes font identity, logical size, physical scale, and render mode.
type Bitmap struct {
	Key                         string
	Width, Height               int
	LogicalWidth, LogicalHeight float32
	OffsetX                     float32
	LineIndex                   int
	AccountedBytes              int
	Alpha                       []byte
}

// CacheStats reports exact configured accounting for the logical layout and
// fallback decision caches.
type CacheStats struct {
	BudgetBytes          int
	Bytes                int
	Entries              int
	Hits                 uint64
	Misses               uint64
	Evictions            uint64
	Faces                int
	MemoryFontBytes      int64
	FileFontLogicalBytes int64
	OpenFontSources      int
	FontCloseErrors      uint64
}

type faceRecord struct {
	family   string
	weight   uint16
	slant    uint8
	identity string
	font     *sfnt.Font
	data     []byte
}

type layoutKey struct {
	text, families          string
	size, lineHeight, width uint32
	weight                  uint16
	slant, wrap, align      uint8
	widthSet                bool
	maxLines                int
}

type layoutEntry struct {
	layout *Layout
	bytes  int
	stamp  uint64
}

type fallbackKey struct {
	chain  string
	r      rune
	weight uint16
	slant  uint8
}

type fallbackValue struct {
	face, bytes int
	glyph       sfnt.GlyphIndex
	renderRune  rune
	missing     bool
	stamp       uint64
}

// Engine owns parsed fonts and strictly byte-bounded logical caches. It has no
// SDL dependency and is intended to live for exactly one App.Run lifecycle.
type Engine struct {
	faces         []faceRecord
	families      map[string][]int
	defaultFamily string
	appFamilies   []string
	defaultChain  []int
	defaultKey    string
	systemFaces   []int
	resolver      FontFallbackResolver
	fontBudget    int
	fontBytes     int
	sources       map[*FontSource]struct{}
	buffer        sfnt.Buffer
	closeErr      error
	closeErrors   uint64

	budget, layoutBudget, fallbackBudget int
	layoutBytes, fallbackBytes           int
	stamp                                uint64
	layouts                              map[layoutKey]layoutEntry
	fallbacks                            map[fallbackKey]fallbackValue
	stats                                CacheStats
}

// NewEngine parses application fonts and installs the small built-in Latin
// fallback. Collection face selection is explicit and bounds checked.
func NewEngine(options Options) (*Engine, error) {
	budget := options.MeasureCacheBytes
	if budget == 0 {
		budget = defaultMeasureCacheBytes
	}
	if budget < 0 {
		budget = 0
	}
	engine := &Engine{
		families: make(map[string][]int), defaultFamily: options.DefaultFamily,
		resolver: options.FallbackResolver, fontBudget: options.FontSourceBytes,
		budget: budget, layoutBudget: budget * 3 / 4, fallbackBudget: budget - budget*3/4,
		layouts: make(map[layoutKey]layoutEntry), fallbacks: make(map[fallbackKey]fallbackValue), sources: make(map[*FontSource]struct{}),
	}
	ok := false
	defer func() {
		if !ok {
			for _, spec := range options.Fonts {
				_ = spec.Source.close()
			}
			engine.Close()
		}
	}()
	for index, spec := range options.Fonts {
		if spec.Source == nil && len(spec.Data) != 0 {
			spec.Source = &FontSource{Data: spec.Data, Size: int64(len(spec.Data))}
		}
		if err := engine.addFont(spec); err != nil {
			return nil, fmt.Errorf("text: font %d: %w", index, err)
		}
		engine.fontBytes += int(spec.Source.Size)
		if !seenString(engine.appFamilies, spec.Family) {
			engine.appFamilies = append(engine.appFamilies, spec.Family)
		}
	}
	if err := engine.addFont(FontSpec{Family: BuiltinFamily, Weight: 400, Source: &FontSource{Data: goregular.TTF, Size: int64(len(goregular.TTF)), Identity: "builtin:go-regular"}, Identity: "builtin:go-regular"}); err != nil {
		return nil, fmt.Errorf("text: built-in font: %w", err)
	}
	if engine.defaultFamily == "" {
		engine.defaultFamily = BuiltinFamily
	}
	if _, ok := engine.families[engine.defaultFamily]; !ok {
		return nil, fmt.Errorf("text: default family %q is not registered", engine.defaultFamily)
	}
	var err error
	engine.defaultChain, engine.defaultKey, err = engine.fontChain(nil, 400, 0)
	if err != nil {
		return nil, err
	}
	engine.stats.BudgetBytes = budget
	ok = true
	return engine, nil
}

func (engine *Engine) addFont(spec FontSpec) error {
	if spec.Source == nil && len(spec.Data) != 0 {
		spec.Source = &FontSource{Data: spec.Data, Size: int64(len(spec.Data))}
	}
	if spec.Family == "" || strings.IndexByte(spec.Family, 0) >= 0 {
		return fmt.Errorf("invalid family %q", spec.Family)
	}
	if spec.Source == nil || (len(spec.Source.Data) == 0) == (spec.Source.ReaderAt == nil) || spec.Source.Size <= 0 {
		return fmt.Errorf("family %q has invalid font source", spec.Family)
	}
	var collection *sfnt.Collection
	var err error
	if spec.Source.ReaderAt != nil {
		collection, err = sfnt.ParseCollectionReaderAt(spec.Source.ReaderAt)
	} else {
		collection, err = sfnt.ParseCollection(spec.Source.Data)
	}
	if err != nil {
		return fmt.Errorf("parse family %q: %w", spec.Family, err)
	}
	if spec.FaceIndex < 0 || spec.FaceIndex >= collection.NumFonts() {
		return fmt.Errorf("family %q face index %d outside collection of %d", spec.Family, spec.FaceIndex, collection.NumFonts())
	}
	parsed, err := collection.Font(spec.FaceIndex)
	if err != nil {
		return fmt.Errorf("parse family %q face %d: %w", spec.Family, spec.FaceIndex, err)
	}
	weight := spec.Weight
	if weight == 0 {
		weight = 400
	}
	identity := spec.Identity
	if identity == "" {
		sum := sha256.Sum256(spec.Source.Data)
		identity = fmt.Sprintf("sha256:%x:%d", sum[:12], spec.FaceIndex)
	}
	face := len(engine.faces)
	engine.faces = append(engine.faces, faceRecord{
		family: spec.Family, weight: weight, slant: spec.Slant,
		identity: identity, font: parsed, data: spec.Source.Data,
	})
	if _, exists := engine.sources[spec.Source]; !exists && spec.Source.Identity != "builtin:go-regular" {
		engine.sources[spec.Source] = struct{}{}
		if spec.Source.ReaderAt != nil {
			engine.stats.FileFontLogicalBytes += spec.Source.Size
		} else {
			engine.stats.MemoryFontBytes += spec.Source.Size
		}
	}
	engine.families[spec.Family] = append(engine.families[spec.Family], face)
	return nil
}

// Close drops parsed-font references and all cached values. Renderer textures
// are owned and destroyed by the renderer before this call.
func (engine *Engine) Close() {
	if engine == nil {
		return
	}
	for source := range engine.sources {
		if err := source.close(); err != nil {
			engine.closeErr = errors.Join(engine.closeErr, err)
			engine.closeErrors++
		}
	}
	engine.sources = nil
	engine.stats.MemoryFontBytes = 0
	engine.stats.FileFontLogicalBytes = 0
	engine.faces = nil
	engine.families = nil
	engine.appFamilies = nil
	engine.systemFaces = nil
	engine.resolver = nil
	engine.fontBytes = 0
	engine.layouts = nil
	engine.fallbacks = nil
	engine.layoutBytes = 0
	engine.fallbackBytes = 0
	engine.refreshStats()
}

// Stats returns current cache accounting and cumulative lookup counters.
func (engine *Engine) Stats() CacheStats {
	engine.refreshStats()
	return engine.stats
}

func (engine *Engine) refreshStats() {
	engine.stats.Bytes = engine.layoutBytes + engine.fallbackBytes
	engine.stats.Entries = len(engine.layouts) + len(engine.fallbacks)
	engine.stats.Faces = len(engine.faces)
	engine.stats.OpenFontSources = len(engine.sources)
	engine.stats.FontCloseErrors = engine.closeErrors
}

// Measure implements the small backend-neutral measurer port. It uses default
// style values and exists primarily for simple layout fakes.
func (engine *Engine) Measure(request MeasureRequest) (Metrics, error) {
	layout, err := engine.Layout(Request{Text: request.Text})
	if err != nil {
		return Metrics{}, err
	}
	return layout.Metrics, nil
}

// Layout returns logical metrics and glyph placement from one shared path used
// by both intrinsic measurement and paint.
func (engine *Engine) Layout(request Request) (*Layout, error) {
	request.Text = strings.ToValidUTF8(request.Text, "\uFFFD")
	if request.Size == 0 {
		request.Size = defaultFontSize
	}
	if !finitePositive(request.Size) {
		return nil, fmt.Errorf("text: font size must be finite and positive")
	}
	if request.LineHeight < 0 || !finite(request.LineHeight) {
		return nil, fmt.Errorf("text: line height must be finite and non-negative")
	}
	if request.MaxWidthSet && (!finite(request.MaxWidth) || request.MaxWidth < 0) {
		return nil, fmt.Errorf("text: maximum width must be finite and non-negative")
	}
	if request.MaxLines < 0 || request.Wrap > WordWrap || request.Align > AlignEnd {
		return nil, fmt.Errorf("text: invalid layout enum or maximum lines")
	}
	if request.Weight == 0 {
		request.Weight = 400
	}
	chain, chainKey := engine.defaultChain, engine.defaultKey
	if len(request.Families) != 0 || request.Weight != 400 || request.Slant != 0 {
		var err error
		chain, chainKey, err = engine.fontChain(request.Families, request.Weight, request.Slant)
		if err != nil {
			return nil, err
		}
	}
	key := makeLayoutKey(request, chainKey)
	if entry, ok := engine.layouts[key]; ok {
		engine.stamp++
		entry.stamp = engine.stamp
		engine.layouts[key] = entry
		engine.stats.Hits++
		return entry.layout, nil
	}
	engine.stats.Misses++
	layout, err := engine.buildLayout(request, chain, chainKey, key)
	if err != nil {
		return nil, err
	}
	engine.putLayout(key, layout)
	return layout, nil
}

func makeLayoutKey(request Request, familyKey string) layoutKey {
	return layoutKey{
		text: request.Text, families: familyKey,
		size: math.Float32bits(request.Size), lineHeight: math.Float32bits(request.LineHeight),
		width: math.Float32bits(request.MaxWidth), widthSet: request.MaxWidthSet,
		weight: request.Weight, slant: request.Slant, wrap: uint8(request.Wrap),
		align: uint8(request.Align), maxLines: request.MaxLines,
	}
}

func (engine *Engine) fontChain(families []string, weight uint16, slant uint8) ([]int, string, error) {
	names := make([]string, 0, len(families)+len(engine.appFamilies)+2)
	seen := make(map[string]struct{}, len(families)+len(engine.appFamilies)+2)
	for _, family := range families {
		if family == "" || strings.IndexByte(family, 0) >= 0 {
			return nil, "", fmt.Errorf("text: invalid font family %q", family)
		}
		if _, ok := seen[family]; !ok {
			names, seen[family] = append(names, family), struct{}{}
		}
	}
	ordered := make([]string, 0, len(engine.appFamilies)+2)
	if engine.defaultFamily != BuiltinFamily {
		ordered = append(ordered, engine.defaultFamily)
	}
	ordered = append(ordered, engine.appFamilies...)
	ordered = append(ordered, BuiltinFamily)
	for _, family := range ordered {
		if _, ok := seen[family]; !ok {
			names, seen[family] = append(names, family), struct{}{}
		}
	}
	chain := make([]int, 0, len(names))
	var key strings.Builder
	for _, family := range names {
		candidates, ok := engine.families[family]
		if !ok {
			return nil, "", fmt.Errorf("text: font family %q is not registered", family)
		}
		face := bestFace(engine.faces, candidates, weight, slant)
		chain = append(chain, face)
		key.WriteString(engine.faces[face].identity)
		key.WriteByte(0)
	}
	return chain, key.String(), nil
}

func bestFace(faces []faceRecord, candidates []int, weight uint16, slant uint8) int {
	best, bestScore := candidates[0], math.MaxInt
	for _, candidate := range candidates {
		face := faces[candidate]
		score := absInt(int(face.weight) - int(weight))
		if face.slant != slant {
			score += 1000
		}
		if score < bestScore {
			best, bestScore = candidate, score
		}
	}
	return best
}

func (engine *Engine) buildLayout(request Request, chain []int, chainKey string, key layoutKey) (*Layout, error) {
	layout := &Layout{Request: request, key: key}
	if request.Text == "" {
		return layout, nil
	}
	paragraphs := strings.Split(request.Text, "\n")
	for _, paragraph := range paragraphs {
		if request.MaxLines > 0 && len(layout.Lines) >= request.MaxLines {
			break
		}
		lines, err := engine.layoutParagraph(paragraph, request, chain, chainKey)
		if err != nil {
			return nil, err
		}
		for _, line := range lines {
			if request.MaxLines > 0 && len(layout.Lines) >= request.MaxLines {
				break
			}
			layout.Lines = append(layout.Lines, line)
		}
	}
	for _, line := range layout.Lines {
		layout.Metrics.Width = max(layout.Metrics.Width, line.Width)
		layout.Metrics.Height += line.Height
		layout.Metrics.MissingGlyphs += line.MissingGlyphs
	}
	layout.Metrics.Lines = len(layout.Lines)
	if request.Wrap == NoWrap {
		// A no-wrap paragraph's minimum and preferred width are identical.
		// Re-shaping every line here doubled work on cache misses.
		layout.Metrics.MinimumWidth = layout.Metrics.Width
	} else {
		layout.Metrics.MinimumWidth = engine.minimumWidth(request, chain, chainKey)
	}
	return layout, nil
}

func (engine *Engine) layoutParagraph(paragraph string, request Request, chain []int, chainKey string) ([]Line, error) {
	if request.Wrap == NoWrap {
		line, err := engine.shapeLine(paragraph, request, chain, chainKey)
		return []Line{line}, err
	}
	words := strings.Fields(paragraph)
	if len(words) == 0 {
		line, err := engine.shapeLine("", request, chain, chainKey)
		return []Line{line}, err
	}
	if !request.MaxWidthSet {
		line, err := engine.shapeLine(strings.Join(words, " "), request, chain, chainKey)
		return []Line{line}, err
	}
	result := make([]Line, 0, 1)
	start := 0
	for start < len(words) {
		end := start + 1
		line, err := engine.shapeLine(words[start], request, chain, chainKey)
		if err != nil {
			return nil, err
		}
		for end < len(words) {
			candidate, candidateErr := engine.shapeLine(strings.Join(words[start:end+1], " "), request, chain, chainKey)
			if candidateErr != nil {
				return nil, candidateErr
			}
			if candidate.Width > request.MaxWidth {
				break
			}
			line, end = candidate, end+1
		}
		result = append(result, line)
		start = end
	}
	return result, nil
}

func (engine *Engine) shapeLine(value string, request Request, chain []int, chainKey string) (Line, error) {
	line := Line{Text: value}
	previousFace := -1
	var previousGlyph sfnt.GlyphIndex
	ppem := fixed.Int26_6(math.Round(float64(request.Size * 64)))
	for _, original := range value {
		selected, err := engine.resolveRune(chain, chainKey, original, request.Weight, request.Slant)
		if err != nil {
			return Line{}, err
		}
		face := &engine.faces[selected.face]
		advance, err := face.font.GlyphAdvance(&engine.buffer, selected.glyph, ppem, font.HintingNone)
		if err != nil {
			return Line{}, fmt.Errorf("text: glyph advance %U in %s: %w", original, face.identity, err)
		}
		if previousFace == selected.face {
			kern, kernErr := face.font.Kern(&engine.buffer, previousGlyph, selected.glyph, ppem, font.HintingNone)
			if kernErr == nil {
				line.Width += fixedToFloat(kern)
			}
		}
		logicalAdvance := fixedToFloat(advance)
		line.Glyphs = append(line.Glyphs, Glyph{
			Rune: selected.renderRune, OriginalRune: original, FontIdentity: face.identity,
			X: line.Width, Advance: logicalAdvance, face: selected.face, glyph: selected.glyph,
		})
		line.Width += logicalAdvance
		metrics, metricsErr := face.font.Metrics(&engine.buffer, ppem, font.HintingNone)
		if metricsErr != nil {
			return Line{}, fmt.Errorf("text: metrics for %s: %w", face.identity, metricsErr)
		}
		line.Ascent = max(line.Ascent, fixedToFloat(metrics.Ascent))
		line.Descent = max(line.Descent, fixedToFloat(metrics.Descent))
		if selected.missing {
			line.MissingGlyphs++
		}
		previousFace, previousGlyph = selected.face, selected.glyph
	}
	if len(line.Glyphs) == 0 {
		face := &engine.faces[chain[0]]
		metrics, err := face.font.Metrics(&engine.buffer, ppem, font.HintingNone)
		if err != nil {
			return Line{}, fmt.Errorf("text: empty-line metrics: %w", err)
		}
		line.Ascent, line.Descent = fixedToFloat(metrics.Ascent), fixedToFloat(metrics.Descent)
	}
	line.Height = request.LineHeight
	if line.Height == 0 {
		line.Height = line.Ascent + line.Descent
	}
	if line.Height == 0 {
		line.Height = request.Size
	}
	line.Baseline = (line.Height-line.Ascent-line.Descent)/2 + line.Ascent
	return line, nil
}

func (engine *Engine) minimumWidth(request Request, chain []int, chainKey string) float32 {
	if request.Wrap != WordWrap {
		var width float32
		for _, paragraph := range strings.Split(request.Text, "\n") {
			line, err := engine.shapeLine(paragraph, request, chain, chainKey)
			if err == nil {
				width = max(width, line.Width)
			}
		}
		return width
	}
	var width float32
	for _, word := range strings.Fields(request.Text) {
		line, err := engine.shapeLine(word, request, chain, chainKey)
		if err == nil {
			width = max(width, line.Width)
		}
	}
	return width
}

func (engine *Engine) resolveRune(chain []int, chainKey string, r rune, weight uint16, slant uint8) (fallbackValue, error) {
	key := fallbackKey{chain: chainKey, r: r, weight: weight, slant: slant}
	if value, ok := engine.fallbacks[key]; ok {
		engine.stamp++
		value.stamp = engine.stamp
		engine.fallbacks[key] = value
		return value, nil
	}
	lookupFaces := func(faces []int, candidate rune, tolerateReadError bool) (fallbackValue, bool, error) {
		for _, face := range faces {
			glyph, err := engine.faces[face].font.GlyphIndex(&engine.buffer, candidate)
			if err != nil {
				if tolerateReadError {
					continue
				}
				return fallbackValue{}, false, err
			}
			if glyph != 0 {
				return fallbackValue{face: face, glyph: glyph, renderRune: candidate}, true, nil
			}
		}
		return fallbackValue{}, false, nil
	}
	value, ok, err := lookupFaces(chain, r, false)
	if err != nil {
		return fallbackValue{}, fmt.Errorf("text: glyph lookup %U: %w", r, err)
	}
	if !ok {
		value, ok, err = lookupFaces(engine.systemFaces, r, true)
		if err != nil {
			return fallbackValue{}, fmt.Errorf("text: system glyph lookup %U: %w", r, err)
		}
	}
	if !ok && engine.resolver != nil && isCJKRune(r) {
		engine.resolver.Resolve(r, max(0, engine.fontBudget-engine.fontBytes), func(spec FontSpec) (bool, error) {
			if spec.Source == nil && len(spec.Data) != 0 {
				spec.Source = &FontSource{Data: spec.Data, Size: int64(len(spec.Data))}
			}
			if spec.Source == nil || spec.Source.Size <= 0 || spec.Source.Size > int64(engine.fontBudget-engine.fontBytes) {
				_ = spec.Source.close()
				return false, fmt.Errorf("font source exceeds remaining budget")
			}
			face := len(engine.faces)
			if addErr := engine.addFont(spec); addErr != nil {
				_ = spec.Source.close()
				return false, addErr
			}
			glyph, glyphErr := engine.faces[face].font.GlyphIndex(&engine.buffer, r)
			if glyphErr != nil {
				registered := engine.families[spec.Family]
				registered = registered[:len(registered)-1]
				if len(registered) == 0 {
					delete(engine.families, spec.Family)
				} else {
					engine.families[spec.Family] = registered
				}
				engine.faces = engine.faces[:face]
				delete(engine.sources, spec.Source)
				if spec.Source.ReaderAt != nil {
					engine.stats.FileFontLogicalBytes -= spec.Source.Size
				} else {
					engine.stats.MemoryFontBytes -= spec.Source.Size
				}
				_ = spec.Source.close()
				return false, glyphErr
			}
			engine.fontBytes += int(spec.Source.Size)
			engine.systemFaces = append(engine.systemFaces, face)
			engine.clearLogicalCaches()
			return glyph != 0, nil
		})
		value, ok, err = lookupFaces(engine.systemFaces, r, true)
		if err != nil {
			return fallbackValue{}, fmt.Errorf("text: resolved system glyph lookup %U: %w", r, err)
		}
	}
	if !ok {
		value, ok, err = lookupFaces(chain, utf8.RuneError, false)
		if !ok && err == nil {
			value, ok, err = lookupFaces(engine.systemFaces, utf8.RuneError, true)
		}
		if err != nil {
			return fallbackValue{}, fmt.Errorf("text: replacement glyph lookup: %w", err)
		}
		value.missing = true
		if !ok {
			face := chain[len(chain)-1]
			value = fallbackValue{face: face, glyph: 0, renderRune: utf8.RuneError, missing: true}
		}
	}
	value.bytes = cacheEntryOverhead + len(chainKey)
	engine.putFallback(key, value)
	return value, nil
}

func (engine *Engine) clearLogicalCaches() {
	clear(engine.layouts)
	clear(engine.fallbacks)
	engine.layoutBytes = 0
	engine.fallbackBytes = 0
}

func seenString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func isCJKRune(r rune) bool {
	return r >= 0x3400 && r <= 0x9fff || r >= 0x20000 && r <= 0x323af ||
		r >= 0x3040 && r <= 0x30ff || r >= 0xac00 && r <= 0xd7af
}

func (engine *Engine) putLayout(key layoutKey, layout *Layout) {
	bytes := cacheEntryOverhead + len(key.text) + len(key.families) + len(layout.Lines)*64
	for _, line := range layout.Lines {
		bytes += len(line.Text) + len(line.Glyphs)*64
	}
	if engine.layoutBudget == 0 || bytes > engine.layoutBudget {
		return
	}
	for engine.layoutBytes+bytes > engine.layoutBudget {
		var oldestKey layoutKey
		oldestStamp := uint64(math.MaxUint64)
		found := false
		for candidate, entry := range engine.layouts {
			if entry.stamp < oldestStamp {
				oldestKey, oldestStamp, found = candidate, entry.stamp, true
			}
		}
		if !found {
			break
		}
		engine.layoutBytes -= engine.layouts[oldestKey].bytes
		delete(engine.layouts, oldestKey)
		engine.stats.Evictions++
	}
	engine.stamp++
	engine.layouts[key] = layoutEntry{layout: layout, bytes: bytes, stamp: engine.stamp}
	engine.layoutBytes += bytes
}

func (engine *Engine) putFallback(key fallbackKey, value fallbackValue) {
	if engine.fallbackBudget == 0 || value.bytes > engine.fallbackBudget {
		return
	}
	for engine.fallbackBytes+value.bytes > engine.fallbackBudget {
		var oldestKey fallbackKey
		oldestStamp := uint64(math.MaxUint64)
		found := false
		for candidate, entry := range engine.fallbacks {
			if entry.stamp < oldestStamp {
				oldestKey, oldestStamp, found = candidate, entry.stamp, true
			}
		}
		if !found {
			break
		}
		engine.fallbackBytes -= engine.fallbacks[oldestKey].bytes
		delete(engine.fallbacks, oldestKey)
		engine.stats.Evictions++
	}
	engine.stamp++
	value.stamp = engine.stamp
	engine.fallbacks[key] = value
	engine.fallbackBytes += value.bytes
}

// RasterizeLines converts placed glyphs into one grayscale alpha bitmap per
// non-empty line. Logical placement is unchanged across scales; only the
// physical masks and cache identities vary.
func (engine *Engine) RasterizeLines(layout *Layout, scaleX, scaleY float32) ([]Bitmap, error) {
	return engine.rasterizeLines(layout, scaleX, scaleY, -1, nil)
}

// RasterizeLinesWithin is RasterizeLines with a strict accounting limit for
// all returned alpha masks. The full result is preflighted before allocation.
func (engine *Engine) RasterizeLinesWithin(layout *Layout, scaleX, scaleY float32, byteBudget int) ([]Bitmap, error) {
	return engine.RasterizeLinesWithinReuse(layout, scaleX, scaleY, byteBudget, nil)
}

// RasterizeLinesWithinReuse reuses immutable masks retained by the previous
// display when their complete scale/style/content key and dimensions match.
func (engine *Engine) RasterizeLinesWithinReuse(layout *Layout, scaleX, scaleY float32, byteBudget int, reuse func(string, int, int) ([]byte, bool)) ([]Bitmap, error) {
	if byteBudget < 0 {
		byteBudget = 0
	}
	return engine.rasterizeLines(layout, scaleX, scaleY, int64(byteBudget), reuse)
}

type rasterSpec struct {
	lineIndex      int
	width, height  int
	padding        int
	accountedBytes int64
}

func (engine *Engine) rasterizeLines(layout *Layout, scaleX, scaleY float32, byteBudget int64, reuse func(string, int, int) ([]byte, bool)) ([]Bitmap, error) {
	if layout == nil {
		return nil, fmt.Errorf("text: nil layout")
	}
	if !finitePositive(scaleX) || !finitePositive(scaleY) {
		return nil, fmt.Errorf("text: renderer scale must be finite and positive")
	}
	specs := make([]rasterSpec, 0, len(layout.Lines))
	var totalBytes int64
	for lineIndex, line := range layout.Lines {
		if len(line.Glyphs) == 0 || line.Width <= 0 || line.Height <= 0 {
			continue
		}
		padding := int(math.Ceil(float64(layout.Request.Size*scaleY))) + 2
		width := int(math.Ceil(float64(line.Width*scaleX))) + padding*2
		height := int(math.Ceil(float64(line.Height * scaleY)))
		if width <= 0 || height <= 0 || int64(width)*int64(height) > maxRasterLinePixels {
			return nil, fmt.Errorf("text: raster line %d is too large: %dx%d", lineIndex, width, height)
		}
		accounted := int64(width)*int64(height) + bitmapAccountedOverhead
		if totalBytes > math.MaxInt64-accounted {
			return nil, fmt.Errorf("text: raster source accounting overflow")
		}
		totalBytes += accounted
		specs = append(specs, rasterSpec{lineIndex: lineIndex, width: width, height: height, padding: padding, accountedBytes: accounted})
	}
	if byteBudget >= 0 && totalBytes > byteBudget {
		return nil, fmt.Errorf("text: retained raster sources require %d bytes, budget is %d", totalBytes, byteBudget)
	}

	result := make([]Bitmap, 0, len(specs))
	for _, spec := range specs {
		lineIndex, line := spec.lineIndex, layout.Lines[spec.lineIndex]
		width, height, padding := spec.width, spec.height, spec.padding
		key := engine.bitmapKey(layout, lineIndex, line, scaleX, scaleY)
		if reuse != nil {
			if alpha, ok := reuse(key, width, height); ok && len(alpha) == width*height {
				result = append(result, Bitmap{
					Key: key, Width: width, Height: height,
					LogicalWidth: float32(width) / scaleX, LogicalHeight: float32(height) / scaleY,
					OffsetX: -float32(padding) / scaleX, LineIndex: lineIndex,
					AccountedBytes: int(spec.accountedBytes), Alpha: alpha,
				})
				continue
			}
		}
		mask := image.NewAlpha(image.Rect(0, 0, width, height))
		faces := make(map[int]font.Face)
		for _, glyph := range line.Glyphs {
			face := faces[glyph.face]
			if face == nil {
				created, err := opentype.NewFace(engine.faces[glyph.face].font, &opentype.FaceOptions{
					Size: float64(layout.Request.Size * scaleY), DPI: 72, Hinting: font.HintingNone,
				})
				if err != nil {
					for _, opened := range faces {
						_ = opened.Close()
					}
					return nil, fmt.Errorf("text: raster face %s: %w", engine.faces[glyph.face].identity, err)
				}
				face = created
				faces[glyph.face] = face
			}
			dot := fixed.Point26_6{
				X: fixed.Int26_6(math.Round(float64((glyph.X*scaleX + float32(padding)) * 64))),
				Y: fixed.Int26_6(math.Round(float64(line.Baseline * scaleY * 64))),
			}
			destination, glyphMask, maskPoint, _, _ := face.Glyph(dot, glyph.Rune)
			if glyphMask != nil && !destination.Empty() {
				draw.Draw(mask, destination, glyphMask, maskPoint, draw.Over)
			}
		}
		for _, face := range faces {
			_ = face.Close()
		}
		result = append(result, Bitmap{
			Key:   key,
			Width: width, Height: height,
			LogicalWidth: float32(width) / scaleX, LogicalHeight: float32(height) / scaleY,
			OffsetX: -float32(padding) / scaleX, LineIndex: lineIndex, AccountedBytes: int(spec.accountedBytes), Alpha: mask.Pix,
		})
	}
	return result, nil
}

func (engine *Engine) bitmapKey(layout *Layout, lineIndex int, line Line, scaleX, scaleY float32) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(layout.key.families))
	_, _ = hash.Write([]byte(line.Text))
	for _, glyph := range line.Glyphs {
		_, _ = hash.Write([]byte(glyph.FontIdentity))
		_, _ = hash.Write([]byte{0})
	}
	var numbers [32]byte
	binary.LittleEndian.PutUint32(numbers[0:4], layout.key.size)
	binary.LittleEndian.PutUint32(numbers[4:8], layout.key.lineHeight)
	binary.LittleEndian.PutUint32(numbers[8:12], math.Float32bits(scaleX))
	binary.LittleEndian.PutUint32(numbers[12:16], math.Float32bits(scaleY))
	binary.LittleEndian.PutUint32(numbers[16:20], uint32(lineIndex))
	binary.LittleEndian.PutUint16(numbers[20:22], layout.key.weight)
	numbers[22], numbers[23] = layout.key.slant, 1 // grayscale render mode v1
	_, _ = hash.Write(numbers[:24])
	return fmt.Sprintf("text-v1:%x", hash.Sum(nil))
}

// WrapWords performs the explicitly limited MVP whitespace wrapping using an
// injected measurer. It is not Unicode line-breaking conformance.
func WrapWords(value string, maxWidth float32, measurer Measurer) ([]string, error) {
	if value == "" {
		return []string{""}, nil
	}
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			candidate := line + " " + word
			metrics, err := measurer.Measure(MeasureRequest{Text: candidate})
			if err != nil {
				return nil, err
			}
			if maxWidth >= 0 && metrics.Width > maxWidth {
				lines, line = append(lines, line), word
			} else {
				line = candidate
			}
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func fixedToFloat(value fixed.Int26_6) float32 { return float32(value) / 64 }
func finite(value float32) bool                { return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0) }
func finitePositive(value float32) bool        { return value > 0 && finite(value) }
func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// DeterministicFamilies returns registered families for diagnostics/tests.
func (engine *Engine) DeterministicFamilies() []string {
	result := make([]string, 0, len(engine.families))
	for family := range engine.families {
		result = append(result, family)
	}
	sort.Strings(result)
	return result
}
