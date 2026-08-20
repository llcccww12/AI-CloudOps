package model

import "time"

const (
	OpsReminderSceneTrialExpire     = "trial_expire"
	OpsReminderSceneContractRenew   = "contract_renew"
	OpsReminderSceneSettlementDue   = "settlement_due"
	OpsReminderSceneInvoice         = "invoice"
	OpsReminderScenePaymentFollowup = "payment_followup"
	OpsReminderSceneVisitPreDue     = "visit_pre_due"
	OpsReminderSceneCustom          = "custom"

	OpsReminderChannelInbox  = "inbox"
	OpsReminderChannelFeishu = "feishu"
	OpsReminderChannelSMS    = "sms"
	OpsReminderChannelEmail  = "email"

	OpsReminderDeliveryPending = "pending"
	OpsReminderDeliverySuccess = "success"
	OpsReminderDeliverySkipped = "skipped"
	OpsReminderDeliveryFailed  = "failed"

	OpsReminderTaskPending = "pending"
	OpsReminderTaskSent    = "sent"
	OpsReminderTaskCancelled = "cancelled"
)

// OpsReminderRule 场景提醒规则
type OpsReminderRule struct {
	Model
	Scene        string     `json:"scene" gorm:"column:scene;type:varchar(64);not null;index;comment:场景"`
	Name         string     `json:"name" gorm:"column:name;type:varchar(100);not null;comment:名称"`
	AdvanceDays  int        `json:"advance_days" gorm:"column:advance_days;not null;default:0;comment:提前天数"`
	Enabled      int8       `json:"enabled" gorm:"column:enabled;not null;default:1;comment:启用1是2否"`
	Channels     StringList `json:"channels" gorm:"column:channels;type:text;serializer:json;comment:渠道"`
	ExtraUserIDs []int      `json:"extra_user_ids" gorm:"column:extra_user_ids;type:text;serializer:json;comment:附加接收人"`
	Remark       string     `json:"remark" gorm:"column:remark;type:varchar(255);comment:备注"`
}

func (OpsReminderRule) TableName() string { return "cl_ops_reminder_rule" }

type CreateOpsReminderRuleReq struct {
	Scene        string     `json:"scene" binding:"required,oneof=trial_expire contract_renew settlement_due invoice payment_followup visit_pre_due"`
	Name         string     `json:"name" binding:"required,min=1,max=100"`
	AdvanceDays  int        `json:"advance_days" binding:"min=0,max=365"`
	Enabled      int8       `json:"enabled" binding:"omitempty,oneof=1 2"`
	Channels     StringList `json:"channels"`
	ExtraUserIDs []int      `json:"extra_user_ids"`
	Remark       string     `json:"remark"`
}

type UpdateOpsReminderRuleReq struct {
	ID           int        `json:"id" binding:"required,min=1"`
	Name         string     `json:"name"`
	AdvanceDays  int        `json:"advance_days"`
	Enabled      int8       `json:"enabled" binding:"omitempty,oneof=1 2"`
	Channels     StringList `json:"channels"`
	ExtraUserIDs []int      `json:"extra_user_ids"`
	Remark       string     `json:"remark"`
}

// OpsReminderTask 自定义到期提醒
type OpsReminderTask struct {
	Model
	Title        string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:标题"`
	CustomerID   int        `json:"customer_id" gorm:"column:customer_id;index;comment:客户ID"`
	BizType      string     `json:"biz_type" gorm:"column:biz_type;type:varchar(32);comment:业务类型"`
	BizID        int        `json:"biz_id" gorm:"column:biz_id;index;comment:业务ID"`
	DueAt        time.Time  `json:"due_at" gorm:"column:due_at;not null;index;comment:到期时间"`
	AdvanceDays  int        `json:"advance_days" gorm:"column:advance_days;not null;default:0;comment:提前天数"`
	Channels     StringList `json:"channels" gorm:"column:channels;type:text;serializer:json;comment:渠道"`
	TargetUserID int        `json:"target_user_id" gorm:"column:target_user_id;not null;index;comment:主接收人"`
	ExtraUserIDs []int      `json:"extra_user_ids" gorm:"column:extra_user_ids;type:text;serializer:json;comment:附加接收人"`
	Content      string     `json:"content" gorm:"column:content;type:varchar(500);comment:提醒内容"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:pending;index;comment:状态"`
	SentAt       *time.Time `json:"sent_at" gorm:"column:sent_at;comment:最近发送时间"`
	CreatorID    int        `json:"creator_id" gorm:"column:creator_id;index;comment:创建人"`
	CreatorName  string     `json:"creator_name" gorm:"column:creator_name;type:varchar(100);comment:创建人姓名"`
}

