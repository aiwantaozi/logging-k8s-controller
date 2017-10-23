package api

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
)

type HandleFuncWithError func(http.ResponseWriter, *http.Request) error

func HandleError(s *client.Schemas, t HandleFuncWithError) http.Handler {
	return api.ApiHandler(s, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if err := t(rw, req); err != nil {
			rw.WriteHeader(500)
			logrus.Warnf("HTTP handling error %v", err)
			apiContext := api.GetApiContext(req)
			WriteErr(err, apiContext)
		}
	}))
}

func WriteErr(err error, a *api.ApiContext) {
	logrus.Errorf("Error in request: %v", err)
	writeErr := a.WriteResource(&ServerApiError{
		Resource: client.Resource{
			Type: "error",
		},
		Status:  500,
		Code:    "Server Error",
		Message: err.Error(),
	})
	if writeErr != nil {
		logrus.Errorf("Failed to write err: %v", err)
	}
}

func NewRouter(s *Server) *mux.Router {
	schemas := NewSchema()
	r := mux.NewRouter().StrictSlash(true)
	f := HandleError
	versionsHandler := api.VersionsHandler(schemas, "v6")
	versionHandler := api.VersionHandler(schemas, "v6")
	r.Methods("GET").Path("/").Handler(versionsHandler)
	r.Methods("GET").Path("/v6").Handler(versionHandler)
	r.Methods("GET").Path("/v6/apiversions").Handler(versionsHandler)
	r.Methods("GET").Path("/v6/apiversions/v1").Handler(versionHandler)
	r.Methods("GET").Path("/v6/schemas").Handler(api.SchemasHandler(schemas))
	r.Methods("GET").Path("/v6/schemas/{id}").Handler(api.SchemaHandler(schemas))

	r.Methods("POST").Path("/v6/envloggings").Handler(f(schemas, s.EnvLoggingsSet))
	r.Methods("GET").Path("/v6/envloggings").Handler(f(schemas, s.EnvLoggingsList))
	r.Methods("GET").Path("/v6/envloggings/{name}").Handler(f(schemas, s.EnvLoggingsGet))
	r.Methods("PUT").Path("/v6/envloggings/{name}").Handler(f(schemas, s.EnvLoggingsSet))
	r.Methods("DELETE").Path("/v6/envloggings/{name}").Handler(f(schemas, s.EnvLoggingsDelete))

	r.Methods("POST").Path("/v6/serviceloggings").Handler(f(schemas, s.ServiceLoggingsSet))
	r.Methods("GET").Path("/v6/serviceloggings").Handler(f(schemas, s.ServiceLoggingsList))
	r.Methods("GET").Path("/v6/serviceloggings/{name}").Handler(f(schemas, s.ServiceLoggingsGet))
	r.Methods("PUT").Path("/v6/serviceloggings/{name}").Handler(f(schemas, s.ServiceLoggingsSet))
	r.Methods("DELETE").Path("/v6/serviceloggings/{name}").Handler(f(schemas, s.ServiceLoggingsDelete))
	return r
}
