package app

import "github.com/giolekva/pcloud/core/kg/log"

type logger interface {
	Debug(message string, fields ...log.Field)
	Info(message string, fields ...log.Field)
	Warn(message string, fields ...log.Field)
	Error(message string, fields ...log.Field)
}
