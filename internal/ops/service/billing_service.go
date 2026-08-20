package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	"go.uber.org/zap"
)

type OpsBillingService interface {
	GenerateMonthlyDrafts(ctx context.Context) (*OpsBillingGenerateResult, error)
}

type OpsBillingGenerateResult struct {
	ContractChecked   int      `json:"contract_checked"`
	SettlementCreated int      `json:"settlement_created"`
	InvoiceCreated    int      `json:"invoice_created"`
	Skipped           int      `json:"skipped"`
	SkipReasons       []string `json:"skip_reasons,omitempty"`
	Period            string   `json:"period"`
	Hint              string   `json:"hint,omitempty"`
}

type opsBillingService struct {
	contractDAO   dao.OpsContractDAO
	settlementDAO dao.OpsSettlementDAO
	invoiceDAO    dao.OpsInvoiceDAO
	logger        *zap.Logger
}

func NewOpsBillingService(
	contractDAO dao.OpsContractDAO,
	settlementDAO dao.OpsSettlementDAO,
	invoiceDAO dao.OpsInvoiceDAO,
	logger *zap.Logger,
) OpsBillingService {
	return &opsBillingService{
		contractDAO: contractDAO, settlementDAO: settlementDAO, invoiceDAO: invoiceDAO, logger: logger,
	}
}

func isMonthlyBilling(ct *model.OpsContract) bool {
	cycle := strings.ToLower(strings.TrimSpace(ct.BillingCycle))
	if cycle == "monthly" || cycle == "month" {
		return true
	}
	if cycle != "" {
		return false
	}
	// 未填周期时，包月/按量按月结处理；都空也默认按月
	mode := strings.ToLower(strings.TrimSpace(ct.BillingMode))
	return mode == "" || mode == "monthly" || mode == "month" || mode == "usage"
}

func (s *opsBillingService) GenerateMonthlyDrafts(ctx context.Context) (*OpsBillingGenerateResult, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, -1)
	result := &OpsBillingGenerateResult{
		Period: periodStart.Format("2006-01"),
	}

	contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
		ListReq: model.ListReq{Page: 1, Size: 500},
		Type:    model.OpsContractTypeFormal,
		Status:  model.OpsContractStatusActive,
	})
	if err != nil {
		return nil, err
	}
	for _, ct := range contracts {
		result.ContractChecked++
		if !isMonthlyBilling(ct) {
			result.Skipped++
			result.SkipReasons = append(result.SkipReasons,
				fmt.Sprintf("合同#%d「%s」计费周期非按月（cycle=%s mode=%s）", ct.ID, ct.Title, ct.BillingCycle, ct.BillingMode))
			continue
		}
		// 已存在同周期草稿/确认结算则跳过
		existing, _, _ := s.settlementDAO.List(ctx, &model.ListOpsSettlementReq{
			ListReq: model.ListReq{Page: 1, Size: 20}, CustomerID: ct.CustomerID,
		})
		duplicated := false
		for _, st := range existing {
			if st.ContractID == ct.ID && st.PeriodStart != nil && st.PeriodEnd != nil &&
				sameDay(*st.PeriodStart, periodStart) && sameDay(*st.PeriodEnd, periodEnd) {
				duplicated = true
				break
			}
		}
		if duplicated {
			result.Skipped++
			result.SkipReasons = append(result.SkipReasons,
				fmt.Sprintf("合同#%d「%s」本月结算已存在，跳过", ct.ID, ct.Title))
			continue
		}
		ps, pe := periodStart, periodEnd
		due := pe.AddDate(0, 0, ct.PaymentTermDays)
		title := fmt.Sprintf("%s-%s月结算", ct.Title, periodStart.Format("2006-01"))
		st := &model.OpsSettlement{
			CustomerID: ct.CustomerID, ContractID: ct.ID, Title: title,
			PeriodStart: &ps, PeriodEnd: &pe, Amount: ct.UnitPrice, DueAt: &due,
			Status: model.OpsSettlementStatusDraft, Remark: "系统按月自动生成草稿",
			OperatorID: 0, OperatorName: "system",
		}
		if err := s.settlementDAO.Create(ctx, st); err != nil {
			s.logger.Warn("自动生成结算失败", zap.Int("contract_id", ct.ID), zap.Error(err))
			result.Skipped++
			result.SkipReasons = append(result.SkipReasons,
				fmt.Sprintf("合同#%d「%s」创建结算失败: %v", ct.ID, ct.Title, err))
			continue
		}
		result.SettlementCreated++
		inv := &model.OpsInvoice{
			CustomerID: ct.CustomerID, SettlementID: st.ID,
			InvoiceType: "vat_normal", Amount: ct.UnitPrice,
			Status: model.OpsInvoiceStatusDraft, OperatorID: 0, OperatorName: "system",
		}
		if err := s.invoiceDAO.Create(ctx, inv); err == nil {
			result.InvoiceCreated++
		}
	}
	if result.ContractChecked == 0 {
		result.Hint = "没有「正式 + 生效中」的合同。请到「合同与开通」将正式合同状态改为「生效中(active)」，计费周期选「按月」。"
	} else if result.SettlementCreated == 0 {
		result.Hint = "未新建草稿：合同不满足按月条件，或本月结算已存在。可到「结算对账」查看已有单据。"
	} else {
		result.Hint = "已生成结算/发票草稿，请到「结算对账」确认后开票、收款。"
	}
	return result, nil
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
