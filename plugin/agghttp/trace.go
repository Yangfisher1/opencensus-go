package agghttp

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Yangfisher1/opencensus-go/aggregator"
	"github.com/Yangfisher1/opencensus-go/aggregator/propagation"
	"github.com/Yangfisher1/opencensus-go/plugin/agghttp/propagation/b3"
	"github.com/Yangfisher1/opencensus-go/trace"
)

var defaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}

// Attributes recorded on the span for the requests.
// Only trace exporters will need them.
const (
	HostAttribute       = "http.host"
	MethodAttribute     = "http.method"
	PathAttribute       = "http.path"
	URLAttribute        = "http.url"
	UserAgentAttribute  = "http.user_agent"
	StatusCodeAttribute = "http.status_code"
)

type traceTransport struct {
	base               http.RoundTripper
	format             propagation.HTTPFormat
	formatSpanName     func(*http.Request) string
	isAggregationPoint bool
}

func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	name := t.formatSpanName(req)
	ctx, span := aggregator.StartSpan(req.Context(), name, aggregator.SpanKindClient)
	req = req.WithContext(ctx)

	if t.format != nil {
		// SpanContextToRequest will modify its Request argument, which is
		// contrary to the contract for http.RoundTripper, so we need to
		// pass it a copy of the Request.
		// However, the Request struct itself was already copied by
		// the WithContext calls above and so we just need to copy the header.
		header := make(http.Header)
		for k, v := range req.Header {
			header[k] = v
		}
		req.Header = header
		t.format.SpanContextToRequest(span.SpanContext(), req)
	}

	span.AddAttributes(requestAttrs(req)...)

	if t.isAggregationPoint {
		attrs := aggregator.StringAttribute("agg", "y")
		span.AddAttributes(attrs)
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		panic(err)
	}

	// Prevent overloading of trailer header
	if resp.Trailer != nil {
		fmt.Println(resp.Trailer)
	} else {
		resp.Trailer = make(http.Header)
	}

	span.AddAttributes(responseAttrs(resp)...)
	span.SetStatus(TraceStatus(resp.StatusCode, resp.Status))
	bt := &bodyTracker{rc: resp.Body, span: span, trailer: &resp.Trailer}
	resp.Body = wrappedBody(bt, resp.Body)
	return resp, err
}

// bodyTracker wraps a response.Body and invokes
// trace.EndSpan on encountering io.EOF on reading
// the body of the original response.
type bodyTracker struct {
	rc      io.ReadCloser
	span    *aggregator.Span
	trailer *http.Header
}

var _ io.ReadCloser = (*bodyTracker)(nil)

func (bt *bodyTracker) Read(b []byte) (int, error) {
	n, err := bt.rc.Read(b)
	// n, err := bt.rc.Read(b)
	switch err {
	case nil:
		return n, nil
	case io.EOF:
		bt.span.EndAtClient(bt.trailer)
	default:
		// For all other errors, set the span status
		bt.span.SetStatus(aggregator.Status{
			// Code 2 is the error code for Internal server error.
			Code:    2,
			Message: err.Error(),
		})
	}
	return n, err
}

func (bt *bodyTracker) Close() error {
	// Invoking endSpan on Close will help catch the cases
	// in which a read returned a non-nil error, we set the
	// span status but didn't end the span.
	bt.span.EndAtClient(bt.trailer)
	return bt.rc.Close()
}

// CancelRequest cancels an in-flight request by closing its connection.
func (t *traceTransport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if cr, ok := t.base.(canceler); ok {
		cr.CancelRequest(req)
	}
}

func spanNameFromURL(req *http.Request) string {
	return req.URL.Path
}

func responseAttrs(resp *http.Response) []aggregator.Attribute {
	return []aggregator.Attribute{
		aggregator.Int64Attribute(StatusCodeAttribute, int64(resp.StatusCode)),
	}
}

func requestAttrs(r *http.Request) []aggregator.Attribute {
	userAgent := r.UserAgent()

	attrs := make([]aggregator.Attribute, 0, 5)
	attrs = append(attrs,
		aggregator.StringAttribute(PathAttribute, r.URL.Path),
		aggregator.StringAttribute(URLAttribute, r.URL.String()),
		aggregator.StringAttribute(HostAttribute, r.Host),
		aggregator.StringAttribute(MethodAttribute, r.Method),
	)

	if userAgent != "" {
		attrs = append(attrs, aggregator.StringAttribute(UserAgentAttribute, userAgent))
	}

	return attrs
}

// TraceStatus is a utility to convert the HTTP status code to a trace.Status that
// represents the outcome as closely as possible.
func TraceStatus(httpStatusCode int, statusLine string) aggregator.Status {
	var code int32
	if httpStatusCode < 200 || httpStatusCode >= 400 {
		code = trace.StatusCodeUnknown
	}
	switch httpStatusCode {
	case 499:
		code = trace.StatusCodeCancelled
	case http.StatusBadRequest:
		code = trace.StatusCodeInvalidArgument
	case http.StatusUnprocessableEntity:
		code = trace.StatusCodeInvalidArgument
	case http.StatusGatewayTimeout:
		code = trace.StatusCodeDeadlineExceeded
	case http.StatusNotFound:
		code = trace.StatusCodeNotFound
	case http.StatusForbidden:
		code = trace.StatusCodePermissionDenied
	case http.StatusUnauthorized: // 401 is actually unauthenticated.
		code = trace.StatusCodeUnauthenticated
	case http.StatusTooManyRequests:
		code = trace.StatusCodeResourceExhausted
	case http.StatusNotImplemented:
		code = trace.StatusCodeUnimplemented
	case http.StatusServiceUnavailable:
		code = trace.StatusCodeUnavailable
	case http.StatusOK:
		code = trace.StatusCodeOK
	case http.StatusConflict:
		code = trace.StatusCodeAlreadyExists
	}

	return aggregator.Status{Code: code, Message: codeToStr[code]}
}

var codeToStr = map[int32]string{
	trace.StatusCodeOK:                 `OK`,
	trace.StatusCodeCancelled:          `CANCELLED`,
	trace.StatusCodeUnknown:            `UNKNOWN`,
	trace.StatusCodeInvalidArgument:    `INVALID_ARGUMENT`,
	trace.StatusCodeDeadlineExceeded:   `DEADLINE_EXCEEDED`,
	trace.StatusCodeNotFound:           `NOT_FOUND`,
	trace.StatusCodeAlreadyExists:      `ALREADY_EXISTS`,
	trace.StatusCodePermissionDenied:   `PERMISSION_DENIED`,
	trace.StatusCodeResourceExhausted:  `RESOURCE_EXHAUSTED`,
	trace.StatusCodeFailedPrecondition: `FAILED_PRECONDITION`,
	trace.StatusCodeAborted:            `ABORTED`,
	trace.StatusCodeOutOfRange:         `OUT_OF_RANGE`,
	trace.StatusCodeUnimplemented:      `UNIMPLEMENTED`,
	trace.StatusCodeInternal:           `INTERNAL`,
	trace.StatusCodeUnavailable:        `UNAVAILABLE`,
	trace.StatusCodeDataLoss:           `DATA_LOSS`,
	trace.StatusCodeUnauthenticated:    `UNAUTHENTICATED`,
}

func isHealthEndpoint(path string) bool {
	// Health checking is pretty frequent and
	// traces collected for health endpoints
	// can be extremely noisy and expensive.
	// Disable canonical health checking endpoints
	// like /healthz and /_ah/health for now.
	if path == "/healthz" || path == "/_ah/health" || path == "/probe" {
		return true
	}
	return false
}
