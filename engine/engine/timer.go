package engine

import (
	"rpg/engine/engine/spinLock"
	"rpg/engine/engine/timerWheel"
	lua "github.com/yuin/gopher-lua"
	"time"
)

type timerMgr struct {
	sl       spinLock.SpinLock
	tw       *timerWheel.TimerWheel
	timerMap map[int64]*timerWheel.Timer
}

func initTimer() error {
	timer = &timerMgr{}
	if err := timer.init(); err != nil {
		return err
	}
	log.Infof("timer inited.")
	return nil
}

func GetTimer() *timerMgr {
	return timer
}

func (m *timerMgr) init() error {
	var err error
	if m.tw, err = timerWheel.NewTimerWheel(ServerTick, int64(10*time.Minute/ServerTick)); err != nil {
		return err
	}
	m.timerMap = make(map[int64]*timerWheel.Timer)
	return nil
}

func (m *timerMgr) close() {
}

func (m *timerMgr) HandleMainTick() {
	m.tw.HandleMainTick(time.Now().Add(time.Duration(GetTimeOffset()) * time.Second))
}

/*AddTimer 添加定时器,精度0.1秒

d: 定时器首次触发间隔(毫秒)

repeatDuration: 大于0, 循环触发间隔

cb: 回调函数

params: 回调参数

返回值：定时器ID
*/
func (m *timerMgr) AddTimer(d time.Duration, repeatDuration time.Duration, cb func(params ...interface{}), params ...interface{}) int64 {
	m.sl.Lock()
	defer m.sl.UnLock()
	var tm *timerWheel.Timer
	if repeatDuration > 0 {
		tm = m.tw.Repeat(d, repeatDuration, cb, params...)
	} else {
		tm = m.tw.After(d, cb, params...)
	}
	m.timerMap[tm.TimerID()] = tm
	log.Debugf("add timer, id: %d, expiration: %v", tm.TimerID(), timerWheel.MsToTime(tm.Expiration()))
	return tm.TimerID()
}

func (m *timerMgr) Cancel(timerId int64) {
	m.sl.Lock()
	defer m.sl.UnLock()
	if tm, ok := m.timerMap[timerId]; ok {
		tm.Stop()
		delete(m.timerMap, timerId)
		log.Debugf("cancel timer id: %d", timerId)
	}
}

func (m *timerMgr) ShowTimeWheelInfo() {
	log.Info(m.tw.String())
}

//脚本层添加的entity定时器回调触发
func entityScriptTimerCallback(params ...interface{}) {
	//1: entityId
	//2: lua function name
	//3-n: params(last is timerId)
	if len(params) < 3 {
		log.Warnf("Entity timer timeout, args not enough, len: %d", len(params))
		return
	}
	ent := GetEntityManager().GetEntityById(params[0].(EntityIdType))
	if ent == nil {
		return
	}
	//回调触发后将entity身上记录的信息移除
	ent.removeActiveTimerId(params[len(params)-1].(int64))

	methodName := params[1].(lua.LString)
	argLen := len(params) - 2
	args := make([]lua.LValue, argLen, argLen)
	args[0] = ent.luaEntity
	for i := 2; i < len(params)-1; i++ {
		args[i-1] = params[i].(lua.LValue)
	}
	if err := CallLuaMethodByName(ent.luaEntity, methodName.String(), 0, args...); err != nil {
		log.Errorf("%s timer callback, error: %s", ent.String(), err.Error())
	}
}

func SetTimeOffset(offset int32) bool {
	if GetConfig().Release {
		log.Errorf("update server time is forbidden in release")
		return false
	}
	timeOffset = offset
	tm := time.Now().Add(time.Duration(timeOffset) * time.Second).Format("2006-01-02 15:04:05")
	if err := CallLuaMethodByName(GetGlobalEntry(), onServerTimeUpdate, 0, lua.LString(tm)); err != nil {
		log.Warnf("set lua time error: %s", err.Error())
		timeOffset = 0
		return false
	}
	log.Infof("set server time to %s", time.Unix(time.Now().Unix()+int64(offset), 0).Format("2006-01-02 15:04:05"))
	return true
}

func GetTimeOffset() int32 {
	return timeOffset
}
