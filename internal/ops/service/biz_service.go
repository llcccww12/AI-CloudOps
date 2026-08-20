package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	workorderService "github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type OpsBizService interface {
	CreateTrial(ctx context.Context, req *model.CreateOpsTrialReq) error
	UpdateTrial(ctx context.Context, req *model.UpdateOpsTrialReq) error
	DeleteTrial(ctx context.Context, id int) error
	GetTrial(ctx context.Context, id int) (*model.OpsTrial, error)
	ListTrial(ctx context.Context, req *model.ListOpsTrialReq) (*model.ListResp[*model.OpsTrial], error)
	SubmitTrial(ctx context.Context, id, operatorID int, operatorName string) error

	CreateContract(ctx context.Context, req *model.CreateOpsContractReq) error
	UpdateContract(ctx context.Context, req *model.UpdateOpsContractReq) error
	DeleteContract(ctx context.Context, id int) error
	GetContract(ctx context.Context, id int) (*model.OpsContract, error)
	ListContract(ctx context.Context, req *model.ListOpsContractReq) (*model.ListResp[*model.OpsContract], error)

	CreateActivation(ctx context.Context, req *model.CreateOpsActivationReq) error
	UpdateActivation(ctx context.Context, req *model.UpdateOpsActivationReq) error
	FeedbackActivation(ctx context.Context, req *model.FeedbackOpsActivationReq) error
	DeleteActivation(ctx context.Context, id int) error
	GetActivation(ctx context.Context, id int) (*model.OpsActivation, error)
	ListActivation(ctx context.Context, req *model.ListOpsActivationReq) (*model.ListResp[*model.OpsActivation], error)
	SubmitActivation(ctx context.Context, id, operatorID int, operatorName string) error

	StartCustomerLifecycle(ctx context.Context, customerID, operatorID int, operatorName string) (*model.OpsCustomerLifecycleWorkorder, error)
	ListCustomerLifecycle(ctx context.Context, customerID int) ([]*model.OpsCustomerLifecycleWorkorder, error)
	GetLifecycleApproveContext(ctx context.Context, instanceID int) (*model.OpsLifecycleApproveContext, error)
	ApproveLifecycleNode(ctx context.Context, req *model.OpsLifecycleApproveReq, operatorID int, operatorName string) error

	OnWorkorderTerminal(ctx context.Context, instanceID int, status int8) error
	SyncPendingApprovals(ctx context.Context) error

	ListContractItems(ctx context.Context, contractID int) (*model.ListResp[*model.OpsContractItem], error)
	CreateContractItem(ctx context.Context, req *model.CreateOpsContractItemReq) error
	DeleteContractItem(ctx context.Context, id int) error
}

type opsBizService struct {
	trialDAO      dao.OpsTrialDAO
	contractDAO   dao.OpsContractDAO
	activationDAO dao.OpsActivationDAO
	approvalDAO   dao.OpsApprovalLinkDAO
	customerDAO   dao.OpsCustomerDAO
	followupDAO   dao.OpsFollowupDAO
	settlementDAO dao.OpsSettlementDAO
	invoiceDAO    dao.OpsInvoiceDAO
	paymentDAO    dao.OpsPaymentDAO
	itemDAO       dao.OpsContractItemDAO
	processDao    workorderDao.WorkorderProcessDAO
	instanceSvc   workorderService.InstanceService
	logger        *zap.Logger
}

func NewOpsBizService(
	trialDAO dao.OpsTrialDAO,
	contractDAO dao.OpsContractDAO,
	activationDAO dao.OpsActivationDAO,
	approvalDAO dao.OpsApprovalLinkDAO,
	customerDAO dao.OpsCustomerDAO,
	followupDAO dao.OpsFollowupDAO,
	settlementDAO dao.OpsSettlementDAO,
	invoiceDAO dao.OpsInvoiceDAO,
	paymentDAO dao.OpsPaymentDAO,
	itemDAO dao.OpsContractItemDAO,
	processDao workorderDao.WorkorderProcessDAO,
	instanceSvc workorderService.InstanceService,
	logger *zap.Logger,
) OpsBizService {
	return &opsBizService{
		trialDAO: trialDAO, contractDAO: contractDAO, activationDAO: activationDAO,
		approvalDAO: approvalDAO, customerDAO: customerDAO, followupDAO: followupDAO,
		settlementDAO: settlementDAO, invoiceDAO: invoiceDAO, paymentDAO: paymentDAO,
		itemDAO: itemDAO, processDao: processDao, instanceSvc: instanceSvc, logger: logger,
	}
}

