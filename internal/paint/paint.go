// Package paint owns backend-neutral display commands, replay state, stacking,
// hit testing, and dirty bounds.
package paint

import (
	"fmt"
	"math"
	"sort"
)

// Rect is a logical paint rectangle.
type Rect struct{ X, Y, Width, Height float32 }

// PixelRect is a rectangle in physical pixels.
type PixelRect struct{ X, Y, Width, Height int32 }

// Color is an 8-bit non-premultiplied RGBA color. Replay multiplies only A by
// the current world opacity; the renderer uses source-over blending.
type Color struct{ R, G, B, A uint8 }

// Radii contains top-left, top-right, bottom-right, bottom-left radii.
type Radii struct{ TopLeft, TopRight, BottomRight, BottomLeft float32 }

// Shadow is the resolved outer-shadow input.
type Shadow struct {
	OffsetX, OffsetY float32
	Blur, Spread     float32
	Color            Color
}

// CommandKind identifies one display-list operation. CommandItem is retained
// as the geometry-only hit-test item used by the layout milestone.
type CommandKind uint8

const (
	CommandItem CommandKind = iota
	CommandBeginDisplayList
	CommandPushClip
	CommandPopClip
	CommandPushOpacity
	CommandPopOpacity
	CommandDrawShadow
	CommandFillRoundedRect
	CommandStrokeRoundedRect
	CommandFillStrokeRoundedRect
	CommandDrawText
	CommandDrawIcon
	CommandDrawImage
	CommandBorder
)

// JoinedEdges marks edges that connect directly to another member of the same
// painted shape. Joined edges are solid and have no coverage-band triangles;
// only exposed edges receive an alpha fringe. Natural source order gives the
// later left/top edge ownership of the shared native pixels.
type JoinedEdges uint8

const (
	JoinedLeft JoinedEdges = 1 << iota
	JoinedTop
	JoinedRight
	JoinedBottom
)

// TextBitmap is an immutable grayscale line mask produced by internal/text.
// Key contains font identity, size, scale, and render mode; Pixels contains
// exactly Width*Height alpha bytes and is retained only by the display list.
type TextBitmap struct {
	Key           string
	Width, Height int
	Pixels        []byte
}

// ImageBitmap is immutable tightly packed non-premultiplied RGBA data.
type ImageBitmap struct {
	Key           string
	Width, Height int
	Pixels        []byte
}

// Command is a compact tagged paint operation. The legacy item fields are
// also the retained hit-test representation; structured command lists are
// emitted in final stable paint order and are never globally resorted.
type Command struct {
	Kind        CommandKind
	NodeID      uint64
	Rect        Rect
	Radii       Radii
	Color       Color
	JoinedEdges JoinedEdges
	Sides       BorderSides
	Dashed      bool
	// BorderColor is used only by CommandFillStrokeRoundedRect.
	BorderColor Color
	Width       float32
	Shadow      Shadow
	Opacity     float32
	Text        *TextBitmap
	Image       *ImageBitmap
	// ImageClip is an optional rounded destination clip used by image-backed
	// components such as Avatar. A zero rectangle keeps the ordinary image path.
	ImageClip Rect

	Bounds      Rect
	Clip        Rect
	ClipSet     bool
	ZIndex      int
	SourceIndex int
	Visibility  bool
	Interactive bool
}

// Item is the compatibility name for a retained hit-test command.
type Item = Command

// DisplayList is an immutable-by-convention ordered command stream.
type DisplayList []Command

// Painter is the SDL-independent execution boundary. SetClip receives the
// already-intersected world clip. Drawing colors already contain composed
// parent/child opacity.
type Painter interface {
	SetClip(Rect, bool) error
	DrawShadow(Rect, Radii, Shadow) error
	FillRoundedRect(Rect, Radii, Color, JoinedEdges) error
	StrokeRoundedRect(Rect, Radii, float32, Color) error
	FillStrokeRoundedRect(Rect, Radii, float32, Color, Color) error
	DrawBorder(Rect, Radii, float32, Color, BorderSides, bool) error
	DrawText(Rect, *TextBitmap, Color) error
	DrawIcon(Rect, *TextBitmap, Color) error
	DrawImage(Rect, *ImageBitmap, uint8, Rect, Radii) error
}

