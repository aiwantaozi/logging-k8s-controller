package utils

import (
	"github.com/rs/xid"
)

func GenerateUUID() string {
	return xid.New().String()
}
