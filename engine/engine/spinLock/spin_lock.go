package spinLock

import (
	"runtime"
	"sync/atomic"
)

type SpinLock int32

func (m *SpinLock) Lock() {
	for !atomic.CompareAndSwapInt32((*int32)(m), 0, 1) {
		runtime.Gosched()
	}
}

func (m *SpinLock) UnLock() {
	atomic.StoreInt32((*int32)(m), 0)
}
