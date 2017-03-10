// @APIVersion 1.0.0
// @APITitle 9volt API
// @APIDescription 9volt's API for fetching check state, event data and cluster information
// @Contact daniel.selans@gmail.com
// @License MIT
// @LicenseUrl https://opensource.org/licenses/MIT
// @BasePath /api/v1
// @SubApi Cluster State [/cluster]
// @SubApi Monitor Configuration [/monitor]

package api

import (
	"net/http"
	"os"

	"github.com/InVisionApp/rye"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/9corp/9volt/config"
)

type Api struct {
	Config     *config.Config
	Version    string
	MemberID   string
	Identifier string
	MWHandler  *rye.MWHandler
	DebugUI    bool
}

type JSONStatus struct {
	Status  string
	Message string
}

func New(cfg *config.Config, mwHandler *rye.MWHandler, version string, debugUI bool) *Api {
	return &Api{
		Config:     cfg,
		Version:    version,
		MemberID:   cfg.MemberID,
		Identifier: "api",
		MWHandler:  mwHandler,
		DebugUI:    debugUI,
	}
}

func (a *Api) Run() {
	log.Debugf("%v: Starting API server", a.Identifier)

	routes := mux.NewRouter().StrictSlash(true)

	routes.Handle(a.setupHandler(
		"/", []rye.Handler{
			a.HomeHandler,
		})).Methods("GET")

	// Common handlers
	routes.Handle(a.setupHandler(
		"/version", []rye.Handler{
			a.VersionHandler,
		})).Methods("GET")

	routes.Handle(a.setupHandler(
		"/status/check", []rye.Handler{
			a.StatusHandler,
		})).Methods("GET")

	// State handlers (route order matters!)
	routes.Handle(a.setupHandler(
		"/api/v1/state", []rye.Handler{
			a.StateWithTagsHandler,
		})).Methods("GET").Queries("tags", "")

	routes.Handle(a.setupHandler(
		"/api/v1/state", []rye.Handler{
			a.StateHandler,
		})).Methods("GET")

	// Cluster handlers
	routes.Handle(a.setupHandler(
		"/api/v1/cluster", []rye.Handler{
			a.ClusterHandler,
		})).Methods("GET")

	// Monitor handlers (route order matters!)
	routes.Handle(a.setupHandler(
		"/api/v1/monitor", []rye.Handler{
			a.MonitorHandler,
		})).Methods("GET")

	routes.Handle(a.setupHandler(
		"/api/v1/monitor/{check}", []rye.Handler{
			a.MonitorDisableHandler,
		})).Methods("GET").Queries("disable", "")

	routes.Handle(a.setupHandler(
		"/api/v1/monitor/{check}", []rye.Handler{
			a.MonitorCheckHandler,
		})).Methods("GET")

	// Events handlers
	routes.Handle(a.setupHandler(
		"/api/v1/event", []rye.Handler{
			a.EventWithTypeHandler,
		})).Methods("GET").Queries("type", "")

	routes.Handle(a.setupHandler(
		"/api/v1/event", []rye.Handler{
			a.EventHandler,
		})).Methods("GET")

	if a.DebugUI {
		log.Info("ui: local debug mode (from /ui/dist)")
		routes.PathPrefix("/dist").Handler(a.MWHandler.Handle([]rye.Handler{
			rye.MiddlewareRouteLogger(),
			a.uiDistHandler,
		}))

		routes.PathPrefix("/ui").Handler(a.MWHandler.Handle([]rye.Handler{
			rye.MiddlewareRouteLogger(),
			a.uiHandler,
		}))
	} else {
		log.Info("ui: statik mode (from statik.go)")
		routes.PathPrefix("/dist").Handler(a.MWHandler.Handle([]rye.Handler{
			rye.MiddlewareRouteLogger(),
			a.uiDistStatikHandler,
		}))

		routes.PathPrefix("/ui").Handler(a.MWHandler.Handle([]rye.Handler{
			rye.MiddlewareRouteLogger(),
			a.uiStatikHandler,
		}))
	}

	http.ListenAndServe(a.Config.ListenAddress, routes)
}

// appends an apache style logger to each route. also dry up some boiler plate
func (a *Api) setupHandler(path string, ryeStack []rye.Handler) (string, http.Handler) {
	return path, handlers.LoggingHandler(os.Stdout, a.MWHandler.Handle(ryeStack))
}
