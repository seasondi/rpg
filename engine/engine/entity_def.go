package engine

import (
	"fmt"
	"github.com/beevik/etree"
	lua "github.com/yuin/gopher-lua"
	"strconv"
	"strings"
)

// 属性同步方式
type syncFlag int

const (
	noClient     syncFlag = iota //无需同步给客户端
	ownClient                    //只同步给自己
	otherClients                 //同步给aoi内除自己外的其他entity
	allClients                   //同步给aoi内的所有entity
)

type readType int

const (
	readTypeProp        readType = iota //读取属性
	readTypeAlias                       //读取alias
	readTypeMapKey                      //读取map的key
	readTypeMapValue                    //读取map的value
	readTypeArrayValue                  //读取array的value
	readTypeFunctionArg                 //读取函数的参数
)

// def文件结构
const (
	defFieldRoot               = "root"          //根节点
	defFieldVolatile           = "Volatile"      //基础配置
	defFieldVolatileHasClient  = "HasClient"     //是否拥有客户端实体
	defFieldVolatilePersistent = "Persistent"    //entity是否需要存盘
	defFieldVolatileIsStub     = "IsStub"        //entity是否为stub
	defFieldImplements         = "Implements"    //继承的其他def
	defFieldProperties         = "Properties"    //属性列表
	defFieldClientMethods      = "ClientMethods" //客户端rpc函数声明
	defFieldServerMethods      = "ServerMethods" //服务端rpc函数声明
	defFieldOwnClient          = "own_client"    //同步方式-own_client
	defFieldOtherClients       = "other_clients" //同步方式-other_clients
	defFieldAllClients         = "all_clients"   //同步方式-all_clients
	defFieldFloatUnit          = "Unit"          //浮点数保留的小数位数
	defFieldArrayValue         = "Value"         //array的value
	defFieldMapKey             = "Key"           //map类型的key
	defFieldMapValue           = "Value"         //map类型的value
	defFieldPropType           = "Type"          //属性类型
	defFieldPropFlags          = "Flags"         //属性同步方式
	defFieldPropDefault        = "Default"       //属性默认值
	defFieldPropPersistent     = "Persistent"    //属性是否持久化
	defFieldRpcExposed         = "Exposed"       //服务器rpc函数是否暴露给客户端
)

var currentLoadDefFile string
var entryEntityName string

/*
propType def配置中属性的Type字段或者函数参数.

name: 属性名或者函数名

typeName: 类型名称.

decimal: float/double类型的小数位数

keyType: map类型独有, key的类型.

valueType: 值的类型(array, map类型有值).

props: struct类型独有, 各属性的描述信息.
*/
type propType struct {
	name      string
	typeName  string
	decimal   int
	keyType   *propType
	valueType *propType
	props     map[string]propertyInfo
}

/*
volatileDef def中Volatile配置
hasClient: 是否跟客户端连接绑定
persistent: 是否要自动存盘
*/
type volatileDef struct {
	hasClient  bool
	persistent bool
	isStub     bool
}

// propertyDef def文件中的属性配置信息
type propertyDef struct {
	Type       propType //属性类型
	Flags      syncFlag //属性同步方式
	Default    string   //默认值
	Persistent bool     //是否持久化
}

type propertyInfo struct {
	config propertyDef
	dt     dataType
}

type argInfo struct {
	ty propType
	dt dataType
}

// methodDef def文件中的函数参数
type methodDef struct {
	methodName string    //函数名
	exposed    bool      //是否暴露给客户端访问
	args       []argInfo //函数参数类型列表
}

// entityDef def文件描述信息
type entityDef struct {
	entityName        string
	volatile          volatileDef
	properties        map[string]propertyInfo
	clientMethods     map[string]*methodDef //mask name -> method def
	clientMethodsName map[string]string     //origin name -> mask name
	serverMethods     map[string]*methodDef //mask name -> method def
	serverMethodsName map[string]string     //origin name -> mask name
	interfaces        []string              //entity关联的所有子类
}

