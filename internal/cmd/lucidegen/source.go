package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/dxui-org/dxui/internal/icondata"
)

const (
	lucideVersion = "1.41.0"
	lucideCommit  = "bca7e75a816dcf1e75e8feb5a3198a68cbb8a052"
	lucideSHA256  = "d10f583e2076986ddcf79def168e6ce1e64f742fa06b6014347c59fa581aa6cb"
)

type sourceIcon struct {
	slug      string
	encoded   string
	canonical bool
	aliasOf   string
}

type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Nodes   []xmlNode  `xml:",any"`
}

func loadSource(filename string) ([]sourceIcon, string, error) {
	archive, err := os.Open(filename)
	if err != nil {
		return nil, "", err
	}
	defer archive.Close()
	hash := sha256.New()
	gzipReader, err := gzip.NewReader(io.TeeReader(archive, hash))
	if err != nil {
		return nil, "", err
	}
	tarReader := tar.NewReader(gzipReader)
	svgs := make(map[string][]byte)
	canonical := make(map[string]json.RawMessage)
	var license string
	for {
		header, nextErr := tarReader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, "", nextErr
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if header.Size < 0 || header.Size > 32<<20 {
			return nil, "", fmt.Errorf("source entry %s has invalid size %d", header.Name, header.Size)
		}
		if header.Name != "package/LICENSE" && header.Name != "package/icon-nodes.json" && !(strings.HasPrefix(header.Name, "package/icons/") && strings.HasSuffix(header.Name, ".svg")) {
			continue
		}
		value, readErr := io.ReadAll(io.LimitReader(tarReader, header.Size+1))
		if readErr != nil || int64(len(value)) != header.Size {
			return nil, "", fmt.Errorf("read %s: %w", header.Name, readErr)
		}
		switch header.Name {
		case "package/LICENSE":
			license = string(value)
		case "package/icon-nodes.json":
			if err := json.Unmarshal(value, &canonical); err != nil {
				return nil, "", fmt.Errorf("icon-nodes.json: %w", err)
			}
		default:
			slug := strings.TrimSuffix(path.Base(header.Name), ".svg")
			svgs[slug] = value
		}
	}
	if err := gzipReader.Close(); err != nil {
		return nil, "", err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != lucideSHA256 {
		return nil, "", fmt.Errorf("source SHA-256 %s, require %s", got, lucideSHA256)
	}
	if len(canonical) == 0 || license == "" {
		return nil, "", fmt.Errorf("source archive lacks icon metadata or license")
	}
	icons := make([]sourceIcon, 0, len(svgs))
	canonicalByData := make(map[string]string)
	for slug, svg := range svgs {
		encoded, convertErr := convertSVG(svg)
		if convertErr != nil {
			return nil, "", fmt.Errorf("%s.svg: %w", slug, convertErr)
		}
		_, isCanonical := canonical[slug]
		icons = append(icons, sourceIcon{slug: slug, encoded: encoded, canonical: isCanonical})
		if isCanonical {
			if old, exists := canonicalByData[encoded]; !exists || slug < old {
				canonicalByData[encoded] = slug
			}
		}
	}
	for index := range icons {
		if icons[index].canonical {
			continue
		}
		target, ok := canonicalByData[icons[index].encoded]
		if !ok {
			return nil, "", fmt.Errorf("alias %s has no canonical icon with identical geometry", icons[index].slug)
		}
		icons[index].aliasOf = target
	}
	sort.Slice(icons, func(one, two int) bool { return icons[one].slug < icons[two].slug })
	return icons, license, nil
}

func convertSVG(source []byte) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(source)))
	var root xmlNode
	if err := decoder.Decode(&root); err != nil {
		return "", err
	}
	if root.XMLName.Local != "svg" {
		return "", fmt.Errorf("root element is %s", root.XMLName.Local)
	}
	attrs, err := attributes(root.Attrs, "class", "xmlns", "width", "height", "viewBox", "fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin")
	if err != nil {
		return "", err
	}
	if attrs["fill"] != "none" || attrs["stroke"] != "currentColor" || attrs["stroke-linecap"] != "round" || attrs["stroke-linejoin"] != "round" || attrs["stroke-width"] != "2" {
		return "", fmt.Errorf("unsupported root paint attributes")
	}
	boxValues, err := numbers(attrs["viewBox"], 4)
	if err != nil || boxValues[2] <= 0 || boxValues[3] <= 0 {
		return "", fmt.Errorf("invalid viewBox %q", attrs["viewBox"])
	}
	box := icondata.Rect{X: boxValues[0], Y: boxValues[1], Width: boxValues[2], Height: boxValues[3]}
	var paths []icondata.Path
	for _, node := range root.Nodes {
		commands, fill, nodeErr := convertNode(node)
		if nodeErr != nil {
			return "", nodeErr
		}
		paths = append(paths, icondata.Path{Mode: icondata.PaintStroke, Commands: commands})
		if fill {
			paths = append(paths, icondata.Path{Mode: icondata.PaintFill, Commands: commands})
		}
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("icon has no drawable elements")
	}
	return encodeIcon(box, paths)
}

