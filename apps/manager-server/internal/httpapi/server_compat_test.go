package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/collector"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/config"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/testutil"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/usage"
)

func newCompatHandler(t *testing.T, cfg config.Config, setup *store.Setup) (http.Handler, *store.Store) {
	t.Helper()
	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Join(t.TempDir(), "usage.sqlite")
	}
	if cfg.Queue == "" {
		cfg.Queue = "usage"
	}
	if cfg.PopSide == "" {
		cfg.PopSide = "right"
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.QueryLimit == 0 {
		cfg.QueryLimit = 50000
	}
	if len(cfg.CORSOrigins) == 0 {
		cfg.CORSOrigins = []string{"*"}
	}
	if cfg.CollectorMode == "" {
		cfg.CollectorMode = "auto"
	}

	db := testutil.NewStore(t, cfg)
	if setup != nil {
		if err := db.SaveSetup(context.Background(), *setup); err != nil {
			t.Fatalf("save setup: %v", err)
		}
	}
	manager := collector.NewManager(cfg, db)
	return New(cfg, db, manager).Handler(), db
}

func TestServerCompatHealthInfoAndPanel(t *testing.T) {
	cfg := testutil.NewConfig(t)
	handler, _ := newCompatHandler(t, cfg, nil)

	healthRR := testutil.Request(t, handler, http.MethodGet, "/health", "", "")
	testutil.RequireStatus(t, healthRR, http.StatusOK)
	var health struct {
		OK      bool   `json:"ok"`
		Service string `json:"service"`
	}
	testutil.DecodeJSON(t, healthRR, &health)
	if !health.OK || health.Service == "" {
		t.Fatalf("health response = %#v", health)
	}

	infoRR := testutil.Request(t, handler, http.MethodGet, "/usage-service/info", "", "")
	testutil.RequireStatus(t, infoRR, http.StatusOK)
	var info struct {
		Service    string `json:"service"`
		Mode       string `json:"mode"`
		StartedAt  int64  `json:"startedAt"`
		Configured bool   `json:"configured"`
	}
	testutil.DecodeJSON(t, infoRR, &info)
	if info.Service != serviceID || info.Mode != "embedded" || info.StartedAt <= 0 || info.Configured {
		t.Fatalf("info response = %#v", info)
	}

	rootRR := testutil.Request(t, handler, http.MethodGet, "/", "", "")
	testutil.RequireStatus(t, rootRR, http.StatusTemporaryRedirect)
	if rootRR.Header().Get("Location") != "/management.html" {
		t.Fatalf("root location = %q", rootRR.Header().Get("Location"))
	}

	panelRR := testutil.Request(t, handler, http.MethodGet, "/management.html", "", "")
	testutil.RequireStatus(t, panelRR, http.StatusOK)
	if !strings.Contains(panelRR.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("panel content type = %q", panelRR.Header().Get("Content-Type"))
	}
	if !strings.Contains(strings.ToLower(panelRR.Body.String()), "<html") {
		t.Fatalf("panel body does not look like html")
	}
}

func TestServerCompatPanelPathOverridesEmbeddedPanel(t *testing.T) {
	cfg := testutil.NewConfig(t)
	panelPath := filepath.Join(t.TempDir(), "management.html")
	if err := osWriteFile(panelPath, []byte("<html><body>custom panel</body></html>")); err != nil {
		t.Fatalf("write panel: %v", err)
	}
	cfg.PanelPath = panelPath
	handler, _ := newCompatHandler(t, cfg, nil)

	rr := testutil.Request(t, handler, http.MethodGet, "/management.html", "", "")
	testutil.RequireStatus(t, rr, http.StatusOK)
	if rr.Body.String() != "<html><body>custom panel</body></html>" {
		t.Fatalf("panel body = %q", rr.Body.String())
	}
}

