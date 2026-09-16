package icon

import (
	"testing"
)

func TestMetadataAndAlias(t *testing.T) {
	if Version != "1.41.0" || Count != 2066 || CanonicalCount != 1807 {
		t.Fatalf("metadata = %s/%d/%d", Version, Count, CanonicalCount)
	}
	canonical, alias := SquareActivity(), ActivitySquare()
	if !canonical.IsPacked() || canonical.Identity() != alias.Identity() {
		t.Fatal("alias does not reuse canonical immutable data")
	}
	if canonical.Commands != nil || alias.Commands != nil {
		t.Fatal("generated icon exposes a mutable command slice")
	}
}

func TestConstructorAllocations(t *testing.T) {
	if allocations := testing.AllocsPerRun(1000, func() { _ = Search() }); allocations != 0 {
		t.Fatalf("Search allocations = %g, want 0", allocations)
	}
}

func BenchmarkSearchDataConstructor(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Search()
	}
}
