package httpapi

import (
	"embed"
	"net/http"
	"time"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/app"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/collector"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/config"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/http/router"
	"github.com/seakee/cpa-statistics/apps/manager-server/internal/store"
)

//go:embed web/management.html
var embeddedPanel embed.FS

const serviceID = "cpa-statistics"


type Server struct {
	handler http.Handler
	appCtx  *app.Context
}

func New(cfg config.Config, store *store.Store, collector *collector.Manager) *Server {
	startedAt := time.Now().UnixMilli()
	appCtx := app.FromExisting(
		cfg,
		store,
		collector,
		startedAt,
		embeddedPanel,
		serviceID,
	)
	return &Server{
		handler: router.New(appCtx),
		appCtx:  appCtx,
	}
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) AppContext() *app.Context {
	return s.appCtx
}
