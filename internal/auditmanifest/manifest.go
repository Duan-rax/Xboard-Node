// Package auditmanifest publishes the local sing-box audit endpoints that are
// created for nodes dynamically discovered by machine mode.
package auditmanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Target struct {
	NodeID             string `json:"node_id"`
	PanelNodeID        int    `json:"panel_node_id"`
	Kernel             string `json:"kernel"`
	ClashAPI           string `json:"clash_api"`
	ClashSecret        string `json:"clash_secret"`
	SingBoxJournalUnit string `json:"singbox_journal_unit"`
}

type Manifest struct {
	Version int      `json:"version"`
	Targets []Target `json:"targets"`
}

// Write atomically replaces a root-only manifest. The manifest deliberately
// contains local-only controller secrets, so it is always mode 0600.
func Write(path string, targets []Target) error {
	if path == "" {
		return nil
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].PanelNodeID < targets[j].PanelNodeID })
	data, err := json.MarshalIndent(Manifest{Version: 1, Targets: targets}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal audit manifest: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create audit manifest directory: %w", err)
	}
	temporary, err := os.CreateTemp(dir, ".traffic-audit-*.json")
	if err != nil {
		return fmt.Errorf("create audit manifest: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("chmod audit manifest: %w", err)
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return fmt.Errorf("write audit manifest: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close audit manifest: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace audit manifest: %w", err)
	}
	return os.Chmod(path, 0600)
}
