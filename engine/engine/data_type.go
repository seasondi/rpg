package engine

import (
	"errors"
	"fmt"
	lua "github.com/seasondi/gopher-lua"
	"math"
	"reflect"
	"strconv"
)

type dataType interface {
	Name() string                               //名称
	Type() string                               //类型
	Detail() *dataTypeDetail                    //详细信息
	IsSameType(lua.LValue) bool                 //是否是相同类型
	Default() lua.LValue                        //获取默认值
	SetDefault(string) error                    //设置默认值
	ParseDefaultVal(string) (lua.LValue, error) //解析默认值
	ParseFromLua(lua.LValue) interface{}        //将lua类型解析为golang类型
	ParseRawFromLua(lua.LValue) interface{}     //将lua类型解析为golang类型,但是不对数字key做特殊处理,适用于发给客户端
	ParseToLua(interface{}) lua.LValue          //将golang类型解析为lua类型
}

type dataTypeDetail struct {
	defaultVal lua.LValue //默认值(仅描述属性时有意义)
	name       string     //属性名称(如果dataType描述的是函数参数,则是函数名)
}

// --------------------------------------------------------------------
type dtInt8 struct {
	detail dataTypeDetail
}

func (m *dtInt8) Name() string {
	return dataTypeNameInt8
}

func (m *dtInt8) Type() string {
	return m.Name()
}

func (m *dtInt8) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtInt8) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtInt8) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtInt8) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= int8Max && value >= int8Min {
			return true
		}
	}
	return false
}

func (m *dtInt8) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		log.Warnf("cannot parse %s to int8, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtInt8) ParseFromLua(v lua.LValue) interface{} {
	return int8(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt8) ParseRawFromLua(v lua.LValue) interface{} {
	return int8(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt8) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtInt16 struct {
	detail dataTypeDetail
}

func (m *dtInt16) Name() string {
	return dataTypeNameInt16
}

func (m *dtInt16) Type() string {
	return m.Name()
}

func (m *dtInt16) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtInt16) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtInt16) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtInt16) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= int16Max && value >= int16Min {
			return true
		}
	}
	return false
}

func (m *dtInt16) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseInt(v, 10, 16)
	if err != nil {
		log.Warnf("cannot parse %s to int16, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtInt16) ParseFromLua(v lua.LValue) interface{} {
	return int16(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt16) ParseRawFromLua(v lua.LValue) interface{} {
	return int16(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt16) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtInt32 struct {
	detail dataTypeDetail
}

func (m *dtInt32) Name() string {
	return dataTypeNameInt32
}

func (m *dtInt32) Type() string {
	return m.Name()
}

func (m *dtInt32) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtInt32) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtInt32) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtInt32) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= int32Max && value >= int32Min {
			return true
		}
	}
	return false
}

func (m *dtInt32) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		log.Warnf("cannot parse %s to int32, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtInt32) ParseFromLua(v lua.LValue) interface{} {
	return int32(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt32) ParseRawFromLua(v lua.LValue) interface{} {
	return int32(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt32) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtInt64 struct {
	detail dataTypeDetail
}

func (m *dtInt64) Name() string {
	return dataTypeNameInt64
}

func (m *dtInt64) Type() string {
	return m.Name()
}

func (m *dtInt64) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtInt64) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtInt64) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtInt64) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= int64Max && value >= int64Min {
			return true
		}
	}
	return false
}

func (m *dtInt64) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		log.Warnf("cannot parse %s to int64, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtInt64) ParseFromLua(v lua.LValue) interface{} {
	return int64(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt64) ParseRawFromLua(v lua.LValue) interface{} {
	return int64(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtInt64) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtUint8 struct {
	detail dataTypeDetail
}

func (m *dtUint8) Name() string {
	return dataTypeNameUint8
}

func (m *dtUint8) Type() string {
	return m.Name()
}

func (m *dtUint8) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtUint8) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtUint8) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtUint8) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= uint8Max && value >= uint8Min {
			return true
		}
	}
	return false
}