func TestServerCompatSetupConfigAndEnvLock(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	cfg := testutil.NewConfig(t)
	handler, db := newCompatHandler(t, cfg, nil)

	setupBody := `{"cpaBaseUrl":"` + cpa.URL() + `","managementKey":"management-key","requestMonitoringEnabled":false,"ensureUsageStatisticsEnabled":false}`
	setupRR := testutil.Request(t, handler, http.MethodPost, "/setup", setupBody, testutil.AdminKey)
	testutil.RequireStatus(t, setupRR, http.StatusOK)
	if !strings.Contains(setupRR.Body.String(), `"ok":true`) || !strings.Contains(setupRR.Body.String(), cpa.URL()) {
		t.Fatalf("setup body = %s", setupRR.Body.String())
	}

	infoRR := testutil.Request(t, handler, http.MethodGet, "/usage-service/info", "", "")
	testutil.RequireStatus(t, infoRR, http.StatusOK)
	var info struct {
		Configured bool `json:"configured"`
	}
	testutil.DecodeJSON(t, infoRR, &info)
	if !info.Configured {
		t.Fatalf("configured = false after setup")
	}
	state, ok, err := db.LoadBootstrapState(context.Background())
	if err != nil || !ok {
		t.Fatalf("load bootstrap state ok=%v err=%v", ok, err)
	}
	if !state.ProjectInitialized || !state.AdminReady || !state.DataKeyReady || state.Status != "ready" {
		t.Fatalf("bootstrap state after setup = %#v", state)
	}

	configRR := testutil.Request(t, handler, http.MethodGet, "/usage-service/config", "", testutil.AdminKey)
	testutil.RequireStatus(t, configRR, http.StatusOK)
	if !strings.Contains(configRR.Body.String(), `"source":"db"`) ||
		!strings.Contains(configRR.Body.String(), `"cpaBaseUrl":"`+cpa.URL()+`"`) ||
		!strings.Contains(configRR.Body.String(), `"cpaUsage"`) {
		t.Fatalf("config body = %s", configRR.Body.String())
	}

	updateBody := `{"config":{"cpaConnection":{"cpaBaseUrl":"` + cpa.URL() + `","managementKey":"management-key"},"collector":{"enabled":false,"collectorMode":"auto","queue":"usage","popSide":"right","batchSize":100,"pollIntervalMs":500,"queryLimit":50000},"externalUsageService":{"enabled":true,"serviceBase":"http://usage.local"}}}`
	updateRR := testutil.Request(t, handler, http.MethodPut, "/usage-service/config", updateBody, testutil.AdminKey)
	testutil.RequireStatus(t, updateRR, http.StatusOK)
	if !strings.Contains(updateRR.Body.String(), `"externalUsageService":{"enabled":true,"serviceBase":"http://usage.local"}`) {
		t.Fatalf("updated config body = %s", updateRR.Body.String())
	}

	cpa.ManagementKey = "rotated-management-key"
	rotateKeyBody := `{"config":{"cpaConnection":{"cpaBaseUrl":"` + cpa.URL() + `","managementKey":"rotated-management-key"},"collector":{"enabled":false,"collectorMode":"auto","queue":"usage","popSide":"right","batchSize":100,"pollIntervalMs":500,"queryLimit":50000}}}`
	rotateKeyRR := testutil.Request(t, handler, http.MethodPut, "/usage-service/config", rotateKeyBody, testutil.AdminKey)
	testutil.RequireStatus(t, rotateKeyRR, http.StatusOK)
	if !strings.Contains(rotateKeyRR.Body.String(), `"cpaBaseUrl":"`+cpa.URL()+`"`) {
		t.Fatalf("rotated key config body = %s", rotateKeyRR.Body.String())
	}
	if !strings.Contains(rotateKeyRR.Body.String(), `"externalUsageService":{"enabled":true,"serviceBase":"http://usage.local"}`) {
		t.Fatalf("rotated key cleared external usage service: %s", rotateKeyRR.Body.String())
	}
	rotatedSetup, ok, err := db.LoadSetup(context.Background())
	if err != nil || !ok {
		t.Fatalf("load rotated setup ok=%v err=%v", ok, err)
	}
	if rotatedSetup.CPAUpstreamURL != cpa.URL() || rotatedSetup.ManagementKey != "rotated-management-key" {
		t.Fatalf("rotated setup = %#v", rotatedSetup)
	}

	disableExternalBody := `{"config":{"cpaConnection":{"cpaBaseUrl":"` + cpa.URL() + `","managementKey":"rotated-management-key"},"collector":{"enabled":false},"externalUsageService":{"enabled":false,"serviceBase":""}}}`
	disableExternalRR := testutil.Request(t, handler, http.MethodPut, "/usage-service/config", disableExternalBody, testutil.AdminKey)
	testutil.RequireStatus(t, disableExternalRR, http.StatusOK)
	if !strings.Contains(disableExternalRR.Body.String(), `"externalUsageService":{"enabled":false}`) ||
		strings.Contains(disableExternalRR.Body.String(), "http://usage.local") {
		t.Fatalf("disable external usage service body = %s", disableExternalRR.Body.String())
	}

	otherCPA := testutil.NewCPAMock(t)
	otherCPA.ManagementKey = "other-key"
	rebindBody := `{"config":{"cpaConnection":{"cpaBaseUrl":"` + otherCPA.URL() + `","managementKey":"other-key"},"collector":{"enabled":false}}}`
	rebindRR := testutil.Request(t, handler, http.MethodPut, "/usage-service/config", rebindBody, testutil.AdminKey)
	testutil.RequireStatus(t, rebindRR, http.StatusOK)
	if !strings.Contains(rebindRR.Body.String(), `"cpaBaseUrl":"`+otherCPA.URL()+`"`) {
		t.Fatalf("rebind body = %s", rebindRR.Body.String())
	}
	reboundSetup, ok, err := db.LoadSetup(context.Background())
	if err != nil || !ok {
		t.Fatalf("load rebound setup ok=%v err=%v", ok, err)
	}
	if reboundSetup.CPAUpstreamURL != otherCPA.URL() || reboundSetup.ManagementKey != "other-key" {
		t.Fatalf("rebound setup = %#v", reboundSetup)
	}

	envCfg := testutil.NewConfig(t)
	envCfg.CPAUpstreamURL = cpa.URL()
	envCfg.ManagementKey = "management-key"
	envHandler, _ := newCompatHandler(t, envCfg, nil)
	conflictBody := `{"config":{"cpaConnection":{"cpaBaseUrl":"http://other.local","managementKey":"other-key"},"collector":{"enabled":false}}}`
	conflictRR := testutil.Request(t, envHandler, http.MethodPut, "/usage-service/config", conflictBody, testutil.AdminKey)
	testutil.RequireStatus(t, conflictRR, http.StatusConflict)
	if !strings.Contains(conflictRR.Body.String(), `"code":"connection_env_managed"`) {
		t.Fatalf("conflict body = %s", conflictRR.Body.String())
	}
}

