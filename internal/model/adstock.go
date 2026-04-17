package model

// GeometricAdstock 几何递减 adstock 变换
// spend[t] 经过衰减 lambda 的历史累积效果
func GeometricAdstock(spend []float64, lambda float64) []float64 {
	if len(spend) == 0 {
		return nil
	}
	result := make([]float64, len(spend))
	result[0] = spend[0]
	for i := 1; i < len(spend); i++ {
		result[i] = spend[i] + lambda*result[i-1]
	}
	return result
}

// WeightedAdstock 加权 adstock，支持自定义衰减权重窗口
func WeightedAdstock(spend []float64, weights []float64) []float64 {
	if len(spend) == 0 || len(weights) == 0 {
		return nil
	}
	result := make([]float64, len(spend))
	for t := range spend {
		var sum float64
		for w, weight := range weights {
			if t-w >= 0 {
				sum += weight * spend[t-w]
			}
		}
		result[t] = sum
	}
	return result
}
