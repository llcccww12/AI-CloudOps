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
	"github.com/spf13/viper"
	"go.uber.org/zap"
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

	// 材料上传挂在工单绑定的试用/开通单上；勿直接取客户「最新一条」，否则重复单会导致已上传却显示尚未上传
	out.LatestTrial = s.resolveTrialForApproveContext(ctx, customerID, instanceID, instance.FormData)
	out.LatestActivation = s.resolveActivationForApproveContext(ctx, customerID, instanceID, instance.FormData)
	out.LatestContract = s.ensureContractForApproveContext(ctx, customer, nodeKey, out.LatestTrial, out.LatestActivation)
	if settlements, _, err := s.settlementDAO.List(ctx, &model.ListOpsSettlementReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	}); err == nil && len(settlements) > 0 {
		out.LatestSettlement = settlements[0]
	}
	out.VendorProfileURL = strings.TrimSpace(viper.GetString("ops.vendor_profile_url"))
	out.VendorProfileDone = customer.VendorProfileDone
	return out, nil
}

// ensureContractForApproveContext 合同/试用合同节点预置草稿合同，便于审批前上传扫描件（避免 biz_id=0）。
func (s *opsBizService) ensureContractForApproveContext(
	ctx context.Context,
	customer *model.OpsCustomer,
	nodeKey string,
	trial *model.OpsTrial,
	act *model.OpsActivation,
) *model.OpsContract {
	if customer == nil {
		return nil
	}
	if act != nil && act.ContractID > 0 {
		if c, err := s.contractDAO.GetByID(ctx, act.ContractID); err == nil {
			return c
		}
	}

	wantType := ""
	switch nodeKey {
	case model.OpsLifecycleStepTrialContract:
		wantType = model.OpsContractTypeTrial
	case model.OpsLifecycleStepContract:
		wantType = model.OpsContractTypeFormal
	}

	if wantType != "" {
		if contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
			ListReq: model.ListReq{Page: 1, Size: 20}, CustomerID: customer.ID, Type: wantType,
		}); err == nil {
			for _, c := range contracts {
				if c == nil {
					continue
				}
				if c.Status == model.OpsContractStatusDraft || c.Status == model.OpsContractStatusActive {
					return c
				}
			}
			if len(contracts) > 0 && contracts[0] != nil {
				return contracts[0]
			}
		}
		titleSuffix := "正式合同"
		if wantType == model.OpsContractTypeTrial {
			titleSuffix = "试用合同"
		}
		draft := &model.OpsContract{
			CustomerID: customer.ID,
			Type:       wantType,
			Title:      fmt.Sprintf("%s-%s", customer.Name, titleSuffix),
			Status:     model.OpsContractStatusDraft,
			AutoRenew:  2,
			PaymentTermDays: 30,
		}
		if trial != nil {
			tid := trial.ID
			draft.TrialID = &tid
			if trial.ContractNo != "" {
				draft.ContractNo = trial.ContractNo
			}
			if trial.ProductType != "" {
				draft.ProductType = trial.ProductType
			}
		}
		if err := s.contractDAO.Create(ctx, draft); err != nil {
			s.logger.Warn("预创建合同草稿失败，合同扫描件需审批后上传",
				zap.Int("customerID", customer.ID), zap.String("type", wantType), zap.Error(err))
			return nil
		}
		return draft
	}

	if contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customer.ID, Type: model.OpsContractTypeFormal,
	}); err == nil && len(contracts) > 0 {
		return contracts[0]
	}
	if contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customer.ID,
	}); err == nil && len(contracts) > 0 {
		return contracts[0]
	}
	return nil
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

	if err := s.applyLifecycleNodePayload(ctx, nodeKey, customerID, req.InstanceID, payload, operatorID, operatorName); err != nil {
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

func (s *opsBizService) applyLifecycleNodePayload(ctx context.Context, nodeKey string, customerID, instanceID int, payload model.JSONMap, operatorID int, operatorName string) error {
	switch nodeKey {
	case model.OpsLifecycleStepIntent:
		return s.applyIntentNode(ctx, customerID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepTrial:
		return s.applyTrialNode(ctx, customerID, instanceID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepTrialContract:
		return s.applyTrialContractNode(ctx, customerID, instanceID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepOpenRequest:
		return s.applyOpenRequestNode(ctx, customerID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepOpenFeedback:
		return s.applyOpenFeedbackNode(ctx, customerID, instanceID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepTrialAccept:
		return s.applyTrialAcceptNode(ctx, customerID, instanceID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepProvision:
		return s.applyProvisionNode(ctx, customerID, instanceID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepContract:
		return s.applyContractNode(ctx, customerID, instanceID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepSettlement:
		return s.applySettlementNode(ctx, customerID, payload, operatorID, operatorName)
	case model.OpsLifecycleStepInvoice:
		return s.applyInvoiceNode(ctx, customerID, payload, operatorID, operatorName)
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
	// 外访转入即为意向；发起全流程后进入试用。意向节点只补历史线索→意向，不回退阶段。
	if c, err := s.customerDAO.GetByID(ctx, customerID); err == nil && c != nil &&
		(c.Stage == "" || c.Stage == model.OpsCustomerStageLead) {
		_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageIntent, "")
	}
	return nil
}

func (s *opsBizService) applyTrialNode(ctx context.Context, customerID, instanceID int, payload model.JSONMap, operatorID int, operatorName string) error {
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

	// 同一测试/全流程只维护一张试用单：优先工单已绑定的，其次未结束的最新一张
	trial, err := s.resolveTrialForProcess(ctx, customerID, instanceID)
	if err != nil {
		return err
	}
	if trial != nil {
		// 开通台账已写入更具体标题时，避免被表单默认「客户名-试用」覆盖
		if !(isPlaceholderTrialTitle(title) && !isPlaceholderTrialTitle(trial.Title) && strings.TrimSpace(trial.Title) != "") {
			trial.Title = title
		}
		trial.DemandType = payloadString(payload, "demand_type")
		trial.ResourceScale = payloadString(payload, "resource_scale")
		trial.Purpose = payloadString(payload, "purpose")
		trial.PlanStartAt = planStart
		trial.PlanEndAt = planEnd
		if trial.OpenMethod == "" {
			trial.OpenMethod = model.OpsOpenMethodTrial
		}
		trial.OperatorID = operatorID
		trial.OperatorName = operatorName
		if err := s.trialDAO.Update(ctx, trial); err != nil {
			return fmt.Errorf("更新试用单失败: %w", err)
		}
		if err := s.trialDAO.UpdateStatus(ctx, trial.ID, model.OpsTrialStatusApproved); err != nil {
			return fmt.Errorf("更新试用单状态失败: %w", err)
		}
	} else {
		trial = &model.OpsTrial{
			CustomerID:    customerID,
			Title:         title,
			DemandType:    payloadString(payload, "demand_type"),
			ResourceScale: payloadString(payload, "resource_scale"),
			Purpose:       payloadString(payload, "purpose"),
			Status:        model.OpsTrialStatusApproved,
			OpenMethod:    model.OpsOpenMethodTrial,
			PlanStartAt:   planStart,
			PlanEndAt:     planEnd,
			OperatorID:    operatorID,
			OperatorName:  operatorName,
		}
		if err := s.trialDAO.Create(ctx, trial); err != nil {
			return fmt.Errorf("创建试用单失败: %w", err)
		}
		s.bindTrialWorkorderIfNeeded(ctx, instanceID, trial.ID)
	}
	_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageTrial, "")
	return nil
}

func (s *opsBizService) applyTrialAcceptNode(ctx context.Context, customerID, instanceID int, payload model.JSONMap, operatorID int, operatorName string) error {
	evaluation := strings.TrimSpace(payloadString(payload, "evaluation"))
	convertIntent := strings.TrimSpace(payloadString(payload, "convert_intent"))
	if evaluation == "" {
		return fmt.Errorf("请填写试用验收评价")
	}
	if convertIntent == "" {
		return fmt.Errorf("请选择转正意向")
	}
	trial, err := s.resolveTrialForProcess(ctx, customerID, instanceID)
	if err != nil {
		return err
	}
	if trial == nil {
		return fmt.Errorf("未找到该客户的试用单，请先完成「资源开通」节点登记")
	}
	// 开通归档读的是试用单台账；若资源开通未写入（跳过/节点名未识别/非生命周期审批），验收时允许补登，否则拦截
	needLedgerBackfill := trialLedgerIncomplete(trial)
	if needLedgerBackfill {
		if err := requireDeliveryLedger(payload); err != nil {
			return fmt.Errorf("开通台账尚未登记，验收后归档会为空：%w。请先走「资源开通」节点，或在本节点一并补齐台账", err)
		}
		if err := requireComputeAlloc(payload); err != nil {
			return err
		}
		applyDeliveryLedgerToTrial(trial, payload, operatorID, operatorName)
		if pn := strings.TrimSpace(payloadString(payload, "project_name")); pn != "" && isPlaceholderTrialTitle(trial.Title) {
			trial.Title = pn
		}
	}
	trial.Evaluation = evaluation
	trial.ConvertIntent = convertIntent
	trial.UpdaterID = operatorID
	trial.UpdaterName = operatorName
	if trial.Title == "" {
		trial.Title = fmt.Sprintf("试用-%d", trial.ID)
	}
	if trial.OpenMethod == "" {
		trial.OpenMethod = model.OpsOpenMethodTrial
	}
	if err := s.trialDAO.Update(ctx, trial); err != nil {
		return fmt.Errorf("更新试用验收失败: %w", err)
	}
	_ = s.trialDAO.UpdateStatus(ctx, trial.ID, model.OpsTrialStatusEnded)
	if needLedgerBackfill {
		if err := s.upsertComputeAllocation(ctx, customerID, instanceID, payload, operatorID, operatorName, computeAllocBizRefs{
			Source:        "open",
			TrialID:       trial.ID,
			ApplyNo:       trial.OrderNo,
			ContractNo:    trial.ContractNo,
			ContractStart: trial.ContractStartAt,
			ContractEnd:   trial.ContractEndAt,
		}); err != nil {
			return err
		}
	}
	if convertIntent == "strong" {
		_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageFormal, "")
	}
	return nil
}

func (s *opsBizService) applyProvisionNode(ctx context.Context, customerID, instanceID int, payload model.JSONMap, operatorID int, operatorName string) error {
	if err := requireDeliveryLedger(payload); err != nil {
		return err
	}
	if err := requireComputeAlloc(payload); err != nil {
		return err
	}
	trial, err := s.resolveTrialForProcess(ctx, customerID, instanceID)
	if err != nil {
		return err
	}
	created := false
	if trial == nil {
		title := strings.TrimSpace(payloadString(payload, "project_name"))
		if title == "" {
			title = strings.TrimSpace(payloadString(payload, "title"))
		}
		if title == "" {
			title = "测试开通"
		}
		trial = &model.OpsTrial{
			CustomerID: customerID, Title: title, Status: model.OpsTrialStatusApproved,
			OpenMethod: model.OpsOpenMethodTrial, OperatorID: operatorID, OperatorName: operatorName,
		}
		if err := s.trialDAO.Create(ctx, trial); err != nil {
			return fmt.Errorf("创建试用开通单失败: %w", err)
		}
		created = true
		s.bindTrialWorkorderIfNeeded(ctx, instanceID, trial.ID)
	}
	applyDeliveryLedgerToTrial(trial, payload, operatorID, operatorName)
	if trial.OpenMethod == "" {
		trial.OpenMethod = model.OpsOpenMethodTrial
	}
	needApprove := trial.Status == "" || trial.Status == model.OpsTrialStatusDraft || trial.Status == model.OpsTrialStatusPending
	if needApprove {
		trial.Status = model.OpsTrialStatusApproved
	}
	// 开通台账项目名可同步标题，避免归档显示默认「客户-试用」与项目名两套
	if pn := strings.TrimSpace(payloadString(payload, "project_name")); pn != "" && isPlaceholderTrialTitle(trial.Title) {
		trial.Title = pn
	}
	if err := s.trialDAO.Update(ctx, trial); err != nil {
		return fmt.Errorf("保存开通台账失败: %w", err)
	}
	if needApprove || created {
		if err := s.trialDAO.UpdateStatus(ctx, trial.ID, model.OpsTrialStatusApproved); err != nil {
			return fmt.Errorf("更新试用单状态失败: %w", err)
		}
	}
	_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageTrial, "")
	return s.upsertComputeAllocation(ctx, customerID, instanceID, payload, operatorID, operatorName, computeAllocBizRefs{
		Source:        "open",
		TrialID:       trial.ID,
		ApplyNo:       trial.OrderNo,
		ContractNo:    trial.ContractNo,
		ContractStart: trial.ContractStartAt,
		ContractEnd:   trial.ContractEndAt,
	})
}

func isPlaceholderTrialTitle(title string) bool {
	t := strings.TrimSpace(title)
	return t == "" || t == "测试开通" || strings.HasSuffix(t, "-试用") || strings.HasPrefix(t, "运营测试开通-")
}

func (s *opsBizService) bindTrialWorkorderIfNeeded(ctx context.Context, instanceID, trialID int) {
	if instanceID <= 0 || trialID <= 0 {
		return
	}
	link, err := s.approvalDAO.GetByInstanceID(ctx, instanceID)
	if err != nil || link == nil || link.BizType != model.OpsApprovalBizTrial {
		return
	}
	if link.BizID == 0 || link.BizID == trialID {
		_ = s.bindWorkorder(ctx, model.OpsApprovalBizTrial, trialID, instanceID)
	}
}

// resolveTrialForProcess 解析本流程应复用的试用单，避免一流程多张测试开通归档。
func (s *opsBizService) resolveTrialForProcess(ctx context.Context, customerID, instanceID int) (*model.OpsTrial, error) {
	candidates := make([]*model.OpsTrial, 0, 8)
	seen := map[int]struct{}{}
	add := func(t *model.OpsTrial) {
		if t == nil || t.ID <= 0 {
			return
		}
		if _, ok := seen[t.ID]; ok {
			return
		}
		seen[t.ID] = struct{}{}
		candidates = append(candidates, t)
	}

	if instanceID > 0 {
		if inst, err := s.instanceSvc.GetInstance(ctx, instanceID); err == nil && inst != nil {
			if payloadString(inst.FormData, "ops_biz_type") == model.OpsApprovalBizTrial {
				if id := payloadInt(inst.FormData, "ops_biz_id"); id > 0 {
					if trial, err := s.trialDAO.GetByID(ctx, id); err == nil {
						add(trial)
					}
				}
			}
		}
		if link, err := s.approvalDAO.GetByInstanceID(ctx, instanceID); err == nil && link != nil {
			if link.BizType == model.OpsApprovalBizTrial && link.BizID > 0 {
				trial, err := s.trialDAO.GetByID(ctx, link.BizID)
				if err != nil {
					return nil, fmt.Errorf("流程绑定的试用单不存在: %w", err)
				}
				add(trial)
			}
		}
	}

	trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
		ListReq: model.ListReq{Page: 1, Size: 20}, CustomerID: customerID,
	})
	if err != nil {
		return nil, fmt.Errorf("查询试用单失败: %w", err)
	}
	for _, t := range trials {
		if t == nil {
			continue
		}
		switch t.Status {
		case model.OpsTrialStatusEnded, model.OpsTrialStatusRejected:
			continue
		default:
			add(t)
		}
	}
	if best := pickBestTrial(candidates); best != nil {
		return best, nil
	}
	return nil, nil
}

func (s *opsBizService) resolveTrialForApproveContext(ctx context.Context, customerID, instanceID int, formData model.JSONMap) *model.OpsTrial {
	if trial, err := s.resolveTrialForProcess(ctx, customerID, instanceID); err == nil && trial != nil {
		return trial
	}
	if bizType := payloadString(formData, "ops_biz_type"); bizType == model.OpsApprovalBizTrial {
		if id := payloadInt(formData, "ops_biz_id"); id > 0 {
			if trial, err := s.trialDAO.GetByID(ctx, id); err == nil {
				return trial
			}
		}
	}
	if trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
		ListReq: model.ListReq{Page: 1, Size: 20}, CustomerID: customerID,
	}); err == nil {
		return pickBestTrial(trials)
	}
	return nil
}

func trialLedgerIncomplete(t *model.OpsTrial) bool {
	if t == nil {
		return true
	}
	return strings.TrimSpace(t.ProductType) == "" ||
		strings.TrimSpace(t.ProjectName) == "" ||
		strings.TrimSpace(t.CustomerShortName) == "" ||
		strings.TrimSpace(t.Region) == ""
}

func isHollowTrial(t *model.OpsTrial) bool {
	if t == nil {
		return true
	}
	return trialLedgerIncomplete(t) ||
		(isPlaceholderTrialTitle(t.Title) &&
			strings.TrimSpace(t.ProjectName) == "" &&
			strings.TrimSpace(t.OrderNo) == "" &&
			strings.TrimSpace(t.MainAccount) == "")
}

func trialSubstanceScore(t *model.OpsTrial) int {
	if t == nil {
		return -1000
	}
	score := t.ID // 同分时偏新
	switch t.Status {
	case model.OpsTrialStatusEnded, model.OpsTrialStatusRejected:
		score -= 100000
	default:
		score += 10000
	}
	if !isPlaceholderTrialTitle(t.Title) {
		score += 5000
	}
	if strings.TrimSpace(t.ProjectName) != "" {
		score += 4000
	}
	if strings.TrimSpace(t.OrderNo) != "" {
		score += 2000
	}
	if strings.TrimSpace(t.MainAccount) != "" {
		score += 1000
	}
	if strings.TrimSpace(t.ContractNo) != "" {
		score += 500
	}
	return score
}

func pickBestTrial(trials []*model.OpsTrial) *model.OpsTrial {
	var best *model.OpsTrial
	bestScore := -1 << 30
	for _, t := range trials {
		if t == nil {
			continue
		}
		score := trialSubstanceScore(t)
		if best == nil || score > bestScore {
			best = t
			bestScore = score
		}
	}
	return best
}

func (s *opsBizService) resolveActivationForApproveContext(ctx context.Context, customerID, instanceID int, formData model.JSONMap) *model.OpsActivation {
	if instanceID > 0 {
		if link, err := s.approvalDAO.GetByInstanceID(ctx, instanceID); err == nil && link != nil &&
			link.BizType == model.OpsApprovalBizActivation && link.BizID > 0 {
			if act, err := s.activationDAO.GetByID(ctx, link.BizID); err == nil {
				return act
			}
		}
	}
	if bizType := payloadString(formData, "ops_biz_type"); bizType == model.OpsApprovalBizActivation {
		if id := payloadInt(formData, "ops_biz_id"); id > 0 {
			if act, err := s.activationDAO.GetByID(ctx, id); err == nil {
				return act
			}
		}
	}
	if id := payloadInt(formData, "activation_id"); id > 0 {
		if act, err := s.activationDAO.GetByID(ctx, id); err == nil {
			return act
		}
	}
	if acts, _, err := s.activationDAO.List(ctx, &model.ListOpsActivationReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	}); err == nil && len(acts) > 0 {
		return acts[0]
	}
	return nil
}

func (s *opsBizService) applyTrialContractNode(ctx context.Context, customerID, instanceID int, payload model.JSONMap, operatorID int, operatorName string) error {
	payload["type"] = model.OpsContractTypeTrial
	return s.applyContractNode(ctx, customerID, instanceID, payload, operatorID, operatorName)
}

func (s *opsBizService) applyOpenRequestNode(ctx context.Context, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
	title := strings.TrimSpace(payloadString(payload, "title"))
	if title == "" {
		title = strings.TrimSpace(payloadString(payload, "activation_title"))
	}
	if title == "" {
		return fmt.Errorf("请填写开通申请标题")
	}
	contractID := payloadInt(payload, "contract_id")
	if contractID <= 0 {
		contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
			ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
		})
		if err != nil || len(contracts) == 0 {
			return fmt.Errorf("未找到合同，请先完成合同签约节点")
		}
		contractID = contracts[0].ID
	}
	return s.activationDAO.Create(ctx, &model.OpsActivation{
		CustomerID: customerID, ContractID: contractID, Title: title,
		ResourceSummary: payloadString(payload, "resource_summary"),
		Purpose:         payloadString(payload, "purpose"),
		Status:          model.OpsActivationStatusPending,
		OperatorID:      operatorID, OperatorName: operatorName,
	})
}

