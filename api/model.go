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
	DefaultEnviroment         = "default"
	SchemaServiceLogging      = "servicelogging"
	SchemaServiceLoggingPluge = "serviceloggings"
	SchemaEnvLogging          = "envlogging"
	SchemaEnvLoggingPluge     = "envloggings"
)

type EnvLogging struct {
	client.Resource
	Name                     string `json:"name"`
	Provider                 string `json:"provider"`
	Environment              string `json:"environment"`
	OutputType               string `json:"outputType"`
	OutputHost               string `json:"outputHost"`
	OutputPort               int    `json:"outputPort"`
	OutputLogstashPrefix     string `json:"outputLogstashPrefix"`
	OutputLogstashDateformat string `json:"outputLogstashDateformat"`
	OutputTagKey             string `json:"outputTagKey"` // (optional; default=fluentd)
}

type ServiceLogging struct {
	client.Resource
	Name        string `json:"name"`
	Environment string `json:"environment"`
	InputPath   string `json:"inputPath"`
	InputFormat string `json:"inputFormat"`
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

	envLoggingSchema(schemas.AddType(SchemaEnvLogging, EnvLogging{}))
	serviceLoggingSchema(schemas.AddType(SchemaServiceLogging, ServiceLogging{}))
	return schemas
}

func envLoggingSchema(envLogging *client.Schema) {
	envLogging.CollectionMethods = []string{"GET", "POST"}
	envLogging.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	envLogging.IncludeableLinks = []string{SchemaEnvLoggingPluge}

	envLoggingName := envLogging.ResourceFields["name"]
	envLoggingName.Required = true
	envLoggingName.Unique = true
	envLogging.ResourceFields["name"] = envLoggingName

	provider := envLogging.ResourceFields["provider"]
	provider.Create = true
	provider.Update = true
	provider.Required = true
	provider.Default = "fluentd"
	envLogging.ResourceFields["provider"] = provider

	outputType := envLogging.ResourceFields["outputType"]
	outputType.Create = true
	outputType.Update = true
	outputType.Required = true
	envLogging.ResourceFields["outputType"] = outputType

	outputHost := envLogging.ResourceFields["outputHost"]
	outputHost.Create = true
	outputHost.Update = true
	outputHost.Required = true
	envLogging.ResourceFields["outputHost"] = outputHost

	outputPort := envLogging.ResourceFields["outputPort"]
	outputPort.Create = true
	outputPort.Update = true
	outputPort.Required = true
	envLogging.ResourceFields["outputPort"] = outputPort

	outputLogstashPrefix := envLogging.ResourceFields["outputLogstashPrefix"]
	outputLogstashPrefix.Create = true
	outputLogstashPrefix.Update = true
	outputLogstashPrefix.Default = true
	envLogging.ResourceFields["outputLogstashPrefix"] = outputLogstashPrefix

	outputLogstashDateformat := envLogging.ResourceFields["outputLogstashDateformat"]
	outputLogstashDateformat.Create = true
	outputLogstashDateformat.Update = true
	envLogging.ResourceFields["outputLogstashDateformat"] = outputLogstashDateformat

	outputTagKey := envLogging.ResourceFields["outputTagKey"]
	outputTagKey.Create = true
	outputTagKey.Update = true
	outputTagKey.Default = "kubernetes"
	envLogging.ResourceFields["outputTagKey"] = outputTagKey

}

func serviceLoggingSchema(serviceLogging *client.Schema) {
	serviceLogging.CollectionMethods = []string{"GET", "POST"}
	serviceLogging.ResourceMethods = []string{"GET", "PUT", "DELETE"}

	serviceLoggingName := serviceLogging.ResourceFields["name"]
	serviceLoggingName.Required = true
	serviceLoggingName.Unique = true
	serviceLogging.ResourceFields["name"] = serviceLoggingName

	environment := serviceLogging.ResourceFields["environment"]
	environment.Create = true
	environment.Update = true
	environment.Default = DefaultEnviroment
	serviceLogging.ResourceFields["environment"] = environment

	inputPath := serviceLogging.ResourceFields["inputPath"]
	inputPath.Create = true
	inputPath.Update = true
	inputPath.Required = true
	serviceLogging.ResourceFields["inputPath"] = inputPath

	inputFormat := serviceLogging.ResourceFields["inputFormat"]
	inputFormat.Create = true
	inputFormat.Update = true
	inputFormat.Required = true
	serviceLogging.ResourceFields["inputFormat"] = inputFormat

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
