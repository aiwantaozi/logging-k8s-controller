package api

import (
	logging "github.com/aiwantaozi/infra-logging-client/logging"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/client"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DefaultEnviroment  = "default"
	SchemaLogging      = "logging"
	SchemaLoggingPluge = "loggings"
)

type Logging struct {
	client.Resource
	Name                     string `json:"name"`
	Namespace                string `json:"namespace"`
	OutputType               string `json:"outputType"`
	OutputHost               string `json:"outputHost"`
	OutputPort               int    `json:"outputPort"`
	OutputLogstashPrefix     string `json:"outputLogstashPrefix"`
	OutputLogstashDateformat string `json:"outputLogstashDateformat"`
	OutputTagKey             string `json:"outputTagKey"` // (optional; default=fluentd)
	OutputExtraData          string `json:"outputExtraData"`
}

type ServerApiError struct {
	client.Resource
	Status   int    `json:"status"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Detail   string `json:"detail"`
	BaseType string `json:"baseType"`
}

func NewSchema() *client.Schemas {
	schemas := &client.Schemas{}

	schemas.AddType("apiVersion", client.Resource{})
	schemas.AddType("schema", client.Schema{})
	schemas.AddType("error", ServerApiError{})

	loggingSchema(schemas.AddType(SchemaLogging, Logging{}))
	return schemas
}

func loggingSchema(logging *client.Schema) {
	logging.CollectionMethods = []string{"GET", "POST"}
	logging.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	logging.IncludeableLinks = []string{SchemaLoggingPluge}

	loggingName := logging.ResourceFields["name"]
	loggingName.Required = true
	loggingName.Unique = true
	logging.ResourceFields["name"] = loggingName

	provider := logging.ResourceFields["namespace"]
	provider.Create = true
	provider.Update = true
	provider.Required = true
	logging.ResourceFields["namespace"] = provider

	outputType := logging.ResourceFields["outputType"]
	outputType.Create = true
	outputType.Update = true
	outputType.Required = true
	logging.ResourceFields["outputType"] = outputType

	outputHost := logging.ResourceFields["outputHost"]
	outputHost.Create = true
	outputHost.Update = true
	outputHost.Required = true
	logging.ResourceFields["outputHost"] = outputHost

	outputPort := logging.ResourceFields["outputPort"]
	outputPort.Create = true
	outputPort.Update = true
	outputPort.Required = true
	logging.ResourceFields["outputPort"] = outputPort

	outputLogstashPrefix := logging.ResourceFields["outputLogstashPrefix"]
	outputLogstashPrefix.Create = true
	outputLogstashPrefix.Update = true
	outputLogstashPrefix.Default = true
	logging.ResourceFields["outputLogstashPrefix"] = outputLogstashPrefix

	outputLogstashDateformat := logging.ResourceFields["outputLogstashDateformat"]
	outputLogstashDateformat.Create = true
	outputLogstashDateformat.Update = true
	logging.ResourceFields["outputLogstashDateformat"] = outputLogstashDateformat

	outputTagKey := logging.ResourceFields["outputTagKey"]
	outputTagKey.Create = true
	outputTagKey.Update = true
	logging.ResourceFields["outputTagKey"] = outputTagKey

}

type Server struct {
	kclient   kubernetes.Interface
	mclient   logging.Interface
	crdclient apiextensionsclient.Interface
}

func NewServer(cfg *rest.Config) (*Server, error) {
	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating kubernetes client failed")
	}

	mclient, err := logging.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating loggingv1 client failed")
	}

	crdclient, err := apiextensionsclient.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating apiextensions client failed")
	}

	return &Server{
		kclient:   kclient,
		mclient:   mclient,
		crdclient: crdclient,
	}, nil
}
