package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsVendorProfileDAO interface {
	GetByCustomerID(ctx context.Context, customerID int) (*model.OpsVendorProfile, error)
	Upsert(ctx context.Context, p *model.OpsVendorProfile) error
}

type opsVendorProfileDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsVendorProfileDAO(db *gorm.DB, logger *zap.Logger) OpsVendorProfileDAO {
	return &opsVendorProfileDAO{db: db, logger: logger}
}

func (d *opsVendorProfileDAO) GetByCustomerID(ctx context.Context, customerID int) (*model.OpsVendorProfile, error) {
	var p model.OpsVendorProfile
	err := d.db.WithContext(ctx).Where("customer_id = ?", customerID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询客商档案失败: %w", err)
	}
	return &p, nil
}

func (d *opsVendorProfileDAO) Upsert(ctx context.Context, p *model.OpsVendorProfile) error {
	existing, err := d.GetByCustomerID(ctx, p.CustomerID)
	if err != nil {
		return err
	}
	if existing == nil {
		if err := d.db.WithContext(ctx).Create(p).Error; err != nil {
			return fmt.Errorf("创建客商档案失败: %w", err)
		}
		return nil
	}
	p.ID = existing.ID
	result := d.db.WithContext(ctx).Model(&model.OpsVendorProfile{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
		"unit_name":             p.UnitName,
		"credit_code":           p.CreditCode,
		"principal":             p.Principal,
		"contact_address_phone": p.ContactAddressPhone,
		"cnaps_code":            p.CnapsCode,
		"account_name":          p.AccountName,
		"bank_name":             p.BankName,
		"bank_account":          p.BankAccount,
		"bank_province":         p.BankProvince,
		"bank_city":             p.BankCity,
		"phone":                 p.Phone,
		"account_type":          p.AccountType,
		"operator_id":           p.OperatorID,
		"operator_name":         p.OperatorName,
	})
	if result.Error != nil {
		return fmt.Errorf("更新客商档案失败: %w", result.Error)
	}
	return nil
}
