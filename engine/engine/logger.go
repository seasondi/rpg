package engine

import (
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"strings"
)

func initLogger() error {
	logBase := logrus.New()
	logBase.SetLevel(strToLogLevel(cfg.Logger.LogLevel))
	logBase.SetReportCaller(true)
	log = logBase.WithFields(logrus.Fields{
		//"Pid": os.Getpid(),
	})
	// 输出到控制台
	if len(cfg.Logger.Output) == 0 {
		logBase.SetFormatter(&nested.Formatter{
			HideKeys: true,
			//TimestampFormat: time.RFC3339,
			TimestampFormat: "2006-01-02 15:04:05.000 Z07:00",
			CustomCallerFormatter: func(frame *runtime.Frame) string {
				arr := strings.Split(path.Dir(frame.File), "/")
				return fmt.Sprintf(" [%s/%s:%d]", arr[len(arr)-1], path.Base(frame.File), frame.Line)
			},
		})
		logBase.SetOutput(os.Stdout)
	}
	log.Infof("log inited. config: %+v", cfg.Logger)
	return nil
}

func strToLogLevel(level string) logrus.Level {
	switch level {
	case "error":
		return logrus.ErrorLevel
	case "warn":
		return logrus.WarnLevel
	case "info":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	case "trace":
		return logrus.TraceLevel
	}
	return logrus.DebugLevel
}

func GetLogger() *logrus.Entry {
	return log
}
