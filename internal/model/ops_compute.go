/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 */

package model

import "time"

// 算力资产状态
const (
	OpsComputeAssetAvailable   = "available"
	OpsComputeAssetMaintenance = "maintenance"
	OpsComputeAssetFault       = "fault"
	OpsComputeAssetRetired     = "retired"
)

// 租赁粒度
const (
	OpsComputeLeaseFull     = "full"      // 全机租赁
	OpsComputeLeaseGPUPool  = "gpu_pool"  // 按卡池化
	OpsComputeLeaseTempTest = "temp_test" // 临时测试
)

// 资源来源
const (
	OpsComputeSourceAudit   = "audit"   // 运营审核
	OpsComputeSourceContract = "contract" // 合同签约
	OpsComputeSourceOpen    = "open"    // 开通执行
	OpsComputeSourceRelease = "release" // 释放执行
)

// 生命周期状态（计算字段，不落库手填）
const (
	OpsComputeLifePendingOpen = "pending_open"    // 待开通
	OpsComputeLifeInUse       = "in_use"          // 在用
	OpsComputeLifePendingRel  = "pending_release" // 到期待处理（可续签/转阶段或释放）
	OpsComputeLifeReleased    = "released"        // 已释放
)

// 商业阶段（同一物理占用上的测试→正式→续签）
const (
	OpsComputePhaseTrial  = "trial"  // 测试
	OpsComputePhaseFormal = "formal" // 正式
	OpsComputePhaseRenew  = "renew"  // 续签
)

// 容量校验
const (
	OpsComputeCapacityOK       = "ok"
	OpsComputeCapacityOverbook = "overbook"
)

// OpsComputeAsset 算力服务器资产
type OpsComputeAsset struct {
	Model
	ServerCode       string     `json:"server_code" gorm:"column:server_code;type:varchar(64);not null;uniqueIndex;comment:服务器唯一标识"`
	Name             string     `json:"name" gorm:"column:name;type:varchar(200);not null;comment:设备名称"`
	GPUModel         string     `json:"gpu_model" gorm:"column:gpu_model;type:varchar(64);not null;index;comment:GPU型号"`
	SerialNo         string     `json:"serial_no" gorm:"column:serial_no;type:varchar(100);comment:资产/序列号"`
	RackLocation     string     `json:"rack_location" gorm:"column:rack_location;type:varchar(200);comment:机架/位置"`
	MgmtIP           string     `json:"mgmt_ip" gorm:"column:mgmt_ip;type:varchar(64);comment:管理IP"`
	CommissionedAt   *time.Time `json:"commissioned_at" gorm:"column:commissioned_at;comment:投产日期"`
	GPUCount         *int       `json:"gpu_count" gorm:"column:gpu_count;comment:单机GPU卡数"`
	DefaultLeaseMode string     `json:"default_lease_mode" gorm:"column:default_lease_mode;type:varchar(32);default:full;comment:默认出租方式"`
	Status           string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:available;index;comment:资产状态"`
	TechOwner        string     `json:"tech_owner" gorm:"column:tech_owner;type:varchar(100);comment:技术负责人"`
	NextInspectAt    *time.Time `json:"next_inspect_at" gorm:"column:next_inspect_at;comment:下次巡检日"`
	Remark           string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
	OperatorID       int        `json:"operator_id" gorm:"column:operator_id;comment:操作人ID"`
	OperatorName     string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
}

func (OpsComputeAsset) TableName() string { return "cl_ops_compute_asset" }

type CreateOpsComputeAssetReq struct {
	ServerCode       string     `json:"server_code" binding:"required,min=1,max=64"`
	Name             string     `json:"name" binding:"required,min=1,max=200"`
	GPUModel         string     `json:"gpu_model" binding:"required,min=1,max=64"`
	SerialNo         string     `json:"serial_no"`
	RackLocation     string     `json:"rack_location"`
	MgmtIP           string     `json:"mgmt_ip"`
	CommissionedAt   *time.Time `json:"commissioned_at"`
	GPUCount         *int       `json:"gpu_count"`
	DefaultLeaseMode string     `json:"default_lease_mode" binding:"omitempty,oneof=full gpu_pool temp_test"`
	Status           string     `json:"status" binding:"omitempty,oneof=available maintenance fault retired"`
	TechOwner        string     `json:"tech_owner"`
	NextInspectAt    *time.Time `json:"next_inspect_at"`
	Remark           string     `json:"remark"`
	OperatorID       int        `json:"operator_id"`
	OperatorName     string     `json:"operator_name"`
}

