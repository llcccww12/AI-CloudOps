package api

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type OpsHandler struct {
	customerSvc   service.OpsCustomerService
	leadSvc       service.OpsLeadService
	bizSvc        service.OpsBizService
	financeSvc    service.OpsFinanceService
	reminderSvc   service.OpsReminderService
	attachmentSvc service.OpsAttachmentService
	dashboardSvc  service.OpsDashboardService
	surveySvc     service.OpsSurveyService
	billingSvc    service.OpsBillingService
	workbenchSvc  service.OpsWorkbenchService
	managerRptSvc service.OpsManagerReportService
	publicFaultSvc service.OpsPublicFaultService
}

func NewOpsHandler(
	customerSvc service.OpsCustomerService,
	leadSvc service.OpsLeadService,
	bizSvc service.OpsBizService,
	financeSvc service.OpsFinanceService,
	reminderSvc service.OpsReminderService,
	attachmentSvc service.OpsAttachmentService,
	dashboardSvc service.OpsDashboardService,
	surveySvc service.OpsSurveyService,
	billingSvc service.OpsBillingService,
	workbenchSvc service.OpsWorkbenchService,
	managerRptSvc service.OpsManagerReportService,
	publicFaultSvc service.OpsPublicFaultService,
) *OpsHandler {
	return &OpsHandler{
		customerSvc: customerSvc, leadSvc: leadSvc, bizSvc: bizSvc,
		financeSvc: financeSvc, reminderSvc: reminderSvc, attachmentSvc: attachmentSvc,
		dashboardSvc: dashboardSvc, surveySvc: surveySvc, billingSvc: billingSvc,
		workbenchSvc: workbenchSvc, managerRptSvc: managerRptSvc, publicFaultSvc: publicFaultSvc,
	}
}

