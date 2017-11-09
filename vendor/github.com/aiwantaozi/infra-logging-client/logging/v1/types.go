package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ClusterNamespace = "cattle-system"
	GroupName        = "rancher.com"
	Version          = "v1"

	LoggingsKind          = "Logging"
	LoggingResourcePlural = "loggings"
	LoggingName           = "rancherlogging"

	LoggingAuthsKind          = "LoggingAuth"
	LoggingAuthResourcePlural = "loggingauths"
	LoggingAuthsName          = "rancherloggingauth"

	SecretName   = "loggingsecret"
	ProviderName = "fluentd"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}
	LoggingCRDName     = LoggingResourcePlural + "." + GroupName
	LoggingAuthCRDName = LoggingAuthResourcePlural + "." + GroupName
	LabelMaps          = map[string]string{
		"mylabel": "test",
	}
)

type Target struct {
	TargetType string `json:"target_type"`
	// common
	OutputHost          string            `json:"output_host"`
	OutputPort          int               `json:"output_port"`
	OutputFlushInterval int               `json:"output_flush_interval"`
	OutputRecords       map[string]string `json:"output_records"`
	// elasticsearch
	ESLogstashPrefix     string `json:"es_logstash_prefix"`
	ESLogstashDateformat string `json:"es_logstash_dateformat"`
	ESLogstashFormat     bool   `json:"es_logstash_format"`
	ESTagKey             string `json:"es_tag_key"` // (optional; default=fluentd)
	ESIncludeTagKey      bool   `json:"es_include_tag_key"`
	//splunk
	SplunkProtocol   string `json:"splunk_protocol"`
	SplunkSource     string `json:"splunk_source"`
	SplunkSourceType string `json:"splunk_sourcetype"`
	SplunkTimeFormat string `json:"splunk_time_format"`
}

type Logging struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Target            `json:"target"`
	
	SecretVersion     string `json:"secretVersion"`
}

type LoggingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Logging `json:"items"`
}

type LoggingAuth struct {
	metav1.TypeMeta        `json:",inline"`
	metav1.ObjectMeta      `json:"metadata"`
	EnableNamespaceLogging bool `json:"enableNamespaceLogging"`
}

type LoggingAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []LoggingAuth `json:"items"`
}
