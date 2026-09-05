package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	opsUtils "github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OpsAttachmentService interface {
	Upload(ctx context.Context, bizType string, bizID, operatorID int, header *multipart.FileHeader) (*model.OpsAttachment, error)
	UploadPending(ctx context.Context, bizType string, operatorID int, header *multipart.FileHeader) (*model.OpsAttachment, error)
	List(ctx context.Context, bizType string, bizID int) ([]*model.OpsAttachment, error)
	Download(ctx context.Context, id int) (*model.OpsAttachment, string, error)
	Delete(ctx context.Context, id int) error
	DownloadActivationTemplate(scene string) (absPath, downloadName string, err error)
	ListCustomerEvidence(ctx context.Context, customerID int) ([]*model.OpsCustomerEvidencePack, error)
	ListDeliveryPacks(ctx context.Context, req *model.ListOpsDeliveryPackReq) (*model.ListResp[*model.OpsCustomerEvidencePack], error)
}

type opsAttachmentService struct {
	attachmentDAO dao.OpsAttachmentDAO
	contractDAO   dao.OpsContractDAO
	settlementDAO dao.OpsSettlementDAO
	exhibitionDAO dao.OpsExhibitionDAO
	trialDAO      dao.OpsTrialDAO
	activationDAO dao.OpsActivationDAO
	customerDAO   dao.OpsCustomerDAO
	approvalDAO   dao.OpsApprovalLinkDAO
	logger        *zap.Logger
}

func NewOpsAttachmentService(
	attachmentDAO dao.OpsAttachmentDAO,
	contractDAO dao.OpsContractDAO,
	settlementDAO dao.OpsSettlementDAO,
	exhibitionDAO dao.OpsExhibitionDAO,
	trialDAO dao.OpsTrialDAO,
	activationDAO dao.OpsActivationDAO,
	customerDAO dao.OpsCustomerDAO,
	approvalDAO dao.OpsApprovalLinkDAO,
	logger *zap.Logger,
) OpsAttachmentService {
	return &opsAttachmentService{
		attachmentDAO: attachmentDAO,
		contractDAO:   contractDAO,
		settlementDAO: settlementDAO,
		exhibitionDAO: exhibitionDAO,
		trialDAO:      trialDAO,
		activationDAO: activationDAO,
		customerDAO:   customerDAO,
		approvalDAO:   approvalDAO,
		logger:        logger,
	}
}

func (s *opsAttachmentService) ensureBizExists(ctx context.Context, bizType string, bizID int) error {
	switch bizType {
	case model.OpsAttachmentBizContract:
		_, err := s.contractDAO.GetByID(ctx, bizID)
		return err
	case model.OpsAttachmentBizSettlement:
		_, err := s.settlementDAO.GetByID(ctx, bizID)
		return err
	case model.OpsAttachmentBizExhibition:
		_, err := s.exhibitionDAO.GetByID(ctx, bizID)
		return err
	case model.OpsAttachmentBizTrialSheet, model.OpsAttachmentBizTrialEmail:
		_, err := s.trialDAO.GetByID(ctx, bizID)
		return err
	case model.OpsAttachmentBizActivationSheet, model.OpsAttachmentBizActivationEmail:
		_, err := s.activationDAO.GetByID(ctx, bizID)
		return err
	default:
		return fmt.Errorf("不支持的业务类型: %s", bizType)
	}
}

func (s *opsAttachmentService) Upload(ctx context.Context, bizType string, bizID, operatorID int, header *multipart.FileHeader) (*model.OpsAttachment, error) {
	if header == nil {
		return nil, fmt.Errorf("未选择文件")
	}
	if err := s.ensureBizExists(ctx, bizType, bizID); err != nil {
		return nil, fmt.Errorf("业务单据不存在: %w", err)
	}

	maxCount := opsUtils.GetOpsAttachmentMaxCount()
	count, err := s.attachmentDAO.CountByBiz(ctx, bizType, bizID)
	if err != nil {
		return nil, err
	}
	if count >= int64(maxCount) {
		return nil, fmt.Errorf("附件数量已达上限 %d 个", maxCount)
	}

	return s.saveFile(ctx, bizType, bizID, operatorID, header)
}

// UploadPending 公开登记等场景：先上传后绑定业务 ID（biz_id=0）
func (s *opsAttachmentService) UploadPending(ctx context.Context, bizType string, operatorID int, header *multipart.FileHeader) (*model.OpsAttachment, error) {
	if header == nil {
		return nil, fmt.Errorf("未选择文件")
	}
	switch bizType {
	case model.OpsAttachmentBizExhibition, model.OpsAttachmentBizPublicFault:
	default:
		return nil, fmt.Errorf("不支持的业务类型: %s", bizType)
	}
	return s.saveFile(ctx, bizType, 0, operatorID, header)
}

