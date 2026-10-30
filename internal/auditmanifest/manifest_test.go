package auditmanifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSupportsSingBoxAndXrayTargets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-targets.json")
	targets := []Target{
		{NodeID: "4", PanelNodeID: 4, Kernel: "singbox", ClashAPI: "http://127.0.0.1:19094"},
		{NodeID: "28", PanelNodeID: 28, Kernel: "xray", XrayAccessLog: "/var/log/xboard-node/xray-28.log"},
	}
	if err := Write(path, targets); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(manifest.Targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(manifest.Targets))
	}
	if manifest.Targets[1].XrayAccessLog != "/var/log/xboard-node/xray-28.log" {
		t.Fatalf("xray target = %+v", manifest.Targets[1])
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("manifest mode = %o, want 600", info.Mode().Perm())
	}
}