func TestServerCompatInfoIgnoresStaleUninitializedBootstrapState(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	setup := &store.Setup{CPAUpstreamURL: cpa.URL(), ManagementKey: "management-key", Queue: "usage", PopSide: "right"}
	handler, db := newCompatHandler(t, testutil.NewConfig(t), setup)
	if err := db.SaveBootstrapState(context.Background(), store.BootstrapState{
		Version:            1,
		Status:             "fresh",
		AdminReady:         true,
		ProjectInitialized: false,
		DataKeyReady:       true,
	}); err != nil {
		t.Fatalf("save stale bootstrap state: %v", err)
	}

	infoRR := testutil.Request(t, handler, http.MethodGet, "/usage-service/info", "", "")
	testutil.RequireStatus(t, infoRR, http.StatusOK)
	var info struct {
		Configured         bool `json:"configured"`
		ProjectInitialized bool `json:"projectInitialized"`
		SetupRequired      bool `json:"setupRequired"`
	}
	testutil.DecodeJSON(t, infoRR, &info)
	if !info.Configured || !info.ProjectInitialized || info.SetupRequired {
		t.Fatalf("info response = %#v", info)
	}
}

func TestServerCompatCPAPanelKeyCannotUseManagerOnlyRoutes(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	cfg := testutil.NewConfig(t)
	handler, db := newCompatHandler(t, cfg, nil)

	openConfigRR := testutil.Request(t, handler, http.MethodGet, "/usage-service/config", "", "")
	testutil.RequireStatus(t, openConfigRR, http.StatusOK)

	configBody := `{"config":{"cpaConnection":{"cpaBaseUrl":"` + cpa.URL() + `","managementKey":"management-key"},"collector":{"enabled":false,"collectorMode":"auto","queue":"usage","popSide":"right","batchSize":100,"pollIntervalMs":500,"queryLimit":50000},"externalUsageService":{"enabled":true,"serviceBase":"http://usage.local"}}}`
	saveRR := testutil.Request(t, handler, http.MethodPut, "/usage-service/config", configBody, testutil.AdminKey)
	testutil.RequireStatus(t, saveRR, http.StatusOK)
	if !strings.Contains(saveRR.Body.String(), `"externalUsageService":{"enabled":true,"serviceBase":"http://usage.local"}`) {
		t.Fatalf("save body = %s", saveRR.Body.String())
	}

	cpaKeyConfigRR := testutil.Request(t, handler, http.MethodGet, "/usage-service/config", "", "management-key")
	testutil.RequireStatus(t, cpaKeyConfigRR, http.StatusUnauthorized)
	if !strings.Contains(cpaKeyConfigRR.Body.String(), `"code":"invalid_admin_key"`) {
		t.Fatalf("CPA key config body = %s", cpaKeyConfigRR.Body.String())
	}

	configRR := testutil.Request(t, handler, http.MethodGet, "/usage-service/config", "", testutil.AdminKey)
	testutil.RequireStatus(t, configRR, http.StatusOK)
	if !strings.Contains(configRR.Body.String(), `"source":"db"`) ||
		!strings.Contains(configRR.Body.String(), `"cpaBaseUrl":"`+cpa.URL()+`"`) {
		t.Fatalf("config body = %s", configRR.Body.String())
	}

	if _, err := db.InsertEvents(context.Background(), []usage.Event{compatEvent("external-panel-usage", 10)}); err != nil {
		t.Fatalf("insert event: %v", err)
	}
	usageRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/usage", "", "management-key")
	testutil.RequireStatus(t, usageRR, http.StatusUnauthorized)
	if !strings.Contains(usageRR.Body.String(), `"code":"invalid_admin_key"`) {
		t.Fatalf("usage body = %s", usageRR.Body.String())
	}

	proxyRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/config", "", "management-key")
	testutil.RequireStatus(t, proxyRR, http.StatusUnauthorized)
	if !strings.Contains(proxyRR.Body.String(), `"code":"invalid_admin_key"`) {
		t.Fatalf("proxy body = %s", proxyRR.Body.String())
	}
}

