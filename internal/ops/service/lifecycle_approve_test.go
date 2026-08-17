package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestNormalizeLifecycleStep(t *testing.T) {
	cases := []struct {
		id, name, want string
	}{
		{"intent", "意向确认", model.OpsLifecycleStepIntent},
		{"trial", "试用审批", model.OpsLifecycleStepTrial},
		{"trial_accept", "试用验收/转正评估", model.OpsLifecycleStepTrialAccept},
		{"x", "验收评估", model.OpsLifecycleStepTrialAccept},
		{"contract", "合同与开通", model.OpsLifecycleStepContract},
		{"settlement", "结算确认", model.OpsLifecycleStepSettlement},
		{"payment", "开票与回款确认", model.OpsLifecycleStepPayment},
	}
	for _, c := range cases {
		got := normalizeLifecycleStep(&model.ProcessStep{ID: c.id, Name: c.name})
		if got != c.want {
			t.Fatalf("id=%s name=%s got %s want %s", c.id, c.name, got, c.want)
		}
	}
}
