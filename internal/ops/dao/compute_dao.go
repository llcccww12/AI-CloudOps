package dao

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsComputeAssetDAO interface {
	Create(ctx context.Context, a *model.OpsComputeAsset) error
	Update(ctx context.Context, a *model.OpsComputeAsset) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsComputeAsset, error)
	GetByServerCode(ctx context.Context, code string) (*model.OpsComputeAsset, error)
	List(ctx context.Context, req *model.ListOpsComputeAssetReq) ([]*model.OpsComputeAsset, int64, error)
	CountAll(ctx context.Context) (int64, error)
	BatchCreate(ctx context.Context, items []*model.OpsComputeAsset) error
	ListAll(ctx context.Context) ([]*model.OpsComputeAsset, error)
}

type opsComputeAssetDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsComputeAssetDAO(db *gorm.DB, logger *zap.Logger) OpsComputeAssetDAO {
	return &opsComputeAssetDAO{db: db, logger: logger}
}

func (d *opsComputeAssetDAO) Create(ctx context.Context, a *model.OpsComputeAsset) error {
	return d.db.WithContext(ctx).Create(a).Error
}

func (d *opsComputeAssetDAO) BatchCreate(ctx context.Context, items []*model.OpsComputeAsset) error {
	if len(items) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).CreateInBatches(items, 50).Error
}

func (d *opsComputeAssetDAO) Update(ctx context.Context, a *model.OpsComputeAsset) error {
	return d.db.WithContext(ctx).Model(&model.OpsComputeAsset{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
		"name": a.Name, "gpu_model": a.GPUModel, "serial_no": a.SerialNo,
		"rack_location": a.RackLocation, "mgmt_ip": a.MgmtIP, "commissioned_at": a.CommissionedAt,
		"gpu_count": a.GPUCount, "default_lease_mode": a.DefaultLeaseMode, "status": a.Status,
		"tech_owner": a.TechOwner, "next_inspect_at": a.NextInspectAt, "remark": a.Remark,
	}).Error
}

func (d *opsComputeAssetDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsComputeAsset{}, id).Error
}

func (d *opsComputeAssetDAO) GetByID(ctx context.Context, id int) (*model.OpsComputeAsset, error) {
	var a model.OpsComputeAsset
	if err := d.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, fmt.Errorf("算力资产不存在: %w", err)
	}
	return &a, nil
}

func (d *opsComputeAssetDAO) GetByServerCode(ctx context.Context, code string) (*model.OpsComputeAsset, error) {
	var a model.OpsComputeAsset
	if err := d.db.WithContext(ctx).Where("server_code = ?", code).First(&a).Error; err != nil {
		return nil, fmt.Errorf("算力资产不存在: %w", err)
	}
	return &a, nil
}

