package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	opsUtils "github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
)

func (s *opsBizService) GetLifecycleApproveContext(ctx context.Context, instanceID int) (*model.OpsLifecycleApproveContext, error) {
	instance, _, customerID, err := s.loadLifecycleInstance(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	customer, err := s.customerDAO.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}

	step, err := s.instanceSvc.GetCurrentStep(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("获取当前节点失败: %w", err)
	}
	nodeKey := normalizeLifecycleStep(step)

	definition, err := s.loadProcessDefinition(ctx, instance.ProcessID)
	if err != nil {
		return nil, err
	}
	nextStep := getNextProcessStep(step, definition)
	needsNext := nextStep != nil && nextStep.Type != model.ProcessStepTypeEnd

	out := &model.OpsLifecycleApproveContext{
		InstanceID:        instanceID,
		CustomerID:        customer.ID,
		CustomerName:      customer.Name,
		CurrentStepID:     step.ID,
		CurrentStepName:   step.Name,
		NodeKey:           nodeKey,
		NeedsNextAssignee: needsNext,
		FormData:          instance.FormData,
	}
	if needsNext {
		out.NextStepName = nextStep.Name
	}

	if trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	}); err == nil && len(trials) > 0 {
		out.LatestTrial = trials[0]
	}
	if contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID, Type: model.OpsContractTypeFormal,
	}); err == nil && len(contracts) > 0 {
		out.LatestContract = contracts[0]
	} else if contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	}); err == nil && len(contracts) > 0 {
		out.LatestContract = contracts[0]
	}
	if settlements, _, err := s.settlementDAO.List(ctx, &model.ListOpsSettlementReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	}); err == nil && len(settlements) > 0 {
		out.LatestSettlement = settlements[0]
	}
	return out, nil
}

func (s *opsBizService) ApproveLifecycleNode(ctx context.Context, req *model.OpsLifecycleApproveReq, operatorID int, operatorName string) error {
	if req == nil {
		return fmt.Errorf("请求无效")
	}
	_, _, customerID, err := s.loadLifecycleInstance(ctx, req.InstanceID)
	if err != nil {
		return err
	}
	step, err := s.instanceSvc.GetCurrentStep(ctx, req.InstanceID)
	if err != nil {
		return fmt.Errorf("获取当前节点失败: %w", err)
	}
	nodeKey := normalizeLifecycleStep(step)
	if nodeKey == "" {
		return fmt.Errorf("无法识别当前节点「%s」，请联系管理员检查流程定义", step.Name)
	}

	payload := req.Payload
	if payload == nil {
		payload = model.JSONMap{}
	}

	if err := s.applyLifecycleNodePayload(ctx, nodeKey, customerID, payload, operatorID, operatorName); err != nil {
		return err
	}

	assigneeID := 0
	if req.AssigneeID != nil {
		assigneeID = *req.AssigneeID
	}
	if err := s.instanceSvc.ApproveInstance(ctx, req.InstanceID, operatorID, operatorName, req.Comment, assigneeID, req.AttachmentIDs); err != nil {
		return err
	}

	inst, err := s.instanceSvc.GetInstance(ctx, req.InstanceID)
	if err == nil && inst != nil {
		_ = s.OnWorkorderTerminal(ctx, req.InstanceID, inst.Status)
	}
	return nil
}

