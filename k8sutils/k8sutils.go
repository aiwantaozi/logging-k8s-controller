package k8sutils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	loggingv1 "github.com/aiwantaozi/infra-logging-client/logging/v1"
	"github.com/pkg/errors"
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	appsv1beta1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	rbacv1beta1 "k8s.io/client-go/pkg/apis/rbac/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	EmbeddedESName     = "elasticsearch"
	EmbeddedKibanaName = "kibana"
	esImage            = "quay.io/pires/docker-elasticsearch-kubernetes:5.6.2"
	kibanaImage        = "kibana:5.6.4"
	// docker.elastic.co/kibana/kibana:5.6.4
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

func NewLoggingCustomResourceDefinition(namespace string, group string, labels map[string]string) *extensionsobj.CustomResourceDefinition {
	return &extensionsobj.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggingv1.LoggingCRDName,
			Labels:    labels,
			Namespace: namespace,
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

func NewLoggingAuthCustomResourceDefinition(namespace string, group string, labels map[string]string) *extensionsobj.CustomResourceDefinition {
	return &extensionsobj.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      loggingv1.LoggingAuthCRDName,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: extensionsobj.CustomResourceDefinitionSpec{
			Group:   group,
			Version: loggingv1.Version,
			Scope:   extensionsobj.NamespaceScoped,
			Names: extensionsobj.CustomResourceDefinitionNames{
				Plural: loggingv1.LoggingAuthResourcePlural,
				Kind:   loggingv1.LoggingAuthsKind,
			},
		},
	}
}

// WaitForCRDReady waits for a third party resource to be available for use.
func WaitForCRDReady(listFunc func(opts metav1.ListOptions) (*loggingv1.LoggingList, error)) error {
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

func NewESServiceAccount(namespace string) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedESName,
			Namespace: namespace,
		},
	}
}

func NewKibanaServiceAccount(namespace string) *apiv1.ServiceAccount {
	return &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedKibanaName,
			Namespace: namespace,
		},
	}
}

func NewESRole(namespace string) *rbacv1beta1.Role {
	return &rbacv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedESName,
			Namespace: namespace,
		},
		Rules: []rbacv1beta1.PolicyRule{
			{
				APIGroups: []string{rbacv1beta1.APIGroupAll},
				Resources: []string{"endpoints"},
				Verbs:     []string{rbacv1beta1.VerbAll},
			},
		},
	}
}

func NewKibanaRole(namespace string) *rbacv1beta1.Role {
	return &rbacv1beta1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedKibanaName,
			Namespace: namespace,
		},
		Rules: []rbacv1beta1.PolicyRule{
			{
				APIGroups: []string{rbacv1beta1.APIGroupAll},
				Resources: []string{"endpoints"},
				Verbs:     []string{rbacv1beta1.VerbAll},
			},
		},
	}
}

func NewESRoleBinding(namespace string) *rbacv1beta1.RoleBinding {
	return &rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedESName,
			Namespace: namespace,
		},
		RoleRef: rbacv1beta1.RoleRef{
			Name:     EmbeddedESName,
			Kind:     "Role",
			APIGroup: rbacv1beta1.GroupName,
		},
		Subjects: []rbacv1beta1.Subject{
			{
				Kind:      rbacv1beta1.ServiceAccountKind,
				Name:      EmbeddedESName,
				Namespace: namespace,
			},
		},
	}
}

func NewKibanaRoleBinding(namespace string) *rbacv1beta1.RoleBinding {
	return &rbacv1beta1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EmbeddedKibanaName,
			Namespace: namespace,
		},
		RoleRef: rbacv1beta1.RoleRef{
			Name:     EmbeddedKibanaName,
			Kind:     "Role",
			APIGroup: rbacv1beta1.GroupName,
		},
		Subjects: []rbacv1beta1.Subject{
			{
				Kind:      rbacv1beta1.ServiceAccountKind,
				Name:      EmbeddedKibanaName,
				Namespace: namespace,
			},
		},
	}
}

