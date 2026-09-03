package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type OpsWorkbenchService interface {
	Briefing(ctx context.Context, req *model.OpsWorkbenchBriefingReq) (*model.OpsWorkbenchBriefing, error)
	DraftReminder(ctx context.Context, req *model.OpsReminderDraftReq) (*model.OpsReminderDraftResp, error)
}

type opsWorkbenchService struct {
	trialDAO      dao.OpsTrialDAO
	contractDAO   dao.OpsContractDAO
	settlementDAO dao.OpsSettlementDAO
	customerDAO   dao.OpsCustomerDAO
	surveyDAO     dao.OpsSurveyDAO
	logger        *zap.Logger
	httpClient    *http.Client
}

func NewOpsWorkbenchService(
	trialDAO dao.OpsTrialDAO,
	contractDAO dao.OpsContractDAO,
	settlementDAO dao.OpsSettlementDAO,
	customerDAO dao.OpsCustomerDAO,
	surveyDAO dao.OpsSurveyDAO,
	logger *zap.Logger,
) OpsWorkbenchService {
	return &opsWorkbenchService{
		trialDAO: trialDAO, contractDAO: contractDAO, settlementDAO: settlementDAO,
		customerDAO: customerDAO, surveyDAO: surveyDAO, logger: logger,
		httpClient: &http.Client{Timeout: 12 * time.Second},
	}
}

