/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

// 工单事件类型常量定义在 model 包中

var (
	ErrInvalidRequest    = fmt.Errorf("请求参数无效")
	ErrInvalidStatus     = fmt.Errorf("工单状态无效")
	ErrInvalidPermission = fmt.Errorf("权限不足")
	ErrInvalidOperation  = fmt.Errorf("操作无效")
)

type InstanceService interface {
	CreateInstance(ctx context.Context, req *model.CreateWorkorderInstanceReq) error
	CreateInstanceFromTemplate(ctx context.Context, templateID int, req *model.CreateWorkorderInstanceFromTemplateReq) (int, error)
	UpdateInstance(ctx context.Context, req *model.UpdateWorkorderInstanceReq) error
	DeleteInstance(ctx context.Context, id int, operatorID int) error
	GetInstance(ctx context.Context, id int) (*model.WorkorderInstance, error)
	MarkNotificationRead(ctx context.Context, instanceID, userID int)
	ListInstance(ctx context.Context, req *model.ListWorkorderInstanceReq) (*model.ListResp[*model.WorkorderInstance], error)
	SubmitInstance(ctx context.Context, id int, operatorID int, operatorName string) error
	AssignInstance(ctx context.Context, id int, assigneeID int, operatorID int, operatorName string, mode string, comment string) error
	ApproveInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string, nextAssigneeID int, attachmentIDs []int) error
	RejectInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
	CancelInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
	CompleteInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
	ReturnInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error
	GetAvailableActions(ctx context.Context, instanceID int, operatorID int) ([]string, error)
	GetCurrentStep(ctx context.Context, instanceID int) (*model.ProcessStep, error)
	ExportInstance(ctx context.Context, req *model.ExportWorkorderInstanceReq) ([]*model.WorkorderInstance, error)
}

type instanceService struct {
	dao                 dao.WorkorderInstanceDAO
	flowDao             dao.WorkorderInstanceFlowDAO
	timelineDao         dao.WorkorderInstanceTimelineDAO
	commentDao          dao.WorkorderInstanceCommentDAO
	attachmentDao       dao.WorkorderCommentAttachmentDAO
	processDao          dao.WorkorderProcessDAO
	formDesignDao       dao.WorkorderFormDesignDAO
	templateDao         dao.WorkorderTemplateDAO
	notificationService WorkorderNotificationService
	logger              *zap.Logger
}

func NewInstanceService(
	dao dao.WorkorderInstanceDAO,
	flowDao dao.WorkorderInstanceFlowDAO,
	timelineDao dao.WorkorderInstanceTimelineDAO,
	commentDao dao.WorkorderInstanceCommentDAO,
	attachmentDao dao.WorkorderCommentAttachmentDAO,
	processDao dao.WorkorderProcessDAO,
	formDesignDao dao.WorkorderFormDesignDAO,
	templateDao dao.WorkorderTemplateDAO,
	notificationService WorkorderNotificationService,
	logger *zap.Logger,
) InstanceService {
	return &instanceService{
		dao:                 dao,
		flowDao:             flowDao,
		timelineDao:         timelineDao,
		commentDao:          commentDao,
		attachmentDao:       attachmentDao,
		processDao:          processDao,
		formDesignDao:       formDesignDao,
		templateDao:         templateDao,
		notificationService: notificationService,
		logger:              logger,
	}
}

func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateWorkorderInstanceReq) error {
	if req.Status < model.InstanceStatusDraft || req.Status > model.InstanceStatusCancelled {
		return fmt.Errorf("工单状态无效")
	}
	if req.Priority < model.PriorityHigh || req.Priority > model.PriorityLow {
		return fmt.Errorf("优先级无效")
	}

	// 确保实例名称唯一
	if _, err := s.dao.GetInstanceByTitle(ctx, req.Title); err != nil {
		if err != dao.ErrInstanceNotFound {
			s.logger.Error("获取工单实例失败", zap.Error(err), zap.String("title", req.Title))
			return err
		}
	} else {
		return fmt.Errorf("工单实例名称已存在")
	}

	process, err := s.processDao.GetProcessByID(ctx, req.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
		return fmt.Errorf("流程不存在或已停用")
	}

	if process.Status != model.ProcessStatusPublished {
		return fmt.Errorf("只能使用已发布的流程创建工单")
	}

	if err := s.validateFormData(ctx, process.FormDesignID, req.FormData); err != nil {
		s.logger.Error("表单数据验证失败", zap.Error(err), zap.Int("formDesignID", process.FormDesignID))
		return fmt.Errorf("表单数据验证失败: %w", err)
	}

	// 生成工单编号
	serialNumber, err := s.dao.GenerateSerialNumber(ctx)
	if err != nil {
		s.logger.Error("生成工单编号失败", zap.Error(err))
		return err
	}

	// 根据流程定义设置初始当前步骤
	var initialStepID *string
	if process.Definition != nil {
		var definition model.ProcessDefinition
		definitionBytes, _ := json.Marshal(process.Definition)
		if json.Unmarshal(definitionBytes, &definition) == nil && len(definition.Steps) > 0 {
			// 如果是草稿状态，设置为开始步骤；否则设置为第一个非开始步骤
			if req.Status == model.InstanceStatusDraft {
				for _, step := range definition.Steps {
					if step.Type == model.ProcessStepTypeStart {
						initialStepID = &step.ID
						break
					}
				}
			} else {
				// 对于已提交的工单，设置为第一个非开始步骤
				for _, step := range definition.Steps {
					if step.Type != model.ProcessStepTypeStart {
						initialStepID = &step.ID
						break
					}
				}
			}
		}
	}

	instance := &model.WorkorderInstance{
		Title:         req.Title,
		SerialNumber:  serialNumber,
		ProcessID:     req.ProcessID,
		CurrentStepID: initialStepID,
		FormData:      req.FormData,
		Status:        req.Status,
		Priority:      req.Priority,
		OperatorID:    req.OperatorID,
		OperatorName:  req.OperatorName,
		AssigneeID:    req.AssigneeID,
		Description:   req.Description,
		Tags:          req.Tags,
		DueDate:       req.DueDate,
	}

	if err := s.dao.CreateInstance(ctx, instance); err != nil {
		s.logger.Error("创建工单实例失败", zap.Error(err))
		return fmt.Errorf("创建工单实例失败: %w", err)
	}

	s.createFlowRecord(ctx, instance.ID, model.FlowActionSubmit, req.OperatorID, req.OperatorName,
		model.InstanceStatusDraft, req.Status, "", model.FlowRecordTypeSystem)

	s.createTimelineRecord(ctx, instance.ID, model.TimelineActionCreate, req.OperatorID, req.OperatorName, "工单创建")

	s.sendNotificationAsync(instance.ID, model.EventTypeInstanceCreated)

	return nil
}

