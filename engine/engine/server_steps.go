package engine

type ServerStepType int

const (
	ServerStepNone ServerStepType = iota
	ServerStepPrepare
	ServerStepInitScript
	ServerStepCompleted
)

type ServerStepHandler func()

func GetServerStep() *ServerStep {
	if svrStep == nil {
		svrStep = &ServerStep{
			step:     ServerStepNone,
			handlers: make(map[ServerStepType]map[string]ServerStepHandler),
		}
	}
	return svrStep
}

type ServerStep struct {
	step     ServerStepType
	handlers map[ServerStepType]map[string]ServerStepHandler
}

func (m *ServerStep) Step() ServerStepType {
	return m.step
}

func (m *ServerStep) Register(step ServerStepType, idx string, handler ServerStepHandler) {
	if step == ServerStepNone || step == ServerStepCompleted {
		log.Error("cannot register handler for step: ", step)
		return
	}
	if _, ok := m.handlers[step]; !ok {
		m.handlers[step] = make(map[string]ServerStepHandler)
	}
	m.handlers[step][idx] = handler
}

func (m *ServerStep) FinishHandler(name string) {
	if len(m.handlers[m.step]) > 0 {
		delete(m.handlers[m.step], name)
	}
	if len(m.handlers[m.step]) == 0 {
		m.enterNextStep()
	}
}

func (m *ServerStep) Start() {
	m.doStepHandler()
}

func (m *ServerStep) Completed() bool {
	return m.step == ServerStepCompleted
}

func (m *ServerStep) Print() {
	if m.Completed() {
		log.Info("ALL COMPLETED")
	}
	s := "current step " + m.StepName() + " still wait for ["
	if handlers, ok := m.handlers[m.step]; ok {
		for name := range handlers {
			s += name + ", "
		}
	}
	s += "]"
	log.Info(s)
}

func (m *ServerStep) doStepHandler() {
	if handlers, ok := m.handlers[m.step]; ok {
		for _, handler := range handlers {
			if handler != nil {
				handler()
			}
		}
	} else {
		m.enterNextStep()
	}
}

func (m *ServerStep) enterNextStep() {
	if m.Completed() {
		return
	}
	m.step += 1
	log.Infof("===============SERVER INIT ENTER STEP [%s]==============", m.StepName())
	m.doStepHandler()
}

func (m *ServerStep) StepName() string {
	switch m.step {
	case ServerStepNone:
		return "NONE"
	case ServerStepPrepare:
		return "PREPARE"
	case ServerStepInitScript:
		return "INIT_SCRIPT"
	case ServerStepCompleted:
		return "COMPLETED"
	}
	return "UNKNOWN"
}
