/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 */

package model

import "time"

const (
	ReminderStatusPending      int8 = 1 // 待催发
	ReminderStatusAcknowledged int8 = 2 // 已读停发
	ReminderStatusStopped      int8 = 3 // 已停止（工单结束/达上限/配置禁用）
)

// WorkorderNotificationReminder 未读催发状态
type WorkorderNotificationReminder struct {
	Model
	NotificationID  int        `json:"notification_id" gorm:"column:notification_id;not null;uniqueIndex:uk_reminder_target;index;comment:通知配置ID"`
	InstanceID      int        `json:"instance_id" gorm:"column:instance_id;not null;uniqueIndex:uk_reminder_target;index;comment:工单实例ID"`
	UserID          int        `json:"user_id" gorm:"column:user_id;not null;uniqueIndex:uk_reminder_target;index;comment:接收人用户ID"`
	EventType       string     `json:"event_type" gorm:"column:event_type;type:varchar(50);not null;uniqueIndex:uk_reminder_target;comment:触发事件类型"`
	Status          int8       `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-待催发，2-已读，3-已停止"`
	SentCount       int        `json:"sent_count" gorm:"column:sent_count;not null;default:0;comment:已发送次数（含首次）"`
	MaxSend         int        `json:"max_send" gorm:"column:max_send;not null;default:3;comment:最大发送次数"`
	IntervalMinutes int        `json:"interval_minutes" gorm:"column:interval_minutes;not null;default:0;comment:催发间隔(分钟)"`
	LastSentAt      *time.Time `json:"last_sent_at" gorm:"column:last_sent_at;comment:上次发送时间"`
	NextSendAt      *time.Time `json:"next_send_at" gorm:"column:next_send_at;index;comment:下次催发时间"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at" gorm:"column:acknowledged_at;comment:已读时间"`
}

func (WorkorderNotificationReminder) TableName() string {
	return "cl_workorder_notification_reminder"
}
