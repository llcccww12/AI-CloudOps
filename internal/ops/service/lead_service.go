package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OpsLeadService interface {
	CreateExhibition(ctx context.Context, req *model.CreateOpsExhibitionReq) (*model.OpsExhibition, error)
	UpdateExhibition(ctx context.Context, req *model.UpdateOpsExhibitionReq) (*model.OpsExhibition, error)
	DeleteExhibition(ctx context.Context, id int) error
	GetExhibition(ctx context.Context, id int) (*model.OpsExhibition, error)
	ListExhibition(ctx context.Context, req *model.ListOpsExhibitionReq) (*model.ListResp[*model.OpsExhibition], error)
	TransferExhibitionToVisit(ctx context.Context, req *model.ConvertLeadReq) (*model.OpsVisit, error)
	SubmitPublicVisitor(ctx context.Context, req *model.SubmitPublicVisitorReq) (*model.OpsExhibition, error)

	CreateVisit(ctx context.Context, req *model.CreateOpsVisitReq) error
	UpdateVisit(ctx context.Context, req *model.UpdateOpsVisitReq) error
	DeleteVisit(ctx context.Context, id int) error
	GetVisit(ctx context.Context, id int) (*model.OpsVisit, error)
	ListVisit(ctx context.Context, req *model.ListOpsVisitReq) (*model.ListResp[*model.OpsVisit], error)
	ConvertVisit(ctx context.Context, req *model.ConvertLeadReq) (*model.OpsCustomer, error)
}

type opsLeadService struct {
	exhibitionDAO dao.OpsExhibitionDAO
	visitDAO      dao.OpsVisitDAO
	customerDAO   dao.OpsCustomerDAO
	attachmentDAO dao.OpsAttachmentDAO
	inboxDAO      workorderDao.WorkorderInboxDAO
	logger        *zap.Logger
}

func NewOpsLeadService(
	exhibitionDAO dao.OpsExhibitionDAO,
	visitDAO dao.OpsVisitDAO,
	customerDAO dao.OpsCustomerDAO,
	attachmentDAO dao.OpsAttachmentDAO,
	inboxDAO workorderDao.WorkorderInboxDAO,
	logger *zap.Logger,
) OpsLeadService {
	return &opsLeadService{
		exhibitionDAO: exhibitionDAO, visitDAO: visitDAO, customerDAO: customerDAO,
		attachmentDAO: attachmentDAO, inboxDAO: inboxDAO, logger: logger,
	}
}

func (s *opsLeadService) CreateExhibition(ctx context.Context, req *model.CreateOpsExhibitionReq) (*model.OpsExhibition, error) {
	e := &model.OpsExhibition{
		CompanyName: strings.TrimSpace(req.CompanyName), CompanyLevel: req.CompanyLevel,
		VisitorName: req.VisitorName, VisitorTitle: req.VisitorTitle, VisitorCount: req.VisitorCount,
		VisitAt: req.VisitAt, DockingUnit: req.DockingUnit, HostName: req.HostName, Purpose: req.Purpose,
		NeedMeeting: req.NeedMeeting, FocusTags: req.FocusTags, Content: req.Content,
		MeetingMinutes: req.MeetingMinutes, ContactPhone: req.ContactPhone, Companions: req.Companions,
		Intent: req.Intent, Source: model.OpsExhibitionSourceStaff, Status: model.OpsExhibitionStatusDraft,
		CustomerID: req.CustomerID, OperatorID: req.OperatorID, OperatorName: req.OperatorName, Remark: req.Remark,
		UpdaterID: req.OperatorID, UpdaterName: req.OperatorName,
	}
	if err := s.exhibitionDAO.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *opsLeadService) UpdateExhibition(ctx context.Context, req *model.UpdateOpsExhibitionReq) (*model.OpsExhibition, error) {
	if err := s.exhibitionDAO.Update(ctx, &model.OpsExhibition{
		Model: model.Model{ID: req.ID}, CompanyName: strings.TrimSpace(req.CompanyName),
		CompanyLevel: req.CompanyLevel, VisitorName: req.VisitorName, VisitorTitle: req.VisitorTitle,
		VisitorCount: req.VisitorCount, VisitAt: req.VisitAt, DockingUnit: req.DockingUnit,
		HostName: req.HostName, Purpose: req.Purpose, NeedMeeting: req.NeedMeeting,
		FocusTags: req.FocusTags, Content: req.Content, MeetingMinutes: req.MeetingMinutes,
		ContactPhone: req.ContactPhone, Companions: req.Companions, Intent: req.Intent,
		Remark: req.Remark, CustomerID: req.CustomerID,
		UpdaterID: req.UpdaterID, UpdaterName: req.UpdaterName,
	}); err != nil {
		return nil, err
	}
	return s.exhibitionDAO.GetByID(ctx, req.ID)
}

