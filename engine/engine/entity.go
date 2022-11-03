package engine

import (
	"rpg/engine/message"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"strconv"
	"time"
)

type EntityStatus int

const (
	EntityCreate      EntityStatus = iota //创建entity
	EntityReady                           //创建entity完成
	EntityWaitDestroy                     //等待销毁(等待销毁定时器触发)
	EntityDestroying                      //entity销毁中
	EntityDestroyed                       //entity销毁完成
)

type EntityClient struct {
	mailbox ClientMailBox
	primary bool
}

func (m *EntityClient) MailBox() ClientMailBox {
	return m.mailbox
}

type entity struct {
	entityId             EntityIdType     //id
	entityName           string           //名称
	luaEntity            *lua.LTable      //脚本层entity
	propsTable           *lua.LTable      //属性表
	clientTable          *lua.LTable      //客户端rpc函数信息
	def                  *entityDef       //def定义
	client               *EntityClient    //客户端连接信息
	destroyTimerId       int64            //延迟销毁定时器
	destroyingStatusTime int64            //进入销毁中状态的时间
	saveTimerId          int64            //自动存盘定时器
	status               EntityStatus     //entity状态
	stubLeaseResult      *etcdLeaseResult //stub在etcd的租约
	lastHeartBeatTime    int64            //上次心跳时间
	heartbeatTimerId     int64            //心跳定时器
	activeTimerIds       map[int64]bool   //已添加的定时器id
}

func NewEntity(entityId EntityIdType, entityName string) (*entity, error) {
	e := new(entity)
	e.entityId = entityId
	e.entityName = entityName
	if err := e.init(); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *entity) init() error {
	scriptPath := getLuaEntryValue("scriptPath")
	if scriptPath.Type() == lua.LTNil {
		log.Warn(globalEntry + ".scriptPath is necessary, please set in script(relative path to \"WorkPath\" defined in config)")
		return errors.New("scriptPath not defined")
	}
	e.activeTimerIds = make(map[int64]bool)
	e.luaEntity = luaL.NewTable()
	luaL.SetMetatable(e.luaEntity, GetEntityManager().genMetaTable(e.entityName))
	e.propsTable = luaL.NewTable()
	e.propsTable.RawSetString(entityFieldId, EntityIdToLua(e.entityId))
	e.def = defMgr.GetEntityDef(e.entityName)
	if e.def == nil {
		return fmt.Errorf("cannot find entity[%s] def, please check entities.xml", e.entityName)
	}
	if err := e.def.loadInterfaceFiles(); err != nil {
		return err
	}
	if err := luaL.DoFile(cfg.WorkPath + "/" + scriptPath.String() + "/" + e.entityName + ".lua"); err != nil {
		return err
	}
	GetEntityManager().registerEntity(e)
	e.def.registerToEntity(e)
	registerApiToEntity(e.luaEntity)

	e.status = EntityCreate
	return nil
}

func (e *entity) addEntityTimer(d time.Duration, repeat time.Duration, cb func(...interface{}), params ...interface{}) int64 {
	timerId := GetTimer().AddTimer(d, repeat, cb, params...)
	e.activeTimerIds[timerId] = true
	return timerId
}

func (e *entity) cancelEntityTimer(timerId int64) {
	GetTimer().Cancel(timerId)
	e.removeActiveTimerId(timerId)
}

func (e *entity) removeActiveTimerId(timerId int64) {
	delete(e.activeTimerIds, timerId)
}

func (e *entity) cancelAllTimers() {
	for timerId := range e.activeTimerIds {
		GetTimer().Cancel(timerId)
	}
	e.activeTimerIds = make(map[int64]bool)

	e.saveTimerId = 0
	e.destroyTimerId = 0
}

func (e *entity) completeEntity() error {
	if err := CallLuaMethodByName(e.luaEntity, onEntityInit, 0, e.luaEntity); err != nil {
		return err
	}

	if err := e.registerSelf(); err != nil {
		e.removeRegisterInfo()
		return err
	}
	e.status = EntityReady
	initEntitySaveID(e.entityId)
	return nil
}

