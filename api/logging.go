package api

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	loggingv1 "github.com/aiwantaozi/infra-logging-client/logging/v1"
	"github.com/aiwantaozi/logging-k8s-controller/k8sutils"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Server) LoggingsCreate(w http.ResponseWriter, req *http.Request) error {
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
	logrus.Infof("%v", runobj)
	logrus.Infof("%v", runobj.GetObjectKind().GroupVersionKind())
	logrus.Infof("%v", runobj.GetObjectKind().GroupVersionKind().Empty())
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

func (s *Server) LoggingsList(w http.ResponseWriter, req *http.Request) error {
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

func (s *Server) LoggingsGet(w http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	name := mux.Vars(req)["name"]
	var namespace string
	vals := req.URL.Query() // Returns a url.Values, which is a map[string][]string
	if nsarr, ok := vals["namespace"]; ok {
		namespace = nsarr[0]
	}

	sl, err := s.getLogging(apiContext, namespace, name)
	if err != nil {
		return errors.Wrap(err, "fail to get service logging")
	}
	apiContext.Write(sl)
	return nil
}

func (s *Server) LoggingsSet(w http.ResponseWriter, req *http.Request) error {
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

func (s *Server) LoggingsDelete(w http.ResponseWriter, req *http.Request) error {
	var sl Logging
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode service logging fail")
	}

	name := mux.Vars(req)["name"]
	err = s.deleteLogging(name, sl.Namespace)
	if err != nil {
		return errors.Wrapf(err, "fail to get service logging %s", name)
	}
	return nil
}

func (s *Server) getLogging(apiContext *api.ApiContext, namespace string, name string) (res *Logging, err error) {
	reslist, err := s.listLogging(apiContext, namespace)
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

func (s *Server) listLogging(apiContext *api.ApiContext, namespace string) ([]*Logging, error) {
	logres := []*Logging{}
	runobj, err := s.mclient.LoggingV1().Loggings(namespace).List(metav1.ListOptions{})
	if err != nil {
		return logres, errors.Wrap(err, "fail to read settings")
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
	// res := toResLogging(apiContext, logcrdobj.Items[0])
	return res, nil
}

func (s *Server) setLogging(sl Logging) (*Logging, error) {
	runobj, err := s.mclient.LoggingV1().Loggings(sl.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "fail to read loggings")
	}
	logobjs := runobj.(*loggingv1.LoggingList)
	if logobjs == nil || len(logobjs.Items) == 0 {
		return nil, errors.New("could not find logging object")
	}

	lgobj := toCRDLogging(sl, &logobjs.Items[0])
	_, err = s.mclient.LoggingV1().Loggings(sl.Namespace).Update(lgobj)
	if err != nil {
		return nil, errors.Wrap(err, "update service logging fail")
	}

	return &sl, nil
}

func (s *Server) deleteLogging(name string, namespace string) error {
	runobj, err := s.mclient.LoggingV1().Loggings(namespace).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "fail to read settings")
	}
	logobjs := runobj.(*loggingv1.LoggingList)
	if logobjs == nil || len(logobjs.Items) == 0 {
		return nil
	}

	err = s.mclient.LoggingV1().Loggings(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "delete service logging fail")
	}

	return nil
}

func toCRDLogging(res Logging, crd *loggingv1.Logging) *loggingv1.Logging {
	if crd == nil {
		crd = &loggingv1.Logging{
			ObjectMeta: metav1.ObjectMeta{
				Name:      loggingv1.LoggingName,
				Labels:    loggingv1.LabelMaps,
				Namespace: res.Namespace,
			},
		}
	}

	crd.Target = loggingv1.Target{
		OutputType:               res.OutputType,
		OutputHost:               res.OutputHost,
		OutputPort:               res.OutputPort,
		OutputLogstashPrefix:     res.OutputLogstashPrefix,
		OutputLogstashDateformat: res.OutputLogstashDateformat,
		OutputTagKey:             res.OutputTagKey,
	}
	return crd
}

func toResLogging(apiContext *api.ApiContext, crd loggingv1.Logging) *Logging {
	sl := Logging{
		Name:                     crd.Name,
		Namespace:                crd.Namespace,
		OutputHost:               crd.OutputHost,
		OutputPort:               crd.OutputPort,
		OutputLogstashPrefix:     crd.OutputLogstashPrefix,
		OutputLogstashDateformat: crd.OutputLogstashDateformat,
		OutputTagKey:             crd.OutputTagKey,
		OutputType:               crd.OutputType,
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
