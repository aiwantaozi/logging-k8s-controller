package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	LoggingsKind          = "Logging"
	LoggingResourcePlural = "loggings"
	LoggingName           = "rancherlogging"
	GroupName             = "rancher.com"
	Namespace             = "cattle-system"
	Version               = "v1"
	SecretName            = "loggingsecret"
	SecretPath            = "/fluentd/etc/k8ssecret"
	ProviderName          = "fluentd"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1"}
	LoggingCRDName     = LoggingResourcePlural + "." + GroupName
	LabelMaps          = map[string]string{
		"mylabel": "test",
	}
)

type Target struct {
	OutputType               string `json:"output_type"`
	OutputHost               string `json:"output_host"`
	OutputPort               int    `json:"output_port"`
	OutputLogstashPrefix     string `json:"output_logstash_prefix"`
	OutputLogstashDateformat string `json:"output_logstash_dateformat"`
	OutputTagKey             string `json:"output_tag_key"` // (optional; default=fluentd)
}

type Logging struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Target            `json:"target"`
}

type LoggingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Logging `json:"items"`
}
