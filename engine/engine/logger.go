package engine

import (
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

func initLogger() error {
	logBase := logrus.New()
	logBase.SetLevel(strToLogLevel(cfg.Logger.LogLevel))
	logBase.SetReportCaller(true)
	log = logBase.WithFields(logrus.Fields{
		"Name": ServiceName(),
	})
	if cfg.Logger.JsonFormat {
		logBase.SetFormatter(&logrus.JSONFormatter{
			DisableHTMLEscape: true,
			TimestampFormat:   "2006-01-02 15:04:05.000 Z07:00",
			CallerPrettyfier: func(frame *runtime.Frame) (string, string) {
				arr := strings.Split(path.Dir(frame.File), "/")
				return "", fmt.Sprintf(" [%s/%s:%d]", arr[len(arr)-1], path.Base(frame.File), frame.Line)
			},
		})
	} else {
		logBase.SetFormatter(&nested.Formatter{
			HideKeys:        true,
			NoColors:        true,
			TimestampFormat: "2006-01-02 15:04:05.000 Z07:00",
			CustomCallerFormatter: func(frame *runtime.Frame) string {
				arr := strings.Split(path.Dir(frame.File), "/")
				return fmt.Sprintf(" [%s/%s:%d]", arr[len(arr)-1], path.Base(frame.File), frame.Line)
			},
		})
	}
	writers := make([]io.Writer, 0)
	if cfg.Logger.LogPath != "" {
		filename := cfg.Logger.LogPath + "/" + getLogDir() + "/" + ServiceName()
		//file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		//if err != nil {
		//	return err
		//}
		//writers = append(writers, file)
		writer, _ := rotatelogs.New(
			filename+"_%Y%m%d%H.log",
			rotatelogs.WithRotationTime(time.Hour),
		)
		writers = append(writers, writer)
	}
	if cfg.Logger.Console {
		writers = append(writers, os.Stdout)
	}
	if len(writers) > 0 {
		logBase.SetOutput(io.MultiWriter(writers...))
	}
	log.Infof("log inited. config: %+v", cfg.Logger)
	return nil
}

func strToLogLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
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

func getLogDir() string {
	switch gSvrType {
	case STGame:
		return "game"
	case STGate:
		return "gate"
	case STDbMgr:
		return "db"
	case STAdmin:
		return "admin"
	case STRobot:
		return "robot"
	default:
		return ""
	}
}

func GetLogger() *logrus.Entry {
	return log
}
