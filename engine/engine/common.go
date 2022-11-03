package engine

import (
	"crypto/md5"
	"errors"
	"fmt"
	gJson "github.com/layeh/gopher-json"
	"github.com/sirupsen/logrus"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

var (
	luaL        *lua.LState        //lua虚拟机
	luaCmdMgr   *luaCommandMgr     //其他协程待执行的lua命令
	gSvrType    ServerType         //进程类型
	cfg         *config            //配置文件
	dataTypeMgr *dataTypes         //数据类型管理
	defMgr      *entityDefs        //def文件管理
	entityMgr   *entityManager     //entity管理
	rbMgr       *robotManager      //机器人管理
	idMgr       *entityIdGenerator //entityId生成器
	log         *logrus.Entry      //日志Entry
	netMgr      *network           //网络
	protoMgr    *protocol          //协议
	timer       *timerMgr          //定时器
	etcdMgr     *etcd              //服务注册与发现
	redisMgr    *redisManager      //redis
	cmdLineMgr  *commandLine       //命令行
	timeOffset  int32              //时间偏移
	svrStep     *ServerStep        //服务器状态
)

//JsonToTable 不支持数组、字典混合格式
func JsonToTable(v string) (*lua.LTable, error) {
	if len(v) == 0 {
		return luaL.NewTable(), nil
	}
	r, err := gJson.Decode(luaL, []byte(v))
	if err != nil {
		log.Errorf("convert [%s] to table failed, error: %s", v, err.Error())
		return luaL.NewTable(), err
	}
	if r.Type() != lua.LTTable {
		log.Warnf("cannot convert[%s] to table, type is [%s]", v, r.Type())
		return luaL.NewTable(), nil
	}
	return r.(*lua.LTable), nil
}

//TableToJson 不支持数组、字典混合格式
func TableToJson(v *lua.LTable) (string, error) {
	if v == nil {
		return "", nil
	}
	r, err := gJson.Encode(v)
	return string(r[:]), err
}

//numberToMapKey 将数字类型的key转换为带标记的字符串
func numberToMapKey(key lua.LNumber) string {
	return LuaTableNumberKeyPrefix + key.String()
}

//mapKeyToNumber 将带标记的数字字符串还原回数字
func mapKeyToNumber(key string) (lua.LNumber, error) {
	if strings.HasPrefix(key, LuaTableNumberKeyPrefix) {
		if v, err := strconv.ParseFloat(key[len(LuaTableNumberKeyPrefix):], 64); err != nil {
			return lua.LNumber(0), err
		} else {
			return lua.LNumber(v), nil
		}
	} else {
		return lua.LNumber(0), errors.New("not number string")
	}
}

//TableToMap lua的table类型转换为golang map类型
func TableToMap(t *lua.LTable) map[string]interface{} {
	r := make(map[string]interface{})
	k, v := t.Next(lua.LNil)
	for k != lua.LNil {
		key := ""
		switch k.Type() {
		case lua.LTNumber:
			key = numberToMapKey(k.(lua.LNumber))
		case lua.LTString:
			key = k.String()
		default:
			continue
		}
		switch v.Type() {
		case lua.LTTable:
			r[key] = TableToMap(v.(*lua.LTable))
		case lua.LTNumber:
			fallthrough
		case lua.LTBool:
			fallthrough
		case lua.LTString:
			r[key] = v
		}
		k, v = t.Next(k)
	}
	return r
}

//TableToArray lua的table类型转换为golang数组类型
func TableToArray(t *lua.LTable) []interface{} {
	r := make([]interface{}, 0)
	key, value := t.Next(lua.LNil)
	expectedKey := lua.LNumber(1)
	for key != lua.LNil {
		if key.Type() != lua.LTNumber {
			return r
		}
		if expectedKey != key {
			return r
		}
		r = append(r, value)
		expectedKey++
		key, value = t.Next(key)
	}
	return r
}

func mapToTableImpl(m map[string]interface{}) *lua.LTable {
	t := luaL.NewTable()
	var name lua.LValue
	for n, val := range m {
		if n == MongoPrimaryId {
			continue
		}
		if num, err := mapKeyToNumber(n); err == nil {
			name = num
		} else {
			name = lua.LString(n)
		}
		switch value := val.(type) {
		case int8:
			luaL.RawSet(t, name, lua.LNumber(value))
		case int16:
			luaL.RawSet(t, name, lua.LNumber(value))
		case int32:
			luaL.RawSet(t, name, lua.LNumber(value))
		case int64:
			luaL.RawSet(t, name, lua.LNumber(value))
		case int:
			luaL.RawSet(t, name, lua.LNumber(value))
		case uint8:
			luaL.RawSet(t, name, lua.LNumber(value))
		case uint16:
			luaL.RawSet(t, name, lua.LNumber(value))
		case uint32:
			luaL.RawSet(t, name, lua.LNumber(value))
		case uint64:
			luaL.RawSet(t, name, lua.LNumber(value))
		case uint:
			luaL.RawSet(t, name, lua.LNumber(value))
		case float32:
			luaL.RawSet(t, name, lua.LNumber(value))
		case float64:
			luaL.RawSet(t, name, lua.LNumber(value))
		case bool:
			luaL.RawSet(t, name, lua.LBool(value))
		case string:
			luaL.RawSet(t, name, lua.LString(value))
		case map[string]interface{}:
			luaL.RawSet(t, name, MapToTable(value))
		default:
			log.Warnf("map to lua table not support type: %s for %s", reflect.TypeOf(value).String(), name)
		}
	}
	return t
}

//MapToTable golang字典类型转换为lua table类型
func MapToTable(m map[string]interface{}) *lua.LTable {
	if m[mailboxFieldType] != nil {
		return mapToMailBoxTable(m)
	} else {
		return mapToTableImpl(m)
	}
}

func ArrayMapToTable(arr []map[string]interface{}) *lua.LTable {
	r := luaL.NewTable()
	for i, item := range arr {
		r.RawSetInt(i, MapToTable(item))
	}
	return r
}

func getLuaEntryValue(v string) lua.LValue {
	rpg := luaL.GetGlobal(globalEntry)
	field := luaL.GetField(rpg, v)
	return field
}

func setLuaEntryValue(key string, value lua.LValue) {
	rpg := luaL.GetGlobal(globalEntry)
	luaL.SetField(rpg, key, value)
}

func GetGlobalEntry() lua.LValue {
	return luaL.GetGlobal(globalEntry)
}

func GetLuaTraceback() string {
	defer func(top int) { luaL.SetTop(top) }(luaL.GetTop())

	traceback := luaL.GetGlobal("__G__TRACEBACK__")
	if _, ok := traceback.(*lua.LFunction); ok == false {
		debug := luaL.GetGlobal("debug")
		traceback = luaL.GetField(debug, "traceback")
	}
	if err := luaL.CallByParam(lua.P{Fn: traceback, NRet: 1, Protect: true}, luaL); err != nil {
		return err.Error()
	}
	r := luaL.CheckString(-1)
	return r
}

func funcFailedHandler(_ *lua.LState) int {
	log.Error(GetLuaTraceback())
	return 0
}

func luaFunctionWrapper(f lua.LValue, nRet int) lua.P {
	return lua.P{
		Fn:      f,
		NRet:    nRet,
		Protect: true,
		Handler: luaL.NewFunction(funcFailedHandler),
	}
}

func EntityIdToLua(entityId EntityIdType) lua.LNumber {
	return lua.LNumber(entityId)
	//idStr := strconv.FormatInt(int64(entityId), 10)
	//return lua.LString(idStr)
}

func entityIdFromLua(t *lua.LTable, fieldName string) EntityIdType {
	idStr := luaL.GetField(t, fieldName)
	id, _ := strconv.ParseInt(idStr.String(), 10, 64)
	return EntityIdType(id)
}

func entityIdToString(id EntityIdType) string {
	return strconv.FormatInt(int64(id), 10)
}

func newClientFunction(name string, owner EntityIdType) *lua.LTable {
	t := luaL.NewTable()
	meta := luaL.NewTable()
	luaL.SetField(meta, "name", lua.LString(name))
	luaL.SetField(meta, "owner", EntityIdToLua(owner))
	luaL.SetField(meta, "__index", meta)
	luaL.SetField(meta, "__call", luaL.NewFunction(func(L *lua.LState) int {
		clientTable := L.CheckTable(1)
		methodName := L.GetField(clientTable, "name").String()
		id := entityIdFromLua(clientTable, "owner")
		ent := GetEntityManager().GetEntityById(id)
		if ent == nil {
			log.Debugf("call client entity[%d] method[%s] but entity is nil", id, methodName)
			return 0
		}
		if ent.client == nil {
			log.Debugf("call client entity[%s] method[%s] but client conn is nil", ent.String(), methodName)
			return 0
		}
		method := ent.def.getClientMethod(methodName, true)
		if method == nil {
			log.Warnf("%s call client method[%s] but not found", ent.String(), methodName)
			return 0
		}
		needArgsNum := len(method.args)
		//lua栈上第一个参数是self.client
		if L.GetTop() < needArgsNum+1 {
			log.Errorf("call client method[%s] need %d arg(s) but got %d%s", methodName, needArgsNum, L.GetTop()-1, GetLuaTraceback())
			return 0
		} else {
			args := []interface{}{ent.def.getClientMethodMaskName(name)}
			passed := true
			//check params
			for i, argPropType := range method.args {
				arg := L.CheckAny(i + 2)
				if argPropType.dt.IsSameType(arg) == false {
					log.Errorf("call client method[%s], arg[%d] need[%s] but got[%s(%s)]%s", methodName, i+1, argPropType.dt.Type(), arg.String(), arg.Type(), GetLuaTraceback())
					passed = false
					break
				} else {
					args = append(args, argPropType.dt.ParseFromLua(arg))
				}
			}
			if passed {
				buf := map[string]interface{}{
					ClientMsgDataFieldType:     ClientMsgTypeEntityRpc,
					ClientMsgDataFieldEntityID: ent.entityId,
					ClientMsgDataFieldArgs:     args,
				}
				if GetConfig().PrintRpcLog {
					log.WithField("type", "RPC").Debugf("%s call client method: %s, args: %+v", ent.String(), name, args[1:])
				}
				if data, err := genEntityRpcMessage(uint8(ServerMessageTypeEntityRpc), buf, ent.client.mailbox.ClientId); err == nil {
					ent.client.mailbox.Send(data)
				} else {
					log.Errorf("call %s client method[%s] error: %s", ent.String(), name, err.Error())
				}
			}
		}
		return 0
	}))
	luaL.SetMetatable(t, meta)
	return t
}

func isEntityReserveProp(name string) bool {
	return name == entityFieldId || name == entityFieldName || name == entityFieldType
}

var serviceName string

func ServiceName() string {
	if serviceName == "" {
		serverIdStr := strconv.FormatInt(int64(GetConfig().ServerId), 10)
		tagStr := strconv.FormatInt(int64(GetCmdLine().Tag), 10)
		switch gSvrType {
		case STGate:
			serviceName = ServiceGatePrefix + serverIdStr + "." + tagStr
		case STGame:
			serviceName = ServiceGamePrefix + serverIdStr + "." + tagStr
		case STDbMgr:
			serviceName = ServiceDBPrefix + serverIdStr + "." + tagStr
		case STRobot:
			serviceName = ServiceClientPrefix + serverIdStr + "." + tagStr
		case STAdmin:
			serviceName = ServiceAdminPrefix + serverIdStr + "." + tagStr
		default:
			serviceName = "undefined.service.name"
		}
	}
	return serviceName
}

func InterfaceToLValue(item interface{}) lua.LValue {
	switch data := item.(type) {
	case int8:
		return lua.LNumber(data)
	case int16:
		return lua.LNumber(data)
	case int32:
		return lua.LNumber(data)
	case int64:
		return lua.LNumber(data)
	case int:
		return lua.LNumber(data)
	case uint8:
		return lua.LNumber(data)
	case uint16:
		return lua.LNumber(data)
	case uint32:
		return lua.LNumber(data)
	case uint64:
		return lua.LNumber(data)
	case uint:
		return lua.LNumber(data)
	case float32:
		return lua.LNumber(data)
	case float64:
		return lua.LNumber(data)
	case string:
		if data == LuaTableValueNilField {
			return lua.LNil
		} else {
			return lua.LString(data)
		}
	case bool:
		return lua.LBool(data)
	case map[string]interface{}:
		return MapToTable(data)
	case []interface{}:
		t := luaL.NewTable()
		for k, v := range data {
			t.RawSetInt(k+1, InterfaceToLValue(v))
		}
		return t
	case nil:
		return lua.LNil
	case MailBox:
		return MailBoxToTable(data)
	default:
		log.Warnf("InterfaceToLvalues not handler type[%s]", reflect.TypeOf(item).String())
		return lua.LNil
	}
}

func InterfaceToLValues(arr []interface{}) []lua.LValue {
	r := make([]lua.LValue, len(arr), len(arr))
	for i, item := range arr {
		r[i] = InterfaceToLValue(item)
	}
	return r
}

func InterfaceToInt(v interface{}) int64 {
	switch data := v.(type) {
	case int8:
		return int64(data)
	case int16:
		return int64(data)
	case int32:
		return int64(data)
	case int64:
		return data
	case int:
		return int64(data)
	case uint8:
		return int64(data)
	case uint16:
		return int64(data)
	case uint32:
		return int64(data)
	case uint64:
		return int64(data)
	case uint:
		return int64(data)
	case float32:
		return int64(data)
	case float64:
		return int64(data)
	case string:
		if r, err := strconv.ParseInt(data, 10, 64); err == nil {
			return r
		}
	case nil:
		return 0
	default:
		log.Warnf("InterfaceToInt not support type[%s]", reflect.TypeOf(v).String())
		return 0
	}
	return 0
}

func Md5(s string) string {
	r := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", r)
}

func dumpTable(t *lua.LTable) {
	ck, cv := t.Next(lua.LNil)
	for ck != lua.LNil {
		log.Debugf("%v => %v", ck, cv)
		ck, cv = t.Next(ck)
	}
}

func newSyncTable(name string) *lua.LTable {
	t := luaL.NewTable()
	t.RawSetString(SyncTableFieldProps, luaL.NewTable())
	t.RawSetString(SyncTableFieldName, lua.LString(name))
	meta := luaL.NewTable()
	meta.RawSetString("__index", luaL.NewFunction(func(L *lua.LState) int {
		syncTable := L.CheckTable(1)
		key := L.CheckAny(2)
		if propTable, ok := syncTable.RawGetString(SyncTableFieldProps).(*lua.LTable); ok {
			val := propTable.RawGet(key)
			L.Push(val)
		} else {
			L.Push(lua.LNil)
		}
		return 1
	}))
	meta.RawSetString("__newindex", luaL.NewFunction(func(L *lua.LState) int {
		syncTable := L.CheckTable(1)
		key := L.CheckAny(2)
		val := L.CheckAny(3)
		propTable := syncTable.RawGetString(SyncTableFieldProps).(*lua.LTable)
		if old := propTable.RawGet(key); old != val {
			propTable.RawSet(key, val)
			if entityId, ok := syncTable.RawGetString(SyncTableFieldOwner).(lua.LNumber); ok {
				if propName, ok := syncTable.RawGetString(SyncTableFieldName).(lua.LString); ok {
					if ent := GetEntityManager().GetEntityById(EntityIdType(entityId)); ent != nil {
						if prop := ent.def.prop(propName.String()); prop != nil {
							if prop.config.IsSyncProp() {
								ent.onSyncTableUpdated(propName.String(), key, val)
							}
						}
					}
				}
			} else {
				log.Warnf("syncTable field updated but owner field not found")
			}
		}

		return 0
	}))
	luaL.SetMetatable(t, meta)
	return t
}

//LuaArrayToBsonD 将{{"a", 1}, {"b", 2}}格式的lua数组转换为bsonD格式
func LuaArrayToBsonD(t *lua.LTable) (bson.D, error) {
	r := bson.D{}
	key, value := t.Next(lua.LNil)
	expectedKey := lua.LNumber(1)
	for key != lua.LNil {
		if key.Type() != lua.LTNumber {
			return r, errors.New("table not array")
		}
		if expectedKey != key {
			return r, errors.New("table not array")
		}
		if info, ok := value.(*lua.LTable); !ok {
			return r, errors.New("value not table")
		} else {
			k, v := info.RawGetInt(1), info.RawGetInt(2)
			if k.Type() != lua.LTString {
				return r, errors.New("key not string")
			}
			valueType := v.Type()
			if valueType != lua.LTNumber && valueType != lua.LTString && valueType != lua.LTBool {
				return r, errors.New("value only support number, string, bool")
			}
			r = append(r, bson.E{Key: k.String(), Value: v})
		}
		expectedKey++
		key, value = t.Next(key)
	}
	return r, nil
}

// 解析定时器时间字符串
func parseTimerString(t string) (int64, error) {
	t = strings.ToLower(t)
	affix := ""
	prefix := float64(0)
	for i, r := range t {
		if unicode.IsLetter(r) {
			var err error
			if prefix, err = strconv.ParseFloat(t[:i], 64); err != nil {
				return 0, err
			}
			affix = t[i:]
			break
		}
	}
	fmt.Println("string: ", t, ", affix: ", affix, ", prefix: ", prefix)

	ms := int64(0)
	switch affix {
	case "d":
		ms = int64(prefix * 24 * 60 * 60 * 1000)
	case "h":
		ms = int64(prefix * 60 * 60 * 1000)
	case "min":
		ms = int64(prefix * 60 * 1000)
	case "s":
		ms = int64(prefix * 1000)
	case "ms":
		ms = int64(prefix)
	default:
		return ms, fmt.Errorf("parse timer string, not support affix: \"%s\"", affix)
	}

	return ms, nil
}
