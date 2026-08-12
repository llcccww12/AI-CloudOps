/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkorderNotificationService interface {
	CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error
	UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error
	DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error
	ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error)
	DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error)
	GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error)
	TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error
	SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error
	SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error
	GetAvailableChannels() *model.ListResp[*model.WorkorderNotificationChannel]
	ProcessDueReminders(ctx context.Context) error
	AcknowledgeByUser(ctx context.Context, instanceID, userID int) error
	StopRemindersByInstance(ctx context.Context, instanceID int) error
}

type workorderNotificationService struct {
	dao             workorderDao.WorkorderNotificationDAO
	reminderDAO     workorderDao.WorkorderNotificationReminderDAO
	logger          *zap.Logger
	notificationMgr *notification.Manager
	instanceDAO     workorderDao.WorkorderInstanceDAO
	userDAO         userDao.UserDAO
}

func NewWorkorderNotificationService(
	dao workorderDao.WorkorderNotificationDAO,
	reminderDAO workorderDao.WorkorderNotificationReminderDAO,
	notificationMgr *notification.Manager,
	logger *zap.Logger,
	instanceDAO workorderDao.WorkorderInstanceDAO,
	userDAO userDao.UserDAO,
) WorkorderNotificationService {
	return &workorderNotificationService{
		logger:          logger,
		dao:             dao,
		reminderDAO:     reminderDAO,
		notificationMgr: notificationMgr,
		instanceDAO:     instanceDAO,
		userDAO:         userDAO,
	}
}

func (s *workorderNotificationService) CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error {
	return s.dao.CreateNotification(ctx, req)
}

func (s *workorderNotificationService) UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error {
	_, err := s.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	return s.dao.UpdateNotification(ctx, req)
}

func (s *workorderNotificationService) DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error {
	_, err := s.dao.GetNotificationByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	return s.dao.DeleteNotification(ctx, req)
}

func (s *workorderNotificationService) ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (*model.ListResp[*model.WorkorderNotification], error) {
	result, err := s.dao.ListNotification(ctx, req)
	if err != nil {
		s.logger.Error("获取通知配置列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取通知配置列表失败: %w", err)
	}
	return result, nil
}

func (s *workorderNotificationService) DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error) {
	return s.dao.DetailNotification(ctx, req)
}

func (s *workorderNotificationService) GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (*model.ListResp[*model.WorkorderNotificationLog], error) {
	result, err := s.dao.GetSendLogs(ctx, req)
	if err != nil {
		s.logger.Error("获取发送日志失败", zap.Error(err))
		return nil, fmt.Errorf("获取发送日志失败: %w", err)
	}
	return result, nil
}

