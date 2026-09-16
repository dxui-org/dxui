package main

import (
	"testing"

	"github.com/dxui-org/dxui/internal/icondata"
)

func TestPathCommands(t *testing.T) {
	commands, err := parsePathData("M1 2h3v4q1 2 3 4t2 1c1 2 3 4 5 6s2 3 4 5a2 3 30 0 1 4 5z")
	if err != nil {
		t.Fatal(err)
	}
	want := []icondata.PathVerb{icondata.PathMove, icondata.PathLine, icondata.PathLine, icondata.PathQuad, icondata.PathQuad, icondata.PathCubic, icondata.PathCubic, icondata.PathCubic, icondata.PathCubic, icondata.PathClose}
	if len(commands) != len(want) {
		t.Fatalf("commands = %d, want %d", len(commands), len(want))
	}
	for index, verb := range want {
		if commands[index].Verb != verb {
			t.Fatalf("command %d = %d, want %d", index, commands[index].Verb, verb)
		}
	}
}

func TestPathRejectsUnknown(t *testing.T) {
	if _, err := parsePathData("M0 0R1 2"); err == nil {
		t.Fatal("unknown path command accepted")
	}
}
