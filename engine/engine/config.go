package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"strconv"
)

func GetConfig() *config {
	return cfg
}

// config config配置
type config struct {
	WorkPath       string        //工作路径
	ServerId       ServerIdType  //服务器ID
	Release        bool          //是否正式环境
	SaveInterval   int64         //单位: 分钟
	SaveNumPerTick int32         //每个tick存盘的entity数量
	PrintRpcLog    bool          //是否输出rpc日志
	Logger         loggerConfig  //日志配置
	Etcd           etcdConfig    //etcd配置
	Redis          *redisConfig  //redis配置
	server         *serverConfig //服务器配置
	vp             *viper.Viper  //配置文件读取模块
}

type loggerConfig struct {
	LogLevel   string //日志等级
	LogPath    string //日志文件输出路径
	Console    bool   //是否输出到控制台
	JsonFormat bool   //是否使用json格式输出
}

type etcdConfig struct {
	EndPoints string //etcd地址
}

type redisConfig struct {
	Hosts     string
	AloneMode bool
	DB        int
	Password  string
}

type serverConfig struct {
	//==============以下配置所有进程通用======================
	Addr string `json:"addr,omitempty"` //服务器监听地址
	//==============以上配置所有进程通用======================

	//==============以下配置game进程独有======================
	Telnet string `json:"telnet,omitempty"` //telnet监听地址
	IsStub bool   `json:"stub,omitempty"`   //是否stub类型进程
	DB     string `json:"db,omitempty"`     //db配置
	//==============以上配置game进程独有======================

	//==============以下配置db进程独有======================
	Database string `json:"database,omitempty"` //mongo地址
	//==============以上配置db进程独有======================

	//==============以下配置admin进程独有======================
	EnableWeb bool `json:"enable_web,omitempty"` //是否开启调试页面
	//==============以上配置admin进程独有======================
}

func initConfig() error {
	cfg = &config{
		vp: viper.New(),
	}
	cfg.vp.SetConfigFile(cmdLineMgr.Config)
	if err := cfg.vp.ReadInConfig(); err != nil {
		fmt.Printf("config init failed, error: %s\n", err.Error())
		return err
	}
	if err := cfg.vp.Unmarshal(cfg); err != nil {
		fmt.Printf("config init failed, error: %s\n", err.Error())
		return err
	}

	if cfg.SaveInterval <= 0 {
		cfg.SaveInterval = defaultSaveInterval
	}

	if cfg.WorkPath == "" {
		return errors.New("work path is empty")
	}

	if cfg.ServerId <= 0 {
		return fmt.Errorf("invalid server id: %d", cfg.ServerId)
	}

	if cfg.server = cfg.parseServerConfig(cfg.ServerKey()); cfg.server == nil {
		return fmt.Errorf("server key[%s] config load failed", cfg.ServerKey())
	}

	return nil
}

func (m *config) parseServerConfig(key string) *serverConfig {
	if info, ok := m.Get(key).(map[string]interface{}); ok {
		if v, err := json.Marshal(info); err == nil {
			r := serverConfig{}
			if err = json.Unmarshal(v, &r); err == nil {
				return &r
			} else {
				return nil
			}
		}
	}
	return nil
}

func (m *config) IsDebug() bool {
	return m.Release == false
}

func (m *config) Get(name string) interface{} {
	return cfg.vp.Get(name)
}

func (m *config) GetString(name string) string {
	if v, ok := m.Get(name).(string); ok {
		return v
	}
	return ""
}

func (m *config) GetBool(name string) bool {
	if v, ok := m.Get(name).(bool); ok {
		return v
	}
	return false
}

func (m *config) ServerKey() string {
	key := ""
	switch gSvrType {
	case STGate:
		key = "gate_"
	case STGame:
		key = "game_"
	case STDbMgr:
		key = "db_"
	case STRobot:
		key = "robot_"
	case STAdmin:
		key = "admin_"
	}
	return key + strconv.FormatInt(int64(cmdLineMgr.Tag), 10)
}

//ServerConfig 本进程配置
func (m *config) ServerConfig() *serverConfig {
	return m.server
}

func (m *config) GetServerConfigByName(name string) *serverConfig {
	return m.parseServerConfig(name)
}

func (m *config) GetAddr() string {
	return m.server.Addr
}
