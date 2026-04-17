package model

import "math"

// HillSaturation Hill 函数饱和曲线变换
// alpha: 形状参数（S-curve 陡峭度）
// gamma: 半饱和点（支出在此时效果为最大值的 50%）
func HillSaturation(x, alpha, gamma float64) float64 {
	if x <= 0 {
		return 0
	}
	xa := math.Pow(x, alpha)
	ga := math.Pow(gamma, alpha)
	return xa / (xa + ga)
}

// MichaelisMentenSaturation Michaelis-Menten 饱和曲线（alpha=1 的特例）
func MichaelisMentenSaturation(x, gamma float64) float64 {
	if x <= 0 {
		return 0
	}
	return x / (x + gamma)
}

// BatchHillSaturation 批量 Hill 变换
func BatchHillSaturation(xs []float64, alpha, gamma float64) []float64 {
	result := make([]float64, len(xs))
	for i, x := range xs {
		result[i] = HillSaturation(x, alpha, gamma)
	}
	return result
}
