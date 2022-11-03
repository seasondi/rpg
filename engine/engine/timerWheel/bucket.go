package timerWheel

import (
	"container/list"
	"time"
)

type TimerInfo struct {
	Expiration     int64
	RepeatDuration time.Duration
	Callback       func(...interface{})
	Params         []interface{}
}

func NewTimerInfo(expiration int64, repeatDuration time.Duration, cb func(...interface{}), params []interface{}) *TimerInfo {
	return &TimerInfo{
		Expiration:     expiration,
		RepeatDuration: repeatDuration,
		Callback:       cb,
		Params:         params,
	}
}

type Timer struct {
	timerID int64
	info    *TimerInfo
	bucket  *Bucket
	element *list.Element
}

func (t *Timer) getBucket() *Bucket {
	return t.bucket
}

func (t *Timer) setBucket(b *Bucket) {
	t.bucket = b
}

func (t *Timer) TimerID() int64 {
	return t.timerID
}

func (t *Timer) Expiration() int64 {
	return t.info.Expiration
}

func (t *Timer) RepeatDuration() time.Duration {
	return t.info.RepeatDuration
}

func (t *Timer) Stop() {
	for b := t.getBucket(); b != nil; b = t.getBucket() {
		b.Remove(t)
	}
}

type Bucket struct {
	expiration int64
	timers     *list.List
	isInQueue  bool
}

func newBucket(expiration int64) *Bucket {
	return &Bucket{
		timers:     list.New(),
		expiration: expiration,
	}
}

func (b *Bucket) remove(t *Timer) bool {
	if t.getBucket() != b {
		return false
	}
	b.timers.Remove(t.element)
	t.setBucket(nil)
	t.element = nil
	return true
}

func (b *Bucket) GetTimers() []*Timer {
	r := make([]*Timer, 0, b.timers.Len())
	for e := b.timers.Front(); e != nil; e = e.Next() {
		r = append(r, e.Value.(*Timer))
	}
	return r
}

func (b *Bucket) Add(t *Timer) {
	e := b.timers.PushBack(t)
	t.setBucket(b)
	t.element = e
}

func (b *Bucket) Remove(t *Timer) bool {
	return b.remove(t)
}

func (b *Bucket) Expiration() int64 {
	return b.expiration
}

func (b *Bucket) IsInQueue() bool {
	return b.isInQueue
}

func (b *Bucket) SetIsInQueue(in bool) {
	b.isInQueue = in
}
