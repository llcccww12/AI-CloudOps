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
	OpsTrialStatusDraft     = "draft"
	OpsTrialStatusPending   = "pending"
	OpsTrialStatusApproved  = "approved"
	OpsTrialStatusRejected  = "rejected"
	OpsTrialStatusActive    = "active"
	OpsTrialStatusEnded     = "ended"

	OpsContractTypeTrial  = "trial"
	OpsContractTypeFormal = "formal"

	OpsContractStatusDraft    = "draft"
	OpsContractStatusActive   = "active"
	OpsContractStatusExpired  = "expired"
	OpsContractStatusTerminated = "terminated"

	OpsActivationStatusDraft    = "draft"
	OpsActivationStatusPending  = "pending"
	OpsActivationStatusApproved = "approved"
	OpsActivationStatusRejected = "rejected"
	OpsActivationStatusActive   = "active"

	OpsOpenMethodTrial   = "trial"   // 测试开通
	OpsOpenMethodFormal  = "formal"  // 正式开通
	OpsOpenMethodExpand  = "expand"  // 扩容开通

	OpsApprovalBizTrial             = "trial"
	OpsApprovalBizActivation        = "activation"
	OpsApprovalBizCustomerLifecycle = "customer_lifecycle"

	OpsApprovalLinkPending   = "pending"
	OpsApprovalLinkApproved  = "approved"
	OpsApprovalLinkRejected  = "rejected"
	OpsApprovalLinkCancelled = "cancelled"
)

// OpsTrial 试用单
type OpsTrial struct {
	Model
	CustomerID     int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	Title          string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:标题"`
	DemandType     string     `json:"demand_type" gorm:"column:demand_type;type:varchar(64);comment:需求类别"`
	ResourceScale  string     `json:"resource_scale" gorm:"column:resource_scale;type:varchar(200);comment:申请算力规模"`
	Purpose        string     `json:"purpose" gorm:"column:purpose;type:text;comment:用途说明"`
	Status         string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:draft;index;comment:状态"`
	PlanStartAt    *time.Time `json:"plan_start_at" gorm:"column:plan_start_at;comment:计划开始"`
	PlanEndAt      *time.Time `json:"plan_end_at" gorm:"column:plan_end_at;index;comment:计划结束"`
	ActualStartAt  *time.Time `json:"actual_start_at" gorm:"column:actual_start_at;comment:实际开始"`
	ActualEndAt    *time.Time `json:"actual_end_at" gorm:"column:actual_end_at;comment:实际结束"`
	Evaluation     string     `json:"evaluation" gorm:"column:evaluation;type:text;comment:试用评价"`
	ConvertIntent  string     `json:"convert_intent" gorm:"column:convert_intent;type:varchar(64);comment:转化意向"`
	// 开通台账（归档列表）
	CustomerShortName string     `json:"customer_short_name" gorm:"column:customer_short_name;type:varchar(100);comment:客户简称"`
	ProductType       string     `json:"product_type" gorm:"column:product_type;type:varchar(64);index;comment:产品类型"`
	Region            string     `json:"region" gorm:"column:region;type:varchar(64);comment:所属大区"`
	OwnerName         string     `json:"owner_name" gorm:"column:owner_name;type:varchar(100);comment:归属客户经理"`
	MainAccount       string     `json:"main_account" gorm:"column:main_account;type:varchar(200);comment:主账号"`
	ProjectName       string     `json:"project_name" gorm:"column:project_name;type:varchar(200);comment:项目名称"`
	OpenMethod        string     `json:"open_method" gorm:"column:open_method;type:varchar(32);default:trial;comment:开通方式"`
	ContractNo        string     `json:"contract_no" gorm:"column:contract_no;type:varchar(100);comment:合同编号"`
	OrderNo           string     `json:"order_no" gorm:"column:order_no;type:varchar(100);index;comment:订单编号"`
	OpenPeriod        string     `json:"open_period" gorm:"column:open_period;type:varchar(100);comment:开通周期"`
	ContractStartAt   *time.Time `json:"contract_start_at" gorm:"column:contract_start_at;comment:合同开始时间"`
	ContractEndAt     *time.Time `json:"contract_end_at" gorm:"column:contract_end_at;comment:合同结束时间"`
	UpdaterID         int        `json:"updater_id" gorm:"column:updater_id;comment:更新人ID"`
	UpdaterName       string     `json:"updater_name" gorm:"column:updater_name;type:varchar(100);comment:更新人"`
	OperatorID          int    `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName        string `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
	WorkorderInstanceID int    `json:"workorder_instance_id,omitempty" gorm:"-"`
}

func (OpsTrial) TableName() string { return "cl_ops_trial" }

type CreateOpsTrialReq struct {
	CustomerID    int        `json:"customer_id" binding:"required,min=1"`
	Title         string     `json:"title" binding:"required,min=1,max=200"`
	DemandType    string     `json:"demand_type"`
	ResourceScale string     `json:"resource_scale"`
	Purpose       string     `json:"purpose"`
	PlanStartAt   *time.Time `json:"plan_start_at"`
	PlanEndAt     *time.Time `json:"plan_end_at"`
	OperatorID    int        `json:"operator_id"`
	OperatorName  string     `json:"operator_name"`
}