// Replay validates stack balance, composes opacity and nested clips, and
// executes drawing commands in list order.
func Replay(list DisplayList, painter Painter) error {
	if painter == nil {
		return fmt.Errorf("paint: nil painter")
	}
	opacities := []float32{1}
	clips := make([]Rect, 0, 4)
	for index, command := range list {
		switch command.Kind {
		case CommandItem:
			continue
		case CommandBeginDisplayList:
			continue
		case CommandPushOpacity:
			if !finite(command.Opacity) || command.Opacity < 0 || command.Opacity > 1 {
				return fmt.Errorf("paint: command %d has invalid opacity", index)
			}
			opacities = append(opacities, opacities[len(opacities)-1]*command.Opacity)
		case CommandPopOpacity:
			if len(opacities) == 1 {
				return fmt.Errorf("paint: command %d has unbalanced opacity pop", index)
			}
			opacities = opacities[:len(opacities)-1]
		case CommandPushClip:
			if !validRect(command.Rect) {
				return fmt.Errorf("paint: command %d has invalid clip", index)
			}
			clip := command.Rect
			if len(clips) != 0 {
				clip = Intersect(clips[len(clips)-1], clip)
			}
			clips = append(clips, clip)
			if err := painter.SetClip(clip, true); err != nil {
				return fmt.Errorf("paint: command %d push clip: %w", index, err)
			}
		case CommandPopClip:
			if len(clips) == 0 {
				return fmt.Errorf("paint: command %d has unbalanced clip pop", index)
			}
			clips = clips[:len(clips)-1]
			var clip Rect
			set := len(clips) != 0
			if set {
				clip = clips[len(clips)-1]
			}
			if err := painter.SetClip(clip, set); err != nil {
				return fmt.Errorf("paint: command %d pop clip: %w", index, err)
			}
		case CommandDrawShadow:
			shadow := command.Shadow
			shadow.Color = withOpacity(shadow.Color, opacities[len(opacities)-1])
			if err := painter.DrawShadow(command.Rect, command.Radii, shadow); err != nil {
				return fmt.Errorf("paint: command %d shadow: %w", index, err)
			}
		case CommandFillRoundedRect:
			if err := painter.FillRoundedRect(command.Rect, command.Radii, withOpacity(command.Color, opacities[len(opacities)-1]), command.JoinedEdges); err != nil {
				return fmt.Errorf("paint: command %d fill: %w", index, err)
			}
		case CommandStrokeRoundedRect:
			if err := painter.StrokeRoundedRect(command.Rect, command.Radii, command.Width, withOpacity(command.Color, opacities[len(opacities)-1])); err != nil {
				return fmt.Errorf("paint: command %d stroke: %w", index, err)
			}
		case CommandBorder:
			if command.Sides & ^BorderAll != 0 || !validRect(command.Rect) || !finite(command.Width) || command.Width < 0 {
				return fmt.Errorf("paint: command %d has invalid border", index)
			}
			if err := painter.DrawBorder(command.Rect, command.Radii, command.Width, withOpacity(command.Color, opacities[len(opacities)-1]), command.Sides, command.Dashed); err != nil {
				return fmt.Errorf("paint: command %d border: %w", index, err)
			}
		case CommandFillStrokeRoundedRect:
			if err := painter.FillStrokeRoundedRect(command.Rect, command.Radii, command.Width,
				withOpacity(command.Color, opacities[len(opacities)-1]),
				withOpacity(command.BorderColor, opacities[len(opacities)-1])); err != nil {
				return fmt.Errorf("paint: command %d fill/stroke: %w", index, err)
			}
		case CommandDrawText:
			pixels, valid := intPixelBytes(command.Text, 1)
			if !valid || pixels > math.MaxInt || int64(len(command.Text.Pixels)) != pixels {
				return fmt.Errorf("paint: command %d has invalid text bitmap", index)
			}
			if err := painter.DrawText(command.Rect, command.Text, withOpacity(command.Color, opacities[len(opacities)-1])); err != nil {
				return fmt.Errorf("paint: command %d text: %w", index, err)
			}
		case CommandDrawIcon:
			pixels, valid := intPixelBytes(command.Text, 1)
			if !valid || pixels > math.MaxInt || int64(len(command.Text.Pixels)) != pixels {
				return fmt.Errorf("paint: command %d has invalid icon mask", index)
			}
			if err := painter.DrawIcon(command.Rect, command.Text, withOpacity(command.Color, opacities[len(opacities)-1])); err != nil {
				return fmt.Errorf("paint: command %d icon: %w", index, err)
			}
		case CommandDrawImage:
			pixels, valid := imagePixelBytes(command.Image)
			if !valid || pixels > math.MaxInt || int64(len(command.Image.Pixels)) != pixels {
				return fmt.Errorf("paint: command %d has invalid image bitmap", index)
			}
			alpha := withOpacity(Color{A: 255}, opacities[len(opacities)-1]).A
			if command.ImageClip != (Rect{}) && !validRect(command.ImageClip) {
				return fmt.Errorf("paint: command %d has invalid image clip", index)
			}
			if err := painter.DrawImage(command.Rect, command.Image, alpha, command.ImageClip, command.Radii); err != nil {
				return fmt.Errorf("paint: command %d image: %w", index, err)
			}
		default:
			return fmt.Errorf("paint: command %d has unknown kind %d", index, command.Kind)
		}
	}
	if len(opacities) != 1 || len(clips) != 0 {
		return fmt.Errorf("paint: unbalanced display list: opacity depth %d, clip depth %d", len(opacities)-1, len(clips))
	}
	return nil
}