func (m *dtUint8) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		log.Warnf("cannot parse %s to uint8, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtUint8) ParseFromLua(v lua.LValue) interface{} {
	return uint8(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint8) ParseRawFromLua(v lua.LValue) interface{} {
	return uint8(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint8) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtUint16 struct {
	detail dataTypeDetail
}

func (m *dtUint16) Name() string {
	return dataTypeNameUint16
}

func (m *dtUint16) Type() string {
	return m.Name()
}

func (m *dtUint16) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtUint16) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtUint16) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtUint16) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= uint16Max && value >= uint16Min {
			return true
		}
	}
	return false
}

func (m *dtUint16) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		log.Warnf("cannot parse %s to uint16, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtUint16) ParseFromLua(v lua.LValue) interface{} {
	return uint16(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint16) ParseRawFromLua(v lua.LValue) interface{} {
	return uint16(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint16) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtUint32 struct {
	detail dataTypeDetail
}

func (m *dtUint32) Name() string {
	return dataTypeNameUint32
}

func (m *dtUint32) Type() string {
	return m.Name()
}

func (m *dtUint32) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtUint32) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtUint32) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtUint32) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= uint32Max && value >= uint32Min {
			return true
		}
	}
	return false
}

func (m *dtUint32) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		log.Warnf("cannot parse %s to uint32, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtUint32) ParseFromLua(v lua.LValue) interface{} {
	return uint32(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint32) ParseRawFromLua(v lua.LValue) interface{} {
	return uint32(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint32) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtUint64 struct {
	detail dataTypeDetail
}

func (m *dtUint64) Name() string {
	return dataTypeNameUint64
}

func (m *dtUint64) Type() string {
	return m.Name()
}

func (m *dtUint64) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtUint64) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtUint64) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtUint64) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTNumber {
		value := v.(lua.LNumber)
		if value <= uint32Max && value >= uint32Min {
			return true
		}
	}
	return false
}

func (m *dtUint64) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}
	value, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		log.Warnf("cannot parse %s to uint64, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(value), nil
}

func (m *dtUint64) ParseFromLua(v lua.LValue) interface{} {
	return uint32(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint64) ParseRawFromLua(v lua.LValue) interface{} {
	return uint32(defaultSerialize(m, v).(lua.LNumber))
}

func (m *dtUint64) ParseToLua(v interface{}) lua.LValue {
	return defaultParseNumber(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtBool struct {
	detail dataTypeDetail
}

func (m *dtBool) Name() string {
	return dataTypeNameBool
}

func (m *dtBool) Type() string {
	return m.Name()
}

func (m *dtBool) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtBool) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtBool) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtBool) IsSameType(v lua.LValue) bool {
	return v.Type() == lua.LTBool
}

func (m *dtBool) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LBool(false), nil
	}
	value, err := strconv.ParseBool(v)
	if err != nil {
		log.Warnf("cannot parse %s to bool, error: %s", v, err.Error())
		return lua.LBool(false), err
	}
	return lua.LBool(value), nil
}

func (m *dtBool) ParseFromLua(v lua.LValue) interface{} {
	return bool(defaultSerialize(m, v).(lua.LBool))
}

func (m *dtBool) ParseRawFromLua(v lua.LValue) interface{} {
	return bool(defaultSerialize(m, v).(lua.LBool))
}

func (m *dtBool) ParseToLua(v interface{}) lua.LValue {
	switch val := v.(type) {
	case bool:
		return lua.LBool(val)
	default:
		return m.Default()
	}
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtString struct {
	detail dataTypeDetail
}

func (m *dtString) Name() string {
	return dataTypeNameString
}

func (m *dtString) Type() string {
	return m.Name()
}

func (m *dtString) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtString) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtString) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtString) IsSameType(v lua.LValue) bool {
	return v.Type() == lua.LTString
}

func (m *dtString) ParseDefaultVal(v string) (lua.LValue, error) {
	return lua.LString(v), nil
}

func (m *dtString) ParseFromLua(v lua.LValue) interface{} {
	return string(defaultSerialize(m, v).(lua.LString))
}

func (m *dtString) ParseRawFromLua(v lua.LValue) interface{} {
	return string(defaultSerialize(m, v).(lua.LString))
}

