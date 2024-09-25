package engine

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"time"
)

func Init(st ServerType) error {
	gSvrType = st
	var err error
	luaL = lua.NewState()
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
	log.Info("===================engine init successfully===================")
	return nil
}

func Close() {
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
	rpg := luaL.NewTable()
	luaL.SetGlobal(globalEntry, rpg)
	registerApiToEntry()

	//创建entities
	t := luaL.NewTable()
	setLuaEntryValue(entitiesEntry, t)

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