func (s *workorderNotificationService) TestSendNotification(ctx context.Context, req *model.TestSendWorkorderNotificationReq) error {
	notificationConfig, err := s.dao.GetNotificationByID(ctx, req.NotificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("通知配置不存在")
		}
		return fmt.Errorf("查询通知配置失败: %w", err)
	}

	if notificationConfig.Status != 1 {
		return errors.New("通知配置已禁用，无法发送")
	}

	var senderID int
	if uid := ctx.Value("user_id"); uid != nil {
		if id, ok := uid.(int); ok {
			senderID = id
		}
	}

	for _, channel := range notificationConfig.Channels {
		recipientAddrs := s.resolveTestRecipients(ctx, notificationConfig, channel)
		if len(recipientAddrs) == 0 {
			if req.Recipient != "" {
				recipientAddrs = []testRecipient{{ID: "manual", Name: "手动指定", Addr: req.Recipient}}
			} else {
				return fmt.Errorf("没有可发送的接收人：请在通知配置中选择有邮箱的自定义用户，或给创建人账号补全邮箱")
			}
		}

		for _, recipient := range recipientAddrs {
			// 创建模拟的实例ID用于测试
			testInstanceID := 999999

			sendRequest := &notification.SendRequest{
				Subject:       notificationConfig.SubjectTemplate,
				Content:       notificationConfig.MessageTemplate,
				Priority:      notificationConfig.Priority,
				RecipientType: channel,
				RecipientID:   recipient.ID,
				RecipientAddr: recipient.Addr,
				RecipientName: recipient.Name,
				EventType:     "test",
				InstanceID:    &testInstanceID,
				Templates:     make(map[string]string),
				Metadata: map[string]interface{}{
					"notification_id": notificationConfig.ID,
					"sender_id":       senderID,
				},
			}

			sendRequest.Templates["workorder_id"] = fmt.Sprintf("%d", testInstanceID)
			sendRequest.Templates["serial_number"] = fmt.Sprintf("WO-%d", testInstanceID)
			sendRequest.Templates["title"] = "系统测试工单 - 通知功能验证"
			sendRequest.Templates["description"] = "系统测试工单，验证工单通知功能是否正常工作。"
			sendRequest.Templates["operator_name"] = "系统管理员"
			sendRequest.Templates["assignee_name"] = "运维工程师"
			sendRequest.Templates["priority_level"] = fmt.Sprintf("%d", int(notificationConfig.Priority))
			sendRequest.Templates["priority_text"] = notification.FormatPriority(notificationConfig.Priority)
			sendRequest.Templates["status"] = "测试进行中"
			sendRequest.Templates["created_time"] = time.Now().Format("2006-01-02 15:04:05")
			sendRequest.Templates["updated_time"] = time.Now().Format("2006-01-02 15:04:05")
			sendRequest.Templates["event_type"] = notification.GetEventTypeText("test")
			sendRequest.Templates["notification_time"] = time.Now().Format("2006-01-02 15:04:05")
			sendRequest.Templates["company_name"] = "AI-CloudOps"
			sendRequest.Templates["platform_name"] = "运维管理平台"
			sendRequest.Templates["department"] = "技术运维部"
			sendRequest.Templates["test_content"] = "本次测试验证了系统通知功能的完整性，包括邮件发送、飞书消息推送等多个渠道的有效性。"
			response, err := s.notificationMgr.SendNotification(ctx, sendRequest)

			log := &model.WorkorderNotificationLog{
				NotificationID: notificationConfig.ID,
				EventType:      "test",
				Channel:        channel,
				RecipientType:  "test",
				RecipientID:    recipient.ID,
				RecipientName:  recipient.Name,
				RecipientAddr:  recipient.Addr,
				Subject:        notificationConfig.SubjectTemplate,
				Content:        notificationConfig.MessageTemplate,
				Status:         2,
				SendAt:         time.Now(),
				SenderID:       senderID,
			}

			if err != nil {
				log.Status = 4
				log.ErrorMessage = err.Error()
			} else if response != nil {
				log.Status = 3
				if response.ExternalID != "" {
					log.ResponseData = map[string]interface{}{
						"external_id": response.ExternalID,
					}
				}
				if response.Cost != nil {
					log.Cost = response.Cost
				}
			}

			if err := s.dao.AddSendLog(ctx, log); err != nil {
				s.logger.Error("记录发送日志失败", zap.Error(err))
			}
		}
	}

	return s.dao.IncrementSentCount(ctx, notificationConfig.ID)
}

type testRecipient struct {
	ID   string
	Name string
	Addr string
}

