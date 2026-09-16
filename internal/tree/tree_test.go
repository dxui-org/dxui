package tree

import (
	"fmt"
	"testing"
)

func TestKeyedChildrenRetainIdentityWhenReordered(t *testing.T) {
	var retained Tree
	first := Description{Kind: 1, Children: []Description{{Kind: 2, Key: "a"}, {Kind: 2, Key: "b"}}}
	if err := retained.Commit(first); err != nil {
		t.Fatal(err)
	}
	aID, bID := retained.Root().Children[0].ID, retained.Root().Children[1].ID
	second := Description{Kind: 1, Children: []Description{{Kind: 2, Key: "b"}, {Kind: 2, Key: "a"}}}
	if err := retained.Commit(second); err != nil {
		t.Fatal(err)
	}
	if retained.Root().Children[0].ID != bID || retained.Root().Children[1].ID != aID {
		t.Fatal("keyed identity did not follow keys")
	}
}

func TestInsertDeleteReplaceAndRetainedState(t *testing.T) {
	var retained Tree
	first := Description{Kind: 1, Children: []Description{
		{Kind: 2, Key: "a"},
		{Kind: 2, Key: "b"},
	}}
	if _, err := retained.Update(first); err != nil {
		t.Fatal(err)
	}
	retained.ClearDirty()
	a := retained.Root().Children[0]
	b := retained.Root().Children[1]
	a.State.Focused = true
	a.State.Slots = []StateSlot{{Kind: StateText, Text: "retained"}}

	second := Description{Kind: 1, Children: []Description{
		{Kind: 2, Key: "new"},
		{Kind: 2, Key: "b"},
		{Kind: 2, Key: "a"},
	}}
	changes, err := retained.Update(second)
	if err != nil {
		t.Fatal(err)
	}
	if changes.Added != 1 || changes.Reused != 3 || changes.Removed != 0 {
		t.Fatalf("changes = %+v", changes)
	}
	if retained.Root().Children[1].ID != b.ID || retained.Root().Children[2].ID != a.ID {
		t.Fatal("keyed state did not follow identity through insertion/reorder")
	}
	got := retained.Root().Children[2].State
	if !got.Focused || len(got.Slots) != 1 || got.Slots[0].Text != "retained" {
		t.Fatalf("retained state = %+v", got)
	}

	retained.ClearDirty()
	replacement := second
	replacement.Children = append([]Description(nil), second.Children...)
	replacement.Children[2].Kind = 3
	changes, err = retained.Update(replacement)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Root().Children[2].ID == a.ID || retained.Root().Children[2].State.Focused {
		t.Fatal("type replacement retained old identity/state")
	}
	if changes.Added != 1 || changes.Removed != 1 {
		t.Fatalf("replacement changes = %+v", changes)
	}
}

type recordingResource struct {
	name  string
	order *[]string
}

type countingResource struct{ released *int }

func (resource countingResource) Release() { *resource.released++ }

func TestIdenticalUpdateReusesTreeWithoutCopyingStateOrResources(t *testing.T) {
	description := Description{Kind: 1, Children: []Description{{Kind: 2, Key: "stable"}}}
	var retained Tree
	if _, err := retained.Update(description); err != nil {
		t.Fatal(err)
	}
	root, child := retained.Root(), retained.Root().Children[0]
	child.State.SelectionStart = 7
	released := 0
	child.AddResource(countingResource{released: &released})
	changes, candidate, err := retained.Preview(description)
	if err != nil {
		t.Fatal(err)
	}
	if candidate != root || changes.Reused != 2 || changes.Dirty != DirtyBuild {
		t.Fatalf("preview = candidate:%p root:%p changes:%+v", candidate, root, changes)
	}
	changes, err = retained.Update(description)
	if err != nil {
		t.Fatal(err)
	}
	if retained.Root() != root || retained.Root().Children[0] != child || child.State.SelectionStart != 7 || child.ResourceCount() != 1 || released != 0 {
		t.Fatalf("identical update changed retained ownership/state: changes=%+v released=%d", changes, released)
	}
}