func TestServerCompatStatusAuthAndCounts(t *testing.T) {
	cfg := testutil.NewConfig(t)
	unconfiguredHandler, _ := newCompatHandler(t, cfg, nil)
	openRR := testutil.Request(t, unconfiguredHandler, http.MethodGet, "/status", "", "")
	testutil.RequireStatus(t, openRR, http.StatusUnauthorized)
	authorizedOpenRR := testutil.Request(t, unconfiguredHandler, http.MethodGet, "/status", "", testutil.AdminKey)
	testutil.RequireStatus(t, authorizedOpenRR, http.StatusOK)

	cpa := testutil.NewCPAMock(t)
	setup := &store.Setup{CPAUpstreamURL: cpa.URL(), ManagementKey: "management-key", Queue: "usage", PopSide: "right"}
	configuredHandler, db := newCompatHandler(t, testutil.NewConfig(t), setup)
	if err := db.AddDeadLetter(context.Background(), `{"bad":true}`, errors.New("parse failed")); err != nil {
		t.Fatalf("add dead letter: %v", err)
	}
	_, err := db.InsertEvents(context.Background(), []usage.Event{compatEvent("status-event", 1)})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	unauthorizedRR := testutil.Request(t, configuredHandler, http.MethodGet, "/status", "", "")
	testutil.RequireStatus(t, unauthorizedRR, http.StatusUnauthorized)

	statusRR := testutil.Request(t, configuredHandler, http.MethodGet, "/status", "", testutil.AdminKey)
	testutil.RequireStatus(t, statusRR, http.StatusOK)
	if !strings.Contains(statusRR.Body.String(), `"events":1`) ||
		!strings.Contains(statusRR.Body.String(), `"deadLetters":1`) ||
		!strings.Contains(statusRR.Body.String(), `"collector"`) {
		t.Fatalf("status body = %s", statusRR.Body.String())
	}
}

