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

package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DepartmentDAO interface {
	Create(ctx context.Context, dept *model.Department) error
	GetByID(ctx context.Context, id int) (*model.Department, error)
	Update(ctx context.Context, dept *model.Department) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, page, size int, search string, status int8) ([]*model.Department, int64, error)
	ListAll(ctx context.Context) ([]*model.Department, error)
	CountChildren(ctx context.Context, parentID int) (int64, error)
	CountUsers(ctx context.Context, departmentID int) (int64, error)
}

type departmentDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewDepartmentDAO(db *gorm.DB, l *zap.Logger) DepartmentDAO {
	return &departmentDAO{
		db: db,
		l:  l,
	}
}

func (d *departmentDAO) Create(ctx context.Context, dept *model.Department) error {
	if dept == nil {
		return errors.New("部门对象不能为空")
	}
	if dept.Name == "" {
		return errors.New("部门名称不能为空")
	}
	if dept.Code == "" {
		return errors.New("部门编码不能为空")
	}
	if dept.Status == 0 {
		dept.Status = 1
	}

	if dept.ParentID > 0 {
		parent, err := d.GetByID(ctx, dept.ParentID)
		if err != nil {
			return err
		}
		if parent == nil {
			return errors.New("父部门不存在")
		}
	}

	var count int64
	if err := d.db.WithContext(ctx).Model(&model.Department{}).
		Where("code = ?", dept.Code).Count(&count).Error; err != nil {
		return fmt.Errorf("检查部门编码失败: %w", err)
	}
	if count > 0 {
		return errors.New("部门编码已存在")
	}

	return d.db.WithContext(ctx).Create(dept).Error
}

func (d *departmentDAO) GetByID(ctx context.Context, id int) (*model.Department, error) {
	if id <= 0 {
		return nil, errors.New("无效的部门ID")
	}

	var dept model.Department
	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询部门失败: %w", err)
	}
	return &dept, nil
}

func (d *departmentDAO) Update(ctx context.Context, dept *model.Department) error {
	if dept == nil {
		return errors.New("部门对象不能为空")
	}
	if dept.ID <= 0 {
		return errors.New("无效的部门ID")
	}
	if dept.Name == "" {
		return errors.New("部门名称不能为空")
	}
	if dept.Code == "" {
		return errors.New("部门编码不能为空")
	}
	if dept.ParentID == dept.ID {
		return errors.New("父部门不能是自身")
	}

	old, err := d.GetByID(ctx, dept.ID)
	if err != nil {
		return err
	}
	if old == nil {
		return errors.New("部门不存在")
	}

	if dept.ParentID > 0 {
		parent, err := d.GetByID(ctx, dept.ParentID)
		if err != nil {
			return err
		}
		if parent == nil {
			return errors.New("父部门不存在")
		}
	}

	if old.Code != dept.Code {
		var count int64
		if err := d.db.WithContext(ctx).Model(&model.Department{}).
			Where("code = ? AND id != ?", dept.Code, dept.ID).Count(&count).Error; err != nil {
			return fmt.Errorf("检查部门编码失败: %w", err)
		}
		if count > 0 {
			return errors.New("部门编码已被其他记录使用")
		}
	}

	if dept.Status == 0 {
		dept.Status = old.Status
	}

	updates := map[string]interface{}{
		"name":        dept.Name,
		"code":        dept.Code,
		"parent_id":   dept.ParentID,
		"sort":        dept.Sort,
		"status":      dept.Status,
		"leader":      dept.Leader,
		"phone":       dept.Phone,
		"email":       dept.Email,
		"description": dept.Description,
	}

	if err := d.db.WithContext(ctx).Model(&model.Department{}).
		Where("id = ?", dept.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新部门失败: %w", err)
	}
	return nil
}

func (d *departmentDAO) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("无效的部门ID")
	}

	dept, err := d.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if dept == nil {
		return errors.New("部门不存在")
	}

	childCount, err := d.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("该部门存在子部门，无法删除")
	}

	userCount, err := d.CountUsers(ctx, id)
	if err != nil {
		return err
	}
	if userCount > 0 {
		return errors.New("该部门下仍有用户，无法删除")
	}

	result := d.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Department{})
	if result.Error != nil {
		return fmt.Errorf("删除部门失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("部门不存在或已被删除")
	}
	return nil
}

func (d *departmentDAO) List(ctx context.Context, page, size int, search string, status int8) ([]*model.Department, int64, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	query := d.db.WithContext(ctx).Model(&model.Department{})
	if search != "" {
		query = query.Where("name LIKE ? OR code LIKE ? OR leader LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if status != 0 {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取部门总数失败: %w", err)
	}

	var list []*model.Department
	offset := (page - 1) * size
	if err := query.Offset(offset).Limit(size).Order("sort ASC, id ASC").Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("获取部门列表失败: %w", err)
	}
	return list, total, nil
}

func (d *departmentDAO) ListAll(ctx context.Context) ([]*model.Department, error) {
	var list []*model.Department
	if err := d.db.WithContext(ctx).Model(&model.Department{}).
		Order("sort ASC, id ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("获取全部部门失败: %w", err)
	}
	return list, nil
}

func (d *departmentDAO) CountChildren(ctx context.Context, parentID int) (int64, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&model.Department{}).
		Where("parent_id = ?", parentID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计子部门失败: %w", err)
	}
	return count, nil
}

func (d *departmentDAO) CountUsers(ctx context.Context, departmentID int) (int64, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&model.User{}).
		Where("department_id = ?", departmentID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计部门用户失败: %w", err)
	}
	return count, nil
}
