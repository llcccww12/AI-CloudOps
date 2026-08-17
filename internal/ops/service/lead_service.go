package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	"go.uber.org/zap"
)

type OpsLeadService interface {
	CreateExhibition(ctx context.Context, req *model.CreateOpsExhibitionReq) error
	UpdateExhibition(ctx context.Context, req *model.UpdateOpsExhibitionReq) error
	DeleteExhibition(ctx context.Context, id int) error
	GetExhibition(ctx context.Context, id int) (*model.OpsExhibition, error)
	ListExhibition(ctx context.Context, req *model.ListOpsExhibitionReq) (*model.ListResp[*model.OpsExhibition], error)
	ConvertExhibition(ctx context.Context, req *model.ConvertLeadReq) (*model.OpsCustomer, error)

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
	logger        *zap.Logger
}

func NewOpsLeadService(
	exhibitionDAO dao.OpsExhibitionDAO,
	visitDAO dao.OpsVisitDAO,
	customerDAO dao.OpsCustomerDAO,
	logger *zap.Logger,
) OpsLeadService {
	return &opsLeadService{exhibitionDAO: exhibitionDAO, visitDAO: visitDAO, customerDAO: customerDAO, logger: logger}
}

func (s *opsLeadService) CreateExhibition(ctx context.Context, req *model.CreateOpsExhibitionReq) error {
	return s.exhibitionDAO.Create(ctx, &model.OpsExhibition{
		CompanyName: strings.TrimSpace(req.CompanyName), VisitorName: req.VisitorName, VisitorTitle: req.VisitorTitle,
		VisitAt: req.VisitAt, HostName: req.HostName, Purpose: req.Purpose, FocusTags: req.FocusTags,
		Content: req.Content, Companions: req.Companions, Status: model.OpsExhibitionStatusDraft,
		CustomerID: req.CustomerID, OperatorID: req.OperatorID, OperatorName: req.OperatorName, Remark: req.Remark,
	})
}

func (s *opsLeadService) UpdateExhibition(ctx context.Context, req *model.UpdateOpsExhibitionReq) error {
	return s.exhibitionDAO.Update(ctx, &model.OpsExhibition{
		Model: model.Model{ID: req.ID}, CompanyName: strings.TrimSpace(req.CompanyName),
		VisitorName: req.VisitorName, VisitorTitle: req.VisitorTitle, VisitAt: req.VisitAt,
		HostName: req.HostName, Purpose: req.Purpose, FocusTags: req.FocusTags,
		Content: req.Content, Companions: req.Companions, Remark: req.Remark, CustomerID: req.CustomerID,
	})
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

func (s *opsLeadService) ConvertExhibition(ctx context.Context, req *model.ConvertLeadReq) (*model.OpsCustomer, error) {
	e, err := s.exhibitionDAO.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if e.Status == model.OpsExhibitionStatusConverted && e.CustomerID != nil {
		return s.customerDAO.GetByID(ctx, *e.CustomerID)
	}
	ownerID, ownerName := req.OwnerID, req.OwnerName
	if ownerID <= 0 {
		ownerID = req.OperatorID
		ownerName = req.OperatorName
	}
	contactName := req.ContactName
	if contactName == "" {
		contactName = e.VisitorName
	}
	c := &model.OpsCustomer{
		Name: e.CompanyName, Stage: model.OpsCustomerStageIntent, DemandTypes: req.DemandTypes,
		ContactName: contactName, ContactTitle: e.VisitorTitle, ContactPhone: req.ContactPhone,
		Source: model.OpsCustomerSourceExhibition, SourceRefType: "exhibition", SourceRefID: &e.ID,
		OwnerID: ownerID, OwnerName: ownerName, OperatorID: req.OperatorID, OperatorName: req.OperatorName,
		Remark: fmt.Sprintf("来自展厅接待#%d", e.ID),
	}
	if err := s.customerDAO.Create(ctx, c); err != nil {
		return nil, err
	}
	if err := s.exhibitionDAO.MarkConverted(ctx, e.ID, c.ID); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *opsLeadService) CreateVisit(ctx context.Context, req *model.CreateOpsVisitReq) error {
	return s.visitDAO.Create(ctx, &model.OpsVisit{
		Title: strings.TrimSpace(req.Title), TargetOrg: req.TargetOrg, StartAt: req.StartAt, EndAt: req.EndAt,
		Location: req.Location, Participants: req.Participants, Summary: req.Summary, Outcome: req.Outcome,
		Status: model.OpsVisitStatusPlanned, CustomerID: req.CustomerID,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName, Remark: req.Remark,
	})
}

func (s *opsLeadService) UpdateVisit(ctx context.Context, req *model.UpdateOpsVisitReq) error {
	status := req.Status
	if status == "" {
		status = model.OpsVisitStatusPlanned
	}
	return s.visitDAO.Update(ctx, &model.OpsVisit{
		Model: model.Model{ID: req.ID}, Title: strings.TrimSpace(req.Title), TargetOrg: req.TargetOrg,
		StartAt: req.StartAt, EndAt: req.EndAt, Location: req.Location, Participants: req.Participants,
		Summary: req.Summary, Outcome: req.Outcome, Status: status, Remark: req.Remark, CustomerID: req.CustomerID,
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
	if v.Status == model.OpsVisitStatusConverted && v.CustomerID != nil {
		return s.customerDAO.GetByID(ctx, *v.CustomerID)
	}
	ownerID, ownerName := req.OwnerID, req.OwnerName
	if ownerID <= 0 {
		ownerID = req.OperatorID
		ownerName = req.OperatorName
	}
	name := v.TargetOrg
	if name == "" {
		name = v.Title
	}
	c := &model.OpsCustomer{
		Name: name, Stage: model.OpsCustomerStageIntent, DemandTypes: req.DemandTypes,
		ContactName: req.ContactName, ContactPhone: req.ContactPhone,
		Source: model.OpsCustomerSourceVisit, SourceRefType: "visit", SourceRefID: &v.ID,
		OwnerID: ownerID, OwnerName: ownerName, OperatorID: req.OperatorID, OperatorName: req.OperatorName,
		Remark: fmt.Sprintf("来自外访交流#%d %s", v.ID, v.Outcome),
	}
	if err := s.customerDAO.Create(ctx, c); err != nil {
		return nil, err
	}
	if err := s.visitDAO.MarkConverted(ctx, v.ID, c.ID); err != nil {
		return nil, err
	}
	return c, nil
}