func TestServerCompatUsageRoutes(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	setup := &store.Setup{CPAUpstreamURL: cpa.URL(), ManagementKey: "management-key", Queue: "usage", PopSide: "right"}
	handler, db := newCompatHandler(t, testutil.NewConfig(t), setup)

	emptyRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/usage", "", testutil.AdminKey)
	testutil.RequireStatus(t, emptyRR, http.StatusOK)
	if !strings.Contains(emptyRR.Body.String(), `"total_requests":0`) {
		t.Fatalf("empty usage body = %s", emptyRR.Body.String())
	}

	_, err := db.InsertEvents(context.Background(), []usage.Event{compatEvent("usage-event-1", 10)})
	if err != nil {
		t.Fatalf("insert usage event: %v", err)
	}
	usageRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/usage", "", testutil.AdminKey)
	testutil.RequireStatus(t, usageRR, http.StatusOK)
	if !strings.Contains(usageRR.Body.String(), `"total_requests":1`) ||
		!strings.Contains(usageRR.Body.String(), `"gpt-test"`) {
		t.Fatalf("usage body = %s", usageRR.Body.String())
	}

	exportRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/usage/export", "", testutil.AdminKey)
	testutil.RequireStatus(t, exportRR, http.StatusOK)
	if !strings.Contains(exportRR.Header().Get("Content-Type"), "application/x-ndjson") ||
		!strings.Contains(exportRR.Body.String(), `"event_hash":"usage-event-1"`) {
		t.Fatalf("export content type = %q body = %s", exportRR.Header().Get("Content-Type"), exportRR.Body.String())
	}

	importLine := `{"event_hash":"usage-event-2","timestamp_ms":1778000001000,"timestamp":"2026-05-06T00:00:01Z","model":"gpt-test","endpoint":"POST /v1/chat/completions","input_tokens":2,"output_tokens":3,"total_tokens":5,"failed":false}`
	importRR := testutil.Request(t, handler, http.MethodPost, "/v0/management/usage/import", importLine+"\n", testutil.AdminKey)
	testutil.RequireStatus(t, importRR, http.StatusOK)
	if !strings.Contains(importRR.Body.String(), `"format":"usage_service_jsonl"`) ||
		!strings.Contains(importRR.Body.String(), `"added":1`) {
		t.Fatalf("import body = %s", importRR.Body.String())
	}
}

