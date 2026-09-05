package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/ops/dao"
	opsUtils "github.com/GoSimplicity/AI-CloudOps/internal/ops/utils"
	"go.uber.org/zap"
)

type OpsComputeService interface {
	CreateAsset(ctx context.Context, req *model.CreateOpsComputeAssetReq) error
	UpdateAsset(ctx context.Context, req *model.UpdateOpsComputeAssetReq) error
	DeleteAsset(ctx context.Context, id int) error
	GetAsset(ctx context.Context, id int) (*model.OpsComputeAsset, error)
	ListAsset(ctx context.Context, req *model.ListOpsComputeAssetReq) (*model.ListResp[*model.OpsComputeAsset], error)
	SeedDefaultAssets(ctx context.Context, operatorID int, operatorName string) (int, error)

	CreateAllocation(ctx context.Context, req *model.CreateOpsComputeAllocationReq) error
	UpdateAllocation(ctx context.Context, req *model.UpdateOpsComputeAllocationReq) error
	ReleaseAllocation(ctx context.Context, req *model.ReleaseOpsComputeAllocationReq) error
	ExtendAllocation(ctx context.Context, req *model.ExtendOpsComputeAllocationReq) error
	DeleteAllocation(ctx context.Context, id int) error
	GetAllocation(ctx context.Context, id int) (*model.OpsComputeAllocation, error)
	ListAllocation(ctx context.Context, req *model.ListOpsComputeAllocationReq) (*model.ListResp[*model.OpsComputeAllocation], error)

	Dashboard(ctx context.Context) (*model.OpsComputeDashboard, error)
}

type opsComputeService struct {
	assetDAO       dao.OpsComputeAssetDAO
	allocDAO       dao.OpsComputeAllocationDAO
	customerDAO    dao.OpsCustomerDAO
	contractDAO    dao.OpsContractDAO
	activationDAO  dao.OpsActivationDAO
	trialDAO       dao.OpsTrialDAO
	attachmentDAO  dao.OpsAttachmentDAO
	approvalDAO    dao.OpsApprovalLinkDAO
	logger         *zap.Logger
}

func NewOpsComputeService(
	assetDAO dao.OpsComputeAssetDAO,
	allocDAO dao.OpsComputeAllocationDAO,
	customerDAO dao.OpsCustomerDAO,
	contractDAO dao.OpsContractDAO,
	activationDAO dao.OpsActivationDAO,
	trialDAO dao.OpsTrialDAO,
	attachmentDAO dao.OpsAttachmentDAO,
	approvalDAO dao.OpsApprovalLinkDAO,
	logger *zap.Logger,
) OpsComputeService {
	return &opsComputeService{
		assetDAO: assetDAO, allocDAO: allocDAO, customerDAO: customerDAO,
		contractDAO: contractDAO, activationDAO: activationDAO, trialDAO: trialDAO,
		attachmentDAO: attachmentDAO, approvalDAO: approvalDAO, logger: logger,
	}
}

func (s *opsComputeService) CreateAsset(ctx context.Context, req *model.CreateOpsComputeAssetReq) error {
	code := strings.TrimSpace(req.ServerCode)
	if code == "" {
		return fmt.Errorf("请填写服务器唯一标识")
	}
	if _, err := s.assetDAO.GetByServerCode(ctx, code); err == nil {
		return fmt.Errorf("服务器标识已存在: %s", code)
	}
	status := req.Status
	if status == "" {
		status = model.OpsComputeAssetAvailable
	}
	mode := req.DefaultLeaseMode
	if mode == "" {
		mode = model.OpsComputeLeaseFull
	}
	return s.assetDAO.Create(ctx, &model.OpsComputeAsset{
		ServerCode: code, Name: strings.TrimSpace(req.Name), GPUModel: strings.TrimSpace(req.GPUModel),
		SerialNo: req.SerialNo, RackLocation: req.RackLocation, MgmtIP: req.MgmtIP,
		CommissionedAt: req.CommissionedAt, GPUCount: req.GPUCount, DefaultLeaseMode: mode,
		Status: status, TechOwner: req.TechOwner, NextInspectAt: req.NextInspectAt, Remark: req.Remark,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	})
}

func (s *opsComputeService) UpdateAsset(ctx context.Context, req *model.UpdateOpsComputeAssetReq) error {
	status := req.Status
	if status == "" {
		status = model.OpsComputeAssetAvailable
	}
	mode := req.DefaultLeaseMode
	if mode == "" {
		mode = model.OpsComputeLeaseFull
	}
	return s.assetDAO.Update(ctx, &model.OpsComputeAsset{
		Model: model.Model{ID: req.ID}, Name: strings.TrimSpace(req.Name), GPUModel: strings.TrimSpace(req.GPUModel),
		SerialNo: req.SerialNo, RackLocation: req.RackLocation, MgmtIP: req.MgmtIP,
		CommissionedAt: req.CommissionedAt, GPUCount: req.GPUCount, DefaultLeaseMode: mode,
		Status: status, TechOwner: req.TechOwner, NextInspectAt: req.NextInspectAt, Remark: req.Remark,
	})
}

