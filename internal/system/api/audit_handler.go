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

package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuditHandler struct {
	svc    service.AuditService
	logger *zap.Logger
}

func NewAuditHandler(svc service.AuditService, logger *zap.Logger) *AuditHandler {
	return &AuditHandler{
		svc:    svc,
		logger: logger,
	}
}

func (h *AuditHandler) RegisterRouters(server *gin.Engine) {
	auditGroup := server.Group("/api/audit")

	auditGroup.GET("/list", h.ListAuditLogs)
	auditGroup.GET("/detail/:id", h.GetAuditLogDetail)
	auditGroup.GET("/search", h.SearchAuditLogs)

	auditGroup.GET("/statistics", h.GetAuditStatistics)
	auditGroup.GET("/types", h.GetAuditTypes)
	auditGroup.GET("/export", h.ExportAuditLogs)

	// 管理接口 - 需要管理员权限
	auditGroup.DELETE("/:id", h.DeleteAuditLog)
	auditGroup.POST("/batch-delete", h.BatchDeleteLogs)
	auditGroup.POST("/archive", h.ArchiveAuditLogs)

	// 创建接口 - 通常由系统内部调用
	auditGroup.POST("/create", h.CreateAuditLog)
	auditGroup.POST("/batch-create", h.BatchCreateAuditLogs)
}

func (h *AuditHandler) CreateAuditLog(ctx *gin.Context) {
	var req model.CreateAuditLogRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.CreateAuditLog(ctx.Request.Context(), &req)
	})
}

// BatchCreateAuditLogs 批量创建审计日志 - 高性能批处理
func (h *AuditHandler) BatchCreateAuditLogs(ctx *gin.Context) {
	var req model.AuditLogBatch

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.BatchCreateAuditLogs(ctx.Request.Context(), req.Logs)
	})
}

func (h *AuditHandler) ListAuditLogs(ctx *gin.Context) {
	var req model.ListAuditLogsRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.ListAuditLogs(ctx.Request.Context(), &req)
	})
}

func (h *AuditHandler) GetAuditLogDetail(ctx *gin.Context) {
	var req model.GetAuditLogDetailRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的审计日志ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetAuditLogDetail(ctx.Request.Context(), req.ID)
	})
}

// SearchAuditLogs 搜索审计日志
func (h *AuditHandler) SearchAuditLogs(ctx *gin.Context) {
	var req model.SearchAuditLogsRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.SearchAuditLogs(ctx.Request.Context(), &req)
	})
}

func (h *AuditHandler) GetAuditStatistics(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetAuditStatistics(ctx.Request.Context())
	})
}

func (h *AuditHandler) GetAuditTypes(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetAuditTypes(ctx.Request.Context())
	})
}

func (h *AuditHandler) DeleteAuditLog(ctx *gin.Context) {
	var req model.DeleteAuditLogRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的审计日志ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.DeleteAuditLog(ctx.Request.Context(), req.ID)
	})
}

func (h *AuditHandler) BatchDeleteLogs(ctx *gin.Context) {
	var req model.BatchDeleteRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.BatchDeleteAuditLogs(ctx.Request.Context(), req.IDs)
	})
}

// ArchiveAuditLogs 归档审计日志
func (h *AuditHandler) ArchiveAuditLogs(ctx *gin.Context) {
	var req model.ArchiveAuditLogsRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.ArchiveAuditLogs(ctx.Request.Context(), &req)
	})
}

// ExportAuditLogs 导出审计日志为 CSV
func (h *AuditHandler) ExportAuditLogs(ctx *gin.Context) {
	var req model.ExportAuditLogsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	logs, err := h.svc.ExportAuditLogs(ctx.Request.Context(), &req)
	if err != nil {
		h.logger.Error("导出审计日志失败", zap.Error(err))
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102_150405"))
	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	ctx.Writer.WriteHeader(http.StatusOK)
	// UTF-8 BOM，便于 Excel 正确识别中文
	_, _ = ctx.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(ctx.Writer)
	defer writer.Flush()

	_ = writer.Write([]string{
		"ID", "用户ID", "TraceID", "IP", "方法", "接口", "操作类型", "目标类型",
		"目标ID", "状态码", "耗时(ms)", "错误信息", "创建时间",
	})

	for _, log := range logs {
		createdAt := ""
		if !log.CreatedAt.IsZero() {
			createdAt = log.CreatedAt.Format("2006-01-02 15:04:05")
		}
		_ = writer.Write([]string{
			strconv.Itoa(log.ID),
			strconv.Itoa(log.UserID),
			log.TraceID,
			log.IPAddress,
			log.HttpMethod,
			log.Endpoint,
			log.OperationType,
			log.TargetType,
			log.TargetID,
			strconv.Itoa(log.StatusCode),
			strconv.FormatInt(log.Duration, 10),
			log.ErrorMsg,
			createdAt,
		})
	}
}
