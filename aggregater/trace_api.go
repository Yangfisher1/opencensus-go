package aggregater

import (
	"context"
	"net/http"

	"github.com/Yangfisher1/opencensus-go/trace"
)

type Tracer interface {
	// We won't have sampler anymore since we want a full tracing
	StartSpan(ctx context.Context, name string, spanKind int) (context.Context, *Span)

	FromContext(ctx context.Context) *Span

	NewContext(parent context.Context, s *Span) context.Context
}

type SpanInterface interface {
	// EndAndAggregate ends the span with response aggregation
	EndAndAggregate(w http.ResponseWriter, r *http.Request)

	// EndAtClient ends the span as a client span and propagation in resp.
	EndAtClient(h *http.Header)

	// SpanContext returns the SpanContext of the span.
	SpanContext() SpanContext

	// SetName sets the name of the span, if it is recording events.
	SetName(name string)

	// SetStatus sets the status of the span, if it is recording events.
	SetStatus(status trace.Status)

	// AddAttributes sets attributes in the span.
	//
	// Existing attributes whose keys appear in the attributes parameter are overwritten.
	AddAttributes(attributes ...trace.Attribute)

	// String prints a string representation of a span.
	String() string

	// TODO: adding more interfaces later
}

// NewSpan is a convenience function for creating a *Span out of a *span
func NewSpan(s SpanInterface) *Span {
	return &Span{internal: s}
}

type Span struct {
	internal SpanInterface
}

func (s *Span) Internal() SpanInterface {
	return s.internal
}

func (s *Span) EndAndAggregate(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		return
	}
	s.internal.EndAndAggregate(w, r)
}

func (s *Span) EndAtClient(h *http.Header) {
	if s == nil {
		return
	}
	s.internal.EndAtClient(h)
}

// SpanContext returns the SpanContext of the span.
func (s *Span) SpanContext() SpanContext {
	if s == nil {
		return SpanContext{}
	}
	return s.internal.SpanContext()
}

// SetName sets the name of the span, if it is recording events.
func (s *Span) SetName(name string) {
	s.internal.SetName(name)
}

// SetStatus sets the status of the span, if it is recording events.
func (s *Span) SetStatus(status trace.Status) {
	s.internal.SetStatus(status)
}

// AddAttributes sets attributes in the span.
//
// Existing attributes whose keys appear in the attributes parameter are overwritten.
func (s *Span) AddAttributes(attributes ...trace.Attribute) {
	s.internal.AddAttributes(attributes...)
}

// String prints a string representation of a span.
func (s *Span) String() string {
	if s == nil {
		return "<nil>"
	}
	return s.internal.String()
}