func (s *opsBizService) applyLifecycleNodePayload(ctx context.Context, nodeKey string, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
	switch nodeKey {
	case model.OpsLifecycleStepIntent:
		return s.applyIntentNode(ctx, customerID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepTrial:
		return s.applyTrialNode(ctx, customerID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepTrialAccept:
		return s.applyTrialAcceptNode(ctx, customerID, payload)
	case model.OpsLifecycleStepContract:
		return s.applyContractNode(ctx, customerID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepSettlement:
		return s.applySettlementNode(ctx, customerID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepPayment:
		return s.applyPaymentNode(ctx, customerID, payload, operatorID, operatorName)
	default:
		return fmt.Errorf("未支持的节点: %s", nodeKey)
	}
}

func (s *opsBizService) applyIntentNode(ctx context.Context, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
	content := payloadString(payload, "content")
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("请填写意向确认说明")
	}
	if err := s.followupDAO.Create(ctx, &model.OpsFollowup{
		CustomerID: customerID, Type: "intent", Content: strings.TrimSpace(content),
		NextPlan:     payloadString(payload, "next_plan"),
		OperatorID:   operatorID,
		OperatorName: operatorName,
	}); err != nil {
		return fmt.Errorf("写入跟进记录失败: %w", err)
	}
	_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageIntent, "")
	return nil
}

func (s *opsBizService) applyTrialNode(ctx context.Context, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
	title := strings.TrimSpace(payloadString(payload, "title"))
	if title == "" {
		return fmt.Errorf("请填写试用标题")
	}
	planStart, err := parsePayloadTime(payload, "plan_start_at")
	if err != nil {
		return err
	}
	planEnd, err := parsePayloadTime(payload, "plan_end_at")
	if err != nil {
		return err
	}
	trial := &model.OpsTrial{
		CustomerID:    customerID,
		Title:         title,
		DemandType:    payloadString(payload, "demand_type"),
		ResourceScale: payloadString(payload, "resource_scale"),
		Purpose:       payloadString(payload, "purpose"),
		Status:        model.OpsTrialStatusApproved,
		PlanStartAt:   planStart,
		PlanEndAt:     planEnd,
		OperatorID:    operatorID,
		OperatorName:  operatorName,
	}
	if err := s.trialDAO.Create(ctx, trial); err != nil {
		return fmt.Errorf("创建试用单失败: %w", err)
	}
	_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageTrial, "")
	return nil
}

func (s *opsBizService) applyTrialAcceptNode(ctx context.Context, customerID int, payload model.JSONMap) error {
	evaluation := strings.TrimSpace(payloadString(payload, "evaluation"))
	convertIntent := strings.TrimSpace(payloadString(payload, "convert_intent"))
	if evaluation == "" {
		return fmt.Errorf("请填写试用验收评价")
	}
	if convertIntent == "" {
		return fmt.Errorf("请选择转正意向")
	}
	trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	})
	if err != nil || len(trials) == 0 {
		return fmt.Errorf("未找到该客户的试用单，请先完成「试用审批」节点登记")
	}
	trial := trials[0]
	trial.Evaluation = evaluation
	trial.ConvertIntent = convertIntent
	if trial.Title == "" {
		trial.Title = fmt.Sprintf("试用-%d", trial.ID)
	}
	if err := s.trialDAO.Update(ctx, trial); err != nil {
		return fmt.Errorf("更新试用验收失败: %w", err)
	}
	_ = s.trialDAO.UpdateStatus(ctx, trial.ID, model.OpsTrialStatusEnded)
	return nil
}

func (s *opsBizService) applyContractNode(ctx context.Context, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
	title := strings.TrimSpace(payloadString(payload, "title"))
	if title == "" {
		return fmt.Errorf("请填写合同标题")
	}
	contractType := payloadString(payload, "type")
	if contractType == "" {
		contractType = model.OpsContractTypeFormal
	}
	startAt, err := parsePayloadTime(payload, "start_at")
	if err != nil {
		return err
	}
	endAt, err := parsePayloadTime(payload, "end_at")
	if err != nil {
		return err
	}
	var trialID *int
	if tid := payloadInt(payload, "trial_id"); tid > 0 {
		trialID = &tid
	} else if trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	}); err == nil && len(trials) > 0 {
		id := trials[0].ID
		trialID = &id
	}
	term := payloadInt(payload, "payment_term_days")
	if term <= 0 {
		term = 30
	}
	autoRenew := int8(payloadInt(payload, "auto_renew"))
	if autoRenew == 0 {
		autoRenew = 2
	}
	contract := &model.OpsContract{
		CustomerID:      customerID,
		TrialID:         trialID,
		Type:            contractType,
		Title:           title,
		BillingMode:     payloadString(payload, "billing_mode"),
		UnitPrice:       payloadFloat(payload, "unit_price"),
		BillingCycle:    payloadString(payload, "billing_cycle"),
		PaymentTermDays: term,
		StartAt:         startAt,
		EndAt:           endAt,
		AutoRenew:       autoRenew,
		Status:          model.OpsContractStatusActive,
		Remark:          payloadString(payload, "remark"),
		OperatorID:      operatorID,
		OperatorName:    operatorName,
	}
	if err := s.contractDAO.Create(ctx, contract); err != nil {
		return fmt.Errorf("创建合同失败: %w", err)
	}
	actTitle := strings.TrimSpace(payloadString(payload, "activation_title"))
	if actTitle == "" {
		actTitle = "开通-" + title
	}
	now := time.Now()
	act := &model.OpsActivation{
		CustomerID:      customerID,
		ContractID:      contract.ID,
		Title:           actTitle,
		ResourceSummary: payloadString(payload, "resource_summary"),
		Purpose:         payloadString(payload, "activation_purpose"),
		Status:          model.OpsActivationStatusActive,
		ActivatedAt:     &now,
		OperatorID:      operatorID,
		OperatorName:    operatorName,
	}
	if err := s.activationDAO.Create(ctx, act); err != nil {
		return fmt.Errorf("创建开通单失败: %w", err)
	}
	_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageFormal, "")
	return nil
}

