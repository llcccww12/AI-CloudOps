package dao

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestInboxDAO(t *testing.T) WorkorderInboxDAO {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:inboxdao?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkorderInboxMessage{}); err != nil {
		t.Fatalf("migrate inbox: %v", err)
	}
	return NewWorkorderInboxDAO(db, zap.NewNop())
}

func TestInboxCreateAndUnread(t *testing.T) {
	d := newTestInboxDAO(t)
	ctx := context.Background()
	instanceID := 12

	if err := d.Create(ctx, &model.WorkorderInboxMessage{
		UserID:     7,
		Title:      "工单已指派",
		Content:    "请处理工单 WO-1",
		EventType:  model.EventTypeInstanceAssigned,
		InstanceID: &instanceID,
	}); err != nil {
		t.Fatalf("create inbox: %v", err)
	}

	count, err := d.CountUnread(ctx, 7)
	if err != nil {
		t.Fatalf("count unread: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unread 1, got %d", count)
	}

	list, err := d.ListByUser(ctx, &model.ListWorkorderInboxReq{UserID: 7, ListReq: model.ListReq{Page: 1, Size: 20}})
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("unexpected list %#v", list)
	}
	if err := d.MarkRead(ctx, list.Items[0].ID, 7); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	count, err = d.CountUnread(ctx, 7)
	if err != nil {
		t.Fatalf("count unread after read: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected unread 0, got %d", count)
	}
}
