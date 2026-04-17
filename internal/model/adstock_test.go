package model

import (
	"math"
	"testing"
)

func TestGeometricAdstock_Basic(t *testing.T) {
	spend := []float64{100, 0, 0, 0}
	result := GeometricAdstock(spend, 0.5)

	expected := []float64{100, 50, 25, 12.5}
	for i, v := range result {
		if math.Abs(v-expected[i]) > 1e-9 {
			t.Errorf("index %d: got %.4f, want %.4f", i, v, expected[i])
		}
	}
}

func TestGeometricAdstock_ZeroLambda(t *testing.T) {
	spend := []float64{100, 200, 300}
	result := GeometricAdstock(spend, 0)
	for i, v := range result {
		if math.Abs(v-spend[i]) > 1e-9 {
			t.Errorf("index %d: got %.4f, want %.4f", i, v, spend[i])
		}
	}
}

func TestGeometricAdstock_EmptySlice(t *testing.T) {
	result := GeometricAdstock(nil, 0.5)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestWeightedAdstock(t *testing.T) {
	spend := []float64{100, 200}
	weights := []float64{1.0, 0.5}
	result := WeightedAdstock(spend, weights)
	// t=0: 1.0*100 = 100
	// t=1: 1.0*200 + 0.5*100 = 250
	if math.Abs(result[0]-100) > 1e-9 {
		t.Errorf("index 0: got %.4f, want 100", result[0])
	}
	if math.Abs(result[1]-250) > 1e-9 {
		t.Errorf("index 1: got %.4f, want 250", result[1])
	}
}
