package engine

import (
	"fmt"
	"github.com/sirupsen/logrus"
	lua "github.com/yuin/gopher-lua"
	"strings"
)

const fileKey = "source_file"

var logExports = map[string]lua.LGFunction{
	"error": logError,
	"warn":  logWarn,
	"info":  logInfo,
	"debug": logDebug,
	"trace": logTrace,
}

var scriptLogger *logrus.Entry

func preloadLogger() {
	luaL.PreloadModule("logger", loggerLoader)
}

func loggerLoader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), logExports)
	L.Push(mod)
	scriptLogger = log
	//scriptLogger = log.WithFields(logrus.Fields{"source": "LUA"})
	return 1
}

func getLuaLogMessage(L *lua.LState) string {
	logStr := ""
	n := L.GetTop()
	for i := 1; i <= n; i++ {
		v := L.Get(i)
		if i == 1 && v.Type() == lua.LTTable && L.GetMetaField(v, entityFieldType) == lua.LString("entity") {
			logStr += "[" + L.GetMetaField(v, entityFieldName).String() + ":" + L.GetField(v, entityFieldId).String() + "] "
		} else {
			logStr += v.String()
		}
	}
	//L.SetTop(0)
	return logStr
}

func getLuaFile(L *lua.LState) string {
	dbg, ok := L.GetStack(2)
	if !ok {
		for i := 1; i >= -1; i-- {
			dbg, ok = L.GetStack(i)
			if ok {
				break
			}
		}
	}
	if !ok {
		return ""
	}
	if _, err := L.GetInfo("Sl", dbg, lua.LNil); err != nil {
		return ""
	} else {
		source := dbg.Source
		arr := strings.Split(dbg.Source, "/")
		length := len(arr)
		if length < 2 {
			arr = strings.Split(dbg.Source, "\\")
			length = len(arr)
		}
		if length >= 2 {
			source = arr[length-2] + "/" + arr[length-1]
		}
		return fmt.Sprintf("%s:%d", source, dbg.CurrentLine)
	}
}

func logInfo(L *lua.LState) int {
	scriptLogger.WithField(fileKey, getLuaFile(L)).Info(getLuaLogMessage(L))
	return 0
}

func logDebug(L *lua.LState) int {
	scriptLogger.WithField(fileKey, getLuaFile(L)).Debug(getLuaLogMessage(L))
	return 0
}

func logWarn(L *lua.LState) int {
	scriptLogger.WithField(fileKey, getLuaFile(L)).Warn(getLuaLogMessage(L))
	return 0
}

func logError(L *lua.LState) int {
	scriptLogger.WithField(fileKey, getLuaFile(L)).Error(getLuaLogMessage(L))
	return 0
}

func logTrace(L *lua.LState) int {
	scriptLogger.WithField(fileKey, getLuaFile(L)).Trace(getLuaLogMessage(L))
	return 0
}
