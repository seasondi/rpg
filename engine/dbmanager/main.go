package main

import (
	"rpg/engine/engine"
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/sirupsen/logrus"
	"time"
)

var (
	log   *logrus.Entry
	dbMgr *DBMgr
)

type DBMgr struct {
	TaskPool *taskPool
	TaskMgr  *bufferedTasks
	db       *mongoClient
	commonDB *mongoClient
}

func (m *DBMgr) GetDB(dbType engine.DBType) *mongoClient {
	switch dbType {
	case engine.DBTypeCommon:
		return m.commonDB
	case engine.DBTypeProject:
		return m.db
	default:
		return nil
	}
}

func InitDBManager() (*DBMgr, error) {
	log.Info("============DBManager Init=======================")
	mgr := &DBMgr{}
	var err error
	if mgr.TaskPool, err = newTaskPool(); err != nil {
		return nil, err
	}
	if mgr.TaskMgr, err = newBufferedTasks(); err != nil {
		return nil, err
	}

	//连接common库
	if commonUri := engine.GetConfig().GetString("mongo_common.uri"); commonUri != "" {
		if mgr.commonDB, err = newMongoServer(); err != nil {
			return nil, err
		}
		if err = mgr.commonDB.Connect(commonUri); err != nil {
			return nil, err
		} else {
			log.Info("connect to common mongo success")
		}
	}
	//连接项目库
	dbConfigName := engine.GetConfig().ServerConfig().Database
	if uri := engine.GetConfig().GetString(dbConfigName + ".uri"); uri != "" {
		if mgr.db, err = newMongoServer(); err != nil {
			return nil, err
		}
		if err = mgr.db.Connect(uri); err != nil {
			return nil, err
		} else {
			log.Info("connect to project mongo success")
		}
	} else {
		return nil, fmt.Errorf("not found mongodb uri config for %s", engine.GetConfig().ServerKey())
	}

	log.Infof("============DBManager Inited=======================")
	return mgr, nil
}

func main() {
	var err error
	if err = engine.Init(engine.STDbMgr); err != nil {
		fmt.Println("[DBManager] init engine failed, error: ", err.Error())
		return
	}
	defer engine.Close()
	initSysSignalMgr()

	log = engine.GetLogger()
	if dbMgr, err = InitDBManager(); err != nil {
		log.Error("InitDBManager failed, error: ", err.Error())
		return
	}

	_ = gnet.Serve(getEventLoop(), engine.ListenProtoAddr(),
		gnet.WithTicker(true),
		gnet.WithCodec(&engine.GNetCodec{}),
		gnet.WithTCPKeepAlive(3*time.Second),
		gnet.WithLogger(log.Logger),
	)
}
