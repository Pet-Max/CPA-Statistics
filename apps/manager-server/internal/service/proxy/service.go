package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/service/managerconfig"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
)

type Service struct {
	managerConfigService *managerconfig.Service
}

func New(managerConfigService *managerconfig.Service) *Service {
	return &Service{managerConfigService: managerConfigService}
}

func (s *Service) ProxyManagement(w http.ResponseWriter, r *http.Request, writeError func(http.ResponseWriter, int, error)) {
	s.proxyWithSavedManagementKey(w, r, writeError)
}

func (s *Service) ProxyCPA(w http.ResponseWriter, r *http.Request, writeError func(http.ResponseWriter, int, error)) {
	s.proxyWithSavedManagementKey(w, r, writeError)
}

func (s *Service) proxyWithSavedManagementKey(w http.ResponseWriter, r *http.Request, writeError func(http.ResponseWriter, int, error)) {
	setup, ok, err := s.resolveSetup(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusPreconditionRequired, errors.New("usage service is not configured"))
		return
	}
	target, err := url.Parse(setup.CPAUpstreamURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		req.Header.Set("Authorization", "Bearer "+setup.ManagementKey)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		writeError(w, http.StatusBadGateway, err)
	}
	proxy.ServeHTTP(w, r)
}

func (s *Service) ProxyModelList(w http.ResponseWriter, r *http.Request, writeError func(http.ResponseWriter, int, error), methodNotAllowed func(http.ResponseWriter)) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	setup, ok, err := s.resolveSetup(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeError(w, http.StatusPreconditionRequired, errors.New("usage service is not configured"))
		return
	}
	target, err := url.Parse(setup.CPAUpstreamURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		writeError(w, http.StatusBadGateway, err)
	}
	proxy.ServeHTTP(w, r)
}

func IsModelListPath(path string) bool {
	cleaned := strings.TrimRight(path, "/")
	return cleaned == "/v1/models" || cleaned == "/models"
}

func IsCPAProxyPath(path string) bool {
	cleaned := strings.TrimRight(path, "/")
	if cleaned == "" {
		return false
	}
	if _, ok := exactCPAProxyPaths[cleaned]; ok {
		return true
	}
	for _, prefix := range cpaProxyPathPrefixes {
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
			return true
		}
	}
	return false
}

var exactCPAProxyPaths = map[string]struct{}{
	"/config":                                {},
	"/config.yaml":                           {},
	"/latest-version":                        {},
	"/openai-compatibility":                  {},
	"/request-error-logs":                    {},
	"/v0/management/config":                  {},
	"/v0/management/config.yaml":             {},
	"/v0/management/latest-version":          {},
	"/v0/management/openai-compatibility":    {},
	"/v0/management/request-error-logs":      {},
}

var cpaProxyPathPrefixes = []string{
	"/auth-files",
	"/request-error-logs",
	"/request-log-by-id",
	"/v0/management/auth-files",
	"/v0/management/request-error-logs",
	"/v0/management/request-log-by-id",
}

func (s *Service) resolveSetup(ctx context.Context) (store.Setup, bool, error) {
	return s.managerConfigService.ResolveSetup(ctx)
}
