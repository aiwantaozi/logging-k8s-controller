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
	"k8s.io/apimachinery/pkg/fields"
	corev1 "k8s.io/client-go/pkg/api/v1"
)

func (s *Server) CreateLogging(w http.ResponseWriter, req *http.Request) error {
	var sl Logging
	apiContext := api.GetApiContext(req)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode logging fail")
	}

	var action string
	namespace := sl.Namespace
	// create or update secret
	existSec, err := s.kclient.CoreV1().Secrets(namespace).Get(loggingv1.SecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	sec, err := toK8sSecret(sl)
	if err != nil {
		return err
	}
	if existSec == nil {
		action = "create"
		_, err = s.kclient.CoreV1().Secrets(namespace).Create(sec)
	} else {
		action = "update"
		_, err = s.kclient.CoreV1().Secrets(namespace).Update(sec)
	}
	if err != nil {
		return errors.Wrapf(err, "%s secret fail", action)
	}

	//create if crd not exist
	runobj, pErr := s.mclient.LoggingV1().Loggings(namespace).List(metav1.ListOptions{})
	if pErr != nil {
		// If Logging objects are already registered, we
		// won't attempt to do so again.
		if err := s.createCRDs(namespace); err != nil {
			return err
		}
	}

	// create or update logging
	lgobjs := runobj.(*loggingv1.LoggingList)
	if len(lgobjs.Items) == 0 {
		action = "create"
		lgobj := toCRDLogging(sl, nil)
		lgobj.SecretVersion = existSec.ResourceVersion
		_, err = s.mclient.LoggingV1().Loggings(namespace).Create(lgobj)
	} else {
		action = "update"
		lgobj := toCRDLogging(sl, &lgobjs.Items[0])
		lgobj.SecretVersion = existSec.ResourceVersion
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
		if namespace == "" {
			namespace = corev1.NamespaceAll
		}
	}

	res, err := s.listLogging(apiContext, namespace)
	if err != nil {
		return errors.Wrap(err, "list logging fail")
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
		return errors.Wrap(err, "get logging fail")
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
		return errors.Wrap(err, "decode logging fail")
	}

	_, err = s.setLogging(sl)
	if err != nil {
		return errors.Wrap(err, "set logging fail")
	}
	apiContext.Write(&sl)
	return nil
}

func (s *Server) DeleteLogging(w http.ResponseWriter, req *http.Request) error {
	var sl Logging
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode logging fail")
	}

	name := mux.Vars(req)["id"]
	err = s.deleteLogging(name, sl.Namespace)
	if err != nil {
		return errors.Wrapf(err, "delete logging %s fail", name)
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
		logrus.Errorf("list logging fail, details: %v", err)
		return logres, nil
	}
	logcrdobj := runobj.(*loggingv1.LoggingList)
	if logcrdobj == nil || len(logcrdobj.Items) == 0 {
		return logres, nil
	}

	k8sSecs, err := s.kclient.CoreV1().Secrets(namespace).List(metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("type", "Opaque").String()})
	if err != nil {
		return logres, err
	}

	var res []*Logging
	for _, v := range logcrdobj.Items {
		for _, v2 := range k8sSecs.Items {
			if v.Namespace == v2.Namespace {
				sec, err := toResSecret(&v2)
				if err != nil {
					return logres, err
				}
				r := toResLogging(apiContext, v)
				r.ESAuthUser = sec.Data.ESAuthUser
				r.ESAuthPassword = sec.Data.ESAuthPassword
				r.SplunkToken = sec.Data.SplunkToken
				res = append(res, r)
			}
		}
	}
	return res, nil
}