// flagStrToEnum 属性的Flags字段转枚举
func flagStrToEnum(flag string) syncFlag {
	f := strings.ToLower(flag)
	switch f {
	case defFieldOwnClient:
		return ownClient
	case defFieldOtherClients:
		return otherClients
	case defFieldAllClients:
		return allClients
	}
	if len(flag) != 0 {
		log.Errorf("unknown flag: %s, support list: [OWN_CLIENT, OTHER_CLIENTS, ALL_CLIENTS]", flag)
	}
	return noClient
}

// readPropType 读取属性、函数参数.
// el: 读取的xml标签
// propName: 属性名称或者函数名
// rtype: 读取类型
func readPropType(el *etree.Element, propName string, rtype readType) propType {
	typeName := strings.Trim(el.Text(), "\n ")
	lowerTypeName := strings.ToLower(typeName)
	r := propType{name: propName, typeName: typeName}
	switch lowerTypeName {
	case dataTypeNameFloat:
		Unit := el.FindElement(defFieldFloatUnit)
		if Unit == nil {
			r.decimal = 2 //默认保留两位小数
		} else {
			if decimal, err := strconv.ParseInt(Unit.Text(), 10, 64); err != nil {
				log.Fatalf("cannot parse \"Unit\" for prop[%s], error: %s", propName, err.Error())
			} else if decimal >= 0 && decimal <= 8 {
				r.decimal = int(decimal)
			} else {
				log.Fatalf("Unit should between [0,8], prop[%s]", propName)
			}
		}
	case dataTypeNameArray:
		of := el.FindElement(defFieldArrayValue)
		if of == nil {
			log.Fatalf("can not find \"%s\" element for ARRAY, propName[%s]", defFieldArrayValue, propName)
		}
		ptValue := readPropType(of, propName, readTypeArrayValue)
		r.valueType = &ptValue
	case dataTypeNameMap:
		key := el.FindElement(defFieldMapKey)
		if key == nil {
			log.Fatalf("can not find \"%s\" element for MAP, propName[%s]", defFieldMapKey, propName)
		}
		value := el.FindElement(defFieldMapValue)
		if value == nil {
			log.Fatalf("can not find \"%s\" element for MAP, propName[%s]", defFieldMapValue, propName)
		}
		ptKey := readPropType(key, propName, readTypeMapKey)
		ptValue := readPropType(value, propName, readTypeMapValue)
		r.keyType = &ptKey
		r.valueType = &ptValue
	case dataTypeNameStruct:
		r.props = make(map[string]propertyInfo, 0)
		for _, prop := range el.ChildElements() {
			if _, find := r.props[prop.Tag]; find {
				log.Fatalf("duplicate field[%s] for prop[%s]", prop.Tag, propName)
			}
			propDef := readPropertyDef(prop)
			dt, err := dataTypeMgr.NewDataTypeFromPropDef(propDef)
			if err != nil {
				log.Fatalf("cannot create dataType for prop[%s], error: %s", propName, err.Error())
			}

			r.props[prop.Tag] = propertyInfo{
				config: propDef,
				dt:     dt,
			}
		}
	case dataTypeNameSyncTable:
		if rtype != readTypeProp {
			log.Fatalf("read %s failed, %s only can be defined on properties", propName, typeName)
		}
	default:
		if alias := defMgr.GetAlias(typeName); alias != nil {
			r = *alias
			r.name = propName
		}
	}
	return r
}

func readPropertyDef(prop *etree.Element) propertyDef {
	r := propertyDef{}
	for _, e := range prop.ChildElements() {
		v := strings.Trim(e.Text(), "\n ")
		switch e.Tag {
		case defFieldPropType:
			r.Type = readPropType(e, prop.Tag, readTypeProp)
		case defFieldPropFlags:
			r.Flags = flagStrToEnum(v)
		case defFieldPropDefault:
			r.Default = v
		case defFieldPropPersistent:
			{
				if p, err := strconv.ParseBool(v); err == nil {
					r.Persistent = p
				} else {
					log.Fatalf("prop[%s] Persistent field parse error[%s]", prop.Tag, err.Error())
				}
			}
		default:
			log.Fatalf("prop[%s] has unknown tag[%s]", prop.Tag, e.Tag)
		}
	}
	if strings.ToLower(r.Type.typeName) == dataTypeNameSyncTable {
		if r.Flags == noClient {
			log.Fatalf("prop[%s] is %s, \"Flags\" field must be set", prop.Tag, r.Type.typeName)
		}
	}
	return r
}

