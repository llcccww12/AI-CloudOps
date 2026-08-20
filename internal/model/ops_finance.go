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
	OpsSettlementStatusDraft     = "draft"
	OpsSettlementStatusConfirmed = "confirmed"
	OpsSettlementStatusInvoiced  = "invoiced"
	OpsSettlementStatusPaid      = "paid"
	OpsSettlementStatusOverdue   = "overdue"

	OpsInvoiceStatusDraft   = "draft"
	OpsInvoiceStatusIssued  = "issued"
	OpsInvoiceStatusVoid    = "void"

	OpsPaymentStatusPending = "pending"
	OpsPaymentStatusMatched = "matched"
)

// OpsSettlement 结算单
type OpsSettlement struct {
	Model
	CustomerID   int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	ContractID   int        `json:"contract_id" gorm:"column:contract_id;not null;index;comment:合同ID"`
	Title        string     `json:"title" gorm:"column:title;type:varchar(200);not null;comment:标题"`
	PeriodStart  *time.Time `json:"period_start" gorm:"column:period_start;comment:周期开始"`
	PeriodEnd    *time.Time `json:"period_end" gorm:"column:period_end;index;comment:周期结束"`
	Amount       float64    `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:应结金额"`
	DueAt        *time.Time `json:"due_at" gorm:"column:due_at;index;comment:收款到期日"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:draft;index;comment:状态"`
	Remark       string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
	OperatorID   int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
}

func (OpsSettlement) TableName() string { return "cl_ops_settlement" }

type CreateOpsSettlementReq struct {
	CustomerID   int     `json:"customer_id" binding:"required,min=1"`
	ContractID   int     `json:"contract_id" binding:"required,min=1"`
	Title        string  `json:"title" binding:"required,min=1,max=200"`
	PeriodStart  *string `json:"period_start"`
	PeriodEnd    *string `json:"period_end"`
	Amount       float64 `json:"amount"`
	DueAt        *string `json:"due_at"`
	Remark       string  `json:"remark"`
	OperatorID   int     `json:"operator_id"`
	OperatorName string  `json:"operator_name"`
}

type UpdateOpsSettlementReq struct {
	ID          int     `json:"id" binding:"required,min=1"`
	Title       string  `json:"title" binding:"required,min=1,max=200"`
	PeriodStart *string `json:"period_start"`
	PeriodEnd   *string `json:"period_end"`
	Amount      float64 `json:"amount"`
	DueAt       *string `json:"due_at"`
	Status      string  `json:"status" binding:"omitempty,oneof=draft confirmed invoiced paid overdue"`
	Remark      string  `json:"remark"`
}

type ListOpsSettlementReq struct {
	ListReq
	CustomerID int    `json:"customer_id" form:"customer_id"`
	Status     string `json:"status" form:"status"`
}

// OpsInvoice 发票
type OpsInvoice struct {
	Model
	CustomerID   int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	SettlementID int        `json:"settlement_id" gorm:"column:settlement_id;not null;index;comment:结算单ID"`
	InvoiceNo    string     `json:"invoice_no" gorm:"column:invoice_no;type:varchar(100);comment:发票号"`
	InvoiceType  string     `json:"invoice_type" gorm:"column:invoice_type;type:varchar(64);comment:开票类型"`
	Amount       float64    `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:金额"`
	IssuedAt     *time.Time `json:"issued_at" gorm:"column:issued_at;comment:开票日期"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:draft;index;comment:状态"`
	OperatorID   int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
}

func (OpsInvoice) TableName() string { return "cl_ops_invoice" }

type CreateOpsInvoiceReq struct {
	CustomerID   int     `json:"customer_id" binding:"required,min=1"`
	SettlementID int     `json:"settlement_id" binding:"required,min=1"`
	InvoiceNo    string  `json:"invoice_no"`
	InvoiceType  string  `json:"invoice_type"`
	Amount       float64 `json:"amount"`
	IssuedAt     *string `json:"issued_at"`
	OperatorID   int     `json:"operator_id"`
	OperatorName string  `json:"operator_name"`
}

type UpdateOpsInvoiceReq struct {
	ID          int     `json:"id" binding:"required,min=1"`
	InvoiceNo   string  `json:"invoice_no"`
	InvoiceType string  `json:"invoice_type"`
	Amount      float64 `json:"amount"`
	IssuedAt    *string `json:"issued_at"`
	Status      string  `json:"status" binding:"omitempty,oneof=draft issued void"`
}

type ListOpsInvoiceReq struct {
	ListReq
	CustomerID   int `json:"customer_id" form:"customer_id"`
	SettlementID int `json:"settlement_id" form:"settlement_id"`
}

// OpsPayment 回款
type OpsPayment struct {
	Model
	CustomerID   int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	SettlementID int        `json:"settlement_id" gorm:"column:settlement_id;not null;index;comment:结算单ID"`
	Amount       float64    `json:"amount" gorm:"column:amount;type:decimal(14,2);not null;default:0;comment:回款金额"`
	PaidAt       *time.Time `json:"paid_at" gorm:"column:paid_at;index;comment:到账日期"`
	BankRef      string     `json:"bank_ref" gorm:"column:bank_ref;type:varchar(100);comment:流水号"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:pending;index;comment:状态"`
	OperatorID   int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人ID"`
	OperatorName string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
	Remark       string     `json:"remark" gorm:"column:remark;type:text;comment:备注"`
}

func (OpsPayment) TableName() string { return "cl_ops_payment" }

type CreateOpsPaymentReq struct {
	CustomerID   int     `json:"customer_id" binding:"required,min=1"`
	SettlementID int     `json:"settlement_id" binding:"required,min=1"`
	Amount       float64 `json:"amount" binding:"required"`
	PaidAt       *string `json:"paid_at"`
	BankRef      string  `json:"bank_ref"`
	Remark       string  `json:"remark"`
	OperatorID   int     `json:"operator_id"`
	OperatorName string  `json:"operator_name"`
}

type UpdateOpsPaymentReq struct {
	ID      int     `json:"id" binding:"required,min=1"`
	Amount  float64 `json:"amount"`
	PaidAt  *string `json:"paid_at"`
	BankRef string  `json:"bank_ref"`
	Status  string  `json:"status" binding:"omitempty,oneof=pending matched"`
	Remark  string  `json:"remark"`
}

type ListOpsPaymentReq struct {
	ListReq
	CustomerID   int `json:"customer_id" form:"customer_id"`
	SettlementID int `json:"settlement_id" form:"settlement_id"`
}
