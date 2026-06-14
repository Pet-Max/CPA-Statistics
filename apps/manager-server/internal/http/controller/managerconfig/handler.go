package managerconfig

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/app"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/response"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
)

type Handler struct {
	App *app.Context
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if !h.authorizeRead(w, r) {
			return
		}
		result, err := h.App.ManagerConfigService.Get(r.Context())
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		response.JSON(w, http.StatusOK, result)
	case http.MethodPut:
		var rawReq struct {
			Config json.RawMessage `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		var req struct {
			Config store.ManagerConfig
		}
		if len(rawReq.Config) > 0 {
			if err := json.Unmarshal(rawReq.Config, &req.Config); err != nil {
				response.Error(w, http.StatusBadRequest, err)
				return
			}
			var rawConfig map[string]json.RawMessage
			if err := json.Unmarshal(rawReq.Config, &rawConfig); err == nil {
				_, req.Config.ExternalUsageServiceSubmitted = rawConfig["externalUsageService"]
			}
		}
		ok, err := h.App.AdminAuthService.VerifySubmittedExternalConfigHeader(
			r.Context(),
			r.Header.Get("Authorization"),
			req.Config,
		)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			response.Error(w, http.StatusUnauthorized, errors.New("invalid admin key"))
			return
		}
		result, err := h.App.ManagerConfigService.Update(r.Context(), req.Config)
		if err != nil {
			response.Error(w, response.ManagerConfigErrorStatus(err), err)
			return
		}
		response.JSON(w, http.StatusOK, result)
	default:
		response.MethodNotAllowed(w)
	}
}

func (h *Handler) authorizeRead(w http.ResponseWriter, r *http.Request) bool {
	ok, err := h.App.AdminAuthService.VerifyPanelHeader(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return false
	}
	if ok {
		return true
	}
	setup, setupOK, err := h.App.ManagerConfigService.ResolveSetup(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return false
	}
	if !setupOK || setup.ManagementKey == "" {
		return true
	}
	response.Error(w, http.StatusUnauthorized, errors.New("invalid admin key"))
	return false
}
