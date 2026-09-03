package service

import (
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestRenderWeeklyMarkdown(t *testing.T) {
	start := time.Date(2026, 8, 14, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 8, 20, 0, 0, 0, 0, time.Local)
	md := renderWeeklyMarkdown(start, end, 7, model.OpsManagerSnapshot{
		KPIs: model.OpsDashboardKPIs{CustomerFormal: 3, CustomerTrial: 2},
		Funnel: []model.OpsDashboardFunnelStage{
			{Key: "intent", Label: "意向客户", Count: 10, Rate: 100},
			{Key: "formal", Label: "正式", Count: 3, Rate: 30},
		},
		RiskSummary: model.OpsRiskSummary{Total: 2, TrialExpiring: 1, SettlementOverdue: 1},
		FactBullets: []string{"事实A"},
		SuggestedActions: []string{"督办逾期回款"},
		TopRisks: []model.OpsRiskItem{
			{Type: model.OpsRiskSettlementOverdue, Title: "结算已逾期", Summary: "测试", CustomerName: "甲"},
		},
	}, "")
	if !strings.Contains(md, "CacOps 运营周报") {
		t.Fatal("missing title")
	}
	if !strings.Contains(md, "督办逾期回款") {
		t.Fatal("missing action")
	}
	if !strings.Contains(md, "正式") {
		t.Fatal("missing funnel")
	}
}

func TestBuildManagerFactsAndActions(t *testing.T) {
	facts, actions := buildManagerFactsAndActions(
		&model.OpsDashboardOverview{
			KPIs: model.OpsDashboardKPIs{CustomerIntent: 1, VisitOverdue: 2},
			Funnel: []model.OpsDashboardFunnelStage{{Key: "trial", Label: "试用", Count: 5, Rate: 50}},
		},
		&model.OpsWorkbenchBriefing{Summary: model.OpsRiskSummary{Total: 3, SettlementOverdue: 2, TrialExpiring: 1}},
	)
	if len(facts) == 0 || len(actions) == 0 {
		t.Fatal("expected facts and actions")
	}
}
