package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsReminderTaskDAO interface {
	List(ctx context.Context, req *model.ListOpsReminderTaskReq) ([]*model.OpsReminderTask, int64, error)
	GetByID(ctx context.Context, id int) (*model.OpsReminderTask, error)
	Create(ctx context.Context, task *model.OpsReminderTask) error
	Update(ctx context.Context, task *model.OpsReminderTask) error
	Delete(ctx context.Context, id int) error
	ListDue(ctx context.Context, now time.Time) ([]*model.OpsReminderTask, error)
	MarkSent(ctx context.Context, id int, at time.Time) error
}

type opsReminderTaskDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsReminderTaskDAO(db *gorm.DB, logger *zap.Logger) OpsReminderTaskDAO {
	return &opsReminderTaskDAO{db: db, logger: logger}
}

func (d *opsReminderTaskDAO) List(ctx context.Context, req *model.ListOpsReminderTaskReq) ([]*model.OpsReminderTask, int64, error) {
	q := d.db.WithContext(ctx).Model(&model.OpsReminderTask{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.TargetUserID > 0 {
		q = q.Where("target_user_id = ?", req.TargetUserID)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		q = q.Where("title LIKE ?", "%"+req.Search+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := req.Page, req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	var items []*model.OpsReminderTask
	err := q.Order("due_at ASC, id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (d *opsReminderTaskDAO) GetByID(ctx context.Context, id int) (*model.OpsReminderTask, error) {
	var task model.OpsReminderTask
	if err := d.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (d *opsReminderTaskDAO) Create(ctx context.Context, task *model.OpsReminderTask) error {
	return d.db.WithContext(ctx).Create(task).Error
}

func (d *opsReminderTaskDAO) Update(ctx context.Context, task *model.OpsReminderTask) error {
	return d.db.WithContext(ctx).Model(&model.OpsReminderTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
		"title": task.Title, "due_at": task.DueAt, "advance_days": task.AdvanceDays,
		"channels": task.Channels, "target_user_id": task.TargetUserID,
		"extra_user_ids": task.ExtraUserIDs, "content": task.Content, "status": task.Status,
	}).Error
}

func (d *opsReminderTaskDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsReminderTask{}, id).Error
}

func (d *opsReminderTaskDAO) ListDue(ctx context.Context, now time.Time) ([]*model.OpsReminderTask, error) {
	var items []*model.OpsReminderTask
	err := d.db.WithContext(ctx).
		Where("status = ?", model.OpsReminderTaskPending).
		Where("due_at <= ?", now.AddDate(0, 0, 365)).
		Order("due_at ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	var due []*model.OpsReminderTask
	for _, item := range items {
		triggerAt := item.DueAt.AddDate(0, 0, -item.AdvanceDays)
		if !triggerAt.After(now) {
			due = append(due, item)
		}
	}
	return due, nil
}

func (d *opsReminderTaskDAO) MarkSent(ctx context.Context, id int, at time.Time) error {
	return d.db.WithContext(ctx).Model(&model.OpsReminderTask{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": model.OpsReminderTaskSent, "sent_at": at}).Error
}

type OpsReminderDeliveryDAO interface {
	Create(ctx context.Context, item *model.OpsReminderDelivery) error
	ExistsByDedupeKey(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, req *model.ListOpsReminderDeliveryReq) ([]*model.OpsReminderDelivery, int64, error)
}

type opsReminderDeliveryDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsReminderDeliveryDAO(db *gorm.DB, logger *zap.Logger) OpsReminderDeliveryDAO {
	return &opsReminderDeliveryDAO{db: db, logger: logger}
}

func (d *opsReminderDeliveryDAO) Create(ctx context.Context, item *model.OpsReminderDelivery) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *opsReminderDeliveryDAO) ExistsByDedupeKey(ctx context.Context, key string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.OpsReminderDelivery{}).Where("dedupe_key = ?", key).Count(&count).Error
	return count > 0, err
}

func (d *opsReminderDeliveryDAO) List(ctx context.Context, req *model.ListOpsReminderDeliveryReq) ([]*model.OpsReminderDelivery, int64, error) {
	q := d.db.WithContext(ctx).Model(&model.OpsReminderDelivery{})
	if req.Channel != "" {
		q = q.Where("channel = ?", req.Channel)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Scene != "" {
		q = q.Where("scene = ?", req.Scene)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("title LIKE ? OR content LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := req.Page, req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	var items []*model.OpsReminderDelivery
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func BuildReminderDedupeKey(sourceType string, sourceID, userID int, channel, day string, bizType string, bizID int) string {
	return fmt.Sprintf("%s:%d:u%d:%s:%s:%s:%d", sourceType, sourceID, userID, channel, day, bizType, bizID)
}