type UpdateOpsComputeAssetReq struct {
	ID               int        `json:"id" binding:"required,min=1"`
	Name             string     `json:"name" binding:"required,min=1,max=200"`
	GPUModel         string     `json:"gpu_model" binding:"required,min=1,max=64"`
	SerialNo         string     `json:"serial_no"`
	RackLocation     string     `json:"rack_location"`
	MgmtIP           string     `json:"mgmt_ip"`
	CommissionedAt   *time.Time `json:"commissioned_at"`
	GPUCount         *int       `json:"gpu_count"`
	DefaultLeaseMode string     `json:"default_lease_mode" binding:"omitempty,oneof=full gpu_pool temp_test"`
	Status           string     `json:"status" binding:"omitempty,oneof=available maintenance fault retired"`
	TechOwner        string     `json:"tech_owner"`
	NextInspectAt    *time.Time `json:"next_inspect_at"`
	Remark           string     `json:"remark"`
}

type ListOpsComputeAssetReq struct {
	ListReq
	GPUModel string `json:"gpu_model" form:"gpu_model"`
	Status   string `json:"status" form:"status"`
}

// OpsComputeAllocation 算力生命周期分配流水
type OpsComputeAllocation struct {
	Model
	RecordNo          string     `json:"record_no" gorm:"column:record_no;type:varchar(64);not null;uniqueIndex;comment:生命周期记录ID"`
	Source            string     `json:"source" gorm:"column:source;type:varchar(32);comment:资源来源"`
	ServerCode        string     `json:"server_code" gorm:"column:server_code;type:varchar(64);not null;index;comment:物理服务器唯一标识"`
	PartitionID       string     `json:"partition_id" gorm:"column:partition_id;type:varchar(100);comment:卡/分区ID"`
	LeaseMode         string     `json:"lease_mode" gorm:"column:lease_mode;type:varchar(32);not null;comment:租赁粒度"`
	AllocatedGPUs     int        `json:"allocated_gpus" gorm:"column:allocated_gpus;not null;default:0;comment:实际分配GPU卡数"`
	PlannedGPUs       int        `json:"planned_gpus" gorm:"column:planned_gpus;default:0;comment:计划分配GPU卡数"`
	CustomerID        int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	CustomerName      string     `json:"customer_name" gorm:"column:customer_name;type:varchar(200);comment:客户名称"`
	ContractID        int        `json:"contract_id" gorm:"column:contract_id;index;comment:合同ID"`
	ContractNo        string     `json:"contract_no" gorm:"column:contract_no;type:varchar(100);index;comment:合同编号"`
	ContractStartAt   *time.Time `json:"contract_start_at" gorm:"column:contract_start_at;comment:合同开始日"`
	ContractEndAt     *time.Time `json:"contract_end_at" gorm:"column:contract_end_at;comment:合同结束日"`
	ActivationID      int        `json:"activation_id" gorm:"column:activation_id;index;comment:关联开通单ID"`
	TrialID           int        `json:"trial_id" gorm:"column:trial_id;index;comment:关联试用单ID"`
	ApplyNo           string     `json:"apply_no" gorm:"column:apply_no;type:varchar(100);comment:开通申请单号"`
	AuditInstanceID   int        `json:"audit_instance_id" gorm:"column:audit_instance_id;index;comment:运营审核流程实例号"`
	AppliedAt         *time.Time `json:"applied_at" gorm:"column:applied_at;comment:申请提交日"`
	ApprovedAt        *time.Time `json:"approved_at" gorm:"column:approved_at;comment:审核完成日"`
	OpenedAt            *time.Time              `json:"opened_at" gorm:"column:opened_at;index;comment:首次实际开通日"`
	CurrentPeriodStartAt *time.Time             `json:"current_period_start_at" gorm:"column:current_period_start_at;comment:当前商业周期开始日"`
	PlanReleaseAt       *time.Time              `json:"plan_release_at" gorm:"column:plan_release_at;index;comment:当前周期截止/计划释放日"`
	ActualReleaseAt     *time.Time              `json:"actual_release_at" gorm:"column:actual_release_at;index;comment:实际释放日"`
	BizPhase            string                  `json:"biz_phase" gorm:"column:biz_phase;type:varchar(32);index;comment:商业阶段 trial/formal/renew"`
	PhaseNote           string                  `json:"phase_note" gorm:"column:phase_note;type:text;comment:阶段变更摘要"`
	PhaseHistory        OpsComputePhaseHistory  `json:"phase_history" gorm:"column:phase_history;type:json;serializer:json;comment:阶段履历"`
	ExecutorName        string                  `json:"executor_name" gorm:"column:executor_name;type:varchar(100);comment:技术运营执行人"`
	ChangeTicketNo      string                  `json:"change_ticket_no" gorm:"column:change_ticket_no;type:varchar(100);comment:变更记录/工单号"`
	Remark              string                  `json:"remark" gorm:"column:remark;type:text;comment:备注"`
	OperatorID          int                     `json:"operator_id" gorm:"column:operator_id;comment:操作人ID"`
	OperatorName        string                  `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`

	// 计算字段
	GPUModel             string `json:"gpu_model,omitempty" gorm:"-"`
	LifeStatus           string `json:"life_status,omitempty" gorm:"-"`
	LifeStatusLabel      string `json:"life_status_label,omitempty" gorm:"-"`
	BizPhaseLabel        string `json:"biz_phase_label,omitempty" gorm:"-"`
	PhaseHistorySummary  string `json:"phase_history_summary,omitempty" gorm:"-"`
	ContractTrailSummary string `json:"contract_trail_summary,omitempty" gorm:"-"` // 履历中的合同链条
	OccupyingGPUs        int    `json:"occupying_gpus,omitempty" gorm:"-"`
	ServerGPUCount       *int   `json:"server_gpu_count,omitempty" gorm:"-"`
	ServerOccupied       int    `json:"server_occupied,omitempty" gorm:"-"`
	CapacityCheck        string `json:"capacity_check,omitempty" gorm:"-"`
	CapacityCheckLabel   string `json:"capacity_check_label,omitempty" gorm:"-"`
	EvidenceOK           bool   `json:"evidence_ok" gorm:"-"`
	SheetCount           int    `json:"sheet_count,omitempty" gorm:"-"`
	EmailCount           int    `json:"email_count,omitempty" gorm:"-"`
}

