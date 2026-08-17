package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	opsUtils "github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
	"go.uber.org/zap"
)

type OpsFinanceService interface {
	CreateSettlement(ctx context.Context, req *model.CreateOpsSettlementReq) error
	UpdateSettlement(ctx context.Context, req *model.UpdateOpsSettlementReq) error
	DeleteSettlement(ctx context.Context, id int) error
	GetSettlement(ctx context.Context, id int) (*model.OpsSettlement, error)
	ListSettlement(ctx context.Context, req *model.ListOpsSettlementReq) (*model.ListResp[*model.OpsSettlement], error)
	ConfirmSettlement(ctx context.Context, id int) error

	CreateInvoice(ctx context.Context, req *model.CreateOpsInvoiceReq) error
	UpdateInvoice(ctx context.Context, req *model.UpdateOpsInvoiceReq) error
	DeleteInvoice(ctx context.Context, id int) error
	ListInvoice(ctx context.Context, req *model.ListOpsInvoiceReq) (*model.ListResp[*model.OpsInvoice], error)

	CreatePayment(ctx context.Context, req *model.CreateOpsPaymentReq) error
	UpdatePayment(ctx context.Context, req *model.UpdateOpsPaymentReq) error
	DeletePayment(ctx context.Context, id int) error
	ListPayment(ctx context.Context, req *model.ListOpsPaymentReq) (*model.ListResp[*model.OpsPayment], error)
	MatchPayment(ctx context.Context, id int) error
}

type opsFinanceService struct {
	settlementDAO dao.OpsSettlementDAO
	invoiceDAO    dao.OpsInvoiceDAO
	paymentDAO    dao.OpsPaymentDAO
	customerDAO   dao.OpsCustomerDAO
	contractDAO   dao.OpsContractDAO
	logger        *zap.Logger
}

func NewOpsFinanceService(
	settlementDAO dao.OpsSettlementDAO,
	invoiceDAO dao.OpsInvoiceDAO,
	paymentDAO dao.OpsPaymentDAO,
	customerDAO dao.OpsCustomerDAO,
	contractDAO dao.OpsContractDAO,
	logger *zap.Logger,
) OpsFinanceService {
	return &opsFinanceService{
		settlementDAO: settlementDAO, invoiceDAO: invoiceDAO, paymentDAO: paymentDAO,
		customerDAO: customerDAO, contractDAO: contractDAO, logger: logger,
	}
}

func (s *opsFinanceService) CreateSettlement(ctx context.Context, req *model.CreateOpsSettlementReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.CustomerID); err != nil {
		return fmt.Errorf("客户不存在或已删除")
	}
	contract, err := s.contractDAO.GetByID(ctx, req.ContractID)
	if err != nil {
		return fmt.Errorf("合同不存在或已删除")
	}
	if contract.CustomerID != req.CustomerID {
		return fmt.Errorf("合同与所选客户不匹配")
	}
	periodStart, err := opsUtils.ParseOptionalTime(req.PeriodStart)
	if err != nil {
		return err
	}
	periodEnd, err := opsUtils.ParseOptionalTime(req.PeriodEnd)
	if err != nil {
		return err
	}
	dueAt, err := opsUtils.ParseOptionalTime(req.DueAt)
	if err != nil {
		return err
	}
	return s.settlementDAO.Create(ctx, &model.OpsSettlement{
		CustomerID: req.CustomerID, ContractID: req.ContractID, Title: strings.TrimSpace(req.Title),
		PeriodStart: periodStart, PeriodEnd: periodEnd, Amount: req.Amount, DueAt: dueAt,
		Status: model.OpsSettlementStatusDraft, Remark: req.Remark,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	})
}

func (s *opsFinanceService) UpdateSettlement(ctx context.Context, req *model.UpdateOpsSettlementReq) error {
	status := req.Status
	if status == "" {
		status = model.OpsSettlementStatusDraft
	}
	periodStart, err := opsUtils.ParseOptionalTime(req.PeriodStart)
	if err != nil {
		return err
	}
	periodEnd, err := opsUtils.ParseOptionalTime(req.PeriodEnd)
	if err != nil {
		return err
	}
	dueAt, err := opsUtils.ParseOptionalTime(req.DueAt)
	if err != nil {
		return err
	}
	return s.settlementDAO.Update(ctx, &model.OpsSettlement{
		Model: model.Model{ID: req.ID}, Title: strings.TrimSpace(req.Title),
		PeriodStart: periodStart, PeriodEnd: periodEnd, Amount: req.Amount,
		DueAt: dueAt, Status: status, Remark: req.Remark,
	})
}
func (s *opsFinanceService) DeleteSettlement(ctx context.Context, id int) error {
	return s.settlementDAO.Delete(ctx, id)
}
func (s *opsFinanceService) GetSettlement(ctx context.Context, id int) (*model.OpsSettlement, error) {
	return s.settlementDAO.GetByID(ctx, id)
}
func (s *opsFinanceService) ListSettlement(ctx context.Context, req *model.ListOpsSettlementReq) (*model.ListResp[*model.OpsSettlement], error) {
	items, total, err := s.settlementDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsSettlement]{Items: items, Total: total}, nil
}
func (s *opsFinanceService) ConfirmSettlement(ctx context.Context, id int) error {
	st, err := s.settlementDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if st.Status != model.OpsSettlementStatusDraft {
		return fmt.Errorf("仅草稿结算单可确认")
	}
	return s.settlementDAO.UpdateStatus(ctx, id, model.OpsSettlementStatusConfirmed)
}