func (s *Server) setLogging(sl Logging) (*Logging, error) {
	logging, err := s.mclient.LoggingV1().Loggings(sl.Namespace).Get(sl.Id, metav1.GetOptions{})
	if err != nil || logging == nil {
		return nil, errors.Wrap(err, "get logging fail")
	}

	k8sSec, err := toK8sSecret(sl)
	if err != nil {
		return nil, err
	}
	_, err = s.kclient.CoreV1().Secrets(sl.Namespace).Update(k8sSec)
	if err != nil {
		return nil, errors.Wrap(err, "update logging secret fail")
	}

	lgobj := toCRDLogging(sl, logging)
	lgobj.SecretVersion = k8sSec.ResourceVersion
	_, err = s.mclient.LoggingV1().Loggings(sl.Namespace).Update(lgobj)
	if err != nil {
		return nil, errors.Wrap(err, "update logging fail")
	}

	return &sl, nil
}

func (s *Server) deleteLogging(id string, namespace string) error {
	logging, err := s.mclient.LoggingV1().Loggings(namespace).Get(id, metav1.GetOptions{})
	if err != nil || logging == nil {
		return errors.Wrap(err, "get logging fail")
	}

	err = s.mclient.LoggingV1().Loggings(namespace).Delete(id, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "delete logging fail")
	}

	return s.kclient.CoreV1().Secrets(namespace).Delete(loggingv1.SecretName, &metav1.DeleteOptions{})
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
	crd.Enable = res.Enable
	crd.Target = loggingv1.Target{
		TargetType:           res.TargetType,
		OutputHost:           res.OutputHost,
		OutputPort:           res.OutputPort,
		OutputFlushInterval:  res.OutputFlushInterval,
		OutputRecords:        res.OutputRecords,
		ESLogstashPrefix:     res.ESLogstashPrefix,
		ESLogstashDateformat: utils.ToRealDateformat(res.ESLogstashDateformat),
		ESTagKey:             res.ESTagKey,
		ESIncludeTagKey:      res.ESIncludeTagKey,
		ESLogstashFormat:     res.ESLogstashFormat,
		SplunkProtocol:       res.SplunkProtocol,
		SplunkSource:         res.SplunkSourceType,
		SplunkTimeFormat:     res.SplunkTimeFormat,
	}

	return crd
}

func toResLogging(apiContext *api.ApiContext, crd loggingv1.Logging) *Logging {
	sl := Logging{
		Enable:               crd.Enable,
		Namespace:            crd.Namespace,
		TargetType:           TargetPluginMapping[crd.TargetType],
		OutputHost:           crd.OutputHost,
		OutputPort:           crd.OutputPort,
		OutputFlushInterval:  crd.OutputFlushInterval,
		OutputRecords:        crd.OutputRecords,
		ESLogstashPrefix:     crd.ESLogstashPrefix,
		ESLogstashDateformat: utils.ToShowDateformat(crd.ESLogstashDateformat),
		ESTagKey:             crd.ESTagKey,
		ESLogstashFormat:     crd.ESLogstashFormat,
		ESIncludeTagKey:      crd.ESIncludeTagKey,
		SplunkProtocol:       crd.SplunkProtocol,
		SplunkSource:         crd.SplunkSource,
		SplunkSourceType:     crd.SplunkSourceType,
		SplunkTimeFormat:     crd.SplunkTimeFormat,
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

func toK8sSecret(res Logging) (*corev1.Secret, error) {
	sec := Secret{
		Type:  res.TargetType,
		Label: utils.GetTargetLabel(res.TargetType),
		Data: SecretData{
			ESAuthUser:     res.ESAuthUser,
			ESAuthPassword: res.ESAuthPassword,
			SplunkToken:    res.SplunkToken,
		},
	}

	b, err := json.Marshal(sec)
	if err != nil {
		return nil, errors.Wrap(err, "marshal secret data fail")
	}
	data := utils.EncodeBase64(b)
	k8sSec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GenerateUUID(),
			Labels:    loggingv1.LabelMaps,
			Namespace: res.Namespace,
		},
		Data: map[string][]byte{
			loggingv1.SecretName: data,
		},
	}
	return k8sSec, nil
}

func toResSecret(k8sSec *corev1.Secret) (*Secret, error) {
	var resSec Secret
	err := json.Unmarshal(k8sSec.Data[loggingv1.SecretName], &resSec)
	return &resSec, err
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
