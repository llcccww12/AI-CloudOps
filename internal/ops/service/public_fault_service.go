package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	opsUtils "github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	workorderService "github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	workorderUtils "github.com/GoSimplicity/AI-CloudOps/internal/workorder/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type OpsPublicFaultService interface {
	Submit(ctx context.Context, req *model.SubmitPublicFaultReq) (*model.SubmitPublicFaultResp, error)
	Query(ctx context.Context, req *model.QueryPublicFaultReq) (*model.QueryPublicFaultResp, error)
}

type opsPublicFaultService struct {
	customerDAO dao.OpsCustomerDAO
	instanceDAO workorderDao.WorkorderInstanceDAO
	instanceSvc workorderService.InstanceService
	logger      *zap.Logger
}

func NewOpsPublicFaultService(
	customerDAO dao.OpsCustomerDAO,
	instanceDAO workorderDao.WorkorderInstanceDAO,
	instanceSvc workorderService.InstanceService,
	logger *zap.Logger,
) OpsPublicFaultService {
	return &opsPublicFaultService{
		customerDAO: customerDAO,
		instanceDAO: instanceDAO,
		instanceSvc: instanceSvc,
		logger:      logger,
	}
}

func (s *opsPublicFaultService) Submit(ctx context.Context, req *model.SubmitPublicFaultReq) (*model.SubmitPublicFaultResp, error) {
	customer, err := s.verifyReportCredential(ctx, req.ReportCode, req.ReportSecret)
	if err != nil {
		return nil, err
	}

	templateID := viper.GetInt("ops.public_fault_workorder_template_id")
	if templateID <= 0 {
		return nil, fmt.Errorf("系统未配置客户报障工单模板，请联系管理员")
	}

	queryCode := opsUtils.GenerateQueryCode()
	queryHash, err := userutils.HashPassword(queryCode)
	if err != nil {
		return nil, fmt.Errorf("生成查询码失败: %w", err)
	}

	operatorID := customer.OwnerID
	operatorName := customer.OwnerName
	if operatorID <= 0 {
		operatorID = customer.OperatorID
		operatorName = customer.OperatorName
	}
	if operatorName == "" {
		operatorName = "系统"
	}

	formData := model.JSONMap{
		"customer_id":    customer.ID,
		"customer_name":  customer.Name,
		"reporter_name":  strings.TrimSpace(req.ReporterName),
		"reporter_phone": strings.TrimSpace(req.ReporterPhone),
		"reporter_email": strings.TrimSpace(req.ReporterEmail),
		"fault_title":    strings.TrimSpace(req.Title),
		"source":         model.WorkorderSourcePublicFault,
	}
	if len(req.AttachmentIDs) > 0 {
		formData["attachment_ids"] = req.AttachmentIDs
	}

	instance, err := s.instanceSvc.CreatePublicFaultInstance(ctx, templateID, &model.CreatePublicFaultInstanceReq{
		Title:           strings.TrimSpace(req.Title),
		Description:     strings.TrimSpace(req.Description),
		Priority:        req.Priority,
		FormData:        formData,
		OperatorID:      operatorID,
		OperatorName:    operatorName,
		OpsCustomerID:   customer.ID,
		ReporterName:    strings.TrimSpace(req.ReporterName),
		ReporterPhone:   strings.TrimSpace(req.ReporterPhone),
		ReporterEmail:   strings.TrimSpace(req.ReporterEmail),
		PublicQueryHash: queryHash,
	})
	if err != nil {
		return nil, err
	}

	return &model.SubmitPublicFaultResp{
		SerialNumber: instance.SerialNumber,
		QueryCode:    queryCode,
		Message:      "报障已提交，请妥善保存工单号与查询码",
	}, nil
}

func (s *opsPublicFaultService) Query(ctx context.Context, req *model.QueryPublicFaultReq) (*model.QueryPublicFaultResp, error) {
	serial := strings.TrimSpace(req.SerialNumber)
	queryCode := strings.TrimSpace(req.QueryCode)
	instance, err := s.instanceDAO.GetInstanceBySerialNumber(ctx, serial)
	if err != nil {
		return nil, fmt.Errorf("工单不存在或查询码错误")
	}
	if instance.Source != model.WorkorderSourcePublicFault {
		return nil, fmt.Errorf("工单不存在或查询码错误")
	}
	if instance.PublicQueryCodeHash == "" {
		return nil, fmt.Errorf("工单不存在或查询码错误")
	}
	if err := userutils.ComparePassword(instance.PublicQueryCodeHash, queryCode); err != nil {
		return nil, fmt.Errorf("工单不存在或查询码错误")
	}

	timeline := make([]model.PublicFaultTimelineItem, 0, len(instance.Timeline))
	for _, item := range instance.Timeline {
		timeline = append(timeline, model.PublicFaultTimelineItem{
			Time:    item.CreatedAt,
			Action:  timelineActionLabel(item.Action),
			Content: strings.TrimSpace(item.Comment),
		})
	}

	return &model.QueryPublicFaultResp{
		SerialNumber: instance.SerialNumber,
		Title:        publicFaultDisplayTitle(instance.Title),
		Status:       instance.Status,
		StatusText:   workorderUtils.GetInstanceStatusName(instance.Status),
		Priority:     instance.Priority,
		PriorityText: priorityLabel(instance.Priority),
		CreatedAt:    instance.CreatedAt,
		UpdatedAt:    instance.UpdatedAt,
		CompletedAt:  instance.CompletedAt,
		Description:  instance.Description,
		Timeline:     timeline,
	}, nil
}

func (s *opsPublicFaultService) verifyReportCredential(ctx context.Context, code, secret string) (*model.OpsCustomer, error) {
	customer, err := s.customerDAO.GetByReportCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("组织编码或密钥不正确")
	}
	if customer.ReportEnabled != model.OpsReportEnabledYes {
		return nil, fmt.Errorf("该客户未开放公网报障")
	}
	if customer.ReportSecretHash == "" {
		return nil, fmt.Errorf("组织编码或密钥不正确")
	}
	if err := userutils.ComparePassword(customer.ReportSecretHash, secret); err != nil {
		return nil, fmt.Errorf("组织编码或密钥不正确")
	}
	return customer, nil
}

func publicFaultDisplayTitle(title string) string {
	title = strings.TrimSpace(title)
	if strings.HasPrefix(title, "[客户报障]") {
		if idx := strings.LastIndex(title, "-WO"); idx > 0 {
			return strings.TrimPrefix(title[:idx], "[客户报障]")
		}
		return strings.TrimPrefix(title, "[客户报障]")
	}
	return title
}

func priorityLabel(priority int8) string {
	switch priority {
	case model.PriorityHigh:
		return "高"
	case model.PriorityNormal:
		return "中"
	case model.PriorityLow:
		return "低"
	default:
		return "未知"
	}
}

func timelineActionLabel(action string) string {
	switch action {
	case model.TimelineActionCreate:
		return "创建"
	case model.TimelineActionSubmit:
		return "提交"
	case model.TimelineActionApprove:
		return "处理进展"
	case model.TimelineActionReject:
		return "退回"
	case model.TimelineActionAssign:
		return "指派"
	case model.TimelineActionCancel:
		return "取消"
	case model.TimelineActionComplete:
		return "完成"
	case model.TimelineActionReturn:
		return "退回"
	case model.TimelineActionComment:
		return "备注"
	case model.TimelineActionUpdate:
		return "更新"
	case model.TimelineActionNotify:
		return "通知"
	case model.TimelineActionRemind:
		return "催办"
	default:
		return "进展"
	}
}