func (s *opsBizService) applyOpenFeedbackNode(ctx context.Context, customerID, instanceID int, payload model.JSONMap, operatorID int, operatorName string) error {
	account := strings.TrimSpace(payloadString(payload, "feedback_account"))
	if account == "" {
		return fmt.Errorf("请填写开通账号")
	}
	if err := requireComputeAlloc(payload); err != nil {
		return err
	}
	var act *model.OpsActivation
	if aid := payloadInt(payload, "activation_id"); aid > 0 {
		item, err := s.activationDAO.GetByID(ctx, aid)
		if err != nil {
			return err
		}
		act = item
	} else {
		items, _, err := s.activationDAO.List(ctx, &model.ListOpsActivationReq{
			ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
		})
		if err != nil || len(items) == 0 {
			return fmt.Errorf("未找到开通申请单，请先完成开通申请节点")
		}
		act = items[0]
	}
	now := time.Now()
	act.FeedbackAccount = account
	act.FeedbackTenant = payloadString(payload, "feedback_tenant")
	act.FeedbackEndpoint = payloadString(payload, "feedback_endpoint")
	act.FeedbackRemark = payloadString(payload, "feedback_remark")
	if strings.TrimSpace(act.MainAccount) == "" {
		act.MainAccount = account
	}
	act.Status = model.OpsActivationStatusActive
	act.ActivatedAt = &now
	act.OperatorID = operatorID
	act.OperatorName = operatorName
	act.UpdaterID = operatorID
	act.UpdaterName = operatorName
	if err := s.activationDAO.Update(ctx, act); err != nil {
		return err
	}

	var contractStart, contractEnd *time.Time
	contractNo := act.ContractNo
	if act.ContractID > 0 {
		if c, err := s.contractDAO.GetByID(ctx, act.ContractID); err == nil && c != nil {
			contractStart, contractEnd = c.StartAt, c.EndAt
			if contractNo == "" {
				contractNo = c.ContractNo
			}
		}
	}
	applyNo := act.OrderNo
	if applyNo == "" {
		applyNo = fmt.Sprintf("ACT-%d", act.ID)
	}
	return s.upsertComputeAllocation(ctx, customerID, instanceID, payload, operatorID, operatorName, computeAllocBizRefs{
		Source:        "open",
		ContractID:    act.ContractID,
		ContractNo:    contractNo,
		ContractStart: contractStart,
		ContractEnd:   contractEnd,
		ActivationID:  act.ID,
		ApplyNo:       applyNo,
	})
}

