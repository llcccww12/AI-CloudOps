package model

import "time"

const (
	OpsSurveyTypeCSAT       = "csat"
	OpsSurveyTypeNonRenewal = "non_renewal"

	OpsSurveyStatusDraft    = "draft"
	OpsSurveyStatusActive   = "active"
	OpsSurveyStatusArchived = "archived"

	OpsContractItemTypeMain  = "main"
	OpsContractItemTypeAddOn = "addon"
)

// OpsContractItem 合同行项目（主产品/增值服务）
type OpsContractItem struct {
	Model
	ContractID  int     `json:"contract_id" gorm:"column:contract_id;not null;index;comment:合同ID"`
	ItemType    string  `json:"item_type" gorm:"column:item_type;type:varchar(32);not null;default:main;comment:main/addon"`
	Name        string  `json:"name" gorm:"column:name;type:varchar(200);not null;comment:名称"`
	ProductType string  `json:"product_type" gorm:"column:product_type;type:varchar(64);comment:产品类型"`
	Quantity    float64 `json:"quantity" gorm:"column:quantity;type:decimal(14,2);default:1;comment:数量"`
	UnitPrice   float64 `json:"unit_price" gorm:"column:unit_price;type:decimal(14,2);default:0;comment:单价"`
	Amount      float64 `json:"amount" gorm:"column:amount;type:decimal(14,2);default:0;comment:金额"`
	Remark      string  `json:"remark" gorm:"column:remark;type:varchar(500);comment:备注"`
}

func (OpsContractItem) TableName() string { return "cl_ops_contract_item" }

type CreateOpsContractItemReq struct {
	ContractID  int     `json:"contract_id" binding:"required,min=1"`
	ItemType    string  `json:"item_type" binding:"omitempty,oneof=main addon"`
	Name        string  `json:"name" binding:"required,min=1,max=200"`
	ProductType string  `json:"product_type"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
	Remark      string  `json:"remark"`
}

type ListOpsContractItemReq struct {
	ListReq
	ContractID int `json:"contract_id" form:"contract_id" binding:"required,min=1"`
}

// OpsSurvey 问卷模板
type OpsSurvey struct {
	Model
	Code       string  `json:"code" gorm:"column:code;type:varchar(64);not null;uniqueIndex;comment:编码"`
	Title      string  `json:"title" gorm:"column:title;type:varchar(200);not null;comment:标题"`
	SurveyType string  `json:"survey_type" gorm:"column:survey_type;type:varchar(32);not null;index;comment:类型"`
	Questions  JSONMap `json:"questions" gorm:"column:questions;type:json;serializer:json;comment:题目定义"`
	Status     string  `json:"status" gorm:"column:status;type:varchar(32);not null;default:active;index;comment:状态"`
	Remark     string  `json:"remark" gorm:"column:remark;type:varchar(255);comment:备注"`
}

func (OpsSurvey) TableName() string { return "cl_ops_survey" }

// OpsSurveyResponse 问卷作答
type OpsSurveyResponse struct {
	Model
	SurveyID     int        `json:"survey_id" gorm:"column:survey_id;not null;index;comment:问卷ID"`
	CustomerID   int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	SurveyType   string     `json:"survey_type" gorm:"column:survey_type;type:varchar(32);not null;index;comment:类型"`
	Answers      JSONMap    `json:"answers" gorm:"column:answers;type:json;serializer:json;comment:作答"`
	Score        int        `json:"score" gorm:"column:score;default:0;comment:评分"`
	SubmittedAt  *time.Time `json:"submitted_at" gorm:"column:submitted_at;comment:提交时间"`
	OperatorID   int        `json:"operator_id" gorm:"column:operator_id;index;comment:操作人"`
	OperatorName string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
}

func (OpsSurveyResponse) TableName() string { return "cl_ops_survey_response" }

type SubmitOpsSurveyReq struct {
	SurveyCode   string  `json:"survey_code" binding:"required"`
	CustomerID   int     `json:"customer_id" binding:"required,min=1"`
	Answers      JSONMap `json:"answers" binding:"required"`
	Score        int     `json:"score"`
	OperatorID   int     `json:"operator_id"`
	OperatorName string  `json:"operator_name"`
}

type ListOpsSurveyResponseReq struct {
	ListReq
	CustomerID int    `json:"customer_id" form:"customer_id"`
	SurveyType string `json:"survey_type" form:"survey_type"`
}

const (
	OpsSurveyInviteStatusOpen = "open"
	OpsSurveyInviteStatusUsed = "used"
)

// OpsSurveyInvite 对外问卷邀请（客户凭 token 免登录填写）
type OpsSurveyInvite struct {
	Model
	Token        string     `json:"token" gorm:"column:token;type:varchar(64);not null;uniqueIndex;comment:邀请令牌"`
	SurveyCode   string     `json:"survey_code" gorm:"column:survey_code;type:varchar(64);not null;index;comment:问卷编码"`
	SurveyType   string     `json:"survey_type" gorm:"column:survey_type;type:varchar(32);not null;index;comment:问卷类型"`
	CustomerID   int        `json:"customer_id" gorm:"column:customer_id;not null;index;comment:客户ID"`
	CustomerName string     `json:"customer_name" gorm:"column:customer_name;type:varchar(200);comment:客户名称快照"`
	ExpireAt     *time.Time `json:"expire_at" gorm:"column:expire_at;index;comment:过期时间"`
	UsedAt       *time.Time `json:"used_at" gorm:"column:used_at;comment:作答时间"`
	Status       string     `json:"status" gorm:"column:status;type:varchar(32);not null;default:open;index;comment:open/used"`
	OperatorID   int        `json:"operator_id" gorm:"column:operator_id;comment:创建人"`
	OperatorName string     `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:创建人"`
	Remark       string     `json:"remark" gorm:"column:remark;type:varchar(255);comment:备注"`
}

func (OpsSurveyInvite) TableName() string { return "cl_ops_survey_invite" }

type CreateOpsSurveyInviteReq struct {
	SurveyCode   string `json:"survey_code" binding:"required"`
	CustomerID   int    `json:"customer_id" binding:"required,min=1"`
	ExpireDays   int    `json:"expire_days"`
	OperatorID   int    `json:"operator_id"`
	OperatorName string `json:"operator_name"`
}

type CreateOpsSurveyInviteResp struct {
	Token        string `json:"token"`
	Path         string `json:"path"`
	URL          string `json:"url"`
	SurveyCode   string `json:"survey_code"`
	SurveyTitle  string `json:"survey_title"`
	CustomerID   int    `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	ExpireAt     string `json:"expire_at,omitempty"`
}

type PublicSurveyMeta struct {
	Token        string  `json:"token"`
	SurveyCode   string  `json:"survey_code"`
	SurveyTitle  string  `json:"survey_title"`
	SurveyType   string  `json:"survey_type"`
	CustomerName string  `json:"customer_name"`
	Questions    JSONMap `json:"questions"`
	ExpireAt     string  `json:"expire_at,omitempty"`
}

type SubmitPublicSurveyReq struct {
	Token   string  `json:"token" binding:"required"`
	Answers JSONMap `json:"answers" binding:"required"`
	Score   int     `json:"score"`
}

type ListOpsSurveyInviteReq struct {
	ListReq
	CustomerID int    `json:"customer_id" form:"customer_id"`
	SurveyCode string `json:"survey_code" form:"survey_code"`
}
