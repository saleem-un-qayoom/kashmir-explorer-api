package booking

import "testing"

func TestComputeBase(t *testing.T) {
	cases := []struct {
		name   string
		unit   string
		price  int
		guests int
		span   int
		want   int
	}{
		{"per-person multiplies by guests", "per-person", 2000, 4, 1, 8000},
		{"per-person ignores span", "per-person", 2000, 3, 5, 6000},
		{"per-person clamps guests below 1", "per-person", 2000, 0, 1, 2000},
		{"per-night multiplies by span", "per-night", 5000, 2, 3, 15000},
		{"per-night clamps span below 1", "per-night", 5000, 2, 0, 5000},
		{"per-day multiplies by span", "per-day", 1500, 2, 4, 6000},
		{"per-trip is flat", "per-trip", 9000, 6, 5, 9000},
		{"per-hour is flat (no hours captured)", "per-hour", 700, 3, 2, 700},
		{"unknown unit is flat", "weird", 1234, 9, 9, 1234},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := computeBase(c.unit, c.price, c.guests, c.span); got != c.want {
				t.Errorf("computeBase(%q, price=%d, guests=%d, span=%d) = %d, want %d",
					c.unit, c.price, c.guests, c.span, got, c.want)
			}
		})
	}
}
