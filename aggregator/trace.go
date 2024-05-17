package aggregator

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Yangfisher1/opencensus-go/internal"
)

type tracer struct{}

var _ Tracer = &tracer{}

type span struct {
	data          *SpanData
	mu            sync.Mutex // protects the contents of *data (but not the pointer value.)
	spanContext   SpanContext
	lruAttributes *lruMap
	endOnce       sync.Once
}

type SpanContext struct {
	SpanID SpanID
}

type contextKey struct{}

func (t *tracer) FromContext(ctx context.Context) *Span {
	s, _ := ctx.Value(contextKey{}).(*Span)
	return s
}

// NewContext returns a new context with the given Span attached.
func (t *tracer) NewContext(parent context.Context, s *Span) context.Context {
	return context.WithValue(parent, contextKey{}, s)
}

// All available span kinds. Span kind must be either one of these values.
const (
	SpanKindUnspecified = iota
	SpanKindServer
	SpanKindClient
)

func (t *tracer) StartSpan(ctx context.Context, name string, spanKind int) (context.Context, *Span) {
	var parent SpanContext
	hasParent := false
	if p := t.FromContext(ctx); p != nil {
		parent = p.SpanContext()
		hasParent = true
	}
	span := startSpanInternal(name, hasParent, parent, spanKind)

	extSpan := NewSpan(span)
	return t.NewContext(ctx, extSpan), extSpan
}

func (t *tracer) StartSpanWithRemoteParent(ctx context.Context, name string, parent SpanContext, spanKind int) (context.Context, *Span) {
	// Only those who have a remote parent will trigger the function
	span := startSpanInternal(name, true, parent, spanKind)
	extSpan := NewSpan(span)
	return t.NewContext(ctx, extSpan), extSpan
}

func startSpanInternal(name string, hasParent bool, parent SpanContext, spanKind int) *span {
	s := &span{}
	// Check whether this is the first one
	if hasParent {
		s.data.ParentSpanId = parent.SpanID
	}

	s.data = &SpanData{
		SpanContext: s.spanContext,
		StartTime:   time.Now(),
		Name:        name,
		SpanKind:    spanKind,
	}

	s.lruAttributes = newLruMap(DefaultMaxAttributesPerSpan)

	return s
}

func (s *span) EndAndAggregate(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		return
	}
	s.endOnce.Do(func() {
		exp, _ := exporters.Load().(exportersMap)
		sd := s.makeSpanData()
		sd.EndTime = internal.MonotonicEndTime(sd.StartTime)
		// TODO: how about to use goroutine? maybe it's not ok since span needs to be propagated
		for e := range exp {
			errType := e.FilterSpan(sd)
			// FIXME: what if there're many situation happen at the same time?
			// Currently just keep the important info first
			switch errType {
			case OK:
				ssd := makeNormalSpanData(sd)
				// Valid one, encoding information into the response header
				buf := new(bytes.Buffer)
				err := json.NewEncoder(buf).Encode(ssd)
				if err != nil {
					fmt.Println("Failed to encoding data", err)
					return
				}
				w.Header().Add("Agg", buf.String())
			case Aggregate:
				ssd := makeNormalSpanData(sd)
				// Valid one, encoding information into the response header
				buf := new(bytes.Buffer)
				err := json.NewEncoder(buf).Encode(ssd)
				if err != nil {
					fmt.Println("Failed to encoding data", err)
					return
				}
				w.Header().Add("Agg", buf.String())
				e.AggregateSpanFromHeader(w.Header())
			}
		}
	})
}

func (s *span) EndAtClient(h *http.Header) {
	if s == nil {
		return
	}
	s.endOnce.Do(func() {
		exp, _ := exporters.Load().(exportersMap)
		sd := s.makeSpanData()
		sd.EndTime = internal.MonotonicEndTime(sd.StartTime)
		// TODO: how about to use goroutine? maybe it's not ok since span needs to be propagated
		for e := range exp {
			errType := e.FilterSpan(sd)
			// FIXME: what if there're many situation happen at the same time?
			// Currently just keep the important info first
			switch errType {
			case OK:
				ssd := makeNormalSpanData(sd)
				// Valid one, encoding information into the response header
				buf := new(bytes.Buffer)
				err := json.NewEncoder(buf).Encode(ssd)
				if err != nil {
					fmt.Println("Failed to encoding data", err)
					return
				}
				h.Add("Agg", buf.String())
			case Aggregate:
				ssd := makeNormalSpanData(sd)
				// Valid one, encoding information into the response header
				buf := new(bytes.Buffer)
				err := json.NewEncoder(buf).Encode(ssd)
				if err != nil {
					fmt.Println("Failed to encoding data", err)
					return
				}
				h.Add("Agg", buf.String())
				e.AggregateSpanFromHeader(*h)
			}
		}
	})
}

// SpanContext returns the SpanContext of the span.
func (s *span) SpanContext() SpanContext {
	if s == nil {
		return SpanContext{}
	}
	return s.spanContext
}

// SetStatus sets the status of the span, if it is recording events.
func (s *span) SetStatus(status Status) {
	s.mu.Lock()
	s.data.Status = status
	s.mu.Unlock()
}

// SetName sets the name of the span, if it is recording events.
func (s *span) SetName(name string) {
	s.mu.Lock()
	s.data.Name = name
	s.mu.Unlock()
}

func (s *span) copyToCappedAttributes(attributes []Attribute) {
	for _, a := range attributes {
		s.lruAttributes.add(a.key, a.value)
	}
}

func (s *span) AddAttributes(attributes ...Attribute) {
	s.mu.Lock()
	s.copyToCappedAttributes(attributes)
	s.mu.Unlock()
}

func (s *span) String() string {
	if s == nil {
		return "<nil>"
	}
	if s.data == nil {
		return fmt.Sprintf("span %s", s.spanContext.SpanID)
	}
	s.mu.Lock()
	str := fmt.Sprintf("span %s %q", s.spanContext.SpanID, s.data.Name)
	s.mu.Unlock()
	return str
}

func (s *span) makeSpanData() *SpanData {
	var sd SpanData
	s.mu.Lock()
	sd = *s.data
	if s.lruAttributes.len() > 0 {
		attributes := make(map[string]interface{}, s.lruAttributes.len())
		for _, key := range s.lruAttributes.keys() {
			value, ok := s.lruAttributes.get(key)
			if ok {
				keyStr := key.(string)
				attributes[keyStr] = value
			}
		}
		sd.Attributes = attributes
	}
	s.mu.Unlock()
	return &sd
}

func makeNormalSpanData(sd *SpanData) *NormalSpanData {
	return &NormalSpanData{
		SpanID:   ID(binary.BigEndian.Uint64(sd.SpanID[:])),
		ParentID: ID(binary.BigEndian.Uint64(sd.ParentSpanId[:])),
		Kind:     sd.SpanKind,
		Name:     sd.Name,
		// TODO: maybe a smarter way is to use a higher base to simplex the timestamp
		StartTime: strconv.FormatInt(sd.StartTime.UnixMicro(), 10),
		Duration:  strconv.FormatInt(sd.EndTime.UnixMicro()-sd.StartTime.UnixMicro(), 10),
	}
}
