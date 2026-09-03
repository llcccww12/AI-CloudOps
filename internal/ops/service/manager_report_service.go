package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const managerReportCacheTTL = 30 * time.Minute

type OpsManagerReportService interface {
	WeeklyReport(ctx context.Context, days int, refresh bool) (*model.OpsManagerWeeklyReport, error)
}

type managerReportCacheEntry struct {
	report    *model.OpsManagerWeeklyReport
	expiresAt time.Time
}

type opsManagerReportService struct {
	dashboard  OpsDashboardService
	workbench  OpsWorkbenchService
	logger     *zap.Logger
	httpClient *http.Client
	cacheMu    sync.RWMutex
	cache      map[string]managerReportCacheEntry
}

func NewOpsManagerReportService(
	dashboard OpsDashboardService,
	workbench OpsWorkbenchService,
	logger *zap.Logger,
) OpsManagerReportService {
	return &opsManagerReportService{
		dashboard: dashboard, workbench: workbench, logger: logger,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cache:      make(map[string]managerReportCacheEntry),
	}
}

func managerReportCacheKey(days int) string {
	return fmt.Sprintf("%s:%d", time.Now().Format("2006-01-02"), days)
}

func (s *opsManagerReportService) WeeklyReport(ctx context.Context, days int, refresh bool) (*model.OpsManagerWeeklyReport, error) {
	if days <= 0 || days > 31 {
		days = 7
	}
	key := managerReportCacheKey(days)
	if !refresh {
		s.cacheMu.RLock()
		entry, ok := s.cache[key]
		s.cacheMu.RUnlock()
		if ok && time.Now().Before(entry.expiresAt) && entry.report != nil {
			cached := *entry.report
			cached.Cached = true
			if cached.Hint == "" {
				cached.Hint = "展示缓存周报；需要最新数据时点「重新生成」。"
			} else if !strings.Contains(cached.Hint, "缓存") {
				cached.Hint = cached.Hint + "（当前为缓存，点「重新生成」可刷新）"
			}
			return &cached, nil
		}
	}

	report, err := s.buildWeeklyReport(ctx, days)
	if err != nil {
		return nil, err
	}
	report.Cached = false

	s.cacheMu.Lock()
	s.cache[key] = managerReportCacheEntry{report: report, expiresAt: time.Now().Add(managerReportCacheTTL)}
	s.cacheMu.Unlock()
	return report, nil
}

func (s *opsManagerReportService) buildWeeklyReport(ctx context.Context, days int) (*model.OpsManagerWeeklyReport, error) {
	now := time.Now()
	periodEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	periodStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))

	overview, err := s.dashboard.Overview(ctx, &model.OpsDashboardOverviewReq{Days: days})
	if err != nil {
		return nil, fmt.Errorf("加载看板指标失败: %w", err)
	}
	briefing, err := s.workbench.Briefing(ctx, &model.OpsWorkbenchBriefingReq{
		WithinDays: days, MineOnly: false, IsAdmin: true,
	})
	if err != nil {
		return nil, fmt.Errorf("加载风险简报失败: %w", err)
	}

	top := briefing.Items
	if len(top) > 12 {
		top = top[:12]
	}
	facts, actions := buildManagerFactsAndActions(overview, briefing)
	snap := model.OpsManagerSnapshot{
		KPIs: overview.KPIs, Funnel: overview.Funnel,
		RiskSummary: briefing.Summary, TopRisks: top,
		FactBullets: facts, SuggestedActions: actions,
	}

	exec := defaultExecutiveSummary(snap)
	source := "template"
	llmOn := llmConfigured()
	hint := "页面为可视化周报；可复制 Markdown 外发。一句话总览默认由规则生成。"
	if llmOn {
		hint = "已检测到 LLM 配置：生成时会尝试润色「一句话总览」（仍以台账数字为准）。"
		if polished, err := s.polishExecutiveSummary(ctx, snap, days); err == nil && strings.TrimSpace(polished) != "" {
			exec = strings.TrimSpace(polished)
			source = "hybrid"
			hint = "指标与清单来自台账；一句话总览已经模型润色，请确认后再复制外发。"
		} else if err != nil {
			s.logger.Warn("周报总览润色失败，回退模板", zap.Error(err))
			hint = "LLM 已配置但润色失败，已回退规则总览。请检查 LLM_API_KEY / LLM_BASE_URL。"
		}
	} else {
		hint = "未配置 LLM：总览为规则文案。在环境变量设置 LLM_API_KEY（可选 LLM_BASE_URL）后点「重新生成」即可润色总览。"
	}

	md := renderWeeklyMarkdown(periodStart, periodEnd, days, snap, exec)
	pageHTML := renderWeeklyHTML(periodStart, periodEnd, days, snap, exec, source)

	return &model.OpsManagerWeeklyReport{
		GeneratedAt:      now.Format(time.RFC3339),
		PeriodStart:      periodStart.Format("2006-01-02"),
		PeriodEnd:        periodEnd.Format("2006-01-02"),
		Days:             days,
		Snapshot:         snap,
		ExecutiveSummary: exec,
		Markdown:         md,
		HTML:             pageHTML,
		Source:           source,
		LLMEnabled:       llmOn,
		Hint:             hint,
	}, nil
}