func (h *OpsHandler) RegisterRouters(server *gin.Engine) {
	h.registerPublicVisitorRoutes(server)
	h.registerPublicSurveyRoutes(server)
	h.registerPublicFaultRoutes(server)

	g := server.Group("/api/ops")
	{
		g.GET("/dashboard/overview", h.GetDashboardOverview)
		g.GET("/workbench/briefing", h.GetWorkbenchBriefing)
		g.POST("/workbench/reminder-draft", h.CreateReminderDraft)
		g.GET("/manager/weekly-report", h.GetManagerWeeklyReport)

		g.POST("/customer/create", h.CreateCustomer)
		g.PUT("/customer/update/:id", h.UpdateCustomer)
		g.DELETE("/customer/delete/:id", h.DeleteCustomer)
		g.GET("/customer/detail/:id", h.GetCustomer)
		g.GET("/customer/list", h.ListCustomer)
		g.POST("/customer/change-stage", h.ChangeCustomerStage)
		g.PUT("/customer/report/:id", h.UpdateCustomerReport)
		g.POST("/customer/report/:id/rotate-secret", h.RotateCustomerReportSecret)
		g.GET("/customer/vendor-profile/:id", h.GetVendorProfile)
		g.POST("/customer/vendor-profile", h.UpsertVendorProfile)
		g.POST("/followup/create", h.CreateFollowup)
		g.GET("/followup/list", h.ListFollowup)
		g.POST("/customer/lifecycle/start/:id", h.StartCustomerLifecycle)
		g.GET("/customer/lifecycle/list/:id", h.ListCustomerLifecycle)
		g.GET("/customer/lifecycle/approve-context/:id", h.GetLifecycleApproveContext)
		g.POST("/customer/lifecycle/approve", h.ApproveLifecycleNode)

		g.POST("/exhibition/create", h.CreateExhibition)
		g.PUT("/exhibition/update/:id", h.UpdateExhibition)
		g.DELETE("/exhibition/delete/:id", h.DeleteExhibition)
		g.GET("/exhibition/detail/:id", h.GetExhibition)
		g.GET("/exhibition/list", h.ListExhibition)
		g.POST("/exhibition/convert", h.ConvertExhibition)

		g.POST("/visit/create", h.CreateVisit)
		g.PUT("/visit/update/:id", h.UpdateVisit)
		g.DELETE("/visit/delete/:id", h.DeleteVisit)
		g.GET("/visit/detail/:id", h.GetVisit)
		g.GET("/visit/list", h.ListVisit)
		g.POST("/visit/convert", h.ConvertVisit)

		g.POST("/trial/create", h.CreateTrial)
		g.PUT("/trial/update/:id", h.UpdateTrial)
		g.DELETE("/trial/delete/:id", h.DeleteTrial)
		g.GET("/trial/detail/:id", h.GetTrial)
		g.GET("/trial/list", h.ListTrial)
		g.POST("/trial/submit/:id", h.SubmitTrial)

		g.POST("/contract/create", h.CreateContract)
		g.PUT("/contract/update/:id", h.UpdateContract)
		g.DELETE("/contract/delete/:id", h.DeleteContract)
		g.GET("/contract/detail/:id", h.GetContract)
		g.GET("/contract/list", h.ListContract)

		g.POST("/activation/create", h.CreateActivation)
		g.PUT("/activation/update/:id", h.UpdateActivation)
		g.POST("/activation/feedback/:id", h.FeedbackActivation)
		g.DELETE("/activation/delete/:id", h.DeleteActivation)
		g.GET("/activation/detail/:id", h.GetActivation)
		g.GET("/activation/list", h.ListActivation)
		g.POST("/activation/submit/:id", h.SubmitActivation)

		g.POST("/settlement/create", h.CreateSettlement)
		g.PUT("/settlement/update/:id", h.UpdateSettlement)
		g.DELETE("/settlement/delete/:id", h.DeleteSettlement)
		g.GET("/settlement/detail/:id", h.GetSettlement)
		g.GET("/settlement/list", h.ListSettlement)
		g.POST("/settlement/confirm/:id", h.ConfirmSettlement)

		g.POST("/invoice/create", h.CreateInvoice)
		g.PUT("/invoice/update/:id", h.UpdateInvoice)
		g.DELETE("/invoice/delete/:id", h.DeleteInvoice)
		g.GET("/invoice/list", h.ListInvoice)

		g.POST("/payment/create", h.CreatePayment)
		g.PUT("/payment/update/:id", h.UpdatePayment)
		g.DELETE("/payment/delete/:id", h.DeletePayment)
		g.GET("/payment/list", h.ListPayment)
		g.POST("/payment/match/:id", h.MatchPayment)

		g.GET("/reminder/list", h.ListReminder)
		g.POST("/reminder/create", h.CreateReminder)
		g.PUT("/reminder/update/:id", h.UpdateReminder)
		g.GET("/reminder/preview", h.PreviewReminder)
		g.POST("/reminder/scan", h.ScanReminder)
		g.GET("/reminder/task/list", h.ListReminderTask)
		g.POST("/reminder/task/create", h.CreateReminderTask)
		g.PUT("/reminder/task/update/:id", h.UpdateReminderTask)
		g.DELETE("/reminder/task/delete/:id", h.DeleteReminderTask)
		g.GET("/reminder/delivery/list", h.ListReminderDelivery)

		g.GET("/survey/list", h.ListSurvey)
		g.POST("/survey/submit", h.SubmitSurvey)
		g.GET("/survey/response/list", h.ListSurveyResponse)
		g.POST("/survey/invite/create", h.CreateSurveyInvite)
		g.GET("/survey/invite/list", h.ListSurveyInvite)

		g.GET("/contract/item/list", h.ListContractItem)
		g.POST("/contract/item/create", h.CreateContractItem)
		g.DELETE("/contract/item/delete/:id", h.DeleteContractItem)

		g.POST("/billing/generate-monthly", h.GenerateMonthlyBilling)

		g.POST("/attachment/upload", h.UploadAttachment)
		g.GET("/attachment/list", h.ListAttachment)
		g.GET("/attachment/:id/download", h.DownloadAttachment)
		g.DELETE("/attachment/delete/:id", h.DeleteAttachment)
	}
}

func userClaims(ctx *gin.Context) jwt.UserClaims {
	return ctx.MustGet("user").(jwt.UserClaims)
}