// OpsComputePhaseSegment 同一占用上的一段商业周期（每段可挂不同合同，也可沿用上一份）
type OpsComputePhaseSegment struct {
	Phase         string     `json:"phase"`
	StartAt       *time.Time `json:"start_at,omitempty"`
	EndAt         *time.Time `json:"end_at,omitempty"`
	ContractID    int        `json:"contract_id,omitempty"`
	ContractNo    string     `json:"contract_no,omitempty"`
	LeaseMode     string     `json:"lease_mode,omitempty"`
	AllocatedGPUs int        `json:"allocated_gpus,omitempty"`
	Note          string     `json:"note,omitempty"`
	ChangedAt     *time.Time `json:"changed_at,omitempty"`
}

// OpsComputePhaseHistory 阶段履历（测试→正式→续签各段）
type OpsComputePhaseHistory []OpsComputePhaseSegment

func (OpsComputeAllocation) TableName() string { return "cl_ops_compute_allocation" }

type CreateOpsComputeAllocationReq struct {
	Source          string     `json:"source" binding:"omitempty,oneof=audit contract open release"`
	ServerCode      string     `json:"server_code" binding:"required,min=1,max=64"`
	PartitionID     string     `json:"partition_id"`
	LeaseMode       string     `json:"lease_mode" binding:"required,oneof=full gpu_pool temp_test"`
	AllocatedGPUs   int        `json:"allocated_gpus" binding:"required,min=1"`
	PlannedGPUs     int        `json:"planned_gpus"`
	CustomerID      int        `json:"customer_id" binding:"required,min=1"`
	ContractID      int        `json:"contract_id"`
	ContractNo      string     `json:"contract_no"`
	ContractStartAt *time.Time `json:"contract_start_at"`
	ContractEndAt   *time.Time `json:"contract_end_at"`
	ActivationID    int        `json:"activation_id"`
	TrialID         int        `json:"trial_id"`
	ApplyNo         string     `json:"apply_no"`
	AuditInstanceID int        `json:"audit_instance_id"`
	AppliedAt       *time.Time `json:"applied_at"`
	ApprovedAt      *time.Time `json:"approved_at"`
	OpenedAt        *time.Time `json:"opened_at"`
	PlanReleaseAt   *time.Time `json:"plan_release_at"`
	BizPhase        string     `json:"biz_phase" binding:"omitempty,oneof=trial formal renew"`
	PhaseNote       string     `json:"phase_note"`
	ExecutorName    string     `json:"executor_name"`
	ChangeTicketNo  string     `json:"change_ticket_no"`
	Remark          string     `json:"remark"`
	OperatorID      int        `json:"operator_id"`
	OperatorName    string     `json:"operator_name"`
}

