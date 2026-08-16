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

package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WorkorderCommentAttachmentDAO interface {
	CreateAttachment(ctx context.Context, attachment *model.WorkorderInstanceCommentAttachment) error
	GetAttachmentByID(ctx context.Context, id int) (*model.WorkorderInstanceCommentAttachment, error)
	BindAttachmentsToComment(ctx context.Context, commentID, instanceID, operatorID int, ids []int) error
	DeleteUnboundAttachment(ctx context.Context, id, operatorID int) (*model.WorkorderInstanceCommentAttachment, error)
	CountUnboundByOperator(ctx context.Context, instanceID, operatorID int) (int64, error)
}

type workorderCommentAttachmentDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewWorkorderCommentAttachmentDAO(db *gorm.DB, logger *zap.Logger) WorkorderCommentAttachmentDAO {
	return &workorderCommentAttachmentDAO{db: db, logger: logger}
}

func (d *workorderCommentAttachmentDAO) CreateAttachment(ctx context.Context, attachment *model.WorkorderInstanceCommentAttachment) error {
	if attachment == nil {
		return fmt.Errorf("附件对象为空")
	}
	if err := d.db.WithContext(ctx).Create(attachment).Error; err != nil {
		d.logger.Error("创建评论附件失败", zap.Error(err))
		return fmt.Errorf("创建评论附件失败: %w", err)
	}
	return nil
}

func (d *workorderCommentAttachmentDAO) GetAttachmentByID(ctx context.Context, id int) (*model.WorkorderInstanceCommentAttachment, error) {
	if id <= 0 {
		return nil, fmt.Errorf("附件ID无效")
	}
	var attachment model.WorkorderInstanceCommentAttachment
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&attachment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("附件不存在")
		}
		return nil, fmt.Errorf("获取附件失败: %w", err)
	}
	return &attachment, nil
}

func (d *workorderCommentAttachmentDAO) BindAttachmentsToComment(ctx context.Context, commentID, instanceID, operatorID int, ids []int) error {
	if commentID <= 0 || instanceID <= 0 || operatorID <= 0 {
		return fmt.Errorf("绑定参数无效")
	}
	if len(ids) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attachments []model.WorkorderInstanceCommentAttachment
		if err := tx.Where("id IN ? AND instance_id = ? AND operator_id = ? AND comment_id IS NULL", ids, instanceID, operatorID).
			Find(&attachments).Error; err != nil {
			return fmt.Errorf("查询待绑定附件失败: %w", err)
		}
		if len(attachments) != len(ids) {
			return fmt.Errorf("存在无效或已绑定的附件")
		}
		result := tx.Model(&model.WorkorderInstanceCommentAttachment{}).
			Where("id IN ? AND instance_id = ? AND operator_id = ? AND comment_id IS NULL", ids, instanceID, operatorID).
			Update("comment_id", commentID)
		if result.Error != nil {
			return fmt.Errorf("绑定附件失败: %w", result.Error)
		}
		if int(result.RowsAffected) != len(ids) {
			return fmt.Errorf("绑定附件数量不匹配")
		}
		return nil
	})
}

func (d *workorderCommentAttachmentDAO) DeleteUnboundAttachment(ctx context.Context, id, operatorID int) (*model.WorkorderInstanceCommentAttachment, error) {
	attachment, err := d.GetAttachmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if attachment.OperatorID != operatorID {
		return nil, fmt.Errorf("只能删除自己上传的附件")
	}
	if attachment.CommentID != nil {
		return nil, fmt.Errorf("已绑定评论的附件不能删除")
	}
	if err := d.db.WithContext(ctx).Delete(&model.WorkorderInstanceCommentAttachment{}, id).Error; err != nil {
		return nil, fmt.Errorf("删除附件失败: %w", err)
	}
	return attachment, nil
}

func (d *workorderCommentAttachmentDAO) CountUnboundByOperator(ctx context.Context, instanceID, operatorID int) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.WorkorderInstanceCommentAttachment{}).
		Where("instance_id = ? AND operator_id = ? AND comment_id IS NULL", instanceID, operatorID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("统计未绑定附件失败: %w", err)
	}
	return count, nil
}
