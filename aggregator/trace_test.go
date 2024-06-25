package aggregator

import (
	"fmt"
	"testing"
)

func TestSerializeHack(t *testing.T) {
	str := exportSpanDataToStringV3()
	fmt.Println(str)
}
