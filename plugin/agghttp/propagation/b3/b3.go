package b3

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/Yangfisher1/opencensus-go/aggregator"
	"github.com/Yangfisher1/opencensus-go/aggregator/propagation"
)

const (
	SpanHeightHeader = "X-B3-SpanHeight"
)

type HTTPFormat struct{}

var _ propagation.HTTPFormat = (*HTTPFormat)(nil)

// SpanContextFromRequest extracts a B3 span context from incoming requests.
func (f *HTTPFormat) SpanContextFromRequest(req *http.Request) (sc aggregator.SpanContext, ok bool) {
	height, ok := ParseHeight(req.Header.Get(SpanHeightHeader))
	if !ok {
		fmt.Println("Failed to parse Height")
		return aggregator.SpanContext{}, false
	}

	return aggregator.SpanContext{
		Height: height,
	}, true
}

func ParseHeight(height string) (uint32, bool) {
	if height == "" {
		return 0, false
	}
	b, err := hex.DecodeString(height)
	if err != nil || len(b) > 4 {
		return 0, false
	}

	h := binary.LittleEndian.Uint32(b)
	return h, true
}

func (f *HTTPFormat) SpanContextToRequest(sc aggregator.SpanContext, req *http.Request) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], sc.Height)
	req.Header.Set(SpanHeightHeader, hex.EncodeToString(b[:]))
}
