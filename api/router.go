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
			e := ServerAPIError{
				Resource: client.Resource{
					Type: "error",
				},
				Status:   500,
				Message:  err.Error(),
				BaseType: "error",
			}
			api.GetApiContext(req).Write(&e)
		}
	}))
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

	r.Methods("POST").Path("/v6/logging").Handler(f(schemas, s.CreateLogging))
	r.Methods("POST").Path("/v6/loggings").Handler(f(schemas, s.CreateLogging))
	r.Methods("GET").Path("/v6/logging").Handler(f(schemas, s.ListLoggings))
	r.Methods("GET").Path("/v6/loggings").Handler(f(schemas, s.ListLoggings))
	r.Methods("GET").Path("/v6/loggings/{id}").Handler(f(schemas, s.GetLogging))
	r.Methods("PUT").Path("/v6/loggings/{id}").Handler(f(schemas, s.SetLogging))
	r.Methods("DELETE").Path("/v6/loggings/{id}").Handler(f(schemas, s.DeleteLogging))

	r.Methods("GET").Path("/v6/loggingauth").Handler(f(schemas, s.ListLoggingAuths))
	r.Methods("GET").Path("/v6/loggingauths").Handler(f(schemas, s.ListLoggingAuths))
	r.Methods("GET").Path("/v6/loggingauths/{id}").Handler(f(schemas, s.GetLoggingAuth))
	r.Methods("PUT").Path("/v6/loggingauths/{id}").Handler(f(schemas, s.SetLoggingAuth))

	loggingAction := map[string]http.Handler{
		"update": f(schemas, s.SetLogging),
		"remove": f(schemas, s.DeleteLogging),
	}
	for name, actions := range loggingAction {
		r.Methods(http.MethodPost).Path("/v1/logging/{id}").Queries("action", name).Handler(actions)
	}
	return r
}
