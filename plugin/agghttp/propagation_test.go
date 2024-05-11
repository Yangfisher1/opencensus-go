package agghttp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Yangfisher1/opencensus-go/aggregator"
	"github.com/Yangfisher1/opencensus-go/plugin/agghttp/propagation/b3"
)

func TestRoundTripFormat(t *testing.T) {
	ctx := context.Background()
	ctx, span := aggregator.StartSpan(ctx, "test", aggregator.SpanKindUnspecified)
	sc := span.SpanContext()
	wantStr := fmt.Sprintf("height=%d", sc.Height)
	format := &b3.HTTPFormat{}

	srv := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		sc, ok := format.SpanContextFromRequest(req)
		if !ok {
			resp.WriteHeader(http.StatusBadRequest)
		}
		fmt.Fprintf(resp, "height=%d", sc.Height)
	}))
	req, err := http.NewRequest("GET", srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	format.SpanContextToRequest(span.SpanContext(), req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got, want := string(body), wantStr; got != want {
		t.Errorf("%s; want %s", got, want)
	}
	srv.Close()

}
