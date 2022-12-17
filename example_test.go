package diff_test

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
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
	// net.Dialer.LocalAddr: nil != &net.TCPAddr{
	//     IP:   nil,
	//     Port: 0,
	//     Zone: "",
	// }
}

func ExampleLog() {
	logger := log.New(os.Stdout, "", 0)

	reqURL, err := url.Parse("https://example.org/?q=one")
	if err != nil {
		return
	}

	knownURL := &url.URL{
		Scheme: "https",
		Host:   "example.org",
		Path:   "/",
	}

	diff.Log(reqURL, knownURL, diff.Logger(logger))
	// Output:
	// url.URL.RawQuery: "q=one" != ""
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