func (s *opsBizService) CreateTrial(ctx context.Context, req *model.CreateOpsTrialReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.CustomerID); err != nil {
		return err
	}
	return s.trialDAO.Create(ctx, &model.OpsTrial{
		CustomerID: req.CustomerID, Title: strings.TrimSpace(req.Title), DemandType: req.DemandType,
		ResourceScale: req.ResourceScale, Purpose: req.Purpose, Status: model.OpsTrialStatusDraft,
		PlanStartAt: req.PlanStartAt, PlanEndAt: req.PlanEndAt,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	})
}

func (s *opsBizService) UpdateTrial(ctx context.Context, req *model.UpdateOpsTrialReq) error {
	return s.trialDAO.Update(ctx, &model.OpsTrial{
		Model: model.Model{ID: req.ID}, Title: strings.TrimSpace(req.Title), DemandType: req.DemandType,
		ResourceScale: req.ResourceScale, Purpose: req.Purpose, PlanStartAt: req.PlanStartAt,
		PlanEndAt: req.PlanEndAt, Evaluation: req.Evaluation, ConvertIntent: req.ConvertIntent,
	})
}
func (s *opsBizService) DeleteTrial(ctx context.Context, id int) error {
	return s.trialDAO.Delete(ctx, id)
}
func (s *opsBizService) GetTrial(ctx context.Context, id int) (*model.OpsTrial, error) {
	item, err := s.trialDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.attachTrialApproval(ctx, item)
	return item, nil
}
func (s *opsBizService) ListTrial(ctx context.Context, req *model.ListOpsTrialReq) (*model.ListResp[*model.OpsTrial], error) {
	items, total, err := s.trialDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		s.attachTrialApproval(ctx, item)
	}
	return &model.ListResp[*model.OpsTrial]{Items: items, Total: total}, nil
}

func (s *opsBizService) attachTrialApproval(ctx context.Context, item *model.OpsTrial) {
	if item == nil {
		return
	}
	link, err := s.approvalDAO.GetByBiz(ctx, model.OpsApprovalBizTrial, item.ID)
	if err == nil && link != nil {
		item.WorkorderInstanceID = link.WorkorderInstanceID
	}
}

func (s *opsBizService) SubmitTrial(ctx context.Context, id, operatorID int, operatorName string) error {
	trial, err := s.trialDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if trial.Status != model.OpsTrialStatusDraft && trial.Status != model.OpsTrialStatusRejected {
		return fmt.Errorf("当前状态不可提交审批")
	}
	templateID := viper.GetInt("ops.trial_workorder_template_id")
	if templateID <= 0 {
		// 未配置工单模板时本地直接通过，便于开箱使用
		if err := s.trialDAO.UpdateStatus(ctx, id, model.OpsTrialStatusApproved); err != nil {
			return err
		}
		_ = s.customerDAO.UpdateStage(ctx, trial.CustomerID, model.OpsCustomerStageTrial, "")
		return nil
	}
	title := fmt.Sprintf("试用审批-%s", trial.Title)
	req := &model.CreateWorkorderInstanceFromTemplateReq{
		Title: title, Description: trial.Purpose, Priority: model.PriorityNormal,
		FormData: model.JSONMap{
			"ops_biz_type": model.OpsApprovalBizTrial,
			"ops_biz_id":   trial.ID,
			"customer_id":  trial.CustomerID,
		},
		OperatorID: operatorID, OperatorName: operatorName,
	}
	if instanceID, err := s.instanceSvc.CreateInstanceFromTemplate(ctx, templateID, req); err != nil {
		return fmt.Errorf("创建审批工单失败: %w", err)
	} else if err := s.bindWorkorder(ctx, model.OpsApprovalBizTrial, id, instanceID); err != nil {
		return err
	}
	if err := s.trialDAO.UpdateStatus(ctx, id, model.OpsTrialStatusPending); err != nil {
		return err
	}
	return nil
}

func (s *opsBizService) bindWorkorder(ctx context.Context, bizType string, bizID, instanceID int) error {
	if instanceID <= 0 {
		return fmt.Errorf("工单实例ID无效")
	}
	link := &model.OpsApprovalLink{
		BizType: bizType, BizID: bizID, WorkorderInstanceID: instanceID, Status: model.OpsApprovalLinkPending,
	}
	if err := s.approvalDAO.Create(ctx, link); err != nil {
		if !isDuplicate(err) {
			return err
		}
	}
	return nil
}

