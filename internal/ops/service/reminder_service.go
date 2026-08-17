package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type OpsReminderService interface {
	ListRules(ctx context.Context) (*model.ListResp[*model.OpsReminderRule], error)
	CreateRule(ctx context.Context, req *model.CreateOpsReminderRuleReq) error
	UpdateRule(ctx context.Context, req *model.UpdateOpsReminderRuleReq) error
	EnsureDefaults(ctx context.Context) error
	PreviewHits(ctx context.Context) ([]*model.OpsReminderHit, error)
	RunScan(ctx context.Context) (*model.OpsReminderScanResult, error)
	ScanAndNotify(ctx context.Context) error
}

type opsReminderService struct {
	ruleDAO       dao.OpsReminderRuleDAO
	trialDAO      dao.OpsTrialDAO
	contractDAO   dao.OpsContractDAO
	settlementDAO dao.OpsSettlementDAO
	customerDAO   dao.OpsCustomerDAO
	inboxDAO      workorderDao.WorkorderInboxDAO
	bizSvc        OpsBizService
	logger        *zap.Logger
}

func NewOpsReminderService(
	ruleDAO dao.OpsReminderRuleDAO,
	trialDAO dao.OpsTrialDAO,
	contractDAO dao.OpsContractDAO,
	settlementDAO dao.OpsSettlementDAO,
	customerDAO dao.OpsCustomerDAO,
	inboxDAO workorderDao.WorkorderInboxDAO,
	bizSvc OpsBizService,
	logger *zap.Logger,
) OpsReminderService {
	return &opsReminderService{
		ruleDAO: ruleDAO, trialDAO: trialDAO, contractDAO: contractDAO,
		settlementDAO: settlementDAO, customerDAO: customerDAO, inboxDAO: inboxDAO,
		bizSvc: bizSvc, logger: logger,
	}
}

func (s *opsReminderService) ListRules(ctx context.Context) (*model.ListResp[*model.OpsReminderRule], error) {
	_ = s.ruleDAO.EnsureDefaults(ctx)
	items, err := s.ruleDAO.List(ctx)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsReminderRule]{Items: items, Total: int64(len(items))}, nil
}

func (s *opsReminderService) CreateRule(ctx context.Context, req *model.CreateOpsReminderRuleReq) error {
	enabled := req.Enabled
	if enabled != 1 && enabled != 2 {
		enabled = 1
	}
	channels := req.Channels
	if len(channels) == 0 {
		channels = model.StringList{"inbox"}
	}
	return s.ruleDAO.Create(ctx, &model.OpsReminderRule{
		Scene:       req.Scene,
		Name:        strings.TrimSpace(req.Name),
		AdvanceDays: req.AdvanceDays,
		Enabled:     enabled,
		Channels:    channels,
		Remark:      req.Remark,
	})
}

func (s *opsReminderService) UpdateRule(ctx context.Context, req *model.UpdateOpsReminderRuleReq) error {
	rule, err := s.ruleDAO.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.AdvanceDays >= 0 {
		rule.AdvanceDays = req.AdvanceDays
	}
	if req.Enabled == 1 || req.Enabled == 2 {
		rule.Enabled = req.Enabled
	}
	if req.Channels != nil {
		rule.Channels = req.Channels
	}
	rule.Remark = req.Remark
	return s.ruleDAO.Update(ctx, rule)
}

func (s *opsReminderService) EnsureDefaults(ctx context.Context) error {
	return s.ruleDAO.EnsureDefaults(ctx)
}

func (s *opsReminderService) PreviewHits(ctx context.Context) ([]*model.OpsReminderHit, error) {
	_ = s.ruleDAO.EnsureDefaults(ctx)
	return s.collectHits(ctx)
}

func (s *opsReminderService) RunScan(ctx context.Context) (*model.OpsReminderScanResult, error) {
	_ = s.bizSvc.SyncPendingApprovals(ctx)
	hits, err := s.collectHits(ctx)
	if err != nil {
		return nil, err
	}
	result := &model.OpsReminderScanResult{HitCount: len(hits)}
	for _, hit := range hits {
		if hit.TargetUserID <= 0 {
			result.SkippedNoOwner++
			continue
		}
		title := hit.RuleName
		if title == "" {
			title = "运营提醒"
		}
		if err := s.pushInbox(ctx, hit.TargetUserID, title, hit.Reason, hit.Link); err == nil {
			result.NotifyCount++
		}
	}
	return result, nil
}

func (s *opsReminderService) ScanAndNotify(ctx context.Context) error {
	_, err := s.RunScan(ctx)
	return err
}

