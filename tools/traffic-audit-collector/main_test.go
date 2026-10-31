package main

import (
	"testing"
	"time"
)

func TestSingBoxJournalIdentityCorrelation(t *testing.T) {
	collector := &Collector{
		singBoxContexts:    make(map[string]singBoxLogContext),
		singBoxIdentities:  make(map[string]singBoxIdentity),
		singBoxConnections: make(map[string]singBoxIdentity),
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
		singBoxContexts:    make(map[string]singBoxLogContext),
		singBoxIdentities:  make(map[string]singBoxIdentity),
		singBoxConnections: make(map[string]singBoxIdentity),
	}
	unit := "xboard-node"
	collector.handleSingBoxLogLine(unit, "INFO [core] audit connection user_id=7 connection_id=clash-id-7 source=203.0.113.8:52144 destination=1.1.1.1:443 domain=speed.cloudflare.com")

	target := Target{NodeID: "jp-01", SingBoxJournalUnit: unit}
	for _, destination := range []string{"1.1.1.1:443", "speed.cloudflare.com:443"} {
		if got := collector.singBoxUser(target, "203.0.113.8:52144", destination); got != "@7" {
			t.Fatalf("resolved audit user for %s = %q", destination, got)
		}
	}
	if got := collector.singBoxUser(target, "198.51.100.2:55221", "speed.cloudflare.com:443"); got != "@7" {
		t.Fatalf("destination fallback user = %q", got)
	}
	if got := collector.singBoxUserByConnection(target, "clash-id-7"); got != "@7" {
		t.Fatalf("connection ID user = %q", got)
	}
	if got := parseUserID("@7"); got == nil || *got != 7 {
		t.Fatalf("parsed audit user = %v", got)
	}
}

func TestPendingEventResolvesLateIdentity(t *testing.T) {
	collector := &Collector{
		singBoxContexts:    make(map[string]singBoxLogContext),
		singBoxIdentities:  make(map[string]singBoxIdentity),
		singBoxConnections: make(map[string]singBoxIdentity),
		singBoxRecentUsers: make(map[string]singBoxIdentity),
		pendingEvents:      make(map[string]pendingAuditEvent),
	}
	target := Target{NodeID: "jp-01", SingBoxJournalUnit: "xboard-node"}
	item := pendingAuditEvent{
		Target: target,
		Event: Event{
			EventID:      "event-late-user",
			ClientSource: "203.0.113.8:52144",
			Destination:  "speed.cloudflare.com:443",
			Metadata:     map[string]any{"clash_connection_id": "clash-id-9"},
		},
		ExpiresAt: time.Now().Add(20 * time.Second),
	}

	collector.handleSingBoxLogLine(
		"xboard-node",
		"INFO [core] audit connection user_id=9 connection_id=clash-id-9 source=203.0.113.8:52144 destination=speed.cloudflare.com:443 domain=speed.cloudflare.com",
	)
	resolved := collector.resolvePendingEvent(item)
	if resolved.Event.UserID == nil || *resolved.Event.UserID != 9 {
		t.Fatalf("late event user = %v", resolved.Event.UserID)
	}
	if resolved.Event.Metadata["user"] != "@9" {
		t.Fatalf("late event metadata user = %v", resolved.Event.Metadata["user"])
	}
	if resolved.Event.Metadata["user_match"] != "connection_id" {
		t.Fatalf("late event match method = %v", resolved.Event.Metadata["user_match"])
	}
}

func TestConnectionIDWinsWhenSourceAndDestinationAreAmbiguous(t *testing.T) {
	collector := &Collector{
		singBoxContexts:    make(map[string]singBoxLogContext),
		singBoxIdentities:  make(map[string]singBoxIdentity),
		singBoxConnections: make(map[string]singBoxIdentity),
		singBoxRecentUsers: make(map[string]singBoxIdentity),
		pendingEvents:      make(map[string]pendingAuditEvent),
	}
	unit := "xboard-node"
	collector.handleSingBoxLogLine(unit, "INFO [core] audit connection user_id=7 connection_id=clash-7 source=203.0.113.8:52144 destination=example.com:443 domain=example.com")
	collector.handleSingBoxLogLine(unit, "INFO [core] audit connection user_id=8 connection_id=clash-8 source=203.0.113.8:52144 destination=example.com:443 domain=example.com")

	item := pendingAuditEvent{
		Target: Target{NodeID: "jp-01", SingBoxJournalUnit: unit},
		Event: Event{
			ClientSource: "203.0.113.8:52144",
			Destination:  "example.com:443",
			Metadata:     map[string]any{"clash_connection_id": "clash-8"},
		},
		ExpiresAt: time.Now().Add(20 * time.Second),
	}
	resolved := collector.resolvePendingEvent(item)
	if resolved.Event.UserID == nil || *resolved.Event.UserID != 8 {
		t.Fatalf("resolved user = %v, want 8", resolved.Event.UserID)
	}
	if resolved.Event.Metadata["user_match"] != "connection_id" {
		t.Fatalf("match method = %v", resolved.Event.Metadata["user_match"])
	}
}

