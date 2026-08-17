package dao

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsSettlementDAO interface {
	Create(ctx context.Context, s *model.OpsSettlement) error
	Update(ctx context.Context, s *model.OpsSettlement) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsSettlement, error)
	List(ctx context.Context, req *model.ListOpsSettlementReq) ([]*model.OpsSettlement, int64, error)
	UpdateStatus(ctx context.Context, id int, status string) error
	ListDueSoon(ctx context.Context, withinDays int) ([]*model.OpsSettlement, error)
	ListOverdue(ctx context.Context) ([]*model.OpsSettlement, error)
}

type opsSettlementDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsSettlementDAO(db *gorm.DB, logger *zap.Logger) OpsSettlementDAO {
	return &opsSettlementDAO{db: db, logger: logger}
}

func (d *opsSettlementDAO) Create(ctx context.Context, s *model.OpsSettlement) error {
	return d.db.WithContext(ctx).Create(s).Error
}
func (d *opsSettlementDAO) Update(ctx context.Context, s *model.OpsSettlement) error {
	return d.db.WithContext(ctx).Model(&model.OpsSettlement{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
		"title": s.Title, "period_start": s.PeriodStart, "period_end": s.PeriodEnd,
		"amount": s.Amount, "due_at": s.DueAt, "status": s.Status, "remark": s.Remark,
	}).Error
}
func (d *opsSettlementDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsSettlement{}, id).Error
}
func (d *opsSettlementDAO) GetByID(ctx context.Context, id int) (*model.OpsSettlement, error) {
	var s model.OpsSettlement
	if err := d.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, fmt.Errorf("结算单不存在: %w", err)
	}
	return &s, nil
}
func (d *opsSettlementDAO) List(ctx context.Context, req *model.ListOpsSettlementReq) ([]*model.OpsSettlement, int64, error) {
	var items []*model.OpsSettlement
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsSettlement{})
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
func (d *opsSettlementDAO) UpdateStatus(ctx context.Context, id int, status string) error {
	return d.db.WithContext(ctx).Model(&model.OpsSettlement{}).Where("id = ?", id).Update("status", status).Error
}
func (d *opsSettlementDAO) ListDueSoon(ctx context.Context, withinDays int) ([]*model.OpsSettlement, error) {
	var items []*model.OpsSettlement
	err := d.db.WithContext(ctx).Where("status IN ? AND due_at IS NOT NULL AND due_at BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL ? DAY)",
		[]string{model.OpsSettlementStatusConfirmed, model.OpsSettlementStatusInvoiced}, withinDays).Find(&items).Error
	return items, err
}
func (d *opsSettlementDAO) ListOverdue(ctx context.Context) ([]*model.OpsSettlement, error) {
	var items []*model.OpsSettlement
	err := d.db.WithContext(ctx).Where("status IN ? AND due_at IS NOT NULL AND due_at < NOW()",
		[]string{model.OpsSettlementStatusConfirmed, model.OpsSettlementStatusInvoiced, model.OpsSettlementStatusOverdue}).Find(&items).Error
	return items, err
}

type OpsInvoiceDAO interface {
	Create(ctx context.Context, inv *model.OpsInvoice) error
	Update(ctx context.Context, inv *model.OpsInvoice) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsInvoice, error)
	List(ctx context.Context, req *model.ListOpsInvoiceReq) ([]*model.OpsInvoice, int64, error)
}

type opsInvoiceDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsInvoiceDAO(db *gorm.DB, logger *zap.Logger) OpsInvoiceDAO {
	return &opsInvoiceDAO{db: db, logger: logger}
}

