package dxui

import "math"

// normalizeProgress prevents non-finite application values from reaching the
// retained display model. NaN and negative infinity paint empty; positive
// infinity paints complete.
func normalizeProgress(value float32) float32 {
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), -1) || value < 0 {
		return 0
	}
	if math.IsInf(float64(value), 1) || value > 1 {
		return 1
	}
	return value
}
