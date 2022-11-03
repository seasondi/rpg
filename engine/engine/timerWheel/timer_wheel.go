package timerWheel

import (
	"errors"
	"fmt"
	"time"
)

var TWHandler *TimerWheel

type TimerWheel struct {
	tick          int64           //精度(毫秒)
	wheelSize     int64           //格子数量
	interval      int64           //容量(毫秒)
	startMs       int64           //启动时间(毫秒)
	buckets       map[int]*Bucket //包含的桶
	queue         *PriorityQueue  //优先队列,用于触发到时间的桶
	overflowWheel *TimerWheel     //时间最新的下个时间轮
}

func NewTimerWheel(tick time.Duration, wheelSize int64) (*TimerWheel, error) {
	tickMs := int64(tick / time.Millisecond)
	if tickMs <= 0 {
		return nil, errors.New("tick must be greater than or equal to 1ms")
	}
	if TWHandler != nil {
		return nil, errors.New("timer wheel already inited")
	}

	now := TimeToMs(time.Now().UTC())
	TWHandler = newTimerWheel(
		tickMs,
		wheelSize,
		now,
		NewQueue(int(wheelSize)),
	)
	return TWHandler, nil
}

func newTimerWheel(tickMs int64, wheelSize int64, startMs int64, queue *PriorityQueue) *TimerWheel {
	return &TimerWheel{
		tick:      tickMs,
		wheelSize: wheelSize,
		interval:  tickMs * wheelSize,
		startMs:   startMs,
		buckets:   make(map[int]*Bucket),
		queue:     queue,
	}
}

func (tw *TimerWheel) add(t *Timer) {
	//already expired timer add to next tick
	if t.info.Expiration < tw.startMs {
		t.info.Expiration = tw.startMs
	}
	if t.info.Expiration < tw.startMs+tw.interval {
		index := int((t.info.Expiration - tw.startMs) / tw.tick)
		if _, ok := tw.buckets[index]; !ok {
			tw.buckets[index] = newBucket(tw.startMs + int64(index)*tw.tick)
		}
		b := tw.buckets[index]
		b.Add(t)
		if !b.IsInQueue() {
			tw.queue.Add(b, timeToPrecision(b.Expiration(), tw.tick))
		}
	} else {
		if tw.overflowWheel != nil {
			if t.info.Expiration < tw.overflowWheel.startMs {
				newWheel := newTimerWheel(
					tw.tick,
					tw.wheelSize,
					truncate(tw.startMs, t.info.Expiration, tw.interval),
					tw.queue,
				)
				newWheel.overflowWheel = tw.overflowWheel
				tw.overflowWheel = newWheel
			}
		} else {
			tw.overflowWheel = newTimerWheel(
				tw.tick,
				tw.wheelSize,
				truncate(tw.startMs, t.info.Expiration, tw.interval),
				tw.queue,
			)
		}
		tw.overflowWheel.add(t)
	}
}

func (tw *TimerWheel) onTimeout(t *Timer) {
	//bucket为nil表明定时器已被提前取消了
	if t.getBucket() == nil {
		return
	}
	t.info.Callback(t.info.Params...)
	//对于循环定时器,回调函数中可能会移除该定时器, 回调之后需要再次检查一次bucket
	if t.info.RepeatDuration > 0 && t.getBucket() != nil {
		t.info.Expiration += t.info.RepeatDuration.Milliseconds()
		//t.info.Expiration = TimeToMs(time.Now().UTC().Add(t.RepeatDuration))
		tw.add(t)
	} else {
		t.Stop()
	}
}

func (tw *TimerWheel) addTimer(duration time.Duration, repeatDuration time.Duration, f func(p ...interface{}), params ...interface{}) *Timer {
	currentTime := TimeToMs(time.Now().UTC())
	t := &Timer{
		timerID: nextTimerID(),
		info:    NewTimerInfo(currentTime+duration.Milliseconds(), repeatDuration, f, params),
	}
	t.info.Params = append(t.info.Params, t.timerID)
	tw.add(t)
	return t
}

func (tw *TimerWheel) onTimeOut(currentTime int64) {
	expiredBuckets := tw.queue.GetExpiredAndShift(currentTime)
	if len(expiredBuckets) == 0 {
		return
	}
	for _, bucket := range expiredBuckets {
		timers := bucket.GetTimers()
		for _, timer := range timers {
			tw.onTimeout(timer)
		}
	}
}

func (tw *TimerWheel) isExpired(currentTime int64) bool {
	return currentTime >= tw.startMs+tw.interval
}

func (tw *TimerWheel) After(duration time.Duration, f func(...interface{}), params ...interface{}) *Timer {
	return tw.addTimer(duration, 0, f, params...)
}

func (tw *TimerWheel) Repeat(duration time.Duration, repeatDuration time.Duration, f func(...interface{}), params ...interface{}) *Timer {
	return tw.addTimer(duration, repeatDuration, f, params...)
}

func (tw *TimerWheel) HandleMainTick(now time.Time) {
	currentTime := TimeToMs(now.UTC())

	TWHandler.onTimeOut(currentTime)
	for expired := TWHandler.isExpired(currentTime); expired == true; {
		if TWHandler.overflowWheel == nil {
			break
		}
		//overflow时间轮可能比当前时间大,只有时间运行到overflow时间轮时才能切换
		if currentTime >= TWHandler.overflowWheel.startMs {
			TWHandler = TWHandler.overflowWheel
			TWHandler.onTimeOut(currentTime)
		} else {
			break
		}
	}
}

func (tw *TimerWheel) String() string {
	return fmt.Sprintf("当前时间轮信息: 初始时间(毫秒): %d, 支持格子数量: %d, 当前格子数量: %d, 精度(毫秒): %d, 容量(毫秒): %d",
		tw.startMs, tw.wheelSize, len(tw.buckets), tw.tick, tw.interval)
}
