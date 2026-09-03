package model

// OpsManagerWeeklyReportReq 管理者周报
type OpsManagerWeeklyReportReq struct {
	Days    int  `json:"days" form:"days"`       // 默认 7
	Refresh bool `json:"refresh" form:"refresh"` // true 强制重新生成
}

// OpsManagerWeeklyReport 超管周报（预览 + 复制）
type OpsManagerWeeklyReport struct {
	GeneratedAt      string             `json:"generated_at"`
	PeriodStart      string             `json:"period_start"`
	PeriodEnd        string             `json:"period_end"`
	Days             int                `json:"days"`
	Snapshot         OpsManagerSnapshot `json:"snapshot"`
	ExecutiveSummary string             `json:"executive_summary"`
	Markdown         string             `json:"markdown"`
	HTML             string             `json:"html"`
	Source           string             `json:"source"` // template / hybrid
	LLMEnabled       bool               `json:"llm_enabled"`
	Cached           bool               `json:"cached"`
	Hint             string             `json:"hint,omitempty"`
}

// OpsManagerSnapshot 报告事实层（可复核）
type OpsManagerSnapshot struct {
	KPIs            OpsDashboardKPIs          `json:"kpis"`
	Funnel          []OpsDashboardFunnelStage `json:"funnel"`
	RiskSummary     OpsRiskSummary            `json:"risk_summary"`
	TopRisks        []OpsRiskItem             `json:"top_risks"`
	FactBullets     []string                  `json:"fact_bullets"`
	SuggestedActions []string                 `json:"suggested_actions"`
}