func (s *opsReminderService) collectHits(ctx context.Context) ([]*model.OpsReminderHit, error) {
	rules, err := s.ruleDAO.List(ctx)
	if err != nil {
		return nil, err
	}
	var hits []*model.OpsReminderHit
	for _, rule := range rules {
		if rule.Enabled != 1 {
			continue
		}
		switch rule.Scene {
		case model.OpsReminderSceneTrialExpire:
			items, _ := s.trialDAO.ListExpiring(ctx, rule.AdvanceDays)
			for _, t := range items {
				c, err := s.customerDAO.GetByID(ctx, t.CustomerID)
				if err != nil {
					continue
				}
				hits = append(hits, s.buildHit(rule, "trial", t.ID, t.Title, c,
					fmt.Sprintf("试用将在 %d 天内到期，请跟进转化/续约", rule.AdvanceDays)))
			}
		case model.OpsReminderSceneContractRenew:
			items, _ := s.contractDAO.ListExpiring(ctx, rule.AdvanceDays)
			for _, ct := range items {
				c, err := s.customerDAO.GetByID(ctx, ct.CustomerID)
				if err != nil {
					continue
				}
				hits = append(hits, s.buildHit(rule, "contract", ct.ID, ct.Title, c,
					fmt.Sprintf("合同将在 %d 天内到期，请跟进续约", rule.AdvanceDays)))
			}
		case model.OpsReminderSceneSettlementDue, model.OpsReminderScenePaymentFollowup:
			items, _ := s.settlementDAO.ListDueSoon(ctx, rule.AdvanceDays)
			for _, st := range items {
				c, err := s.customerDAO.GetByID(ctx, st.CustomerID)
				if err != nil {
					continue
				}
				hits = append(hits, s.buildHit(rule, "settlement", st.ID, st.Title, c,
					fmt.Sprintf("结算单临近账期（提前 %d 天），请跟进回款", rule.AdvanceDays)))
			}
			if rule.Scene == model.OpsReminderScenePaymentFollowup {
				overdue, _ := s.settlementDAO.ListOverdue(ctx)
				for _, st := range overdue {
					c, err := s.customerDAO.GetByID(ctx, st.CustomerID)
					if err != nil {
						continue
					}
					hits = append(hits, s.buildHit(rule, "settlement", st.ID, st.Title, c, "结算单已逾期，请尽快跟进回款"))
				}
			}
		case model.OpsReminderSceneInvoice:
			items, _, _ := s.settlementDAO.List(ctx, &model.ListOpsSettlementReq{
				ListReq: model.ListReq{Page: 1, Size: 50}, Status: model.OpsSettlementStatusConfirmed,
			})
			for _, st := range items {
				c, err := s.customerDAO.GetByID(ctx, st.CustomerID)
				if err != nil {
					continue
				}
				hits = append(hits, s.buildHit(rule, "settlement", st.ID, st.Title, c, "结算单已确认，待开票"))
			}
		}
	}
	return hits, nil
}

func (s *opsReminderService) buildHit(
	rule *model.OpsReminderRule,
	bizType string,
	bizID int,
	bizTitle string,
	customer *model.OpsCustomer,
	reason string,
) *model.OpsReminderHit {
	targetID := 0
	hint := "未设置客户负责人，将无法发送"
	if customer != nil && customer.OwnerID > 0 {
		targetID = customer.OwnerID
		hint = customer.OwnerName
		if hint == "" {
			hint = fmt.Sprintf("用户#%d", customer.OwnerID)
		}
	}
	link := "/ops/settlements"
	if customer != nil {
		link = fmt.Sprintf("/ops/customers/detail/%d", customer.ID)
	}
	customerID, customerName := 0, ""
	if customer != nil {
		customerID, customerName = customer.ID, customer.Name
	}
	return &model.OpsReminderHit{
		RuleID: rule.ID, Scene: rule.Scene, RuleName: rule.Name, AdvanceDays: rule.AdvanceDays,
		TargetUserID: targetID, TargetUserHint: hint,
		BizType: bizType, BizID: bizID, BizTitle: bizTitle,
		CustomerID: customerID, CustomerName: customerName,
		Link: link, Reason: fmt.Sprintf("客户「%s」%s：%s", customerName, reason, bizTitle),
	}
}

func (s *opsReminderService) pushInbox(ctx context.Context, userID int, title, content, link string) error {
	if userID <= 0 || s.inboxDAO == nil {
		return nil
	}
	body := content
	if link != "" {
		body = content + " → 打开: " + link
	}
	return s.inboxDAO.Create(ctx, &model.WorkorderInboxMessage{
		UserID: userID, Title: "[运营] " + title, Content: body,
		EventType: "ops_reminder", IsRead: model.InboxMessageUnread,
	})
}
