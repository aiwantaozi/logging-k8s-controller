package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Sirupsen/logrus"
	loggingv1 "github.com/aiwantaozi/infra-logging-client/logging/v1"
	"github.com/aiwantaozi/logging-k8s-controller/k8sutils"
	"github.com/aiwantaozi/logging-k8s-controller/utils"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/api"
	"github.com/rancher/go-rancher/client"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	corev1 "k8s.io/client-go/pkg/api/v1"
	v1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func (s *Server) CreateLogging(w http.ResponseWriter, req *http.Request) error {
	var sl Logging
	apiContext := api.GetApiContext(req)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&sl)
	if err != nil {
		return errors.Wrap(err, "decode logging fail")
	}

	if sl.TargetType == Embedded {
		if sl.EmResReqCPU == "" {
			sl.EmResReqCPU = defaultEmResReqCPU
		}
		if sl.EmResReqMemory == "" {
			sl.EmResReqMemory = defaultEmResReqMemory
		}
		err = s.CreateEmbeddedTarget(loggingv1.ClusterNamespace, sl.EmResReqCPU, sl.EmResReqMemory)
		if err != nil {
			return err
		}
	}

	var action string
	var newSec *corev1.Secret
	namespace := sl.Namespace
	sec, err := toK8sSecret(sl)
	if err != nil {
		return err
	}
	// create or update secret
	existSecs, err := s.kclient.CoreV1().Secrets(namespace).Get(loggingv1.SecretName, metav1.GetOptions{})
	if existSecs == nil || err != nil {
		if err != nil {
			logrus.Errorf("get secret fail %v", err)
		}
		action = "create"
		newSec, err = s.kclient.CoreV1().Secrets(namespace).Create(sec)
	} else {
		action = "update"
		newSec, err = s.kclient.CoreV1().Secrets(namespace).Update(sec)
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
		lgobj.SecretVersion = newSec.ResourceVersion
		_, err = s.mclient.LoggingV1().Loggings(namespace).Create(lgobj)
	} else {
		action = "update"
		lgobj := toCRDLogging(sl, &lgobjs.Items[0])
		lgobj.SecretVersion = newSec.ResourceVersion
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

	if sl.TargetType == Embedded {
		if sl.EmResReqCPU == "" {
			sl.EmResReqCPU = defaultEmResReqCPU
		}
		if sl.EmResReqMemory == "" {
			sl.EmResReqMemory = defaultEmResReqMemory
		}
		err = s.CreateEmbeddedTarget(loggingv1.ClusterNamespace, sl.EmResReqCPU, sl.EmResReqMemory)
		if err != nil {
			return err
		}
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

	k8sSecs, err := s.kclient.CoreV1().Secrets(namespace).List(metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", loggingv1.SecretName).String()})
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
	crd.Target = loggingv1.Target{
		Enable:               strconv.FormatBool(res.Enable),
		TargetType:           res.TargetType,
		OutputFlushInterval:  res.OutputFlushInterval,
		OutputTags:           res.OutputTags,
		ESHost:               res.ESHost,
		ESPort:               res.ESPort,
		ESLogstashPrefix:     res.ESLogstashPrefix,
		ESLogstashDateformat: utils.ToRealDateformat(res.ESLogstashDateformat),
		ESLogstashFormat:     res.ESLogstashFormat,
		SplunkHost:           res.SplunkHost,
		SplunkPort:           res.SplunkPort,
		SplunkProtocol:       res.SplunkProtocol,
		SplunkTimeFormat:     res.SplunkTimeFormat,
		EmResReqCPU:          res.EmResReqCPU,
		EmResReqMemory:       res.EmResReqMemory,
	}

	return crd
}

func toResLogging(apiContext *api.ApiContext, crd loggingv1.Logging) *Logging {
	enable, err := strconv.ParseBool(crd.Enable)
	if err != nil {
		logrus.Errorf("in toResLogging, parse bool to string fail, %v", err)
	}
	sl := Logging{
		Enable:               enable,
		Namespace:            crd.Namespace,
		TargetType:           crd.TargetType,
		ESHost:               crd.ESHost,
		ESPort:               crd.ESPort,
		OutputFlushInterval:  crd.OutputFlushInterval,
		OutputTags:           crd.OutputTags,
		ESLogstashPrefix:     crd.ESLogstashPrefix,
		ESLogstashDateformat: utils.ToShowDateformat(crd.ESLogstashDateformat),
		ESLogstashFormat:     crd.ESLogstashFormat,
		SplunkHost:           crd.SplunkHost,
		SplunkPort:           crd.SplunkPort,
		SplunkProtocol:       crd.SplunkProtocol,
		SplunkSource:         crd.SplunkSource,
		SplunkTimeFormat:     crd.SplunkTimeFormat,
		EmResReqCPU:          crd.EmResReqCPU,
		EmResReqMemory:       crd.EmResReqMemory,
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
	logrus.Infof("toK8sSecret k8s sec namespace: %s, data: %s", res.Namespace, string(b))
	k8sSec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggingv1.SecretName,
			Namespace: res.Namespace,
		},
		Data: map[string][]byte{
			loggingv1.SecretName: b,
		},
	}
	return k8sSec, nil
}

func toResSecret(k8sSec *corev1.Secret) (*Secret, error) {
	var resSec Secret
	err := json.Unmarshal(k8sSec.Data[loggingv1.SecretName], &resSec)
	logrus.Infof("secret is namespace %s, name %s, data: %s", k8sSec.Namespace, k8sSec.Name, string(k8sSec.Data[loggingv1.SecretName]))
	logrus.Infof("after secret is: %v", resSec)
	return &resSec, errors.Wrap(err, "decode secret fail")
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

func (s *Server) CreateEmbeddedTarget(namespace string, emResReqCPU string, emResReqMemory string) error {
	cpu, err := strconv.ParseInt(emResReqCPU, 10, 64)
	if err != nil {
		return errors.Wrap(err, "parse request cpu fail")
	}
	memory, err := strconv.ParseInt(emResReqMemory, 10, 64)
	if err != nil {
		return errors.Wrap(err, "parse request memory fail")
	}
	// create es deployment
	var existESDep *v1beta1.DeploymentList
	existESDep, err = s.kclient.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", k8sutils.EmbeddedESName).String()})
	if err != nil {
		return errors.Wrapf(err, "get deployment %s fail", k8sutils.EmbeddedESName)
	}
	if len(existESDep.Items) == 0 {
		// create service account, role and rolebinding
		sc := k8sutils.NewESServiceAccount(namespace)
		role := k8sutils.NewESRole(namespace)
		roleBind := k8sutils.NewESRoleBinding(namespace)

		defer func() {
			if err != nil {
				s.kclient.CoreV1().ServiceAccounts(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		_, err = s.kclient.CoreV1().ServiceAccounts(namespace).Create(sc)
		if err != nil {
			return errors.Wrapf(err, "create service account %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				s.kclient.RbacV1beta1().Roles(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		_, err = s.kclient.RbacV1beta1().Roles(namespace).Create(role)
		if err != nil {
			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				s.kclient.RbacV1beta1().RoleBindings(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		_, err = s.kclient.RbacV1beta1().RoleBindings(namespace).Create(roleBind)
		if err != nil {
			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				s.kclient.CoreV1().Services(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		// create service and deployment
		newService := k8sutils.NewESService(namespace)
		_, err = s.kclient.CoreV1().Services(namespace).Create(newService)
		if err != nil {
			return errors.Wrapf(err, "create service %s fail", k8sutils.EmbeddedESName)
		}

		defer func() {
			if err != nil {
				s.kclient.ExtensionsV1beta1().Deployments(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
			}
		}()
		esDeployment := k8sutils.NewESDeployment(namespace, cpu, memory)
		_, err = s.kclient.ExtensionsV1beta1().Deployments(namespace).Create(esDeployment)
		if err != nil {
			return errors.Wrapf(err, "create deployment %s fail", k8sutils.EmbeddedESName)
		}
	} else {
		// update config
		newESDep := existESDep.Items[0]
		newESDep.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				//CPU is always requested as an absolute quantity, never as a relative quantity; 0.1 is the same amount of CPU on a single-core, dual-core, or 48-core machine
				corev1.ResourceCPU: *resource.NewMilliQuantity(cpu, resource.DecimalSI),
				//Limits and requests for memory are measured in bytes.
				corev1.ResourceMemory: *resource.NewQuantity(memory*(1024*1024), resource.DecimalSI), // unit is byte
			},
		}
		_, err = s.kclient.ExtensionsV1beta1().Deployments(namespace).Update(&newESDep)
		if err != nil {
			return errors.Wrapf(err, "update deployment %s fail", k8sutils.EmbeddedESName)
		}
	}

	// create kibana deployment
	var existKibanaDep *v1beta1.DeploymentList
	existKibanaDep, err = s.kclient.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", k8sutils.EmbeddedKibanaName).String()})
	if err != nil {
		return errors.Wrapf(err, "get deployment %s fail", k8sutils.EmbeddedKibanaName)
	}
	if len(existKibanaDep.Items) == 0 {
		// create service account, role and rolebinding
		sc := k8sutils.NewKibanaServiceAccount(namespace)
		role := k8sutils.NewKibanaRole(namespace)
		roleBind := k8sutils.NewKibanaRoleBinding(namespace)

		defer func() {
			if err != nil {
				s.kclient.CoreV1().ServiceAccounts(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		_, err = s.kclient.CoreV1().ServiceAccounts(namespace).Create(sc)
		if err != nil {
			return errors.Wrapf(err, "create service account  %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				s.kclient.RbacV1beta1().Roles(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		_, err = s.kclient.RbacV1beta1().Roles(namespace).Create(role)
		if err != nil {

			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				s.kclient.RbacV1beta1().RoleBindings(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		_, err = s.kclient.RbacV1beta1().RoleBindings(namespace).Create(roleBind)
		if err != nil {
			return errors.Wrapf(err, "create role %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				s.kclient.CoreV1().Services(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		newService := k8sutils.NewKibanaService(namespace)
		_, err = s.kclient.CoreV1().Services(namespace).Create(newService)
		if err != nil {
			return errors.Wrapf(err, "create service %s fail", k8sutils.EmbeddedKibanaName)
		}

		defer func() {
			if err != nil {
				s.kclient.ExtensionsV1beta1().Deployments(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
			}
		}()
		kibanaDeployment := k8sutils.NewKibanaDeployment(namespace)
		_, err = s.kclient.ExtensionsV1beta1().Deployments(namespace).Create(kibanaDeployment)
		if err != nil {
			return errors.Wrapf(err, "create deployment %s fail", k8sutils.EmbeddedKibanaName)
		}
	}
	return nil
}

func (s *Server) DeleteEmbeddedTarget(namespace string) error {
	//service account
	err := s.kclient.CoreV1().ServiceAccounts(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service account %s fail", k8sutils.EmbeddedESName)
	}
	err = s.kclient.CoreV1().ServiceAccounts(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service account %s fail", k8sutils.EmbeddedKibanaName)
	}

	//role
	err = s.kclient.RbacV1beta1().Roles(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedESName)
	}
	err = s.kclient.RbacV1beta1().Roles(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedKibanaName)
	}

	//rolebinding
	err = s.kclient.RbacV1beta1().RoleBindings(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedESName)
	}
	err = s.kclient.RbacV1beta1().RoleBindings(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete role %s fail", k8sutils.EmbeddedKibanaName)
	}

	//service
	err = s.kclient.CoreV1().Services(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service %s fail", k8sutils.EmbeddedESName)
	}
	err = s.kclient.CoreV1().Services(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete service %s fail", k8sutils.EmbeddedKibanaName)
	}

	//deployment
	err = s.kclient.AppsV1beta1().Deployments(namespace).Delete(k8sutils.EmbeddedESName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete deployment %s fail", k8sutils.EmbeddedESName)
	}

	err = s.kclient.AppsV1beta1().Deployments(namespace).Delete(k8sutils.EmbeddedKibanaName, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "delete deployment %s fail", k8sutils.EmbeddedKibanaName)
	}
	return nil
}
