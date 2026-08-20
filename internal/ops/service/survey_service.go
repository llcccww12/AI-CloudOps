package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type OpsSurveyService interface {
	EnsureDefaults(ctx context.Context) error
	ListSurveys(ctx context.Context) (*model.ListResp[*model.OpsSurvey], error)
	Submit(ctx context.Context, req *model.SubmitOpsSurveyReq) error
	ListResponses(ctx context.Context, req *model.ListOpsSurveyResponseReq) (*model.ListResp[*model.OpsSurveyResponse], error)
	HasNonRenewal(ctx context.Context, customerID int) (bool, error)
	CreateInvite(ctx context.Context, req *model.CreateOpsSurveyInviteReq) (*model.CreateOpsSurveyInviteResp, error)
	ListInvites(ctx context.Context, req *model.ListOpsSurveyInviteReq) (*model.ListResp[*model.OpsSurveyInvite], error)
	GetPublicMeta(ctx context.Context, token string) (*model.PublicSurveyMeta, error)
	SubmitPublic(ctx context.Context, req *model.SubmitPublicSurveyReq) error
}

type opsSurveyService struct {
	surveyDAO   dao.OpsSurveyDAO
	customerDAO dao.OpsCustomerDAO
	logger      *zap.Logger
}

func NewOpsSurveyService(surveyDAO dao.OpsSurveyDAO, customerDAO dao.OpsCustomerDAO, logger *zap.Logger) OpsSurveyService {
	return &opsSurveyService{surveyDAO: surveyDAO, customerDAO: customerDAO, logger: logger}
}

func (s *opsSurveyService) EnsureDefaults(ctx context.Context) error {
	return s.surveyDAO.EnsureDefaults(ctx)
}

func (s *opsSurveyService) ListSurveys(ctx context.Context) (*model.ListResp[*model.OpsSurvey], error) {
	_ = s.surveyDAO.EnsureDefaults(ctx)
	items, err := s.surveyDAO.List(ctx)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsSurvey]{Items: items, Total: int64(len(items))}, nil
}

func (s *opsSurveyService) Submit(ctx context.Context, req *model.SubmitOpsSurveyReq) error {
	if _, err := s.customerDAO.GetByID(ctx, req.CustomerID); err != nil {
		return err
	}
	survey, err := s.surveyDAO.GetByCode(ctx, strings.TrimSpace(req.SurveyCode))
	if err != nil {
		return err
	}
	now := time.Now()
	return s.surveyDAO.CreateResponse(ctx, &model.OpsSurveyResponse{
		SurveyID: survey.ID, CustomerID: req.CustomerID, SurveyType: survey.SurveyType,
		Answers: req.Answers, Score: req.Score, SubmittedAt: &now,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	})
}

func (s *opsSurveyService) ListResponses(ctx context.Context, req *model.ListOpsSurveyResponseReq) (*model.ListResp[*model.OpsSurveyResponse], error) {
	items, total, err := s.surveyDAO.ListResponses(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsSurveyResponse]{Items: items, Total: total}, nil
}

func (s *opsSurveyService) HasNonRenewal(ctx context.Context, customerID int) (bool, error) {
	ok, err := s.surveyDAO.HasResponse(ctx, customerID, model.OpsSurveyTypeNonRenewal)
	if err != nil {
		return false, fmt.Errorf("查询不续费问卷失败: %w", err)
	}
	return ok, nil
}

