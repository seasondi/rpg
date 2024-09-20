package engine

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/panjf2000/gnet"
	lua "github.com/yuin/gopher-lua"
	"time"
)

type entityGateConnFinder func(serverName string) gnet.Conn
type entityIdMap map[EntityIdType]interface{}
type clientIdToEntitiesMap map[ConnectIdType]entityIdMap //一个client连接可能会关联一个avatar与一个account

func getLuaEntities() *lua.LTable {
	v := getLuaEntryValue(entitiesEntry)
	return v.(*lua.LTable)
}

func GetEntityManager() *entityManager {
	return entityMgr
}

type entityManager struct {
	metas             map[string]*lua.LTable           //entity类型名称->元表
	allEntities       map[EntityIdType]*entity         //entityId->entity
	luaEntityToEntity map[*lua.LTable]*entity          //脚本层entity到引擎层entity的映射
	saveList          *list.List                       //待存盘列表,保存entity的存盘信息
	connFinder        entityGateConnFinder             //查询entity的gate连接
	connMap           map[string]clientIdToEntitiesMap //gate server name -> clientId->entities
}

func initEntityManager() error {
	entityMgr = new(entityManager)
	entityMgr.init()

	log.Infof("entity manager inited.")
	return nil
}

func (em *entityManager) init() {
	em.metas = make(map[string]*lua.LTable)
	em.allEntities = make(map[EntityIdType]*entity)
	em.luaEntityToEntity = make(map[*lua.LTable]*entity)
	em.saveList = list.New()
	em.connMap = make(map[string]clientIdToEntitiesMap)
}

func (em *entityManager) saveEntity(e *entity, saveType int) {
	if data := e.genSaveInfo(); data != nil {
		if saveType == saveTypeBack {
			em.saveList.PushBack(data)
		} else {
			em.saveList.PushFront(data)
		}
	}
}

func (em *entityManager) saveEntityOnDestroy(e *entity) {
	if data := e.genSaveInfo(); data != nil {
		data.PreferCallback = true
		em.saveList.PushFront(data)
	} else {
		log.Warnf("save %s on destroy but save data generate failed", e.String())
	}
}

func (em *entityManager) GetSaveList() *list.List {
	return em.saveList
}

func (em *entityManager) CreateEntity(entityName string) (*entity, error) {
	return em.CreateEntityWithId(generateEntityId(), entityName)
}

func (em *entityManager) GetEntityById(entityId EntityIdType) *entity {
	return em.allEntities[entityId]
}

func (em *entityManager) GetEntityByLua(t *lua.LTable) *entity {
	return em.luaEntityToEntity[t]
}

func (em *entityManager) CreateEntityWithId(entityId EntityIdType, entityName string) (*entity, error) {
	log.Infof("create entity[%s:%d]", entityName, entityId)
	if entityId == 0 {
		return nil, errors.New("entity id error")
	}
	ent, err := NewEntity(entityId, entityName)
	if err != nil {
		log.Errorf("entity create failed, entityName: %s, id: %d, error: %s", entityName, entityId, err.Error())
		return nil, err
	}
	if err = ent.completeEntity(); err != nil {
		return nil, err
	}

	log.Infof("create %s success", ent.String())
	return ent, nil
}

func (em *entityManager) registerEntity(ent *entity) {
	em.allEntities[ent.entityId] = ent
	em.luaEntityToEntity[ent.luaEntity] = ent
	luaL.RawSet(getLuaEntities(), EntityIdToLua(ent.entityId), ent.luaEntity)
}

func (em *entityManager) unRegisterEntity(ent *entity) {
	delete(em.allEntities, ent.entityId)
	delete(em.luaEntityToEntity, ent.luaEntity)
	luaL.RawSet(getLuaEntities(), EntityIdToLua(ent.entityId), lua.LNil)
}

