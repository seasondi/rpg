package main

import (
	"errors"
	lua "github.com/seasondi/gopher-lua"
	"rpg/engine/engine"
)

//==================================DB加载entity回调==================================

type queryDBEntityCallback struct {
	timerId int64
	luaFunc lua.LValue
}

func (m *queryDBEntityCallback) setTimerId(id int64) {
	m.timerId = id
}

func (m *queryDBEntityCallback) cancelTimer() {
	if m.timerId > 0 {
		engine.GetTimer().Cancel(m.timerId)
		m.timerId = 0
	}
}

func (m *queryDBEntityCallback) Process(err error, params ...interface{}) {
	id := engine.EntityIdType(0)
	if err != nil {
		log.Errorf("queryDBEntityCallback error: %s", err.Error())
	} else if len(params) >= 2 {
		entityId := engine.InterfaceToInt(params[0])
		if data, ok := params[1].(map[string]interface{}); ok {
			if len(data) > 0 {
				if ent := engine.GetEntityManager().CreateEntityFromData(engine.EntityIdType(entityId), data); ent != nil {
					id = ent.GetEntityId()
				} else {
					err = errors.New("create entity failed")
				}
			} else {
				err = errors.New("entity not found")
			}
		} else {
			log.Warnf("queryDBEntityCallback invalid params: %+v", params)
			err = errors.New("invalid params")
		}
	} else {
		log.Warnf("queryDBEntityCallback invalid params length: %+v", params)
		err = errors.New("invalid params length")
	}
	args := []lua.LValue{engine.EntityIdToLua(id)}
	if err != nil {
		args = append(args, lua.LString(err.Error()))
	}
	_ = engine.CallLuaMethod(m.luaFunc, 0, args...)
}

//==================================在其他game创建entity回调==================================

type createEntityAnywhereCallback struct {
	timerId int64
	luaFunc lua.LValue
}

func (m *createEntityAnywhereCallback) setTimerId(id int64) {
	m.timerId = id
}

func (m *createEntityAnywhereCallback) cancelTimer() {
	if m.timerId > 0 {
		engine.GetTimer().Cancel(m.timerId)
		m.timerId = 0
	}
}

func (m *createEntityAnywhereCallback) Process(err error, params ...interface{}) {
	id := engine.EntityIdType(0)
	if err != nil {
		log.Errorf("createEntityAnywhereCallback return error: %s", err.Error())
	} else if len(params) > 0 {
		id = engine.EntityIdType(engine.InterfaceToInt(params[0]))
	} else {
		log.Warnf("createEntityAnywhereCallback invalid params length: %+v", params)
	}
	args := []lua.LValue{engine.EntityIdToLua(id)}
	if err != nil {
		args = append(args, lua.LString(err.Error()))
	}
	_ = engine.CallLuaMethod(m.luaFunc, 0, args...)
}

//==================================entity销毁时存盘回调==================================

type saveEntityOnDestroyCallback struct {
	timerId int64
}

func (m *saveEntityOnDestroyCallback) setTimerId(id int64) {
	m.timerId = id
}

func (m *saveEntityOnDestroyCallback) cancelTimer() {
	if m.timerId > 0 {
		engine.GetTimer().Cancel(m.timerId)
		m.timerId = 0
	}
}

func (m *saveEntityOnDestroyCallback) Process(err error, params ...interface{}) {
	id := engine.EntityIdType(0)
	if err != nil {
		log.Warnf("saveEntityOnDestroyCallback return error: %s, do nothing", err.Error())
		return
	} else if len(params) > 0 {
		id = engine.EntityIdType(engine.InterfaceToInt(params[0]))
	} else {
		log.Warnf("saveEntityOnDestroyCallback invalid params length: %+v, do nothing", params)
		return
	}
	if ent := engine.GetEntityManager().GetEntityById(id); ent != nil {
		ent.SavedOnDestroyCallback()
	} else {
		log.Errorf("saveEntityOnDestroyCallback but not found entity, id: %d", id)
	}
}

//==================================DB操作回调==================================

type dbRawCommandCallback struct {
	timerId int64
	luaFunc lua.LValue
}

func (m *dbRawCommandCallback) setTimerId(id int64) {
	m.timerId = id
}

func (m *dbRawCommandCallback) cancelTimer() {
	if m.timerId > 0 {
		engine.GetTimer().Cancel(m.timerId)
		m.timerId = 0
	}
}

func (m *dbRawCommandCallback) Process(err error, params ...interface{}) {
	args := make([]lua.LValue, 0)
	if err != nil {
		log.Errorf("dbRawCommandCallback error: %s", err.Error())
	} else if len(params) < 2 {
		log.Warnf("dbRawCommandCallback invalid params length: %+v", params)
	} else {
		switch data := params[1].(type) {
		case map[string]interface{}:
			args = append(args, engine.MapToTable(data))
		case []map[string]interface{}:
			args = append(args, engine.ArrayMapToTable(data))
		}
	}
	if err != nil {
		args = append(args, lua.LString(err.Error()))
	}
	_ = engine.CallLuaMethod(m.luaFunc, 0, args...)
}
