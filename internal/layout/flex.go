// Package layout contains backend-independent logical-unit layout algorithms.
package layout

import (
	"fmt"
	"math"
)

// Justify is the ADR-0005 main-axis alignment subset.
type Justify uint8

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
)

// FlexItem is one resolved in-flow flex input. Absolute items are omitted from
// flex sizing and receive a zero result in this stage.
type FlexItem struct {
	Basis, Min, Max float32
	Grow, Shrink    float32
	MarginStart     float32
	MarginEnd       float32
	Absolute        bool
}

// FlexResult contains a logical main-axis position and size.
type FlexResult struct{ Position, Size float32 }

// FlexLine applies the single-line ADR-0005 grow/shrink and alignment rules.
func FlexLine(available, gap float32, justify Justify, items []FlexItem) ([]FlexResult, error) {
	return flexLine(available, gap, 0, justify, items)
}

func flexLine(available, gap, overlap float32, justify Justify, items []FlexItem) ([]FlexResult, error) {
	if !finiteNonNegative(available) || !finiteNonNegative(gap) || !finiteNonNegative(overlap) {
		return nil, fmt.Errorf("layout: available size, gap, and overlap must be finite and non-negative")
	}
	results := make([]FlexResult, len(items))
	normalized := append([]FlexItem(nil), items...)
	flow := make([]int, 0, len(items))
	for index, item := range normalized {
		if item.Absolute {
			continue
		}
		if !validItem(item) {
			return nil, fmt.Errorf("layout: invalid flex item %d", index)
		}
		if item.Max < item.Min {
			item.Max = item.Min
			normalized[index] = item
		}
		flow = append(flow, index)
		results[index].Size = clamp(item.Basis, item.Min, item.Max)
	}

	used := marginsAndSizes(flow, normalized, results)
	if len(flow) > 1 {
		used += (gap - overlap) * float32(len(flow)-1)
	}
	if !finite(used) {
		return nil, fmt.Errorf("layout: flex arithmetic overflow")
	}
	used = max(0, used)
	free := available - used
	if free > 0 {
		distributeGrow(free, flow, normalized, results)
	} else if free < 0 {
		distributeShrink(-free, flow, normalized, results)
	}

	used = marginsAndSizes(flow, normalized, results)
	if len(flow) > 1 {
		used += (gap - overlap) * float32(len(flow)-1)
	}
	if !finite(used) {
		return nil, fmt.Errorf("layout: flex arithmetic overflow")
	}
	used = max(0, used)
	remaining := max(float32(0), available-used)
	start, extraGap := float32(0), float32(0)
	switch justify {
	case JustifyStart:
	case JustifyCenter:
		start = remaining / 2
	case JustifyEnd:
		start = remaining
	case JustifySpaceBetween:
		if len(flow) > 1 {
			extraGap = remaining / float32(len(flow)-1)
		}
	default:
		return nil, fmt.Errorf("layout: unsupported justification %d", justify)
	}

	cursor := start
	for order, index := range flow {
		cursor += normalized[index].MarginStart
		if !finite(cursor) {
			return nil, fmt.Errorf("layout: flex position arithmetic overflow")
		}
		results[index].Position = cursor
		cursor += results[index].Size + normalized[index].MarginEnd
		if order+1 < len(flow) {
			cursor += gap - overlap + extraGap
		}
		if !finite(cursor) {
			return nil, fmt.Errorf("layout: flex position arithmetic overflow")
		}
	}
	return results, nil
}

func distributeGrow(free float32, flow []int, items []FlexItem, results []FlexResult) {
	unfrozen := append([]int(nil), flow...)
	for free > 0 && len(unfrozen) > 0 {
		factor := float32(0)
		for _, index := range unfrozen {
			factor += items[index].Grow
		}
		if factor == 0 {
			return
		}
		next := unfrozen[:0]
		consumed := float32(0)
		for _, index := range unfrozen {
			share := free * items[index].Grow / factor
			proposed := results[index].Size + share
			clamped := min(proposed, items[index].Max)
			consumed += clamped - results[index].Size
			results[index].Size = clamped
			if clamped < items[index].Max {
				next = append(next, index)
			}
		}
		if consumed <= 0 {
			return
		}
		free -= consumed
		unfrozen = next
	}
}

func distributeShrink(deficit float32, flow []int, items []FlexItem, results []FlexResult) {
	unfrozen := append([]int(nil), flow...)
	for deficit > 0 && len(unfrozen) > 0 {
		factor := float32(0)
		for _, index := range unfrozen {
			factor += items[index].Shrink * items[index].Basis
		}
		if factor == 0 {
			return
		}
		next := unfrozen[:0]
		consumed := float32(0)
		for _, index := range unfrozen {
			weight := items[index].Shrink * items[index].Basis
			share := deficit * weight / factor
			proposed := results[index].Size - share
			clamped := max(proposed, items[index].Min)
			consumed += results[index].Size - clamped
			results[index].Size = clamped
			if clamped > items[index].Min {
				next = append(next, index)
			}
		}
		if consumed <= 0 {
			return
		}
		deficit -= consumed
		unfrozen = next
	}
}

func marginsAndSizes(flow []int, items []FlexItem, results []FlexResult) float32 {
	total := float32(0)
	for _, index := range flow {
		total += items[index].MarginStart + results[index].Size + items[index].MarginEnd
	}
	return total
}

func validItem(item FlexItem) bool {
	values := []float32{item.Basis, item.Min, item.Max, item.Grow, item.Shrink, item.MarginStart, item.MarginEnd}
	for _, value := range values {
		if !finiteNonNegative(value) {
			return false
		}
	}
	return true
}

func finiteNonNegative(value float32) bool {
	return value >= 0 && !float32IsNaNOrInf(value)
}

func float32IsNaNOrInf(value float32) bool {
	return math.IsNaN(float64(value)) || math.IsInf(float64(value), 0)
}

func clamp(value, minimum, maximum float32) float32 {
	return min(max(value, minimum), maximum)
}
