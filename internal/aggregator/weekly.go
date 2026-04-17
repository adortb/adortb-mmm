package aggregator

import (
	"sort"
	"time"

	"github.com/adortb/adortb-mmm/internal/model"
)

// DailyRecord 日粒度记录
type DailyRecord struct {
	Date    time.Time
	Channel string
	Spend   float64
	Target  float64
}

// AggregateWeekly 将日粒度记录聚合为周粒度 DataPoint 序列
func AggregateWeekly(records []DailyRecord, channels []string) []model.DataPoint {
	chIdx := make(map[string]int, len(channels))
	for i, ch := range channels {
		chIdx[ch] = i
	}

	type weekEntry struct {
		spends []float64
		target float64
	}
	weekMap := make(map[time.Time]*weekEntry)

	for _, r := range records {
		week := toMonday(r.Date)
		if weekMap[week] == nil {
			weekMap[week] = &weekEntry{spends: make([]float64, len(channels))}
		}
		if idx, ok := chIdx[r.Channel]; ok {
			weekMap[week].spends[idx] += r.Spend
		}
		weekMap[week].target += r.Target
	}

	weeks := make([]time.Time, 0, len(weekMap))
	for w := range weekMap {
		weeks = append(weeks, w)
	}
	sort.Slice(weeks, func(i, j int) bool { return weeks[i].Before(weeks[j]) })

	result := make([]model.DataPoint, 0, len(weeks))
	for _, w := range weeks {
		e := weekMap[w]
		result = append(result, model.DataPoint{
			Spends: e.spends,
			Target: e.target,
		})
	}
	return result
}

// toMonday 返回所在周的周一日期
func toMonday(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	offset := wd - 1
	monday := t.AddDate(0, 0, -offset)
	y, m, d := monday.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