func (s *opsComputeService) DeleteAsset(ctx context.Context, id int) error {
	asset, err := s.assetDAO.GetByID(ctx, id)
	if err != nil {
		return err
	}
	items, err := s.allocDAO.ListByServerCode(ctx, asset.ServerCode)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, item := range items {
		st := opsUtils.ComputeLifeStatus(item.OpenedAt, item.PlanReleaseAt, item.ActualReleaseAt, now)
		if st != model.OpsComputeLifeReleased {
			return fmt.Errorf("服务器仍有未释放的分配记录，无法删除")
		}
	}
	return s.assetDAO.Delete(ctx, id)
}

func (s *opsComputeService) GetAsset(ctx context.Context, id int) (*model.OpsComputeAsset, error) {
	return s.assetDAO.GetByID(ctx, id)
}

func (s *opsComputeService) ListAsset(ctx context.Context, req *model.ListOpsComputeAssetReq) (*model.ListResp[*model.OpsComputeAsset], error) {
	items, total, err := s.assetDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &model.ListResp[*model.OpsComputeAsset]{Items: items, Total: total}, nil
}

func (s *opsComputeService) SeedDefaultAssets(ctx context.Context, operatorID int, operatorName string) (int, error) {
	n, err := s.assetDAO.CountAll(ctx)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, nil
	}
	items := make([]*model.OpsComputeAsset, 0, 24)
	h100Count := 8
	v100Count := 8
	for i := 1; i <= 20; i++ {
		items = append(items, &model.OpsComputeAsset{
			ServerCode: fmt.Sprintf("H100-SRV-%02d", i), Name: "H100算力服务器", GPUModel: "H100",
			GPUCount: &h100Count, DefaultLeaseMode: model.OpsComputeLeaseFull, Status: model.OpsComputeAssetAvailable,
			Remark: "预置资产（默认8卡，请按实物核验序列号/机架）", OperatorID: operatorID, OperatorName: operatorName,
		})
	}
	for i := 1; i <= 4; i++ {
		items = append(items, &model.OpsComputeAsset{
			ServerCode: fmt.Sprintf("V100-SRV-%02d", i), Name: "智凯V100算力服务器", GPUModel: "智凯 V100",
			GPUCount: &v100Count, DefaultLeaseMode: model.OpsComputeLeaseFull, Status: model.OpsComputeAssetAvailable,
			Remark: "预置资产（默认8卡，请按实物核验序列号/机架）", OperatorID: operatorID, OperatorName: operatorName,
		})
	}
	if err := s.assetDAO.BatchCreate(ctx, items); err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *opsComputeService) CreateAllocation(ctx context.Context, req *model.CreateOpsComputeAllocationReq) error {
	item, err := s.buildAllocationFromCreate(ctx, req)
	if err != nil {
		return err
	}
	if item.CurrentPeriodStartAt == nil {
		if item.OpenedAt != nil {
			item.CurrentPeriodStartAt = item.OpenedAt
		} else if item.ContractStartAt != nil {
			item.CurrentPeriodStartAt = item.ContractStartAt
		}
	}
	opsUtils.SeedPhaseHistoryIfEmpty(item)
	if err := s.validateLease(item); err != nil {
		return err
	}
	if err := s.ensureNoOverbook(ctx, item.ServerCode, item.AllocatedGPUs, 0, item.OpenedAt, item.PlanReleaseAt, item.ActualReleaseAt); err != nil {
		return err
	}
	item.RecordNo = fmt.Sprintf("LCA-%s-%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%100000)
	return s.allocDAO.Create(ctx, item)
}

func (s *opsComputeService) UpdateAllocation(ctx context.Context, req *model.UpdateOpsComputeAllocationReq) error {
	exist, err := s.allocDAO.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if exist.ActualReleaseAt != nil {
		return fmt.Errorf("已释放记录不可修改，请新建分配")
	}
	item, err := s.buildAllocationFromUpdate(ctx, req, exist)
	if err != nil {
		return err
	}
	opsUtils.SeedPhaseHistoryIfEmpty(exist)
	oldPhase := exist.BizPhase
	if oldPhase == "" {
		oldPhase = model.OpsComputePhaseTrial
	}
	newPhase := item.BizPhase
	if newPhase == "" {
		newPhase = oldPhase
	}
	phaseChanged := newPhase != oldPhase
	endChanged := !sameCalendarDayPtr(exist.PlanReleaseAt, item.PlanReleaseAt)
	if phaseChanged || endChanged {
		periodStart := item.CurrentPeriodStartAt
		if phaseChanged {
			if item.ContractStartAt != nil {
				periodStart = item.ContractStartAt
			} else {
				now := time.Now()
				periodStart = &now
			}
			item.CurrentPeriodStartAt = periodStart
		} else if periodStart == nil {
			periodStart = exist.CurrentPeriodStartAt
			if periodStart == nil {
				periodStart = exist.OpenedAt
			}
			item.CurrentPeriodStartAt = periodStart
		}
		note := ""
		if phaseChanged {
			note = fmt.Sprintf("%s→%s", opsUtils.ComputeBizPhaseLabel(oldPhase), opsUtils.ComputeBizPhaseLabel(newPhase))
		} else {
			note = "延期"
		}
		item.PhaseHistory = opsUtils.AppendPhaseSegment(
			exist.PhaseHistory, oldPhase, newPhase,
			periodStart, item.PlanReleaseAt,
			item.ContractID, item.ContractNo, item.LeaseMode, item.AllocatedGPUs, note,
		)
		item.BizPhase = newPhase
	} else {
		item.PhaseHistory = exist.PhaseHistory
		if item.CurrentPeriodStartAt == nil {
			item.CurrentPeriodStartAt = exist.CurrentPeriodStartAt
		}
	}
	if err := s.validateLease(item); err != nil {
		return err
	}
	if err := s.ensureNoOverbook(ctx, item.ServerCode, item.AllocatedGPUs, exist.ID, item.OpenedAt, item.PlanReleaseAt, item.ActualReleaseAt); err != nil {
		return err
	}
	item.ID = exist.ID
	item.RecordNo = exist.RecordNo
	return s.allocDAO.Update(ctx, item)
}

func sameCalendarDayPtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func (s *opsComputeService) ReleaseAllocation(ctx context.Context, req *model.ReleaseOpsComputeAllocationReq) error {
	exist, err := s.allocDAO.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if exist.ActualReleaseAt != nil {
		return fmt.Errorf("该记录已释放")
	}
	if strings.TrimSpace(req.ChangeTicketNo) == "" {
		return fmt.Errorf("请填写变更单号")
	}
	if strings.TrimSpace(req.Remark) == "" {
		return fmt.Errorf("请填写释放说明")
	}
	at := req.ActualReleaseAt
	if at == nil {
		now := time.Now()
		at = &now
	}
	exist.ActualReleaseAt = at
	exist.ChangeTicketNo = strings.TrimSpace(req.ChangeTicketNo)
	exist.Remark = strings.TrimSpace(req.Remark)
	return s.allocDAO.Update(ctx, exist)
}

func (s *opsComputeService) ExtendAllocation(ctx context.Context, req *model.ExtendOpsComputeAllocationReq) error {
	exist, err := s.allocDAO.GetByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if exist.ActualReleaseAt != nil {
		return fmt.Errorf("已释放记录不可续期，请新建分配")
	}
	if req.PlanReleaseAt == nil {
		return fmt.Errorf("请填写新的周期截止日")
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	planDay := time.Date(req.PlanReleaseAt.Year(), req.PlanReleaseAt.Month(), req.PlanReleaseAt.Day(), 0, 0, 0, 0, req.PlanReleaseAt.Location())
	if planDay.Before(today) {
		return fmt.Errorf("新的周期截止日不能早于今天")
	}
	phase := strings.TrimSpace(req.BizPhase)
	if phase == "" {
		return fmt.Errorf("请选择商业阶段")
	}

	opsUtils.SeedPhaseHistoryIfEmpty(exist)
	oldPhase := exist.BizPhase
	if oldPhase == "" {
		oldPhase = model.OpsComputePhaseTrial
	}

	periodStart := req.PeriodStartAt
	if periodStart == nil {
		periodStart = req.ContractStartAt
	}
	if periodStart == nil {
		periodStart = &today
	}

	// 合同：未传则沿用当前；传了新编号/ID 则换绑（履历另起一段）
	reuseContract := req.ContractID <= 0 && strings.TrimSpace(req.ContractNo) == ""
	segContractID := exist.ContractID
	segContractNo := strings.TrimSpace(exist.ContractNo)
	if !reuseContract {
		if req.ContractID > 0 {
			segContractID = req.ContractID
			exist.ContractID = req.ContractID
			if c, err := s.contractDAO.GetByID(ctx, req.ContractID); err == nil && c != nil {
				if strings.TrimSpace(req.ContractNo) == "" {
					segContractNo = c.ContractNo
				}
				if req.ContractStartAt == nil {
					req.ContractStartAt = c.StartAt
				}
				if req.ContractEndAt == nil {
					req.ContractEndAt = c.EndAt
				}
			}
		}
		if cn := strings.TrimSpace(req.ContractNo); cn != "" {
			segContractNo = cn
		}
		exist.ContractNo = segContractNo
	}

	exist.BizPhase = phase
	exist.CurrentPeriodStartAt = periodStart
	exist.PlanReleaseAt = req.PlanReleaseAt
	if req.ContractStartAt != nil {
		exist.ContractStartAt = req.ContractStartAt
	} else {
		exist.ContractStartAt = periodStart
	}
	if req.ContractEndAt != nil {
		exist.ContractEndAt = req.ContractEndAt
	} else {
		exist.ContractEndAt = req.PlanReleaseAt
	}
	if lm := strings.TrimSpace(req.LeaseMode); lm != "" {
		exist.LeaseMode = lm
	}
	if req.AllocatedGPUs > 0 {
		exist.AllocatedGPUs = req.AllocatedGPUs
		exist.PlannedGPUs = req.AllocatedGPUs
	}
	if pid := strings.TrimSpace(req.PartitionID); pid != "" {
		exist.PartitionID = pid
	}
	if ticket := strings.TrimSpace(req.ChangeTicketNo); ticket != "" {
		exist.ChangeTicketNo = ticket
	}

	contractHint := "沿用合同"
	if !reuseContract {
		contractHint = "新合同 " + segContractNo
	} else if segContractNo != "" {
		contractHint = "沿用合同 " + segContractNo
	}
	note := fmt.Sprintf("%s %s→%s，%s~%s，%s",
		now.Format("2006-01-02"),
		opsUtils.ComputeBizPhaseLabel(oldPhase),
		opsUtils.ComputeBizPhaseLabel(phase),
		periodStart.Format("2006-01-02"),
		planDay.Format("2006-01-02"),
		contractHint,
	)
	if r := strings.TrimSpace(req.Remark); r != "" {
		note = note + "；" + r
	}
	if exist.PhaseNote == "" {
		exist.PhaseNote = note
	} else {
		exist.PhaseNote = exist.PhaseNote + "\n" + note
	}
	if exist.Remark == "" {
		exist.Remark = note
	} else {
		exist.Remark = exist.Remark + "\n" + note
	}

	exist.PhaseHistory = opsUtils.AppendPhaseSegment(
		exist.PhaseHistory, oldPhase, phase,
		periodStart, req.PlanReleaseAt,
		segContractID, segContractNo, exist.LeaseMode, exist.AllocatedGPUs, note,
	)

	if err := s.validateLease(exist); err != nil {
		return err
	}
	if err := s.ensureNoOverbook(ctx, exist.ServerCode, exist.AllocatedGPUs, exist.ID, exist.OpenedAt, exist.PlanReleaseAt, exist.ActualReleaseAt); err != nil {
		return err
	}
	return s.allocDAO.Update(ctx, exist)
}

func (s *opsComputeService) DeleteAllocation(ctx context.Context, id int) error {
	return s.allocDAO.Delete(ctx, id)
}

func (s *opsComputeService) GetAllocation(ctx context.Context, id int) (*model.OpsComputeAllocation, error) {
	item, err := s.allocDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	assetMap, err := s.loadAssetMap(ctx)
	if err != nil {
		return nil, err
	}
	occupied := s.serverOccupiedMap(ctx, assetMap, nil)
	s.enrichAllocation(ctx, item, assetMap, occupied, time.Now())
	return item, nil
}

func (s *opsComputeService) ListAllocation(ctx context.Context, req *model.ListOpsComputeAllocationReq) (*model.ListResp[*model.OpsComputeAllocation], error) {
	if req == nil {
		req = &model.ListOpsComputeAllocationReq{}
	}
	page, size := req.Page, req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	needFilter := req.LifeStatus != "" || req.OverbookOnly || req.GPUModel != ""
	items, total, err := s.allocDAO.List(ctx, req)
	if err != nil {
		return nil, err
	}
	assetMap, err := s.loadAssetMap(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	occupied := s.serverOccupiedMap(ctx, assetMap, nil)
	out := make([]*model.OpsComputeAllocation, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		s.enrichAllocation(ctx, item, assetMap, occupied, now)
		if req.LifeStatus != "" && item.LifeStatus != req.LifeStatus {
			continue
		}
		if req.GPUModel != "" && item.GPUModel != req.GPUModel {
			continue
		}
		if req.OverbookOnly && item.CapacityCheck != model.OpsComputeCapacityOverbook {
			continue
		}
		out = append(out, item)
	}
	if needFilter {
		total = int64(len(out))
		start := (page - 1) * size
		if start > len(out) {
			start = len(out)
		}
		end := start + size
		if end > len(out) {
			end = len(out)
		}
		out = out[start:end]
	}
	return &model.ListResp[*model.OpsComputeAllocation]{Items: out, Total: total}, nil
}

func (s *opsComputeService) Dashboard(ctx context.Context) (*model.OpsComputeDashboard, error) {
	assets, err := s.assetDAO.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	allocs, err := s.allocDAO.ListActiveLike(ctx)
	if err != nil {
		return nil, err
	}
	assetMap := map[string]*model.OpsComputeAsset{}
	for _, a := range assets {
		assetMap[a.ServerCode] = a
	}
	now := time.Now()
	occupied := s.serverOccupiedMap(ctx, assetMap, allocs)

	type agg struct {
		servers, availableServers, totalGPUs, usedGPUs, overbook int64
		serverSet map[string]struct{}
	}
	byModel := map[string]*agg{}
	ensure := func(m string) *agg {
		if _, ok := byModel[m]; !ok {
			byModel[m] = &agg{serverSet: map[string]struct{}{}}
		}
		return byModel[m]
	}
	for _, a := range assets {
		g := ensure(a.GPUModel)
		g.servers++
		g.serverSet[a.ServerCode] = struct{}{}
		if a.Status == model.OpsComputeAssetAvailable {
			g.availableServers++
		}
		if a.GPUCount != nil {
			g.totalGPUs += int64(*a.GPUCount)
		}
	}

	var inUse, pendingRel, pendingOpen, overbook, missingEvidence int64
	overbookServers := map[string]struct{}{}
	for code, occ := range occupied {
		asset := assetMap[code]
		if asset == nil || asset.GPUCount == nil {
			continue
		}
		if occ > *asset.GPUCount {
			overbookServers[code] = struct{}{}
		}
	}
	overbook = int64(len(overbookServers))

	for _, item := range allocs {
		st := opsUtils.ComputeLifeStatus(item.OpenedAt, item.PlanReleaseAt, item.ActualReleaseAt, now)
		asset := assetMap[item.ServerCode]
		gpuModel := ""
		if asset != nil {
			gpuModel = asset.GPUModel
		}
		switch st {
		case model.OpsComputeLifeInUse:
			inUse++
		case model.OpsComputeLifePendingRel:
			pendingRel++
		case model.OpsComputeLifePendingOpen:
			pendingOpen++
		}
		occ := opsUtils.OccupyingGPUs(st, item.AllocatedGPUs)
		if occ > 0 && gpuModel != "" {
			ensure(gpuModel).usedGPUs += int64(occ)
		}
		if st != model.OpsComputeLifeReleased && !s.hasEvidence(ctx, item) {
			missingEvidence++
		}
	}
	for code := range overbookServers {
		if asset := assetMap[code]; asset != nil {
			ensure(asset.GPUModel).overbook++
		}
	}

	stats := make([]model.OpsComputeModelStat, 0, len(byModel))
	var totalGPUs, usedGPUs int64
	for modelName, g := range byModel {
		avail := g.totalGPUs - g.usedGPUs
		if avail < 0 {
			avail = 0
		}
		totalGPUs += g.totalGPUs
		usedGPUs += g.usedGPUs
		stats = append(stats, model.OpsComputeModelStat{
			GPUModel: modelName, ServerCount: g.servers, TotalGPUs: g.totalGPUs,
			UsedGPUs: g.usedGPUs, AvailableGPUs: avail, AvailableServers: g.availableServers,
			OverbookCount: g.overbook,
		})
	}
	availAll := totalGPUs - usedGPUs
	if availAll < 0 {
		availAll = 0
	}
	return &model.OpsComputeDashboard{
		ByModel: stats, AssetTotal: int64(len(assets)),
		InUseRecords: inUse, PendingOpen: pendingOpen, PendingRelease: pendingRel,
		OverbookCount: overbook, MissingEvidence: missingEvidence,
		TotalGPUs: totalGPUs, UsedGPUs: usedGPUs, AvailableGPUs: availAll,
	}, nil
}

func (s *opsComputeService) buildAllocationFromCreate(ctx context.Context, req *model.CreateOpsComputeAllocationReq) (*model.OpsComputeAllocation, error) {
	if req.ActivationID <= 0 && req.TrialID <= 0 {
		return nil, fmt.Errorf("请关联正式开通单或试用单，便于台账与佐证闭环")
	}
	customer, err := s.customerDAO.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}
	asset, err := s.assetDAO.GetByServerCode(ctx, strings.TrimSpace(req.ServerCode))
	if err != nil {
		return nil, fmt.Errorf("物理服务器不存在，请先在资产清单建档")
	}
	if asset.Status != "" && asset.Status != model.OpsComputeAssetAvailable {
		return nil, fmt.Errorf("服务器 %s 当前状态为「%s」，不可新分配", asset.ServerCode, asset.Status)
	}
	item := &model.OpsComputeAllocation{
		Source: req.Source, ServerCode: strings.TrimSpace(req.ServerCode), PartitionID: strings.TrimSpace(req.PartitionID),
		LeaseMode: req.LeaseMode, AllocatedGPUs: req.AllocatedGPUs, PlannedGPUs: req.PlannedGPUs,
		CustomerID: customer.ID, CustomerName: customer.Name,
		ContractID: req.ContractID, ContractNo: strings.TrimSpace(req.ContractNo),
		ContractStartAt: req.ContractStartAt, ContractEndAt: req.ContractEndAt,
		ActivationID: req.ActivationID, TrialID: req.TrialID,
		ApplyNo: strings.TrimSpace(req.ApplyNo), AuditInstanceID: req.AuditInstanceID,
		AppliedAt: req.AppliedAt, ApprovedAt: req.ApprovedAt, OpenedAt: req.OpenedAt, PlanReleaseAt: req.PlanReleaseAt,
		BizPhase: strings.TrimSpace(req.BizPhase), PhaseNote: strings.TrimSpace(req.PhaseNote),
		ExecutorName: req.ExecutorName, ChangeTicketNo: req.ChangeTicketNo, Remark: req.Remark,
		OperatorID: req.OperatorID, OperatorName: req.OperatorName,
	}
	if item.Source == "" {
		item.Source = model.OpsComputeSourceOpen
	}
	if item.BizPhase == "" {
		if item.ActivationID > 0 {
			item.BizPhase = model.OpsComputePhaseFormal
		} else {
			item.BizPhase = model.OpsComputePhaseTrial
		}
	}
	s.fillFromBiz(ctx, item)
	return item, nil
}

func (s *opsComputeService) buildAllocationFromUpdate(ctx context.Context, req *model.UpdateOpsComputeAllocationReq, exist *model.OpsComputeAllocation) (*model.OpsComputeAllocation, error) {
	if req.ActivationID <= 0 && req.TrialID <= 0 {
		return nil, fmt.Errorf("请关联正式开通单或试用单，便于台账与佐证闭环")
	}
	customer, err := s.customerDAO.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}
	asset, err := s.assetDAO.GetByServerCode(ctx, strings.TrimSpace(req.ServerCode))
	if err != nil {
		return nil, fmt.Errorf("物理服务器不存在，请先在资产清单建档")
	}
	if asset.Status != "" && asset.Status != model.OpsComputeAssetAvailable &&
		strings.TrimSpace(req.ServerCode) != strings.TrimSpace(exist.ServerCode) {
		return nil, fmt.Errorf("服务器 %s 当前状态为「%s」，不可换绑", asset.ServerCode, asset.Status)
	}
	item := &model.OpsComputeAllocation{
		Source: req.Source, ServerCode: strings.TrimSpace(req.ServerCode), PartitionID: strings.TrimSpace(req.PartitionID),
		LeaseMode: req.LeaseMode, AllocatedGPUs: req.AllocatedGPUs, PlannedGPUs: req.PlannedGPUs,
		CustomerID: customer.ID, CustomerName: customer.Name,
		ContractID: req.ContractID, ContractNo: strings.TrimSpace(req.ContractNo),
		ContractStartAt: req.ContractStartAt, ContractEndAt: req.ContractEndAt,
		ActivationID: req.ActivationID, TrialID: req.TrialID,
		ApplyNo: strings.TrimSpace(req.ApplyNo), AuditInstanceID: req.AuditInstanceID,
		AppliedAt: req.AppliedAt, ApprovedAt: req.ApprovedAt, OpenedAt: req.OpenedAt, PlanReleaseAt: req.PlanReleaseAt,
		ActualReleaseAt: exist.ActualReleaseAt,
		BizPhase:        strings.TrimSpace(req.BizPhase),
		PhaseNote:       strings.TrimSpace(req.PhaseNote),
		ExecutorName: req.ExecutorName, ChangeTicketNo: req.ChangeTicketNo, Remark: req.Remark,
		OperatorID: exist.OperatorID, OperatorName: exist.OperatorName,
	}
	if item.Source == "" {
		item.Source = exist.Source
	}
	if item.BizPhase == "" {
		item.BizPhase = exist.BizPhase
	}
	if item.PhaseNote == "" {
		item.PhaseNote = exist.PhaseNote
	}
	s.fillFromBiz(ctx, item)
	return item, nil
}

func (s *opsComputeService) fillFromBiz(ctx context.Context, item *model.OpsComputeAllocation) {
	// 开通单 → 客户/合同/申请单号/台账日期
	if item.ActivationID > 0 {
		if act, err := s.activationDAO.GetByID(ctx, item.ActivationID); err == nil && act != nil {
			if item.CustomerID == 0 {
				item.CustomerID = act.CustomerID
			}
			if item.ContractID == 0 {
				item.ContractID = act.ContractID
			}
			if item.ApplyNo == "" {
				if strings.TrimSpace(act.OrderNo) != "" {
					item.ApplyNo = strings.TrimSpace(act.OrderNo)
				} else {
					item.ApplyNo = fmt.Sprintf("ACT-%d", act.ID)
				}
			}
			if item.ContractNo == "" && strings.TrimSpace(act.ContractNo) != "" {
				item.ContractNo = strings.TrimSpace(act.ContractNo)
			}
			if item.ContractStartAt == nil {
				item.ContractStartAt = act.ContractStartAt
			}
			if item.ContractEndAt == nil {
				item.ContractEndAt = act.ContractEndAt
			}
			if item.OpenedAt == nil {
				item.OpenedAt = act.ContractStartAt
			}
			if item.PlanReleaseAt == nil {
				item.PlanReleaseAt = act.ContractEndAt
			}
			if link, err := s.approvalDAO.GetByBiz(ctx, model.OpsApprovalBizActivation, act.ID); err == nil && link != nil && item.AuditInstanceID == 0 {
				item.AuditInstanceID = link.WorkorderInstanceID
			}
		}
	}

	// 试用单 → 客户/申请单号/合同编号/日期
	if item.TrialID > 0 {
		if trial, err := s.trialDAO.GetByID(ctx, item.TrialID); err == nil && trial != nil {
			if item.CustomerID == 0 {
				item.CustomerID = trial.CustomerID
			}
			if item.ApplyNo == "" {
				if strings.TrimSpace(trial.OrderNo) != "" {
					item.ApplyNo = strings.TrimSpace(trial.OrderNo)
				} else {
					item.ApplyNo = fmt.Sprintf("TRL-%d", trial.ID)
				}
			}
			if item.ContractNo == "" && strings.TrimSpace(trial.ContractNo) != "" {
				item.ContractNo = strings.TrimSpace(trial.ContractNo)
			}
			if item.ContractStartAt == nil {
				item.ContractStartAt = trial.ContractStartAt
			}
			if item.ContractEndAt == nil {
				item.ContractEndAt = trial.ContractEndAt
			}
			if item.OpenedAt == nil {
				item.OpenedAt = trial.ContractStartAt
			}
			if item.PlanReleaseAt == nil {
				item.PlanReleaseAt = trial.ContractEndAt
			}
			if link, err := s.approvalDAO.GetByBiz(ctx, model.OpsApprovalBizTrial, trial.ID); err == nil && link != nil && item.AuditInstanceID == 0 {
				item.AuditInstanceID = link.WorkorderInstanceID
			}
			// 试用合同
			if item.ContractID == 0 {
				if contracts, _, err := s.contractDAO.List(ctx, &model.ListOpsContractReq{
					ListReq: model.ListReq{Page: 1, Size: 20}, CustomerID: trial.CustomerID,
				}); err == nil {
					for _, c := range contracts {
						if c != nil && c.TrialID != nil && *c.TrialID == trial.ID {
							item.ContractID = c.ID
							break
						}
					}
				}
			}
		}
	}

	// 合同 → 编号/日期/试用；并反查开通单
	if item.ContractID > 0 {
		if c, err := s.contractDAO.GetByID(ctx, item.ContractID); err == nil && c != nil {
			if item.CustomerID == 0 {
				item.CustomerID = c.CustomerID
			}
			if item.ContractNo == "" {
				item.ContractNo = strings.TrimSpace(c.ContractNo)
			}
			if item.ContractStartAt == nil {
				item.ContractStartAt = c.StartAt
			}
			if item.ContractEndAt == nil {
				item.ContractEndAt = c.EndAt
			}
			if item.OpenedAt == nil {
				item.OpenedAt = c.StartAt
			}
			if item.PlanReleaseAt == nil {
				item.PlanReleaseAt = c.EndAt
			}
			if item.TrialID == 0 && c.TrialID != nil {
				item.TrialID = *c.TrialID
			}
			if item.ActivationID == 0 {
				if acts, _, err := s.activationDAO.List(ctx, &model.ListOpsActivationReq{
					ListReq: model.ListReq{Page: 1, Size: 20}, CustomerID: c.CustomerID,
				}); err == nil {
					for _, act := range acts {
						if act != nil && act.ContractID == c.ID {
							item.ActivationID = act.ID
							if item.ApplyNo == "" {
								if strings.TrimSpace(act.OrderNo) != "" {
									item.ApplyNo = strings.TrimSpace(act.OrderNo)
								} else {
									item.ApplyNo = fmt.Sprintf("ACT-%d", act.ID)
								}
							}
							if item.ContractNo == "" && strings.TrimSpace(act.ContractNo) != "" {
								item.ContractNo = strings.TrimSpace(act.ContractNo)
							}
							break
						}
					}
				}
			}
		}
	}

	if item.LeaseMode == model.OpsComputeLeaseFull {
		if asset, err := s.assetDAO.GetByServerCode(ctx, item.ServerCode); err == nil && asset != nil && asset.GPUCount != nil && item.AllocatedGPUs <= 0 {
			item.AllocatedGPUs = *asset.GPUCount
		}
	}
}

func (s *opsComputeService) validateLease(item *model.OpsComputeAllocation) error {
	if item.LeaseMode == model.OpsComputeLeaseGPUPool {
		if strings.TrimSpace(item.PartitionID) == "" {
			return fmt.Errorf("按卡池化租赁必须填写卡/分区ID")
		}
		if item.AllocatedGPUs <= 0 {
			return fmt.Errorf("按卡池化租赁必须填写实际分配卡数")
		}
	}
	if item.AllocatedGPUs <= 0 {
		return fmt.Errorf("请填写实际分配GPU卡数")
	}
	return nil
}

func (s *opsComputeService) ensureNoOverbook(ctx context.Context, serverCode string, addGPUs, excludeID int, openedAt, planReleaseAt, actualReleaseAt *time.Time) error {
	asset, err := s.assetDAO.GetByServerCode(ctx, serverCode)
	if err != nil {
		return err
	}
	if asset.GPUCount == nil || *asset.GPUCount <= 0 {
		return fmt.Errorf("服务器 %s 尚未核验单机GPU卡数，请先在资产清单补齐", serverCode)
	}
	now := time.Now()
	life := opsUtils.ComputeLifeStatus(openedAt, planReleaseAt, actualReleaseAt, now)
	add := opsUtils.OccupyingGPUs(life, addGPUs)
	items, err := s.allocDAO.ListByServerCode(ctx, serverCode)
	if err != nil {
		return err
	}
	occupied := 0
	for _, item := range items {
		if item == nil || item.ID == excludeID {
			continue
		}
		st := opsUtils.ComputeLifeStatus(item.OpenedAt, item.PlanReleaseAt, item.ActualReleaseAt, now)
		occupied += opsUtils.OccupyingGPUs(st, item.AllocatedGPUs)
	}
	if occupied+add > *asset.GPUCount {
		return fmt.Errorf("超配：服务器 %s 单机 %d 卡，当前已占用 %d，本次再占 %d", serverCode, *asset.GPUCount, occupied, add)
	}
	return nil
}

func (s *opsComputeService) loadAssetMap(ctx context.Context) (map[string]*model.OpsComputeAsset, error) {
	assets, err := s.assetDAO.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*model.OpsComputeAsset, len(assets))
	for _, a := range assets {
		m[a.ServerCode] = a
	}
	return m, nil
}