func (s *opsBizService) linkLatestWorkorder(ctx context.Context, bizType string, bizID int, title string) error {
	list, err := s.instanceSvc.ListInstance(ctx, &model.ListWorkorderInstanceReq{
		ListReq: model.ListReq{Page: 1, Size: 10, Search: title},
	})
	if err != nil || list == nil || len(list.Items) == 0 {
		s.logger.Warn("审批工单已创建但未能绑定 link", zap.String("title", title), zap.Error(err))
		return nil
	}
	inst := list.Items[0]
	return s.bindWorkorder(ctx, bizType, bizID, inst.ID)
}

func isDuplicate(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate"))
}

func instanceStatusLabel(status int8) string {
	switch status {
	case model.InstanceStatusDraft:
		return "草稿"
	case model.InstanceStatusPending:
		return "待处理"
	case model.InstanceStatusProcessing:
		return "处理中"
	case model.InstanceStatusCompleted:
		return "已完成"
	case model.InstanceStatusRejected:
		return "已拒绝"
	case model.InstanceStatusCancelled:
		return "已取消"
	default:
		return fmt.Sprintf("%d", status)
	}
}

func (s *opsBizService) CreateContract(ctx context.Context, req *model.CreateOpsContractReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.CustomerID); err != nil {
		return err
	}
	term := req.PaymentTermDays
	if term <= 0 {
		term = 30
	}
	autoRenew := req.AutoRenew
	if autoRenew == 0 {
		autoRenew = 2
	}
	return s.contractDAO.Create(ctx, &model.OpsContract{
		CustomerID: req.CustomerID, TrialID: req.TrialID, Type: req.Type, Title: strings.TrimSpace(req.Title),
		ProductType: req.ProductType, BillingMode: req.BillingMode, UnitPrice: req.UnitPrice, BillingCycle: req.BillingCycle,
		PaymentMethod: req.PaymentMethod, PaymentTermDays: term, StartAt: req.StartAt, EndAt: req.EndAt, AutoRenew: autoRenew,
		Status: model.OpsContractStatusDraft, Remark: req.Remark,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	})
}

func (s *opsBizService) UpdateContract(ctx context.Context, req *model.UpdateOpsContractReq) error {
	status := req.Status
	if status == "" {
		status = model.OpsContractStatusDraft
	}
	return s.contractDAO.Update(ctx, &model.OpsContract{
		Model: model.Model{ID: req.ID}, Title: strings.TrimSpace(req.Title),
		ProductType: req.ProductType, BillingMode: req.BillingMode,
		UnitPrice: req.UnitPrice, BillingCycle: req.BillingCycle, PaymentMethod: req.PaymentMethod,
		PaymentTermDays: req.PaymentTermDays,
		StartAt: req.StartAt, EndAt: req.EndAt, AutoRenew: req.AutoRenew, Status: status, Remark: req.Remark,
	})
}
func (s *opsBizService) DeleteContract(ctx context.Context, id int) error {
	return s.contractDAO.Delete(ctx, id)
}
func (s *opsBizService) GetContract(ctx context.Context, id int) (*model.OpsContract, error) {
	return s.contractDAO.GetByID(ctx, id)
}
func (s *opsBizService) ListContract(ctx context.Context, req *model.ListOpsContractReq) (*model.ListResp[*model.OpsContract], error) {
	items, total, err := s.contractDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsContract]{Items: items, Total: total}, nil
}

func (s *opsBizService) CreateActivation(ctx context.Context, req *model.CreateOpsActivationReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.CustomerID); err != nil {
		return err
	}
	if _, err := s.contractDAO.GetByID(ctx, req.ContractID); err != nil {
		return err
	}
	return s.activationDAO.Create(ctx, &model.OpsActivation{
		CustomerID: req.CustomerID, ContractID: req.ContractID, Title: strings.TrimSpace(req.Title),
		ResourceSummary: req.ResourceSummary, Purpose: req.Purpose, Status: model.OpsActivationStatusDraft,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	})
}

