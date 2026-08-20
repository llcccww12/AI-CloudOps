package dao

import (
	"context"
	"fmt"
	"time"

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
	MarkTransferred(ctx context.Context, id, visitID int) error
	CountCreatedBetween(ctx context.Context, start, end time.Time) (int64, error)
	CountAll(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status string) (int64, error)
	CountDailyCreated(ctx context.Context, start, end time.Time) ([]model.OpsDashboardTrendPoint, error)
	GroupByIntent(ctx context.Context) ([]model.OpsDashboardIntentItem, error)
	ListRecentTransferred(ctx context.Context, limit int) ([]*model.OpsExhibition, error)
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
		"company_name":    e.CompanyName,
		"company_level":   e.CompanyLevel,
		"visitor_name":    e.VisitorName,
		"visitor_title":   e.VisitorTitle,
		"visitor_count":   e.VisitorCount,
		"visit_at":        e.VisitAt,
		"docking_unit":    e.DockingUnit,
		"host_name":       e.HostName,
		"purpose":         e.Purpose,
		"need_meeting":    e.NeedMeeting,
		"focus_tags":      e.FocusTags,
		"content":         e.Content,
		"meeting_minutes": e.MeetingMinutes,
		"contact_phone":   e.ContactPhone,
		"companions":      e.Companions,
		"intent":          e.Intent,
		"remark":          e.Remark,
		"customer_id":     e.CustomerID,
		"updater_id":      e.UpdaterID,
		"updater_name":    e.UpdaterName,
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
	if req.Source != "" {
		q = q.Where("source = ?", req.Source)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("company_name LIKE ? OR visitor_name LIKE ? OR contact_phone LIKE ?", like, like, like)
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

func (d *opsExhibitionDAO) MarkTransferred(ctx context.Context, id, visitID int) error {
	return d.db.WithContext(ctx).Model(&model.OpsExhibition{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":   model.OpsExhibitionStatusTransferred,
		"visit_id": visitID,
	}).Error
}

func (d *opsExhibitionDAO) CountCreatedBetween(ctx context.Context, start, end time.Time) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsExhibition{}).
		Where("created_at >= ? AND created_at < ?", start, end).Count(&n).Error
	return n, err
}

func (d *opsExhibitionDAO) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsExhibition{}).Count(&n).Error
	return n, err
}

func (d *opsExhibitionDAO) CountByStatus(ctx context.Context, status string) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsExhibition{}).Where("status = ?", status).Count(&n).Error
	return n, err
}

