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

// 客户阶段
const (
	OpsCustomerStageLead   = "lead"
	OpsCustomerStageIntent = "intent"
	OpsCustomerStageTrial  = "trial"
	OpsCustomerStageFormal = "formal"
	OpsCustomerStageClosed = "closed"
)

// 客户来源
const (
	OpsCustomerSourceManual     = "manual"
	OpsCustomerSourceExhibition = "exhibition"
	OpsCustomerSourceVisit      = "visit"
	OpsCustomerSourcePartner    = "partner"
)

// OpsCustomer 运营客户主档
type OpsCustomer struct {
	Model
	Name           string     `json:"name" gorm:"column:name;type:varchar(200);not null;index;comment:客户名称"`
	Stage          string     `json:"stage" gorm:"column:stage;type:varchar(32);not null;index;default:intent;comment:阶段"`
	DemandTypes    StringList `json:"demand_types" gorm:"column:demand_types;type:text;serializer:json;comment:需求类别"`
	Industry       string     `json:"industry" gorm:"column:industry;type:varchar(100);comment:行业"`
	ContactName    string     `json:"contact_name" gorm:"column:contact_name;type:varchar(100);comment:联系人"`
	ContactTitle   string     `json:"contact_title" gorm:"column:contact_title;type:varchar(100);comment:职务"`
	ContactPhone   string     `json:"contact_phone" gorm:"column:contact_phone;type:varchar(50);comment:电话"`
	ContactEmail   string     `json:"contact_email" gorm:"column:contact_email;type:varchar(120);comment:邮箱"`
	Source         string     `json:"source" gorm:"column:source;type:varchar(32);comment:来源"`
	SourceRefType  string     `json:"source_ref_type" gorm:"column:source_ref_type;type:varchar(32);comment:来源单据类型"`
	SourceRefID    *int       `json:"source_ref_id" gorm:"column:source_ref_id;index;comment:来源单据ID"`
	OwnerID        int        `json:"owner_id" gorm:"column:owner_id;not null;index;comment:负责人ID"`
	OwnerName      string     `json:"owner_name" gorm:"column:owner_name;type:varchar(100);comment:负责人"`
	BudgetRange    string     `json:"budget_range" gorm:"column:budget_range;type:varchar(100);comment:预算区间"`
	NextFollowAt   *time.Time `json:"next_follow_at" gorm:"column:next_follow_at;index;comment:下次跟进时间"`
	ClosedReason   string     `json:"closed_reason" gorm:"column:closed_reason;type:varchar(100);comment:闭环原因"`
	Remark         string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
	OperatorID     int        `json:"operator_id" gorm:"column:operator_id;comment:创建人ID"`
	OperatorName   string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:创建人"`
}

func (OpsCustomer) TableName() string { return "cl_ops_customer" }

type CreateOpsCustomerReq struct {
	Name         string     `json:"name" binding:"required,min=1,max=200"`
	Stage        string     `json:"stage" binding:"omitempty,oneof=intent trial formal closed"`
	DemandTypes  StringList `json:"demand_types"`
	Industry     string     `json:"industry"`
	ContactName  string     `json:"contact_name"`
	ContactTitle string     `json:"contact_title"`
	ContactPhone string     `json:"contact_phone"`
	ContactEmail string     `json:"contact_email"`
	Source       string     `json:"source"`
	OwnerID      int        `json:"owner_id"`
	OwnerName    string     `json:"owner_name"`
	BudgetRange  string     `json:"budget_range"`
	NextFollowAt *time.Time `json:"next_follow_at"`
	Remark       string     `json:"remark"`
	OperatorID   int        `json:"operator_id"`
	OperatorName string     `json:"operator_name"`
}

type UpdateOpsCustomerReq struct {
	ID           int        `json:"id" binding:"required,min=1"`
	Name         string     `json:"name" binding:"required,min=1,max=200"`
	DemandTypes  StringList `json:"demand_types"`
	Industry     string     `json:"industry"`
	ContactName  string     `json:"contact_name"`
	ContactTitle string     `json:"contact_title"`
	ContactPhone string     `json:"contact_phone"`
	ContactEmail string     `json:"contact_email"`
	OwnerID      int        `json:"owner_id"`
	OwnerName    string     `json:"owner_name"`
	BudgetRange  string     `json:"budget_range"`
	NextFollowAt *time.Time `json:"next_follow_at"`
	Remark       string     `json:"remark"`
}

type ChangeOpsCustomerStageReq struct {
	ID           int    `json:"id" binding:"required,min=1"`
	Stage        string `json:"stage" binding:"required,oneof=intent trial formal closed"`
	ClosedReason string `json:"closed_reason"`
}

type ListOpsCustomerReq struct {
	ListReq
	Stage   string `json:"stage" form:"stage"`
	OwnerID int    `json:"owner_id" form:"owner_id"`
	Source  string `json:"source" form:"source"`
}

// OpsFollowup 客户跟进记录
type OpsFollowup struct {
	Model
	CustomerID   int    `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	Type         string `json:"type" gorm:"column:type;type:varchar(32);not null;comment:跟进类型"`
	Content      string `json:"content" gorm:"column:content;type:text;not null;comment:内容"`
	NextPlan     string `json:"next_plan" gorm:"column:next_plan;type:varchar(500);comment:下一步计划"`
	OperatorID   int    `json:"operator_id" gorm:"column:operator_id;not null;index;comment:操作人ID"`
	OperatorName string `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
}

func (OpsFollowup) TableName() string { return "cl_ops_followup" }

type CreateOpsFollowupReq struct {
	CustomerID   int    `json:"customer_id" binding:"required,min=1"`
	Type         string `json:"type" binding:"required,min=1,max=32"`
	Content      string `json:"content" binding:"required,min=1"`
	NextPlan     string `json:"next_plan"`
	OperatorID   int    `json:"operator_id"`
	OperatorName string `json:"operator_name"`
}

type ListOpsFollowupReq struct {
	ListReq
	CustomerID int `json:"customer_id" form:"customer_id" binding:"required,min=1"`
}

// OpsCustomerLifecycleWorkorder 客户绑定的运营全流程工单
type OpsCustomerLifecycleWorkorder struct {
	CustomerID          int       `json:"customer_id"`
	CustomerName        string    `json:"customer_name,omitempty"`
	WorkorderInstanceID int       `json:"workorder_instance_id"`
	Title               string    `json:"title"`
	SerialNumber        string    `json:"serial_number,omitempty"`
	Status              string    `json:"status,omitempty"`
	InstanceStatus      int8      `json:"instance_status"`
	CurrentStepID       string    `json:"current_step_id,omitempty"`
	LinkStatus          string    `json:"link_status"`
	CreatedAt           time.Time `json:"created_at,omitempty"`
}