func intPixelBytes(bitmap *TextBitmap, channels int64) (int64, bool) {
	if bitmap == nil || bitmap.Width < 0 || bitmap.Height < 0 || channels <= 0 {
		return 0, false
	}
	width, height := int64(bitmap.Width), int64(bitmap.Height)
	if height != 0 && width > math.MaxInt64/height {
		return 0, false
	}
	pixels := width * height
	if pixels > math.MaxInt64/channels {
		return 0, false
	}
	return pixels * channels, true
}

func imagePixelBytes(bitmap *ImageBitmap) (int64, bool) {
	if bitmap == nil {
		return 0, false
	}
	return intPixelBytes(&TextBitmap{Width: bitmap.Width, Height: bitmap.Height}, 4)
}

func withOpacity(color Color, opacity float32) Color {
	alpha := math.Round(float64(color.A) * float64(opacity))
	alpha = math.Max(0, math.Min(255, alpha))
	color.A = uint8(alpha)
	return color
}

// RecordingPainter is a deterministic fake backend. Records contain effective
// world clips and alpha after Replay composition.
type RecordingPainter struct{ Records DisplayList }

func (recorder *RecordingPainter) SetClip(rect Rect, set bool) error {
	kind := CommandPopClip
	if set {
		kind = CommandPushClip
	}
	recorder.Records = append(recorder.Records, Command{Kind: kind, Rect: rect})
	return nil
}

func (recorder *RecordingPainter) DrawShadow(rect Rect, radii Radii, shadow Shadow) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandDrawShadow, Rect: rect, Radii: radii, Shadow: shadow})
	return nil
}

func (recorder *RecordingPainter) FillRoundedRect(rect Rect, radii Radii, color Color, joined JoinedEdges) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandFillRoundedRect, Rect: rect, Radii: radii, Color: color, JoinedEdges: joined})
	return nil
}

func (recorder *RecordingPainter) StrokeRoundedRect(rect Rect, radii Radii, width float32, color Color) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandStrokeRoundedRect, Rect: rect, Radii: radii, Width: width, Color: color})
	return nil
}

func (recorder *RecordingPainter) DrawBorder(rect Rect, radii Radii, width float32, color Color, sides BorderSides, dashed bool) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandBorder, Rect: rect, Radii: radii, Width: width, Color: color, Sides: sides, Dashed: dashed})
	return nil
}

func (recorder *RecordingPainter) FillStrokeRoundedRect(rect Rect, radii Radii, width float32, fill, border Color) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandFillStrokeRoundedRect, Rect: rect, Radii: radii, Width: width, Color: fill, BorderColor: border})
	return nil
}

func (recorder *RecordingPainter) DrawText(rect Rect, bitmap *TextBitmap, color Color) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandDrawText, Rect: rect, Text: bitmap, Color: color})
	return nil
}

func (recorder *RecordingPainter) DrawIcon(rect Rect, bitmap *TextBitmap, color Color) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandDrawIcon, Rect: rect, Text: bitmap, Color: color})
	return nil
}

