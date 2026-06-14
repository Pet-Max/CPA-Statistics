package data

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/app"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/middleware"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/response"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
)

const clearDataConfirmation = "CLEAR DATA"

type Handler struct {
	App *app.Context
}

type clearDataScopeRequest struct {
	UsageEvents bool `json:"usageEvents"`
	DeadLetters bool `json:"deadLetters"`
}

type clearDataRequest struct {
	Confirmation string                `json:"confirmation"`
	Scope        clearDataScopeRequest `json:"scope"`
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if !middleware.AuthorizeAdmin(w, r, h.App.AdminAuthService) {
		return
	}
	cleanPath := strings.TrimRight(r.URL.Path, "/")
	switch cleanPath {
	case "/v0/management/data/clear":
		h.handleClear(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.MethodNotAllowed(w)
		return
	}

	var req clearDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Confirmation) != clearDataConfirmation {
		response.Error(w, http.StatusBadRequest, errors.New("clear data confirmation is required"))
		return
	}

	opts := store.ClearDataOptions{
		UsageEvents: req.Scope.UsageEvents,
		DeadLetters: req.Scope.DeadLetters,
	}
	if !opts.UsageEvents && !opts.DeadLetters {
		response.Error(w, http.StatusBadRequest, errors.New("clear data scope is required"))
		return
	}

	cleared, err := h.App.Store.ClearData(r.Context(), opts)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"cleared": cleared,
	})
}
