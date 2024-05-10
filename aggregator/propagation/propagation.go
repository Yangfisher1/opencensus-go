package propagation

import (
	"encoding/binary"
	"net/http"

	"github.com/Yangfisher1/opencensus-go/aggregator"
)

func Binary(sc aggregator.SpanContext) []byte {
	if sc == (aggregator.SpanContext{}) {
		return nil
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], sc.Height)
	return b[:]
}

func FromBinary(b []byte) (sc aggregator.SpanContext, ok bool) {
	if len(b) == 0 {
		return aggregator.SpanContext{}, false
	}
	sc.Height = binary.LittleEndian.Uint32(b)
	return sc, true
}

type HTTPFormat interface {
	SpanContextFromRequest(req *http.Request) (sc aggregator.SpanContext, ok bool)
	SpanContextToRequest(sc aggregator.SpanContext, req *http.Request)
}