func (s *instanceService) CreateInstanceFromTemplate(ctx context.Context, templateID int, req *model.CreateWorkorderInstanceFromTemplateReq) (int, error) {
	if req.Priority < model.PriorityHigh || req.Priority > model.PriorityLow {
		return 0, fmt.Errorf("优先级无效")
	}

	template, err := s.templateDao.GetTemplate(ctx, templateID)
	if err != nil {
		s.logger.Error("获取工单模板失败", zap.Error(err), zap.Int("templateID", templateID))
		return 0, fmt.Errorf("工单模板不存在或已禁用")
	}

	if template.Status != model.TemplateStatusEnabled {
		return 0, fmt.Errorf("只能使用启用状态的模板创建工单")
	}

	// 合并表单数据：模板默认值 + 用户提交的数据
	formData := make(model.JSONMap)

	// 先使用模板的默认值
	if template.DefaultValues != nil {
		for key, value := range template.DefaultValues {
			formData[key] = value
		}
	}

	if req.FormData != nil {
		for key, value := range req.FormData {
			formData[key] = value
		}
	}

	process, err := s.processDao.GetProcessByID(ctx, template.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", template.ProcessID))
		return 0, fmt.Errorf("流程不存在或已停用")
	}

	if process.Status != model.ProcessStatusPublished {
		return 0, fmt.Errorf("只能使用已发布的流程创建工单")
	}

	if err := s.validateFormData(ctx, process.FormDesignID, formData); err != nil {
		s.logger.Error("表单数据验证失败", zap.Error(err), zap.Int("formDesignID", process.FormDesignID))
		return 0, fmt.Errorf("表单数据验证失败: %w", err)
	}

	// 确保实例名称唯一
	if _, err := s.dao.GetInstanceByTitle(ctx, req.Title); err != nil {
		if err != dao.ErrInstanceNotFound {
			s.logger.Error("获取工单实例失败", zap.Error(err), zap.String("title", req.Title))
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("工单实例名称已存在")
	}

	// 生成工单编号
	serialNumber, err := s.dao.GenerateSerialNumber(ctx)
	if err != nil {
		s.logger.Error("生成工单编号失败", zap.Error(err))
		return 0, err
	}

	instance := &model.WorkorderInstance{
		Title:        req.Title,
		SerialNumber: serialNumber,
		ProcessID:    template.ProcessID,
		FormData:     formData,
		Status:       model.InstanceStatusDraft, // 从模板创建的工单默认为草稿状态
		Priority:     req.Priority,
		OperatorID:   req.OperatorID,
		OperatorName: req.OperatorName,
		AssigneeID:   req.AssigneeID,
		Description:  req.Description,
		Tags:         req.Tags,
		DueDate:      req.DueDate,
	}

	if err := s.dao.CreateInstance(ctx, instance); err != nil {
		s.logger.Error("创建工单实例失败", zap.Error(err))
		return 0, fmt.Errorf("创建工单实例失败: %w", err)
	}

	s.createFlowRecord(ctx, instance.ID, model.FlowActionSubmit, req.OperatorID, req.OperatorName,
		model.InstanceStatusDraft, model.InstanceStatusDraft, "", model.FlowRecordTypeSystem)

	s.createTimelineRecord(ctx, instance.ID, model.TimelineActionCreate, req.OperatorID, req.OperatorName, fmt.Sprintf("从模板 %s 创建工单", template.Name))

	s.sendNotificationAsync(instance.ID, model.EventTypeInstanceCreated, fmt.Sprintf("从模板 %s 创建", template.Name))

	return instance.ID, nil
}

func (s *instanceService) UpdateInstance(ctx context.Context, req *model.UpdateWorkorderInstanceReq) error {
	ins, err := s.dao.GetInstanceByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", req.ID))
		return err
	}

	// 只有草稿和待处理状态可以更新
	if ins.Status != model.InstanceStatusDraft && ins.Status != model.InstanceStatusPending {
		return fmt.Errorf("当前状态的工单不允许修改")
	}

	// 确保实例名称唯一 (排除当前实例)
	if instance, err := s.dao.GetInstanceByTitle(ctx, req.Title); err != nil {
		if err != dao.ErrInstanceNotFound {
			s.logger.Error("获取工单实例失败", zap.Error(err), zap.String("title", req.Title))
			return err
		}
	} else if instance.ID != req.ID {
		return fmt.Errorf("工单实例名称已存在")
	}

	instance := &model.WorkorderInstance{
		Model:       model.Model{ID: req.ID},
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Tags:        req.Tags,
		DueDate:     req.DueDate,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
		FormData:    req.FormData,
		CompletedAt: req.CompletedAt,
	}

	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单实例失败", zap.Error(err), zap.Int("instanceID", req.ID))
		return err
	}

	s.sendNotificationAsync(req.ID, model.EventTypeInstanceUpdated, "工单信息已更新")

	return nil
}

