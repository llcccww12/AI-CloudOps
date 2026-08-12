/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 */

package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WorkorderNotificationReminderDAO interface {
	UpsertReminder(ctx context.Context, reminder *model.WorkorderNotificationReminder) error
	ListDuePending(ctx context.Context, now time.Time, limit int) ([]*model.WorkorderNotificationReminder, error)
	UpdateReminder(ctx context.Context, reminder *model.WorkorderNotificationReminder) error
	AcknowledgeByUser(ctx context.Context, instanceID, userID int) error
	StopByInstance(ctx context.Context, instanceID int) error
	GetReminderByID(ctx context.Context, id int) (*model.WorkorderNotificationReminder, error)
}

type notificationReminderDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewNotificationReminderDAO(db *gorm.DB, logger *zap.Logger) WorkorderNotificationReminderDAO {
	return &notificationReminderDAO{db: db, logger: logger}
}

func (d *notificationReminderDAO) UpsertReminder(ctx context.Context, reminder *model.WorkorderNotificationReminder) error {
	if reminder == nil {
		return fmt.Errorf("催发记录不能为空")
	}

	err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "notification_id"},
			{Name: "instance_id"},
			{Name: "user_id"},
			{Name: "event_type"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"status":           reminder.Status,
			"sent_count":       reminder.SentCount,
			"max_send":         reminder.MaxSend,
			"interval_minutes": reminder.IntervalMinutes,
			"last_sent_at":     reminder.LastSentAt,
			"next_send_at":     reminder.NextSendAt,
			"acknowledged_at":  reminder.AcknowledgedAt,
			"updated_at":       time.Now(),
		}),
	}).Create(reminder).Error
	if err != nil {
		d.logger.Error("upsert 催发记录失败", zap.Error(err))
		return fmt.Errorf("upsert 催发记录失败: %w", err)
	}
	return nil
}

func (d *notificationReminderDAO) ListDuePending(ctx context.Context, now time.Time, limit int) ([]*model.WorkorderNotificationReminder, error) {
	if limit <= 0 {
		limit = 100
	}
	var items []*model.WorkorderNotificationReminder
	err := d.db.WithContext(ctx).
		Where("status = ? AND next_send_at IS NOT NULL AND next_send_at <= ?", model.ReminderStatusPending, now).
		Order("next_send_at ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return nil, fmt.Errorf("查询到期催发失败: %w", err)
	}
	return items, nil
}

func (d *notificationReminderDAO) UpdateReminder(ctx context.Context, reminder *model.WorkorderNotificationReminder) error {
	if reminder == nil || reminder.ID <= 0 {
		return fmt.Errorf("催发记录无效")
	}
	result := d.db.WithContext(ctx).Model(&model.WorkorderNotificationReminder{}).
		Where("id = ?", reminder.ID).
		Updates(map[string]interface{}{
			"status":           reminder.Status,
			"sent_count":       reminder.SentCount,
			"max_send":         reminder.MaxSend,
			"interval_minutes": reminder.IntervalMinutes,
			"last_sent_at":     reminder.LastSentAt,
			"next_send_at":     reminder.NextSendAt,
			"acknowledged_at":  reminder.AcknowledgedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("更新催发记录失败: %w", result.Error)
	}
	return nil
}

func (d *notificationReminderDAO) AcknowledgeByUser(ctx context.Context, instanceID, userID int) error {
	if instanceID <= 0 || userID <= 0 {
		return nil
	}
	now := time.Now()
	result := d.db.WithContext(ctx).Model(&model.WorkorderNotificationReminder{}).
		Where("instance_id = ? AND user_id = ? AND status = ?", instanceID, userID, model.ReminderStatusPending).
		Updates(map[string]interface{}{
			"status":          model.ReminderStatusAcknowledged,
			"acknowledged_at": now,
			"next_send_at":    nil,
		})
	if result.Error != nil {
		return fmt.Errorf("标记催发已读失败: %w", result.Error)
	}
	return nil
}

func (d *notificationReminderDAO) StopByInstance(ctx context.Context, instanceID int) error {
	if instanceID <= 0 {
		return nil
	}
	result := d.db.WithContext(ctx).Model(&model.WorkorderNotificationReminder{}).
		Where("instance_id = ? AND status = ?", instanceID, model.ReminderStatusPending).
		Updates(map[string]interface{}{
			"status":       model.ReminderStatusStopped,
			"next_send_at": nil,
		})
	if result.Error != nil {
		return fmt.Errorf("停止工单催发失败: %w", result.Error)
	}
	return nil
}

func (d *notificationReminderDAO) GetReminderByID(ctx context.Context, id int) (*model.WorkorderNotificationReminder, error) {
	var item model.WorkorderNotificationReminder
	if err := d.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
