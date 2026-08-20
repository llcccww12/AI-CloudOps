package model

import "time"

// 运营工作台风险类型
const (
	OpsRiskTrialExpiring      = "trial_expiring"
	OpsRiskContractExpiring   = "contract_expiring"
	OpsRiskSettlementDueSoon  = "settlement_due_soon"
	OpsRiskSettlementOverdue  = "settlement_overdue"
	OpsRiskSurveyLowScore     = "survey_low_score"
	OpsRiskNonRenewalPending  = "non_renewal_pending"
)

const (
	OpsRiskSeverityHigh   = "high"
	OpsRiskSeverityMedium = "medium"
	OpsRiskSeverityLow    = "low"
)

// OpsWorkbenchBriefingReq 工作台简报查询
type OpsWorkbenchBriefingReq struct {
	WithinDays int  `json:"within_days" form:"within_days"`
	MineOnly   bool `json:"mine_only" form:"mine_only"`
	OwnerID    int  `json:"-"` // 由 API 注入当前用户
	IsAdmin    bool `json:"-"`
}

// OpsRiskItem 风险事项
type OpsRiskItem struct {
	Type         string     `json:"type"`
	Severity     string     `json:"severity"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary"`
	Suggestion   string     `json:"suggestion"`
	CustomerID   int        `json:"customer_id"`
	CustomerName string     `json:"customer_name"`
	OwnerID      int        `json:"owner_id"`
	OwnerName    string     `json:"owner_name"`
	BizType      string     `json:"biz_type"`
	BizID        int        `json:"biz_id"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	RefPath      string     `json:"ref_path"`
	DraftHint    string     `json:"draft_hint,omitempty"` // 文案草稿场景提示
}

// OpsWorkbenchBriefing 运营工作台简报
type OpsWorkbenchBriefing struct {
	GeneratedAt string         `json:"generated_at"`
	WithinDays  int            `json:"within_days"`
	Summary     OpsRiskSummary `json:"summary"`
	Items       []OpsRiskItem  `json:"items"`
}

type OpsRiskSummary struct {
	Total              int `json:"total"`
	TrialExpiring      int `json:"trial_expiring"`
	ContractExpiring   int `json:"contract_expiring"`
	SettlementDueSoon  int `json:"settlement_due_soon"`
	SettlementOverdue  int `json:"settlement_overdue"`
	SurveyLowScore     int `json:"survey_low_score"`
	NonRenewalPending  int `json:"non_renewal_pending"`
}

// OpsReminderDraftReq 提醒文案草稿
type OpsReminderDraftReq struct {
	Type         string `json:"type" binding:"required"`
	Channel      string `json:"channel"` // inbox/feishu/sms/email
	CustomerID   int    `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	Suggestion   string `json:"suggestion"`
	BizType      string `json:"biz_type"`
	BizID        int    `json:"biz_id"`
	DueAt        string `json:"due_at"`
	Tone         string `json:"tone"` // formal/friendly
}

// OpsReminderDraftResp 文案草稿结果
type OpsReminderDraftResp struct {
	Channel    string `json:"channel"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	Source     string `json:"source"` // template / llm
	Editable   bool   `json:"editable"`
	Hint       string `json:"hint,omitempty"`
}
