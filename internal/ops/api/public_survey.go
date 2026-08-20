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

var publicSurveyLimiter = opsUtils.NewIPRateLimiter(20, time.Minute)

func (h *OpsHandler) registerPublicSurveyRoutes(server *gin.Engine) {
	server.GET("/public/survey", h.ServePublicSurveyPage)
	pg := server.Group("/api/public/survey")
	{
		pg.GET("/meta", h.GetPublicSurveyMeta)
		pg.POST("/submit", h.SubmitPublicSurvey)
	}
}

func (h *OpsHandler) ServePublicSurveyPage(ctx *gin.Context) {
	if len(opsPublic.SurveyHTML) == 0 {
		ctx.String(http.StatusInternalServerError, "页面加载失败")
		return
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", opsPublic.SurveyHTML)
}

func (h *OpsHandler) GetPublicSurveyMeta(ctx *gin.Context) {
	if !publicSurveyLimiter.Allow(ctx.ClientIP()) {
		base.ErrorWithMessage(ctx, "请求过于频繁，请稍后再试")
		return
	}
	token := ctx.Query("token")
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.surveySvc.GetPublicMeta(ctx.Request.Context(), token)
	})
}

func (h *OpsHandler) SubmitPublicSurvey(ctx *gin.Context) {
	if !publicSurveyLimiter.Allow(ctx.ClientIP()) {
		base.ErrorWithMessage(ctx, "提交过于频繁，请稍后再试")
		return
	}
	var req model.SubmitPublicSurveyReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.surveySvc.SubmitPublic(ctx.Request.Context(), &req)
	})
}
