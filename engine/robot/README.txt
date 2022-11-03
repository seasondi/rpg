消息格式：
双端交互的消息结构为一个msgpack打包的字典,包含以下字段
ClientMsgDataFieldType: 消息类型
ClientMsgDataFieldEntityID: entityID
ClientMsgDataFieldArgs: 消息参数

枚举定义：
ClientMsgDataFieldType     = "6d5e7" //__type
ClientMsgDataFieldEntityID = "2ec86" //__entity
ClientMsgDataFieldArgs     = "4ac22" //__args

ClientMsgDataFieldType的值:
ClientMsgTypePropSync     = 1 //同步属性给客户端
ClientMsgTypeMethodCall   = 2 //调用entity方法
ClientMsgTypeCreateEntity = 3 //创建客户端entity
ClientMsgTypeLogin        = 4 //客户端登录
ClientMsgTypeError        = 6 //服务器错误消息

以下说明中rpc函数名均为函数名经过统一算法换算后的一个字符串

服务端发给客户端的消息
1.创建客户端实体(S->C)
ClientMsgDataFieldType: ClientMsgTypeCreateEntity
ClientMsgDataFieldEntityID: 被创建的entityID
ClientMsgDataFieldArgs:包含两个元素的数组,第一个为entity名称,第二个为属性字典(属性名称->属性值的映射)

2.同步entity属性(S->C)
ClientMsgDataFieldType: ClientMsgTypePropSync
ClientMsgDataFieldEntityID: entityID
ClientMsgDataFieldArgs: 包含两个元素的数组,第一个为属性名称,第二个为属性值

3.调用客户端rpc函数(S->C)
ClientMsgDataFieldType: ClientMsgTypeMethodCall
ClientMsgDataFieldEntityID: entityID
ClientMsgDataFieldArgs: 包含多个元素的数组,第一个为rpc函数名,剩下的为该函数参数

4.服务器出错(S->C)
ClientMsgDataFieldType: ClientMsgTypeError
ClientMsgDataFieldArgs: 包含单个元素的数组,第一个元素为错误消息


客户端发给服务端的消息
1.登录(C->S)
固定调用服务端定义在stub entity的entry函数(def中entry函数第一个参数固定为客户端连接信息的table,客户端无需传入)
ClientMsgDataFieldArgs: 包含多个元素的数组,为entry函数的第2-n个参数

2.调用服务端rpc函数(C->S)
ClientMsgDataFieldType: MessageTypeEntityRpc
ClientMsgDataFieldEntityID: entityID
ClientMsgDataFieldArgs: 包含多个元素的数组,第一个为rpc函数名,剩下的为该函数参数
