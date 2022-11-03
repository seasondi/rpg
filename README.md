# rpg
## 一款基于属性同步与rpc通信的游戏服务器lua框架

---
##目录结构
+ rpg
+ +  config: 配置文件目录
+ + engine:引擎目录
+ + + amdin: 提供web服务与其他进程通信
+ + + dbmanager: 数据库进程
+ + + engine: 通用基础逻辑
+ + + game: 游戏逻辑进程
+ + + gate: 网关进程
+ + + libs: 第三方库
+ + + message: 内部通信消息定义
+ + + robot: 机器人进程
+ + scripts: lua脚本业务逻辑示例
+ + + defs: 属性定义与rpc描述文件
+ + + game: 业务逻辑脚本
+ + tools: 工具目录
+ + + web: 调试页面(包含控制台调试指令, gm指令, 导表)
+ + + xls2lua: 导表工具,excel导出lua文件

---
##TODO
+ 私服管理工具
+ entity注册、管理、发现
+ AOI模块
+ 停服存盘逻辑调整
+ ...

---
###具体示例参考scripts目录