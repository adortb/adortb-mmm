package model

import "fmt"

// PredictRequest 预测请求
type PredictRequest struct {
	// ChannelSpends: channel name -> spend
	ChannelSpends map[string]float64
}

// PredictResult 预测结果
type PredictResult struct {
	PredictedConversion float64
	ChannelContributions map[string]float64
}

// Predict 给定各渠道支出预测转化，使用训练好的 FittedModel
// spendHistory: 历史支出序列（用于 adstock 的状态续接）
func (m *FittedModel) Predict(spendHistory []DataPoint, futureSpends [][]float64) ([]float64, error) {
	if len(m.BestParams) == 0 {
		return nil, fmt.Errorf("模型未训练")
	}
	nCh := len(m.BestParams)
	nFuture := len(futureSpends)
	if nFuture == 0 {
		return nil, fmt.Errorf("未来支出序列不能为空")
	}

	// 构建全量序列 = 历史 + 未来
	combined := make([]DataPoint, len(spendHistory)+nFuture)
	copy(combined, spendHistory)
	for i, sp := range futureSpends {
		if len(sp) != nCh {
			return nil, fmt.Errorf("支出维度不匹配：期望 %d 个渠道", nCh)
		}
		combined[len(spendHistory)+i] = DataPoint{Spends: sp}
	}

	X, _ := buildFeatureMatrix(combined, m.BestParams)
	futurePart := X[len(spendHistory):]

	preds, err := m.Regression.Predict(futurePart)
	if err != nil {
		return nil, err
	}
	return preds, nil
}

// ChannelContribution 计算各渠道在给定数据上的贡献度
// 贡献 = beta_ch * mean(transformed_ch)
func (m *FittedModel) ChannelContribution(data []DataPoint) (map[string]float64, error) {
	if len(m.BestParams) == 0 {
		return nil, fmt.Errorf("模型未训练")
	}

	X, _ := buildFeatureMatrix(data, m.BestParams)
	result := make(map[string]float64, len(m.BestParams))
	// coefs[0] 是截距，coefs[1+ch] 是各渠道系数
	for ch, p := range m.BestParams {
		if ch+1 >= len(m.Regression.Coefs) {
			continue
		}
		var meanX float64
		for _, row := range X {
			meanX += row[ch]
		}
		if len(X) > 0 {
			meanX /= float64(len(X))
		}
		result[p.Name] = m.Regression.Coefs[ch+1] * meanX
	}
	return result, nil
}