func TestServerCompatModelPriceSyncReturnsStableContract(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	setup := &store.Setup{CPAUpstreamURL: cpa.URL(), ManagementKey: "management-key", Queue: "usage", PopSide: "right"}
	handler, db := newCompatHandler(t, testutil.NewConfig(t), setup)
	if err := db.SaveModelPrices(context.Background(), map[string]store.ModelPrice{
		"gpt-test": {Prompt: 1, Completion: 2, Cache: 0.5},
	}); err != nil {
		t.Fatalf("save model prices: %v", err)
	}

	rr := testutil.Request(t, handler, http.MethodPost, "/v0/management/model-prices/sync", `{"models":["gpt-test","missing-model","missing-model"," "]}`, testutil.AdminKey)
	testutil.RequireStatus(t, rr, http.StatusOK)
	var payload struct {
		Prices        map[string]store.ModelPrice `json:"prices"`
		Imported      int                         `json:"imported"`
		Skipped       int                         `json:"skipped"`
		Candidates    []struct{}                  `json:"candidates"`
		Unmatched     []string                    `json:"unmatched"`
		SourceResults []struct {
			Source  string `json:"source"`
			Models  int    `json:"models"`
			Skipped int    `json:"skipped"`
		} `json:"sourceResults"`
		ProxyUsed bool `json:"proxyUsed"`
	}
	testutil.DecodeJSON(t, rr, &payload)
	if len(payload.Prices) != 1 || payload.Prices["gpt-test"].Prompt != 1 {
		t.Fatalf("sync prices = %#v", payload.Prices)
	}
	if payload.Imported != 0 || payload.Skipped != 0 || len(payload.Candidates) != 0 ||
		len(payload.Unmatched) != 1 || payload.Unmatched[0] != "missing-model" ||
		len(payload.SourceResults) != 1 || payload.SourceResults[0].Source != "local" ||
		payload.SourceResults[0].Models != 1 || payload.SourceResults[0].Skipped != 0 ||
		payload.ProxyUsed {
		t.Fatalf("sync payload = %#v", payload)
	}
}

func TestServerCompatDashboardSummary(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	setup := &store.Setup{CPAUpstreamURL: cpa.URL(), ManagementKey: "management-key", Queue: "usage", PopSide: "right"}
	handler, db := newCompatHandler(t, testutil.NewConfig(t), setup)
	todayStart := int64(1_778_000_000_000)
	nowMS := todayStart + 60_000
	latency := int64(88)

	if err := db.SaveModelPrices(context.Background(), map[string]store.ModelPrice{
		"gpt-test": {Prompt: 1, Completion: 2, Cache: 0.5},
	}); err != nil {
		t.Fatalf("save model prices: %v", err)
	}
	success := compatEvent("dashboard-success", 10)
	success.LatencyMS = &latency
	failure := compatEvent("dashboard-failure", 20)
	failure.Failed = true
	_, err := db.InsertEvents(context.Background(), []usage.Event{success, failure})
	if err != nil {
		t.Fatalf("insert events: %v", err)
	}

	unauthorizedRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/dashboard/summary?today_start_ms=1778000000000", "", "")
	testutil.RequireStatus(t, unauthorizedRR, http.StatusUnauthorized)

	badRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/dashboard/summary", "", testutil.AdminKey)
	testutil.RequireStatus(t, badRR, http.StatusBadRequest)

	target := "/v0/management/dashboard/summary?today_start_ms=1778000000000&now_ms=" + strconv.FormatInt(nowMS, 10)
	rr := testutil.Request(t, handler, http.MethodGet, target, "", testutil.AdminKey)
	testutil.RequireStatus(t, rr, http.StatusOK)
	var payload struct {
		Today struct {
			TotalCalls       int64    `json:"total_calls"`
			SuccessCalls     int64    `json:"success_calls"`
			FailureCalls     int64    `json:"failure_calls"`
			AverageLatencyMS *float64 `json:"average_latency_ms"`
		} `json:"today"`
		TopModelsToday []struct {
			Model string `json:"model"`
			Calls int64  `json:"calls"`
		} `json:"top_models_today"`
		RecentFailures []struct {
			Model string `json:"model"`
		} `json:"recent_failures"`
	}
	testutil.DecodeJSON(t, rr, &payload)
	if payload.Today.TotalCalls != 2 || payload.Today.SuccessCalls != 1 || payload.Today.FailureCalls != 1 ||
		payload.Today.AverageLatencyMS == nil || *payload.Today.AverageLatencyMS != 88 {
		t.Fatalf("dashboard summary = %#v", payload.Today)
	}
	if len(payload.TopModelsToday) != 1 || payload.TopModelsToday[0].Model != "gpt-test" || payload.TopModelsToday[0].Calls != 2 {
		t.Fatalf("top models = %#v", payload.TopModelsToday)
	}
	if len(payload.RecentFailures) != 1 || payload.RecentFailures[0].Model != "gpt-test" {
		t.Fatalf("recent failures = %#v", payload.RecentFailures)
	}
}

