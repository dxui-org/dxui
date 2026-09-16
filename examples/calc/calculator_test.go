package main

import (
	"testing"

	"github.com/dxui-org/dxui"
)

func TestCalculatorOperations(t *testing.T) {
	tests := []struct {
		name       string
		keys       []string
		display    string
		expression string
	}{
		{name: "addition", keys: []string{"6", "9", "8", "+", "3", "0", "2", "="}, display: "1000", expression: "698 + 302 ="},
		{name: "precedence is immediate", keys: []string{"8", "+", "2", "x", "3", "="}, display: "30", expression: "10 x 3 ="},
		{name: "decimal", keys: []string{"1", ".", "5", "+", "2", ".", "2", "5", "="}, display: "3.75", expression: "1.5 + 2.25 ="},
		{name: "negative", keys: []string{"9", "{}", "x", "2", "="}, display: "-18", expression: "-9 x 2 ="},
		{name: "percent", keys: []string{"2", "5", "%"}, display: "0.25", expression: ""},
		{name: "backspace", keys: []string{"1", "2", "3", "backspace"}, display: "12", expression: ""},
		{name: "division by zero", keys: []string{"7", "/", "0", "="}, display: "Error", expression: "Cannot divide by zero"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calculator := newCalculator()
			for _, key := range test.keys {
				calculator.input(key)
			}
			if calculator.display != test.display {
				t.Fatalf("display = %q, want %q", calculator.display, test.display)
			}
			if calculator.expression != test.expression {
				t.Fatalf("expression = %q, want %q", calculator.expression, test.expression)
			}
		})
	}
}

func TestCalculatorKeyboardBindingsMatchButtonActions(t *testing.T) {
	keyboard := newCalculator()
	bindings := calculatorShortcuts(keyboard)
	press := func(key dxui.ShortcutKey) {
		t.Helper()
		for _, binding := range bindings {
			if binding.Key == key {
				binding.OnPress()
				return
			}
		}
		t.Fatalf("missing shortcut %v", key)
	}
	for _, key := range []dxui.ShortcutKey{dxui.Key1, dxui.Key2, dxui.KeyPlus, dxui.Key3, dxui.KeyEnter, dxui.KeyBackspace} {
		press(key)
	}

	pointer := newCalculator()
	for _, label := range []string{"1", "2", "+", "3", "=", "backspace"} {
		pointer.input(label)
	}
	if keyboard.display != pointer.display || keyboard.expression != pointer.expression {
		t.Fatalf("keyboard=%+v pointer=%+v", keyboard, pointer)
	}
}

func TestCalculatorClearAndOperatorReplacement(t *testing.T) {
	calculator := newCalculator()
	for _, key := range []string{"4", "+", "x", "5", "C"} {
		calculator.input(key)
	}
	if calculator.display != "0" || calculator.expression != "" || calculator.pending != "" {
		t.Fatalf("clear left unexpected state: %+v", calculator)
	}
}
