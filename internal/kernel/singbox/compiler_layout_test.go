package singbox

import (
	"os"
	"strings"
	"testing"
)

func TestCompilerBoundaries(t *testing.T) {
	configSource := readCompilerFile(t, "config.go")
	for _, signature := range []string{
		"func buildRoutes(",
		"func egressProxyToSingbox(",
		"func automaticGeoRuleSets(",
		"func mergeSingboxRuleSets(",
	} {
		if strings.Contains(configSource, signature) {
			t.Fatalf("config.go still owns %s", signature)
		}
	}

	files := map[string][]string{
		"route_compiler.go":    {"func buildRoutes(", "func singboxMatchGroups("},
		"outbound_compiler.go": {"func egressProxyToSingbox(", "func outboundConfigToSingbox("},
		"rule_set_compiler.go": {"func automaticGeoRuleSets(", "func mergeSingboxRuleSets("},
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
