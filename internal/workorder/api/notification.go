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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	service service.WorkorderNotificationService
}

func NewNotificationHandler(service service.WorkorderNotificationService) *NotificationHandler {
	return &NotificationHandler{
		service: service,
	}
}

func (h *NotificationHandler) RegisterRouters(server *gin.Engine) {
	notificationGroup := server.Group("/api/workorder/notification")
	{
		notificationGroup.POST("/create", h.CreateNotification)
		notificationGroup.PUT("/update/:id", h.UpdateNotification)
		notificationGroup.DELETE("/delete/:id", h.DeleteNotification)
		notificationGroup.GET("/list", h.ListNotification)
		notificationGroup.GET("/detail/:id", h.DetailNotification)
		notificationGroup.GET("/logs", h.GetSendLogs)
		notificationGroup.POST("/test/send", h.TestSendNotification)
		notificationGroup.GET("/channels", h.GetAvailableChannels)
		notificationGroup.POST("/send", h.SendNotificationManually)
		notificationGroup.GET("/inbox/list", h.ListInbox)
		notificationGroup.GET("/inbox/unread_count", h.CountUnreadInbox)
		notificationGroup.POST("/inbox/read/:id", h.MarkInboxRead)
		notificationGroup.POST("/inbox/read_all", h.MarkAllInboxRead)
		notificationGroup.DELETE("/inbox/clear", h.ClearInbox)
		notificationGroup.POST("/:id/duplicate", h.DuplicateNotification)
	}
}

func (h *NotificationHandler) CreateNotification(ctx *gin.Context) {
	var req model.CreateWorkorderNotificationReq

	user := ctx.MustGet("user").(jwt.UserClaims)
	req.UserID = user.Uid

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateNotification(ctx.Request.Context(), &req)
	})

}

func (h *NotificationHandler) UpdateNotification(ctx *gin.Context) {
	var req model.UpdateWorkorderNotificationReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) DeleteNotification(ctx *gin.Context) {
	var req model.DeleteWorkorderNotificationReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) ListNotification(ctx *gin.Context) {
	var req model.ListWorkorderNotificationReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) DetailNotification(ctx *gin.Context) {
	var req model.DetailWorkorderNotificationReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) GetSendLogs(ctx *gin.Context) {
	var req model.ListWorkorderNotificationLogReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetSendLogs(ctx.Request.Context(), &req)
	})
}

// TestSendNotification 测试发送通知
func (h *NotificationHandler) TestSendNotification(ctx *gin.Context) {
	var req model.TestSendWorkorderNotificationReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.TestSendNotification(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) GetAvailableChannels(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetAvailableChannels(), nil
	})
}

// SendNotificationManually 手动发送通知
func (h *NotificationHandler) SendNotificationManually(ctx *gin.Context) {
	var req model.ManualSendNotificationReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.SendNotificationByChannels(ctx.Request.Context(), req.Channels, req.Recipient, req.Subject, req.Content)
	})
}

func (h *NotificationHandler) DuplicateNotification(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	user := ctx.MustGet("user").(jwt.UserClaims)
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DuplicateNotification(ctx.Request.Context(), id, user.Uid)
	})
}

func (h *NotificationHandler) ListInbox(ctx *gin.Context) {
	var req model.ListWorkorderInboxReq
	user := ctx.MustGet("user").(jwt.UserClaims)
	req.UserID = user.Uid
	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListInbox(ctx.Request.Context(), &req)
	})
}

func (h *NotificationHandler) CountUnreadInbox(ctx *gin.Context) {
	user := ctx.MustGet("user").(jwt.UserClaims)
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.CountUnreadInbox(ctx.Request.Context(), user.Uid)
	})
}

func (h *NotificationHandler) MarkInboxRead(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	user := ctx.MustGet("user").(jwt.UserClaims)
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.MarkInboxRead(ctx.Request.Context(), id, user.Uid)
	})
}

func (h *NotificationHandler) MarkAllInboxRead(ctx *gin.Context) {
	user := ctx.MustGet("user").(jwt.UserClaims)
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.MarkAllInboxRead(ctx.Request.Context(), user.Uid)
	})
}

func (h *NotificationHandler) ClearInbox(ctx *gin.Context) {
	user := ctx.MustGet("user").(jwt.UserClaims)
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.ClearInbox(ctx.Request.Context(), user.Uid)
	})
}
