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

package model

// WorkorderInstanceCommentAttachment 工单评论附件
type WorkorderInstanceCommentAttachment struct {
	Model
	InstanceID  int    `json:"instance_id" gorm:"column:instance_id;not null;index;comment:工单实例ID"`
	CommentID   *int   `json:"comment_id,omitempty" gorm:"column:comment_id;index;comment:评论ID,上传后待绑定可为空"`
	OperatorID  int    `json:"operator_id" gorm:"column:operator_id;not null;index;comment:上传人ID"`
	FileName    string `json:"file_name" gorm:"column:file_name;type:varchar(255);not null;comment:原始文件名"`
	StoredName  string `json:"stored_name" gorm:"column:stored_name;type:varchar(255);not null;comment:存储文件名"`
	ContentType string `json:"content_type" gorm:"column:content_type;type:varchar(128);comment:MIME类型"`
	Size        int64  `json:"size" gorm:"column:size;not null;default:0;comment:文件大小(字节)"`
	StoragePath string `json:"-" gorm:"column:storage_path;type:varchar(512);not null;comment:相对存储路径"`
}

func (WorkorderInstanceCommentAttachment) TableName() string {
	return "cl_workorder_instance_comment_attachment"
}

type UploadWorkorderCommentAttachmentReq struct {
	InstanceID int `form:"instance_id" binding:"required,min=1"`
}

type DeleteWorkorderCommentAttachmentReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

type DownloadWorkorderCommentAttachmentReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}
