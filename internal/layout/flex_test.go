package layout

import "testing"

func TestFlexLineGrowFreezesAtMax(t *testing.T) {
	items := []FlexItem{
		{Basis: 20, Min: 0, Max: 25, Grow: 1, Shrink: 1},
		{Basis: 20, Min: 0, Max: 100, Grow: 1, Shrink: 1},
	}
	got, err := FlexLine(80, 0, JustifyStart, items)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Size != 25 || got[1].Size != 55 {
		t.Fatalf("sizes = %v, %v; want 25, 55", got[0].Size, got[1].Size)
	}
}

func TestFlexLineShrinkFreezesAtMin(t *testing.T) {
	items := []FlexItem{
		{Basis: 60, Min: 50, Max: 100, Shrink: 1},
		{Basis: 40, Min: 0, Max: 100, Shrink: 1},
	}
	got, err := FlexLine(70, 0, JustifyStart, items)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Size != 50 || got[1].Size != 20 {
		t.Fatalf("sizes = %v, %v; want 50, 20", got[0].Size, got[1].Size)
	}
}

func TestFlexLineAbsoluteDoesNotConsumeGap(t *testing.T) {
	items := []FlexItem{
		{Basis: 10, Max: 10},
		{Basis: 100, Max: 100, Absolute: true},
		{Basis: 10, Max: 10},
	}
	got, err := FlexLine(30, 10, JustifyStart, items)
	if err != nil {
		t.Fatal(err)
	}
	if got[2].Position != 20 {
		t.Fatalf("second flow position = %v, want 20", got[2].Position)
	}
}

func TestFlexLineInternalOverlapSharesAdjacentEdge(t *testing.T) {
	items := []FlexItem{
		{Basis: 40, Max: 40},
		{Basis: 40, Max: 40},
		{Basis: 40, Max: 40},
	}
	got, err := flexLine(118, 0, 1, JustifyStart, items)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Position != 0 || got[1].Position != 39 || got[2].Position != 78 {
		t.Fatalf("overlap positions = %v/%v/%v, want 0/39/78", got[0].Position, got[1].Position, got[2].Position)
	}
}

func TestFlexLineRaisesConflictingMaxToMin(t *testing.T) {
	got, err := FlexLine(20, 0, JustifyStart, []FlexItem{{Basis: 5, Min: 10, Max: 8}})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Size != 10 {
		t.Fatalf("size = %v, want raised minimum 10", got[0].Size)
	}
}
