package model

import (
	"math"
	"testing"
)

func TestRidgeRegression_KnownLinear(t *testing.T) {
	// y = 2 + 3*x1 + 5*x2（已知系数）
	X := [][]float64{
		{1, 2}, {2, 3}, {3, 4}, {4, 5}, {5, 6},
		{6, 7}, {7, 8}, {8, 9}, {9, 10}, {10, 11},
	}
	y := make([]float64, len(X))
	for i, row := range X {
		y[i] = 2 + 3*row[0] + 5*row[1]
	}

	reg := RidgeRegression{Lambda: 0.001}
	if err := reg.Fit(X, y); err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	// 预测应非常接近真实值
	preds, err := reg.Predict(X)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	for i, pred := range preds {
		if math.Abs(pred-y[i]) > 1.0 {
			t.Errorf("index %d: pred=%.4f, actual=%.4f", i, pred, y[i])
		}
	}
}

func TestRidgeRegression_EmptyData(t *testing.T) {
	reg := RidgeRegression{Lambda: 1.0}
	err := reg.Fit(nil, nil)
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestRidgeRegression_PredictUnfitted(t *testing.T) {
	reg := RidgeRegression{}
	_, err := reg.Predict([][]float64{{1, 2}})
	if err == nil {
		t.Error("expected error for unfitted model")
	}
}