func (s *workorderNotificationService) resolveTestRecipients(
	ctx context.Context,
	notificationConfig *model.WorkorderNotification,
	channel string,
) []testRecipient {
	seen := make(map[string]struct{})
	var result []testRecipient

	addUser := func(userID int) {
		if userID <= 0 {
			return
		}
		key := fmt.Sprintf("%d", userID)
		if _, ok := seen[key]; ok {
			return
		}
		user, err := s.userDAO.GetByID(ctx, userID)
		if err != nil || user == nil {
			return
		}
		addr := ""
		switch channel {
		case model.NotificationChannelEmail:
			addr = user.Email
		case model.NotificationChannelFeishu:
			addr = user.FeiShuUserId
		case model.NotificationChannelSMS:
			addr = user.Mobile
		}
		if addr == "" {
			s.logger.Warn("测试发送跳过无渠道地址的用户",
				zap.Int("user_id", userID),
				zap.String("channel", channel))
			return
		}
		seen[key] = struct{}{}
		name := user.RealName
		if name == "" {
			name = user.Username
		}
		result = append(result, testRecipient{ID: key, Name: name, Addr: addr})
	}

	for _, recipientType := range notificationConfig.RecipientTypes {
		switch recipientType {
		case model.RecipientTypeUser, model.RecipientTypeCustom:
			for _, userIDStr := range notificationConfig.RecipientUsers {
				userID, err := strconv.Atoi(userIDStr)
				if err != nil {
					continue
				}
				addUser(userID)
			}
		}
	}

	// 即便未勾选 user/custom，只要配置了 recipient_users 也用于测试发送
	if len(result) == 0 {
		for _, userIDStr := range notificationConfig.RecipientUsers {
			userID, err := strconv.Atoi(userIDStr)
			if err != nil {
				continue
			}
			addUser(userID)
		}
	}

	return result
}

// SendWorkorderNotification 发送工单相关通知
func (s *workorderNotificationService) SendWorkorderNotification(ctx context.Context, instanceID int, eventType string, customContent ...string) error {
	instance, err := s.instanceDAO.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单实例失败",
			zap.Int("instance_id", instanceID),
			zap.Error(err))
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	var senderID int
	if uid := ctx.Value("user_id"); uid != nil {
		if id, ok := uid.(int); ok {
			senderID = id
		}
	}

	notifications, err := s.dao.GetActiveNotificationsByEventType(ctx, eventType, instance.ProcessID)
	if err != nil {
		s.logger.Error("获取通知配置失败",
			zap.String("event_type", eventType),
			zap.Int("process_id", instance.ProcessID),
			zap.Error(err))
		return fmt.Errorf("获取通知配置失败: %w", err)
	}

	if len(notifications) == 0 {
		s.logger.Info("没有找到匹配的通知配置",
			zap.String("event_type", eventType),
			zap.Int("process_id", instance.ProcessID))
		return nil
	}

	for _, notification := range notifications {
		if err := s.processNotification(ctx, notification, instance, eventType, senderID, customContent...); err != nil {
			s.logger.Error("处理通知配置失败",
				zap.Int("notification_id", notification.ID),
				zap.Int("instance_id", instanceID),
				zap.Error(err))
			continue
		}
	}

	s.logger.Info("工单通知发送完成",
		zap.Int("instance_id", instanceID),
		zap.String("event_type", eventType),
		zap.Int("notification_count", len(notifications)))

	return nil
}

func (s *workorderNotificationService) processNotification(ctx context.Context, notification *model.WorkorderNotification,
	instance *model.WorkorderInstance, eventType string, senderID int, customContent ...string) error {

	recipients, err := s.getRecipients(ctx, notification, instance)
	if err != nil {
		return fmt.Errorf("获取接收人失败: %w", err)
	}

	if len(recipients) == 0 {
		s.logger.Info("没有找到接收人",
			zap.Int("notification_id", notification.ID),
			zap.Int("instance_id", instance.ID))
		return nil
	}

	var wg sync.WaitGroup
	channelErrors := make(chan error, len(notification.Channels))

	for _, channel := range notification.Channels {
		wg.Add(1)
		go func(ch string) {
			defer wg.Done()

			channelCtx := context.Background()
			if deadline, ok := ctx.Deadline(); ok {
				var cancel context.CancelFunc
				channelCtx, cancel = context.WithDeadline(context.Background(), deadline)
				defer cancel()
			}

			if err := s.sendChannelNotification(channelCtx, notification, instance, ch, recipients, eventType, senderID, customContent...); err != nil {
				s.logger.Error("发送渠道通知失败",
					zap.String("channel", ch),
					zap.Int("notification_id", notification.ID),
					zap.Error(err))
				channelErrors <- fmt.Errorf("渠道 %s 发送失败: %w", ch, err)
			} else {
				s.logger.Info("渠道通知发送成功",
					zap.String("channel", ch),
					zap.Int("notification_id", notification.ID))
			}
		}(channel)
	}

	wg.Wait()
	close(channelErrors)

	var errors []string
	for err := range channelErrors {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		s.logger.Warn("部分渠道发送失败，但其他渠道已成功发送",
			zap.Strings("errors", errors),
			zap.Int("notification_id", notification.ID))
	}

	// 首次发送后建立未读催发（repeat_interval>0）
	s.createRemindersAfterSend(ctx, notification, instance, eventType, recipients)

	return nil
}

