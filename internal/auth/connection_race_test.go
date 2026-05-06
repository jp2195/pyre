package auth_test

import (
	"sync"
	"testing"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/models"
)

// TestConnection_ConcurrentReadWriteFields probes the access pattern
// documented in the deep-dive review: dispatch.go writes
// conn.IsPanorama / conn.ManagedDevices in the Bubble Tea Update flow
// while Cmd goroutines (commands.go) read the same fields concurrently.
//
// Run with: go test -race ./internal/auth/
//
// If the race detector reports a data race here, follow up with
// locking in Task 3.2.
func TestConnection_ConcurrentReadWriteFields(t *testing.T) {
	conn := &auth.Connection{}

	devs := []models.ManagedDevice{{Serial: "0123456789", Hostname: "fw01"}}

	var wg sync.WaitGroup
	for range 200 {
		wg.Go(func() {
			conn.SetPanoramaInfo(true)
			conn.SetManagedDevices(devs)
		})
		wg.Go(func() {
			_ = conn.PanoramaInfo()
			_ = conn.ManagedDevicesSnapshot()
		})
	}
	wg.Wait()
}