func (s *opsBizService) applyContractNode(ctx context.Context, customerID, instanceID int, payload model.JSONMap, operatorID int, operatorName string) error {
	if err := requireDeliveryLedger(payload); err != nil {
		return err
	}
	if err := requireComputeAlloc(payload); err != nil {
		return err
	}
	title := strings.TrimSpace(payloadString(payload, "title"))
	if title == "" {
		title = strings.TrimSpace(payloadString(payload, "project_name"))
	}
	if title == "" {
		return fmt.Errorf("请填写合同标题或项目名称")
	}
	contractType := payloadString(payload, "type")
	if contractType == "" {
		contractType = model.OpsContractTypeFormal
	}
	startAt, err := parsePayloadTimeOptional(payload, "contract_start_at", "start_at")
	if err != nil {
		return err
	}
	endAt, err := parsePayloadTimeOptional(payload, "contract_end_at", "end_at")
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
	contractNo := strings.TrimSpace(payloadString(payload, "contract_no"))
	productType := strings.TrimSpace(payloadString(payload, "product_type"))

	// 优先更新正式开通流程已创建的草稿合同/开通单
	var existingAct *model.OpsActivation
	if acts, _, err := s.activationDAO.List(ctx, &model.ListOpsActivationReq{
		ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
	}); err == nil && len(acts) > 0 {
		existingAct = acts[0]
	}

	var contract *model.OpsContract
	if existingAct != nil && existingAct.ContractID > 0 {
		c, err := s.contractDAO.GetByID(ctx, existingAct.ContractID)
		if err != nil {
			return fmt.Errorf("关联合同不存在: %w", err)
		}
		contract = c
	} else if contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
		ListReq: model.ListReq{Page: 1, Size: 20}, CustomerID: customerID, Type: contractType,
	}); err == nil {
		for _, c := range contracts {
			if c == nil {
				continue
			}
			if c.Status == model.OpsContractStatusDraft || c.Status == model.OpsContractStatusActive {
				contract = c
				break
			}
		}
	}

	if contract != nil {
		contract.Title = title
		contract.ContractNo = contractNo
		contract.Type = contractType
		contract.TrialID = trialID
		contract.ProductType = productType
		contract.BillingMode = payloadString(payload, "billing_mode")
		contract.UnitPrice = payloadFloat(payload, "unit_price")
		contract.BillingCycle = payloadString(payload, "billing_cycle")
		contract.PaymentMethod = payloadString(payload, "payment_method")
		contract.PaymentTermDays = term
		contract.StartAt = startAt
		contract.EndAt = endAt
		contract.AutoRenew = autoRenew
		contract.Status = model.OpsContractStatusActive
		contract.Remark = payloadString(payload, "remark")
		if err := s.contractDAO.Update(ctx, contract); err != nil {
			return fmt.Errorf("更新合同失败: %w", err)
		}
	} else {
		contract = &model.OpsContract{
			CustomerID: customerID, TrialID: trialID, Type: contractType, Title: title, ContractNo: contractNo,
			ProductType: productType, BillingMode: payloadString(payload, "billing_mode"),
			UnitPrice: payloadFloat(payload, "unit_price"), BillingCycle: payloadString(payload, "billing_cycle"),
			PaymentMethod: payloadString(payload, "payment_method"), PaymentTermDays: term,
			StartAt: startAt, EndAt: endAt, AutoRenew: autoRenew,
			Status: model.OpsContractStatusActive, Remark: payloadString(payload, "remark"),
			OperatorID: operatorID, OperatorName: operatorName,
		}
		if err := s.contractDAO.Create(ctx, contract); err != nil {
			return fmt.Errorf("创建合同失败: %w", err)
		}
	}

	var activationID, linkedTrialID int
	var applyNo string
	if existingAct != nil && contractType == model.OpsContractTypeFormal {
		applyDeliveryLedgerToActivation(existingAct, payload, operatorID, operatorName)
		if existingAct.Title == "" {
			existingAct.Title = title
		}
		if existingAct.ContractID == 0 {
			existingAct.ContractID = contract.ID
		}
		if existingAct.OpenMethod == "" {
			existingAct.OpenMethod = model.OpsOpenMethodFormal
		}
		if err := s.activationDAO.Update(ctx, existingAct); err != nil {
			return fmt.Errorf("保存开通台账失败: %w", err)
		}
		activationID = existingAct.ID
		applyNo = existingAct.OrderNo
	} else if contractType == model.OpsContractTypeFormal {
		act := &model.OpsActivation{
			CustomerID: customerID, ContractID: contract.ID, Title: title,
			Status: model.OpsActivationStatusActive, OpenMethod: model.OpsOpenMethodFormal,
			OperatorID: operatorID, OperatorName: operatorName,
		}
		applyDeliveryLedgerToActivation(act, payload, operatorID, operatorName)
		if err := s.activationDAO.Create(ctx, act); err != nil {
			return fmt.Errorf("创建开通单失败: %w", err)
		}
		activationID = act.ID
		applyNo = act.OrderNo
	} else if contractType == model.OpsContractTypeTrial {
		if trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
			ListReq: model.ListReq{Page: 1, Size: 1}, CustomerID: customerID,
		}); err == nil && len(trials) > 0 {
			trial := trials[0]
			applyDeliveryLedgerToTrial(trial, payload, operatorID, operatorName)
			if trial.OpenMethod == "" {
				trial.OpenMethod = model.OpsOpenMethodTrial
			}
			if err := s.trialDAO.Update(ctx, trial); err != nil {
				return fmt.Errorf("保存试用开通台账失败: %w", err)
			}
			linkedTrialID = trial.ID
			applyNo = trial.OrderNo
		}
	}
	if linkedTrialID == 0 && trialID != nil {
		linkedTrialID = *trialID
	}

	if contractType == model.OpsContractTypeFormal {
		_ = s.customerDAO.UpdateStage(ctx, customerID, model.OpsCustomerStageFormal, "")
	}

	source := "contract"
	if contractType == model.OpsContractTypeTrial {
		source = "open"
	}
	return s.upsertComputeAllocation(ctx, customerID, instanceID, payload, operatorID, operatorName, computeAllocBizRefs{
		Source:        source,
		ContractID:    contract.ID,
		ContractNo:    contract.ContractNo,
		ContractStart: contract.StartAt,
		ContractEnd:   contract.EndAt,
		ActivationID:  activationID,
		TrialID:       linkedTrialID,
		ApplyNo:       applyNo,
	})
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

