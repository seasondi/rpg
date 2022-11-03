package main

import (
	"errors"
	"github.com/panjf2000/ants/v2"
	"time"
)

func newTaskPool() (*taskPool, error) {
	p := new(taskPool)
	if err := p.Init(); err != nil {
		return nil, err
	}
	return p, nil
}

type taskPool struct {
	pool *ants.Pool
}

func (m *taskPool) Init() error {
	opt := func(opts *ants.Options) {
		opts.ExpiryDuration = 10 * time.Minute
		//opts.PreAlloc = true
		opts.PanicHandler = func(i interface{}) {
			log.Error(i)
		}
		opts.Logger = log
	}
	var err error
	m.pool, err = ants.NewPool(100000, opt)
	return err
}

func (m *taskPool) IsAlive() bool {
	return m.pool != nil && m.pool.IsClosed() == false
}

func (m *taskPool) SubmitTask(task Task) error {
	if task == nil {
		return errors.New("task is nil")
	}
	if m.pool == nil {
		return errors.New("task pool is nil")
	}
	if err := m.pool.Submit(task.Process); err != nil {
		return err
	}
	return nil
}

func (m *taskPool) Release() {
	if m.pool != nil {
		m.pool.Release()
	}
}

func (m *taskPool) Running() int {
	if m.pool != nil {
		return m.pool.Running()
	}
	return 0
}