func (s *opsBizService) UpdateActivation(ctx context.Context, req *model.UpdateOpsActivationReq) error {
	return s.activationDAO.Update(ctx, &model.OpsActivation{
		Model: model.Model{ID: req.ID}, Title: strings.TrimSpace(req.Title),
		ResourceSummary: req.ResourceSummary, Purpose: req.Purpose,
		FeedbackAccount: req.FeedbackAccount, FeedbackTenant: req.FeedbackTenant,
		FeedbackEndpoint: req.FeedbackEndpoint, FeedbackRemark: req.FeedbackRemark,
	})
}

func (s *opsBizService) FeedbackActivation(ctx context.Context, req *model.FeedbackOpsActivationReq) error {
	act, err := s.activationDAO.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	now := time.Now()
	act.FeedbackAccount = strings.TrimSpace(req.FeedbackAccount)
	act.FeedbackTenant = strings.TrimSpace(req.FeedbackTenant)
	act.FeedbackEndpoint = strings.TrimSpace(req.FeedbackEndpoint)
	act.FeedbackRemark = strings.TrimSpace(req.FeedbackRemark)
	act.Status = model.OpsActivationStatusActive
	act.ActivatedAt = &now
	if err := s.activationDAO.Update(ctx, act); err != nil {
		return err
	}
	return s.activationDAO.UpdateStatus(ctx, act.ID, model.OpsActivationStatusActive)
}
func (s *opsBizService) DeleteActivation(ctx context.Context, id int) error {
	return s.activationDAO.Delete(ctx, id)
}
func (s *opsBizService) GetActivation(ctx context.Context, id int) (*model.OpsActivation, error) {
	item, err := s.activationDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.attachActivationApproval(ctx, item)
	return item, nil
}
func (s *opsBizService) ListActivation(ctx context.Context, req *model.ListOpsActivationReq) (*model.ListResp[*model.OpsActivation], error) {
	items, total, err := s.activationDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		s.attachActivationApproval(ctx, item)
	}
	return &model.ListResp[*model.OpsActivation]{Items: items, Total: total}, nil
}

func (s *opsBizService) attachActivationApproval(ctx context.Context, item *model.OpsActivation) {
	if item == nil {
		return
	}
	link, err := s.approvalDAO.GetByBiz(ctx, model.OpsApprovalBizActivation, item.ID)
	if err == nil && link != nil {
		item.WorkorderInstanceID = link.WorkorderInstanceID
	}
}

func (s *opsBizService) SubmitActivation(ctx context.Context, id, operatorID int, operatorName string) error {
	act, err := s.activationDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if act.Status != model.OpsActivationStatusDraft && act.Status != model.OpsActivationStatusRejected {
		return fmt.Errorf("当前状态不可提交审批")
	}
	templateID := viper.GetInt("ops.activation_workorder_template_id")
	if templateID <= 0 {
		if err := s.activationDAO.UpdateStatus(ctx, id, model.OpsActivationStatusApproved); err != nil {
			return err
		}
		_ = s.contractDAO.UpdateStatus(ctx, act.ContractID, model.OpsContractStatusActive)
		_ = s.customerDAO.UpdateStage(ctx, act.CustomerID, model.OpsCustomerStageFormal, "")
		return nil
	}
	title := fmt.Sprintf("开通审批-%s", act.Title)
	req := &model.CreateWorkorderInstanceFromTemplateReq{
		Title: title, Description: act.Purpose, Priority: model.PriorityNormal,
		FormData: model.JSONMap{
			"ops_biz_type": model.OpsApprovalBizActivation,
			"ops_biz_id":   act.ID,
			"customer_id":  act.CustomerID,
		},
		OperatorID: operatorID, OperatorName: operatorName,
	}
	if instanceID, err := s.instanceSvc.CreateInstanceFromTemplate(ctx, templateID, req); err != nil {
		return fmt.Errorf("创建审批工单失败: %w", err)
	} else if err := s.bindWorkorder(ctx, model.OpsApprovalBizActivation, id, instanceID); err != nil {
		return err
	}
	if err := s.activationDAO.UpdateStatus(ctx, id, model.OpsActivationStatusPending); err != nil {
		return err
	}
	return nil
}

