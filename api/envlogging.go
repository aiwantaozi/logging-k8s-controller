package api

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	loggingv1 "github.com/aiwantaozi/infra-logging/client/logging/v1"
	"github.com/aiwantaozi/logging-k8s-controller/k8sutils"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Server) EnvLoggingsCreate(w http.ResponseWriter, req *http.Request) error {
	logrus.Info("-------create")
	var sl EnvLogging
	apiContext := api.GetApiContext(req)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	lgobjs, pErr := s.mclient.LoggingV1().Loggings(loggingv1.Namespace).List(metav1.ListOptions{})
	if pErr != nil {
		// If Logging objects are already registered, we
		// won't attempt to do so again.
		if err := s.createCRDs(); err != nil {
			return err
		}

	}

	if len(lgobjs.Items) == 0 {
		lgobj, _ := toCRDEnvLogging([]EnvLogging{sl}, nil)
		_, err = s.mclient.LoggingV1().Loggings(loggingv1.Namespace).Create(lgobj)
	} else {
		lgobj, _ := toCRDEnvLogging([]EnvLogging{sl}, &lgobjs.Items[0])
		_, err = s.mclient.LoggingV1().Loggings(loggingv1.Namespace).Update(lgobj)
	}

	if err != nil {
		//TODO: could better
		return errors.Wrap(err, "create or update crd object fail")
	}

	apiContext.Write(&sl)
	return nil
}

func (s *Server) EnvLoggingsList(w http.ResponseWriter, req *http.Request) error {
	logrus.Info("-------list")
	apiContext := api.GetApiContext(req)
	res, err := s.listEnvLogging(apiContext)
	if err != nil {
		return errors.Wrap(err, "fail to list service logging")
	}
	resp := &client.GenericCollection{}
	resp.ResourceType = "envlogging"
	resp.CreateTypes = map[string]string{
		"envlogging": apiContext.UrlBuilder.Collection("envlogging"),
	}
	data := []interface{}{}
	for _, item := range res {
		data = append(data, item)
	}
	resp.Data = data
	apiContext.Write(resp)
	return nil
}

func (s *Server) EnvLoggingsGet(w http.ResponseWriter, req *http.Request) error {
	logrus.Info("-------get")
	name := mux.Vars(req)["name"]

	apiContext := api.GetApiContext(req)

	sl, err := s.getEnvLogging(apiContext, name)
	if err != nil {
		return errors.Wrap(err, "fail to get service logging")
	}
	apiContext.Write(sl)
	return nil
}

func (s *Server) EnvLoggingsSet(w http.ResponseWriter, req *http.Request) error {
	logrus.Info("-------set")
	var sl EnvLogging
	apiContext := api.GetApiContext(req)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	_, err = s.setEnvLogging(sl)
	if err != nil {
		return errors.Wrap(err, "set env logging success")
	}
	apiContext.Write(&sl)
	return nil
}

func (s *Server) EnvLoggingsDelete(w http.ResponseWriter, req *http.Request) error {
	//TODO: need also remove the deployment when no env open the setting
	logrus.Info("-------delete")

	var sl EnvLogging
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	name := mux.Vars(req)["name"]
	err = s.deleteEnvLogging(name, sl.Environment)
	if err != nil {
		return errors.Wrapf(err, "fail to get service logging %s", name)
	}
	return nil
}

func (s *Server) getEnvLogging(apiContext *api.ApiContext, name string) (res *EnvLogging, err error) {
	reslist, err := s.listEnvLogging(apiContext)
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

func (s *Server) listEnvLogging(apiContext *api.ApiContext) (res []*EnvLogging, err error) {
	logres := []*EnvLogging{}
	logcrdobj, err := s.mclient.LoggingV1().Loggings(loggingv1.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return logres, errors.Wrap(err, "fail to read settings")
	}
	if logcrdobj == nil || len(logcrdobj.Items) == 0 {
		return logres, nil
	}

	return toResEnvLogging(apiContext, logcrdobj.Items[0]), nil
}

func (s *Server) setEnvLogging(sl EnvLogging) (*EnvLogging, error) {
	logobjs, err := s.mclient.LoggingV1().Loggings(loggingv1.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "fail to read loggings")
	}
	if logobjs == nil || len(logobjs.Items) == 0 {
		return nil, errors.New("could not find logging object")
	}

	lgobj, _ := toCRDEnvLogging([]EnvLogging{sl}, &logobjs.Items[0])
	logrus.Info("here2 %v", lgobj)
	_, err = s.mclient.LoggingV1().Loggings(loggingv1.Namespace).Update(lgobj)
	if err != nil {
		return nil, errors.Wrap(err, "update service logging fail")
	}

	return &sl, nil
}