type UpdateOpsComputeAllocationReq struct {
	ID              int        `json:"id" binding:"required,min=1"`
	Source          string     `json:"source" binding:"omitempty,oneof=audit contract open release"`
	ServerCode      string     `json:"server_code" binding:"required,min=1,max=64"`
	PartitionID     string     `json:"partition_id"`
	LeaseMode       string     `json:"lease_mode" binding:"required,oneof=full gpu_pool temp_test"`
	AllocatedGPUs   int        `json:"allocated_gpus" binding:"required,min=1"`
	PlannedGPUs     int        `json:"planned_gpus"`
	CustomerID      int        `json:"customer_id" binding:"required,min=1"`
	ContractID      int        `json:"contract_id"`
	ContractNo      string     `json:"contract_no"`
	ContractStartAt *time.Time `json:"contract_start_at"`
	ContractEndAt   *time.Time `json:"contract_end_at"`
	ActivationID    int        `json:"activation_id"`
	TrialID         int        `json:"trial_id"`
	ApplyNo         string     `json:"apply_no"`
	AuditInstanceID int        `json:"audit_instance_id"`
	AppliedAt       *time.Time `json:"applied_at"`
	ApprovedAt      *time.Time `json:"approved_at"`
	OpenedAt        *time.Time `json:"opened_at"`
	PlanReleaseAt   *time.Time `json:"plan_release_at"`
	BizPhase        string     `json:"biz_phase" binding:"omitempty,oneof=trial formal renew"`
	PhaseNote       string     `json:"phase_note"`
	ExecutorName    string     `json:"executor_name"`
	ChangeTicketNo  string     `json:"change_ticket_no"`
	Remark          string     `json:"remark"`
}

type ReleaseOpsComputeAllocationReq struct {
	ID              int        `json:"id" binding:"required,min=1"`
	ActualReleaseAt *time.Time `json:"actual_release_at"`
	ChangeTicketNo  string     `json:"change_ticket_no"`
	Remark          string     `json:"remark"`
	OperatorID      int        `json:"-"`
	OperatorName    string     `json:"-"`
}

// ExtendOpsComputeAllocationReq 续期/转阶段（不释放占用）
type ExtendOpsComputeAllocationReq struct {
	ID              int        `json:"id" binding:"required,min=1"`
	BizPhase        string     `json:"biz_phase" binding:"required,oneof=trial formal renew"`
	PeriodStartAt   *time.Time `json:"period_start_at"` // 新阶段开始日，默认今天或上一周期截止次日
	PlanReleaseAt   *time.Time `json:"plan_release_at" binding:"required"`
	ContractID      int        `json:"contract_id"`
	ContractNo      string     `json:"contract_no"`
	ContractStartAt *time.Time `json:"contract_start_at"`
	ContractEndAt   *time.Time `json:"contract_end_at"`
	LeaseMode       string     `json:"lease_mode" binding:"omitempty,oneof=full gpu_pool temp_test"`
	AllocatedGPUs   int        `json:"allocated_gpus"`
	PartitionID     string     `json:"partition_id"`
	ChangeTicketNo  string     `json:"change_ticket_no"`
	Remark          string     `json:"remark"`
	OperatorID      int        `json:"-"`
	OperatorName    string     `json:"-"`
}

type ListOpsComputeAllocationReq struct {
	ListReq
	CustomerID   int    `json:"customer_id" form:"customer_id"`
	ServerCode   string `json:"server_code" form:"server_code"`
	ActivationID int    `json:"activation_id" form:"activation_id"`
	TrialID      int    `json:"trial_id" form:"trial_id"`
	LifeStatus   string `json:"life_status" form:"life_status"`
	GPUModel     string `json:"gpu_model" form:"gpu_model"`
	OverbookOnly bool   `json:"overbook_only" form:"overbook_only"`
}

// OpsComputeDashboard 算力容量看板
type OpsComputeDashboard struct {
	ByModel         []OpsComputeModelStat `json:"by_model"`
	AssetTotal      int64                 `json:"asset_total"`
	InUseRecords    int64                 `json:"in_use_records"`
	PendingOpen     int64                 `json:"pending_open"`
	PendingRelease  int64                 `json:"pending_release"`
	OverbookCount   int64                 `json:"overbook_count"`
	MissingEvidence int64                 `json:"missing_evidence"`
	TotalGPUs       int64                 `json:"total_gpus"`
	UsedGPUs        int64                 `json:"used_gpus"`
	AvailableGPUs   int64                 `json:"available_gpus"`
}

type OpsComputeModelStat struct {
	GPUModel         string `json:"gpu_model"`
	ServerCount      int64  `json:"server_count"`
	TotalGPUs        int64  `json:"total_gpus"`
	UsedGPUs         int64  `json:"used_gpus"`
	AvailableGPUs    int64  `json:"available_gpus"`
	AvailableServers int64  `json:"available_servers"`
	OverbookCount    int64  `json:"overbook_count"`
}