func buildManagerFactsAndActions(ov *model.OpsDashboardOverview, br *model.OpsWorkbenchBriefing) (facts, actions []string) {
	k := ov.KPIs
	facts = []string{
		fmt.Sprintf("客户阶段：意向 %d / 试用 %d / 正式 %d；本月闭环 %d", k.CustomerIntent, k.CustomerTrial, k.CustomerFormal, k.CustomerClosedMonth),
		fmt.Sprintf("展厅本周接待 %d，今日 %d；外访待跟进 %d，将到期 %d，已逾期 %d", k.ExhibitionWeek, k.ExhibitionToday, k.VisitPending, k.VisitDueSoon, k.VisitOverdue),
		fmt.Sprintf("风险合计 %d：试用将到期 %d、合同将到期 %d、结算将到期 %d、结算逾期 %d、低满意度 %d、不续费待闭环 %d",
			br.Summary.Total, br.Summary.TrialExpiring, br.Summary.ContractExpiring,
			br.Summary.SettlementDueSoon, br.Summary.SettlementOverdue,
			br.Summary.SurveyLowScore, br.Summary.NonRenewalPending),
	}
	for _, f := range ov.Funnel {
		if f.Key == "trial" || f.Key == "formal" {
			facts = append(facts, fmt.Sprintf("漏斗「%s」存量 %d，相对上一阶转化约 %.1f%%", f.Label, f.Count, f.Rate))
		}
	}

	if br.Summary.SettlementOverdue > 0 {
		actions = append(actions, "督办结算逾期客户回款，要求负责人更新跟进记录")
	}
	if br.Summary.TrialExpiring > 0 {
		actions = append(actions, "对试用将到期客户确认转正意向，避免静默流失")
	}
	if br.Summary.SurveyLowScore > 0 {
		actions = append(actions, "安排低满意度客户回访，沉淀问题与改进项")
	}
	if br.Summary.NonRenewalPending > 0 {
		actions = append(actions, "复核已填不续费问卷客户，完成闭环或启动挽留")
	}
	if br.Summary.ContractExpiring > 0 {
		actions = append(actions, "推进正式合同续约沟通，不续费同步问卷外链")
	}
	if k.VisitOverdue > 0 {
		actions = append(actions, "清理外访逾期事项，避免高意向线索掉队")
	}
	if len(actions) == 0 {
		actions = append(actions, "本周无显著风险堆积，维持正常跟进节奏并关注转化漏斗")
	}
	if len(actions) > 5 {
		actions = actions[:5]
	}
	return facts, actions
}

