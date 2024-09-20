package engine

import (
	"container/list"
	"errors"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"sync"
)

type luaCommandHandler func(...interface{})

type luaCommandInfo struct {
	f    luaCommandHandler
	args []interface{}
}

type luaCommandMgr struct {
	sync.Mutex
	commands *list.List
}

func (m *luaCommandMgr) init() {
	m.Lock()
	defer m.Unlock()
	m.commands = list.New()
}

func (m *luaCommandMgr) addCommand(handler luaCommandHandler, args []interface{}) {
	m.Lock()
	defer m.Unlock()
	m.commands.PushBack(&luaCommandInfo{
		f:    handler,
		args: args,
	})
}

func (m *luaCommandMgr) doCommands() {
	m.Lock()
	defer m.Unlock()
	if m.commands.Len() == 0 {
		return
	}
	for cmd := m.commands.Front(); cmd != nil; cmd = cmd.Next() {
		if command, ok := cmd.Value.(*luaCommandInfo); ok {
			command.f(command.args...)
		}
	}
	m.commands = list.New()
}

func initLuaMachine() error {
	luaL = lua.NewState()
	luaCmdMgr = new(luaCommandMgr)
	luaCmdMgr.init()
	registerApiToRegistry()
	registerGlobalEntry()
	registerModuleToLua()
	if err := luaL.DoFile(cfg.WorkPath + "/" + bootstrapLua); err != nil {
		log.Errorf("load [%s] in path [%s], error: %s", bootstrapLua, cfg.WorkPath, err.Error())
		return err
	}
	log.Infof("lua vm machine inited.")
	return nil
}

func GetLuaState() *lua.LState {
	return luaL
}

func CallLuaMethod(f lua.LValue, nRet int, args ...lua.LValue) error {
	if luaL == nil || f == nil {
		return errors.New("luaL or function is nil")
	}
	if err := luaL.CallByParam(luaFunctionWrapper(f, nRet), args...); err != nil {
		log.Warnf("call lua function[%s] error: %s", f.String(), err.Error())
		return err
	}
	return nil
}

func CallLuaMethodByName(t lua.LValue, name string, nRet int, args ...lua.LValue) error {
	field := luaL.GetField(t, name)
	switch field.Type() {
	case lua.LTFunction:
		fallthrough
	case lua.LTTable:
		return CallLuaMethod(field, nRet, args...)
	default:
		return fmt.Errorf("call %s but function not found", name)
	}
}
