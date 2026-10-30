package geodata

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuleSetURLsProviders(t *testing.T) {
	meta := NormalizeSource(Source{Provider: "metacubex", Variant: "lite"})
	urls := RuleSetURLs(meta, "geosite", "category-ads-all")
	if len(urls) != 2 || !strings.Contains(urls[0], "testingcf.jsdelivr.net") ||
		!strings.Contains(urls[0], "/geo-lite/geosite/category-ads-all.srs") {
		t.Fatalf("unexpected MetaCubeX URLs: %#v", urls)
	}
	if !strings.Contains(urls[1], "raw.githubusercontent.com") {
		t.Fatalf("MetaCubeX fallback URL missing: %#v", urls)
	}

	sager := NormalizeSource(Source{Provider: "sagernet"})
	urls = RuleSetURLs(sager, "geoip", "cn")
	if len(urls) != 2 {
		t.Fatalf("unexpected SagerNet URLs: %#v", urls)
	}
	if !strings.HasSuffix(urls[0], "/geoip-cn.srs") || !strings.HasSuffix(urls[1], "/geoip-cn.srs") {
		t.Fatalf("SagerNet file names are incorrect: %#v", urls)
	}
}

func TestRuleSetURLsCustomTemplate(t *testing.T) {
	src := NormalizeSource(Source{
		Provider:          "custom",
		GeoSiteRuleSetURL: "https://rules.example/{tag}.srs",
	})
	urls := RuleSetURLs(src, "geosite", "category-ads-all")
	if len(urls) != 1 || urls[0] != "https://rules.example/category-ads-all.srs" {
		t.Fatalf("custom template not expanded: %#v", urls)
	}
}

func TestRuleSetCachePathSeparatesProviders(t *testing.T) {
	dir := t.TempDir()
	meta := RuleSetCachePath(Source{Provider: "metacubex", Variant: "lite", CacheDir: dir}, "geoip", "cn")
	sager := RuleSetCachePath(Source{Provider: "sagernet", CacheDir: dir}, "geoip", "cn")
	if meta == sager {
		t.Fatalf("provider cache paths must differ: %q", meta)
	}
	if got := RuleSetCachePath(Source{CacheDir: dir}, "geoip", "../cn"); got != "" {
		t.Fatalf("unsafe tag accepted: %q", got)
	}
}

func TestEnsureFileFallsBackAndRefreshes(t *testing.T) {
	var fallbackCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/primary" {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		fallbackCalls++
		_, _ = w.Write([]byte("rule-set"))
	}))
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "cn.srs")
	urls := []string{server.URL + "/primary", server.URL + "/fallback"}
	if err := ensureFile(dst, urls, time.Hour); err != nil {
		t.Fatalf("ensureFile: %v", err)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fallback calls = %d, want 1", fallbackCalls)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "rule-set" {
		t.Fatalf("downloaded data = %q, err = %v", data, err)
	}
	if err := ensureFile(dst, urls, time.Hour); err != nil {
		t.Fatalf("fresh cache should be reused: %v", err)
	}
	if fallbackCalls != 1 {
		t.Fatalf("fresh cache caused another download: %d calls", fallbackCalls)
	}
}
func TestAtomicDownloadRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(maxDownloadSize+1))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dst := filepath.Join(t.TempDir(), "oversized.srs")
	if err := atomicDownload(dst, server.URL); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("oversized response error = %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("oversized response left destination behind: %v", err)
	}
}

