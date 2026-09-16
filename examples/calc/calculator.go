package main

import (
	"math"
	"strconv"
	"strings"
)

type calculator struct {
	display    string
	expression string
	left       float64
	pending    string
	replace    bool
	dark       bool
}

func newCalculator() *calculator {
	return &calculator{display: "0", replace: true}
}

func (c *calculator) input(label string) {
	if c.display == "Error" && label != "C" {
		c.clear()
	}

	switch label {
	case "C":
		c.clear()
	case "{}":
		c.toggleSign()
	case "%":
		c.percent()
	case "/", "x", "-", "+":
		c.operator(label)
	case "=":
		c.equals()
	case "backspace":
		c.backspace()
	case ".":
		c.decimal()
	default:
		c.digit(label)
	}
}

func (c *calculator) clear() {
	c.display = "0"
	c.expression = ""
	c.left = 0
	c.pending = ""
	c.replace = true
}

func (c *calculator) digit(digit string) {
	if len(digit) != 1 || digit[0] < '0' || digit[0] > '9' {
		return
	}
	if c.replace || c.display == "0" {
		c.display = digit
		c.replace = false
		return
	}
	if len(strings.TrimPrefix(c.display, "-")) < 12 {
		c.display += digit
	}
}

func (c *calculator) decimal() {
	if c.replace {
		c.display = "0."
		c.replace = false
		return
	}
	if !strings.Contains(c.display, ".") {
		c.display += "."
	}
}

func (c *calculator) toggleSign() {
	if c.display == "0" {
		return
	}
	if strings.HasPrefix(c.display, "-") {
		c.display = strings.TrimPrefix(c.display, "-")
	} else {
		c.display = "-" + c.display
	}
	if c.replace {
		c.left = -c.left
	}
}

func (c *calculator) percent() {
	value, ok := c.value()
	if !ok {
		return
	}
	c.display = formatNumber(value / 100)
	c.replace = true
}

func (c *calculator) backspace() {
	if c.replace {
		c.display = "0"
		return
	}
	c.display = strings.TrimSuffix(c.display, c.display[len(c.display)-1:])
	if c.display == "" || c.display == "-" {
		c.display = "0"
		c.replace = true
	}
}

func (c *calculator) operator(next string) {
	value, ok := c.value()
	if !ok {
		return
	}
	if c.pending != "" && !c.replace {
		result, valid := calculate(c.left, value, c.pending)
		if !valid {
			c.fail()
			return
		}
		c.left = result
		c.display = formatNumber(result)
	} else if c.pending == "" {
		c.left = value
	}
	c.pending = next
	c.expression = formatNumber(c.left) + " " + next
	c.replace = true
}

func (c *calculator) equals() {
	if c.pending == "" {
		return
	}
	right, ok := c.value()
	if !ok {
		return
	}
	result, valid := calculate(c.left, right, c.pending)
	if !valid {
		c.fail()
		return
	}
	c.expression = formatNumber(c.left) + " " + c.pending + " " + formatNumber(right) + " ="
	c.display = formatNumber(result)
	c.left = result
	c.pending = ""
	c.replace = true
}

func (c *calculator) value() (float64, bool) {
	value, err := strconv.ParseFloat(c.display, 64)
	return value, err == nil
}

func (c *calculator) fail() {
	c.display = "Error"
	c.expression = "Cannot divide by zero"
	c.pending = ""
	c.replace = true
}

func calculate(left, right float64, operator string) (float64, bool) {
	var result float64
	switch operator {
	case "+":
		result = left + right
	case "-":
		result = left - right
	case "x":
		result = left * right
	case "/":
		if right == 0 {
			return 0, false
		}
		result = left / right
	default:
		return 0, false
	}
	return result, !math.IsNaN(result) && !math.IsInf(result, 0)
}

func formatNumber(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'g', 12, 64)
}
