package model

import (
	"fmt"
	"math"
)

// ChannelParams 单渠道超参
type ChannelParams struct {
	Name   string
	Lambda float64 // adstock 衰减率
	Alpha  float64 // Hill alpha
	Gamma  float64 // Hill gamma（半饱和点，自动从数据估计）
}

// FitConfig 训练配置
type FitConfig struct {
	Channels     []string
	Lambdas      []float64 // 网格搜索候选值
	Alphas       []float64
	RidgeLambda  float64
	ValFraction  float64 // 验证集比例
}

// DefaultFitConfig 默认超参网格
func DefaultFitConfig(channels []string) FitConfig {
	return FitConfig{
		Channels:    channels,
		Lambdas:     []float64{0.1, 0.3, 0.5, 0.7, 0.9},
		Alphas:      []float64{1.0, 2.0, 3.0},
		RidgeLambda: 1.0,
		ValFraction: 0.2,
	}
}

// FittedModel 训练后的模型
type FittedModel struct {
	Config      FitConfig
	BestParams  []ChannelParams
	Regression  RidgeRegression
	BaselineIdx int
	ValidationMAPE float64
}

// MMM 数据点：每期的支出和目标值
type DataPoint struct {
	Spends []float64 // 各渠道支出，顺序与 Config.Channels 一致
	Target float64   // 目标指标（如 revenue）
}

// Fit 网格搜索 + Ridge 回归训练 MMM 模型
func Fit(data []DataPoint, cfg FitConfig) (*FittedModel, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("数据量不足：至少需要 10 个周期")
	}
	nCh := len(cfg.Channels)
	if nCh == 0 {
		return nil, fmt.Errorf("渠道列表不能为空")
	}

	// 拆分训练/验证集
	valStart := int(float64(len(data)) * (1 - cfg.ValFraction))
	if valStart < 5 {
		valStart = 5
	}
	trainData := data[:valStart]
	valData := data[valStart:]

	bestMAPE := math.MaxFloat64
	var bestParams []ChannelParams

	// 为每个渠道独立网格搜索最优 lambda/alpha
	bestParams = make([]ChannelParams, nCh)
	for ch := 0; ch < nCh; ch++ {
		bestChMAPE := math.MaxFloat64
		bestLambda, bestAlpha := cfg.Lambdas[0], cfg.Alphas[0]

		for _, lam := range cfg.Lambdas {
			for _, alpha := range cfg.Alphas {
				mape := evaluateChannel(trainData, valData, ch, nCh, lam, alpha, cfg.RidgeLambda)
				if mape < bestChMAPE {
					bestChMAPE = mape
					bestLambda = lam
					bestAlpha = alpha
				}
			}
		}
		gamma := estimateGamma(trainData, ch, bestLambda)
		bestParams[ch] = ChannelParams{
			Name:   cfg.Channels[ch],
			Lambda: bestLambda,
			Alpha:  bestAlpha,
			Gamma:  gamma,
		}
	}

	// 用最优超参在全量数据训练最终模型
	X, y := buildFeatureMatrix(data, bestParams)
	reg := RidgeRegression{Lambda: cfg.RidgeLambda}
	if err := reg.Fit(X, y); err != nil {
		return nil, fmt.Errorf("最终回归训练失败: %w", err)
	}

	// 计算验证集 MAPE
	Xval, yval := buildFeatureMatrix(valData, bestParams)
	preds, _ := reg.Predict(Xval)
	bestMAPE = mapeSlice(preds, yval)

	return &FittedModel{
		Config:         cfg,
		BestParams:     bestParams,
		Regression:     reg,
		ValidationMAPE: bestMAPE,
	}, nil
}

// evaluateChannel 对单渠道参数组合评估验证集 MAPE
func evaluateChannel(train, val []DataPoint, chIdx, nCh int, lambda, alpha, ridgeLambda float64) float64 {
	params := make([]ChannelParams, nCh)
	for i := range params {
		params[i] = ChannelParams{Lambda: 0.5, Alpha: 1.0, Gamma: 1.0}
	}
	params[chIdx] = ChannelParams{
		Lambda: lambda,
		Alpha:  alpha,
		Gamma:  estimateGamma(train, chIdx, lambda),
	}

	X, y := buildFeatureMatrix(train, params)
	reg := RidgeRegression{Lambda: ridgeLambda}
	if err := reg.Fit(X, y); err != nil {
		return math.MaxFloat64
	}
	Xval, yval := buildFeatureMatrix(val, params)
	preds, err := reg.Predict(Xval)
	if err != nil {
		return math.MaxFloat64
	}
	return mapeSlice(preds, yval)
}

// buildFeatureMatrix 构建回归特征矩阵
func buildFeatureMatrix(data []DataPoint, params []ChannelParams) ([][]float64, []float64) {
	nCh := len(params)
	// 先提取每渠道支出序列
	spendSeries := make([][]float64, nCh)
	for ch := 0; ch < nCh; ch++ {
		spendSeries[ch] = make([]float64, len(data))
		for t, d := range data {
			spendSeries[ch][t] = d.Spends[ch]
		}
	}

	// adstock + saturation 变换
	transformed := make([][]float64, nCh)
	for ch, p := range params {
		adstocked := GeometricAdstock(spendSeries[ch], p.Lambda)
		transformed[ch] = BatchHillSaturation(adstocked, p.Alpha, p.Gamma)
	}

	// 组装 X, y
	X := make([][]float64, len(data))
	y := make([]float64, len(data))
	for t, d := range data {
		row := make([]float64, nCh)
		for ch := 0; ch < nCh; ch++ {
			row[ch] = transformed[ch][t]
		}
		X[t] = row
		y[t] = d.Target
	}
	return X, y
}

// estimateGamma 用均值作为半饱和点的简单估计
func estimateGamma(data []DataPoint, chIdx int, lambda float64) float64 {
	spends := make([]float64, len(data))
	for t, d := range data {
		spends[t] = d.Spends[chIdx]
	}
	adstocked := GeometricAdstock(spends, lambda)
	var sum float64
	for _, v := range adstocked {
		sum += v
	}
	if len(adstocked) == 0 {
		return 1.0
	}
	avg := sum / float64(len(adstocked))
	if avg <= 0 {
		return 1.0
	}
	return avg
}

func mapeSlice(preds, actual []float64) float64 {
	var mape float64
	count := 0
	for i, a := range actual {
		if a == 0 {
			continue
		}
		mape += math.Abs((a - preds[i]) / a)
		count++
	}
	if count == 0 {
		return 0
	}
	return mape / float64(count)
}
