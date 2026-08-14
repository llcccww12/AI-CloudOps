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
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
)

type DepartmentService interface {
	CreateDepartment(ctx context.Context, req *model.CreateDepartmentRequest) error
	GetDepartmentByID(ctx context.Context, id int) (*model.Department, error)
	UpdateDepartment(ctx context.Context, req *model.UpdateDepartmentRequest) error
	DeleteDepartment(ctx context.Context, id int) error
	ListDepartments(ctx context.Context, req *model.ListDepartmentsRequest) (model.ListResp[*model.Department], error)
	GetDepartmentTree(ctx context.Context) ([]*model.Department, error)
}

type departmentService struct {
	l   *zap.Logger
	dao dao.DepartmentDAO
}

func NewDepartmentService(l *zap.Logger, dao dao.DepartmentDAO) DepartmentService {
	return &departmentService{
		l:   l,
		dao: dao,
	}
}

func (s *departmentService) CreateDepartment(ctx context.Context, req *model.CreateDepartmentRequest) error {
	if req == nil {
		return errors.New("部门不能为空")
	}
	return s.dao.Create(ctx, s.buildCreateDepartment(req))
}

func (s *departmentService) GetDepartmentByID(ctx context.Context, id int) (*model.Department, error) {
	if id <= 0 {
		return nil, errors.New("部门ID无效")
	}
	dept, err := s.dao.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, errors.New("部门不存在")
	}
	return dept, nil
}

func (s *departmentService) UpdateDepartment(ctx context.Context, req *model.UpdateDepartmentRequest) error {
	if req == nil {
		return errors.New("部门不能为空")
	}
	return s.dao.Update(ctx, s.buildUpdateDepartment(req))
}

func (s *departmentService) DeleteDepartment(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("部门ID无效")
	}
	return s.dao.Delete(ctx, id)
}

func (s *departmentService) ListDepartments(ctx context.Context, req *model.ListDepartmentsRequest) (model.ListResp[*model.Department], error) {
	if req.Page < 1 || req.Size < 1 {
		return model.ListResp[*model.Department]{}, errors.New("分页参数无效")
	}

	list, total, err := s.dao.List(ctx, req.Page, req.Size, req.Search, req.Status)
	if err != nil {
		s.l.Error("获取部门列表失败", zap.Error(err))
		return model.ListResp[*model.Department]{}, err
	}
	return model.ListResp[*model.Department]{
		Items: list,
		Total: total,
	}, nil
}

func (s *departmentService) GetDepartmentTree(ctx context.Context) ([]*model.Department, error) {
	list, err := s.dao.ListAll(ctx)
	if err != nil {
		s.l.Error("获取部门树失败", zap.Error(err))
		return nil, err
	}
	return buildDepartmentTree(list), nil
}

func (s *departmentService) buildCreateDepartment(req *model.CreateDepartmentRequest) *model.Department {
	status := req.Status
	if status == 0 {
		status = 1
	}
	return &model.Department{
		Name:        req.Name,
		Code:        req.Code,
		ParentID:    req.ParentID,
		Sort:        req.Sort,
		Status:      status,
		Leader:      req.Leader,
		Phone:       req.Phone,
		Email:       req.Email,
		Description: req.Description,
	}
}

func (s *departmentService) buildUpdateDepartment(req *model.UpdateDepartmentRequest) *model.Department {
	return &model.Department{
		Model: model.Model{
			ID: req.ID,
		},
		Name:        req.Name,
		Code:        req.Code,
		ParentID:    req.ParentID,
		Sort:        req.Sort,
		Status:      req.Status,
		Leader:      req.Leader,
		Phone:       req.Phone,
		Email:       req.Email,
		Description: req.Description,
	}
}

func buildDepartmentTree(nodes []*model.Department) []*model.Department {
	nodeMap := make(map[int]*model.Department, len(nodes))
	roots := make([]*model.Department, 0)

	for _, node := range nodes {
		clone := *node
		clone.Children = make([]*model.Department, 0)
		nodeMap[node.ID] = &clone
	}

	for _, node := range nodes {
		current := nodeMap[node.ID]
		if node.ParentID == 0 || nodeMap[node.ParentID] == nil {
			roots = append(roots, current)
			continue
		}
		parent := nodeMap[node.ParentID]
		parent.Children = append(parent.Children, current)
	}
	return roots
}
