# Pyre Refinement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Address every bug, design issue, and modernization opportunity surfaced by the 2026-05-04 deep-dive review.

**Architecture:** Six independent phases ordered by value/risk. Each phase is self-contained and produces a PR-sized commit set, so phases can be merged separately. Phase 1 fixes verified bugs with TDD. Phase 2 is a mechanical modernization sweep. Phase 3 hardens concurrency around shared `auth.Connection` state. Phase 4 reduces drift in the TUI views layer. Phase 5 unblocks troubleshoot engine testing. Phase 6 collapses structural duplication in the logs API.

**Tech Stack:** Go 1.26.2, Bubble Tea v2 (`charm.land/{bubbletea,lipgloss,bubbles}/v2`), `go.yaml.in/yaml/v4`, `encoding/xml` with custom `xmlsafe` decoder.

**Verification cadence:** After every task, run `go test ./...` and `go vet ./...`. After every phase, run `go test -race -v ./...`.

---

## Phase 1 — Bug Fixes

Nine narrow fixes for verified bugs. Each lands its own commit. No dependencies between tasks within the phase.

---

### Task 1.1: Fix UTF-8 byte-slicing in InterfacesModel renderer

**Why:** `internal/tui/views/interfaces.go:359, 361` does `row[2:]` to strip the formatted status prefix. The prefix begins with `●` (3-byte UTF-8) or `○` (3-byte UTF-8), so byte offset 2 lands inside the UTF-8 sequence and emits an invalid byte to the terminal. Verified by reading the file. There is no existing test for `InterfacesModel`.

**Fix shape:** Stop pre-embedding the status indicator inside `formatInterfaceRow`. Have the renderer combine a separately-styled bullet with the unprefixed content row.

**Files:**
- Modify: `internal/tui/views/interfaces.go` (lines ~349–363, ~387–415)
- Create: `internal/tui/views/interfaces_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/views/interfaces_test.go`:

```go
package views

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jp2195/pyre/internal/models"
)

func TestInterfacesModel_RenderEmitsValidUTF8(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSize(120, 20)
	m = m.SetInterfaces([]models.Interface{
		{Name: "ethernet1/1", State: "up", Type: "layer3", Zone: "trust", IP: "10.0.0.1/24", MAC: "aa:bb:cc:dd:ee:ff", VirtualRouter: "default"},
		{Name: "ethernet1/2", State: "down", Type: "layer3", Zone: "untrust", IP: "", MAC: "", VirtualRouter: ""},
	})

	out := m.View()
	if !utf8.ValidString(out) {
		t.Fatalf("View() output contains invalid UTF-8\n--- output ---\n%s\n--- end ---", out)
	}
	if !strings.Contains(out, "ethernet1/1") {
		t.Errorf("expected 'ethernet1/1' in output, got: %s", out)
	}
	if !strings.Contains(out, "ethernet1/2") {
		t.Errorf("expected 'ethernet1/2' in output, got: %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/tui/views/ -run TestInterfacesModel_RenderEmitsValidUTF8 -v
```

Expected: FAIL with `View() output contains invalid UTF-8`. The `row[2:]` slice produces an invalid leading byte, so `utf8.ValidString` returns false.

- [ ] **Step 3: Apply the fix**

In `internal/tui/views/interfaces.go`, remove the `%-2s` status field from `formatInterfaceRow` (lines 387–416) and let the renderer prepend a styled bullet:

```go
func (m InterfacesModel) formatInterfaceRow(iface models.Interface, width int) string {
	ip := cleanValue(iface.IP)
	if ip == "" {
		ip = "—"
	}
	name := cleanValue(iface.Name)
	ifType := cleanValue(iface.Type)
	zone := cleanValue(iface.Zone)
	mac := cleanValue(iface.MAC)
	vr := cleanValue(iface.VirtualRouter)

	if width >= 120 {
		return fmt.Sprintf("%-16s %-10s %-12s %-18s %-17s %-12s",
			truncateStr(name, 16), truncateStr(ifType, 10),
			truncateStr(zone, 12), truncateStr(ip, 18), truncateStr(mac, 17),
			truncateStr(vr, 12))
	} else if width >= 90 {
		return fmt.Sprintf("%-14s %-8s %-10s %-16s %-12s",
			truncateStr(name, 14), truncateStr(ifType, 8),
			truncateStr(zone, 10), truncateStr(ip, 16), truncateStr(vr, 12))
	}
	return fmt.Sprintf("%-14s %-10s %-16s",
		truncateStr(name, 14), truncateStr(zone, 10), truncateStr(ip, 16))
}
```

Then in the renderer (lines 349–363), build the row from a styled bullet plus the unprefixed content:

```go
for i := m.Offset; i < end; i++ {
	iface := m.filtered[i]
	isSelected := i == m.Cursor

	bullet := "●"
	bulletStyle := upStyle
	if iface.State != "up" {
		bullet = "○"
		bulletStyle = downStyle
	}
	content := m.formatInterfaceRow(iface, availableWidth)

	if isSelected {
		// selectedStyle wraps the whole row including the bullet
		b.WriteString(selectedStyle.Render(bullet + " " + content))
	} else {
		b.WriteString(bulletStyle.Render(bullet) + " " + normalStyle.Render(content))
	}
	b.WriteString("\n")
}
```

Also update `formatHeaderRow` (lines 375–384) to keep its leading `St` column aligned with the rendered bullet. Drop the `%-2s` for the header's status column and prepend `"St "` literally:

```go
func (m InterfacesModel) formatHeaderRow(width int) string {
	if width >= 120 {
		return fmt.Sprintf("St %-16s %-10s %-12s %-18s %-17s %-12s",
			"Name", "Type", "Zone", "IP", "MAC", "VR")
	} else if width >= 90 {
		return fmt.Sprintf("St %-14s %-8s %-10s %-16s %-12s",
			"Name", "Type", "Zone", "IP", "VR")
	}
	return fmt.Sprintf("St %-14s %-10s %-16s",
		"Name", "Zone", "IP")
}
```

- [ ] **Step 4: Run test to verify it passes**

```
go test ./internal/tui/views/ -run TestInterfacesModel_RenderEmitsValidUTF8 -v
go test ./internal/tui/views/ -v
```

Expected: PASS for the new test, no regressions in existing view tests.

- [ ] **Step 5: Commit**

```
git add internal/tui/views/interfaces.go internal/tui/views/interfaces_test.go
git commit -m "fix(views): remove UTF-8 byte-slice in interfaces renderer

formatInterfaceRow embedded a 3-byte status indicator in a row that the
renderer then sliced at byte offset 2, leaking an invalid UTF-8 byte for
every non-selected interface row. Move the bullet out of the format
string and into the renderer so it can be styled independently without
slicing."
```

---

### Task 1.2: Wrap inner XML in parseRuleHitCounts

**Why:** `internal/api/policies.go:210` is the lone `decodeXML` call site in the package that does not call `WrapInner` on the inner bytes (verified: 28 wrapped call sites elsewhere in `internal/api/`). It works today only because `encoding/xml` adopts the first `StartElement` as the root. If PAN-OS prepends any sibling element, hit counts silently drop to zero.

**Files:**
- Modify: `internal/api/policies.go:198–224`
- Modify: `internal/api/policies_test.go` *(file may need creation if missing — check first)*

- [ ] **Step 1: Verify or create the test file**

```
ls internal/api/policies_test.go 2>&1 || echo "missing — will create"
```

If missing, create `internal/api/policies_test.go` with package declaration:

```go
package api

import (
	"testing"
	"time"
)
```

- [ ] **Step 2: Write the failing test**

Append to `internal/api/policies_test.go`:

