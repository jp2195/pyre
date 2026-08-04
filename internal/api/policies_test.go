package api

import (
	"testing"
	"time"
)

func TestParseRuleHitCounts_HandlesWrappedInner(t *testing.T) {
	// Inner bytes as PAN-OS returns them: <rule-hit-count> as the first
	// element with no document-level wrapper.
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