func NewESService(namespace string) *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedESName,
			Labels: map[string]string{
				"app": EmbeddedESName,
			},
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeNodePort,
			Ports: []apiv1.ServicePort{
				apiv1.ServicePort{
					Name:       "http",
					Port:       9200,
					TargetPort: intstr.FromInt(9200),
					NodePort:   30032,
				},
				apiv1.ServicePort{
					Name:       "tcp",
					Port:       9300,
					TargetPort: intstr.FromInt(9300),
					NodePort:   30033,
				},
			},
			Selector: map[string]string{
				"app": EmbeddedESName,
			},
		},
	}
}

func NewKibanaService(namespace string) *apiv1.Service {
	return &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedKibanaName,
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				apiv1.ServicePort{
					Name:       "http",
					Port:       5601,
					TargetPort: intstr.FromInt(5601),
					NodePort:   30034,
				},
			},
			Type: apiv1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": EmbeddedKibanaName,
			},
		},
	}
}

func NewESDeployment(namespace string, cpu int64, memory int64) *appsv1beta1.Deployment {
	deployment := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedESName,
			Labels: map[string]string{
				"app": EmbeddedESName,
			},
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": EmbeddedESName,
					},
				},
				Spec: apiv1.PodSpec{
					ServiceAccountName: EmbeddedESName,
					InitContainers: []apiv1.Container{
						{
							Name:            "init-sysctl",
							Image:           "busybox",
							ImagePullPolicy: apiv1.PullIfNotPresent,
							Command:         []string{"sysctl", "-w", "vm.max_map_count=262144"},
							SecurityContext: &apiv1.SecurityContext{
								Privileged: boolPtr(true),
							},
						},
					},
					Containers: []apiv1.Container{
						{
							Name: EmbeddedESName,
							SecurityContext: &apiv1.SecurityContext{
								Capabilities: &apiv1.Capabilities{
									Add: []apiv1.Capability{"IPC_LOCK"},
								},
							},
							Image: esImage,
							Env: []apiv1.EnvVar{
								{
									Name:  "KUBERNETES_CA_CERTIFICATE_FILE",
									Value: "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
								},
								{
									Name: "NAMESPACE",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "CLUSTER_NAME",
									Value: "myesdb",
								},
								{
									Name:  "DISCOVERY_SERVICE",
									Value: EmbeddedESName,
								},
								{
									Name:  "NODE_MASTER",
									Value: "true",
								},
								{
									Name:  "NODE_DATA",
									Value: "true",
								},
								{
									Name:  "HTTP_ENABLE",
									Value: "true",
								},
							},
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 9200,
								},
								{
									Name:          "tcp",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 9300,
								},
							},
							Resources: apiv1.ResourceRequirements{
								Requests: map[apiv1.ResourceName]resource.Quantity{
									//CPU is always requested as an absolute quantity, never as a relative quantity; 0.1 is the same amount of CPU on a single-core, dual-core, or 48-core machine
									apiv1.ResourceCPU: *resource.NewMilliQuantity(cpu, resource.DecimalSI),
									//Limits and requests for memory are measured in bytes.
									apiv1.ResourceMemory: *resource.NewQuantity(memory*(1024*1024), resource.DecimalSI), // unit is byte
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									MountPath: "/data",
									Name:      "storage",
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name: "storage",
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{},
							},
						},
					},
					RestartPolicy: apiv1.RestartPolicyAlways,
				},
			},
		},
	}

	return deployment
}

func NewKibanaDeployment(namespace string) *appsv1beta1.Deployment {
	deployment := &appsv1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      EmbeddedKibanaName,
		},
		Spec: appsv1beta1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": EmbeddedKibanaName,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  EmbeddedKibanaName,
							Image: kibanaImage,
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 5601,
								},
							},
							Env: []apiv1.EnvVar{
								{
									Name:  "ELASTICSEARCH_URL",
									Value: "http://" + EmbeddedESName + "." + namespace + ":9200",
								},
							},
						},
					},
					RestartPolicy: apiv1.RestartPolicyAlways,
				},
			},
		},
	}

	return deployment
}

func int32Ptr(i int32) *int32 { return &i }

func boolPtr(b bool) *bool { return &b }