func (s *instanceService) DeleteInstance(ctx context.Context, id int, operatorID int) error {
	if id <= 0 {
		return ErrInvalidRequest
	}

	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	if operatorID <= 0 || instance.OperatorID != operatorID {
		return fmt.Errorf("仅工单创建者可以删除")
	}

	if err := s.dao.DeleteInstance(ctx, id); err != nil {
		s.logger.Error("删除工单实例失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	s.sendNotificationAsync(id, model.EventTypeInstanceDeleted, fmt.Sprintf("工单 %s 已删除", instance.Title))

	return nil
}

func (s *instanceService) GetInstance(ctx context.Context, id int) (*model.WorkorderInstance, error) {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		// 定时同步/列表补全会查询已删工单，降为 Warn 避免刷屏
		s.logger.Warn("获取工单实例失败", zap.Error(err), zap.Int("instanceID", id))
		return nil, err
	}

	return instance, nil
}

// MarkNotificationRead 打开工单详情视为已读，停止对该用户的未读催发
func (s *instanceService) MarkNotificationRead(ctx context.Context, instanceID, userID int) {
	if s.notificationService == nil || instanceID <= 0 || userID <= 0 {
		return
	}
	if err := s.notificationService.AcknowledgeByUser(ctx, instanceID, userID); err != nil {
		s.logger.Warn("标记通知已读失败",
			zap.Error(err),
			zap.Int("instance_id", instanceID),
			zap.Int("user_id", userID))
	}
}

func (s *instanceService) stopNotificationReminders(ctx context.Context, instanceID int) {
	if s.notificationService == nil || instanceID <= 0 {
		return
	}
	if err := s.notificationService.StopRemindersByInstance(ctx, instanceID); err != nil {
		s.logger.Warn("停止工单催发失败",
			zap.Error(err),
			zap.Int("instance_id", instanceID))
	}
}

func (s *instanceService) ListInstance(ctx context.Context, req *model.ListWorkorderInstanceReq) (*model.ListResp[*model.WorkorderInstance], error) {
	result, total, err := s.dao.ListInstance(ctx, req)
	if err != nil {
		s.logger.Error("获取工单实例列表失败", zap.Error(err))
		return nil, err
	}
	if result == nil {
		result = []*model.WorkorderInstance{}
	}

	return &model.ListResp[*model.WorkorderInstance]{
		Items: result,
		Total: total,
	}, nil
}

// SubmitInstance 提交工单
func (s *instanceService) SubmitInstance(ctx context.Context, id int, operatorID int, operatorName string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以提交")
	}

	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionSubmit {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限提交此工单")
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusPending

	// 获取流程定义，确定提交后的第一个实际步骤
	process, err := s.processDao.GetProcessByID(ctx, instance.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", instance.ProcessID))
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	if process.Definition != nil {
		var definition model.ProcessDefinition
		definitionBytes, _ := json.Marshal(process.Definition)
		if json.Unmarshal(definitionBytes, &definition) == nil {
			// 找到第一个非开始步骤作为提交后的当前步骤
			for _, step := range definition.Steps {
				if step.Type != model.ProcessStepTypeStart {
					instance.CurrentStepID = &step.ID
					s.logger.Info("设置提交后的当前步骤",
						zap.Int("instanceID", id),
						zap.String("stepID", step.ID),
						zap.String("stepName", step.Name))
					break
				}
			}
		}
	}

	instance.Status = toStatus
	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		return err
	}

	s.createFlowRecord(ctx, id, model.FlowActionSubmit, operatorID, operatorName, fromStatus, toStatus, "", 2)

	s.createTimelineRecord(ctx, id, model.TimelineActionSubmit, operatorID, operatorName, "工单提交")

	// 发送工单提交通知
	s.sendNotificationAsync(id, model.EventTypeInstanceSubmitted)

	return nil
}

// AssignInstance 指派工单。transfer=同节点转办/协同，forward=当前节点处理完后流转到下一节点。
func (s *instanceService) AssignInstance(ctx context.Context, id int, assigneeID int, operatorID int, operatorName string, mode string, comment string) error {
	if mode == "" {
		mode = model.AssignModeTransfer
	}
	if assigneeID <= 0 {
		return fmt.Errorf("无效的受理人ID")
	}

	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusPending && instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("只有待处理或处理中的工单可以指派")
	}

	currentStep, err := s.GetCurrentStep(ctx, id)
	if err != nil {
		return fmt.Errorf("获取当前步骤失败: %w", err)
	}

	assigned := instance.AssigneeID != nil && *instance.AssigneeID > 0
	if assigned {
		if err := s.ensureAssigneePermission(instance, operatorID); err != nil {
			return err
		}
	} else if !s.canUserClaim(currentStep, operatorID) && instance.OperatorID != operatorID {
		return fmt.Errorf("当前用户无权领取或下发此工单")
	}

	if mode == model.AssignModeForward {
		return s.forwardInstance(ctx, instance, currentStep, assigneeID, operatorID, operatorName, comment)
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusProcessing
	if err := s.dao.UpdateInstanceAssignee(ctx, id, &assigneeID); err != nil {
		return err
	}
	if err := s.dao.UpdateInstanceStatus(ctx, id, toStatus); err != nil {
		return err
	}

	note := comment
	if note == "" {
		if assigned {
			note = fmt.Sprintf("转办给用户ID: %d", assigneeID)
		} else {
			note = fmt.Sprintf("领取/指派给用户ID: %d", assigneeID)
		}
	}

	s.createFlowRecord(ctx, id, model.FlowActionAssign, operatorID, operatorName, fromStatus, toStatus, note, 2)
	s.createTimelineRecord(ctx, id, model.TimelineActionAssign, operatorID, operatorName, note)
	s.sendNotificationAsync(id, model.EventTypeInstanceAssigned, note)
	return nil
}

func (s *instanceService) forwardInstance(ctx context.Context, instance *model.WorkorderInstance, currentStep *model.ProcessStep, assigneeID, operatorID int, operatorName, comment string) error {
	definition, err := s.loadProcessDefinition(ctx, instance.ProcessID)
	if err != nil {
		return err
	}
	nextStep := s.getNextStep(currentStep, definition)
	if nextStep == nil || nextStep.Type == model.ProcessStepTypeEnd {
		return fmt.Errorf("当前已是最后节点，请使用完成结单将工单归档")
	}

	fromStatus := instance.Status
	toStatus := s.getStatusForStep(nextStep)
	if toStatus == model.InstanceStatusCompleted {
		return fmt.Errorf("当前已是最后节点，请使用完成结单将工单归档")
	}

	instance.Status = toStatus
	instance.CurrentStepID = &nextStep.ID
	instance.AssigneeID = &assigneeID
	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		return err
	}
	if err := s.dao.UpdateInstanceAssignee(ctx, instance.ID, &assigneeID); err != nil {
		return err
	}

	note := comment
	if note == "" {
		note = fmt.Sprintf("流转到「%s」，处理人用户ID: %d", nextStep.Name, assigneeID)
	} else {
		note = fmt.Sprintf("流转到「%s」：%s", nextStep.Name, note)
	}

	s.createFlowRecord(ctx, instance.ID, model.FlowActionAssign, operatorID, operatorName, fromStatus, toStatus, note, 2)
	s.createTimelineRecord(ctx, instance.ID, model.TimelineActionAssign, operatorID, operatorName, note)
	s.sendNotificationAsync(instance.ID, model.EventTypeInstanceAssigned, note)
	return nil
}