func (s *opsAttachmentService) saveFile(ctx context.Context, bizType string, bizID, operatorID int, header *multipart.FileHeader) (*model.OpsAttachment, error) {
	maxSize := opsUtils.GetOpsAttachmentMaxSizeBytes()
	if header.Size > maxSize {
		return nil, fmt.Errorf("文件大小不能超过 %dMB", maxSize/1024/1024)
	}

	_, contentType, err := opsUtils.ValidateOpsAttachmentFileName(header.Filename)
	if err != nil {
		return nil, err
	}

	safeName := opsUtils.SanitizeOpsAttachmentFileName(header.Filename)
	storedName := fmt.Sprintf("%s_%s", uuid.NewString(), safeName)
	folder := fmt.Sprintf("%d", bizID)
	if bizID == 0 {
		folder = "pending"
	}
	relPath := filepath.ToSlash(filepath.Join(bizType, folder, storedName))
	absDir := filepath.Join(opsUtils.GetOpsAttachmentDir(), bizType, folder)
	if err := os.MkdirAll(absDir, 0o750); err != nil {
		return nil, fmt.Errorf("创建附件目录失败: %w", err)
	}
	absPath := filepath.Join(absDir, storedName)

	src, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return nil, fmt.Errorf("保存附件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(src, maxSize+1))
	if err != nil {
		_ = os.Remove(absPath)
		return nil, fmt.Errorf("写入附件失败: %w", err)
	}
	if written > maxSize {
		_ = os.Remove(absPath)
		return nil, fmt.Errorf("文件大小不能超过 %dMB", maxSize/1024/1024)
	}

	item := &model.OpsAttachment{
		BizType:     bizType,
		BizID:       bizID,
		OperatorID:  operatorID,
		FileName:    filepath.Base(header.Filename),
		StoredName:  storedName,
		ContentType: contentType,
		Size:        written,
		StoragePath: relPath,
	}
	if err := s.attachmentDAO.Create(ctx, item); err != nil {
		_ = os.Remove(absPath)
		return nil, err
	}
	return item, nil
}

func (s *opsAttachmentService) List(ctx context.Context, bizType string, bizID int) ([]*model.OpsAttachment, error) {
	if err := s.ensureBizExists(ctx, bizType, bizID); err != nil {
		return nil, fmt.Errorf("业务单据不存在: %w", err)
	}
	return s.attachmentDAO.ListByBiz(ctx, bizType, bizID)
}

func (s *opsAttachmentService) Download(ctx context.Context, id int) (*model.OpsAttachment, string, error) {
	item, err := s.attachmentDAO.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	absPath := filepath.Join(opsUtils.GetOpsAttachmentDir(), filepath.FromSlash(item.StoragePath))
	if _, err := os.Stat(absPath); err != nil {
		return nil, "", fmt.Errorf("附件文件不存在")
	}
	return item, absPath, nil
}

func (s *opsAttachmentService) Delete(ctx context.Context, id int) error {
	item, err := s.attachmentDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.attachmentDAO.Delete(ctx, id); err != nil {
		return err
	}
	absPath := filepath.Join(opsUtils.GetOpsAttachmentDir(), filepath.FromSlash(item.StoragePath))
	_ = os.Remove(absPath)
	return nil
}

func (s *opsAttachmentService) DownloadActivationTemplate(scene string) (absPath, downloadName string, err error) {
	absPath = opsUtils.GetOpsActivationTemplatePath()
	if _, err = os.Stat(absPath); err != nil {
		return "", "", fmt.Errorf("开通单模版不存在，请检查 ops.activation_template_path")
	}
	switch strings.TrimSpace(scene) {
	case "formal", "activation":
		downloadName = "思明智算业务开通单v1.0-正式开通.xlsx"
	default:
		downloadName = "思明智算业务开通单v1.0-测试开通.xlsx"
	}
	return absPath, downloadName, nil
}

func (s *opsAttachmentService) listSafe(ctx context.Context, bizType string, bizID int) []*model.OpsAttachment {
	items, err := s.attachmentDAO.ListByBiz(ctx, bizType, bizID)
	if err != nil || items == nil {
		return []*model.OpsAttachment{}
	}
	return items
}

func (s *opsAttachmentService) workorderIDByBiz(ctx context.Context, bizType string, bizID int) int {
	link, err := s.approvalDAO.GetByBiz(ctx, bizType, bizID)
	if err != nil || link == nil {
		return 0
	}
	return link.WorkorderInstanceID
}

func (s *opsAttachmentService) fillPackCounts(p *model.OpsCustomerEvidencePack) {
	if p == nil {
		return
	}
	p.SheetCount = len(p.Sheets)
	p.EmailCount = len(p.Emails)
	p.ContractFileCount = len(p.Contracts)
	if p.Sheets == nil {
		p.Sheets = []*model.OpsAttachment{}
	}
	if p.Emails == nil {
		p.Emails = []*model.OpsAttachment{}
	}
}

func deliveryOpenMethodLabel(method string) string {
	switch method {
	case model.OpsOpenMethodTrial:
		return "测试开通"
	case model.OpsOpenMethodFormal:
		return "正式开通"
	case model.OpsOpenMethodExpand:
		return "扩容开通"
	default:
		return method
	}
}

// filterHollowTrialDuplicatePacks 同一客户若已有实质测试开通归档，则隐藏空的「客户名-试用」占位单。
func filterHollowTrialDuplicatePacks(packs []*model.OpsCustomerEvidencePack) []*model.OpsCustomerEvidencePack {
	hasSolid := map[int]bool{}
	for _, p := range packs {
		if p == nil || p.Scene != "trial" {
			continue
		}
		if !isHollowTrialPack(p) {
			hasSolid[p.CustomerID] = true
		}
	}
	out := make([]*model.OpsCustomerEvidencePack, 0, len(packs))
	for _, p := range packs {
		if p == nil {
			continue
		}
		if p.Scene == "trial" && isHollowTrialPack(p) && hasSolid[p.CustomerID] {
			continue
		}
		out = append(out, p)
	}
	return out
}

func isHollowTrialPack(p *model.OpsCustomerEvidencePack) bool {
	if p == nil {
		return true
	}
	title := strings.TrimSpace(p.Title)
	placeholder := title == "" || title == "测试开通" || strings.HasSuffix(title, "-试用") || strings.HasPrefix(title, "运营测试开通-")
	noLedger := strings.TrimSpace(p.ProductType) == "" && strings.TrimSpace(p.ProjectName) == ""
	return (placeholder || noLedger) &&
		p.SheetCount == 0 &&
		p.EmailCount == 0 &&
		strings.TrimSpace(p.ProjectName) == "" &&
		strings.TrimSpace(p.OrderNo) == "" &&
		strings.TrimSpace(p.MainAccount) == "" &&
		strings.TrimSpace(p.ProductType) == ""
}

func (s *opsAttachmentService) fillTrialLedger(p *model.OpsCustomerEvidencePack, trial *model.OpsTrial, customerName string) {
	if p == nil || trial == nil {
		return
	}
	p.CustomerShortName = trial.CustomerShortName
	if p.CustomerShortName == "" {
		p.CustomerShortName = customerName
	}
	p.ProductType = trial.ProductType
	p.Region = trial.Region
	p.OwnerName = trial.OwnerName
	p.MainAccount = trial.MainAccount
	p.ProjectName = trial.ProjectName
	p.OpenMethod = trial.OpenMethod
	if p.OpenMethod == "" {
		p.OpenMethod = model.OpsOpenMethodTrial
	}
	p.OpenMethodLabel = deliveryOpenMethodLabel(p.OpenMethod)
	p.ContractNo = trial.ContractNo
	p.OrderNo = trial.OrderNo
	p.OpenPeriod = trial.OpenPeriod
	p.ContractStartAt = trial.ContractStartAt
	p.ContractEndAt = trial.ContractEndAt
	p.OperatorName = trial.OperatorName
	p.UpdaterName = trial.UpdaterName
	p.UpdatedAt = trial.UpdatedAt
	if p.Title == "" && trial.ProjectName != "" {
		p.Title = trial.ProjectName
	}
	p.LedgerIncomplete = strings.TrimSpace(p.ProductType) == "" || strings.TrimSpace(p.ProjectName) == ""
}

func (s *opsAttachmentService) fillActivationLedger(ctx context.Context, p *model.OpsCustomerEvidencePack, act *model.OpsActivation, customerName string) {
	if p == nil || act == nil {
		return
	}
	p.CustomerShortName = act.CustomerShortName
	if p.CustomerShortName == "" {
		p.CustomerShortName = customerName
	}
	p.ProductType = act.ProductType
	p.Region = act.Region
	p.OwnerName = act.OwnerName
	p.MainAccount = act.MainAccount
	if p.MainAccount == "" {
		p.MainAccount = act.FeedbackAccount
	}
	p.ProjectName = act.ProjectName
	p.OpenMethod = act.OpenMethod
	if p.OpenMethod == "" {
		p.OpenMethod = model.OpsOpenMethodFormal
	}
	p.OpenMethodLabel = deliveryOpenMethodLabel(p.OpenMethod)
	p.ContractNo = act.ContractNo
	p.OrderNo = act.OrderNo
	p.OpenPeriod = act.OpenPeriod
	p.ContractStartAt = act.ContractStartAt
	p.ContractEndAt = act.ContractEndAt
	p.OperatorName = act.OperatorName
	p.UpdaterName = act.UpdaterName
	p.UpdatedAt = act.UpdatedAt
	if act.ContractID > 0 {
		if c, err := s.contractDAO.GetByID(ctx, act.ContractID); err == nil && c != nil {
			if p.ContractNo == "" {
				p.ContractNo = c.ContractNo
			}
			if p.ProductType == "" {
				p.ProductType = c.ProductType
			}
			if p.ContractStartAt == nil {
				p.ContractStartAt = c.StartAt
			}
			if p.ContractEndAt == nil {
				p.ContractEndAt = c.EndAt
			}
		}
	}
	if p.Title == "" && act.ProjectName != "" {
		p.Title = act.ProjectName
	}
	p.LedgerIncomplete = strings.TrimSpace(p.ProductType) == "" || strings.TrimSpace(p.ProjectName) == ""
}

func (s *opsAttachmentService) ListCustomerEvidence(ctx context.Context, customerID int) ([]*model.OpsCustomerEvidencePack, error) {
	customer, err := s.customerDAO.GetByID(ctx, customerID)
	if err != nil {
		// 客户已删除：不再继续查开通佐证
		return []*model.OpsCustomerEvidencePack{}, nil
	}
	packs, err := s.buildDeliveryPacks(ctx, customerID, customer.Name, "", "")
	if err != nil {
		return nil, err
	}
	return packs, nil
}

func (s *opsAttachmentService) buildDeliveryPacks(ctx context.Context, customerID int, customerName, scene, search string) ([]*model.OpsCustomerEvidencePack, error) {
	packs := make([]*model.OpsCustomerEvidencePack, 0)
	includeTrial := scene == "" || scene == "trial"
	includeFormal := scene == "" || scene == "formal"
	search = strings.TrimSpace(search)

	if includeTrial {
		trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
			CustomerID: customerID, ListReq: model.ListReq{Page: 1, Size: 100, Search: search},
		})
		if err != nil {
			return nil, err
		}
		for _, trial := range trials {
			if trial == nil {
				continue
			}
			p := &model.OpsCustomerEvidencePack{
				Scene: "trial", SceneLabel: "测试开通", BizType: "trial", BizID: trial.ID,
				CustomerID: trial.CustomerID, CustomerName: customerName, Title: trial.Title,
				Status: trial.Status, CreatedAt: trial.CreatedAt,
				WorkorderInstanceID: s.workorderIDByBiz(ctx, model.OpsApprovalBizTrial, trial.ID),
				Sheets: s.listSafe(ctx, model.OpsAttachmentBizTrialSheet, trial.ID),
				Emails: s.listSafe(ctx, model.OpsAttachmentBizTrialEmail, trial.ID),
			}
			s.fillTrialLedger(p, trial, customerName)
			s.fillPackCounts(p)
			packs = append(packs, p)
		}
		packs = filterHollowTrialDuplicatePacks(packs)
	}

	if includeFormal {
		acts, _, err := s.activationDAO.List(ctx, &model.ListOpsActivationReq{
			CustomerID: customerID, ListReq: model.ListReq{Page: 1, Size: 100, Search: search},
		})
		if err != nil {
			return nil, err
		}
		for _, act := range acts {
			if act == nil {
				continue
			}
			p := &model.OpsCustomerEvidencePack{
				Scene: "formal", SceneLabel: "正式开通", BizType: "activation", BizID: act.ID,
				CustomerID: act.CustomerID, CustomerName: customerName, Title: act.Title,
				Status: act.Status, CreatedAt: act.CreatedAt, ContractID: act.ContractID,
				WorkorderInstanceID: s.workorderIDByBiz(ctx, model.OpsApprovalBizActivation, act.ID),
				Sheets:    s.listSafe(ctx, model.OpsAttachmentBizActivationSheet, act.ID),
				Emails:    s.listSafe(ctx, model.OpsAttachmentBizActivationEmail, act.ID),
				Contracts: s.listSafe(ctx, model.OpsAttachmentBizContract, act.ContractID),
			}
			s.fillActivationLedger(ctx, p, act, customerName)
			s.fillPackCounts(p)
			packs = append(packs, p)
		}
	}

	sort.Slice(packs, func(i, j int) bool {
		if packs[i].CreatedAt.Equal(packs[j].CreatedAt) {
			return packs[i].BizID > packs[j].BizID
		}
		return packs[i].CreatedAt.After(packs[j].CreatedAt)
	})
	return packs, nil
}

