package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestPublicFaultDisplayTitle(t *testing.T) {
	got := publicFaultDisplayTitle("[客户报障]系统故障-WO202508270001")
	if got != "系统故障" {
		t.Fatalf("expected 系统故障, got %s", got)
	}
	if publicFaultDisplayTitle("普通标题") != "普通标题" {
		t.Fatal("plain title should stay unchanged")
	}
}

func TestPriorityLabel(t *testing.T) {
	if priorityLabel(model.PriorityHigh) != "高" {
		t.Fatal("high priority label mismatch")
	}
}