func (s *workorderNotificationService) createRemindersAfterSend(
	ctx context.Context,
	notification *model.WorkorderNotification,
	instance *model.WorkorderInstance,
	eventType string,
	recipients []RecipientInfo,
) {
	if s.reminderDAO == nil || notification == nil || instance == nil {
		return
	}
	if notification.RepeatInterval == nil || *notification.RepeatInterval <= 0 {
		return
	}
	if isInstanceTerminal(instance.Status) {
		return
	}

	interval := *notification.RepeatInterval
	maxSend := notification.MaxRetries
	if maxSend <= 0 {
		maxSend = 3
	}
	now := time.Now()
	next := now.Add(time.Duration(interval) * time.Minute)

	for _, rec := range recipients {
		userID, err := strconv.Atoi(rec.ID)
		if err != nil || userID <= 0 {
			continue
		}
		reminder := &model.WorkorderNotificationReminder{
			NotificationID:  notification.ID,
			InstanceID:      instance.ID,
			UserID:          userID,
			EventType:       eventType,
			Status:          model.ReminderStatusPending,
			SentCount:       1,
			MaxSend:         maxSend,
			IntervalMinutes: interval,
			LastSentAt:      &now,
			NextSendAt:      &next,
		}
		if err := s.reminderDAO.UpsertReminder(ctx, reminder); err != nil {
			s.logger.Error("创建催发记录失败",
				zap.Error(err),
				zap.Int("notification_id", notification.ID),
				zap.Int("instance_id", instance.ID),
				zap.Int("user_id", userID))
		}
	}
}

func isInstanceTerminal(status int8) bool {
	return status == model.InstanceStatusCompleted ||
		status == model.InstanceStatusCancelled ||
		status == model.InstanceStatusRejected
}

func (s *workorderNotificationService) AcknowledgeByUser(ctx context.Context, instanceID, userID int) error {
	if s.reminderDAO == nil {
		return nil
	}
	return s.reminderDAO.AcknowledgeByUser(ctx, instanceID, userID)
}

func (s *workorderNotificationService) StopRemindersByInstance(ctx context.Context, instanceID int) error {
	if s.reminderDAO == nil {
		return nil
	}
	return s.reminderDAO.StopByInstance(ctx, instanceID)
}

func (s *workorderNotificationService) ProcessDueReminders(ctx context.Context) error {
	if s.reminderDAO == nil {
		return nil
	}

	dueList, err := s.reminderDAO.ListDuePending(ctx, time.Now(), 100)
	if err != nil {
		return err
	}
	if len(dueList) == 0 {
		return nil
	}

	s.logger.Info("开始处理到期催发", zap.Int("count", len(dueList)))

	for _, reminder := range dueList {
		if err := s.processOneReminder(ctx, reminder); err != nil {
			s.logger.Error("处理催发失败",
				zap.Error(err),
				zap.Int("reminder_id", reminder.ID),
				zap.Int("instance_id", reminder.InstanceID))
		}
	}
	return nil
}

