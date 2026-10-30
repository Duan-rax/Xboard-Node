package geodata

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cedar2025/xboard-node/internal/nlog"
)

const (
	providerMetaCubeX  = "metacubex"
	providerSagerNet   = "sagernet"
	providerLoyal      = "loyalsoldier"
	providerCustom     = "custom"
	defaultGeoProvider = providerMetaCubeX
)

var httpClient = &http.Client{Timeout: 3 * time.Minute}

// Source describes database and sing-box rule-set download sources.
type Source struct {
	Provider          string
	Variant           string
	GeoIPURL          string
	GeoSiteURL        string
	GeoIPRuleSetURL   string
	GeoSiteRuleSetURL string
	CacheDir          string
}

// Requirement identifies one sing-box rule-set needed by a generated config.
type Requirement struct {
	Kind string // geoip or geosite
	Tag  string
}

// NormalizeSource applies the MetaCubeX full-data defaults and canonicalizes
// provider and variant names.
func NormalizeSource(src Source) Source {
	src.Provider = strings.ToLower(strings.TrimSpace(src.Provider))
	switch src.Provider {
	case "", "meta", "metacube", providerMetaCubeX:
		src.Provider = providerMetaCubeX
	case "sager", providerSagerNet:
		src.Provider = providerSagerNet
	case "loyal", providerLoyal:
		src.Provider = providerLoyal
	case providerCustom:
	default:
		src.Provider = defaultGeoProvider
	}
	src.Variant = strings.ToLower(strings.TrimSpace(src.Variant))
	if src.Variant != "lite" {
		src.Variant = "full"
	}
	return src
}

// DatabaseURLs returns the primary and fallback database URLs for a kernel.
func DatabaseURLs(src Source, kernelType, kind string) []string {
	src = NormalizeSource(src)
	if kind == "geoip" && src.GeoIPURL != "" {
		return []string{src.GeoIPURL}
	}
	if kind == "geosite" && src.GeoSiteURL != "" {
		return []string{src.GeoSiteURL}
	}

	var file string
	switch kind {
	case "geoip":
		file = "geoip"
	case "geosite":
		file = "geosite"
	default:
		return nil
	}

	if kernelType == "singbox" {
		if src.Provider == providerSagerNet {
			return []string{
				"https://cdn.jsdelivr.net/gh/SagerNet/sing-" + file + "@release/" + file + ".db",
				"https://github.com/SagerNet/sing-" + file + "/releases/latest/download/" + file + ".db",
			}
		}
		return []string{
			"https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/" + file + ".db",
			"https://github.com/MetaCubeX/meta-rules-dat/releases/latest/download/" + file + ".db",
		}
	}

	if src.Provider == providerMetaCubeX {
		return []string{
			"https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/" + file + ".dat",
			"https://github.com/MetaCubeX/meta-rules-dat/releases/latest/download/" + file + ".dat",
		}
	}
	// SagerNet does not publish Xray .dat files. Use Loyalsoldier as its
	// explicit Xray-compatible fallback.
	return []string{
		"https://cdn.jsdelivr.net/gh/Loyalsoldier/v2ray-rules-dat@release/" + file + ".dat",
		"https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/" + file + ".dat",
	}
}

// RuleSetURLs returns the primary and fallback sing-box SRS URLs.
func RuleSetURLs(src Source, kind, tag string) []string {
	src = NormalizeSource(src)
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	escapedTag := url.PathEscape(tag)
	var override string
	switch kind {
	case "geoip":
		override = src.GeoIPRuleSetURL
	case "geosite":
		override = src.GeoSiteRuleSetURL
	default:
		return nil
	}
	if override != "" {
		return []string{strings.ReplaceAll(override, "{tag}", escapedTag)}
	}

	if src.Provider == providerSagerNet {
		name := kind + "-" + escapedTag + ".srs"
		return []string{
			"https://cdn.jsdelivr.net/gh/SagerNet/sing-" + kind + "@rule-set/" + name,
			"https://raw.githubusercontent.com/SagerNet/sing-" + kind + "/rule-set/" + name,
		}
	}

	root := "geo"
	if src.Variant == "lite" {
		root = "geo-lite"
	}
	path := root + "/" + kind + "/" + escapedTag + ".srs"
	return []string{
		"https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/" + path,
		"https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/" + path,
	}
}

