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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	svc service.DepartmentService
}

func NewDepartmentHandler(svc service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{
		svc: svc,
	}
}

func (h *DepartmentHandler) RegisterRouters(server *gin.Engine) {
	deptGroup := server.Group("/api/department")
	{
		deptGroup.GET("/list", h.ListDepartments)
		deptGroup.GET("/tree", h.GetDepartmentTree)
		deptGroup.GET("/detail/:id", h.GetDepartmentDetail)
		deptGroup.POST("/create", h.CreateDepartment)
		deptGroup.PUT("/update/:id", h.UpdateDepartment)
		deptGroup.DELETE("/delete/:id", h.DeleteDepartment)
	}
}

func (h *DepartmentHandler) ListDepartments(ctx *gin.Context) {
	var req model.ListDepartmentsRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.ListDepartments(ctx.Request.Context(), &req)
	})
}

func (h *DepartmentHandler) GetDepartmentTree(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetDepartmentTree(ctx.Request.Context())
	})
}

func (h *DepartmentHandler) GetDepartmentDetail(ctx *gin.Context) {
	var req model.GetDepartmentRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetDepartmentByID(ctx.Request.Context(), id)
	})
}

func (h *DepartmentHandler) CreateDepartment(ctx *gin.Context) {
	var req model.CreateDepartmentRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.CreateDepartment(ctx.Request.Context(), &req)
	})
}

func (h *DepartmentHandler) UpdateDepartment(ctx *gin.Context) {
	var req model.UpdateDepartmentRequest

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID = id
		return nil, h.svc.UpdateDepartment(ctx.Request.Context(), &req)
	})
}

func (h *DepartmentHandler) DeleteDepartment(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.svc.DeleteDepartment(ctx.Request.Context(), id)
	})
}
