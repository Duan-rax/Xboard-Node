package main

import "testing"

func TestSingBoxJournalIdentityCorrelation(t *testing.T) {
	collector := &Collector{
		singBoxContexts:   make(map[string]singBoxLogContext),
		singBoxIdentities: make(map[string]singBoxIdentity),
	}
	target := Target{NodeID: "sg-anytls-01"}
	collector.handleSingBoxLogLine(target, "+0800 2026-09-09 12:00:00 INFO [123456 0ms] inbound/anytls[anytls-in]: inbound connection from 119.97.37.32:6185")
	collector.handleSingBoxLogLine(target, "+0800 2026-09-09 12:00:00 INFO [123456 1ms] inbound/anytls[anytls-in]: [d02236c7-a58c-4f44-9254-d54907546a22] inbound connection to api.termius.com:443")

	if got := collector.singBoxUser(target, "119.97.37.32:6185", "api.termius.com:443"); got != "d02236c7-a58c-4f44-9254-d54907546a22" {
		t.Fatalf("resolved sing-box user = %q", got)
	}
}
