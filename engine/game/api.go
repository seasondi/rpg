package main

import (
	"context"
	lua "github.com/seasondi/gopher-lua"
	clientV3 "go.etcd.io/etcd/client/v3"
	"rpg/engine/engine"
	"rpg/engine/message"
	"time"
)

func registerApi() {
	engine.RegisterEntryApi(gameAPI)
}

var gameAPI = map[string]lua.LGFunction{
	/*
		loadEntityFromDB: 从数据库加载entity
		参数1: entityId
		参数2: 回调函数, 格式function(entityId, errMsg)
		参数3: 超时时间,秒(未指定则使用默认值)
		返回值: 无
	*/
	"loadEntityFromDB": loadEntityFromDB,
	/*
		executeDBRawCommand: 执行数据库命令
		参数1: 数据库类型 engine.DBType枚举
		参数2: 任务类型 engine.DBTaskType枚举
		参数3: 数据库名称
		参数4: 集合名称
		参数5: 查询条件,数组类型,只支持number,string,bool
		参数6: 更新数据,字典类型,查询时该字段传空table
		参数7: 回调函数, 格式function(dataTable, errMsg)
		参数8: 超时时间,秒(未指定则使用默认值)
	*/
	"executeDBRawCommand": executeDBRawCommand,
	/*
		setConnInfo: 设置entity的连接信息
		参数1：clientMailBox
		参数2：entityId
		返回值：true: 设置成功, false: 设置失败
	*/
	"setConnInfo": setConnInfo,
	/*
		getConnInfo: 获取entity的连接信息
		参数1: entityId
		返回值：连接信息的table,如果没有连接返回nil
	*/
	"getConnInfo": getConnInfo,
	/*
		callEntity: 调用entity的方法
		参数1: 被调用的entity id
		参数2: 被调用的函数名(需要定义在def中)
		参数3-n: 函数参数
		返回值: 无
	*/
	"callEntity": callEntity,
	/*
		callStub: 调用stub的方法
		参数1: 被调用的stub名称
		参数2: 被调用的函数名(需要定义在def中)
		参数3-n: 函数参数
		返回值: 无
	*/
	"callStub": callStub,
	/*
		createEntityLocally: 在本进程创建一个entity
		参数1：entity名称
		返回值：成功: entityId, 失败：0
	*/
	"createEntityLocally": createEntityLocally,
	/*
		createEntityAnywhere: 根据负载选择一个game进程创建entity
		参数1：entity名称
		参数2：回调函数, function(entityId, errMsg) end
		返回值：无
	*/
	"createEntityAnywhere": createEntityAnywhere,
	/*
		setTimeOffset: 设置时间偏移
		参数1: 相对于系统时间的偏移秒数
		参数2: 是否广播给其他game, 默认true
		返回值: true/false
	*/
	"setTimeOffset": setTimeOffset,
}

func loadEntityFromDB(L *lua.LState) int {
	//1: entityId
	//2: 回调函数
	//3: 超时时间(可选)

	entityId := L.CheckNumber(1)
	cb := L.CheckAny(2)
	if cb.Type() != lua.LTFunction && cb.Type() != lua.LTTable {
		log.Errorf("loadEntityFromDB args 2 must be function or callable table")
		return 0
	}

	timeout := 3 * time.Second
	if L.GetTop() > 2 {
		t := L.CheckNumber(3)
		if t > 0 {
			timeout = time.Duration(t) * time.Second
		}
	}

	getDBProxy().loadEntityFromDB(engine.EntityIdType(entityId), cb, timeout)
	return 0
}

func setConnInfo(L *lua.LState) int {
	//1: clientMailBox
	//2: entityId

	mailBox := L.CheckTable(1)
	entityId := L.CheckNumber(2)
	//primary := L.CheckBool(3)
	mb := engine.ClientMailBoxFromLua(mailBox)
	if mb == nil {
		L.Push(lua.LBool(false))
		return 1
	}
	if err := engine.GetEntityManager().UpdateEntityConnInfo(mb, engine.EntityIdType(entityId), true); err != nil {
		L.Push(lua.LBool(false))
	} else {
		L.Push(lua.LBool(true))
	}
	return 1
}

func getConnInfo(L *lua.LState) int {
	//1: entityId
	entityId := L.CheckNumber(1)
	ent := engine.GetEntityManager().GetEntityById(engine.EntityIdType(entityId))
	if ent == nil {
		L.Push(lua.LNil)
		return 1
	}
	client := ent.GetClient()
	if client == nil {
		L.Push(lua.LNil)
		return 1
	}
	t := L.NewTable()
	t.RawSetString("gate_name", lua.LString(client.MailBox().GateName))
	t.RawSetString("client_id", lua.LNumber(client.MailBox().ClientId))
	L.Push(t)
	return 1
}

