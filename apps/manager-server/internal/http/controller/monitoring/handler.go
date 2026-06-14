package monitoring

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/app"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/middleware"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/response"
	monitoringsvc "github.com/seakee/cpa-statistics/apps/manager-server/internal/service/monitoring"
)

type Handler struct {
	App *app.Context
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if !middleware.AuthorizePanel(w, r, h.App.AdminAuthService) {
		return
	}

	path := strings.TrimRight(r.URL.Path, "/")
	if path != "/v0/management/monitoring/analytics" {
		response.MethodNotAllowed(w)
		return
	}
	if r.Method != http.MethodPost {
		response.MethodNotAllowed(w)
		return
	}

	var req monitoringsvc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, err)
		return
	}
	if err := validateRequest(req); err != nil {
		response.Error(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.App.MonitoringService.Analytics(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func validateRequest(req monitoringsvc.Request) error {
	if req.FromMS <= 0 || req.ToMS <= 0 || req.FromMS >= req.ToMS {
		return errors.New("from_ms and to_ms are required and from_ms must be less than to_ms")
	}
	if req.Include.EventsPage != nil && req.Include.EventsPage.Limit > 50000 {
		return errors.New("events_page.limit must be less than or equal to 50000")
	}
	return nil
}