func (d *opsExhibitionDAO) CountDailyCreated(ctx context.Context, start, end time.Time) ([]model.OpsDashboardTrendPoint, error) {
	type row struct {
		Day   string
		Count int64
	}
	var rows []row
	err := d.db.WithContext(ctx).Model(&model.OpsExhibition{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') as day, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", start, end).
		Group("day").Order("day ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]model.OpsDashboardTrendPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, model.OpsDashboardTrendPoint{Date: r.Day, Count: r.Count})
	}
	return out, nil
}

func (d *opsExhibitionDAO) GroupByIntent(ctx context.Context) ([]model.OpsDashboardIntentItem, error) {
	type row struct {
		Intent string
		Count  int64
	}
	var rows []row
	err := d.db.WithContext(ctx).Model(&model.OpsExhibition{}).
		Select("COALESCE(NULLIF(intent,''), 'unknown') as intent, COUNT(*) as count").
		Group("intent").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	label := map[string]string{"low": "低", "medium": "中", "high": "高", "unknown": "未评"}
	out := make([]model.OpsDashboardIntentItem, 0, len(rows))
	for _, r := range rows {
		l := label[r.Intent]
		if l == "" {
			l = r.Intent
		}
		out = append(out, model.OpsDashboardIntentItem{Intent: r.Intent, Label: l, Count: r.Count})
	}
	return out, nil
}

func (d *opsExhibitionDAO) ListRecentTransferred(ctx context.Context, limit int) ([]*model.OpsExhibition, error) {
	if limit <= 0 {
		limit = 10
	}
	var items []*model.OpsExhibition
	err := d.db.WithContext(ctx).Model(&model.OpsExhibition{}).
		Where("status = ?", model.OpsExhibitionStatusTransferred).
		Order("updated_at DESC").Limit(limit).Find(&items).Error
	return items, err
}

type OpsVisitDAO interface {
	Create(ctx context.Context, v *model.OpsVisit) error
	Update(ctx context.Context, v *model.OpsVisit) error
	UpdateTransferFields(ctx context.Context, v *model.OpsVisit) error
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.OpsVisit, error)
	GetByExhibitionID(ctx context.Context, exhibitionID int) (*model.OpsVisit, error)
	FindOrphanFromExhibition(ctx context.Context, exhibitionID int, companyName string) (*model.OpsVisit, error)
	List(ctx context.Context, req *model.ListOpsVisitReq) ([]*model.OpsVisit, int64, error)
	ListPreDueRemind(ctx context.Context, withinDays int) ([]*model.OpsVisit, error)
	MarkPreDueReminded(ctx context.Context, id int) error
	MarkConverted(ctx context.Context, id, customerID int) error
	CountAll(ctx context.Context) (int64, error)
	CountByStatuses(ctx context.Context, statuses []string) (int64, error)
	CountDueSoon(ctx context.Context, withinDays int) (int64, error)
	CountOverdue(ctx context.Context) (int64, error)
	ListAlerts(ctx context.Context, withinDays, limit int) ([]*model.OpsVisit, error)
	ListHighIntentPending(ctx context.Context, limit int) ([]*model.OpsVisit, error)
	ListRecentConverted(ctx context.Context, limit int) ([]*model.OpsVisit, error)
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

// UpdateTransferFields 展厅转入/回填外访时更新快照字段（含对接人与截止日）
func (d *opsVisitDAO) UpdateTransferFields(ctx context.Context, v *model.OpsVisit) error {
	result := d.db.WithContext(ctx).Model(&model.OpsVisit{}).Where("id = ?", v.ID).Updates(map[string]interface{}{
		"title":             v.Title,
		"target_org":        v.TargetOrg,
		"status":            v.Status,
		"source":            v.Source,
		"lead_source":       v.LeadSource,
		"exhibition_id":     v.ExhibitionID,
		"contact_name":      v.ContactName,
		"contact_title":     v.ContactTitle,
		"contact_phone":     v.ContactPhone,
		"host_name":         v.HostName,
		"participants":      v.Participants,
		"d0_at":             v.D0At,
		"planned_at":        v.PlannedAt,
		"visit_goal":        v.VisitGoal,
		"summary":           v.Summary,
		"intent":            v.Intent,
		"follow_owner_id":   v.FollowOwnerID,
		"follow_owner_name": v.FollowOwnerName,
		"assigned_at":       v.AssignedAt,
		"due_at":            v.DueAt,
		"updater_id":        v.UpdaterID,
		"updater_name":      v.UpdaterName,
		"remark":            v.Remark,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("外访记录不存在")
	}
	return nil
}

func (d *opsVisitDAO) Update(ctx context.Context, v *model.OpsVisit) error {
	result := d.db.WithContext(ctx).Model(&model.OpsVisit{}).Where("id = ?", v.ID).Updates(map[string]interface{}{
		"title":               v.Title,
		"target_org":          v.TargetOrg,
		"start_at":            v.StartAt,
		"end_at":              v.EndAt,
		"location":            v.Location,
		"participants":        v.Participants,
		"summary":             v.Summary,
		"outcome":             v.Outcome,
		"status":              v.Status,
		"lead_source":         v.LeadSource,
		"credit_code":         v.CreditCode,
		"industry":            v.Industry,
		"company_scale":       v.CompanyScale,
		"qualifications":      v.Qualifications,
		"finance_status":      v.FinanceStatus,
		"address":             v.Address,
		"product_line":        v.ProductLine,
		"has_cooperation":     v.HasCooperation,
		"contact_name":        v.ContactName,
		"contact_title":       v.ContactTitle,
		"decision_role":       v.DecisionRole,
		"contact_phone":       v.ContactPhone,
		"contact_email":       v.ContactEmail,
		"contact_wechat":      v.ContactWechat,
		"referrer_name":       v.ReferrerName,
		"host_name":           v.HostName,
		"d0_at":               v.D0At,
		"planned_at":          v.PlannedAt,
		"visit_goal":          v.VisitGoal,
		"prep_materials":      v.PrepMaterials,
		"duration_min":        v.DurationMin,
		"location_type":       v.LocationType,
		"pain_points":         v.PainPoints,
		"objections":          v.Objections,
		"competitor_info":     v.CompetitorInfo,
		"site_feedback":       v.SiteFeedback,
		"intent":              v.Intent,
		"match_score":         v.MatchScore,
		"opportunity_amount":  v.OpportunityAmount,
		"next_action":         v.NextAction,
		"follow_owner_id":     v.FollowOwnerID,
		"follow_owner_name":   v.FollowOwnerName,
		"next_follow_at":      v.NextFollowAt,
		"remark":              v.Remark,
		"customer_id":         v.CustomerID,
		"updater_id":          v.UpdaterID,
		"updater_name":        v.UpdaterName,
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

func (d *opsVisitDAO) GetByExhibitionID(ctx context.Context, exhibitionID int) (*model.OpsVisit, error) {
	var v model.OpsVisit
	err := d.db.WithContext(ctx).Where("exhibition_id = ?", exhibitionID).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// FindOrphanFromExhibition 兼容旧数据：曾转入但未写入 exhibition_id 的外访单
func (d *opsVisitDAO) FindOrphanFromExhibition(ctx context.Context, exhibitionID int, companyName string) (*model.OpsVisit, error) {
	title := fmt.Sprintf("展厅跟进-%s", companyName)
	var v model.OpsVisit
	err := d.db.WithContext(ctx).
		Where("(exhibition_id = ? OR (exhibition_id IS NULL AND title = ? AND target_org = ?))", exhibitionID, title, companyName).
		Order("id DESC").
		First(&v).Error
	if err != nil {
		return nil, err
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
	if req.Source != "" {
		q = q.Where("source = ?", req.Source)
	}
	if req.Intent != "" {
		q = q.Where("intent = ?", req.Intent)
	}
	if req.Search != "" {
		like := "%" + req.Search + "%"
		q = q.Where("title LIKE ? OR target_org LIKE ? OR contact_name LIKE ?", like, like, like)
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

func (d *opsVisitDAO) ListPreDueRemind(ctx context.Context, withinDays int) ([]*model.OpsVisit, error) {
	if withinDays <= 0 {
		withinDays = model.OpsVisitPreDueDays
	}
	now := time.Now()
	deadline := now.Add(time.Duration(withinDays) * 24 * time.Hour)
	var items []*model.OpsVisit
	err := d.db.WithContext(ctx).
		Where("status IN ?", []string{model.OpsVisitStatusPending, model.OpsVisitStatusBooked, model.OpsVisitStatusPlanned}).
		Where("pre_due_reminded = 0").
		Where("follow_owner_id > 0").
		Where("due_at IS NOT NULL AND due_at > ? AND due_at <= ?", now, deadline).
		Order("due_at ASC").
		Find(&items).Error
	return items, err
}

func (d *opsVisitDAO) MarkPreDueReminded(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Model(&model.OpsVisit{}).Where("id = ?", id).Update("pre_due_reminded", 1).Error
}

func (d *opsVisitDAO) MarkConverted(ctx context.Context, id, customerID int) error {
	return d.db.WithContext(ctx).Model(&model.OpsVisit{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":           model.OpsVisitStatusConverted,
		"customer_id":      customerID,
		"crm_transferred":  1,
	}).Error
}

func openVisitStatuses() []string {
	return []string{model.OpsVisitStatusPending, model.OpsVisitStatusBooked, model.OpsVisitStatusPlanned, model.OpsVisitStatusVisited}
}

func (d *opsVisitDAO) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsVisit{}).Count(&n).Error
	return n, err
}

func (d *opsVisitDAO) CountByStatuses(ctx context.Context, statuses []string) (int64, error) {
	var n int64
	q := d.db.WithContext(ctx).Model(&model.OpsVisit{})
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	err := q.Count(&n).Error
	return n, err
}

func (d *opsVisitDAO) CountDueSoon(ctx context.Context, withinDays int) (int64, error) {
	if withinDays <= 0 {
		withinDays = model.OpsVisitPreDueDays
	}
	now := time.Now()
	deadline := now.Add(time.Duration(withinDays) * 24 * time.Hour)
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsVisit{}).
		Where("status IN ?", openVisitStatuses()).
		Where("due_at IS NOT NULL AND due_at > ? AND due_at <= ?", now, deadline).
		Count(&n).Error
	return n, err
}

func (d *opsVisitDAO) CountOverdue(ctx context.Context) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(&model.OpsVisit{}).
		Where("status IN ?", openVisitStatuses()).
		Where("due_at IS NOT NULL AND due_at < ?", time.Now()).
		Count(&n).Error
	return n, err
}

func (d *opsVisitDAO) ListAlerts(ctx context.Context, withinDays, limit int) ([]*model.OpsVisit, error) {
	if withinDays <= 0 {
		withinDays = model.OpsVisitPreDueDays
	}
	if limit <= 0 {
		limit = 15
	}
	deadline := time.Now().Add(time.Duration(withinDays) * 24 * time.Hour)
	var items []*model.OpsVisit
	err := d.db.WithContext(ctx).Model(&model.OpsVisit{}).
		Where("status IN ?", openVisitStatuses()).
		Where("due_at IS NOT NULL AND due_at <= ?", deadline).
		Order("due_at ASC").Limit(limit).Find(&items).Error
	return items, err
}

func (d *opsVisitDAO) ListHighIntentPending(ctx context.Context, limit int) ([]*model.OpsVisit, error) {
	if limit <= 0 {
		limit = 10
	}
	var items []*model.OpsVisit
	err := d.db.WithContext(ctx).Model(&model.OpsVisit{}).
		Where("status IN ?", openVisitStatuses()).
		Where("intent IN ?", []string{model.OpsVisitIntentMedium, model.OpsVisitIntentHigh}).
		Order("updated_at DESC").Limit(limit).Find(&items).Error
	return items, err
}

func (d *opsVisitDAO) ListRecentConverted(ctx context.Context, limit int) ([]*model.OpsVisit, error) {
	if limit <= 0 {
		limit = 10
	}
	var items []*model.OpsVisit
	err := d.db.WithContext(ctx).Model(&model.OpsVisit{}).
		Where("status = ?", model.OpsVisitStatusConverted).
		Order("updated_at DESC").Limit(limit).Find(&items).Error
	return items, err
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