func callEntity(L *lua.LState) int {
	//1: entityId
	//2: def server method name
	//3-n: args
	top := L.GetTop()
	entityId := L.CheckNumber(1)
	funcName := L.CheckString(2)

	ent := engine.GetEntityManager().GetEntityById(engine.EntityIdType(entityId))
	if ent != nil {
		args := make([]lua.LValue, 0, 0)
		for i := 3; i <= top; i++ {
			args = append(args, L.CheckAny(i))
		}
		if err := ent.CallDefServerMethod(funcName, args, false); err != nil {
			log.Errorf("call %s function[%s] error: %s", ent.String(), funcName, err.Error())
			return 0
		}
	} else {
		ctx, cancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
		defer cancel()

		result := engine.EtcdValue{}
		if err := engine.GetRedisMgr().Get(ctx, engine.GetRedisEntityKey(engine.EntityIdType(entityId)), &result); err != nil {
			log.Warnf("call entity[%d] function[%s], error: %s", entityId, funcName, err.Error())
			return 0
		}
		targetServer, ok := result[engine.EtcdValueServer].(string)
		if !ok {
			log.Warnf("call entity[%d] function[%s] but target server not found, etcd info: %+v", entityId, funcName, result)
			return 0
		}
		args := []interface{}{funcName}
		for i := 3; i <= top; i++ {
			val := L.CheckAny(i)
			if val.Type() == lua.LTTable {
				r := engine.TableToMap(val.(*lua.LTable))
				args = append(args, r)
			} else if val.Type() == lua.LTNil {
				args = append(args, engine.LuaTableValueNilField)
			} else {
				args = append(args, L.CheckAny(i))
			}
		}
		dataMap := map[string]interface{}{
			engine.ClientMsgDataFieldEntityID: entityId,
			engine.ClientMsgDataFieldArgs:     args,
		}
		data, err := engine.GetProtocol().Marshal(dataMap)
		if err != nil {
			log.Warnf("call entity[%d] function[%s] but message marshal error: %s", entityId, funcName, err.Error())
			return 0
		}
		msg := &message.GameRouterRpc{
			Target: targetServer,
			Data:   data,
		}
		if err = getGateProxy().SendToGate(engine.GenMessageHeader(engine.ServerMessageTypeEntityRouter, 0), msg, nil); err != nil {
			log.Warnf("call entity[%d] function[%s] msg send error: %s", entityId, funcName, err.Error())
			return 0
		}
	}
	return 0
}

func callStub(L *lua.LState) int {
	top := L.GetTop()
	stubName := L.CheckString(1)
	funcName := L.CheckString(2)

	entityId := getStubProxy().GetStubId(stubName)
	if entityId <= 0 {
		log.Warnf("call stub[%s] but not found", stubName)
		return 0
	}

	ent := engine.GetEntityManager().GetEntityById(entityId)
	if ent != nil {
		args := make([]lua.LValue, 0, 0)
		for i := 3; i <= top; i++ {
			args = append(args, L.CheckAny(i))
		}
		if err := ent.CallDefServerMethod(funcName, args, false); err != nil {
			log.Errorf("call %s function[%s] error: %s", ent.String(), funcName, err.Error())
			return 0
		}
	} else {
		ctx, cancel := context.WithTimeout(context.TODO(), 500*time.Millisecond)
		defer cancel()
		result := engine.EtcdValue{}
		if err := engine.GetRedisMgr().Get(ctx, engine.GetRedisEntityKey(entityId), &result); err != nil {
			log.Warnf("call entity[%d] function[%s] but not found entity", entityId, funcName)
			return 0
		}
		targetServer, ok := result[engine.EtcdValueServer].(string)
		if !ok {
			log.Warnf("call entity[%d] function[%s] but target server not found, etcd info: %+v", entityId, funcName, result)
			return 0
		}
		args := []interface{}{funcName}
		for i := 3; i <= top; i++ {
			val := L.CheckAny(i)
			if val.Type() == lua.LTTable {
				r := engine.TableToMap(val.(*lua.LTable))
				args = append(args, r)
			} else if val.Type() == lua.LTNil {
				args = append(args, engine.LuaTableValueNilField)
			} else {
				args = append(args, L.CheckAny(i))
			}
		}
		dataMap := map[string]interface{}{
			engine.ClientMsgDataFieldEntityID: entityId,
			engine.ClientMsgDataFieldArgs:     args,
		}
		data, err := engine.GetProtocol().Marshal(dataMap)
		if err != nil {
			log.Warnf("call entity[%d] function[%s] but message marshal error: %s", entityId, funcName, err.Error())
			return 0
		}
		msg := &message.GameRouterRpc{
			Target: targetServer,
			Data:   data,
		}
		if err = getGateProxy().SendToGate(engine.GenMessageHeader(engine.ServerMessageTypeEntityRouter, 0), msg, nil); err != nil {
			log.Warnf("call entity[%d] function[%s] msg send error: %s", entityId, funcName, err.Error())
			return 0
		} else {
			log.Tracef("call entity[%d] function[%s], target server[%s] send success", entityId, funcName, targetServer)
		}
	}
	return 0
}

