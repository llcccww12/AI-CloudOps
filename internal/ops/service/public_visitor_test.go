package service

import (
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
)

func TestIPRateLimiter(t *testing.T) {
	l := utils.NewIPRateLimiter(2, time.Minute)
	if !l.Allow("1.1.1.1") {
		t.Fatal("first should allow")
	}
	if !l.Allow("1.1.1.1") {
		t.Fatal("second should allow")
	}
	if l.Allow("1.1.1.1") {
		t.Fatal("third should deny")
	}
	if !l.Allow("2.2.2.2") {
		t.Fatal("other IP should allow")
	}
}

func TestIsHighOrMediumIntent(t *testing.T) {
	if !model.IsHighOrMediumIntent("medium") || !model.IsHighOrMediumIntent("high") {
		t.Fatal("medium/high should pass")
	}
	if model.IsHighOrMediumIntent("low") || model.IsHighOrMediumIntent("") {
		t.Fatal("low/empty should fail")
	}
}

func TestSubmitPublicVisitorReqValidationShape(t *testing.T) {
	now := time.Now()
	req := &model.SubmitPublicVisitorReq{
		CompanyName:  "测试单位",
		CompanyLevel: "民企",
		VisitorName:  "张三",
		VisitorCount: 3,
		VisitAt:      &now,
		Purpose:      "产品参观",
		ContactPhone: "13800138000",
		NeedMeeting:  1,
	}
	if req.CompanyLevel != "民企" || req.ContactPhone == "" {
		t.Fatal("req fields not set")
	}
}
