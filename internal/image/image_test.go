package image

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"strings"
	"testing"
)

func encodedPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	value := image.NewNRGBA(image.Rect(0, 0, width, height))
	value.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 128})
	var output bytes.Buffer
	if err := png.Encode(&output, value); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestDecodeValidatesDimensionsAndReusesCache(t *testing.T) {
	cache := NewCache(1024)
	source := Source{ID: 1, Encoded: encodedPNG(t, 3, 2)}
	first, err := cache.Decode(source, 6)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Decode(source, 6)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.Width != 3 || first.Height != 2 || len(first.Pixels) != 24 {
		t.Fatalf("decoded/cache result = %#v %#v", first, second)
	}
	if got := cache.Stats(); got.Entries != 1 || got.Bytes != entryOverhead+24 {
		t.Fatalf("cache stats = %+v", got)
	}
	if _, err := cache.Decode(Source{ID: 2, Encoded: encodedPNG(t, 4, 4)}, 15); err == nil || !strings.Contains(err.Error(), "exceed") {
		t.Fatalf("oversized dimensions error = %v", err)
	}
}

func TestDecodeRejectsInvalidAndAnimatedGIF(t *testing.T) {
	cache := NewCache(1024)
	if _, err := cache.Decode(Source{ID: 1, Encoded: []byte("not an image")}, 10); err == nil {
		t.Fatal("invalid image decoded")
	}
	animation := &gif.GIF{
		Image: []*image.Paletted{
			image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black}),
			image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.White}),
		},
		Delay: []int{1, 1},
	}
	var encoded bytes.Buffer
	if err := gif.EncodeAll(&encoded, animation); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Decode(Source{ID: 2, Encoded: encoded.Bytes()}, 10); err == nil || !strings.Contains(err.Error(), "animated GIF") {
		t.Fatalf("animated GIF error = %v", err)
	}
}

func TestCacheEvictsByBytesAndClearReleasesReferences(t *testing.T) {
	oneEntry := entryOverhead + 16
	cache := NewCache(oneEntry)
	first, err := cache.Decode(Source{ID: 1, Encoded: encodedPNG(t, 2, 2)}, 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Decode(Source{ID: 2, Encoded: encodedPNG(t, 2, 2)}, 4); err != nil {
		t.Fatal(err)
	}
	if got := cache.Stats(); got.Entries != 1 || got.Evictions != 1 {
		t.Fatalf("post-eviction stats = %+v", got)
	}
	again, err := cache.Decode(Source{ID: 1, Encoded: encodedPNG(t, 2, 2)}, 4)
	if err != nil {
		t.Fatal(err)
	}
	if again == first {
		t.Fatal("evicted bitmap was unexpectedly reused")
	}
	cache.Clear()
	if got := cache.Stats(); got.Bytes != 0 || got.Entries != 0 {
		t.Fatalf("clear stats = %+v", got)
	}
}

func TestActiveDecodedEntryIsNotEvictedUntilReplacement(t *testing.T) {
	oneEntry := entryOverhead + 16
	cache := NewCache(oneEntry)
	firstSource := Source{ID: 1, Encoded: encodedPNG(t, 2, 2)}
	secondSource := Source{ID: 2, Encoded: encodedPNG(t, 2, 2)}
	cache.SetActive([]Key{SourceKey(1, 4)})
	first, err := cache.Decode(firstSource, 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Decode(secondSource, 4); err != nil {
		t.Fatal(err)
	}
	if got, err := cache.Decode(firstSource, 4); err != nil || got != first {
		t.Fatalf("active entry was evicted: bitmap=%p want=%p err=%v", got, first, err)
	}
	cache.SetActive([]Key{SourceKey(2, 4)})
	second, err := cache.Decode(secondSource, 4)
	if err != nil {
		t.Fatal(err)
	}
	if stats := cache.Stats(); stats.Entries != 1 || second == nil || stats.Evictions == 0 {
		t.Fatalf("replacement did not release capacity: %+v", stats)
	}
}

func TestActiveOversizedEntryLivesUntilDisplayRelease(t *testing.T) {
	cache := NewCache(entryOverhead + 4)
	source := Source{ID: 1, Encoded: encodedPNG(t, 2, 2)}
	key := SourceKey(source.ID, 4)
	cache.SetActive([]Key{key})
	first, err := cache.Decode(source, 4)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Decode(source, 4)
	if err != nil || second != first {
		t.Fatalf("active oversized decode = %p/%p, %v", first, second, err)
	}
	stats := cache.Stats()
	if stats.DecodeCount != 1 || stats.ActiveBytes != 16 || stats.Bytes <= stats.BudgetBytes {
		t.Fatalf("active oversized stats = %+v", stats)
	}
	cache.SetActive(nil)
	stats = cache.Stats()
	if stats.Bytes != 0 || stats.Entries != 0 || stats.Evictions != 1 {
		t.Fatalf("released oversized stats = %+v", stats)
	}
}

func TestActiveOversizedRollbackRestoresPreviousWorkingSet(t *testing.T) {
	cache := NewCache(entryOverhead + 4)
	firstSource := Source{ID: 1, Encoded: encodedPNG(t, 2, 2)}
	secondSource := Source{ID: 2, Encoded: encodedPNG(t, 2, 2)}
	firstKey, secondKey := SourceKey(1, 4), SourceKey(2, 4)
	cache.SetActive([]Key{firstKey})
	first, err := cache.Decode(firstSource, 4)
	if err != nil {
		t.Fatal(err)
	}
	previous := cache.ActiveKeys()
	cache.SetActive([]Key{firstKey, secondKey})
	if _, err := cache.Decode(secondSource, 4); err != nil {
		t.Fatal(err)
	}
	cache.SetActive(previous)
	got, err := cache.Decode(firstSource, 4)
	if err != nil || got != first {
		t.Fatalf("rollback lost prior bitmap: got=%p want=%p err=%v", got, first, err)
	}
	if stats := cache.Stats(); stats.Entries != 1 || stats.ActiveBytes != 16 || stats.DecodeCount != 2 {
		t.Fatalf("rollback stats = %+v", stats)
	}
}
