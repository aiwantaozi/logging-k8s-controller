package api

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	loggingv1 "github.com/aiwantaozi/infra-logging/client/logging/v1"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Server) ServiceLoggingsList(w http.ResponseWriter, req *http.Request) error {
	logrus.Info("-------list")
	apiContext := api.GetApiContext(req)
	res, err := s.listServiceLogging(apiContext)
	if err != nil {
		return errors.Wrap(err, "fail to list service logging")
	}
	resp := &client.GenericCollection{}
	resp.ResourceType = SchemaServiceLogging
	resp.CreateTypes = map[string]string{
		"servicelogging": apiContext.UrlBuilder.Collection("servicelogging"),
	}

	data := []interface{}{}
	for _, item := range res {
		data = append(data, item)
	}
	resp.Data = data
	apiContext.Write(resp)
	return nil
}

func (s *Server) ServiceLoggingsGet(w http.ResponseWriter, req *http.Request) error {
	logrus.Info("-------get")
	name := mux.Vars(req)["name"]

	apiContext := api.GetApiContext(req)

	sl, err := s.getServiceLogging(apiContext, name)
	if err != nil {
		return errors.Wrap(err, "fail to get service logging")
	}
	apiContext.Write(sl)
	return nil
}

func (s *Server) ServiceLoggingsSet(w http.ResponseWriter, req *http.Request) error {
	logrus.Info("-------set")
	var sl ServiceLogging

	apiContext := api.GetApiContext(req)

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	sln, err := s.setServiceLogging(sl)
	if err != nil {
		return errors.Wrap(err, "set service logging fail")
	}

	apiContext.Write(sln)
	return nil
}

func (s *Server) ServiceLoggingsDelete(w http.ResponseWriter, req *http.Request) error {
	//TODO: need also remove the deployment when no env open the setting
	logrus.Info("-------delete")

	var sl ServiceLogging
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	name := mux.Vars(req)["name"]
	err = s.deleteServiceLogging(name, sl.Environment)
	if err != nil {
		return errors.Wrapf(err, "fail to get service logging %s", name)
	}
	return nil
}

func (s *Server) getServiceLogging(apiContext *api.ApiContext, name string) (res *ServiceLogging, err error) {
	reslist, err := s.listServiceLogging(apiContext)
	if err != nil {
		return nil, err
	}

	for _, v := range reslist {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, nil
}

func (s *Server) listServiceLogging(apiContext *api.ApiContext) (res []*ServiceLogging, err error) {
	logres := []*ServiceLogging{}
	logcrdobj, err := s.mclient.LoggingV1().Loggings(loggingv1.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return logres, errors.Wrap(err, "fail to read settings")
	}
	if logcrdobj == nil || len(logcrdobj.Items) == 0 {
		return logres, nil
	}

	return toResServiceLogging(apiContext, logcrdobj.Items[0].Spec.Sources), nil
}

func (s *Server) setServiceLogging(sl ServiceLogging) (*ServiceLogging, error) {
	res := []ServiceLogging{sl}
	crds := toCRDServiceLogging(res)

	logcrdobj, err := s.mclient.LoggingV1().Loggings(loggingv1.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "fail to read settings")
	}
	if logcrdobj == nil || len(logcrdobj.Items) == 0 {
		return nil, nil
	}

	newCrdObj := logcrdobj.Items[0]
	newSource := []loggingv1.Source{}
	for _, v := range newCrdObj.Spec.Sources {
		if v.Name == crds[0].Name {
			v.InputPath = crds[0].InputPath
			v.InputFormat = crds[0].InputFormat
		}
		newSource = append(newSource, v)
	}

	_, err = s.mclient.LoggingV1().Loggings(loggingv1.Namespace).Update(&newCrdObj)
	if err != nil {
		return nil, errors.Wrap(err, "update service logging fail")
	}

	return &sl, nil
}

func (s *Server) deleteServiceLogging(name string, env string) error {
	logcrdobj, err := s.mclient.LoggingV1().Loggings(loggingv1.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "fail to read settings")
	}
	if logcrdobj == nil || len(logcrdobj.Items) == 0 {
		return nil
	}

	newCrdObj := logcrdobj.Items[0]
	newSource := []loggingv1.Source{}
	for _, v := range newCrdObj.Spec.Sources {
		if v.Name != name && v.Environment != env {
			newSource = append(newSource, v)
		}
	}

	_, err = s.mclient.LoggingV1().Loggings(loggingv1.Namespace).Update(&newCrdObj)
	if err != nil {
		return errors.Wrap(err, "update service logging fail")
	}

	return nil
}

func toCRDServiceLogging(res []ServiceLogging) (crd []loggingv1.Source) {
	for _, v := range res {
		cr := loggingv1.Source{
			Environment: v.Environment,
			InputPath:   v.InputPath,
			InputFormat: v.InputFormat,
		}
		crd = append(crd, cr)
	}
	return
}

func toResServiceLogging(apiContext *api.ApiContext, crd []loggingv1.Source) (res []*ServiceLogging) {
	for _, v := range crd {
		sl := ServiceLogging{
			Name:        v.Name,
			Environment: v.Environment,
			InputPath:   v.InputPath,
			InputFormat: v.InputFormat,
			Resource: client.Resource{
				//TODO: decide what should be id
				Id:      v.Name,
				Type:    SchemaServiceLogging,
				Actions: map[string]string{},
				Links:   map[string]string{},
			},
		}
		sl.Actions["update"] = apiContext.UrlBuilder.ReferenceLink(sl.Resource) + "?action=update"
		sl.Actions["delete"] = apiContext.UrlBuilder.ReferenceLink(sl.Resource) + "?action=delete"
		res = append(res, &sl)
	}
	return
}
