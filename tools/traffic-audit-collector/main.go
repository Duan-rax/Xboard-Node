package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
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
	PanelEventURL string   `json:"panel_event_url"`
	WebhookSecret string   `json:"webhook_secret"`
	IntervalSeconds int    `json:"interval_seconds"`
	Targets        []Target `json:"targets"`
}

type Target struct {
	NodeID        string `json:"node_id"`
	Kernel        string `json:"kernel"`
	ClashAPI      string `json:"clash_api"`
	ClashSecret   string `json:"clash_secret"`
	SingBoxJournalUnit string `json:"singbox_journal_unit"`
	XrayAccessLog string `json:"xray_access_log"`
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

type counter struct { up, down uint64 }

type singBoxLogContext struct {
	Source      string
	User        string
	Destination string
	UpdatedAt   time.Time
}

type singBoxIdentity struct {
	User      string
	ExpiresAt time.Time
}

type Collector struct {
	cfg     Config
	client  *http.Client
	mu      sync.Mutex
	counters map[string]counter
	singBoxContexts  map[string]singBoxLogContext
	singBoxIdentities map[string]singBoxIdentity
}

var (
	xrayAccepted = regexp.MustCompile(`from\s+([^\s]+)\s+accepted\s+([^\s]+)`)
	xrayEmail    = regexp.MustCompile(`email:\s*([^\s]+)`)
	userIDSuffix = regexp.MustCompile(`(?:^|@)([1-9][0-9]*)$`)
	singBoxContextID = regexp.MustCompile(`\[(\d+)(?:\s+[^\]]+)?\]`)
	singBoxSource = regexp.MustCompile(`inbound connection from ([^\s]+)`)
	singBoxUserDestination = regexp.MustCompile(`\]: \[([^\]]+)\] inbound(?: packet)? connection to ([^\s]+)`)
)

func main() {
	configPath := flag.String("config", "/etc/traffic-audit-collector/config.json", "collector config path")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil { log.Fatal(err) }
	collector := &Collector{
		cfg: cfg,
		client: &http.Client{Timeout: 8 * time.Second},
		counters: make(map[string]counter),
		singBoxContexts: make(map[string]singBoxLogContext),
		singBoxIdentities: make(map[string]singBoxIdentity),
	}

	for _, target := range cfg.Targets {
		target := target
		if target.ClashAPI != "" {
			go collector.pollSingBox(target)
		}
		if target.SingBoxJournalUnit != "" {
			go collector.tailSingBoxJournal(target)
		}
		if target.XrayAccessLog != "" {
			go collector.tailXrayAccessLog(target)
		}
	}
	select {}
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil { return Config{}, fmt.Errorf("read config: %w", err) }
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil { return Config{}, fmt.Errorf("parse config: %w", err) }
	if cfg.PanelEventURL == "" || cfg.WebhookSecret == "" { return Config{}, errors.New("panel_event_url and webhook_secret are required") }
	if len(cfg.Targets) == 0 { return Config{}, errors.New("at least one target is required") }
	if cfg.IntervalSeconds < 1 { cfg.IntervalSeconds = 2 }
	for i := range cfg.Targets {
		if cfg.Targets[i].NodeID == "" { return Config{}, fmt.Errorf("targets[%d].node_id is required", i) }
	}
	return cfg, nil
}

func (c *Collector) pollSingBox(target Target) {
	ticker := time.NewTicker(time.Duration(c.cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()
	for {
		if err := c.collectSingBox(target); err != nil { log.Printf("sing-box %s: %v", target.NodeID, err) }
		<-ticker.C
	}
}

func (c *Collector) collectSingBox(target Target) error {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(target.ClashAPI, "/")+"/connections", nil)
	if err != nil { return err }
	if target.ClashSecret != "" { req.Header.Set("Authorization", "Bearer "+target.ClashSecret) }
	res, err := c.client.Do(req)
	if err != nil { return err }
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK { body, _ := io.ReadAll(io.LimitReader(res.Body, 1024)); return fmt.Errorf("connections API returned %s: %s", res.Status, strings.TrimSpace(string(body))) }
	var payload ClashResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil { return fmt.Errorf("decode connections: %w", err) }

	seen := make(map[string]bool, len(payload.Connections))
	for _, conn := range payload.Connections {
		if conn.ID == "" { continue }
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
			if conn.Upload >= previous.up { upDelta = conn.Upload - previous.up } else { upDelta = conn.Upload }
			if conn.Download >= previous.down { downDelta = conn.Download - previous.down } else { downDelta = conn.Download }
			if upDelta == 0 && downDelta == 0 { continue }
		}
		event := c.eventFromClash(target, conn, eventType, upDelta, downDelta)
		if err := c.post(event); err != nil { return err }
	}

	c.mu.Lock()
	for key := range c.counters {
		if strings.HasPrefix(key, target.NodeID+":") && !seen[key] { delete(c.counters, key) }
	}
	c.mu.Unlock()
	return nil
}

func (c *Collector) eventFromClash(target Target, conn ClashConnection, eventType string, upload, download uint64) Event {
	destination := conn.Metadata.Host
	if destination == "" { destination = conn.Metadata.DestinationIP }
	if conn.Metadata.DestinationPort != "" && destination != "" { destination += ":" + conn.Metadata.DestinationPort }
	source := conn.Metadata.SourceIP
	if conn.Metadata.SourcePort != "" && source != "" { source += ":" + conn.Metadata.SourcePort }
	start := conn.Start.UTC().Format(time.RFC3339Nano)
	eventID := eventHash(target.NodeID, conn.ID, start, strconv.FormatUint(conn.Upload, 10), strconv.FormatUint(conn.Download, 10), eventType)
	user := conn.Metadata.User
	if user == "" {
		user = c.singBoxUser(target, source, destination)
	}
	metadata := map[string]any{"kernel": "sing-box", "inbound": conn.Metadata.Inbound, "rule": conn.Rule, "type": conn.Metadata.Type, "user": user}
	return Event{EventID: eventID, ConnectionID: "singbox-" + eventHash(target.NodeID, conn.ID, start), EventType: eventType, UserID: parseUserID(user), NodeID: target.NodeID, ClientSource: source, OccurredAt: time.Now().UTC(), Action: "observe", RuleTag: conn.Rule, Destination: destination, DestinationIP: conn.Metadata.DestinationIP, Network: conn.Metadata.Network, Protocol: conn.Metadata.Type, UploadBytes: upload, DownloadBytes: download, Metadata: metadata}
}

func (c *Collector) tailSingBoxJournal(target Target) {
	for {
		command := exec.Command("journalctl", "-u", target.SingBoxJournalUnit, "-f", "-n", "0", "-o", "cat", "--no-pager")
		output, err := command.StdoutPipe()
		if err != nil {
			log.Printf("sing-box journal %s: %v", target.NodeID, err)
			time.Sleep(3 * time.Second)
			continue
		}
		if err := command.Start(); err != nil {
			log.Printf("sing-box journal %s: %v", target.NodeID, err)
			time.Sleep(3 * time.Second)
			continue
		}
		scanner := bufio.NewScanner(output)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			c.handleSingBoxLogLine(target, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("sing-box journal %s scanner: %v", target.NodeID, err)
		}
		if err := command.Wait(); err != nil {
			log.Printf("sing-box journal %s exited: %v", target.NodeID, err)
		}
		time.Sleep(3 * time.Second)
	}
}

func (c *Collector) handleSingBoxLogLine(target Target, line string) {
	contextMatch := singBoxContextID.FindStringSubmatch(line)
	if len(contextMatch) != 2 {
		return
	}
	contextKey := target.NodeID + ":" + contextMatch[1]
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
		c.singBoxIdentities[singBoxIdentityKey(target, entry.Source, entry.Destination)] = singBoxIdentity{
			User: entry.User,
			ExpiresAt: time.Now().Add(2 * time.Minute),
		}
	}
	for key, item := range c.singBoxIdentities {
		if item.ExpiresAt.Before(time.Now()) {
			delete(c.singBoxIdentities, key)
		}
	}
}