func (s *opsBizService) applySettlementNode(ctx context.Context, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
	title := strings.TrimSpace(payloadString(payload, "title"))
	if title == "" {
		return fmt.Errorf("请填写结算标题")
	}
	contractID := payloadInt(payload, "contract_id")
	if contractID <= 0 {
		contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
			ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
		})
		if err != nil || len(contracts) == 0 {
			return fmt.Errorf("未找到合同，请先完成「合同与开通」节点登记")
		}
		contractID = contracts[0].ID
	}
	periodStart, err := parsePayloadTime(payload, "period_start")
	if err != nil {
		return err
	}
	periodEnd, err := parsePayloadTime(payload, "period_end")
	if err != nil {
		return err
	}
	dueAt, err := parsePayloadTime(payload, "due_at")
	if err != nil {
		return err
	}
	st := &model.OpsSettlement{
		CustomerID:   customerID,
		ContractID:   contractID,
		Title:        title,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		Amount:       payloadFloat(payload, "amount"),
		DueAt:        dueAt,
		Status:       model.OpsSettlementStatusConfirmed,
		Remark:       payloadString(payload, "remark"),
		OperatorID:   operatorID,
		OperatorName: operatorName,
	}
	if err := s.settlementDAO.Create(ctx, st); err != nil {
		return fmt.Errorf("创建结算单失败: %w", err)
	}
	return nil
}

func (s *opsBizService) applyPaymentNode(ctx context.Context, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
	settlementID := payloadInt(payload, "settlement_id")
	if settlementID <= 0 {
		items, _, err := s.settlementDAO.List(ctx, &model.ListOpsSettlementReq{
			ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
		})
		if err != nil || len(items) == 0 {
			return fmt.Errorf("未找到结算单，请先完成「结算确认」节点登记")
		}
		settlementID = items[0].ID
	}
	amount := payloadFloat(payload, "amount")
	if amount <= 0 {
		return fmt.Errorf("请填写回款金额")
	}
	issuedAt, err := parsePayloadTime(payload, "issued_at")
	if err != nil {
		return err
	}
	invAmount := payloadFloat(payload, "invoice_amount")
	if invAmount <= 0 {
		invAmount = amount
	}
	inv := &model.OpsInvoice{
		CustomerID:   customerID,
		SettlementID: settlementID,
		InvoiceNo:    payloadString(payload, "invoice_no"),
		InvoiceType:  payloadString(payload, "invoice_type"),
		Amount:       invAmount,
		IssuedAt:     issuedAt,
		Status:       model.OpsInvoiceStatusIssued,
		OperatorID:   operatorID,
		OperatorName: operatorName,
	}
	if err := s.invoiceDAO.Create(ctx, inv); err != nil {
		return fmt.Errorf("创建发票失败: %w", err)
	}
	_ = s.settlementDAO.UpdateStatus(ctx, settlementID, model.OpsSettlementStatusInvoiced)

	paidAt, err := parsePayloadTime(payload, "paid_at")
	if err != nil {
		return err
	}
	payment := &model.OpsPayment{
		CustomerID:   customerID,
		SettlementID: settlementID,
		Amount:       amount,
		PaidAt:       paidAt,
		BankRef:      payloadString(payload, "bank_ref"),
		Status:       model.OpsPaymentStatusMatched,
		Remark:       payloadString(payload, "remark"),
		OperatorID:   operatorID,
		OperatorName: operatorName,
	}
	if err := s.paymentDAO.Create(ctx, payment); err != nil {
		return fmt.Errorf("创建回款失败: %w", err)
	}
	_ = s.settlementDAO.UpdateStatus(ctx, settlementID, model.OpsSettlementStatusPaid)
	return nil
}

