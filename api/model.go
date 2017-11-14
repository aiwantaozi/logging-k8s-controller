package api

import (
	logging "github.com/aiwantaozi/infra-logging-client/logging"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/client"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	utils "github.com/aiwantaozi/logging-k8s-controller/utils"
)

const (
	DefaultEnviroment      = "default"
	SchemaLogging          = "logging"
	SchemaLoggingPluge     = "loggings"
	SchemaLoggingAuth      = "loggingAuth"
	SchemaLoggingAuthPluge = "loggingAuths"
)

const (
	AwsElasticsearch = "aws-elasticsearch"
	Elasticsearch    = "elasticsearch"
	Splunk           = "splunk"
	None             = "none"
	Embedded         = "embedded"
)

// target type
var (
	TargetPluginMapping = map[string]string{
		AwsElasticsearch: "aws-elasticsearch-service",
		Elasticsearch:    "elasticsearch",
		Splunk:           "splunk-http-eventcollector",
		None:             "none",
	}
	TargetLabelMapping = map[string]string{
		AwsElasticsearch: "endpoint",
	}
)

type Logging struct {
	client.Resource
	Enable              bool              `json:"enable"`
	Namespace           string            `json:"namespace"`
	TargetType          string            `json:"targetType"`
	OutputFlushInterval int               `json:"outputFlushInterval"`
	OutputTags          map[string]string `json:"outputTags"`
	//elasticsearch
	ESHost               string `json:"esHost"`
	ESPort               int    `json:"esPort"`
	ESLogstashPrefix     string `json:"esLogstashPrefix"`
	ESLogstashDateformat string `json:"esLogstashDateformat"`
	ESLogstashFormat     bool   `json:"esLogstashFormat"`
	ESIncludeTagKey      bool   `json:"esIncludeTagKey"`
	ESAuthUser           string `json:"esAuthUser"`     //secret
	ESAuthPassword       string `json:"esAuthPassword"` //secret
	//splunk
	SplunkHost       string `json:"splunkHost"`
	SplunkPort       int    `json:"splunkPort"`
	SplunkProtocol   string `json:"splunkProtocol"`
	SplunkSource     string `json:"splunkSource"`
	SplunkTimeFormat string `json:"splunkTimeFormat"`
	SplunkToken      string `json:"splunkToken"` //secret
}

type LoggingAuth struct {
	client.Resource
	EnableNamespaceLogging bool `json:"enableNamespaceLogging"`
}

type Secret struct {
	Type  string     `json:"type"`
	Label string     `json:"label"`
	Data  SecretData `json:"data"`
}

type SecretData struct {
	ESAuthUser     string `json:"user"`     //secret
	ESAuthPassword string `json:"password"` //secret
	SplunkToken    string `json:"token"`    //secret
}

type ServerAPIError struct {
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
	schemas.AddType("error", ServerAPIError{})

	loggingSchema(schemas.AddType(SchemaLogging, Logging{}))
	loggingAuthSchema(schemas.AddType(SchemaLoggingAuth, LoggingAuth{}))
	return schemas
}

func loggingSchema(logging *client.Schema) {
	logging.CollectionMethods = []string{"GET", "POST"}
	logging.ResourceMethods = []string{"GET", "PUT", "DELETE"}
	logging.IncludeableLinks = []string{SchemaLoggingPluge}

	enable := logging.ResourceFields["enable"]
	enable.Create = true
	enable.Required = true
	enable.Default = false
	logging.ResourceFields["enable"] = enable

	namespace := logging.ResourceFields["namespace"]
	namespace.Create = true
	namespace.Required = true
	logging.ResourceFields["namespace"] = namespace

	targetType := logging.ResourceFields["targetType"]
	targetType.Create = true
	targetType.Update = true
	targetType.Required = true
	targetType.Type = "enum"
	targetType.Options = []string{Elasticsearch, Splunk, None, Embedded}
	logging.ResourceFields["targetType"] = targetType

	esHost := logging.ResourceFields["esHost"]
	esHost.Create = true
	esHost.Update = true
	esHost.Required = true
	logging.ResourceFields["esHost"] = esHost

	esPort := logging.ResourceFields["esPort"]
	esPort.Create = true
	esPort.Update = true
	esPort.Required = true
	logging.ResourceFields["esPort"] = esPort

	esLogstashPrefix := logging.ResourceFields["esLogstashPrefix"]
	esLogstashPrefix.Create = true
	esLogstashPrefix.Update = true
	logging.ResourceFields["esLogstashPrefix"] = esLogstashPrefix

	outputFlushInterval := logging.ResourceFields["outputFlushInterval"]
	outputFlushInterval.Create = true
	outputFlushInterval.Update = true
	outputFlushInterval.Default = 1
	logging.ResourceFields["outputFlushInterval"] = outputFlushInterval

	// elasticsearch
	esLogstashFormat := logging.ResourceFields["esLogstashFormat"]
	esLogstashFormat.Create = true
	esLogstashFormat.Update = true
	logging.ResourceFields["esLogstashFormat"] = esLogstashFormat

	esLogstashDateformat := logging.ResourceFields["esLogstashDateformat"]
	esLogstashDateformat.Create = true
	esLogstashDateformat.Update = true
	esLogstashDateformat.Type = "enum"
	esLogstashDateformat.Options = utils.GetShowDateformat()
	logging.ResourceFields["esLogstashDateformat"] = esLogstashDateformat

	esIncludeTagKey := logging.ResourceFields["esIncludeTagKey"]
	esIncludeTagKey.Create = true
	esIncludeTagKey.Update = true
	esIncludeTagKey.Default = true
	logging.ResourceFields["esIncludeTagKey"] = esIncludeTagKey

	esAuthUser := logging.ResourceFields["esAuthUser"]
	esAuthUser.Create = true
	esAuthUser.Update = true
	esAuthUser.Default = ""
	logging.ResourceFields["esAuthUser"] = esAuthUser

	esAuthPassword := logging.ResourceFields["esAuthPassword"]
	esAuthPassword.Create = true
	esAuthPassword.Update = true
	esAuthPassword.Default = ""
	logging.ResourceFields["esAuthPassword"] = esAuthPassword

	//splunk
	splunkProtocol := logging.ResourceFields["splunkProtocol"]
	splunkProtocol.Create = true
	splunkProtocol.Update = true
	splunkProtocol.Type = "enum"
	splunkProtocol.Options = []string{"https", "http"}
	logging.ResourceFields["splunkProtocol"] = splunkProtocol

	splunkToken := logging.ResourceFields["splunkToken"]
	splunkToken.Create = true
	splunkToken.Update = true
	splunkToken.Default = ""
	logging.ResourceFields["splunkToken"] = splunkToken

	splunkTimeFormat := logging.ResourceFields["splunkTimeFormat"]
	splunkTimeFormat.Create = true
	splunkTimeFormat.Update = true
	splunkTimeFormat.Type = "enum"
	splunkTimeFormat.Options = []string{"none", "unixtime", "localtime"}
	logging.ResourceFields["splunkTimeFormat"] = splunkTimeFormat
}

func loggingAuthSchema(loggingAuth *client.Schema) {
	loggingAuth.CollectionMethods = []string{"GET"}
	loggingAuth.ResourceMethods = []string{"PUT"}
	loggingAuth.IncludeableLinks = []string{SchemaLoggingAuthPluge}

	enableNamespaceLogging := loggingAuth.ResourceFields["enableNamespaceLogging"]
	enableNamespaceLogging.Create = true
	enableNamespaceLogging.Required = true
	loggingAuth.ResourceFields["enableNamespaceLogging"] = enableNamespaceLogging
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
