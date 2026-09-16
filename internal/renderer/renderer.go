// Package renderer defines backend-neutral paint execution ports.
package renderer

import "github.com/dxui-org/dxui/internal/paint"

// Renderer executes a retained display list and presents exactly once.
type Renderer interface {
	Render(paint.DisplayList) error
	Present() error
}