// RuleSetCachePath returns the node-owned cache path for one rule-set.
func RuleSetCachePath(src Source, kind, tag string) string {
	src = NormalizeSource(src)
	tag = strings.TrimSpace(tag)
	if tag == "" || filepath.Base(tag) != tag || strings.ContainsAny(tag, `/\`) {
		return ""
	}
	provider := src.Provider
	variant := "default"
	if provider == providerMetaCubeX {
		variant = src.Variant
	}
	return filepath.Join(src.CacheDir, "rulesets", provider, variant, kind, tag+".srs")
}

// Ensure downloads missing Geo databases for a kernel.
func Ensure(src Source, needGeoIP, needGeoSite bool, kernelType string) error {
	src = NormalizeSource(src)
	if err := os.MkdirAll(src.CacheDir, 0o755); err != nil {
		return fmt.Errorf("create geo_data_dir %q: %w", src.CacheDir, err)
	}

	type entry struct {
		name string
		kind string
	}
	var files []entry
	if kernelType == "singbox" {
		if needGeoIP {
			files = append(files, entry{"geoip.db", "geoip"})
		}
		if needGeoSite {
			files = append(files, entry{"geosite.db", "geosite"})
		}
	} else {
		if needGeoIP {
			files = append(files, entry{"geoip.dat", "geoip"})
		}
		if needGeoSite {
			files = append(files, entry{"geosite.dat", "geosite"})
		}
	}

	var firstErr error
	for _, f := range files {
		dst := filepath.Join(src.CacheDir, f.name)
		urls := DatabaseURLs(src, kernelType, f.kind)
		if err := ensureFile(dst, urls, 7*24*time.Hour); err != nil {
			nlog.Core().Warn("geo database auto-download failed", "file", f.name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// EnsureRuleSets downloads missing SRS files and returns the first error.
// Successfully downloaded files are used through local rule_set entries.
func EnsureRuleSets(src Source, requirements []Requirement) error {
	src = NormalizeSource(src)
	var firstErr error
	seen := make(map[string]struct{}, len(requirements))
	for _, requirement := range requirements {
		requirement.Kind = strings.ToLower(strings.TrimSpace(requirement.Kind))
		requirement.Tag = strings.TrimSpace(requirement.Tag)
		if requirement.Kind != "geoip" && requirement.Kind != "geosite" {
			continue
		}
		key := requirement.Kind + "\x00" + requirement.Tag
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		dst := RuleSetCachePath(src, requirement.Kind, requirement.Tag)
		if dst == "" {
			continue
		}
		urls := RuleSetURLs(src, requirement.Kind, requirement.Tag)
		if err := ensureFile(dst, urls, 24*time.Hour); err != nil {
			nlog.Core().Warn("rule-set auto-download failed",
				"kind", requirement.Kind, "tag", requirement.Tag, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func ensureFile(dst string, urls []string, maxAge time.Duration) error {
	stale := false
	if info, err := os.Stat(dst); err == nil && info.Size() > 0 {
		if maxAge <= 0 || time.Since(info.ModTime()) < maxAge {
			return nil
		}
		stale = true
	}
	if len(urls) == 0 {
		return fmt.Errorf("no download URL for %s", filepath.Base(dst))
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", dst, err)
	}

	var failures []string
	for _, sourceURL := range urls {
		nlog.Core().Info("geo data missing, downloading automatically", "file", filepath.Base(dst), "url", sourceURL)
		if err := atomicDownload(dst, sourceURL); err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if fi, err := os.Stat(dst); err == nil {
			nlog.Core().Info("geo data ready", "file", filepath.Base(dst), "size_kb", fi.Size()/1024)
		}
		return nil
	}
	if stale {
		nlog.Core().Warn("geo data refresh failed, keeping cached file",
			"file", filepath.Base(dst), "errors", strings.Join(failures, "; "))
		return nil
	}
	return fmt.Errorf("all download sources failed for %s: %s", filepath.Base(dst), strings.Join(failures, "; "))
}

func atomicDownload(dst, sourceURL string) error {
	resp, err := httpClient.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("http get %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status from %s: %s", sourceURL, resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(dst), ".geo-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("write geo data: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		return fmt.Errorf("rename to destination: %w", err)
	}
	return nil
}
