package diff_test

import (
	"fmt"
	"net"
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
