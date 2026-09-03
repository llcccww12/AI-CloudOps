package service

import (
	"os"
	"strings"
	"testing"
)

func TestResolveLLMConfigModelFromMiniMaxURL(t *testing.T) {
	t.Setenv("LLM_API_KEY", "sk-test")
	t.Setenv("LLM_BASE_URL", "https://api.minimaxi.com/v1")
	os.Unsetenv("LLM_MODEL")

	_, _, model, err := resolveLLMConfig()
	if err != nil {
		t.Fatal(err)
	}
	if model != "MiniMax-M3" {
		t.Fatalf("want MiniMax-M3, got %s", model)
	}
}

func TestResolveLLMConfigModelEnvOverride(t *testing.T) {
	t.Setenv("LLM_API_KEY", "sk-test")
	t.Setenv("LLM_BASE_URL", "https://api.minimaxi.com/v1")
	t.Setenv("LLM_MODEL", "MiniMax-Text-01")

	_, _, model, err := resolveLLMConfig()
	if err != nil {
		t.Fatal(err)
	}
	if model != "MiniMax-Text-01" {
		t.Fatalf("want MiniMax-Text-01, got %s", model)
	}
}

func TestSanitizeExecutiveSummaryStripsThink(t *testing.T) {
	in := "<think>\ninternal notes\n</think>\n本周正式客户 3，试用 1，建议催收逾期。"
	got := sanitizeExecutiveSummary(in)
	if strings.Contains(got, "think") || strings.Contains(got, "internal") {
		t.Fatalf("think tags not stripped: %q", got)
	}
	if !strings.Contains(got, "正式客户") {
		t.Fatalf("summary lost: %q", got)
	}
}