func (m *propertyDef) IsSyncProp() bool {
	return m.Flags != noClient
}

func (m *entityDef) GetEntityFileName() string {
	return m.entityName + ".def"
}

func (m *entityDef) Load(name string) {
	m.entityName = name
	m.properties = make(map[string]propertyInfo)
	m.clientMethods = make(map[string]*methodDef)
	m.clientMethodsName = make(map[string]string)
	m.serverMethods = make(map[string]*methodDef)
	m.serverMethodsName = make(map[string]string)

	doc := etree.NewDocument()
	fileName := cfg.WorkPath + "/defs/" + m.GetEntityFileName()
	if err := doc.ReadFromFile(fileName); err != nil {
		log.Fatalf("load entity[%s] def failed, error: %s", name, err.Error())
	}
	currentLoadDefFile = fileName
	root := doc.SelectElement(defFieldRoot)
	if root == nil {
		log.Fatalf("def file[%s] must start with \"root\"", currentLoadDefFile)
	}

	m.loadVolatile(root)
	m.interfaces = append(m.interfaces, m.loadImplements(root)...)
	m.loadProperties(root)
	m.loadClientMethods(root)
	m.loadServerMethods(root)
	currentLoadDefFile = ""
}

func (m *entityDef) loadInterface(name string) {
	doc := etree.NewDocument()
	fileName := cfg.WorkPath + "/defs/interfaces/" + name + ".def"
	if err := doc.ReadFromFile(fileName); err != nil {
		log.Fatalf("load interface[%s] in file[%s], error: %s", name, currentLoadDefFile, err.Error())
	}
	currentLoadDefFile = fileName
	root := doc.SelectElement(defFieldRoot)
	if root == nil {
		log.Fatalf("def file[%s] must start with \"root\"", fileName)
	}
	m.interfaces = append(m.loadImplements(root), m.interfaces...)
	m.loadProperties(root)
	m.loadClientMethods(root)
	m.loadServerMethods(root)
	currentLoadDefFile = fileName
}

func (m *entityDef) loadVolatile(el *etree.Element) {
	if vel := el.SelectElement(defFieldVolatile); vel != nil {
		for _, v := range vel.ChildElements() {
			switch v.Tag {
			case defFieldVolatileHasClient:
				{
					if r, err := strconv.ParseBool(v.Text()); err == nil {
						m.volatile.hasClient = r
					} else {
						log.Fatalf("Volatile.HasClient should be bool error[%s], file[%s]", err.Error(), currentLoadDefFile)
					}
				}
			case defFieldVolatilePersistent:
				{
					if r, err := strconv.ParseBool(v.Text()); err == nil {
						m.volatile.persistent = r
					} else {
						log.Fatalf("Volatile.Persistent should be bool error[%s], file[%s]", err.Error(), currentLoadDefFile)
					}
				}
			case defFieldVolatileIsStub:
				{
					if r, err := strconv.ParseBool(v.Text()); err == nil {
						m.volatile.isStub = r
					} else {
						log.Fatalf("Volatile.IsStub should be bool error[%s], file[%s]", err.Error(), currentLoadDefFile)
					}
				}
			}
		}
	}
}

func (m *entityDef) loadImplements(el *etree.Element) []string {
	iFace := make([]string, 0)
	if iel := el.SelectElement(defFieldImplements); iel != nil {
		for _, v := range iel.ChildElements() {
			iFace = append(iFace, strings.Trim(v.Text(), " \t"))
		}
		for _, v := range iel.ChildElements() {
			m.loadInterface(v.Text())
		}
	}
	return iFace
}

