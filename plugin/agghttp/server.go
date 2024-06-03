package agghttp

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Yangfisher1/opencensus-go/aggregator"
	"github.com/Yangfisher1/opencensus-go/aggregator/propagation"
	"github.com/Yangfisher1/opencensus-go/stats"
	"github.com/Yangfisher1/opencensus-go/tag"
)

type Handler struct {
	Propagation        propagation.HTTPFormat
	Handler            http.Handler
	FormatSpanName     func(*http.Request) string
	IsAggregationPoint bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var tags addedTags
	// Adding the trailer headers here
	w.Header().Set("Trailer", "Agg")
	w.Header().Set("Transfer-Encoding", "chunked")

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
		// Ok, we have a span from the remote requests
		ctx, span = aggregator.StartSpanWithRemoteParent(ctx, name, sc, aggregator.SpanKindServer)
	} else {
		// Start a local span by itself
		ctx, span = aggregator.StartSpan(ctx, name, aggregator.SpanKindServer)
	}

	span.AddAttributes(requestAttrs(r)...)

	if h.IsAggregationPoint {
		attrs := aggregator.StringAttribute("agg", "y")
		span.AddAttributes(attrs)
	}

	// TODO: msgreceiveEvent?
	return r.WithContext(ctx), span.EndAndAggregate
}

func (h *Handler) extractSpanContext(r *http.Request) (aggregator.SpanContext, bool) {
	if h.Propagation == nil {
		return defaultFormat.SpanContextFromRequest(r)
	}
	return h.Propagation.SpanContextFromRequest(r)
}

func (h *Handler) startStats(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, func(tags *addedTags)) {
	ctx, _ := tag.New(r.Context(),
		tag.Upsert(Host, r.Host),
		tag.Upsert(Path, r.URL.Path),
		tag.Upsert(Method, r.Method))
	track := &trackingResponseWriter{
		start:  time.Now(),
		ctx:    ctx,
		writer: w,
	}
	if r.Body == nil {
		// TODO: Handle cases where ContentLength is not set.
		track.reqSize = -1
	} else if r.ContentLength > 0 {
		track.reqSize = r.ContentLength
	}
	stats.Record(ctx, ServerRequestCount.M(1))
	return track.wrappedResponseWriter(), track.end
}

type trackingResponseWriter struct {
	ctx        context.Context
	reqSize    int64
	respSize   int64
	start      time.Time
	statusCode int
	statusLine string
	endOnce    sync.Once
	writer     http.ResponseWriter
}

// Compile time assertion for ResponseWriter interface
var _ http.ResponseWriter = (*trackingResponseWriter)(nil)

func (t *trackingResponseWriter) end(tags *addedTags) {
	t.endOnce.Do(func() {
		if t.statusCode == 0 {
			t.statusCode = 200
		}

		span := aggregator.FromContext(t.ctx)
		span.SetStatus(TraceStatus(t.statusCode, t.statusLine))
		span.AddAttributes(aggregator.Int64Attribute(StatusCodeAttribute, int64(t.statusCode)))
	})
}

func (t *trackingResponseWriter) Header() http.Header {
	return t.writer.Header()
}

func (t *trackingResponseWriter) Write(data []byte) (int, error) {
	n, err := t.writer.Write(data)
	t.respSize += int64(n)
	// Add message event for request bytes sent.
	// TODO: add msg event
	// span := aggregator.FromContext(t.ctx)
	// span.AddMessageSendEvent(0 /* TODO: messageID */, int64(n), -1)
	return n, err
}

func (t *trackingResponseWriter) WriteHeader(statusCode int) {
	t.writer.WriteHeader(statusCode)
	t.statusCode = statusCode
	t.statusLine = http.StatusText(t.statusCode)
}

// wrappedResponseWriter returns a wrapped version of the original
//
//	ResponseWriter and only implements the same combination of additional
//
// interfaces as the original.
// This implementation is based on https://github.com/felixge/httpsnoop.
func (t *trackingResponseWriter) wrappedResponseWriter() http.ResponseWriter {
	var (
		hj, i0 = t.writer.(http.Hijacker)
		cn, i1 = t.writer.(http.CloseNotifier)
		pu, i2 = t.writer.(http.Pusher)
		fl, i3 = t.writer.(http.Flusher)
		rf, i4 = t.writer.(io.ReaderFrom)
	)

	switch {
	case !i0 && !i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
		}{t}
	case !i0 && !i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			io.ReaderFrom
		}{t, rf}
	case !i0 && !i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Flusher
		}{t, fl}
	case !i0 && !i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{t, fl, rf}
	case !i0 && !i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
		}{t, pu}
	case !i0 && !i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			io.ReaderFrom
		}{t, pu, rf}
	case !i0 && !i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
		}{t, pu, fl}
	case !i0 && !i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, pu, fl, rf}
	case !i0 && i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
		}{t, cn}
	case !i0 && i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			io.ReaderFrom
		}{t, cn, rf}
	case !i0 && i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
		}{t, cn, fl}
	case !i0 && i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{t, cn, fl, rf}
	case !i0 && i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
		}{t, cn, pu}
	case !i0 && i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{t, cn, pu, rf}
	case !i0 && i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			http.Flusher
		}{t, cn, pu, fl}
	case !i0 && i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, cn, pu, fl, rf}
	case i0 && !i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
		}{t, hj}
	case i0 && !i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{t, hj, rf}
	case i0 && !i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
		}{t, hj, fl}
	case i0 && !i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Flusher
			io.ReaderFrom
		}{t, hj, fl, rf}
	case i0 && !i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
		}{t, hj, pu}
	case i0 && !i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{t, hj, pu, rf}
	case i0 && !i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
		}{t, hj, pu, fl}
	case i0 && !i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, hj, pu, fl, rf}
	case i0 && i1 && !i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
		}{t, hj, cn}
	case i0 && i1 && !i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			io.ReaderFrom
		}{t, hj, cn, rf}
	case i0 && i1 && !i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Flusher
		}{t, hj, cn, fl}
	case i0 && i1 && !i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{t, hj, cn, fl, rf}
	case i0 && i1 && i2 && !i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
		}{t, hj, cn, pu}
	case i0 && i1 && i2 && !i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{t, hj, cn, pu, rf}
	case i0 && i1 && i2 && i3 && !i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			http.Flusher
		}{t, hj, cn, pu, fl}
	case i0 && i1 && i2 && i3 && i4:
		return struct {
			http.ResponseWriter
			http.Hijacker
			http.CloseNotifier
			http.Pusher
			http.Flusher
			io.ReaderFrom
		}{t, hj, cn, pu, fl, rf}
	default:
		return struct {
			http.ResponseWriter
		}{t}
	}
}