func (s *opsBizService) StartCustomerLifecycle(ctx context.Context, customerID, operatorID int, operatorName string) (*model.OpsCustomerLifecycleWorkorder, error) {
	customer, err := s.customerDAO.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	templateID := viper.GetInt("ops.lifecycle_workorder_template_id")
	if templateID <= 0 {
		return nil, fmt.Errorf("未配置运营全流程工单模板（ops.lifecycle_workorder_template_id）")
	}
	if existing, err := s.findCustomerLifecycle(ctx, customerID); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("该客户已绑定运营全流程工单（工单ID=%d，状态：%s），一个客户项目仅允许一个工单，请直接查看进度", existing.WorkorderInstanceID, existing.Status)
	}
	title := fmt.Sprintf("运营全流程-%s-%d-%d", customer.Name, customer.ID, time.Now().Unix())
	stageForForm := customer.Stage
	if stageForForm != model.OpsCustomerStageFormal && stageForForm != model.OpsCustomerStageClosed {
		stageForForm = model.OpsCustomerStageTrial
	}
	req := &model.CreateWorkorderInstanceFromTemplateReq{
		Title: title,
		Description: fmt.Sprintf("客户「%s」从试用到正式合同与回款的运营全流程", customer.Name),
		Priority: model.PriorityNormal,
		FormData: model.JSONMap{
			"ops_biz_type":   model.OpsApprovalBizCustomerLifecycle,
			"ops_biz_id":     customer.ID,
			"customer_id":    customer.ID,
			"customer_name":  customer.Name,
			"customer_stage": stageForForm,
			"owner_name":     customer.OwnerName,
			"contact_name":   customer.ContactName,
			"contact_phone":  customer.ContactPhone,
		},
		OperatorID: operatorID, OperatorName: operatorName,
	}
	instanceID, err := s.instanceSvc.CreateInstanceFromTemplate(ctx, templateID, req)
	if err != nil {
		return nil, fmt.Errorf("创建运营全流程工单失败: %w", err)
	}
	if err := s.bindWorkorder(ctx, model.OpsApprovalBizCustomerLifecycle, customerID, instanceID); err != nil {
		return nil, err
	}
	// 自动提交，进入流程第一个审批节点
	if err := s.instanceSvc.SubmitInstance(ctx, instanceID, operatorID, operatorName); err != nil {
		s.logger.Warn("运营全流程工单已创建但自动提交失败，可在工单中心手动提交",
			zap.Int("instanceID", instanceID), zap.Error(err))
	}
	// 发起全流程后客户进入试用阶段（正式/闭环不回退）
	if customer.Stage != model.OpsCustomerStageFormal && customer.Stage != model.OpsCustomerStageClosed {
		if err := s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageTrial, ""); err != nil {
			s.logger.Warn("发起全流程后更新客户试用阶段失败", zap.Int("customerID", customerID), zap.Error(err))
		} else {
			customer.Stage = model.OpsCustomerStageTrial
		}
	}
	return &model.OpsCustomerLifecycleWorkorder{
		CustomerID:          customerID,
		CustomerName:        customer.Name,
		WorkorderInstanceID: instanceID,
		Title:               req.Title,
		Status:              model.OpsApprovalLinkPending,
		LinkStatus:          model.OpsApprovalLinkPending,
	}, nil
}

