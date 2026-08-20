package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

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
}

type opsAttachmentService struct {
	attachmentDAO dao.OpsAttachmentDAO
	contractDAO   dao.OpsContractDAO
	settlementDAO dao.OpsSettlementDAO
	exhibitionDAO dao.OpsExhibitionDAO
	logger        *zap.Logger
}

func NewOpsAttachmentService(
	attachmentDAO dao.OpsAttachmentDAO,
	contractDAO dao.OpsContractDAO,
	settlementDAO dao.OpsSettlementDAO,
	exhibitionDAO dao.OpsExhibitionDAO,
	logger *zap.Logger,
) OpsAttachmentService {
	return &opsAttachmentService{
		attachmentDAO: attachmentDAO,
		contractDAO:   contractDAO,
		settlementDAO: settlementDAO,
		exhibitionDAO: exhibitionDAO,
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
	case model.OpsAttachmentBizExhibition:
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
