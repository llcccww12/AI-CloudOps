/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	workorderUtils "github.com/GoSimplicity/AI-CloudOps/internal/workorder/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type InstanceCommentService interface {
	CreateInstanceComment(ctx context.Context, req *model.CreateWorkorderInstanceCommentReq) error
	UpdateInstanceComment(ctx context.Context, req *model.UpdateWorkorderInstanceCommentReq, userID int) error
	DeleteInstanceComment(ctx context.Context, id int, userID int) error
	GetInstanceComment(ctx context.Context, id int) (*model.WorkorderInstanceComment, error)
	ListInstanceComments(ctx context.Context, req *model.ListWorkorderInstanceCommentReq) (*model.ListResp[*model.WorkorderInstanceComment], error)
	GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error)
	UploadCommentAttachment(ctx context.Context, instanceID, operatorID int, header *multipart.FileHeader) (*model.WorkorderInstanceCommentAttachment, error)
	DownloadCommentAttachment(ctx context.Context, id int) (*model.WorkorderInstanceCommentAttachment, string, error)
	DeleteCommentAttachment(ctx context.Context, id, operatorID int) error
}

type instanceCommentService struct {
	dao                 dao.WorkorderInstanceCommentDAO
	attachmentDAO       dao.WorkorderCommentAttachmentDAO
	instanceDao         dao.WorkorderInstanceDAO
	notificationService WorkorderNotificationService
	logger              *zap.Logger
}

func NewInstanceCommentService(
	dao dao.WorkorderInstanceCommentDAO,
	attachmentDAO dao.WorkorderCommentAttachmentDAO,
	instanceDao dao.WorkorderInstanceDAO,
	notificationService WorkorderNotificationService,
	logger *zap.Logger,
) InstanceCommentService {
	return &instanceCommentService{
		dao:                 dao,
		attachmentDAO:       attachmentDAO,
		instanceDao:         instanceDao,
		notificationService: notificationService,
		logger:              logger,
	}
}