func (s *Server) deleteEnvLogging(name string, env string) error {
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

func toCRDEnvLogging(res []EnvLogging, crd *loggingv1.Logging) (*loggingv1.Logging, bool) {
	isExist := true
	// isServiceExist := false
	isTargetExist := false
	if crd == nil {
		isExist = false
		crd = &loggingv1.Logging{
			ObjectMeta: metav1.ObjectMeta{
				Name:   loggingv1.LoggingName,
				Labels: loggingv1.LabelMaps,
			},
			Spec: loggingv1.LoggingSpec{
				Provider:      loggingv1.ProviderName,
				LatestVersion: "000",
				Sources:       []loggingv1.Source{},
				Targets:       []loggingv1.Target{},
			},
			//TODO: volumes init
		}
	}

	newTargets := []loggingv1.Target{}
	var ntg loggingv1.Target
	for _, v := range res {
		// for _, v2 := range v.ServicesLogging {
		// 	if isExist {
		// 		for _, v3 := range crd.Spec.Sources {
		// 			if v3.Environment == v2.Environment && v3.Name == v2.Name {
		// 				isServiceExist = true
		// 				v3.InputPath = v2.InputPath
		// 				v3.InputFormat = v2.InputFormat
		// 			}
		// 		}
		// 	}
		// 	if !isExist || !isServiceExist {
		// 		sv := loggingv1.Source{
		// 			Name:        v2.Name,
		// 			InputPath:   v2.InputPath,
		// 			InputFormat: v2.InputFormat,
		// 			Environment: v2.Environment,
		// 		}
		// 		crd.Spec.Sources = append(crd.Spec.Sources, sv)
		// 	}
		// }

		if isExist {
			for _, v5 := range crd.Spec.Targets {
				if v5.Environment == v.Environment && v5.OutputType == v.OutputType {
					isTargetExist = true
					ntg.OutputType = v.OutputType
					ntg.OutputHost = v.OutputHost
					ntg.OutputPort = v.OutputPort
					ntg.OutputLogstashPrefix = v.OutputLogstashPrefix
					ntg.OutputLogstashDateformat = v.OutputLogstashDateformat
					ntg.OutputTagKey = v.OutputTagKey
				} else {
					newTargets = append(newTargets, v5)
				}
			}
		}
		if !isExist || !isTargetExist {
			ntg = loggingv1.Target{
				Environment:              v.Environment,
				OutputType:               v.OutputType,
				OutputHost:               v.OutputHost,
				OutputPort:               v.OutputPort,
				OutputLogstashPrefix:     v.OutputLogstashPrefix,
				OutputLogstashDateformat: v.OutputLogstashDateformat,
				OutputTagKey:             v.OutputTagKey,
			}
		}
	}
	crd.Spec.Targets = append(newTargets, ntg)
	return crd, isExist
}

func toResEnvLogging(apiContext *api.ApiContext, crd loggingv1.Logging) (res []*EnvLogging) {
	//TODO: filter different namespace
	// var svl []*ServiceLogging
	// for _, v := range crd.Spec.Sources {
	// 	sv := ServiceLogging{
	// 		Name:        v.Name,
	// 		Environment: v.Environment,
	// 		InputPath:   v.InputPath,
	// 		InputFormat: v.InputFormat,
	// 		Resource: client.Resource{
	// 			//TODO: decide what should be id
	// 			Id:      v.Name,
	// 			Type:    SchemaServiceLogging,
	// 			Actions: map[string]string{},
	// 			Links:   map[string]string{},
	// 		},
	// 	}

	// 	sv.Actions["update"] = apiContext.UrlBuilder.ReferenceLink(sv.Resource) + "?action=update"
	// 	sv.Actions["delete"] = apiContext.UrlBuilder.ReferenceLink(sv.Resource) + "?action=delete"
	// 	svl = append(svl, &sv)
	// }
	for _, v := range crd.Spec.Targets {
		sl := EnvLogging{
			Name:                     crd.Name,
			Environment:              v.Environment,
			OutputHost:               v.OutputHost,
			OutputPort:               v.OutputPort,
			OutputLogstashPrefix:     v.OutputLogstashPrefix,
			OutputLogstashDateformat: v.OutputLogstashDateformat,
			OutputTagKey:             v.OutputTagKey,
			OutputType:               v.OutputType,
			// ServicesLogging:          svl,
			Resource: client.Resource{
				//TODO: decide what should be id
				Id:      crd.Name,
				Type:    SchemaEnvLogging,
				Actions: map[string]string{},
				Links:   map[string]string{},
			},
		}
		sl.Actions["update"] = apiContext.UrlBuilder.ReferenceLink(sl.Resource) + "?action=update"
		sl.Actions["delete"] = apiContext.UrlBuilder.ReferenceLink(sl.Resource) + "?action=delete"
		sl.Links["serviceloggings"] = apiContext.UrlBuilder.Link(sl.Resource, "serviceloggings")
		res = append(res, &sl)
	}
	return
}

func (s *Server) createCRDs() error {
	//TODO: change labels
	labels := map[string]string{"test": "test"}

	crd := k8sutils.NewLoggingCustomResourceDefinition(loggingv1.GroupName, labels)

	if _, err := s.crdclient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "creating CRD: %s", crd.Spec.Names.Kind)
	}

	logrus.Info("msg", "CRD created", "crd", crd.Spec.Names.Kind)

	return k8sutils.WaitForCRDReady(s.mclient.LoggingV1().Loggings(loggingv1.Namespace).List)
}
