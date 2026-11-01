package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/jp2195/pyre/internal/models"
)

// Selection accessors: each returns the row under the cursor, in the order
// the table shows it (filtered and sorted), and false when there is none.
// The copy key reads them to decide what to put on the clipboard.

// Selected returns the item under the cursor.
func (m RuleListModel[T]) Selected() (T, bool) {
	if m.Cursor < 0 || m.Cursor >= len(m.filtered) {
		var zero T
		return zero, false
	}
	return m.filtered[m.Cursor], true
}

func (m PoliciesModel) Selected() (models.SecurityRule, bool)     { return m.list.Selected() }
func (m NATPoliciesModel) Selected() (models.NATRule, bool)       { return m.list.Selected() }
func (m SessionsModel) Selected() (models.Session, bool)          { return m.list.Selected() }
func (m InterfacesModel) Selected() (models.Interface, bool)      { return m.list.Selected() }
func (m IPSecTunnelsModel) Selected() (models.IPSecTunnel, bool)  { return m.list.Selected() }
func (m GPUsersModel) Selected() (models.GlobalProtectUser, bool) { return m.list.Selected() }

// SelectedPeer returns what the cursor is on: a route's destination on the
// Routes tab, or the peer column of the Neighbors tab (a BGP peer address,
// an OSPF neighbor ID), with a label saying which.
func (m RoutesModel) SelectedPeer() (label, value string, ok bool) {
	if m.activeTab == RoutesTabNeighbors {
		i := m.neighborCursor
		switch {
		case i >= 0 && i < len(m.bgpNeighbors):
			return "BGP peer", m.bgpNeighbors[i].PeerAddress, true
		case i >= len(m.bgpNeighbors) && i < len(m.bgpNeighbors)+len(m.ospfNeighbors):
			return "OSPF neighbor", m.ospfNeighbors[i-len(m.bgpNeighbors)].NeighborID, true
		}
		return "", "", false
	}
	if m.Cursor < 0 || m.Cursor >= len(m.filtered) {
		return "", "", false
	}
	return "route", m.filtered[m.Cursor].Destination, true
}

// SelectedName returns the name of the object under the cursor on whichever
// sub-tab is showing, with a label saying which kind it is.
func (m ObjectsModel) SelectedName() (label, value string, ok bool) {
	switch m.tab {
	case ObjectsTabService:
		t := m.serviceTab
		if t.Cursor >= 0 && t.Cursor < len(t.filtered) {
			return "service", t.filtered[t.Cursor].Name, true
		}
	default:
		t := m.addressTab
		if t.Cursor >= 0 && t.Cursor < len(t.filtered) {
			return "address", t.filtered[t.Cursor].Name, true
		}
	}
	return "", "", false
}

// SelectedLine returns the log entry under the cursor as one line of plain
// text, for pasting into a ticket or a chat. A log entry has no single
// identifier worth copying on its own, so the line carries the fields an
// operator quotes: when, who talked to whom, and what the firewall did.
func (m LogsModel) SelectedLine() (string, bool) {
	i := m.Cursor
	switch m.activeLogType {
	case models.LogTypeTraffic:
		if i < 0 || i >= len(m.filteredTraffic) {
			return "", false
		}
		e := m.filteredTraffic[i]
		return joinFields(
			logTime(e.Time),
			fmt.Sprintf("%s:%d -> %s:%d", e.SourceIP, e.SourcePort, e.DestIP, e.DestPort),
			e.Application, e.Action, kv("rule", e.Rule), kv("user", e.User),
		), true
	case models.LogTypeThreat:
		if i < 0 || i >= len(m.filteredThreat) {
			return "", false
		}
		e := m.filteredThreat[i]
		return joinFields(
			logTime(e.Time), e.Severity, e.ThreatName,
			fmt.Sprintf("%s:%d -> %s:%d", e.SourceIP, e.SourcePort, e.DestIP, e.DestPort),
			e.Action, kv("rule", e.Rule), kv("id", e.ThreatID),
		), true
	default:
		if i < 0 || i >= len(m.filteredSystem) {
			return "", false
		}
		e := m.filteredSystem[i]
		return joinFields(logTime(e.Time), e.Severity, e.Type, e.Description), true
	}
}

// logTime renders a log timestamp with its offset, so a pasted line still
// says when it happened once it leaves the device's time zone.
func logTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func kv(k, v string) string {
	if v == "" {
		return ""
	}
	return k + "=" + v
}

func joinFields(fields ...string) string {
	kept := fields[:0]
	for _, f := range fields {
		if f != "" {
			kept = append(kept, f)
		}
	}
	return strings.Join(kept, " ")
}
