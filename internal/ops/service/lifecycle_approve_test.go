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
		{"trial_contract", "试用合同签约", model.OpsLifecycleStepTrialContract},
		{"open_request", "开通申请", model.OpsLifecycleStepOpenRequest},
		{"open_feedback", "开通回馈", model.OpsLifecycleStepOpenFeedback},
		{"trial_accept", "试用验收/转正评估", model.OpsLifecycleStepTrialAccept},
		{"x", "验收评估", model.OpsLifecycleStepTrialAccept},
		{"provision", "资源开通", model.OpsLifecycleStepProvision},
		{"provision", "开通台账", model.OpsLifecycleStepProvision},
		{"x", "资源交付", model.OpsLifecycleStepProvision},
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

func TestIsPlaceholderTrialTitle(t *testing.T) {
	cases := []struct {
		title string
		want  bool
	}{
		{"", true},
		{"测试开通", true},
		{"林德叉车（中国）有限公司-试用", true},
		{"运营测试开通-林德-20260904", true},
		{"20250606-20250625-测试-H100-8", false},
	}
	for _, c := range cases {
		if got := isPlaceholderTrialTitle(c.title); got != c.want {
			t.Fatalf("title=%q got=%v want=%v", c.title, got, c.want)
		}
	}
}

func TestPickBestTrialPrefersProjectNamed(t *testing.T) {
	hollow := &model.OpsTrial{
		Model:  model.Model{ID: 9},
		Title:  "林德叉车（中国）有限公司-试用",
		Status: model.OpsTrialStatusApproved,
	}
	solid := &model.OpsTrial{
		Model:       model.Model{ID: 8},
		Title:       "20250605-20250626-H00-测试-8卡",
		ProjectName: "20250605-20250626-H00-测试-8卡",
		Status:      model.OpsTrialStatusApproved,
		OrderNo:     "ORD-1",
	}
	got := pickBestTrial([]*model.OpsTrial{hollow, solid})
	if got == nil || got.ID != solid.ID {
		t.Fatalf("want solid trial id=%d, got %#v", solid.ID, got)
	}
}

func TestResolveComputeBizPhase(t *testing.T) {
	cases := []struct {
		name    string
		refs    computeAllocBizRefs
		updating bool
		existing string
		want    string
	}{
		{"provision_new", computeAllocBizRefs{Source: "open", TrialID: 1}, false, "", model.OpsComputePhaseTrial},
		{"trial_to_formal", computeAllocBizRefs{Source: "contract", ActivationID: 2}, true, model.OpsComputePhaseTrial, model.OpsComputePhaseFormal},
		{"formal_to_renew", computeAllocBizRefs{Source: "contract", ActivationID: 2}, true, model.OpsComputePhaseFormal, model.OpsComputePhaseRenew},
		{"renew_again", computeAllocBizRefs{Source: "open", ActivationID: 3}, true, model.OpsComputePhaseRenew, model.OpsComputePhaseRenew},
		{"explicit", computeAllocBizRefs{BizPhase: model.OpsComputePhaseRenew}, false, "", model.OpsComputePhaseRenew},
	}
	for _, c := range cases {
		got := resolveComputeBizPhase(c.refs, c.updating, c.existing)
		if got != c.want {
			t.Fatalf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}
