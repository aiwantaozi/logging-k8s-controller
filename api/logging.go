package api

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	loggingv1 "github.com/aiwantaozi/infra-logging-client/logging/v1"
	"github.com/aiwantaozi/logging-k8s-controller/k8sutils"
	"github.com/aiwantaozi/logging-k8s-controller/utils"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Server) CreateLogging(w http.ResponseWriter, req *http.Request) error {
	var sl Logging
	apiContext := api.GetApiContext(req)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	namespace := sl.Namespace
	//create if crd not exist
	runobj, pErr := s.mclient.LoggingV1().Loggings(namespace).List(metav1.ListOptions{})
	if pErr != nil {
		// If Logging objects are already registered, we
		// won't attempt to do so again.
		if err := s.createCRDs(namespace); err != nil {
			return err
		}
	}

	var action string
	lgobjs := runobj.(*loggingv1.LoggingList)
	if len(lgobjs.Items) == 0 {
		action = "create"
		lgobj := toCRDLogging(sl, nil)
		_, err = s.mclient.LoggingV1().Loggings(namespace).Create(lgobj)
	} else {
		action = "update"
		lgobj := toCRDLogging(sl, &lgobjs.Items[0])
		_, err = s.mclient.LoggingV1().Loggings(namespace).Update(lgobj)
	}
	if err != nil {
		return errors.Wrapf(err, "%s crd object fail", action)
	}

	apiContext.Write(&sl)
	return nil
}

func (s *Server) ListLoggings(w http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	var namespace string
	vals := req.URL.Query() // Returns a url.Values, which is a map[string][]string
	if nsarr, ok := vals["namespace"]; ok {
		namespace = nsarr[0]
	}

	res, err := s.listLogging(apiContext, namespace)
	if err != nil {
		return errors.Wrap(err, "fail to list logging crd object")
	}
	resp := &client.GenericCollection{}
	resp.ResourceType = "logging"
	resp.CreateTypes = map[string]string{
		"logging": apiContext.UrlBuilder.Collection("logging"),
	}
	data := []interface{}{}
	for _, item := range res {
		data = append(data, item)
	}
	resp.Data = data
	apiContext.Write(resp)
	return nil
}

func (s *Server) GetLogging(w http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	id := mux.Vars(req)["id"]
	var namespace string
	vals := req.URL.Query() // Returns a url.Values, which is a map[string][]string
	if nsarr, ok := vals["namespace"]; ok {
		namespace = nsarr[0]
	}

	sl, err := s.getLogging(apiContext, namespace, id)
	if err != nil {
		return errors.Wrap(err, "fail to get service logging")
	}
	apiContext.Write(sl)
	return nil
}

func (s *Server) SetLogging(w http.ResponseWriter, req *http.Request) error {
	var sl Logging
	apiContext := api.GetApiContext(req)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	_, err = s.setLogging(sl)
	if err != nil {
		return errors.Wrap(err, "set env logging success")
	}
	apiContext.Write(&sl)
	return nil
}

func (s *Server) DeleteLogging(w http.ResponseWriter, req *http.Request) error {
	var sl Logging
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	name := mux.Vars(req)["id"]
	err = s.deleteLogging(name, sl.Namespace)
	if err != nil {
		return errors.Wrapf(err, "fail to get service logging %s", name)
	}
	return nil
}

func (s *Server) getLogging(apiContext *api.ApiContext, namespace string, id string) (res *Logging, err error) {
	//use list in case of get could not send empty namespace as all namespace
	reslist, err := s.listLogging(apiContext, namespace)
	if err != nil {
		return nil, err
	}

	for _, v := range reslist {
		if v.Resource.Id == id {
			return v, nil
		}
	}
	return nil, nil
}

func (s *Server) listLogging(apiContext *api.ApiContext, namespace string) ([]*Logging, error) {
	logres := []*Logging{}
	runobj, err := s.mclient.LoggingV1().Loggings(namespace).List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("fail to read logging, details: %v", err)
		return logres, nil
	}
	logcrdobj := runobj.(*loggingv1.LoggingList)
	if logcrdobj == nil || len(logcrdobj.Items) == 0 {
		return logres, nil
	}
	var res []*Logging
	for _, v := range logcrdobj.Items {
		r := toResLogging(apiContext, v)
		res = append(res, r)
	}
	return res, nil
}