type UpdateOpsTrialReq struct {
	ID            int        `json:"id" binding:"required,min=1"`
	Title         string     `json:"title" binding:"required,min=1,max=200"`
	DemandType    string     `json:"demand_type"`
	ResourceScale string     `json:"resource_scale"`
	Purpose       string     `json:"purpose"`
	PlanStartAt   *time.Time `json:"plan_start_at"`
	PlanEndAt     *time.Time `json:"plan_end_at"`
	Evaluation    string     `json:"evaluation"`
	ConvertIntent string     `json:"convert_intent"`
}

type ListOpsTrialReq struct {
	ListReq
	CustomerID int    `json:"customer_id" form:"customer_id"`
	Status     string `json:"status" form:"status"`
}

// OpsContract 合同
type OpsContract struct {
	Model
	CustomerID     int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	TrialID        *int       `json:"trial_id" gorm:"column:trial_id;index;comment:关联试用"`
	Type           string     `json:"type" gorm:"column:type;type:varchar(32);not null;index;comment:合同类型"`
	Title          string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:合同标题"`
	ContractNo     string     `json:"contract_no" gorm:"column:contract_no;type:varchar(100);index;comment:合同编号"`
	ProductType    string     `json:"product_type" gorm:"column:product_type;type:varchar(64);index;comment:算力产品类型"`
	BillingMode    string     `json:"billing_mode" gorm:"column:billing_mode;type:varchar(64);comment:计费模式"`
	UnitPrice      float64    `json:"unit_price" gorm:"column:unit_price;type:decimal(14,2);comment:单价"`
	BillingCycle   string     `json:"billing_cycle" gorm:"column:billing_cycle;type:varchar(32);comment:计费周期"`
	PaymentMethod  string     `json:"payment_method" gorm:"column:payment_method;type:varchar(64);comment:付费方式"`
	PaymentTermDays int       `json:"payment_term_days" gorm:"column:payment_term_days;default:30;comment:账期天数"`
	StartAt        *time.Time `json:"start_at" gorm:"column:start_at;index;comment:开始"`
	EndAt          *time.Time `json:"end_at" gorm:"column:end_at;index;comment:结束"`
	AutoRenew      int8       `json:"auto_renew" gorm:"column:auto_renew;default:2;comment:自动续约1是2否"`
	Status         string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:draft;index;comment:状态"`
	Remark         string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
	OperatorID     int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName   string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
}

func (OpsContract) TableName() string { return "cl_ops_contract" }

type CreateOpsContractReq struct {
	CustomerID      int        `json:"customer_id" binding:"required,min=1"`
	TrialID         *int       `json:"trial_id"`
	Type            string     `json:"type" binding:"required,oneof=trial formal"`
	Title           string     `json:"title" binding:"required,min=1,max=200"`
	ContractNo      string     `json:"contract_no"`
	ProductType     string     `json:"product_type"`
	BillingMode     string     `json:"billing_mode"`
	UnitPrice       float64    `json:"unit_price"`
	BillingCycle    string     `json:"billing_cycle"`
	PaymentMethod   string     `json:"payment_method"`
	PaymentTermDays int        `json:"payment_term_days"`
	StartAt         *time.Time `json:"start_at"`
	EndAt           *time.Time `json:"end_at"`
	AutoRenew       int8       `json:"auto_renew"`
	Remark          string     `json:"remark"`
	OperatorID      int        `json:"operator_id"`
	OperatorName    string     `json:"operator_name"`
}

type UpdateOpsContractReq struct {
	ID              int        `json:"id" binding:"required,min=1"`
	Title           string     `json:"title" binding:"required,min=1,max=200"`
	ContractNo      string     `json:"contract_no"`
	ProductType     string     `json:"product_type"`
	BillingMode     string     `json:"billing_mode"`
	UnitPrice       float64    `json:"unit_price"`
	BillingCycle    string     `json:"billing_cycle"`
	PaymentMethod   string     `json:"payment_method"`
	PaymentTermDays int        `json:"payment_term_days"`
	StartAt         *time.Time `json:"start_at"`
	EndAt           *time.Time `json:"end_at"`
	AutoRenew       int8       `json:"auto_renew"`
	Status          string     `json:"status" binding:"omitempty,oneof=draft active expired terminated"`
	Remark          string     `json:"remark"`
}

type ListOpsContractReq struct {
	ListReq
	CustomerID int    `json:"customer_id" form:"customer_id"`
	Type       string `json:"type" form:"type"`
	Status     string `json:"status" form:"status"`
}

