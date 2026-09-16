// Package input contains backend-independent controlled editing state.
package input

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	// MaxTextRunes bounds one editor value before allocating replacement
	// buffers. Larger documents need a virtualized editor.
	MaxTextRunes = 1 << 20
	// Undo and redo share one per-editor budget. Accounting includes cloned
	// UTF-8 payload plus a conservative 64-bit historyEntry footprint.
	maxHistoryEntries    = 100
	historyByteBudget    = 4 << 20
	historyEntryOverhead = 32
)

var ErrTextTooLarge = errors.New("input: text exceeds the 1,048,576-rune limit")

// Range uses Unicode code-point (rune) offsets. Start is the selection anchor
// and End is the active/caret end, so a range may be reversed.
type Range struct{ Start, End int }

// Reason identifies the operation that proposed a controlled value.
type Reason uint8

const (
	ReasonTyping Reason = iota
	ReasonPaste
	ReasonCut
	ReasonDelete
	ReasonIMECommit
	ReasonUndo
	ReasonRedo
)

// Change is a complete proposed controlled value and selection.
type Change struct {
	Value     string
	Selection Range
	Reason    Reason
}

// Composition is uncommitted native pre-edit state. Selection is relative to
// Text and is {-1,-1} when the platform did not report it.
type Composition struct {
	Text      string
	Selection Range
	Active    bool
}

type historyEntry struct {
	value     string
	selection Range
}

// State retains selection, composition, scrolling and bounded undo history
// separately from the authoritative controlled Value.
type State struct {
	Selection        Range
	Composition      Composition
	ScrollX, ScrollY float32
	PreferredX       float32
	PreferredXSet    bool
	undo, redo       []historyEntry
	undoBytes        int
	redoBytes        int
	pending          *historyEntry
}

// Normalize converts malformed UTF-8 to U+FFFD before rune indexing.
func Normalize(value string) string { return strings.ToValidUTF8(value, "\uFFFD") }

// SetControlledValue clamps retained state after an accepted or rejected
// rebuild. An optional controlled selection replaces the retained one.
func (s *State) SetControlledValue(value string, controlled ...Range) {
	length := utf8.RuneCountInString(Normalize(value))
	if len(controlled) != 0 {
		s.Selection = controlled[0]
	}
	s.Selection = ClampRange(s.Selection, length)
}

// ProposeInsert creates an edit without mutating the authoritative value.
func (s *State) ProposeInsert(value, inserted string) Change {
	change, _ := s.Insert(value, inserted, ReasonTyping)
	return change
}

// Insert replaces the current selection at rune boundaries.
func (s *State) Insert(value, inserted string, reason Reason) (Change, error) {
	value, inserted = Normalize(value), Normalize(inserted)
	runes, addition := []rune(value), []rune(inserted)
	start, end := Ordered(ClampRange(s.Selection, len(runes)))
	if len(runes)-(end-start)+len(addition) > MaxTextRunes {
		return Change{}, ErrTextTooLarge
	}
	result := make([]rune, 0, len(runes)-(end-start)+len(addition))
	result = append(result, runes[:start]...)
	result = append(result, addition...)
	result = append(result, runes[end:]...)
	caret := start + len(addition)
	change := Change{Value: string(result), Selection: Range{caret, caret}, Reason: reason}
	s.recordProposal(value, s.Selection)
	return change, nil
}

// DeleteBackward and DeleteForward delete a selection or one code point.
// MVP cursor semantics are rune-based, not grapheme-cluster based.
func (s *State) DeleteBackward(value string) (Change, bool) {
	value = Normalize(value)
	runes := []rune(value)
	start, end := Ordered(ClampRange(s.Selection, len(runes)))
	if start == end {
		if start == 0 {
			return Change{}, false
		}
		start--
	}
	s.Selection = Range{start, end}
	change, err := s.Insert(value, "", ReasonDelete)
	return change, err == nil
}

func (s *State) DeleteForward(value string) (Change, bool) {
	value = Normalize(value)
	runes := []rune(value)
	start, end := Ordered(ClampRange(s.Selection, len(runes)))
	if start == end {
		if end == len(runes) {
			return Change{}, false
		}
		end++
	}
	s.Selection = Range{start, end}
	change, err := s.Insert(value, "", ReasonDelete)
	return change, err == nil
}

// MoveHorizontal moves by code point and optionally extends from the anchor.
func (s *State) MoveHorizontal(value string, delta int, extend bool) bool {
	length := utf8.RuneCountInString(Normalize(value))
	old := s.Selection
	selection := ClampRange(old, length)
	if !extend && selection.Start != selection.End {
		start, end := Ordered(selection)
		if delta < 0 {
			selection = Range{start, start}
		} else {
			selection = Range{end, end}
		}
	} else {
		active := clampIndex(selection.End+delta, length)
		if extend {
			selection.End = active
		} else {
			selection = Range{active, active}
		}
	}
	s.Selection = selection
	s.PreferredXSet = false
	return old != selection
}

