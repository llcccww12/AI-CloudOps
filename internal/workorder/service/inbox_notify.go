package service

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
)

func withInboxChannel(channels []string) []string {
	out := make([]string, 0, len(channels)+1)
	seen := make(map[string]struct{}, len(channels)+1)
	for _, channel := range channels {
		if channel == "" {
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		out = append(out, channel)
	}
	if _, ok := seen[model.NotificationChannelInbox]; !ok {
		out = append([]string{model.NotificationChannelInbox}, out...)
	}
	return out
}

func uniqueRecipients(recipients []RecipientInfo) []RecipientInfo {
	out := make([]RecipientInfo, 0, len(recipients))
	seen := make(map[string]struct{}, len(recipients))
	for _, rec := range recipients {
		if rec.ID == "" || rec.ID == "0" {
			continue
		}
		if _, ok := seen[rec.ID]; ok {
			continue
		}
		seen[rec.ID] = struct{}{}
		out = append(out, rec)
	}
	return out
}

func mergeWorkorderInvolvedRecipients(recipients []RecipientInfo, instance *model.WorkorderInstance, senderID int) []RecipientInfo {
	if instance == nil {
		return uniqueRecipients(recipients)
	}

	merged := append([]RecipientInfo(nil), recipients...)
	if instance.OperatorID > 0 {
		merged = append(merged, RecipientInfo{
			ID:   fmt.Sprintf("%d", instance.OperatorID),
			Name: instance.OperatorName,
			Type: model.RecipientTypeCreator,
		})
	}
	if instance.AssigneeID != nil && *instance.AssigneeID > 0 {
		merged = append(merged, RecipientInfo{
			ID:   fmt.Sprintf("%d", *instance.AssigneeID),
			Name: "处理人",
			Type: model.RecipientTypeAssignee,
		})
	}

	result := uniqueRecipients(merged)
	if senderID <= 0 {
		return result
	}
	senderKey := fmt.Sprintf("%d", senderID)
	filtered := make([]RecipientInfo, 0, len(result))
	for _, rec := range result {
		if rec.ID == senderKey {
			continue
		}
		filtered = append(filtered, rec)
	}
	return filtered
}

func (s *workorderNotificationService) sendInvolvedInbox(ctx context.Context, instance *model.WorkorderInstance, eventType string, senderID int, customContent ...string) error {
	if s.notificationMgr == nil {
		return nil
	}

	recipients := mergeWorkorderInvolvedRecipients(nil, instance, senderID)
	if len(recipients) == 0 {
		return nil
	}

	cfg := &model.WorkorderNotification{
		Name:            "工单站内信",
		Channels:        model.StringList{model.NotificationChannelInbox},
		RecipientTypes:  model.StringList{model.RecipientTypeCreator, model.RecipientTypeAssignee},
		Priority:        model.NotificationPriorityMedium,
		SubjectTemplate: fmt.Sprintf("【CacOps】%s - %s", notification.GetEventTypeText(eventType), instance.Title),
		MessageTemplate: "您好 {recipient_name}！\n\n工单通知：{title}\n工单编号：{serial_number}\n优先级：{priority_text}\n状态：{status}\n\n详情请查看系统。\n\n通知时间：{notification_time}\n平台：{platform_name}",
	}
	return s.sendChannelNotification(ctx, cfg, instance, model.NotificationChannelInbox, recipients, eventType, senderID, customContent...)
}