func TestServerCompatDataClear(t *testing.T) {
	cfg := testutil.NewConfig(t)
	handler, db := newCompatHandler(t, cfg, nil)
	ctx := context.Background()

	if _, err := db.InsertEvents(ctx, []usage.Event{compatEvent("clear-event-1", 1), compatEvent("clear-event-2", 2)}); err != nil {
		t.Fatalf("insert events: %v", err)
	}
	if err := db.AddDeadLetter(ctx, `{"bad":true}`, errors.New("parse failed")); err != nil {
		t.Fatalf("add dead letter: %v", err)
	}
	if err := db.SaveModelPrices(ctx, map[string]store.ModelPrice{"gpt-test": {Prompt: 1, Completion: 2}}); err != nil {
		t.Fatalf("save model prices: %v", err)
	}
	const aliasHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := db.UpsertAPIKeyAliases(ctx, []store.APIKeyAlias{{APIKeyHash: aliasHash, Alias: "Team A"}}); err != nil {
		t.Fatalf("save alias: %v", err)
	}

	missingAuth := testutil.Request(t, handler, http.MethodPost, "/v0/management/data/clear", `{"confirmation":"CLEAR DATA","scope":{"usageEvents":true,"deadLetters":true}}`, "")
	if missingAuth.Code != http.StatusUnauthorized && missingAuth.Code != http.StatusForbidden {
		t.Fatalf("missing auth status = %d body = %s", missingAuth.Code, missingAuth.Body.String())
	}

	badConfirm := testutil.Request(t, handler, http.MethodPost, "/v0/management/data/clear", `{"confirmation":"WRONG","scope":{"usageEvents":true,"deadLetters":true}}`, testutil.AdminKey)
	testutil.RequireStatus(t, badConfirm, http.StatusBadRequest)

	rr := testutil.Request(t, handler, http.MethodPost, "/v0/management/data/clear", `{"confirmation":"CLEAR DATA","scope":{"usageEvents":true,"deadLetters":true}}`, testutil.AdminKey)
	testutil.RequireStatus(t, rr, http.StatusOK)
	var clearResp struct {
		OK      bool `json:"ok"`
		Cleared struct {
			UsageEvents int64 `json:"usageEvents"`
			DeadLetters int64 `json:"deadLetters"`
		} `json:"cleared"`
	}
	testutil.DecodeJSON(t, rr, &clearResp)
	if !clearResp.OK || clearResp.Cleared.UsageEvents != 2 || clearResp.Cleared.DeadLetters != 1 {
		t.Fatalf("clear response = %#v", clearResp)
	}

	events, deadLetters, err := db.Counts(ctx)
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if events != 0 || deadLetters != 0 {
		t.Fatalf("counts after clear events=%d deadLetters=%d", events, deadLetters)
	}
	prices, err := db.LoadModelPrices(ctx)
	if err != nil {
		t.Fatalf("load model prices: %v", err)
	}
	if len(prices) != 1 || prices["gpt-test"].Prompt != 1 {
		t.Fatalf("model prices after clear = %#v", prices)
	}
	aliases, err := db.LoadAPIKeyAliases(ctx)
	if err != nil {
		t.Fatalf("load aliases: %v", err)
	}
	if len(aliases) != 1 || aliases[0].APIKeyHash != aliasHash {
		t.Fatalf("aliases after clear = %#v", aliases)
	}
}

