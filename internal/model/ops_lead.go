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

import "time"

const (
	OpsExhibitionStatusDraft    = "draft"
	OpsExhibitionStatusReviewed = "reviewed"
	OpsExhibitionStatusConverted = "converted"

	OpsVisitStatusPlanned   = "planned"
	OpsVisitStatusDone      = "done"
	OpsVisitStatusConverted = "converted"
)

// OpsExhibition 展厅接待
type OpsExhibition struct {
	Model
	CompanyName   string     `json:"company_name" gorm:"column:company_name;type:varchar(200);not null;index;comment:来访单位"`
	VisitorName   string     `json:"visitor_name" gorm:"column:visitor_name;type:varchar(100);comment:来访人"`
	VisitorTitle  string     `json:"visitor_title" gorm:"column:visitor_title;type:varchar(100);comment:职务"`
	VisitAt       *time.Time `json:"visit_at" gorm:"column:visit_at;index;comment:来访时间"`
	HostName      string     `json:"host_name" gorm:"column:host_name;type:varchar(100);comment:接待人"`
	Purpose       string     `json:"purpose" gorm:"column:purpose;type:varchar(100);comment:来访目的"`
	FocusTags     StringList `json:"focus_tags" gorm:"column:focus_tags;type:text;serializer:json;comment:关注重点"`
	Content       string     `json:"content" gorm:"column:content;type:text;comment:讲解内容"`
	Companions    string     `json:"companions" gorm:"column:companions;type:varchar(255);comment:陪同人员"`
	Status        string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:draft;index;comment:状态"`
	CustomerID    *int       `json:"customer_id" gorm:"column:customer_id;index;comment:关联客户"`
	OperatorID    int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName  string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
	Remark        string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (OpsExhibition) TableName() string { return "cl_ops_exhibition" }

type CreateOpsExhibitionReq struct {
	CompanyName  string     `json:"company_name" binding:"required,min=1,max=200"`
	VisitorName  string     `json:"visitor_name"`
	VisitorTitle string     `json:"visitor_title"`
	VisitAt      *time.Time `json:"visit_at"`
	HostName     string     `json:"host_name"`
	Purpose      string     `json:"purpose"`
	FocusTags    StringList `json:"focus_tags"`
	Content      string     `json:"content"`
	Companions   string     `json:"companions"`
	Remark       string     `json:"remark"`
	CustomerID   *int       `json:"customer_id"`
	OperatorID   int        `json:"operator_id"`
	OperatorName string     `json:"operator_name"`
}

type UpdateOpsExhibitionReq struct {
	ID           int        `json:"id" binding:"required,min=1"`
	CompanyName  string     `json:"company_name" binding:"required,min=1,max=200"`
	VisitorName  string     `json:"visitor_name"`
	VisitorTitle string     `json:"visitor_title"`
	VisitAt      *time.Time `json:"visit_at"`
	HostName     string     `json:"host_name"`
	Purpose      string     `json:"purpose"`
	FocusTags    StringList `json:"focus_tags"`
	Content      string     `json:"content"`
	Companions   string     `json:"companions"`
	Remark       string     `json:"remark"`
	CustomerID   *int       `json:"customer_id"`
}

type ListOpsExhibitionReq struct {
	ListReq
	Status string `json:"status" form:"status"`
}

// OpsVisit 外访交流
type OpsVisit struct {
	Model
	Title        string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:外访主题"`
	TargetOrg    string     `json:"target_org" gorm:"column:target_org;type:varchar(200);index;comment:对象单位"`
	StartAt      *time.Time `json:"start_at" gorm:"column:start_at;comment:开始时间"`
	EndAt        *time.Time `json:"end_at" gorm:"column:end_at;comment:结束时间"`
	Location     string     `json:"location" gorm:"column:location;type:varchar(200);comment:地点"`
	Participants string     `json:"participants" gorm:"column:participants;type:varchar(255);comment:参与人"`
	Summary      string     `json:"summary" gorm:"column:summary;type:text;comment:交流要点"`
	Outcome      string     `json:"outcome" gorm:"column:outcome;type:text;comment:成果与线索"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:planned;index;comment:状态"`
	CustomerID   *int       `json:"customer_id" gorm:"column:customer_id;index;comment:关联客户"`
	OperatorID   int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
	Remark       string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (OpsVisit) TableName() string { return "cl_ops_visit" }

type CreateOpsVisitReq struct {
	Title        string     `json:"title" binding:"required,min=1,max=200"`
	TargetOrg    string     `json:"target_org"`
	StartAt      *time.Time `json:"start_at"`
	EndAt        *time.Time `json:"end_at"`
	Location     string     `json:"location"`
	Participants string     `json:"participants"`
	Summary      string     `json:"summary"`
	Outcome      string     `json:"outcome"`
	Remark       string     `json:"remark"`
	CustomerID   *int       `json:"customer_id"`
	OperatorID   int        `json:"operator_id"`
	OperatorName string     `json:"operator_name"`
}

type UpdateOpsVisitReq struct {
	ID           int        `json:"id" binding:"required,min=1"`
	Title        string     `json:"title" binding:"required,min=1,max=200"`
	TargetOrg    string     `json:"target_org"`
	StartAt      *time.Time `json:"start_at"`
	EndAt        *time.Time `json:"end_at"`
	Location     string     `json:"location"`
	Participants string     `json:"participants"`
	Summary      string     `json:"summary"`
	Outcome      string     `json:"outcome"`
	Status       string     `json:"status" binding:"omitempty,oneof=planned done converted"`
	Remark       string     `json:"remark"`
	CustomerID   *int       `json:"customer_id"`
}

type ListOpsVisitReq struct {
	ListReq
	Status string `json:"status" form:"status"`
}

type ConvertLeadReq struct {
	ID           int        `json:"id" binding:"required,min=1"`
	DemandTypes  StringList `json:"demand_types"`
	ContactName  string     `json:"contact_name"`
	ContactPhone string     `json:"contact_phone"`
	OwnerID      int        `json:"owner_id"`
	OwnerName    string     `json:"owner_name"`
	OperatorID   int        `json:"operator_id"`
	OperatorName string     `json:"operator_name"`
}