func (s *Server) setLogging(sl Logging) (*Logging, error) {
	logging, err := s.mclient.LoggingV1().Loggings(sl.Namespace).Get(sl.Id, metav1.GetOptions{})
	if err != nil || logging == nil {
		return nil, errors.Wrap(err, "fail to get logging")
	}

	lgobj := toCRDLogging(sl, logging)
	_, err = s.mclient.LoggingV1().Loggings(sl.Namespace).Update(lgobj)
	if err != nil {
		return nil, errors.Wrap(err, "update service logging fail")
	}

	return &sl, nil
}

func (s *Server) deleteLogging(id string, namespace string) error {
	logging, err := s.mclient.LoggingV1().Loggings(namespace).Get(id, metav1.GetOptions{})
	if err != nil || logging == nil {
		return errors.Wrap(err, "fail to read get logging")
	}

	err = s.mclient.LoggingV1().Loggings(namespace).Delete(id, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "delete service logging fail")
	}

	return nil
}

func toCRDLogging(res Logging, crd *loggingv1.Logging) *loggingv1.Logging {
	if crd == nil {
		crd = &loggingv1.Logging{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.GenerateUUID(),
				Labels:    loggingv1.LabelMaps,
				Namespace: res.Namespace,
			},
		}
	}

	crd.Target = loggingv1.Target{
		TargetType:               res.TargetType,
		OutputTypeName:           res.OutputTypeName,
		OutputHost:               res.OutputHost,
		OutputPort:               res.OutputPort,
		OutputLogstashPrefix:     utils.ToRealDateformat(res.OutputLogstashPrefix),
		OutputLogstashDateformat: res.OutputLogstashDateformat,
		OutputTagKey:             res.OutputTagKey,
		OutputIncludeTagKey:      res.OutputIncludeTagKey,
		OutputLogstashFormat:     res.OutputLogstashFormat,
		OutputFlushInterval:      res.OutputFlushInterval,
	}
	return crd
}

func toResLogging(apiContext *api.ApiContext, crd loggingv1.Logging) *Logging {
	sl := Logging{
		Namespace:                crd.Namespace,
		TargetType:               crd.TargetType,
		OutputHost:               crd.OutputHost,
		OutputPort:               crd.OutputPort,
		OutputLogstashPrefix:     crd.OutputLogstashPrefix,
		OutputLogstashDateformat: utils.ToShowDateformat(crd.OutputLogstashDateformat),
		OutputTagKey:             crd.OutputTagKey,
		OutputTypeName:           crd.OutputTypeName,
		OutputLogstashFormat:     crd.OutputLogstashFormat,
		OutputIncludeTagKey:      crd.OutputIncludeTagKey,
		OutputFlushInterval:      crd.OutputFlushInterval,
		Resource: client.Resource{
			Id:      crd.Name,
			Type:    SchemaLogging,
			Actions: map[string]string{},
			Links:   map[string]string{},
		},
	}

	sl.Resource.Links["update"] = apiContext.UrlBuilder.ReferenceByIdLink(SchemaLogging, sl.Id)
	sl.Resource.Links["remove"] = apiContext.UrlBuilder.ReferenceByIdLink(SchemaLogging, sl.Id)
	return &sl
}

func (s *Server) createCRDs(namespace string) error {
	crd := k8sutils.NewLoggingCustomResourceDefinition(namespace, loggingv1.GroupName, loggingv1.LabelMaps)

	if _, err := s.crdclient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd); err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "creating CRD: %s", crd.Spec.Names.Kind)
	}

	logrus.Info("msg", "CRD created", "crd", crd.Spec.Names.Kind)

	// TODO: wait for crd
	// k8sutils.WaitForCRDReady(s.mclient.LoggingV1().Loggings(namespace).List)
	/* error
	api/logging.go:257:75: cannot use s.mclient.LoggingV1().Loggings(namespace).List (type func("github.com/aiwantaozi/logging-k8s-controller/vendor/k8s.io/apimachinery/pkg/apis/meta/v1".ListOptions) (runtime.Object, error))
	as type func("github.com/aiwantaozi/logging-k8s-controller/vendor/k8s.io/apimachinery/pkg/apis/meta/v1".ListOptions) (*"github.com/aiwantaozi/logging-k8s-controller/vendor/github.com/aiwantaozi/infra-logging-client/logging/v1".LoggingList, error) in argument to k8sutils.WaitForCRDReady
	*/
	return nil
}