func (em *entityManager) genMetaTable(name string) *lua.LTable {
	if _, ok := em.metas[name]; ok == false {
		newMetaTable := luaL.NewTable()
		luaL.SetGlobal(name, newMetaTable)
		luaL.SetField(newMetaTable, "__index", luaL.NewFunction(func(L *lua.LState) int {
			entTable := L.CheckTable(1)
			key := L.CheckString(2)
			ent := em.GetEntityByLua(entTable)
			if ent.def.isDefProp(key) {
				value := L.GetField(ent.propsTable, key)
				L.Push(value)
			} else {
				v := L.RawGet(newMetaTable, lua.LString(key))
				if v == lua.LNil {
					v = L.GetField(ent.propsTable, key)
				}
				L.Push(v)
			}
			return 1
		}))
		luaL.SetField(newMetaTable, "__newindex", luaL.NewFunction(func(L *lua.LState) int {
			entTable := L.CheckTable(1)
			propName := L.CheckString(2)
			newValue := L.CheckAny(3)
			ent := em.GetEntityByLua(entTable)
			if ent.def.isDefProp(propName) {
				dt := ent.def.propDataType(propName)
				typeName := dt.Name()
				switch typeName {
				case dataTypeNameStruct:
					value := luaL.GetField(entTable, propName).(*lua.LTable)
					if err := dt.(*dtStruct).AssignToStruct(value, newValue); err != nil {
						log.Errorf("value[%s](type[%s]) cannot set to prop[%s] error: %s stack info: %s", newValue, newValue.Type().String(), propName, err.Error(), GetLuaTraceback())
						return 0
					} else {
						newValue = value
					}
				case dataTypeNameSyncTable:
					value := luaL.GetField(entTable, propName).(*lua.LTable)
					if err := dt.(*dtSyncTable).AssignToSyncTable(value, newValue); err != nil {
						log.Errorf("value[%s](type[%s]) cannot set to prop[%s] error: %s stack info: %s", newValue, newValue.Type().String(), propName, err.Error(), GetLuaTraceback())
						return 0
					} else {
						newValue = value
					}
				default:
					if dt.IsSameType(newValue) == false {
						log.Errorf("value[(%s)](type[%s]) cannot set to prop[%s](type is %s) of %s, stack info: %s", newValue, newValue.Type().String(), propName, dt.Type(), ent.String(), GetLuaTraceback())
						return 0
					}
				}
				if typeName == dataTypeNameFloat {
					//浮点数按小数位截断
					newValue = lua.LNumber(dt.ParseFromLua(newValue).(float64))
				}
				L.SetField(ent.propsTable, propName, newValue)
				if ent.def.isSyncClientProp(propName) {
					ent.onSyncPropChanged(propName, newValue, dt)
				}
			} else {
				if !isEntityReserveProp(propName) {
					L.RawSet(entTable, lua.LString(propName), newValue)
				} else {
					log.Errorf("cannot modify READ ONLY prop[%s] of %s", propName, ent.String())
				}
			}
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

func (em *entityManager) CreateEntityFromData(entityId EntityIdType, data map[string]interface{}) *entity {
	if name, ok := data[entityFieldName].(string); ok == false || defMgr.GetEntityDef(name) == nil {
		log.Warnf("CreateEntityFromData unknown entity name[%s] for entityId[%d], data: %+v", name, entityId, data)
		return nil
	} else {
		ent, err := NewEntity(entityId, name)
		if err != nil {
			log.Errorf("entity create failed, entityName: %s, id: %d, error: %s", name, entityId, err.Error())
			return nil
		}
		ent.loadData(data)
		if err = ent.completeEntity(); err != nil {
			return nil
		}
		log.Infof("%s CreateEntityFromData success.", ent.String())
		return ent
	}
}

func (em *entityManager) SetConnFinder(finder entityGateConnFinder) {
	em.connFinder = finder
}

func (em *entityManager) GetEntityGateConn(e *entity) gnet.Conn {
	if e == nil || e.client == nil || em.connFinder == nil {
		return nil
	}
	return em.connFinder(e.client.mailbox.GateName)
}

func (em *entityManager) GetGateConn(gateName string) gnet.Conn {
	if em.connFinder == nil {
		return nil
	}
	return em.connFinder(gateName)
}

func (em *entityManager) GetEntitiesByConn(mb *ClientMailBox) []EntityIdType {
	r := make([]EntityIdType, 0)
	if mb == nil {
		return r
	}
	if info, ok := em.connMap[mb.GateName]; ok {
		for entityId := range info[mb.ClientId] {
			r = append(r, entityId)
		}
	}
	return r
}

func (em *entityManager) SetHeartbeat(gateName string, clientId ConnectIdType) {
	if info, ok := em.connMap[gateName]; ok {
		ids := info[clientId]
		for id := range ids {
			if ent := em.GetEntityById(id); ent != nil {
				ent.lastHeartBeatTime = time.Now()
			}
		}
	}
}

func (em *entityManager) UpdateEntityConnInfo(mailbox *ClientMailBox, entityId EntityIdType, primary bool) error {
	if mailbox == nil {
		return fmt.Errorf("client mailbox nil")
	}
	ent := em.GetEntityById(entityId)
	if ent == nil {
		return fmt.Errorf("entity[%d] not found", entityId)
	} else if ent.def.volatile.hasClient == false {
		return fmt.Errorf("%s cannot has client", ent.String())
	}

	if _, ok := em.connMap[mailbox.GateName]; ok == false {
		em.connMap[mailbox.GateName] = make(clientIdToEntitiesMap)
	}
	if _, ok := em.connMap[mailbox.GateName][mailbox.ClientId]; ok == false {
		em.connMap[mailbox.GateName][mailbox.ClientId] = make(entityIdMap)
	}
	em.connMap[mailbox.GateName][mailbox.ClientId][entityId] = true
	return ent.setClient(mailbox, primary)
}

func (em *entityManager) RemoveEntityConnInfo(gateName string, clientId ConnectIdType) {
	if clientsMap, ok := em.connMap[gateName]; ok {
		if entityMap, ok := clientsMap[clientId]; ok {
			for entityId := range entityMap {
				if ent := em.GetEntityById(entityId); ent != nil {
					if old := ent.GetClient(); old != nil && old.MailBox().GateName == gateName && old.MailBox().ClientId == clientId {
						_ = ent.setClient(nil, false)
					}
				}
			}
		}
		delete(clientsMap, clientId)
	}
}

func (em *entityManager) RemoveGateEntitiesConn(gateName string) {
	if clientsMap, ok := em.connMap[gateName]; ok {
		for _, entityMap := range clientsMap {
			for entityId := range entityMap {
				if ent := em.GetEntityById(entityId); ent != nil {
					_ = ent.setClient(nil, false)
				}
			}
		}
		delete(em.connMap, gateName)
	}
}

func (em *entityManager) EntityIsLoaded(entityName string) bool {
	if _, find := em.metas[entityName]; find {
		return true
	}
	return false
}