func (s *instanceService) loadProcessDefinition(ctx context.Context, processID int) (model.ProcessDefinition, error) {
	var definition model.ProcessDefinition
	process, err := s.processDao.GetProcessByID(ctx, processID)
	if err != nil {
		return definition, fmt.Errorf("获取流程定义失败: %w", err)
	}
	definitionBytes, err := json.Marshal(process.Definition)
	if err != nil {
		return definition, fmt.Errorf("流程定义序列化失败: %w", err)
	}
	if err := json.Unmarshal(definitionBytes, &definition); err != nil {
		return definition, fmt.Errorf("流程定义解析失败: %w", err)
	}
	return definition, nil
}

func (s *instanceService) canUserClaim(step *model.ProcessStep, operatorID int) bool {
	if step == nil || operatorID <= 0 {
		return false
	}
	if len(step.AssigneeIDs) == 0 {
		return true
	}
	return s.canUserOperateStep(step, operatorID)
}

// ApproveInstance 审批通过工单；有后续节点时必须指定 nextAssigneeID
func (s *instanceService) ApproveInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string, nextAssigneeID int, attachmentIDs []int) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusPending && instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("只有待处理或处理中状态的工单可以审批")
	}

	if err := s.ensureAssigneePermission(instance, operatorID); err != nil {
		return err
	}

	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionApprove {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限审批此工单")
	}

	currentStep, err := s.GetCurrentStep(ctx, id)
	if err != nil {
		s.logger.Error("获取当前步骤失败", zap.Error(err), zap.Int("instanceID", id))
		return fmt.Errorf("获取当前步骤失败: %w", err)
	}

	definition, err := s.loadProcessDefinition(ctx, instance.ProcessID)
	if err != nil {
		return err
	}

	nextStep := s.getNextStep(currentStep, definition)
	needsNextAssignee := nextStep != nil && nextStep.Type != model.ProcessStepTypeEnd

	fromStatus := instance.Status
	var toStatus int8
	var completedAt *time.Time

	if !needsNextAssignee {
		toStatus = model.InstanceStatusCompleted
		now := time.Now()
		completedAt = &now
		s.logger.Info("工单审批完成，进入结束状态", zap.Int("instanceID", id))
	} else {
		if nextAssigneeID <= 0 {
			return fmt.Errorf("审批通过后需指定下一节点「%s」的处理人", nextStep.Name)
		}
		toStatus = s.getStatusForStep(nextStep)
		s.logger.Info("工单审批通过，进入下一步骤",
			zap.Int("instanceID", id),
			zap.String("nextStepID", nextStep.ID),
			zap.String("nextStepType", nextStep.Type),
			zap.Int("nextAssigneeID", nextAssigneeID),
			zap.Int8("nextStatus", toStatus))
	}

	instance.Status = toStatus
	if completedAt != nil {
		instance.CompletedAt = completedAt
	}

	if needsNextAssignee {
		instance.CurrentStepID = &nextStep.ID
		instance.AssigneeID = &nextAssigneeID
	} else if nextStep != nil {
		instance.CurrentStepID = &nextStep.ID
		instance.AssigneeID = nil
	} else {
		instance.CurrentStepID = nil
		instance.AssigneeID = nil
	}

	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	if needsNextAssignee {
		if err := s.dao.UpdateInstanceAssignee(ctx, id, &nextAssigneeID); err != nil {
			return fmt.Errorf("指定下一节点处理人失败: %w", err)
		}
	} else {
		if err := s.dao.UpdateInstanceAssignee(ctx, id, nil); err != nil {
			s.logger.Warn("审批完成后清空处理人失败", zap.Error(err), zap.Int("instanceID", id))
		}
	}

	flowNote := comment
	if needsNextAssignee {
		if flowNote == "" {
			flowNote = fmt.Sprintf("进入「%s」，处理人用户ID: %d", nextStep.Name, nextAssigneeID)
		} else {
			flowNote = fmt.Sprintf("进入「%s」（处理人用户ID: %d）：%s", nextStep.Name, nextAssigneeID, flowNote)
		}
	}
	s.createFlowRecord(ctx, id, model.FlowActionApprove, operatorID, operatorName, fromStatus, toStatus, flowNote, 2)

	timelineComment := "工单审批通过"
	if comment != "" {
		timelineComment = fmt.Sprintf("工单审批通过: %s", comment)
	}
	if needsNextAssignee {
		timelineComment += fmt.Sprintf("，进入步骤: %s（处理人用户ID: %d）", nextStep.Name, nextAssigneeID)
	}
	s.createTimelineRecord(ctx, id, model.TimelineActionApprove, operatorID, operatorName, timelineComment)

	if comment != "" || len(attachmentIDs) > 0 {
		content := "审批通过"
		if comment != "" {
			content = fmt.Sprintf("审批通过：%s", comment)
		}
		commentEntity := &model.WorkorderInstanceComment{
			InstanceID:   id,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			Content:      content,
			Type:         model.CommentTypeSystem,
			Status:       model.CommentStatusNormal,
			IsSystem:     1,
		}
		if err := s.commentDao.CreateInstanceComment(ctx, commentEntity); err != nil {
			s.logger.Error("创建审批评论失败", zap.Error(err), zap.Int("instanceID", id))
		} else if len(attachmentIDs) > 0 {
			if err := s.attachmentDao.BindAttachmentsToComment(ctx, commentEntity.ID, id, operatorID, attachmentIDs); err != nil {
				s.logger.Error("绑定审批附件失败", zap.Error(err), zap.Int("instanceID", id), zap.Int("commentID", commentEntity.ID))
				return fmt.Errorf("绑定审批附件失败: %w", err)
			}
		}
	}

	eventType := model.EventTypeInstanceApproved
	if toStatus == model.InstanceStatusCompleted {
		eventType = model.EventTypeInstanceCompleted
		s.stopNotificationReminders(ctx, id)
	}
	s.sendNotificationAsync(id, eventType, comment)
	if needsNextAssignee {
		s.sendNotificationAsync(id, model.EventTypeInstanceAssigned, fmt.Sprintf("已指派处理「%s」", nextStep.Name))
	}

	return nil
}

