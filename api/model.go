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
	Namespace                string `json:"namespace"`
	TargetType               string `json:"targetType"`
	OutputTypeName           string `json:"outputTypeName"`
	OutputHost               string `json:"outputHost"`
	OutputPort               int    `json:"outputPort"`
	OutputLogstashPrefix     string `json:"outputLogstashPrefix"`
	OutputLogstashDateformat string `json:"outputLogstashDateformat"`
	OutputTagKey             string `json:"outputTagKey"` // (optional; default=fluentd)
	OutputExtraData          string `json:"outputExtraData"`
	OutputLogstashFormat     bool   `json:"outputLogstashFormat"`
	OutputIncludeTagKey      bool   `json:"outputIncludeTagKey"`
	OutputFlushInterval      int    `json:"outputFlushInterval"`
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

	namespace := logging.ResourceFields["namespace"]
	namespace.Create = true
	namespace.Required = true
	logging.ResourceFields["namespace"] = namespace

	targetType := logging.ResourceFields["targetType"]
	targetType.Create = true
	targetType.Update = true
	targetType.Required = true
	targetType.Type = "enum"
	targetType.Options = []string{"elasticsearch", "splunk"}
	logging.ResourceFields["targetType"] = targetType

	outputTypeName := logging.ResourceFields["outputTypeName"]
	outputTypeName.Create = true
	outputTypeName.Update = true
	outputTypeName.Required = true
	logging.ResourceFields["outputTypeName"] = outputTypeName

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

	outputFlushInterval := logging.ResourceFields["outputFlushInterval"]
	outputFlushInterval.Create = true
	outputFlushInterval.Update = true
	outputFlushInterval.Default = 1
	logging.ResourceFields["outputFlushInterval"] = outputFlushInterval

	outputLogstashFormat := logging.ResourceFields["outputLogstashFormat"]
	outputLogstashFormat.Create = true
	outputLogstashFormat.Update = true
	outputLogstashFormat.Default = true
	logging.ResourceFields["outputLogstashFormat"] = outputLogstashFormat

	outputLogstashDateformat := logging.ResourceFields["outputLogstashDateformat"]
	outputLogstashDateformat.Create = true
	outputLogstashDateformat.Update = true
	outputLogstashDateformat.Required = true
	outputLogstashDateformat.Type = "enum"
	outputLogstashDateformat.Options = []string{"%Y.%m.%d", "%Y.%m.", "%Y."}
	logging.ResourceFields["outputLogstashDateformat"] = outputLogstashDateformat

	outputIncludeTagKey := logging.ResourceFields["outputIncludeTagKey"]
	outputIncludeTagKey.Create = true
	outputIncludeTagKey.Update = true
	outputIncludeTagKey.Default = true
	logging.ResourceFields["outputIncludeTagKey"] = outputIncludeTagKey

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
