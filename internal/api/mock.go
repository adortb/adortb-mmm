package api

import (
	"math"
	"math/rand/v2"

	"github.com/adortb/adortb-mmm/internal/model"
)

// generateMockData 生成用于测试和演示的模拟 MMM 数据
func generateMockData(channels []string, weeks int) []model.DataPoint {
	nCh := len(channels)
	data := make([]model.DataPoint, weeks)

	// 每渠道固定 beta（真实效应）
	betas := make([]float64, nCh)
	for i := range betas {
		betas[i] = 0.3 + rand.Float64()*0.4
	}

	for t := 0; t < weeks; t++ {
		spends := make([]float64, nCh)
		for i := range spends {
			spends[i] = 5000 + rand.Float64()*20000
		}

		// 简单线性 + 噪声
		target := 10000.0 // baseline
		for i, sp := range spends {
			target += betas[i] * sp
		}
		// 季节性 + 噪声
		target += 2000 * math.Sin(float64(t)*2*math.Pi/52)
		target += (rand.Float64() - 0.5) * 5000

		data[t] = model.DataPoint{Spends: spends, Target: target}
	}
	return data
}