func (e *entity) registerSelf() error {

	val := EtcdValue{
		EtcdValueType:     EtcdTypeEntity,
		EtcdValueServer:   ServiceName(),
		EtcdValueName:     e.entityName,
		EtcdValueEntityId: e.entityId,
	}
	if e.def.volatile.isStub {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

		val[EtcdValueType] = EtcdTypeStub
		if entryEntityName == e.def.entityName {
			val[EtcdStubValueEntry] = e.entityName
		}
		if r, err := GetEtcd().Register(ctx, EtcdStubLeaseTTL, NewEtcdKV(GetEtcdStubKey(e.entityId), val)); err != nil {
			log.Errorf("register stub to etcd error: %s", err.Error())
			return err
		} else {
			e.stubLeaseResult = r
		}
	}

	//注册到redis
	{
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()
		info, _ := json.Marshal(val)
		if err := GetRedisMgr().Set(ctx, GetEtcdEntityKey(e.entityId), info, 0); err != nil {
			log.Errorf("put %s to redis error: %s", e.String(), err.Error())
			return err
		}
	}
	return nil
}

func (e *entity) removeRegisterInfo() {
	//移除entity信息
	{
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()
		if err := GetRedisMgr().Del(ctx, GetEtcdEntityKey(e.entityId)); err != nil {
			log.Errorf("remove %s from redis error: %s", e.String(), err.Error())
		}
	}
	//移除stub信息
	if e.def.volatile.isStub {
		if e.stubLeaseResult != nil {
			e.stubLeaseResult.Close()
			e.stubLeaseResult = nil
		}
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()
		if err := GetEtcd().Delete(ctx, GetEtcdStubKey(e.entityId)); err != nil {
			log.Errorf("remove %s stub info from etcd error: %s", e.String(), err.Error())
		}
	}
}

func (e *entity) final() {
	GetEntityManager().unRegisterEntity(e)
	e.removeRegisterInfo()
	e.status = EntityDestroyed
	clearEntitySaveID(e.entityId)
}

func (e *entity) Status() EntityStatus {
	return e.status
}

func (e *entity) Destroy(isSaveDB bool, destroyImmediately bool) {
	log.Infof("%s destroy. status: %d, isSaveDB: %v, immediately: %v", e.String(), e.status, isSaveDB, destroyImmediately)
	if e.status == EntityDestroyed || e.status == EntityWaitDestroy {
		return
	} else if e.status == EntityDestroying {
		//10秒销毁超时时间,数据库异常可能造成存盘失败,entity一直处于EntityDestroying状态,这里允许重入destroy
		if time.Now().Unix() < e.destroyingStatusTime+10 {
			return
		}
	}
	//初始化失败了, 直接销毁掉
	if e.status < EntityReady {
		e.final()
		return
	}
	if e.status == EntityReady {
		_ = CallLuaMethodByName(e.luaEntity, onEntityDestroy, 0, e.luaEntity)
		e.cancelAllTimers()
	}

	var mb *ClientMailBox
	if e.client != nil {
		mb = &e.client.mailbox
		_ = e.setClient(nil, e.client.primary)
	}
	e.status = EntityWaitDestroy

	if destroyImmediately {
		if e.destroyTimerId > 0 {
			e.cancelEntityTimer(e.destroyTimerId)
			e.destroyTimerId = 0
		}
	}

	if e.destroyTimerId > 0 {
		return
	}

	e.status = EntityDestroying
	e.destroyingStatusTime = time.Now().Unix()
	if isSaveDB && e.def.volatile.persistent {
		GetEntityManager().saveEntityOnDestroy(e)
	} else {
		e.final()
	}

	//销毁该连接关联的所有其他entity
	if mb != nil {
		for _, entityId := range GetEntityManager().GetEntitiesByConn(mb) {
			if entityId != e.entityId {
				if ent := GetEntityManager().GetEntityById(entityId); ent != nil {
					ent.Destroy(isSaveDB, destroyImmediately)
				}
			}
		}
	}
}