func (s *opsBizService) ListCustomerLifecycle(ctx context.Context, customerID int) ([]*model.OpsCustomerLifecycleWorkorder, error) {
	if _, err := s.customerDAO.GetByID(ctx, customerID); err != nil {
		return nil, err
	}
	links, err := s.approvalDAO.ListByBiz(ctx, model.OpsApprovalBizCustomerLifecycle, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]*model.OpsCustomerLifecycleWorkorder, 0, len(links))
	for _, link := range links {
		inst, err := s.instanceSvc.GetInstance(ctx, link.WorkorderInstanceID)
		if err != nil || inst == nil {
			continue
		}
		item := &model.OpsCustomerLifecycleWorkorder{
			CustomerID:          customerID,
			WorkorderInstanceID: link.WorkorderInstanceID,
			LinkStatus:          link.Status,
			CreatedAt:           link.CreatedAt,
			Title:               inst.Title,
			InstanceStatus:      inst.Status,
			SerialNumber:        inst.SerialNumber,
			Status:              instanceStatusLabel(inst.Status),
		}
		if inst.CurrentStepID != nil {
			item.CurrentStepID = *inst.CurrentStepID
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *opsBizService) findCustomerLifecycle(ctx context.Context, customerID int) (*model.OpsCustomerLifecycleWorkorder, error) {
	items, err := s.ListCustomerLifecycle(ctx, customerID)
	if err != nil {
		return nil, err
	}
	// 一客户一工单：取仍存在的最新一条（含拒绝/完成/取消）
	for _, item := range items {
		if item != nil && item.WorkorderInstanceID > 0 {
			return item, nil
		}
	}
	return nil, nil
}

func (s *opsBizService) SyncPendingApprovals(ctx context.Context) error {
	links, err := s.approvalDAO.ListPending(ctx)
	if err != nil {
		return err
	}
	for _, link := range links {
		inst, err := s.instanceSvc.GetInstance(ctx, link.WorkorderInstanceID)
		if err != nil || inst == nil {
			// 工单已删：静默收尾，避免定时扫描刷 ERROR
			s.logger.Debug("审批链接对应工单不存在，标记取消",
				zap.Int("link_id", link.ID),
				zap.Int("instance_id", link.WorkorderInstanceID),
				zap.Error(err),
			)
			_ = s.approvalDAO.UpdateStatus(ctx, link.ID, model.OpsApprovalLinkCancelled)
			continue
		}
		if inst.Status == model.InstanceStatusCompleted || inst.Status == model.InstanceStatusRejected {
			_ = s.OnWorkorderTerminal(ctx, inst.ID, inst.Status)
		}
	}
	return nil
}

func (s *opsBizService) OnWorkorderTerminal(ctx context.Context, instanceID int, status int8) error {
	link, err := s.approvalDAO.GetByInstanceID(ctx, instanceID)
	if err != nil {
		return nil
	}
	approved := status == model.InstanceStatusCompleted
	rejected := status == model.InstanceStatusRejected
	if !approved && !rejected {
		return nil
	}
	linkStatus := model.OpsApprovalLinkApproved
	if rejected {
		linkStatus = model.OpsApprovalLinkRejected
	}
	_ = s.approvalDAO.UpdateStatus(ctx, link.ID, linkStatus)

	switch link.BizType {
	case model.OpsApprovalBizTrial:
		trial, err := s.trialDAO.GetByID(ctx, link.BizID)
		if err != nil {
			return err
		}
		if approved {
			_ = s.trialDAO.UpdateStatus(ctx, trial.ID, model.OpsTrialStatusApproved)
			_ = s.customerDAO.UpdateStage(ctx, trial.CustomerID, model.OpsCustomerStageTrial, "")
		} else {
			_ = s.trialDAO.UpdateStatus(ctx, trial.ID, model.OpsTrialStatusRejected)
		}
	case model.OpsApprovalBizActivation:
		act, err := s.activationDAO.GetByID(ctx, link.BizID)
		if err != nil {
			return err
		}
		if approved {
			_ = s.activationDAO.UpdateStatus(ctx, act.ID, model.OpsActivationStatusApproved)
			_ = s.customerDAO.UpdateStage(ctx, act.CustomerID, model.OpsCustomerStageFormal, "")
		} else {
			_ = s.activationDAO.UpdateStatus(ctx, act.ID, model.OpsActivationStatusRejected)
		}
	case model.OpsApprovalBizCustomerLifecycle:
		// 台账已在各节点审批时写入；此处仅同步绑定状态
		if approved {
			_ = s.customerDAO.UpdateStage(ctx, link.BizID, model.OpsCustomerStageFormal, "")
		}
	}
	return nil
}

func (s *opsBizService) ListContractItems(ctx context.Context, contractID int) (*model.ListResp[*model.OpsContractItem], error) {
	if s.itemDAO == nil {
		return &model.ListResp[*model.OpsContractItem]{}, nil
	}
	items, err := s.itemDAO.ListByContract(ctx, contractID)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsContractItem]{Items: items, Total: int64(len(items))}, nil
}

func (s *opsBizService) CreateContractItem(ctx context.Context, req *model.CreateOpsContractItemReq) error {
	if _, err := s.contractDAO.GetByID(ctx, req.ContractID); err != nil {
		return err
	}
	itemType := req.ItemType
	if itemType == "" {
		itemType = model.OpsContractItemTypeAddOn
	}
	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}
	amount := req.Amount
	if amount <= 0 {
		amount = qty * req.UnitPrice
	}
	return s.itemDAO.Create(ctx, &model.OpsContractItem{
		ContractID: req.ContractID, ItemType: itemType, Name: strings.TrimSpace(req.Name),
		ProductType: req.ProductType, Quantity: qty, UnitPrice: req.UnitPrice, Amount: amount, Remark: req.Remark,
	})
}

func (s *opsBizService) DeleteContractItem(ctx context.Context, id int) error {
	return s.itemDAO.Delete(ctx, id)
}
