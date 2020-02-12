package logging

import (
	"github.com/onepanelio/core/util/env"
	"github.com/sirupsen/logrus"
	"os"
)

type Log struct {
	Log logrus.Logger
}

var log = logrus.New()
var Logger = Log{}

func init() {
	enableMethodCaller := env.GetEnv("LOGGING_ENABLE_CALLER_TRACE", "false")
	log.Out = os.Stderr
	Logger.Log = *log
	if enableMethodCaller == "true" {
		Logger.Log.SetReportCaller(true)
	}
}
