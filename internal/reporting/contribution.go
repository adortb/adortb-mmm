package reporting

import (
	"fmt"
	"sort"

	"github.com/adortb/adortb-mmm/internal/model"
)

// ChannelContributionReport 渠道贡献度报告
type ChannelContributionReport struct {
	Contributions  []ChannelContrib
	TotalEffect    float64
	BaselineEffect float64
}

// ChannelContrib 单渠道贡献
type ChannelContrib struct {
	Channel    string
	AbsEffect  float64
	SharePct   float64
}

// GenerateContribution 生成各渠道贡献度报告
func GenerateContribution(m *model.FittedModel, data []model.DataPoint) (*ChannelContributionReport, error) {
	if m == nil {
		return nil, fmt.Errorf("模型不能为空")
	}

	contribs, err := m.ChannelContribution(data)
	if err != nil {
		return nil, err
	}

	// 基线 = 截距
	baseline := 0.0
	if len(m.Regression.Coefs) > 0 {
		baseline = m.Regression.Coefs[0]
	}

	var totalEffect float64
	for _, v := range contribs {
		if v > 0 {
			totalEffect += v
		}
	}

	result := make([]ChannelContrib, 0, len(contribs))
	for ch, effect := range contribs {
		var share float64
		if totalEffect > 0 {
			share = effect / totalEffect * 100.0
		}
		result = append(result, ChannelContrib{
			Channel:   ch,
			AbsEffect: effect,
			SharePct:  share,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].AbsEffect > result[j].AbsEffect
	})

	return &ChannelContributionReport{
		Contributions:  result,
		TotalEffect:    totalEffect,
		BaselineEffect: baseline,
	}, nil
}