func (s *opsWorkbenchService) Briefing(ctx context.Context, req *model.OpsWorkbenchBriefingReq) (*model.OpsWorkbenchBriefing, error) {
	within := 7
	if req != nil && req.WithinDays > 0 && req.WithinDays <= 30 {
		within = req.WithinDays
	}
	mineOnly := req != nil && req.MineOnly && !req.IsAdmin
	ownerID := 0
	if req != nil {
		ownerID = req.OwnerID
	}

	var (
		trials      []*model.OpsTrial
		contracts   []*model.OpsContract
		dueSoon     []*model.OpsSettlement
		overdue     []*model.OpsSettlement
		lowScores   []*model.OpsSurveyResponse
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		trials, err = s.trialDAO.ListExpiring(gctx, within)
		return err
	})
	g.Go(func() error {
		var err error
		contracts, err = s.contractDAO.ListExpiring(gctx, within)
		return err
	})
	g.Go(func() error {
		var err error
		dueSoon, err = s.settlementDAO.ListDueSoon(gctx, within)
		return err
	})
	g.Go(func() error {
		var err error
		overdue, err = s.settlementDAO.ListOverdue(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		lowScores, err = s.surveyDAO.ListLowScoreResponses(gctx, 3, 30)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("加载工作台数据失败: %w", err)
	}

	custCache := map[int]*model.OpsCustomer{}
	getCustomer := func(id int) *model.OpsCustomer {
		if id <= 0 {
			return nil
		}
		if c, ok := custCache[id]; ok {
			return c
		}
		c, err := s.customerDAO.GetByID(ctx, id)
		if err != nil {
			return nil
		}
		custCache[id] = c
		return c
	}

	items := make([]model.OpsRiskItem, 0, 64)
	summary := model.OpsRiskSummary{}

	for _, t := range trials {
		c := getCustomer(t.CustomerID)
		if mineOnly && (c == nil || c.OwnerID != ownerID) {
			continue
		}
		name := ""
		ownerName := ""
		oid := 0
		if c != nil {
			name, ownerName, oid = c.Name, c.OwnerName, c.OwnerID
		}
		due := t.PlanEndAt
		items = append(items, model.OpsRiskItem{
			Type: model.OpsRiskTrialExpiring, Severity: model.OpsRiskSeverityHigh,
			Title: "试用即将到期", Summary: fmt.Sprintf("试用「%s」将在窗口内到期，需确认是否转正式", t.Title),
			Suggestion: "联系客户确认转正意向，必要时创建到期提醒",
			CustomerID: t.CustomerID, CustomerName: name, OwnerID: oid, OwnerName: ownerName,
			BizType: "trial", BizID: t.ID, DueAt: due,
			RefPath: fmt.Sprintf("/ops/customers/detail/%d", t.CustomerID),
			DraftHint: model.OpsRiskTrialExpiring,
		})
		summary.TrialExpiring++
	}

	for _, ct := range contracts {
		c := getCustomer(ct.CustomerID)
		if mineOnly && (c == nil || c.OwnerID != ownerID) {
			continue
		}
		name, ownerName, oid := "", "", 0
		if c != nil {
			name, ownerName, oid = c.Name, c.OwnerName, c.OwnerID
		}
		items = append(items, model.OpsRiskItem{
			Type: model.OpsRiskContractExpiring, Severity: model.OpsRiskSeverityMedium,
			Title: "正式合同即将到期", Summary: fmt.Sprintf("合同「%s」即将到期，关注续约或增值", ct.Title),
			Suggestion: "确认续约意向；不续费则发起问卷外链",
			CustomerID: ct.CustomerID, CustomerName: name, OwnerID: oid, OwnerName: ownerName,
			BizType: "contract", BizID: ct.ID, DueAt: ct.EndAt,
			RefPath: fmt.Sprintf("/ops/customers/detail/%d", ct.CustomerID),
			DraftHint: model.OpsRiskContractExpiring,
		})
		summary.ContractExpiring++
	}

	for _, st := range dueSoon {
		c := getCustomer(st.CustomerID)
		if mineOnly && (c == nil || c.OwnerID != ownerID) {
			continue
		}
		name, ownerName, oid := "", "", 0
		if c != nil {
			name, ownerName, oid = c.Name, c.OwnerName, c.OwnerID
		}
		items = append(items, model.OpsRiskItem{
			Type: model.OpsRiskSettlementDueSoon, Severity: model.OpsRiskSeverityMedium,
			Title: "结算即将到期", Summary: fmt.Sprintf("结算「%s」金额 %.2f 即将到期", st.Title, st.Amount),
			Suggestion: "确认开票与回款计划，必要时发送催款提醒",
			CustomerID: st.CustomerID, CustomerName: name, OwnerID: oid, OwnerName: ownerName,
			BizType: "settlement", BizID: st.ID, DueAt: st.DueAt,
			RefPath: "/ops/settlements",
			DraftHint: model.OpsRiskSettlementDueSoon,
		})
		summary.SettlementDueSoon++
	}

	for _, st := range overdue {
		c := getCustomer(st.CustomerID)
		if mineOnly && (c == nil || c.OwnerID != ownerID) {
			continue
		}
		name, ownerName, oid := "", "", 0
		if c != nil {
			name, ownerName, oid = c.Name, c.OwnerName, c.OwnerID
		}
		items = append(items, model.OpsRiskItem{
			Type: model.OpsRiskSettlementOverdue, Severity: model.OpsRiskSeverityHigh,
			Title: "结算已逾期", Summary: fmt.Sprintf("结算「%s」金额 %.2f 已逾期未结清", st.Title, st.Amount),
			Suggestion: "升级催收并同步财务；记录跟进",
			CustomerID: st.CustomerID, CustomerName: name, OwnerID: oid, OwnerName: ownerName,
			BizType: "settlement", BizID: st.ID, DueAt: st.DueAt,
			RefPath: "/ops/settlements",
			DraftHint: model.OpsRiskSettlementOverdue,
		})
		summary.SettlementOverdue++
	}

	seenLow := map[int]bool{}
	for _, resp := range lowScores {
		if seenLow[resp.CustomerID] {
			continue
		}
		c := getCustomer(resp.CustomerID)
		if mineOnly && (c == nil || c.OwnerID != ownerID) {
			continue
		}
		seenLow[resp.CustomerID] = true
		name, ownerName, oid := "", "", 0
		if c != nil {
			name, ownerName, oid = c.Name, c.OwnerName, c.OwnerID
		}
		items = append(items, model.OpsRiskItem{
			Type: model.OpsRiskSurveyLowScore, Severity: model.OpsRiskSeverityHigh,
			Title: "满意度偏低", Summary: fmt.Sprintf("客户满意度评分 %d，需跟进问题与建议", resp.Score),
			Suggestion: "电话回访并记录跟进；必要时升级交付",
			CustomerID: resp.CustomerID, CustomerName: name, OwnerID: oid, OwnerName: ownerName,
			BizType: "survey_response", BizID: resp.ID, DueAt: resp.SubmittedAt,
			RefPath: "/ops/surveys",
			DraftHint: model.OpsRiskSurveyLowScore,
		})
		summary.SurveyLowScore++
	}

	// 已填不续费问卷但尚未闭环
	nonRenewals, _, _ := s.surveyDAO.ListResponses(ctx, &model.ListOpsSurveyResponseReq{
		ListReq: model.ListReq{Page: 1, Size: 50}, SurveyType: model.OpsSurveyTypeNonRenewal,
	})
	seenNR := map[int]bool{}
	for _, resp := range nonRenewals {
		if seenNR[resp.CustomerID] {
			continue
		}
		c := getCustomer(resp.CustomerID)
		if c == nil || c.Stage == model.OpsCustomerStageClosed {
			continue
		}
		if mineOnly && c.OwnerID != ownerID {
			continue
		}
		seenNR[resp.CustomerID] = true
		items = append(items, model.OpsRiskItem{
			Type: model.OpsRiskNonRenewalPending, Severity: model.OpsRiskSeverityMedium,
			Title: "不续费问卷已回收", Summary: "客户已提交不续费原因，待确认闭环或挽留",
			Suggestion: "复核原因后变更阶段闭环，或安排挽留跟进",
			CustomerID: c.ID, CustomerName: c.Name, OwnerID: c.OwnerID, OwnerName: c.OwnerName,
			BizType: "survey_response", BizID: resp.ID, DueAt: resp.SubmittedAt,
			RefPath: fmt.Sprintf("/ops/customers/detail/%d", c.ID),
			DraftHint: model.OpsRiskNonRenewalPending,
		})
		summary.NonRenewalPending++
	}

	summary.Total = len(items)
	return &model.OpsWorkbenchBriefing{
		GeneratedAt: time.Now().Format(time.RFC3339),
		WithinDays:  within,
		Summary:     summary,
		Items:       items,
	}, nil
}

func (s *opsWorkbenchService) DraftReminder(ctx context.Context, req *model.OpsReminderDraftReq) (*model.OpsReminderDraftResp, error) {
	channel := strings.TrimSpace(req.Channel)
	if channel == "" {
		channel = model.OpsReminderChannelInbox
	}
	tpl := buildReminderTemplate(req, channel)
	if useLLMDraft() {
		if body, err := s.generateLLMDraft(ctx, req, channel); err == nil && strings.TrimSpace(body) != "" {
			return &model.OpsReminderDraftResp{
				Channel: channel, Subject: tpl.Subject, Body: body,
				Source: "llm", Editable: true, Hint: "由大模型生成，请人工确认后再发送",
			}, nil
		}
	}
	return &model.OpsReminderDraftResp{
		Channel: channel, Subject: tpl.Subject, Body: tpl.Body,
		Source: "template", Editable: true, Hint: "模板生成；配置 LLM_API_KEY 后可尝试智能润色",
	}, nil
}

type reminderTpl struct {
	Subject string
	Body    string
}

func buildReminderTemplate(req *model.OpsReminderDraftReq, channel string) reminderTpl {
	name := strings.TrimSpace(req.CustomerName)
	if name == "" {
		name = "客户"
	}
	due := strings.TrimSpace(req.DueAt)
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "运营事项提醒"
	}
	var subject, body string
	switch req.Type {
	case model.OpsRiskTrialExpiring:
		subject = fmt.Sprintf("【试用到期】%s", name)
		body = fmt.Sprintf("您好，客户「%s」的算力试用即将到期（%s）。请确认是否转正式合作，并同步开通与合同安排。", name, dueOrDash(due))
	case model.OpsRiskContractExpiring:
		subject = fmt.Sprintf("【合同到期】%s", name)
		body = fmt.Sprintf("您好，客户「%s」正式合同即将到期（%s）。请跟进续约意向；若不续费，请发送不续费问卷外链回收原因。", name, dueOrDash(due))
	case model.OpsRiskSettlementDueSoon:
		subject = fmt.Sprintf("【结算提醒】%s", name)
		body = fmt.Sprintf("您好，客户「%s」有结算即将到期：%s（%s）。请确认开票与回款进度。", name, title, dueOrDash(due))
	case model.OpsRiskSettlementOverdue:
		subject = fmt.Sprintf("【逾期催收】%s", name)
		body = fmt.Sprintf("您好，客户「%s」结算已逾期：%s（应付日 %s）。请尽快跟进回款并更新跟进记录。", name, title, dueOrDash(due))
	case model.OpsRiskSurveyLowScore:
		subject = fmt.Sprintf("【满意度跟进】%s", name)
		body = fmt.Sprintf("您好，客户「%s」近期满意度评分偏低。请尽快回访了解问题与建议，并在客户详情补充跟进。", name)
	case model.OpsRiskNonRenewalPending:
		subject = fmt.Sprintf("【不续费待闭环】%s", name)
		body = fmt.Sprintf("您好，客户「%s」已提交不续费原因问卷，请复核后办理闭环或安排挽留。", name)
	default:
		subject = title
		body = fmt.Sprintf("您好，关于客户「%s」：%s。建议：%s", name, firstNonEmpty(req.Summary, title), firstNonEmpty(req.Suggestion, "请及时跟进"))
	}
	if channel == model.OpsReminderChannelSMS {
		// 短信控制长度
		runes := []rune(body)
		if len(runes) > 120 {
			body = string(runes[:117]) + "..."
		}
	}
	return reminderTpl{Subject: subject, Body: body}
}

func dueOrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "近期"
	}
	return s
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func useLLMDraft() bool {
	return strings.TrimSpace(os.Getenv("LLM_API_KEY")) != "" ||
		strings.TrimSpace(viper.GetString("external.llm.api_key")) != "" ||
		strings.TrimSpace(viper.GetString("llm.api_key")) != ""
}

func (s *opsWorkbenchService) generateLLMDraft(ctx context.Context, req *model.OpsReminderDraftReq, channel string) (string, error) {
	apiKey, baseURL, model, err := resolveLLMConfig()
	if err != nil {
		return "", err
	}
	prompt := fmt.Sprintf(
		"你是 CacOps 运营助手。请为渠道「%s」写一条简短中文提醒正文（不要标题），对象客户「%s」，事项类型「%s」，摘要「%s」，建议「%s」，到期「%s」。语气专业、可直接发给内部负责人，不要虚构合同金额。",
		channel, req.CustomerName, req.Type, req.Summary, req.Suggestion, req.DueAt,
	)
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "你只输出提醒正文，不要解释。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.4,
	}
	raw, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm status %d model=%s: %s", resp.StatusCode, model, truncateLLMErrBody(body))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm empty")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