func (s *opsBizService) loadLifecycleInstance(ctx context.Context, instanceID int) (*model.WorkorderInstance, *model.OpsApprovalLink, int, error) {
	link, err := s.approvalDAO.GetByInstanceID(ctx, instanceID)
	if err != nil || link == nil {
		return nil, nil, 0, fmt.Errorf("未找到运营全流程绑定，请从客户详情发起")
	}
	if link.BizType != model.OpsApprovalBizCustomerLifecycle {
		return nil, nil, 0, fmt.Errorf("当前工单不是运营全流程，请使用普通审批")
	}
	instance, err := s.instanceSvc.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, nil, 0, err
	}
	return instance, link, link.BizID, nil
}

func (s *opsBizService) loadProcessDefinition(ctx context.Context, processID int) (model.ProcessDefinition, error) {
	var definition model.ProcessDefinition
	process, err := s.processDao.GetProcessByID(ctx, processID)
	if err != nil {
		return definition, fmt.Errorf("获取流程定义失败: %w", err)
	}
	raw, err := json.Marshal(process.Definition)
	if err != nil {
		return definition, fmt.Errorf("流程定义序列化失败: %w", err)
	}
	if err := json.Unmarshal(raw, &definition); err != nil {
		return definition, fmt.Errorf("流程定义解析失败: %w", err)
	}
	return definition, nil
}

func normalizeLifecycleStep(step *model.ProcessStep) string {
	if step == nil {
		return ""
	}
	id := strings.ToLower(strings.TrimSpace(step.ID))
	name := step.Name
	switch {
	case id == model.OpsLifecycleStepIntent || strings.Contains(name, "意向"):
		return model.OpsLifecycleStepIntent
	case id == model.OpsLifecycleStepTrialAccept || strings.Contains(name, "验收") || strings.Contains(name, "转正"):
		return model.OpsLifecycleStepTrialAccept
	case id == model.OpsLifecycleStepTrial || (strings.Contains(name, "试用") && strings.Contains(name, "审批")):
		return model.OpsLifecycleStepTrial
	case id == model.OpsLifecycleStepContract || strings.Contains(name, "合同") || strings.Contains(name, "开通"):
		return model.OpsLifecycleStepContract
	case id == model.OpsLifecycleStepSettlement || strings.Contains(name, "结算"):
		return model.OpsLifecycleStepSettlement
	case id == model.OpsLifecycleStepPayment || strings.Contains(name, "开票") || strings.Contains(name, "回款"):
		return model.OpsLifecycleStepPayment
	default:
		return id
	}
}

func getNextProcessStep(current *model.ProcessStep, definition model.ProcessDefinition) *model.ProcessStep {
	if current == nil {
		return nil
	}
	var nextID string
	for _, c := range definition.Connections {
		if c.From == current.ID {
			nextID = c.To
			break
		}
	}
	if nextID == "" {
		return nil
	}
	for i := range definition.Steps {
		if definition.Steps[i].ID == nextID {
			return &definition.Steps[i]
		}
	}
	return nil
}

func payloadString(m model.JSONMap, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func payloadInt(m model.JSONMap, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		return 0
	}
}

func payloadFloat(m model.JSONMap, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		n, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return n
	default:
		return 0
	}
}

func parsePayloadTime(m model.JSONMap, key string) (*time.Time, error) {
	raw := strings.TrimSpace(payloadString(m, key))
	if raw == "" {
		return nil, nil
	}
	return opsUtils.ParseOptionalTime(&raw)
}
