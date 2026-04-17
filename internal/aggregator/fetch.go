package aggregator

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/adortb/adortb-mmm/internal/model"
)

// SpendRow ClickHouse 原始行
type SpendRow struct {
	Date    time.Time
	Channel string
	Spend   float64
	Revenue float64
}

// Fetcher 从 ClickHouse 拉取历史数据
type Fetcher struct {
	db *sql.DB
}

// NewFetcher 创建 Fetcher
func NewFetcher(db *sql.DB) *Fetcher {
	return &Fetcher{db: db}
}

// FetchWeekly 拉取周粒度支出和目标数据
func (f *Fetcher) FetchWeekly(ctx context.Context, channels []string, from, to time.Time, targetMetric string) ([]model.DataPoint, error) {
	if len(channels) == 0 {
		return nil, fmt.Errorf("渠道列表不能为空")
	}

	// 构建 IN 子句
	inClause := ""
	args := []any{from, to}
	for i, ch := range channels {
		if i > 0 {
			inClause += ","
		}
		inClause += fmt.Sprintf("$%d", len(args)+1)
		args = append(args, ch)
	}

	query := fmt.Sprintf(`
		SELECT
			toMonday(date) AS week,
			channel,
			sum(spend) AS total_spend,
			sum(%s) AS total_target
		FROM ad_spend_metrics
		WHERE date >= $1 AND date < $2
		  AND channel IN (%s)
		GROUP BY week, channel
		ORDER BY week, channel
	`, targetMetric, inClause)

	rows, err := f.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ClickHouse 查询失败: %w", err)
	}
	defer rows.Close()

	// 按周聚合
	type weekKey = time.Time
	spendMap := make(map[weekKey]map[string]float64)
	targetMap := make(map[weekKey]float64)

	for rows.Next() {
		var week time.Time
		var channel string
		var spend, target float64
		if err := rows.Scan(&week, &channel, &spend, &target); err != nil {
			return nil, fmt.Errorf("扫描行失败: %w", err)
		}
		if spendMap[week] == nil {
			spendMap[week] = make(map[string]float64)
		}
		spendMap[week][channel] = spend
		targetMap[week] += target
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return buildDataPoints(spendMap, targetMap, channels), nil
}

// buildDataPoints 构建有序 DataPoint 序列
func buildDataPoints(spendMap map[time.Time]map[string]float64, targetMap map[time.Time]float64, channels []string) []model.DataPoint {
	weeks := make([]time.Time, 0, len(spendMap))
	for w := range spendMap {
		weeks = append(weeks, w)
	}
	// 按时间排序
	for i := 1; i < len(weeks); i++ {
		for j := i; j > 0 && weeks[j].Before(weeks[j-1]); j-- {
			weeks[j], weeks[j-1] = weeks[j-1], weeks[j]
		}
	}

	result := make([]model.DataPoint, 0, len(weeks))
	for _, w := range weeks {
		spends := make([]float64, len(channels))
		for ci, ch := range channels {
			spends[ci] = spendMap[w][ch]
		}
		result = append(result, model.DataPoint{
			Spends: spends,
			Target: targetMap[w],
		})
	}
	return result
}