func (s *workorderNotificationService) processOneReminder(ctx context.Context, reminder *model.WorkorderNotificationReminder) error {
	notification, err := s.dao.GetNotificationByID(ctx, reminder.NotificationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reminder.Status = model.ReminderStatusStopped
			reminder.NextSendAt = nil
			_ = s.reminderDAO.UpdateReminder(ctx, reminder)
			return nil
		}
		return err
	}
	if notification.Status != model.NotificationStatusEnabled {
		reminder.Status = model.ReminderStatusStopped
		reminder.NextSendAt = nil
		return s.reminderDAO.UpdateReminder(ctx, reminder)
	}

	instance, err := s.instanceDAO.GetInstanceByID(ctx, reminder.InstanceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reminder.Status = model.ReminderStatusStopped
			reminder.NextSendAt = nil
			_ = s.reminderDAO.UpdateReminder(ctx, reminder)
			return nil
		}
		return err
	}
	if isInstanceTerminal(instance.Status) {
		reminder.Status = model.ReminderStatusStopped
		reminder.NextSendAt = nil
		return s.reminderDAO.UpdateReminder(ctx, reminder)
	}

	if reminder.SentCount >= reminder.MaxSend {
		reminder.Status = model.ReminderStatusStopped
		reminder.NextSendAt = nil
		return s.reminderDAO.UpdateReminder(ctx, reminder)
	}

	user, err := s.userDAO.GetByID(ctx, reminder.UserID)
	if err != nil || user == nil {
		reminder.Status = model.ReminderStatusStopped
		reminder.NextSendAt = nil
		_ = s.reminderDAO.UpdateReminder(ctx, reminder)
		return fmt.Errorf("催发接收人无效: %w", err)
	}

	recipients := []RecipientInfo{{
		ID:   fmt.Sprintf("%d", user.ID),
		Name: user.RealName,
		Type: model.RecipientTypeUser,
	}}

	customContent := fmt.Sprintf("【未读催发 %d/%d】请及时打开工单查看", reminder.SentCount+1, reminder.MaxSend)
	var sendErrs []string
	for _, channel := range notification.Channels {
		if err := s.sendChannelNotification(ctx, notification, instance, channel, recipients, reminder.EventType, 0, customContent); err != nil {
			sendErrs = append(sendErrs, err.Error())
		}
	}

	now := time.Now()
	reminder.SentCount++
	reminder.LastSentAt = &now
	if reminder.SentCount >= reminder.MaxSend {
		reminder.Status = model.ReminderStatusStopped
		reminder.NextSendAt = nil
	} else {
		next := now.Add(time.Duration(reminder.IntervalMinutes) * time.Minute)
		reminder.NextSendAt = &next
		reminder.Status = model.ReminderStatusPending
	}

	if err := s.reminderDAO.UpdateReminder(ctx, reminder); err != nil {
		return err
	}
	if len(sendErrs) > 0 {
		return fmt.Errorf("部分渠道催发失败: %s", sendErrs[0])
	}
	return nil
}

func (s *workorderNotificationService) getRecipients(ctx context.Context, notification *model.WorkorderNotification,
	instance *model.WorkorderInstance) ([]RecipientInfo, error) {

	var recipients []RecipientInfo

	for _, recipientType := range notification.RecipientTypes {
		switch recipientType {
		case model.RecipientTypeCreator:
			recipients = append(recipients, RecipientInfo{
				ID:   fmt.Sprintf("%d", instance.OperatorID),
				Name: instance.OperatorName,
				Type: recipientType,
			})
		case model.RecipientTypeAssignee:
			if instance.AssigneeID != nil {
				assigneeName := "处理人"
				if user, err := s.userDAO.GetByID(ctx, *instance.AssigneeID); err == nil {
					assigneeName = user.RealName
				}

				recipients = append(recipients, RecipientInfo{
					ID:   fmt.Sprintf("%d", *instance.AssigneeID),
					Name: assigneeName,
					Type: recipientType,
				})
			}
		case model.RecipientTypeUser:
			for _, userIDStr := range notification.RecipientUsers {
				userID, err := strconv.Atoi(userIDStr)
				if err != nil {
					s.logger.Warn("无效的用户ID",
						zap.String("user_id", userIDStr))
					continue
				}

				userName := "指定用户"
				if user, err := s.userDAO.GetByID(ctx, userID); err == nil {
					userName = user.RealName
				}

				recipients = append(recipients, RecipientInfo{
					ID:   userIDStr,
					Name: userName,
					Type: recipientType,
				})
			}
		case model.RecipientTypeRole:
			s.logger.Info("角色用户通知暂未实现",
				zap.Strings("roles", notification.RecipientRoles))
		case model.RecipientTypeDept:
			s.logger.Info("部门用户通知暂未实现",
				zap.Strings("depts", notification.RecipientDepts))
		case model.RecipientTypeCustom:
			// 与指定用户相同：使用 recipient_users
			for _, userIDStr := range notification.RecipientUsers {
				if userID, err := strconv.Atoi(userIDStr); err == nil {
					if user, err := s.userDAO.GetByID(ctx, userID); err == nil {
						recipients = append(recipients, RecipientInfo{
							ID:   userIDStr,
							Name: user.RealName,
							Type: recipientType,
						})
					}
				}
			}
		}
	}

	return recipients, nil
}

