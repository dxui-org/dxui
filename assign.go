package dxui

// Assign returns a callback that stores its argument in target. It is intended
// for simple controlled UI callbacks. Assign panics when target is nil; it does
// not schedule work or replace App.Update for cross-goroutine mutation.
func Assign[T any](target *T) func(T) {
	if target == nil {
		panic("dxui: Assign target is nil")
	}
	return func(value T) { *target = value }
}
