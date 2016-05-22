package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"github.com/9corp/9volt/config"
	"github.com/9corp/9volt/util"
)

type Api struct {
	Config     *config.Config
	Version    string
	MemberID   string
	Identifier string
}

type JSONStatus struct {
	Status  string
	Message string
}

func New(cfg *config.Config, version string) *Api {
	a := &Api{}
	a.Config = cfg
	a.Version = version
	a.MemberID = util.GetMemberID(cfg.ListenAddress)
	a.Identifier = "api"

	return a
}

func (a *Api) HomeHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprint(rw, "Refer to README.md for 9volt API usage")
}

func (a *Api) StatusHandler(rw http.ResponseWriter, r *http.Request) {
	jsonStatus := &JSONStatus{
		Status:  "OK",
		Message: fmt.Sprintf("Our member ID: %v", a.MemberID),
	}

	data, err := json.Marshal(jsonStatus)
	if err != nil {
		rw.WriteHeader(400)
		rw.Write([]byte(fmt.Sprintf("Unable to generate status JSON: %v", err.Error())))
		return
	}

	rw.WriteHeader(200)
	rw.Write(data)
}

func (a *Api) VersionHandler(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(rw, "9volt %v", a.Version)
}

func (a *Api) Run() {
	log.Debugf("%v: Starting API server", a.Identifier)

	routes := mux.NewRouter().StrictSlash(true)

	routes.HandleFunc("/", a.HomeHandler).
		Methods("GET")

	routes.HandleFunc("/version", a.VersionHandler).
		Methods("GET")

	routes.HandleFunc("/status/check", a.StatusHandler).
		Methods("GET")

	http.ListenAndServe(a.Config.ListenAddress, routes)
}