// sendChannelNotification 通过指定渠道发送通知
func (s *workorderNotificationService) sendChannelNotification(ctx context.Context, notificationConfig *model.WorkorderNotification,
	instance *model.WorkorderInstance, channel string, recipients []RecipientInfo, eventType string, senderID int, customContent ...string) error {

	subject, content := s.buildMessageContent(notificationConfig, instance, eventType, customContent...)

	var wg sync.WaitGroup
	recipientErrors := make(chan error, len(recipients))

	for _, recipient := range recipients {
		wg.Add(1)
		go func(rec RecipientInfo) {
			defer wg.Done()

			recipientCtx := context.Background()
			if deadline, ok := ctx.Deadline(); ok {
				var cancel context.CancelFunc
				recipientCtx, cancel = context.WithDeadline(context.Background(), deadline)
				defer cancel()
			}

			recipientAddr := s.getRecipientAddress(rec, channel)
			if recipientAddr == "" {
				s.logger.Warn("无法获取接收人地址",
					zap.String("recipient_id", rec.ID),
					zap.String("channel", channel))
				return
			}

			recipientType := s.getRecipientTypeForChannel(channel)

			sendRequest := &notification.SendRequest{
				Subject:       subject,
				Content:       content,
				Priority:      notificationConfig.Priority,
				RecipientType: recipientType,
				RecipientID:   rec.ID,
				RecipientAddr: recipientAddr,
				RecipientName: rec.Name,
				InstanceID:    &instance.ID,
				EventType:     eventType,
				Metadata: map[string]interface{}{
					"notification_id": notificationConfig.ID,
					"instance_id":     instance.ID,
					"sender_id":       senderID,
					"recipient_type":  rec.Type,
				},
			}

			response, err := s.notificationMgr.SendNotification(recipientCtx, sendRequest)

			log := &model.WorkorderNotificationLog{
				NotificationID: notificationConfig.ID,
				InstanceID:     &instance.ID,
				EventType:      eventType,
				Channel:        channel,
				RecipientType:  rec.Type,
				RecipientID:    rec.ID,
				RecipientName:  rec.Name,
				RecipientAddr:  recipientAddr,
				Subject:        subject,
				Content:        content,
				Status:         2,
				SendAt:         time.Now(),
				SenderID:       senderID,
			}

			if err != nil {
				log.Status = 4
				log.ErrorMessage = err.Error()
				s.logger.Error("发送通知失败",
					zap.String("channel", channel),
					zap.String("recipient", recipientAddr),
					zap.Error(err))
				recipientErrors <- fmt.Errorf("接收人 %s 发送失败: %w", recipientAddr, err)
			} else if response != nil {
				log.Status = 3
				if response.ExternalID != "" {
					log.ResponseData = map[string]interface{}{
						"external_id": response.ExternalID,
					}
				}
				if response.Cost != nil {
					log.Cost = response.Cost
				}
				s.logger.Info("通知发送成功",
					zap.String("channel", channel),
					zap.String("recipient", recipientAddr))
			}

			if err := s.dao.AddSendLog(recipientCtx, log); err != nil {
				s.logger.Error("记录发送日志失败", zap.Error(err))
			}
		}(recipient)
	}

	wg.Wait()
	close(recipientErrors)

	var errors []string
	for err := range recipientErrors {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		s.logger.Warn("部分接收人发送失败，但其他接收人已成功发送",
			zap.Strings("errors", errors),
			zap.String("channel", channel))
	}

	return nil
}

