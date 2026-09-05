/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 */

package model

// 运营全流程节点键（与工单步骤 id / 名称对齐）
const (
	OpsLifecycleStepIntent        = "intent"
	OpsLifecycleStepTrial         = "trial"
	OpsLifecycleStepTrialContract = "trial_contract"
	OpsLifecycleStepOpenRequest   = "open_request"
	OpsLifecycleStepOpenFeedback  = "open_feedback"
	OpsLifecycleStepTrialAccept   = "trial_accept"
	OpsLifecycleStepContract      = "contract"
	OpsLifecycleStepProvision     = "provision" // 测试/资源开通节点：填写开通台账
	OpsLifecycleStepSettlement    = "settlement"
	OpsLifecycleStepInvoice       = "invoice"
	OpsLifecycleStepPayment       = "payment"
)

// OpsLifecycleApproveContext 审批弹窗上下文
type OpsLifecycleApproveContext struct {
	InstanceID        int           `json:"instance_id"`
	CustomerID        int           `json:"customer_id"`
	CustomerName      string        `json:"customer_name"`
	CurrentStepID     string        `json:"current_step_id"`
	CurrentStepName   string        `json:"current_step_name"`
	NodeKey           string        `json:"node_key"`
	NeedsNextAssignee bool          `json:"needs_next_assignee"`
	NextStepName      string        `json:"next_step_name,omitempty"`
	LatestTrial       *OpsTrial     `json:"latest_trial,omitempty"`
	LatestContract    *OpsContract  `json:"latest_contract,omitempty"`
	LatestActivation  *OpsActivation `json:"latest_activation,omitempty"`
	LatestSettlement  *OpsSettlement `json:"latest_settlement,omitempty"`
	FormData          JSONMap       `json:"form_data,omitempty"`
	VendorProfileURL  string        `json:"vendor_profile_url,omitempty"`
	VendorProfileDone int8          `json:"vendor_profile_done,omitempty"`
}

// OpsLifecycleApproveReq 生命周期节点审批（写台账 + 推进工单）
type OpsLifecycleApproveReq struct {
	InstanceID    int     `json:"instance_id" binding:"required,min=1"`
	Comment       string  `json:"comment" binding:"omitempty,max=500"`
	AssigneeID    *int    `json:"assignee_id" binding:"omitempty,min=1"`
	AttachmentIDs []int   `json:"attachment_ids" binding:"omitempty,dive,min=1"`
	Payload       JSONMap `json:"payload" binding:"omitempty"`
}
