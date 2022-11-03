package engine

import (
	"context"
	"encoding/json"
	lua "github.com/yuin/gopher-lua"
	"time"
)

const (
	redisSingleKeyPrefix = "__"
)

var redisExports = map[string]lua.LGFunction{
	"get": luaRedisGet,
	"set": luaRedisSet,
}

func preloadRedis() {
	luaL.PreloadModule("redis", redisLoader)
}

func redisLoader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), redisExports)
	L.Push(mod)
	return 1
}

func luaRedisGet(L *lua.LState) int {
	if GetRedisMgr() == nil {
		L.Push(lua.LNil)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	key := L.CheckString(1)

	b, err := GetRedisMgr().GetBytes(ctx, key)
	if err != nil {
		L.Push(lua.LNil)
		return 1
	}

	r := make(map[string]interface{})
	if err = json.Unmarshal(b, &r); err != nil {
		arr := make([]interface{}, 0)
		if err = json.Unmarshal(b, &arr); err != nil {
			L.Push(lua.LNil)
			return 1
		} else {
			t := luaL.NewTable()
			for k, v := range InterfaceToLValues(arr) {
				t.RawSetInt(k+1, v)
			}
			L.Push(t)
			return 1
		}
	}

	//redis set单个元素会被调整为字典写入,这里还原
	if len(r) == 1 {
		for k, v := range r {
			//只有符合前缀
			if k == redisSingleKeyPrefix {
				L.Push(InterfaceToLValue(v))
				return 1
			}
		}
	}

	L.Push(MapToTable(r))
	return 1
}

func luaRedisSet(L *lua.LState) int {
	key := L.CheckString(1)
	value := L.CheckAny(2)
	second := lua.LNumber(0)
	if L.GetTop() >= 3 {
		second = L.CheckNumber(3)
	}

	duration := time.Duration(int64(second)) * time.Second

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	var data *lua.LTable
	switch value.Type() {
	case lua.LTTable:
		data = value.(*lua.LTable)
	case lua.LTBool:
	case lua.LTNumber:
		fallthrough
	case lua.LTString:
		data = luaL.NewTable()
		data.RawSetString(redisSingleKeyPrefix, value)
	case lua.LTNil:
		if err := GetRedisMgr().Del(ctx, key); err != nil {
			log.Warnf("del %s from redis error: %s", key, err.Error())
		}
		return 0
	default:
		log.Errorf("lua redis set unsupport type: %s, key: %s", value.Type().String(), key)
		return 0
	}

	if info, err := TableToJson(data); err == nil {
		if err = GetRedisMgr().Set(ctx, key, info, duration); err != nil {
			log.Warnf("lua redis set key: %s, value: %v, error: %s", key, value, err.Error())
		}
	} else {
		log.Warnf("lua redis set, key: %s, value: %v, to json error: %s", key, value, err.Error())
	}

	return 0
}
