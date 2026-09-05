package views

import (
	"testing"
	"time"
)

func TestLogRange_SinceAndLabel(t *testing.T) {
	now := time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		rng       LogRange
		wantLabel string
		wantSince time.Time
	}{
		{LogRangeAll, "all", time.Time{}},
		{LogRange15m, "15m", now.Add(-15 * time.Minute)},
		{LogRange1h, "1h", now.Add(-time.Hour)},
		{LogRange24h, "24h", now.Add(-24 * time.Hour)},
		{LogRange7d, "7d", now.Add(-7 * 24 * time.Hour)},
	}
	for _, tt := range tests {
		t.Run(tt.wantLabel, func(t *testing.T) {
			if got := tt.rng.Label(); got != tt.wantLabel {
				t.Errorf("Label() = %q, want %q", got, tt.wantLabel)
			}
			if got := tt.rng.Since(now); !got.Equal(tt.wantSince) {
				t.Errorf("Since() = %v, want %v", got, tt.wantSince)
			}
		})
	}
}

// The default must be `all`: no bound, nothing hidden, which is exactly what
// the view did before server-side queries existed.
func TestLogsModel_DefaultRangeIsAll(t *testing.T) {
	m := NewLogsModel()
	if m.Range() != LogRangeAll {
		t.Errorf("default range = %v, want LogRangeAll", m.Range())
	}
	if !m.RangeSince().IsZero() {
		t.Errorf("default RangeSince() = %v, want zero", m.RangeSince())
	}
	if m.Query() != "" {
		t.Errorf("default Query() = %q, want empty", m.Query())
	}
}
