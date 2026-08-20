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

package model

import "time"

const (
	OpsExhibitionStatusDraft      = "draft"
	OpsExhibitionStatusReviewed   = "reviewed"
	OpsExhibitionStatusConverted  = "converted"  // 兼容旧数据：已转客户
	OpsExhibitionStatusTransferred = "transferred" // 已转入外访

	OpsExhibitionSourceStaff  = "staff"
	OpsExhibitionSourcePublic = "public"

	OpsExhibitionIntentLow    = "low"
	OpsExhibitionIntentMedium = "medium"
	OpsExhibitionIntentHigh   = "high"

	OpsVisitStatusPending   = "pending"   // 待约
	OpsVisitStatusBooked    = "booked"    // 已约
	OpsVisitStatusVisited   = "visited"   // 已访
	OpsVisitStatusConverted = "converted" // 已转客户
	OpsVisitStatusPlanned   = "planned"   // 兼容旧值
	OpsVisitStatusDone      = "done"      // 兼容旧值

	OpsVisitSourceStaff      = "staff"
	OpsVisitSourceExhibition = "exhibition"

	OpsVisitIntentLow    = "low"
	OpsVisitIntentMedium = "medium"
	OpsVisitIntentHigh   = "high"
)

// OpsExhibition 展厅接待
type OpsExhibition struct {
	Model
	CompanyName    string     `json:"company_name" gorm:"column:company_name;type:varchar(200);not null;index;comment:来访单位"`
	CompanyLevel   string     `json:"company_level" gorm:"column:company_level;type:varchar(64);comment:来访单位层级"`
	VisitorName    string     `json:"visitor_name" gorm:"column:visitor_name;type:varchar(100);comment:来访人"`
	VisitorTitle   string     `json:"visitor_title" gorm:"column:visitor_title;type:varchar(100);comment:职务"`
	VisitorCount   int        `json:"visitor_count" gorm:"column:visitor_count;default:0;comment:来访人数"`
	VisitAt        *time.Time `json:"visit_at" gorm:"column:visit_at;index;comment:来访时间"`
	DockingUnit    string     `json:"docking_unit" gorm:"column:docking_unit;type:varchar(200);comment:对接单位"`
	HostName       string     `json:"host_name" gorm:"column:host_name;type:varchar(100);comment:接待人"`
	Purpose        string     `json:"purpose" gorm:"column:purpose;type:varchar(100);comment:来访目的"`
	NeedMeeting    int8       `json:"need_meeting" gorm:"column:need_meeting;type:tinyint;default:0;comment:是否会谈"`
	FocusTags      StringList `json:"focus_tags" gorm:"column:focus_tags;type:text;serializer:json;comment:关注重点"`
	Content        string     `json:"content" gorm:"column:content;type:text;comment:讲解内容"`
	MeetingMinutes string     `json:"meeting_minutes" gorm:"column:meeting_minutes;type:text;comment:会议纪要"`
	ContactPhone   string     `json:"contact_phone" gorm:"column:contact_phone;type:varchar(32);comment:带队人联系电话"`
	Companions     string     `json:"companions" gorm:"column:companions;type:varchar(255);comment:陪同人员"`
	Intent         string     `json:"intent" gorm:"column:intent;type:varchar(16);index;comment:意向低中高"`
	Source         string     `json:"source" gorm:"column:source;type:varchar(16);not null;default:staff;index;comment:来源staff/public"`
	Status         string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:draft;index;comment:状态"`
	VisitID        *int       `json:"visit_id" gorm:"column:visit_id;index;comment:关联外访"`
	CustomerID     *int       `json:"customer_id" gorm:"column:customer_id;index;comment:关联客户"`
	OperatorID     int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName   string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
	UpdaterID      int        `json:"updater_id" gorm:"column:updater_id;index;comment:最后编辑人ID"`
	UpdaterName    string     `json:"updater_name" gorm:"column:updater_name;type:varchar(100);comment:最后编辑人"`
	Remark         string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (OpsExhibition) TableName() string { return "cl_ops_exhibition" }

type CreateOpsExhibitionReq struct {
	CompanyName    string     `json:"company_name" binding:"required,min=1,max=200"`
	CompanyLevel   string     `json:"company_level" binding:"required"`
	VisitorName    string     `json:"visitor_name" binding:"required,min=1,max=100"`
	VisitorTitle   string     `json:"visitor_title"`
	VisitorCount   int        `json:"visitor_count" binding:"required,min=1"`
	VisitAt        *time.Time `json:"visit_at" binding:"required"`
	DockingUnit    string     `json:"docking_unit"`
	HostName       string     `json:"host_name"`
	Purpose        string     `json:"purpose" binding:"required"`
	NeedMeeting    int8       `json:"need_meeting"`
	FocusTags      StringList `json:"focus_tags"`
	Content        string     `json:"content"`
	MeetingMinutes string     `json:"meeting_minutes"`
	ContactPhone   string     `json:"contact_phone" binding:"required,min=5,max=32"`
	Companions     string     `json:"companions"`
	Intent         string     `json:"intent" binding:"omitempty,oneof=low medium high"`
	Remark         string     `json:"remark"`
	CustomerID     *int       `json:"customer_id"`
	OperatorID     int        `json:"operator_id"`
	OperatorName   string     `json:"operator_name"`
}

type UpdateOpsExhibitionReq struct {
	ID             int        `json:"id" binding:"required,min=1"`
	CompanyName    string     `json:"company_name" binding:"required,min=1,max=200"`
	CompanyLevel   string     `json:"company_level" binding:"required"`
	VisitorName    string     `json:"visitor_name" binding:"required,min=1,max=100"`
	VisitorTitle   string     `json:"visitor_title"`
	VisitorCount   int        `json:"visitor_count" binding:"required,min=1"`
	VisitAt        *time.Time `json:"visit_at" binding:"required"`
	DockingUnit    string     `json:"docking_unit"`
	HostName       string     `json:"host_name"`
	Purpose        string     `json:"purpose" binding:"required"`
	NeedMeeting    int8       `json:"need_meeting"`
	FocusTags      StringList `json:"focus_tags"`
	Content        string     `json:"content"`
	MeetingMinutes string     `json:"meeting_minutes"`
	ContactPhone   string     `json:"contact_phone" binding:"required,min=5,max=32"`
	Companions     string     `json:"companions"`
	Intent         string     `json:"intent" binding:"omitempty,oneof=low medium high"`
	Remark         string     `json:"remark"`
	CustomerID     *int       `json:"customer_id"`
	UpdaterID      int        `json:"-"`
	UpdaterName    string     `json:"-"`
}

type ListOpsExhibitionReq struct {
	ListReq
	Status string `json:"status" form:"status"`
	Source string `json:"source" form:"source"`
}

// SubmitPublicVisitorReq 公开访客登记提交
type SubmitPublicVisitorReq struct {
	CompanyName    string     `json:"company_name" binding:"required,min=1,max=200"`
	CompanyLevel   string     `json:"company_level" binding:"required"`
	VisitorName    string     `json:"visitor_name" binding:"required,min=1,max=100"`
	VisitorTitle   string     `json:"visitor_title"`
	VisitorCount   int        `json:"visitor_count" binding:"required,min=1"`
	VisitAt        *time.Time `json:"visit_at" binding:"required"`
	DockingUnit    string     `json:"docking_unit"`
	HostName       string     `json:"host_name"`
	Purpose        string     `json:"purpose" binding:"required"`
	NeedMeeting    int8       `json:"need_meeting"`
	Content        string     `json:"content"`
	MeetingMinutes string     `json:"meeting_minutes"`
	ContactPhone   string     `json:"contact_phone" binding:"required,min=5,max=32"`
	AttachmentIDs  []int      `json:"attachment_ids"`
}

// OpsVisit 外访交流（企业走访台账一期：单表覆盖档案/联系人/计划/记录/转化）
type OpsVisit struct {
	Model
	Title        string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:外访主题"`
	TargetOrg    string     `json:"target_org" gorm:"column:target_org;type:varchar(200);index;comment:对象单位"`
	StartAt      *time.Time `json:"start_at" gorm:"column:start_at;comment:开始时间"`
	EndAt        *time.Time `json:"end_at" gorm:"column:end_at;comment:结束时间"`
	Location     string     `json:"location" gorm:"column:location;type:varchar(200);comment:地点"`
	Participants string     `json:"participants" gorm:"column:participants;type:varchar(255);comment:参与人"`
	Summary      string     `json:"summary" gorm:"column:summary;type:text;comment:交流要点"`
	Outcome      string     `json:"outcome" gorm:"column:outcome;type:text;comment:成果与线索"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:pending;index;comment:状态待约已约已访"`
	Source       string     `json:"source" gorm:"column:source;type:varchar(32);not null;default:staff;index;comment:来源"`
	LeadSource   string     `json:"lead_source" gorm:"column:lead_source;type:varchar(64);comment:线索来源"`
	ExhibitionID *int       `json:"exhibition_id" gorm:"column:exhibition_id;index;comment:来源展厅ID"`
	CustomerID   *int       `json:"customer_id" gorm:"column:customer_id;index;comment:关联客户"`

	// A 企业档案
	CreditCode     string `json:"credit_code" gorm:"column:credit_code;type:varchar(64);comment:信用代码"`
	Industry       string `json:"industry" gorm:"column:industry;type:varchar(100);comment:行业细分"`
	CompanyScale   string `json:"company_scale" gorm:"column:company_scale;type:varchar(64);comment:规模"`
	Qualifications string `json:"qualifications" gorm:"column:qualifications;type:varchar(255);comment:资质"`
	FinanceStatus  string `json:"finance_status" gorm:"column:finance_status;type:varchar(64);comment:融资上市情况"`
	Address        string `json:"address" gorm:"column:address;type:varchar(255);comment:地址"`
	ProductLine    string `json:"product_line" gorm:"column:product_line;type:varchar(255);comment:产品线"`
	HasCooperation int8   `json:"has_cooperation" gorm:"column:has_cooperation;type:tinyint;default:0;comment:是否已有合作"`

	// B 联系人
	ContactName   string `json:"contact_name" gorm:"column:contact_name;type:varchar(100);comment:联系人"`
	ContactTitle  string `json:"contact_title" gorm:"column:contact_title;type:varchar(100);comment:职务"`
	DecisionRole  string `json:"decision_role" gorm:"column:decision_role;type:varchar(64);comment:决策角色"`
	ContactPhone  string `json:"contact_phone" gorm:"column:contact_phone;type:varchar(50);comment:电话"`
	ContactEmail  string `json:"contact_email" gorm:"column:contact_email;type:varchar(120);comment:邮箱"`
	ContactWechat string `json:"contact_wechat" gorm:"column:contact_wechat;type:varchar(100);comment:微信"`
	ReferrerName  string `json:"referrer_name" gorm:"column:referrer_name;type:varchar(100);comment:引荐人"`
	HostName      string `json:"host_name" gorm:"column:host_name;type:varchar(100);comment:接待人"`

	// C 走访计划
	D0At          *time.Time `json:"d0_at" gorm:"column:d0_at;index;comment:来访登记日D0"`
	PlannedAt     *time.Time `json:"planned_at" gorm:"column:planned_at;comment:计划走访日"`
	VisitGoal     string     `json:"visit_goal" gorm:"column:visit_goal;type:text;comment:走访目标"`
	PrepMaterials string     `json:"prep_materials" gorm:"column:prep_materials;type:text;comment:准备材料"`

	// D 走访记录
	DurationMin    int    `json:"duration_min" gorm:"column:duration_min;default:0;comment:时长分钟"`
	LocationType   string `json:"location_type" gorm:"column:location_type;type:varchar(32);comment:上门来访线上"`
	PainPoints     string `json:"pain_points" gorm:"column:pain_points;type:text;comment:需求痛点"`
	Objections     string `json:"objections" gorm:"column:objections;type:text;comment:异议"`
	CompetitorInfo string `json:"competitor_info" gorm:"column:competitor_info;type:text;comment:竞品信息"`
	SiteFeedback   string `json:"site_feedback" gorm:"column:site_feedback;type:text;comment:现场反馈"`

	// E 跟进转化
	Intent             string     `json:"intent" gorm:"column:intent;type:varchar(16);index;comment:意向低中高"`
	MatchScore         int        `json:"match_score" gorm:"column:match_score;default:0;comment:需求匹配分"`
	OpportunityAmount  float64    `json:"opportunity_amount" gorm:"column:opportunity_amount;type:decimal(14,2);default:0;comment:预估商机金额"`
	NextAction         string     `json:"next_action" gorm:"column:next_action;type:varchar(255);comment:下一步行动"`
	FollowOwnerID      int        `json:"follow_owner_id" gorm:"column:follow_owner_id;index;comment:外访对接人ID"`
	FollowOwnerName    string     `json:"follow_owner_name" gorm:"column:follow_owner_name;type:varchar(100);comment:外访对接人"`
	NextFollowAt       *time.Time `json:"next_follow_at" gorm:"column:next_follow_at;index;comment:下次跟进日"`
	AssignedAt         *time.Time `json:"assigned_at" gorm:"column:assigned_at;index;comment:任务下发时间"`
	DueAt              *time.Time `json:"due_at" gorm:"column:due_at;index;comment:走访截止时间"`
	PreDueReminded     int8       `json:"pre_due_reminded" gorm:"column:pre_due_reminded;type:tinyint;default:0;comment:到期前3天是否已提醒"`
	CRMTransferred     int8       `json:"crm_transferred" gorm:"column:crm_transferred;type:tinyint;default:0;comment:是否已转CRM"`

	OperatorID   int    `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName string `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
	UpdaterID    int    `json:"updater_id" gorm:"column:updater_id;index;comment:最后编辑人ID"`
	UpdaterName  string `json:"updater_name" gorm:"column:updater_name;type:varchar(100);comment:最后编辑人"`
	Remark       string `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (OpsVisit) TableName() string { return "cl_ops_visit" }

type CreateOpsVisitReq struct {
	Title              string     `json:"title" binding:"required,min=1,max=200"`
	TargetOrg          string     `json:"target_org" binding:"required,min=1,max=200"`
	StartAt            *time.Time `json:"start_at"`
	EndAt              *time.Time `json:"end_at"`
	Location           string     `json:"location"`
	Participants       string     `json:"participants"`
	Summary            string     `json:"summary"`
	Outcome            string     `json:"outcome"`
	Status             string     `json:"status" binding:"omitempty,oneof=pending booked visited planned done converted"`
	LeadSource         string     `json:"lead_source"`
	CreditCode         string     `json:"credit_code"`
	Industry           string     `json:"industry"`
	CompanyScale       string     `json:"company_scale"`
	Qualifications     string     `json:"qualifications"`
	FinanceStatus      string     `json:"finance_status"`
	Address            string     `json:"address"`
	ProductLine        string     `json:"product_line"`
	HasCooperation     int8       `json:"has_cooperation"`
	ContactName        string     `json:"contact_name" binding:"required,min=1,max=100"`
	ContactTitle       string     `json:"contact_title"`
	DecisionRole       string     `json:"decision_role"`
	ContactPhone       string     `json:"contact_phone" binding:"required,min=5,max=32"`
	ContactEmail       string     `json:"contact_email"`
	ContactWechat      string     `json:"contact_wechat"`
	ReferrerName       string     `json:"referrer_name"`
	HostName           string     `json:"host_name"`
	D0At               *time.Time `json:"d0_at"`
	PlannedAt          *time.Time `json:"planned_at"`
	VisitGoal          string     `json:"visit_goal"`
	PrepMaterials      string     `json:"prep_materials"`
	DurationMin        int        `json:"duration_min"`
	LocationType       string     `json:"location_type"`
	PainPoints         string     `json:"pain_points"`
	Objections         string     `json:"objections"`
	CompetitorInfo     string     `json:"competitor_info"`
	SiteFeedback       string     `json:"site_feedback"`
	Intent             string     `json:"intent" binding:"omitempty,oneof=low medium high"`
	MatchScore         int        `json:"match_score"`
	OpportunityAmount  float64    `json:"opportunity_amount"`
	NextAction         string     `json:"next_action"`
	FollowOwnerID      int        `json:"follow_owner_id"`
	FollowOwnerName    string     `json:"follow_owner_name"`
	NextFollowAt       *time.Time `json:"next_follow_at"`
	Remark             string     `json:"remark"`
	CustomerID         *int       `json:"customer_id"`
	OperatorID         int        `json:"operator_id"`
	OperatorName       string     `json:"operator_name"`
}

type UpdateOpsVisitReq struct {
	ID                 int        `json:"id" binding:"required,min=1"`
	Title              string     `json:"title" binding:"required,min=1,max=200"`
	TargetOrg          string     `json:"target_org" binding:"required,min=1,max=200"`
	StartAt            *time.Time `json:"start_at"`
	EndAt              *time.Time `json:"end_at"`
	Location           string     `json:"location"`
	Participants       string     `json:"participants"`
	Summary            string     `json:"summary"`
	Outcome            string     `json:"outcome"`
	Status             string     `json:"status" binding:"omitempty,oneof=pending booked visited planned done converted"`
	LeadSource         string     `json:"lead_source"`
	CreditCode         string     `json:"credit_code"`
	Industry           string     `json:"industry"`
	CompanyScale       string     `json:"company_scale"`
	Qualifications     string     `json:"qualifications"`
	FinanceStatus      string     `json:"finance_status"`
	Address            string     `json:"address"`
	ProductLine        string     `json:"product_line"`
	HasCooperation     int8       `json:"has_cooperation"`
	ContactName        string     `json:"contact_name" binding:"required,min=1,max=100"`
	ContactTitle       string     `json:"contact_title"`
	DecisionRole       string     `json:"decision_role"`
	ContactPhone       string     `json:"contact_phone" binding:"required,min=5,max=32"`
	ContactEmail       string     `json:"contact_email"`
	ContactWechat      string     `json:"contact_wechat"`
	ReferrerName       string     `json:"referrer_name"`
	HostName           string     `json:"host_name"`
	D0At               *time.Time `json:"d0_at"`
	PlannedAt          *time.Time `json:"planned_at"`
	VisitGoal          string     `json:"visit_goal"`
	PrepMaterials      string     `json:"prep_materials"`
	DurationMin        int        `json:"duration_min"`
	LocationType       string     `json:"location_type"`
	PainPoints         string     `json:"pain_points"`
	Objections         string     `json:"objections"`
	CompetitorInfo     string     `json:"competitor_info"`
	SiteFeedback       string     `json:"site_feedback"`
	Intent             string     `json:"intent" binding:"omitempty,oneof=low medium high"`
	MatchScore         int        `json:"match_score"`
	OpportunityAmount  float64    `json:"opportunity_amount"`
	NextAction         string     `json:"next_action"`
	FollowOwnerID      int        `json:"follow_owner_id"`
	FollowOwnerName    string     `json:"follow_owner_name"`
	NextFollowAt       *time.Time `json:"next_follow_at"`
	Remark             string     `json:"remark"`
	CustomerID         *int       `json:"customer_id"`
	UpdaterID          int        `json:"-"`
	UpdaterName        string     `json:"-"`
}

type ListOpsVisitReq struct {
	ListReq
	Status string `json:"status" form:"status"`
	Source string `json:"source" form:"source"`
	Intent string `json:"intent" form:"intent"`
}

type ConvertLeadReq struct {
	ID           int        `json:"id" binding:"required,min=1"`
	DemandTypes  StringList `json:"demand_types"`
	ContactName  string     `json:"contact_name"`
	ContactPhone string     `json:"contact_phone"`
	OwnerID      int        `json:"owner_id"` // 外访对接人 / 客户负责人
	OwnerName    string     `json:"owner_name"`
	OperatorID   int        `json:"operator_id"`
	OperatorName string     `json:"operator_name"`
}

const (
	OpsVisitAssignDays  = 15 // 任务下发后需在此天数内完成走访
	OpsVisitPreDueDays  = 3  // 截止前几天发送提醒
)

// IsHighOrMediumIntent 意向是否为中/高
func IsHighOrMediumIntent(intent string) bool {
	return intent == OpsExhibitionIntentMedium || intent == OpsExhibitionIntentHigh ||
		intent == OpsVisitIntentMedium || intent == OpsVisitIntentHigh
}
