package b3

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/Yangfisher1/opencensus-go/aggregater"
)

func TestHTTPFormat_FromReq(t *testing.T) {
	tests := []struct {
		name    string
		makeReq func() *http.Request
		wantSc  aggregater.SpanContext
		wantOk  bool
	}{
		{
			name: "height=257",
			makeReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set(SpanHeightHeader, "01010000")
				return req
			},
			wantSc: aggregater.SpanContext{
				Height: 257,
			},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &HTTPFormat{}
			sc, ok := f.SpanContextFromRequest(tt.makeReq())
			if ok != tt.wantOk {
				t.Errorf("SpanContextFromRequest() ok = %v, want %v", ok, tt.wantOk)
			}
			if !reflect.DeepEqual(sc, tt.wantSc) {
				t.Errorf("SpanContextFromRequest() span context = %v, want %v", sc, tt.wantSc)
			}
		})
	}
}

func TestHTTPFormat_ToReq(t *testing.T) {
	tests := []struct {
		name        string
		sc          aggregater.SpanContext
		wantHeaders map[string]string
	}{
		{
			name: "height=257",
			sc: aggregater.SpanContext{
				Height: 257,
			},
			wantHeaders: map[string]string{
				SpanHeightHeader: "01010000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &HTTPFormat{}
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			f.SpanContextToRequest(tt.sc, req)

			for k, v := range tt.wantHeaders {
				if got, want := req.Header.Get(k), v; got != want {
					t.Errorf("req.Header.Get(%q) = %q; want %q", k, got, want)
				}
			}
		})
	}
}