func (s *opsFinanceService) CreateInvoice(ctx context.Context, req *model.CreateOpsInvoiceReq) error {
	st, err := s.settlementDAO.GetByID(ctx, req.SettlementID)
	if err != nil {
		return err
	}
	amount := req.Amount
	if amount <= 0 {
		amount = st.Amount
	}
	issuedAt, err := opsUtils.ParseOptionalTime(req.IssuedAt)
	if err != nil {
		return err
	}
	if err := s.invoiceDAO.Create(ctx, &model.OpsInvoice{
		CustomerID: req.CustomerID, SettlementID: req.SettlementID, InvoiceNo: req.InvoiceNo,
		InvoiceType: req.InvoiceType, Amount: amount, IssuedAt: issuedAt,
		Status: model.OpsInvoiceStatusIssued, OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	}); err != nil {
		return err
	}
	return s.settlementDAO.UpdateStatus(ctx, req.SettlementID, model.OpsSettlementStatusInvoiced)
}

func (s *opsFinanceService) UpdateInvoice(ctx context.Context, req *model.UpdateOpsInvoiceReq) error {
	status := req.Status
	if status == "" {
		status = model.OpsInvoiceStatusIssued
	}
	issuedAt, err := opsUtils.ParseOptionalTime(req.IssuedAt)
	if err != nil {
		return err
	}
	return s.invoiceDAO.Update(ctx, &model.OpsInvoice{
		Model: model.Model{ID: req.ID}, InvoiceNo: req.InvoiceNo, InvoiceType: req.InvoiceType,
		Amount: req.Amount, IssuedAt: issuedAt, Status: status,
	})
}
func (s *opsFinanceService) DeleteInvoice(ctx context.Context, id int) error {
	return s.invoiceDAO.Delete(ctx, id)
}
func (s *opsFinanceService) ListInvoice(ctx context.Context, req *model.ListOpsInvoiceReq) (*model.ListResp[*model.OpsInvoice], error) {
	items, total, err := s.invoiceDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsInvoice]{Items: items, Total: total}, nil
}

func (s *opsFinanceService) CreatePayment(ctx context.Context, req *model.CreateOpsPaymentReq) error {
	if _, err := s.settlementDAO.GetByID(ctx, req.SettlementID); err != nil {
		return err
	}
	paidAt, err := opsUtils.ParseOptionalTime(req.PaidAt)
	if err != nil {
		return err
	}
	return s.paymentDAO.Create(ctx, &model.OpsPayment{
		CustomerID: req.CustomerID, SettlementID: req.SettlementID, Amount: req.Amount,
		PaidAt: paidAt, BankRef: req.BankRef, Status: model.OpsPaymentStatusPending,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName, Remark: req.Remark,
	})
}

func (s *opsFinanceService) UpdatePayment(ctx context.Context, req *model.UpdateOpsPaymentReq) error {
	status := req.Status
	if status == "" {
		status = model.OpsPaymentStatusPending
	}
	paidAt, err := opsUtils.ParseOptionalTime(req.PaidAt)
	if err != nil {
		return err
	}
	return s.paymentDAO.Update(ctx, &model.OpsPayment{
		Model: model.Model{ID: req.ID}, Amount: req.Amount, PaidAt: paidAt,
		BankRef: req.BankRef, Status: status, Remark: req.Remark,
	})
}
func (s *opsFinanceService) DeletePayment(ctx context.Context, id int) error {
	return s.paymentDAO.Delete(ctx, id)
}
func (s *opsFinanceService) ListPayment(ctx context.Context, req *model.ListOpsPaymentReq) (*model.ListResp[*model.OpsPayment], error) {
	items, total, err := s.paymentDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsPayment]{Items: items, Total: total}, nil
}

func (s *opsFinanceService) MatchPayment(ctx context.Context, id int) error {
	p, err := s.paymentDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.paymentDAO.Update(ctx, &model.OpsPayment{
		Model: model.Model{ID: id}, Amount: p.Amount, PaidAt: p.PaidAt, BankRef: p.BankRef,
		Status: model.OpsPaymentStatusMatched, Remark: p.Remark,
	}); err != nil {
		return err
	}
	return s.settlementDAO.UpdateStatus(ctx, p.SettlementID, model.OpsSettlementStatusPaid)
}
