package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	PanelEventURL   string   `json:"panel_event_url"`
	WebhookSecret   string   `json:"webhook_secret"`
	IntervalSeconds int      `json:"interval_seconds"`
	Targets         []Target `json:"targets"`
	ManifestPaths   []string `json:"manifest_paths"`
}

type Target struct {
	NodeID             string `json:"node_id"`
	Kernel             string `json:"kernel"`
	ClashAPI           string `json:"clash_api"`
	ClashSecret        string `json:"clash_secret"`
	SingBoxJournalUnit string `json:"singbox_journal_unit"`
	XrayAccessLog      string `json:"xray_access_log"`
}

type Event struct {
	EventID       string         `json:"event_id"`
	ConnectionID  string         `json:"connection_id"`
	EventType     string         `json:"event_type"`
	UserID        *int           `json:"user_id,omitempty"`
	NodeID        string         `json:"node_id"`
	ClientSource  string         `json:"client_source,omitempty"`
	OccurredAt    time.Time      `json:"occurred_at"`
	Action        string         `json:"action"`
	RuleTag       string         `json:"rule_tag,omitempty"`
	RouteID       *int64         `json:"route_id,omitempty"`
	RouteTag      string         `json:"route_tag,omitempty"`
	RouteSource   string         `json:"route_source,omitempty"`
	Outbound      string         `json:"outbound,omitempty"`
	Destination   string         `json:"destination,omitempty"`
	DestinationIP string         `json:"destination_ip,omitempty"`
	Network       string         `json:"network,omitempty"`
	Protocol      string         `json:"protocol,omitempty"`
	UploadBytes   uint64         `json:"upload_bytes"`
	DownloadBytes uint64         `json:"download_bytes"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type ClashResponse struct {
	Connections []ClashConnection `json:"connections"`
}

type ClashConnection struct {
	ID       string        `json:"id"`
	Upload   uint64        `json:"upload"`
	Download uint64        `json:"download"`
	Start    time.Time     `json:"start"`
	Rule     string        `json:"rule"`
	Chains   []string      `json:"chains"`
	Metadata ClashMetadata `json:"metadata"`
}

type ClashMetadata struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationIP   string `json:"destinationIP"`
	DestinationPort string `json:"destinationPort"`
	Host            string `json:"host"`
	Inbound         string `json:"inbound"`
	User            string `json:"user"`
}

type counter struct{ up, down uint64 }

type singBoxLogContext struct {
	Source      string
	User        string
	Destination string
	UpdatedAt   time.Time
}

type singBoxIdentity struct {
	User         string
	ConnectionID string
	ExpiresAt    time.Time
}

type managedTarget struct {
	target Target
	cancel context.CancelFunc
	owners map[string]bool
}

type pendingAuditEvent struct {
	Target    Target
	Event     Event
	ExpiresAt time.Time
}

type Collector struct {
	cfg                 Config
	client              *http.Client
	mu                  sync.Mutex
	counters            map[string]counter
	singBoxContexts     map[string]singBoxLogContext
	singBoxIdentities   map[string]singBoxIdentity
	singBoxConnections  map[string]singBoxIdentity
	singBoxRecentUsers  map[string]singBoxIdentity
	pendingEvents       map[string]pendingAuditEvent
	startMu             sync.Mutex
	startedTargets      map[string]*managedTarget
	startedJournalUnits map[string]bool
}

var (
	xrayAccepted           = regexp.MustCompile(`from\s+([^\s]+)\s+accepted\s+([^\s]+)`)
	xrayEmail              = regexp.MustCompile(`email:\s*([^\s]+)`)
	xrayTraffic            = regexp.MustCompile(`\bup=([0-9]+)\s+down=([0-9]+)\b`)
	userIDSuffix           = regexp.MustCompile(`(?:^|@)([1-9][0-9]*)$`)
	singBoxContextID       = regexp.MustCompile(`\[(\d+)(?:\s+[^\]]+)?\]`)
	singBoxSource          = regexp.MustCompile(`inbound connection from ([^\s]+)`)
	singBoxUserDestination = regexp.MustCompile(`\]: \[([^\]]+)\] inbound(?: packet)? connection to ([^\s]+)`)
	singBoxAuditIdentity   = regexp.MustCompile(`audit connection user_id=([0-9]+)(?: connection_id=([^\s]+))? source=([^\s]+) destination=([^\s]+) domain=([^\s]*)`)
)

func main() {
	configPath := flag.String("config", "/etc/traffic-audit-collector/config.json", "collector config path")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	collector := &Collector{
		cfg:                 cfg,
		client:              &http.Client{Timeout: 8 * time.Second},
		counters:            make(map[string]counter),
		singBoxContexts:     make(map[string]singBoxLogContext),
		singBoxIdentities:   make(map[string]singBoxIdentity),
		singBoxConnections:  make(map[string]singBoxIdentity),
		singBoxRecentUsers:  make(map[string]singBoxIdentity),
		pendingEvents:       make(map[string]pendingAuditEvent),
		startedTargets:      make(map[string]*managedTarget),
		startedJournalUnits: make(map[string]bool),
	}

	collector.syncTargets("static", cfg.Targets)
	for _, path := range cfg.ManifestPaths {
		if strings.TrimSpace(path) != "" {
			go collector.watchManifest(path)
		}
	}
	select {}
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.PanelEventURL == "" || cfg.WebhookSecret == "" {
		return Config{}, errors.New("panel_event_url and webhook_secret are required")
	}
	if len(cfg.Targets) == 0 && len(cfg.ManifestPaths) == 0 {
		return Config{}, errors.New("at least one target or manifest_path is required")
	}
	if cfg.IntervalSeconds < 1 {
		cfg.IntervalSeconds = 2
	}
	for i := range cfg.Targets {
		if cfg.Targets[i].NodeID == "" {
			return Config{}, fmt.Errorf("targets[%d].node_id is required", i)
		}
	}
	return cfg, nil
}

type targetManifest struct {
	Targets []Target `json:"targets"`
}

func (c *Collector) watchManifest(path string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("audit manifest %s: %v", path, err)
		} else {
			var manifest targetManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				log.Printf("audit manifest %s: parse: %v", path, err)
			} else {
				c.syncTargets("manifest:"+path, manifest.Targets)
			}
		}
		<-ticker.C
	}
}

// syncTargets makes a manifest authoritative for its own targets while
// preserving static targets. It starts new panel nodes and stops polling
// controller ports that disappeared after a panel node was removed.
func (c *Collector) syncTargets(owner string, targets []Target) {
	wanted := make(map[string]Target, len(targets))
	for _, target := range targets {
		if target.NodeID == "" {
			continue
		}
		wanted[targetKey(target)] = target
	}
	type launch struct {
		target  Target
		ctx     context.Context
		xray    bool
		journal bool
	}
	var launches []launch
	c.startMu.Lock()
	for key, managed := range c.startedTargets {
		if !managed.owners[owner] || wanted[key].NodeID != "" {
			continue
		}
		delete(managed.owners, owner)
		if len(managed.owners) == 0 {
			managed.cancel()
			delete(c.startedTargets, key)
		}
	}
	for key, target := range wanted {
		if managed := c.startedTargets[key]; managed != nil {
			managed.owners[owner] = true
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		c.startedTargets[key] = &managedTarget{target: target, cancel: cancel, owners: map[string]bool{owner: true}}
		startJournal := target.SingBoxJournalUnit != "" && !c.startedJournalUnits[target.SingBoxJournalUnit]
		if startJournal {
			c.startedJournalUnits[target.SingBoxJournalUnit] = true
		}
		launches = append(launches, launch{target: target, ctx: ctx, xray: target.XrayAccessLog != "", journal: startJournal})
	}
	c.startMu.Unlock()
	for _, item := range launches {
		if item.target.ClashAPI != "" {
			go c.pollSingBox(item.ctx, item.target)
		}
		if item.xray {
			go c.tailXrayAccessLog(item.target)
		}
		if item.journal {
			go c.tailSingBoxJournal(item.target.SingBoxJournalUnit)
		}
	}
}

func targetKey(target Target) string {
	return target.NodeID + "\x00" + target.ClashAPI + "\x00" + target.XrayAccessLog
}

func (c *Collector) pollSingBox(ctx context.Context, target Target) {
	ticker := time.NewTicker(time.Duration(c.cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		if err := c.collectSingBox(target); err != nil {
			log.Printf("sing-box %s: %v", target.NodeID, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Collector) collectSingBox(target Target) error {
	if err := c.flushPendingEvents(target); err != nil {
		log.Printf("sing-box %s pending event: %v", target.NodeID, err)
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(target.ClashAPI, "/")+"/connections", nil)
	if err != nil {
		return err
	}
	if target.ClashSecret != "" {
		req.Header.Set("Authorization", "Bearer "+target.ClashSecret)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("connections API returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}
	var payload ClashResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode connections: %w", err)
	}

	seen := make(map[string]bool, len(payload.Connections))
	for _, conn := range payload.Connections {
		if conn.ID == "" {
			continue
		}
		key := target.NodeID + ":" + conn.ID
		seen[key] = true
		c.mu.Lock()
		previous, exists := c.counters[key]
		c.counters[key] = counter{up: conn.Upload, down: conn.Download}
		c.mu.Unlock()

		upDelta, downDelta := conn.Upload, conn.Download
		eventType := "start"
		if exists {
			eventType = "delta"
			if conn.Upload >= previous.up {
				upDelta = conn.Upload - previous.up
			} else {
				upDelta = conn.Upload
			}
			if conn.Download >= previous.down {
				downDelta = conn.Download - previous.down
			} else {
				downDelta = conn.Download
			}
			if upDelta == 0 && downDelta == 0 {
				continue
			}
		}
		event := c.eventFromClash(target, conn, eventType, upDelta, downDelta)
		if err := c.queueSingBoxEvent(target, event); err != nil {
			return err
		}
	}
	if err := c.flushPendingEvents(target); err != nil {
		log.Printf("sing-box %s pending event: %v", target.NodeID, err)
	}

	c.mu.Lock()
	for key := range c.counters {
		if strings.HasPrefix(key, target.NodeID+":") && !seen[key] {
			delete(c.counters, key)
		}
	}
	c.mu.Unlock()
	return nil
}

func (c *Collector) queueSingBoxEvent(target Target, event Event) error {
	if event.UserID != nil {
		return c.post(event)
	}
	c.mu.Lock()
	if c.pendingEvents == nil {
		c.pendingEvents = make(map[string]pendingAuditEvent)
	}
	c.pendingEvents[event.EventID] = pendingAuditEvent{
		Target:    target,
		Event:     event,
		ExpiresAt: time.Now().Add(20 * time.Second),
	}
	c.mu.Unlock()
	return nil
}

func (c *Collector) resolvePendingEvent(item pendingAuditEvent) pendingAuditEvent {
	if item.Event.UserID != nil {
		return item
	}
	connectionID, _ := item.Event.Metadata["clash_connection_id"].(string)
	user := c.singBoxUserByConnection(item.Target, connectionID)
	matchMethod := "connection_id"
	if user == "" {
		user = c.singBoxUser(item.Target, item.Event.ClientSource, item.Event.Destination)
		matchMethod = "legacy_destination"
	}
	if user == "" {
		return item
	}
	item.Event.UserID = parseUserID(user)
	if item.Event.Metadata == nil {
		item.Event.Metadata = make(map[string]any)
	}
	item.Event.Metadata["user"] = user
	item.Event.Metadata["user_match"] = matchMethod
	return item
}

func (c *Collector) flushPendingEvents(target Target) error {
	now := time.Now()
	c.mu.Lock()
	pending := make([]pendingAuditEvent, 0, len(c.pendingEvents))
	for _, item := range c.pendingEvents {
		if item.Target.NodeID == target.NodeID {
			pending = append(pending, item)
		}
	}
	c.mu.Unlock()

	var lastErr error
	for _, item := range pending {
		resolved := c.resolvePendingEvent(item)
		if resolved.Event.UserID == nil && resolved.ExpiresAt.After(now) {
			continue
		}
		if err := c.post(resolved.Event); err != nil {
			lastErr = err
			continue
		}
		c.mu.Lock()
		delete(c.pendingEvents, resolved.Event.EventID)
		c.mu.Unlock()
	}
	return lastErr
}

func (c *Collector) eventFromClash(target Target, conn ClashConnection, eventType string, upload, download uint64) Event {
	destination := conn.Metadata.Host
	if destination == "" {
		destination = conn.Metadata.DestinationIP
	}
	if conn.Metadata.DestinationPort != "" && destination != "" {
		destination += ":" + conn.Metadata.DestinationPort
	}
	source := conn.Metadata.SourceIP
	if conn.Metadata.SourcePort != "" && source != "" {
		source += ":" + conn.Metadata.SourcePort
	}
	start := conn.Start.UTC().Format(time.RFC3339Nano)
	eventID := eventHash(target.NodeID, conn.ID, start, strconv.FormatUint(conn.Upload, 10), strconv.FormatUint(conn.Download, 10), eventType)
	user := conn.Metadata.User
	matchMethod := "clash_metadata"
	if user == "" {
		user = c.singBoxUserByConnection(target, conn.ID)
		matchMethod = "connection_id"
	}
	if user == "" {
		user = c.singBoxUser(target, source, destination)
		matchMethod = "legacy_destination"
	}
	metadata := map[string]any{"kernel": "sing-box", "inbound": conn.Metadata.Inbound, "rule": conn.Rule, "type": conn.Metadata.Type, "user": user, "user_match": matchMethod, "clash_connection_id": conn.ID}
	routeID, routeTag := panelRouteIdentity(conn.Rule)
	return Event{EventID: eventID, ConnectionID: "singbox-" + eventHash(target.NodeID, conn.ID, start), EventType: eventType, UserID: parseUserID(user), NodeID: target.NodeID, ClientSource: source, OccurredAt: time.Now().UTC(), Action: "observe", RuleTag: conn.Rule, RouteID: routeID, RouteTag: routeTag, RouteSource: "kernel", Outbound: strings.Join(conn.Chains, " > "), Destination: destination, DestinationIP: conn.Metadata.DestinationIP, Network: conn.Metadata.Network, Protocol: conn.Metadata.Type, UploadBytes: upload, DownloadBytes: download, Metadata: metadata}
}

func panelRouteIdentity(rule string) (*int64, string) {
	rule = strings.TrimSpace(rule)
	if !strings.HasPrefix(rule, "panel-route-") {
		return nil, ""
	}
	raw := strings.TrimPrefix(rule, "panel-route-")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, rule
	}
	return &id, rule
}

func (c *Collector) tailSingBoxJournal(unit string) {
	for {
		command := exec.Command("journalctl", "-u", unit, "-f", "-n", "0", "-o", "cat", "--no-pager")
		output, err := command.StdoutPipe()
		if err != nil {
			log.Printf("sing-box journal %s: %v", unit, err)
			time.Sleep(3 * time.Second)
			continue
		}
		if err := command.Start(); err != nil {
			log.Printf("sing-box journal %s: %v", unit, err)
			time.Sleep(3 * time.Second)
			continue
		}
		scanner := bufio.NewScanner(output)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			c.handleSingBoxLogLine(unit, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("sing-box journal %s scanner: %v", unit, err)
		}
		if err := command.Wait(); err != nil {
			log.Printf("sing-box journal %s exited: %v", unit, err)
		}
		time.Sleep(3 * time.Second)
	}
}

func (c *Collector) handleSingBoxLogLine(unit, line string) {
	if identityMatch := singBoxAuditIdentity.FindStringSubmatch(line); len(identityMatch) == 6 {
		user := "@" + identityMatch[1]
		connectionID := identityMatch[2]
		source := identityMatch[3]
		destinations := []string{identityMatch[4]}
		if identityMatch[5] != "" {
			if _, port, err := net.SplitHostPort(identityMatch[4]); err == nil && port != "" {
				destinations = append(destinations, net.JoinHostPort(identityMatch[5], port))
			}
		}
		c.mu.Lock()
		for _, destination := range destinations {
			identity := singBoxIdentity{
				User:      user,
				ExpiresAt: time.Now().Add(15 * time.Second),
			}
			c.singBoxIdentities[singBoxIdentityKey(unit, source, destination)] = identity
			if connectionID != "" {
				if c.singBoxConnections == nil {
					c.singBoxConnections = make(map[string]singBoxIdentity)
				}
				c.singBoxConnections[singBoxConnectionKey(unit, connectionID)] = identity
			}
			if c.singBoxRecentUsers == nil {
				c.singBoxRecentUsers = make(map[string]singBoxIdentity)
			}
			c.singBoxRecentUsers[singBoxDestinationKey(unit, destination)] = identity
		}
		c.mu.Unlock()
		return
	}
	contextMatch := singBoxContextID.FindStringSubmatch(line)
	if len(contextMatch) != 2 {
		return
	}
	contextKey := unit + ":" + contextMatch[1]
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.singBoxContexts[contextKey]
	entry.UpdatedAt = time.Now()
	if sourceMatch := singBoxSource.FindStringSubmatch(line); len(sourceMatch) == 2 {
		entry.Source = sourceMatch[1]
	}
	if userDestinationMatch := singBoxUserDestination.FindStringSubmatch(line); len(userDestinationMatch) == 3 {
		entry.User = userDestinationMatch[1]
		entry.Destination = userDestinationMatch[2]
	}
	c.singBoxContexts[contextKey] = entry
	if entry.Source != "" && entry.User != "" && entry.Destination != "" {
		c.singBoxIdentities[singBoxIdentityKey(unit, entry.Source, entry.Destination)] = singBoxIdentity{
			User:      entry.User,
			ExpiresAt: time.Now().Add(2 * time.Minute),
		}
	}
	for key, item := range c.singBoxIdentities {
		if item.ExpiresAt.Before(time.Now()) {
			delete(c.singBoxIdentities, key)
		}
	}
	for key, item := range c.singBoxConnections {
		if item.ExpiresAt.Before(time.Now()) {
			delete(c.singBoxConnections, key)
		}
	}
	for key, item := range c.singBoxContexts {
		if item.UpdatedAt.Before(time.Now().Add(-2 * time.Minute)) {
			delete(c.singBoxContexts, key)
		}
	}
}

func (c *Collector) singBoxUser(target Target, source, destination string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := singBoxIdentityKey(target.SingBoxJournalUnit, source, destination)
	item, found := c.singBoxIdentities[key]
	if !found || item.ExpiresAt.Before(time.Now()) {
		delete(c.singBoxIdentities, key)
		item, found = c.singBoxRecentUsers[singBoxDestinationKey(target.SingBoxJournalUnit, destination)]
		if !found || item.ExpiresAt.Before(time.Now()) {
			delete(c.singBoxRecentUsers, singBoxDestinationKey(target.SingBoxJournalUnit, destination))
			return ""
		}
	}
	return item.User
}

func (c *Collector) singBoxUserByConnection(target Target, connectionID string) string {
	if connectionID == "" {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := singBoxConnectionKey(target.SingBoxJournalUnit, connectionID)
	item, found := c.singBoxConnections[key]
	if !found || item.ExpiresAt.Before(time.Now()) {
		delete(c.singBoxConnections, key)
		return ""
	}
	return item.User
}

func singBoxIdentityKey(unit, source, destination string) string {
	return unit + "\x00" + strings.ToLower(strings.TrimSpace(source)) + "\x00" + strings.ToLower(strings.TrimSpace(destination))
}

func singBoxDestinationKey(unit, destination string) string {
	return unit + "\x00" + strings.ToLower(strings.TrimSpace(destination))
}

func singBoxConnectionKey(unit, connectionID string) string {
	return unit + "\x00" + strings.TrimSpace(connectionID)
}

func (c *Collector) tailXrayAccessLog(target Target) {
	for {
		if err := c.followFile(target); err != nil {
			log.Printf("xray %s: %v", target.NodeID, err)
			time.Sleep(3 * time.Second)
		}
	}
}

func (c *Collector) followFile(target Target) error {
	file, err := os.Open(target.XrayAccessLog)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err == nil {
			c.handleXrayLine(target, strings.TrimSpace(line))
			continue
		}
		if !errors.Is(err, io.EOF) {
			return err
		}
		time.Sleep(time.Second)
		stat, statErr := file.Stat()
		if statErr != nil {
			return statErr
		}
		position, _ := file.Seek(0, io.SeekCurrent)
		if stat.Size() < position {
			return nil
		} // rotated or truncated; reopen it.
	}
}

func (c *Collector) handleXrayLine(target Target, line string) {
	event := parseXrayLine(target, line)
	if event == nil {
		return
	}
	if err := c.post(*event); err != nil {
		log.Printf("xray %s post event: %v", target.NodeID, err)
	}
}

func parseXrayLine(target Target, line string) *Event {
	matches := xrayAccepted.FindStringSubmatch(line)
	if len(matches) != 3 {
		return nil
	}
	email := ""
	if user := xrayEmail.FindStringSubmatch(line); len(user) == 2 {
		email = user[1]
	}
	var upload, download uint64
	eventType := "start"
	metadata := map[string]any{"kernel": "xray", "email": email, "byte_counters": false}
	if traffic := xrayTraffic.FindStringSubmatch(line); len(traffic) == 3 {
		upload, _ = strconv.ParseUint(traffic[1], 10, 64)
		download, _ = strconv.ParseUint(traffic[2], 10, 64)
		eventType = "completed"
		metadata["byte_counters"] = true
	}
	lineHash := eventHash(target.NodeID, line)
	return &Event{
		EventID:       lineHash,
		ConnectionID:  "xray-" + lineHash,
		EventType:     eventType,
		UserID:        parseUserID(email),
		NodeID:        target.NodeID,
		ClientSource:  matches[1],
		OccurredAt:    time.Now().UTC(),
		Action:        "observe",
		Destination:   matches[2],
		Network:       networkOf(matches[2]),
		UploadBytes:   upload,
		DownloadBytes: download,
		Metadata:      metadata,
	}
}

func (c *Collector) post(event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(c.cfg.WebhookSecret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	req, err := http.NewRequest(http.MethodPost, c.cfg.PanelEventURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audit-Timestamp", timestamp)
	req.Header.Set("X-Audit-Signature", hex.EncodeToString(mac.Sum(nil)))
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		response, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("panel returned %s: %s", res.Status, strings.TrimSpace(string(response)))
	}
	return nil
}

func eventHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:64]
}
func parseUserID(value string) *int {
	match := userIDSuffix.FindStringSubmatch(value)
	if len(match) != 2 {
		return nil
	}
	n, err := strconv.Atoi(match[1])
	if err != nil {
		return nil
	}
	return &n
}
func networkOf(destination string) string {
	if strings.HasPrefix(destination, "udp:") {
		return "udp"
	}
	return "tcp"
}
