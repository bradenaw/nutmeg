package tsched

import (
	"runtime"
	"testing"
)

func TestGetGID(t *testing.T) {
	t.Log(getGID())

	b := make([]byte, 65536)
	b = b[:runtime.Stack(b, true)]
	t.Log(string(b))
}
