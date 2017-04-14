package rye

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	//log "github.com/Sirupsen/logrus"
	"github.com/cactus/go-statsd-client/statsd"
)

//go:generate counterfeiter -o fakes/statsdfakes/fake_statter.go $GOPATH/src/github.com/cactus/go-statsd-client/statsd/client.go Statter
//go:generate perl -pi -e 's/$GOPATH\/src\///g' fakes/statsdfakes/fake_statter.go

// MWHandler struct is used to configure and access rye's basic functionality.
type MWHandler struct {
	Config         Config
	beforeHandlers []Handler
}

// Config struct allows you to set a reference to a statsd.Statter and include it's stats rate.
type Config struct {
	Statter  statsd.Statter
	StatRate float32
}

// JSONStatus is a simple container used for conveying status messages.
type JSONStatus struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Response struct is utilized by middlewares as a way to share state;
// ie. a middleware can return a *Response as a way to indicate
// that further middleware execution should stop (without an error) or return a
// a hard error by setting `Err` + `StatusCode`.
type Response struct {
	Err           error
	StatusCode    int
	StopExecution bool
	Context       context.Context
}

// Error bubbles a response error providing an implementation of the Error interface.
// It returns the error as a string.
func (r *Response) Error() string {
	return r.Err.Error()
}

// Handler is the primary type that any rye middleware must implement to be called in the Handle() function.
// In order to use this you must return a *rye.Response.
type Handler func(w http.ResponseWriter, r *http.Request) *Response

// Constructor for new instantiating new rye instances
// It returns a constructed *MWHandler instance.
func NewMWHandler(config Config) *MWHandler {
	return &MWHandler{
		Config: config,
	}
}

// Use adds a handler to every request. All handlers set up with use
// are fired first and then any route specific handlers are called
func (m *MWHandler) Use(handler Handler) {
	m.beforeHandlers = append(m.beforeHandlers, handler)
}

// The Handle function is the primary way to set up your chain of middlewares to be called by rye.
// It returns a http.HandlerFunc from net/http that can be set as a route in your http server.
func (m *MWHandler) Handle(customHandlers []Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers := append(m.beforeHandlers, customHandlers...)
		for _, handler := range handlers {
			var resp *Response

			// Record handler runtime
			func() {
				statusCode := "2xx"
				startTime := time.Now()

				if resp = handler(w, r); resp != nil {
					func() {
						// Stop execution if it's passed
						if resp.StopExecution {
							return
						}

						// If a context is returned, we will
						// replace the current request with a new request
						if resp.Context != nil {
							r = r.WithContext(resp.Context)
							return
						}

						// If there's no error but we have a response
						if resp.Err == nil {
							resp.Err = errors.New("Problem with middleware; neither Err or StopExecution is set")
							resp.StatusCode = http.StatusInternalServerError
						}

						// Now assume we have an error.
						if m.Config.Statter != nil && resp.StatusCode >= 500 {
							go m.Config.Statter.Inc("errors", 1, m.Config.StatRate)
						}

						// Write the error out
						statusCode = strconv.Itoa(resp.StatusCode)
						WriteJSONStatus(w, "error", resp.Error(), resp.StatusCode)
					}()
				}

				handlerName := getFuncName(handler)

				if m.Config.Statter != nil {
					// Record runtime metric
					go m.Config.Statter.TimingDuration(
						"handlers."+handlerName+".runtime",
						time.Since(startTime), // delta
						m.Config.StatRate,
					)

					// Record status code metric (default 2xx)
					go m.Config.Statter.Inc(
						"handlers."+handlerName+"."+statusCode,
						1,
						m.Config.StatRate,
					)
				}
			}()

			// stop executing rest of the
			// handlers if we encounter an error
			if resp != nil && (resp.StopExecution || resp.Err != nil) {
				return
			}
		}
	})
}

// WriteJSONStatus is a wrapper for WriteJSONResponse that returns a marshalled JSONStatus blob
func WriteJSONStatus(rw http.ResponseWriter, status, message string, statusCode int) {
	jsonData, _ := json.Marshal(&JSONStatus{
		Message: message,
		Status:  status,
	})

	WriteJSONResponse(rw, statusCode, jsonData)
}

// WriteJSONResponse writes data and status code to the ResponseWriter
func WriteJSONResponse(rw http.ResponseWriter, statusCode int, content []byte) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	rw.Write(content)
}

// getFuncName uses reflection to determine a given function name
// It returns a string version of the function name (and performs string cleanup)
func getFuncName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	ns := strings.Split(fullName, ".")

	// when we get a method (not a raw function) it comes attached to whatever struct is in its
	// method receiver via a function closure, this is not precisely the same as that method itself
	// so the compiler appends "-fm" so the name of the closure does not conflict with the actual function
	// http://grokbase.com/t/gg/golang-nuts/153jyb5b7p/go-nuts-fm-suffix-in-function-name-what-does-it-mean#20150318ssinqqzrmhx2ep45wjkxsa4rua
	return strings.TrimSuffix(ns[len(ns)-1], ")-fm")
}