// RejectInstance 拒绝工单
func (s *instanceService) RejectInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusPending && instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("只有待处理或处理中状态的工单可以拒绝")
	}

	if err := s.ensureAssigneePermission(instance, operatorID); err != nil {
		return err
	}

	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionReject {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限拒绝此工单")
	}

	if comment == "" {
		return fmt.Errorf("拒绝工单必须提供理由")
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusRejected

	if err := s.dao.UpdateInstanceStatus(ctx, id, toStatus); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	s.createFlowRecord(ctx, id, model.FlowActionReject, operatorID, operatorName, fromStatus, toStatus, comment, 2)

	s.createTimelineRecord(ctx, id, model.TimelineActionReject, operatorID, operatorName, fmt.Sprintf("工单审批拒绝: %s", comment))

	// 添加拒绝原因的系统评论
	commentEntity := &model.WorkorderInstanceComment{
		InstanceID:   id,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Content:      fmt.Sprintf("审批拒绝：%s", comment),
		Type:         model.CommentTypeSystem,
		Status:       model.CommentStatusNormal,
		IsSystem:     1,
	}

	if err := s.commentDao.CreateInstanceComment(ctx, commentEntity); err != nil {
		s.logger.Error("创建拒绝评论失败", zap.Error(err), zap.Int("instanceID", id))
	}

	s.stopNotificationReminders(ctx, id)

	// 发送工单拒绝通知
	s.sendNotificationAsync(id, model.EventTypeInstanceRejected, comment)

	return nil
}

func (s *instanceService) createFlowRecord(ctx context.Context, instanceID int, action string, operatorID int, operatorName string, fromStatus, toStatus int8, comment string, isSystem int8) {
	flow := &model.WorkorderInstanceFlow{
		InstanceID:     instanceID,
		Action:         action,
		OperatorID:     operatorID,
		OperatorName:   operatorName,
		FromStatus:     fromStatus,
		ToStatus:       toStatus,
		Comment:        comment,
		IsSystemAction: isSystem,
	}

	if err := s.flowDao.Create(ctx, flow); err != nil {
		s.logger.Error("创建流转记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
	}
}

func (s *instanceService) createTimelineRecord(ctx context.Context, instanceID int, action string, operatorID int, operatorName string, comment string) {
	timeline := &model.WorkorderInstanceTimeline{
		InstanceID:   instanceID,
		Action:       action,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		ActionDetail: "", // 简单操作不需要详细信息
		Comment:      comment,
		RelatedID:    nil, // 无关联记录
	}

	if err := s.timelineDao.Create(ctx, timeline); err != nil {
		s.logger.Error("创建时间线记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
	}
}

func (s *instanceService) validateFormData(ctx context.Context, formDesignID int, formData model.JSONMap) error {
	if formDesignID <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	formDesign, err := s.formDesignDao.GetFormDesign(ctx, formDesignID)
	if err != nil {
		return fmt.Errorf("获取表单设计失败: %w", err)
	}

	if formDesign.Status != model.FormDesignStatusPublished {
		return fmt.Errorf("只能使用已发布的表单设计")
	}

	// 解析表单Schema
	var schema model.FormSchema
	schemaBytes, err := json.Marshal(formDesign.Schema)
	if err != nil {
		return fmt.Errorf("表单Schema序列化失败: %w", err)
	}

	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return fmt.Errorf("表单Schema解析失败: %w", err)
	}

	for _, field := range schema.Fields {
		if err := s.validateFormField(field, formData); err != nil {
			return fmt.Errorf("字段 %s 验证失败: %w", field.Label, err)
		}
	}

	return nil
}

func (s *instanceService) validateFormField(field model.FormField, formData model.JSONMap) error {
	value, exists := formData[field.ID]

	if !exists {
		if err := s.validateFieldRequired(field, nil); err != nil {
			return err
		}
	} else {
		if err := s.validateFieldRequired(field, value); err != nil {
			return err
		}
	}

	// 如果字段不存在或为空，且不是必填的，则跳过验证
	if !exists || s.isEmptyValue(value) {
		return nil
	}

	switch field.Type {
	case model.FormFieldTypeText, model.FormFieldTypePassword, model.FormFieldTypeTextarea:
		return s.validateStringField(field, value)
	case model.FormFieldTypeNumber:
		return s.validateNumberField(field, value)
	case model.FormFieldTypeSelect, model.FormFieldTypeRadio:
		return s.validateSelectField(field, value)
	case model.FormFieldTypeCheckbox:
		return s.validateCheckboxField(field, value)
	case model.FormFieldTypeDate:
		return s.validateDateField(field, value)
	case model.FormFieldTypeSwitch:
		return s.validateSwitchField(field, value)
	default:
		s.logger.Warn("未知的字段类型", zap.String("type", field.Type), zap.String("fieldID", field.ID))
		return nil
	}
}

func (s *instanceService) isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func (s *instanceService) validateStringField(field model.FormField, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("期望字符串类型，实际类型: %T", value)
	}

	// 这里可以添加更多的字符串验证逻辑，如长度限制等
	if len(str) > 2000 { // 假设最大长度为2000
		return fmt.Errorf("字符串长度超过限制")
	}

	return nil
}

func (s *instanceService) validateNumberField(field model.FormField, value interface{}) error {
	switch v := value.(type) {
	case float64, int, int64:
		return nil
	case string:
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return fmt.Errorf("无法解析为数字: %s", v)
		}
		return nil
	default:
		return fmt.Errorf("期望数字类型，实际类型: %T", value)
	}
}

func (s *instanceService) validateSelectField(field model.FormField, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("期望字符串类型，实际类型: %T", value)
	}

	if len(field.Options) > 0 {
		for _, option := range field.Options {
			if option == str {
				return nil
			}
		}
		return fmt.Errorf("值 %s 不在可选项中", str)
	}

	return nil
}

func (s *instanceService) validateCheckboxField(field model.FormField, value interface{}) error {
	// 复选框可以是数组或单个值
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("期望字符串数组，实际包含类型: %T", item)
			}

			if len(field.Options) > 0 {
				found := false
				for _, option := range field.Options {
					if option == str {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("值 %s 不在可选项中", str)
				}
			}
		}
		return nil
	case string:
		// 单个值的情况
		return s.validateSelectField(field, v)
	default:
		return fmt.Errorf("期望字符串或字符串数组类型，实际类型: %T", value)
	}
}