func (m *dtString) ParseToLua(v interface{}) lua.LValue {
	return defaultParseString(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtFloat struct {
	detail  dataTypeDetail
	decimal int //示例: 保留两位小数则decimal为100
}

func (m *dtFloat) Name() string {
	return dataTypeNameFloat
}

func (m *dtFloat) Type() string {
	return m.Name()
}

func (m *dtFloat) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtFloat) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtFloat) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtFloat) IsSameType(v lua.LValue) bool {
	return v.Type() == lua.LTNumber
}

func (m *dtFloat) ParseDefaultVal(v string) (lua.LValue, error) {
	if len(v) == 0 {
		return lua.LNumber(0), nil
	}

	value, err := strconv.ParseFloat(v, 64)
	if err != nil {
		log.Warnf("cannot parse %s to float64, error: %s", v, err.Error())
		return lua.LNumber(0), err
	}
	return lua.LNumber(math.Trunc(value*float64(m.decimal)) / float64(m.decimal)), nil
}

func (m *dtFloat) ParseFromLua(v lua.LValue) interface{} {
	var r lua.LValue
	if m.IsSameType(v) {
		val := v.(lua.LNumber)
		r = lua.LNumber(math.Trunc(float64(val)*float64(m.decimal)) / float64(m.decimal))
	} else {
		r = m.Default()
	}
	return float64(r.(lua.LNumber))
}

func (m *dtFloat) ParseRawFromLua(v lua.LValue) interface{} {
	return m.ParseFromLua(v)
}

func (m *dtFloat) ParseToLua(v interface{}) lua.LValue {
	val := defaultParseNumber(m, v).(lua.LNumber)
	return lua.LNumber(math.Trunc(float64(val)*float64(m.decimal)) / float64(m.decimal))
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtTable struct {
	detail dataTypeDetail
}

func (m *dtTable) Name() string {
	return dataTypeNameTable
}

func (m *dtTable) Type() string {
	return m.Name()
}

func (m *dtTable) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtTable) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtTable) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtTable) IsSameType(v lua.LValue) bool {
	return v.Type() == lua.LTTable
}

func (m *dtTable) ParseDefaultVal(v string) (lua.LValue, error) {
	return JsonToTable(v)
}

func (m *dtTable) ParseFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}
	return TableToMap(v.(*lua.LTable))
}

func (m *dtTable) ParseRawFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}
	return TableToMap(v.(*lua.LTable))
}

func (m *dtTable) ParseToLua(v interface{}) lua.LValue {
	return defaultParseTable(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtMap struct {
	detail dataTypeDetail
	key    dataType
	value  dataType
}

func (m *dtMap) Name() string {
	return dataTypeNameMap
}

func (m *dtMap) Type() string {
	return m.Name() + "[" + m.key.Type() + "]" + m.value.Type()
}

func (m *dtMap) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtMap) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtMap) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtMap) IsSameType(v lua.LValue) bool {
	switch converted := v.(type) {
	case *lua.LTable:
		for ck, cv := converted.Next(lua.LNil); ck != lua.LNil; ck, cv = converted.Next(ck) {
			if m.key.IsSameType(ck) == false {
				return false
			}
			if m.value.IsSameType(cv) == false {
				return false
			}
		}
	default:
		return false
	}

	return true
}

func (m *dtMap) ParseDefaultVal(v string) (lua.LValue, error) {
	r, err := JsonToTable(v)
	if err == nil && m.IsSameType(r) == false {
		return r, errors.New("type check failed")
	}
	return r, err
}

func (m *dtMap) ParseFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}

	r := make(map[string]interface{})
	switch converted := v.(type) {
	case *lua.LTable:
		for ck, cv := converted.Next(lua.LNil); ck != lua.LNil; ck, cv = converted.Next(ck) {
			if ck.Type() == lua.LTNumber {
				r[numberToMapKey(ck.(lua.LNumber))] = m.value.ParseFromLua(cv)
			} else {
				r[ck.String()] = m.value.ParseFromLua(cv)
			}
		}
	}
	return r
}

