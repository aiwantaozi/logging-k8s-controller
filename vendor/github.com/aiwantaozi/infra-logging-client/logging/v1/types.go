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
	Enable              string            `json:"enable"`
	OutputFlushInterval int               `json:"output_flush_interval"`
	OutputTags          map[string]string `json:"output_records"`
	// elasticsearch
	ESHost               string `json:"es_host"`
	ESPort               int    `json:"es_port"`
	ESLogstashPrefix     string `json:"es_logstash_prefix"`
	ESLogstashDateformat string `json:"es_logstash_dateformat"`
	//splunk
	SplunkHost       string `json:"splunk_host"`
	SplunkPort       int    `json:"splunk_port"`
	SplunkProtocol   string `json:"splunk_protocol"`
	SplunkSource     string `json:"splunk_source"`
	SplunkSourceType string `json:"splunk_sourcetype"`
	SplunkTimeFormat string `json:"splunk_time_format"`
	//embedded
	EmResReqCPU    string `json:"em_res_req_cpu"`
	EmResReqMemory string `json:"em_res_req_memory"`
	//kafka
	KafkaBrokerType    string `json:"kafka_broker_type"`
	KafkaBrokers       string `json:"kafka_brokers"`
	KafkaZookeeperHost string `json:"kafka_zookeeper_host"`
	KafkaZookeeperPort int    `json:"kafka_zookeeper_port"`
	KafkaTopic         string `json:"kafka_topic"`
	//kafka data type
	KafkaOutputDataType string `json:"kafka_output_data_type"`
	//kafka producer settings
	KafkaMaxSendRetries int `json:"kafka_max_send_retries"`
}

type Logging struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Target            `json:"target"`

	SecretVersion string `json:"secretVersion"`
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
