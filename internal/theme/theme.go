// Package theme validates and resolves token layers without importing the
// public package. Root-package adapters keep the final-user API at dxui.
package theme

import (
	"fmt"
	"math"
	"time"
)

// Color is a backend-neutral non-premultiplied RGBA value.
type Color struct{ R, G, B, A uint8 }

// ColorValue and MetricValue are literal-or-reference inputs.
type ColorValue struct {
	Literal Color
	Token   string
	IsToken bool
	Set     bool
}

type MetricValue struct {
	Literal float32
	Token   string
	IsToken bool
	Set     bool
}

// Source contains typed primitive and semantic layers. Component patches are
// validated by the root adapter against the resulting immutable tables.
type Source struct {
	PrimitiveColors    map[string]Color
	PrimitiveMetrics   map[string]float32
	PrimitiveDurations map[string]time.Duration
	SemanticColors     map[string]ColorValue
	SemanticMetrics    map[string]MetricValue
	SemanticDurations  map[string]string
}

// Resolved is a copied, fully concrete token lookup table.
type Resolved struct {
	Colors    map[string]Color
	Metrics   map[string]float32
	Durations map[string]time.Duration
}

// Resolve rejects missing references, same-layer cycles, invalid metrics, and
// invalid durations without exposing a partial result.
func Resolve(source Source) (Resolved, error) {
	result := Resolved{
		Colors:    make(map[string]Color, len(source.PrimitiveColors)+len(source.SemanticColors)),
		Metrics:   make(map[string]float32, len(source.PrimitiveMetrics)+len(source.SemanticMetrics)),
		Durations: make(map[string]time.Duration, len(source.PrimitiveDurations)+len(source.SemanticDurations)),
	}
	for token, value := range source.PrimitiveColors {
		if token == "" {
			return Resolved{}, fmt.Errorf("theme: empty primitive color token")
		}
		result.Colors[token] = value
	}
	for token, value := range source.PrimitiveMetrics {
		if token == "" || !validMetric(value) {
			return Resolved{}, fmt.Errorf("theme: invalid primitive metric %q", token)
		}
		result.Metrics[token] = value
	}
	for token, value := range source.PrimitiveDurations {
		if token == "" || value < 0 {
			return Resolved{}, fmt.Errorf("theme: invalid primitive duration %q", token)
		}
		result.Durations[token] = value
	}

	colorVisiting := make(map[string]bool, len(source.SemanticColors))
	for token := range source.SemanticColors {
		if _, duplicate := source.PrimitiveColors[token]; duplicate {
			return Resolved{}, fmt.Errorf("theme: color token %q exists in primitive and semantic layers", token)
		}
	}
	var color func(string) (Color, error)
	color = func(token string) (Color, error) {
		if value, ok := result.Colors[token]; ok {
			return value, nil
		}
		value, ok := source.SemanticColors[token]
		if !ok {
			return Color{}, fmt.Errorf("missing color token %q", token)
		}
		if colorVisiting[token] {
			return Color{}, fmt.Errorf("semantic color cycle at %q", token)
		}
		colorVisiting[token] = true
		var resolved Color
		var err error
		if value.IsToken {
			resolved, err = color(value.Token)
		} else if value.Set {
			resolved = value.Literal
		} else {
			err = fmt.Errorf("unset semantic color %q", token)
		}
		delete(colorVisiting, token)
		if err != nil {
			return Color{}, err
		}
		result.Colors[token] = resolved
		return resolved, nil
	}
	for token := range source.SemanticColors {
		if _, err := color(token); err != nil {
			return Resolved{}, fmt.Errorf("theme: resolve semantic color %q: %w", token, err)
		}
	}

	metricVisiting := make(map[string]bool, len(source.SemanticMetrics))
	for token := range source.SemanticMetrics {
		if _, duplicate := source.PrimitiveMetrics[token]; duplicate {
			return Resolved{}, fmt.Errorf("theme: metric token %q exists in primitive and semantic layers", token)
		}
	}
	var metric func(string) (float32, error)
	metric = func(token string) (float32, error) {
		if value, ok := result.Metrics[token]; ok {
			return value, nil
		}
		value, ok := source.SemanticMetrics[token]
		if !ok {
			return 0, fmt.Errorf("missing metric token %q", token)
		}
		if metricVisiting[token] {
			return 0, fmt.Errorf("semantic metric cycle at %q", token)
		}
		metricVisiting[token] = true
		resolved := value.Literal
		var err error
		if value.IsToken {
			resolved, err = metric(value.Token)
		} else if !value.Set {
			err = fmt.Errorf("unset semantic metric %q", token)
		}
		delete(metricVisiting, token)
		if err != nil {
			return 0, err
		}
		if !validMetric(resolved) {
			return 0, fmt.Errorf("invalid metric %q", token)
		}
		result.Metrics[token] = resolved
		return resolved, nil
	}
	for token := range source.SemanticMetrics {
		if _, err := metric(token); err != nil {
			return Resolved{}, fmt.Errorf("theme: resolve semantic metric %q: %w", token, err)
		}
	}

	durationVisiting := make(map[string]bool, len(source.SemanticDurations))
	for token := range source.SemanticDurations {
		if _, duplicate := source.PrimitiveDurations[token]; duplicate {
			return Resolved{}, fmt.Errorf("theme: duration token %q exists in primitive and semantic layers", token)
		}
	}
	var duration func(string) (time.Duration, error)
	duration = func(token string) (time.Duration, error) {
		if value, ok := result.Durations[token]; ok {
			return value, nil
		}
		ref, ok := source.SemanticDurations[token]
		if !ok {
			return 0, fmt.Errorf("missing duration token %q", token)
		}
		if durationVisiting[token] {
			return 0, fmt.Errorf("semantic duration cycle at %q", token)
		}
		durationVisiting[token] = true
		value, err := duration(ref)
		delete(durationVisiting, token)
		if err != nil {
			return 0, err
		}
		result.Durations[token] = value
		return value, nil
	}
	for token := range source.SemanticDurations {
		if _, err := duration(token); err != nil {
			return Resolved{}, fmt.Errorf("theme: resolve semantic duration %q: %w", token, err)
		}
	}
	return result, nil
}

// Color resolves an explicitly set literal or token value.
func (resolved Resolved) Color(value ColorValue) (Color, error) {
	if !value.Set {
		return Color{}, fmt.Errorf("theme: unset color")
	}
	if !value.IsToken {
		return value.Literal, nil
	}
	result, ok := resolved.Colors[value.Token]
	if !ok {
		return Color{}, fmt.Errorf("theme: missing color token %q", value.Token)
	}
	return result, nil
}

// Metric resolves an unset value as zero, matching zero-value public layout
// metrics, or resolves an explicitly set literal/token.
func (resolved Resolved) Metric(value MetricValue) (float32, error) {
	if !value.Set {
		return 0, nil
	}
	if !value.IsToken {
		if !validMetric(value.Literal) {
			return 0, fmt.Errorf("theme: invalid metric %v", value.Literal)
		}
		return value.Literal, nil
	}
	result, ok := resolved.Metrics[value.Token]
	if !ok {
		return 0, fmt.Errorf("theme: missing metric token %q", value.Token)
	}
	return result, nil
}

func validMetric(value float32) bool {
	return value >= 0 && !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0)
}
