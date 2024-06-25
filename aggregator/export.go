package aggregator

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type ErrorType int

// Error types for filtering to use
const (
	OK ErrorType = iota
	Aggregate
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
	// Filter whether the span is normal or not
	FilterSpan(s *SpanData) ErrorType
	// Report the whole trace chain from the header(whether its request or response)
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
	ParentSpanId SpanID
	SpanKind     int
	Name         string
	StartTime    time.Time
	// The wall clock time of EndTime will be adjusted to always be offset
	// from StartTime by the duration of the span.
	EndTime time.Time
	// The values of Attributes each have type string, bool, or int64.
	Attributes map[string]interface{}
	Status
	// TODO: adding more fields later
}

// Only contains important information
type NormalSpanData struct {
	SpanID    ID     `json:"s"`
	ParentID  ID     `json:"p"`
	Kind      int    `json:"k"`
	Name      string `json:"n"`
	StartTime string `json:"t"`
	Duration  string `json:"d"`
}

// Version 2 of the spandata with fewer information
type SpanDataV2 struct {
	Height    int    `json:h`
	Name      string `json:n`
	StartTime string `json:t`
	Duration  string `json:d`
}