func (s *opsLeadService) DeleteExhibition(ctx context.Context, id int) error {
	return s.exhibitionDAO.Delete(ctx, id)
}
func (s *opsLeadService) GetExhibition(ctx context.Context, id int) (*model.OpsExhibition, error) {
	return s.exhibitionDAO.GetByID(ctx, id)
}
func (s *opsLeadService) ListExhibition(ctx context.Context, req *model.ListOpsExhibitionReq) (*model.ListResp[*model.OpsExhibition], error) {
	items, total, err := s.exhibitionDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsExhibition]{Items: items, Total: total}, nil
}

// TransferExhibitionToVisit 展厅意向中/高时转入外访（需指定对接人；幂等）
func (s *opsLeadService) TransferExhibitionToVisit(ctx context.Context, req *model.ConvertLeadReq) (*model.OpsVisit, error) {
	if req.OwnerID <= 0 || strings.TrimSpace(req.OwnerName) == "" {
		return nil, fmt.Errorf("请指定外访对接人")
	}
	e, err := s.exhibitionDAO.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return s.transferExhibitionLocked(ctx, e, req)
}

func (s *opsLeadService) transferExhibitionLocked(ctx context.Context, e *model.OpsExhibition, req *model.ConvertLeadReq) (*model.OpsVisit, error) {
	if !model.IsHighOrMediumIntent(e.Intent) {
		return nil, fmt.Errorf("仅意向为中/高时可转入外访交流")
	}

	now := time.Now()
	due := now.Add(time.Duration(model.OpsVisitAssignDays) * 24 * time.Hour)
	exID := e.ID
	operatorID, operatorName := req.OperatorID, req.OperatorName
	if operatorName == "" {
		operatorID, operatorName = e.OperatorID, e.OperatorName
	}
	snapshot := s.buildVisitFromExhibition(e, req, exID, operatorID, operatorName, now, due)

	existing, err := s.findExistingVisitForExhibition(ctx, e)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		snapshot.ID = existing.ID
		if existing.Status == model.OpsVisitStatusConverted {
			snapshot.Status = existing.Status
		} else if existing.Status != "" {
			snapshot.Status = existing.Status
		}
		// 已有对接人且本次未改派时保留原截止日
		if existing.FollowOwnerID == req.OwnerID && existing.DueAt != nil {
			snapshot.DueAt = existing.DueAt
			snapshot.AssignedAt = existing.AssignedAt
			if existing.PlannedAt != nil {
				snapshot.PlannedAt = existing.PlannedAt
			}
		}
		if err := s.visitDAO.UpdateTransferFields(ctx, snapshot); err != nil {
			return nil, err
		}
		if err := s.exhibitionDAO.MarkTransferred(ctx, e.ID, existing.ID); err != nil {
			return nil, err
		}
		if existing.FollowOwnerID != req.OwnerID {
			s.notifyVisitAssignee(ctx, snapshot, true)
		}
		return s.visitDAO.GetByID(ctx, existing.ID)
	}

	if err := s.visitDAO.Create(ctx, snapshot); err != nil {
		return nil, err
	}
	if err := s.exhibitionDAO.MarkTransferred(ctx, e.ID, snapshot.ID); err != nil {
		return nil, err
	}
	s.notifyVisitAssignee(ctx, snapshot, true)
	return snapshot, nil
}

