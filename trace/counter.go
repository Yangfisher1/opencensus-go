package trace

import (
	"sync/atomic"
)

type globalCounter int64

var (
	generatedSpanCounter globalCounter
	reportedSpanCounter  globalCounter
)

func (c *globalCounter) set(value int64) {
	atomic.StoreInt64((*int64)(c), value)
}

func (c *globalCounter) inc() int64 {
	return atomic.AddInt64((*int64)(c), 1)
}

func (c *globalCounter) get() int64 {
	return atomic.LoadInt64((*int64)(c))
}
