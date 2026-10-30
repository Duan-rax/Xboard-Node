package main

import "testing"

func TestSingBoxJournalIdentityCorrelation(t *testing.T) {
	collector := &Collector{
		singBoxContexts:    make(map[string]singBoxLogContext),
		singBoxIdentities:  make(map[string]singBoxIdentity),
		singBoxRecentUsers: make(map[string]singBoxIdentity),
	}
	unit := "xboard-node"
	collector.handleSingBoxLogLine(unit, "+0800 2026-09-09 12:00:00 INFO [123456 0ms] inbound/anytls[anytls-in]: inbound connection from 119.97.37.32:6185")
	collector.handleSingBoxLogLine(unit, "+0800 2026-09-09 12:00:00 INFO [123456 1ms] inbound/anytls[anytls-in]: [d02236c7-a58c-4f44-9254-d54907546a22] inbound connection to api.termius.com:443")

	otherTarget := Target{NodeID: "sg-anytls-02", SingBoxJournalUnit: unit}
	if got := collector.singBoxUser(otherTarget, "119.97.37.32:6185", "api.termius.com:443"); got != "d02236c7-a58c-4f44-9254-d54907546a22" {
		t.Fatalf("resolved sing-box user = %q", got)
	}
}

func TestSingBoxAuditConnectionIdentity(t *testing.T) {
	collector := &Collector{
		singBoxContexts:   make(map[string]singBoxLogContext),
		singBoxIdentities: make(map[string]singBoxIdentity),
	}
	unit := "xboard-node"
	collector.handleSingBoxLogLine(unit, "INFO [core] audit connection user_id=7 source=203.0.113.8:52144 destination=1.1.1.1:443 domain=speed.cloudflare.com")

	target := Target{NodeID: "jp-01", SingBoxJournalUnit: unit}
	for _, destination := range []string{"1.1.1.1:443", "speed.cloudflare.com:443"} {
		if got := collector.singBoxUser(target, "203.0.113.8:52144", destination); got != "@7" {
			t.Fatalf("resolved audit user for %s = %q", destination, got)
		}
	}
	if got := collector.singBoxUser(target, "198.51.100.2:55221", "speed.cloudflare.com:443"); got != "@7" {
		t.Fatalf("destination fallback user = %q", got)
	}
	if got := parseUserID("@7"); got == nil || *got != 7 {
		t.Fatalf("parsed audit user = %v", got)
	}
}

func TestPanelRouteIdentity(t *testing.T) {
	id, tag := panelRouteIdentity("panel-route-38")
	if id == nil || *id != 38 || tag != "panel-route-38" {
		t.Fatalf("unexpected route identity: id=%v tag=%q", id, tag)
	}
	if id, tag := panelRouteIdentity("rule_set=cn"); id != nil || tag != "" {
		t.Fatalf("non-panel rule must not be mapped: id=%v tag=%q", id, tag)
	}
}