func (d *opsComputeAssetDAO) List(ctx context.Context, req *model.ListOpsComputeAssetReq) ([]*model.OpsComputeAsset, int64, error) {
	var items []*model.OpsComputeAsset
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsComputeAsset{})
	if req.GPUModel != "" {
		q = q.Where("gpu_model = ?", req.GPUModel)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("server_code LIKE ? OR name LIKE ? OR serial_no LIKE ? OR mgmt_ip LIKE ?", like, like, like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	err := q.Order("server_code ASC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (d *opsComputeAssetDAO) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsComputeAsset{}).Count(&n).Error
	return n, err
}

func (d *opsComputeAssetDAO) ListAll(ctx context.Context) ([]*model.OpsComputeAsset, error) {
	var items []*model.OpsComputeAsset
	err := d.db.WithContext(ctx).Order("server_code ASC").Find(&items).Error
	return items, err
}

type OpsComputeAllocationDAO interface {
	Create(ctx context.Context, a *model.OpsComputeAllocation) error
	Update(ctx context.Context, a *model.OpsComputeAllocation) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsComputeAllocation, error)
	List(ctx context.Context, req *model.ListOpsComputeAllocationReq) ([]*model.OpsComputeAllocation, int64, error)
	ListByServerCode(ctx context.Context, serverCode string) ([]*model.OpsComputeAllocation, error)
	ListActiveLike(ctx context.Context) ([]*model.OpsComputeAllocation, error)
}

type opsComputeAllocationDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsComputeAllocationDAO(db *gorm.DB, logger *zap.Logger) OpsComputeAllocationDAO {
	return &opsComputeAllocationDAO{db: db, logger: logger}
}

func (d *opsComputeAllocationDAO) Create(ctx context.Context, a *model.OpsComputeAllocation) error {
	return d.db.WithContext(ctx).Create(a).Error
}

func (d *opsComputeAllocationDAO) Update(ctx context.Context, a *model.OpsComputeAllocation) error {
	return d.db.WithContext(ctx).Model(&model.OpsComputeAllocation{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
		"source": a.Source, "server_code": a.ServerCode, "partition_id": a.PartitionID,
		"lease_mode": a.LeaseMode, "allocated_gpus": a.AllocatedGPUs, "planned_gpus": a.PlannedGPUs,
		"customer_id": a.CustomerID, "customer_name": a.CustomerName,
		"contract_id": a.ContractID, "contract_no": a.ContractNo,
		"contract_start_at": a.ContractStartAt, "contract_end_at": a.ContractEndAt,
		"activation_id": a.ActivationID, "trial_id": a.TrialID,
		"apply_no": a.ApplyNo, "audit_instance_id": a.AuditInstanceID,
		"applied_at": a.AppliedAt, "approved_at": a.ApprovedAt, "opened_at": a.OpenedAt,
		"plan_release_at": a.PlanReleaseAt, "actual_release_at": a.ActualReleaseAt,
		"current_period_start_at": a.CurrentPeriodStartAt,
		"biz_phase": a.BizPhase, "phase_note": a.PhaseNote, "phase_history": a.PhaseHistory,
		"executor_name": a.ExecutorName, "change_ticket_no": a.ChangeTicketNo, "remark": a.Remark,
	}).Error
}

func (d *opsComputeAllocationDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsComputeAllocation{}, id).Error
}

func (d *opsComputeAllocationDAO) GetByID(ctx context.Context, id int) (*model.OpsComputeAllocation, error) {
	var a model.OpsComputeAllocation
	if err := d.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, fmt.Errorf("分配记录不存在: %w", err)
	}
	return &a, nil
}

func (d *opsComputeAllocationDAO) List(ctx context.Context, req *model.ListOpsComputeAllocationReq) ([]*model.OpsComputeAllocation, int64, error) {
	var items []*model.OpsComputeAllocation
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsComputeAllocation{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.ServerCode != "" {
		q = q.Where("server_code = ?", req.ServerCode)
	}
	if req.ActivationID > 0 {
		q = q.Where("activation_id = ?", req.ActivationID)
	}
	if req.TrialID > 0 {
		q = q.Where("trial_id = ?", req.TrialID)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("record_no LIKE ? OR server_code LIKE ? OR customer_name LIKE ? OR contract_no LIKE ? OR apply_no LIKE ?",
			like, like, like, like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	// 列表后在 service 按状态过滤时可能分页不准；一期先拉当前页再过滤会影响 total。
	// 简化：先查较大集合在 service 过滤时对 life_status/overbook 做内存过滤，并重新分页。
	if req.LifeStatus != "" || req.OverbookOnly || req.GPUModel != "" {
		err := q.Order("id DESC").Find(&items).Error
		return items, int64(len(items)), err
	}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (d *opsComputeAllocationDAO) ListByServerCode(ctx context.Context, serverCode string) ([]*model.OpsComputeAllocation, error) {
	var items []*model.OpsComputeAllocation
	err := d.db.WithContext(ctx).Where("server_code = ?", serverCode).Find(&items).Error
	return items, err
}

func (d *opsComputeAllocationDAO) ListActiveLike(ctx context.Context) ([]*model.OpsComputeAllocation, error) {
	var items []*model.OpsComputeAllocation
	err := d.db.WithContext(ctx).Where("actual_release_at IS NULL").Find(&items).Error
	return items, err
}