func convertNode(node xmlNode) ([]icondata.PathCommand, bool, error) {
	allowed := map[string][]string{
		"path": {"d", "fill"}, "circle": {"cx", "cy", "r", "fill"},
		"ellipse": {"cx", "cy", "rx", "ry", "fill"}, "line": {"x1", "y1", "x2", "y2", "fill"},
		"polyline": {"points", "fill"}, "polygon": {"points", "fill"},
		"rect": {"x", "y", "width", "height", "rx", "ry", "fill"},
	}
	fields, ok := allowed[node.XMLName.Local]
	if !ok || len(node.Nodes) != 0 {
		return nil, false, fmt.Errorf("unsupported element <%s>", node.XMLName.Local)
	}
	attrs, err := attributes(node.Attrs, fields...)
	if err != nil {
		return nil, false, fmt.Errorf("<%s>: %w", node.XMLName.Local, err)
	}
	fill := attrs["fill"] != ""
	if fill && attrs["fill"] != "currentColor" {
		return nil, false, fmt.Errorf("<%s>: unsupported fill %q", node.XMLName.Local, attrs["fill"])
	}
	var commands []icondata.PathCommand
	switch node.XMLName.Local {
	case "path":
		commands, err = parsePathData(attrs["d"])
	case "line":
		values, valueErr := requiredNumbers(attrs, "x1", "y1", "x2", "y2")
		err = valueErr
		if err == nil {
			commands = lineCommands(icondata.Point{X: values[0], Y: values[1]}, icondata.Point{X: values[2], Y: values[3]})
		}
	case "polyline", "polygon":
		var points []icondata.Point
		points, err = parsePointList(attrs["points"])
		if err == nil {
			commands = append(commands, icondata.PathCommand{Verb: icondata.PathMove, Points: [3]icondata.Point{points[0]}})
			for _, point := range points[1:] {
				commands = append(commands, icondata.PathCommand{Verb: icondata.PathLine, Points: [3]icondata.Point{point}})
			}
			if node.XMLName.Local == "polygon" {
				commands = append(commands, icondata.PathCommand{Verb: icondata.PathClose})
			}
		}
	case "circle", "ellipse":
		keys := []string{"cx", "cy", "rx", "ry"}
		if node.XMLName.Local == "circle" {
			keys = []string{"cx", "cy", "r"}
		}
		values, valueErr := requiredNumbers(attrs, keys...)
		err = valueErr
		if err == nil {
			rx, ry := values[2], values[2]
			if node.XMLName.Local == "ellipse" {
				ry = values[3]
			}
			commands, err = ellipseCommands(values[0], values[1], rx, ry)
		}
	case "rect":
		values, valueErr := requiredNumbers(attrs, "x", "y", "width", "height")
		err = valueErr
		if err == nil {
			rx, ry := float32(0), float32(0)
			if attrs["rx"] != "" {
				rx, err = parseNumber(attrs["rx"])
			}
			if err == nil && attrs["ry"] != "" {
				ry, err = parseNumber(attrs["ry"])
			} else if attrs["ry"] == "" {
				ry = rx
			}
			if err == nil && attrs["rx"] == "" {
				rx = ry
			}
			if err == nil {
				commands, err = rectCommands(values[0], values[1], values[2], values[3], rx, ry)
			}
		}
	}
	if err != nil {
		return nil, false, fmt.Errorf("<%s>: %w", node.XMLName.Local, err)
	}
	return commands, fill, nil
}

func attributes(source []xml.Attr, allowed ...string) (map[string]string, error) {
	valid := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		valid[name] = struct{}{}
	}
	result := make(map[string]string, len(source))
	for _, attr := range source {
		name := attr.Name.Local
		if _, ok := valid[name]; !ok {
			return nil, fmt.Errorf("unsupported attribute %s", name)
		}
		if _, duplicate := result[name]; duplicate {
			return nil, fmt.Errorf("duplicate attribute %s", name)
		}
		result[name] = attr.Value
	}
	return result, nil
}