func (c *Collector) singBoxUser(target Target, source, destination string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := singBoxIdentityKey(target, source, destination)
	item, found := c.singBoxIdentities[key]
	if !found || item.ExpiresAt.Before(time.Now()) {
		delete(c.singBoxIdentities, key)
		return ""
	}
	return item.User
}

func singBoxIdentityKey(target Target, source, destination string) string {
	return target.NodeID + "\x00" + strings.ToLower(strings.TrimSpace(source)) + "\x00" + strings.ToLower(strings.TrimSpace(destination))
}

func (c *Collector) tailXrayAccessLog(target Target) {
	for {
		if err := c.followFile(target); err != nil { log.Printf("xray %s: %v", target.NodeID, err); time.Sleep(3 * time.Second) }
	}
}

func (c *Collector) followFile(target Target) error {
	file, err := os.Open(target.XrayAccessLog)
	if err != nil { return err }
	defer file.Close()
	if _, err := file.Seek(0, io.SeekEnd); err != nil { return err }
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err == nil {
			c.handleXrayLine(target, strings.TrimSpace(line))
			continue
		}
		if !errors.Is(err, io.EOF) { return err }
		time.Sleep(time.Second)
		stat, statErr := file.Stat()
		if statErr != nil { return statErr }
		position, _ := file.Seek(0, io.SeekCurrent)
		if stat.Size() < position { return nil } // rotated or truncated; reopen it.
	}
}

func (c *Collector) handleXrayLine(target Target, line string) {
	matches := xrayAccepted.FindStringSubmatch(line)
	if len(matches) != 3 { return }
	email := ""
	if user := xrayEmail.FindStringSubmatch(line); len(user) == 2 { email = user[1] }
	event := Event{EventID: eventHash(target.NodeID, line), ConnectionID: "xray-" + eventHash(target.NodeID, line), EventType: "start", UserID: parseUserID(email), NodeID: target.NodeID, ClientSource: matches[1], OccurredAt: time.Now().UTC(), Action: "observe", Destination: matches[2], Network: networkOf(matches[2]), Protocol: "", Metadata: map[string]any{"kernel": "xray", "email": email}}
	if err := c.post(event); err != nil { log.Printf("xray %s post event: %v", target.NodeID, err) }
}

func (c *Collector) post(event Event) error {
	body, err := json.Marshal(event)
	if err != nil { return err }
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(c.cfg.WebhookSecret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	req, err := http.NewRequest(http.MethodPost, c.cfg.PanelEventURL, bytes.NewReader(body))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Audit-Timestamp", timestamp)
	req.Header.Set("X-Audit-Signature", hex.EncodeToString(mac.Sum(nil)))
	res, err := c.client.Do(req)
	if err != nil { return err }
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 { response, _ := io.ReadAll(io.LimitReader(res.Body, 1024)); return fmt.Errorf("panel returned %s: %s", res.Status, strings.TrimSpace(string(response))) }
	return nil
}

func eventHash(parts ...string) string { sum := sha256.Sum256([]byte(strings.Join(parts, "\x00"))); return hex.EncodeToString(sum[:])[:64] }
func parseUserID(value string) *int { match := userIDSuffix.FindStringSubmatch(value); if len(match) != 2 { return nil }; n, err := strconv.Atoi(match[1]); if err != nil { return nil }; return &n }
func networkOf(destination string) string { if strings.HasPrefix(destination, "udp:") { return "udp" }; return "tcp" }