```go
func TestParseRuleHitCounts_HandlesWrappedInner(t *testing.T) {
	// Inner bytes as PAN-OS returns them: <rule-hit-count> as the first element
	// with no document-level wrapper.
	inner := []byte(`<rule-hit-count>
		<vsys>
			<entry>
				<rule-base>
					<entry>
						<rules>
							<entry name="allow-web">
								<hit-count>42</hit-count>
								<last-hit-timestamp>1700000000</last-hit-timestamp>
								<first-hit-timestamp>1690000000</first-hit-timestamp>
								<last-reset-timestamp>0</last-reset-timestamp>
							</entry>
						</rules>
					</entry>
				</rule-base>
			</entry>
		</vsys>
	</rule-hit-count>`)

	got := parseRuleHitCounts(inner)
	if got == nil {
		t.Fatal("parseRuleHitCounts returned nil")
	}
	stats, ok := got["allow-web"]
	if !ok {
		t.Fatalf("expected entry for 'allow-web', got: %#v", got)
	}
	if stats.count != 42 {
		t.Errorf("count: want 42, got %d", stats.count)
	}
	if stats.lastHit.Equal(time.Time{}) {
		t.Errorf("lastHit should not be zero")
	}
}

func TestParseRuleHitCounts_HandlesPrecedingSibling(t *testing.T) {
	// Defensive: if PAN-OS ever prefixes the result with another element,
	// parsing must still succeed. Without WrapInner this returns nil.
	inner := []byte(`<status>ok</status><rule-hit-count>
		<vsys><entry><rule-base><entry><rules><entry name="rule-a">
			<hit-count>7</hit-count>
		</entry></rules></entry></rule-base></entry></vsys>
	</rule-hit-count>`)

	got := parseRuleHitCounts(inner)
	if got == nil || got["rule-a"].count != 7 {
		t.Fatalf("expected hit count for 'rule-a' = 7, got: %#v", got)
	}
}
```

- [ ] **Step 3: Run tests to verify failure**

```
go test ./internal/api/ -run TestParseRuleHitCounts -v
```

Expected: `TestParseRuleHitCounts_HandlesWrappedInner` may PASS today (coincidental); `TestParseRuleHitCounts_HandlesPrecedingSibling` FAILS with nil map.

- [ ] **Step 4: Apply the fix**

In `internal/api/policies.go`, change line 200–224:

```go
func parseRuleHitCounts(inner []byte) map[string]hitStats {
	var hitResult struct {
		Entry []struct {
			Name      string `xml:"name,attr"`
			HitCount  int64  `xml:"hit-count"`
			LastHit   string `xml:"last-hit-timestamp"`
			FirstHit  string `xml:"first-hit-timestamp"`
			LastReset string `xml:"last-reset-timestamp"`
		} `xml:"rule-hit-count>vsys>entry>rule-base>entry>rules>entry"`
	}
	if decodeXML(bytes.NewReader(WrapInner(inner)), &hitResult) != nil {
		return nil
	}

	hitMap := make(map[string]hitStats, len(hitResult.Entry))
	for _, h := range hitResult.Entry {
		hitMap[h.Name] = hitStats{
			count:     h.HitCount,
			lastHit:   parseUnixTimestamp(h.LastHit),
			firstHit:  parseUnixTimestamp(h.FirstHit),
			lastReset: parseUnixTimestamp(h.LastReset),
		}
	}
	return hitMap
}
```

The XPath stays the same — the wrapping element added by `WrapInner` is implicit because the struct's `xml:` tag starts at the immediate child level under the root.

- [ ] **Step 5: Run tests to verify pass**

```
go test ./internal/api/ -run TestParseRuleHitCounts -v
```

Expected: both tests PASS.

- [ ] **Step 6: Commit**

```
git add internal/api/policies.go internal/api/policies_test.go
git commit -m "fix(api): wrap inner XML in parseRuleHitCounts

parseRuleHitCounts was the lone decodeXML call site that did not pass
through WrapInner. It worked coincidentally because <rule-hit-count> was
always the leading element. WrapInner makes parsing robust to any
preceding sibling and matches every other call site in the package."
```

---

### Task 1.3: Poll log job before sleeping

**Why:** `internal/api/logs.go:77–82` waits a full `logPollInterval` (500 ms) before issuing the first `LogGet`. Tests already shrink the interval to compensate, indicating the delay is felt.

**Files:**
- Modify: `internal/api/logs.go:71–117`
- Modify: `internal/api/logs_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/api/logs_test.go`:

```go
func TestPollLogJob_FirstAttemptIsImmediate(t *testing.T) {
	// Stand up a mock that returns a "done" status on the first call.
	srv, calls := newImmediateLogServer(t)
	defer srv.Close()

	c := newTestClient(t, srv)
	// Use a deliberately long interval — if the loop sleeps before the first
	// poll, this test exceeds the 500ms budget.
	restore := setLogPollTimings(2*time.Second, 5)
	defer restore()

	start := time.Now()
	resp, err := c.pollLogJob(context.Background(), "1234", "")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("pollLogJob returned err: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil resp")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("first poll should be immediate; elapsed=%v (interval=2s)", elapsed)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 LogGet call on first-attempt success, got %d", calls.Load())
	}
}
```

If `newImmediateLogServer` and `setLogPollTimings` do not exist, add them as helpers in the same file. They should serve a minimal `<response status="success">…</response>` body containing a job in `FIN` state and expose a counter of received requests. Reference the existing `newTestClient` and `shrinkPollTimings` helpers and follow their style.

- [ ] **Step 2: Run test to verify it fails**

```
go test ./internal/api/ -run TestPollLogJob_FirstAttemptIsImmediate -v
```

Expected: FAIL with `elapsed=2.00…s` because the loop sleeps before polling.

- [ ] **Step 3: Apply the fix**

In `internal/api/logs.go:71–117`, restructure the loop so the poll happens first, the sleep happens between attempts:

```go
func (c *Client) pollLogJob(ctx context.Context, jobID, target string) (*XMLResponse, error) {
	const maxConsecErrors = 3
	interval := logPollInterval

	var consecErr int
	var lastErr error
	for attempt := 1; attempt <= logPollMaxAttempts; attempt++ {
		resp, err := c.LogGet(ctx, jobID, target)
		if err == nil {
			err = CheckResponse(resp)
		}
		if err != nil {
			consecErr++
			lastErr = err
			if consecErr >= maxConsecErrors {
				return nil, fmt.Errorf("log poll failed after %d consecutive errors: %w", consecErr, err)
			}
		} else {
			consecErr = 0
			switch status, raw := classifyJobStatus(resp); status {
			case logJobDone:
				return resp, nil
			case logJobFailed:
				return nil, fmt.Errorf("log job %s reported failure: %s", jobID, SanitizeForDisplay(raw))
			case logJobRunning:
				interval = min(interval*2, 2*time.Second)
			}
		}

		// Sleep between attempts (skipped after the final attempt).
		if attempt == logPollMaxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("log poll exhausted %d attempts; last error: %w", logPollMaxAttempts, lastErr)
	}
	return nil, fmt.Errorf("log poll timed out after %d attempts", logPollMaxAttempts)
}
```

Note: this also adopts `min()` builtin per CLAUDE.md style guidance, replacing the hand-rolled clamp at the previous lines 105–109.

- [ ] **Step 4: Run tests to verify pass**

```
go test ./internal/api/ -run TestPollLogJob -v
go test ./internal/api/ -v
```

Expected: new test PASSES; no regressions in `TestPollLogJob_*` already in the file.

- [ ] **Step 5: Commit**

```
git add internal/api/logs.go internal/api/logs_test.go
git commit -m "fix(api): poll log job before sleeping

pollLogJob slept the full interval before the first LogGet, adding
500ms to every log query even when the job was already finished. Move
the sleep between attempts. Also replace the hand-rolled min-clamp on
interval with the min() builtin."
```

---

### Task 1.4: Robust host normalization for env-var lookup

**Why:** `internal/auth/auth.go:193` only replaces `.` and `-`. A host with a port (`10.0.0.1:8443`) or IPv6 brackets (`[2001:db8::1]:8443`) produces an env-var name containing colons and brackets, which `os.Getenv` cannot match.

**Files:**
- Modify: `internal/auth/auth.go` (extract a normalize helper, call from line 193)
- Modify: `internal/auth/auth_test.go` (add tests)

- [ ] **Step 1: Write the failing tests**

Append to `internal/auth/auth_test.go`:

