package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestAllowedStageTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{model.OpsCustomerStageLead, model.OpsCustomerStageIntent, true},
		{model.OpsCustomerStageLead, model.OpsCustomerStageTrial, true},
		{model.OpsCustomerStageLead, model.OpsCustomerStageFormal, false},
		{model.OpsCustomerStageIntent, model.OpsCustomerStageTrial, true},
		{model.OpsCustomerStageTrial, model.OpsCustomerStageFormal, true},
		{model.OpsCustomerStageFormal, model.OpsCustomerStageClosed, true},
		{model.OpsCustomerStageClosed, model.OpsCustomerStageIntent, true},
	}
	for _, c := range cases {
		got := allowedStageTransitions[c.from][c.to]
		if got != c.ok {
			t.Fatalf("%s -> %s got %v want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestNeedNonRenewalSurvey(t *testing.T) {
	if needNonRenewalSurvey["已签约交付"] {
		t.Fatal("成功交付不应强制不续费问卷")
	}
	for _, r := range []string{"客户放弃", "竞品赢单", "预算不足", "需求变更", "长期无跟进", "其他"} {
		if !needNonRenewalSurvey[r] {
			t.Fatalf("%s 应强制不续费问卷", r)
		}
	}
}
