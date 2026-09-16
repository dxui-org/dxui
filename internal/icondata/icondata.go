// Package icondata owns dxui's compact immutable vector-icon representation.
package icondata

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
)

// Point is a position in icon view-box coordinates.
type Point struct{ X, Y float32 }

// Rect is a rectangle in icon view-box coordinates.
type Rect struct{ X, Y, Width, Height float32 }

// PathVerb identifies one vector path command.
type PathVerb uint8

const (
	PathMove PathVerb = iota
	PathLine
	PathQuad
	PathCubic
	PathClose
)

// PathCommand stores up to three points.
type PathCommand struct {
	Verb   PathVerb
	Points [3]Point
}

// PaintMode identifies whether a packed path is filled or stroked.
type PaintMode uint8

const (
	PaintStroke PaintMode = iota
	PaintFill
)

// Path is one decoded path with a single paint mode.
type Path struct {
	Mode     PaintMode
	Commands []PathCommand
}

// Data is immutable when returned by Packed. Commands remains exported for
// compatibility with application-authored filled icons; callers own that
// slice and dxui copies it when constructing a View.
type Data struct {
	ViewBox  Rect
	Commands []PathCommand
	packed   string
}

// Packed constructs immutable generated icon data. The encoding starts with
// a four-float view box, followed by styled command groups.
func Packed(encoded string) Data {
	if len(encoded) < 16 {
		return Data{}
	}
	return Data{ViewBox: Rect{
		X: float32From(encoded[0:4]), Y: float32From(encoded[4:8]),
		Width: float32From(encoded[8:12]), Height: float32From(encoded[12:16]),
	}, packed: encoded}
}

// IsPacked reports whether data uses the immutable generated representation.
func (data Data) IsPacked() bool { return data.packed != "" }

// Identity returns a stable content identity without exposing packed bytes.
func (data Data) Identity() uint64 {
	hash := fnv.New64a()
	if data.packed != "" {
		_, _ = hash.Write([]byte(data.packed))
		return hash.Sum64()
	}
	writeFloat := func(value float32) {
		var bytes [4]byte
		binary.LittleEndian.PutUint32(bytes[:], math.Float32bits(value))
		_, _ = hash.Write(bytes[:])
	}
	writeFloat(data.ViewBox.X)
	writeFloat(data.ViewBox.Y)
	writeFloat(data.ViewBox.Width)
	writeFloat(data.ViewBox.Height)
	for _, command := range data.Commands {
		_, _ = hash.Write([]byte{byte(command.Verb)})
		for _, point := range command.Points {
			writeFloat(point.X)
			writeFloat(point.Y)
		}
	}
	return hash.Sum64()
}

// Paths validates and decodes data. Legacy Commands form one filled path.
func (data Data) Paths(maxCommands int) ([]Path, error) {
	if maxCommands <= 0 {
		return nil, fmt.Errorf("icondata: command limit must be positive")
	}
	if data.packed == "" {
		if len(data.Commands) > maxCommands {
			return nil, fmt.Errorf("icondata: %d commands exceed limit %d", len(data.Commands), maxCommands)
		}
		return []Path{{Mode: PaintFill, Commands: data.Commands}}, nil
	}
	if len(data.packed) < 16 {
		return nil, fmt.Errorf("icondata: truncated header")
	}
	offset := 16
	paths := make([]Path, 0, 8)
	total := 0
	for offset < len(data.packed) {
		if len(data.packed)-offset < 3 {
			return nil, fmt.Errorf("icondata: truncated path header at byte %d", offset)
		}
		mode := PaintMode(data.packed[offset])
		if mode > PaintFill {
			return nil, fmt.Errorf("icondata: unknown paint mode %d", mode)
		}
		count := int(binary.LittleEndian.Uint16([]byte(data.packed[offset+1 : offset+3])))
		offset += 3
		if count == 0 || count > maxCommands-total {
			return nil, fmt.Errorf("icondata: invalid command count %d", count)
		}
		commands := make([]PathCommand, 0, count)
		for index := 0; index < count; index++ {
			if offset >= len(data.packed) {
				return nil, fmt.Errorf("icondata: truncated command %d", total+index)
			}
			verb := PathVerb(data.packed[offset])
			offset++
			points := commandPoints(verb)
			if points < 0 {
				return nil, fmt.Errorf("icondata: unknown path verb %d", verb)
			}
			need := points * 8
			if len(data.packed)-offset < need {
				return nil, fmt.Errorf("icondata: truncated path verb %d", verb)
			}
			var command PathCommand
			command.Verb = verb
			for point := 0; point < points; point++ {
				command.Points[point] = Point{X: float32From(data.packed[offset : offset+4]), Y: float32From(data.packed[offset+4 : offset+8])}
				offset += 8
			}
			commands = append(commands, command)
		}
		total += count
		paths = append(paths, Path{Mode: mode, Commands: commands})
	}
	return paths, nil
}

func commandPoints(verb PathVerb) int {
	switch verb {
	case PathMove, PathLine:
		return 1
	case PathQuad:
		return 2
	case PathCubic:
		return 3
	case PathClose:
		return 0
	default:
		return -1
	}
}

func float32From(value string) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32([]byte(value)))
}