func createEntityLocally(L *lua.LState) int {
	//1: entity name

	entityName := L.CheckString(1)
	ent, err := engine.GetEntityManager().CreateEntity(entityName)
	if err != nil {
		log.Errorf("createEntityLocally error: %s", err.Error())
	}
	entityId := lua.LNumber(0)
	if ent != nil {
		entityId = engine.EntityIdToLua(ent.GetEntityId())
	}
	L.Push(entityId)
	return 1
}

func createEntityAnywhere(L *lua.LState) int {
	//1: entity name
	//2: 回调函数

	entityName := L.CheckString(1)
	cb := L.CheckAny(2)
	if cb.Type() != lua.LTFunction && cb.Type() != lua.LTTable {
		log.Errorf("createEntityAnywhere args 2 must be function or callable table")
		return 0
	}
	getGateProxy().CreateEntityAnywhere(entityName, cb)
	return 0
}

func executeDBRawCommand(L *lua.LState) int {
	//1: dbType
	//2: taskType
	//3: database
	//4: collection
	//5: 查询条件
	//6: 数据
	//7: 回调函数
	//8: 超时时间

	dbType := L.CheckNumber(1)
	taskType := L.CheckNumber(2)
	database := L.CheckString(3)
	collection := L.CheckString(4)
	filter := L.CheckTable(5) //格式必须是形如: {{"a", 1}, {"b", 2}}
	data := L.CheckTable(6)
	cb := L.CheckAny(7)

	timeout := 2 * time.Second
	if L.GetTop() >= 8 {
		sec := L.CheckNumber(8)
		timeout = time.Duration(sec) * time.Second
	}
	if dbType < 0 || dbType >= lua.LNumber(engine.DBTypeMax) {
		log.Errorf("dbRawCommandQuery db type %d error, stack: %s", dbType, engine.GetLuaTraceback())
		return 0
	}

	if taskType < 0 || taskType >= lua.LNumber(engine.DBTaskTypeMax) {
		log.Errorf("dbRawCommandQuery task type %d error, stack: %s", taskType, engine.GetLuaTraceback())
		return 0
	}
	if cb.Type() != lua.LTFunction && cb.Type() != lua.LTTable {
		log.Errorf("executeDBRawCommand callback must be function or callable table")
		return 0
	}
	bsonFilter, err := engine.LuaArrayToBsonD(filter)
	if err != nil {
		log.Errorf("executeDBRawCommand parse filter error: %s", err.Error())
		return 0
	}
	getDBProxy().executeDBRawCommand(engine.DBType(dbType), engine.DBTaskType(taskType),
		database, collection, bsonFilter, engine.TableToMap(data), cb, timeout)

	return 0
}

func setTimeOffset(L *lua.LState) int {
	//1: offset seconds
	//2: broadcast

	if engine.GetConfig().Release {
		log.Errorf("set server time is forbidden in release")
		L.Push(lua.LBool(false))
		return 1
	}

	offset := L.CheckNumber(1)
	if offset < 0 {
		log.Errorf("setTimeOffset failed, offset: %d must >= 0", offset)
		L.Push(lua.LBool(false))
		return 1
	}
	broadcast := true
	if L.GetTop() >= 2 {
		broadcast = L.CheckBool(2)
	}
	//广播给其他进程
	if broadcast {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()
		currServiceName := engine.ServiceName()
		servers := engine.GetEtcd().Get(ctx, engine.GetEtcdPrefixWithServer(engine.ServiceGamePrefix), clientV3.WithPrefix())
		msg := &message.SetServerTimeOffset{Offset: int32(offset)}
		for _, server := range servers {
			if server.Key() == currServiceName {
				continue
			}
			msg.Targets = append(msg.Targets, server.Key())
		}
		_ = getGateProxy().SendToGate(engine.GenMessageHeader(engine.ServerMessageTypeSetServerTime, 0), msg, nil)
	}

	ret := engine.SetTimeOffset(int32(offset))
	L.Push(lua.LBool(ret))
	return 1
}
