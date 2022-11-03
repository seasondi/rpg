package engine

import (
	lua "github.com/yuin/gopher-lua"
	"strings"
)

var configExports = map[string]lua.LGFunction{
	"getConfig":    luaConfigHandler,
	"getServerKey": getServerKey,
}

func preloadConfig() {
	luaL.PreloadModule("config", configLoader)
}

func configLoader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), configExports)
	L.Push(mod)
	return 1
}

//config的table查询时无视大小写
func newConfigTable() *lua.LTable {
	t := luaL.NewTable()
	meta := luaL.NewTable()
	luaL.SetField(meta, "__index", luaL.NewFunction(func(L *lua.LState) int {
		tb := L.CheckTable(1)
		key := L.CheckString(2)
		v := L.RawGet(tb, lua.LString(strings.ToLower(key)))
		L.Push(v)
		return 1
	}))
	luaL.SetMetatable(t, meta)
	return t
}

func cfgArrayToTable(t *lua.LTable, arr []interface{}) {
	for i, v := range arr {
		tableIndex := i + 1
		if r, ok := v.(string); ok {
			luaL.RawSetInt(t, tableIndex, lua.LString(r))
		} else if r, ok := v.(float64); ok {
			luaL.RawSetInt(t, tableIndex, lua.LNumber(r))
		} else if r, ok := v.(bool); ok {
			luaL.RawSetInt(t, tableIndex, lua.LBool(r))
		} else if r, ok := v.(map[string]interface{}); ok {
			nt := newConfigTable()
			cfgMapToTable(nt, r)
			luaL.RawSetInt(t, tableIndex, nt)
		} else if r, ok := v.([]interface{}); ok {
			nt := luaL.NewTable()
			cfgArrayToTable(nt, r)
			luaL.RawSetInt(t, tableIndex, nt)
		}
	}
}

func cfgMapToTable(t *lua.LTable, m map[string]interface{}) {
	for k, v := range m {
		if r, ok := v.(string); ok {
			luaL.SetField(t, k, lua.LString(r))
		} else if r, ok := v.(float64); ok {
			luaL.SetField(t, k, lua.LNumber(r))
		} else if r, ok := v.(bool); ok {
			luaL.SetField(t, k, lua.LBool(r))
		} else if r, ok := v.(map[string]interface{}); ok {
			nt := newConfigTable()
			cfgMapToTable(nt, r)
			luaL.SetField(t, k, nt)
		} else if r, ok := v.([]interface{}); ok {
			nt := luaL.NewTable()
			cfgArrayToTable(nt, r)
			luaL.SetField(t, k, nt)
		}
	}
}

func luaConfigHandler(L *lua.LState) int {
	key := L.CheckString(1)
	v := cfg.vp.Get(key)
	if r, ok := v.(string); ok {
		L.Push(lua.LString(r))
	} else if r, ok := v.(float64); ok {
		L.Push(lua.LNumber(r))
	} else if r, ok := v.(bool); ok {
		L.Push(lua.LBool(r))
	} else if r, ok := v.(map[string]interface{}); ok {
		t := newConfigTable()
		cfgMapToTable(t, r)
		L.Push(t)
	} else if r, ok := v.([]interface{}); ok {
		t := L.NewTable()
		cfgArrayToTable(t, r)
		L.Push(t)
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

func getServerKey(L *lua.LState) int {
	L.Push(lua.LString(GetConfig().ServerKey()))
	return 1
}
