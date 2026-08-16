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
	"fmt"
	"net/url"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type InstanceCommentHandler struct {
	commentService service.InstanceCommentService
}

func NewInstanceCommentHandler(commentService service.InstanceCommentService) *InstanceCommentHandler {
	return &InstanceCommentHandler{
		commentService: commentService,
	}
}

func (h *InstanceCommentHandler) RegisterRouters(server *gin.Engine) {
	commentGroup := server.Group("/api/workorder/instance/comment")
	{
		commentGroup.POST("/create", h.CreateInstanceComment)
		commentGroup.PUT("/update/:id", h.UpdateInstanceComment)
		commentGroup.DELETE("/delete/:id", h.DeleteInstanceComment)
		commentGroup.GET("/detail/:id", h.GetInstanceComment)
		commentGroup.GET("/list", h.ListInstanceComments)
		commentGroup.GET("/tree/:id", h.GetInstanceCommentsTree)
		commentGroup.POST("/attachment/upload", h.UploadCommentAttachment)
		commentGroup.GET("/attachment/:id/download", h.DownloadCommentAttachment)
		commentGroup.DELETE("/attachment/:id", h.DeleteCommentAttachment)
	}
}

func (h *InstanceCommentHandler) CreateInstanceComment(ctx *gin.Context) {
	var req model.CreateWorkorderInstanceCommentReq
	user := ctx.MustGet("user").(jwt.UserClaims)

	req.OperatorID = user.Uid
	req.OperatorName = user.Username

	base.HandleRequest(ctx, &req, func() (any, error) {
		req.OperatorID = user.Uid
		req.OperatorName = user.Username
		if req.Type == "" {
			req.Type = model.CommentTypeNormal
		}
		if req.IsSystem == 0 {
			req.IsSystem = 2
		}
		if req.Status == 0 {
			req.Status = model.CommentStatusNormal
		}
		return nil, h.commentService.CreateInstanceComment(ctx.Request.Context(), &req)
	})
}

func (h *InstanceCommentHandler) UpdateInstanceComment(ctx *gin.Context) {
	var req model.UpdateWorkorderInstanceCommentReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id
	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.commentService.UpdateInstanceComment(ctx.Request.Context(), &req, user.Uid)
	})
}

func (h *InstanceCommentHandler) DeleteInstanceComment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}

	user := ctx.MustGet("user").(jwt.UserClaims)

	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.commentService.DeleteInstanceComment(ctx.Request.Context(), id, user.Uid)
	})
}

func (h *InstanceCommentHandler) GetInstanceComment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}

	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.commentService.GetInstanceComment(ctx.Request.Context(), id)
	})
}

func (h *InstanceCommentHandler) ListInstanceComments(ctx *gin.Context) {
	var req model.ListWorkorderInstanceCommentReq

	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.commentService.ListInstanceComments(ctx.Request.Context(), &req)
	})
}

func (h *InstanceCommentHandler) GetInstanceCommentsTree(ctx *gin.Context) {
	var req model.GetInstanceCommentsTreeReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.commentService.GetInstanceCommentsTree(ctx.Request.Context(), req.ID)
	})
}

func (h *InstanceCommentHandler) UploadCommentAttachment(ctx *gin.Context) {
	user := ctx.MustGet("user").(jwt.UserClaims)
	instanceID, err := strconv.Atoi(ctx.PostForm("instance_id"))
	if err != nil || instanceID <= 0 {
		base.ErrorWithMessage(ctx, "instance_id 无效")
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		base.ErrorWithMessage(ctx, "请上传文件")
		return
	}

	attachment, err := h.commentService.UploadCommentAttachment(ctx.Request.Context(), instanceID, user.Uid, fileHeader)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	base.SuccessWithData(ctx, attachment)
}

func (h *InstanceCommentHandler) DownloadCommentAttachment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}

	attachment, absPath, err := h.commentService.DownloadCommentAttachment(ctx.Request.Context(), id)
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

func (h *InstanceCommentHandler) DeleteCommentAttachment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	user := ctx.MustGet("user").(jwt.UserClaims)
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.commentService.DeleteCommentAttachment(ctx.Request.Context(), id, user.Uid)
	})
}