func renderWeeklyMarkdown(start, end time.Time, days int, snap model.OpsManagerSnapshot, executive string) string {
	var b strings.Builder
	b.WriteString("# CacOps 运营周报\n\n")
	b.WriteString(fmt.Sprintf("> 周期：%s ~ %s（近 %d 天）  \n", start.Format("2006-01-02"), end.Format("2006-01-02"), days))
	b.WriteString(fmt.Sprintf("> 生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04")))

	b.WriteString("## 一、一句话总览\n\n")
	if strings.TrimSpace(executive) != "" {
		b.WriteString(strings.TrimSpace(executive))
		b.WriteString("\n\n")
	} else {
		b.WriteString(fmt.Sprintf(
			"本周期客户正式存量 %d、试用 %d；风险事项合计 %d（其中结算逾期 %d、试用将到期 %d）。建议优先处理逾期回款与试用转正确认。\n\n",
			snap.KPIs.CustomerFormal, snap.KPIs.CustomerTrial, snap.RiskSummary.Total,
			snap.RiskSummary.SettlementOverdue, snap.RiskSummary.TrialExpiring,
		))
	}

	b.WriteString("## 二、关键事实\n\n")
	for _, f := range snap.FactBullets {
		b.WriteString("- ")
		b.WriteString(f)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## 三、漏斗快照\n\n")
	b.WriteString("| 阶段 | 数量 | 相对上一阶转化%% |\n| --- | ---: | ---: |\n")
	for _, f := range snap.Funnel {
		b.WriteString(fmt.Sprintf("| %s | %d | %.1f |\n", f.Label, f.Count, f.Rate))
	}
	b.WriteString("\n")

	b.WriteString("## 四、重点风险（TOP）\n\n")
	if len(snap.TopRisks) == 0 {
		b.WriteString("暂无高优先级风险项。\n\n")
	} else {
		for i, r := range snap.TopRisks {
			due := "-"
			if r.DueAt != nil {
				due = r.DueAt.Format("2006-01-02")
			}
			b.WriteString(fmt.Sprintf("%d. **[%s]** %s — %s（客户：%s，负责人：%s，截止：%s）\n",
				i+1, riskTypeCN(r.Type), r.Title, r.Summary, emptyDash(r.CustomerName), emptyDash(r.OwnerName), due))
		}
		b.WriteString("\n")
	}

	b.WriteString("## 五、建议督办\n\n")
	for i, a := range snap.SuggestedActions {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, a))
	}
	b.WriteString("\n")

	b.WriteString("## 六、附录指标\n\n")
	k := snap.KPIs
	b.WriteString(fmt.Sprintf("- 展厅：今日 %d / 本周 %d\n", k.ExhibitionToday, k.ExhibitionWeek))
	b.WriteString(fmt.Sprintf("- 外访：待跟进 %d / 将到期 %d / 逾期 %d\n", k.VisitPending, k.VisitDueSoon, k.VisitOverdue))
	b.WriteString(fmt.Sprintf("- 客户：意向 %d / 试用 %d / 正式 %d / 本月闭环 %d\n", k.CustomerIntent, k.CustomerTrial, k.CustomerFormal, k.CustomerClosedMonth))
	b.WriteString(fmt.Sprintf("- 风险：试用到期 %d / 合同到期 %d / 结算将到期 %d / 结算逾期 %d / 低满意度 %d / 不续费待闭环 %d\n",
		snap.RiskSummary.TrialExpiring, snap.RiskSummary.ContractExpiring,
		snap.RiskSummary.SettlementDueSoon, snap.RiskSummary.SettlementOverdue,
		snap.RiskSummary.SurveyLowScore, snap.RiskSummary.NonRenewalPending))
	b.WriteString("\n---\n*本报告由 CacOps 根据台账自动生成，仅供内部管理决策参考。*\n")
	return b.String()
}

func defaultExecutiveSummary(snap model.OpsManagerSnapshot) string {
	return fmt.Sprintf(
		"本周期客户正式存量 %d、试用 %d；风险事项合计 %d（其中结算逾期 %d、试用将到期 %d）。建议优先处理逾期回款与试用转正确认。",
		snap.KPIs.CustomerFormal, snap.KPIs.CustomerTrial, snap.RiskSummary.Total,
		snap.RiskSummary.SettlementOverdue, snap.RiskSummary.TrialExpiring,
	)
}