func (h *OpsHandler) GetDashboardOverview(ctx *gin.Context) {
	var req model.OpsDashboardOverviewReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.dashboardSvc.Overview(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) GetWorkbenchBriefing(ctx *gin.Context) {
	var req model.OpsWorkbenchBriefingReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OwnerID = u.Uid
		req.IsAdmin = u.Username == "admin"
		if !req.MineOnly && !req.IsAdmin {
			// 非管理员默认只看本人
			req.MineOnly = true
		}
		return h.workbenchSvc.Briefing(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) CreateReminderDraft(ctx *gin.Context) {
	var req model.OpsReminderDraftReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.workbenchSvc.DraftReminder(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) GetManagerWeeklyReport(ctx *gin.Context) {
	u := userClaims(ctx)
	if u.Username != "admin" {
		base.ForbiddenError(ctx, "仅超管可查看管理者周报")
		return
	}
	var req model.OpsManagerWeeklyReportReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.managerRptSvc.WeeklyReport(ctx.Request.Context(), req.Days, req.Refresh)
	})
}

func (h *OpsHandler) CreateCustomer(ctx *gin.Context) {
	var req model.CreateOpsCustomerReq
	u := userClaims(ctx)
	req.OperatorID, req.OperatorName = u.Uid, u.Username
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.customerSvc.Create(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateCustomer(ctx *gin.Context) {
	var req model.UpdateOpsCustomerReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.customerSvc.Update(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteCustomer(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.customerSvc.Delete(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetCustomer(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.customerSvc.Get(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListCustomer(ctx *gin.Context) {
	var req model.ListOpsCustomerReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.customerSvc.List(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) ChangeCustomerStage(ctx *gin.Context) {
	var req model.ChangeOpsCustomerStageReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.customerSvc.ChangeStage(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateCustomerReport(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	var req model.UpdateOpsCustomerReportReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.ID = id
		return nil, h.customerSvc.UpdateReportSettings(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) RotateCustomerReportSecret(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.customerSvc.RotateReportSecret(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetVendorProfile(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.customerSvc.GetVendorProfile(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) UpsertVendorProfile(ctx *gin.Context) {
	var req model.UpsertOpsVendorProfileReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.customerSvc.UpsertVendorProfile(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) CreateFollowup(ctx *gin.Context) {
	var req model.CreateOpsFollowupReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.customerSvc.CreateFollowup(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) ListFollowup(ctx *gin.Context) {
	var req model.ListOpsFollowupReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.customerSvc.ListFollowups(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) StartCustomerLifecycle(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	u := userClaims(ctx)
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.bizSvc.StartCustomerLifecycle(ctx.Request.Context(), id, u.Uid, u.Username)
	})
}

func (h *OpsHandler) ListCustomerLifecycle(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.bizSvc.ListCustomerLifecycle(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) GetLifecycleApproveContext(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.bizSvc.GetLifecycleApproveContext(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) ApproveLifecycleNode(ctx *gin.Context) {
	var req model.OpsLifecycleApproveReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.bizSvc.ApproveLifecycleNode(ctx.Request.Context(), &req, u.Uid, u.Username)
	})
}

func (h *OpsHandler) CreateExhibition(ctx *gin.Context) {
	var req model.CreateOpsExhibitionReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return h.leadSvc.CreateExhibition(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateExhibition(ctx *gin.Context) {
	var req model.UpdateOpsExhibitionReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	u := userClaims(ctx)
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.UpdaterID, req.UpdaterName = u.Uid, u.Username
		return h.leadSvc.UpdateExhibition(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteExhibition(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.leadSvc.DeleteExhibition(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetExhibition(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.leadSvc.GetExhibition(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListExhibition(ctx *gin.Context) {
	var req model.ListOpsExhibitionReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.leadSvc.ListExhibition(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) ConvertExhibition(ctx *gin.Context) {
	var req model.ConvertLeadReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return h.leadSvc.TransferExhibitionToVisit(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) CreateVisit(ctx *gin.Context) {
	var req model.CreateOpsVisitReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.leadSvc.CreateVisit(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateVisit(ctx *gin.Context) {
	var req model.UpdateOpsVisitReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	u := userClaims(ctx)
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.UpdaterID, req.UpdaterName = u.Uid, u.Username
		return nil, h.leadSvc.UpdateVisit(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteVisit(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.leadSvc.DeleteVisit(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetVisit(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.leadSvc.GetVisit(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListVisit(ctx *gin.Context) {
	var req model.ListOpsVisitReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.leadSvc.ListVisit(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) ConvertVisit(ctx *gin.Context) {
	var req model.ConvertLeadReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return h.leadSvc.ConvertVisit(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) CreateTrial(ctx *gin.Context) {
	var req model.CreateOpsTrialReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.bizSvc.CreateTrial(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateTrial(ctx *gin.Context) {
	var req model.UpdateOpsTrialReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.bizSvc.UpdateTrial(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteTrial(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.bizSvc.DeleteTrial(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetTrial(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.bizSvc.GetTrial(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListTrial(ctx *gin.Context) {
	var req model.ListOpsTrialReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.bizSvc.ListTrial(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) SubmitTrial(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	u := userClaims(ctx)
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.bizSvc.SubmitTrial(ctx.Request.Context(), id, u.Uid, u.Username)
	})
}

func (h *OpsHandler) CreateContract(ctx *gin.Context) {
	var req model.CreateOpsContractReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.bizSvc.CreateContract(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateContract(ctx *gin.Context) {
	var req model.UpdateOpsContractReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.bizSvc.UpdateContract(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteContract(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.bizSvc.DeleteContract(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetContract(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.bizSvc.GetContract(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListContract(ctx *gin.Context) {
	var req model.ListOpsContractReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.bizSvc.ListContract(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) CreateActivation(ctx *gin.Context) {
	var req model.CreateOpsActivationReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.bizSvc.CreateActivation(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateActivation(ctx *gin.Context) {
	var req model.UpdateOpsActivationReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.bizSvc.UpdateActivation(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) FeedbackActivation(ctx *gin.Context) {
	var req model.FeedbackOpsActivationReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.bizSvc.FeedbackActivation(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteActivation(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.bizSvc.DeleteActivation(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetActivation(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.bizSvc.GetActivation(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListActivation(ctx *gin.Context) {
	var req model.ListOpsActivationReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.bizSvc.ListActivation(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) SubmitActivation(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	u := userClaims(ctx)
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.bizSvc.SubmitActivation(ctx.Request.Context(), id, u.Uid, u.Username)
	})
}

func (h *OpsHandler) CreateSettlement(ctx *gin.Context) {
	var req model.CreateOpsSettlementReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.financeSvc.CreateSettlement(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateSettlement(ctx *gin.Context) {
	var req model.UpdateOpsSettlementReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.financeSvc.UpdateSettlement(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteSettlement(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.financeSvc.DeleteSettlement(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) GetSettlement(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.financeSvc.GetSettlement(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListSettlement(ctx *gin.Context) {
	var req model.ListOpsSettlementReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.financeSvc.ListSettlement(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) ConfirmSettlement(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.financeSvc.ConfirmSettlement(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) CreateInvoice(ctx *gin.Context) {
	var req model.CreateOpsInvoiceReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.financeSvc.CreateInvoice(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateInvoice(ctx *gin.Context) {
	var req model.UpdateOpsInvoiceReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.financeSvc.UpdateInvoice(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteInvoice(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.financeSvc.DeleteInvoice(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListInvoice(ctx *gin.Context) {
	var req model.ListOpsInvoiceReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.financeSvc.ListInvoice(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) CreatePayment(ctx *gin.Context) {
	var req model.CreateOpsPaymentReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.financeSvc.CreatePayment(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdatePayment(ctx *gin.Context) {
	var req model.UpdateOpsPaymentReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.financeSvc.UpdatePayment(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeletePayment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.financeSvc.DeletePayment(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListPayment(ctx *gin.Context) {
	var req model.ListOpsPaymentReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.financeSvc.ListPayment(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) MatchPayment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.financeSvc.MatchPayment(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) ListReminder(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.reminderSvc.ListRules(ctx.Request.Context())
	})
}
func (h *OpsHandler) CreateReminder(ctx *gin.Context) {
	var req model.CreateOpsReminderRuleReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.reminderSvc.CreateRule(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateReminder(ctx *gin.Context) {
	var req model.UpdateOpsReminderRuleReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.reminderSvc.UpdateRule(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) PreviewReminder(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.reminderSvc.PreviewHits(ctx.Request.Context())
	})
}
func (h *OpsHandler) ScanReminder(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.reminderSvc.RunScan(ctx.Request.Context())
	})
}
func (h *OpsHandler) ListReminderTask(ctx *gin.Context) {
	var req model.ListOpsReminderTaskReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.reminderSvc.ListTasks(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) CreateReminderTask(ctx *gin.Context) {
	var req model.CreateOpsReminderTaskReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.CreatorID, req.CreatorName = u.Uid, u.Username
		return nil, h.reminderSvc.CreateTask(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) UpdateReminderTask(ctx *gin.Context) {
	var req model.UpdateOpsReminderTaskReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.reminderSvc.UpdateTask(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteReminderTask(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.reminderSvc.DeleteTask(ctx.Request.Context(), id)
	})
}
func (h *OpsHandler) ListReminderDelivery(ctx *gin.Context) {
	var req model.ListOpsReminderDeliveryReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.reminderSvc.ListDeliveries(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) ListSurvey(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.surveySvc.ListSurveys(ctx.Request.Context())
	})
}
func (h *OpsHandler) SubmitSurvey(ctx *gin.Context) {
	var req model.SubmitOpsSurveyReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return nil, h.surveySvc.Submit(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) ListSurveyResponse(ctx *gin.Context) {
	var req model.ListOpsSurveyResponseReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.surveySvc.ListResponses(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) CreateSurveyInvite(ctx *gin.Context) {
	var req model.CreateOpsSurveyInviteReq
	u := userClaims(ctx)
	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID, req.OperatorName = u.Uid, u.Username
		return h.surveySvc.CreateInvite(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) ListSurveyInvite(ctx *gin.Context) {
	var req model.ListOpsSurveyInviteReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.surveySvc.ListInvites(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) ListContractItem(ctx *gin.Context) {
	var req model.ListOpsContractItemReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.bizSvc.ListContractItems(ctx.Request.Context(), req.ContractID)
	})
}
func (h *OpsHandler) CreateContractItem(ctx *gin.Context) {
	var req model.CreateOpsContractItemReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.bizSvc.CreateContractItem(ctx.Request.Context(), &req)
	})
}
func (h *OpsHandler) DeleteContractItem(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.bizSvc.DeleteContractItem(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) GenerateMonthlyBilling(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.billingSvc.GenerateMonthlyDrafts(ctx.Request.Context())
	})
}

func (h *OpsHandler) UploadAttachment(ctx *gin.Context) {
	u := userClaims(ctx)
	bizType := ctx.PostForm("biz_type")
	bizID, err := strconv.Atoi(ctx.PostForm("biz_id"))
	if err != nil || bizID <= 0 {
		base.ErrorWithMessage(ctx, "biz_id 无效")
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		base.ErrorWithMessage(ctx, "请上传文件")
		return
	}
	attachment, err := h.attachmentSvc.Upload(ctx.Request.Context(), bizType, bizID, u.Uid, fileHeader)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	base.SuccessWithData(ctx, attachment)
}

func (h *OpsHandler) ListAttachment(ctx *gin.Context) {
	var req model.ListOpsAttachmentReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.attachmentSvc.List(ctx.Request.Context(), req.BizType, req.BizID)
	})
}

func (h *OpsHandler) DownloadAttachment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	attachment, absPath, err := h.attachmentSvc.Download(ctx.Request.Context(), id)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	disposition := fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(attachment.FileName))
	ctx.Header("Content-Disposition", disposition)
	if attachment.ContentType != "" {
		ctx.Header("Content-Type", attachment.ContentType)
	}
	ctx.File(absPath)
}

func (h *OpsHandler) DeleteAttachment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.attachmentSvc.Delete(ctx.Request.Context(), id)
	})
}
