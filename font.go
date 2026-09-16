package dxui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	internaltext "github.com/dxui-org/dxui/internal/text"
)

// FontFamily is an application-visible family name used for deterministic
// ordered fallback. It is not an operating-system font handle.
type FontFamily string

const (
	// FontFamilyDefault is dxui's embedded, BSD-licensed Go Regular font. It
	// covers the MVP Latin samples but not CJK.
	FontFamilyDefault FontFamily = internaltext.BuiltinFamily
	// FontFamilySystemSans resolves one fixed candidate list per operating
	// system; it is never searched by fuzzy display name.
	FontFamilySystemSans FontFamily = "system-ui"
	// FontFamilySystemMono is the deterministic system monospace family.
	FontFamilySystemMono FontFamily = "system-monospace"
	// FontFamilySystemCJK is the deterministic system CJK fallback family.
	FontFamilySystemCJK FontFamily = "system-cjk"
)

type fontSourceKind uint8

const (
	fontSourceBytes fontSourceKind = iota + 1
	fontSourceFile
	fontSourceSystem
)

type fontSource struct {
	kind fontSourceKind
	data []byte
	path string
}

// Font registers one TTF/OTF/TTC face. A FontBytes value owns its copied bytes
// for as long as that value or an App configured with it remains reachable.
// FontFile handles and all parsed faces live only from App.Run text startup until
// renderer teardown completes. The file is not copied into the Go heap.
// FaceIndex selects a collection face and is zero
// for ordinary fonts.
type Font struct {
	Family    FontFamily
	Weight    FontWeight
	Slant     FontSlant
	FaceIndex int
	source    fontSource
}

// FontBytes creates an application-owned font from a defensive copy of data.
func FontBytes(family FontFamily, data []byte) Font {
	return Font{Family: family, Weight: WeightRegular, source: fontSource{kind: fontSourceBytes, data: append([]byte(nil), data...)}}
}

// FontFile creates a font loaded from the exact path during App.Run startup.
// dxui does not search or copy the file into an application package.
func FontFile(family FontFamily, path string) Font {
	return Font{Family: family, Weight: WeightRegular, source: fontSource{kind: fontSourceFile, path: path}}
}

// SystemFont requests one of the documented generic system families. The
// resolver tests a fixed path list for the current OS and returns a startup
// error when none exists; it never depends on fontconfig or fuzzy name search.
func SystemFont(family FontFamily) Font {
	return Font{Family: family, Weight: WeightRegular, source: fontSource{kind: fontSourceSystem}}
}

type systemFontCandidate struct {
	path      string
	faceIndex int
}

type fontFileLoader interface {
	Open(path string, maxBytes int) (*internaltext.FontSource, error)
}

type osFontFileLoader struct{}

func (osFontFileLoader) Open(path string, maxBytes int) (*internaltext.FontSource, error) {
	return openFontFile(path, maxBytes)
}

type systemFallbackResolver struct {
	candidates []systemFontCandidate
	loader     fontFileLoader
	failed     []bool
	loaded     []bool
}

func newSystemFallbackResolver(candidates []systemFontCandidate, loader fontFileLoader) *systemFallbackResolver {
	return &systemFallbackResolver{candidates: append([]systemFontCandidate(nil), candidates...), loader: loader, failed: make([]bool, len(candidates)), loaded: make([]bool, len(candidates))}
}

func (resolver *systemFallbackResolver) Resolve(r rune, remainingBytes int, try func(internaltext.FontSpec) (bool, error)) {
	for index, candidate := range resolver.candidates {
		if resolver == nil || resolver.failed[index] || resolver.loaded[index] {
			continue
		}
		source, err := resolver.loader.Open(candidate.path, remainingBytes)
		if err != nil {
			resolver.failed[index] = true
			continue
		}
		covers, err := try(internaltext.FontSpec{
			Family: string(FontFamilySystemCJK), Weight: 400, FaceIndex: candidate.faceIndex,
			Source: source, Identity: source.Identity + fmt.Sprintf("#%d", candidate.faceIndex),
		})
		if err == nil {
			resolver.loaded[index] = true
			remainingBytes -= int(source.Size)
			if covers {
				return
			}
		} else {
			resolver.failed[index] = true
		}
	}
}

func resolveFontSpecs(fonts []Font, byteBudget int) ([]internaltext.FontSpec, error) {
	result, _, err := resolveFontSpecsWithUsage(fonts, byteBudget)
	return result, err
}

