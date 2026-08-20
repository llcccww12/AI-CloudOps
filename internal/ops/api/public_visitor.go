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

var publicVisitorLimiter = opsUtils.NewIPRateLimiter(10, time.Minute)

func (h *OpsHandler) registerPublicVisitorRoutes(server *gin.Engine) {
	server.GET("/public/visitor", h.ServePublicVisitorPage)
	pg := server.Group("/api/public/visitor")
	{
		pg.POST("/submit", h.SubmitPublicVisitor)
		pg.POST("/upload", h.UploadPublicVisitorAttachment)
	}
}

func (h *OpsHandler) ServePublicVisitorPage(ctx *gin.Context) {
	if len(opsPublic.VisitorHTML) == 0 {
		ctx.String(http.StatusInternalServerError, "页面加载失败")
		return
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", opsPublic.VisitorHTML)
}

func (h *OpsHandler) SubmitPublicVisitor(ctx *gin.Context) {
	if !publicVisitorLimiter.Allow(ctx.ClientIP()) {
		base.ErrorWithMessage(ctx, "提交过于频繁，请稍后再试")
		return
	}
	var req model.SubmitPublicVisitorReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.leadSvc.SubmitPublicVisitor(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) UploadPublicVisitorAttachment(ctx *gin.Context) {
	if !publicVisitorLimiter.Allow(ctx.ClientIP()) {
		base.ErrorWithMessage(ctx, "上传过于频繁，请稍后再试")
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		base.ErrorWithMessage(ctx, "请上传文件")
		return
	}
	attachment, err := h.attachmentSvc.UploadPending(ctx.Request.Context(), model.OpsAttachmentBizExhibition, 0, fileHeader)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	base.SuccessWithData(ctx, attachment)
}