```go
func TestNormalizeHostForEnv(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"firewall.example.com", "FIREWALL_EXAMPLE_COM"},
		{"firewall.example.com:8443", "FIREWALL_EXAMPLE_COM"},
		{"10.0.0.1", "10_0_0_1"},
		{"10.0.0.1:8443", "10_0_0_1"},
		{"my-fw-01.example.com", "MY_FW_01_EXAMPLE_COM"},
		{"[2001:db8::1]:8443", "2001_DB8__1"},
		{"2001:db8::1", "2001_DB8__1"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			got := normalizeHostForEnv(tc.host)
			if got != tc.want {
				t.Errorf("normalizeHostForEnv(%q) = %q, want %q", tc.host, got, tc.want)
			}
		})
	}
}

func TestResolveCredentials_HostPortEnvVar(t *testing.T) {
	t.Setenv("PYRE_FIREWALL_EXAMPLE_COM_API_KEY", "secret-from-env")

	creds := ResolveCredentials(Flags{Host: "firewall.example.com:8443"})

	if creds.APIKey != "secret-from-env" {
		t.Errorf("expected env-var resolution despite :port, got APIKey=%q", creds.APIKey)
	}
	if creds.PromptForPassword {
		t.Error("PromptForPassword should be false when env-var resolves the key")
	}
}
```

(Adjust `Flags` field name and `ResolveCredentials` signature to match current code — verify by reading `internal/auth/auth.go` first.)

- [ ] **Step 2: Run tests to verify failure**

```
go test ./internal/auth/ -run "TestNormalizeHostForEnv|TestResolveCredentials_HostPortEnvVar" -v
```

Expected: `TestNormalizeHostForEnv` FAILS — function does not exist. `TestResolveCredentials_HostPortEnvVar` FAILS — env-var lookup misses because the env name is `FIREWALL_EXAMPLE_COM:8443` not `FIREWALL_EXAMPLE_COM`.

- [ ] **Step 3: Add the helper and use it**

In `internal/auth/auth.go`, near the existing imports add `"net"`. Add the helper near the bottom of the file:

```go
// normalizeHostForEnv converts a connection host into an env-var-safe
// suffix. Strips any :port (including bracketed IPv6 forms) and
// replaces ".", "-", and ":" with "_" before uppercasing.
func normalizeHostForEnv(host string) string {
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	r := strings.NewReplacer(".", "_", "-", "_", ":", "_")
	return strings.ToUpper(r.Replace(host))
}
```

Replace line 193:

```go
envName := normalizeHostForEnv(creds.Host)
```

- [ ] **Step 4: Run tests to verify pass**

```
go test ./internal/auth/ -v
```

Expected: all auth tests pass, including the new ones.

- [ ] **Step 5: Commit**

```
git add internal/auth/auth.go internal/auth/auth_test.go
git commit -m "fix(auth): handle :port and IPv6 in host env-var normalization

A host like 10.0.0.1:8443 or [2001:db8::1]:8443 produced an env-var name
containing illegal characters, so PYRE_<HOST>_API_KEY lookup silently
missed and pyre fell through to the password prompt. Use net.SplitHostPort
to strip ports and replace remaining colons before uppercasing."
```

---

### Task 1.5: Remove dead test block in policies_test.go

**Why:** `internal/tui/views/policies_test.go:19–21` has `if !m.HasData() == true {` with an empty body. The expression evaluates to `false == true` (always false), so the body never runs. The actual assertion on lines 22–24 covers the case correctly.

**Files:**
- Modify: `internal/tui/views/policies_test.go:19–21`

- [ ] **Step 1: Make the change**

In `internal/tui/views/policies_test.go`, delete lines 19–21 (the entire dead `if` block). The function should look like:

```go
func TestNewPoliciesModel(t *testing.T) {
	m := NewPoliciesModel()

	if m.list.Cursor != 0 {
		t.Errorf("expected Cursor=0, got %d", m.list.Cursor)
	}
	if m.HasData() {
		t.Error("expected HasData=false for new model")
	}
}
```

- [ ] **Step 2: Run tests to verify no regression**

```
go test ./internal/tui/views/ -run TestNewPoliciesModel -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```
git add internal/tui/views/policies_test.go
git commit -m "test(views): remove dead if-block in TestNewPoliciesModel

The 'if !m.HasData() == true' guard had an empty body and an expression
that always evaluates to false. The intended assertion is already covered
by the if-block immediately below."
```

---

### Task 1.6: Replace hardcoded color in dashboard_vpn.go

**Why:** `internal/tui/views/dashboard_vpn.go:158` calls `renderBar(upPct, barWidth, lipgloss.Color("#10B981"))` with a literal hex value, bypassing the theme system. Every other dashboard call uses `theme.Colors().Success` etc.

**Files:**
- Modify: `internal/tui/views/dashboard_vpn.go:158`

- [ ] **Step 1: Make the change**

In `internal/tui/views/dashboard_vpn.go` line 158:

```go
b.WriteString(renderBar(upPct, barWidth, theme.Colors().Success))
```

Verify the file already imports `github.com/jp2195/pyre/internal/tui/theme` (it should — other functions in the same file use it). If not, add the import.

- [ ] **Step 2: Run tests and vet**

```
go vet ./...
go test ./internal/tui/views/ -v
```

Expected: clean.

- [ ] **Step 3: Commit**

```
git add internal/tui/views/dashboard_vpn.go
git commit -m "fix(views): use theme color for VPN dashboard up-bar

