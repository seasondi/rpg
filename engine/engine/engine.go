package engine

import (
	"errors"
	"fmt"
	lua "github.com/seasondi/gopher-lua"
	"time"
)

var serverInitialized = false

func Init(st ServerType) error {
	if serverInitialized {
		return errors.New("server is already initialized")
	}
	gSvrType = st
	var err error
	if err = GetCmdLine().Parse(); err != nil {
		return err
	}
	if err = initConfig(); err != nil {
		return err
	}
	if err = initLogger(); err != nil {
		return err
	}
	if st == STGame || st == STRobot {
		if err = initEntityDefs(); err != nil {
			return err
		}
	}
	if err = initEtcd(); err != nil {
		return err
	}
	if st != STDbMgr && st != STRobot {
		if err = initRedis(); err != nil {
			return err
		}
	}
	if err = initProtocol(); err != nil {
		return err
	}
	if st == STGame {
		//仅game类型进程可生成entity
		if err = initEntityIDGenerator(); err != nil {
			return err
		}
		if err = initEntityManager(); err != nil {
			return err
		}
	}
	if st == STRobot {
		if err = initRobotManager(); err != nil {
			return err
		}
	}
	if err = initTimer(); err != nil {
		return err
	}

	//lua虚拟机最后初始化
	if st == STGame || st == STRobot {
		if err = initLuaMachine(); err != nil {
			return err
		}
	}
	serverInitialized = true
	log.Info("===================engine init successfully===================")
	return nil
}

func Close() {
	if scriptChecker != nil {
		scriptChecker.Stop()
	}
	if luaL != nil {
		luaL.Close()
	}
	if etcdMgr != nil {
		etcdMgr.close()
	}
	if timer != nil {
		timer.close()
	}
}

func registerModuleToLua() {
	preloadLogger()
	preloadConfig()
	preloadRedis()
}

func registerGlobalEntry() {
	entry := luaL.NewTable()
	luaL.SetGlobal(globalEntry, entry)
	registerApiToEntry()

	//创建entities
	t := luaL.NewTable()
	setLuaEntryValue(entitiesEntry, t)
	setLuaEntryValue("database_name", lua.LString(GetProjectDB()))

	switch gSvrType {
	case STGate:
		setLuaEntryValue("is_gate", lua.LTrue)
	case STGame:
		setLuaEntryValue("is_game", lua.LTrue)
	case STDbMgr:
		setLuaEntryValue("is_db", lua.LTrue)
	case STRobot:
		setLuaEntryValue("is_robot", lua.LTrue)
	}
}

func Tick() {
	if luaCmdMgr != nil {
		luaCmdMgr.doCommands()
	}
	GetTimer().Tick()
}

var lastCheckStopTime time.Time

func CanStopped() bool {
	entitiesNum := GetEntityManager().GetEntityCount()
	saveLen := GetEntitySaveManager().Length()
	if entitiesNum == 0 && saveLen == 0 {
		return true
	}
	if time.Since(lastCheckStopTime).Seconds() >= 5 {
		log.Info("check can stop, left entities num: ", entitiesNum, ", left save num: ", saveLen)
		lastCheckStopTime = time.Now()
	}
	return false
}

func ListenProtoAddr() string {
	return fmt.Sprintf("tcp://%s", GetConfig().GetAddr())
}