func (s *instanceService) validateDateField(field model.FormField, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("期望字符串类型，实际类型: %T", value)
	}

	// 尝试解析日期
	if _, err := time.Parse("2006-01-02", str); err != nil {
		if _, err := time.Parse("2006-01-02T15:04:05Z07:00", str); err != nil {
			return fmt.Errorf("无法解析日期: %s", str)
		}
	}

	return nil
}

func (s *instanceService) validateSwitchField(field model.FormField, value interface{}) error {
	switch v := value.(type) {
	case bool:
		return nil
	case string:
		if v == "true" || v == "false" || v == "1" || v == "0" {
			return nil
		}
		return fmt.Errorf("无效的开关值: %s", v)
	case int, int64, float64:
		return nil
	default:
		return fmt.Errorf("期望布尔或数字类型，实际类型: %T", value)
	}
}

func (s *instanceService) GetCurrentStep(ctx context.Context, instanceID int) (*model.ProcessStep, error) {
	instance, err := s.dao.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	s.logger.Debug("获取到工单实例", zap.Int("processID", instance.ProcessID), zap.Int8("status", instance.Status))

	process, err := s.processDao.GetProcessByID(ctx, instance.ProcessID)
	if err != nil {
		s.logger.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", instance.ProcessID))
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	s.logger.Debug("获取到流程定义", zap.String("processName", process.Name))

	if process.Definition == nil {
		s.logger.Error("流程定义为空", zap.Int("processID", instance.ProcessID))
		return nil, fmt.Errorf("流程定义为空")
	}

	// 解析流程定义
	var definition model.ProcessDefinition
	definitionBytes, err := json.Marshal(process.Definition)
	if err != nil {
		s.logger.Error("流程定义序列化失败", zap.Error(err))
		return nil, fmt.Errorf("流程定义序列化失败: %w", err)
	}

	if err := json.Unmarshal(definitionBytes, &definition); err != nil {
		s.logger.Error("流程定义解析失败", zap.Error(err))
		return nil, fmt.Errorf("流程定义解析失败: %w", err)
	}

	s.logger.Debug("解析流程定义成功", zap.Int("stepCount", len(definition.Steps)))

	var currentStep *model.ProcessStep

	if instance.CurrentStepID != nil && *instance.CurrentStepID != "" {
		s.logger.Debug("使用CurrentStepID查找当前步骤", zap.String("currentStepID", *instance.CurrentStepID))
		for i := range definition.Steps {
			if definition.Steps[i].ID == *instance.CurrentStepID {
				currentStep = &definition.Steps[i]
				s.logger.Debug("通过CurrentStepID找到当前步骤",
					zap.String("stepID", currentStep.ID),
					zap.String("stepName", currentStep.Name),
					zap.String("stepType", currentStep.Type))
				break
			}
		}
	}

	if currentStep == nil {
		s.logger.Debug("CurrentStepID未找到步骤，使用状态映射查找", zap.Int8("status", instance.Status))
		currentStep = s.findStepByStatus(definition.Steps, instance.Status)
		if currentStep == nil {
			s.logger.Error("未找到匹配状态的流程步骤", zap.Int8("status", instance.Status), zap.Int("stepCount", len(definition.Steps)))
			return nil, fmt.Errorf("未找到匹配状态的流程步骤")
		}

		if instance.CurrentStepID == nil || *instance.CurrentStepID == "" {
			s.logger.Info("更新工单CurrentStepID",
				zap.Int("instanceID", instanceID),
				zap.String("stepID", currentStep.ID))
			instance.CurrentStepID = &currentStep.ID
			if updateErr := s.dao.UpdateInstance(ctx, instance); updateErr != nil {
				s.logger.Error("更新CurrentStepID失败", zap.Error(updateErr))
			}
		}
	}

	s.logger.Debug("找到当前步骤", zap.String("stepID", currentStep.ID),
		zap.String("stepName", currentStep.Name), zap.String("stepType", currentStep.Type))

	return currentStep, nil
}

func (s *instanceService) GetAvailableActions(ctx context.Context, instanceID int, operatorID int) ([]string, error) {
	instance, err := s.dao.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单实例失败: %w", err)
	}

	s.logger.Debug("工单实例信息", zap.Int8("status", instance.Status), zap.Int("processID", instance.ProcessID))

	currentStep, err := s.GetCurrentStep(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取当前步骤失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取当前步骤失败: %w", err)
	}

	if currentStep == nil {
		s.logger.Warn("当前步骤为空", zap.Int("instanceID", instanceID))
		return []string{}, nil
	}

	s.logger.Debug("当前步骤信息", zap.String("stepID", currentStep.ID), zap.String("stepType", currentStep.Type), zap.String("assigneeType", currentStep.AssigneeType))

	assigned := instance.AssigneeID != nil && *instance.AssigneeID > 0
	if instance.Status == model.InstanceStatusDraft {
		if instance.OperatorID == operatorID {
			return s.getActionsForStep(currentStep, instance.Status), nil
		}
		return []string{}, nil
	}

	if assigned {
		if *instance.AssigneeID != operatorID {
			return []string{}, nil
		}
		return s.getActionsForStep(currentStep, instance.Status), nil
	}

	// 未领取：可领取/下发（节点受理人或创建者）
	if s.canUserClaim(currentStep, operatorID) || instance.OperatorID == operatorID {
		return []string{model.FlowActionAssign}, nil
	}
	return []string{}, nil
}

func (s *instanceService) findStepByStatus(steps []model.ProcessStep, status int8) *model.ProcessStep {
	if len(steps) == 0 {
		s.logger.Warn("流程步骤为空")
		return nil
	}

	switch status {
	case model.InstanceStatusDraft:
		for i := range steps {
			if steps[i].Type == model.ProcessStepTypeStart {
				return &steps[i]
			}
		}
		s.logger.Debug("未找到开始步骤，使用第一个步骤", zap.Int8("status", status))
	case model.InstanceStatusPending, model.InstanceStatusProcessing:
		// 待处理/处理中状态对应审批或任务步骤
		for i := range steps {
			if steps[i].Type == model.ProcessStepTypeApproval || steps[i].Type == model.ProcessStepTypeTask {
				return &steps[i]
			}
		}
		s.logger.Debug("未找到审批或任务步骤，使用第一个步骤", zap.Int8("status", status))
	case model.InstanceStatusCompleted, model.InstanceStatusRejected, model.InstanceStatusCancelled:
		for i := range steps {
			if steps[i].Type == model.ProcessStepTypeEnd {
				return &steps[i]
			}
		}
		s.logger.Debug("未找到结束步骤，使用第一个步骤", zap.Int8("status", status))
	}

	// 如果没找到合适的步骤，返回第一个步骤作为默认
	s.logger.Info("使用默认步骤", zap.Int8("status", status), zap.String("stepType", steps[0].Type))
	return &steps[0]
}

