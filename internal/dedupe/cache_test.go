package dedupe

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := New(time.Second)
	n := time.Unix(10,0)
	if !c.Accept("x",n) || c.Accept("x",n.Add(time.Millisecond)) || !c.Accept("x",n.Add(time.Second)) {
		t.Fatal("ttl behavior")
	}
}
