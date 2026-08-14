package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkorderInboxDAO interface {
	Create(ctx context.Context, msg *model.WorkorderInboxMessage) error
	ListByUser(ctx context.Context, req *model.ListWorkorderInboxReq) (*model.ListResp[*model.WorkorderInboxMessage], error)
	CountUnread(ctx context.Context, userID int) (int64, error)
	MarkRead(ctx context.Context, id int, userID int) error
	MarkAllRead(ctx context.Context, userID int) error
	ClearByUser(ctx context.Context, userID int) error
}

type workorderInboxDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewWorkorderInboxDAO(db *gorm.DB, logger *zap.Logger) WorkorderInboxDAO {
	return &workorderInboxDAO{db: db, logger: logger}
}

func (d *workorderInboxDAO) Create(ctx context.Context, msg *model.WorkorderInboxMessage) error {
	if msg.IsRead == 0 {
		msg.IsRead = model.InboxMessageUnread
	}
	if err := d.db.WithContext(ctx).Create(msg).Error; err != nil {
		d.logger.Error("创建站内信失败", zap.Error(err), zap.Int("user_id", msg.UserID))
		return fmt.Errorf("创建站内信失败: %w", err)
	}
	return nil
}

func (d *workorderInboxDAO) ListByUser(ctx context.Context, req *model.ListWorkorderInboxReq) (*model.ListResp[*model.WorkorderInboxMessage], error) {
	page := req.Page
	size := req.Size
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	query := d.db.WithContext(ctx).Model(&model.WorkorderInboxMessage{}).Where("user_id = ?", req.UserID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计站内信失败: %w", err)
	}

	var items []*model.WorkorderInboxMessage
	if err := query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询站内信失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderInboxMessage]{Items: items, Total: total}, nil
}

func (d *workorderInboxDAO) CountUnread(ctx context.Context, userID int) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.WorkorderInboxMessage{}).
		Where("user_id = ? AND is_read = ?", userID, model.InboxMessageUnread).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("统计未读站内信失败: %w", err)
	}
	return count, nil
}

func (d *workorderInboxDAO) MarkRead(ctx context.Context, id int, userID int) error {
	now := time.Now()
	result := d.db.WithContext(ctx).Model(&model.WorkorderInboxMessage{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{
			"is_read": model.InboxMessageRead,
			"read_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("标记站内信已读失败: %w", result.Error)
	}
	return nil
}

func (d *workorderInboxDAO) MarkAllRead(ctx context.Context, userID int) error {
	now := time.Now()
	if err := d.db.WithContext(ctx).Model(&model.WorkorderInboxMessage{}).
		Where("user_id = ? AND is_read = ?", userID, model.InboxMessageUnread).
		Updates(map[string]any{
			"is_read": model.InboxMessageRead,
			"read_at": now,
		}).Error; err != nil {
		return fmt.Errorf("全部已读失败: %w", err)
	}
	return nil
}

func (d *workorderInboxDAO) ClearByUser(ctx context.Context, userID int) error {
	if err := d.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.WorkorderInboxMessage{}).Error; err != nil {
		return fmt.Errorf("清空站内信失败: %w", err)
	}
	return nil
}
