package dao

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsTrialDAO interface {
	Create(ctx context.Context, t *model.OpsTrial) error
	Update(ctx context.Context, t *model.OpsTrial) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsTrial, error)
	List(ctx context.Context, req *model.ListOpsTrialReq) ([]*model.OpsTrial, int64, error)
	UpdateStatus(ctx context.Context, id int, status string) error
	ListExpiring(ctx context.Context, withinDays int) ([]*model.OpsTrial, error)
}

type opsTrialDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsTrialDAO(db *gorm.DB, logger *zap.Logger) OpsTrialDAO {
	return &opsTrialDAO{db: db, logger: logger}
}

func (d *opsTrialDAO) Create(ctx context.Context, t *model.OpsTrial) error {
	return d.db.WithContext(ctx).Create(t).Error
}
func (d *opsTrialDAO) Update(ctx context.Context, t *model.OpsTrial) error {
	return d.db.WithContext(ctx).Model(&model.OpsTrial{}).Where("id = ?", t.ID).Updates(map[string]interface{}{
		"title": t.Title, "demand_type": t.DemandType, "resource_scale": t.ResourceScale,
		"purpose": t.Purpose, "plan_start_at": t.PlanStartAt, "plan_end_at": t.PlanEndAt,
		"evaluation": t.Evaluation, "convert_intent": t.ConvertIntent,
	}).Error
}
func (d *opsTrialDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsTrial{}, id).Error
}
func (d *opsTrialDAO) GetByID(ctx context.Context, id int) (*model.OpsTrial, error) {
	var t model.OpsTrial
	if err := d.db.WithContext(ctx).First(&t, id).Error; err != nil {
		return nil, fmt.Errorf("试用单不存在: %w", err)
	}
	return &t, nil
}
func (d *opsTrialDAO) List(ctx context.Context, req *model.ListOpsTrialReq) ([]*model.OpsTrial, int64, error) {
	var items []*model.OpsTrial
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsTrial{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		q = q.Where("title LIKE ?", "%"+req.Search+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}
func (d *opsTrialDAO) UpdateStatus(ctx context.Context, id int, status string) error {
	return d.db.WithContext(ctx).Model(&model.OpsTrial{}).Where("id = ?", id).Update("status", status).Error
}
func (d *opsTrialDAO) ListExpiring(ctx context.Context, withinDays int) ([]*model.OpsTrial, error) {
	var items []*model.OpsTrial
	err := d.db.WithContext(ctx).Where("status IN ? AND plan_end_at IS NOT NULL AND plan_end_at BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL ? DAY)",
		[]string{model.OpsTrialStatusActive, model.OpsTrialStatusApproved}, withinDays).Find(&items).Error
	return items, err
}

type OpsContractDAO interface {
	Create(ctx context.Context, c *model.OpsContract) error
	Update(ctx context.Context, c *model.OpsContract) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsContract, error)
	List(ctx context.Context, req *model.ListOpsContractReq) ([]*model.OpsContract, int64, error)
	ListExpiring(ctx context.Context, withinDays int) ([]*model.OpsContract, error)
	UpdateStatus(ctx context.Context, id int, status string) error
}

type opsContractDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsContractDAO(db *gorm.DB, logger *zap.Logger) OpsContractDAO {
	return &opsContractDAO{db: db, logger: logger}
}

func (d *opsContractDAO) Create(ctx context.Context, c *model.OpsContract) error {
	return d.db.WithContext(ctx).Create(c).Error
}
func (d *opsContractDAO) Update(ctx context.Context, c *model.OpsContract) error {
	return d.db.WithContext(ctx).Model(&model.OpsContract{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
		"title": c.Title, "billing_mode": c.BillingMode, "unit_price": c.UnitPrice,
		"billing_cycle": c.BillingCycle, "payment_term_days": c.PaymentTermDays,
		"start_at": c.StartAt, "end_at": c.EndAt, "auto_renew": c.AutoRenew,
		"status": c.Status, "remark": c.Remark,
	}).Error
}
func (d *opsContractDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsContract{}, id).Error
}
func (d *opsContractDAO) GetByID(ctx context.Context, id int) (*model.OpsContract, error) {
	var c model.OpsContract
	if err := d.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, fmt.Errorf("合同不存在: %w", err)
	}
	return &c, nil
}
func (d *opsContractDAO) List(ctx context.Context, req *model.ListOpsContractReq) ([]*model.OpsContract, int64, error) {
	var items []*model.OpsContract
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsContract{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.Type != "" {
		q = q.Where("type = ?", req.Type)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		q = q.Where("title LIKE ?", "%"+req.Search+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}
func (d *opsContractDAO) ListExpiring(ctx context.Context, withinDays int) ([]*model.OpsContract, error) {
	var items []*model.OpsContract
	err := d.db.WithContext(ctx).Where("status = ? AND end_at IS NOT NULL AND end_at BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL ? DAY)",
		model.OpsContractStatusActive, withinDays).Find(&items).Error
	return items, err
}
func (d *opsContractDAO) UpdateStatus(ctx context.Context, id int, status string) error {
	return d.db.WithContext(ctx).Model(&model.OpsContract{}).Where("id = ?", id).Update("status", status).Error
}

type OpsActivationDAO interface {
	Create(ctx context.Context, a *model.OpsActivation) error
	Update(ctx context.Context, a *model.OpsActivation) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsActivation, error)
	List(ctx context.Context, req *model.ListOpsActivationReq) ([]*model.OpsActivation, int64, error)
	UpdateStatus(ctx context.Context, id int, status string) error
}

type opsActivationDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsActivationDAO(db *gorm.DB, logger *zap.Logger) OpsActivationDAO {
	return &opsActivationDAO{db: db, logger: logger}
}

func (d *opsActivationDAO) Create(ctx context.Context, a *model.OpsActivation) error {
	return d.db.WithContext(ctx).Create(a).Error
}
func (d *opsActivationDAO) Update(ctx context.Context, a *model.OpsActivation) error {
	return d.db.WithContext(ctx).Model(&model.OpsActivation{}).Where("id = ?", a.ID).Updates(map[string]interface{}{
		"title": a.Title, "resource_summary": a.ResourceSummary, "purpose": a.Purpose,
	}).Error
}
func (d *opsActivationDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsActivation{}, id).Error
}
func (d *opsActivationDAO) GetByID(ctx context.Context, id int) (*model.OpsActivation, error) {
	var a model.OpsActivation
	if err := d.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, fmt.Errorf("开通单不存在: %w", err)
	}
	return &a, nil
}
func (d *opsActivationDAO) List(ctx context.Context, req *model.ListOpsActivationReq) ([]*model.OpsActivation, int64, error) {
	var items []*model.OpsActivation
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsActivation{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}
func (d *opsActivationDAO) UpdateStatus(ctx context.Context, id int, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == model.OpsActivationStatusActive || status == model.OpsActivationStatusApproved {
		updates["activated_at"] = gorm.Expr("NOW()")
	}
	return d.db.WithContext(ctx).Model(&model.OpsActivation{}).Where("id = ?", id).Updates(updates).Error
}

type OpsApprovalLinkDAO interface {
	Create(ctx context.Context, link *model.OpsApprovalLink) error
	GetByInstanceID(ctx context.Context, instanceID int) (*model.OpsApprovalLink, error)
	GetByBiz(ctx context.Context, bizType string, bizID int) (*model.OpsApprovalLink, error)
	ListByBiz(ctx context.Context, bizType string, bizID int) ([]*model.OpsApprovalLink, error)
	UpdateStatus(ctx context.Context, id int, status string) error
	ListPending(ctx context.Context) ([]*model.OpsApprovalLink, error)
}

type opsApprovalLinkDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsApprovalLinkDAO(db *gorm.DB, logger *zap.Logger) OpsApprovalLinkDAO {
	return &opsApprovalLinkDAO{db: db, logger: logger}
}

func (d *opsApprovalLinkDAO) Create(ctx context.Context, link *model.OpsApprovalLink) error {
	return d.db.WithContext(ctx).Create(link).Error
}
func (d *opsApprovalLinkDAO) GetByInstanceID(ctx context.Context, instanceID int) (*model.OpsApprovalLink, error) {
	var link model.OpsApprovalLink
	if err := d.db.WithContext(ctx).Where("workorder_instance_id = ?", instanceID).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (d *opsApprovalLinkDAO) GetByBiz(ctx context.Context, bizType string, bizID int) (*model.OpsApprovalLink, error) {
	var link model.OpsApprovalLink
	if err := d.db.WithContext(ctx).
		Where("biz_type = ? AND biz_id = ?", bizType, bizID).
		Order("id DESC").
		First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

func (d *opsApprovalLinkDAO) ListByBiz(ctx context.Context, bizType string, bizID int) ([]*model.OpsApprovalLink, error) {
	var items []*model.OpsApprovalLink
	err := d.db.WithContext(ctx).
		Where("biz_type = ? AND biz_id = ?", bizType, bizID).
		Order("id DESC").
		Find(&items).Error
	return items, err
}
func (d *opsApprovalLinkDAO) UpdateStatus(ctx context.Context, id int, status string) error {
	return d.db.WithContext(ctx).Model(&model.OpsApprovalLink{}).Where("id = ?", id).Update("status", status).Error
}
func (d *opsApprovalLinkDAO) ListPending(ctx context.Context) ([]*model.OpsApprovalLink, error) {
	var items []*model.OpsApprovalLink
	err := d.db.WithContext(ctx).Where("status = ?", model.OpsApprovalLinkPending).Find(&items).Error
	return items, err
}