func (s *opsBizService) applyInvoiceNode(ctx context.Context, customerID int, payload model.JSONMap, operatorID int, operatorName string) error {
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
	amount := payloadFloat(payload, "invoice_amount")
	if amount <= 0 {
		amount = payloadFloat(payload, "amount")
	}
	if amount <= 0 {
		return fmt.Errorf("请填写开票金额")
	}
	issuedAt, err := parsePayloadTime(payload, "issued_at")
	if err != nil {
		return err
	}
	inv := &model.OpsInvoice{
		CustomerID: customerID, SettlementID: settlementID,
		InvoiceNo: payloadString(payload, "invoice_no"), InvoiceType: payloadString(payload, "invoice_type"),
		Amount: amount, IssuedAt: issuedAt, Status: model.OpsInvoiceStatusIssued,
		OperatorID: operatorID, OperatorName: operatorName,
	}
	if err := s.invoiceDAO.Create(ctx, inv); err != nil {
		return fmt.Errorf("创建发票失败: %w", err)
	}
	_ = s.settlementDAO.UpdateStatus(ctx, settlementID, model.OpsSettlementStatusInvoiced)
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
	paidAt, err := parsePayloadTime(payload, "paid_at")
	if err != nil {
		return err
	}
	payment := &model.OpsPayment{
		CustomerID: customerID, SettlementID: settlementID, Amount: amount, PaidAt: paidAt,
		BankRef: payloadString(payload, "bank_ref"), Status: model.OpsPaymentStatusMatched,
		Remark: payloadString(payload, "remark"), OperatorID: operatorID, OperatorName: operatorName,
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
		return nil, nil, 0, fmt.Errorf("未找到运营流程绑定，请从客户详情发起")
	}
	customerID := 0
	switch link.BizType {
	case model.OpsApprovalBizCustomerLifecycle:
		customerID = link.BizID
	case model.OpsApprovalBizActivation:
		act, err := s.activationDAO.GetByID(ctx, link.BizID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("开通单不存在: %w", err)
		}
		customerID = act.CustomerID
	case model.OpsApprovalBizTrial:
		trial, err := s.trialDAO.GetByID(ctx, link.BizID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("试用单不存在: %w", err)
		}
		customerID = trial.CustomerID
	default:
		return nil, nil, 0, fmt.Errorf("当前工单不是运营全流程/测试开通/正式开通流程，请使用普通审批")
	}
	instance, err := s.instanceSvc.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, nil, 0, err
	}
	return instance, link, customerID, nil
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
	case id == model.OpsLifecycleStepProvision || strings.Contains(name, "资源开通") || strings.Contains(name, "开通台账") ||
		strings.Contains(name, "资源交付") ||
		(strings.Contains(name, "测试开通") && !strings.Contains(name, "正式")):
		return model.OpsLifecycleStepProvision
	case id == model.OpsLifecycleStepTrialContract || (strings.Contains(name, "试用") && strings.Contains(name, "合同")):
		return model.OpsLifecycleStepTrialContract
	case id == model.OpsLifecycleStepOpenFeedback || strings.Contains(name, "开通回馈") || strings.Contains(name, "账号回馈"):
		return model.OpsLifecycleStepOpenFeedback
	case id == model.OpsLifecycleStepOpenRequest || strings.Contains(name, "开通申请"):
		return model.OpsLifecycleStepOpenRequest
	case id == model.OpsLifecycleStepTrial || (strings.Contains(name, "试用") && strings.Contains(name, "审批")):
		return model.OpsLifecycleStepTrial
	case id == model.OpsLifecycleStepInvoice || (strings.Contains(name, "开票") && !strings.Contains(name, "回款")):
		return model.OpsLifecycleStepInvoice
	case id == model.OpsLifecycleStepPayment || strings.Contains(name, "回款"):
		return model.OpsLifecycleStepPayment
	case id == model.OpsLifecycleStepSettlement || strings.Contains(name, "结算"):
		return model.OpsLifecycleStepSettlement
	case id == model.OpsLifecycleStepContract || strings.Contains(name, "正式合同") || strings.Contains(name, "合同与开通") ||
		(strings.Contains(name, "合同") && !strings.Contains(name, "试用")):
		return model.OpsLifecycleStepContract
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

