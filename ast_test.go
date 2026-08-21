package lua

import (
	"math"
	"testing"
)

func TestNumberExprFloat(t *testing.T) {
	cases := []struct {
		text string
		want float64
	}{
		{"0", 0},
		{"42", 42},
		{"3.14", 3.14},
		{"1e10", 1e10},
		{"1.5e-3", 1.5e-3},
		{"1E2", 100},
		{"0xFF", 255},
		{"0x10", 16},
		{"0X1a", 26},
	}
	for _, c := range cases {
		n := &NumberExpr{Text: c.text}
		got, err := n.Float()
		if err != nil {
			t.Errorf("%q: unexpected error: %v", c.text, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q: Float() = %g, want %g", c.text, got, c.want)
		}
	}
}

func TestNumberExprFloatInvalid(t *testing.T) {
	n := &NumberExpr{Text: "not_a_number"}
	if _, err := n.Float(); err == nil {
		t.Error("expected error for invalid number, got nil")
	}
}

func TestNumberExprIsInteger(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"0", true},
		{"42", true},
		{"0xFF", true},
		{"0xE", true},   // 'E' is a hex digit, not an exponent
		{"0X10a", true}, // uppercase X, mixed-case digits
		{"3.14", false},
		{"1e10", false},
		{"1.5e-3", false},
		{".5", false},
	}
	for _, c := range cases {
		n := &NumberExpr{Text: c.text}
		if got := n.IsInteger(); got != c.want {
			t.Errorf("%q: IsInteger() = %v, want %v", c.text, got, c.want)
		}
	}
}

// Regression: `0xE` must not be classified as a float — `E` inside a hex
// literal is a hexadecimal digit, not an exponent marker. Getting this
// wrong makes callers who dispatch on IsInteger silently corrupt hex
// integer literals that end in E/e.
func TestNumberExprHexEndingInEIsInteger(t *testing.T) {
	n := &NumberExpr{Text: "0xE"}
	if !n.IsInteger() {
		t.Fatal("0xE must be classified as an integer")
	}
	got, err := n.Float()
	if err != nil {
		t.Fatalf("Float() error: %v", err)
	}
	if math.Abs(got-14) > 1e-9 {
		t.Errorf("Float() = %g, want 14", got)
	}
}
