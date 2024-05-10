package propagation

import (
	"bytes"
	"testing"

	"github.com/Yangfisher1/opencensus-go/aggregator"
)

func TestBinary(t *testing.T) {
	ctx := aggregator.SpanContext{Height: 257}
	b := []byte{1, 1, 0, 0}
	if b2 := Binary(ctx); !bytes.Equal(b2, b) {
		t.Errorf("Binary: got serialization %02x want %02x", b2, b)
	}

	sc, ok := FromBinary(b)
	if !ok {
		t.Errorf("FromBinary: got ok==%t, want true", ok)
	}
	if got := sc.Height; got != ctx.Height {
		t.Errorf("FromBinary: got height %d want %d", got, ctx.Height)
	}

	b[1] = 2
	sc, ok = FromBinary(b)
	if !ok {
		t.Errorf("FromBinary: decoding bytes containing an unsupported version: got ok==%t want false", ok)
	}
	if got := sc.Height; got != 513 {
		t.Errorf("FromBinary: got height %d want 258", got)
	}
}
