package api

import (
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

func TestCertificateStatus(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name     string
		notAfter time.Time
		days     int
		status   string
	}{
		{"no expiry read", time.Time{}, 0, models.CertStatusUnknown},
		// The case truncation got wrong: -0.2 days used to become 0 and
		// read as "expiring, 0d".
		{"expired an hour ago", now.Add(-time.Hour), -1, models.CertStatusExpired},
		{"expires this instant", now, 0, models.CertStatusExpired},
		{"expires in an hour", now.Add(time.Hour), 0, models.CertStatusExpiring},
		{"29 days 23 hours", now.Add(30*24*time.Hour - time.Hour), 29, models.CertStatusExpiring},
		{"exactly 30 days", now.Add(30 * 24 * time.Hour), 30, models.CertStatusValid},
		{"a year", now.AddDate(1, 0, 0), 365, models.CertStatusValid},
	} {
		days, status := certificateStatus(tt.notAfter, now)
		if days != tt.days || status != tt.status {
			t.Errorf("%s: got %d %q, want %d %q", tt.name, days, status, tt.days, tt.status)
		}
	}
}
