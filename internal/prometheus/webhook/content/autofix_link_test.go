package content

import (
	"strings"
	"testing"
)

func TestBuildAutoFixDeepLink(t *testing.T) {
	link := buildAutoFixDeepLink("localhost:5666", map[string]string{
		"namespace":  "prod",
		"deployment": "api",
		"alertname":  "HighCPU",
	}, map[string]string{
		"summary": "cpu high",
	}, "fp-1")

	if !strings.HasPrefix(link, "http://localhost:5666/autofix/workflow?") {
		t.Fatalf("unexpected prefix: %s", link)
	}
	for _, want := range []string{"namespace=prod", "deployment=api", "summary=cpu+high", "alertFingerprint=fp-1"} {
		if !strings.Contains(link, want) {
			t.Fatalf("link %s missing %s", link, want)
		}
	}
}

func TestBuildAutoFixDeepLinkNilMaps(t *testing.T) {
	link := buildAutoFixDeepLink("", nil, nil, "")
	if !strings.HasPrefix(link, "http://localhost:3000/autofix/workflow?") {
		t.Fatalf("unexpected link: %s", link)
	}
}
