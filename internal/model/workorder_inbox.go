package model

import "time"

const (
	InboxMessageUnread int8 = 1 // 未读
	InboxMessageRead   int8 = 2 // 已读
)

// WorkorderInboxMessage 工单站内信
type WorkorderInboxMessage struct {
	Model
	UserID     int        `json:"user_id" gorm:"column:user_id;not null;index;comment:接收用户ID"`
	Title      string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:标题"`
	Content    string     `json:"content" gorm:"column:content;type:text;not null;comment:内容"`
	EventType  string     `json:"event_type" gorm:"column:event_type;type:varchar(50);index;comment:工单事件类型"`
	InstanceID *int       `json:"instance_id" gorm:"column:instance_id;index;comment:关联工单ID"`
	IsRead     int8       `json:"is_read" gorm:"column:is_read;not null;default:1;index;comment:1未读 2已读"`
	ReadAt     *time.Time `json:"read_at" gorm:"column:read_at;comment:阅读时间"`
}

func (WorkorderInboxMessage) TableName() string {
	return "cl_workorder_inbox"
}

type ListWorkorderInboxReq struct {
	ListReq
	UserID int `json:"-"`
}

type WorkorderInboxUnreadCount struct {
	Count int64 `json:"count"`
}