func (resource recordingResource) Release() {
	*resource.order = append(*resource.order, resource.name)
}

func TestUnmountReleasesDescendantsBeforeParentsAndOnlyOnce(t *testing.T) {
	var retained Tree
	description := Description{Kind: 1, Children: []Description{{Kind: 2, Children: []Description{{Kind: 3}}}}}
	if err := retained.Commit(description); err != nil {
		t.Fatal(err)
	}
	var order []string
	root := retained.Root()
	root.AddResource(recordingResource{"root", &order})
	root.Children[0].AddResource(recordingResource{"child", &order})
	root.Children[0].Children[0].AddResource(recordingResource{"grandchild", &order})

	retained.Unmount()
	retained.Unmount()
	if got, want := fmt.Sprint(order), "[grandchild child root]"; got != want {
		t.Fatalf("release order = %s, want %s", got, want)
	}
}

func TestReusedResourceTransfersAndRemovedResourceReleases(t *testing.T) {
	var retained Tree
	first := Description{Kind: 1, Children: []Description{{Kind: 2, Key: "keep"}, {Kind: 2, Key: "drop"}}}
	if err := retained.Commit(first); err != nil {
		t.Fatal(err)
	}
	var order []string
	retained.Root().Children[0].AddResource(recordingResource{"keep", &order})
	retained.Root().Children[1].AddResource(recordingResource{"drop", &order})
	keepID := retained.Root().Children[0].ID
	if err := retained.Commit(Description{Kind: 1, Children: []Description{{Kind: 2, Key: "keep"}}}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(order) != "[drop]" {
		t.Fatalf("release after delete = %v", order)
	}
	if retained.Root().Children[0].ID != keepID || retained.Root().Children[0].ResourceCount() != 1 {
		t.Fatal("matched resource ownership was not transferred")
	}
	retained.Unmount()
	if fmt.Sprint(order) != "[drop keep]" {
		t.Fatalf("final release = %v", order)
	}
}

func TestDirtyClassesStopUnchangedAndScopePaintVersusLayout(t *testing.T) {
	base := Description{Kind: 1, Children: []Description{{Kind: 2, Key: "child"}}}
	var retained Tree
	initial, err := retained.Update(base)
	if err != nil {
		t.Fatal(err)
	}
	if !initial.Has(DirtyLayout) || !initial.Has(DirtyPaint) {
		t.Fatalf("initial dirty = %08b", initial.Dirty)
	}
	retained.ClearDirty()

	unchanged, err := retained.Update(base)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Dirty != DirtyBuild {
		t.Fatalf("unchanged dirty = %08b, want build only", unchanged.Dirty)
	}
	retained.ClearDirty()

	paintOnly := base
	paintOnly.Children = append([]Description(nil), base.Children...)
	paintOnly.Children[0].Properties.Paint.BackgroundR = 10
	paintChange, err := retained.Update(paintOnly)
	if err != nil {
		t.Fatal(err)
	}
	child := retained.Root().Children[0]
	if !paintChange.Has(DirtyPaint) || paintChange.Has(DirtyLayout) {
		t.Fatalf("paint-only aggregate = %08b", paintChange.Dirty)
	}
	if child.Dirty&DirtyPaint == 0 || child.Dirty&DirtyLayout != 0 || retained.Root().Dirty != DirtyBuild {
		t.Fatalf("paint dirty scope root/child = %08b/%08b", retained.Root().Dirty, child.Dirty)
	}
	retained.ClearDirty()

	layoutChange := paintOnly
	layoutChange.Children = append([]Description(nil), paintOnly.Children...)
	layoutChange.Children[0].Properties.Layout.Width = Length{Kind: 1, Value: 20}
	change, err := retained.Update(layoutChange)
	if err != nil {
		t.Fatal(err)
	}
	if !change.Has(DirtyLayout) || retained.Root().Dirty&DirtyLayout == 0 || retained.Root().Children[0].Dirty&DirtyLayout == 0 {
		t.Fatalf("layout dirty scope root/child = %08b/%08b", retained.Root().Dirty, retained.Root().Children[0].Dirty)
	}
}

func TestZIndexAndOverflowDoNotInvalidateGeometry(t *testing.T) {
	base := Description{Kind: 1, Children: []Description{{Kind: 2}}}
	var retained Tree
	if err := retained.Commit(base); err != nil {
		t.Fatal(err)
	}
	retained.ClearDirty()

	next := base
	next.Children = append([]Description(nil), base.Children...)
	next.Children[0].Properties.Layout.ZIndex = 4
	next.Children[0].Properties.Layout.Overflow = 1
	changes, err := retained.Update(next)
	if err != nil {
		t.Fatal(err)
	}
	if changes.Has(DirtyMeasure) || changes.Has(DirtyLayout) {
		t.Fatalf("stack/clip-only dirty = %08b", changes.Dirty)
	}
	if !changes.Has(DirtyDisplay) || !changes.Has(DirtyPaint) || !changes.Has(DirtySemantics) {
		t.Fatalf("stack/clip did not invalidate display/hit-test: %08b", changes.Dirty)
	}
}

func TestVisibilityAndOpacityDoNotInvalidateGeometry(t *testing.T) {
	base := Description{Kind: 1, Children: []Description{{Kind: 2}}}
	var retained Tree
	if err := retained.Commit(base); err != nil {
		t.Fatal(err)
	}
	retained.ClearDirty()
	next := base
	next.Children = append([]Description(nil), base.Children...)
	next.Children[0].Properties.Paint.Visibility = 1
	next.Children[0].Properties.Paint.OpacitySet = true
	next.Children[0].Properties.Paint.Opacity = 0
	changes, err := retained.Update(next)
	if err != nil {
		t.Fatal(err)
	}
	if changes.Has(DirtyMeasure) || changes.Has(DirtyLayout) || !changes.Has(DirtyPaint) {
		t.Fatalf("visibility/opacity dirty = %08b", changes.Dirty)
	}
}

func TestDisabledInvalidatesStateStyleWithoutLayout(t *testing.T) {
	base := Description{Kind: 1, Properties: Properties{Semantics: SemanticsProperties{HasPress: true}}}
	var retained Tree
	if err := retained.Commit(base); err != nil {
		t.Fatal(err)
	}
	retained.ClearDirty()
	next := base
	next.Properties.Semantics.Disabled = true
	changes, err := retained.Update(next)
	if err != nil {
		t.Fatal(err)
	}
	if !changes.Has(DirtyStyle) || !changes.Has(DirtyPaint) || !changes.Has(DirtySemantics) || changes.Has(DirtyLayout) {
		t.Fatalf("disabled dirty = %08b", changes.Dirty)
	}
}

func TestFailedCommitPreservesStateResourcesAndDirty(t *testing.T) {
	var retained Tree
	if err := retained.Commit(Description{Kind: 1, Children: []Description{{Kind: 2, Key: "old"}}}); err != nil {
		t.Fatal(err)
	}
	retained.ClearDirty()
	oldRoot := retained.Root()
	oldRoot.Children[0].State.Focused = true
	var released []string
	oldRoot.Children[0].AddResource(recordingResource{"old", &released})
	_, err := retained.Update(Description{Kind: 1, Children: []Description{{Kind: 2, Key: "dup"}, {Kind: 3, Key: "dup"}}})
	if err == nil {
		t.Fatal("duplicate key accepted")
	}
	if retained.Root() != oldRoot || !oldRoot.Children[0].State.Focused || len(released) != 0 || oldRoot.Dirty != 0 {
		t.Fatal("failed transaction changed the committed tree")
	}
}

func TestPreviewPreservesCommittedTreeStateAndResources(t *testing.T) {
	var retained Tree
	if err := retained.Commit(Description{Kind: 1, Children: []Description{{Kind: 2, Key: "child"}}}); err != nil {
		t.Fatal(err)
	}
	retained.ClearDirty()
	oldRoot := retained.Root()
	oldRoot.Children[0].State.Focused = true
	released := 0
	oldRoot.Children[0].AddResource(countingResource{released: &released})
	changes, candidate, err := retained.Preview(Description{Kind: 1, Children: []Description{{Kind: 2, Key: "child", Properties: Properties{Content: ContentProperties{Text: "new"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if candidate == oldRoot || candidate.Children[0].ID != oldRoot.Children[0].ID || !changes.Has(DirtyLayout) {
		t.Fatalf("preview candidate/changes = %p/%+v", candidate, changes)
	}
	if retained.Root() != oldRoot || !oldRoot.Children[0].State.Focused || released != 0 || oldRoot.Dirty != 0 {
		t.Fatal("preview mutated the committed tree")
	}
	retained.Unmount()
	if released != 1 {
		t.Fatalf("resource releases = %d, want 1", released)
	}
}

func TestUpdateAllocationBaseline(t *testing.T) {
	var retained Tree
	description := Description{Kind: 1, Children: []Description{{Kind: 2, Key: "child"}}}
	if err := retained.Commit(description); err != nil {
		t.Fatal(err)
	}
	retained.ClearDirty()
	allocations := testing.AllocsPerRun(100, func() {
		if _, err := retained.Update(description); err != nil {
			panic(err)
		}
		retained.ClearDirty()
	})
	t.Logf("unchanged two-node reconcile baseline: %.0f allocations/update", allocations)
	if allocations > 20 {
		t.Fatalf("allocation regression: %.0f allocations/update", allocations)
	}
}

func FuzzTreeRandomUpdates(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5})
	f.Add([]byte{9, 9, 1, 9, 2, 9})
	f.Fuzz(func(t *testing.T, data []byte) {
		var retained Tree
		acquired, released := 0, 0
		for round := 0; round < 8; round++ {
			children := make([]Description, 0, min(len(data), 12))
			seen := make(map[Key]struct{})
			for index, value := range data {
				if index >= 12 {
					break
				}
				child := Description{Kind: Kind(value%4 + 2)}
				if value&1 != 0 {
					child.Key = Key(fmt.Sprintf("k%d", (int(value)+round)%8))
					if _, duplicate := seen[child.Key]; duplicate {
						continue
					}
					seen[child.Key] = struct{}{}
				}
				child.Properties.Content.Text = fmt.Sprintf("%d", value+byte(round))
				children = append(children, child)
			}
			if _, err := retained.Update(Description{Kind: 1, Children: children}); err != nil {
				t.Fatal(err)
			}
			var attach func(*Node)
			attach = func(node *Node) {
				if node == nil {
					return
				}
				node.AddResource(countingResource{released: &released})
				acquired++
				for _, child := range node.Children {
					attach(child)
				}
			}
			attach(retained.Root())
			retained.ClearDirty()
		}
		retained.Unmount()
		if released != acquired {
			t.Fatalf("resource leak: acquired %d, released %d", acquired, released)
		}
	})
}

func TestDuplicateKeyRejectsCommitTransactionally(t *testing.T) {
	var retained Tree
	if err := retained.Commit(Description{Kind: 1, Children: []Description{{Kind: 2, Key: "old"}}}); err != nil {
		t.Fatal(err)
	}
	root := retained.Root()
	err := retained.Commit(Description{Kind: 1, Children: []Description{{Kind: 2, Key: "dup"}, {Kind: 3, Key: "dup"}}})
	if err == nil {
		t.Fatal("duplicate key accepted")
	}
	if retained.Root() != root {
		t.Fatal("failed commit replaced the prior tree")
	}
}

func TestUnkeyedIdentityIsKindAndIndexOnly(t *testing.T) {
	var retained Tree
	if err := retained.Commit(Description{Kind: 1, Children: []Description{{Kind: 2}, {Kind: 3}}}); err != nil {
		t.Fatal(err)
	}
	secondID := retained.Root().Children[1].ID
	if err := retained.Commit(Description{Kind: 1, Children: []Description{{Kind: 3}, {Kind: 3}}}); err != nil {
		t.Fatal(err)
	}
	if retained.Root().Children[0].ID == secondID {
		t.Fatal("unkeyed child migrated between indices")
	}
	if retained.Root().Children[1].ID != secondID {
		t.Fatal("same-kind child at same index lost identity")
	}
}