func parsePayloadTimeOptional(m model.JSONMap, keys ...string) (*time.Time, error) {
	for _, key := range keys {
		t, err := parsePayloadTime(m, key)
		if err != nil {
			return nil, err
		}
		if t != nil {
			return t, nil
		}
	}
	return nil, nil
}

func requireDeliveryLedger(payload model.JSONMap) error {
	required := []struct {
		key, label string
	}{
		{"customer_short_name", "客户简称"},
		{"product_type", "产品类型"},
		{"region", "所属大区"},
		{"owner_name", "归属客户经理"},
		{"main_account", "主账号"},
		{"project_name", "项目名称"},
		{"open_method", "开通方式"},
		{"contract_no", "合同编号"},
		{"order_no", "订单编号"},
		{"open_period", "开通周期"},
	}
	for _, item := range required {
		if strings.TrimSpace(payloadString(payload, item.key)) == "" {
			return fmt.Errorf("请填写开通台账：%s", item.label)
		}
	}
	start, err := parsePayloadTimeOptional(payload, "contract_start_at", "start_at")
	if err != nil {
		return err
	}
	if start == nil {
		return fmt.Errorf("请填写开通台账：合同开始时间")
	}
	end, err := parsePayloadTimeOptional(payload, "contract_end_at", "end_at")
	if err != nil {
		return err
	}
	if end == nil {
		return fmt.Errorf("请填写开通台账：合同结束时间")
	}
	return nil
}

