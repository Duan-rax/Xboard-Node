package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cedar2025/xboard-node/internal/model"
)

func TestPersistentLastKnownGoodRoundTrip(t *testing.T) {
	stateDir := t.TempDir()
	s := &Service{stateDir: stateDir}
	config := &model.NodeSpec{ConfigRevision: 12, Protocol: "vless", ServerPort: 443}
	users := []model.UserSpec{{ID: 7, UUID: "00000000-0000-0000-0000-000000000007"}}

	s.persistLastKnownGood(config, users)
	loadedConfig, loadedUsers, err := s.loadPersistentLastKnownGood()
	if err != nil {
		t.Fatalf("load persistent state: %v", err)
	}
	if loadedConfig.ConfigRevision != config.ConfigRevision || loadedConfig.Protocol != config.Protocol {
		t.Fatalf("loaded config = %#v", loadedConfig)
	}
	if len(loadedUsers) != 1 || loadedUsers[0].ID != 7 {
		t.Fatalf("loaded users = %#v", loadedUsers)
	}
	info, err := os.Stat(filepath.Join(stateDir, "last-known-good.json"))
	if err != nil {
		t.Fatalf("state file missing: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state permissions = %o, want 600", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(stateDir, "last-known-good.meta.json")); err != nil {
		t.Fatalf("metadata file missing: %v", err)
	}
}

func TestLoadPersistentLastKnownGoodRejectsIncompleteState(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(stateDir, "last-known-good.json"),
		[]byte(`{"version":1,"config":null,"users":[]}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	s := &Service{stateDir: stateDir}
	if _, _, err := s.loadPersistentLastKnownGood(); err == nil {
		t.Fatal("expected incomplete state to be rejected")
	}
}
