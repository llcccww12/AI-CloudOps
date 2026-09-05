package utils

import (
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ComputeLifeStatus 按开通/释放日计算生命周期状态（与 Excel 口径一致）
func ComputeLifeStatus(openedAt, planReleaseAt, actualReleaseAt *time.Time, now time.Time) string {
	if actualReleaseAt != nil {
		return model.OpsComputeLifeReleased
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if openedAt != nil {
		openDay := time.Date(openedAt.Year(), openedAt.Month(), openedAt.Day(), 0, 0, 0, 0, openedAt.Location())
		if openDay.After(today) {
			return model.OpsComputeLifePendingOpen
		}
	} else {
		// 未填开通日：视为待开通
		return model.OpsComputeLifePendingOpen
	}
	if planReleaseAt != nil {
		planDay := time.Date(planReleaseAt.Year(), planReleaseAt.Month(), planReleaseAt.Day(), 0, 0, 0, 0, planReleaseAt.Location())
		if planDay.Before(today) {
			return model.OpsComputeLifePendingRel
		}
	}
	return model.OpsComputeLifeInUse
}

func ComputeLifeStatusLabel(status string) string {
	switch status {
	case model.OpsComputeLifePendingOpen:
		return "待开通"
	case model.OpsComputeLifeInUse:
		return "在用"
	case model.OpsComputeLifePendingRel:
		return "到期待处理"
	case model.OpsComputeLifeReleased:
		return "已释放"
	default:
		return status
	}
}

func ComputeBizPhaseLabel(phase string) string {
	switch phase {
	case model.OpsComputePhaseTrial:
		return "测试"
	case model.OpsComputePhaseFormal:
		return "正式"
	case model.OpsComputePhaseRenew:
		return "续签"
	default:
		return phase
	}
}

// OccupyingGPUs 当前占用卡数：未实际释放前均占容（待开通/在用/待释放）
func OccupyingGPUs(lifeStatus string, allocated int) int {
	if allocated <= 0 {
		return 0
	}
	if lifeStatus == model.OpsComputeLifeReleased {
		return 0
	}
	return allocated
}

func CapacityCheck(serverOccupied int, serverGPUCount *int) (check, label string) {
	if serverGPUCount == nil || *serverGPUCount <= 0 {
		return "", ""
	}
	if serverOccupied > *serverGPUCount {
		return model.OpsComputeCapacityOverbook, "超配"
	}
	return model.OpsComputeCapacityOK, "正常"
}

func datePtrDay(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return &d
}

func formatDay(t *time.Time) string {
	if t == nil {
		return "?"
	}
	return t.Format("2006-01-02")
}

func sameContract(aID, bID int, aNo, bNo string) bool {
	aNo, bNo = strings.TrimSpace(aNo), strings.TrimSpace(bNo)
	if aID > 0 && bID > 0 {
		return aID == bID
	}
	if aNo != "" && bNo != "" {
		return aNo == bNo
	}
	// 两边都空：视为沿用同一「无合同」状态
	return aID == 0 && bID == 0 && aNo == "" && bNo == ""
}

// SeedPhaseHistoryIfEmpty 历史数据无履历时，用当前字段补一条，便于列表展示。
func SeedPhaseHistoryIfEmpty(item *model.OpsComputeAllocation) {
	if item == nil || len(item.PhaseHistory) > 0 {
		return
	}
	phase := item.BizPhase
	if phase == "" {
		if item.ActivationID > 0 {
			phase = model.OpsComputePhaseFormal
		} else {
			phase = model.OpsComputePhaseTrial
		}
	}
	start := item.CurrentPeriodStartAt
	if start == nil {
		start = item.OpenedAt
	}
	if start == nil {
		start = item.ContractStartAt
	}
	now := time.Now()
	item.PhaseHistory = model.OpsComputePhaseHistory{{
		Phase:         phase,
		StartAt:       datePtrDay(start),
		EndAt:         datePtrDay(item.PlanReleaseAt),
		ContractID:    item.ContractID,
		ContractNo:    item.ContractNo,
		LeaseMode:     item.LeaseMode,
		AllocatedGPUs: item.AllocatedGPUs,
		Note:          "初始占用",
		ChangedAt:     &now,
	}}
	if item.CurrentPeriodStartAt == nil {
		item.CurrentPeriodStartAt = datePtrDay(start)
	}
}

// AppendPhaseSegment 关闭上一段并追加新商业周期。
// 同阶段且合同未变：仅延期（更新截止日）；阶段变化或换新合同：追加新段，旧段保留原合同号。
func AppendPhaseSegment(
	history model.OpsComputePhaseHistory,
	oldPhase, newPhase string,
	periodStart, periodEnd *time.Time,
	contractID int,
	contractNo, leaseMode string,
	allocatedGPUs int,
	note string,
) model.OpsComputePhaseHistory {
	now := time.Now()
	start := datePtrDay(periodStart)
	end := datePtrDay(periodEnd)
	contractNo = strings.TrimSpace(contractNo)
	if oldPhase == "" {
		oldPhase = model.OpsComputePhaseTrial
	}
	if newPhase == "" {
		newPhase = oldPhase
	}

	if len(history) == 0 {
		return model.OpsComputePhaseHistory{{
			Phase: newPhase, StartAt: start, EndAt: end,
			ContractID: contractID, ContractNo: contractNo,
			LeaseMode: leaseMode, AllocatedGPUs: allocatedGPUs,
			Note: note, ChangedAt: &now,
		}}
	}

	last := history[len(history)-1]
	samePhase := last.Phase == newPhase || (last.Phase == "" && newPhase == oldPhase)

	// 未传合同视为沿用上一段（续签可不换合同）
	effContractID := contractID
	effContractNo := contractNo
	if effContractID <= 0 && effContractNo == "" {
		effContractID = last.ContractID
		effContractNo = last.ContractNo
	}
	contractUnchanged := sameContract(last.ContractID, effContractID, last.ContractNo, effContractNo)

	// 同阶段 + 同一合同：只延期
	if samePhase && newPhase == oldPhase && contractUnchanged {
		last.EndAt = end
		if leaseMode != "" {
			last.LeaseMode = leaseMode
		}
		if allocatedGPUs > 0 {
			last.AllocatedGPUs = allocatedGPUs
		}
		if note != "" {
			last.Note = note
		}
		last.ChangedAt = &now
		history[len(history)-1] = last
		return history
	}

	// 转阶段或换合同：上一段截止到新阶段开始前一日
	if start != nil {
		prevEnd := start.AddDate(0, 0, -1)
		last.EndAt = &prevEnd
	}
	last.ChangedAt = &now
	history[len(history)-1] = last

	history = append(history, model.OpsComputePhaseSegment{
		Phase: newPhase, StartAt: start, EndAt: end,
		ContractID: effContractID, ContractNo: effContractNo,
		LeaseMode: leaseMode, AllocatedGPUs: allocatedGPUs,
		Note: note, ChangedAt: &now,
	})
	return history
}

// FormatPhaseHistorySummary 列表摘要，含合同号：测试 … → 正式 …(HT-1) → 续签 …(HT-2)
func FormatPhaseHistorySummary(history model.OpsComputePhaseHistory) string {
	if len(history) == 0 {
		return ""
	}
	parts := make([]string, 0, len(history))
	for _, seg := range history {
		label := ComputeBizPhaseLabel(seg.Phase)
		if label == "" {
			label = seg.Phase
		}
		part := label + " " + formatDay(seg.StartAt) + "~" + formatDay(seg.EndAt)
		if cn := strings.TrimSpace(seg.ContractNo); cn != "" {
			part += "(" + cn + ")"
		}
		parts = append(parts, part)
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " → " + parts[i]
	}
	return out
}

// FormatContractTrailSummary 合同链条：HT-A → HT-B（去重相邻相同）
func FormatContractTrailSummary(history model.OpsComputePhaseHistory) string {
	if len(history) == 0 {
		return ""
	}
	parts := make([]string, 0, len(history))
	var prev string
	for _, seg := range history {
		cn := strings.TrimSpace(seg.ContractNo)
		if cn == "" {
			continue
		}
		if cn == prev {
			continue
		}
		parts = append(parts, cn)
		prev = cn
	}
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " → " + parts[i]
	}
	return out
}
