package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/jp2195/pyre/internal/models"
)

// TestCertificatesPanel_UnknownIsNotValid keeps a certificate whose expiry
// could not be read out of the valid count.
func TestCertificatesPanel_UnknownIsNotValid(t *testing.T) {
	InitStyles()
	m := NewDashboardModel().SetCertificates([]models.Certificate{
		{Name: "ok", Status: models.CertStatusValid},
		{Name: "mystery", Status: models.CertStatusUnknown},
	}, nil)
	out := ansi.Strip(m.renderCertificates(80))
	if !strings.Contains(out, "1 valid") || !strings.Contains(out, "1 unknown expiry") {
		t.Errorf("panel = %q, want 1 valid and 1 unknown expiry", out)
	}
}

// TestCertificatesPanel_MostUrgentFirst makes the six-row cut drop the least
// urgent certificates, not whichever came last in the config.
func TestCertificatesPanel_MostUrgentFirst(t *testing.T) {
	InitStyles()
	now := time.Now()
	certs := make([]models.Certificate, 0, 7)
	for i := range 7 {
		days := 25 - i*3 // 25, 22, ..., 7: config order is least urgent first
		certs = append(certs, models.Certificate{
			Name: "cert-" + string(rune('a'+i)), Status: models.CertStatusExpiring,
			DaysLeft: days, NotAfter: now.Add(time.Duration(days) * 24 * time.Hour),
		})
	}
	m := NewDashboardModel().SetCertificates(certs, nil)
	out := ansi.Strip(m.renderCertificates(80))

	if i, j := strings.Index(out, "cert-g"), strings.Index(out, "cert-f"); i < 0 || j < 0 || i > j {
		t.Errorf("most urgent (cert-g, 7d) not listed first:\n%s", out)
	}
	if strings.Contains(out, "cert-a") {
		t.Errorf("least urgent (cert-a, 25d) shown instead of cut:\n%s", out)
	}
	if !strings.Contains(out, "and 1 more") {
		t.Errorf("cut not reported:\n%s", out)
	}
}
