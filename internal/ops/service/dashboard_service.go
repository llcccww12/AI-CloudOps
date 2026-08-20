package service

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type OpsDashboardService interface {
	Overview(ctx context.Context, req *model.OpsDashboardOverviewReq) (*model.OpsDashboardOverview, error)
}

type opsDashboardService struct {
	exhibitionDAO dao.OpsExhibitionDAO
	visitDAO      dao.OpsVisitDAO
	customerDAO   dao.OpsCustomerDAO
	logger        *zap.Logger
}

func NewOpsDashboardService(
	exhibitionDAO dao.OpsExhibitionDAO,
	visitDAO dao.OpsVisitDAO,
	customerDAO dao.OpsCustomerDAO,
	logger *zap.Logger,
) OpsDashboardService {
	return &opsDashboardService{
		exhibitionDAO: exhibitionDAO,
		visitDAO:      visitDAO,
		customerDAO:   customerDAO,
		logger:        logger,
	}
}

func (s *opsDashboardService) Overview(ctx context.Context, req *model.OpsDashboardOverviewReq) (*model.OpsDashboardOverview, error) {
	days := 7
	if req != nil && req.Days > 0 && req.Days <= 90 {
		days = req.Days
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := todayStart.Add(24 * time.Hour)
	weekStart := todayStart.AddDate(0, 0, -6)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	trendStart := todayStart.AddDate(0, 0, -(days - 1))

	out := &model.OpsDashboardOverview{
		Funnel:          make([]model.OpsDashboardFunnelStage, 0, 6),
		ExhibitionTrend: make([]model.OpsDashboardTrendPoint, 0, days),
		IntentDist:      []model.OpsDashboardIntentItem{},
		VisitAlerts:     []model.OpsDashboardVisitAlert{},
		RecentEvents:    []model.OpsDashboardEvent{},
	}

	var (
		exToday, exWeek, exAll, exTransferred int64
		visitAll, visitPending, visitDueSoon, visitOverdue int64
		custIntent, custTrial, custFormal, custClosedMonth int64
		trend []model.OpsDashboardTrendPoint
		intentDist []model.OpsDashboardIntentItem
		alerts []*model.OpsVisit
		highIntent []*model.OpsVisit
		recentEx []*model.OpsExhibition
		recentVisits []*model.OpsVisit
		recentCustomers []*model.OpsCustomer
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		exToday, err = s.exhibitionDAO.CountCreatedBetween(gctx, todayStart, tomorrow)
		return err
	})
	g.Go(func() error {
		var err error
		exWeek, err = s.exhibitionDAO.CountCreatedBetween(gctx, weekStart, tomorrow)
		return err
	})
	g.Go(func() error {
		var err error
		exAll, err = s.exhibitionDAO.CountAll(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		exTransferred, err = s.exhibitionDAO.CountByStatus(gctx, model.OpsExhibitionStatusTransferred)
		return err
	})
	g.Go(func() error {
		var err error
		visitAll, err = s.visitDAO.CountAll(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		visitPending, err = s.visitDAO.CountByStatuses(gctx, []string{
			model.OpsVisitStatusPending, model.OpsVisitStatusBooked, model.OpsVisitStatusPlanned,
		})
		return err
	})
	g.Go(func() error {
		var err error
		visitDueSoon, err = s.visitDAO.CountDueSoon(gctx, model.OpsVisitPreDueDays)
		return err
	})
	g.Go(func() error {
		var err error
		visitOverdue, err = s.visitDAO.CountOverdue(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		custIntent, err = s.customerDAO.CountByStage(gctx, model.OpsCustomerStageIntent)
		return err
	})
	g.Go(func() error {
		var err error
		custTrial, err = s.customerDAO.CountByStage(gctx, model.OpsCustomerStageTrial)
		return err
	})
	g.Go(func() error {
		var err error
		custFormal, err = s.customerDAO.CountByStage(gctx, model.OpsCustomerStageFormal)
		return err
	})
	g.Go(func() error {
		var err error
		custClosedMonth, err = s.customerDAO.CountClosedBetween(gctx, monthStart, tomorrow)
		return err
	})
	g.Go(func() error {
		var err error
		trend, err = s.exhibitionDAO.CountDailyCreated(gctx, trendStart, tomorrow)
		return err
	})
	g.Go(func() error {
		var err error
		intentDist, err = s.exhibitionDAO.GroupByIntent(gctx)
		return err
	})
	g.Go(func() error {
		var err error
		alerts, err = s.visitDAO.ListAlerts(gctx, model.OpsVisitPreDueDays, 12)
		return err
	})
	g.Go(func() error {
		var err error
		highIntent, err = s.visitDAO.ListHighIntentPending(gctx, 8)
		return err
	})
	g.Go(func() error {
		var err error
		recentEx, err = s.exhibitionDAO.ListRecentTransferred(gctx, 8)
		return err
	})
	g.Go(func() error {
		var err error
		recentVisits, err = s.visitDAO.ListRecentConverted(gctx, 8)
		return err
	})
	g.Go(func() error {
		var err error
		recentCustomers, err = s.customerDAO.ListRecentByStages(gctx, []string{
			model.OpsCustomerStageTrial, model.OpsCustomerStageFormal, model.OpsCustomerStageClosed,
		}, 8)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("加载运营看板失败: %w", err)
	}

	out.KPIs = model.OpsDashboardKPIs{
		ExhibitionToday:     exToday,
		ExhibitionWeek:      exWeek,
		VisitPending:        visitPending,
		VisitDueSoon:        visitDueSoon,
		VisitOverdue:        visitOverdue,
		CustomerIntent:      custIntent,
		CustomerTrial:       custTrial,
		CustomerFormal:      custFormal,
		CustomerClosedMonth: custClosedMonth,
	}

	// 漏斗：展厅总量 → 已转外访展厅 → 外访总量 → 意向 → 试用 → 正式
	stages := []struct {
		key, label string
		count      int64
	}{
		{"exhibition", "展厅接待", exAll},
		{"transferred", "转入外访", exTransferred},
		{"visit", "外访交流", visitAll},
		{"intent", "意向客户", custIntent},
		{"trial", "试用", custTrial},
		{"formal", "正式", custFormal},
	}
	var prev int64
	for i, st := range stages {
		rate := 100.0
		if i > 0 && prev > 0 {
			rate = float64(st.count) * 100 / float64(prev)
			if rate > 100 {
				rate = 100
			}
		} else if i > 0 {
			rate = 0
		}
		out.Funnel = append(out.Funnel, model.OpsDashboardFunnelStage{
			Key: st.key, Label: st.label, Count: st.count, Rate: round1(rate),
		})
		prev = st.count
	}

	out.ExhibitionTrend = fillTrendDays(trendStart, days, trend)
	if intentDist != nil {
		out.IntentDist = intentDist
	}
	out.VisitAlerts = buildVisitAlerts(alerts, highIntent, now)
	out.RecentEvents = buildRecentEvents(recentEx, recentVisits, recentCustomers)
	return out, nil
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}

func fillTrendDays(start time.Time, days int, raw []model.OpsDashboardTrendPoint) []model.OpsDashboardTrendPoint {
	m := make(map[string]int64, len(raw))
	for _, p := range raw {
		m[p.Date] = p.Count
	}
	out := make([]model.OpsDashboardTrendPoint, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		out = append(out, model.OpsDashboardTrendPoint{Date: d, Count: m[d]})
	}
	return out
}

func buildVisitAlerts(dueList, highIntent []*model.OpsVisit, now time.Time) []model.OpsDashboardVisitAlert {
	seen := map[int]bool{}
	out := make([]model.OpsDashboardVisitAlert, 0, len(dueList)+len(highIntent))
	appendAlert := func(v *model.OpsVisit, alertType string) {
		if v == nil || seen[v.ID] {
			return
		}
		seen[v.ID] = true
		due := ""
		if v.DueAt != nil {
			due = v.DueAt.Format(time.RFC3339)
			if alertType == "" {
				if v.DueAt.Before(now) {
					alertType = "overdue"
				} else {
					alertType = "due_soon"
				}
			}
		}
		if alertType == "" {
			alertType = "high_intent"
		}
		out = append(out, model.OpsDashboardVisitAlert{
			ID: v.ID, Title: v.Title, TargetOrg: v.TargetOrg,
			FollowOwnerName: v.FollowOwnerName, DueAt: due,
			Intent: v.Intent, Status: v.Status, AlertType: alertType,
		})
	}
	for _, v := range dueList {
		appendAlert(v, "")
	}
	for _, v := range highIntent {
		appendAlert(v, "high_intent")
	}
	return out
}

func buildRecentEvents(ex []*model.OpsExhibition, visits []*model.OpsVisit, customers []*model.OpsCustomer) []model.OpsDashboardEvent {
	out := make([]model.OpsDashboardEvent, 0, 20)
	for _, e := range ex {
		if e == nil {
			continue
		}
		out = append(out, model.OpsDashboardEvent{
			ID: e.ID, Type: "exhibition_transfer",
			Title:   fmt.Sprintf("展厅「%s」已转入外访", e.CompanyName),
			Time:    e.UpdatedAt.Format(time.RFC3339),
			RefID:   e.ID,
			RefPath: "/ops/exhibitions",
		})
	}
	for _, v := range visits {
		if v == nil {
			continue
		}
		name := v.TargetOrg
		if name == "" {
			name = v.Title
		}
		out = append(out, model.OpsDashboardEvent{
			ID: v.ID, Type: "visit_convert",
			Title:   fmt.Sprintf("外访「%s」已转客户", name),
			Time:    v.UpdatedAt.Format(time.RFC3339),
			RefID:   v.ID,
			RefPath: "/ops/visits",
		})
	}
	stageLabel := map[string]string{
		model.OpsCustomerStageTrial: "试用", model.OpsCustomerStageFormal: "正式",
		model.OpsCustomerStageClosed: "闭环", model.OpsCustomerStageIntent: "意向",
	}
	for _, c := range customers {
		if c == nil {
			continue
		}
		label := stageLabel[c.Stage]
		if label == "" {
			label = c.Stage
		}
		out = append(out, model.OpsDashboardEvent{
			ID: c.ID, Type: "customer_stage",
			Title:   fmt.Sprintf("客户「%s」处于%s阶段", c.Name, label),
			Time:    c.UpdatedAt.Format(time.RFC3339),
			RefID:   c.ID,
			RefPath: fmt.Sprintf("/ops/customers/detail/%d", c.ID),
		})
	}
	// 简单按时间倒序截断
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Time > out[i].Time {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > 15 {
		out = out[:15]
	}
	return out
}
