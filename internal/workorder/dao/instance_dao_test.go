package dao

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestWorkorderInstanceDAO(t *testing.T) WorkorderInstanceDAO {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}

	if err := db.AutoMigrate(&model.WorkorderInstance{}); err != nil {
		t.Fatalf("migrate workorder instance table: %v", err)
	}

	return NewWorkorderInstanceDAO(db, zap.NewNop())
}

func TestGenerateSerialNumberStartsAtOneWhenNoInstanceExistsToday(t *testing.T) {
	instanceDAO := newTestWorkorderInstanceDAO(t)
	prefix := "WO" + time.Now().Format("20060102")

	serialNumber, err := instanceDAO.GenerateSerialNumber(context.Background())
	if err != nil {
		t.Fatalf("GenerateSerialNumber returned error: %v", err)
	}

	if !strings.HasPrefix(serialNumber, prefix) {
		t.Fatalf("expected serial number prefix %q, got %q", prefix, serialNumber)
	}
	if serialNumber != prefix+"0001" {
		t.Fatalf("expected first serial number %q, got %q", prefix+"0001", serialNumber)
	}
}

func TestListInstanceScopes(t *testing.T) {
	d := newTestWorkorderInstanceDAO(t)
	ctx := context.Background()
	assignee := 7

	pending := &model.WorkorderInstance{
		Title: "待办单", SerialNumber: "WO-TODO-1", ProcessID: 1,
		Status: model.InstanceStatusProcessing, Priority: model.PriorityNormal,
		OperatorID: 3, OperatorName: "发起人", AssigneeID: &assignee,
	}
	otherTodo := &model.WorkorderInstance{
		Title: "别人的待办", SerialNumber: "WO-TODO-2", ProcessID: 1,
		Status: model.InstanceStatusPending, Priority: model.PriorityNormal,
		OperatorID: 3, OperatorName: "发起人",
	}
	done := &model.WorkorderInstance{
		Title: "已完成", SerialNumber: "WO-DONE-1", ProcessID: 1,
		Status: model.InstanceStatusCompleted, Priority: model.PriorityNormal,
		OperatorID: 7, OperatorName: "发起人", AssigneeID: &assignee,
	}
	for _, inst := range []*model.WorkorderInstance{pending, otherTodo, done} {
		if err := d.CreateInstance(ctx, inst); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	todo, total, err := d.ListInstance(ctx, &model.ListWorkorderInstanceReq{
		ListReq: model.ListReq{Page: 1, Size: 20},
		Scope:   model.InstanceListScopeTodo,
		UserID:  7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(todo) != 1 || todo[0].Title != "待办单" {
		t.Fatalf("todo got total=%d items=%v", total, todo)
	}

	all, total, err := d.ListInstance(ctx, &model.ListWorkorderInstanceReq{
		ListReq: model.ListReq{Page: 1, Size: 20},
		Scope:   model.InstanceListScopeAll,
		UserID:  7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("all unfinished want 2 got %d (%v)", total, all)
	}

	archived, total, err := d.ListInstance(ctx, &model.ListWorkorderInstanceReq{
		ListReq: model.ListReq{Page: 1, Size: 20},
		Scope:   model.InstanceListScopeArchive,
		UserID:  7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || archived[0].Title != "已完成" {
		t.Fatalf("archive got total=%d items=%v", total, archived)
	}
}