func (s *instanceCommentService) CreateInstanceComment(ctx context.Context, req *model.CreateWorkorderInstanceCommentReq) error {
	content := strings.TrimSpace(req.Content)
	if content == "" && len(req.AttachmentIDs) == 0 {
		return fmt.Errorf("评论内容或附件至少填写一项")
	}
	maxCount := workorderUtils.GetWorkorderAttachmentMaxCount()
	if len(req.AttachmentIDs) > maxCount {
		return fmt.Errorf("单次评论最多上传 %d 个附件", maxCount)
	}

	_, err := s.instanceDao.GetInstanceByID(ctx, req.InstanceID)
	if err != nil {
		s.logger.Error("工单不存在", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("工单不存在: %w", err)
	}

	if req.ParentID != nil && *req.ParentID > 0 {
		parentComment, err := s.dao.GetInstanceCommentByID(ctx, *req.ParentID)
		if err != nil {
			s.logger.Error("父评论不存在", zap.Error(err), zap.Int("parentID", *req.ParentID))
			return fmt.Errorf("父评论不存在: %w", err)
		}
		if parentComment.InstanceID != req.InstanceID {
			return fmt.Errorf("父评论不属于当前工单")
		}
	}

	comment := &model.WorkorderInstanceComment{
		InstanceID:   req.InstanceID,
		OperatorID:   req.OperatorID,
		OperatorName: req.OperatorName,
		Content:      content,
		ParentID:     req.ParentID,
		Type:         req.Type,
		Status:       model.CommentStatusNormal,
		IsSystem:     req.IsSystem,
	}

	if comment.Type == "" {
		comment.Type = model.CommentTypeNormal
	}

	if err := s.dao.CreateInstanceComment(ctx, comment); err != nil {
		s.logger.Error("创建工单评论失败", zap.Error(err))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	if len(req.AttachmentIDs) > 0 {
		if err := s.attachmentDAO.BindAttachmentsToComment(ctx, comment.ID, req.InstanceID, req.OperatorID, req.AttachmentIDs); err != nil {
			s.logger.Error("绑定评论附件失败", zap.Error(err), zap.Int("commentID", comment.ID))
			return fmt.Errorf("绑定评论附件失败: %w", err)
		}
	}

	if s.notificationService != nil && comment.IsSystem != 1 {
		instanceID := comment.InstanceID
		notifyContent := content
		if notifyContent == "" {
			notifyContent = fmt.Sprintf("[附件 x%d]", len(req.AttachmentIDs))
		}
		go func() {
			notifyCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			if err := s.notificationService.SendWorkorderNotification(notifyCtx, instanceID, model.EventTypeInstanceCommented, notifyContent); err != nil {
				s.logger.Error("发送工单评论通知失败",
					zap.Error(err),
					zap.Int("instance_id", instanceID))
			}
		}()
	}

	return nil
}

func (s *instanceCommentService) UpdateInstanceComment(ctx context.Context, req *model.UpdateWorkorderInstanceCommentReq, userID int) error {
	existingComment, err := s.dao.GetInstanceCommentByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("获取评论失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("获取评论失败: %w", err)
	}

	if existingComment.IsSystem != 1 && existingComment.OperatorID != userID {
		return fmt.Errorf("只能修改自己的评论")
	}

	comment := &model.WorkorderInstanceComment{
		Model:    model.Model{ID: req.ID},
		Content:  strings.TrimSpace(req.Content),
		Status:   req.Status,
		IsSystem: req.IsSystem,
	}

	if err := s.dao.UpdateInstanceComment(ctx, comment); err != nil {
		s.logger.Error("更新工单评论失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新工单评论失败: %w", err)
	}

	return nil
}

func (s *instanceCommentService) DeleteInstanceComment(ctx context.Context, id int, userID int) error {
	comment, err := s.dao.GetInstanceCommentByID(ctx, id)
	if err != nil {
		s.logger.Error("获取评论失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("获取评论失败: %w", err)
	}

	if comment.IsSystem != 1 && comment.OperatorID != userID {
		return fmt.Errorf("只能删除自己的评论")
	}

	if err := s.dao.DeleteInstanceComment(ctx, id); err != nil {
		s.logger.Error("删除工单评论失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除工单评论失败: %w", err)
	}

	return nil
}

func (s *instanceCommentService) GetInstanceComment(ctx context.Context, id int) (*model.WorkorderInstanceComment, error) {
	comment, err := s.dao.GetInstanceCommentByID(ctx, id)
	if err != nil {
		s.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	return comment, nil
}

func (s *instanceCommentService) ListInstanceComments(ctx context.Context, req *model.ListWorkorderInstanceCommentReq) (*model.ListResp[*model.WorkorderInstanceComment], error) {
	comments, total, err := s.dao.ListInstanceComments(ctx, req)
	if err != nil {
		s.logger.Error("获取工单评论列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单评论列表失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderInstanceComment]{
		Items: comments,
		Total: total,
	}, nil
}

func (s *instanceCommentService) GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error) {
	_, err := s.instanceDao.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("工单不存在", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	comments, err := s.dao.GetInstanceCommentsTree(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单评论树失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单评论树失败: %w", err)
	}

	return comments, nil
}

func (s *instanceCommentService) UploadCommentAttachment(ctx context.Context, instanceID, operatorID int, header *multipart.FileHeader) (*model.WorkorderInstanceCommentAttachment, error) {
	if header == nil {
		return nil, fmt.Errorf("未选择文件")
	}
	if _, err := s.instanceDao.GetInstanceByID(ctx, instanceID); err != nil {
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	maxCount := workorderUtils.GetWorkorderAttachmentMaxCount()
	unbound, err := s.attachmentDAO.CountUnboundByOperator(ctx, instanceID, operatorID)
	if err != nil {
		return nil, err
	}
	if unbound >= int64(maxCount) {
		return nil, fmt.Errorf("待发送附件已达上限 %d 个，请先发送或移除", maxCount)
	}

	maxSize := workorderUtils.GetWorkorderAttachmentMaxSizeBytes()
	if header.Size > maxSize {
		return nil, fmt.Errorf("文件大小不能超过 %dMB", maxSize/1024/1024)
	}

	_, contentType, err := workorderUtils.ValidateCommentAttachmentFileName(header.Filename)
	if err != nil {
		return nil, err
	}
	if header.Header.Get("Content-Type") != "" {
		// 保留浏览器声明的 MIME，但白名单校验以扩展名为准
		_ = header.Header.Get("Content-Type")
	}

	safeName := workorderUtils.SanitizeAttachmentFileName(header.Filename)
	storedName := fmt.Sprintf("%s_%s", uuid.NewString(), safeName)
	relPath := filepath.ToSlash(filepath.Join(fmt.Sprintf("%d", instanceID), storedName))
	absDir := filepath.Join(workorderUtils.GetWorkorderAttachmentDir(), fmt.Sprintf("%d", instanceID))
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

	attachment := &model.WorkorderInstanceCommentAttachment{
		InstanceID:  instanceID,
		OperatorID:  operatorID,
		FileName:    filepath.Base(header.Filename),
		StoredName:  storedName,
		ContentType: contentType,
		Size:        written,
		StoragePath: relPath,
	}
	if err := s.attachmentDAO.CreateAttachment(ctx, attachment); err != nil {
		_ = os.Remove(absPath)
		return nil, err
	}
	return attachment, nil
}

func (s *instanceCommentService) DownloadCommentAttachment(ctx context.Context, id int) (*model.WorkorderInstanceCommentAttachment, string, error) {
	attachment, err := s.attachmentDAO.GetAttachmentByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if _, err := s.instanceDao.GetInstanceByID(ctx, attachment.InstanceID); err != nil {
		return nil, "", fmt.Errorf("工单不存在: %w", err)
	}
	absPath := filepath.Join(workorderUtils.GetWorkorderAttachmentDir(), filepath.FromSlash(attachment.StoragePath))
	if _, err := os.Stat(absPath); err != nil {
		return nil, "", fmt.Errorf("附件文件不存在")
	}
	return attachment, absPath, nil
}

func (s *instanceCommentService) DeleteCommentAttachment(ctx context.Context, id, operatorID int) error {
	attachment, err := s.attachmentDAO.DeleteUnboundAttachment(ctx, id, operatorID)
	if err != nil {
		return err
	}
	absPath := filepath.Join(workorderUtils.GetWorkorderAttachmentDir(), filepath.FromSlash(attachment.StoragePath))
	_ = os.Remove(absPath)
	return nil
}
