package modelprice

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/app"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/middleware"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/response"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/model"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
)

type Handler struct {
	App *app.Context
}

type pricesRequest struct {
	Prices map[string]store.ModelPrice `json:"prices"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if !middleware.AuthorizeAdmin(w, r, h.App.AdminAuthService) {
		return
	}
	cleanPath := strings.TrimRight(r.URL.Path, "/")
	switch {
	case cleanPath == "/v0/management/model-prices":
		h.handlePrices(w, r)
	case cleanPath == "/v0/management/model-prices/sync":
		h.handleSync(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handlePrices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		prices, err := h.App.Store.LoadModelPrices(r.Context())
		if err != nil {
			response.Error(w, response.ModelPriceErrorStatus(err), err)
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"prices": prices})
	case http.MethodPut:
		var req pricesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		if req.Prices == nil {
			response.Error(w, http.StatusBadRequest, errors.New("prices are required"))
			return
		}
		if err := h.App.Store.SaveModelPrices(r.Context(), req.Prices); err != nil {
			response.Error(w, response.ModelPriceErrorStatus(err), err)
			return
		}
		prices, err := h.App.Store.LoadModelPrices(r.Context())
		if err != nil {
			response.Error(w, response.ModelPriceErrorStatus(err), err)
			return
		}
		response.JSON(w, http.StatusOK, map[string]any{"prices": prices})
	default:
		response.MethodNotAllowed(w)
	}
}

func (h *Handler) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.MethodNotAllowed(w)
		return
	}
	var req model.ModelPriceSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		response.Error(w, http.StatusBadRequest, err)
		return
	}
	// 精简版不再保留外部价格同步入口。这里保留兼容响应，避免请求监控页因旧前端逻辑报 404；
	// 成本统计仍使用本地保存的 model_prices 表。
	prices, err := h.App.Store.LoadModelPrices(r.Context())
	if err != nil {
		response.Error(w, response.ModelPriceErrorStatus(err), err)
		return
	}
	response.JSON(w, http.StatusOK, model.ModelPriceSyncResponse{
		Prices:     prices,
		Imported:   0,
		Skipped:    0,
		Candidates: []model.ModelPriceSyncCandidateSet{},
		Unmatched:  unmatchedModels(req.Models, prices),
		SourceResults: []model.ModelPriceSyncSourceResult{{
			Source:  "local",
			Models:  len(prices),
			Skipped: 0,
		}},
		ProxyUsed: false,
	})
}

func unmatchedModels(models []string, prices map[string]store.ModelPrice) []string {
	seen := make(map[string]struct{}, len(models))
	unmatched := make([]string, 0)
	for _, raw := range models {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, ok := prices[name]; !ok {
			unmatched = append(unmatched, name)
		}
	}
	return unmatched
}