func resolveFontSpecsWithUsage(fonts []Font, byteBudget int) ([]internaltext.FontSpec, int, error) {
	byteBudget = normalizedBudget(byteBudget, 32<<20)
	used := 0
	result := make([]internaltext.FontSpec, 0, len(fonts))
	success := false
	defer func() {
		if !success {
			for _, spec := range result {
				if spec.Source != nil && spec.Source.Closer != nil {
					_ = spec.Source.Closer.Close()
				}
			}
		}
	}()
	for index, value := range fonts {
		family := string(value.Family)
		if family == "" || strings.IndexByte(family, 0) >= 0 || value.FaceIndex < 0 || value.Weight > 1000 || value.Slant > SlantItalic {
			return nil, 0, fmt.Errorf("font %d has invalid family, face index, weight, or slant", index)
		}
		source, faceIndex, identity, err := resolveFontSource(value, byteBudget-used)
		if err != nil {
			return nil, 0, fmt.Errorf("font %d (%s): %w", index, family, err)
		}
		if source.Size > int64(byteBudget-used) {
			return nil, 0, fmt.Errorf("font sources exceed %d-byte budget at font %d (%s)", byteBudget, index, family)
		}
		used += int(source.Size)
		if identity != "" {
			identity = fmt.Sprintf("%s#%d", identity, faceIndex)
		}
		result = append(result, internaltext.FontSpec{
			Family: family, Weight: uint16(value.Weight), Slant: uint8(value.Slant),
			FaceIndex: faceIndex, Source: source, Identity: identity,
		})
	}
	success = true
	return result, used, nil
}

func normalizedBudget(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	if value < 0 {
		return 0
	}
	return value
}

func resolveFontSource(value Font, maxBytes int) (*internaltext.FontSource, int, string, error) {
	switch value.source.kind {
	case fontSourceBytes:
		if len(value.source.data) > maxBytes {
			return nil, 0, "", fmt.Errorf("font source requires %d bytes, remaining budget is %d", len(value.source.data), maxBytes)
		}
		return &internaltext.FontSource{Data: value.source.data, Size: int64(len(value.source.data))}, value.FaceIndex, "", nil
	case fontSourceFile:
		path, err := filepath.Abs(value.source.path)
		if err != nil {
			return nil, 0, "", err
		}
		source, err := openFontFile(path, maxBytes)
		if err != nil {
			return nil, 0, "", err
		}
		return source, value.FaceIndex, source.Identity, nil
	case fontSourceSystem:
		candidates := systemFontCandidates(runtime.GOOS, value.Family, os.Getenv("WINDIR"))
		for _, candidate := range candidates {
			source, err := openFontFile(candidate.path, maxBytes)
			if err == nil {
				return source, candidate.faceIndex, source.Identity, nil
			}
			if !errors.Is(err, os.ErrNotExist) {
				return nil, 0, "", fmt.Errorf("read deterministic system-font candidate %q: %w", candidate.path, err)
			}
		}
		return nil, 0, "", fmt.Errorf("no deterministic %q system-font candidate exists", value.Family)
	default:
		return nil, 0, "", fmt.Errorf("font has no source")
	}
}

func openFontFile(path string, maxBytes int) (*internaltext.FontSource, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("font source exceeds remaining %d-byte budget", maxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if info.Size() < 0 || info.Size() > int64(maxBytes) {
		_ = file.Close()
		return nil, fmt.Errorf("font source size %d exceeds remaining %d-byte budget", info.Size(), maxBytes)
	}
	canonical, err := filepath.Abs(path)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	identity := fmt.Sprintf("file:%s:size=%d:mtime=%d", filepath.Clean(canonical), info.Size(), info.ModTime().UnixNano())
	return &internaltext.FontSource{ReaderAt: io.NewSectionReader(file, 0, info.Size()), Size: info.Size(), Identity: identity, Closer: file}, nil
}

func systemFontCandidates(goos string, family FontFamily, windowsDirectory string) []systemFontCandidate {
	joinWindows := func(name string) string {
		root := windowsDirectory
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "Fonts", name)
	}
	switch goos {
	case "windows":
		switch family {
		case FontFamilySystemSans:
			return []systemFontCandidate{{path: joinWindows("segoeui.ttf")}, {path: joinWindows("arial.ttf")}}
		case FontFamilySystemMono:
			return []systemFontCandidate{{path: joinWindows("consola.ttf")}, {path: joinWindows("cour.ttf")}}
		case FontFamilySystemCJK:
			return []systemFontCandidate{{path: joinWindows("msyh.ttc")}, {path: joinWindows("msjh.ttc")}, {path: joinWindows("simsun.ttc")}}
		}
	case "darwin":
		switch family {
		case FontFamilySystemSans:
			return []systemFontCandidate{{path: "/System/Library/Fonts/SFNS.ttf"}, {path: "/System/Library/Fonts/Helvetica.ttc"}}
		case FontFamilySystemMono:
			return []systemFontCandidate{{path: "/System/Library/Fonts/SFNSMono.ttf"}, {path: "/System/Library/Fonts/Monaco.ttf"}}
		case FontFamilySystemCJK:
			return []systemFontCandidate{{path: "/System/Library/Fonts/PingFang.ttc"}, {path: "/System/Library/Fonts/STHeiti Light.ttc"}}
		}
	case "linux", "freebsd", "openbsd", "netbsd":
		switch family {
		case FontFamilySystemSans:
			return []systemFontCandidate{{path: "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"}, {path: "/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf"}}
		case FontFamilySystemMono:
			return []systemFontCandidate{{path: "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"}, {path: "/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf"}}
		case FontFamilySystemCJK:
			return []systemFontCandidate{{path: "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", faceIndex: 2}, {path: "/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc"}}
		}
	}
	return nil
}
