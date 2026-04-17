package model

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestFit_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("端到端训练测试跳过（-short 模式）")
	}

	channels := []string{"search", "display", "social"}
	data := generateTestData(channels, 60)

	cfg := DefaultFitConfig(channels)
	cfg.ValFraction = 0.2

	fitted, err := Fit(data, cfg)
	if err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	if len(fitted.BestParams) != len(channels) {
		t.Errorf("expected %d params, got %d", len(channels), len(fitted.BestParams))
	}

	if fitted.ValidationMAPE > 0.5 {
		t.Errorf("validation MAPE %.4f too high", fitted.ValidationMAPE)
	}

	if fitted.Regression.R2 < 0.5 {
		t.Errorf("R2 %.4f too low", fitted.Regression.R2)
	}
}

func TestFit_InsufficientData(t *testing.T) {
	channels := []string{"search"}
	data := generateTestData(channels, 5)
	cfg := DefaultFitConfig(channels)
	_, err := Fit(data, cfg)
	if err == nil {
		t.Error("expected error for insufficient data")
	}
}

func TestFit_Predict(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	channels := []string{"search", "display"}
	data := generateTestData(channels, 52)
	cfg := DefaultFitConfig(channels)
	fitted, err := Fit(data, cfg)
	if err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	futureSpends := [][]float64{
		{10000, 5000},
		{12000, 6000},
		{8000, 4000},
		{11000, 5500},
	}
	preds, err := fitted.Predict(data, futureSpends)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}
	if len(preds) != len(futureSpends) {
		t.Errorf("expected %d predictions, got %d", len(futureSpends), len(preds))
	}
	for i, p := range preds {
		if math.IsNaN(p) || math.IsInf(p, 0) {
			t.Errorf("prediction[%d] is invalid: %v", i, p)
		}
	}
}

// generateTestData 生成带有已知线性关系的测试数据
func generateTestData(channels []string, weeks int) []DataPoint {
	nCh := len(channels)
	betas := make([]float64, nCh)
	for i := range betas {
		betas[i] = 0.3 + float64(i)*0.1
	}

	data := make([]DataPoint, weeks)
	for t := 0; t < weeks; t++ {
		spends := make([]float64, nCh)
		target := 5000.0
		for i := range spends {
			spends[i] = 5000 + rand.Float64()*15000
			target += betas[i] * spends[i]
		}
		target += (rand.Float64() - 0.5) * 1000
		data[t] = DataPoint{Spends: spends, Target: target}
	}
	return data
}
