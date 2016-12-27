package api

import (
	"fmt"
	"net/http"

	"github.com/InVisionApp/rye"
)

func (a *Api) StateHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprintf(rw, "state handler")
	return nil
}

func (a *Api) StateWithTagsHandler(rw http.ResponseWriter, r *http.Request) *rye.Response {
	fmt.Fprintf(rw, "state with tags handler")
	return nil
}
