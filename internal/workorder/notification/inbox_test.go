package notification

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type fakeInboxWriter struct {
	msgs []*model.WorkorderInboxMessage
}

func (f *fakeInboxWriter) Create(_ context.Context, msg *model.WorkorderInboxMessage) error {
	msg.ID = len(f.msgs) + 1
	copied := *msg
	f.msgs = append(f.msgs, &copied)
	return nil
}

func TestInboxChannelSend(t *testing.T) {
	writer := &fakeInboxWriter{}
	channel := NewInboxChannel(writer, zap.NewNop())
	instanceID := 21

	resp, err := channel.Send(context.Background(), &SendRequest{
		RecipientID: "12",
		Subject:     "工单已指派",
		Content:     "请处理",
		EventType:   model.EventTypeInstanceAssigned,
		InstanceID:  &instanceID,
	})
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("send failed: resp=%#v err=%v", resp, err)
	}
	if len(writer.msgs) != 1 || writer.msgs[0].UserID != 12 || writer.msgs[0].InstanceID == nil || *writer.msgs[0].InstanceID != 21 {
		t.Fatalf("unexpected message %#v", writer.msgs)
	}
}
