package catalog

import (
	"go/token"
	"testing"

	"github.com/dxui-org/dxui/icon"
)

func TestCompleteCatalog(t *testing.T) {
	if len(Icons) != icon.Count {
		t.Fatalf("catalog = %d, want %d", len(Icons), icon.Count)
	}
	seen := make(map[string]struct{}, len(Icons))
	seenGoNames := make(map[string]struct{}, len(Icons))
	for _, entry := range Icons {
		if entry.Name == "" || entry.GoName == "" || entry.Icon == nil {
			t.Fatalf("invalid entry %+v", entry)
		}
		if !token.IsIdentifier(entry.GoName) || !token.IsExported(entry.GoName) {
			t.Fatalf("invalid Go constructor name %q for %q", entry.GoName, entry.Name)
		}
		if _, duplicate := seen[entry.Name]; duplicate {
			t.Fatalf("duplicate name %q", entry.Name)
		}
		if _, duplicate := seenGoNames[entry.GoName]; duplicate {
			t.Fatalf("duplicate Go constructor name %q", entry.GoName)
		}
		seen[entry.Name] = struct{}{}
		seenGoNames[entry.GoName] = struct{}{}
		if entry.Name == "search" && entry.GoName != "Search" {
			t.Fatalf("search GoName = %q, want Search", entry.GoName)
		}
		data := entry.Icon()
		if !data.IsPacked() || data.Commands != nil {
			t.Fatalf("%s is not immutable packed data", entry.Name)
		}
	}
}
