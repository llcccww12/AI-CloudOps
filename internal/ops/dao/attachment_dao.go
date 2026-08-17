package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsAttachmentDAO interface {
	Create(ctx context.Context, item *model.OpsAttachment) error
	GetByID(ctx context.Context, id int) (*model.OpsAttachment, error)
	ListByBiz(ctx context.Context, bizType string, bizID int) ([]*model.OpsAttachment, error)
	CountByBiz(ctx context.Context, bizType string, bizID int) (int64, error)
	Delete(ctx context.Context, id int) error
}

type opsAttachmentDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsAttachmentDAO(db *gorm.DB, logger *zap.Logger) OpsAttachmentDAO {
	return &opsAttachmentDAO{db: db, logger: logger}
}

func (d *opsAttachmentDAO) Create(ctx context.Context, item *model.OpsAttachment) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *opsAttachmentDAO) GetByID(ctx context.Context, id int) (*model.OpsAttachment, error) {
	var item model.OpsAttachment
	if err := d.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *opsAttachmentDAO) ListByBiz(ctx context.Context, bizType string, bizID int) ([]*model.OpsAttachment, error) {
	var items []*model.OpsAttachment
	err := d.db.WithContext(ctx).
		Where("biz_type = ? AND biz_id = ?", bizType, bizID).
		Order("id DESC").
		Find(&items).Error
	return items, err
}

func (d *opsAttachmentDAO) CountByBiz(ctx context.Context, bizType string, bizID int) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.OpsAttachment{}).
		Where("biz_type = ? AND biz_id = ?", bizType, bizID).
		Count(&count).Error
	return count, err
}

func (d *opsAttachmentDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsAttachment{}, id).Error
}
