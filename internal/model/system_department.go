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

package model

// Department 部门模型
type Department struct {
	Model
	Name        string        `json:"name" gorm:"type:varchar(100);not null;comment:部门名称"`
	Code        string        `json:"code" gorm:"type:varchar(64);uniqueIndex:idx_dept_code_del;not null;comment:部门编码"`
	ParentID    int           `json:"parent_id" gorm:"index;default:0;comment:父部门ID,0为根部门"`
	Sort        int           `json:"sort" gorm:"default:0;comment:排序"`
	Status      int8          `json:"status" gorm:"type:tinyint(1);default:1;comment:状态 1启用 2禁用" binding:"omitempty,oneof=1 2"`
	Leader      string        `json:"leader" gorm:"type:varchar(100);comment:负责人"`
	Phone       string        `json:"phone" gorm:"type:varchar(20);comment:联系电话"`
	Email       string        `json:"email" gorm:"type:varchar(100);comment:邮箱"`
	Description string        `json:"description" gorm:"type:varchar(500);comment:描述"`
	Children    []*Department `json:"children,omitempty" gorm:"-"`
}

func (d *Department) TableName() string {
	return "cl_system_departments"
}

type CreateDepartmentRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Code        string `json:"code" binding:"required,min=1,max=64"`
	ParentID    int    `json:"parent_id"`
	Sort        int    `json:"sort"`
	Status      int8   `json:"status" binding:"omitempty,oneof=1 2"`
	Leader      string `json:"leader" binding:"omitempty,max=100"`
	Phone       string `json:"phone" binding:"omitempty,max=20"`
	Email       string `json:"email" binding:"omitempty,email,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
}

type UpdateDepartmentRequest struct {
	ID          int    `json:"id" binding:"required,gt=0"`
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Code        string `json:"code" binding:"required,min=1,max=64"`
	ParentID    int    `json:"parent_id"`
	Sort        int    `json:"sort"`
	Status      int8   `json:"status" binding:"omitempty,oneof=1 2"`
	Leader      string `json:"leader" binding:"omitempty,max=100"`
	Phone       string `json:"phone" binding:"omitempty,max=20"`
	Email       string `json:"email" binding:"omitempty,email,max=100"`
	Description string `json:"description" binding:"omitempty,max=500"`
}

type DeleteDepartmentRequest struct {
	ID int `json:"id" binding:"required,gt=0"`
}

type GetDepartmentRequest struct {
	ID int `json:"id" binding:"required,gt=0"`
}

type ListDepartmentsRequest struct {
	ListReq
	Status int8 `json:"status" form:"status" binding:"omitempty,oneof=1 2"`
}