// OpsActivation 开通单（申请字段 + 回馈账号字段 + 开通台账）
type OpsActivation struct {
	Model
	CustomerID      int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	ContractID      int        `json:"contract_id" gorm:"column:contract_id;not null;index;comment:合同ID"`
	Title           string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:标题"`
	ResourceSummary string     `json:"resource_summary" gorm:"column:resource_summary;type:text;comment:资源摘要"`
	Purpose         string     `json:"purpose" gorm:"column:purpose;type:text;comment:用途"`
	FeedbackAccount string     `json:"feedback_account" gorm:"column:feedback_account;type:varchar(200);comment:开通账号"`
	FeedbackTenant  string     `json:"feedback_tenant" gorm:"column:feedback_tenant;type:varchar(200);comment:租户/项目"`
	FeedbackEndpoint string    `json:"feedback_endpoint" gorm:"column:feedback_endpoint;type:varchar(500);comment:访问入口"`
	FeedbackRemark  string     `json:"feedback_remark" gorm:"column:feedback_remark;type:text;comment:开通回馈说明"`
	Status          string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:draft;index;comment:状态"`
	ActivatedAt     *time.Time `json:"activated_at" gorm:"column:activated_at;comment:开通时间"`
	// 开通台账（归档列表）
	CustomerShortName string     `json:"customer_short_name" gorm:"column:customer_short_name;type:varchar(100);comment:客户简称"`
	ProductType       string     `json:"product_type" gorm:"column:product_type;type:varchar(64);index;comment:产品类型"`
	Region            string     `json:"region" gorm:"column:region;type:varchar(64);comment:所属大区"`
	OwnerName         string     `json:"owner_name" gorm:"column:owner_name;type:varchar(100);comment:归属客户经理"`
	MainAccount       string     `json:"main_account" gorm:"column:main_account;type:varchar(200);comment:主账号"`
	ProjectName       string     `json:"project_name" gorm:"column:project_name;type:varchar(200);comment:项目名称"`
	OpenMethod        string     `json:"open_method" gorm:"column:open_method;type:varchar(32);default:formal;comment:开通方式"`
	ContractNo        string     `json:"contract_no" gorm:"column:contract_no;type:varchar(100);comment:合同编号"`
	OrderNo           string     `json:"order_no" gorm:"column:order_no;type:varchar(100);index;comment:订单编号"`
	OpenPeriod        string     `json:"open_period" gorm:"column:open_period;type:varchar(100);comment:开通周期"`
	ContractStartAt   *time.Time `json:"contract_start_at" gorm:"column:contract_start_at;comment:合同开始时间"`
	ContractEndAt     *time.Time `json:"contract_end_at" gorm:"column:contract_end_at;comment:合同结束时间"`
	UpdaterID         int        `json:"updater_id" gorm:"column:updater_id;comment:更新人ID"`
	UpdaterName       string     `json:"updater_name" gorm:"column:updater_name;type:varchar(100);comment:更新人"`
	OperatorID          int    `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName        string `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
	WorkorderInstanceID int    `json:"workorder_instance_id,omitempty" gorm:"-"`
}

func (OpsActivation) TableName() string { return "cl_ops_activation" }

type CreateOpsActivationReq struct {
	CustomerID      int    `json:"customer_id" binding:"required,min=1"`
	ContractID      int    `json:"contract_id" binding:"required,min=1"`
	Title           string `json:"title" binding:"required,min=1,max=200"`
	ResourceSummary string `json:"resource_summary"`
	Purpose         string `json:"purpose"`
	OperatorID      int    `json:"operator_id"`
	OperatorName    string `json:"operator_name"`
}

type UpdateOpsActivationReq struct {
	ID               int    `json:"id" binding:"required,min=1"`
	Title            string `json:"title" binding:"required,min=1,max=200"`
	ResourceSummary  string `json:"resource_summary"`
	Purpose          string `json:"purpose"`
	FeedbackAccount  string `json:"feedback_account"`
	FeedbackTenant   string `json:"feedback_tenant"`
	FeedbackEndpoint string `json:"feedback_endpoint"`
	FeedbackRemark   string `json:"feedback_remark"`
}

type FeedbackOpsActivationReq struct {
	ID               int    `json:"id" binding:"required,min=1"`
	FeedbackAccount  string `json:"feedback_account" binding:"required,min=1,max=200"`
	FeedbackTenant   string `json:"feedback_tenant"`
	FeedbackEndpoint string `json:"feedback_endpoint"`
	FeedbackRemark   string `json:"feedback_remark"`
}

type ListOpsActivationReq struct {
	ListReq
	CustomerID int    `json:"customer_id" form:"customer_id"`
	Status     string `json:"status" form:"status"`
}

// OpsApprovalLink 运营单据与工单绑定
type OpsApprovalLink struct {
	Model
	BizType            string `json:"biz_type" gorm:"column:biz_type;type:varchar(32);not null;index;comment:业务类型"`
	BizID              int    `json:"biz_id" gorm:"column:biz_id;not null;index;comment:业务ID"`
	WorkorderInstanceID int   `json:"workorder_instance_id" gorm:"column:workorder_instance_id;not null;uniqueIndex;comment:工单实例ID"`
	Status             string `json:"status" gorm:"column:status;type:varchar(32);not null;default:pending;index;comment:状态"`
}

func (OpsApprovalLink) TableName() string { return "cl_ops_approval_link" }
