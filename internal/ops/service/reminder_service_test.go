package service

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
)

func TestUniquePositiveIDs(t *testing.T) {
	got := uniquePositiveIDs([]int{0, 1, 1, 2, -3, 2})
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestBuildReminderDedupeKey(t *testing.T) {
	key := dao.BuildReminderDedupeKey("rule", 3, 9, "feishu", "2026-08-20", "trial", 11)
	want := "rule:3:u9:feishu:2026-08-20:trial:11"
	if key != want {
		t.Fatalf("got %s want %s", key, want)
	}
}

func TestIsChannelSkip(t *testing.T) {
	if !isChannelSkip(errSkip("跳过: 短信未配置")) {
		t.Fatal("expected skip")
	}
	if isChannelSkip(errSkip("发送失败")) {
		t.Fatal("expected not skip")
	}
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errSkip(msg string) error { return simpleErr(msg) }
