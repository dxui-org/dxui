package input

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"
)

const (
	historyPressureEdits     = 160
	historyPressureValueSize = 64 << 10
)

type historyReferenceSnapshot struct {
	activeEntries, backingEntries int
	activeBytes, backingBytes     int
}

type stateBeforeHistoryAccounting struct {
	Selection        Range
	Composition      Composition
	ScrollX, ScrollY float32
	PreferredX       float32
	PreferredXSet    bool
	undo, redo       []historyEntry
	pending          *historyEntry
}

func snapshotHistoryReferences(values []historyEntry) historyReferenceSnapshot {
	result := historyReferenceSnapshot{activeEntries: len(values)}
	for index, entry := range values[:cap(values)] {
		if entry.value == "" {
			continue
		}
		result.backingEntries++
		result.backingBytes += len(entry.value)
		if index < len(values) {
			result.activeBytes += len(entry.value)
		}
	}
	return result
}

func historyPressureValues() []string {
	values := make([]string, historyPressureEdits+2)
	for index := range values {
		prefix := fmt.Sprintf("%06d:", index)
		values[index] = prefix + strings.Repeat("x", historyPressureValueSize-len(prefix))
	}
	return values
}

func applyAcceptedHistoryProposal(state *State, before, after string) {
	state.recordProposal(before, Range{})
	state.AcceptControlled(after, Range{}, true)
}

func TestStateHistoryAccountingStorage(t *testing.T) {
	before := unsafe.Sizeof(stateBeforeHistoryAccounting{})
	after := unsafe.Sizeof(State{})
	if after < before {
		t.Fatalf("State storage shrank unexpectedly: before=%d after=%d", before, after)
	}
	t.Logf("State storage: before=%d after=%d delta=%d bytes", before, after, after-before)
}

func TestEditorHistoryMemoryPressure(t *testing.T) {
	values := historyPressureValues()
	var state State
	for index := 0; index < historyPressureEdits; index++ {
		applyAcceptedHistoryProposal(&state, values[index], values[index+1])
	}
	wantEntries := historyByteBudget / (historyPressureValueSize + historyEntryOverhead)
	if got := len(state.undo); got != wantEntries {
		t.Fatalf("entries after pressure = %d, want %d", got, wantEntries)
	}
	if got := state.undoBytes + state.redoBytes; got > historyByteBudget {
		t.Fatalf("history accounting = %d, budget %d", got, historyByteBudget)
	}

	current := values[historyPressureEdits]
	undos := 0
	for {
		change, ok := state.Undo(current)
		if !ok {
			break
		}
		current = change.Value
		undos++
	}
	if undos != wantEntries || current != values[historyPressureEdits-wantEntries] {
		t.Fatalf("undo window = %d ending at value %q", undos, current[:7])
	}
	applyAcceptedHistoryProposal(&state, current, "branch")

	undo := snapshotHistoryReferences(state.undo)
	redo := snapshotHistoryReferences(state.redo)
	if undo.backingEntries != undo.activeEntries || undo.backingBytes != undo.activeBytes {
		t.Fatalf("undo backing array retains inactive values: %+v", undo)
	}
	if redo != (historyReferenceSnapshot{}) {
		t.Fatalf("cleared redo retains backing references: %+v", redo)
	}
	if state.undoBytes != historyEntryBytes(state.undo[0]) {
		t.Fatalf("undo accounting = %d, want %d", state.undoBytes, historyEntryBytes(state.undo[0]))
	}
	t.Logf("budget=%d bytes; retained undo=%d entries/%d accounted bytes/%d payload bytes; redo backing references=%d",
		historyByteBudget, undo.activeEntries, state.undoBytes, undo.activeBytes, redo.backingEntries)
}