func (m *dtMap) ParseRawFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}

	r := make(map[string]interface{})
	r2 := make(map[int64]interface{})
	switch converted := v.(type) {
	case *lua.LTable:
		for ck, cv := converted.Next(lua.LNil); ck != lua.LNil; ck, cv = converted.Next(ck) {
			if ck.Type() == lua.LTNumber {
				r2[int64(ck.(lua.LNumber))] = m.value.ParseFromLua(cv)
			} else {
				r[ck.String()] = m.value.ParseFromLua(cv)
			}
		}
	}
	if len(r2) > 0 {
		return r2
	}
	return r
}

func (m *dtMap) ParseToLua(v interface{}) lua.LValue {
	t := luaL.NewTable()
	success := true
	switch converted := v.(type) {
	case map[string]interface{}:
		for key, val := range converted {
			var nk lua.LValue
			if k, err := mapKeyToNumber(key); err == nil {
				nk = k
			} else {
				nk = m.key.ParseToLua(key)
			}
			if nk == lua.LNumber(0) || nk == lua.LString("") {
				success = false
				break
			}
			luaL.RawSet(t, nk, m.value.ParseToLua(val))
		}
	default:
		log.Warnf("value[%+v] cannot set to %s", v, m.Type())
	}
	if success == false || m.IsSameType(t) == false {
		log.Warnf("value[%+v] not match type %s, set to default[%+v]", v, m.Type(), m.Default())
		return m.Default()
	}
	return t
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
// 注意！！！脚本层修改数组的某个元素不会被检测到,需要脚本层自行保证修改后仍然是有效的数组
type dtArray struct {
	detail dataTypeDetail
	value  dataType
}

func (m *dtArray) Name() string {
	return dataTypeNameArray
}

func (m *dtArray) Type() string {
	return m.Name() + "[]" + m.value.Type()
}

func (m *dtArray) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtArray) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtArray) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtArray) IsSameType(v lua.LValue) bool {
	switch converted := v.(type) {
	case *lua.LTable:
		expect := 1
		ck, cv := converted.Next(lua.LNil)
		for ck != lua.LNil {
			if ck.Type() != lua.LTNumber || ck.(lua.LNumber) != lua.LNumber(expect) || m.value.IsSameType(cv) == false {
				return false
			}
			expect += 1
			ck, cv = converted.Next(ck)
		}
	default:
		return false
	}

	return true
}

func (m *dtArray) ParseDefaultVal(v string) (lua.LValue, error) {
	r, err := JsonToTable(v)
	if err == nil && m.IsSameType(r) == false {
		return r, errors.New("type check failed")
	}
	return r, err
}

func (m *dtArray) ParseFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}

	r := make(map[string]interface{})
	switch converted := v.(type) {
	case *lua.LTable:
		ck, cv := converted.Next(lua.LNil)
		for ck != lua.LNil {
			r[numberToMapKey(ck.(lua.LNumber))] = m.value.ParseFromLua(cv)
			ck, cv = converted.Next(ck)
		}
	}
	return r
}

func (m *dtArray) ParseRawFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}

	r := make([]interface{}, 0)
	switch converted := v.(type) {
	case *lua.LTable:
		ck, cv := converted.Next(lua.LNil)
		for ck != lua.LNil {
			r = append(r, m.value.ParseRawFromLua(cv))
			ck, cv = converted.Next(ck)
		}
	}
	return r
}