func (s *opsAttachmentService) ListDeliveryPacks(ctx context.Context, req *model.ListOpsDeliveryPackReq) (*model.ListResp[*model.OpsCustomerEvidencePack], error) {
	if req == nil {
		req = &model.ListOpsDeliveryPackReq{}
	}
	page, size := req.Page, req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	all := make([]*model.OpsCustomerEvidencePack, 0)
	if req.CustomerID > 0 {
		customer, err := s.customerDAO.GetByID(ctx, req.CustomerID)
		if err != nil {
			// 客户已删除：返回空归档，避免前端报错
			return &model.ListResp[*model.OpsCustomerEvidencePack]{Items: all, Total: 0}, nil
		}
		packs, err := s.buildDeliveryPacks(ctx, req.CustomerID, customer.Name, req.Scene, req.Search)
		if err != nil {
			return nil, err
		}
		all = packs
	} else {
		// 全量：分别拉试用/正式列表后补客户名
		if req.Scene == "" || req.Scene == "trial" {
			trials, _, err := s.trialDAO.List(ctx, &model.ListOpsTrialReq{
				ListReq: model.ListReq{Page: 1, Size: 100, Search: req.Search},
			})
			if err != nil {
				return nil, err
			}
			for _, trial := range trials {
				if trial == nil {
					continue
				}
				name := ""
				if c, err := s.customerDAO.GetByID(ctx, trial.CustomerID); err == nil && c != nil {
					name = c.Name
				}
				p := &model.OpsCustomerEvidencePack{
					Scene: "trial", SceneLabel: "测试开通", BizType: "trial", BizID: trial.ID,
					CustomerID: trial.CustomerID, CustomerName: name, Title: trial.Title,
					Status: trial.Status, CreatedAt: trial.CreatedAt,
					WorkorderInstanceID: s.workorderIDByBiz(ctx, model.OpsApprovalBizTrial, trial.ID),
					Sheets: s.listSafe(ctx, model.OpsAttachmentBizTrialSheet, trial.ID),
					Emails: s.listSafe(ctx, model.OpsAttachmentBizTrialEmail, trial.ID),
				}
				s.fillTrialLedger(p, trial, name)
				s.fillPackCounts(p)
				all = append(all, p)
			}
		}
		if req.Scene == "" || req.Scene == "formal" {
			acts, _, err := s.activationDAO.List(ctx, &model.ListOpsActivationReq{
				ListReq: model.ListReq{Page: 1, Size: 100, Search: req.Search},
			})
			if err != nil {
				return nil, err
			}
			for _, act := range acts {
				if act == nil {
					continue
				}
				name := ""
				if c, err := s.customerDAO.GetByID(ctx, act.CustomerID); err == nil && c != nil {
					name = c.Name
				}
				p := &model.OpsCustomerEvidencePack{
					Scene: "formal", SceneLabel: "正式开通", BizType: "activation", BizID: act.ID,
					CustomerID: act.CustomerID, CustomerName: name, Title: act.Title,
					Status: act.Status, CreatedAt: act.CreatedAt, ContractID: act.ContractID,
					WorkorderInstanceID: s.workorderIDByBiz(ctx, model.OpsApprovalBizActivation, act.ID),
					Sheets:    s.listSafe(ctx, model.OpsAttachmentBizActivationSheet, act.ID),
					Emails:    s.listSafe(ctx, model.OpsAttachmentBizActivationEmail, act.ID),
					Contracts: s.listSafe(ctx, model.OpsAttachmentBizContract, act.ContractID),
				}
				s.fillActivationLedger(ctx, p, act, name)
				s.fillPackCounts(p)
				all = append(all, p)
			}
		}
		sort.Slice(all, func(i, j int) bool {
			if all[i].CreatedAt.Equal(all[j].CreatedAt) {
				return all[i].BizID > all[j].BizID
			}
			return all[i].CreatedAt.After(all[j].CreatedAt)
		})
		all = filterHollowTrialDuplicatePacks(all)
	}

	total := int64(len(all))
	start := (page - 1) * size
	if start > len(all) {
		start = len(all)
	}
	end := start + size
	if end > len(all) {
		end = len(all)
	}
	return &model.ListResp[*model.OpsCustomerEvidencePack]{
		Items: all[start:end],
		Total: total,
	}, nil
}
