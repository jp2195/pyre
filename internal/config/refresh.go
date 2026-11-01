package config

import (
	"fmt"
	"time"
)

// Auto-refresh bounds. The floor exists for the firewall's sake, not
// pyre's: every refresh is a burst of XML API calls against the management
// plane, `show session all` among them, and a second-by-second poll is
// enough to slow the device's own UI down for everyone else using it.
const (
	MinRefreshInterval     = 10 * time.Second
	DefaultRefreshInterval = 30 * time.Second
)

// RefreshInterval parses settings.refresh_interval. An unset or zero value
// means auto-refresh starts off, and is reported as 0 with no error. A value
// below MinRefreshInterval is raised to it, and the returned note says so.
// A value that does not parse is an error, and auto-refresh starts off.
func (s Settings) RefreshInterval() (d time.Duration, note string, err error) {
	if s.RefreshIntervalRaw == "" {
		return 0, "", nil
	}
	d, err = time.ParseDuration(s.RefreshIntervalRaw)
	if err != nil {
		return 0, "", fmt.Errorf("settings.refresh_interval %q is not a duration like \"30s\" or \"2m\"", s.RefreshIntervalRaw)
	}
	switch {
	case d < 0:
		return 0, "", fmt.Errorf("settings.refresh_interval %q is negative", s.RefreshIntervalRaw)
	case d == 0:
		return 0, "", nil
	case d < MinRefreshInterval:
		return MinRefreshInterval, fmt.Sprintf("settings.refresh_interval %s is below the %s minimum; using %s",
			d, MinRefreshInterval, MinRefreshInterval), nil
	}
	return d, "", nil
}

// warnRefreshInterval reports a refresh_interval that will not be used as
// written, so a typo does not silently leave auto-refresh off.
func (s Settings) warnRefreshInterval() {
	_, note, err := s.RefreshInterval()
	switch {
	case err != nil:
		warnf("%v; auto-refresh starts off", err)
	case note != "":
		warnf("%s", note)
	}
}