Replace lipgloss.Color(\"#10B981\") literal with theme.Colors().Success
so the bar respects the active theme like every other dashboard."
```

---

### Task 1.7: Replace handrolled itoa with strconv.Itoa in command_palette

**Why:** `internal/tui/views/command_palette.go:344` defines a recursive `itoa` with a stale comment ("without importing strconv") even though the package already imports `strconv` elsewhere. The handrolled version overflows on `math.MinInt`.

**Files:**
- Modify: `internal/tui/views/command_palette.go`

- [ ] **Step 1: Audit current usage**

```
grep -n "\bitoa\b" internal/tui/views/command_palette.go
```

Note every call site so they can be updated.

- [ ] **Step 2: Replace call sites and remove the helper**

In `internal/tui/views/command_palette.go`:
- Add `"strconv"` to the import block if missing.
- Replace every `itoa(x)` call with `strconv.Itoa(x)`.
- Delete the `itoa` function definition near line 344 (and the comment above it).

- [ ] **Step 3: Run tests and vet**

```
go vet ./...
go test ./internal/tui/views/ -v
```

Expected: clean.

- [ ] **Step 4: Commit**

```
git add internal/tui/views/command_palette.go
git commit -m "refactor(views): replace handrolled itoa with strconv.Itoa

The handrolled itoa was recursive and overflowed on math.MinInt. The
package already imports strconv, so the comment claiming otherwise was
stale. Drop the helper and use strconv.Itoa directly."
```

---

### Task 1.8: Sanitize disk usage strings in GetDiskUsage

**Why:** `internal/api/monitoring.go:197` (approx) reads the `<disk-space>` op-command output as raw text and assigns directly into `models.DiskUsage` fields. Filesystem and mount-point names can carry attacker-controlled characters; every other server-string path runs through `SanitizeForDisplay` first.

**Files:**
- Modify: `internal/api/monitoring.go` (the `GetDiskUsage` function)
- Modify: `internal/api/monitoring_test.go` (create if missing)

- [ ] **Step 1: Write the failing test**

If `monitoring_test.go` is missing, create it. Append:

```go
func TestGetDiskUsage_SanitizesFields(t *testing.T) {
	// PAN-OS embeds df -h output as text in the XML result. Verify we
	// scrub ANSI/C0 control sequences before populating DiskUsage fields.
	rawDF := "Filesystem      Size  Used Avail Use% Mounted on\n" +
		"\x1b[31m/dev/sda1\x1b[0m  10G   2G   8G  20%  /var\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintf(w, `<response status="success"><result>%s</result></response>`, rawDF)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GetDiskUsage(context.Background(), "")
	if err != nil {
		t.Fatalf("GetDiskUsage err: %v", err)
	}
	for _, du := range got {
		if strings.ContainsAny(du.Filesystem, "\x1b\x07") {
			t.Errorf("Filesystem contains control sequence: %q", du.Filesystem)
		}
	}
}
```

(Adjust to the actual `GetDiskUsage` signature and `DiskUsage` field names by reading the file first.)

- [ ] **Step 2: Run test to verify failure**

```
go test ./internal/api/ -run TestGetDiskUsage_SanitizesFields -v
```

Expected: FAIL with `Filesystem contains control sequence`.

- [ ] **Step 3: Apply the fix**

In `GetDiskUsage` in `internal/api/monitoring.go`, after the existing column-split, run each candidate field through `SanitizeForDisplay` before assigning into the struct. Example shape:

```go
fields := strings.Fields(line)
if len(fields) < 6 {
	continue
}
out = append(out, models.DiskUsage{
	Filesystem: SanitizeForDisplay(fields[0]),
	Size:       SanitizeForDisplay(fields[1]),
	Used:       SanitizeForDisplay(fields[2]),
	Available:  SanitizeForDisplay(fields[3]),
	UsePercent: SanitizeForDisplay(fields[4]),
	MountPoint: SanitizeForDisplay(strings.Join(fields[5:], " ")),
})
```

- [ ] **Step 4: Run tests to verify pass**

```
go test ./internal/api/ -v
```

Expected: clean.

- [ ] **Step 5: Commit**

```
git add internal/api/monitoring.go internal/api/monitoring_test.go
git commit -m "fix(api): sanitize disk usage fields before TUI display

GetDiskUsage spliced df -h output directly into DiskUsage struct fields,
bypassing SanitizeForDisplay. Run every field through the sanitizer to
match every other server-string path in the package."
```

---

### Task 1.9: Fix ConnectionHubModel cursor mismatch

**Why:** `internal/tui/views/connection_hub.go:215` (approx) renders by splitting `m.connections` into `firewalls` and `panoramas` slices and tracking a `pos` counter, but `m.cursor` indexes the original sorted slice. When a Panorama is interleaved with firewalls in the recency-sorted list, `pos == m.cursor` highlights the wrong row.

**Files:**
- Modify: `internal/tui/views/connection_hub.go`
- Create: `internal/tui/views/connection_hub_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/tui/views/connection_hub_test.go`:

```go
package views

import (
	"strings"
	"testing"
	"time"

	"github.com/jp2195/pyre/internal/auth"
	"github.com/jp2195/pyre/internal/config"
)

func TestConnectionHubModel_CursorOnInterleavedPanorama(t *testing.T) {
	now := time.Now()
	conns := []auth.Connection{
		{Config: config.ConnectionConfig{Host: "fw-1"},   IsPanorama: false, LastConnected: now.Add(-1 * time.Hour)},
		{Config: config.ConnectionConfig{Host: "panorama"}, IsPanorama: true,  LastConnected: now.Add(-2 * time.Hour)},
		{Config: config.ConnectionConfig{Host: "fw-2"},   IsPanorama: false, LastConnected: now.Add(-3 * time.Hour)},
	}

	m := NewConnectionHubModel().SetConnections(conns)
	m.SetSize(120, 30)
	m.cursor = 1 // expect "panorama" to be highlighted

	out := m.View()

	// The selected entry must be 'panorama' regardless of the firewall/panorama split.
	lines := strings.Split(out, "\n")
	var selectedLine string
	for _, line := range lines {
		if strings.Contains(line, "▶") || strings.Contains(line, ">") {
			selectedLine = line
			break
		}
	}
	if !strings.Contains(selectedLine, "panorama") {
		t.Errorf("expected 'panorama' to be highlighted with cursor=1, got selected line: %q", selectedLine)
	}
}
```

(Adapt selection-marker check to whatever the view actually uses — read the file to confirm the marker character/style.)

- [ ] **Step 2: Run test to verify failure**

```
go test ./internal/tui/views/ -run TestConnectionHubModel_CursorOnInterleavedPanorama -v
```

Expected: FAIL — wrong row is highlighted.

- [ ] **Step 3: Apply the fix**

Two clean approaches; pick whichever is less invasive:

**Option A — Render in the original sorted order, no split.** Drop the `firewalls` / `panoramas` slices and iterate `m.connections` directly. Use a single `[Panorama]` / `[Firewall]` tag column instead of two sections. The cursor naturally aligns.

**Option B — Track the original index.** When building each section, record the original index alongside each entry: `type rowEntry struct { conn auth.Connection; origIdx int }`. Compare `entry.origIdx == m.cursor` for selection.

Option A is cleaner; Option B is the smaller diff. Choose Option A unless the two-section visual is load-bearing.

- [ ] **Step 4: Run tests to verify pass**

```
go test ./internal/tui/views/ -v
```

Expected: clean.

- [ ] **Step 5: Commit**

```
git add internal/tui/views/connection_hub.go internal/tui/views/connection_hub_test.go
git commit -m "fix(views): align connection hub cursor with sorted slice

ConnectionHubModel rendered by splitting connections into firewalls and
panoramas while m.cursor indexed the original sorted slice. Interleaved
entries highlighted the wrong row. Render in the original order so cursor
position maps directly."
```

---

### Task 1.10: Atomic write for the config backup file

**Why:** `internal/config/config.go:123` writes `~/.pyre.yaml.bak` with `os.WriteFile` (truncate-overwrite). If the process is killed mid-write, the backup is corrupt. The same file already defines `atomicWriteFile`; it just isn't used here.

**Files:**
- Modify: `internal/config/config.go:123`

- [ ] **Step 1: Make the change**

In `internal/config/config.go` near line 123, replace the direct `os.WriteFile(backupPath, data, 0600)` call with `atomicWriteFile(backupPath, data, 0600)`. Match the call signature already used elsewhere in the same file.

- [ ] **Step 2: Verify**

```
go test ./internal/config/ -v
```

Expected: clean.

- [ ] **Step 3: Commit**

```
git add internal/config/config.go
git commit -m "fix(config): atomic write for ~/.pyre.yaml.bak

The backup file was written with os.WriteFile (truncate-overwrite), so
a crash during the write left a corrupt backup. Use the existing
atomicWriteFile helper to match the primary file's write semantics."
```

---

## Phase 2 — Modernization Sweep

Mechanical changes confirmed safe by `go fix -diff` and CLAUDE.md style guidance. One commit per logical group.

---

### Task 2.1: Apply `go fix ./...`

**Why:** `go fix -diff ./...` confirmed clean diffs across 8 files: `for i := 0; i < N; i++` → `for i := range N`, manual map-copy → `maps.Copy`, hand-rolled clamps → `max`/`min` builtins, drop redundant loop-var rebinding (Go 1.22+).

**Files (per `go fix -diff`):**
- Modify: `internal/troubleshoot/patterns.go`
- Modify: `internal/api/client_internal_test.go`
- Modify: `internal/tui/views/dashboard_config.go`
- Modify: `internal/tui/views/dashboard_network.go`
- Modify: `internal/tui/views/dashboard_security.go`
- Modify: `internal/tui/views/dashboard_vpn.go`
- Modify: `internal/tui/views/interfaces.go`
- Modify: `internal/tui/views/routes.go`
- Modify: `internal/tui/styles.go`

- [ ] **Step 1: Preview the diff**

```
go fix -diff ./...
```

Expected: a diff containing only the modernizations listed above. If anything else appears, stop and re-evaluate.

- [ ] **Step 2: Apply**

```
go fix ./...
```

- [ ] **Step 3: Verify build, vet, tests**

```
go build ./cmd/pyre
go vet ./...
go test ./...
```

Expected: all green.

- [ ] **Step 4: Commit**

```
git add -A
git commit -m "refactor: apply go fix modernizations

go fix -diff confirmed safe transforms:
- for i := 0; i < N; i++ -> for i := range N (8 dashboards/views)
- hand-rolled clamps -> max/min builtins (styles.go, routes.go)
- map for-loop copy -> maps.Copy (troubleshoot/patterns.go)
- drop redundant loop-var rebinding (client_internal_test.go)"
```

---

### Task 2.2: Modernize errors.As to errors.AsType[T]

**Why:** CLAUDE.md prescribes the Go 1.26 generic form. `internal/auth/keygen.go:117, 139` use the older `errors.As(err, &target)` pattern.

**Files:**
- Modify: `internal/auth/keygen.go`

- [ ] **Step 1: Make the change**

At `internal/auth/keygen.go:117`, replace:

```go
var keygenErr *KeygenError
if errors.As(err, &keygenErr) { ... }
```

with:

```go
if keygenErr, ok := errors.AsType[*KeygenError](err); ok { ... }
```

Repeat at line 139. Read the surrounding context to ensure the variable name and use-of-`keygenErr` carry over correctly (the `if` body uses `keygenErr` — keep that name).

- [ ] **Step 2: Verify build and tests**

```
go vet ./...
go test ./internal/auth/ -v
```

Expected: clean.

- [ ] **Step 3: Commit**

```
git add internal/auth/keygen.go
git commit -m "refactor(auth): use errors.AsType[T] over errors.As

Go 1.26 generic form prescribed in CLAUDE.md. No behavior change."
```

---

## Phase 3 — Concurrency Audit

Confirms whether the `auth.Connection` field mutations in `dispatch.go:104, 113` produce a race against concurrent Cmd goroutines. If yes, lock the connection.

---

### Task 3.1: Add a race-detector test for concurrent Connection access

**Why:** `internal/tui/dispatch.go:104, 113` write `conn.IsPanorama` and `conn.ManagedDevices` while Cmd goroutines (`commands.go:151`) read the same fields. Bubble Tea serializes Update, but Cmds run concurrently. If two Cmds touching the same `*auth.Connection` overlap, that's a race.

**Files:**
- Create: `internal/auth/connection_race_test.go`

- [ ] **Step 1: Write the test**

Create `internal/auth/connection_race_test.go`:

```go
package auth_test

import (
	"sync"
	"testing"

	"github.com/jp2195/pyre/internal/auth"
)

// Run with: go test -race ./internal/auth/
func TestConnection_ConcurrentReadWriteFields(t *testing.T) {
	conn := &auth.Connection{}

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			// Mimic the dispatch.go writer.
			conn.IsPanorama = true
		})
		wg.Go(func() {
			// Mimic a concurrent Cmd reader (commands.go:151).
			_ = conn.IsPanorama
			_ = conn.ManagedDevices
		})
	}
	wg.Wait()
}
```

- [ ] **Step 2: Run with the race detector**

```
go test -race ./internal/auth/ -run TestConnection_ConcurrentReadWriteFields -v
```

Expected outcomes:
- **If `Connection` already has internal locking:** PASS, no race report. Phase 3 ends here; mark remaining tasks N/A.
- **If unlocked:** the race detector reports `WARNING: DATA RACE`. Proceed to Task 3.2.

- [ ] **Step 3: Commit the test regardless of outcome**

```
git add internal/auth/connection_race_test.go
git commit -m "test(auth): regression test for concurrent Connection field access"
```

---

### Task 3.2: Add a mutex to auth.Connection

**Skip if Task 3.1 reported no race.**

**Why:** Cmd goroutines and Update both touch `conn.IsPanorama`/`conn.ManagedDevices`. Guard with a `sync.RWMutex` and accessor methods.

**Files:**
- Modify: `internal/auth/auth.go` (or wherever `Connection` is defined — confirm path first)
- Modify: `internal/tui/dispatch.go:101–118`
- Modify: `internal/tui/commands.go:151, 161, 164` and any other read site
- Modify: `internal/tui/render.go:21, 141`
- Modify: `internal/tui/app.go:281, 284`

- [ ] **Step 1: Add accessors on Connection**

```go
type Connection struct {
	// existing fields...
	mu sync.RWMutex
}

