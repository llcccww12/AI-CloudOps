package model

// OpsDashboardOverviewReq 运营指挥大屏概览查询
type OpsDashboardOverviewReq struct {
	Days int `json:"days" form:"days"` // 趋势天数，默认 7
}

// OpsDashboardKPIs 顶部核心指标
type OpsDashboardKPIs struct {
	ExhibitionToday      int64 `json:"exhibition_today"`
	ExhibitionWeek       int64 `json:"exhibition_week"`
	VisitPending         int64 `json:"visit_pending"`
	VisitDueSoon         int64 `json:"visit_due_soon"`
	VisitOverdue         int64 `json:"visit_overdue"`
	CustomerIntent       int64 `json:"customer_intent"`
	CustomerTrial        int64 `json:"customer_trial"`
	CustomerFormal       int64 `json:"customer_formal"`
	CustomerClosedMonth  int64 `json:"customer_closed_month"`
}

// OpsDashboardFunnelStage 漏斗阶段
type OpsDashboardFunnelStage struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Count int64   `json:"count"`
	Rate  float64 `json:"rate"` // 相对上一阶段转化率 0-100；首阶为 100
}

// OpsDashboardTrendPoint 日趋势点
type OpsDashboardTrendPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// OpsDashboardIntentItem 意向分布
type OpsDashboardIntentItem struct {
	Intent string `json:"intent"`
	Label  string `json:"label"`
	Count  int64  `json:"count"`
}

// OpsDashboardVisitAlert 外访预警项
type OpsDashboardVisitAlert struct {
	ID              int    `json:"id"`
	Title           string `json:"title"`
	TargetOrg       string `json:"target_org"`
	FollowOwnerName string `json:"follow_owner_name"`
	DueAt           string `json:"due_at,omitempty"`
	Intent          string `json:"intent"`
	Status          string `json:"status"`
	AlertType       string `json:"alert_type"` // overdue | due_soon | high_intent
}

// OpsDashboardEvent 近期动态
type OpsDashboardEvent struct {
	ID        int    `json:"id"`
	Type      string `json:"type"` // exhibition_transfer | visit_convert | customer_stage
	Title     string `json:"title"`
	Time      string `json:"time"`
	RefID     int    `json:"ref_id"`
	RefPath   string `json:"ref_path"`
}

// OpsDashboardOverview 运营指挥大屏总览
type OpsDashboardOverview struct {
	KPIs             OpsDashboardKPIs          `json:"kpis"`
	Funnel           []OpsDashboardFunnelStage `json:"funnel"`
	ExhibitionTrend  []OpsDashboardTrendPoint  `json:"exhibition_trend"`
	IntentDist       []OpsDashboardIntentItem  `json:"intent_dist"`
	VisitAlerts      []OpsDashboardVisitAlert  `json:"visit_alerts"`
	RecentEvents     []OpsDashboardEvent       `json:"recent_events"`
}
