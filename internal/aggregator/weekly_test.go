package aggregator

import (
	"testing"
	"time"
)

func TestAggregateWeekly_Basic(t *testing.T) {
	channels := []string{"search", "display"}
	records := []DailyRecord{
		{Date: time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC), Channel: "search", Spend: 100, Target: 500},
		{Date: time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC), Channel: "search", Spend: 200, Target: 600},
		{Date: time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC), Channel: "display", Spend: 50, Target: 0},
		{Date: time.Date(2025, 1, 13, 0, 0, 0, 0, time.UTC), Channel: "search", Spend: 300, Target: 700},
	}

	result := AggregateWeekly(records, channels)
	if len(result) != 2 {
		t.Fatalf("expected 2 weeks, got %d", len(result))
	}

	// 第一周 search: 300, display: 50
	if result[0].Spends[0] != 300 {
		t.Errorf("week1 search spend: got %.0f, want 300", result[0].Spends[0])
	}
	if result[0].Spends[1] != 50 {
		t.Errorf("week1 display spend: got %.0f, want 50", result[0].Spends[1])
	}
	if result[0].Target != 1100 {
		t.Errorf("week1 target: got %.0f, want 1100", result[0].Target)
	}

	// 第二周 search: 300
	if result[1].Spends[0] != 300 {
		t.Errorf("week2 search spend: got %.0f, want 300", result[1].Spends[0])
	}
}

func TestToMonday(t *testing.T) {
	cases := []struct {
		input    time.Time
		expected time.Time
	}{
		{time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC)},  // Monday
		{time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC)},  // Tuesday
		{time.Date(2025, 1, 12, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC)}, // Sunday
	}
	for _, tc := range cases {
		got := toMonday(tc.input)
		if !got.Equal(tc.expected) {
			t.Errorf("toMonday(%v) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}
