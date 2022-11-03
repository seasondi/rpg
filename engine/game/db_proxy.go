package main

import (
	"rpg/engine/engine"
	"rpg/engine/message"
	"errors"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"strconv"
	"time"
)

const (
	entityCollectionName = "Entity"
)

var dbMgr *dbProxy

//game连接db的TcpClient消息处理handler
type dbHandler struct {
}

func (m *dbHandler) Encode(data []byte) ([]byte, error) {
	return engine.GetProtocol().Encode(data)
}

func (m *dbHandler) Decode(data []byte) (int, []byte, error) {
	return engine.GetProtocol().Decode(data)
}

func (m *dbHandler) OnConnect(conn *engine.TcpClient) {
	log.Infof("connected to [%s]", conn.RemoteAddr())
	engine.GetServerStep().FinishHandler(initDBProxy)
}

func (m *dbHandler) OnDisconnect(conn *engine.TcpClient) {
	log.Infof("disconnect from [%s]", conn.RemoteAddr())
}

func (m *dbHandler) OnMessage(_ *engine.TcpClient, buf []byte) error {
	var err error
	ty := buf[0]
	switch ty {
	case engine.ServerMessageTypeDBCommand:
		err = processDBCommandResponse(buf[1:])
	}
	return err
}

func processDBCommandResponse(buf []byte) error {
	msg := message.DBCommandResponse{}
	if err := msg.Unmarshal(buf); err != nil {
		log.Errorf("received name message error: %s", err.Error())
		return err
	}
	var data interface{}
	var err error
	if msg.TaskType == uint32(engine.DBTaskTypeQueryMany) {
		r := make([]map[string]interface{}, 0)
		if err = engine.GetProtocol().UnMarshalTo(msg.Data, &r); err == nil {
			data = r
		}
	} else {
		data, err = engine.GetProtocol().UnMarshal(msg.Data)
	}
	if err != nil {
		log.Errorf("cannot UnMarshal entity[%d] data, error: %s", msg.EntityId, err.Error())
		return err
	}

	if msg.Ex != nil {
		var e error
		if len(msg.ErrMsg) > 0 {
			e = errors.New(string(msg.ErrMsg))
		}
		getCallbackMgr().Call(msg.Ex.Uuid, e, msg.EntityId, data)
	}
	return nil
}

func getDBProxy() *dbProxy {
	if dbMgr == nil {
		dbMgr = new(dbProxy)
	}
	return dbMgr
}

type dbProxy struct {
	conn *engine.TcpClient
}

func (m *dbProxy) init() {
	h := &dbHandler{}
	m.conn = engine.NewTcpClient(engine.WithTcpClientCodec(h), engine.WithTcpClientHandle(h))
	dbConfigName := engine.GetConfig().ServerConfig().DB
	m.conn.Connect(engine.GetConfig().GetServerConfigByName(dbConfigName).Addr, true)
}

func (m *dbProxy) Close() {
	m.conn.Disconnect()
}

func (m *dbProxy) doSaveEntity() {
	needSaveNum := int32(1) //每个tick存盘数量
	if engine.GetConfig().SaveNumPerTick > 0 {
		needSaveNum = engine.GetConfig().SaveNumPerTick
	}
	if sl := engine.GetEntityManager().GetSaveList(); sl != nil {
		saveNum := int32(0)
		for {
			if el := sl.Front(); el != nil {
				info := el.Value.(*engine.EntitySaveInfo)
				if ent := engine.GetEntityManager().GetEntityById(info.EntityId); ent != nil {
					//存在插队的情况,这里需要判断存盘id是否还有效
					if engine.IsValidSaveID(ent.GetEntityId(), info.SaveID) {
						m.saveEntity(info)
						saveNum += 1
					}
				}
				sl.Remove(el)
				if saveNum >= needSaveNum {
					break
				}
			} else {
				break
			}
		}
	}
}

func (m *dbProxy) HandleMainTick() {
	m.doSaveEntity()
	m.conn.Tick()
}

func (m *dbProxy) entityIdFilter(entityId engine.EntityIdType) []byte {
	filter := bson.D{
		bson.E{Key: engine.MongoFieldId, Value: entityId},
	}
	_, filterBytes, _ := bson.MarshalValue(filter)
	return filterBytes
}