func (d *opsInvoiceDAO) Create(ctx context.Context, inv *model.OpsInvoice) error {
	return d.db.WithContext(ctx).Create(inv).Error
}
func (d *opsInvoiceDAO) Update(ctx context.Context, inv *model.OpsInvoice) error {
	return d.db.WithContext(ctx).Model(&model.OpsInvoice{}).Where("id = ?", inv.ID).Updates(map[string]interface{}{
		"invoice_no": inv.InvoiceNo, "invoice_type": inv.InvoiceType, "amount": inv.Amount,
		"issued_at": inv.IssuedAt, "status": inv.Status,
	}).Error
}
func (d *opsInvoiceDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsInvoice{}, id).Error
}
func (d *opsInvoiceDAO) GetByID(ctx context.Context, id int) (*model.OpsInvoice, error) {
	var inv model.OpsInvoice
	if err := d.db.WithContext(ctx).First(&inv, id).Error; err != nil {
		return nil, fmt.Errorf("发票不存在: %w", err)
	}
	return &inv, nil
}
func (d *opsInvoiceDAO) List(ctx context.Context, req *model.ListOpsInvoiceReq) ([]*model.OpsInvoice, int64, error) {
	var items []*model.OpsInvoice
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsInvoice{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.SettlementID > 0 {
		q = q.Where("settlement_id = ?", req.SettlementID)
	}
	if req.Search != "" {
		q = q.Where("invoice_no LIKE ?", "%"+req.Search+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

type OpsPaymentDAO interface {
	Create(ctx context.Context, p *model.OpsPayment) error
	Update(ctx context.Context, p *model.OpsPayment) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsPayment, error)
	List(ctx context.Context, req *model.ListOpsPaymentReq) ([]*model.OpsPayment, int64, error)
}

type opsPaymentDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsPaymentDAO(db *gorm.DB, logger *zap.Logger) OpsPaymentDAO {
	return &opsPaymentDAO{db: db, logger: logger}
}

func (d *opsPaymentDAO) Create(ctx context.Context, p *model.OpsPayment) error {
	return d.db.WithContext(ctx).Create(p).Error
}
func (d *opsPaymentDAO) Update(ctx context.Context, p *model.OpsPayment) error {
	return d.db.WithContext(ctx).Model(&model.OpsPayment{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"amount": p.Amount, "paid_at": p.PaidAt, "bank_ref": p.BankRef, "status": p.Status, "remark": p.Remark,
	}).Error
}
func (d *opsPaymentDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsPayment{}, id).Error
}
func (d *opsPaymentDAO) GetByID(ctx context.Context, id int) (*model.OpsPayment, error) {
	var p model.OpsPayment
	if err := d.db.WithContext(ctx).First(&p, id).Error; err != nil {
		return nil, fmt.Errorf("回款记录不存在: %w", err)
	}
	return &p, nil
}
func (d *opsPaymentDAO) List(ctx context.Context, req *model.ListOpsPaymentReq) ([]*model.OpsPayment, int64, error) {
	var items []*model.OpsPayment
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsPayment{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.SettlementID > 0 {
		q = q.Where("settlement_id = ?", req.SettlementID)
	}
	if req.Search != "" {
		q = q.Where("bank_ref LIKE ? OR remark LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

type OpsReminderRuleDAO interface {
	List(ctx context.Context) ([]*model.OpsReminderRule, error)
	GetByID(ctx context.Context, id int) (*model.OpsReminderRule, error)
	GetByScene(ctx context.Context, scene string) (*model.OpsReminderRule, error)
	Create(ctx context.Context, rule *model.OpsReminderRule) error
	Update(ctx context.Context, rule *model.OpsReminderRule) error
	EnsureDefaults(ctx context.Context) error
}

type opsReminderRuleDAO struct{ db *gorm.DB; logger *zap.Logger }

func NewOpsReminderRuleDAO(db *gorm.DB, logger *zap.Logger) OpsReminderRuleDAO {
	return &opsReminderRuleDAO{db: db, logger: logger}
}

func (d *opsReminderRuleDAO) List(ctx context.Context) ([]*model.OpsReminderRule, error) {
	var items []*model.OpsReminderRule
	err := d.db.WithContext(ctx).Order("id ASC").Find(&items).Error
	return items, err
}
func (d *opsReminderRuleDAO) GetByID(ctx context.Context, id int) (*model.OpsReminderRule, error) {
	var rule model.OpsReminderRule
	if err := d.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}
func (d *opsReminderRuleDAO) GetByScene(ctx context.Context, scene string) (*model.OpsReminderRule, error) {
	var rule model.OpsReminderRule
	if err := d.db.WithContext(ctx).Where("scene = ?", scene).First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}
func (d *opsReminderRuleDAO) Create(ctx context.Context, rule *model.OpsReminderRule) error {
	return d.db.WithContext(ctx).Create(rule).Error
}

func (d *opsReminderRuleDAO) Update(ctx context.Context, rule *model.OpsReminderRule) error {
	return d.db.WithContext(ctx).Model(&model.OpsReminderRule{}).Where("id = ?", rule.ID).Updates(map[string]interface{}{
		"name": rule.Name, "advance_days": rule.AdvanceDays, "enabled": rule.Enabled,
		"channels": rule.Channels, "remark": rule.Remark,
	}).Error
}
func (d *opsReminderRuleDAO) EnsureDefaults(ctx context.Context) error {
	// 允许同一场景多条规则（不同提前天数），尽量去掉历史唯一索引
	_ = d.db.WithContext(ctx).Exec("ALTER TABLE cl_ops_reminder_rule DROP INDEX uni_cl_ops_reminder_rule_scene").Error
	_ = d.db.WithContext(ctx).Exec("ALTER TABLE cl_ops_reminder_rule DROP INDEX idx_cl_ops_reminder_rule_scene").Error

	defaults := []model.OpsReminderRule{
		{Scene: model.OpsReminderSceneTrialExpire, Name: "试用到期提醒", AdvanceDays: 7, Enabled: 1, Channels: model.StringList{"inbox"}},
		{Scene: model.OpsReminderSceneContractRenew, Name: "合同续约提醒", AdvanceDays: 30, Enabled: 1, Channels: model.StringList{"inbox"}},
		{Scene: model.OpsReminderSceneSettlementDue, Name: "结算待确认提醒", AdvanceDays: 3, Enabled: 1, Channels: model.StringList{"inbox"}},
		{Scene: model.OpsReminderSceneInvoice, Name: "开票提醒", AdvanceDays: 0, Enabled: 1, Channels: model.StringList{"inbox"}},
		{Scene: model.OpsReminderScenePaymentFollowup, Name: "收款跟进提醒", AdvanceDays: 5, Enabled: 1, Channels: model.StringList{"inbox"}},
	}
	for _, item := range defaults {
		var count int64
		if err := d.db.WithContext(ctx).Model(&model.OpsReminderRule{}).Where("scene = ?", item.Scene).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := d.db.WithContext(ctx).Create(&item).Error; err != nil {
				return err
			}
		}
	}
	// 普通索引便于按场景扫描
	_ = d.db.WithContext(ctx).Exec("CREATE INDEX idx_cl_ops_reminder_rule_scene ON cl_ops_reminder_rule(scene)").Error
	return nil
}
