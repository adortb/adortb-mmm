package optimizer

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/adortb/adortb-mmm/internal/model"
)

func TestOptimize_Convergence(t *testing.T) {
	if testing.Short() {
		t.Skip("优化收敛测试跳过（-short 模式）")
	}

	channels := []string{"search", "display", "social"}
	data := generateOptimizerTestData(channels, 52)
	cfg := model.DefaultFitConfig(channels)
	fitted, err := model.Fit(data, cfg)
	if err != nil {
		t.Fatalf("Fit failed: %v", err)
	}

	req := OptimizeRequest{
		FittedModel:  fitted,
		History:      data,
		HorizonWeeks: 4,
		Constraints: BudgetConstraints{
			TotalBudget: 60000,
			PerChannelMin: map[string]float64{
				"search":  5000,
				"display": 3000,
				"social":  2000,
			},
		},
	}

	result, err := Optimize(req)
	if err != nil {
		t.Fatalf("Optimize failed: %v", err)
	}

	// 验证总预算约束
	var total float64
	for _, v := range result.Allocation {
		total += v
	}
	if math.Abs(total-60000) > 1.0 {
		t.Errorf("total allocation %.2f != budget 60000", total)
	}

	// 验证最小预算约束
	for ch, minBudget := range req.Constraints.PerChannelMin {
		if result.Allocation[ch] < minBudget-1e-6 {
			t.Errorf("channel %s allocation %.2f < min %.2f", ch, result.Allocation[ch], minBudget)
		}
	}

	// 验证期望转化为正数
	if result.ExpectedConversions <= 0 {
		t.Errorf("expected positive conversions, got %.4f", result.ExpectedConversions)
	}
}

func TestConstraints_Project(t *testing.T) {
	c := BudgetConstraints{
		TotalBudget: 100,
		PerChannelMin: map[string]float64{
			"a": 10,
			"b": 20,
		},
	}
	channels := []string{"a", "b", "c"}
	alloc := []float64{5, 5, 5} // 违反最小约束

	result := c.Project(channels, alloc)
	var total float64
	for _, v := range result {
		total += v
	}
	if math.Abs(total-100) > 1e-6 {
		t.Errorf("projected total %.4f != 100", total)
	}
	if result[0] < 10 {
		t.Errorf("channel a %.4f < min 10", result[0])
	}
	if result[1] < 20 {
		t.Errorf("channel b %.4f < min 20", result[1])
	}
}

func TestConstraints_Validate_InvalidBudget(t *testing.T) {
	c := BudgetConstraints{TotalBudget: -1}
	err := c.Validate([]string{"a"})
	if err == nil {
		t.Error("expected error for negative budget")
	}
}

func TestConstraints_Validate_MinExceedsBudget(t *testing.T) {
	c := BudgetConstraints{
		TotalBudget:   100,
		PerChannelMin: map[string]float64{"a": 60, "b": 60},
	}
	err := c.Validate([]string{"a", "b"})
	if err == nil {
		t.Error("expected error when min sum exceeds budget")
	}
}

func generateOptimizerTestData(channels []string, weeks int) []model.DataPoint {
	nCh := len(channels)
	data := make([]model.DataPoint, weeks)
	for t := 0; t < weeks; t++ {
		spends := make([]float64, nCh)
		target := 5000.0
		for i := range spends {
			spends[i] = 5000 + rand.Float64()*15000
			target += 0.3 * spends[i]
		}
		target += (rand.Float64() - 0.5) * 1000
		data[t] = model.DataPoint{Spends: spends, Target: target}
	}
	return data
}
