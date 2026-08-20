package model

// OpsVendorProfile 客商档案（智慧思明录入信息，系统内表单落库）
type OpsVendorProfile struct {
	Model
	CustomerID          int    `json:"customer_id" gorm:"column:customer_id;not null;uniqueIndex;comment:客户ID"`
	UnitName            string `json:"unit_name" gorm:"column:unit_name;type:varchar(200);not null;comment:单位名称"`
	CreditCode          string `json:"credit_code" gorm:"column:credit_code;type:varchar(64);comment:统一社会信用代码"`
	Principal           string `json:"principal" gorm:"column:principal;type:varchar(100);comment:负责人"`
	ContactAddressPhone string `json:"contact_address_phone" gorm:"column:contact_address_phone;type:varchar(500);comment:联系地址和电话"`
	CnapsCode           string `json:"cnaps_code" gorm:"column:cnaps_code;type:varchar(64);comment:联行号"`
	AccountName         string `json:"account_name" gorm:"column:account_name;type:varchar(200);comment:账号名称(收款人)"`
	BankName            string `json:"bank_name" gorm:"column:bank_name;type:varchar(200);comment:开户行名称(精确到支行)"`
	BankAccount         string `json:"bank_account" gorm:"column:bank_account;type:varchar(64);comment:银行账号"`
	BankProvince        string `json:"bank_province" gorm:"column:bank_province;type:varchar(64);comment:银行所在省份"`
	BankCity            string `json:"bank_city" gorm:"column:bank_city;type:varchar(64);comment:银行所在市"`
	Phone               string `json:"phone" gorm:"column:phone;type:varchar(50);comment:电话"`
	AccountType         string `json:"account_type" gorm:"column:account_type;type:varchar(32);comment:账户类型"`
	OperatorID          int    `json:"operator_id" gorm:"column:operator_id;comment:操作人ID"`
	OperatorName        string `json:"operator_name" gorm:"column:operator_name;type:varchar(100);comment:操作人"`
}

func (OpsVendorProfile) TableName() string { return "cl_ops_vendor_profile" }

type UpsertOpsVendorProfileReq struct {
	CustomerID          int    `json:"customer_id" binding:"required,min=1"`
	UnitName            string `json:"unit_name" binding:"required,min=1,max=200"`
	CreditCode          string `json:"credit_code" binding:"omitempty,max=64"`
	Principal           string `json:"principal" binding:"omitempty,max=100"`
	ContactAddressPhone string `json:"contact_address_phone" binding:"omitempty,max=500"`
	CnapsCode           string `json:"cnaps_code" binding:"omitempty,max=64"`
	AccountName         string `json:"account_name" binding:"omitempty,max=200"`
	BankName            string `json:"bank_name" binding:"omitempty,max=200"`
	BankAccount         string `json:"bank_account" binding:"omitempty,max=64"`
	BankProvince        string `json:"bank_province" binding:"omitempty,max=64"`
	BankCity            string `json:"bank_city" binding:"omitempty,max=64"`
	Phone               string `json:"phone" binding:"omitempty,max=50"`
	AccountType         string `json:"account_type" binding:"omitempty,oneof=corporate personal"`
	OperatorID          int    `json:"operator_id"`
	OperatorName        string `json:"operator_name"`
}
