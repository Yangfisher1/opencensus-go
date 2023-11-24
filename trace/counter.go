package trace

import (
	"sync/atomic"
)

type GlobalCounter int64

var (
	// Make them exported
	GeneratedSpanCounter GlobalCounter
	ReportedSpanCounter  GlobalCounter
)

func (c *GlobalCounter) Set(value int64) {
	atomic.StoreInt64((*int64)(c), value)
}

func (c *GlobalCounter) Inc() int64 {
	return atomic.AddInt64((*int64)(c), 1)
}

func (c *GlobalCounter) Get() int64 {
	return atomic.LoadInt64((*int64)(c))
}