func TestOversizedHistoryEntryBreaksUndoChain(t *testing.T) {
	var state State
	applyAcceptedHistoryProposal(&state, "first", "second")
	oversized := strings.Repeat("😀", MaxTextRunes)
	applyAcceptedHistoryProposal(&state, oversized, "replacement")
	if len(state.undo) != 0 || state.undoBytes != 0 {
		t.Fatalf("oversized snapshot left skippable undo history: entries=%d bytes=%d", len(state.undo), state.undoBytes)
	}
	if _, ok := state.Undo("replacement"); ok {
		t.Fatal("oversized unretained snapshot allowed undo to skip a state")
	}
}

func TestHistoryBudgetKeepsNearestValuesAndSelections(t *testing.T) {
	values := make([]string, 6)
	for index := range values {
		values[index] = fmt.Sprintf("%d%s", index, strings.Repeat("x", 1<<20))
	}
	var state State
	for index := 0; index < len(values)-1; index++ {
		state.Selection = Range{index, index}
		state.recordProposal(values[index], state.Selection)
		state.AcceptControlled(values[index+1], Range{index + 1, index + 1}, true)
	}
	if len(state.undo) != 3 {
		t.Fatalf("budgeted undo entries = %d, want 3", len(state.undo))
	}

	current := values[5]
	for _, want := range []int{4, 3, 2} {
		change, ok := state.Undo(current)
		if !ok || change.Value != values[want] || change.Selection != (Range{want, want}) {
			t.Fatalf("undo to %d = %#v/%v", want, change, ok)
		}
		state.Commit(change)
		current = change.Value
	}
	if _, ok := state.Undo(current); ok {
		t.Fatal("undo reached an entry evicted by the byte budget")
	}
	redo, ok := state.Redo(current)
	if !ok || redo.Value != values[3] || redo.Selection != (Range{3, 3}) {
		t.Fatalf("redo = %#v/%v", redo, ok)
	}
}

func TestHistoryCountLimitKeepsNearestEntries(t *testing.T) {
	var state State
	for index := 0; index < 150; index++ {
		before := fmt.Sprintf("value-%03d", index)
		after := fmt.Sprintf("value-%03d", index+1)
		applyAcceptedHistoryProposal(&state, before, after)
	}
	if len(state.undo) != maxHistoryEntries || state.undo[0].value != "value-050" || state.undo[maxHistoryEntries-1].value != "value-149" {
		t.Fatalf("count-limited undo window = %d [%q, %q]", len(state.undo), state.undo[0].value, state.undo[maxHistoryEntries-1].value)
	}
}

func TestRejectedControlledProposalPreservesRedo(t *testing.T) {
	var state State
	state.Selection = Range{0, 0}
	state.recordProposal("A", state.Selection)
	state.AcceptControlled("AB", Range{2, 2}, true)
	undo, ok := state.Undo("AB")
	if !ok {
		t.Fatal("undo unavailable")
	}
	state.Commit(undo)
	proposal, err := state.Insert(undo.Value, "X", ReasonTyping)
	if err != nil {
		t.Fatal(err)
	}
	state.Commit(proposal)
	state.AcceptControlled(undo.Value, undo.Selection, false)
	redo, ok := state.Redo(undo.Value)
	if !ok || redo.Value != "AB" || redo.Selection != (Range{2, 2}) {
		t.Fatalf("redo after rejected proposal = %#v/%v", redo, ok)
	}
}

func BenchmarkEditorHistoryPressure(b *testing.B) {
	values := historyPressureValues()
	b.ReportAllocs()
	b.ResetTimer()
	var final State
	for iteration := 0; iteration < b.N; iteration++ {
		var state State
		for index := 0; index < historyPressureEdits; index++ {
			applyAcceptedHistoryProposal(&state, values[index], values[index+1])
		}
		current := values[historyPressureEdits]
		for {
			change, ok := state.Undo(current)
			if !ok {
				break
			}
			current = change.Value
		}
		applyAcceptedHistoryProposal(&state, current, "branch")
		final = state
	}
	b.StopTimer()
	refs := snapshotHistoryReferences(final.undo)
	b.ReportMetric(float64(final.undoBytes+final.redoBytes), "accounted-B")
	b.ReportMetric(float64(refs.backingBytes), "backing-ref-B")
}