func TestIdentityLogCreatesLinkedConnectionWithoutClashSample(t *testing.T) {
	event := singBoxIdentityEvent(
		Target{NodeID: "21"},
		"@7",
		"203.0.113.8:52144",
		"1.1.1.1:443",
		"example.com",
		"clash-connection-7",
	)
	if event.UserID == nil || *event.UserID != 7 {
		t.Fatalf("user = %v", event.UserID)
	}
	if event.ConnectionID != "singbox-"+eventHash("21", "clash-connection-7") {
		t.Fatalf("connection ID = %q", event.ConnectionID)
	}
	if event.Destination != "example.com:443" {
		t.Fatalf("destination = %q", event.Destination)
	}
	if event.Metadata["user_match"] != "connection_id" || event.Metadata["clash_connection_id"] != "clash-connection-7" {
		t.Fatalf("metadata = %#v", event.Metadata)
	}
}

func TestSingBoxUDPAuditSummaryCreatesRealDestinationEvent(t *testing.T) {
	event := singBoxUDPAuditEvent(Target{NodeID: "21"}, []string{
		"audit udp destination user_id=7 connection_id=clash-udp-7 source=203.0.113.8:52144 destination=speed.cloudflare.com:443 upload=123 download=456",
		"7", "clash-udp-7", "203.0.113.8:52144", "speed.cloudflare.com:443", "123", "456",
	})
	if event.UserID == nil || *event.UserID != 7 {
		t.Fatalf("user = %v", event.UserID)
	}
	if event.Destination != "speed.cloudflare.com:443" || event.Network != "udp" {
		t.Fatalf("destination = %q network = %q", event.Destination, event.Network)
	}
	if event.UploadBytes != 123 || event.DownloadBytes != 456 || event.EventType != "close" {
		t.Fatalf("event = %#v", event)
	}
	if event.ConnectionID != "singbox-"+eventHash("21", "clash-udp-7") {
		t.Fatalf("connection ID = %q", event.ConnectionID)
	}
}

func TestUoTControlDestinationIsNotPublishedAsTraffic(t *testing.T) {
	for _, destination := range []string{
		"sp.v2.udp-over-tcp.arpa:443",
		"tcp:sp.udp-over-tcp.arpa:443",
		"udp:sp.v2.udp-over-tcp.arpa:443",
	} {
		if !isUoTControlDestination(destination) {
			t.Fatalf("expected %q to be recognised as UoT control target", destination)
		}
	}
	if isUoTControlDestination("speed.cloudflare.com:443") {
		t.Fatal("real destination must not be treated as UoT control target")
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

func TestParseXrayLineWithConnectionBytes(t *testing.T) {
	target := Target{NodeID: "28"}
	event := parseXrayLine(
		target,
		"2026/10/31 12:34:56 from 203.0.113.8:52144 accepted tcp:example.com:443 [vless-in >> direct] email: user@7 up=1024 down=8192",
	)
	if event == nil {
		t.Fatal("expected Xray event")
	}
	if event.EventType != "close" {
		t.Fatalf("event type = %q", event.EventType)
	}
	if event.UserID == nil || *event.UserID != 7 {
		t.Fatalf("user = %v", event.UserID)
	}
	if event.UploadBytes != 1024 || event.DownloadBytes != 8192 {
		t.Fatalf("bytes = up:%d down:%d", event.UploadBytes, event.DownloadBytes)
	}
	if event.Metadata["byte_counters"] != true {
		t.Fatalf("byte counter metadata = %v", event.Metadata["byte_counters"])
	}
}

func TestParseLegacyXrayLineRemainsSupported(t *testing.T) {
	event := parseXrayLine(
		Target{NodeID: "28"},
		"2026/10/31 12:34:56 from 203.0.113.8:52144 accepted tcp:example.com:443 [vless-in >> direct] email: user@7",
	)
	if event == nil {
		t.Fatal("expected legacy Xray event")
	}
	if event.EventType != "start" {
		t.Fatalf("event type = %q", event.EventType)
	}
	if event.UploadBytes != 0 || event.DownloadBytes != 0 {
		t.Fatalf("legacy bytes must remain unknown, got up:%d down:%d", event.UploadBytes, event.DownloadBytes)
	}
	if event.Metadata["byte_counters"] != false {
		t.Fatalf("byte counter metadata = %v", event.Metadata["byte_counters"])
	}
}