func (c *Connection) SetPanorama(is bool, model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.IsPanorama = is
	c.Model = model
}

func (c *Connection) Panorama() (bool, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.IsPanorama, c.Model
}

func (c *Connection) SetManagedDevices(devs []models.ManagedDevice) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ManagedDevices = devs
}

func (c *Connection) ManagedDevicesSnapshot() []models.ManagedDevice {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]models.ManagedDevice, len(c.ManagedDevices))
	copy(out, c.ManagedDevices)
	return out
}
```

(Adapt field/method names to actual types in the codebase.)

- [ ] **Step 2: Update every read/write site**

Replace direct field access with the accessors:

- `dispatch.go:104`: `conn.SetPanorama(msg.IsPanorama, msg.Model)`
- `dispatch.go:113`: `conn.SetManagedDevices(msg.Devices)`
- `dispatch.go:114`: `if isPano, _ := conn.Panorama(); m.currentView == ViewDashboard && isPano && conn.TargetSerial == "" { … }`
- `commands.go:151`: `conn.ManagedDevicesSnapshot()`
- `commands.go:161, 164`: pure construction — unchanged
- `render.go:21, 141`: `conn.Panorama()` returns
- `app.go:281, 284`: same

- [ ] **Step 3: Re-run race test and full suite**

```
go test -race ./...
```

Expected: clean.

- [ ] **Step 4: Commit**

```
git add -A
git commit -m "fix(auth): guard Connection with RWMutex

