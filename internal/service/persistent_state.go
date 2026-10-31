package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cedar2025/xboard-node/internal/model"
	"github.com/cedar2025/xboard-node/internal/nlog"
)

const persistentStateVersion = 1

type persistentLastKnownGood struct {
	Version int              `json:"version"`
	Config  *model.NodeSpec  `json:"config"`
	Users   []model.UserSpec `json:"users"`
}

type persistentLastKnownGoodMeta struct {
	Version        int    `json:"version"`
	SavedAt        int64  `json:"saved_at"`
	ConfigRevision int64  `json:"config_revision"`
	ConfigHash     string `json:"config_hash"`
	UserHash       string `json:"user_hash"`
}

func (s *Service) persistLastKnownGood(config *model.NodeSpec, users []model.UserSpec) {
	if config == nil || len(users) == 0 || s.stateDir == "" {
		return
	}
	state := persistentLastKnownGood{
		Version: persistentStateVersion,
		Config:  config,
		Users:   append([]model.UserSpec(nil), users...),
	}
	meta := persistentLastKnownGoodMeta{
		Version:        persistentStateVersion,
		SavedAt:        time.Now().Unix(),
		ConfigRevision: config.ConfigRevision,
		ConfigHash:     computeConfigHash(config),
		UserHash:       computeUserHash(users),
	}
	if err := writeJSONAtomic(filepath.Join(s.stateDir, "last-known-good.json"), state); err != nil {
		nlog.Core().Warn("persist last-known-good config failed", "error", err)
		return
	}
	if err := writeJSONAtomic(filepath.Join(s.stateDir, "last-known-good.meta.json"), meta); err != nil {
		nlog.Core().Warn("persist last-known-good metadata failed", "error", err)
	}
}

func (s *Service) loadPersistentLastKnownGood() (*model.NodeSpec, []model.UserSpec, error) {
	if s.stateDir == "" {
		return nil, nil, errors.New("persistent state directory is empty")
	}
	data, err := os.ReadFile(filepath.Join(s.stateDir, "last-known-good.json"))
	if err != nil {
		return nil, nil, err
	}
	var state persistentLastKnownGood
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, nil, fmt.Errorf("decode last-known-good: %w", err)
	}
	if state.Version != persistentStateVersion || state.Config == nil || len(state.Users) == 0 {
		return nil, nil, errors.New("last-known-good state is incomplete")
	}
	return state.Config, state.Users, nil
}

func (s *Service) restorePersistentLastKnownGood(ctx context.Context) bool {
	config, users, err := s.loadPersistentLastKnownGood()
	if err != nil {
		if !os.IsNotExist(err) {
			nlog.Core().Warn("load persistent last-known-good failed", "error", err)
		}
		return false
	}
	if err := validateNodeRuntime(s.cfg, s.kernel.Protocols(), config, s.cert.TLSCert()); err != nil {
		nlog.Core().Warn("persistent last-known-good validation failed", "error", err)
		return false
	}
	s.metricsMu.Lock()
	s.lastConfig = config
	s.lastUsers = append([]model.UserSpec(nil), users...)
	s.lastConfigHash = computeConfigHash(config)
	s.lastUserHash = computeUserHash(users)
	s.metricsMu.Unlock()
	s.applyRemoteOverrides(ctx, config)
	if !s.startKernel(config, users) {
		return false
	}
	nlog.Core().Warn("persistent last-known-good restored",
		"revision", config.ConfigRevision,
		"users", len(users),
	)
	return true
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
