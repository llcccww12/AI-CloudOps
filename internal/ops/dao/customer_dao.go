package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsCustomerDAO interface {
	Create(ctx context.Context, c *model.OpsCustomer) error
	Update(ctx context.Context, c *model.OpsCustomer) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsCustomer, error)
	List(ctx context.Context, req *model.ListOpsCustomerReq) ([]*model.OpsCustomer, int64, error)
	UpdateStage(ctx context.Context, id int, stage, closedReason string) error
	CountByStage(ctx context.Context, stage string) (int64, error)
	CountClosedBetween(ctx context.Context, start, end time.Time) (int64, error)
	ListRecentByStages(ctx context.Context, stages []string, limit int) ([]*model.OpsCustomer, error)
	GetByReportCode(ctx context.Context, code string) (*model.OpsCustomer, error)
	UpdateReportSettings(ctx context.Context, id int, code string, enabled int8) error
	UpdateReportSecret(ctx context.Context, id int, hash string) error
}

type opsCustomerDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsCustomerDAO(db *gorm.DB, logger *zap.Logger) OpsCustomerDAO {
	return &opsCustomerDAO{db: db, logger: logger}
}

func (d *opsCustomerDAO) Create(ctx context.Context, c *model.OpsCustomer) error {
	db := d.db.WithContext(ctx)
	// 未配置组织编码时不写入空字符串，保持 NULL，避免唯一索引冲突
	if strings.TrimSpace(c.ReportCode) == "" {
		db = db.Omit("ReportCode")
	}
	if c.ReportEnabled == 0 {
		c.ReportEnabled = 2
	}
	if err := db.Create(c).Error; err != nil {
		return fmt.Errorf("创建客户失败: %w", err)
	}
	return nil
}

func (d *opsCustomerDAO) Update(ctx context.Context, c *model.OpsCustomer) error {
	result := d.db.WithContext(ctx).Model(&model.OpsCustomer{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
		"name":           c.Name,
		"demand_types":   c.DemandTypes,
		"industry":       c.Industry,
		"contact_name":   c.ContactName,
		"contact_title":  c.ContactTitle,
		"contact_phone":  c.ContactPhone,
		"contact_email":  c.ContactEmail,
		"owner_id":       c.OwnerID,
		"owner_name":     c.OwnerName,
		"budget_range":   c.BudgetRange,
		"next_follow_at": c.NextFollowAt,
		"remark":         c.Remark,
		"vendor_profile_done": c.VendorProfileDone,
	})
	if result.Error != nil {
		return fmt.Errorf("更新客户失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("客户不存在")
	}
	return nil
}

func (d *opsCustomerDAO) Delete(ctx context.Context, id int) error {
	result := d.db.WithContext(ctx).Delete(&model.OpsCustomer{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除客户失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("客户不存在")
	}
	return nil
}

func (d *opsCustomerDAO) GetByID(ctx context.Context, id int) (*model.OpsCustomer, error) {
	var c model.OpsCustomer
	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&c).Error; err != nil {
		return nil, fmt.Errorf("客户不存在: %w", err)
	}
	return &c, nil
}

func (d *opsCustomerDAO) List(ctx context.Context, req *model.ListOpsCustomerReq) ([]*model.OpsCustomer, int64, error) {
	var items []*model.OpsCustomer
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsCustomer{})
	if req.Stage != "" {
		q = q.Where("stage = ?", req.Stage)
	}
	if req.OwnerID > 0 {
		q = q.Where("owner_id = ?", req.OwnerID)
	}
	if req.Source != "" {
		q = q.Where("source = ?", req.Source)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("name LIKE ? OR contact_name LIKE ? OR contact_phone LIKE ?", like, like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := req.Page, req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (d *opsCustomerDAO) UpdateStage(ctx context.Context, id int, stage, closedReason string) error {
	updates := map[string]interface{}{"stage": stage}
	if stage == model.OpsCustomerStageClosed {
		updates["closed_reason"] = closedReason
	}
	result := d.db.WithContext(ctx).Model(&model.OpsCustomer{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("客户不存在")
	}
	return nil
}

func (d *opsCustomerDAO) CountByStage(ctx context.Context, stage string) (int64, error) {
	var n int64
	q := d.db.WithContext(ctx).Model(&model.OpsCustomer{})
	if stage == model.OpsCustomerStageIntent {
		q = q.Where("stage IN ?", []string{model.OpsCustomerStageIntent, model.OpsCustomerStageLead})
	} else {
		q = q.Where("stage = ?", stage)
	}
	err := q.Count(&n).Error
	return n, err
}

func (d *opsCustomerDAO) CountClosedBetween(ctx context.Context, start, end time.Time) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsCustomer{}).
		Where("stage = ?", model.OpsCustomerStageClosed).
		Where("updated_at >= ? AND updated_at < ?", start, end).
		Count(&n).Error
	return n, err
}

func (d *opsCustomerDAO) ListRecentByStages(ctx context.Context, stages []string, limit int) ([]*model.OpsCustomer, error) {
	if limit <= 0 {
		limit = 10
	}
	var items []*model.OpsCustomer
	q := d.db.WithContext(ctx).Model(&model.OpsCustomer{})
	if len(stages) > 0 {
		q = q.Where("stage IN ?", stages)
	}
	err := q.Order("updated_at DESC").Limit(limit).Find(&items).Error
	return items, err
}

func (d *opsCustomerDAO) GetByReportCode(ctx context.Context, code string) (*model.OpsCustomer, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("组织编码无效")
	}
	var c model.OpsCustomer
	if err := d.db.WithContext(ctx).Where("report_code = ?", code).First(&c).Error; err != nil {
		return nil, fmt.Errorf("客户不存在: %w", err)
	}
	return &c, nil
}

func (d *opsCustomerDAO) UpdateReportSettings(ctx context.Context, id int, code string, enabled int8) error {
	code = strings.TrimSpace(code)
	updates := map[string]interface{}{
		"report_enabled": enabled,
	}
	if code == "" {
		updates["report_code"] = nil
	} else {
		updates["report_code"] = code
	}
	result := d.db.WithContext(ctx).Model(&model.OpsCustomer{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新报障配置失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("客户不存在")
	}
	return nil
}

func (d *opsCustomerDAO) UpdateReportSecret(ctx context.Context, id int, hash string) error {
	now := time.Now()
	result := d.db.WithContext(ctx).Model(&model.OpsCustomer{}).Where("id = ?", id).Updates(map[string]interface{}{
		"report_secret_hash":       hash,
		"report_secret_updated_at": now,
	})
	if result.Error != nil {
		return fmt.Errorf("更新报障密钥失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("客户不存在")
	}
	return nil
}

type OpsFollowupDAO interface {
	Create(ctx context.Context, f *model.OpsFollowup) error
	ListByCustomer(ctx context.Context, req *model.ListOpsFollowupReq) ([]*model.OpsFollowup, int64, error)
}

type opsFollowupDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsFollowupDAO(db *gorm.DB, logger *zap.Logger) OpsFollowupDAO {
	return &opsFollowupDAO{db: db, logger: logger}
}

func (d *opsFollowupDAO) Create(ctx context.Context, f *model.OpsFollowup) error {
	return d.db.WithContext(ctx).Create(f).Error
}

func (d *opsFollowupDAO) ListByCustomer(ctx context.Context, req *model.ListOpsFollowupReq) ([]*model.OpsFollowup, int64, error) {
	var items []*model.OpsFollowup
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsFollowup{}).Where("customer_id = ?", req.CustomerID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := req.Page, req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
