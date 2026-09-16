// Package image owns guarded pure-Go raster decoding and its byte-bounded CPU
// cache. It deliberately has no renderer or public-package dependencies.
package image

import (
	"bytes"
	"fmt"
	stdimage "image"
	"image/color"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
)

const (
	DefaultMaxPixels = int64(16 * 1024 * 1024)
	maxEncodedBytes  = int64(64 * 1024 * 1024)
	entryOverhead    = 128
)

// Key identifies one decoded source/options result.
type Key struct {
	SourceID uint64
	Pixels   int64
}

// Source is exactly one immutable encoded/file/Go image input.
type Source struct {
	ID      uint64
	Encoded []byte
	File    string
	Image   stdimage.Image
}

// Bitmap is tightly packed, non-premultiplied RGBA source data.
type Bitmap struct {
	Key           Key
	Width, Height int
	Pixels        []byte
}

// Stats reports configured CPU-cache accounting.
type Stats struct {
	BudgetBytes   int
	Bytes         int
	PixelBytes    int
	Entries       int
	Evictions     uint64
	ActiveEntries int
	ActiveBytes   int
	DecodeCount   uint64
	DecodedBytes  uint64
}

type entry struct {
	bitmap *Bitmap
	err    error
	bytes  int
	stamp  uint64
}

// Cache decodes and retains zero-copy immutable bitmap results. Zero-reference
// entries stay under the byte budget; active display entries remain resident
// until SetActive releases them, even when one entry exceeds that budget.
type Cache struct {
	budget    int
	bytes     int
	stamp     uint64
	evictions uint64
	decodes   uint64
	decoded   uint64
	entries   map[Key]entry
	active    map[Key]struct{}
}

// NewCache creates a decoded-pixel cache. Zero selects 2 MiB; negative means
// no retained decoded pixels.
func NewCache(budget int) *Cache {
	if budget == 0 {
		budget = 2 << 20
	}
	if budget < 0 {
		budget = 0
	}
	return &Cache{budget: budget, entries: make(map[Key]entry), active: make(map[Key]struct{})}
}

// SourceKey normalizes decode options into the cache identity.
func SourceKey(sourceID uint64, maxPixels int64) Key {
	if maxPixels == 0 {
		maxPixels = DefaultMaxPixels
	}
	return Key{SourceID: sourceID, Pixels: maxPixels}
}

// Decode returns a shared immutable bitmap on cache hits.
func (cache *Cache) Decode(source Source, maxPixels int64) (*Bitmap, error) {
	if maxPixels == 0 {
		maxPixels = DefaultMaxPixels
	}
	key := SourceKey(source.ID, maxPixels)
	if value, ok := cache.entries[key]; ok {
		cache.stamp++
		value.stamp = cache.stamp
		cache.entries[key] = value
		return value.bitmap, value.err
	}
	bitmap, err := decode(source, key, maxPixels)
	cache.decodes++
	if err != nil {
		cache.put(key, entry{err: err, bytes: entryOverhead})
		return nil, err
	}
	accounted := entryOverhead + len(bitmap.Pixels)
	cache.decoded += uint64(len(bitmap.Pixels))
	cache.put(key, entry{bitmap: bitmap, bytes: accounted})
	return bitmap, nil
}

func (cache *Cache) put(key Key, value entry) {
	_, active := cache.active[key]
	if value.bytes > cache.budget && !active {
		return
	}
	for cache.bytes+value.bytes > cache.budget {
		if !cache.evictOldest() {
			if active {
				break
			}
			return
		}
	}
	if cache.bytes+value.bytes > cache.budget && !active {
		return
	}
	cache.stamp++
	value.stamp = cache.stamp
	cache.entries[key] = value
	cache.bytes += value.bytes
}

// Stats returns current cache accounting.
func (cache *Cache) Stats() Stats {
	stats := Stats{BudgetBytes: cache.budget, Bytes: cache.bytes, Entries: len(cache.entries), Evictions: cache.evictions, ActiveEntries: len(cache.active), DecodeCount: cache.decodes, DecodedBytes: cache.decoded}
	for key, value := range cache.entries {
		if value.bitmap != nil {
			stats.PixelBytes += len(value.bitmap.Pixels)
			if _, active := cache.active[key]; active {
				stats.ActiveBytes += len(value.bitmap.Pixels)
			}
		}
	}
	return stats
}

// Clear releases every decoded cache reference.
func (cache *Cache) Clear() {
	cache.entries = make(map[Key]entry)
	cache.active = make(map[Key]struct{})
	cache.bytes = 0
}

// ActiveKeys returns a copy suitable for transactional rollback.
func (cache *Cache) ActiveKeys() []Key {
	result := make([]Key, 0, len(cache.active))
	for key := range cache.active {
		result = append(result, key)
	}
	return result
}

// SetActive pins entries referenced by the prepared/committed display. An
// active oversized entry is retained as working-set memory, not cache slack,
// and is evicted as soon as it is no longer active if it cannot fit the budget.
func (cache *Cache) SetActive(keys []Key) {
	next := make(map[Key]struct{}, len(keys))
	for _, key := range keys {
		next[key] = struct{}{}
	}
	cache.active = next
	for cache.bytes > cache.budget {
		if !cache.evictOldest() {
			break
		}
	}
}

func (cache *Cache) evictOldest() bool {
	var oldestKey Key
	var oldest entry
	found := false
	for key, value := range cache.entries {
		if _, pinned := cache.active[key]; !pinned && (!found || value.stamp < oldest.stamp) {
			oldestKey, oldest, found = key, value, true
		}
	}
	if !found {
		return false
	}
	delete(cache.entries, oldestKey)
	cache.bytes -= oldest.bytes
	cache.evictions++
	return true
}

