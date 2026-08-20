package service

import (
	"strings"
	"testing"
)

func TestPublicSurveyURL(t *testing.T) {
	got := publicSurveyURL("/public/survey?token=abc")
	if !strings.Contains(got, "/public/survey?token=abc") {
		t.Fatalf("unexpected url: %s", got)
	}
	if !strings.HasPrefix(got, "http") {
		t.Fatalf("url should include scheme: %s", got)
	}
}
