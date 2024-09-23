package engine

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
)

func GetRobotManager() *robotManager {
	return rbMgr
}

type robotManager struct {
	metas             map[string]*lua.LTable  //entity类型名称->元表
	allEntities       map[EntityIdType]*Robot //entityId->entity
	luaEntityToEntity map[*lua.LTable]*Robot  //脚本层entity到引擎层entity的映射
}

func initRobotManager() error {
	rbMgr = new(robotManager)
	rbMgr.init()

	log.Infof("Robot manager inited.")
	return nil
}

func (em *robotManager) init() {
	em.metas = make(map[string]*lua.LTable)
	em.allEntities = make(map[EntityIdType]*Robot)
	em.luaEntityToEntity = make(map[*lua.LTable]*Robot)
}

func (em *robotManager) CreateEntity(entityId EntityIdType, entityName string, props map[string]interface{}, conn *TcpClient) (*Robot, error) {
	rb, err := newRobot(entityId, entityName, conn)
	if err != nil {
		return nil, err
	}
	rb.loadData(props)
	if err = rb.onInit(); err != nil {
		em.RemoveEntity(rb)
		return nil, err
	}

	return rb, err
}

func (em *robotManager) RemoveEntity(rb *Robot) {
	em.unRegisterEntity(rb)
}

func (em *robotManager) GetEntityById(entityId EntityIdType) *Robot {
	return em.allEntities[entityId]
}

func (em *robotManager) GetEntityByLua(t *lua.LTable) *Robot {
	return em.luaEntityToEntity[t]
}

func (em *robotManager) registerEntity(rb *Robot) {
	em.allEntities[rb.entityId] = rb
	em.luaEntityToEntity[rb.luaEntity] = rb
	luaL.RawSet(getLuaEntities(), EntityIdToLua(rb.entityId), rb.luaEntity)
}

func (em *robotManager) unRegisterEntity(rb *Robot) {
	delete(em.allEntities, rb.entityId)
	delete(em.luaEntityToEntity, rb.luaEntity)
	luaL.RawSet(getLuaEntities(), EntityIdToLua(rb.entityId), lua.LNil)
}

func (em *robotManager) genMetaTable(name string) *lua.LTable {
	if _, ok := em.metas[name]; ok == false {
		newMetaTable := luaL.NewTable()
		luaL.SetGlobal(name, newMetaTable)
		luaL.SetField(newMetaTable, "__index", luaL.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(2)
			value := L.RawGet(newMetaTable, lua.LString(key))
			L.Push(value)
			return 1
		}))
		luaL.SetField(newMetaTable, "__newindex", luaL.NewFunction(func(L *lua.LState) int {
			key := L.CheckString(2)
			value := L.CheckAny(3)
			L.RawSet(newMetaTable, lua.LString(key), value)
			return 0
		}))
		luaL.SetField(newMetaTable, "__tostring", luaL.NewFunction(func(L *lua.LState) int {
			self := L.CheckTable(1)
			str := fmt.Sprintf("entity[%s:%s]", L.GetField(self, entityFieldName), L.GetField(self, entityFieldId))
			L.Push(lua.LString(str))
			return 1
		}))
		luaL.SetField(newMetaTable, entityFieldName, lua.LString(name))
		luaL.SetField(newMetaTable, entityFieldType, lua.LString("entity"))
		em.metas[name] = newMetaTable
	}
	return em.metas[name]
}

func (em *robotManager) IsEntityLoaded(entityName string) bool {
	if _, find := em.metas[entityName]; find {
		return true
	}
	return false
}
