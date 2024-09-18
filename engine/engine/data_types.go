package engine

import (
	"fmt"
	"strings"
)

func initDataTypes() error {
	dataTypeMgr = new(dataTypes)
	return nil
}

type dataTypes struct {
}

// NewDataTypeFromPropDef 从属性配置生成dataType
func (m *dataTypes) NewDataTypeFromPropDef(pDef propertyDef) (dataType, error) {
	dt, err := m.newDataType(pDef.Type.typeName, pDef.Type.name)
	if err != nil {
		return nil, err
	}
	dt, err = m.completeDataType(pDef.Type, dt)
	if err != nil {
		return nil, err
	}
	if err = dt.SetDefault(pDef.Default); err != nil {
		return nil, fmt.Errorf("new dataType error: %s, value[%s] cannot parse to type[%s]", err.Error(), pDef.Default, dt.Type())
	}
	return dt, nil
}

// NewDataTypeFromPropType 从类型生成dataType, 适用于函数参数等只有属性类型配置的字段
func (m *dataTypes) NewDataTypeFromPropType(pType propType) (dataType, error) {
	dt, err := m.newDataType(pType.typeName, pType.name)
	if err != nil {
		return nil, err
	}
	dt, err = m.completeDataType(pType, dt)
	if err != nil {
		return nil, err
	}
	return dt, nil
}

// please call NewDataTypeFromPropDef or NewDataTypeFromPropType
func (m *dataTypes) newDataType(typeName string, name string) (dataType, error) {
	lowerName := strings.ToLower(typeName)
	detail := dataTypeDetail{name: name}
	var dt dataType
	switch lowerName {
	case dataTypeNameInt8:
		dt = &dtInt8{detail: detail}
	case dataTypeNameInt16:
		dt = &dtInt16{detail: detail}
	case dataTypeNameInt32:
		dt = &dtInt32{detail: detail}
	case dataTypeNameInt64:
		dt = &dtInt64{detail: detail}
	case dataTypeNameUint8:
		dt = &dtUint8{detail: detail}
	case dataTypeNameUint16:
		dt = &dtUint16{detail: detail}
	case dataTypeNameUint32:
		dt = &dtUint32{detail: detail}
	case dataTypeNameUint64:
		dt = &dtUint64{detail: detail}
	case dataTypeNameString:
		dt = &dtString{detail: detail}
	case dataTypeNameBool:
		dt = &dtBool{detail: detail}
	case dataTypeNameFloat:
		dt = &dtFloat{detail: detail}
	case dataTypeNameTable:
		dt = &dtTable{detail: detail}
	case dataTypeNameMap:
		dt = &dtMap{detail: detail}
	case dataTypeNameArray:
		dt = &dtArray{detail: detail}
	case dataTypeNameStruct:
		dt = &dtStruct{detail: detail}
	case dataTypeNameMailBox:
		dt = &dtMailBox{detail: detail}
	//case dataTypeNameSyncTable:
	//	if gSvrType == STRobot {
	//		dt = &dtTable{detail: detail} //客户端按table处理即可
	//	} else {
	//		dt = &dtSyncTable{detail: detail}
	//	}
	default:
		return nil, fmt.Errorf("not support type[%s]", typeName)
	}

	return dt, nil
}

// please call NewDataTypeFromPropDef or NewDataTypeFromPropType
func (m *dataTypes) completeDataType(tp propType, dt dataType) (dataType, error) {
	switch dtt := dt.(type) {
	case *dtMap:
		if keyDt, err := m.newDataType(tp.keyType.typeName, tp.valueType.name); err == nil {
			if dtt.key, err = m.completeDataType(*tp.keyType, keyDt); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
		if valDt, err := m.newDataType(tp.valueType.typeName, tp.valueType.name); err == nil {
			if dtt.value, err = m.completeDataType(*tp.valueType, valDt); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	case *dtArray:
		if valDt, err := m.newDataType(tp.valueType.typeName, tp.valueType.name); err == nil {
			if dtt.value, err = m.completeDataType(*tp.valueType, valDt); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	case *dtStruct:
		dtt.props = tp.props
	case *dtFloat:
		dtt.decimal = 1
		for i := 0; i < tp.decimal; i++ {
			dtt.decimal *= 10
		}
	}
	return dt, nil
}
