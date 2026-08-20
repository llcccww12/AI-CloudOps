package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	opsNotify "github.com/GoSimplicity/AI-CloudOps/internal/ops/notify"
	systemDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	workorderDao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
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

	ListTasks(ctx context.Context, req *model.ListOpsReminderTaskReq) (*model.ListResp[*model.OpsReminderTask], error)
	CreateTask(ctx context.Context, req *model.CreateOpsReminderTaskReq) error
	UpdateTask(ctx context.Context, req *model.UpdateOpsReminderTaskReq) error
	DeleteTask(ctx context.Context, id int) error

	ListDeliveries(ctx context.Context, req *model.ListOpsReminderDeliveryReq) (*model.ListResp[*model.OpsReminderDelivery], error)
}

type opsReminderService struct {
	ruleDAO       dao.OpsReminderRuleDAO
	taskDAO       dao.OpsReminderTaskDAO
	deliveryDAO   dao.OpsReminderDeliveryDAO
	trialDAO      dao.OpsTrialDAO
	contractDAO   dao.OpsContractDAO
	settlementDAO dao.OpsSettlementDAO
	customerDAO   dao.OpsCustomerDAO
	visitDAO      dao.OpsVisitDAO
	inboxDAO      workorderDao.WorkorderInboxDAO
	userDAO       systemDao.UserDAO
	notifyMgr     *notification.Manager
	smsSender     opsNotify.SMSSender
	bizSvc        OpsBizService
	logger        *zap.Logger
}