func (m *entityDef) loadProperties(el *etree.Element) {
	if pel := el.SelectElement(defFieldProperties); pel != nil {
		for _, prop := range pel.ChildElements() {
			if strings.HasPrefix(prop.Tag, "_") {
				log.Fatalf("prop[%s] startswith _ is not allowed in file[%s]", prop.Tag, currentLoadDefFile)
			}
			if _, ok := m.properties[prop.Tag]; ok {
				log.Fatalf("duplicate defined prop[%s] in file[%s]", prop.Tag, currentLoadDefFile)
			}
			if isEntityReserveProp(prop.Tag) {
				log.Fatalf("cannot define prop[%s] in file[%s], it is reversed", prop.Tag, currentLoadDefFile)
			}
			propDef := readPropertyDef(prop)
			dt, err := dataTypeMgr.NewDataTypeFromPropDef(propDef)
			if err != nil {
				log.Fatalf("read prop[%s] in file[%s], error: %s", prop.Tag, currentLoadDefFile, err.Error())
			} else if dt.Name() == (&dtMailBox{}).Name() {
				log.Fatalf("prop[%s] in file[%s] type error, %s can only be as function argument", prop.Tag, currentLoadDefFile, dt.Type())
			}
			m.properties[prop.Tag] = propertyInfo{
				config: propDef,
				dt:     dt,
			}
		}
	}
}

func (m *entityDef) loadClientMethods(el *etree.Element) {
	methods := make(map[string]*methodDef)
	if cmEl := el.SelectElement(defFieldClientMethods); cmEl != nil {
		for _, method := range cmEl.ChildElements() {
			if _, ok := m.clientMethodsName[method.Tag]; ok {
				log.Fatalf("duplicate defined def client method[%s]", method.Tag)
			}
			methods[method.Tag] = m.readMethodArgs(method)
		}
	}
	clientMethods, clientMethodsName := m.maskMethods(methods)
	for k, v := range clientMethods {
		m.clientMethods[k] = v
	}
	for k, v := range clientMethodsName {
		m.clientMethodsName[k] = v
	}
}

func (m *entityDef) loadServerMethods(el *etree.Element) {
	methods := make(map[string]*methodDef)
	if smEl := el.SelectElement(defFieldServerMethods); smEl != nil {
		for _, method := range smEl.ChildElements() {
			if method.Tag == StubEntryMethod {
				if entryEntityName != "" && entryEntityName != m.entityName {
					log.Fatalf("duplicate entry method[%s] in file[%s], already defined in entity[%s]", method.Tag, currentLoadDefFile, entryEntityName)
				} else {
					if m.volatile.isStub == false {
						log.Fatalf("entry method[%s] can only be defined in stub def file", method.Tag)
					}
					entryEntityName = m.entityName
				}
			}
			if _, ok := m.serverMethodsName[method.Tag]; ok {
				log.Fatalf("duplicate defined def server method[%s]", method.Tag)
			}
			if _, ok := entityApiExports[method.Tag]; ok {
				log.Fatalf("server method[%s] is reversed.", method.Tag)
			}
			methods[method.Tag] = m.readMethodArgs(method)
		}
	}
	serverMethods, serverMethodsName := m.maskMethods(methods)
	for k, v := range serverMethods {
		m.serverMethods[k] = v
	}
	for k, v := range serverMethodsName {
		m.serverMethodsName[k] = v
	}
}

func (m *entityDef) readMethodArgs(el *etree.Element) *methodDef {
	r := &methodDef{
		methodName: el.Tag,
		exposed:    false,
		args:       make([]argInfo, 0),
	}
	for _, arg := range el.ChildElements() {
		if arg.Tag == defFieldRpcExposed {
			r.exposed = true
			if gSvrType != STRobot {
				r.args = append(r.args, m.genExposedArg())
			}
		} else {
			pType := readPropType(arg, r.methodName, readTypeFunctionArg)
			dt, err := dataTypeMgr.NewDataTypeFromPropType(pType)
			if err != nil {
				log.Fatalf("read method[%s] arg in file[%s] error: %s", r.methodName, currentLoadDefFile, err.Error())
			}
			r.args = append(r.args, argInfo{ty: pType, dt: dt})
		}
	}

	return r
}

func (m *entityDef) genExposedArg() argInfo {
	pType := propType{
		typeName: entityIdTypeString,
	}
	dt, _ := dataTypeMgr.NewDataTypeFromPropType(pType)
	return argInfo{
		ty: pType,
		dt: dt,
	}
}