func TestServerCompatMonitoringAnalytics(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	setup := &store.Setup{CPAUpstreamURL: cpa.URL(), ManagementKey: "management-key", Queue: "usage", PopSide: "right"}
	handler, db := newCompatHandler(t, testutil.NewConfig(t), setup)
	event := compatEvent("monitoring-analytics-event", 10)
	_, err := db.InsertEvents(context.Background(), []usage.Event{event})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	unauthorizedRR := testutil.Request(t, handler, http.MethodPost, "/v0/management/monitoring/analytics", `{"from_ms":1778000000000,"to_ms":1778000060000}`, "")
	testutil.RequireStatus(t, unauthorizedRR, http.StatusUnauthorized)

	badRR := testutil.Request(t, handler, http.MethodPost, "/v0/management/monitoring/analytics", `{"from_ms":2,"to_ms":1}`, testutil.AdminKey)
	testutil.RequireStatus(t, badRR, http.StatusBadRequest)

	body := `{"from_ms":1778000000000,"to_ms":1778000060000,"include":{"summary":true,"events_page":{"limit":10},"recent_failures":5}}`
	rr := testutil.Request(t, handler, http.MethodPost, "/v0/management/monitoring/analytics", body, testutil.AdminKey)
	testutil.RequireStatus(t, rr, http.StatusOK)

	var payload struct {
		Summary *struct {
			TotalCalls int64 `json:"total_calls"`
		} `json:"summary"`
		Events *struct {
			Items []struct {
				EventHash string `json:"event_hash"`
			} `json:"items"`
		} `json:"events"`
	}
	testutil.DecodeJSON(t, rr, &payload)
	if payload.Summary == nil || payload.Summary.TotalCalls != 1 {
		t.Fatalf("summary = %#v", payload.Summary)
	}
	if payload.Events == nil || len(payload.Events.Items) != 1 || payload.Events.Items[0].EventHash != "monitoring-analytics-event" {
		t.Fatalf("events = %#v", payload.Events)
	}
}

func TestServerCompatProxyRoutes(t *testing.T) {
	cpa := testutil.NewCPAMock(t)
	setup := &store.Setup{CPAUpstreamURL: cpa.URL(), ManagementKey: "management-key", Queue: "usage", PopSide: "right"}
	handler, _ := newCompatHandler(t, testutil.NewConfig(t), setup)

	blockedRR := testutil.Request(t, handler, http.MethodGet, "/v0/management/accounts?limit=10", "", testutil.AdminKey)
	testutil.RequireStatus(t, blockedRR, http.StatusNotFound)

	configRR := testutil.Request(t, handler, http.MethodGet, "/config", "", testutil.AdminKey)
	testutil.RequireStatus(t, configRR, http.StatusOK)
	configReq, ok := cpa.LastRequest("/config")
	if !ok {
		t.Fatal("CPA mock did not receive /config")
	}
	if configReq.Authorization != "Bearer management-key" {
		t.Fatalf("config proxy request = %#v", configReq)
	}

	modelsReq := httptest.NewRequest(http.MethodGet, "/v1/models?limit=20", nil)
	modelsReq.Header.Set("Authorization", "Bearer upstream-key")
	modelsRR := httptest.NewRecorder()
	handler.ServeHTTP(modelsRR, modelsReq)
	testutil.RequireStatus(t, modelsRR, http.StatusOK)
	modelsProxyReq, ok := cpa.LastRequest("/v1/models")
	if !ok {
		t.Fatal("CPA mock did not receive /v1/models")
	}
	if modelsProxyReq.Authorization != "Bearer upstream-key" || modelsProxyReq.Query != "limit=20" {
		t.Fatalf("model list proxy request = %#v", modelsProxyReq)
	}
}

func compatEvent(hash string, offset int64) usage.Event {
	return usage.Event{
		EventHash:    hash,
		TimestampMS:  1_778_000_000_000 + offset,
		Timestamp:    time.UnixMilli(1_778_000_000_000 + offset).UTC().Format(time.RFC3339Nano),
		Model:        "gpt-test",
		Endpoint:     "POST /v1/chat/completions",
		Method:       "POST",
		Path:         "/v1/chat/completions",
		AuthIndex:    "auth-1",
		Source:       "user@example.com",
		InputTokens:  1,
		OutputTokens: 2,
		TotalTokens:  3,
		CreatedAtMS:  1_778_000_000_100 + offset,
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
