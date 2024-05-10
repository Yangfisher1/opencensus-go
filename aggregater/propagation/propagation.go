package propagation

import (
	"encoding/binary"
	"net/http"

	"github.com/Yangfisher1/opencensus-go/aggregater"
)

func Binary(sc aggregater.SpanContext) []byte {
	if sc == (aggregater.SpanContext{}) {
		return nil
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], sc.Height)
	return b[:]
}

func FromBinary(b []byte) (sc aggregater.SpanContext, ok bool) {
	if len(b) == 0 {
		return aggregater.SpanContext{}, false
	}
	sc.Height = binary.LittleEndian.Uint32(b)
	return sc, true
}

type HTTPFormat interface {
	SpanContextFromRequest(req *http.Request) (sc aggregater.SpanContext, ok bool)
	SpanContextToRequest(sc aggregater.SpanContext, req *http.Request)
}
