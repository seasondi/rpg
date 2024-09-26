package engine

import (
	lua "github.com/seasondi/gopher-lua"
	"runtime"
	"time"
)

// 注册在entity身上的api
var entityApiExports = map[string]lua.LGFunction{
	/*
		addTimer: 添加定时器, self:addTimer(2000, 0, "callback", arg1, arg2, ...)
		参数1：定时器下次触发间隔毫秒, 数字或者字符串, 支持：ms,s,min,h,d
		参数2：定时器循环触发间隔毫秒, 数字或者字符串, 支持：ms,s,min,h,d
		参数3：回调函数名
		参数4-n: 回调函数的参数
		返回值: timerID (int64)
	*/
	"addTimer": addEntityTimer,
	/*
		cancelTimer: 取消定时器, self:cancelTimer(timerID)
		参数1：addTimer返回的定时器ID
		返回值：无
	*/
	"cancelTimer": cancelEntityTimer,
	/*
		destroy: 立即销毁entity自身, self:destroy(true)
		参数1：是否销毁前存盘,默认true,只对def中定义了Volatile.Persistent的entity生效
		返回值：无
	*/
	"destroy": destroyEntity,
	/*
		save: 主动保存entity, self:save(). entity会定时存盘,非必要不要调用.只对def中定义了Volatile.Persistent的entity生效
		参数：无
		返回值：无
	*/
	"save": saveEntity,
}

// 全局api
var entryApiExports = map[string]lua.LGFunction{
	/*
		getReloadFiles: 获取需要热更的文件前缀
		参数: 无
		返回值: 文件名前缀数组
	*/
	"getReloadFiles": getReloadFiles,
	/*
		platform: 获取平台名称
		参数: 无
		返回值：windows,linux
	*/
	"platform": getPlatform,
}

// 注册到debug的api
var debugApis = map[string]lua.LGFunction{
	/*
		getregistry: 获取registry
		参数：无
		返回值: registry index的table
	*/
	"getregistry": debugGetRegistry,
	/*
		upvalueid: 获取upValue
		参数1: 函数
		参数2: 下标
		返回值: upValue
	*/
	"upvalueid": debugUpValueId,
}

func registerApiToEntity(t *lua.LTable) {
	luaL.SetFuncs(t, entityApiExports)
}

func registerApiToEntry() {
	entry := luaL.GetGlobal(globalEntry).(*lua.LTable)
	luaL.SetFuncs(entry, entryApiExports)
}

func registerApiToRegistry() {
	dbg := luaL.GetGlobal("debug").(*lua.LTable)
	luaL.SetFuncs(dbg, debugApis)
}

func RegisterEntryApi(apis map[string]lua.LGFunction) {
	entry := luaL.GetGlobal(globalEntry).(*lua.LTable)
	luaL.SetFuncs(entry, apis)
}

func addEntityTimer(L *lua.LState) int {
	top := L.GetTop()
	//1: entity table
	//2: first tick ms
	//3: repeat ms
	//4: cb function name
	//5-n: args...
	t := L.CheckTable(1)
	afterTime := L.CheckAny(2)
	repeatTime := L.CheckAny(3)
	cb := L.CheckString(4)

	ms := int64(0)
	repeatMs := int64(0)

	switch afterTime.Type() {
	case lua.LTNumber:
		ms = int64(afterTime.(lua.LNumber))
	case lua.LTString:
		var err error
		if ms, err = parseTimerString(afterTime.String()); err != nil {
			log.Errorf("add timer failed, trigger time string parse error: %s, stack: %s", err.Error(), GetLuaTraceback())
			L.Push(lua.LNumber(0))
			return 1
		}
	default:
		log.Errorf("add timer failed, trigger time must be number or string, stack: %s", GetLuaTraceback())
		L.Push(lua.LNumber(0))
		return 1
	}

	switch repeatTime.Type() {
	case lua.LTNumber:
		repeatMs = int64(repeatTime.(lua.LNumber))
	case lua.LTString:
		var err error
		if repeatMs, err = parseTimerString(repeatTime.String()); err != nil {
			log.Errorf("add timer failed, repeat time string parse error: %s, stack: %s", err.Error(), GetLuaTraceback())
			L.Push(lua.LNumber(0))
			return 1
		}
	default:
		log.Errorf("add timer failed, repeat time must be number or string, stack: %s", GetLuaTraceback())
		L.Push(lua.LNumber(0))
		return 1
	}

	entityId := entityIdFromLua(t, entityFieldId)
	if method := luaL.GetField(t, cb); method.Type() != lua.LTFunction {
		log.Warnf("add timer failed, [%s] is not a entity[%v] function", cb, entityId)
		L.Push(lua.LNumber(0))
		return 1
	}
	ent := GetEntityManager().GetEntityById(entityId)
	if ent == nil {
		L.Push(lua.LNumber(0))
		return 1
	}

	params := []interface{}{entityId, lua.LString(cb)}
	for i := 5; i <= top; i++ {
		params = append(params, L.CheckAny(i))
	}
	timerId := ent.addEntityTimer(time.Duration(ms)*time.Millisecond, time.Duration(repeatMs)*time.Millisecond, entityScriptTimerCallback, params...)
	L.Push(lua.LNumber(timerId))
	return 1
}

func cancelEntityTimer(L *lua.LState) int {
	//1: entity table
	//2: timerId
	t := L.CheckTable(1)
	timerId := int64(L.CheckNumber(2))
	ent := GetEntityManager().GetEntityByLua(t)
	if ent == nil {
		return 0
	}
	ent.cancelEntityTimer(timerId)
	return 0
}

func destroyEntity(L *lua.LState) int {
	//1: entity table
	//2: isSaveDB

	top := L.GetTop()
	t := L.CheckTable(1)
	isSaveDB := true
	if top > 1 {
		isSaveDB = L.CheckBool(2)
	}
	ent := GetEntityManager().GetEntityByLua(t)
	if ent == nil {
		entityId := entityIdFromLua(t, entityFieldId)
		log.Warnf("destroy entity[%d] from lua but not found", entityId)
		return 0
	}
	ent.Destroy(isSaveDB, true)
	return 0
}

func saveEntity(L *lua.LState) int {
	//1: entity table

	t := L.CheckTable(1)
	ent := GetEntityManager().GetEntityByLua(t)
	if ent == nil {
		entityId := entityIdFromLua(t, entityFieldId)
		log.Warnf("saveEntity entity[%d] from lua but not found", entityId)
		return 0
	}
	//立即存盘
	ent.SaveToDB()

	return 0
}

func debugGetRegistry(L *lua.LState) int {
	v := L.Get(lua.RegistryIndex)
	L.Push(v)
	return 1
}

func debugUpValueId(L *lua.LState) int {
	f := L.CheckFunction(1)
	idx := L.CheckNumber(2)
	_, v := L.GetUpvalue(f, int(idx))
	L.Push(v)
	return 1
}

func getReloadFiles(L *lua.LState) int {
	t := luaL.NewTable()

	idx := 0
	for name := range GetEntityManager().metas {
		ifs := defMgr.GetInterfaces(name)
		for _, interfaceName := range ifs {
			idx += 1
			t.RawSetInt(idx, lua.LString(interfaceName))
		}
		idx += 1
		t.RawSetInt(idx, lua.LString(name))
	}

	L.Push(t)
	return 1
}

func getPlatform(L *lua.LState) int {
	L.Push(lua.LString(runtime.GOOS))
	return 1
}