func (OpsReminderTask) TableName() string { return "cl_ops_reminder_task" }

type CreateOpsReminderTaskReq struct {
	Title        string     `json:"title" binding:"required,min=1,max=200"`
	CustomerID   int        `json:"customer_id"`
	BizType      string     `json:"biz_type"`
	BizID        int        `json:"biz_id"`
	DueAt        time.Time  `json:"due_at" binding:"required"`
	AdvanceDays  int        `json:"advance_days" binding:"min=0,max=365"`
	Channels     StringList `json:"channels"`
	TargetUserID int        `json:"target_user_id" binding:"required,min=1"`
	ExtraUserIDs []int      `json:"extra_user_ids"`
	Content      string     `json:"content" binding:"omitempty,max=500"`
	CreatorID    int        `json:"creator_id"`
	CreatorName  string     `json:"creator_name"`
}

type UpdateOpsReminderTaskReq struct {
	ID           int        `json:"id" binding:"required,min=1"`
	Title        string     `json:"title"`
	DueAt        *time.Time `json:"due_at"`
	AdvanceDays  *int       `json:"advance_days"`
	Channels     StringList `json:"channels"`
	TargetUserID int        `json:"target_user_id"`
	ExtraUserIDs []int      `json:"extra_user_ids"`
	Content      string     `json:"content"`
	Status       string     `json:"status" binding:"omitempty,oneof=pending sent cancelled"`
}

type ListOpsReminderTaskReq struct {
	ListReq
	CustomerID   int    `json:"customer_id" form:"customer_id"`
	TargetUserID int    `json:"target_user_id" form:"target_user_id"`
	Status       string `json:"status" form:"status"`
}

// OpsReminderDelivery 投递日志
type OpsReminderDelivery struct {
	Model
	SourceType   string `json:"source_type" gorm:"column:source_type;type:varchar(32);not null;index;comment:来源rule/task"`
	SourceID     int    `json:"source_id" gorm:"column:source_id;not null;index;comment:来源ID"`
	Scene        string `json:"scene" gorm:"column:scene;type:varchar(64);index;comment:场景"`
	BizType      string `json:"biz_type" gorm:"column:biz_type;type:varchar(32);comment:业务类型"`
	BizID        int    `json:"biz_id" gorm:"column:biz_id;index;comment:业务ID"`
	CustomerID   int    `json:"customer_id" gorm:"column:customer_id;index;comment:客户ID"`
	TargetUserID int    `json:"target_user_id" gorm:"column:target_user_id;index;comment:接收人"`
	Channel      string `json:"channel" gorm:"column:channel;type:varchar(32);not null;index;comment:渠道"`
	Status       string `json:"status" gorm:"column:status;type:varchar(32);not null;index;comment:状态"`
	Title        string `json:"title" gorm:"column:title;type:varchar(200);comment:标题"`
	Content      string `json:"content" gorm:"column:content;type:text;comment:内容"`
	ErrorMsg     string `json:"error_msg" gorm:"column:error_msg;type:varchar(500);comment:错误信息"`
	DedupeKey    string `json:"dedupe_key" gorm:"column:dedupe_key;type:varchar(191);not null;uniqueIndex;comment:去重键"`
}

func (OpsReminderDelivery) TableName() string { return "cl_ops_reminder_delivery" }

type ListOpsReminderDeliveryReq struct {
	ListReq
	Channel string `json:"channel" form:"channel"`
	Status  string `json:"status" form:"status"`
	Scene   string `json:"scene" form:"scene"`
}

// OpsReminderHit 提醒命中预览
type OpsReminderHit struct {
	RuleID         int      `json:"rule_id"`
	TaskID         int      `json:"task_id,omitempty"`
	Scene          string   `json:"scene"`
	RuleName       string   `json:"rule_name"`
	AdvanceDays    int      `json:"advance_days"`
	TargetUserID   int      `json:"target_user_id"`
	TargetUserHint string   `json:"target_user_hint"`
	ExtraUserIDs   []int    `json:"extra_user_ids,omitempty"`
	Channels       []string `json:"channels,omitempty"`
	BizType        string   `json:"biz_type"`
	BizID          int      `json:"biz_id"`
	BizTitle       string   `json:"biz_title"`
	CustomerID     int      `json:"customer_id"`
	CustomerName   string   `json:"customer_name"`
	Link           string   `json:"link"`
	Reason         string   `json:"reason"`
	SourceType     string   `json:"source_type"` // rule | task
}

type OpsReminderScanResult struct {
	HitCount       int `json:"hit_count"`
	NotifyCount    int `json:"notify_count"`
	SkippedNoOwner int `json:"skipped_no_owner"`
	DeliveryCount  int `json:"delivery_count"`
}
