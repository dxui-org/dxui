package dxui

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	internaltext "github.com/dxui-org/dxui/internal/text"
)

func TestFontBytesCopiesApplicationData(t *testing.T) {
	data := []byte{1, 2, 3}
	font := FontBytes("test", data)
	data[0] = 9
	if font.source.data[0] != 1 {
		t.Fatal("FontBytes retained mutable caller data")
	}
}

func TestSystemFontCandidatesAreDeterministicPerPlatform(t *testing.T) {
	tests := []struct {
		goos   string
		family FontFamily
		root   string
		want   []systemFontCandidate
	}{
		{goos: "windows", family: FontFamilySystemCJK, root: `D:\Windows`, want: []systemFontCandidate{
			{path: filepath.Join(`D:\Windows`, "Fonts", "msyh.ttc")}, {path: filepath.Join(`D:\Windows`, "Fonts", "msjh.ttc")}, {path: filepath.Join(`D:\Windows`, "Fonts", "simsun.ttc")},
		}},
		{goos: "darwin", family: FontFamilySystemCJK, want: []systemFontCandidate{{path: "/System/Library/Fonts/PingFang.ttc"}, {path: "/System/Library/Fonts/STHeiti Light.ttc"}}},
		{goos: "linux", family: FontFamilySystemCJK, want: []systemFontCandidate{{path: "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", faceIndex: 2}, {path: "/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"}}},
		{goos: "freebsd", family: FontFamilySystemCJK, want: []systemFontCandidate{{path: "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", faceIndex: 2}, {path: "/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"}}},
		{goos: "openbsd", family: FontFamilySystemCJK, want: []systemFontCandidate{{path: "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", faceIndex: 2}, {path: "/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"}}},
		{goos: "netbsd", family: FontFamilySystemCJK, want: []systemFontCandidate{{path: "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", faceIndex: 2}, {path: "/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"}}},
	}
	for _, test := range tests {
		candidates := systemFontCandidates(test.goos, test.family, test.root)
		if !reflect.DeepEqual(candidates, test.want) {
			t.Fatalf("%s/%s candidates = %+v, want %+v", test.goos, test.family, candidates, test.want)
		}
	}
	if candidates := systemFontCandidates("plan9", FontFamilySystemSans, ""); candidates != nil {
		t.Fatalf("unsupported platform candidates = %+v", candidates)
	}
}

type fakeFontFileLoader struct {
	data  map[string][]byte
	errs  map[string]error
	calls []string
	max   []int
}

func (loader *fakeFontFileLoader) Open(path string, maxBytes int) (*internaltext.FontSource, error) {
	loader.calls = append(loader.calls, path)
	loader.max = append(loader.max, maxBytes)
	if err := loader.errs[path]; err != nil {
		return nil, err
	}
	data := loader.data[path]
	if len(data) > maxBytes {
		return nil, errors.New("over budget")
	}
	return &internaltext.FontSource{ReaderAt: bytes.NewReader(data), Size: int64(len(data)), Identity: "system:" + path}, nil
}

func TestSystemFallbackResolverCachesAttemptsAndPreservesOrder(t *testing.T) {
	loader := &fakeFontFileLoader{
		data: map[string][]byte{"bad": []byte("broken"), "second": []byte("font")},
		errs: map[string]error{"missing": os.ErrNotExist},
	}
	resolver := newSystemFallbackResolver([]systemFontCandidate{{path: "missing"}, {path: "bad", faceIndex: 8}, {path: "second", faceIndex: 2}}, loader)
	tries := 0
	resolver.Resolve('中', 16, func(spec internaltext.FontSpec) (bool, error) {
		tries++
		if spec.Identity == "system:bad#8" {
			return false, errors.New("invalid collection face")
		}
		if spec.Identity != "system:second#2" || spec.FaceIndex != 2 {
			t.Fatalf("unexpected spec: %+v", spec)
		}
		return true, nil
	})
	resolver.Resolve('文', 16, func(internaltext.FontSpec) (bool, error) {
		t.Fatal("cached candidates were retried")
		return false, nil
	})
	if !reflect.DeepEqual(loader.calls, []string{"missing", "bad", "second"}) || tries != 2 {
		t.Fatalf("calls=%v tries=%d", loader.calls, tries)
	}
}

func TestDisableAutomaticSystemFallback(t *testing.T) {
	app := NewApp(AppOptions{DisableSystemFontFallback: true})
	engine, err := app.textEngine()
	if err != nil {
		t.Fatal(err)
	}
	defer app.releaseTextEngine()
	layout, err := engine.Layout(internaltext.Request{Text: "中"})
	if err != nil || layout.Metrics.MissingGlyphs != 1 {
		t.Fatalf("layout=%+v err=%v", layout, err)
	}
}

func TestAppTextEngineOwnsAndReleasesCaches(t *testing.T) {
	app := NewApp(AppOptions{Caches: CacheBudgets{TextMeasureBytes: 2048}})
	engine, err := app.textEngine()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Layout(internaltext.Request{Text: "lifecycle"}); err != nil {
		t.Fatal(err)
	}
	if stats := engine.Stats(); stats.Bytes == 0 || stats.Bytes > 2048 {
		t.Fatalf("live text cache = %+v", stats)
	}
	app.releaseTextEngine()
	if stats := engine.Stats(); stats.Bytes != 0 || stats.Entries != 0 {
		t.Fatalf("released text cache = %+v", stats)
	}
}

func TestRegisteredFontSourcesHaveStrictByteBudget(t *testing.T) {
	fonts := []Font{FontBytes("too-large", make([]byte, 9))}
	if _, err := resolveFontSpecs(fonts, 8); err == nil {
		t.Fatal("font source exceeding byte budget was accepted")
	}
	if _, err := resolveFontSpecs(fonts, -1); err == nil {
		t.Fatal("registered font was accepted with disabled font-source budget")
	}
}

func TestFontFileReadIsLimitedBeforeParsing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large-font.ttf")
	if err := os.WriteFile(path, make([]byte, 64), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := resolveFontSource(FontFile("limited", path), 32)
	if err == nil {
		t.Fatal("font file exceeding read limit was accepted")
	}
}
