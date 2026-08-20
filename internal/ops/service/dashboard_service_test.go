package service

import (
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestFillTrendDays(t *testing.T) {
	start := time.Date(2026, 8, 11, 0, 0, 0, 0, time.Local)
	raw := []model.OpsDashboardTrendPoint{
		{Date: "2026-08-12", Count: 3},
		{Date: "2026-08-14", Count: 1},
	}
	got := fillTrendDays(start, 5, raw)
	if len(got) != 5 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Date != "2026-08-11" || got[0].Count != 0 {
		t.Fatalf("day0=%+v", got[0])
	}
	if got[1].Count != 3 || got[3].Count != 1 {
		t.Fatalf("unexpected counts: %+v", got)
	}
}

func TestRound1(t *testing.T) {
	if round1(33.333) != 33.3 {
		t.Fatalf("got %v", round1(33.333))
	}
	if round1(50) != 50 {
		t.Fatalf("got %v", round1(50))
	}
}

func TestBuildVisitAlertsDedup(t *testing.T) {
	now := time.Now()
	due := now.Add(-time.Hour)
	v := &model.OpsVisit{Model: model.Model{ID: 7}, Title: "t", TargetOrg: "org", DueAt: &due, Intent: "high"}
	out := buildVisitAlerts([]*model.OpsVisit{v}, []*model.OpsVisit{v}, now)
	if len(out) != 1 {
		t.Fatalf("want 1 got %d", len(out))
	}
	if out[0].AlertType != "overdue" {
		t.Fatalf("type=%s", out[0].AlertType)
	}
}