func (s *instanceService) canUserOperate(step *model.ProcessStep, operatorID int, assigneeID *int) bool {
	if step == nil {
		s.logger.Warn("步骤为空，拒绝操作")
		return false
	}

	// 已指派时仅当前处理人可操作（创建人/管理员也不例外）
	if assigneeID != nil && *assigneeID > 0 {
		allowed := *assigneeID == operatorID
		if !allowed {
			s.logger.Info("工单已指派给其他用户，拒绝操作",
				zap.Int("operatorID", operatorID),
				zap.Int("assigneeID", *assigneeID))
		}
		return allowed
	}

	switch step.AssigneeType {
	case model.AssigneeTypeUser, model.AssigneeTypeGroup:
		return s.canUserOperateStep(step, operatorID)
	case "":
		s.logger.Warn("步骤未配置受理人类型，拒绝操作")
		return false
	default:
		s.logger.Warn("未知的受理人类型，拒绝操作", zap.String("assigneeType", step.AssigneeType))
		return false
	}
}

// ensureAssigneePermission 已指派时强制校验当前操作人必须是处理人
func (s *instanceService) ensureAssigneePermission(instance *model.WorkorderInstance, operatorID int) error {
	if instance == nil {
		return fmt.Errorf("工单不存在")
	}
	if instance.AssigneeID != nil && *instance.AssigneeID > 0 && *instance.AssigneeID != operatorID {
		return fmt.Errorf("工单已指派给其他处理人，您无权操作")
	}
	return nil
}

func (s *instanceService) getActionsForStep(step *model.ProcessStep, currentStatus int8) []string {
	var actions []string

	// 基础动作：从步骤定义中获取
	actions = append(actions, step.Actions...)

	// 根据当前状态添加额外的动作
	switch currentStatus {
	case model.InstanceStatusDraft:
		actions = append(actions, model.FlowActionSubmit, model.FlowActionCancel)
	case model.InstanceStatusPending:
		actions = append(actions, model.FlowActionAssign, model.FlowActionApprove, model.FlowActionReject, model.FlowActionCancel)
	case model.InstanceStatusProcessing:
		actions = append(actions, model.FlowActionAssign, model.FlowActionComplete, model.FlowActionReturn, model.FlowActionApprove, model.FlowActionReject, model.FlowActionCancel)
	}

	// 去重
	actionSet := make(map[string]bool)
	var uniqueActions []string
	for _, action := range actions {
		if !actionSet[action] {
			actionSet[action] = true
			uniqueActions = append(uniqueActions, action)
		}
	}

	return uniqueActions
}

func (s *instanceService) getNextStep(currentStep *model.ProcessStep, definition model.ProcessDefinition) *model.ProcessStep {
	if currentStep == nil {
		s.logger.Warn("当前步骤为空，无法获取下一个步骤")
		return nil
	}

	// 查找从当前步骤出发的连接
	var nextStepID string
	for _, connection := range definition.Connections {
		if connection.From == currentStep.ID {
			nextStepID = connection.To
			break
		}
	}

	if nextStepID == "" {
		s.logger.Info("未找到当前步骤的下一个步骤", zap.String("currentStepID", currentStep.ID))
		return nil
	}

	for i := range definition.Steps {
		if definition.Steps[i].ID == nextStepID {
			s.logger.Debug("找到下一个步骤",
				zap.String("currentStepID", currentStep.ID),
				zap.String("nextStepID", nextStepID),
				zap.String("nextStepType", definition.Steps[i].Type))
			return &definition.Steps[i]
		}
	}

	s.logger.Warn("未找到下一个步骤的详细信息",
		zap.String("currentStepID", currentStep.ID),
		zap.String("nextStepID", nextStepID))
	return nil
}

func (s *instanceService) getStatusForStep(step *model.ProcessStep) int8 {
	if step == nil {
		s.logger.Warn("步骤为空，返回默认状态")
		return model.InstanceStatusPending
	}

	switch step.Type {
	case model.ProcessStepTypeStart:
		return model.InstanceStatusDraft
	case model.ProcessStepTypeApproval:
		return model.InstanceStatusPending
	case model.ProcessStepTypeTask:
		return model.InstanceStatusProcessing
	case model.ProcessStepTypeEnd:
		return model.InstanceStatusCompleted
	default:
		s.logger.Warn("未知的步骤类型，返回默认状态",
			zap.String("stepType", step.Type),
			zap.String("stepID", step.ID))
		return model.InstanceStatusPending
	}
}

// CancelInstance 取消工单
func (s *instanceService) CancelInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status == model.InstanceStatusCompleted || instance.Status == model.InstanceStatusCancelled {
		return fmt.Errorf("已完成或已取消的工单不能再次取消")
	}

	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionCancel {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限取消此工单")
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusCancelled

	if err := s.dao.UpdateInstanceStatus(ctx, id, toStatus); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	s.createFlowRecord(ctx, id, model.FlowActionCancel, operatorID, operatorName, fromStatus, toStatus, comment, 2)

	s.createTimelineRecord(ctx, id, model.TimelineActionCancel, operatorID, operatorName, fmt.Sprintf("工单已取消: %s", comment))

	// 添加取消原因的系统评论
	if comment != "" {
		commentEntity := &model.WorkorderInstanceComment{
			InstanceID:   id,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			Content:      fmt.Sprintf("工单取消：%s", comment),
			Type:         model.CommentTypeSystem,
			Status:       model.CommentStatusNormal,
			IsSystem:     1,
		}

		if err := s.commentDao.CreateInstanceComment(ctx, commentEntity); err != nil {
			s.logger.Error("创建取消评论失败", zap.Error(err), zap.Int("instanceID", id))
		}
	}

	s.stopNotificationReminders(ctx, id)

	// 发送工单取消通知
	s.sendNotificationAsync(id, model.EventTypeInstanceCancelled, comment)

	return nil
}