// MoveLineBoundary moves to a logical newline-delimited line boundary.
func (s *State) MoveLineBoundary(value string, end, extend bool) bool {
	runes := []rune(Normalize(value))
	old := s.Selection
	caret := clampIndex(s.Selection.End, len(runes))
	if end {
		for caret < len(runes) && runes[caret] != '\n' {
			caret++
		}
	} else {
		for caret > 0 && runes[caret-1] != '\n' {
			caret--
		}
	}
	if extend {
		s.Selection.End = caret
	} else {
		s.Selection = Range{caret, caret}
	}
	s.PreferredXSet = false
	return old != s.Selection
}

func (s *State) SelectAll(value string) bool {
	next := Range{0, utf8.RuneCountInString(Normalize(value))}
	changed := s.Selection != next
	s.Selection = next
	return changed
}

func (s *State) Selected(value string) string {
	runes := []rune(Normalize(value))
	start, end := Ordered(ClampRange(s.Selection, len(runes)))
	return string(runes[start:end])
}

// SetComposition replaces native pre-edit state. Empty text is cancellation.
func (s *State) SetComposition(value string, selection ...Range) {
	value = Normalize(value)
	if value == "" {
		s.Composition = Composition{}
		return
	}
	r := Range{-1, -1}
	if len(selection) != 0 && selection[0].Start >= 0 && selection[0].End >= 0 {
		r = ClampRange(selection[0], utf8.RuneCountInString(value))
	}
	s.Composition = Composition{Text: value, Selection: r, Active: true}
}

func (s *State) CancelComposition() bool {
	active := s.Composition.Active
	s.Composition = Composition{}
	return active
}

// Commit updates retained state; the next build still accepts or rejects Value.
func (s *State) Commit(change Change) {
	s.Selection = change.Selection
	s.Composition = Composition{}
	s.PreferredXSet = false
}

// AcceptControlled finalizes pending undo only after an application rebuild.
func (s *State) AcceptControlled(value string, selection Range, accepted bool) {
	if accepted && s.pending != nil {
		clearHistory(&s.redo, &s.redoBytes)
		pushHistory(&s.undo, &s.undoBytes, len(s.redo), s.redoBytes, *s.pending)
	}
	s.pending = nil
	s.Selection = selection
	s.SetControlledValue(value)
}

func (s *State) Undo(value string) (Change, bool) {
	entry, ok := popHistory(&s.undo, &s.undoBytes)
	if !ok {
		return Change{}, false
	}
	pushHistory(&s.redo, &s.redoBytes, len(s.undo), s.undoBytes, historyEntry{Normalize(value), s.Selection})
	return Change{Value: entry.value, Selection: entry.selection, Reason: ReasonUndo}, true
}

func (s *State) Redo(value string) (Change, bool) {
	entry, ok := popHistory(&s.redo, &s.redoBytes)
	if !ok {
		return Change{}, false
	}
	pushHistory(&s.undo, &s.undoBytes, len(s.redo), s.redoBytes, historyEntry{Normalize(value), s.Selection})
	return Change{Value: entry.value, Selection: entry.selection, Reason: ReasonRedo}, true
}

func (s *State) recordProposal(value string, selection Range) {
	entry := historyEntry{value: Normalize(value), selection: selection}
	s.pending = &entry
}

func historyEntryBytes(value historyEntry) int {
	return len(value.value) + historyEntryOverhead
}

func pushHistory(values *[]historyEntry, byteCount *int, otherEntries, otherBytes int, value historyEntry) {
	cost := historyEntryBytes(value)
	if cost > historyByteBudget-otherBytes {
		clearHistory(values, byteCount)
		return
	}
	value.value = strings.Clone(value.value)
	*values = append(*values, value)
	*byteCount += cost
	for len(*values)+otherEntries > maxHistoryEntries || *byteCount+otherBytes > historyByteBudget {
		dropOldestHistory(values, byteCount)
	}
}

func popHistory(values *[]historyEntry, byteCount *int) (historyEntry, bool) {
	if len(*values) == 0 {
		return historyEntry{}, false
	}
	last := len(*values) - 1
	value := (*values)[last]
	(*values)[last] = historyEntry{}
	*values = (*values)[:last]
	*byteCount -= historyEntryBytes(value)
	return value, true
}

func dropOldestHistory(values *[]historyEntry, byteCount *int) {
	if len(*values) == 0 {
		return
	}
	*byteCount -= historyEntryBytes((*values)[0])
	copy(*values, (*values)[1:])
	last := len(*values) - 1
	(*values)[last] = historyEntry{}
	*values = (*values)[:last]
}

func clearHistory(values *[]historyEntry, byteCount *int) {
	if cap(*values) != 0 {
		clear((*values)[:cap(*values)])
	}
	*values = nil
	*byteCount = 0
}

func ClampRange(value Range, length int) Range {
	return Range{clampIndex(value.Start, length), clampIndex(value.End, length)}
}

func Ordered(value Range) (int, int) {
	if value.Start <= value.End {
		return value.Start, value.End
	}
	return value.End, value.Start
}

func clampIndex(index, length int) int {
	if index < 0 {
		return 0
	}
	if index > length {
		return length
	}
	return index
}
