package engine

import (
	"fmt"
	lua "github.com/seasondi/gopher-lua"
	"strconv"
)

type Robot struct {
	entityId    EntityIdType //id
	entityName  string       //名称
	luaEntity   *lua.LTable  //脚本层entity
	serverTable *lua.LTable  //服务端rpc函数信息
	def         *entityDef   //def定义
	server      *TcpClient   //服务端连接
}

func newRobot(entityId EntityIdType, entityName string, conn *TcpClient) (*Robot, error) {
	rb := new(Robot)
	rb.entityId = entityId
	rb.entityName = entityName
	rb.server = conn
	if err := rb.init(); err != nil {
		return nil, err
	}
	return rb, nil
}

func (m *Robot) init() error {
	m.luaEntity = luaL.NewTable()
	luaL.SetMetatable(m.luaEntity, GetRobotManager().genMetaTable(m.entityName))
	m.luaEntity.RawSetString(entityFieldId, EntityIdToLua(m.entityId))
	m.def = defMgr.GetEntityDef(m.entityName)
	if m.def == nil {
		return fmt.Errorf("cannot find entity[%s] def, please check entities.xml", m.entityName)
	}
	GetRobotManager().registerEntity(m)
	m.registerDef()
	registerApiToEntity(m.luaEntity)
	return nil
}

func (m *Robot) String() string {
	return "entity[" + m.entityName + ":" + strconv.FormatInt(int64(m.entityId), 10) + "]"
}

func (m *Robot) EntityID() EntityIdType {
	return m.entityId
}

func (m *Robot) loadData(props map[string]interface{}) {
	for name, value := range props {
		propInfo := m.def.prop(name)
		if propInfo == nil {
			log.Debugf("%s def has no prop[%s]", m.String(), name)
			continue
		}
		m.luaEntity.RawSetString(name, propInfo.dt.ParseToLua(value))
	}
}

func (m *Robot) registerDef() {
	m.registerProperties()
	m.registerServerMethods()
}

func (m *Robot) registerProperties() {
	if m.entityName != m.def.entityName {
		log.Errorf("register props to entity[%s], but def is [%s]", m.entityName, m.entityName)
		return
	}
	for propName, prop := range m.def.properties {
		if prop.config.Flags == noClient {
			continue
		}
		val := prop.dt.Default()
		if prop.dt.Name() == dataTypeNameSyncTable {
			val.(*lua.LTable).RawSetString(SyncTableFieldOwner, EntityIdToLua(m.entityId))
		}
		luaL.RawSet(m.luaEntity, lua.LString(propName), val)
	}
}

func (m *Robot) registerServerMethods() {
	m.serverTable = luaL.NewTable()
	luaL.SetField(m.luaEntity, "server", m.serverTable)

	for _, method := range m.def.serverMethods {
		if method.exposed {
			luaL.SetField(m.serverTable, method.methodName, m.newServerFunction(method.methodName, m.entityId))
		}
	}
}

