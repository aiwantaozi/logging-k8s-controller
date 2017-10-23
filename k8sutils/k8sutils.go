package k8sutils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	loggingv1 "github.com/aiwantaozi/infra-logging/client/logging/v1"
	"github.com/pkg/errors"
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClientConfig struct {
	ConfigPath string
}

func (k *K8sClientConfig) IsReachable() error {
	cfg, err := k.New()
	if cfg == nil || err != nil || cfg.Host == "" {
		logrus.Error("Could not communicate with k8s")
		return errors.Wrap(err, "could not reach k8s")
	}
	return nil
}

func (k *K8sClientConfig) New() (cfg *rest.Config, err error) {
	if k.ConfigPath != "" {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.ExplicitPath = k.ConfigPath
		overrides := &clientcmd.ConfigOverrides{}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	}
	return rest.InClusterConfig()
}

func NewLoggingCustomResourceDefinition(group string, labels map[string]string) *extensionsobj.CustomResourceDefinition {
	return &extensionsobj.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   loggingv1.LoggingName + "." + group,
			Labels: labels,
		},
		Spec: extensionsobj.CustomResourceDefinitionSpec{
			Group:   group,
			Version: loggingv1.Version,
			Scope:   extensionsobj.NamespaceScoped,
			Names: extensionsobj.CustomResourceDefinitionNames{
				Plural: loggingv1.LoggingResourcePlural,
				Kind:   loggingv1.LoggingsKind,
			},
		},
	}
}

func NewLoggingHostCustomResourceDefinition(group string, labels map[string]string) *extensionsobj.CustomResourceDefinition {
	return &extensionsobj.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:   loggingv1.LoggingName + "." + group,
			Labels: labels,
		},
		Spec: extensionsobj.CustomResourceDefinitionSpec{
			Group:   group,
			Version: loggingv1.Version,
			Scope:   extensionsobj.NamespaceScoped,
			Names: extensionsobj.CustomResourceDefinitionNames{
				Plural: loggingv1.LoggingResourcePlural,
				Kind:   loggingv1.LoggingsKind,
			},
		},
	}
}

// WaitForCRDReady waits for a third party resource to be available for use.
func WaitForCRDReady(listFunc func(opts metav1.ListOptions) (runtime.Object, error)) error {
	err := wait.Poll(3*time.Second, 10*time.Minute, func() (bool, error) {
		_, err := listFunc(metav1.ListOptions{})
		if err != nil {
			if se, ok := err.(*apierrors.StatusError); ok {
				if se.Status().Code == http.StatusNotFound {
					return false, nil
				}
			}
			return false, err
		}
		return true, nil
	})

	return errors.Wrap(err, fmt.Sprintf("timed out waiting for Custom Resoruce"))
}