func (m *dbProxy) saveEntity(data *engine.EntitySaveInfo) {
	msg := &message.DBCommandRequest{
		TaskType:   uint32(engine.DBTaskTypeReplaceOne),
		EntityId:   int64(data.EntityId),
		Database:   strconv.FormatInt(int64(engine.GetConfig().ServerId), 10),
		Collection: entityCollectionName,
		Filter:     m.entityIdFilter(data.EntityId),
		Data:       data.Data,
		DbType:     uint32(engine.DBTypeProject),
	}

	if data.PreferCallback {
		msg.Ex = &message.ExtraInfo{Uuid: getCallbackMgr().NextUniqueID()}
		getCallbackMgr().setCallbackWithTimeout(msg.Ex.Uuid, &saveEntityOnDestroyCallback{}, 5*time.Second)
	}

	if buf, err := engine.GetProtocol().MessageWithHead([]byte{engine.ServerMessageTypeDBCommand}, msg); err != nil {
		log.Errorf("saveEntity[%d] generate message error: %s", msg.EntityId, err.Error())
		return
	} else {
		if _, err = m.conn.Send(buf); err != nil {
			log.Warnf("saveEntity send message error: %s", err.Error())
		}
	}
}

func (m *dbProxy) loadEntityFromDB(entityId engine.EntityIdType, luaCb lua.LValue, timeout time.Duration) {
	msg := &message.DBCommandRequest{
		TaskType:   uint32(engine.DBTaskTypeQueryOne),
		EntityId:   int64(entityId),
		Database:   strconv.FormatInt(int64(engine.GetConfig().ServerId), 10),
		Collection: entityCollectionName,
		Filter:     m.entityIdFilter(entityId),
		DbType:     uint32(engine.DBTypeProject),
	}

	if luaCb != nil {
		msg.Ex = &message.ExtraInfo{Uuid: getCallbackMgr().NextUniqueID()}
		getCallbackMgr().setCallbackWithTimeout(msg.Ex.Uuid, &queryDBEntityCallback{luaFunc: luaCb}, timeout)
	}

	if buf, err := engine.GetProtocol().MessageWithHead([]byte{engine.ServerMessageTypeDBCommand}, msg); err != nil {
		log.Errorf("loadEntityFromDB generate message error: %s", err.Error())
	} else {
		if _, err = m.conn.Send(buf); err != nil {
			log.Warnf("loadEntityFromDB send message error: %s, entityId: %d", err.Error(), entityId)
		}
	}
}

func (m *dbProxy) executeDBRawCommand(dbType engine.DBType, taskType engine.DBTaskType, database, collection string, filter bson.D, data interface{}, luaCb lua.LValue, timeout time.Duration) {
	_, filterBytes, err := bson.MarshalValue(filter)
	if err != nil {
		log.Warnf("executeDBRawCommand filter marshal error: %s, dbType: %v, taskType: %v, database: %s, collection: %s, filter: %+v", err.Error(), dbType, taskType, database, collection, filter)
		return
	}
	_, dataBytes, err := bson.MarshalValue(data)
	if err != nil {
		log.Warnf("executeDBRawCommand data marshal error: %s, dbType: %v, taskType: %v, database: %s, collection: %s, data: %+v", err.Error(), dbType, taskType, database, collection, data)
		return
	}
	msg := &message.DBCommandRequest{
		TaskType:   uint32(taskType),
		EntityId:   int64(0),
		Database:   database,
		Collection: collection,
		Filter:     filterBytes,
		Data:       dataBytes,
		DbType:     uint32(dbType),
	}
	if luaCb != nil {
		msg.Ex = &message.ExtraInfo{Uuid: getCallbackMgr().NextUniqueID()}
		getCallbackMgr().setCallbackWithTimeout(msg.Ex.Uuid, &dbRawCommandCallback{luaFunc: luaCb}, timeout)
	}

	if buf, err := engine.GetProtocol().MessageWithHead([]byte{engine.ServerMessageTypeDBCommand}, msg); err != nil {
		log.Errorf("executeDBRawCommand generate message error: %s", err.Error())
	} else {
		if _, err = m.conn.Send(buf); err != nil {
			log.Warnf("executeDBRawCommand send message error: %s", err.Error())
		}
	}
}
