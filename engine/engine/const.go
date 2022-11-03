package engine

import "time"

type EntityIdType int64   //entityId类型
type ServerType int       //服务器类型
type ServerIdType int32   //服务器ID类型
type ServerTagType int    //服务器编号类型
type ConnectIdType uint32 //客户端连接ID类型

const entityIdTypeString = "int64" //entityId类型名

const (
	ServerTick          = 100 * time.Millisecond //服务器tick间隔
	defaultSaveInterval = 5                      //自动存盘间隔, 单位: 分钟
	heartbeatTick       = 3                      //心跳时间,单位：秒
)

const (
	saveTypeBack  = 0 //存盘方式 --加到队列尾
	saveTypeFront = 1 //存盘方式 --加到队列头
)

const (
	dataTypeNameInt8      = "int8"
	dataTypeNameInt16     = "int16"
	dataTypeNameInt32     = "int32"
	dataTypeNameInt64     = "int64"
	dataTypeNameUint8     = "uint8"
	dataTypeNameUint16    = "uint16"
	dataTypeNameUint32    = "uint32"
	dataTypeNameUint64    = "uint64"
	dataTypeNameString    = "string"
	dataTypeNameBool      = "bool"
	dataTypeNameFloat     = "float"
	dataTypeNameTable     = "table"
	dataTypeNameMap       = "map"
	dataTypeNameArray     = "array"
	dataTypeNameStruct    = "struct"
	dataTypeNameMailBox   = "mailbox"
	dataTypeNameSyncTable = "sync_table"
)

const (
	SyncTableFieldProps = "__props"
	SyncTableFieldOwner = "__owner"
	SyncTableFieldName  = "__name"
)

const (
	globalEntry   = "rpg"           //lua脚本中全局访问入口
	entitiesEntry = "entities"      //entity集合
	bootstrapLua  = "bootstrap.lua" //初始启动脚本
)

//进程类型
const (
	STGate ServerType = iota
	STGame
	STDbMgr
	STAdmin
	STRobot
)

//引擎注册给entity的属性
const (
	entityFieldType = "__type"
	entityFieldName = "__name"
	entityFieldId   = "id"
)

//mongo中记录的非def定义的字段
const (
	MongoFieldId   = entityFieldId
	MongoFieldName = entityFieldName
	MongoPrimaryId = "_id"
)

type DBType uint32

const (
	DBTypeProject DBType = iota //项目数据库
	DBTypeCommon                //公共库
	DBTypeMax
)

type DBTaskType uint32

//db的任务类型
const (
	DBTaskTypeQueryOne   DBTaskType = iota //查询单条数据
	DBTaskTypeUpdateOne                    //更新单条数据
	DBTaskTypeReplaceOne                   //替换单条数据
	DBTaskTypeDeleteOne                    //删除单条数据
	DBTaskTypeQueryMany                    //查询多条数据
	DBTaskTypeDeleteMany                   //删除多条数据

	DBTaskTypeMax
)

//etcd关注的key前缀
const (
	ServiceGamePrefix   = "game."
	ServiceDBPrefix     = "db."
	ServiceGatePrefix   = "gate."
	ServiceClientPrefix = "client."
	ServiceAdminPrefix  = "admin."
	StubPrefix          = "stub."
	EntityPrefix        = "entity."
)

//注册到etcd的对象类型, EtcdValueType取值
const (
	EtcdTypeServer = "server" //游戏进程
	EtcdTypeStub   = "stub"   //stub类型entity
	EtcdTypeEntity = "entity" //entity
)

//注册到etcd的value map的key
const (
	EtcdValueAddr   = "addr"   //进程地址
	EtcdValueType   = "type"   //对象类型
	EtcdValueIsStub = "isStub" //是否是stub进程

	EtcdValueServer    = "server"   //所在进程名称
	EtcdValueName      = "name"     //名字
	EtcdStubValueEntry = "entry"    //entry stub名字
	EtcdValueEntityId  = "entityId" //entity的id
)

//etcd租约ttl
const (
	EtcdStubLeaseTTL   = 3
	EtcdServerLeaseTTL = 3
)

const (
	ClientMsgDataFieldType     = "6d5e7" //__type, 消息类型
	ClientMsgDataFieldEntityID = "2ec86" //__entity, entity id
	ClientMsgDataFieldArgs     = "4ac22" //__args, 参数
)

//与客户端交互消息类型, 取值范围[1,150]
const (
	ClientMsgTypePropSync       = iota + 1 //同步属性给客户端 S->C
	ClientMsgTypeEntityRpc                 //调用entity方法 C->S & S->C
	ClientMsgTypeCreateEntity              //创建客户端entity S->C
	ClientMsgTypeLogin                     //客户端登录 C->S
	ClientMsgTypeClose                     //客户端断开连接 C->S
	ClientMsgTypeTips                      //服务器提示消息 S->C
	ClientMsgTypePropSyncUpdate            //属性增量同步给客户端 S->C
	ClientMsgTypeHeartBeat                 //客户端心跳 C->S & S->C
)

//服务器内部消息类型,取值范围[151,255]
const (
	ServerMessageTypeHeartBeat           = iota + 151 //心跳
	ServerMessageTypeLogin                            //登录
	ServerMessageTypeDBCommand                        //与db交互的消息
	ServerMessageTypeEntityRpc                        //发给entity的rpc消息
	ServerMessageTypeEntityRouter                     //交由gate转发的集群内部entity消息
	ServerMessageTypeDisconnectClient                 //连接断开的消息
	ServerMessageTypeSayHello                         //连接建立后同步信息
	ServerMessageTypeCreateGameEntity                 //选择game创建entity消息
	ServerMessageTypeCreateGameEntityRsp              //选择game创建entity消息回包
	ServerMessageTypeHeartBeatRsp                     //心跳回包
	ServerMessageTypeLoginByOther                     //被顶号
	ServerMessageTypeServerError                      //服务器错误消息
	ServerMessageTypeChangeEntityClient               //entity与客户端连接绑定/解绑
	ServerMessageTypeSetServerTime                    //修改服务器时间
)

//ClientMsgTypeError类型的消息内容
const (
	ErrMsgClientConnectionInvalid = "INVALID_CLIENT_CONNECTION" //无效的客户端连接
	ErrMsgClientNotLogin          = "CLIENT_NOT_LOGIN"          //尚未登录
	ErrMsgServerNotReady          = "SERVER_NOT_READY"          //服务器尚未准备好
	ErrMsgLoginByOther            = "LOGIN_BY_OTHER"            //被顶号
	ErrMsgInvalidMessage          = "INVALID_MESSAGE"           //无效消息
)

const StubEntryMethod = "entry" //entry stub必须定义的函数,登录主入口

//脚本层的接口名
const (
	onServerTimeUpdate = "on_server_time_update" //服务器时间变化时脚本层回调
	onReload           = "on_reload"             //热更
	doGmCommand        = "do_gm_command"         //执行gm命令
	getGmListCommand   = "get_gm_list"           //获取gm列表
	onEntityInit       = "on_init"               //entity初始化
	onEntityDestroy    = "on_destroy"            //entity销毁
	onEntityFinal      = "on_final"              //存盘的entity销毁完成
	onEntityGetClient  = "on_get_client"         //entity绑定到客户端连接
	onEntityLostClient = "on_lose_client"        //entity失去客户端连接
)

const (
	LuaTableNumberKeyPrefix = "__NUM"
	LuaTableValueNilField   = "__NIL"
)