func (m *dtArray) ParseToLua(v interface{}) lua.LValue {
	t := luaL.NewTable()
	switch converted := v.(type) {
	case map[string]interface{}:
		for key, val := range converted {
			if nk, err := mapKeyToNumber(key); err != nil {
				log.Warnf("value[%+v] not match type %s, set to default[%+v]", v, m.Type(), m.Default())
				return m.Default()
			} else {
				luaL.RawSet(t, nk, m.value.ParseToLua(val))
			}
		}
	case []interface{}:
		for idx, val := range converted {
			luaL.RawSet(t, lua.LNumber(idx+1), m.value.ParseToLua(val))
		}
	default:
		log.Warnf("value[%+v] cannot set to %s", v, m.Type())
	}
	if m.IsSameType(t) == false {
		log.Warnf("value[%v] not match type %s, set to default[%+v]", v, m.Type(), m.Default())
		return m.Default()
	}
	return t
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtStruct struct {
	detail dataTypeDetail
	props  map[string]propertyInfo
}

func (m *dtStruct) Name() string {
	return dataTypeNameStruct
}

func (m *dtStruct) Type() string {
	r := m.Name() + "{"
	if m.props != nil {
		for k, v := range m.props {
			r += k + "(" + v.dt.Type() + "), "
		}
	}
	r += "}"
	return r
}

func (m *dtStruct) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtStruct) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtStruct) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtStruct) IsSameType(v lua.LValue) bool {
	switch converted := v.(type) {
	case *lua.LTable:
		fieldCount := 0

		for ck, cv := converted.Next(lua.LNil); ck != lua.LNil; ck, cv = converted.Next(ck) {
			fieldCount += 1
			if pInfo, find := m.props[ck.String()]; find {
				if pInfo.dt.IsSameType(cv) == false {
					return false
				}
			} else {
				log.Errorf("unknown field[%s] for type[%s]", ck.String(), m.Type())
				return false
			}
		}
		//if fieldCount != len(m.props) {
		//	log.Errorf("value[%+v] field count[%d] not match to type[%s]", v, fieldCount, m.Type())
		//	return false
		//}
	default:
		return false
	}

	return true
}

func (m *dtStruct) ParseDefaultVal(_ string) (lua.LValue, error) {
	r := luaL.NewTable()
	for propName, pInfo := range m.props {
		luaL.SetField(r, propName, pInfo.dt.Default())
	}
	if m.IsSameType(r) == false {
		return r, errors.New("type check failed")
	}
	return r, nil
}

// AssignToStruct 赋值给struct,非table类型则失败,否则只取匹配的部分
func (m *dtStruct) AssignToStruct(t *lua.LTable, v lua.LValue) error {
	if v.Type() != lua.LTTable {
		return fmt.Errorf("%s cannot assign to %s", v.Type().String(), m.Type())
	}
	for propName, pInfo := range m.props {
		if val := luaL.GetField(v, propName); pInfo.dt.IsSameType(val) {
			t.RawSet(lua.LString(propName), val)
		} else {
			t.RawSet(lua.LString(propName), pInfo.dt.Default())
		}
	}
	return nil
}

func (m *dtStruct) ParseFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}

	r := make(map[string]interface{})
	switch converted := v.(type) {
	case *lua.LTable:
		for propName, pInfo := range m.props {
			val := converted.RawGet(lua.LString(propName))
			r[propName] = pInfo.dt.ParseFromLua(val)
		}
	}
	return r
}

func (m *dtStruct) ParseRawFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}

	r := make(map[string]interface{})
	switch converted := v.(type) {
	case *lua.LTable:
		for propName, pInfo := range m.props {
			val := converted.RawGet(lua.LString(propName))
			r[propName] = pInfo.dt.ParseRawFromLua(val)
		}
	}
	return r
}

func (m *dtStruct) ParseToLua(v interface{}) lua.LValue {
	t := m.Default().(*lua.LTable)
	success := true
	switch converted := v.(type) {
	case map[string]interface{}:
		for propName, pInfo := range m.props {
			if value, find := converted[propName]; find == false {
				luaL.RawSet(t, lua.LString(propName), pInfo.dt.Default())
			} else {
				luaL.RawSet(t, lua.LString(propName), pInfo.dt.ParseToLua(value))
			}
		}
	default:
		success = false
	}
	if success == false || m.IsSameType(t) == false {
		log.Warnf("value[%+v] not match type %s, set to default[%+v]", v, m.Type(), m.Default())
		return m.Default()
	}
	return t
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtMailBox struct {
	detail dataTypeDetail
}

func (m *dtMailBox) Name() string {
	return dataTypeNameMailBox
}

func (m *dtMailBox) Type() string {
	return m.Name()
}

func (m *dtMailBox) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtMailBox) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtMailBox) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtMailBox) IsSameType(v lua.LValue) bool {
	if v.Type() == lua.LTTable {
		return luaL.GetField(v, mailboxFieldType).Type() == lua.LTNumber
	}
	return false
}