type computeAllocBizRefs struct {
	Source        string
	BizPhase      string
	ContractID    int
	ContractNo    string
	ContractStart *time.Time
	ContractEnd   *time.Time
	ActivationID  int
	TrialID       int
	ApplyNo       string
}

func requireComputeAlloc(payload model.JSONMap) error {
	if strings.TrimSpace(payloadString(payload, "server_code")) == "" {
		return fmt.Errorf("请填写算力生命周期台账：物理服务器")
	}
	lease := strings.TrimSpace(payloadString(payload, "lease_mode"))
	if lease == "" {
		return fmt.Errorf("请填写算力生命周期台账：租赁粒度")
	}
	if payloadInt(payload, "allocated_gpus") < 1 {
		return fmt.Errorf("请填写算力生命周期台账：实际分配卡数")
	}
	if lease == model.OpsComputeLeaseGPUPool && strings.TrimSpace(payloadString(payload, "partition_id")) == "" {
		return fmt.Errorf("请填写算力生命周期台账：卡/分区ID")
	}
	// 未填开通/释放日时，默认用开通台账合同起止
	if strings.TrimSpace(payloadString(payload, "opened_at")) == "" {
		if start := strings.TrimSpace(payloadString(payload, "contract_start_at")); start != "" {
			payload["opened_at"] = start
		}
	}
	if strings.TrimSpace(payloadString(payload, "plan_release_at")) == "" {
		if end := strings.TrimSpace(payloadString(payload, "contract_end_at")); end != "" {
			payload["plan_release_at"] = end
		}
	}
	opened, err := parsePayloadTimeOptional(payload, "opened_at")
	if err != nil {
		return err
	}
	if opened == nil {
		return fmt.Errorf("请填写算力生命周期台账：实际开通日")
	}
	return nil
}

