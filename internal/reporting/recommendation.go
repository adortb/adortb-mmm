package reporting

import (
	"fmt"
	"sort"

	"github.com/adortb/adortb-mmm/internal/optimizer"
)

// BudgetRecommendation 预算分配推荐报告
type BudgetRecommendation struct {
	Recommendations []ChannelBudget
	TotalBudget     float64
	ExpectedConversions float64
	LiftVsEqual     float64
}

// ChannelBudget 单渠道预算推荐
type ChannelBudget struct {
	Channel    string
	Budget     float64
	SharePct   float64
}

// GenerateRecommendation 生成预算推荐报告
func GenerateRecommendation(result *optimizer.OptimizeResult) (*BudgetRecommendation, error) {
	if result == nil {
		return nil, fmt.Errorf("优化结果不能为空")
	}

	var total float64
	for _, v := range result.Allocation {
		total += v
	}

	recs := make([]ChannelBudget, 0, len(result.Allocation))
	for ch, budget := range result.Allocation {
		var share float64
		if total > 0 {
			share = budget / total * 100.0
		}
		recs = append(recs, ChannelBudget{
			Channel:  ch,
			Budget:   budget,
			SharePct: share,
		})
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].Budget > recs[j].Budget
	})

	return &BudgetRecommendation{
		Recommendations:     recs,
		TotalBudget:         total,
		ExpectedConversions: result.ExpectedConversions,
		LiftVsEqual:         result.LiftVsEqual,
	}, nil
}