func (s *opsLeadService) findExistingVisitForExhibition(ctx context.Context, e *model.OpsExhibition) (*model.OpsVisit, error) {
	if e.VisitID != nil && *e.VisitID > 0 {
		v, err := s.visitDAO.GetByID(ctx, *e.VisitID)
		if err == nil {
			return v, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) && !strings.Contains(err.Error(), "不存在") {
			return nil, err
		}
	}
	if v, err := s.visitDAO.GetByExhibitionID(ctx, e.ID); err == nil {
		return v, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if v, err := s.visitDAO.FindOrphanFromExhibition(ctx, e.ID, e.CompanyName); err == nil {
		return v, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return nil, nil
}

func (s *opsLeadService) buildVisitFromExhibition(
	e *model.OpsExhibition,
	req *model.ConvertLeadReq,
	exID, operatorID int,
	operatorName string,
	now, due time.Time,
) *model.OpsVisit {
	summaryParts := []string{}
	if e.CompanyLevel != "" {
		summaryParts = append(summaryParts, "单位层级："+e.CompanyLevel)
	}
	if e.DockingUnit != "" {
		summaryParts = append(summaryParts, "对接单位："+e.DockingUnit)
	}
	if e.Purpose != "" {
		summaryParts = append(summaryParts, "来访目的："+e.Purpose)
	}
	if e.Content != "" {
		summaryParts = append(summaryParts, "展厅讲解："+e.Content)
	}
	if e.MeetingMinutes != "" {
		summaryParts = append(summaryParts, "会议纪要："+e.MeetingMinutes)
	}
	if len(e.FocusTags) > 0 {
		summaryParts = append(summaryParts, "关注重点："+strings.Join(e.FocusTags, "、"))
	}
	contactName := strings.TrimSpace(req.ContactName)
	if contactName == "" {
		contactName = e.VisitorName
	}
	contactPhone := strings.TrimSpace(req.ContactPhone)
	if contactPhone == "" {
		contactPhone = e.ContactPhone
	}
	return &model.OpsVisit{
		Title:           fmt.Sprintf("展厅跟进-%s", e.CompanyName),
		TargetOrg:       e.CompanyName,
		Status:          model.OpsVisitStatusPending,
		Source:          model.OpsVisitSourceExhibition,
		LeadSource:      "展厅接待",
		ExhibitionID:    &exID,
		ContactName:     contactName,
		ContactTitle:    e.VisitorTitle,
		ContactPhone:    contactPhone,
		HostName:        e.HostName,
		Participants:    e.Companions,
		D0At:            e.VisitAt,
		VisitGoal:       e.Purpose,
		Summary:         strings.Join(summaryParts, "\n"),
		Intent:          e.Intent,
		FollowOwnerID:   req.OwnerID,
		FollowOwnerName: strings.TrimSpace(req.OwnerName),
		AssignedAt:      &now,
		DueAt:           &due,
		PlannedAt:       &due,
		OperatorID:      operatorID,
		OperatorName:    operatorName,
		UpdaterID:       operatorID,
		UpdaterName:     operatorName,
		Remark:          fmt.Sprintf("来自展厅接待#%d", e.ID),
	}
}

func (s *opsLeadService) notifyVisitAssignee(ctx context.Context, v *model.OpsVisit, isAssign bool) {
	if s.inboxDAO == nil || v == nil || v.FollowOwnerID <= 0 {
		return
	}
	dueText := ""
	if v.DueAt != nil {
		dueText = v.DueAt.Format("2006-01-02")
	}
	var title, content string
	if isAssign {
		title = "外访任务已下发"
		content = fmt.Sprintf("您被指定为「%s」的外访对接人，请在 %d 天内（截止 %s）完成走访。外访单#%d → 打开: /ops/visits",
			v.TargetOrg, model.OpsVisitAssignDays, dueText, v.ID)
	} else {
		title = "外访任务即将到期"
		content = fmt.Sprintf("外访「%s」将于 %s 到期，请尽快完成走访。外访单#%d → 打开: /ops/visits",
			v.TargetOrg, dueText, v.ID)
	}
	if err := s.inboxDAO.Create(ctx, &model.WorkorderInboxMessage{
		UserID: v.FollowOwnerID, Title: "[运营] " + title, Content: content,
		EventType: "ops_visit_assign", IsRead: model.InboxMessageUnread,
	}); err != nil {
		s.logger.Warn("外访对接人站内信发送失败", zap.Int("visit_id", v.ID), zap.Error(err))
	}
}

func (s *opsLeadService) SubmitPublicVisitor(ctx context.Context, req *model.SubmitPublicVisitorReq) (*model.OpsExhibition, error) {
	companyLevel := strings.TrimSpace(req.CompanyLevel)
	allowedLevels := map[string]bool{
		"省级": true, "市级": true, "区县级": true, "央企": true, "国企": true,
		"民企": true, "高校": true, "其他": true,
	}
	if !allowedLevels[companyLevel] {
		return nil, fmt.Errorf("来访单位层级无效")
	}
	phone := strings.TrimSpace(req.ContactPhone)
	if phone == "" {
		return nil, fmt.Errorf("带队人联系电话不能为空")
	}
	e := &model.OpsExhibition{
		CompanyName:    strings.TrimSpace(req.CompanyName),
		CompanyLevel:   companyLevel,
		VisitorName:    strings.TrimSpace(req.VisitorName),
		VisitorTitle:   strings.TrimSpace(req.VisitorTitle),
		VisitorCount:   req.VisitorCount,
		VisitAt:        req.VisitAt,
		DockingUnit:    strings.TrimSpace(req.DockingUnit),
		HostName:       strings.TrimSpace(req.HostName),
		Purpose:        strings.TrimSpace(req.Purpose),
		NeedMeeting:    req.NeedMeeting,
		Content:        strings.TrimSpace(req.Content),
		MeetingMinutes: strings.TrimSpace(req.MeetingMinutes),
		ContactPhone:   phone,
		Intent:         "",
		Source:         model.OpsExhibitionSourcePublic,
		Status:         model.OpsExhibitionStatusDraft,
		OperatorName:   "公开登记",
	}
	if err := s.exhibitionDAO.Create(ctx, e); err != nil {
		return nil, err
	}
	if len(req.AttachmentIDs) > 0 {
		maxCount := 20
		if len(req.AttachmentIDs) > maxCount {
			return nil, fmt.Errorf("附件数量不能超过 %d 个", maxCount)
		}
		if err := s.attachmentDAO.BindBizIDs(ctx, model.OpsAttachmentBizExhibition, e.ID, req.AttachmentIDs); err != nil {
			s.logger.Warn("绑定公开访客附件失败", zap.Int("exhibition_id", e.ID), zap.Error(err))
		}
	}
	return e, nil
}

func visitFromCreateReq(req *model.CreateOpsVisitReq) *model.OpsVisit {
	status := req.Status
	if status == "" {
		status = model.OpsVisitStatusPending
	}
	return &model.OpsVisit{
		Title: strings.TrimSpace(req.Title), TargetOrg: req.TargetOrg, StartAt: req.StartAt, EndAt: req.EndAt,
		Location: req.Location, Participants: req.Participants, Summary: req.Summary, Outcome: req.Outcome,
		Status: status, Source: model.OpsVisitSourceStaff, LeadSource: req.LeadSource,
		CreditCode: req.CreditCode, Industry: req.Industry, CompanyScale: req.CompanyScale,
		Qualifications: req.Qualifications, FinanceStatus: req.FinanceStatus, Address: req.Address,
		ProductLine: req.ProductLine, HasCooperation: req.HasCooperation,
		ContactName: req.ContactName, ContactTitle: req.ContactTitle, DecisionRole: req.DecisionRole,
		ContactPhone: req.ContactPhone, ContactEmail: req.ContactEmail, ContactWechat: req.ContactWechat,
		ReferrerName: req.ReferrerName, HostName: req.HostName,
		D0At: req.D0At, PlannedAt: req.PlannedAt, VisitGoal: req.VisitGoal, PrepMaterials: req.PrepMaterials,
		DurationMin: req.DurationMin, LocationType: req.LocationType, PainPoints: req.PainPoints,
		Objections: req.Objections, CompetitorInfo: req.CompetitorInfo, SiteFeedback: req.SiteFeedback,
		Intent: req.Intent, MatchScore: req.MatchScore, OpportunityAmount: req.OpportunityAmount,
		NextAction: req.NextAction, FollowOwnerID: req.FollowOwnerID, FollowOwnerName: req.FollowOwnerName,
		NextFollowAt: req.NextFollowAt, CustomerID: req.CustomerID,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName, Remark: req.Remark,
		UpdaterID: req.OperatorID, UpdaterName: req.OperatorName,
	}
}

func (s *opsLeadService) CreateVisit(ctx context.Context, req *model.CreateOpsVisitReq) error {
	return s.visitDAO.Create(ctx, visitFromCreateReq(req))
}

func (s *opsLeadService) UpdateVisit(ctx context.Context, req *model.UpdateOpsVisitReq) error {
	status := req.Status
	if status == "" {
		status = model.OpsVisitStatusPending
	}
	return s.visitDAO.Update(ctx, &model.OpsVisit{
		Model: model.Model{ID: req.ID}, Title: strings.TrimSpace(req.Title), TargetOrg: req.TargetOrg,
		StartAt: req.StartAt, EndAt: req.EndAt, Location: req.Location, Participants: req.Participants,
		Summary: req.Summary, Outcome: req.Outcome, Status: status, LeadSource: req.LeadSource,
		CreditCode: req.CreditCode, Industry: req.Industry, CompanyScale: req.CompanyScale,
		Qualifications: req.Qualifications, FinanceStatus: req.FinanceStatus, Address: req.Address,
		ProductLine: req.ProductLine, HasCooperation: req.HasCooperation,
		ContactName: req.ContactName, ContactTitle: req.ContactTitle, DecisionRole: req.DecisionRole,
		ContactPhone: req.ContactPhone, ContactEmail: req.ContactEmail, ContactWechat: req.ContactWechat,
		ReferrerName: req.ReferrerName, HostName: req.HostName,
		D0At: req.D0At, PlannedAt: req.PlannedAt, VisitGoal: req.VisitGoal, PrepMaterials: req.PrepMaterials,
		DurationMin: req.DurationMin, LocationType: req.LocationType, PainPoints: req.PainPoints,
		Objections: req.Objections, CompetitorInfo: req.CompetitorInfo, SiteFeedback: req.SiteFeedback,
		Intent: req.Intent, MatchScore: req.MatchScore, OpportunityAmount: req.OpportunityAmount,
		NextAction: req.NextAction, FollowOwnerID: req.FollowOwnerID, FollowOwnerName: req.FollowOwnerName,
		NextFollowAt: req.NextFollowAt, Remark: req.Remark, CustomerID: req.CustomerID,
		UpdaterID: req.UpdaterID, UpdaterName: req.UpdaterName,
	})
}

func (s *opsLeadService) DeleteVisit(ctx context.Context, id int) error {
	return s.visitDAO.Delete(ctx, id)
}
func (s *opsLeadService) GetVisit(ctx context.Context, id int) (*model.OpsVisit, error) {
	return s.visitDAO.GetByID(ctx, id)
}
func (s *opsLeadService) ListVisit(ctx context.Context, req *model.ListOpsVisitReq) (*model.ListResp[*model.OpsVisit], error) {
	items, total, err := s.visitDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsVisit]{Items: items, Total: total}, nil
}

func (s *opsLeadService) ConvertVisit(ctx context.Context, req *model.ConvertLeadReq) (*model.OpsCustomer, error) {
	v, err := s.visitDAO.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if !model.IsHighOrMediumIntent(v.Intent) {
		return nil, fmt.Errorf("仅客户意向为中/高时可转入客户中心")
	}
	if v.Status == model.OpsVisitStatusConverted && v.CustomerID != nil {
		return s.customerDAO.GetByID(ctx, *v.CustomerID)
	}
	ownerID, ownerName := req.OwnerID, req.OwnerName
	if ownerID <= 0 {
		if v.FollowOwnerID > 0 {
			ownerID = v.FollowOwnerID
			ownerName = v.FollowOwnerName
		} else {
			ownerID = req.OperatorID
			ownerName = req.OperatorName
		}
	}
	name := v.TargetOrg
	if name == "" {
		name = v.Title
	}
	contactName := req.ContactName
	if contactName == "" {
		contactName = v.ContactName
	}
	contactPhone := req.ContactPhone
	if contactPhone == "" {
		contactPhone = v.ContactPhone
	}
	c := &model.OpsCustomer{
		Name: name, Stage: model.OpsCustomerStageIntent, DemandTypes: req.DemandTypes,
		Industry: v.Industry, ContactName: contactName, ContactTitle: v.ContactTitle,
		ContactPhone: contactPhone, ContactEmail: v.ContactEmail,
		Source: model.OpsCustomerSourceVisit, SourceRefType: "visit", SourceRefID: &v.ID,
		OwnerID: ownerID, OwnerName: ownerName, OperatorID: req.OperatorID, OperatorName: req.OperatorName,
		NextFollowAt: v.NextFollowAt,
		Remark:       fmt.Sprintf("来自外访交流#%d %s", v.ID, v.Outcome),
	}
	if err := s.customerDAO.Create(ctx, c); err != nil {
		return nil, err
	}
	if err := s.visitDAO.MarkConverted(ctx, v.ID, c.ID); err != nil {
		return nil, err
	}
	return c, nil
}