func decode(source Source, key Key, maxPixels int64) (bitmap *Bitmap, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			bitmap = nil
			err = fmt.Errorf("image: source panic: %v", recovered)
		}
	}()
	if source.ID == 0 || maxPixels <= 0 {
		return nil, fmt.Errorf("image: invalid source or maximum pixel count")
	}
	if source.Image != nil {
		bounds := source.Image.Bounds()
		width, height, boundsErr := boundsDimensions(bounds)
		if boundsErr != nil {
			return nil, boundsErr
		}
		if err := validateDimensions(width, height, maxPixels); err != nil {
			return nil, err
		}
		return convert(key, source.Image, width, height), nil
	}
	data := source.Encoded
	if source.File != "" {
		file, err := os.Open(source.File)
		if err != nil {
			return nil, fmt.Errorf("image: open %q: %w", source.File, err)
		}
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, maxEncodedBytes+1))
		if err != nil {
			return nil, fmt.Errorf("image: read %q: %w", source.File, err)
		}
	}
	if len(data) == 0 || int64(len(data)) > maxEncodedBytes {
		return nil, fmt.Errorf("image: encoded source is empty or exceeds %d bytes", maxEncodedBytes)
	}
	config, format, err := stdimage.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image: decode header: %w", err)
	}
	if format != "png" && format != "jpeg" && format != "gif" {
		return nil, fmt.Errorf("image: unsupported format %q", format)
	}
	if err := validateDimensions(config.Width, config.Height, maxPixels); err != nil {
		return nil, err
	}
	var decoded stdimage.Image
	if format == "gif" {
		frames, scanErr := gifFrameCount(data)
		if scanErr != nil {
			return nil, fmt.Errorf("image: inspect gif: %w", scanErr)
		}
		if frames != 1 {
			return nil, fmt.Errorf("image: animated GIF is not supported")
		}
		decoded, err = gif.Decode(bytes.NewReader(data))
	} else if format == "png" {
		decoded, err = png.Decode(bytes.NewReader(data))
	} else {
		decoded, _, err = stdimage.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil, fmt.Errorf("image: decode pixels: %w", err)
	}
	return convert(key, decoded, config.Width, config.Height), nil
}

func gifFrameCount(data []byte) (int, error) {
	if len(data) < 13 || string(data[:3]) != "GIF" {
		return 0, fmt.Errorf("invalid header")
	}
	position := 13
	packed := data[10]
	if packed&0x80 != 0 {
		position += 3 * (1 << ((packed & 0x07) + 1))
	}
	frames := 0
	skipSubBlocks := func() bool {
		for {
			if position >= len(data) {
				return false
			}
			size := int(data[position])
			position++
			if size == 0 {
				return true
			}
			if size > len(data)-position {
				return false
			}
			position += size
		}
	}
	for position < len(data) {
		switch data[position] {
		case 0x3b:
			return frames, nil
		case 0x21:
			position += 2
			if position > len(data) || !skipSubBlocks() {
				return 0, fmt.Errorf("truncated extension")
			}
		case 0x2c:
			frames++
			position++
			if len(data)-position < 9 {
				return 0, fmt.Errorf("truncated image descriptor")
			}
			localPacked := data[position+8]
			position += 9
			if localPacked&0x80 != 0 {
				position += 3 * (1 << ((localPacked & 0x07) + 1))
			}
			if position >= len(data) {
				return 0, fmt.Errorf("truncated image data")
			}
			position++ // LZW minimum code size.
			if !skipSubBlocks() {
				return 0, fmt.Errorf("truncated image data")
			}
		default:
			return 0, fmt.Errorf("invalid block marker 0x%02x", data[position])
		}
	}
	return 0, fmt.Errorf("missing trailer")
}

func boundsDimensions(bounds stdimage.Rectangle) (int, int, error) {
	if bounds.Max.X < bounds.Min.X || bounds.Max.Y < bounds.Min.Y {
		return 0, 0, fmt.Errorf("image: invalid bounds")
	}
	width := uint64(bounds.Max.X) - uint64(bounds.Min.X)
	height := uint64(bounds.Max.Y) - uint64(bounds.Min.Y)
	if width > uint64(math.MaxInt) || height > uint64(math.MaxInt) {
		return 0, 0, fmt.Errorf("image: bounds exceed platform integer dimensions")
	}
	return int(width), int(height), nil
}

func validateDimensions(width, height int, maxPixels int64) error {
	if width <= 0 || height <= 0 || int64(width) > math.MaxInt64/int64(height) {
		return fmt.Errorf("image: invalid dimensions %dx%d", width, height)
	}
	pixels := int64(width) * int64(height)
	if pixels > maxPixels || pixels > int64(math.MaxInt-entryOverhead)/4 {
		return fmt.Errorf("image: dimensions %dx%d exceed %d pixels", width, height, maxPixels)
	}
	return nil
}

func convert(key Key, source stdimage.Image, width, height int) *Bitmap {
	bounds := source.Bounds()
	pixels := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := color.NRGBAModel.Convert(source.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.NRGBA)
			offset := (y*width + x) * 4
			pixels[offset], pixels[offset+1], pixels[offset+2], pixels[offset+3] = value.R, value.G, value.B, value.A
		}
	}
	return &Bitmap{Key: key, Width: width, Height: height, Pixels: pixels}
}