func (s *opsSurveyService) CreateInvite(ctx context.Context, req *model.CreateOpsSurveyInviteReq) (*model.CreateOpsSurveyInviteResp, error) {
	_ = s.surveyDAO.EnsureDefaults(ctx)
	customer, err := s.customerDAO.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}
	survey, err := s.surveyDAO.GetByCode(ctx, strings.TrimSpace(req.SurveyCode))
	if err != nil {
		return nil, err
	}
	if survey.Status != model.OpsSurveyStatusActive {
		return nil, fmt.Errorf("问卷未启用")
	}
	days := req.ExpireDays
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	expireAt := time.Now().AddDate(0, 0, days)
	token := strings.ReplaceAll(uuid.NewString(), "-", "")
	invite := &model.OpsSurveyInvite{
		Token: token, SurveyCode: survey.Code, SurveyType: survey.SurveyType,
		CustomerID: customer.ID, CustomerName: customer.Name, ExpireAt: &expireAt,
		Status: model.OpsSurveyInviteStatusOpen,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	}
	if err := s.surveyDAO.CreateInvite(ctx, invite); err != nil {
		return nil, fmt.Errorf("创建问卷链接失败: %w", err)
	}
	path := "/public/survey?token=" + token
	return &model.CreateOpsSurveyInviteResp{
		Token: token, Path: path, URL: publicSurveyURL(path),
		SurveyCode: survey.Code, SurveyTitle: survey.Title,
		CustomerID: customer.ID, CustomerName: customer.Name,
		ExpireAt: expireAt.Format(time.RFC3339),
	}, nil
}

func publicSurveyURL(path string) string {
	base := strings.TrimRight(strings.TrimSpace(viper.GetString("ops.public_base_url")), "/")
	if base == "" {
		port := strings.TrimSpace(viper.GetString("server.port"))
		if port == "" {
			port = "8889"
		}
		base = "http://127.0.0.1:" + port
	}
	return base + path
}

func (s *opsSurveyService) ListInvites(ctx context.Context, req *model.ListOpsSurveyInviteReq) (*model.ListResp[*model.OpsSurveyInvite], error) {
	items, total, err := s.surveyDAO.ListInvites(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsSurveyInvite]{Items: items, Total: total}, nil
}

func (s *opsSurveyService) loadOpenInvite(ctx context.Context, token string) (*model.OpsSurveyInvite, *model.OpsSurvey, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil, fmt.Errorf("缺少问卷令牌")
	}
	invite, err := s.surveyDAO.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	if invite.Status != model.OpsSurveyInviteStatusOpen {
		return nil, nil, fmt.Errorf("该问卷链接已使用")
	}
	if invite.ExpireAt != nil && time.Now().After(*invite.ExpireAt) {
		return nil, nil, fmt.Errorf("该问卷链接已过期")
	}
	survey, err := s.surveyDAO.GetByCode(ctx, invite.SurveyCode)
	if err != nil {
		return nil, nil, err
	}
	return invite, survey, nil
}

func (s *opsSurveyService) GetPublicMeta(ctx context.Context, token string) (*model.PublicSurveyMeta, error) {
	invite, survey, err := s.loadOpenInvite(ctx, token)
	if err != nil {
		return nil, err
	}
	meta := &model.PublicSurveyMeta{
		Token: invite.Token, SurveyCode: survey.Code, SurveyTitle: survey.Title,
		SurveyType: survey.SurveyType, CustomerName: invite.CustomerName, Questions: survey.Questions,
	}
	if invite.ExpireAt != nil {
		meta.ExpireAt = invite.ExpireAt.Format(time.RFC3339)
	}
	return meta, nil
}

func (s *opsSurveyService) SubmitPublic(ctx context.Context, req *model.SubmitPublicSurveyReq) error {
	invite, survey, err := s.loadOpenInvite(ctx, req.Token)
	if err != nil {
		return err
	}
	if req.Answers == nil || len(req.Answers) == 0 {
		return fmt.Errorf("请填写问卷内容")
	}
	now := time.Now()
	if err := s.surveyDAO.CreateResponse(ctx, &model.OpsSurveyResponse{
		SurveyID: survey.ID, CustomerID: invite.CustomerID, SurveyType: survey.SurveyType,
		Answers: req.Answers, Score: req.Score, SubmittedAt: &now,
		OperatorID: 0, OperatorName: "customer",
	}); err != nil {
		return fmt.Errorf("提交失败: %w", err)
	}
	if err := s.surveyDAO.MarkInviteUsed(ctx, invite.ID, now); err != nil {
		s.logger.Warn("标记问卷邀请已使用失败", zap.Int("invite_id", invite.ID), zap.Error(err))
	}
	return nil
}
