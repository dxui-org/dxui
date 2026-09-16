// Package sdl3 is the only production boundary allowed to import go-sdl3.
// It owns dynamic loading, events, windows, and renderers for roadmap M1.
package sdl3

import "github.com/dxui-org/go-sdl3/sdl"

// BindingVersion is the concrete Go binding version selected for this build.
const BindingVersion = "v0.0.0-20260912133812-24cf274387b0"

// RequiredSDLVersion is the embedded native ABI required by ADR-0001.
const RequiredSDLVersion = "3.4.0"

// LoadedVersion reports the already-loaded SDL version.
func LoadedVersion() string { return sdl.GetVersion().String() }
