package agghttp

import (
	"context"
	"net/http"

	"github.com/Yangfisher1/opencensus-go/aggregator"
	"github.com/Yangfisher1/opencensus-go/aggregator/propagation"
)

type Handler struct {
	Propagation    propagation.HTTPFormat
	Handler        http.Handler
	FormatSpanName func(*http.Request) string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var tags addedTags
	// Adding the trailer headers here
	w.Header().Set("Trailer", "Agg")

	r, traceEnd := h.startTrace(w, r)
	defer traceEnd(w, r)
	w, statsEnd := h.startStats(w, r)
	defer statsEnd(&tags)
	handler := h.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	r = r.WithContext(context.WithValue(r.Context(), addedTagsKey{}, &tags))
	handler.ServeHTTP(w, r)
}

func (h *Handler) startTrace(w http.ResponseWriter, r *http.Request) (*http.Request, func(http.ResponseWriter, *http.Request)) {
	var name string
	if h.FormatSpanName == nil {
		name = spanNameFromURL(r)
	} else {
		name = h.FormatSpanName(r)
	}
	ctx := r.Context()

	var span *aggregator.Span
	sc, ok := h.extractSpanContext(r)
	if ok {
		// So we have a parent indeeded
		// TODO: how to propagate from `sc`?
		ctx, span = aggregator.StartSpan(ctx, name, aggregator.SpanKindServer)
	}

	// if ok && !h.IsPublicEndpoint {
	// 	ctx, span = trace.StartSpanWithRemoteParent(ctx, name, sc,
	// 		trace.WithSampler(startOpts.Sampler),
	// 		trace.WithSpanKind(trace.SpanKindServer))
	// } else {
	// 	ctx, span = trace.StartSpan(ctx, name,
	// 		trace.WithSampler(startOpts.Sampler),
	// 		trace.WithSpanKind(trace.SpanKindServer),
	// 	)
	// 	if ok {
	// 		span.AddLink(trace.Link{
	// 			TraceID:    sc.TraceID,
	// 			SpanID:     sc.SpanID,
	// 			Type:       trace.LinkTypeParent,
	// 			Attributes: nil,
	// 		})
	// 	}
	// }
	span.AddAttributes(requestAttrs(r)...)
	return r.WithContext(ctx), span.EndAndAggregate
}

func (h *Handler) extractSpanContext(r *http.Request) (aggregator.SpanContext, bool) {
	if h.Propagation == nil {
		return defaultFormat.SpanContextFromRequest(r)
	}
	return h.Propagation.SpanContextFromRequest(r)
}
