package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsContractItemDAO interface {
	ListByContract(ctx context.Context, contractID int) ([]*model.OpsContractItem, error)
	Create(ctx context.Context, item *model.OpsContractItem) error
	Delete(ctx context.Context, id int) error
}

type opsContractItemDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsContractItemDAO(db *gorm.DB, logger *zap.Logger) OpsContractItemDAO {
	return &opsContractItemDAO{db: db, logger: logger}
}

func (d *opsContractItemDAO) ListByContract(ctx context.Context, contractID int) ([]*model.OpsContractItem, error) {
	var items []*model.OpsContractItem
	err := d.db.WithContext(ctx).Where("contract_id = ?", contractID).Order("id ASC").Find(&items).Error
	return items, err
}

func (d *opsContractItemDAO) Create(ctx context.Context, item *model.OpsContractItem) error {
	return d.db.WithContext(ctx).Create(item).Error
}

func (d *opsContractItemDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Delete(&model.OpsContractItem{}, id).Error
}

type OpsSurveyDAO interface {
	EnsureDefaults(ctx context.Context) error
	GetByCode(ctx context.Context, code string) (*model.OpsSurvey, error)
	List(ctx context.Context) ([]*model.OpsSurvey, error)
	CreateResponse(ctx context.Context, resp *model.OpsSurveyResponse) error
	ListResponses(ctx context.Context, req *model.ListOpsSurveyResponseReq) ([]*model.OpsSurveyResponse, int64, error)
	HasResponse(ctx context.Context, customerID int, surveyType string) (bool, error)
	ListLowScoreResponses(ctx context.Context, maxScore int, limit int) ([]*model.OpsSurveyResponse, error)
	CreateInvite(ctx context.Context, invite *model.OpsSurveyInvite) error
	GetInviteByToken(ctx context.Context, token string) (*model.OpsSurveyInvite, error)
	MarkInviteUsed(ctx context.Context, id int, usedAt time.Time) error
	ListInvites(ctx context.Context, req *model.ListOpsSurveyInviteReq) ([]*model.OpsSurveyInvite, int64, error)
}

type opsSurveyDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewOpsSurveyDAO(db *gorm.DB, logger *zap.Logger) OpsSurveyDAO {
	return &opsSurveyDAO{db: db, logger: logger}
}

func (d *opsSurveyDAO) EnsureDefaults(ctx context.Context) error {
	defaults := []model.OpsSurvey{
		{
			Code: "formal_csat", Title: "正式过程客户满意度", SurveyType: model.OpsSurveyTypeCSAT,
			Status: model.OpsSurveyStatusActive,
			Questions: model.JSONMap{
				"items": []any{
					map[string]any{"key": "score", "label": "整体满意度（1-5）", "type": "score"},
					map[string]any{"key": "service", "label": "服务情况反馈", "type": "text"},
					map[string]any{"key": "suggestion", "label": "问题与建议", "type": "text"},
				},
			},
		},
		{
			Code: "non_renewal", Title: "不续费原因调研", SurveyType: model.OpsSurveyTypeNonRenewal,
			Status: model.OpsSurveyStatusActive,
			Questions: model.JSONMap{
				"items": []any{
					map[string]any{"key": "reason", "label": "不续费主要原因", "type": "text"},
					map[string]any{"key": "detail", "label": "具体说明", "type": "text"},
					map[string]any{"key": "competitor", "label": "是否转向竞品", "type": "text"},
				},
			},
		},
	}
	for _, item := range defaults {
		var count int64
		if err := d.db.WithContext(ctx).Model(&model.OpsSurvey{}).Where("code = ?", item.Code).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := d.db.WithContext(ctx).Create(&item).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *opsSurveyDAO) GetByCode(ctx context.Context, code string) (*model.OpsSurvey, error) {
	var item model.OpsSurvey
	if err := d.db.WithContext(ctx).Where("code = ?", code).First(&item).Error; err != nil {
		return nil, fmt.Errorf("问卷不存在: %w", err)
	}
	return &item, nil
}

func (d *opsSurveyDAO) List(ctx context.Context) ([]*model.OpsSurvey, error) {
	var items []*model.OpsSurvey
	err := d.db.WithContext(ctx).Order("id ASC").Find(&items).Error
	return items, err
}

func (d *opsSurveyDAO) CreateResponse(ctx context.Context, resp *model.OpsSurveyResponse) error {
	return d.db.WithContext(ctx).Create(resp).Error
}

func (d *opsSurveyDAO) ListResponses(ctx context.Context, req *model.ListOpsSurveyResponseReq) ([]*model.OpsSurveyResponse, int64, error) {
	q := d.db.WithContext(ctx).Model(&model.OpsSurveyResponse{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.SurveyType != "" {
		q = q.Where("survey_type = ?", req.SurveyType)
	}
	var total int64
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
	var items []*model.OpsSurveyResponse
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (d *opsSurveyDAO) HasResponse(ctx context.Context, customerID int, surveyType string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.OpsSurveyResponse{}).
		Where("customer_id = ? AND survey_type = ?", customerID, surveyType).Count(&count).Error
	return count > 0, err
}

func (d *opsSurveyDAO) ListLowScoreResponses(ctx context.Context, maxScore int, limit int) ([]*model.OpsSurveyResponse, error) {
	if maxScore <= 0 {
		maxScore = 3
	}
	if limit <= 0 {
		limit = 20
	}
	var items []*model.OpsSurveyResponse
	err := d.db.WithContext(ctx).Model(&model.OpsSurveyResponse{}).
		Where("survey_type = ? AND score > 0 AND score <= ?", model.OpsSurveyTypeCSAT, maxScore).
		Order("id DESC").Limit(limit).Find(&items).Error
	return items, err
}

func (d *opsSurveyDAO) CreateInvite(ctx context.Context, invite *model.OpsSurveyInvite) error {
	return d.db.WithContext(ctx).Create(invite).Error
}

func (d *opsSurveyDAO) GetInviteByToken(ctx context.Context, token string) (*model.OpsSurveyInvite, error) {
	var item model.OpsSurveyInvite
	if err := d.db.WithContext(ctx).Where("token = ?", token).First(&item).Error; err != nil {
		return nil, fmt.Errorf("问卷链接无效或已失效: %w", err)
	}
	return &item, nil
}

func (d *opsSurveyDAO) MarkInviteUsed(ctx context.Context, id int, usedAt time.Time) error {
	return d.db.WithContext(ctx).Model(&model.OpsSurveyInvite{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":  model.OpsSurveyInviteStatusUsed,
		"used_at": usedAt,
	}).Error
}

func (d *opsSurveyDAO) ListInvites(ctx context.Context, req *model.ListOpsSurveyInviteReq) ([]*model.OpsSurveyInvite, int64, error) {
	q := d.db.WithContext(ctx).Model(&model.OpsSurveyInvite{})
	if req.CustomerID > 0 {
		q = q.Where("customer_id = ?", req.CustomerID)
	}
	if req.SurveyCode != "" {
		q = q.Where("survey_code = ?", req.SurveyCode)
	}
	var total int64
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
	var items []*model.OpsSurveyInvite
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}
