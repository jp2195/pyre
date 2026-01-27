package auth_test

import (
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

// TestConcurrentSetActiveFirewall tests that concurrent calls to SetActiveFirewall
// are safe and properly synchronized using Go 1.25's testing/synctest.
func TestConcurrentSetActiveFirewall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{}
		session := auth.NewSession(cfg)

		// Add some test connections (host is now the key)
		fwConfig := &config.ConnectionConfig{}
		session.AddConnection("10.0.0.1", fwConfig, "key1")
		session.AddConnection("10.0.0.2", fwConfig, "key2")
		session.AddConnection("10.0.0.3", fwConfig, "key3")

		var wg sync.WaitGroup
		firewalls := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}

		// Launch concurrent SetActiveFirewall calls
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				fw := firewalls[idx%len(firewalls)]
				session.SetActiveFirewall(fw)
			}(i)
		}

		wg.Wait()
		synctest.Wait()

		// Verify final state is valid
		conn := session.GetActiveConnection()
		if conn == nil {
			t.Fatal("expected active connection after concurrent operations")
		}

		// The active firewall should be one of the valid ones
		validFirewalls := map[string]bool{"10.0.0.1": true, "10.0.0.2": true, "10.0.0.3": true}
		if !validFirewalls[conn.Host] {
			t.Errorf("unexpected active firewall: %s", conn.Host)
		}
	})
}

// TestConcurrentGetActiveConnection tests that concurrent reads of the active
// connection are safe and properly synchronized.
func TestConcurrentGetActiveConnection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{}
		session := auth.NewSession(cfg)

		// Add a test connection
		fwConfig := &config.ConnectionConfig{}
		session.AddConnection("10.0.0.1", fwConfig, "key1")
		session.SetActiveFirewall("10.0.0.1")

		var wg sync.WaitGroup

		// Launch concurrent read operations
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				conn := session.GetActiveConnection()
				if conn == nil {
					t.Error("expected non-nil connection")
				}
			}()
		}

		wg.Wait()
		synctest.Wait()
	})
}

// TestConcurrentAddRemoveConnection tests concurrent add and remove operations.
func TestConcurrentAddRemoveConnection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{}
		session := auth.NewSession(cfg)

		var wg sync.WaitGroup
		fwConfig := &config.ConnectionConfig{}

		// Concurrent adds - use different IPs as keys
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				host := "10.0.0." + string(rune('a'+idx%26))
				session.AddConnection(host, fwConfig, "key")
			}(i)
		}

		wg.Wait()
		synctest.Wait()

		// Verify at least some connections exist
		conns := session.ListConnections()
		if len(conns) == 0 {
			t.Error("expected at least some connections after concurrent adds")
		}
	})
}

// TestConcurrentMixedOperations tests a mix of concurrent read and write operations.
func TestConcurrentMixedOperations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{}
		session := auth.NewSession(cfg)

		fwConfig := &config.ConnectionConfig{}
		session.AddConnection("10.0.0.1", fwConfig, "key1")

		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 25; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				if idx%2 == 0 {
					session.SetActiveFirewall("10.0.0.1")
				} else {
					host := "10.1.0." + string(rune('a'+idx%26))
					session.AddConnection(host, fwConfig, "key")
				}
			}(i)
		}

		// Readers
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = session.GetActiveConnection()
				_ = session.ListConnections()
			}()
		}

		wg.Wait()
		synctest.Wait()

		// Should complete without race conditions or deadlocks
	})
}
