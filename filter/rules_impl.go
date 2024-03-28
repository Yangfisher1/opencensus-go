package filter

import (
	"github.com/Yangfisher1/opencensus-go/trace"
)

// Implementation of different filter rules

// Check error codes
func isErrorByCode(span *trace.SpanData, sf *SpanFilter) bool {
	return span.Status.Code != 0
}

// Check error
func isErrorByLatency(span *trace.SpanData, sf *SpanFilter) bool {
	// TODO: Get the component name from the span name
	duration := span.EndTime.Sub(span.StartTime)
	avgLat, exists := sf.avgLantecies[span.Name]
	if !exists {
		return false // FIXME: Just assume its normal maybe
	}

	// Check if the latency is abnormal
	if float64(duration) > avgLat*(1+sf.deviation) || float64(duration) < avgLat*(1-sf.deviation) {
		return true
	}

	return false

}