func NewOpsReminderService(
	ruleDAO dao.OpsReminderRuleDAO,
	taskDAO dao.OpsReminderTaskDAO,
	deliveryDAO dao.OpsReminderDeliveryDAO,
	trialDAO dao.OpsTrialDAO,
	contractDAO dao.OpsContractDAO,
	settlementDAO dao.OpsSettlementDAO,
	customerDAO dao.OpsCustomerDAO,
	visitDAO dao.OpsVisitDAO,
	inboxDAO workorderDao.WorkorderInboxDAO,
	userDAO systemDao.UserDAO,
	notifyMgr *notification.Manager,
	bizSvc OpsBizService,
	logger *zap.Logger,
) OpsReminderService {
	return &opsReminderService{
		ruleDAO: ruleDAO, taskDAO: taskDAO, deliveryDAO: deliveryDAO,
		trialDAO: trialDAO, contractDAO: contractDAO, settlementDAO: settlementDAO,
		customerDAO: customerDAO, visitDAO: visitDAO, inboxDAO: inboxDAO,
		userDAO: userDAO, notifyMgr: notifyMgr, smsSender: opsNotify.NewAliyunSMSSender(logger),
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
		channels = model.StringList{model.OpsReminderChannelInbox}
	}
	return s.ruleDAO.Create(ctx, &model.OpsReminderRule{
		Scene: req.Scene, Name: strings.TrimSpace(req.Name), AdvanceDays: req.AdvanceDays,
		Enabled: enabled, Channels: channels, ExtraUserIDs: req.ExtraUserIDs, Remark: req.Remark,
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
	if req.ExtraUserIDs != nil {
		rule.ExtraUserIDs = req.ExtraUserIDs
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

func (s *opsReminderService) ListTasks(ctx context.Context, req *model.ListOpsReminderTaskReq) (*model.ListResp[*model.OpsReminderTask], error) {
	items, total, err := s.taskDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsReminderTask]{Items: items, Total: total}, nil
}

func (s *opsReminderService) CreateTask(ctx context.Context, req *model.CreateOpsReminderTaskReq) error {
	channels := req.Channels
	if len(channels) == 0 {
		channels = model.StringList{model.OpsReminderChannelInbox}
	}
	return s.taskDAO.Create(ctx, &model.OpsReminderTask{
		Title: strings.TrimSpace(req.Title), CustomerID: req.CustomerID,
		BizType: req.BizType, BizID: req.BizID, DueAt: req.DueAt, AdvanceDays: req.AdvanceDays,
		Channels: channels, TargetUserID: req.TargetUserID, ExtraUserIDs: req.ExtraUserIDs,
		Content: strings.TrimSpace(req.Content), Status: model.OpsReminderTaskPending,
		CreatorID: req.CreatorID, CreatorName: req.CreatorName,
	})
}

func (s *opsReminderService) UpdateTask(ctx context.Context, req *model.UpdateOpsReminderTaskReq) error {
	task, err := s.taskDAO.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.DueAt != nil {
		task.DueAt = *req.DueAt
	}
	if req.AdvanceDays != nil {
		task.AdvanceDays = *req.AdvanceDays
	}
	if req.Channels != nil {
		task.Channels = req.Channels
	}
	if req.TargetUserID > 0 {
		task.TargetUserID = req.TargetUserID
	}
	if req.ExtraUserIDs != nil {
		task.ExtraUserIDs = req.ExtraUserIDs
	}
	if req.Content != "" {
		task.Content = req.Content
	}
	if req.Status != "" {
		task.Status = req.Status
	}
	return s.taskDAO.Update(ctx, task)
}

func (s *opsReminderService) DeleteTask(ctx context.Context, id int) error {
	return s.taskDAO.Delete(ctx, id)
}

func (s *opsReminderService) ListDeliveries(ctx context.Context, req *model.ListOpsReminderDeliveryReq) (*model.ListResp[*model.OpsReminderDelivery], error) {
	items, total, err := s.deliveryDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsReminderDelivery]{Items: items, Total: total}, nil
}

func (s *opsReminderService) RunScan(ctx context.Context) (*model.OpsReminderScanResult, error) {
	_ = s.bizSvc.SyncPendingApprovals(ctx)
	hits, err := s.collectHits(ctx)
	if err != nil {
		return nil, err
	}
	result := &model.OpsReminderScanResult{HitCount: len(hits)}
	day := time.Now().Format("2006-01-02")
	for _, hit := range hits {
		recipients := uniquePositiveIDs(append([]int{hit.TargetUserID}, hit.ExtraUserIDs...))
		if len(recipients) == 0 {
			result.SkippedNoOwner++
			continue
		}
		channels := hit.Channels
		if len(channels) == 0 {
			channels = []string{model.OpsReminderChannelInbox}
		}
		title := hit.RuleName
		if title == "" {
			title = "运营提醒"
		}
		notified := false
		for _, uid := range recipients {
			for _, ch := range channels {
				ok, _ := s.dispatchOne(ctx, hit, uid, ch, title, day)
				if ok {
					result.DeliveryCount++
					notified = true
				}
			}
		}
		if notified {
			result.NotifyCount++
			if hit.Scene == model.OpsReminderSceneVisitPreDue && hit.BizID > 0 {
				_ = s.visitDAO.MarkPreDueReminded(ctx, hit.BizID)
			}
			if hit.SourceType == "task" && hit.TaskID > 0 {
				_ = s.taskDAO.MarkSent(ctx, hit.TaskID, time.Now())
			}
		}
	}
	return result, nil
}

func (s *opsReminderService) ScanAndNotify(ctx context.Context) error {
	_, err := s.RunScan(ctx)
	return err
}

func (s *opsReminderService) dispatchOne(
	ctx context.Context,
	hit *model.OpsReminderHit,
	userID int,
	channel, title, day string,
) (bool, error) {
	sourceType := hit.SourceType
	if sourceType == "" {
		sourceType = "rule"
	}
	sourceID := hit.RuleID
	if sourceType == "task" {
		sourceID = hit.TaskID
	}
	dedupe := dao.BuildReminderDedupeKey(sourceType, sourceID, userID, channel, day, hit.BizType, hit.BizID)
	exists, err := s.deliveryDAO.ExistsByDedupeKey(ctx, dedupe)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	delivery := &model.OpsReminderDelivery{
		SourceType: sourceType, SourceID: sourceID, Scene: hit.Scene,
		BizType: hit.BizType, BizID: hit.BizID, CustomerID: hit.CustomerID,
		TargetUserID: userID, Channel: channel, Title: title, Content: hit.Reason,
		DedupeKey: dedupe, Status: model.OpsReminderDeliveryPending,
	}

	sendErr := s.sendChannel(ctx, userID, channel, title, hit.Reason, hit.Link)
	if sendErr != nil {
		if isChannelSkip(sendErr) {
			delivery.Status = model.OpsReminderDeliverySkipped
		} else {
			delivery.Status = model.OpsReminderDeliveryFailed
		}
		delivery.ErrorMsg = truncateErr(sendErr.Error(), 480)
		_ = s.deliveryDAO.Create(ctx, delivery)
		return false, sendErr
	}
	delivery.Status = model.OpsReminderDeliverySuccess
	_ = s.deliveryDAO.Create(ctx, delivery)
	return true, nil
}

func (s *opsReminderService) sendChannel(ctx context.Context, userID int, channel, title, content, link string) error {
	body := content
	if link != "" {
		body = content + " → 打开: " + link
	}
	switch channel {
	case model.OpsReminderChannelInbox, "":
		return s.pushInbox(ctx, userID, title, body)
	case model.OpsReminderChannelFeishu:
		return s.sendFeishu(ctx, userID, title, body)
	case model.OpsReminderChannelEmail:
		return s.sendEmail(ctx, userID, title, body)
	case model.OpsReminderChannelSMS:
		return s.sendSMS(ctx, userID, body)
	default:
		return fmt.Errorf("不支持的渠道: %s", channel)
	}
}

func (s *opsReminderService) pushInbox(ctx context.Context, userID int, title, content string) error {
	if userID <= 0 || s.inboxDAO == nil {
		return fmt.Errorf("跳过: 无接收人或站内信不可用")
	}
	return s.inboxDAO.Create(ctx, &model.WorkorderInboxMessage{
		UserID: userID, Title: "[运营] " + title, Content: content,
		EventType: "ops_reminder", IsRead: model.InboxMessageUnread,
	})
}

func (s *opsReminderService) sendFeishu(ctx context.Context, userID int, title, content string) error {
	if s.notifyMgr == nil {
		return fmt.Errorf("跳过: 飞书通道未启用")
	}
	user, err := s.userDAO.GetByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("跳过: 用户不存在")
	}
	if strings.TrimSpace(user.FeiShuUserId) == "" {
		return fmt.Errorf("跳过: 用户未绑定飞书ID")
	}
	_, err = s.notifyMgr.SendNotification(ctx, &notification.SendRequest{
		Subject: title, Content: content, RecipientType: "feishu_user",
		RecipientID: strconv.Itoa(userID), RecipientAddr: user.FeiShuUserId,
		RecipientName: user.Username, EventType: "ops_reminder",
	})
	return err
}

func (s *opsReminderService) sendEmail(ctx context.Context, userID int, title, content string) error {
	if s.notifyMgr == nil {
		return fmt.Errorf("跳过: 邮件通道未启用")
	}
	user, err := s.userDAO.GetByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("跳过: 用户不存在")
	}
	if strings.TrimSpace(user.Email) == "" {
		return fmt.Errorf("跳过: 用户未配置邮箱")
	}
	_, err = s.notifyMgr.SendNotification(ctx, &notification.SendRequest{
		Subject: title, Content: content, RecipientType: "email",
		RecipientID: strconv.Itoa(userID), RecipientAddr: user.Email,
		RecipientName: user.Username, EventType: "ops_reminder",
	})
	return err
}

