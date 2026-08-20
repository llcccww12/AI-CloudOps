package service

import "testing"

import "github.com/GoSimplicity/AI-CloudOps/internal/model"

func TestNormalizeLifecycleStep(t *testing.T) {
	cases := []struct {
		id, name, want string
	}{
		{"intent", "意向确认", model.OpsLifecycleStepIntent},
		{"trial", "试用审批", model.OpsLifecycleStepTrial},
		{"trial_contract", "试用合同签约", model.OpsLifecycleStepTrialContract},
		{"open_request", "开通申请", model.OpsLifecycleStepOpenRequest},
		{"open_feedback", "开通回馈", model.OpsLifecycleStepOpenFeedback},
		{"trial_accept", "试用验收/转正评估", model.OpsLifecycleStepTrialAccept},
		{"x", "验收评估", model.OpsLifecycleStepTrialAccept},
		{"contract", "正式合同", model.OpsLifecycleStepContract},
		{"settlement", "结算确认", model.OpsLifecycleStepSettlement},
		{"invoice", "开票确认", model.OpsLifecycleStepInvoice},
		{"payment", "回款确认", model.OpsLifecycleStepPayment},
	}
	for _, c := range cases {
		got := normalizeLifecycleStep(&model.ProcessStep{ID: c.id, Name: c.name})
		if got != c.want {
			t.Fatalf("id=%s name=%s got=%s want=%s", c.id, c.name, got, c.want)
		}
	}
}
