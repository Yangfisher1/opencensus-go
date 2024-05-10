package aggregator

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Yangfisher1/opencensus-go/trace"
)

type ErrorType int

// Error types for filtering to use
const (
	OK ErrorType = iota
	Aggregate
	UserSpec
	PerformanceDown
	Error
)

// Exporter is a type for functions that receive sampled trace spans.
//
// The ExportSpan method should be safe for concurrent use and should return
// quickly; if an Exporter takes a significant amount of time to process a
// SpanData, that work should be done on another goroutine.
//
// The SpanData should not be modified, but a pointer to it can be kept.
type Exporter interface {
	ExportSpan(s *SpanData)
	// Filter whether span is valid or not
	FilterSpan(s *SpanData) ErrorType
	AggregateSpanFromHeader(w http.Header)
}

type exportersMap map[Exporter]struct{}

var (
	exporterMu sync.Mutex
	exporters  atomic.Value
)

// RegisterExporter adds to the list of Exporters that will receive sampled
// trace spans.
//
// Binaries can register exporters, libraries shouldn't register exporters.
func RegisterExporter(e Exporter) {
	exporterMu.Lock()
	new := make(exportersMap)
	if old, ok := exporters.Load().(exportersMap); ok {
		for k, v := range old {
			new[k] = v
		}
	}
	new[e] = struct{}{}
	exporters.Store(new)
	exporterMu.Unlock()
}

// UnregisterExporter removes from the list of Exporters the Exporter that was
// registered with the given name.
func UnregisterExporter(e Exporter) {
	exporterMu.Lock()
	new := make(exportersMap)
	if old, ok := exporters.Load().(exportersMap); ok {
		for k, v := range old {
			new[k] = v
		}
	}
	delete(new, e)
	exporters.Store(new)
	exporterMu.Unlock()
}

// SpanData contains all the information collected by a Span.
type SpanData struct {
	SpanContext
	SpanKind  int
	Name      string
	StartTime time.Time
	// The wall clock time of EndTime will be adjusted to always be offset
	// from StartTime by the duration of the span.
	EndTime time.Time
	// The values of Attributes each have type string, bool, or int64.
	Attributes map[string]interface{}
	trace.Status
	// TODO: adding more fields later
}

// Only contains important information
type NormalSpanData struct {
	Height    uint32 `json:"h"`
	Kind      int    `json:"k"`
	Name      string `json:"n"`
	StartTime string `json:"t"`
	Duration  string `json:"d"`
}
