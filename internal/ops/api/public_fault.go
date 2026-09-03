package api

import (
	"net/http"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	opsPublic "github.com/GoSimplicity/AI-CloudOps/internal/ops/public"
	opsUtils "github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
)

var publicFaultLimiter = opsUtils.NewIPRateLimiter(15, time.Minute)

func (h *OpsHandler) registerPublicFaultRoutes(server *gin.Engine) {
	server.GET("/public/fault", h.ServePublicFaultPage)
	server.GET("/public/fault/query", h.ServePublicFaultQueryPage)
	pg := server.Group("/api/public/fault")
	{
		pg.POST("/submit", h.SubmitPublicFault)
		pg.POST("/query", h.QueryPublicFault)
		pg.POST("/upload", h.UploadPublicFaultAttachment)
	}
}

func (h *OpsHandler) ServePublicFaultPage(ctx *gin.Context) {
	if len(opsPublic.FaultHTML) == 0 {
		ctx.String(http.StatusInternalServerError, "页面加载失败")
		return
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", opsPublic.FaultHTML)
}

func (h *OpsHandler) ServePublicFaultQueryPage(ctx *gin.Context) {
	if len(opsPublic.FaultQueryHTML) == 0 {
		ctx.String(http.StatusInternalServerError, "页面加载失败")
		return
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", opsPublic.FaultQueryHTML)
}

func (h *OpsHandler) SubmitPublicFault(ctx *gin.Context) {
	if !publicFaultLimiter.Allow(ctx.ClientIP()) {
		base.ErrorWithMessage(ctx, "提交过于频繁，请稍后再试")
		return
	}
	var req model.SubmitPublicFaultReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.publicFaultSvc.Submit(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) QueryPublicFault(ctx *gin.Context) {
	if !publicFaultLimiter.Allow(ctx.ClientIP()) {
		base.ErrorWithMessage(ctx, "查询过于频繁，请稍后再试")
		return
	}
	var req model.QueryPublicFaultReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.publicFaultSvc.Query(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) UploadPublicFaultAttachment(ctx *gin.Context) {
	if !publicFaultLimiter.Allow(ctx.ClientIP()) {
		base.ErrorWithMessage(ctx, "上传过于频繁，请稍后再试")
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		base.ErrorWithMessage(ctx, "请上传文件")
		return
	}
	attachment, err := h.attachmentSvc.UploadPending(ctx.Request.Context(), model.OpsAttachmentBizPublicFault, 0, fileHeader)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	base.SuccessWithData(ctx, attachment)
}
