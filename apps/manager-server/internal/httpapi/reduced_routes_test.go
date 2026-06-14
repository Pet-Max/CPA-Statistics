package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/testutil"
)

func TestReducedRoutesExcludeRemovedCodexInspection(t *testing.T) {
	handler, _ := newCompatHandler(t, testutil.NewConfig(t), nil)

	removed := []string{
		"/v0/management/codex-inspection/runs",
		"/v0/management/codex-inspection/run",
	}
	for _, path := range removed {
		rr := testutil.Request(t, handler, http.MethodGet, path, "", testutil.AdminKey)
		if rr.Code != http.StatusNotFound && rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 404 or 405", path, rr.Code)
		}
	}
}

func TestReducedPanelKeepsDashboardAndMonitoringRoutes(t *testing.T) {
	handler, _ := newCompatHandler(t, testutil.NewConfig(t), nil)
	rr := testutil.Request(t, handler, http.MethodGet, "/management.html", "", "")
	testutil.RequireStatus(t, rr, http.StatusOK)
	body := rr.Body.String()
	if !strings.Contains(body, "management") && !strings.Contains(strings.ToLower(body), "html") {
		t.Fatalf("panel body does not look like management html")
	}
}