func (s *opsBizService) upsertComputeAllocation(
	ctx context.Context,
	customerID, instanceID int,
	payload model.JSONMap,
	operatorID int,
	operatorName string,
	refs computeAllocBizRefs,
) error {
	if s.computeSvc == nil {
		return fmt.Errorf("算力服务未初始化")
	}
	if err := requireComputeAlloc(payload); err != nil {
		return err
	}
	openedAt, err := parsePayloadTimeOptional(payload, "opened_at")
	if err != nil {
		return err
	}
	planRelease, err := parsePayloadTimeOptional(payload, "plan_release_at")
	if err != nil {
		return err
	}
	if planRelease == nil {
		planRelease = refs.ContractEnd
	}
	now := time.Now()
	source := strings.TrimSpace(refs.Source)
	if source == "" {
		source = "open"
	}
	gpus := payloadInt(payload, "allocated_gpus")
	lease := strings.TrimSpace(payloadString(payload, "lease_mode"))
	serverCode := strings.TrimSpace(payloadString(payload, "server_code"))
	partitionID := strings.TrimSpace(payloadString(payload, "partition_id"))
	executor := strings.TrimSpace(payloadString(payload, "executor_name"))
	if executor == "" {
		executor = operatorName
	}
	applyNo := strings.TrimSpace(refs.ApplyNo)
	if applyNo == "" {
		applyNo = strings.TrimSpace(payloadString(payload, "order_no"))
	}
	contractNo := strings.TrimSpace(refs.ContractNo)
	if contractNo == "" {
		contractNo = strings.TrimSpace(payloadString(payload, "contract_no"))
	}

	existID, err := s.findActiveComputeAllocID(ctx, customerID, instanceID, serverCode, refs)
	if err != nil {
		return fmt.Errorf("查询算力台账失败: %w", err)
	}

	var exist *model.OpsComputeAllocation
	if existID > 0 {
		exist, err = s.computeSvc.GetAllocation(ctx, existID)
		if err != nil {
			return fmt.Errorf("读取算力台账失败: %w", err)
		}
	}
	existingPhase := ""
	if exist != nil {
		existingPhase = exist.BizPhase
	}
	bizPhase := resolveComputeBizPhase(refs, exist != nil, existingPhase)

	if exist != nil {
		// 连续占用：保留原始开通日，仅延长周期 / 更新商业阶段
		if exist.OpenedAt != nil {
			openedAt = exist.OpenedAt
		}
		actID := refs.ActivationID
		if actID <= 0 {
			actID = exist.ActivationID
		}
		trialID := refs.TrialID
		if trialID <= 0 {
			trialID = exist.TrialID
		}
		phaseNote := exist.PhaseNote
		oldPhase := exist.BizPhase
		if oldPhase == "" {
			oldPhase = model.OpsComputePhaseTrial
		}
		if oldPhase != bizPhase {
			line := fmt.Sprintf("%s 流程写入 %s→%s",
				now.Format("2006-01-02"),
				opsUtils.ComputeBizPhaseLabel(oldPhase),
				opsUtils.ComputeBizPhaseLabel(bizPhase),
			)
			if phaseNote == "" {
				phaseNote = line
			} else {
				phaseNote = phaseNote + "\n" + line
			}
		}
		remark := strings.TrimSpace(payloadString(payload, "remark"))
		if remark == "" {
			remark = exist.Remark
		}
		return s.computeSvc.UpdateAllocation(ctx, &model.UpdateOpsComputeAllocationReq{
			ID:              exist.ID,
			Source:          source,
			ServerCode:      serverCode,
			PartitionID:     partitionID,
			LeaseMode:       lease,
			AllocatedGPUs:   gpus,
			PlannedGPUs:     gpus,
			CustomerID:      customerID,
			ContractID:      refs.ContractID,
			ContractNo:      contractNo,
			ContractStartAt: refs.ContractStart,
			ContractEndAt:   refs.ContractEnd,
			ActivationID:    actID,
			TrialID:         trialID,
			ApplyNo:         applyNo,
			AuditInstanceID: instanceID,
			ApprovedAt:      &now,
			OpenedAt:        openedAt,
			PlanReleaseAt:   planRelease,
			BizPhase:        bizPhase,
			PhaseNote:       phaseNote,
			ExecutorName:    executor,
			Remark:          remark,
		})
	}

	return s.computeSvc.CreateAllocation(ctx, &model.CreateOpsComputeAllocationReq{
		Source:          source,
		ServerCode:      serverCode,
		PartitionID:     partitionID,
		LeaseMode:       lease,
		AllocatedGPUs:   gpus,
		PlannedGPUs:     gpus,
		CustomerID:      customerID,
		ContractID:      refs.ContractID,
		ContractNo:      contractNo,
		ContractStartAt: refs.ContractStart,
		ContractEndAt:   refs.ContractEnd,
		ActivationID:    refs.ActivationID,
		TrialID:         refs.TrialID,
		ApplyNo:         applyNo,
		AuditInstanceID: instanceID,
		ApprovedAt:      &now,
		OpenedAt:        openedAt,
		PlanReleaseAt:   planRelease,
		BizPhase:        bizPhase,
		ExecutorName:    executor,
		Remark:          payloadString(payload, "remark"),
		OperatorID:      operatorID,
		OperatorName:    operatorName,
	})
}

