package main

import "testing"

func TestSingBoxJournalIdentityCorrelation(t *testing.T) {
	collector := &Collector{
		singBoxContexts:   make(map[string]singBoxLogContext),
		singBoxIdentities: make(map[string]singBoxIdentity),
	}
	unit := "xboard-node"
	collector.handleSingBoxLogLine(unit, "+0800 2026-09-09 12:00:00 INFO [123456 0ms] inbound/anytls[anytls-in]: inbound connection from 119.97.37.32:6185")
	collector.handleSingBoxLogLine(unit, "+0800 2026-09-09 12:00:00 INFO [123456 1ms] inbound/anytls[anytls-in]: [d02236c7-a58c-4f44-9254-d54907546a22] inbound connection to api.termius.com:443")

	otherTarget := Target{NodeID: "sg-anytls-02", SingBoxJournalUnit: unit}
	if got := collector.singBoxUser(otherTarget, "119.97.37.32:6185", "api.termius.com:443"); got != "d02236c7-a58c-4f44-9254-d54907546a22" {
		t.Fatalf("resolved sing-box user = %q", got)
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
