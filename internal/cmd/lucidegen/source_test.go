package main

import (
	"path/filepath"
	"testing"
)

func TestPinnedSourceConvertsCompletely(t *testing.T) {
	source := filepath.Join("..", "..", "..", "third_party", "lucide", "lucide-static-"+lucideVersion+".tgz")
	icons, license, err := loadSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(icons), 2066; got != want {
		t.Fatalf("icons = %d, want %d", got, want)
	}
	canonical, aliases := 0, 0
	for _, icon := range icons {
		if icon.canonical {
			canonical++
		} else {
			aliases++
			if icon.aliasOf == "" {
				t.Fatalf("alias %q has no target", icon.slug)
			}
		}
	}
	if names, err := resolveNames(icons); err != nil || len(names) != len(icons) {
		t.Fatalf("resolved names = %d/%v", len(names), err)
	}
	if canonical != 1807 || aliases != 259 {
		t.Fatalf("canonical/aliases = %d/%d, want 1807/259", canonical, aliases)
	}
	if license == "" {
		t.Fatal("empty license")
	}
}

func TestConvertRejectsUnknownInput(t *testing.T) {
	inputs := []string{
		`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><g/></svg>`,
		`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M0 0R1 1"/></svg>`,
		`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="square" stroke-linejoin="round"><path d="M0 0L1 1"/></svg>`,
	}
	for _, input := range inputs {
		if _, err := convertSVG([]byte(input)); err == nil {
			t.Fatalf("accepted unsupported SVG %s", input)
		}
	}
}
