package service

import (
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestSameDay(t *testing.T) {
	a := time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)
	b := time.Date(2026, 8, 1, 23, 59, 0, 0, time.Local)
	c := time.Date(2026, 8, 2, 0, 0, 0, 0, time.Local)
	if !sameDay(a, b) {
		t.Fatal("same calendar day should match")
	}
	if sameDay(a, c) {
		t.Fatal("different day should not match")
	}
}

func TestIsMonthlyBilling(t *testing.T) {
	cases := []struct {
		cycle, mode string
		ok          bool
	}{
		{"", "", true},
		{"monthly", "", true},
		{"", "monthly", true},
		{"yearly", "", false},
		{"yearly", "monthly", false},
		{"quarterly", "usage", false},
	}
	for _, c := range cases {
		got := isMonthlyBilling(&model.OpsContract{BillingCycle: c.cycle, BillingMode: c.mode})
		if got != c.ok {
			t.Fatalf("cycle=%q mode=%q got %v want %v", c.cycle, c.mode, got, c.ok)
		}
	}
}
