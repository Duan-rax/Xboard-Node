package xray

import (
	"os"
	"strings"
	"testing"
)

func TestCompilerBoundaries(t *testing.T) {
	configSource := readCompilerFile(t, "config.go")
	for _, signature := range []string{
		"func buildRouting(",
		"func egressProxyToXray(",
		"func applyRealityRuntimeOverride(",
	} {
		if strings.Contains(configSource, signature) {
			t.Fatalf("config.go still owns %s", signature)
		}
	}

	files := map[string][]string{
		"route_compiler.go":    {"func buildRouting("},
		"outbound_compiler.go": {"func egressProxyToXray(", "func outboundConfigToXray("},
		"reality_override.go":  {"func applyRealityRuntimeOverride("},
	}
	for file, signatures := range files {
		source := readCompilerFile(t, file)
		for _, signature := range signatures {
			if !strings.Contains(source, signature) {
				t.Fatalf("%s does not define %s", file, signature)
			}
		}
	}
}

func readCompilerFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}