// CompleteInstance 完成工单
func (s *instanceService) CompleteInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("只有处理中状态的工单可以完成")
	}

	if err := s.ensureAssigneePermission(instance, operatorID); err != nil {
		return err
	}

	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionComplete {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限完成此工单")
	}

	currentStep, err := s.GetCurrentStep(ctx, id)
	if err != nil {
		return fmt.Errorf("获取当前步骤失败: %w", err)
	}
	if currentStep != nil {
		definition, defErr := s.loadProcessDefinition(ctx, instance.ProcessID)
		if defErr == nil {
			nextStep := s.getNextStep(currentStep, definition)
			if nextStep != nil && nextStep.Type != model.ProcessStepTypeEnd {
				return fmt.Errorf("当前还有下一节点「%s」，请使用流转指派给下一级处理人", nextStep.Name)
			}
		}
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusCompleted

	// 更新工单状态为已完成，并设置完成时间
	now := time.Now()
	instance.Status = toStatus
	instance.CompletedAt = &now

	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	s.createFlowRecord(ctx, id, model.FlowActionComplete, operatorID, operatorName, fromStatus, toStatus, comment, 2)

	s.createTimelineRecord(ctx, id, model.TimelineActionComplete, operatorID, operatorName, fmt.Sprintf("工单已完成: %s", comment))

	// 添加完成说明的系统评论
	if comment != "" {
		commentEntity := &model.WorkorderInstanceComment{
			InstanceID:   id,
			OperatorID:   operatorID,
			OperatorName: operatorName,
			Content:      fmt.Sprintf("工单完成：%s", comment),
			Type:         model.CommentTypeSystem,
			Status:       model.CommentStatusNormal,
			IsSystem:     1,
		}

		if err := s.commentDao.CreateInstanceComment(ctx, commentEntity); err != nil {
			s.logger.Error("创建完成评论失败", zap.Error(err), zap.Int("instanceID", id))
		}
	}

	s.stopNotificationReminders(ctx, id)

	// 发送工单完成通知
	s.sendNotificationAsync(id, model.EventTypeInstanceCompleted, comment)

	return nil
}

// ReturnInstance 退回工单
func (s *instanceService) ReturnInstance(ctx context.Context, id int, operatorID int, operatorName string, comment string) error {
	instance, err := s.dao.GetInstanceByID(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusPending && instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("只有待处理或处理中状态的工单可以退回")
	}

	if err := s.ensureAssigneePermission(instance, operatorID); err != nil {
		return err
	}

	availableActions, err := s.GetAvailableActions(ctx, id, operatorID)
	if err != nil {
		return fmt.Errorf("获取可用动作失败: %w", err)
	}

	actionAllowed := false
	for _, action := range availableActions {
		if action == model.FlowActionReturn {
			actionAllowed = true
			break
		}
	}

	if !actionAllowed {
		return fmt.Errorf("当前用户无权限退回此工单")
	}

	if comment == "" {
		return fmt.Errorf("退回工单必须提供理由")
	}

	fromStatus := instance.Status
	toStatus := model.InstanceStatusDraft // 退回到草稿状态

	// 更新工单状态为草稿，清空受理人
	instance.Status = toStatus
	instance.AssigneeID = nil

	if err := s.dao.UpdateInstance(ctx, instance); err != nil {
		s.logger.Error("更新工单状态失败", zap.Error(err), zap.Int("instanceID", id))
		return err
	}

	s.createFlowRecord(ctx, id, model.FlowActionReturn, operatorID, operatorName, fromStatus, toStatus, comment, 2)

	s.createTimelineRecord(ctx, id, model.TimelineActionReturn, operatorID, operatorName, fmt.Sprintf("工单已退回: %s", comment))

	// 添加退回原因的系统评论
	commentEntity := &model.WorkorderInstanceComment{
		InstanceID:   id,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Content:      fmt.Sprintf("工单退回：%s", comment),
		Type:         model.CommentTypeSystem,
		Status:       model.CommentStatusNormal,
		IsSystem:     1,
	}

	if err := s.commentDao.CreateInstanceComment(ctx, commentEntity); err != nil {
		s.logger.Error("创建退回评论失败", zap.Error(err), zap.Int("instanceID", id))
	}

	// 发送工单退回通知
	s.sendNotificationAsync(id, model.EventTypeInstanceReturned, comment)

	return nil
}

func (s *instanceService) validateFieldRequired(field model.FormField, value interface{}) error {
	if field.Required == model.FieldRequiredYes && s.isEmptyValue(value) {
		return fmt.Errorf("字段 %s 为必填项", field.Label)
	}
	return nil
}

func (s *instanceService) canUserOperateStep(step *model.ProcessStep, operatorID int) bool {
	if len(step.AssigneeIDs) == 0 {
		return false
	}

	for _, assigneeID := range step.AssigneeIDs {
		if assigneeID == operatorID {
			return true
		}
	}

	return false
}

func (s *instanceService) sendNotificationAsync(instanceID int, eventType string, customContent ...string) {
	if s.notificationService == nil || instanceID <= 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := s.notificationService.SendWorkorderNotification(ctx, instanceID, eventType, customContent...); err != nil {
			s.logger.Error("发送工单通知失败",
				zap.Error(err),
				zap.Int("instance_id", instanceID),
				zap.String("event_type", eventType))
		}
	}()
}

func (s *instanceService) getDefaultPageSize() int {
	return 20 // 默认分页大小
}

func (s *instanceService) getMaxPageSize() int {
	return 100 // 最大分页大小
}

// ExportInstance 导出工单实例列表
func (s *instanceService) ExportInstance(ctx context.Context, req *model.ExportWorkorderInstanceReq) ([]*model.WorkorderInstance, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 5000
	}
	if limit > 10000 {
		limit = 10000
	}
	listReq := &model.ListWorkorderInstanceReq{
		ListReq: model.ListReq{
			Page:   1,
			Size:   limit,
			Search: req.Search,
		},
		Status:    req.Status,
		Priority:  req.Priority,
		ProcessID: req.ProcessID,
		Scope:     req.Scope,
		UserID:    req.UserID,
	}
	resp, err := s.ListInstance(ctx, listReq)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}
