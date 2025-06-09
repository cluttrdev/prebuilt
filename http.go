package main

import (
	"net"
	"net/http"
	"runtime"
	"time"
)

func defaultClient() *http.Client {
	return &http.Client{
		Transport: defaultTransport(),
	}
}

func defaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
}

func newAuthedClient(token string) *http.Client {
	return &http.Client{
		Transport: &authedTransport{
			Transport: defaultTransport(),
			token:     token,
		},
	}
}

type authedTransport struct {
	*http.Transport
	token string
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if values := req.Header.Values("Authorization"); len(values) == 0 {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.Transport.RoundTrip(req)
}
