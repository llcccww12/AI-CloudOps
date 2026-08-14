package notification

import (
	"context"
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type InboxWriter interface {
	Create(ctx context.Context, msg *model.WorkorderInboxMessage) error
}

type InboxChannel struct {
	writer InboxWriter
	logger *zap.Logger
}

func NewInboxChannel(writer InboxWriter, logger *zap.Logger) *InboxChannel {
	return &InboxChannel{writer: writer, logger: logger}
}

func (c *InboxChannel) GetName() string {
	return model.NotificationChannelInbox
}

func (c *InboxChannel) Validate() error {
	return nil
}

func (c *InboxChannel) IsEnabled() bool {
	return true
}

func (c *InboxChannel) GetMaxRetries() int {
	return 0
}

func (c *InboxChannel) GetRetryInterval() time.Duration {
	return 0
}

func (c *InboxChannel) Send(ctx context.Context, request *SendRequest) (*SendResponse, error) {
	userID, err := strconv.Atoi(request.RecipientID)
	if err != nil || userID <= 0 {
		userID, err = strconv.Atoi(request.RecipientAddr)
	}
	if err != nil || userID <= 0 {
		return &SendResponse{
			Success:      false,
			MessageID:    request.MessageID,
			Status:       "failed",
			ErrorMessage: "站内信接收人ID无效",
			SendTime:     time.Now(),
		}, err
	}

	title := request.Subject
	if title == "" {
		title = "工单通知"
	}

	var instanceID *int
	if request.InstanceID != nil && *request.InstanceID > 0 && *request.InstanceID != 999999 {
		instanceID = request.InstanceID
	}

	msg := &model.WorkorderInboxMessage{
		UserID:     userID,
		Title:      title,
		Content:    request.Content,
		EventType:  request.EventType,
		InstanceID: instanceID,
		IsRead:     model.InboxMessageUnread,
	}

	if err := c.writer.Create(ctx, msg); err != nil {
		c.logger.Error("写入站内信失败", zap.Error(err), zap.Int("user_id", userID))
		return &SendResponse{
			Success:      false,
			MessageID:    request.MessageID,
			Status:       "failed",
			ErrorMessage: err.Error(),
			SendTime:     time.Now(),
		}, err
	}

	return &SendResponse{
		Success:    true,
		MessageID:  request.MessageID,
		ExternalID: strconv.Itoa(msg.ID),
		Status:     "success",
		SendTime:   time.Now(),
	}, nil
}