dispatch.go mutated conn.IsPanorama and conn.ManagedDevices in Update
while Cmd goroutines read the same fields. Add accessor methods backed
by a sync.RWMutex. Regression-tested by TestConnection_ConcurrentReadWriteFields."
```

---

## Phase 4 — TUI Refactor

Reduce duplication and inconsistency in the views layer. Each task is a self-contained refactor that should leave existing tests green.

---

### Task 4.1: Make `setError` a value-returning helper

**Why:** `setError` is `func (m *Model) setError(err error) tea.Cmd`. Today every caller follows it with `return m, ...` or stores its Cmd in a slice and returns `m` afterward, so the local-copy mutation survives. Any future caller that returns a different model loses the error silently. Convert to value semantics to remove the footgun.

**Files:**
- Modify: `internal/tui/app.go:152–158`
- Modify: `internal/tui/handlers.go:99` (and any other call site)
- Modify: `internal/tui/dispatch.go:278, 282, 285`

- [ ] **Step 1: Audit call sites**

```
grep -rn "\.setError\b\|setError(" internal/tui/*.go
```

Note every call site.

- [ ] **Step 2: Change the signature**

In `internal/tui/app.go:152–158`:

```go
// setError returns a copy of the model with the error set, plus a Cmd
// that auto-dismisses the error after errorDismissTimeout.
func (m Model) setError(err error) (Model, tea.Cmd) {
	m.err = err
	return m, tea.Tick(errorDismissTimeout, func(time.Time) tea.Msg {
		return ErrorDismissMsg{}
	})
}
```

- [ ] **Step 3: Update each call site**

In `internal/tui/handlers.go:99`:

```go
m, cmd := m.setError(err)
return m, cmd
```

In `internal/tui/dispatch.go:278, 282, 285`, the existing pattern accumulates Cmds:

```go
case ConfigSavedMsg:
	if msg.Err != nil {
		var cmd tea.Cmd
		m, cmd = m.setError(msg.Err)
		cmds = append(cmds, cmd)
	}
```

Apply the same pattern at each call site.

- [ ] **Step 4: Verify build, vet, tests**

```
go vet ./...
go test ./...
```

Expected: clean. Existing tests should still pass since the observable behavior (error appears in the model, dismiss Cmd fires) is identical.

- [ ] **Step 5: Commit**

```
git add internal/tui/app.go internal/tui/handlers.go internal/tui/dispatch.go
git commit -m "refactor(tui): make setError value-returning

setError was a pointer-receiver method whose mutation only survived
because every existing caller returned the same local m. Switch to
(Model, tea.Cmd) so future callers can't silently drop the error."
```

---

### Task 4.2: Unify dashboard switch messages

**Why:** `internal/tui/messages.go:127–139` defines `DashboardSelectedMsg` and `SwitchDashboardMsg` that both set `m.currentDashboard` and re-fetch. Only the latter calls `syncNavbarToCurrentView()`. One unified message removes the fork.

**Files:**
- Modify: `internal/tui/messages.go`
- Modify: `internal/tui/dispatch.go:212–225`
- Modify: any sender (search the codebase)

- [ ] **Step 1: Find all senders**

```
grep -rn "DashboardSelectedMsg\|SwitchDashboardMsg" internal/
```

- [ ] **Step 2: Pick one canonical message**

Keep `SwitchDashboardMsg` (the one that syncs the navbar). Delete `DashboardSelectedMsg` from `messages.go`. In `dispatch.go`, remove the `DashboardSelectedMsg` arm and let `SwitchDashboardMsg` handle both flows. Update every sender to send `SwitchDashboardMsg`.

- [ ] **Step 3: Verify build, vet, tests**

```
go vet ./...
go test ./...
```

Expected: clean.

- [ ] **Step 4: Commit**

```
git add internal/tui/messages.go internal/tui/dispatch.go internal/tui/views/
git commit -m "refactor(tui): unify dashboard switch messages

DashboardSelectedMsg and SwitchDashboardMsg did the same work; only the
latter synced the navbar. Drop DashboardSelectedMsg and route every
caller through SwitchDashboardMsg."
```

---

### Task 4.3: Add esc handling to InterfacesModel

**Why:** Every other table view (`sessions`, `rule_list`, `ipsec_tunnels`) handles `"esc"` to collapse the detail panel and clear the filter. `InterfacesModel.Update` (around `internal/tui/views/interfaces.go:176`) silently ignores `esc` in normal mode.

**Files:**
- Modify: `internal/tui/views/interfaces.go`
- Modify: `internal/tui/views/interfaces_test.go` (created in Task 1.1)

- [ ] **Step 1: Write the failing test**

Append to `internal/tui/views/interfaces_test.go`:

```go
func TestInterfacesModel_EscClearsFilterAndCollapsesDetail(t *testing.T) {
	m := NewInterfacesModel()
	m = m.SetSize(120, 20)
	m = m.SetInterfaces([]models.Interface{{Name: "ethernet1/1", State: "up"}})
	m.Filter = "eth"
	m.Expanded = true

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	got := updated.(InterfacesModel)

	if got.Filter != "" {
		t.Errorf("expected filter cleared, got %q", got.Filter)
	}
	if got.Expanded {
		t.Error("expected detail panel collapsed")
	}
}
```

(Verify field names `Filter` / `Expanded` against actual `InterfacesModel` definition; adjust if the view exposes these via methods rather than fields.)

- [ ] **Step 2: Run test to verify failure**

```
go test ./internal/tui/views/ -run TestInterfacesModel_EscClearsFilterAndCollapsesDetail -v
```

Expected: FAIL.

- [ ] **Step 3: Add the esc arm**

In `InterfacesModel.Update`, mirror the pattern from `sessions.go:206`:

```go
case "esc":
	m = m.HandleCollapseIfExpanded()
	m = m.HandleClearFilter()
	return m, nil
```

(Verify those helper methods exist on `TableBase` and return the receiver type. If they're defined elsewhere, adapt accordingly.)

- [ ] **Step 4: Verify pass**

```
go test ./internal/tui/views/ -v
```

- [ ] **Step 5: Commit**

```
git add internal/tui/views/interfaces.go internal/tui/views/interfaces_test.go
git commit -m "fix(views): handle esc in InterfacesModel

Mirror sessions.go behavior: esc collapses the detail panel and clears
the filter. Previously InterfacesModel silently ignored esc."
```

---

### Task 4.4: Adopt TableBase.VisibleRows in concrete views

**Why:** `sessions`, `interfaces`, `ipsec_tunnels`, `rule_list`, `routes` each open-code `visibleRows()` instead of using the existing `TableBase.VisibleRows(overhead, expandedOverhead int)` method.

**Files:**
- Modify: `internal/tui/views/sessions.go:257`
- Modify: `internal/tui/views/interfaces.go:154`
- Modify: `internal/tui/views/ipsec_tunnels.go:189`
- Modify: `internal/tui/views/rule_list.go:143`
- Modify: `internal/tui/views/routes.go` (the `visibleRows` method)

- [ ] **Step 1: Inspect the TableBase method**

```
grep -n "func (.*TableBase).*VisibleRows" internal/tui/views/table_base.go
```

Confirm signature. Expected something like `func (t TableBase) VisibleRows(overhead, expandedOverhead int) int`.

- [ ] **Step 2: Replace each visibleRows method with a delegation**

For each file, replace the local `visibleRows()` body with a call. Example for `sessions.go:257`:

```go
func (m SessionsModel) visibleRows() int {
	return m.TableBase.VisibleRows(8, 8)
}
```

For views with different overhead per expanded state, pass the appropriate values (per the previous deep-dive, `interfaces`/`ipsec`/`rule_list` use 8 normal, 14 expanded).

If `TableBase.VisibleRows` already returns a min of 1, you can drop the local clamp; otherwise leave the clamp inside the helper.

- [ ] **Step 3: Verify build and tests**

```
go test ./internal/tui/views/ -v
```

- [ ] **Step 4: Commit**

```
git add internal/tui/views/
git commit -m "refactor(views): use TableBase.VisibleRows in concrete views

sessions, interfaces, ipsec_tunnels, rule_list, and routes each
duplicated the visible-rows arithmetic. Delegate to TableBase."
```

---

### Task 4.5: Consolidate number formatters

**Why:** `helpers.go:34 formatNumberWithCommas`, `dashboard_helpers.go:88 formatNumber`, and `format_helpers.go:122 formatHitCountFull` are three near-identical implementations of comma-grouped int64 formatting.

**Files:**
- Modify: `internal/tui/views/format_helpers.go`
- Modify: `internal/tui/views/helpers.go`
- Modify: `internal/tui/views/dashboard_helpers.go`
- Modify: any caller of the deprecated names

- [ ] **Step 1: Identify the canonical implementation**

Read all three. The cleanest signature wins. Default to keeping `formatHitCountFull` in `format_helpers.go` but rename it to `formatInt64WithCommas` (or just reuse `formatNumberWithCommas` if that's the name everyone reaches for).

- [ ] **Step 2: Update every caller**

```
grep -rn "formatNumberWithCommas\|formatNumber\|formatHitCountFull" internal/tui/views/
```

Replace each call with the canonical name.

- [ ] **Step 3: Delete the duplicates**

Remove the redundant function bodies from `helpers.go` and `dashboard_helpers.go`.

- [ ] **Step 4: Verify**

```
go vet ./...
go test ./internal/tui/views/ -v
```

- [ ] **Step 5: Commit**

```
git add internal/tui/views/
git commit -m "refactor(views): single number-with-commas formatter

Three identical comma-grouping helpers existed in helpers.go,
dashboard_helpers.go, and format_helpers.go. Keep the format_helpers.go
one, drop the others."
```

---

### Task 4.6: Consolidate time-ago formatters

**Why:** `format_helpers.go:142 formatLastHit` and `dashboard_helpers.go:116 formatTimeAgo` are essentially the same "X ago" formatter; the dashboard version is slightly more precise (pluralization). `connection_hub.go:310 formatConnectionTimeAgo` adds a "weeks ago" tier.

**Files:**
- Modify: `internal/tui/views/format_helpers.go`
- Modify: `internal/tui/views/dashboard_helpers.go`
- Modify: `internal/tui/views/connection_hub.go`

- [ ] **Step 1: Pick the canonical implementation**

Promote `formatTimeAgo` (the more precise one) to `format_helpers.go` and add the "weeks ago" tier from `formatConnectionTimeAgo` so the single function covers all three call sites.

- [ ] **Step 2: Update callers and remove duplicates**

```
grep -rn "formatLastHit\|formatTimeAgo\|formatConnectionTimeAgo" internal/tui/views/
```

Replace each call with the canonical name. Delete the obsolete functions.

- [ ] **Step 3: Verify**

```
go vet ./...
go test ./internal/tui/views/ -v
```

- [ ] **Step 4: Commit**

```
git add internal/tui/views/
git commit -m "refactor(views): single time-ago formatter

formatLastHit, formatTimeAgo, and formatConnectionTimeAgo did the same
job with slightly different output. Promote the most precise version
to format_helpers.go and route every caller through it."
```

---

### Task 4.7: Adopt TableBase cursor/offset clamps in SetSize

**Why:** Sessions, Interfaces, IPSecTunnels, LogsModel, RuleListModel each open-code the cursor/offset clamp in `SetSize`. `TableBase.EnsureCursorValid` and `EnsureVisible` exist for this exact purpose.

**Files:**
- Modify: `internal/tui/views/sessions.go`
- Modify: `internal/tui/views/interfaces.go`
- Modify: `internal/tui/views/ipsec_tunnels.go`
- Modify: `internal/tui/views/logs.go`
- Modify: `internal/tui/views/rule_list.go`

- [ ] **Step 1: Verify TableBase signatures**

```
grep -n "EnsureCursorValid\|EnsureVisible" internal/tui/views/table_base.go
```

- [ ] **Step 2: Replace each open-coded clamp**

Replace the duplicated arithmetic with calls. Example for `sessions.go`:

```go
func (m SessionsModel) SetSize(width, height int) SessionsModel {
	m.Width = width
	m.Height = height
	m.TableBase = m.TableBase.EnsureCursorValid(len(m.filtered))
	m.TableBase = m.TableBase.EnsureVisible(m.visibleRows())
	return m
}
```

(Adapt to the actual `TableBase` chaining style.)

- [ ] **Step 3: Verify**

```
go test ./internal/tui/views/ -v
```

- [ ] **Step 4: Commit**

```
git add internal/tui/views/
git commit -m "refactor(views): use TableBase cursor/offset helpers in SetSize

Five views duplicated cursor and offset clamping arithmetic in SetSize.
Delegate to TableBase.EnsureCursorValid / EnsureVisible."
```

---

### Task 4.8: Make LogsModel use TableBase.HandleFilterMode

**Why:** `internal/tui/views/logs.go:255` reimplements filter-mode key dispatch instead of delegating to `TableBase.HandleFilterMode` (which is what every other table view uses).

**Files:**
- Modify: `internal/tui/views/logs.go`

- [ ] **Step 1: Read both implementations**

Compare `logs.go:255` to `table_base.go:164`. Confirm they have the same observable behavior.

- [ ] **Step 2: Replace LogsModel.updateFilterMode**

If the contract matches, replace the body with a delegation to `TableBase.HandleFilterMode`. If `LogsModel` adds log-specific behavior on `enter`, keep that wrapper but call `TableBase.HandleFilterMode` for the common case and only override the handler for log-specific keys.

- [ ] **Step 3: Verify**

```
go test ./internal/tui/views/ -run TestLogsModel -v
```

- [ ] **Step 4: Commit**

```
git add internal/tui/views/logs.go
git commit -m "refactor(views): delegate logs filter mode to TableBase

LogsModel.updateFilterMode reimplemented TableBase.HandleFilterMode.
Delegate so future filter-mode changes apply uniformly."
```

---

### Task 4.9: Deduplicate severity style helpers

**Why:** `internal/tui/views/logs.go:453` defines a lowercase `severityStyle` and `internal/tui/views/styles.go:359` defines an exported `SeverityStyle`. The case logic is identical except `severityStyle` maps `"informational"` to `StatusMutedStyle` while `SeverityStyle` maps it to `SeverityInfoStyle`. Pick the canonical one.

**Files:**
- Modify: `internal/tui/views/logs.go`
- Modify: `internal/tui/views/styles.go` (only if the canonical version needs the `informational` arm preserved)

- [ ] **Step 1: Decide which mapping is correct**

Read both definitions. The exported `SeverityStyle` is the public API — keep it. Confirm `SeverityInfoStyle` is the right rendering for informational logs (look at how the dashboard uses it). If `StatusMutedStyle` is the visually-correct choice, update `SeverityStyle` to match before the merge.

- [ ] **Step 2: Replace callers and remove the unexported version**

```
grep -n "severityStyle(" internal/tui/views/logs.go
```

Replace each call with `SeverityStyle(...)`. Delete the unexported `severityStyle` function and any helpers it depends on that aren't reused.

- [ ] **Step 3: Verify**

```
go vet ./...
go test ./internal/tui/views/ -v
```

- [ ] **Step 4: Commit**

```
git add internal/tui/views/logs.go internal/tui/views/styles.go
git commit -m "refactor(views): single severity style helper

logs.go had a private severityStyle that diverged from the exported
SeverityStyle in styles.go on 'informational' mapping. Route all callers
through SeverityStyle."
```

---

## Phase 5 — Troubleshoot Engine Refactor

Make the engine testable end-to-end and tighten runbook validation.

---

### Task 5.1: Introduce APIClient interface in troubleshoot package

**Why:** `internal/troubleshoot/engine.go` imports `*api.Client` concretely, so every API-step test bottoms out at the nil-client guard. Define a narrow interface satisfied implicitly by `*api.Client`.

**Files:**
- Modify: `internal/troubleshoot/engine.go`

- [ ] **Step 1: Catalog the methods used**

```
grep -n "c\.client\." internal/troubleshoot/engine.go
```

Note every method called on the client (e.g., `GetSystemInfo`, `GetSystemResources`, `GetHAState`, `GetSessionInfo`).

- [ ] **Step 2: Define the interface**

In `internal/troubleshoot/engine.go`, near the top:

```go
// APIClient is the narrow PAN-OS API surface the troubleshoot engine needs.
// *api.Client satisfies this interface implicitly.
type APIClient interface {
	GetSystemInfo(ctx context.Context, target string) (*models.SystemInfo, error)
	GetSystemResources(ctx context.Context, target string) (*models.SystemResources, error)
	GetHAState(ctx context.Context, target string) (*models.HAState, error)
	GetSessionInfo(ctx context.Context, target string) (*models.SessionInfo, error)
}
```

(Match actual return types from `internal/api/`.)

Change the `Engine` struct's field from `*api.Client` to `APIClient`. `NewEngine` accepts `APIClient`.

- [ ] **Step 3: Verify build**

```
go build ./...
go test ./...
```

Expected: clean. `*api.Client` satisfies the interface implicitly so no caller change is needed.

- [ ] **Step 4: Commit**

```
git add internal/troubleshoot/engine.go
git commit -m "refactor(troubleshoot): accept APIClient interface in Engine

Define a narrow APIClient interface in the troubleshoot package so tests
can inject fakes. *api.Client continues to satisfy it implicitly."
```

---

### Task 5.2: Add tests for executeAPIStep branches

**Why:** All four `step.APICall` branches (`system_info`, `system_resources`, `ha_status`, `session_info`) currently have zero coverage because every test hits the nil-client guard. With the interface in place, fakes can exercise each branch.

**Files:**
- Modify: `internal/troubleshoot/engine_test.go`

- [ ] **Step 1: Write a fake APIClient**

In `internal/troubleshoot/engine_test.go`:

```go
type fakeAPIClient struct {
	systemInfo      *models.SystemInfo
	systemResources *models.SystemResources
	haState         *models.HAState
	sessionInfo     *models.SessionInfo
	err             error
}

func (f *fakeAPIClient) GetSystemInfo(_ context.Context, _ string) (*models.SystemInfo, error) {
	return f.systemInfo, f.err
}
func (f *fakeAPIClient) GetSystemResources(_ context.Context, _ string) (*models.SystemResources, error) {
	return f.systemResources, f.err
}
func (f *fakeAPIClient) GetHAState(_ context.Context, _ string) (*models.HAState, error) {
	return f.haState, f.err
}
func (f *fakeAPIClient) GetSessionInfo(_ context.Context, _ string) (*models.SessionInfo, error) {
	return f.sessionInfo, f.err
}
```

- [ ] **Step 2: Add table-driven tests for the four branches**

```go
func TestExecuteAPIStep_SystemInfo(t *testing.T) {
	fake := &fakeAPIClient{systemInfo: &models.SystemInfo{Hostname: "fw01", SWVersion: "11.1.0"}}
	eng := NewEngine(fake)
	step := Step{Type: StepTypeAPI, APICall: "system_info"}
	out, err := eng.executeStep(context.Background(), step, "")
	if err != nil { t.Fatalf("err: %v", err) }
	if !strings.Contains(out, "fw01") || !strings.Contains(out, "11.1.0") {
		t.Errorf("expected hostname and version in output: %q", out)
	}
}
// ... TestExecuteAPIStep_SystemResources, TestExecuteAPIStep_HAStatus,
// ... TestExecuteAPIStep_SessionInfo, TestExecuteAPIStep_HADisabled, etc.
```

(Adapt to actual `executeStep` / `executeAPIStep` signature. If they're unexported and not testable from `_test.go` in the same package, run tests in the same package — not an `_test` external package.)

- [ ] **Step 3: Verify**

```
go test ./internal/troubleshoot/ -v
```

- [ ] **Step 4: Commit**

```
git add internal/troubleshoot/engine_test.go
git commit -m "test(troubleshoot): cover all four executeAPIStep branches

Use a fake APIClient to exercise system_info, system_resources, ha_status,
and session_info. Previously the nil-client guard prevented any branch
from running under test."
```

---

### Task 5.3: Validate `Step.APICall` against known values

**Why:** A runbook with `type: api` and an empty or unknown `api_call` passes `Step.Validate()` and only fails at runtime with a generic error.

**Files:**
- Modify: `internal/troubleshoot/runbook.go`
- Modify: `internal/troubleshoot/runbook_validate_test.go`

- [ ] **Step 1: Define the valid set**

In `internal/troubleshoot/runbook.go`, add a package-level set kept in lockstep with `executeAPIStep`'s switch in `engine.go`:

```go
var validAPICalls = map[string]struct{}{
	"system_info":      {},
	"system_resources": {},
	"ha_status":        {},
	"session_info":     {},
}
```

- [ ] **Step 2: Tighten Validate**

In `Step.Validate`:

```go
if s.Type == StepTypeAPI {
	if s.APICall == "" {
		return fmt.Errorf("step %q: type=api requires non-empty api_call", s.Name)
	}
	if _, ok := validAPICalls[s.APICall]; !ok {
		return fmt.Errorf("step %q: unknown api_call %q", s.Name, s.APICall)
	}
}
```

- [ ] **Step 3: Add tests**

In `internal/troubleshoot/runbook_validate_test.go`:

```go
func TestStep_Validate_APIRequiresKnownCall(t *testing.T) {
	tests := []struct {
		name    string
		step    Step
		wantErr string
	}{
		{"empty api_call", Step{Name: "s1", Type: StepTypeAPI}, "non-empty api_call"},
		{"unknown api_call", Step{Name: "s2", Type: StepTypeAPI, APICall: "bogus"}, "unknown api_call"},
		{"valid api_call", Step{Name: "s3", Type: StepTypeAPI, APICall: "system_info"}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.step.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("unexpected err: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Errorf("err = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 4: Verify (and update embedded runbooks if any fail validation)**

```
go test ./internal/troubleshoot/ -v
```

If `TestEmbeddedRunbooks_AllValidate` now fails for an existing runbook, fix the runbook YAML (it likely has a typo that was previously silent).

- [ ] **Step 5: Commit**

```
git add internal/troubleshoot/runbook.go internal/troubleshoot/runbook_validate_test.go
git commit -m "feat(troubleshoot): validate api_call against known values

Step.Validate now rejects empty or unknown api_call values at load time
instead of failing at runtime. validAPICalls is kept in lockstep with
the switch in engine.executeAPIStep."
```

---

### Task 5.4: Sort Registry.List output

**Why:** `Registry.List()` iterates a map and returns in non-deterministic order. The TUI sorts before display, but any other caller can't rely on stable ordering.

**Files:**
- Modify: `internal/troubleshoot/runbook.go` (`Registry.List`)
- Modify: `internal/troubleshoot/runbook_test.go`

- [ ] **Step 1: Sort by Name in List**

In `internal/troubleshoot/runbook.go`:

```go
import "sort"

func (r *Registry) List() []*Runbook {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Runbook, 0, len(r.runbooks))
	for _, rb := range r.runbooks {
		out = append(out, rb)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
```

- [ ] **Step 2: Add a stability test**

```go
func TestRegistry_List_StableOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(&Runbook{Name: "zeta"})
	r.Register(&Runbook{Name: "alpha"})
	r.Register(&Runbook{Name: "mike"})

	got := r.List()
	want := []string{"alpha", "mike", "zeta"}
	for i, rb := range got {
		if rb.Name != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, rb.Name, want[i])
		}
	}
}
```

- [ ] **Step 3: Verify and commit**

```
go test ./internal/troubleshoot/ -v
```

```
git add internal/troubleshoot/runbook.go internal/troubleshoot/runbook_test.go
git commit -m "refactor(troubleshoot): sort Registry.List by name

Map iteration order was nondeterministic. Sort by Name so callers can
rely on stable ordering without re-sorting."
```

---

### Task 5.5: Correct the misleading comment about MatchAll context

**Why:** `internal/troubleshoot/engine.go` (around lines 90–95) has a comment claiming `stepCtx` scopes "the subsequent MatchAll pass". `MatchAll` does not currently take a context, so the comment is wrong. Either update the comment to match reality or thread context through `MatchAll`. Threading is API-breaking and probably overkill for current pattern volumes; fix the comment.

**Files:**
- Modify: `internal/troubleshoot/engine.go`

- [ ] **Step 1: Update the comment**

Read lines 88–100 in `internal/troubleshoot/engine.go`. Replace any phrase that claims `stepCtx` scopes the `MatchAll` pass with an accurate statement. Example:

```go
// stepCtx applies the per-step timeout to executeAPIStep only.
// MatchAll runs after the API call completes and is not context-bounded;
// pattern matching is fast enough that this is intentional. If patterns
// ever scan large outputs, thread ctx through PatternMatcher.MatchAll.
```

- [ ] **Step 2: Verify**

```
go vet ./...
go test ./internal/troubleshoot/ -v
```

- [ ] **Step 3: Commit**

```
git add internal/troubleshoot/engine.go
git commit -m "docs(troubleshoot): correct stepCtx comment

The comment claimed stepCtx bounded the subsequent MatchAll pass, but
MatchAll takes no context and runs unbounded. Document the actual
scope and the rationale for not threading context through pattern
matching today."
```

---

## Phase 6 — Logs API Generic Helper

Optional. Collapses ~90 lines of structural duplication across three log-fetch functions.

---

### Task 6.1: Extract a generic log fetch helper

**Why:** `GetSystemLogs`, `GetTrafficLogs`, `GetThreatLogs` in `internal/api/logs.go` share the submit→parse-job-ID→poll→decode-entries pattern. Only the log-type string and decode target differ.

**Files:**
- Modify: `internal/api/logs.go`

- [ ] **Step 1: Identify the shared shape**

Read the three functions. The differences are:
1. The `logType` string ("system" / "traffic" / "threat")
2. The decode target type (`models.SystemLogEntry`, `models.TrafficLogEntry`, `models.ThreatLogEntry`)

- [ ] **Step 2: Add a generic helper**

```go
func fetchLogsByType[T any](
	ctx context.Context,
	c *Client,
	logType string,
	target string,
	query LogQuery,
) ([]T, error) {
	jobID, err := c.submitLogJob(ctx, logType, target, query)
	if err != nil {
		return nil, err
	}
	resp, err := c.pollLogJob(ctx, jobID, target)
	if err != nil {
		return nil, err
	}
	var result struct {
		Entries []T `xml:"job>result>log>logs>entry"`
	}
	if err := decodeXML(bytes.NewReader(WrapInner(resp.Result.Inner)), &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}
```

(Adapt the XPath to match the existing pattern; verify against the current `Decode` calls in `GetSystemLogs` etc.)

- [ ] **Step 3: Rewrite the three callers**

```go
func (c *Client) GetSystemLogs(ctx context.Context, target string, q LogQuery) ([]models.SystemLogEntry, error) {
	return fetchLogsByType[models.SystemLogEntry](ctx, c, "system", target, q)
}
// ... similarly for GetTrafficLogs, GetThreatLogs
```

- [ ] **Step 4: Verify**

```
go test ./internal/api/ -v
```

Existing log tests should continue to pass without modification.

- [ ] **Step 5: Commit**

```
git add internal/api/logs.go
git commit -m "refactor(api): generic helper for log type fetching

GetSystemLogs, GetTrafficLogs, and GetThreatLogs duplicated the same
submit-poll-decode pipeline. Extract fetchLogsByType[T] and route the
three through it."
```

---

## Closing checklist

After all phases complete, confirm:

- [ ] `go build ./cmd/pyre` succeeds
- [ ] `go vet ./...` clean
- [ ] `go test -race -v ./...` clean
- [ ] `go fix -diff ./...` produces no further suggestions
- [ ] `golangci-lint run` clean (if installed locally)
- [ ] `./pyre --version` runs the smoke target

If any phase was skipped because the verification (e.g., race test in Phase 3) showed it wasn't needed, document that in the PR description.
