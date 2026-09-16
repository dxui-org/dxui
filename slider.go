package dxui

import "math"

const (
	defaultSliderMin  float32 = 0
	defaultSliderMax  float32 = 100
	defaultSliderStep float32 = 1
)

type sliderModel struct {
	min, max, step, value float32
	valid                 bool
}

// normalizeSlider keeps malformed scalar inputs out of layout, input, and
// paint. The all-zero range is the useful zero-value default [0,100]. Other
// invalid ranges are inert. A bad step falls back to one logical value unit.
func normalizeSlider(props SliderProps) sliderModel {
	minimum, maximum := props.Min, props.Max
	if minimum == 0 && maximum == 0 {
		minimum, maximum = defaultSliderMin, defaultSliderMax
	}
	if !finite(minimum) || !finite(maximum) || minimum >= maximum {
		if !finite(minimum) {
			minimum = defaultSliderMin
		}
		return sliderModel{min: minimum, max: minimum, step: defaultSliderStep, value: minimum}
	}
	step := props.Step
	if !finite(step) || step <= 0 {
		step = defaultSliderStep
	}
	value := props.Value
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), -1) {
		value = minimum
	} else if math.IsInf(float64(value), 1) {
		value = maximum
	} else {
		value = min(max(value, minimum), maximum)
	}
	return sliderModel{min: minimum, max: maximum, step: step, value: value, valid: true}
}

func (model sliderModel) fraction() float32 {
	if !model.valid {
		return 0
	}
	return (model.value - model.min) / (model.max - model.min)
}

func (model sliderModel) proposal(candidate float64, endpoint bool) float32 {
	if !model.valid {
		return model.min
	}
	minimum, maximum := float64(model.min), float64(model.max)
	if endpoint || candidate <= minimum {
		if candidate >= maximum {
			return model.max
		}
		return model.min
	}
	if candidate >= maximum {
		return model.max
	}
	index := math.Round((candidate - minimum) / float64(model.step))
	value := minimum + index*float64(model.step)
	if value <= minimum {
		return model.min
	}
	if value >= maximum {
		return model.max
	}
	return float32(value)
}

func (model sliderModel) stepBy(direction int) float32 {
	if !model.valid || direction == 0 {
		return model.value
	}
	minimum := float64(model.min)
	position := (float64(model.value) - minimum) / float64(model.step)
	nearest := math.Round(position)
	var index float64
	// Compare the represented float32 grid value, not an accumulated delta or
	// a fixed epsilon. This makes a value returned by proposal exactly
	// recognizable as aligned across both small and large step indices.
	if float32(minimum+nearest*float64(model.step)) == model.value {
		index = nearest + float64(direction)
	} else if direction > 0 {
		index = math.Ceil(position)
	} else {
		index = math.Floor(position)
	}
	return model.proposal(minimum+index*float64(model.step), false)
}

func sliderTravel(x, width, thumbSize float32) (start, end float32) {
	radius := min(max(float32(0), thumbSize)/2, max(float32(0), width)/2)
	start, end = x+radius, x+width-radius
	if end < start {
		middle := x + width/2
		return middle, middle
	}
	return start, end
}