func (e *entity) SavedOnDestroyCallback() {
	_ = CallLuaMethodByName(e.luaEntity, onEntityFinal, 0, e.luaEntity)
	e.final()
}

func (e *entity) GetEntityId() EntityIdType {
	return e.entityId
}

//onSyncPropChanged 需要同步给客户端的属性变化
func (e *entity) onSyncPropChanged(propName string, newVal lua.LValue, dt dataType) {
	if e.client == nil {
		return
	}
	propInfo := e.def.prop(propName)
	if propInfo == nil {
		return
	}
	if !propInfo.config.IsSyncProp() {
		return
	}
	buf := map[string]interface{}{
		ClientMsgDataFieldType:     ClientMsgTypePropSync,
		ClientMsgDataFieldEntityID: e.entityId,
		ClientMsgDataFieldArgs:     []interface{}{propName, dt.ParseFromLua(newVal)},
	}
	if data, err := genEntityRpcMessage(uint8(ServerMessageTypeEntityRpc), buf, e.client.mailbox.ClientId); err == nil {
		e.client.mailbox.Send(data)
		log.Tracef("%s prop[%s] changed, new: %v", e.String(), propName, newVal)
	} else {
		log.Errorf("%s sync prop[%s] error: %s", e.String(), propName, err.Error())
	}
}

//onSyncTableUpdated SyncOnUpdate标记的table变化
func (e *entity) onSyncTableUpdated(propName string, key lua.LValue, val lua.LValue) {
	if e.client == nil {
		return
	}
	buf := map[string]interface{}{
		ClientMsgDataFieldType:     ClientMsgTypePropSyncUpdate,
		ClientMsgDataFieldEntityID: e.entityId,
	}
	switch val.Type() {
	case lua.LTNil:
		buf[ClientMsgDataFieldArgs] = []interface{}{propName, key, LuaTableValueNilField}
	case lua.LTTable:
		buf[ClientMsgDataFieldArgs] = []interface{}{propName, key, TableToMap(val.(*lua.LTable))}
	default:
		buf[ClientMsgDataFieldArgs] = []interface{}{propName, key, val}
	}
	if data, err := genEntityRpcMessage(uint8(ServerMessageTypeEntityRpc), buf, e.client.mailbox.ClientId); err == nil {
		e.client.mailbox.Send(data)
		log.Tracef("%s prop[%s] key[%s] changed to [%v]", e.String(), propName, key, val)
	} else {
		log.Errorf("%s sync prop[%s] error: %s", e.String(), propName, err.Error())
	}
}

func (e *entity) String() string {
	return "entity[" + e.entityName + ":" + strconv.FormatInt(int64(e.entityId), 10) + "]"
}

func (e *entity) genSaveInfo() *EntitySaveInfo {
	if e.def.volatile.persistent == false {
		return nil
	}
	r := make(map[string]interface{})
	for name, prop := range e.def.properties {
		if !prop.config.Persistent {
			continue
		}
		v := luaL.GetField(e.luaEntity, name)
		if v != lua.LNil {
			r[name] = prop.dt.ParseFromLua(v)
		} else {
			r[name] = prop.dt.Default()
		}
	}
	r[MongoFieldId] = e.entityId
	r[MongoFieldName] = e.entityName
	_, data, err := bson.MarshalValue(r)
	if err != nil {
		log.Warnf("%s genSaveInfo marshal error: %s", e.String(), err.Error())
		return nil
	}

	if saveId, err := nextSaveID(e.entityId); err == nil {
		return &EntitySaveInfo{EntityId: e.entityId, Data: data, SaveID: saveId}
	} else {
		return nil
	}
}

func (e *entity) loadData(data map[string]interface{}) {
	for name, value := range data {
		if isEntityReserveProp(name) || name == MongoPrimaryId {
			continue
		}
		propInfo := e.def.prop(name)
		if propInfo == nil {
			log.Debugf("%s def has no prop[%s]", e.String(), name)
			continue
		}
		val := propInfo.dt.ParseToLua(value)
		if propInfo.dt.Name() == dataTypeNameSyncTable {
			val.(*lua.LTable).RawSetString(SyncTableFieldOwner, EntityIdToLua(e.entityId))
		}
		e.propsTable.RawSetString(name, val)
	}
}

