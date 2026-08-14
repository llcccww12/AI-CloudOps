package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestWithInboxChannel(t *testing.T) {
	got := withInboxChannel([]string{"email", "feishu"})
	if len(got) != 3 || got[0] != model.NotificationChannelInbox {
		t.Fatalf("expected inbox first, got %#v", got)
	}

	got = withInboxChannel([]string{"inbox", "email", "inbox"})
	if len(got) != 2 {
		t.Fatalf("expected deduped channels, got %#v", got)
	}
}

func TestMergeWorkorderInvolvedRecipients(t *testing.T) {
	assigneeID := 8
	instance := &model.WorkorderInstance{
		OperatorID:   3,
		OperatorName: "创建人",
		AssigneeID:   &assigneeID,
	}

	got := mergeWorkorderInvolvedRecipients([]RecipientInfo{
		{ID: "9", Name: "指定用户", Type: model.RecipientTypeUser},
	}, instance, 3)

	if len(got) != 2 {
		t.Fatalf("expected creator skipped as sender, got %#v", got)
	}
	if got[0].ID != "9" || got[1].ID != "8" {
		t.Fatalf("unexpected recipients %#v", got)
	}
}