func (m *dtMailBox) ParseDefaultVal(_ string) (lua.LValue, error) {
	t := MailBoxToTable(nil)
	return t, nil
}

func (m *dtMailBox) ParseFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}
	return TableToMailBox(v.(*lua.LTable))
}

func (m *dtMailBox) ParseRawFromLua(v lua.LValue) interface{} {
	return m.ParseFromLua(v)
}

func (m *dtMailBox) ParseToLua(v interface{}) lua.LValue {
	return defaultParseMailBox(m, v)
}

//--------------------------------------------------------------------

// --------------------------------------------------------------------
type dtSyncTable struct {
	detail dataTypeDetail
}

func (m *dtSyncTable) Name() string {
	return dataTypeNameSyncTable
}

func (m *dtSyncTable) Type() string {
	return m.Name()
}

func (m *dtSyncTable) Detail() *dataTypeDetail {
	return &m.detail
}

func (m *dtSyncTable) Default() lua.LValue {
	return m.detail.defaultVal
}

func (m *dtSyncTable) SetDefault(val string) error {
	if v, err := m.ParseDefaultVal(val); err != nil {
		return err
	} else {
		m.detail.defaultVal = v
	}
	return nil
}

func (m *dtSyncTable) IsSameType(v lua.LValue) bool {
	if v.Type() != lua.LTTable {
		return false
	}
	if _, ok := v.(*lua.LTable).RawGetString(SyncTableFieldProps).(*lua.LTable); !ok {
		return false
	}
	return true
}

func (m *dtSyncTable) ParseDefaultVal(v string) (lua.LValue, error) {
	t := newSyncTable(m.detail.name)
	r, err := JsonToTable(v)
	if err == nil {
		t.RawSetString(SyncTableFieldProps, r)
	}
	return t, err
}

func (m *dtSyncTable) ParseFromLua(v lua.LValue) interface{} {
	if m.IsSameType(v) == false {
		v = m.Default()
	}
	return TableToMap(luaL.GetField(v, SyncTableFieldProps).(*lua.LTable))
}

func (m *dtSyncTable) ParseRawFromLua(v lua.LValue) interface{} {
	return m.ParseFromLua(v)
}

func (m *dtSyncTable) ParseToLua(v interface{}) lua.LValue {
	switch val := v.(type) {
	case map[string]interface{}:
		r := MapToTable(val)
		var t *lua.LTable
		t = newSyncTable(m.detail.name)
		t.RawSetString(SyncTableFieldProps, r)
		if m.IsSameType(t) {
			return t
		}
	}
	log.Warnf("value[%v] type[%+v] not match type %s, set to default[%+v]", v, reflect.TypeOf(v).Name(), m.Type(), m.Default())
	return m.Default()
}

func (m *dtSyncTable) AssignToSyncTable(t *lua.LTable, v lua.LValue) error {
	if v.Type() != lua.LTTable {
		return fmt.Errorf("%s cannot assign to %s", v.Type().String(), m.Type())
	}
	newVal := v.(*lua.LTable)
	props := luaL.GetField(v, SyncTableFieldProps)
	var newPropTable *lua.LTable
	if props.Type() == lua.LTTable {
		newPropTable = props.(*lua.LTable)
	} else {
		newPropTable = luaL.NewTable()
		for ck, cv := newVal.Next(lua.LNil); ck != lua.LNil; ck, cv = newVal.Next(ck) {
			luaL.RawSet(newPropTable, ck, cv)
		}
	}
	t.RawSetString(SyncTableFieldProps, newPropTable)
	if ownerId, ok := luaL.GetField(t, SyncTableFieldOwner).(lua.LNumber); ok {
		if ent := GetEntityManager().GetEntityById(EntityIdType(ownerId)); ent != nil {
			if propName, ok := luaL.GetField(t, SyncTableFieldName).(lua.LString); ok {
				if propInfo := ent.def.prop(propName.String()); propInfo != nil && propInfo.config.IsSyncProp() {
					ent.onSyncPropChanged(propName.String(), newPropTable, propInfo.dt)
				}
			}
		}
	}
	return nil
}

//--------------------------------------------------------------------
