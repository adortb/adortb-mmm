package optimizer

import (
	"fmt"
	"math"

	"github.com/adortb/adortb-mmm/internal/model"
)

// OptimizeRequest 预算优化请求
type OptimizeRequest struct {
	FittedModel   *model.FittedModel
	History       []model.DataPoint
	HorizonWeeks  int
	Constraints   BudgetConstraints
}

// OptimizeResult 预算优化结果
type OptimizeResult struct {
	Allocation          map[string]float64
	ExpectedConversions float64
	LiftVsEqual         float64
}

const (
	defaultLR         = 0.01
	defaultIterations = 500
	convergenceEps    = 1e-6
)

// Optimize 投影梯度上升最大化预期转化
func Optimize(req OptimizeRequest) (*OptimizeResult, error) {
	m := req.FittedModel
	if m == nil {
		return nil, fmt.Errorf("模型不能为空")
	}
	channels := make([]string, len(m.BestParams))
	for i, p := range m.BestParams {
		channels[i] = p.Name
	}
	if err := req.Constraints.Validate(channels); err != nil {
		return nil, err
	}

	n := len(channels)
	// 初始分配：均匀
	alloc := make([]float64, n)
	for i := range alloc {
		alloc[i] = req.Constraints.TotalBudget / float64(n)
	}
	alloc = req.Constraints.Project(channels, alloc)

	// 等额分配基准
	equalAlloc := make([]float64, n)
	for i := range equalAlloc {
		equalAlloc[i] = req.Constraints.TotalBudget / float64(n)
	}
	equalConv, err := predictTotalConversion(m, req.History, equalAlloc, req.HorizonWeeks)
	if err != nil {
		return nil, err
	}

	lr := defaultLR * req.Constraints.TotalBudget / 100.0
	prevConv := 0.0

	for iter := 0; iter < defaultIterations; iter++ {
		grad := computeGradient(m, req.History, alloc, req.HorizonWeeks, lr*0.01)

		// 梯度上升
		newAlloc := make([]float64, n)
		for i := range alloc {
			newAlloc[i] = alloc[i] + lr*grad[i]
		}
		newAlloc = req.Constraints.Project(channels, newAlloc)

		conv, err := predictTotalConversion(m, req.History, newAlloc, req.HorizonWeeks)
		if err != nil {
			break
		}
		if math.Abs(conv-prevConv) < convergenceEps*conv {
			alloc = newAlloc
			break
		}
		alloc = newAlloc
		prevConv = conv
	}

	finalConv, err := predictTotalConversion(m, req.History, alloc, req.HorizonWeeks)
	if err != nil {
		return nil, err
	}

	allocation := make(map[string]float64, n)
	for i, ch := range channels {
		allocation[ch] = alloc[i]
	}

	var lift float64
	if equalConv > 0 {
		lift = (finalConv - equalConv) / equalConv
	}

	return &OptimizeResult{
		Allocation:          allocation,
		ExpectedConversions: finalConv,
		LiftVsEqual:         lift,
	}, nil
}

// predictTotalConversion 预测 horizonWeeks 周的总转化
func predictTotalConversion(m *model.FittedModel, history []model.DataPoint, alloc []float64, horizonWeeks int) (float64, error) {
	futureSpends := make([][]float64, horizonWeeks)
	for w := range futureSpends {
		row := make([]float64, len(alloc))
		copy(row, alloc)
		futureSpends[w] = row
	}
	preds, err := m.Predict(history, futureSpends)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, v := range preds {
		total += v
	}
	return total, nil
}

// computeGradient 数值梯度（有限差分）
func computeGradient(m *model.FittedModel, history []model.DataPoint, alloc []float64, horizonWeeks int, eps float64) []float64 {
	n := len(alloc)
	grad := make([]float64, n)
	baseConv, err := predictTotalConversion(m, history, alloc, horizonWeeks)
	if err != nil {
		return grad
	}
	for i := range alloc {
		perturbed := make([]float64, n)
		copy(perturbed, alloc)
		perturbed[i] += eps
		conv, err := predictTotalConversion(m, history, perturbed, horizonWeeks)
		if err != nil {
			continue
		}
		grad[i] = (conv - baseConv) / eps
	}
	return grad
}
