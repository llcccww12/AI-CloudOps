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

import "time"

const (
	OpsAttachmentBizContract         = "contract"
	OpsAttachmentBizSettlement       = "settlement"
	OpsAttachmentBizExhibition       = "exhibition"
	OpsAttachmentBizPublicFault      = "public_fault"
	OpsAttachmentBizTrialSheet       = "trial_sheet"       // 测试开通单（xlsx）
	OpsAttachmentBizTrialEmail       = "trial_email"       // 测试开通邮件佐证
	OpsAttachmentBizActivationSheet  = "activation_sheet"  // 正式开通单（xlsx）
	OpsAttachmentBizActivationEmail  = "activation_email"  // 正式开通邮件佐证
)

// OpsAttachment 运营业务附件（合同/结算/开通单/邮件佐证等）
type OpsAttachment struct {
	Model
	BizType     string `json:"biz_type" gorm:"column:biz_type;type:varchar(32);not null;index;comment:业务类型"`
	BizID       int    `json:"biz_id" gorm:"column:biz_id;not null;index;comment:业务ID"`
	OperatorID  int    `json:"operator_id" gorm:"column:operator_id;not null;index;comment:上传人ID"`
	FileName    string `json:"file_name" gorm:"column:file_name;type:varchar(255);not null;comment:原始文件名"`
	StoredName  string `json:"stored_name" gorm:"column:stored_name;type:varchar(255);not null;comment:存储文件名"`
	ContentType string `json:"content_type" gorm:"column:content_type;type:varchar(128);comment:MIME类型"`
	Size        int64  `json:"size" gorm:"column:size;not null;default:0;comment:文件大小(字节)"`
	StoragePath string `json:"-" gorm:"column:storage_path;type:varchar(512);not null;comment:相对存储路径"`
}

func (OpsAttachment) TableName() string { return "cl_ops_attachment" }

type ListOpsAttachmentReq struct {
	BizType string `json:"biz_type" form:"biz_type" binding:"required,oneof=contract settlement exhibition trial_sheet trial_email activation_sheet activation_email"`
	BizID   int    `json:"biz_id" form:"biz_id" binding:"required,min=1"`
}

// OpsCustomerEvidencePack 客户交付佐证包（测试/正式开通或合同）
type OpsCustomerEvidencePack struct {
	Scene               string           `json:"scene"` // trial | formal | contract
	SceneLabel          string           `json:"scene_label"`
	BizType             string           `json:"biz_type"` // trial | activation | contract
	BizID               int              `json:"biz_id"`
	CustomerID          int              `json:"customer_id"`
	CustomerName        string           `json:"customer_name,omitempty"`
	CustomerShortName   string           `json:"customer_short_name,omitempty"`
	ProductType         string           `json:"product_type,omitempty"`
	Region              string           `json:"region,omitempty"`
	OwnerName           string           `json:"owner_name,omitempty"`
	MainAccount         string           `json:"main_account,omitempty"`
	ProjectName         string           `json:"project_name,omitempty"`
	OpenMethod          string           `json:"open_method,omitempty"`
	OpenMethodLabel     string           `json:"open_method_label,omitempty"`
	ContractNo          string           `json:"contract_no,omitempty"`
	OrderNo             string           `json:"order_no,omitempty"`
	OpenPeriod          string           `json:"open_period,omitempty"`
	ContractStartAt     *time.Time       `json:"contract_start_at,omitempty"`
	ContractEndAt       *time.Time       `json:"contract_end_at,omitempty"`
	Title               string           `json:"title"`
	Status              string           `json:"status,omitempty"`
	OperatorName        string           `json:"operator_name,omitempty"`
	UpdaterName         string           `json:"updater_name,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at,omitempty"`
	WorkorderInstanceID int              `json:"workorder_instance_id,omitempty"`
	ContractID          int              `json:"contract_id,omitempty"`
	SheetCount          int              `json:"sheet_count"`
	EmailCount          int              `json:"email_count"`
	ContractFileCount   int              `json:"contract_file_count"`
	LedgerIncomplete    bool             `json:"ledger_incomplete,omitempty"` // 开通台账未写全，归档列多为空
	Sheets              []*OpsAttachment `json:"sheets"`
	Emails              []*OpsAttachment `json:"emails"`
	Contracts           []*OpsAttachment `json:"contracts,omitempty"`
}

// ListOpsDeliveryPackReq 开通单统一归档列表
type ListOpsDeliveryPackReq struct {
	ListReq
	Scene      string `json:"scene" form:"scene" binding:"omitempty,oneof=trial formal"`
	CustomerID int    `json:"customer_id" form:"customer_id"`
}