func requiredNumbers(attrs map[string]string, keys ...string) ([]float32, error) {
	result := make([]float32, len(keys))
	for index, key := range keys {
		if attrs[key] == "" {
			return nil, fmt.Errorf("missing %s", key)
		}
		value, err := parseNumber(attrs[key])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		result[index] = value
	}
	return result, nil
}

func parseNumber(source string) (float32, error) {
	value, err := strconv.ParseFloat(source, 32)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid number %q", source)
	}
	return float32(value), nil
}

func numbers(source string, count int) ([]float32, error) {
	fields := strings.Fields(source)
	if len(fields) != count {
		return nil, fmt.Errorf("got %d values, require %d", len(fields), count)
	}
	result := make([]float32, count)
	for index, field := range fields {
		value, err := parseNumber(field)
		if err != nil {
			return nil, err
		}
		result[index] = value
	}
	return result, nil
}

func lineCommands(one, two icondata.Point) []icondata.PathCommand {
	return []icondata.PathCommand{{Verb: icondata.PathMove, Points: [3]icondata.Point{one}}, {Verb: icondata.PathLine, Points: [3]icondata.Point{two}}}
}

func ellipseCommands(cx, cy, rx, ry float32) ([]icondata.PathCommand, error) {
	if rx < 0 || ry < 0 {
		return nil, fmt.Errorf("negative radius")
	}
	const kappa = float32(0.5522847498307936)
	return []icondata.PathCommand{
		{Verb: icondata.PathMove, Points: [3]icondata.Point{{X: cx + rx, Y: cy}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: cx + rx, Y: cy + kappa*ry}, {X: cx + kappa*rx, Y: cy + ry}, {X: cx, Y: cy + ry}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: cx - kappa*rx, Y: cy + ry}, {X: cx - rx, Y: cy + kappa*ry}, {X: cx - rx, Y: cy}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: cx - rx, Y: cy - kappa*ry}, {X: cx - kappa*rx, Y: cy - ry}, {X: cx, Y: cy - ry}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: cx + kappa*rx, Y: cy - ry}, {X: cx + rx, Y: cy - kappa*ry}, {X: cx + rx, Y: cy}}},
		{Verb: icondata.PathClose},
	}, nil
}

func rectCommands(x, y, width, height, rx, ry float32) ([]icondata.PathCommand, error) {
	if width < 0 || height < 0 || rx < 0 || ry < 0 {
		return nil, fmt.Errorf("negative rectangle dimension")
	}
	rx, ry = min(rx, width/2), min(ry, height/2)
	if rx == 0 || ry == 0 {
		return []icondata.PathCommand{
			{Verb: icondata.PathMove, Points: [3]icondata.Point{{X: x, Y: y}}},
			{Verb: icondata.PathLine, Points: [3]icondata.Point{{X: x + width, Y: y}}},
			{Verb: icondata.PathLine, Points: [3]icondata.Point{{X: x + width, Y: y + height}}},
			{Verb: icondata.PathLine, Points: [3]icondata.Point{{X: x, Y: y + height}}},
			{Verb: icondata.PathClose},
		}, nil
	}
	const kappa = float32(0.5522847498307936)
	return []icondata.PathCommand{
		{Verb: icondata.PathMove, Points: [3]icondata.Point{{X: x + rx, Y: y}}},
		{Verb: icondata.PathLine, Points: [3]icondata.Point{{X: x + width - rx, Y: y}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: x + width - rx + kappa*rx, Y: y}, {X: x + width, Y: y + ry - kappa*ry}, {X: x + width, Y: y + ry}}},
		{Verb: icondata.PathLine, Points: [3]icondata.Point{{X: x + width, Y: y + height - ry}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: x + width, Y: y + height - ry + kappa*ry}, {X: x + width - rx + kappa*rx, Y: y + height}, {X: x + width - rx, Y: y + height}}},
		{Verb: icondata.PathLine, Points: [3]icondata.Point{{X: x + rx, Y: y + height}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: x + rx - kappa*rx, Y: y + height}, {X: x, Y: y + height - ry + kappa*ry}, {X: x, Y: y + height - ry}}},
		{Verb: icondata.PathLine, Points: [3]icondata.Point{{X: x, Y: y + ry}}},
		{Verb: icondata.PathCubic, Points: [3]icondata.Point{{X: x, Y: y + ry - kappa*ry}, {X: x + rx - kappa*rx, Y: y}, {X: x + rx, Y: y}}},
		{Verb: icondata.PathClose},
	}, nil
}
