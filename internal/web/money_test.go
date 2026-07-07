package web

import "testing"

func TestParseMoney(t *testing.T) {
	tests := map[string]int64{
		"19,99":    1999,
		"1.234,56": 123456,
		"5":        500,
		"0,7":      70,
		"":         0,
	}
	for input, want := range tests {
		got, err := ParseMoney(input)
		if err != nil {
			t.Fatalf("ParseMoney(%q) failed: %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseMoney(%q) = %d, want %d", input, got, want)
		}
	}
}
