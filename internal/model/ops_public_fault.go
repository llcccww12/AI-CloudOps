package model

import "time"

type SubmitPublicFaultReq struct {
	ReportCode     string `json:"report_code" binding:"required,min=2,max=64"`
	ReportSecret   string `json:"report_secret" binding:"required,min=4,max=64"`
	Title          string `json:"title" binding:"required,min=1,max=200"`
	Description    string `json:"description" binding:"required,min=1,max=2000"`
	Priority       int8   `json:"priority" binding:"required,oneof=1 2 3"`
	ReporterName   string `json:"reporter_name" binding:"required,min=1,max=100"`
	ReporterPhone  string `json:"reporter_phone" binding:"required,min=5,max=50"`
	ReporterEmail  string `json:"reporter_email" binding:"omitempty,max=120"`
	AttachmentIDs  []int `json:"attachment_ids" binding:"omitempty,dive,min=1"`
}

type SubmitPublicFaultResp struct {
	SerialNumber string `json:"serial_number"`
	QueryCode    string `json:"query_code"`
	Message      string `json:"message"`
}

type QueryPublicFaultReq struct {
	SerialNumber string `json:"serial_number" binding:"required,min=1,max=50"`
	QueryCode    string `json:"query_code" binding:"required,min=4,max=32"`
}

type PublicFaultTimelineItem struct {
	Time    time.Time `json:"time"`
	Action  string    `json:"action"`
	Content string    `json:"content"`
}

type QueryPublicFaultResp struct {
	SerialNumber string                    `json:"serial_number"`
	Title        string                    `json:"title"`
	Status       int8                      `json:"status"`
	StatusText   string                    `json:"status_text"`
	Priority     int8                      `json:"priority"`
	PriorityText string                    `json:"priority_text"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
	CompletedAt  *time.Time                `json:"completed_at,omitempty"`
	Description  string                    `json:"description"`
	Timeline     []PublicFaultTimelineItem `json:"timeline"`
}

type CreatePublicFaultInstanceReq struct {
	Title          string
	Description    string
	Priority       int8
	FormData       JSONMap
	OperatorID     int
	OperatorName   string
	OpsCustomerID  int
	ReporterName   string
	ReporterPhone  string
	ReporterEmail  string
	PublicQueryHash string
}