func (s *opsReminderService) sendSMS(ctx context.Context, userID int, content string) error {
	if s.smsSender == nil || !s.smsSender.Enabled() {
		return fmt.Errorf("跳过: 短信未配置")
	}
	user, err := s.userDAO.GetByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("跳过: 用户不存在")
	}
	if strings.TrimSpace(user.Mobile) == "" {
		return fmt.Errorf("跳过: 用户未配置手机号")
	}
	return s.smsSender.Send(ctx, user.Mobile, content)
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
		case model.OpsReminderSceneVisitPreDue:
			days := rule.AdvanceDays
			if days <= 0 {
				days = model.OpsVisitPreDueDays
			}
			items, _ := s.visitDAO.ListPreDueRemind(ctx, days)
			for _, v := range items {
				dueText := ""
				if v.DueAt != nil {
					dueText = v.DueAt.Format("2006-01-02")
				}
				channels := []string(rule.Channels)
				hits = append(hits, &model.OpsReminderHit{
					RuleID: rule.ID, Scene: rule.Scene, RuleName: rule.Name, AdvanceDays: days,
					TargetUserID: v.FollowOwnerID, TargetUserHint: v.FollowOwnerName,
					ExtraUserIDs: rule.ExtraUserIDs, Channels: channels,
					BizType: "visit", BizID: v.ID, BizTitle: v.Title, SourceType: "rule",
					Link: "/ops/visits",
					Reason: fmt.Sprintf("外访「%s」将于 %s 到期（任务下发后 %d 天内需完成），请尽快走访",
						v.TargetOrg, dueText, model.OpsVisitAssignDays),
				})
			}
		}
	}

	tasks, _ := s.taskDAO.ListDue(ctx, time.Now())
	for _, task := range tasks {
		customerName := ""
		if task.CustomerID > 0 {
			if c, err := s.customerDAO.GetByID(ctx, task.CustomerID); err == nil && c != nil {
				customerName = c.Name
			}
		}
		reason := task.Content
		if reason == "" {
			reason = fmt.Sprintf("自定义提醒「%s」已到提醒窗口（到期 %s）", task.Title, task.DueAt.Format("2006-01-02"))
		}
		link := "/ops/reminders"
		if task.CustomerID > 0 {
			link = fmt.Sprintf("/ops/customers/detail/%d", task.CustomerID)
		}
		hits = append(hits, &model.OpsReminderHit{
			TaskID: task.ID, Scene: model.OpsReminderSceneCustom, RuleName: task.Title,
			AdvanceDays: task.AdvanceDays, TargetUserID: task.TargetUserID,
			TargetUserHint: fmt.Sprintf("用户#%d", task.TargetUserID),
			ExtraUserIDs: task.ExtraUserIDs, Channels: []string(task.Channels),
			BizType: task.BizType, BizID: task.BizID, BizTitle: task.Title,
			CustomerID: task.CustomerID, CustomerName: customerName,
			Link: link, Reason: reason, SourceType: "task",
		})
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
	channels := []string(rule.Channels)
	return &model.OpsReminderHit{
		RuleID: rule.ID, Scene: rule.Scene, RuleName: rule.Name, AdvanceDays: rule.AdvanceDays,
		TargetUserID: targetID, TargetUserHint: hint, ExtraUserIDs: rule.ExtraUserIDs, Channels: channels,
		BizType: bizType, BizID: bizID, BizTitle: bizTitle,
		CustomerID: customerID, CustomerName: customerName, SourceType: "rule",
		Link: link, Reason: fmt.Sprintf("客户「%s」%s：%s", customerName, reason, bizTitle),
	}
}

func uniquePositiveIDs(ids []int) []int {
	seen := map[int]struct{}{}
	var out []int
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func isChannelSkip(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "跳过:")
}

func truncateErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