func (m *entityDef) maskMethods(methods map[string]*methodDef) (map[string]*methodDef, map[string]string) {
	result := make(map[string]*methodDef)
	mapping := make(map[string]string)
	for name, def := range methods {
		md5Name := Md5(name)
		success := false
		for n := 4; n <= 32; n++ {
			newName := md5Name[0:n]
			if _, find := result[newName]; !find {
				result[newName] = def
				mapping[name] = newName
				success = true
				break
			}
		}
		if !success {
			log.Fatalf("method[%s] conflict", name)
		}
	}
	return result, mapping
}

func (m *entityDef) registerToEntity(ent *entity) {
	m.registerPropsToEntity(ent)
	m.registerClientMethodsToEntity(ent)
}

func (m *entityDef) loadInterfaceFiles() error {
	implementsPath := getLuaEntryValue("implementsPath")
	if implementsPath.Type() != lua.LTString {
		return fmt.Errorf(globalEntry + ".implementsPath is necessary, please set in script(relative path to \"WorkPath\" defined in config")
	}
	paths := strings.Split(implementsPath.String(), ";")
	for _, name := range m.interfaces {
		find := false
		for _, path := range paths {
			mod := cfg.WorkPath + "/" + path + "/" + name + ".lua"
			if err := luaL.DoFile(mod); err == nil {
				find = true
				break
			}
		}
		if !find {
			return fmt.Errorf("cannot find interface file[%s.lua] from all paths[%s]", name, implementsPath)
		}
	}
	return nil
}

func (m *entityDef) registerPropsToEntity(ent *entity) {
	if m.entityName != ent.entityName {
		log.Errorf("register props to entity[%s], but def is [%s]", ent.entityName, m.entityName)
		return
	}
	for propName, prop := range m.properties {
		val := prop.dt.Default()
		if prop.dt.Name() == dataTypeNameSyncTable {
			val.(*lua.LTable).RawSetString(SyncTableFieldOwner, EntityIdToLua(ent.entityId))
		}
		luaL.RawSet(ent.propsTable, lua.LString(propName), val)
	}
}

func (m *entityDef) registerClientMethodsToEntity(ent *entity) {
	if m.volatile.hasClient != true {
		return
	}
	ent.clientTable = luaL.NewTable()

	for _, method := range m.clientMethods {
		luaL.SetField(ent.clientTable, method.methodName, newClientFunction(method.methodName, ent.entityId))
	}
}

func (m *entityDef) isSyncClientProp(propName string) bool {
	if prop, ok := m.properties[propName]; ok {
		return prop.config.IsSyncProp()
	}
	return false
}

func (m *entityDef) isDefProp(propName string) bool {
	if _, ok := m.properties[propName]; ok {
		return true
	}
	return false
}

func (m *entityDef) propDataType(propName string) dataType {
	prop, ok := m.properties[propName]
	if !ok {
		return nil
	}
	return prop.dt
}

func (m *entityDef) prop(propName string) *propertyInfo {
	prop, ok := m.properties[propName]
	if !ok {
		return nil
	}
	return &prop
}

func (m *entityDef) getServerMethod(method string, isOriginName bool) *methodDef {
	name := method
	if isOriginName {
		if maskName, find := m.serverMethodsName[method]; find {
			name = maskName
		}
	}
	if md, ok := m.serverMethods[name]; ok {
		return md
	}
	return nil
}

func (m *entityDef) getClientMethod(method string, isOriginName bool) *methodDef {
	name := method
	if isOriginName {
		if maskName, find := m.clientMethodsName[method]; find {
			name = maskName
		}
	}
	if md, ok := m.clientMethods[name]; ok {
		return md
	}
	return nil
}

func (m *entityDef) getClientMethodMaskName(originName string) string {
	if maskName, find := m.clientMethodsName[originName]; find {
		return maskName
	}
	return originName
}

func (m *entityDef) getServerMethodMaskName(originName string) string {
	if maskName, find := m.serverMethodsName[originName]; find {
		return maskName
	}
	return originName
}
