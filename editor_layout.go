package dxui

import (
	"math"
	"strings"
	"unicode"

	internalinput "github.com/dxui-org/dxui/internal/input"
	internaltext "github.com/dxui-org/dxui/internal/text"
)

type editorVisualLine struct {
	start, end int
	y, height  float32
	x          []float32
}

type editorVisualLayout struct {
	lines         []editorVisualLine
	width, height float32
}

func (a *App) layoutEditor(action inputAction) (editorVisualLayout, error) {
	engine, err := a.textEngine()
	if err != nil {
		return editorVisualLayout{}, err
	}
	a.mu.Lock()
	theme := a.theme
	a.mu.Unlock()
	return layoutEditorWith(engine, theme, action)
}

func layoutEditorWith(engine *internaltext.Engine, theme resolvedTheme, action inputAction) (editorVisualLayout, error) {
	if !action.multiline {
		action.value = strings.NewReplacer("\r", " ", "\n", " ").Replace(action.value)
	}
	if action.password && !action.passwordVisible {
		action.value = strings.Repeat("•", len([]rune(action.value)))
	}
	request, err := resolvedTextRequest(textPropsFromCommon(action.node, action.value, TextNoWrap, 0), theme)
	if err != nil {
		return editorVisualLayout{}, err
	}
	request.Wrap = internaltext.NoWrap
	request.MaxLines = 0
	runes := []rune(action.value)
	paragraphStart := 0
	result := editorVisualLayout{}
	paragraphs := strings.Split(action.value, "\n")
	for paragraphIndex, paragraph := range paragraphs {
		request.Text = paragraph
		laidOut, layoutErr := engine.Layout(request)
		if layoutErr != nil {
			return editorVisualLayout{}, layoutErr
		}
		var glyphs []internaltext.Glyph
		lineHeight := request.LineHeight
		if len(laidOut.Lines) != 0 {
			glyphs = laidOut.Lines[0].Glyphs
			lineHeight = laidOut.Lines[0].Height
		}
		if lineHeight <= 0 {
			lineHeight = 14
		}
		if len(glyphs) == 0 {
			result.lines = append(result.lines, editorVisualLine{start: paragraphStart, end: paragraphStart, y: result.height, height: lineHeight, x: []float32{0}})
			result.height += lineHeight
		} else {
			segment := 0
			for segment < len(glyphs) {
				end := len(glyphs)
				if action.multiline && action.wrap == TextWrapWords && action.content.Width > 0 {
					lastSpace := -1
					base := glyphs[segment].X
					for i := segment; i < len(glyphs); i++ {
						if unicode.IsSpace(glyphs[i].OriginalRune) {
							lastSpace = i + 1
						}
						if glyphs[i].X+glyphs[i].Advance-base > action.content.Width && i > segment {
							end = i
							if lastSpace > segment {
								end = lastSpace
							}
							break
						}
					}
				}
				base := glyphs[segment].X
				xs := make([]float32, end-segment+1)
				for i := segment; i < end; i++ {
					xs[i-segment] = glyphs[i].X - base
				}
				xs[len(xs)-1] = glyphs[end-1].X + glyphs[end-1].Advance - base
				line := editorVisualLine{start: paragraphStart + segment, end: paragraphStart + end, y: result.height, height: lineHeight, x: xs}
				result.lines = append(result.lines, line)
				result.width = max(result.width, xs[len(xs)-1])
				result.height += lineHeight
				segment = end
			}
		}
		paragraphStart += len([]rune(paragraph))
		if paragraphIndex != len(paragraphs)-1 {
			paragraphStart++
		}
	}
	if len(result.lines) == 0 {
		result.lines = []editorVisualLine{{x: []float32{0}, height: 14}}
		result.height = 14
	}
	_ = runes
	return result, nil
}

func editorCaret(layout editorVisualLayout, index int) (x, y, height float32) {
	for _, line := range layout.lines {
		if index >= line.start && index <= line.end {
			offset := min(max(index-line.start, 0), len(line.x)-1)
			return line.x[offset], line.y, line.height
		}
	}
	line := layout.lines[len(layout.lines)-1]
	return line.x[len(line.x)-1], line.y, line.height
}

func editorIndexAt(layout editorVisualLayout, x, y float32) int {
	line := layout.lines[0]
	for _, candidate := range layout.lines {
		line = candidate
		if y < candidate.y+candidate.height {
			break
		}
	}
	for i := 0; i+1 < len(line.x); i++ {
		if x < (line.x[i]+line.x[i+1])/2 {
			return line.start + i
		}
	}
	return line.end
}

func editorVertical(layout editorVisualLayout, index, delta int, preferred float32) int {
	lineIndex := 0
	for i, line := range layout.lines {
		if index >= line.start && index <= line.end {
			lineIndex = i
			break
		}
	}
	target := min(max(lineIndex+delta, 0), len(layout.lines)-1)
	line := layout.lines[target]
	return editorIndexAt(layout, preferred, line.y+line.height/2)
}

func (a *App) ensureEditorCaretVisible(action inputAction, editor *retainedEditor) error {
	layout, err := a.layoutEditor(action)
	if err != nil {
		return err
	}
	x, y, h := editorCaret(layout, editor.state.Selection.End)
	viewportW, viewportH := action.content.Width, action.content.Height
	if !action.multiline {
		if x-editor.state.ScrollX < 0 {
			editor.state.ScrollX = x
		}
		if x-editor.state.ScrollX > viewportW-1 {
			editor.state.ScrollX = max(0, x-viewportW+1)
		}
		editor.state.ScrollY = 0
	} else {
		if y-editor.state.ScrollY < 0 {
			editor.state.ScrollY = y
		}
		if y+h-editor.state.ScrollY > viewportH {
			editor.state.ScrollY = max(0, y+h-viewportH)
		}
		if action.wrap == TextNoWrap {
			if x-editor.state.ScrollX < 0 {
				editor.state.ScrollX = x
			}
			if x-editor.state.ScrollX > viewportW-1 {
				editor.state.ScrollX = max(0, x-viewportW+1)
			}
		} else {
			editor.state.ScrollX = 0
		}
	}
	clampEditorScroll(action, editor, layout)
	return nil
}

func clampEditorScroll(action inputAction, editor *retainedEditor, layout editorVisualLayout) {
	maximumX := max(float32(0), layout.width-action.content.Width+1)
	maximumY := max(float32(0), layout.height-action.content.Height)
	if action.multiline && action.wrap != TextNoWrap {
		maximumX = 0
	}
	if !action.multiline {
		maximumY = 0
	}
	if math.IsNaN(float64(editor.state.ScrollX)) || math.IsInf(float64(editor.state.ScrollX), 0) {
		editor.state.ScrollX = 0
	}
	if math.IsNaN(float64(editor.state.ScrollY)) || math.IsInf(float64(editor.state.ScrollY), 0) {
		editor.state.ScrollY = 0
	}
	editor.state.ScrollX = min(max(float32(0), editor.state.ScrollX), maximumX)
	editor.state.ScrollY = min(max(float32(0), editor.state.ScrollY), maximumY)
}

func toInternalRange(value TextRange) internalinput.Range {
	return internalinput.Range{Start: value.Start, End: value.End}
}
func publicRange(value internalinput.Range) TextRange {
	return TextRange{Start: value.Start, End: value.End}
}