func (e *entity) SaveToDB(saveType int) {
	if e.def.volatile.persistent == true {
		GetEntityManager().saveEntity(e, saveType)
	}
}

func (e *entity) saveTimerCb(...interface{}) {
	e.saveTimerId = 0
	e.SaveToDB(saveTypeBack)
}

func (e *entity) destroyTimerCb(...interface{}) {
	e.destroyTimerId = 0
	e.Destroy(true, true)
}

//CheckDefServerMethod 检查通过返回函数原始名称, 否则返回传入的名称
func (e *entity) CheckDefServerMethod(method string, args []lua.LValue, fromClient bool) (string, error) {
	mi := e.def.getServerMethod(method, !fromClient)
	if mi == nil {
		return method, fmt.Errorf("[%s] is not def server method for %s", method, e.String())
	} else {
		if fromClient && mi.exposed == false {
			return method, fmt.Errorf("client cannot call method[%s] for %s, add <Exposed/> after method in def file", method, e.String())
		}
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

//CallDefServerMethod fromClient为false时method传入原始函数名, 否则传入mask name
func (e *entity) CallDefServerMethod(method string, args []lua.LValue, fromClient bool) error {
	name, err := e.CheckDefServerMethod(method, args, fromClient)
	if err != nil {
		return err
	}

	if GetConfig().PrintRpcLog {
		log.WithField("type", "RPC").Debugf("call %s server method: %s, args: %+v, is from client: %+v", e.String(), name, args, fromClient)
	}
	params := append([]lua.LValue{e.luaEntity}, args...)
	if err = CallLuaMethodByName(e.luaEntity, name, 0, params...); err != nil {
		return err
	}
	return nil
}

func (e *entity) GetClient() *EntityClient {
	return e.client
}

/*setClient将客户端连接绑定到entity,当c是nil时,primary无意义

c: 客户端连接信息

primary: entity是否是该连接的主entity, 只有主entity连接信息变化时才触发断开客户端或者通知顶号等
(一个连接可绑定给多个entity,这些entity之间应当属于同个玩家,如account+avatar,只能有一个entity为主,其他均为副,需业务层保证,引擎层不做检查)
*/
func (e *entity) setClient(c *ClientMailBox, primary bool) error {
	if e.status != EntityReady {
		return fmt.Errorf("cannot set entity client info, status %d", e.status)
	}
	if e.def.volatile.hasClient == false {
		return fmt.Errorf("entity cannot has client")
	}
	if e.client != nil {
		//连接不变,只是修改主从
		if e.client.mailbox.Equal(c) {
			e.client.primary = primary
			return nil
		}
		//通知gate解绑entity与客户端连接
		{
			header := GenMessageHeader(ServerMessageTypeChangeEntityClient, 0)
			body := message.ClientBindEntity{EntityId: int64(e.entityId), ClientId: uint32(e.client.mailbox.ClientId), Unbind: true}
			if data, err := GetProtocol().MessageWithHead(header, &body); err == nil {
				e.client.mailbox.Send(data)
			}
		}
		if c != nil {
			//连接被替换, 通知前个连接被顶号
			if e.client.primary {
				if e.client.mailbox.GateName != c.GateName || e.client.mailbox.ClientId != c.ClientId {
					header := GenMessageHeader(ServerMessageTypeLoginByOther, e.client.mailbox.ClientId)
					if data, err := GetProtocol().MessageWithHead(header, nil); err == nil {
						e.client.mailbox.Send(data)
					}
				}
			}
		} else {
			//断开客户端连接
			if e.client.primary {
				header := GenMessageHeader(ServerMessageTypeDisconnectClient, e.client.mailbox.ClientId)
				if data, err := GetProtocol().MessageWithHead(header, nil); err == nil {
					e.client.mailbox.Send(data)
				}
			}
			e.destroyTimerId = e.addEntityTimer(time.Duration(cfg.SaveInterval)*time.Minute, 0, e.destroyTimerCb)
			log.Infof("add destroy timer for %s, timer id: %d", e.String(), e.destroyTimerId)
		}
	}
	if c == nil {
		e.onLoseClient()
		e.client = nil
		e.delCheckHeartBeatTimer()
	} else {
		e.client = &EntityClient{mailbox: *c, primary: primary}
		if e.client.primary {
			e.addCheckHeartbeatTimer()
		}
	}
	if e.client != nil {
		if e.destroyTimerId > 0 {
			e.cancelEntityTimer(e.destroyTimerId)
			e.destroyTimerId = 0
		}
		//通知gate客户端连接绑定到了entity
		header := GenMessageHeader(ServerMessageTypeChangeEntityClient, 0)
		body := message.ClientBindEntity{EntityId: int64(e.entityId), ClientId: uint32(e.client.mailbox.ClientId)}
		if data, err := GetProtocol().MessageWithHead(header, &body); err == nil {
			e.client.mailbox.Send(data)
		}
		e.onGetClient()
		if e.def.volatile.persistent == true {
			if e.saveTimerId == 0 {
				interval := time.Duration(cfg.SaveInterval) * time.Minute
				e.saveTimerId = e.addEntityTimer(interval, interval, e.saveTimerCb)
				log.Infof("add entity save timer for %s, id: %d", e.String(), e.saveTimerId)
			}
		}
	}
	log.Infof("%s conn info set to %+v", e.String(), e.client)
	return nil
}

func (e *entity) onGetClient() {
	luaL.SetField(e.luaEntity, "client", e.clientTable)
	if err := e.createClientEntity(); err != nil {
		log.Errorf("create client entity error: %s", err.Error())
	} else {
		_ = CallLuaMethodByName(e.luaEntity, onEntityGetClient, 0, e.luaEntity)
	}
}

func (e *entity) onLoseClient() {
	luaL.SetField(e.luaEntity, "client", lua.LNil)
	_ = CallLuaMethodByName(e.luaEntity, onEntityLostClient, 0, e.luaEntity)
}

func (e *entity) createClientEntity() error {
	if e.client == nil {
		return fmt.Errorf("%s createClientEntity but client nil", e.String())
	}
	props := make(map[string]interface{})
	for name, prop := range e.def.properties {
		if prop.config.IsSyncProp() {
			val := luaL.GetField(e.propsTable, name)
			props[name] = prop.dt.ParseFromLua(val)
		}
	}
	args := []interface{}{e.entityName, props}
	msg := map[string]interface{}{
		ClientMsgDataFieldType:     ClientMsgTypeCreateEntity,
		ClientMsgDataFieldEntityID: e.entityId,
		ClientMsgDataFieldArgs:     args,
	}
	if data, err := genEntityRpcMessage(uint8(ServerMessageTypeEntityRpc), msg, e.client.mailbox.ClientId); err == nil {
		e.client.mailbox.Send(data)
		return nil
	} else {
		return err
	}
}

func (e *entity) checkHeartbeatCb(...interface{}) {
	now := time.Now().Unix()
	if e.lastHeartBeatTime == 0 {
		e.lastHeartBeatTime = now
	}
	if now-e.lastHeartBeatTime > 2*heartbeatTick {
		log.Warnf("%s heartbeat check timeout", e.String())
		_ = e.setClient(nil, false)
	}
}

func (e *entity) addCheckHeartbeatTimer() {
	e.delCheckHeartBeatTimer()
	e.heartbeatTimerId = GetTimer().AddTimer(time.Second, time.Second, e.checkHeartbeatCb)
}

func (e *entity) delCheckHeartBeatTimer() {
	if e.heartbeatTimerId > 0 {
		GetTimer().Cancel(e.heartbeatTimerId)
		e.heartbeatTimerId = 0
	}
}
