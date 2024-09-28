package engine

import (
	"sync"
	"time"
)

type luaFunctionChecker struct {
	name            string
	startTime       time.Time
	timeoutLogged   bool
	deadlineLogTime int64
}

func newLuaChecker() *luaChecker {
	return &luaChecker{
		method:          &luaFunctionChecker{},
		methodTimeout:   200 * time.Millisecond,
		deadlineTimeout: time.Second,
		stopChan:        make(chan bool, 1),
	}
}

type luaChecker struct {
	sync.Mutex
	method          *luaFunctionChecker
	methodTimeout   time.Duration
	deadlineTimeout time.Duration
	stopChan        chan bool
}

func (m *luaChecker) Start() {
	go func() {
		for {
			select {
			case <-m.stopChan:
				log.Info("stop lua checker")
				return
			case <-time.After(100 * time.Millisecond):
				m.doCheck()
			}
		}
	}()
}

func (m *luaChecker) Stop() {
	m.stopChan <- true
}

func (m *luaChecker) doCheck() {
	m.checkMethod()
}

func (m *luaChecker) checkMethod() {
	m.Lock()
	defer m.Unlock()

	if m.method.name == "" {
		return
	}

	expiredTime := m.method.startTime.Add(m.methodTimeout)
	if time.Now().After(expiredTime) {
		expire := time.Since(m.method.startTime)
		if !m.method.timeoutLogged {
			m.method.timeoutLogged = true
			log.Warnf("[Lua Checker] method[%s] timeout, expire: %s", m.method.name, expire.String())
		}

		if expire > m.deadlineTimeout {
			now := time.Now().Unix()
			if now-m.method.deadlineLogTime > 5 {
				log.Errorf("[Lua Checker] method[%s] deadline, expire: %s", m.method.name, expire.String())
				m.method.deadlineLogTime = now
			}
		}
	}
}

func (m *luaChecker) setCheckMethod(name string) {
	m.Lock()
	defer m.Unlock()

	m.method.name = name
	if name != "" {
		m.method.startTime = time.Now()
		m.method.timeoutLogged = false
		m.method.deadlineLogTime = 0
	}
}
