package api

import (
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"
)

func (a *Api) ClusterHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprint(rw, "cluster handler")
	return nil
}
