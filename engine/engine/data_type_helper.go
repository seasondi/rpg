package engine

import (
	lua "github.com/yuin/gopher-lua"
	"reflect"
	"strconv"
)

const (
	int8Min   = -128
	int8Max   = 127
	int16Min  = -32768
	int16Max  = 32767
	int32Min  = -2147483648
	int32Max  = 2147483647
	int64Min  = -9223372036854775808
	int64Max  = 9223372036854775807
	uint8Min  = 0
	uint8Max  = 255
	uint16Min = 0
	uint16Max = 65535
	uint32Min = 0
	uint32Max = 4294967295
)

func defaultSerialize(m dataType, v lua.LValue) lua.LValue {
	if m.IsSameType(v) == false {
		if v != lua.LNil {
			log.Errorf("ParseFromLua value[%v] not match %s, set to value[%+v]", v, m.Type(), m.Default())
		}
		return m.Default()
	}
	return v
}

func defaultParseNumber(m dataType, v interface{}) lua.LValue {
	tmp := m.Default()
	switch val := v.(type) {
	case int8:
		tmp = lua.LNumber(val)
	case int16:
		tmp = lua.LNumber(val)
	case int32:
		tmp = lua.LNumber(val)
	case int64:
		tmp = lua.LNumber(val)
	case int:
		tmp = lua.LNumber(val)
	case uint8:
		tmp = lua.LNumber(val)
	case uint16:
		tmp = lua.LNumber(val)
	case uint32:
		tmp = lua.LNumber(val)
	case uint64:
		tmp = lua.LNumber(val)
	case uint:
		tmp = lua.LNumber(val)
	case float32:
		tmp = lua.LNumber(val)
	case float64:
		tmp = lua.LNumber(val)
	case string:
		var err error
		if tmp, err = mapKeyToNumber(val); err != nil {
			if i, err := strconv.ParseInt(val, 10, 64); err != nil {
				return m.Default()
			} else {
				tmp = lua.LNumber(i)
			}
		}
	}
	if m.IsSameType(tmp) == false {
		log.Warnf("value[%v] not match type %s, set to default[%+v]", v, m.Type(), m.Default())
		return m.Default()
	}
	return tmp
}

func defaultParseTable(m dataType, v interface{}) lua.LValue {
	switch val := v.(type) {
	case map[string]interface{}:
		r := MapToTable(val)
		if m.IsSameType(r) {
			return r
		}
	}
	log.Warnf("value[%v] type[%+v] not match type %s, set to default[%+v]", v, reflect.TypeOf(v).Name(), m.Type(), m.Default())
	return m.Default()
}

func defaultParseString(m dataType, v interface{}) lua.LValue {
	switch val := v.(type) {
	case string:
		return lua.LString(val)
	case int8:
		return lua.LString(strconv.FormatInt(int64(val), 10))
	case int16:
		return lua.LString(strconv.FormatInt(int64(val), 10))
	case int32:
		return lua.LString(strconv.FormatInt(int64(val), 10))
	case int64:
		return lua.LString(strconv.FormatInt(val, 10))
	case int:
		return lua.LString(strconv.FormatInt(int64(val), 10))
	case uint8:
		return lua.LString(strconv.FormatUint(uint64(val), 10))
	case uint16:
		return lua.LString(strconv.FormatUint(uint64(val), 10))
	case uint32:
		return lua.LString(strconv.FormatUint(uint64(val), 10))
	case uint64:
		return lua.LString(strconv.FormatUint(val, 10))
	case uint:
		return lua.LString(strconv.FormatUint(uint64(val), 10))
	case float32:
		return lua.LString(strconv.FormatFloat(float64(val), 'f', -1, 64))
	case float64:
		return lua.LString(strconv.FormatFloat(val, 'f', -1, 64))
	default:
		log.Warnf("cannot parse value[%+v] type[%+v] to string, set to default[%+v]", v, reflect.TypeOf(v).Name(), m.Default())
		return m.Default()
	}
}

func defaultParseMailBox(m dataType, v interface{}) lua.LValue {
	log.Debugf("v: %+v", v)
	switch val := v.(type) {
	case map[string]interface{}:
		return mapToMailBoxTable(val)
	}
	log.Warnf("value[%v] type[%+v] not match type %s, set to default[%+v]", v, reflect.TypeOf(v).Name(), m.Type(), m.Default())
	return m.Default()
}
