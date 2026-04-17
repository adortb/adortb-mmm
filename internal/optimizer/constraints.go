package optimizer

import "fmt"

// BudgetConstraints 预算约束
type BudgetConstraints struct {
	TotalBudget float64
	PerChannelMin map[string]float64
	PerChannelMax map[string]float64
}

// Validate 校验约束是否合法
func (c *BudgetConstraints) Validate(channels []string) error {
	if c.TotalBudget <= 0 {
		return fmt.Errorf("总预算必须大于 0")
	}
	var minSum float64
	for _, ch := range channels {
		minSum += c.PerChannelMin[ch]
	}
	if minSum > c.TotalBudget {
		return fmt.Errorf("各渠道最小预算之和 %.2f 超过总预算 %.2f", minSum, c.TotalBudget)
	}
	return nil
}

// Project 将分配向量投影到可行域：
// 1. 各渠道 >= min，<= max
// 2. 总和 == TotalBudget
func (c *BudgetConstraints) Project(channels []string, alloc []float64) []float64 {
	n := len(channels)
	result := make([]float64, n)
	copy(result, alloc)

	// Clip to [min, max]
	for i, ch := range channels {
		lo := c.PerChannelMin[ch]
		hi := c.TotalBudget
		if v, ok := c.PerChannelMax[ch]; ok {
			hi = v
		}
		if result[i] < lo {
			result[i] = lo
		}
		if result[i] > hi {
			result[i] = hi
		}
	}

	// 缩放使总和等于 TotalBudget
	var sum float64
	for _, v := range result {
		sum += v
	}
	if sum <= 0 {
		// 均匀分配
		for i := range result {
			result[i] = c.TotalBudget / float64(n)
		}
		return result
	}
	scale := c.TotalBudget / sum
	for i := range result {
		result[i] *= scale
	}

	// 再次 clip 后归一（迭代一次足够）
	var fixedSum float64
	var freeIdx []int
	for i, ch := range channels {
		lo := c.PerChannelMin[ch]
		if result[i] < lo {
			result[i] = lo
			fixedSum += lo
		} else {
			freeIdx = append(freeIdx, i)
		}
	}
	remaining := c.TotalBudget - fixedSum
	if len(freeIdx) > 0 && remaining > 0 {
		var freeSum float64
		for _, i := range freeIdx {
			freeSum += result[i]
		}
		if freeSum > 0 {
			for _, i := range freeIdx {
				result[i] = result[i] / freeSum * remaining
			}
		}
	}
	return result
}
