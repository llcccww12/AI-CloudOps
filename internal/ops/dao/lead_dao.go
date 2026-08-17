package dao

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsExhibitionDAO interface {
	Create(ctx context.Context, e *model.OpsExhibition) error
	Update(ctx context.Context, e *model.OpsExhibition) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsExhibition, error)
	List(ctx context.Context, req *model.ListOpsExhibitionReq) ([]*model.OpsExhibition, int64, error)
	MarkConverted(ctx context.Context, id, customerID int) error
}

type opsExhibitionDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsExhibitionDAO(db *gorm.DB, logger *zap.Logger) OpsExhibitionDAO {
	return &opsExhibitionDAO{db: db, logger: logger}
}

func (d *opsExhibitionDAO) Create(ctx context.Context, e *model.OpsExhibition) error {
	return d.db.WithContext(ctx).Create(e).Error
}

func (d *opsExhibitionDAO) Update(ctx context.Context, e *model.OpsExhibition) error {
	result := d.db.WithContext(ctx).Model(&model.OpsExhibition{}).Where("id = ?", e.ID).Updates(map[string]interface{}{
		"company_name":  e.CompanyName,
		"visitor_name":  e.VisitorName,
		"visitor_title": e.VisitorTitle,
		"visit_at":      e.VisitAt,
		"host_name":     e.HostName,
		"purpose":       e.Purpose,
		"focus_tags":    e.FocusTags,
		"content":       e.Content,
		"companions":    e.Companions,
		"remark":        e.Remark,
		"customer_id":   e.CustomerID,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("展厅记录不存在")
	}
	return nil
}

func (d *opsExhibitionDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsExhibition{}, id).Error
}

func (d *opsExhibitionDAO) GetByID(ctx context.Context, id int) (*model.OpsExhibition, error) {
	var e model.OpsExhibition
	if err := d.db.WithContext(ctx).First(&e, id).Error; err != nil {
		return nil, fmt.Errorf("展厅记录不存在: %w", err)
	}
	return &e, nil
}

func (d *opsExhibitionDAO) List(ctx context.Context, req *model.ListOpsExhibitionReq) ([]*model.OpsExhibition, int64, error) {
	var items []*model.OpsExhibition
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsExhibition{})
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("company_name LIKE ? OR visitor_name LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (d *opsExhibitionDAO) MarkConverted(ctx context.Context, id, customerID int) error {
	return d.db.WithContext(ctx).Model(&model.OpsExhibition{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      model.OpsExhibitionStatusConverted,
		"customer_id": customerID,
	}).Error
}

type OpsVisitDAO interface {
	Create(ctx context.Context, v *model.OpsVisit) error
	Update(ctx context.Context, v *model.OpsVisit) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsVisit, error)
	List(ctx context.Context, req *model.ListOpsVisitReq) ([]*model.OpsVisit, int64, error)
	MarkConverted(ctx context.Context, id, customerID int) error
}

type opsVisitDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsVisitDAO(db *gorm.DB, logger *zap.Logger) OpsVisitDAO {
	return &opsVisitDAO{db: db, logger: logger}
}

func (d *opsVisitDAO) Create(ctx context.Context, v *model.OpsVisit) error {
	return d.db.WithContext(ctx).Create(v).Error
}

func (d *opsVisitDAO) Update(ctx context.Context, v *model.OpsVisit) error {
	result := d.db.WithContext(ctx).Model(&model.OpsVisit{}).Where("id = ?", v.ID).Updates(map[string]interface{}{
		"title":        v.Title,
		"target_org":   v.TargetOrg,
		"start_at":     v.StartAt,
		"end_at":       v.EndAt,
		"location":     v.Location,
		"participants": v.Participants,
		"summary":      v.Summary,
		"outcome":      v.Outcome,
		"status":       v.Status,
		"remark":       v.Remark,
		"customer_id":  v.CustomerID,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("外访记录不存在")
	}
	return nil
}

func (d *opsVisitDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsVisit{}, id).Error
}

func (d *opsVisitDAO) GetByID(ctx context.Context, id int) (*model.OpsVisit, error) {
	var v model.OpsVisit
	if err := d.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, fmt.Errorf("外访记录不存在: %w", err)
	}
	return &v, nil
}

func (d *opsVisitDAO) List(ctx context.Context, req *model.ListOpsVisitReq) ([]*model.OpsVisit, int64, error) {
	var items []*model.OpsVisit
	var total int64
	q := d.db.WithContext(ctx).Model(&model.OpsVisit{})
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("title LIKE ? OR target_org LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := normalizePage(req.Page, req.Size)
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (d *opsVisitDAO) MarkConverted(ctx context.Context, id, customerID int) error {
	return d.db.WithContext(ctx).Model(&model.OpsVisit{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      model.OpsVisitStatusConverted,
		"customer_id": customerID,
	}).Error
}

func normalizePage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	return page, size
}
