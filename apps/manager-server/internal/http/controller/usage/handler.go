package usage

import (
	"io"
	"net/http"
	"strings"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/app"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/middleware"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/response"
)

type Handler struct {
	App *app.Context
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if !middleware.AuthorizePanel(w, r, h.App.AdminAuthService) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if strings.HasSuffix(r.URL.Path, "/export") {
			h.Export(w, r)
			return
		}
		payload, err := h.App.UsageService.GetCompatibleUsage(r.Context(), h.App.Config.QueryLimit)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		response.JSON(w, http.StatusOK, payload)
	case http.MethodPost:
		if strings.HasSuffix(r.URL.Path, "/import") {
			h.Import(w, r)
			return
		}
		response.MethodNotAllowed(w)
	default:
		response.MethodNotAllowed(w)
	}
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	data, err := h.App.UsageService.Export(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Disposition", `attachment; filename="usage-events.jsonl"`)
	_, _ = w.Write(data)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err)
		return
	}

	result, parsed, err := h.App.UsageService.Import(r.Context(), data)
	if err != nil {
		if parsed != nil && len(parsed.Events) > 0 {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		response.JSON(w, http.StatusBadRequest, map[string]any{
			"error":       err.Error(),
			"format":      parsed.Format,
			"failed":      parsed.Failed,
			"unsupported": parsed.Unsupported,
			"warnings":    parsed.Warnings,
		})
		return
	}
	response.JSON(w, http.StatusOK, result)
}
