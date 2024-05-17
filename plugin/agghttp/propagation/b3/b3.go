package b3

import (
	"encoding/hex"
	"net/http"

	"github.com/Yangfisher1/opencensus-go/aggregator"
	"github.com/Yangfisher1/opencensus-go/aggregator/propagation"
)

const (
	SpanIDHeader = "X-B3-SpanId"
)

type HTTPFormat struct{}

var _ propagation.HTTPFormat = (*HTTPFormat)(nil)

// SpanContextFromRequest extracts a B3 span context from incoming requests.
func (f *HTTPFormat) SpanContextFromRequest(req *http.Request) (sc aggregator.SpanContext, ok bool) {
	sid, ok := ParseSpanID(req.Header.Get(SpanIDHeader))
	if !ok {
		return aggregator.SpanContext{}, false
	}

	return aggregator.SpanContext{
		SpanID: sid,
	}, true
}

// ParseSpanID parses the value of the X-B3-SpanId or X-B3-ParentSpanId headers.
func ParseSpanID(sid string) (spanID aggregator.SpanID, ok bool) {
	if sid == "" {
		return aggregator.SpanID{}, false
	}
	b, err := hex.DecodeString(sid)
	if err != nil || len(b) > 8 {
		return aggregator.SpanID{}, false
	}
	start := 8 - len(b)
	copy(spanID[start:], b)
	return spanID, true
}

func (f *HTTPFormat) SpanContextToRequest(sc aggregator.SpanContext, req *http.Request) {
	req.Header.Set(SpanIDHeader, hex.EncodeToString(sc.SpanID[:]))
}
