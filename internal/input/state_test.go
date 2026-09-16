package input

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestProposeInsertUsesRuneIndicesAndDoesNotMutateValue(t *testing.T) {
	state := State{Selection: Range{Start: 1, End: 2}, Composition: Composition{Text: "候选", Active: true}}
	change := state.ProposeInsert("A中B", "文")
	if change.Value != "A文B" || change.Selection != (Range{Start: 2, End: 2}) {
		t.Fatalf("change = %#v", change)
	}
	if state.Composition.Text != "候选" {
		t.Fatal("proposal cleared composition before commit")
	}
	state.Commit(change)
	if state.Composition.Active {
		t.Fatal("commit did not clear composition")
	}
}

func TestEditingMatrixNormalizesUTF8AndNeverSplitsRunes(t *testing.T) {
	tests := []struct {
		name, value, insert, want string
		selection                 Range
	}{
		{name: "ASCII", value: "abc", insert: "X", selection: Range{1, 2}, want: "aXc"},
		{name: "Chinese", value: "甲乙丙", insert: "文", selection: Range{1, 2}, want: "甲文丙"},
		{name: "reversed selection", value: "A中B", insert: "文", selection: Range{2, 1}, want: "A文B"},
		{name: "invalid UTF-8", value: "a\xffb", insert: "\xfe", selection: Range{1, 2}, want: "a�b"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := State{Selection: test.selection}
			change, err := state.Insert(test.value, test.insert, ReasonTyping)
			if err != nil {
				t.Fatal(err)
			}
			if change.Value != test.want || !utf8.ValidString(change.Value) {
				t.Fatalf("change = %#v", change)
			}
		})
	}
}

func TestDeleteMoveCompositionAndBoundedUndo(t *testing.T) {
	state := State{Selection: Range{2, 2}}
	back, ok := state.DeleteBackward("A中B")
	if !ok || back.Value != "AB" || back.Selection != (Range{1, 1}) {
		t.Fatalf("backspace = %#v/%v", back, ok)
	}
	state.AcceptControlled(back.Value, back.Selection, true)
	state.Selection = Range{1, 1}
	forward, ok := state.DeleteForward("AB")
	if !ok || forward.Value != "A" {
		t.Fatalf("delete = %#v/%v", forward, ok)
	}
	state.SetComposition("候选", Range{1, 2})
	if !state.Composition.Active || state.Composition.Selection != (Range{1, 2}) {
		t.Fatalf("composition = %+v", state.Composition)
	}
	state.SetComposition("")
	if state.Composition.Active {
		t.Fatal("composition cancel failed")
	}
	undo, ok := state.Undo("AB")
	if !ok || undo.Value != "A中B" || undo.Reason != ReasonUndo {
		t.Fatalf("undo = %#v/%v", undo, ok)
	}
}

func TestMaximumTextGuardPrecedesLargeAllocation(t *testing.T) {
	state := State{Selection: Range{MaxTextRunes, MaxTextRunes}}
	_, err := state.Insert(strings.Repeat("a", MaxTextRunes), "b", ReasonPaste)
	if err != ErrTextTooLarge {
		t.Fatalf("limit error = %v", err)
	}
}

func TestRejectedValueClampsRetainedCaret(t *testing.T) {
	state := State{Selection: Range{Start: 9, End: 9}}
	state.SetControlledValue("中A")
	if state.Selection != (Range{Start: 2, End: 2}) {
		t.Fatalf("selection = %#v", state.Selection)
	}
}
