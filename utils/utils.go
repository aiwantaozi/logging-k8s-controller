package utils

import (
	"encoding/base64"

	"github.com/Sirupsen/logrus"
	"github.com/rs/xid"
)

var (
	ToRealMap = map[string]string{
		"YYYY.MM.DD": "%Y.%m.%d",
		"YYYY.MM":    "%Y.%m.",
		"YYYY":       "%Y.",
	}
	ToShowMap = map[string]string{
		"%Y.%m.%d": "YYYY.MM.DD",
		"%Y.%m.":   "YYYY.MM",
		"%Y.":      "YYYY",
	}

	targetLabels = map[string]string{
		"aws-elasticsearch-service": "endpoint",
	}
)

func GenerateUUID() string {
	return xid.New().String()
}

func ToRealDateformat(format string) string {
	if res, ok := ToRealMap[format]; ok {
		return res
	}
	logrus.Warnf("could for find logstash format %s, use default setting %s", format, "%Y.%m.%d")
	return "%Y.%m.%d"
}

func ToShowDateformat(format string) string {
	if res, ok := ToShowMap[format]; ok {
		return res
	}
	return format
}

func GetShowDateformat() (keys []string) {
	for k := range ToRealMap {
		keys = append(keys, k)
	}
	return
}

func GetTargetLabel(target string) string {
	return targetLabels[target]
}

func EncodeBase64(src []byte) (dst []byte) {
	dst = make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(dst, src)
	return
}

func DecodeBase64(src []byte) (dst []byte, err error) {
	_, err = base64.StdEncoding.Decode(dst, src)
	return
}
