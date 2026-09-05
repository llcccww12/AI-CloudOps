package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
)

func (h *OpsHandler) CreateComputeAsset(ctx *gin.Context) {
	var req model.CreateOpsComputeAssetReq
	u := userClaims(ctx)
	req.OperatorID, req.OperatorName = u.Uid, u.Username
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.computeSvc.CreateAsset(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) UpdateComputeAsset(ctx *gin.Context) {
	var req model.UpdateOpsComputeAssetReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.computeSvc.UpdateAsset(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) DeleteComputeAsset(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.computeSvc.DeleteAsset(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) GetComputeAsset(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.computeSvc.GetAsset(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) ListComputeAsset(ctx *gin.Context) {
	var req model.ListOpsComputeAssetReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.computeSvc.ListAsset(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) SeedComputeAssets(ctx *gin.Context) {
	u := userClaims(ctx)
	base.HandleRequest(ctx, nil, func() (any, error) {
		n, err := h.computeSvc.SeedDefaultAssets(ctx.Request.Context(), u.Uid, u.Username)
		if err != nil {
			return nil, err
		}
		return gin.H{"seeded": n}, nil
	})
}

func (h *OpsHandler) CreateComputeAllocation(ctx *gin.Context) {
	var req model.CreateOpsComputeAllocationReq
	u := userClaims(ctx)
	req.OperatorID, req.OperatorName = u.Uid, u.Username
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.computeSvc.CreateAllocation(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) UpdateComputeAllocation(ctx *gin.Context) {
	var req model.UpdateOpsComputeAllocationReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.computeSvc.UpdateAllocation(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) ReleaseComputeAllocation(ctx *gin.Context) {
	var req model.ReleaseOpsComputeAllocationReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	u := userClaims(ctx)
	req.OperatorID, req.OperatorName = u.Uid, u.Username
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.computeSvc.ReleaseAllocation(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) ExtendComputeAllocation(ctx *gin.Context) {
	var req model.ExtendOpsComputeAllocationReq
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	req.ID = id
	u := userClaims(ctx)
	req.OperatorID, req.OperatorName = u.Uid, u.Username
	base.HandleRequest(ctx, &req, func() (any, error) {
		return nil, h.computeSvc.ExtendAllocation(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) DeleteComputeAllocation(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return nil, h.computeSvc.DeleteAllocation(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) GetComputeAllocation(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil {
		return
	}
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.computeSvc.GetAllocation(ctx.Request.Context(), id)
	})
}

func (h *OpsHandler) ListComputeAllocation(ctx *gin.Context) {
	var req model.ListOpsComputeAllocationReq
	base.HandleRequest(ctx, &req, func() (any, error) {
		return h.computeSvc.ListAllocation(ctx.Request.Context(), &req)
	})
}

func (h *OpsHandler) GetComputeDashboard(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (any, error) {
		return h.computeSvc.Dashboard(ctx.Request.Context())
	})
}
