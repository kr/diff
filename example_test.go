package diff_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"kr.dev/diff"
)

func ExampleEach() {
	var a, b net.Dialer
	a.Timeout = 5 * time.Second
	b.Timeout = 10 * time.Second
	b.LocalAddr = &net.TCPAddr{}
	diff.Each(fmt.Printf, a, b)
	// Output:
	// net.Dialer.Timeout: 5s != 10s
	// net.Dialer.LocalAddr: nil != &net.TCPAddr{IP:nil, ...}
}

var t = new(testing.T)

func ExampleTest() {
	// TestExample(t *testing.T) {
	got := makeDialer()

	want := &net.Dialer{
		Timeout:   10 * time.Second,
		LocalAddr: &net.TCPAddr{},
	}

	diff.Test(t, t.Errorf, got, want)
	// }
}

func makeDialer() *net.Dialer {
	return &net.Dialer{Timeout: 5 * time.Second}
}
