package utils

import (
	"strings"
	"testing"
)

func TestGenerateQueryCodeLength(t *testing.T) {
	code := GenerateQueryCode()
	if len(code) != 6 {
		t.Fatalf("expected length 6, got %d", len(code))
	}
}

func TestGenerateReportCodePrefix(t *testing.T) {
	code := GenerateReportCode(42)
	if !strings.HasPrefix(code, "CAC-42-") {
		t.Fatalf("unexpected code: %s", code)
	}
}