func resolveComputeBizPhase(refs computeAllocBizRefs, updating bool, existingPhase string) string {
	if p := strings.TrimSpace(refs.BizPhase); p != "" {
		return p
	}
	// 正式开通/合同节点：已是正式或续签且再次写入 → renew；测试转正式 → formal
	if refs.ActivationID > 0 || refs.Source == "contract" {
		ep := strings.TrimSpace(existingPhase)
		if updating && (ep == model.OpsComputePhaseFormal || ep == model.OpsComputePhaseRenew) {
			return model.OpsComputePhaseRenew
		}
		return model.OpsComputePhaseFormal
	}
	if updating && existingPhase != "" {
		return existingPhase
	}
	return model.OpsComputePhaseTrial
}

func (s *opsBizService) findActiveComputeAllocID(ctx context.Context, customerID, instanceID int, serverCode string, refs computeAllocBizRefs) (int, error) {
	// 1) 同开通单 / 试用单
	req := &model.ListOpsComputeAllocationReq{ListReq: model.ListReq{Page: 1, Size: 50}}
	switch {
	case refs.ActivationID > 0:
		req.ActivationID = refs.ActivationID
	case refs.TrialID > 0:
		req.TrialID = refs.TrialID
	case customerID > 0:
		req.CustomerID = customerID
	default:
		return 0, nil
	}
	list, err := s.computeSvc.ListAllocation(ctx, req)
	if err != nil {
		return 0, err
	}
	if list != nil {
		for _, item := range list.Items {
			if item == nil || item.ActualReleaseAt != nil {
				continue
			}
			if refs.ActivationID > 0 && item.ActivationID == refs.ActivationID {
				return item.ID, nil
			}
			if refs.TrialID > 0 && item.TrialID == refs.TrialID {
				return item.ID, nil
			}
			if refs.ActivationID == 0 && refs.TrialID == 0 && instanceID > 0 && item.AuditInstanceID == instanceID {
				return item.ID, nil
			}
		}
	}

	// 2) 同客户 + 同服务器：测试转正式连续占用
	serverCode = strings.TrimSpace(serverCode)
	if customerID > 0 && serverCode != "" {
		byCustomer, err := s.computeSvc.ListAllocation(ctx, &model.ListOpsComputeAllocationReq{
			ListReq: model.ListReq{Page: 1, Size: 50}, CustomerID: customerID, ServerCode: serverCode,
		})
		if err != nil {
			return 0, err
		}
		if byCustomer != nil {
			for _, item := range byCustomer.Items {
				if item == nil || item.ActualReleaseAt != nil {
					continue
				}
				if strings.TrimSpace(item.ServerCode) == serverCode {
					return item.ID, nil
				}
			}
		}
	}

	// 3) 同客户 + 同工单实例
	if customerID > 0 && instanceID > 0 {
		byCustomer, err := s.computeSvc.ListAllocation(ctx, &model.ListOpsComputeAllocationReq{
			ListReq: model.ListReq{Page: 1, Size: 50}, CustomerID: customerID,
		})
		if err != nil {
			return 0, err
		}
		if byCustomer != nil {
			for _, item := range byCustomer.Items {
				if item == nil || item.ActualReleaseAt != nil {
					continue
				}
				if item.AuditInstanceID == instanceID {
					return item.ID, nil
				}
			}
		}
	}
	return 0, nil
}

func applyDeliveryLedgerToTrial(t *model.OpsTrial, payload model.JSONMap, operatorID int, operatorName string) {
	if t == nil {
		return
	}
	t.CustomerShortName = strings.TrimSpace(payloadString(payload, "customer_short_name"))
	t.ProductType = strings.TrimSpace(payloadString(payload, "product_type"))
	t.Region = strings.TrimSpace(payloadString(payload, "region"))
	t.OwnerName = strings.TrimSpace(payloadString(payload, "owner_name"))
	t.MainAccount = strings.TrimSpace(payloadString(payload, "main_account"))
	t.ProjectName = strings.TrimSpace(payloadString(payload, "project_name"))
	t.OpenMethod = strings.TrimSpace(payloadString(payload, "open_method"))
	t.ContractNo = strings.TrimSpace(payloadString(payload, "contract_no"))
	t.OrderNo = strings.TrimSpace(payloadString(payload, "order_no"))
	t.OpenPeriod = strings.TrimSpace(payloadString(payload, "open_period"))
	if start, err := parsePayloadTimeOptional(payload, "contract_start_at", "start_at"); err == nil {
		t.ContractStartAt = start
	}
	if end, err := parsePayloadTimeOptional(payload, "contract_end_at", "end_at"); err == nil {
		t.ContractEndAt = end
	}
	t.UpdaterID = operatorID
	t.UpdaterName = operatorName
}

func applyDeliveryLedgerToActivation(a *model.OpsActivation, payload model.JSONMap, operatorID int, operatorName string) {
	if a == nil {
		return
	}
	a.CustomerShortName = strings.TrimSpace(payloadString(payload, "customer_short_name"))
	a.ProductType = strings.TrimSpace(payloadString(payload, "product_type"))
	a.Region = strings.TrimSpace(payloadString(payload, "region"))
	a.OwnerName = strings.TrimSpace(payloadString(payload, "owner_name"))
	a.MainAccount = strings.TrimSpace(payloadString(payload, "main_account"))
	a.ProjectName = strings.TrimSpace(payloadString(payload, "project_name"))
	a.OpenMethod = strings.TrimSpace(payloadString(payload, "open_method"))
	a.ContractNo = strings.TrimSpace(payloadString(payload, "contract_no"))
	a.OrderNo = strings.TrimSpace(payloadString(payload, "order_no"))
	a.OpenPeriod = strings.TrimSpace(payloadString(payload, "open_period"))
	if start, err := parsePayloadTimeOptional(payload, "contract_start_at", "start_at"); err == nil {
		a.ContractStartAt = start
	}
	if end, err := parsePayloadTimeOptional(payload, "contract_end_at", "end_at"); err == nil {
		a.ContractEndAt = end
	}
	a.UpdaterID = operatorID
	a.UpdaterName = operatorName
}