func renderWeeklyHTML(start, end time.Time, days int, snap model.OpsManagerSnapshot, executive, source string) string {
	esc := html.EscapeString
	var cards strings.Builder
	type kpi struct{ label string; value int64; tone string }
	kpis := []kpi{
		{"正式客户", snap.KPIs.CustomerFormal, ""},
		{"试用客户", snap.KPIs.CustomerTrial, ""},
		{"意向客户", snap.KPIs.CustomerIntent, ""},
		{"本月闭环", snap.KPIs.CustomerClosedMonth, ""},
		{"风险合计", int64(snap.RiskSummary.Total), "warn"},
		{"结算逾期", int64(snap.RiskSummary.SettlementOverdue), "danger"},
		{"试用将到期", int64(snap.RiskSummary.TrialExpiring), "warn"},
		{"外访逾期", snap.KPIs.VisitOverdue, "danger"},
	}
	for _, k := range kpis {
		cls := "kpi"
		if k.tone != "" {
			cls += " " + k.tone
		}
		cards.WriteString(fmt.Sprintf(`<div class="%s"><div class="n">%d</div><div class="l">%s</div></div>`, cls, k.value, esc(k.label)))
	}

	var funnel strings.Builder
	funnel.WriteString(`<table><thead><tr><th>阶段</th><th>数量</th><th>转化%</th></tr></thead><tbody>`)
	for _, f := range snap.Funnel {
		funnel.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%.1f</td></tr>", esc(f.Label), f.Count, f.Rate))
	}
	funnel.WriteString("</tbody></table>")

	var facts strings.Builder
	facts.WriteString("<ul>")
	for _, f := range snap.FactBullets {
		facts.WriteString("<li>" + esc(f) + "</li>")
	}
	facts.WriteString("</ul>")

	var risks strings.Builder
	if len(snap.TopRisks) == 0 {
		risks.WriteString(`<p class="empty">暂无高优先级风险项</p>`)
	} else {
		risks.WriteString(`<div class="risk-list">`)
		for _, r := range snap.TopRisks {
			due := "-"
			if r.DueAt != nil {
				due = r.DueAt.Format("2006-01-02")
			}
			sev := esc(r.Severity)
			risks.WriteString(fmt.Sprintf(
				`<div class="risk %s"><div class="risk-h"><span class="tag">%s</span><strong>%s</strong></div><div class="risk-b">%s</div><div class="risk-m">客户 %s · 负责人 %s · 截止 %s</div></div>`,
				sev, esc(riskTypeCN(r.Type)), esc(r.Title), esc(r.Summary),
				esc(emptyDash(r.CustomerName)), esc(emptyDash(r.OwnerName)), due,
			))
		}
		risks.WriteString(`</div>`)
	}

	var actions strings.Builder
	actions.WriteString("<ol>")
	for _, a := range snap.SuggestedActions {
		actions.WriteString("<li>" + esc(a) + "</li>")
	}
	actions.WriteString("</ol>")

	srcLabel := "台账模板"
	if source == "hybrid" {
		srcLabel = "事实 + LLM 润色"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="UTF-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>CacOps 运营周报</title>
<style>
:root{--bg:#f3f6fa;--card:#fff;--text:#1f2937;--muted:#6b7280;--line:#e5e7eb;--primary:#0f4c81;--warn:#b45309;--danger:#b91c1c;}
*{box-sizing:border-box}body{margin:0;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;background:var(--bg);color:var(--text);}
.wrap{width:100%%;max-width:none;margin:0;padding:20px 24px 40px}
.hero{background:linear-gradient(135deg,#0f4c81,#1d6fa5);color:#fff;border-radius:16px;padding:28px 24px;margin-bottom:18px}
.hero h1{margin:0 0 8px;font-size:1.6rem}.hero p{margin:0;opacity:.92}
.meta{margin-top:12px;font-size:.85rem;opacity:.85}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px;margin-bottom:16px}
@media(max-width:800px){.grid{grid-template-columns:repeat(2,1fr)}}
.kpi{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:14px 12px}
.kpi .n{font-size:1.6rem;font-weight:700;color:var(--primary)}.kpi .l{color:var(--muted);font-size:.85rem;margin-top:4px}
.kpi.warn .n{color:var(--warn)}.kpi.danger .n{color:var(--danger)}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:18px 16px;margin-bottom:14px;width:100%%}
.card h2{margin:0 0 12px;font-size:1.05rem;color:var(--primary)}
.exec{font-size:1.05rem;line-height:1.7;background:#eef5fb;border-left:4px solid var(--primary);padding:12px 14px;border-radius:0 8px 8px 0;white-space:pre-wrap;word-break:break-word}
table{width:100%%;border-collapse:collapse;font-size:.92rem}th,td{border-bottom:1px solid var(--line);padding:8px;text-align:left}th{color:var(--muted);font-weight:600}
.risk-list{display:flex;flex-direction:column;gap:10px}
.risk{border:1px solid var(--line);border-radius:10px;padding:12px}
.risk.high{border-color:#fecaca;background:#fff7f7}.risk.medium{border-color:#fde68a;background:#fffbeb}
.risk-h{display:flex;gap:8px;align-items:center;margin-bottom:6px}.tag{font-size:.75rem;background:#e5e7eb;padding:2px 8px;border-radius:999px}
.risk-b{font-size:.92rem}.risk-m{margin-top:6px;color:var(--muted);font-size:.8rem}
ol,ul{margin:0;padding-left:1.2rem;line-height:1.7}.empty{color:var(--muted)}
.foot{color:var(--muted);font-size:.8rem;text-align:center;margin-top:18px}
</style></head><body><div class="wrap">
<div class="hero">
  <h1>CacOps 运营周报</h1>
  <p>管理者决策视图 · 数字来自台账，建议可督办</p>
  <div class="meta">周期 %s ~ %s（近 %d 天） · %s · 生成于 %s</div>
</div>
<div class="grid">%s</div>
<div class="card"><h2>一句话总览</h2><div class="exec">%s</div></div>
<div class="card"><h2>关键事实</h2>%s</div>
<div class="card"><h2>漏斗快照</h2>%s</div>
<div class="card"><h2>重点风险</h2>%s</div>
<div class="card"><h2>建议督办</h2>%s</div>
<div class="foot">本报告由 CacOps 自动生成，仅供内部管理参考</div>
</div></body></html>`,
		start.Format("2006-01-02"), end.Format("2006-01-02"), days, esc(srcLabel), time.Now().Format("2006-01-02 15:04"),
		cards.String(), esc(executive), facts.String(), funnel.String(), risks.String(), actions.String(),
	)
}

func llmConfigured() bool {
	if strings.TrimSpace(os.Getenv("LLM_API_KEY")) != "" {
		return true
	}
	return strings.TrimSpace(viper.GetString("external.llm.api_key")) != "" ||
		strings.TrimSpace(viper.GetString("llm.api_key")) != ""
}

func riskTypeCN(t string) string {
	m := map[string]string{
		model.OpsRiskTrialExpiring:     "试用到期",
		model.OpsRiskContractExpiring:  "合同到期",
		model.OpsRiskSettlementDueSoon: "结算将到期",
		model.OpsRiskSettlementOverdue: "结算逾期",
		model.OpsRiskSurveyLowScore:    "低满意度",
		model.OpsRiskNonRenewalPending: "不续费待闭环",
	}
	if v, ok := m[t]; ok {
		return v
	}
	return t
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func (s *opsManagerReportService) polishExecutiveSummary(ctx context.Context, snap model.OpsManagerSnapshot, days int) (string, error) {
	apiKey, baseURL, model, err := resolveLLMConfig()
	if err != nil {
		return "", err
	}
	facts, _ := json.Marshal(map[string]any{
		"days": days, "kpis": snap.KPIs, "risk_summary": snap.RiskSummary,
		"fact_bullets": snap.FactBullets, "suggested_actions": snap.SuggestedActions,
	})
	prompt := "你是 CacOps 运营分析助手。根据下列 JSON 事实，用中文写 2～4 句管理者周报「一句话总览」。只基于给定数字，禁止编造。语气冷静、可决策。\n" + string(facts)
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "只输出总览正文，不要标题和列表。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := s.httpClient.Do(req)
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
		return "", fmt.Errorf("empty")
	}
	return sanitizeExecutiveSummary(parsed.Choices[0].Message.Content), nil
}

func sanitizeExecutiveSummary(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// MiniMax 等模型可能夹带 thinking / think 标签，展示前剥离
	for _, tag := range []string{"thinking", "think", "reasoning"} {
		reOpen := "<" + tag + ">"
		reClose := "</" + tag + ">"
		for {
			start := strings.Index(strings.ToLower(s), reOpen)
			if start < 0 {
				break
			}
			end := strings.Index(strings.ToLower(s[start:]), reClose)
			if end < 0 {
				s = strings.TrimSpace(s[:start])
				break
			}
			s = strings.TrimSpace(s[:start] + s[start+end+len(reClose):])
		}
	}
	return strings.TrimSpace(s)
}
