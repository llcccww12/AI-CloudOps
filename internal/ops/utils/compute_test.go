package utils

import (
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestComputeLifeStatus(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.Local)
	openFuture := time.Date(2026, 9, 10, 0, 0, 0, 0, time.Local)
	openPast := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	planPast := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	planFuture := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	released := time.Date(2026, 9, 2, 0, 0, 0, 0, time.Local)

	cases := []struct {
		name string
		open, plan, actual *time.Time
		want string
	}{
		{"released", &openPast, &planPast, &released, model.OpsComputeLifeReleased},
		{"pending_open", &openFuture, &planFuture, nil, model.OpsComputeLifePendingOpen},
		{"pending_release", &openPast, &planPast, nil, model.OpsComputeLifePendingRel},
		{"in_use", &openPast, &planFuture, nil, model.OpsComputeLifeInUse},
		{"no_open", nil, nil, nil, model.OpsComputeLifePendingOpen},
	}
	for _, c := range cases {
		got := ComputeLifeStatus(c.open, c.plan, c.actual, now)
		if got != c.want {
			t.Fatalf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestCapacityCheck(t *testing.T) {
	n := 8
	check, label := CapacityCheck(10, &n)
	if check != model.OpsComputeCapacityOverbook || label != "超配" {
		t.Fatalf("expect overbook, got %s %s", check, label)
	}
	check, label = CapacityCheck(4, &n)
	if check != model.OpsComputeCapacityOK {
		t.Fatalf("expect ok, got %s", check)
	}
}

func TestComputeLifeStatusLabel(t *testing.T) {
	if ComputeLifeStatusLabel(model.OpsComputeLifePendingRel) != "到期待处理" {
		t.Fatalf("pending_release label should be 到期待处理")
	}
	if ComputeLifeStatusLabel(model.OpsComputeLifeInUse) != "在用" {
		t.Fatalf("in_use label")
	}
}

func TestComputeBizPhaseLabel(t *testing.T) {
	if ComputeBizPhaseLabel(model.OpsComputePhaseTrial) != "测试" {
		t.Fatal("trial")
	}
	if ComputeBizPhaseLabel(model.OpsComputePhaseFormal) != "正式" {
		t.Fatal("formal")
	}
	if ComputeBizPhaseLabel(model.OpsComputePhaseRenew) != "续签" {
		t.Fatal("renew")
	}
}

func TestFormatPhaseHistorySummary(t *testing.T) {
	s1 := time.Date(2025, 7, 1, 0, 0, 0, 0, time.Local)
	e1 := time.Date(2025, 9, 30, 0, 0, 0, 0, time.Local)
	s2 := time.Date(2025, 10, 1, 0, 0, 0, 0, time.Local)
	e2 := time.Date(2025, 12, 31, 0, 0, 0, 0, time.Local)
	hist := AppendPhaseSegment(nil, "", model.OpsComputePhaseTrial, &s1, &e1, 0, "T-1", "temp_test", 8, "测试")
	hist = AppendPhaseSegment(hist, model.OpsComputePhaseTrial, model.OpsComputePhaseFormal, &s2, &e2, 11, "F-1", "full", 8, "转正")
	sum := FormatPhaseHistorySummary(hist)
	want := "测试 2025-07-01~2025-09-30(T-1) → 正式 2025-10-01~2025-12-31(F-1)"
	if sum != want {
		t.Fatalf("summary=%q want=%q", sum, want)
	}
	if len(hist) != 2 {
		t.Fatalf("want 2 segments, got %d", len(hist))
	}
	// 同阶段同合同延期不增段
	e3 := time.Date(2026, 6, 30, 0, 0, 0, 0, time.Local)
	hist = AppendPhaseSegment(hist, model.OpsComputePhaseFormal, model.OpsComputePhaseFormal, &s2, &e3, 11, "F-1", "full", 8, "延期")
	if len(hist) != 2 {
		t.Fatalf("extend same contract should keep 2 segments, got %d", len(hist))
	}
	// 续签换新合同 → 新段，旧合同保留在上一段
	s3 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	e4 := time.Date(2027, 6, 30, 0, 0, 0, 0, time.Local)
	hist = AppendPhaseSegment(hist, model.OpsComputePhaseFormal, model.OpsComputePhaseRenew, &s3, &e4, 22, "R-2", "full", 8, "续签新合同")
	if len(hist) != 3 {
		t.Fatalf("new contract should append segment, got %d", len(hist))
	}
	if hist[1].ContractNo != "F-1" || hist[2].ContractNo != "R-2" {
		t.Fatalf("contracts preserved/appended: %#v %#v", hist[1], hist[2])
	}
	// 同阶段续签且未传合同 → 沿用，只延期
	e5 := time.Date(2028, 6, 30, 0, 0, 0, 0, time.Local)
	hist = AppendPhaseSegment(hist, model.OpsComputePhaseRenew, model.OpsComputePhaseRenew, &s3, &e5, 0, "", "full", 8, "再续签沿用")
	if len(hist) != 3 {
		t.Fatalf("reuse contract same phase should not append, got %d", len(hist))
	}
	if hist[2].ContractNo != "R-2" || formatDay(hist[2].EndAt) != "2028-06-30" {
		t.Fatalf("should extend last renew segment: %#v", hist[2])
	}
	trail := FormatContractTrailSummary(hist)
	if trail != "T-1 → F-1 → R-2" {
		t.Fatalf("trail=%q", trail)
	}
}

func TestOccupyingGPUs(t *testing.T) {
	if OccupyingGPUs(model.OpsComputeLifeInUse, 8) != 8 {
		t.Fatal("in_use should occupy")
	}
	if OccupyingGPUs(model.OpsComputeLifePendingRel, 8) != 8 {
		t.Fatal("pending_release should still occupy")
	}
	if OccupyingGPUs(model.OpsComputeLifePendingOpen, 8) != 8 {
		t.Fatal("pending_open should reserve")
	}
	if OccupyingGPUs(model.OpsComputeLifeReleased, 8) != 0 {
		t.Fatal("released should not occupy")
	}
}
