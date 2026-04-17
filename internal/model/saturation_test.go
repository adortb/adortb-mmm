package model

import (
	"math"
	"testing"
)

func TestHillSaturation_HalfPoint(t *testing.T) {
	// 当 x == gamma 时，Hill(x) = 0.5
	alpha, gamma := 2.0, 100.0
	got := HillSaturation(gamma, alpha, gamma)
	if math.Abs(got-0.5) > 1e-9 {
		t.Errorf("got %.6f, want 0.5", got)
	}
}

func TestHillSaturation_ZeroInput(t *testing.T) {
	got := HillSaturation(0, 2.0, 100.0)
	if got != 0 {
		t.Errorf("got %v, want 0", got)
	}
}

func TestHillSaturation_LargeInput(t *testing.T) {
	// 极大值时趋近 1
	got := HillSaturation(1e9, 2.0, 100.0)
	if got < 0.9999 {
		t.Errorf("got %.6f, want close to 1", got)
	}
}

func TestMichaelisMenten(t *testing.T) {
	// x == gamma 时 = 0.5
	gamma := 50.0
	got := MichaelisMentenSaturation(gamma, gamma)
	if math.Abs(got-0.5) > 1e-9 {
		t.Errorf("got %.6f, want 0.5", got)
	}
}

func TestBatchHillSaturation(t *testing.T) {
	xs := []float64{0, 100, 1e9}
	result := BatchHillSaturation(xs, 2.0, 100.0)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	if result[0] != 0 {
		t.Errorf("result[0]: got %v, want 0", result[0])
	}
	if math.Abs(result[1]-0.5) > 1e-9 {
		t.Errorf("result[1]: got %.6f, want 0.5", result[1])
	}
	if result[2] < 0.9999 {
		t.Errorf("result[2]: got %.6f, want close to 1", result[2])
	}
}
