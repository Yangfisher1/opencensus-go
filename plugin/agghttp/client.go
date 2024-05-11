package agghttp

import (
	"net/http"

	"github.com/Yangfisher1/opencensus-go/aggregator/propagation"
)

type Transport struct {
	// Base may be set to wrap another http.RoundTripper that does the actual
	// requests. By default http.DefaultTransport is used.
	//
	// If base HTTP roundtripper implements CancelRequest,
	// the returned round tripper will be cancelable.
	Base http.RoundTripper

	// Propagation defines how traces are propagated. If unspecified, a default
	// (currently B3 format) will be used.
	Propagation propagation.HTTPFormat

	// NameFromRequest holds the function to use for generating the span name
	// from the information found in the outgoing HTTP Request. By default the
	// name equals the URL Path.
	FormatSpanName func(*http.Request) string

	// Whether to treat the span as an aggregation point.
	IsAggregationPoint bool
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.base()
	format := t.Propagation
	if format == nil {
		format = defaultFormat
	}
	spanNameFormatter := t.FormatSpanName
	if spanNameFormatter == nil {
		spanNameFormatter = spanNameFromURL
	}

	rt = &traceTransport{
		base:               rt,
		format:             format,
		formatSpanName:     spanNameFormatter,
		isAggregationPoint: t.IsAggregationPoint,
	}

	return rt.RoundTrip(req)
}

func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// CancelRequest cancels an in-flight request by closing its connection.
func (t *Transport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if cr, ok := t.base().(canceler); ok {
		cr.CancelRequest(req)
	}
}
