package commands

import (
	"testing"
	"time"
)

func TestHumanDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "less than a minute"},
		{time.Minute, "1 minute"},
		{45 * time.Minute, "45 minutes"},
		{time.Hour, "1 hour"},
		{2*time.Hour + 20*time.Minute, "2 hours"},
		{25 * time.Hour, "1 day"},
		{6 * 24 * time.Hour, "6 days"},
		{40 * 24 * time.Hour, "1 month"},
		{400 * 24 * time.Hour, "1 year"},
	}
	for _, c := range cases {
		if got := humanDuration(c.d); got != c.want {
			t.Errorf("humanDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestDurationSince(t *testing.T) {
	if _, ok := durationSince(""); ok {
		t.Error("empty timestamp should not parse")
	}
	if _, ok := durationSince("not-a-time"); ok {
		t.Error("invalid timestamp should not parse")
	}
	d, ok := durationSince(time.Now().Add(-2 * time.Hour).Format(time.RFC3339))
	if !ok || d < time.Hour || d > 3*time.Hour {
		t.Errorf("expected ~2h, got %v (ok=%v)", d, ok)
	}
	// Rails ActiveSupport emits fractional seconds.
	if _, ok := durationSince("2026-07-17T10:48:11.000Z"); !ok {
		t.Error("fractional-second timestamp should parse")
	}
}
