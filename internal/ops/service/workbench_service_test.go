package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestBuildReminderTemplate(t *testing.T) {
	tpl := buildReminderTemplate(&model.OpsReminderDraftReq{
		Type:         model.OpsRiskTrialExpiring,
		CustomerName: "测试客户",
		DueAt:        "2026-08-25",
	}, model.OpsReminderChannelInbox)
	if tpl.Subject == "" || tpl.Body == "" {
		t.Fatal("expected subject and body")
	}
	sms := buildReminderTemplate(&model.OpsReminderDraftReq{
		Type:         model.OpsRiskSettlementOverdue,
		CustomerName: "测试客户",
		Title:        "8月结算",
		DueAt:        "2026-08-01",
	}, model.OpsReminderChannelSMS)
	if len([]rune(sms.Body)) > 120 {
		t.Fatalf("sms body too long: %d", len([]rune(sms.Body)))
	}
}