func (s *workorderNotificationService) buildMessageContent(notificationConfig *model.WorkorderNotification,
	instance *model.WorkorderInstance, eventType string, customContent ...string) (string, string) {

	// 创建发送请求对象，用于模板渲染
	sendRequest := &notification.SendRequest{
		Subject:    notificationConfig.SubjectTemplate,
		Content:    notificationConfig.MessageTemplate,
		Priority:   notificationConfig.Priority,
		EventType:  eventType,
		InstanceID: &instance.ID,
		Templates:  make(map[string]string),
		Metadata:   make(map[string]interface{}),
	}

	sendRequest.Templates["workorder_id"] = fmt.Sprintf("%d", instance.ID)
	sendRequest.Templates["serial_number"] = instance.SerialNumber
	sendRequest.Templates["title"] = instance.Title
	sendRequest.Templates["description"] = instance.Description
	sendRequest.Templates["operator_name"] = instance.OperatorName
	sendRequest.Templates["priority_level"] = fmt.Sprintf("%d", int(instance.Priority))
	sendRequest.Templates["priority_text"] = notification.FormatPriority(instance.Priority)
	sendRequest.Templates["status"] = utils.GetInstanceStatusName(instance.Status)
	sendRequest.Templates["created_time"] = instance.CreatedAt.Format("2006-01-02 15:04:05")
	sendRequest.Templates["event_type"] = notification.GetEventTypeText(eventType)
	sendRequest.Templates["event_type_text"] = notification.GetEventTypeText(eventType)
	sendRequest.Templates["notification_time"] = time.Now().Format("2006-01-02 15:04:05")
	sendRequest.Templates["company_name"] = "AI-CloudOps"
	sendRequest.Templates["platform_name"] = "运维管理平台"
	sendRequest.Templates["department"] = "技术运维部"

	assigneeName := "待分配"
	if instance.AssigneeID != nil {
		if user, err := s.userDAO.GetByID(context.Background(), *instance.AssigneeID); err == nil && user != nil {
			assigneeName = user.RealName
		}
	}
	sendRequest.Templates["assignee_name"] = assigneeName

	// 如果有更新时间，添加更新时间
	if !instance.UpdatedAt.IsZero() {
		sendRequest.Templates["updated_time"] = instance.UpdatedAt.Format("2006-01-02 15:04:05")
	} else {
		sendRequest.Templates["updated_time"] = sendRequest.Templates["created_time"]
	}

	// 如果有自定义内容，添加到变量中
	if len(customContent) > 0 && customContent[0] != "" {
		sendRequest.Templates["custom_content"] = customContent[0]
	} else {
		sendRequest.Templates["custom_content"] = ""
	}

	// 渲染主题
	subject := notificationConfig.SubjectTemplate
	if subject == "" {
		subject = fmt.Sprintf("【AI-CloudOps】工单通知 - %s", instance.Title)
	} else {
		renderedSubject, _ := notification.RenderTemplate(subject, sendRequest)
		subject = renderedSubject
	}

	content := notificationConfig.MessageTemplate
	if content == "" {
		content = fmt.Sprintf(`尊敬的用户，您好！

您收到一条来自AI-CloudOps运维管理平台的工单通知：

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 工单基本信息
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
工单编号：%s
工单标题：%s
当前状态：%s
优先级别：%s
操作人员：%s
处理人员：%s
事件类型：%s
创建时间：%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📝 工单详情
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
%s

%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

此消息由AI-CloudOps运维管理平台发送，请及时处理相关工单。
如有疑问，请联系技术运维部门。

AI-CloudOps 技术运维部
发送时间：%s`,
			instance.SerialNumber,
			instance.Title,
			utils.GetInstanceStatusName(instance.Status),
			notification.FormatPriority(instance.Priority),
			instance.OperatorName,
			assigneeName,
			notification.GetEventTypeText(eventType),
			instance.CreatedAt.Format("2006-01-02 15:04:05"),
			instance.Description,
			sendRequest.Templates["custom_content"],
			time.Now().Format("2006-01-02 15:04:05"))
	} else {
		renderedContent, _ := notification.RenderTemplate(content, sendRequest)
		content = renderedContent
	}

	s.logger.Debug("消息内容构建完成",
		zap.String("subject", subject),
		zap.String("content", content))

	return subject, content
}

