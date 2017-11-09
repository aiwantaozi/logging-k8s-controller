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

func (s *Server) ListLoggingAuths(w http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	res, err := s.listLoggingAuth(apiContext, loggingv1.ClusterNamespace)
	if err != nil {
		return errors.Wrap(err, "list loggingAuth fail")
	}
	resp := &client.GenericCollection{}
	resp.ResourceType = SchemaLoggingAuth
	resp.CreateTypes = map[string]string{
		SchemaLoggingAuth: apiContext.UrlBuilder.Collection(SchemaLoggingAuth),
	}
	data := []interface{}{}
	for _, item := range res {

		data = append(data, item)
	}
	resp.Data = data
	apiContext.Write(resp)
	return nil
}

func (s *Server) GetLoggingAuth(w http.ResponseWriter, req *http.Request) error {
	apiContext := api.GetApiContext(req)

	id := mux.Vars(req)["id"]
	sl, err := s.getLoggingAuth(apiContext, loggingv1.ClusterNamespace, id)
	if err != nil {
		return errors.Wrap(err, "get logging fail")
	}
	apiContext.Write(sl)
	return nil
}

func (s *Server) SetLoggingAuth(w http.ResponseWriter, req *http.Request) error {
	var sl LoggingAuth
	apiContext := api.GetApiContext(req)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode loggingAuth fail")
	}

	_, err = s.setLoggingAuth(sl, loggingv1.ClusterNamespace)
	if err != nil {
		return errors.Wrap(err, "set loggingAuth fail")
	}
	apiContext.Write(&sl)
	return nil
}

func (s *Server) getLoggingAuth(apiContext *api.ApiContext, namespace string, id string) (res *LoggingAuth, err error) {
	lg, err := s.mclient.LoggingV1().LoggingAuths(namespace).Get(id, metav1.GetOptions{})
	return toResLoggingAuth(apiContext, lg), err
}

func (s *Server) listLoggingAuth(apiContext *api.ApiContext, namespace string) ([]*LoggingAuth, error) {
	logres := []*LoggingAuth{}
	runobj, err := s.mclient.LoggingV1().LoggingAuths(namespace).List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("fail to list loggingAuth, details: %v", err)
		return logres, nil
	}
	logcrdobj := runobj.(*loggingv1.LoggingAuthList)
	if logcrdobj == nil || len(logcrdobj.Items) == 0 {
		return logres, nil
	}

	var res []*LoggingAuth
	for _, v := range logcrdobj.Items {
		res = append(res, toResLoggingAuth(apiContext, &v))
	}

	return res, nil
}

func (s *Server) setLoggingAuth(sl LoggingAuth, namespace string) (*LoggingAuth, error) {
	loggingAuth, err := s.mclient.LoggingV1().LoggingAuths(namespace).Get(sl.Id, metav1.GetOptions{})
	if err != nil || loggingAuth == nil {
		return nil, errors.Wrap(err, "fail to get loggingAuth")
	}
	loggingAuth.EnableNamespaceLogging = sl.EnableNamespaceLogging
	_, err = s.mclient.LoggingV1().LoggingAuths(namespace).Update(loggingAuth)
	if err != nil {
		return nil, errors.Wrap(err, "update loggingAuth fail")
	}
	return &sl, nil
}

func (s *Server) deleteLoggingAuth(id string, namespace string) error {
	loggingAuth, err := s.mclient.LoggingV1().LoggingAuths(namespace).Get(id, metav1.GetOptions{})
	if err != nil || loggingAuth == nil {
		return errors.Wrap(err, "get loggingAuth fail")
	}

	return s.mclient.LoggingV1().LoggingAuths(namespace).Delete(id, &metav1.DeleteOptions{})
}

func (s *Server) createLoggingAuthCRDs(namespace string) error {
	crd := k8sutils.NewLoggingAuthCustomResourceDefinition(namespace, loggingv1.GroupName, loggingv1.LabelMaps)

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

func toResLoggingAuth(apiContext *api.ApiContext, crd *loggingv1.LoggingAuth) *LoggingAuth {
	sl := LoggingAuth{
		EnableNamespaceLogging: crd.EnableNamespaceLogging,
		Resource: client.Resource{
			Id:      crd.Name,
			Type:    SchemaLoggingAuth,
			Actions: map[string]string{},
			Links:   map[string]string{},
		},
	}

	sl.Resource.Links["update"] = apiContext.UrlBuilder.ReferenceByIdLink(SchemaLoggingAuth, sl.Id)
	sl.Resource.Links["remove"] = apiContext.UrlBuilder.ReferenceByIdLink(SchemaLoggingAuth, sl.Id)
	return &sl
}