func (recorder *RecordingPainter) DrawImage(rect Rect, bitmap *ImageBitmap, alpha uint8, imageClip Rect, radii Radii) error {
	recorder.Records = append(recorder.Records, Command{Kind: CommandDrawImage, Rect: rect, Image: bitmap, Color: Color{A: alpha}, ImageClip: imageClip, Radii: radii})
	return nil
}

// SnapRect applies the one paint-boundary rounding rule: snap both logical
// edges independently, then derive size from the snapped edges.
func SnapRect(rect Rect, scale float32) (PixelRect, error) {
	values := []float32{rect.X, rect.Y, rect.Width, rect.Height, scale}
	for _, value := range values {
		if !finite(value) {
			return PixelRect{}, fmt.Errorf("paint: non-finite rectangle or scale")
		}
	}
	if rect.Width < 0 || rect.Height < 0 || scale <= 0 {
		return PixelRect{}, fmt.Errorf("paint: size must be non-negative and scale positive")
	}
	left := math.Round(float64(rect.X) * float64(scale))
	top := math.Round(float64(rect.Y) * float64(scale))
	right := math.Round((float64(rect.X) + float64(rect.Width)) * float64(scale))
	bottom := math.Round((float64(rect.Y) + float64(rect.Height)) * float64(scale))
	for _, value := range []float64{left, top, right, bottom} {
		if value < math.MinInt32 || value > math.MaxInt32 {
			return PixelRect{}, fmt.Errorf("paint: snapped rectangle exceeds int32 pixels")
		}
	}
	return PixelRect{X: int32(left), Y: int32(top), Width: int32(right - left), Height: int32(bottom - top)}, nil
}

// Sorted returns legacy hit-test items in deterministic stacking order.
// Structured streams are already hierarchy-sorted and are copied unchanged.
func (list DisplayList) Sorted() DisplayList {
	for _, command := range list {
		if command.Kind != CommandItem {
			return list
		}
	}
	result := append(DisplayList(nil), list...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ZIndex != result[j].ZIndex {
			return result[i].ZIndex < result[j].ZIndex
		}
		return result[i].SourceIndex < result[j].SourceIndex
	})
	return result
}

// HitTestOrder returns interactive legacy items from topmost to bottommost.
func (list DisplayList) HitTestOrder() DisplayList {
	ordered := list.Sorted()
	result := make(DisplayList, 0, len(ordered))
	for index := len(ordered) - 1; index >= 0; index-- {
		item := ordered[index]
		if item.Kind == CommandItem && item.Visibility && item.Opacity > 0 && item.Interactive {
			result = append(result, item)
		}
	}
	return result
}

// HitTestAt returns the topmost interactive legacy item containing a point.
func (list DisplayList) HitTestAt(x, y float32) (Item, bool) {
	if !finite(x) || !finite(y) {
		return Item{}, false
	}
	for _, item := range list.HitTestOrder() {
		if contains(item.Bounds, x, y) && (!item.ClipSet || contains(item.Clip, x, y)) {
			return item, true
		}
	}
	return Item{}, false
}

func contains(rect Rect, x, y float32) bool {
	return rect.Width > 0 && rect.Height > 0 && x >= rect.X && y >= rect.Y && x < rect.X+rect.Width && y < rect.Y+rect.Height
}

// Intersect returns the rectangular clip intersection, preserving an explicit
// empty rectangle when the inputs do not overlap.
func Intersect(a, b Rect) Rect {
	left, top := max(a.X, b.X), max(a.Y, b.Y)
	right, bottom := min(a.X+a.Width, b.X+b.Width), min(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: left, Y: top, Width: max(0, right-left), Height: max(0, bottom-top)}
}

// Union returns the smallest rectangle containing both inputs.
func Union(a, b Rect) Rect {
	if a.Width <= 0 || a.Height <= 0 {
		return b
	}
	if b.Width <= 0 || b.Height <= 0 {
		return a
	}
	left, top := min(a.X, b.X), min(a.Y, b.Y)
	right, bottom := max(a.X+a.Width, b.X+b.Width), max(a.Y+a.Height, b.Y+b.Height)
	return Rect{X: left, Y: top, Width: right - left, Height: bottom - top}
}

func validRect(value Rect) bool {
	return finite(value.X) && finite(value.Y) && finite(value.Width) && finite(value.Height) && value.Width >= 0 && value.Height >= 0
}

func finite(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
