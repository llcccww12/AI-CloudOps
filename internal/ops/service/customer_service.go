package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	opsUtils "github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
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
	GetVendorProfile(ctx context.Context, customerID int) (*model.OpsVendorProfile, error)
	UpsertVendorProfile(ctx context.Context, req *model.UpsertOpsVendorProfileReq) error
	UpdateReportSettings(ctx context.Context, req *model.UpdateOpsCustomerReportReq) error
	RotateReportSecret(ctx context.Context, id int) (*model.RotateOpsCustomerReportSecretResp, error)
}

type opsCustomerService struct {
	customerDAO dao.OpsCustomerDAO
	followupDAO dao.OpsFollowupDAO
	vendorDAO   dao.OpsVendorProfileDAO
	surveyDAO   dao.OpsSurveyDAO
	logger      *zap.Logger
}

func NewOpsCustomerService(
	customerDAO dao.OpsCustomerDAO,
	followupDAO dao.OpsFollowupDAO,
	vendorDAO dao.OpsVendorProfileDAO,
	surveyDAO dao.OpsSurveyDAO,
	logger *zap.Logger,
) OpsCustomerService {
	return &opsCustomerService{
		customerDAO: customerDAO, followupDAO: followupDAO,
		vendorDAO: vendorDAO, surveyDAO: surveyDAO, logger: logger,
	}
}

// needNonRenewalSurvey 非成功交付类闭环需先填不续费问卷
var needNonRenewalSurvey = map[string]bool{
	"客户放弃": true, "竞品赢单": true, "预算不足": true,
	"需求变更": true, "长期无跟进": true, "其他": true,
}

func (s *opsCustomerService) Create(ctx context.Context, req *model.CreateOpsCustomerReq) error {
	stage := req.Stage
	if stage == "" || stage == model.OpsCustomerStageLead {
		stage = model.OpsCustomerStageIntent
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
		NextFollowAt: req.NextFollowAt, Remark: req.Remark, VendorProfileDone: req.VendorProfileDone,
	})
}

func (s *opsCustomerService) Delete(ctx context.Context, id int) error {
	return s.customerDAO.Delete(ctx, id)
}

func (s *opsCustomerService) Get(ctx context.Context, id int) (*model.OpsCustomer, error) {
	c, err := s.customerDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	maskCustomerReportFields(c)
	return c, nil
}

func (s *opsCustomerService) List(ctx context.Context, req *model.ListOpsCustomerReq) (*model.ListResp[*model.OpsCustomer], error) {
	items, total, err := s.customerDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		maskCustomerReportFields(item)
	}
	return &model.ListResp[*model.OpsCustomer]{Items: items, Total: total}, nil
}

func maskCustomerReportFields(c *model.OpsCustomer) {
	if c == nil {
		return
	}
	c.ReportSecretConfigured = c.ReportSecretHash != ""
	c.ReportSecretHash = ""
}

var allowedStageTransitions = map[string]map[string]bool{
	// lead 仅兼容历史数据，可升到意向/试用或闭环
	model.OpsCustomerStageLead:   {model.OpsCustomerStageIntent: true, model.OpsCustomerStageTrial: true, model.OpsCustomerStageClosed: true},
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
	reason := strings.TrimSpace(req.ClosedReason)
	if req.Stage == model.OpsCustomerStageClosed && reason == "" {
		return fmt.Errorf("闭环需填写原因")
	}
	if req.Stage == model.OpsCustomerStageClosed && needNonRenewalSurvey[reason] && s.surveyDAO != nil {
		ok, err := s.surveyDAO.HasResponse(ctx, req.ID, model.OpsSurveyTypeNonRenewal)
		if err != nil {
			return fmt.Errorf("校验不续费问卷失败: %w", err)
		}
		if !ok {
			return fmt.Errorf("闭环前请先提交不续费原因问卷")
		}
	}
	return s.customerDAO.UpdateStage(ctx, req.ID, req.Stage, reason)
}

func (s *opsCustomerService) GetVendorProfile(ctx context.Context, customerID int) (*model.OpsVendorProfile, error) {
	if _, err := s.customerDAO.GetByID(ctx, customerID); err != nil {
		return nil, err
	}
	return s.vendorDAO.GetByCustomerID(ctx, customerID)
}

func (s *opsCustomerService) UpsertVendorProfile(ctx context.Context, req *model.UpsertOpsVendorProfileReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.CustomerID); err != nil {
		return err
	}
	accountType := strings.TrimSpace(req.AccountType)
	if accountType == "" {
		accountType = "corporate"
	}
	p := &model.OpsVendorProfile{
		CustomerID: req.CustomerID, UnitName: strings.TrimSpace(req.UnitName),
		CreditCode: strings.TrimSpace(req.CreditCode), Principal: strings.TrimSpace(req.Principal),
		ContactAddressPhone: strings.TrimSpace(req.ContactAddressPhone),
		CnapsCode: strings.TrimSpace(req.CnapsCode), AccountName: strings.TrimSpace(req.AccountName),
		BankName: strings.TrimSpace(req.BankName), BankAccount: strings.TrimSpace(req.BankAccount),
		BankProvince: strings.TrimSpace(req.BankProvince), BankCity: strings.TrimSpace(req.BankCity),
		Phone: strings.TrimSpace(req.Phone), AccountType: accountType,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	}
	if err := s.vendorDAO.Upsert(ctx, p); err != nil {
		return err
	}
	c, err := s.customerDAO.GetByID(ctx, req.CustomerID)
	if err != nil {
		return err
	}
	c.VendorProfileDone = 1
	return s.customerDAO.Update(ctx, c)
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

func (s *opsCustomerService) UpdateReportSettings(ctx context.Context, req *model.UpdateOpsCustomerReportReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.ID); err != nil {
		return err
	}
	code := strings.TrimSpace(req.ReportCode)
	if code == "" {
		return fmt.Errorf("组织编码不能为空")
	}
	existing, err := s.customerDAO.GetByReportCode(ctx, code)
	if err == nil && existing != nil && existing.ID != req.ID {
		return fmt.Errorf("组织编码已被其他客户使用")
	}
	return s.customerDAO.UpdateReportSettings(ctx, req.ID, code, req.ReportEnabled)
}

func (s *opsCustomerService) RotateReportSecret(ctx context.Context, id int) (*model.RotateOpsCustomerReportSecretResp, error) {
	customer, err := s.customerDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	secret := opsUtils.GenerateReportSecret()
	hash, err := userutils.HashPassword(secret)
	if err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}
	code := strings.TrimSpace(customer.ReportCode)
	if code == "" {
		code = opsUtils.GenerateReportCode(customer.ID)
		if err := s.customerDAO.UpdateReportSettings(ctx, id, code, model.OpsReportEnabledYes); err != nil {
			return nil, err
		}
	}
	if err := s.customerDAO.UpdateReportSecret(ctx, id, hash); err != nil {
		return nil, err
	}
	return &model.RotateOpsCustomerReportSecretResp{
		ReportCode:   code,
		ReportSecret: secret,
		Message:      "请立即保存密钥，关闭后将无法再次查看",
	}, nil
}