// getRecipientAddress 根据渠道类型获取接收人地址
func (s *workorderNotificationService) getRecipientAddress(recipient RecipientInfo, channel string) string {
	userID, err := strconv.Atoi(recipient.ID)
	if err != nil {
		s.logger.Error("无效的用户ID",
			zap.String("recipient_id", recipient.ID),
			zap.Error(err))
		return ""
	}

	user, err := s.userDAO.GetByID(context.Background(), userID)
	if err != nil {
		s.logger.Error("获取用户信息失败",
			zap.Int("user_id", userID),
			zap.Error(err))
		return ""
	}

	switch channel {
	case model.NotificationChannelEmail:
		if user.Email == "" {
			s.logger.Warn("用户没有配置邮箱",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.Email
	case model.NotificationChannelFeishu:
		if user.FeiShuUserId == "" {
			s.logger.Warn("用户没有配置飞书用户ID",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.FeiShuUserId
	case model.NotificationChannelSMS:
		if user.Mobile == "" {
			s.logger.Warn("用户没有配置手机号",
				zap.Int("user_id", userID),
				zap.String("user_name", user.RealName))
			return ""
		}
		return user.Mobile
	case model.NotificationChannelWebhook:
		s.logger.Warn("Webhook地址需要配置",
			zap.Int("user_id", userID),
			zap.String("user_name", user.RealName))
		return ""
	default:
		s.logger.Warn("不支持的通知渠道",
			zap.String("channel", channel))
		return ""
	}
}

func (s *workorderNotificationService) getRecipientTypeForChannel(channel string) string {
	switch channel {
	case model.NotificationChannelEmail:
		return "email"
	case model.NotificationChannelFeishu:
		return "feishu_user"
	case model.NotificationChannelSMS:
		return "sms"
	case model.NotificationChannelWebhook:
		return "webhook"
	default:
		return channel
	}
}

// SendNotificationByChannels 通过多个渠道发送通知
func (s *workorderNotificationService) SendNotificationByChannels(ctx context.Context, channels []string, recipient, subject, content string) error {
	if s.notificationMgr == nil {
		return errors.New("通知管理器未初始化")
	}

	for _, channel := range channels {
		sendRequest := &notification.SendRequest{
			Subject:       subject,
			Content:       content,
			Priority:      2,
			RecipientType: channel,
			RecipientAddr: recipient,
			EventType:     "manual",
		}

		_, err := s.notificationMgr.SendNotification(ctx, sendRequest)
		if err != nil {
			s.logger.Error("发送通知失败",
				zap.String("channel", channel),
				zap.String("recipient", recipient),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// GetAvailableChannels 获取当前可用的通知渠道列表
func (s *workorderNotificationService) GetAvailableChannels() *model.ListResp[*model.WorkorderNotificationChannel] {
	if s.notificationMgr == nil {
		return &model.ListResp[*model.WorkorderNotificationChannel]{
			Items: []*model.WorkorderNotificationChannel{},
			Total: 0,
		}
	}

	availableChannels := s.notificationMgr.GetAvailableChannels()
	channels := make([]*model.WorkorderNotificationChannel, 0, len(availableChannels))

	for _, channel := range availableChannels {
		channels = append(channels, &model.WorkorderNotificationChannel{
			Channels: model.StringList{channel},
		})
	}

	return &model.ListResp[*model.WorkorderNotificationChannel]{
		Items: channels,
		Total: int64(len(channels)),
	}
}

type RecipientInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}
