package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	"go.uber.org/zap"
)

type OpsCustomerService interface {
	Create(ctx context.Context, req *model.CreateOpsCustomerReq) error
	Update(ctx context.Context, req *model.UpdateOpsCustomerReq) error
	Delete(ctx context.Context, id int) error
	Get(ctx context.Context, id int) (*model.OpsCustomer, error)
	List(ctx context.Context, req *model.ListOpsCustomerReq) (*model.ListResp[*model.OpsCustomer], error)
	ChangeStage(ctx context.Context, req *model.ChangeOpsCustomerStageReq) error
	CreateFollowup(ctx context.Context, req *model.CreateOpsFollowupReq) error
	ListFollowups(ctx context.Context, req *model.ListOpsFollowupReq) (*model.ListResp[*model.OpsFollowup], error)
}

type opsCustomerService struct {
	customerDAO dao.OpsCustomerDAO
	followupDAO dao.OpsFollowupDAO
	logger      *zap.Logger
}

func NewOpsCustomerService(customerDAO dao.OpsCustomerDAO, followupDAO dao.OpsFollowupDAO, logger *zap.Logger) OpsCustomerService {
	return &opsCustomerService{customerDAO: customerDAO, followupDAO: followupDAO, logger: logger}
}

func (s *opsCustomerService) Create(ctx context.Context, req *model.CreateOpsCustomerReq) error {
	stage := req.Stage
	if stage == "" {
		stage = model.OpsCustomerStageLead
	}
	source := req.Source
	if source == "" {
		source = model.OpsCustomerSourceManual
	}
	ownerID := req.OwnerID
	if ownerID <= 0 {
		ownerID = req.OperatorID
	}
	ownerName := req.OwnerName
	if ownerName == "" {
		ownerName = req.OperatorName
	}
	c := &model.OpsCustomer{
		Name: strings.TrimSpace(req.Name), Stage: stage, DemandTypes: req.DemandTypes,
		Industry: req.Industry, ContactName: req.ContactName, ContactTitle: req.ContactTitle,
		ContactPhone: req.ContactPhone, ContactEmail: req.ContactEmail, Source: source,
		OwnerID: ownerID, OwnerName: ownerName, BudgetRange: req.BudgetRange,
		NextFollowAt: req.NextFollowAt, Remark: req.Remark,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	}
	return s.customerDAO.Create(ctx, c)
}

func (s *opsCustomerService) Update(ctx context.Context, req *model.UpdateOpsCustomerReq) error {
	return s.customerDAO.Update(ctx, &model.OpsCustomer{
		Model: model.Model{ID: req.ID}, Name: strings.TrimSpace(req.Name), DemandTypes: req.DemandTypes,
		Industry: req.Industry, ContactName: req.ContactName, ContactTitle: req.ContactTitle,
		ContactPhone: req.ContactPhone, ContactEmail: req.ContactEmail,
		OwnerID: req.OwnerID, OwnerName: req.OwnerName, BudgetRange: req.BudgetRange,
		NextFollowAt: req.NextFollowAt, Remark: req.Remark,
	})
}

func (s *opsCustomerService) Delete(ctx context.Context, id int) error {
	return s.customerDAO.Delete(ctx, id)
}

func (s *opsCustomerService) Get(ctx context.Context, id int) (*model.OpsCustomer, error) {
	return s.customerDAO.GetByID(ctx, id)
}

func (s *opsCustomerService) List(ctx context.Context, req *model.ListOpsCustomerReq) (*model.ListResp[*model.OpsCustomer], error) {
	items, total, err := s.customerDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsCustomer]{Items: items, Total: total}, nil
}

var allowedStageTransitions = map[string]map[string]bool{
	model.OpsCustomerStageLead:   {model.OpsCustomerStageIntent: true, model.OpsCustomerStageClosed: true},
	model.OpsCustomerStageIntent: {model.OpsCustomerStageTrial: true, model.OpsCustomerStageFormal: true, model.OpsCustomerStageClosed: true},
	model.OpsCustomerStageTrial:  {model.OpsCustomerStageFormal: true, model.OpsCustomerStageClosed: true},
	model.OpsCustomerStageFormal: {model.OpsCustomerStageClosed: true},
	model.OpsCustomerStageClosed: {model.OpsCustomerStageIntent: true},
}

func (s *opsCustomerService) ChangeStage(ctx context.Context, req *model.ChangeOpsCustomerStageReq) error {
	c, err := s.customerDAO.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if c.Stage == req.Stage {
		return nil
	}
	if allowed := allowedStageTransitions[c.Stage]; !allowed[req.Stage] {
		return fmt.Errorf("不允许从 %s 流转到 %s", c.Stage, req.Stage)
	}
	if req.Stage == model.OpsCustomerStageClosed && strings.TrimSpace(req.ClosedReason) == "" {
		return fmt.Errorf("闭环需填写原因")
	}
	return s.customerDAO.UpdateStage(ctx, req.ID, req.Stage, req.ClosedReason)
}

func (s *opsCustomerService) CreateFollowup(ctx context.Context, req *model.CreateOpsFollowupReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.CustomerID); err != nil {
		return err
	}
	return s.followupDAO.Create(ctx, &model.OpsFollowup{
		CustomerID: req.CustomerID, Type: req.Type, Content: strings.TrimSpace(req.Content),
		NextPlan: req.NextPlan, OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	})
}

func (s *opsCustomerService) ListFollowups(ctx context.Context, req *model.ListOpsFollowupReq) (*model.ListResp[*model.OpsFollowup], error) {
	items, total, err := s.followupDAO.ListByCustomer(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsFollowup]{Items: items, Total: total}, nil
}
