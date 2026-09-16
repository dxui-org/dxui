// Package icon provides the complete Lucide 1.41.0 icon set as immutable dxui.IconData values.
//
// Generated functions allocate no memory and the Go linker can discard every unused icon.
// StrokeWidth defaults to 2 icon view-box units and can be overridden through dxui.IconProps.
package icon

//go:generate go run ../internal/cmd/lucidegen -source ../third_party/lucide/lucide-static-1.41.0.tgz -out . -catalog catalog -fixtures ..