func (m *Robot) newServerFunction(name string, owner EntityIdType) *lua.LTable {
	t := luaL.NewTable()
	meta := luaL.NewTable()
	luaL.SetField(meta, "name", lua.LString(name))
	luaL.SetField(meta, "owner", EntityIdToLua(owner))
	luaL.SetField(meta, "__index", meta)
	luaL.SetField(meta, "__call", luaL.NewFunction(func(L *lua.LState) int {
		serverTable := L.CheckTable(1)
		methodName := L.GetField(serverTable, "name").String()
		id := entityIdFromLua(serverTable, "owner")
		rb := GetRobotManager().GetEntityById(id)
		if rb == nil {
			log.Debugf("call entity[%d] server method[%s] but entity is nil", id, methodName)
			return 0
		}
		if rb.server == nil {
			log.Debugf("call %s server method[%s] but client conn is nil", rb.String(), methodName)
			return 0
		}
		method := rb.def.getServerMethod(methodName, true)
		if method == nil {
			log.Debugf("call %s server method[%s] but method not found", rb.String(), methodName)
			return 0
		}
		needArgsNum := len(method.args)
		//lua栈上第一个参数是self.server
		if L.GetTop() < needArgsNum+1 {
			log.Errorf("call server method[%s] need %d arg(s) but got %d%s", methodName, needArgsNum, L.GetTop()-1, GetLuaTraceback())
			return 0
		} else {
			args := []interface{}{rb.def.getServerMethodMaskName(name)}
			passed := true
			//check params
			for i, argPropType := range method.args {
				arg := L.CheckAny(i + 2)
				if argPropType.dt.IsSameType(arg) == false {
					log.Errorf("call server method[%s], arg[%d] need[%s] but got[%s(%s)]%s", methodName, i+1, argPropType.dt.Type(), arg.String(), arg.Type(), GetLuaTraceback())
					passed = false
					break
				} else {
					args = append(args, argPropType.dt.ParseFromLua(arg))
				}
			}
			if passed {
				buf := map[string]interface{}{
					ClientMsgDataFieldEntityID: rb.entityId,
					ClientMsgDataFieldArgs:     args,
				}
				if data, err := genC2SMessage(uint8(ClientMsgTypeEntityRpc), buf); err == nil {
					_, _ = rb.server.Send(data)
				} else {
					log.Errorf("%s call server method[%s] error: %s", rb.String(), name, err.Error())
				}
			}
		}
		return 0
	}))
	luaL.SetMetatable(t, meta)
	return t
}

func (m *Robot) onInit() error {
	if err := CallLuaMethodByName(m.luaEntity, onEntityCreated, 0, m.luaEntity); err != nil {
		return err
	}
	return nil
}

func (m *Robot) CheckDefClientMethod(method string, args []lua.LValue) (string, error) {
	mi := m.def.getClientMethod(method, false)
	if mi == nil {
		return method, fmt.Errorf("[%s] is not def server method for %s", method, m.String())
	} else {
		argLen := len(args)
		for i := 0; i < len(mi.args); i++ {
			if argLen <= i {
				break
			}
			if mi.args[i].dt.IsSameType(args[i]) == false {
				return method, fmt.Errorf("function[%s] arg[%d] expect %s but got %+v(%s)", method, i+1, mi.args[i].dt.Type(), args[i], args[i].Type())
			}
		}
	}
	return mi.methodName, nil
}

func (m *Robot) CallDefClientMethod(method string, args []lua.LValue) error {
	name, err := m.CheckDefClientMethod(method, args)
	if err != nil {
		return err
	}

	params := append([]lua.LValue{m.luaEntity}, args...)
	if err = CallLuaMethodByName(m.luaEntity, name, 0, params...); err != nil {
		return err
	}
	return nil
}

func (m *Robot) OnServerSyncProp(name string, value lua.LValue) {
	prop := m.def.prop(name)
	if prop == nil {
		return
	}
	if prop.dt.IsSameType(value) == false {
		log.Debugf("%s prop[%s] type check failed, value: %+v", m.String(), name, value)
		return
	}
	old := luaL.GetField(m.luaEntity, name)
	luaL.SetField(m.luaEntity, name, value)
	if f := luaL.GetField(m.luaEntity, "on_update_"+name); f.Type() == lua.LTFunction {
		_ = CallLuaMethod(NewLuaMethod(f, "on_update_"+name), 0, m.luaEntity, old)
	}
}

func (m *Robot) OnServerSyncPropPart(name string, key lua.LValue, value lua.LValue) {
	prop := m.def.prop(name)
	if prop == nil {
		return
	}

	v := luaL.GetField(m.luaEntity, name)
	if v.Type() != lua.LTTable {
		return
	}
	t := v.(*lua.LTable)
	if f := luaL.GetField(m.luaEntity, "on_update_"+name); f.Type() == lua.LTFunction {
		old := luaL.NewTable()
		for ck, cv := t.Next(lua.LNil); ck != lua.LNil; ck, cv = t.Next(ck) {
			luaL.RawSet(old, ck, cv)
		}
		luaL.RawSet(t, key, value)
		_ = CallLuaMethod(NewLuaMethod(f, "on_update_"+name), 0, m.luaEntity, old, key)
	} else {
		luaL.RawSet(t, key, value)
	}
}