func (s *opsComputeService) serverOccupiedMap(ctx context.Context, assetMap map[string]*model.OpsComputeAsset, allocs []*model.OpsComputeAllocation) map[string]int {
	now := time.Now()
	var err error
	if allocs == nil {
		allocs, err = s.allocDAO.ListActiveLike(ctx)
		if err != nil {
			return map[string]int{}
		}
	}
	out := map[string]int{}
	for _, item := range allocs {
		st := opsUtils.ComputeLifeStatus(item.OpenedAt, item.PlanReleaseAt, item.ActualReleaseAt, now)
		out[item.ServerCode] += opsUtils.OccupyingGPUs(st, item.AllocatedGPUs)
	}
	_ = assetMap
	return out
}

func (s *opsComputeService) enrichAllocation(ctx context.Context, item *model.OpsComputeAllocation, assetMap map[string]*model.OpsComputeAsset, occupied map[string]int, now time.Time) {
	opsUtils.SeedPhaseHistoryIfEmpty(item)
	item.LifeStatus = opsUtils.ComputeLifeStatus(item.OpenedAt, item.PlanReleaseAt, item.ActualReleaseAt, now)
	item.LifeStatusLabel = opsUtils.ComputeLifeStatusLabel(item.LifeStatus)
	item.BizPhaseLabel = opsUtils.ComputeBizPhaseLabel(item.BizPhase)
	item.PhaseHistorySummary = opsUtils.FormatPhaseHistorySummary(item.PhaseHistory)
	item.ContractTrailSummary = opsUtils.FormatContractTrailSummary(item.PhaseHistory)
	item.OccupyingGPUs = opsUtils.OccupyingGPUs(item.LifeStatus, item.AllocatedGPUs)
	if asset := assetMap[item.ServerCode]; asset != nil {
		item.GPUModel = asset.GPUModel
		item.ServerGPUCount = asset.GPUCount
	}
	item.ServerOccupied = occupied[item.ServerCode]
	item.CapacityCheck, item.CapacityCheckLabel = opsUtils.CapacityCheck(item.ServerOccupied, item.ServerGPUCount)
	item.EvidenceOK = s.hasEvidence(ctx, item)
}

func (s *opsComputeService) hasEvidence(ctx context.Context, item *model.OpsComputeAllocation) bool {
	ok := false
	if item.ActivationID > 0 {
		if n, err := s.attachmentDAO.CountByBiz(ctx, model.OpsAttachmentBizActivationSheet, item.ActivationID); err == nil {
			item.SheetCount = int(n)
			if n > 0 {
				ok = true
			}
		}
		if n, err := s.attachmentDAO.CountByBiz(ctx, model.OpsAttachmentBizActivationEmail, item.ActivationID); err == nil {
			item.EmailCount = int(n)
			if n > 0 {
				ok = true
			}
		}
	}
	if item.TrialID > 0 {
		if n, err := s.attachmentDAO.CountByBiz(ctx, model.OpsAttachmentBizTrialSheet, item.TrialID); err == nil {
			item.SheetCount += int(n)
			if n > 0 {
				ok = true
			}
		}
		if n, err := s.attachmentDAO.CountByBiz(ctx, model.OpsAttachmentBizTrialEmail, item.TrialID); err == nil {
			item.EmailCount += int(n)
			if n > 0 {
				ok = true
			}
		}
	}
	return ok
}
